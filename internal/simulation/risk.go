package simulation

import (
	"fmt"
	"time"

	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/intelligence"
	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/selfobs"
	"github.com/mel-project/mel/internal/status"
	"github.com/mel-project/mel/internal/transport"
)

// RiskScorer provides explicit risk evaluation for control actions.
// Risk scoring is transparent - every score must have explainable contributing factors.
// No black-box algorithms are used. All scores are deterministic and reproducible.
type RiskScorer struct {
	database           *db.DB
	freshnessTracker   *selfobs.FreshnessTracker
	healthRegistry     *selfobs.HealthRegistry
	actionHistoryCache map[string][]db.ControlActionRecord
}

// RiskEvaluationResult contains the complete risk assessment for an action.
// This is distinct from the simulation's RiskAssessment to provide more
// detailed factor-by-factor scoring with full transparency.
type RiskEvaluationResult struct {
	ActionType             string                  `json:"action_type"`
	TargetTransport        string                  `json:"target_transport,omitempty"`
	RiskLevel              RiskLevel               `json:"risk_level"`
	RiskScore              RiskScore               `json:"risk_score"`
	Reversibility          ReversibilityAssessment `json:"reversibility"`
	Uncertainty            UncertaintyAssessment   `json:"uncertainty"`
	ContributingFactors    []RiskFactorDetail      `json:"contributing_factors"`
	BlastRadiusEstimate    float64                 `json:"blast_radius_estimate"`
	BlastRadiusExplanation string                  `json:"blast_radius_explanation"`
	RecommendedPrecautions []string                `json:"recommended_precautions,omitempty"`
	EvaluatedAt            time.Time               `json:"evaluated_at"`
}

// RiskScore represents a numeric risk score with explanation
type RiskScore struct {
	Score       float64 `json:"score"`
	Explanation string  `json:"explanation"`
}

// RiskFactorDetail represents a single factor that contributed to the risk assessment.
// This extends the base RiskFactor type with weight and impact metrics.
type RiskFactorDetail struct {
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Level       RiskLevel `json:"level"`
	Weight      float64 `json:"weight"`
	Impact      float64 `json:"impact"`
	Likelihood  float64 `json:"likelihood"`
	Mitigatable bool    `json:"mitigatable"`
	Explanation string  `json:"explanation,omitempty"`
}

// ReversibilityAssessment evaluates how reversible an action is
type ReversibilityAssessment struct {
	IsReversible       bool     `json:"is_reversible"`
	ReversibilityLevel string   `json:"reversibility_level"`
	ReversalAction     string   `json:"reversal_action,omitempty"`
	ReversalConfidence float64  `json:"reversal_confidence"`
	TimeWindowSeconds  int      `json:"time_window_seconds,omitempty"`
	SideEffects        []string `json:"side_effects,omitempty"`
	Explanation        string   `json:"explanation"`
}

// UncertaintyAssessment identifies unknown factors and their impact on confidence
type UncertaintyAssessment struct {
	UnknownFactors      []string       `json:"unknown_factors"`
	ConfidenceReduction float64        `json:"confidence_reduction"`
	Mitigations         []string       `json:"mitigations,omitempty"`
	KnowledgeGaps       []KnowledgeGap `json:"knowledge_gaps"`
	Explanation         string         `json:"explanation"`
}

// KnowledgeGap represents a specific area of missing information
type KnowledgeGap struct {
	Domain      string  `json:"domain"`
	Impact      float64 `json:"impact"`
	Mitigatable bool    `json:"mitigatable"`
	Description string  `json:"description"`
}

// NewRiskScorer creates a new RiskScorer with the given dependencies
func NewRiskScorer(database *db.DB) *RiskScorer {
	return &RiskScorer{
		database:           database,
		freshnessTracker:   selfobs.GetGlobalFreshnessTracker(),
		healthRegistry:     selfobs.GetGlobalRegistry(),
		actionHistoryCache: make(map[string][]db.ControlActionRecord),
	}
}

// NewRiskScorerWithDeps creates a RiskScorer with explicit dependencies (useful for testing)
func NewRiskScorerWithDeps(database *db.DB, freshnessTracker *selfobs.FreshnessTracker, healthRegistry *selfobs.HealthRegistry) *RiskScorer {
	if freshnessTracker == nil {
		freshnessTracker = selfobs.GetGlobalFreshnessTracker()
	}
	if healthRegistry == nil {
		healthRegistry = selfobs.GetGlobalRegistry()
	}
	return &RiskScorer{
		database:           database,
		freshnessTracker:   freshnessTracker,
		healthRegistry:     healthRegistry,
		actionHistoryCache: make(map[string][]db.ControlActionRecord),
	}
}

