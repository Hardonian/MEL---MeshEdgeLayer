// Package simulation provides integration with existing MEL systems.
// This file enables non-breaking enrichment of intelligence, control, and briefing
// systems with simulation capabilities.
package simulation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/intelligence"
	"github.com/mel-project/mel/internal/models"
)

// SimulationEnrichment holds simulation results attached to existing MEL types.
// It provides optional, non-breaking extension of core types.
type SimulationEnrichment struct {
	// SimulationID uniquely identifies the simulation run
	SimulationID string `json:"simulation_id,omitempty"`

	// Timestamp when enrichment was created
	Timestamp time.Time `json:"timestamp,omitempty"`

	// PredictedOutcome contains the simulation prediction
	PredictedOutcome *PredictedOutcome `json:"predicted_outcome,omitempty"`

	// RiskAssessment provides risk evaluation
	RiskAssessment *RiskAssessment `json:"risk_assessment,omitempty"`

	// SafeToAct provides the safety decision
	SafeToAct *SafeToActDecision `json:"safe_to_act,omitempty"`

	// Confidence is the overall simulation confidence
	Confidence float64 `json:"confidence,omitempty"`

	// Metadata contains additional enrichment data
	Metadata map[string]any `json:"metadata,omitempty"`
}

// EnrichedRecommendation extends intelligence.Recommendation with simulation data.
// It embeds the original recommendation and adds optional simulation enrichment.
type EnrichedRecommendation struct {
	intelligence.Recommendation

	// Simulation contains optional simulation results
	Simulation *SimulationEnrichment `json:"simulation,omitempty"`
}

// EnrichedControlDecision extends control.ControlDecision with simulation data.
// It embeds the original decision and adds optional predictive information.
type EnrichedControlDecision struct {
	control.ControlDecision

	// Simulation contains optional simulation results
	Simulation *SimulationEnrichment `json:"simulation,omitempty"`

	// PredictiveOutcome describes the predicted result if executed
	PredictiveOutcome *PredictedOutcome `json:"predictive_outcome,omitempty"`

	// AlternativeScenarios describes what-if scenarios
	AlternativeScenarios []OutcomeBranch `json:"alternative_scenarios,omitempty"`
}

// EnrichedOperatorBriefing extends models.OperatorBriefingDTO with simulation insights.
type EnrichedOperatorBriefing struct {
	models.OperatorBriefingDTO

	// SimulationInsights provides simulation-based analysis
	SimulationInsights *SimulationInsights `json:"simulation_insights,omitempty"`
}

// SimulationInsights contains simulation-derived briefing enhancements.
type SimulationInsights struct {
	// ActionSafetyMap indicates safety of each recommended action
	ActionSafetyMap map[string]SafetyLevel `json:"action_safety_map,omitempty"`

	// RiskDistribution summarizes risk across all recommendations
	RiskDistribution map[RiskLevel]int `json:"risk_distribution,omitempty"`

	// SafeToExecute lists actions deemed safe for immediate execution
	SafeToExecute []string `json:"safe_to_execute,omitempty"`

	// RequiresReview lists actions needing operator review
	RequiresReview []string `json:"requires_review,omitempty"`

	// PredictedOutcomes maps action codes to predicted results
	PredictedOutcomes map[string]PredictedOutcome `json:"predicted_outcomes,omitempty"`

	// BlastRadiusWarnings contains high-impact warnings
	BlastRadiusWarnings []BlastRadiusWarning `json:"blast_radius_warnings,omitempty"`

	// SimulationTimestamp when insights were generated
	SimulationTimestamp time.Time `json:"simulation_timestamp,omitempty"`
}

