package simulation

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/status"
)

// OutcomeBranchingAnalyzer provides evidence-based outcome branching for control actions.
// It analyzes current mesh state and historical patterns to predict best, expected, and worst case scenarios.
type OutcomeBranchingAnalyzer struct {
	mesh        status.MeshDrilldown
	history     []db.ControlActionRecord
	action      control.ControlAction
	evaluatedAt time.Time
}

// ScenarioBranch represents a single scenario outcome with detailed analysis.
type ScenarioBranch struct {
	Scenario          string         `json:"scenario"`
	Description       string         `json:"description"`
	Probability       float64        `json:"probability"`
	Confidence        float64        `json:"confidence"`
	Assumptions       []string       `json:"assumptions"`
	TriggerConditions []string       `json:"trigger_conditions"`
	TimeToResolution  time.Duration  `json:"time_to_resolution"`
	HealthScore       int            `json:"health_score"`
	Evidence          map[string]any `json:"evidence"`
}

// OutcomeAnalysis contains all scenario branches for an action.
type OutcomeAnalysis struct {
	BestCase     ScenarioBranch `json:"best_case"`
	ExpectedCase ScenarioBranch `json:"expected_case"`
	WorstCase    ScenarioBranch `json:"worst_case"`
	EvidenceBase map[string]any `json:"evidence_base"`
	Limitations  []string       `json:"limitations"`
}

// NewOutcomeBranchingAnalyzer creates a new analyzer for the given mesh state and action.
func NewOutcomeBranchingAnalyzer(mesh status.MeshDrilldown, history []db.ControlActionRecord, action control.ControlAction) *OutcomeBranchingAnalyzer {
	return &OutcomeBranchingAnalyzer{
		mesh:        mesh,
		history:     history,
		action:      action,
		evaluatedAt: time.Now().UTC(),
	}
}

// AnalyzeOutcomes generates evidence-based scenario branches for the action.
// It considers action type, target state, mesh health, and historical patterns
// to produce grounded probabilities rather than arbitrary estimates.
func (a *OutcomeBranchingAnalyzer) AnalyzeOutcomes() OutcomeAnalysis {
	baseProbabilities := a.calculateBaseProbabilities()
	meshContext := a.extractMeshContext()
	historicalContext := a.extractHistoricalContext()

	return OutcomeAnalysis{
		BestCase:     a.buildBestCase(baseProbabilities, meshContext, historicalContext),
		ExpectedCase: a.buildExpectedCase(baseProbabilities, meshContext, historicalContext),
		WorstCase:    a.buildWorstCase(baseProbabilities, meshContext, historicalContext),
		EvidenceBase: map[string]any{
			"mesh_score":          a.mesh.MeshHealth.Score,
			"mesh_state":          a.mesh.MeshHealth.State,
			"action_type":         a.action.ActionType,
			"target_transport":    a.action.TargetTransport,
			"historical_actions":  len(a.history),
			"correlated_failures": len(a.mesh.CorrelatedFailures),
		},
		Limitations: a.identifyLimitations(),
	}
}

// calculateBaseProbabilities determines baseline probabilities from action type and mesh state.
func (a *OutcomeBranchingAnalyzer) calculateBaseProbabilities() map[string]float64 {
	// Base success rates derived from action characteristics
	successRates := map[string]float64{
		control.ActionBackoffIncrease:                0.95,
		control.ActionBackoffReset:                   0.95,
		control.ActionTriggerHealthRecheck:           0.90,
		control.ActionResubscribeTransport:           0.85,
		control.ActionRestartTransport:               0.75,
		control.ActionTemporarilyDeprioritize:        0.40,
		control.ActionTemporarilySuppressNoisySource: 0.35,
		control.ActionClearSuppression:               0.30,
	}

	baseRate := successRates[a.action.ActionType]
	if baseRate == 0 {
		baseRate = 0.60 // Default for unknown actions
	}

	// Adjust based on action confidence
	confidenceFactor := a.action.Confidence
	if confidenceFactor == 0 {
		confidenceFactor = 0.75
	}

	adjusted := baseRate * (0.8 + 0.4*confidenceFactor) // Scale: 0.8x to 1.2x

	return map[string]float64{
		"best_case":  minFloat64(adjusted*1.15, 0.98),
		"expected":   adjusted,
		"worst_case": 1.0 - adjusted,
	}
}