// CalculateRisk performs comprehensive risk evaluation for a control action.
// This method evaluates:
//   - Action type risk (from reality matrix)
//   - Target transport state risk
//   - Mesh health context
//   - Historical action success/failure rates
//   - Dependency confidence
//   - Data freshness
//
// It produces RiskLevel (low/medium/high/critical) with detailed explanations
// for each contributing factor.
func (rs *RiskScorer) CalculateRisk(
	action control.ControlAction,
	reality control.ActionReality,
	mesh status.MeshDrilldown,
	runtimeTransports []transport.Health,
) RiskEvaluationResult {
	evaluatedAt := time.Now().UTC()
	factors := make([]RiskFactorDetail, 0)

	// Factor 1: Action type risk from reality matrix
	actionTypeRisk := rs.evaluateActionTypeRisk(action, reality)
	factors = append(factors, actionTypeRisk)

	// Factor 2: Target transport state risk
	transportRisk := rs.evaluateTransportRisk(action, runtimeTransports)
	factors = append(factors, transportRisk)

	// Factor 3: Mesh health context
	meshRisk := rs.evaluateMeshRisk(mesh)
	factors = append(factors, meshRisk)

	// Factor 4: Historical action success/failure rates
	historyRisk := rs.evaluateHistoricalRisk(action)
	factors = append(factors, historyRisk)

	// Factor 5: Dependency confidence
	dependencyRisk := rs.evaluateDependencyRisk(action, mesh, runtimeTransports)
	factors = append(factors, dependencyRisk)

	// Factor 6: Data freshness
	freshnessRisk := rs.evaluateDataFreshness(action)
	factors = append(factors, freshnessRisk)

	// Calculate blast radius using existing intelligence function
	blastRadius, blastExplanation := rs.calculateBlastRadius(mesh, action)

	// Calculate overall risk score
	riskScore := rs.aggregateRiskScore(factors, blastRadius)

	// Determine risk level
	riskLevel := rs.determineRiskLevel(riskScore.Score, reality, mesh)

	// Assess reversibility
	reversibility := rs.assessReversibility(action, reality)

	// Assess uncertainty
	uncertainty := rs.assessUncertainty(action, mesh, runtimeTransports)

	// Generate recommendations
	precautions := rs.generatePrecautions(riskLevel, reversibility, action, reality)

	return RiskEvaluationResult{
		ActionType:             action.ActionType,
		TargetTransport:        action.TargetTransport,
		RiskLevel:              riskLevel,
		RiskScore:              riskScore,
		Reversibility:          reversibility,
		Uncertainty:            uncertainty,
		ContributingFactors:    factors,
		BlastRadiusEstimate:    blastRadius,
		BlastRadiusExplanation: blastExplanation,
		RecommendedPrecautions: precautions,
		EvaluatedAt:            evaluatedAt,
	}
}

// evaluateActionTypeRisk assesses risk based on the action type from reality matrix
func (rs *RiskScorer) evaluateActionTypeRisk(action control.ControlAction, reality control.ActionReality) RiskFactorDetail {
	baseWeight := 0.15
	impact := 0.0
	level := RiskLevelNone
	explanation := ""

	switch action.ActionType {
	case control.ActionRestartTransport:
		if reality.ActuatorExists {
			impact = 0.3
			level = RiskLevelLow
			explanation = "Transport restart interrupts connectivity but is bounded by reconnect logic"
		} else {
			impact = 0.8
			level = RiskLevelHigh
			explanation = "No verified actuator exists for transport restart"
		}
	case control.ActionResubscribeTransport:
		if reality.ActuatorExists {
			impact = 0.2
			level = RiskLevelLow
			explanation = "Resubscription has limited blast radius within transport"
		} else {
			impact = 0.8
			level = RiskLevelHigh
			explanation = "No verified actuator exists for resubscription"
		}
	case control.ActionBackoffIncrease:
		if reality.ActuatorExists {
			impact = 0.1
			level = RiskLevelLow
			explanation = "Backoff increase is fully reversible and low-risk"
		} else {
			impact = 0.6
			level = RiskLevelMedium
			explanation = "Backoff control not verified in this build"
		}
	case control.ActionBackoffReset:
		if reality.ActuatorExists {
			impact = 0.05
			level = RiskLevelNone
			explanation = "Backoff reset is fully reversible with minimal impact"
		} else {
			impact = 0.6
			level = RiskLevelMedium
			explanation = "Backoff control not verified in this build"
		}
	case control.ActionTemporarilyDeprioritize:
		impact = 0.5
		level = RiskLevelMedium
		explanation = "Routing deprioritization requires verified path selection which is not shipped"
	case control.ActionTemporarilySuppressNoisySource:
		impact = 0.6
		level = RiskLevelMedium
		explanation = "Source suppression requires metrics-backed verification which is not shipped"
	case control.ActionClearSuppression:
		impact = 0.4
		level = RiskLevelMedium
		explanation = "Suppression clearing requires actuator that is not shipped"
	case control.ActionTriggerHealthRecheck:
		if reality.ActuatorExists {
			impact = 0.05
			level = RiskLevelNone
			explanation = "Health recheck is read-only and low-risk"
		} else {
			impact = 0.4
			level = RiskLevelMedium
			explanation = "Health recheck actuator not fully verified"
		}
	default:
		impact = 0.7
		level = RiskLevelHigh
		explanation = fmt.Sprintf("Unknown action type %s has unbounded risk", action.ActionType)
	}

	// Adjust for reality matrix safety indicators
	if reality.SafeForGuardedAuto {
		impact *= 0.8
		explanation += "; marked safe for guarded automation"
	}
	if reality.AdvisoryOnly {
		impact = maxFloat(impact, 0.5)
		if level == RiskLevelNone || level == RiskLevelLow {
			level = RiskLevelMedium
		}
		explanation += "; advisory-only action has execution risk"
	}

	return RiskFactorDetail{
		Category:    "action_type",
		Description: "Action type risk from reality matrix",
		Level:       level,
		Weight:      baseWeight,
		Impact:      impact,
		Likelihood:  1.0, // Action type is certain
		Mitigatable: reality.Reversible,
		Explanation: explanation,
	}
}