// BlastRadiusWarning highlights high-impact predictions.
type BlastRadiusWarning struct {
	ActionCode  string  `json:"action_code"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
}

// EnrichRecommendation upgrades an existing Recommendation with simulation results.
// If simulation results are nil, returns a copy of the original recommendation
// with nil simulation enrichment (existing code continues to work).
func EnrichRecommendation(rec intelligence.Recommendation, result *SimulationResult) EnrichedRecommendation {
	enriched := EnrichedRecommendation{
		Recommendation: rec,
	}

	if result == nil {
		return enriched
	}

	enriched.Simulation = &SimulationEnrichment{
		SimulationID:     result.SimulationID,
		Timestamp:        result.CompletedAt,
		PredictedOutcome: &result.PredictedOutcome,
		RiskAssessment:   &result.RiskAssessment,
		SafeToAct:        &result.SafeToAct,
		Confidence:       result.SafeToAct.Confidence,
		Metadata: map[string]any{
			"model_version": result.Metadata.ModelVersion,
			"duration_ms":   result.Metadata.Duration.Milliseconds(),
		},
	}

	// Upgrade risk level if simulation indicates higher risk
	if result.RiskAssessment.OverallRisk == RiskLevelHigh && rec.RiskLevel == intelligence.RiskLow {
		enriched.RiskLevel = intelligence.RiskHigh
	}
	if result.RiskAssessment.OverallRisk == RiskLevelCritical {
		enriched.RiskLevel = intelligence.RiskHigh
	}

	// Update confidence with simulation confidence
	if result.SafeToAct.Confidence > 0 {
		enriched.Confidence = (enriched.Confidence + result.SafeToAct.Confidence) / 2
	}

	// Add simulation guidance to rationale
	if result.SafeToAct.OperatorGuidance != "" {
		enriched.Rationale = enriched.Rationale + " [Simulation: " + result.SafeToAct.OperatorGuidance + "]"
	}

	// Update automation eligibility based on simulation
	if result.PolicyPreview.Allowed && result.SafeToAct.SafeToAct {
		enriched.CanAutomate = true
	}

	return enriched
}

// EnrichControlDecision upgrades a ControlDecision with simulation results.
// The original decision remains unchanged; simulation adds predictive context.
func EnrichControlDecision(decision control.ControlDecision, result *SimulationResult) EnrichedControlDecision {
	enriched := EnrichedControlDecision{
		ControlDecision: decision,
	}

	if result == nil {
		return enriched
	}

	enriched.Simulation = &SimulationEnrichment{
		SimulationID:     result.SimulationID,
		Timestamp:        result.CompletedAt,
		PredictedOutcome: &result.PredictedOutcome,
		RiskAssessment:   &result.RiskAssessment,
		SafeToAct:        &result.SafeToAct,
		Confidence:       result.SafeToAct.Confidence,
		Metadata: map[string]any{
			"model_version":      result.Metadata.ModelVersion,
			"blast_radius_score": result.BlastRadius.Score,
		},
	}

	enriched.PredictiveOutcome = &result.PredictedOutcome
	enriched.AlternativeScenarios = result.OutcomeBranches

	// Enhance denial reason with simulation context if denied
	if !decision.Allowed && result.PolicyPreview.DenialReason != "" {
		enriched.DenialReason = decision.DenialReason + " (Simulation: " + result.PolicyPreview.DenialReason + ")"
	}

	// Add safety checks from simulation
	if enriched.SafetyChecks == nil {
		enriched.SafetyChecks = make(map[string]any)
	}
	enriched.SafetyChecks["simulation_safe_to_act"] = result.SafeToAct.SafeToAct
	enriched.SafetyChecks["simulation_risk_level"] = result.RiskAssessment.OverallRisk
	enriched.SafetyChecks["simulation_confidence"] = result.SafeToAct.Confidence

	return enriched
}

// GenerateOperatorBriefing creates an enhanced briefing with simulation insights.
// It extends the standard briefing with predictive analysis of recommended actions.
func GenerateOperatorBriefing(
	briefing models.OperatorBriefingDTO,
	simulations map[string]*SimulationResult,
) EnrichedOperatorBriefing {
	enriched := EnrichedOperatorBriefing{
		OperatorBriefingDTO: briefing,
	}

	if len(simulations) == 0 {
		return enriched
	}

	insights := &SimulationInsights{
		ActionSafetyMap:     make(map[string]SafetyLevel),
		RiskDistribution:    make(map[RiskLevel]int),
		SafeToExecute:       []string{},
		RequiresReview:      []string{},
		PredictedOutcomes:   make(map[string]PredictedOutcome),
		BlastRadiusWarnings: []BlastRadiusWarning{},
		SimulationTimestamp: time.Now(),
	}

	for actionCode, result := range simulations {
		if result == nil {
			continue
		}

		insights.ActionSafetyMap[actionCode] = result.SafeToAct.Decision
		insights.RiskDistribution[result.RiskAssessment.OverallRisk]++
		insights.PredictedOutcomes[actionCode] = result.PredictedOutcome

		if result.SafeToAct.SafeToAct && result.PolicyPreview.Allowed {
			insights.SafeToExecute = append(insights.SafeToExecute, actionCode)
		} else {
			insights.RequiresReview = append(insights.RequiresReview, actionCode)
		}

		if result.BlastRadius.Score > 0.5 {
			insights.BlastRadiusWarnings = append(insights.BlastRadiusWarnings, BlastRadiusWarning{
				ActionCode:  actionCode,
				Severity:    "high",
				Description: result.BlastRadius.Description,
				Score:       result.BlastRadius.Score,
			})
		}
	}

	enriched.SimulationInsights = insights

	// Add simulation notes to uncertainty notes
	if len(insights.BlastRadiusWarnings) > 0 {
		briefing.UncertaintyNotes = append(
			briefing.UncertaintyNotes,
			fmt.Sprintf("Simulation detected %d high-impact actions requiring attention", len(insights.BlastRadiusWarnings)),
		)
	}

	if len(insights.SafeToExecute) > 0 {
		briefing.UncertaintyNotes = append(
			briefing.UncertaintyNotes,
			fmt.Sprintf("Simulation indicates %d actions are safe for automated execution", len(insights.SafeToExecute)),
		)
	}

	return enriched
}

// SimulationMiddleware wraps control plane evaluation with simulation.
// It provides an optional decorator pattern for integrating simulation
// into existing control flows without breaking changes.
type SimulationMiddleware struct {
	engine SimulationEngine
}

// NewSimulationMiddleware creates a middleware wrapper around a simulation engine.
func NewSimulationMiddleware(engine SimulationEngine) *SimulationMiddleware {
	return &SimulationMiddleware{engine: engine}
}

// WrapEvaluation enriches a control Evaluation with simulation results.
// Returns enriched decisions while preserving original evaluation structure.
func (m *SimulationMiddleware) WrapEvaluation(
	eval control.Evaluation,
	input SimulationInput,
) (control.Evaluation, []EnrichedControlDecision, error) {
	if m.engine == nil {
		return eval, nil, fmt.Errorf("simulation engine not initialized")
	}

	enrichedDecisions := make([]EnrichedControlDecision, 0, len(eval.Decisions))

	for _, decision := range eval.Decisions {
		// Create input for this specific decision
		simInput := input
		simInput.ProposedAction = decision.CandidateAction
		simInput.SimulationID = generateSimulationID(decision.CandidateAction)
		simInput.Timestamp = time.Now()

		// Run simulation if engine supports this action type
		if m.engine.SupportsAction(decision.CandidateAction.ActionType) {
			result, err := m.engine.Simulate(simInput)
			if err != nil {
				// Log error but continue without enrichment
				continue
			}
			enriched := EnrichControlDecision(decision, &result)
			enrichedDecisions = append(enrichedDecisions, enriched)
		} else {
			enriched := EnrichControlDecision(decision, nil)
			enrichedDecisions = append(enrichedDecisions, enriched)
		}
	}

	return eval, enrichedDecisions, nil
}

// WrapRecommendations enriches recommendations with simulation results.
func (m *SimulationMiddleware) WrapRecommendations(
	recs []intelligence.Recommendation,
	input SimulationInput,
) ([]EnrichedRecommendation, error) {
	if m.engine == nil {
		return nil, fmt.Errorf("simulation engine not initialized")
	}

	enriched := make([]EnrichedRecommendation, 0, len(recs))

	for _, rec := range recs {
		// Skip if no action type can be derived
		actionType := inferActionType(rec)
		if actionType == "" {
			enriched = append(enriched, EnrichRecommendation(rec, nil))
			continue
		}

		// Create synthetic control action for simulation
		candidate := control.ControlAction{
			ActionType: actionType,
			Reason:     rec.Rationale,
			Confidence: rec.Confidence,
		}

		simInput := input
		simInput.ProposedAction = candidate
		simInput.SimulationID = generateSimulationID(candidate)
		simInput.Timestamp = time.Now()

		if m.engine.SupportsAction(actionType) {
			result, err := m.engine.Simulate(simInput)
			if err != nil {
				enriched = append(enriched, EnrichRecommendation(rec, nil))
				continue
			}
			enriched = append(enriched, EnrichRecommendation(rec, &result))
		} else {
			enriched = append(enriched, EnrichRecommendation(rec, nil))
		}
	}

	return enriched, nil
}

// SimulationMiddlewareOption configures middleware behavior.
type SimulationMiddlewareOption func(*SimulationMiddleware)

// WithSimulationEngine sets a custom simulation engine.
func WithSimulationEngine(engine SimulationEngine) SimulationMiddlewareOption {
	return func(m *SimulationMiddleware) {
		m.engine = engine
	}
}

// Helper functions for type conversions

// ToIntelligenceRiskLevel converts simulation RiskLevel to intelligence risk constants.
func ToIntelligenceRiskLevel(level RiskLevel) string {
	switch level {
	case RiskLevelNone, RiskLevelLow:
		return intelligence.RiskLow
	case RiskLevelMedium:
		return intelligence.RiskMedium
	case RiskLevelHigh, RiskLevelCritical:
		return intelligence.RiskHigh
	default:
		return intelligence.RiskLow
	}
}

// ToIntelligenceReversibility converts rollback info to reversibility constants.
func ToIntelligenceReversibility(rollback RollbackInfo) string {
	if !rollback.Reversible {
		return intelligence.RevNone
	}
	if rollback.Automatic {
		return intelligence.RevHigh
	}
	return intelligence.RevMedium
}

// FromControlDecision creates a simulation input from a control decision.
func FromControlDecision(decision control.ControlDecision, baseInput SimulationInput) SimulationInput {
	input := baseInput
	input.ProposedAction = decision.CandidateAction
	input.SimulationID = generateSimulationID(decision.CandidateAction)
	input.Timestamp = time.Now()
	return input
}

// ToRiskLevel converts a string risk level to simulation RiskLevel.
func ToRiskLevel(level string) RiskLevel {
	switch level {
	case "low":
		return RiskLevelLow
	case "medium":
		return RiskLevelMedium
	case "high":
		return RiskLevelHigh
	case "critical":
		return RiskLevelCritical
	default:
		return RiskLevelLow
	}
}

// ToSafetyLevel converts a string safety assessment to SafetyLevel.
func ToSafetyLevel(level string) SafetyLevel {
	switch level {
	case "safe":
		return SafetyLevelSafe
	case "caution":
		return SafetyLevelCaution
	case "at_risk":
		return SafetyLevelAtRisk
	case "unsafe":
		return SafetyLevelUnsafe
	case "forbidden":
		return SafetyLevelForbidden
	default:
		return SafetyLevelSafe
	}
}

// internal helpers

func generateSimulationID(action control.ControlAction) string {
	h := sha256.New()
	h.Write([]byte(action.ActionType))
	h.Write([]byte(action.TargetTransport))
	h.Write([]byte(action.TargetSegment))
	h.Write([]byte(time.Now().Format(time.RFC3339Nano)))
	return "sim-" + hex.EncodeToString(h.Sum(nil))[:16]
}

func inferActionType(rec intelligence.Recommendation) string {
	// Map recommendation codes to action types
	switch rec.Code {
	case "reconnect_transport":
		return control.ActionRestartTransport
	case "resubscribe_transport":
		return control.ActionResubscribeTransport
	case "increase_backoff":
		return control.ActionBackoffIncrease
	case "reset_backoff":
		return control.ActionBackoffReset
	case "trigger_health_check":
		return control.ActionTriggerHealthRecheck
	default:
		// Try to infer from code prefix
		if len(rec.Code) > 10 && rec.Code[:10] == "transport_" {
			return control.ActionRestartTransport
		}
		return ""
	}
}

// ExtractSimulationResult retrieves simulation data from an enriched recommendation.
// Returns nil if no simulation data is present.
func ExtractSimulationResult(rec EnrichedRecommendation) *SimulationResult {
	if rec.Simulation == nil {
		return nil
	}

	return &SimulationResult{
		SimulationID: rec.Simulation.SimulationID,
		CompletedAt:  rec.Simulation.Timestamp,
		PredictedOutcome: func() PredictedOutcome {
			if rec.Simulation.PredictedOutcome != nil {
				return *rec.Simulation.PredictedOutcome
			}
			return PredictedOutcome{}
		}(),
		RiskAssessment: func() RiskAssessment {
			if rec.Simulation.RiskAssessment != nil {
				return *rec.Simulation.RiskAssessment
			}
			return RiskAssessment{}
		}(),
		SafeToAct: func() SafeToActDecision {
			if rec.Simulation.SafeToAct != nil {
				return *rec.Simulation.SafeToAct
			}
			return SafeToActDecision{}
		}(),
	}
}

// ExtractSimulationResultFromDecision retrieves simulation data from an enriched decision.
func ExtractSimulationResultFromDecision(decision EnrichedControlDecision) *SimulationResult {
	if decision.Simulation == nil {
		return nil
	}

	result := &SimulationResult{
		SimulationID:     decision.Simulation.SimulationID,
		CompletedAt:      decision.Simulation.Timestamp,
		PredictedOutcome: *decision.Simulation.PredictedOutcome,
		RiskAssessment:   *decision.Simulation.RiskAssessment,
		SafeToAct:        *decision.Simulation.SafeToAct,
	}

	if decision.PredictiveOutcome != nil {
		result.PredictedOutcome = *decision.PredictiveOutcome
	}

	if len(decision.AlternativeScenarios) > 0 {
		result.OutcomeBranches = decision.AlternativeScenarios
	}

	return result
}

// HasSimulationData checks if a recommendation has been enriched with simulation data.
func HasSimulationData(rec EnrichedRecommendation) bool {
	return rec.Simulation != nil
}

// HasSimulationDataForDecision checks if a decision has been enriched with simulation data.
func HasSimulationDataForDecision(decision EnrichedControlDecision) bool {
	return decision.Simulation != nil
}

// GetSafeRecommendations filters enriched recommendations to only those deemed safe by simulation.
func GetSafeRecommendations(recs []EnrichedRecommendation) []EnrichedRecommendation {
	var safe []EnrichedRecommendation
	for _, rec := range recs {
		if rec.Simulation != nil && rec.Simulation.SafeToAct != nil && rec.Simulation.SafeToAct.SafeToAct {
			safe = append(safe, rec)
		}
	}
	return safe
}

// GetAutomatableDecisions filters enriched decisions to only those safe for automation.
func GetAutomatableDecisions(decisions []EnrichedControlDecision) []EnrichedControlDecision {
	var automatable []EnrichedControlDecision
	for _, dec := range decisions {
		if dec.Simulation != nil && dec.Simulation.SafeToAct != nil &&
			dec.Simulation.SafeToAct.SafeToAct && dec.Simulation.PredictedOutcome != nil {
			automatable = append(automatable, dec)
		}
	}
	return automatable
}