// extractMeshContext gathers relevant context from mesh drilldown.
func (a *OutcomeBranchingAnalyzer) extractMeshContext() map[string]any {
	ctx := map[string]any{
		"health_score":    a.mesh.MeshHealth.Score,
		"health_state":    a.mesh.MeshHealth.State,
		"has_degradation": len(a.mesh.DegradedSegments) > 0,
	}

	// Count relevant transports
	targetInDegradedSegment := false
	degradedCount := 0
	for _, segment := range a.mesh.DegradedSegments {
		if segment.Severity == "critical" || segment.Severity == "warn" {
			degradedCount++
		}
		for _, t := range segment.Transports {
			if t == a.action.TargetTransport {
				targetInDegradedSegment = true
			}
		}
	}
	ctx["target_in_degraded_segment"] = targetInDegradedSegment
	ctx["degraded_segments_count"] = degradedCount

	// Check for active alerts on target
	targetHasAlerts := false
	alertSeverity := "none"
	for _, alert := range a.mesh.ActiveAlerts {
		if alert.TransportName == a.action.TargetTransport {
			targetHasAlerts = true
			if alertSeverityRank(alert.Severity) > alertSeverityRank(alertSeverity) {
				alertSeverity = alert.Severity
			}
		}
	}
	ctx["target_has_alerts"] = targetHasAlerts
	ctx["target_alert_severity"] = alertSeverity

	// Analyze correlated failures
	correlatedCount := 0
	for _, cf := range a.mesh.CorrelatedFailures {
		for _, t := range cf.Transports {
			if t == a.action.TargetTransport {
				correlatedCount++
				break
			}
		}
	}
	ctx["target_correlated_failures"] = correlatedCount

	return ctx
}

// extractHistoricalContext analyzes past actions for patterns.
func (a *OutcomeBranchingAnalyzer) extractHistoricalContext() map[string]any {
	ctx := map[string]any{
		"total_history": len(a.history),
	}

	if len(a.history) == 0 {
		ctx["success_rate"] = 0.0
		ctx["avg_resolution_time"] = 0
		return ctx
	}

	// Filter to same action type and target
	relevant := make([]db.ControlActionRecord, 0)
	for _, h := range a.history {
		if h.ActionType == a.action.ActionType {
			if a.action.TargetTransport == "" || h.TargetTransport == a.action.TargetTransport {
				relevant = append(relevant, h)
			}
		}
	}

	if len(relevant) == 0 {
		ctx["success_rate"] = 0.0
		ctx["relevant_actions"] = 0
		return ctx
	}

	successCount := 0
	totalDuration := time.Duration(0)
	durationCount := 0

	for _, h := range relevant {
		if h.Result == control.ResultExecutedSuccessfully || h.Result == control.ResultExecutedNoop {
			successCount++
		}

		if h.ExecutedAt != "" && h.CompletedAt != "" {
			executed, err1 := time.Parse(time.RFC3339, h.ExecutedAt)
			completed, err2 := time.Parse(time.RFC3339, h.CompletedAt)
			if err1 == nil && err2 == nil && completed.After(executed) {
				totalDuration += completed.Sub(executed)
				durationCount++
			}
		}
	}

	ctx["relevant_actions"] = len(relevant)
	ctx["success_rate"] = float64(successCount) / float64(len(relevant))

	if durationCount > 0 {
		ctx["avg_resolution_time"] = totalDuration / time.Duration(durationCount)
	}

	return ctx
}

// buildBestCase constructs the optimistic but realistic scenario.
func (a *OutcomeBranchingAnalyzer) buildBestCase(probs map[string]float64, meshCtx, histCtx map[string]any) ScenarioBranch {
	probability := probs["best_case"]
	confidence := a.action.Confidence * 0.95

	// Adjust based on mesh health
	healthScore := a.mesh.MeshHealth.Score
	if healthScore > 80 {
		probability = minFloat64(probability*1.05, 0.99)
		confidence *= 1.05
	}

	// Adjust based on historical success
	if histRate, ok := histCtx["success_rate"].(float64); ok && histRate > 0.8 {
		probability = minFloat64(probability*1.02, 0.99)
	}

	description := a.bestCaseDescription()
	assumptions := a.bestCaseAssumptions(meshCtx)
	triggers := a.bestCaseTriggers()
	ttr := a.bestCaseTimeToResolution(histCtx)

	return ScenarioBranch{
		Scenario:          "best_case",
		Description:       description,
		Probability:       probability,
		Confidence:        minFloat64(confidence, 0.99),
		Assumptions:       assumptions,
		TriggerConditions: triggers,
		TimeToResolution:  ttr,
		HealthScore:       minInt(100, healthScore+15),
		Evidence: map[string]any{
			"mesh_score_boost":   healthScore > 70,
			"no_blockers":        len(a.mesh.MeshHealthExplanation.RecoveryBlockers) == 0,
			"historical_success": histCtx["success_rate"],
		},
	}
}