// evaluateTransportRisk assesses risk based on target transport state
func (rs *RiskScorer) evaluateTransportRisk(action control.ControlAction, runtimeTransports []transport.Health) RiskFactorDetail {
	baseWeight := 0.20

	if action.TargetTransport == "" {
		return RiskFactorDetail{
			Category:    "transport_state",
			Description: "Target transport state risk",
			Level:       RiskLevelNone,
			Weight:      baseWeight,
			Impact:      0.0,
			Likelihood:  0.0,
			Mitigatable: true,
			Explanation: "No target transport specified; risk assessed at mesh level",
		}
	}

	// Find transport health
	var transportHealth *transport.Health
	for i := range runtimeTransports {
		if runtimeTransports[i].Name == action.TargetTransport {
			transportHealth = &runtimeTransports[i]
			break
		}
	}

	if transportHealth == nil {
		return RiskFactorDetail{
			Category:    "transport_state",
			Description: "Target transport state risk",
			Level:       RiskLevelMedium,
			Weight:      baseWeight,
			Impact:      0.5,
			Likelihood:  1.0,
			Mitigatable: false,
			Explanation: fmt.Sprintf("Target transport %s not found in runtime; cannot assess current state", action.TargetTransport),
		}
	}

	impact := 0.0
	level := RiskLevelNone
	explanation := ""

	switch transportHealth.State {
	case transport.StateLive:
		impact = 0.1
		level = RiskLevelLow
		explanation = fmt.Sprintf("Target transport %s is live; action poses minimal disruption risk", action.TargetTransport)
	case transport.StateIdle:
		impact = 0.2
		level = RiskLevelLow
		explanation = fmt.Sprintf("Target transport %s is idle (connected but no recent ingest)", action.TargetTransport)
	case transport.StateRetrying:
		impact = 0.4
		level = RiskLevelMedium
		explanation = fmt.Sprintf("Target transport %s is retrying; action may compound recovery effort", action.TargetTransport)
	case transport.StateFailed:
		impact = 0.3
		level = RiskLevelLow
		explanation = fmt.Sprintf("Target transport %s is already failed; action is attempting recovery", action.TargetTransport)
	case transport.StateConnecting:
		impact = 0.5
		level = RiskLevelMedium
		explanation = fmt.Sprintf("Target transport %s is connecting; concurrent action risks race conditions", action.TargetTransport)
	case transport.StateDisconnected:
		impact = 0.3
		level = RiskLevelLow
		explanation = fmt.Sprintf("Target transport %s is disconnected; restart may be appropriate", action.TargetTransport)
	default:
		impact = 0.4
		level = RiskLevelMedium
		explanation = fmt.Sprintf("Target transport %s is in unknown state %s", action.TargetTransport, transportHealth.State)
	}

	// Adjust for error indicators
	if transportHealth.ErrorCount > 10 {
		impact += 0.1
		explanation += "; high error count suggests instability"
	}
	if transportHealth.ConsecutiveTimeouts > 5 {
		impact += 0.1
		explanation += "; multiple consecutive timeouts indicate persistent issues"
	}

	return RiskFactorDetail{
		Category:    "transport_state",
		Description: fmt.Sprintf("Transport %s state: %s", action.TargetTransport, transportHealth.State),
		Level:       level,
		Weight:      baseWeight,
		Impact:      minFloat(impact, 1.0),
		Likelihood:  1.0,
		Mitigatable: true,
		Explanation: explanation,
	}
}

// evaluateMeshRisk assesses risk based on overall mesh health context
func (rs *RiskScorer) evaluateMeshRisk(mesh status.MeshDrilldown) RiskFactorDetail {
	baseWeight := 0.20
	impact := 0.0
	level := RiskLevelNone
	explanations := make([]string, 0)

	// Score-based risk adjustment
	switch {
	case mesh.MeshHealth.Score >= 80:
		impact += 0.0
		level = RiskLevelNone
		explanations = append(explanations, "mesh health is good")
	case mesh.MeshHealth.Score >= 60:
		impact += 0.1
		level = RiskLevelLow
		explanations = append(explanations, "mesh health is degraded")
	case mesh.MeshHealth.Score >= 40:
		impact += 0.2
		level = RiskLevelMedium
		explanations = append(explanations, "mesh health is significantly degraded")
	default:
		impact += 0.3
		level = RiskLevelHigh
		explanations = append(explanations, "mesh health is poor")
	}

	// State-based risk
	switch mesh.MeshHealth.State {
	case "healthy":
		// No additional impact
	case "degraded":
		impact += 0.1
		if level == RiskLevelNone {
			level = RiskLevelLow
		}
		explanations = append(explanations, "mesh state is degraded")
	case "unstable":
		impact += 0.2
		if level == RiskLevelNone || level == RiskLevelLow {
			level = RiskLevelMedium
		}
		explanations = append(explanations, "mesh state is unstable")
	case "failed":
		impact += 0.3
		level = RiskLevelHigh
		explanations = append(explanations, "mesh state is failed")
	default:
		impact += 0.1
		if level == RiskLevelNone {
			level = RiskLevelLow
		}
		explanations = append(explanations, fmt.Sprintf("mesh state is %s", mesh.MeshHealth.State))
	}

	// Correlated failures increase risk
	if len(mesh.CorrelatedFailures) > 0 {
		impact += float64(len(mesh.CorrelatedFailures)) * 0.05
		explanations = append(explanations, fmt.Sprintf("%d correlated failure patterns detected", len(mesh.CorrelatedFailures)))
	}

	// Critical segments increase risk
	if len(mesh.MeshHealth.CriticalSegments) > 0 {
		impact += float64(len(mesh.MeshHealth.CriticalSegments)) * 0.1
		if level != RiskLevelCritical {
			level = RiskLevelHigh
		}
		explanations = append(explanations, fmt.Sprintf("%d critical segments present", len(mesh.MeshHealth.CriticalSegments)))
	}

	// Recovery blockers indicate systemic issues
	if len(mesh.MeshHealthExplanation.RecoveryBlockers) > 0 {
		impact += 0.1
		explanations = append(explanations, fmt.Sprintf("%d recovery blockers present", len(mesh.MeshHealthExplanation.RecoveryBlockers)))
	}

	return RiskFactorDetail{
		Category:    "mesh_health",
		Description: fmt.Sprintf("Mesh health: score=%d, state=%s", mesh.MeshHealth.Score, mesh.MeshHealth.State),
		Level:       level,
		Weight:      baseWeight,
		Impact:      minFloat(impact, 1.0),
		Likelihood:  0.8, // Mesh conditions are likely to affect outcome
		Mitigatable: true,
		Explanation: fmt.Sprintf("Mesh context: %s", formatExplanations(explanations)),
	}
}

// evaluateHistoricalRisk assesses risk based on historical action outcomes
func (rs *RiskScorer) evaluateHistoricalRisk(action control.ControlAction) RiskFactorDetail {
	baseWeight := 0.15

	if rs.database == nil {
		return RiskFactorDetail{
			Category:    "historical_performance",
			Description: "Historical action success/failure rates",
			Level:       RiskLevelNone,
			Weight:      baseWeight,
			Impact:      0.0,
			Likelihood:  0.0,
			Mitigatable: true,
			Explanation: "No database available; cannot assess historical action performance",
		}
	}

	// Check cache or fetch from database
	var history []db.ControlActionRecord
	cacheKey := action.ActionType + ":" + action.TargetTransport
	if cached, ok := rs.actionHistoryCache[cacheKey]; ok {
		history = cached
	} else {
		// Fetch last 24 hours of similar actions
		start := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
		records, err := rs.database.ControlActions(action.TargetTransport, action.ActionType, start, "", 50, 0)
		if err != nil {
			return RiskFactorDetail{
				Category:    "historical_performance",
				Description: "Historical action success/failure rates",
				Level:       RiskLevelLow,
				Weight:      baseWeight,
				Impact:      0.1,
				Likelihood:  0.5,
				Mitigatable: true,
				Explanation: fmt.Sprintf("Failed to fetch historical data: %v", err),
			}
		}
		history = records
		rs.actionHistoryCache[cacheKey] = records
	}

	if len(history) == 0 {
		return RiskFactorDetail{
			Category:    "historical_performance",
			Description: "Historical action success/failure rates",
			Level:       RiskLevelLow,
			Weight:      baseWeight,
			Impact:      0.1,
			Likelihood:  0.5,
			Mitigatable: true,
			Explanation: "No historical data for this action type; limited confidence in outcome prediction",
		}
	}

	// Calculate success rate
	successes := 0
	failures := 0
	for _, record := range history {
		switch record.Result {
		case control.ResultExecutedSuccessfully, control.ResultExecutedNoop:
			successes++
		case control.ResultFailedTransient, control.ResultFailedTerminal:
			failures++
		}
	}

	total := successes + failures
	if total == 0 {
		return RiskFactorDetail{
			Category:    "historical_performance",
			Description: "Historical action success/failure rates",
			Level:       RiskLevelLow,
			Weight:      baseWeight,
			Impact:      0.1,
			Likelihood:  0.5,
			Mitigatable: true,
			Explanation: fmt.Sprintf("Found %d historical actions but no completed outcomes", len(history)),
		}
	}

	successRate := float64(successes) / float64(total)
	impact := (1.0 - successRate) * 0.5 // Max 0.5 impact from history
	level := RiskLevelNone
	if impact > 0.3 {
		level = RiskLevelMedium
	} else if impact > 0.1 {
		level = RiskLevelLow
	}

	explanation := fmt.Sprintf("Historical success rate: %.0f%% (%d/%d actions)", successRate*100, successes, total)
	if successRate < 0.5 {
		explanation += "; poor historical performance increases risk"
		level = RiskLevelHigh
	} else if successRate > 0.8 {
		explanation += "; strong historical performance reduces risk"
	}

	return RiskFactorDetail{
		Category:    "historical_performance",
		Description: "Historical action success/failure rates",
		Level:       level,
		Weight:      baseWeight,
		Impact:      impact,
		Likelihood:  0.7, // Historical patterns are likely to repeat
		Mitigatable: true,
		Explanation: explanation,
	}
}