// buildExpectedCase constructs the most likely scenario based on evidence.
func (a *OutcomeBranchingAnalyzer) buildExpectedCase(probs map[string]float64, meshCtx, histCtx map[string]any) ScenarioBranch {
	probability := probs["expected"]
	confidence := a.action.Confidence

	// Adjust based on mesh complexity
	complexityFactor := 1.0
	if len(a.mesh.CorrelatedFailures) > 0 {
		complexityFactor -= 0.05 * float64(len(a.mesh.CorrelatedFailures))
	}
	if degradedCount, ok := meshCtx["degraded_segments_count"].(int); ok && degradedCount > 0 {
		complexityFactor -= 0.03 * float64(degradedCount)
	}
	if complexityFactor < 0.7 {
		complexityFactor = 0.7
	}
	probability *= complexityFactor
	confidence *= complexityFactor

	description := a.expectedCaseDescription(meshCtx)
	assumptions := a.expectedCaseAssumptions(meshCtx, histCtx)
	triggers := a.expectedCaseTriggers(meshCtx)
	ttr := a.expectedCaseTimeToResolution(histCtx)

	return ScenarioBranch{
		Scenario:          "expected_case",
		Description:       description,
		Probability:       probability,
		Confidence:        confidence,
		Assumptions:       assumptions,
		TriggerConditions: triggers,
		TimeToResolution:  ttr,
		HealthScore:       a.expectedHealthScore(meshCtx),
		Evidence: map[string]any{
			"mesh_state":         a.mesh.MeshHealth.State,
			"correlated_count":   len(a.mesh.CorrelatedFailures),
			"historical_success": histCtx["success_rate"],
		},
	}
}

// buildWorstCase constructs the pessimistic but possible scenario.
func (a *OutcomeBranchingAnalyzer) buildWorstCase(probs map[string]float64, meshCtx, histCtx map[string]any) ScenarioBranch {
	probability := probs["worst_case"]
	confidence := a.action.Confidence * 0.85

	// Adjust based on risk factors
	riskMultiplier := 1.0
	if a.mesh.MeshHealth.State == "failed" || a.mesh.MeshHealth.State == "unstable" {
		riskMultiplier += 0.3
	}
	if targetHasAlerts, ok := meshCtx["target_has_alerts"].(bool); ok && targetHasAlerts {
		riskMultiplier += 0.15
	}
	if targetInSeg, ok := meshCtx["target_in_degraded_segment"].(bool); ok && targetInSeg {
		riskMultiplier += 0.1
	}
	probability *= riskMultiplier
	if probability > 0.6 {
		probability = 0.6 // Cap worst case probability
	}

	description := a.worstCaseDescription(meshCtx)
	assumptions := a.worstCaseAssumptions(meshCtx)
	triggers := a.worstCaseTriggers(meshCtx)
	ttr := a.worstCaseTimeToResolution()

	return ScenarioBranch{
		Scenario:          "worst_case",
		Description:       description,
		Probability:       probability,
		Confidence:        confidence,
		Assumptions:       assumptions,
		TriggerConditions: triggers,
		TimeToResolution:  ttr,
		HealthScore:       maxInt(0, a.mesh.MeshHealth.Score-25),
		Evidence: map[string]any{
			"mesh_risk":       a.mesh.MeshHealth.State == "failed" || a.mesh.MeshHealth.State == "unstable",
			"target_alerts":   meshCtx["target_has_alerts"],
			"correlated_risk": meshCtx["target_correlated_failures"],
		},
	}
}