// evaluateDependencyRisk assesses risk based on confidence in dependencies
func (rs *RiskScorer) evaluateDependencyRisk(
	action control.ControlAction,
	mesh status.MeshDrilldown,
	runtimeTransports []transport.Health,
) RiskFactorDetail {
	baseWeight := 0.15
	impact := 0.0
	level := RiskLevelNone
	explanations := make([]string, 0)

	// Transport dependency check
	if action.TargetTransport != "" {
		transportFound := false
		for _, th := range runtimeTransports {
			if th.Name == action.TargetTransport {
				transportFound = true
				break
			}
		}
		if !transportFound {
			impact += 0.3
			level = RiskLevelMedium
			explanations = append(explanations, fmt.Sprintf("target transport %s not in runtime", action.TargetTransport))
		}
	}

	// Episode correlation check
	if action.EpisodeID != "" {
		// Check if episode is still active
		episodeActive := false
		for _, alert := range mesh.ActiveAlerts {
			if alert.EpisodeID == action.EpisodeID {
				episodeActive = true
				break
			}
		}
		if !episodeActive {
			impact += 0.1
			if level == RiskLevelNone {
				level = RiskLevelLow
			}
			explanations = append(explanations, "action episode not currently active; may be stale")
		} else {
			explanations = append(explanations, "action correlates with active episode")
		}
	}

	// Attribution confidence from mesh
	attributionConfidence := 0.5 // Default medium confidence
	if mesh.RootCauseAnalysis.Confidence == "high" {
		attributionConfidence = 0.9
	} else if mesh.RootCauseAnalysis.Confidence == "medium" {
		attributionConfidence = 0.7
	} else if mesh.RootCauseAnalysis.Confidence == "low" {
		attributionConfidence = 0.4
	}

	if attributionConfidence < 0.6 {
		impact += (0.6 - attributionConfidence) * 0.5
		if level == RiskLevelNone {
			level = RiskLevelLow
		}
		explanations = append(explanations, fmt.Sprintf("low attribution confidence (%s)", mesh.RootCauseAnalysis.Confidence))
	}

	// Component health dependencies
	controlHealth := rs.healthRegistry.GetComponent("control")
	if controlHealth.Health == selfobs.HealthFailing {
		impact += 0.2
		level = RiskLevelMedium
		explanations = append(explanations, "control component is failing")
	} else if controlHealth.Health == selfobs.HealthDegraded {
		impact += 0.1
		if level == RiskLevelNone {
			level = RiskLevelLow
		}
		explanations = append(explanations, "control component is degraded")
	}

	return RiskFactorDetail{
		Category:    "dependency_confidence",
		Description: "Confidence in action dependencies",
		Level:       level,
		Weight:      baseWeight,
		Impact:      minFloat(impact, 1.0),
		Likelihood:  0.6,
		Mitigatable: true,
		Explanation: fmt.Sprintf("Dependency status: %s", formatExplanations(explanations)),
	}
}