// bestCaseDescription generates a description for the best case scenario.
func (a *OutcomeBranchingAnalyzer) bestCaseDescription() string {
	switch a.action.ActionType {
	case control.ActionRestartTransport:
		return fmt.Sprintf("Transport %s restarts cleanly, reconnects immediately, and resumes full throughput without message loss", a.action.TargetTransport)
	case control.ActionResubscribeTransport:
		return fmt.Sprintf("Transport %s successfully resubscribes with minimal interruption and no missed messages", a.action.TargetTransport)
	case control.ActionBackoffIncrease:
		return "Backoff increase reduces retry pressure, allowing the transport to stabilize naturally"
	case control.ActionBackoffReset:
		return "Backoff reset restores normal reconnection timing with immediate effect"
	case control.ActionTriggerHealthRecheck:
		return "Health recheck confirms improved state and clears stale alerts"
	default:
		return fmt.Sprintf("Action %s succeeds with optimal timing and no side effects", a.action.ActionType)
	}
}

// expectedCaseDescription generates a description for the expected case scenario.
func (a *OutcomeBranchingAnalyzer) expectedCaseDescription(meshCtx map[string]any) string {
	base := ""
	switch a.action.ActionType {
	case control.ActionRestartTransport:
		base = fmt.Sprintf("Transport %s restarts, reconnects within expected window", a.action.TargetTransport)
	case control.ActionResubscribeTransport:
		base = fmt.Sprintf("Transport %s resubscribes successfully", a.action.TargetTransport)
	case control.ActionBackoffIncrease:
		base = "Backoff increase takes effect, reducing retry frequency"
	case control.ActionBackoffReset:
		base = "Backoff reset completes, normal retry timing restored"
	case control.ActionTriggerHealthRecheck:
		base = "Health recheck completes, status updated"
	default:
		base = fmt.Sprintf("Action %s completes as designed", a.action.ActionType)
	}

	// Add context about any expected friction
	if hasDegradation, ok := meshCtx["has_degradation"].(bool); ok && hasDegradation {
		base += " with some delay due to degraded mesh segments"
	}

	return base
}

// worstCaseDescription generates a description for the worst case scenario.
func (a *OutcomeBranchingAnalyzer) worstCaseDescription(meshCtx map[string]any) string {
	var issues []string

	if targetHasAlerts, ok := meshCtx["target_has_alerts"].(bool); ok && targetHasAlerts {
		issues = append(issues, "persistent alerts remain unresolved")
	}
	if targetInSeg, ok := meshCtx["target_in_degraded_segment"].(bool); ok && targetInSeg {
		issues = append(issues, "segment-level degradation continues")
	}
	if len(a.mesh.CorrelatedFailures) > 0 {
		issues = append(issues, "correlated failures suggest systemic issue")
	}
	if a.mesh.MeshHealth.State == "failed" {
		issues = append(issues, "mesh health continues to fail")
	}

	base := ""
	switch a.action.ActionType {
	case control.ActionRestartTransport:
		base = fmt.Sprintf("Transport %s fails to restart or enters crash loop", a.action.TargetTransport)
	case control.ActionResubscribeTransport:
		base = fmt.Sprintf("Transport %s fails to resubscribe or experiences extended downtime", a.action.TargetTransport)
	case control.ActionBackoffIncrease:
		base = "Backoff increase has no effect, underlying issue persists"
	case control.ActionBackoffReset:
		base = "Backoff reset triggers aggressive retries that fail"
	default:
		base = fmt.Sprintf("Action %s fails or exacerbates existing issues", a.action.ActionType)
	}

	if len(issues) > 0 {
		return fmt.Sprintf("%s; %s", base, strings.Join(issues, ", "))
	}
	return base
}

// bestCaseAssumptions lists assumptions for the best case scenario.
func (a *OutcomeBranchingAnalyzer) bestCaseAssumptions(meshCtx map[string]any) []string {
	assumptions := []string{
		"Target transport responds normally to control signals",
		"Network connectivity remains stable during action",
		"No concurrent changes to transport configuration",
	}

	if healthScore := a.mesh.MeshHealth.Score; healthScore > 75 {
		assumptions = append(assumptions, "Strong mesh health supports quick recovery")
	}

	if len(a.mesh.CorrelatedFailures) == 0 {
		assumptions = append(assumptions, "No correlated failures suggesting systemic issues")
	}

	if targetHasAlerts, ok := meshCtx["target_has_alerts"].(bool); ok && !targetHasAlerts {
		assumptions = append(assumptions, "No active alerts on target transport")
	}

	return assumptions
}