// evaluateDataFreshness assesses risk based on freshness of underlying data
func (rs *RiskScorer) evaluateDataFreshness(action control.ControlAction) RiskFactorDetail {
	baseWeight := 0.15
	impact := 0.0
	level := RiskLevelNone
	explanations := make([]string, 0)

	// Check component freshness
	components := []string{"ingest", "classify", "control"}
	for _, component := range components {
		marker := rs.freshnessTracker.GetMarker(component)
		if marker.IsStale() {
			impact += 0.1
			if level == RiskLevelNone {
				level = RiskLevelLow
			}
			explanations = append(explanations, fmt.Sprintf("%s data is stale (last update %v ago)", component, marker.Age()))
		}
	}

	// Check action evidence freshness
	if len(action.TriggerEvidence) == 0 {
		impact += 0.15
		if level == RiskLevelNone {
			level = RiskLevelLow
		}
		explanations = append(explanations, "no trigger evidence provided")
	}

	// Check database connectivity freshness
	if rs.database == nil {
		impact += 0.1
		if level == RiskLevelNone {
			level = RiskLevelLow
		}
		explanations = append(explanations, "database unavailable for evidence verification")
	}

	if len(explanations) == 0 {
		return RiskFactorDetail{
			Category:    "data_freshness",
			Description: "Freshness of underlying data sources",
			Level:       RiskLevelNone,
			Weight:      baseWeight,
			Impact:      0.0,
			Likelihood:  0.0,
			Mitigatable: true,
			Explanation: "All data sources are fresh",
		}
	}

	return RiskFactorDetail{
		Category:    "data_freshness",
		Description: "Freshness of underlying data sources",
		Level:       level,
		Weight:      baseWeight,
		Impact:      minFloat(impact, 1.0),
		Likelihood:  0.8,
		Mitigatable: true,
		Explanation: fmt.Sprintf("Freshness concerns: %s", formatExplanations(explanations)),
	}
}

// calculateBlastRadius estimates the impact scope of the action
func (rs *RiskScorer) calculateBlastRadius(mesh status.MeshDrilldown, action control.ControlAction) (float64, string) {
	// Convert mesh drilldown to priority items for existing blast radius function
	priorities := make([]models.PriorityItem, 0)

	// Add active alerts as priority items
	for _, alert := range mesh.ActiveAlerts {
		if alert.TransportName == action.TargetTransport || action.TargetTransport == "" {
			priority := models.PriorityItem{
				ID:       alert.ID,
				Category: "transport",
				Severity: alert.Severity,
				Title:    alert.Reason,
				Metadata: map[string]any{
					"resource_id":        alert.TransportName,
					"affected_transport": alert.TransportName,
				},
			}
			priorities = append(priorities, priority)
		}
	}

	// Convert nodes to models.Node
	nodes := make([]models.Node, 0)
	for _, nodeID := range mesh.MeshHealthExplanation.AffectedNodes {
		nodes = append(nodes, models.Node{
			NodeID: nodeID,
		})
	}

	// Use existing intelligence function
	return intelligence.EstimateBlastRadius(priorities, nodes)
}

// aggregateRiskScore combines all factors into an overall risk score
func (rs *RiskScorer) aggregateRiskScore(factors []RiskFactorDetail, blastRadius float64) RiskScore {
	if len(factors) == 0 {
		return RiskScore{
			Score:       0.5,
			Explanation: "No risk factors evaluated; default medium risk",
		}
	}

	totalWeight := 0.0
	weightedSum := 0.0
	explanations := make([]string, 0)

	for _, factor := range factors {
		totalWeight += factor.Weight
		weightedSum += factor.Weight * factor.Impact
		explanations = append(explanations, fmt.Sprintf("%s: %.2f", factor.Category, factor.Impact))
	}

	// Normalize by total weight
	normalizedScore := weightedSum / totalWeight

	// Adjust for blast radius
	blastAdjustment := blastRadius * 0.1 // Max 0.1 adjustment from blast radius
	finalScore := minFloat(normalizedScore+blastAdjustment, 1.0)

	return RiskScore{
		Score:       finalScore,
		Explanation: fmt.Sprintf("Weighted score %.2f with blast radius adjustment %.2f; factors: %s", normalizedScore, blastAdjustment, formatExplanations(explanations)),
	}
}

// determineRiskLevel categorizes the numeric risk score
func (rs *RiskScorer) determineRiskLevel(score float64, reality control.ActionReality, mesh status.MeshDrilldown) RiskLevel {
	// Hard constraints override score
	if reality.DenialCode == control.DenialIrreversible && !reality.Reversible {
		return RiskLevelCritical
	}
	if mesh.MeshHealth.State == "failed" && score > 0.3 {
		return RiskLevelHigh
	}

	// Score-based categorization
	switch {
	case score < 0.2:
		return RiskLevelLow
	case score < 0.4:
		return RiskLevelMedium
	case score < 0.7:
		return RiskLevelHigh
	default:
		return RiskLevelCritical
	}
}

// assessReversibility evaluates how reversible the action is
func (rs *RiskScorer) assessReversibility(action control.ControlAction, reality control.ActionReality) ReversibilityAssessment {
	assessment := ReversibilityAssessment{
		IsReversible:       reality.Reversible,
		ReversibilityLevel: intelligence.RevNone,
		ReversalConfidence: 0.5,
		Explanation:        "",
		SideEffects:        make([]string, 0),
	}

	if !reality.Reversible {
		assessment.ReversibilityLevel = intelligence.RevNone
		assessment.Explanation = "Action is irreversible according to reality matrix"
		return assessment
	}

	// Determine reversal action and level based on action type
	switch action.ActionType {
	case control.ActionRestartTransport:
		assessment.ReversibilityLevel = intelligence.RevHigh
		assessment.ReversalAction = "Automatic via reconnect loop"
		assessment.ReversalConfidence = 0.9
		assessment.TimeWindowSeconds = 300
		assessment.Explanation = "Transport restart is fully reversible through normal reconnect behavior"

	case control.ActionResubscribeTransport:
		assessment.ReversibilityLevel = intelligence.RevHigh
		assessment.ReversalAction = "Automatic via subscription loop"
		assessment.ReversalConfidence = 0.9
		assessment.TimeWindowSeconds = 180
		assessment.Explanation = "Resubscription is reversible through retry logic"

	case control.ActionBackoffIncrease:
		assessment.ReversibilityLevel = intelligence.RevHigh
		assessment.ReversalAction = control.ActionBackoffReset
		assessment.ReversalConfidence = 0.95
		assessment.TimeWindowSeconds = 600
		assessment.Explanation = "Backoff increase is explicitly reversible via backoff_reset"

	case control.ActionBackoffReset:
		assessment.ReversibilityLevel = intelligence.RevMedium
		assessment.ReversalAction = control.ActionBackoffIncrease
		assessment.ReversalConfidence = 0.8
		assessment.TimeWindowSeconds = 300
		assessment.Explanation = "Backoff reset can be undone by increasing backoff again"
		assessment.SideEffects = append(assessment.SideEffects, "May trigger rapid reconnection attempts")

	case control.ActionTriggerHealthRecheck:
		assessment.ReversibilityLevel = intelligence.RevHigh
		assessment.ReversalAction = "N/A (read-only operation)"
		assessment.ReversalConfidence = 1.0
		assessment.Explanation = "Health recheck is read-only with no state change to reverse"

	default:
		assessment.ReversibilityLevel = intelligence.RevLow
		assessment.ReversalAction = "Manual intervention required"
		assessment.ReversalConfidence = 0.3
		assessment.Explanation = "Reversibility not defined for this action type"
	}

	// Adjust for expiration-based reversibility
	if action.ExpiresAt != "" {
		expires, err := time.Parse(time.RFC3339, action.ExpiresAt)
		if err == nil {
			duration := int(time.Until(expires).Seconds())
			if duration > 0 {
				assessment.TimeWindowSeconds = duration
				assessment.Explanation += fmt.Sprintf("; expires in %d seconds", duration)
			}
		}
	}

	return assessment
}

// assessUncertainty identifies unknown factors affecting confidence
func (rs *RiskScorer) assessUncertainty(
	action control.ControlAction,
	mesh status.MeshDrilldown,
	runtimeTransports []transport.Health,
) UncertaintyAssessment {
	assessment := UncertaintyAssessment{
		UnknownFactors:      make([]string, 0),
		ConfidenceReduction: 0.0,
		Mitigations:         make([]string, 0),
		KnowledgeGaps:       make([]KnowledgeGap, 0),
		Explanation:         "",
	}

	// Check for unknown transport state
	if action.TargetTransport != "" {
		transportFound := false
		for _, th := range runtimeTransports {
			if th.Name == action.TargetTransport {
				transportFound = true
				break
			}
		}
		if !transportFound {
			assessment.UnknownFactors = append(assessment.UnknownFactors, "target_transport_state")
			assessment.ConfidenceReduction += 0.15
			assessment.KnowledgeGaps = append(assessment.KnowledgeGaps, KnowledgeGap{
				Domain:      "transport",
				Impact:      0.15,
				Mitigatable: false,
				Description: fmt.Sprintf("Current runtime state of transport %s is unknown", action.TargetTransport),
			})
		}
	}

	// Check for insufficient mesh evidence
	if mesh.RootCauseAnalysis.Confidence == "low" {
		assessment.UnknownFactors = append(assessment.UnknownFactors, "root_cause_confidence")
		assessment.ConfidenceReduction += 0.1
		assessment.KnowledgeGaps = append(assessment.KnowledgeGaps, KnowledgeGap{
			Domain:      "causality",
			Impact:      0.1,
			Mitigatable: true,
			Description: "Root cause analysis has low confidence; may misattribute problem",
		})
		assessment.Mitigations = append(assessment.Mitigations, "Collect additional diagnostic data before acting")
	}

	// Check for missing trigger evidence
	if len(action.TriggerEvidence) == 0 {
		assessment.UnknownFactors = append(assessment.UnknownFactors, "trigger_evidence")
		assessment.ConfidenceReduction += 0.1
		assessment.KnowledgeGaps = append(assessment.KnowledgeGaps, KnowledgeGap{
			Domain:      "evidence",
			Impact:      0.1,
			Mitigatable: true,
			Description: "No trigger evidence provided for action",
		})
		assessment.Mitigations = append(assessment.Mitigations, "Document trigger conditions and evidence")
	}

	// Check for stale data
	if rs.database == nil {
		assessment.UnknownFactors = append(assessment.UnknownFactors, "historical_context")
		assessment.ConfidenceReduction += 0.1
		assessment.KnowledgeGaps = append(assessment.KnowledgeGaps, KnowledgeGap{
			Domain:      "history",
			Impact:      0.1,
			Mitigatable: false,
			Description: "Cannot access historical action outcomes without database",
		})
	}

	// Check for actuator uncertainty
	realityByType := control.ActionRealityByType()
	if reality, ok := realityByType[action.ActionType]; ok && !reality.ActuatorExists {
		assessment.UnknownFactors = append(assessment.UnknownFactors, "actuator_implementation")
		assessment.ConfidenceReduction += 0.2
		assessment.KnowledgeGaps = append(assessment.KnowledgeGaps, KnowledgeGap{
			Domain:      "implementation",
			Impact:      0.2,
			Mitigatable: false,
			Description: "Actuator for this action type is not verified in this build",
		})
	}

	if len(assessment.UnknownFactors) == 0 {
		assessment.Explanation = "No significant uncertainties identified"
	} else {
		assessment.Explanation = fmt.Sprintf("Identified %d uncertainty factors reducing confidence by %.0f%%",
			len(assessment.UnknownFactors), assessment.ConfidenceReduction*100)
	}

	return assessment
}