// expectedCaseAssumptions lists assumptions for the expected case scenario.
func (a *OutcomeBranchingAnalyzer) expectedCaseAssumptions(meshCtx, histCtx map[string]any) []string {
	assumptions := []string{
		"Transport behavior follows historical patterns",
		"Mesh health remains within current range during execution",
	}

	// Add historical context if available
	if successRate, ok := histCtx["success_rate"].(float64); ok && successRate > 0 {
		assumptions = append(assumptions, fmt.Sprintf("Historical success rate of %.0f%% supports expected outcome", successRate*100))
	}

	// Add mesh context
	if hasDegradation, ok := meshCtx["has_degradation"].(bool); ok && hasDegradation {
		assumptions = append(assumptions, "Degraded segments introduce minor delays but do not block recovery")
	}

	return assumptions
}

// worstCaseAssumptions lists assumptions for the worst case scenario.
func (a *OutcomeBranchingAnalyzer) worstCaseAssumptions(meshCtx map[string]any) []string {
	assumptions := []string{
		"Underlying issue is more severe than current metrics indicate",
		"Action may interact with unknown failure modes",
	}

	if a.mesh.MeshHealth.State == "failed" {
		assumptions = append(assumptions, "Mesh-level failure state prevents isolated recovery")
	}

	if targetHasAlerts, ok := meshCtx["target_has_alerts"].(bool); ok && targetHasAlerts {
		assumptions = append(assumptions, "Active alerts indicate persistent unaddressed issues")
	}

	if cf, ok := meshCtx["target_correlated_failures"].(int); ok && cf > 0 {
		assumptions = append(assumptions, fmt.Sprintf("%d correlated failure(s) suggest broader systemic problem", cf))
	}

	return assumptions
}

// bestCaseTriggers lists conditions that would lead to the best case.
func (a *OutcomeBranchingAnalyzer) bestCaseTriggers() []string {
	triggers := []string{
		"Transport is responsive to control signals",
		"Network path to target is stable",
	}

	switch a.action.ActionType {
	case control.ActionRestartTransport:
		triggers = append(triggers, "Clean disconnect/reconnect cycle completes")
		triggers = append(triggers, "No message backlog accumulation")
	case control.ActionResubscribeTransport:
		triggers = append(triggers, "Subscription endpoint available")
		triggers = append(triggers, "Authentication credentials valid")
	case control.ActionBackoffIncrease:
		triggers = append(triggers, "Retry storm is primary cause of instability")
	}

	return triggers
}

// expectedCaseTriggers lists conditions that would lead to the expected case.
func (a *OutcomeBranchingAnalyzer) expectedCaseTriggers(meshCtx map[string]any) []string {
	triggers := []string{
		"Action executes without operator intervention",
		"Target transport state matches assumptions",
	}

	if hasDegradation, ok := meshCtx["has_degradation"].(bool); ok && hasDegradation {
		triggers = append(triggers, "Degraded segments do not worsen during execution")
	}

	return triggers
}

// worstCaseTriggers lists conditions that would lead to the worst case.
func (a *OutcomeBranchingAnalyzer) worstCaseTriggers(meshCtx map[string]any) []string {
	triggers := []string{
		"Underlying root cause is systemic, not transport-local",
	}

	if a.mesh.MeshHealth.State == "failed" {
		triggers = append(triggers, "Mesh-level failure prevents any single-transport recovery")
	}

	if targetInSeg, ok := meshCtx["target_in_degraded_segment"].(bool); ok && targetInSeg {
		triggers = append(triggers, "Segment-level degradation is primary failure mode")
	}

	return triggers
}

// bestCaseTimeToResolution estimates time to resolution for best case.
func (a *OutcomeBranchingAnalyzer) bestCaseTimeToResolution(histCtx map[string]any) time.Duration {
	base := a.actionBaseDuration()

	// Use historical data if available
	if avgTTR, ok := histCtx["avg_resolution_time"].(time.Duration); ok && avgTTR > 0 {
		return avgTTR / 2 // Best case is half the average
	}

	return base / 2
}

// expectedCaseTimeToResolution estimates time to resolution for expected case.
func (a *OutcomeBranchingAnalyzer) expectedCaseTimeToResolution(histCtx map[string]any) time.Duration {
	base := a.actionBaseDuration()

	if avgTTR, ok := histCtx["avg_resolution_time"].(time.Duration); ok && avgTTR > 0 {
		return avgTTR
	}

	return base
}