// generatePrecautions recommends safety measures based on risk assessment
func (rs *RiskScorer) generatePrecautions(
	riskLevel RiskLevel,
	reversibility ReversibilityAssessment,
	action control.ControlAction,
	reality control.ActionReality,
) []string {
	precautions := make([]string, 0)

	switch riskLevel {
	case RiskLevelCritical:
		precautions = append(precautions, "REQUIRE operator approval before execution")
		precautions = append(precautions, "Document explicit rationale for override")
		precautions = append(precautions, "Establish rollback procedure")
		precautions = append(precautions, "Monitor mesh health during execution")

	case RiskLevelHigh:
		precautions = append(precautions, "RECOMMEND operator review before execution")
		precautions = append(precautions, "Verify reversibility mechanism is functional")
		precautions = append(precautions, "Ensure fallback transport is available")

	case RiskLevelMedium:
		precautions = append(precautions, "Monitor action outcome metrics")
		if !reality.Reversible {
			precautions = append(precautions, "Consider reversible alternative actions")
		}
	}

	// Add reversibility-specific precautions
	if reversibility.ReversibilityLevel == intelligence.RevNone {
		precautions = append(precautions, "Action is irreversible; verify necessity")
	} else if reversibility.ReversibilityLevel == intelligence.RevLow {
		precautions = append(precautions, "Limited reversibility; prepare manual recovery")
	}

	// Add time-window precaution for expiring actions
	if action.ExpiresAt != "" {
		precautions = append(precautions, "Action has expiration; ensure timely execution or cancellation")
	}

	return precautions
}

// Helper functions

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func formatExplanations(explanations []string) string {
	if len(explanations) == 0 {
		return "no factors"
	}
	if len(explanations) == 1 {
		return explanations[0]
	}
	result := explanations[0]
	for i := 1; i < len(explanations); i++ {
		result += "; " + explanations[i]
	}
	return result
}

// ValidateRiskEvaluation performs validation on risk evaluation results
func ValidateRiskEvaluation(result RiskEvaluationResult) []string {
	issues := make([]string, 0)

	if result.ActionType == "" {
		issues = append(issues, "missing action type")
	}

	if result.RiskScore.Score < 0 || result.RiskScore.Score > 1 {
		issues = append(issues, fmt.Sprintf("risk score %.2f out of bounds [0,1]", result.RiskScore.Score))
	}

	if result.RiskLevel == "" {
		issues = append(issues, "missing risk level")
	}

	validLevels := map[RiskLevel]bool{
		RiskLevelNone:     true,
		RiskLevelLow:      true,
		RiskLevelMedium:   true,
		RiskLevelHigh:     true,
		RiskLevelCritical: true,
	}
	if !validLevels[result.RiskLevel] {
		issues = append(issues, fmt.Sprintf("invalid risk level: %s", result.RiskLevel))
	}

	if len(result.ContributingFactors) == 0 {
		issues = append(issues, "no contributing factors provided")
	}

	if result.EvaluatedAt.IsZero() {
		issues = append(issues, "missing evaluation timestamp")
	}

	return issues
}

// RiskLevelAllowsAutomation determines if a given risk level permits automated execution
func RiskLevelAllowsAutomation(level RiskLevel, mode string) bool {
	switch level {
	case RiskLevelNone, RiskLevelLow:
		return mode == control.ModeGuardedAuto
	case RiskLevelMedium:
		return mode == control.ModeGuardedAuto
	case RiskLevelHigh:
		return false // Always requires operator review
	case RiskLevelCritical:
		return false // Never automated
	default:
		return false
	}
}