// worstCaseTimeToResolution estimates time to resolution for worst case.
func (a *OutcomeBranchingAnalyzer) worstCaseTimeToResolution() time.Duration {
	base := a.actionBaseDuration()

	// Worst case can be significantly longer
	multiplier := 5
	if a.mesh.MeshHealth.State == "failed" {
		multiplier = 10
	}

	return base * time.Duration(multiplier)
}

// actionBaseDuration returns the base duration estimate for the action type.
func (a *OutcomeBranchingAnalyzer) actionBaseDuration() time.Duration {
	durations := map[string]time.Duration{
		control.ActionBackoffIncrease:      0,
		control.ActionBackoffReset:         0,
		control.ActionTriggerHealthRecheck: 2 * time.Second,
		control.ActionResubscribeTransport: 3 * time.Second,
		control.ActionRestartTransport:     5 * time.Second,
	}

	if d, ok := durations[a.action.ActionType]; ok {
		return d
	}
	return 3 * time.Second // Default
}

// expectedHealthScore predicts the resulting health score for expected case.
func (a *OutcomeBranchingAnalyzer) expectedHealthScore(meshCtx map[string]any) int {
	current := a.mesh.MeshHealth.Score

	// Base improvement estimate
	improvement := 10
	switch a.action.ActionType {
	case control.ActionRestartTransport:
		improvement = 15
	case control.ActionResubscribeTransport:
		improvement = 12
	case control.ActionBackoffIncrease, control.ActionBackoffReset:
		improvement = 5
	}

	// Reduce improvement if there are complications
	if targetInSeg, ok := meshCtx["target_in_degraded_segment"].(bool); ok && targetInSeg {
		improvement -= 5
	}
	if cf, ok := meshCtx["target_correlated_failures"].(int); ok && cf > 0 {
		improvement -= 3 * cf
	}

	return minInt(100, current+improvement)
}

// identifyLimitations notes any limitations in the analysis.
func (a *OutcomeBranchingAnalyzer) identifyLimitations() []string {
	limitations := []string{
		"Predictions based on current mesh state only",
		"Does not account for concurrent operator actions",
	}

	if len(a.history) == 0 {
		limitations = append(limitations, "No historical action data available for pattern matching")
	}

	if a.action.ActionType == control.ActionTemporarilyDeprioritize ||
		a.action.ActionType == control.ActionTemporarilySuppressNoisySource ||
		a.action.ActionType == control.ActionClearSuppression {
		limitations = append(limitations, "Advisory-only action: actual outcome depends on operator follow-through")
	}

	return limitations
}

// Helper functions

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func alertSeverityRank(severity string) int {
	switch severity {
	case "critical":
		return 3
	case "warn":
		return 2
	default:
		return 1
	}
}

// ToOutcomeBranches converts ScenarioBranches to the OutcomeBranch type used in SimulationResult.
func (a *OutcomeAnalysis) ToOutcomeBranches() []OutcomeBranch {
	branches := []struct {
		name   string
		branch ScenarioBranch
	}{
		{"best_case", a.BestCase},
		{"expected_case", a.ExpectedCase},
		{"worst_case", a.WorstCase},
	}

	out := make([]OutcomeBranch, 0, len(branches))
	for _, b := range branches {
		out = append(out, OutcomeBranch{
			Scenario:             b.name,
			Probability:          b.branch.Probability,
			Description:          b.branch.Description,
			SystemState:          a.inferSystemState(b.branch),
			HealthScore:          b.branch.HealthScore,
			RecoveryTime:         b.branch.TimeToResolution,
			TriggeringConditions: b.branch.TriggerConditions,
		})
	}

	// Sort by probability descending
	sort.Slice(out, func(i, j int) bool {
		return out[i].Probability > out[j].Probability
	})

	return out
}

// inferSystemState maps a scenario branch to a system state string.
func (a *OutcomeAnalysis) inferSystemState(branch ScenarioBranch) string {
	switch branch.Scenario {
	case "best_case":
		return "healthy"
	case "expected_case":
		if branch.HealthScore >= 70 {
			return "degraded_improving"
		}
		return "degraded_stable"
	case "worst_case":
		if branch.HealthScore < 30 {
			return "failed"
		}
		return "degraded_worsening"
	default:
		return "unknown"
	}
}
