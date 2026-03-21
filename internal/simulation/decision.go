package simulation

import (
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/control"
)

// SafeToActStatus provides clear categorical decision guidance for operators.
type SafeToActStatus string

const (
	// SafeToActStatusSafe indicates the action can proceed without conditions.
	SafeToActStatusSafe SafeToActStatus = "SAFE_TO_ACT"

	// SafeToActStatusConditional indicates the action is safe if specific conditions are met.
	SafeToActStatusConditional SafeToActStatus = "SAFE_AFTER_CONDITION"

	// SafeToActStatusUnsafe indicates the action should not proceed.
	SafeToActStatusUnsafe SafeToActStatus = "NOT_SAFE"

	// SafeToActStatusInsufficientData indicates more information is needed.
	SafeToActStatusInsufficientData SafeToActStatus = "INSUFFICIENT_DATA"
)

// SafeToActEvaluator provides the final decision layer for control actions.
// It synthesizes all simulation outputs (risk, conflicts, policy, blast radius,
// outcomes) into clear, actionable operator guidance.
type SafeToActEvaluator struct {
	// Minimum confidence threshold for automated decisions
	minConfidence float64

	// Risk levels that require operator acknowledgment
	riskLevelsRequiringAck []RiskLevel

	// Time after which recommendations expire
	recommendationTTL time.Duration
}

// EvaluationResult contains the comprehensive safe-to-act decision with
// full operator guidance, prerequisites, next steps, and alternatives.
type EvaluationResult struct {
	// Status provides the categorical decision
	Status SafeToActStatus `json:"status"`

	// SafeToAct indicates if the action is deemed safe to execute
	SafeToAct bool `json:"safe_to_act"`

	// Decision provides the legacy safety level for compatibility
	Decision SafetyLevel `json:"decision"`

	// PrimaryReason is the main factor in this decision
	PrimaryReason string `json:"primary_reason"`

	// SupportingReasons lists additional factors considered
	SupportingReasons []string `json:"supporting_reasons,omitempty"`

	// OperatorGuidance provides actionable guidance
	OperatorGuidance string `json:"operator_guidance"`

	// RequiresAcknowledgment indicates if operator acknowledgment is required
	RequiresAcknowledgment bool `json:"requires_acknowledgment"`

	// MissingPrerequisites lists conditions that must be satisfied
	MissingPrerequisites []Prerequisite `json:"missing_prerequisites,omitempty"`

	// NextSteps provides recommended actions to proceed safely
	NextSteps []NextStep `json:"next_steps,omitempty"`

	// AlternativeActions suggests safer alternatives if primary is not safe
	AlternativeActions []AlternativeAction `json:"alternative_actions,omitempty"`

	// RiskSummary provides a summary of identified risks
	RiskSummary RiskSummary `json:"risk_summary"`

	// Confidence in this decision (0.0-1.0)
	Confidence float64 `json:"confidence"`

	// EvaluatedAt timestamp
	EvaluatedAt time.Time `json:"evaluated_at"`

	// ExpiresAt indicates when this evaluation should be refreshed
	ExpiresAt time.Time `json:"expires_at"`
}

// RiskSummary provides a condensed view of risk assessment results.
type RiskSummary struct {
	OverallRisk       RiskLevel `json:"overall_risk"`
	RiskScore         float64   `json:"risk_score"`
	ConflictCount     int       `json:"conflict_count"`
	CriticalConflicts int       `json:"critical_conflicts"`
	PolicyAllows      bool      `json:"policy_allows"`
	BlastRadiusScore  float64   `json:"blast_radius_score"`
	Reversibility     string    `json:"reversibility"`
	DataFreshness     string    `json:"data_freshness"`
}

// NextStep provides a specific actionable recommendation.
type NextStep struct {
	// Priority indicates the order in which steps should be taken (1 = highest)
	Priority int `json:"priority"`

	// Description of the step
	Description string `json:"description"`

	// ActionType indicates what kind of action this is
	ActionType string `json:"action_type"` // "verify", "wait", "resolve", "escalate", "proceed"

	// EstimatedTime to complete this step (0 if immediate)
	EstimatedTime time.Duration `json:"estimated_time,omitempty"`

	// Blocking indicates if this step must complete before proceeding
	Blocking bool `json:"blocking"`

	// Condition describes what condition this step will satisfy
	Condition string `json:"condition,omitempty"`
}

// AlternativeAction represents a safer alternative to the proposed action.
type AlternativeAction struct {
	// ActionType is the type of alternative action
	ActionType string `json:"action_type"`

	// Description explains why this is an alternative
	Description string `json:"description"`

	// RiskReduction describes how this reduces risk compared to original
	RiskReduction string `json:"risk_reduction"`

	// ExpectedEffectiveness (0.0-1.0) of achieving similar outcome
	ExpectedEffectiveness float64 `json:"expected_effectiveness"`

	// WhyRecommended explains the rationale
	WhyRecommended string `json:"why_recommended"`
}

// NewSafeToActEvaluator creates a new evaluator with default settings.
func NewSafeToActEvaluator() *SafeToActEvaluator {
	return &SafeToActEvaluator{
		minConfidence: 0.6,
		riskLevelsRequiringAck: []RiskLevel{
			RiskLevelHigh,
			RiskLevelCritical,
		},
		recommendationTTL: 5 * time.Minute,
	}
}

// NewSafeToActEvaluatorWithOptions creates an evaluator with custom settings.
func NewSafeToActEvaluatorWithOptions(
	minConfidence float64,
	riskLevelsRequiringAck []RiskLevel,
	recommendationTTL time.Duration,
) *SafeToActEvaluator {
	if minConfidence < 0 || minConfidence > 1 {
		minConfidence = 0.6
	}
	if recommendationTTL <= 0 {
		recommendationTTL = 5 * time.Minute
	}
	return &SafeToActEvaluator{
		minConfidence:          minConfidence,
		riskLevelsRequiringAck: riskLevelsRequiringAck,
		recommendationTTL:      recommendationTTL,
	}
}

// Evaluate performs the comprehensive safe-to-act analysis using all simulation
// components' outputs to provide clear operator guidance.
//
// The evaluation considers:
//   - Risk assessment (overall risk level and factors)
//   - Conflict reports (active conflicts and their severity)
//   - Policy preview (admission result and guardrails)
//   - Blast radius prediction (scope of impact)
//   - Outcome branches (best/expected/worst case scenarios)
//   - Predicted outcome (success probability and side effects)
func (e *SafeToActEvaluator) Evaluate(
	action control.ControlAction,
	riskAssessment RiskAssessment,
	conflicts []ConflictReport,
	policyPreview PolicyPreview,
	blastRadius BlastRadiusPrediction,
	outcomeBranches []OutcomeBranch,
	predictedOutcome PredictedOutcome,
) EvaluationResult {
	evaluatedAt := time.Now().UTC()
	expiresAt := evaluatedAt.Add(e.recommendationTTL)

	// Build risk summary
	riskSummary := e.buildRiskSummary(riskAssessment, conflicts, policyPreview, blastRadius)

	// Determine primary decision factors
	primaryReason, supportingReasons := e.determineReasons(
		action, riskAssessment, conflicts, policyPreview, blastRadius, predictedOutcome,
	)

	// Determine decision status
	status, safeToAct, decision := e.determineStatus(
		action, riskAssessment, conflicts, policyPreview, blastRadius, predictedOutcome,
	)

	// Build operator guidance
	guidance := e.buildOperatorGuidance(status, action, riskAssessment, conflicts, policyPreview)

	// Identify missing prerequisites
	missingPrereqs := e.identifyMissingPrerequisites(
		action, riskAssessment, conflicts, policyPreview, blastRadius,
	)

	// Generate next steps
	nextSteps := e.generateNextSteps(
		status, action, riskAssessment, conflicts, policyPreview, missingPrereqs,
	)

	// Generate alternative actions
	alternatives := e.generateAlternatives(action, status, riskAssessment, conflicts)

	// Determine if acknowledgment is required
	requiresAck := e.requiresAcknowledgment(riskAssessment, conflicts, policyPreview)

	// Calculate overall confidence
	confidence := e.calculateConfidence(riskAssessment, conflicts, policyPreview, blastRadius)

	return EvaluationResult{
		Status:                 status,
		SafeToAct:              safeToAct,
		Decision:               decision,
		PrimaryReason:          primaryReason,
		SupportingReasons:      supportingReasons,
		OperatorGuidance:       guidance,
		RequiresAcknowledgment: requiresAck,
		MissingPrerequisites:   missingPrereqs,
		NextSteps:              nextSteps,
		AlternativeActions:     alternatives,
		RiskSummary:            riskSummary,
		Confidence:             confidence,
		EvaluatedAt:            evaluatedAt,
		ExpiresAt:              expiresAt,
	}
}

// buildRiskSummary creates a condensed summary of all risk factors.
func (e *SafeToActEvaluator) buildRiskSummary(
	riskAssessment RiskAssessment,
	conflicts []ConflictReport,
	policyPreview PolicyPreview,
	blastRadius BlastRadiusPrediction,
) RiskSummary {
	criticalConflicts := 0
	for _, c := range conflicts {
		if c.Severity == ConflictSeverityCritical || c.Severity == ConflictSeverityMajor {
			criticalConflicts++
		}
	}

	reversibility := "unknown"
	for _, factor := range riskAssessment.RiskFactors {
		if factor.Category == "reversibility" || strings.Contains(factor.Description, "reversible") {
			if factor.Mitigatable {
				reversibility = "reversible"
			} else {
				reversibility = "irreversible"
			}
			break
		}
	}
	if reversibility == "unknown" {
		reversibility = "reversible" // Default assumption
	}

	dataFreshness := "fresh"
	for _, factor := range riskAssessment.RiskFactors {
		if factor.Category == "data_freshness" {
			if factor.Level != RiskLevelNone {
				dataFreshness = "stale"
			}
			break
		}
	}

	return RiskSummary{
		OverallRisk:       riskAssessment.OverallRisk,
		RiskScore:         e.extractRiskScore(riskAssessment),
		ConflictCount:     len(conflicts),
		CriticalConflicts: criticalConflicts,
		PolicyAllows:      policyPreview.Allowed,
		BlastRadiusScore:  blastRadius.Score,
		Reversibility:     reversibility,
		DataFreshness:     dataFreshness,
	}
}

// determineReasons identifies the primary and supporting reasons for the decision.
func (e *SafeToActEvaluator) determineReasons(
	action control.ControlAction,
	riskAssessment RiskAssessment,
	conflicts []ConflictReport,
	policyPreview PolicyPreview,
	blastRadius BlastRadiusPrediction,
	predictedOutcome PredictedOutcome,
) (string, []string) {
	var primary string
	supporting := []string{}

	// Check policy first - it's a hard gate
	if !policyPreview.Allowed {
		primary = fmt.Sprintf("Policy denies action: %s", policyPreview.DenialReason)
		if policyPreview.DenialCode != "" {
			supporting = append(supporting, fmt.Sprintf("Denial code: %s", policyPreview.DenialCode))
		}
	} else if policyPreview.Result == AdmissionAdvisory {
		primary = "Action is advisory-only and will not be executed"
		supporting = append(supporting, "Current control mode prevents execution")
	}

	// Check for critical conflicts
	criticalCount := 0
	for _, c := range conflicts {
		if c.Severity == ConflictSeverityCritical {
			criticalCount++
		}
	}
	if criticalCount > 0 {
		if primary == "" {
			primary = fmt.Sprintf("Found %d critical conflict(s) that must be resolved", criticalCount)
		} else {
			supporting = append(supporting, fmt.Sprintf("%d critical conflict(s) present", criticalCount))
		}
	}

	// Check risk level
	if riskAssessment.OverallRisk == RiskLevelCritical {
		if primary == "" {
			primary = "Critical risk level detected"
		}
		supporting = append(supporting, fmt.Sprintf("Risk factors: %d", len(riskAssessment.RiskFactors)))
	} else if riskAssessment.OverallRisk == RiskLevelHigh {
		if primary == "" {
			primary = "High risk level requires careful consideration"
		}
	}

	// Check blast radius
	if blastRadius.Score > 0.7 {
		if primary == "" {
			primary = fmt.Sprintf("High blast radius (%.0f%%) - systemic impact possible", blastRadius.Score*100)
		} else {
			supporting = append(supporting, fmt.Sprintf("Blast radius: %.0f%%", blastRadius.Score*100))
		}
	}

	// Check success probability
	if predictedOutcome.SuccessProbability < 0.5 {
		if primary == "" {
			primary = fmt.Sprintf("Low success probability (%.0f%%)", predictedOutcome.SuccessProbability*100)
		} else {
			supporting = append(supporting, fmt.Sprintf("Success probability: %.0f%%", predictedOutcome.SuccessProbability*100))
		}
	}

	// If nothing critical found, use a positive reason
	if primary == "" {
		primary = "All safety checks passed"
		if riskAssessment.OverallRisk != RiskLevelNone {
			supporting = append(supporting, fmt.Sprintf("Risk level: %s", riskAssessment.OverallRisk))
		}
		if len(conflicts) > 0 {
			supporting = append(supporting, fmt.Sprintf("Minor conflicts: %d", len(conflicts)))
		}
	}

	return primary, supporting
}

// determineStatus decides the final safe-to-act status.
func (e *SafeToActEvaluator) determineStatus(
	action control.ControlAction,
	riskAssessment RiskAssessment,
	conflicts []ConflictReport,
	policyPreview PolicyPreview,
	blastRadius BlastRadiusPrediction,
	predictedOutcome PredictedOutcome,
) (SafeToActStatus, bool, SafetyLevel) {

	// Check for insufficient data
	if e.hasInsufficientData(riskAssessment, blastRadius, policyPreview) {
		return SafeToActStatusInsufficientData, false, SafetyLevelUnsafe
	}

	// Policy is a hard gate
	if policyPreview.Result == AdmissionDenied {
		return SafeToActStatusUnsafe, false, SafetyLevelForbidden
	}

	// Check for critical conflicts
	for _, c := range conflicts {
		if c.Severity == ConflictSeverityCritical {
			return SafeToActStatusUnsafe, false, SafetyLevelUnsafe
		}
	}

	// Critical risk level
	if riskAssessment.OverallRisk == RiskLevelCritical {
		return SafeToActStatusUnsafe, false, SafetyLevelUnsafe
	}

	// Check for advisory-only mode
	if policyPreview.Result == AdmissionAdvisory {
		return SafeToActStatusConditional, false, SafetyLevelCaution
	}

	// High risk requires conditions
	if riskAssessment.OverallRisk == RiskLevelHigh {
		return SafeToActStatusConditional, false, SafetyLevelAtRisk
	}

	// Major conflicts require conditions
	hasMajorConflicts := false
	for _, c := range conflicts {
		if c.Severity == ConflictSeverityMajor || c.Severity == ConflictSeverityModerate {
			hasMajorConflicts = true
			break
		}
	}
	if hasMajorConflicts {
		return SafeToActStatusConditional, false, SafetyLevelCaution
	}

	// Low success probability
	if predictedOutcome.SuccessProbability < 0.7 {
		return SafeToActStatusConditional, false, SafetyLevelCaution
	}

	// High blast radius
	if blastRadius.Score > 0.5 {
		return SafeToActStatusConditional, false, SafetyLevelCaution
	}

	// Check if truly safe
	if policyPreview.Allowed &&
		riskAssessment.SafetyLevel != SafetyLevelUnsafe &&
		riskAssessment.OverallRisk != RiskLevelHigh &&
		riskAssessment.OverallRisk != RiskLevelCritical &&
		!hasMajorConflicts &&
		predictedOutcome.SuccessProbability >= 0.8 {
		return SafeToActStatusSafe, true, SafetyLevelSafe
	}

	// Default to conditional for medium risk
	return SafeToActStatusConditional, false, SafetyLevelCaution
}

// hasInsufficientData checks if we lack enough information to make a decision.
func (e *SafeToActEvaluator) hasInsufficientData(
	riskAssessment RiskAssessment,
	blastRadius BlastRadiusPrediction,
	policyPreview PolicyPreview,
) bool {
	// No risk factors means we didn't get a proper assessment
	if len(riskAssessment.RiskFactors) == 0 && riskAssessment.OverallRisk == "" {
		return true
	}

	// Very low confidence in blast radius
	if blastRadius.Confidence < 0.3 {
		return true
	}

	// Unknown policy mode
	if policyPreview.Mode == "" {
		return true
	}

	return false
}

// buildOperatorGuidance creates clear, actionable guidance text.
func (e *SafeToActEvaluator) buildOperatorGuidance(
	status SafeToActStatus,
	action control.ControlAction,
	riskAssessment RiskAssessment,
	conflicts []ConflictReport,
	policyPreview PolicyPreview,
) string {
	var parts []string

	switch status {
	case SafeToActStatusSafe:
		parts = append(parts, fmt.Sprintf("✓ Action '%s' on '%s' is SAFE to execute.",
			action.ActionType, e.targetString(action)))
		parts = append(parts, "All safety checks have passed.")
		if riskAssessment.OverallRisk != RiskLevelNone {
			parts = append(parts, fmt.Sprintf("Note: %s risk level - monitor outcome.", riskAssessment.OverallRisk))
		}

	case SafeToActStatusConditional:
		parts = append(parts, fmt.Sprintf("⚠ Action '%s' on '%s' can proceed AFTER conditions are met.",
			action.ActionType, e.targetString(action)))

		if !policyPreview.Allowed {
			parts = append(parts, fmt.Sprintf("Policy restriction: %s", policyPreview.DenialReason))
		}

		if len(conflicts) > 0 {
			parts = append(parts, fmt.Sprintf("Resolve %d conflict(s) before proceeding.", len(conflicts)))
		}

		if riskAssessment.OverallRisk == RiskLevelHigh {
			parts = append(parts, "HIGH RISK: Operator acknowledgment required.")
		}

	case SafeToActStatusUnsafe:
		parts = append(parts, fmt.Sprintf("✗ Action '%s' on '%s' is NOT SAFE to execute.",
			action.ActionType, e.targetString(action)))

		if policyPreview.Result == AdmissionDenied {
			parts = append(parts, fmt.Sprintf("Blocked by policy: %s", policyPreview.DenialReason))
		} else if riskAssessment.OverallRisk == RiskLevelCritical {
			parts = append(parts, "CRITICAL RISK: This action poses severe danger to system stability.")
		} else {
			parts = append(parts, "The action poses unacceptable risk given current conditions.")
		}

		parts = append(parts, "See Alternative Actions for safer options.")

	case SafeToActStatusInsufficientData:
		parts = append(parts, fmt.Sprintf("? Action '%s' on '%s' cannot be evaluated due to insufficient data.",
			action.ActionType, e.targetString(action)))
		parts = append(parts, "Collect additional diagnostic information before proceeding.")
	}

	return strings.Join(parts, " ")
}

// identifyMissingPrerequisites lists all conditions that must be satisfied.
func (e *SafeToActEvaluator) identifyMissingPrerequisites(
	action control.ControlAction,
	riskAssessment RiskAssessment,
	conflicts []ConflictReport,
	policyPreview PolicyPreview,
	blastRadius BlastRadiusPrediction,
) []Prerequisite {
	prereqs := []Prerequisite{}

	// Policy prerequisites
	if !policyPreview.Allowed {
		prereqs = append(prereqs, Prerequisite{
			Code:               "policy_allows",
			Description:        fmt.Sprintf("Policy must allow action: %s", policyPreview.DenialReason),
			CurrentlySatisfied: false,
			SuggestedAction:    e.getPolicyResolution(policyPreview),
		})
	}

	// Conflict resolution prerequisites
	for _, conflict := range conflicts {
		if conflict.Severity == ConflictSeverityCritical || conflict.Severity == ConflictSeverityMajor {
			prereqs = append(prereqs, Prerequisite{
				Code:               fmt.Sprintf("resolve_conflict_%s", conflict.ConflictID),
				Description:        fmt.Sprintf("Resolve conflict: %s", conflict.Description),
				CurrentlySatisfied: false,
				SuggestedAction:    conflict.Resolution,
			})
		}
	}

	// Risk mitigation prerequisites
	for _, factor := range riskAssessment.RiskFactors {
		if factor.Level == RiskLevelHigh || factor.Level == RiskLevelCritical {
			prereqs = append(prereqs, Prerequisite{
				Code:               fmt.Sprintf("mitigate_%s", factor.Category),
				Description:        fmt.Sprintf("Mitigate %s risk: %s", factor.Category, factor.Description),
				CurrentlySatisfied: false,
				SuggestedAction:    e.getMitigationSuggestion(factor),
			})
		}
	}

	// Confidence prerequisites
	if riskAssessment.Confidence < e.minConfidence {
		prereqs = append(prereqs, Prerequisite{
			Code: "sufficient_confidence",
			Description: fmt.Sprintf("Increase confidence from %.0f%% to at least %.0f%%",
				riskAssessment.Confidence*100, e.minConfidence*100),
			CurrentlySatisfied: false,
			SuggestedAction:    "Gather more evidence or wait for additional telemetry",
		})
	}

	// Blast radius prerequisites
	if blastRadius.Score > 0.6 {
		prereqs = append(prereqs, Prerequisite{
			Code:               "blast_radius_acceptable",
			Description:        fmt.Sprintf("High blast radius (%.0f%%) must be acceptable to operators", blastRadius.Score*100),
			CurrentlySatisfied: false,
			SuggestedAction:    "Verify impact scope and confirm acceptance of affected transports",
		})
	}

	return prereqs
}

// generateNextSteps creates a prioritized list of recommended actions.
func (e *SafeToActEvaluator) generateNextSteps(
	status SafeToActStatus,
	action control.ControlAction,
	riskAssessment RiskAssessment,
	conflicts []ConflictReport,
	policyPreview PolicyPreview,
	missingPrereqs []Prerequisite,
) []NextStep {
	steps := []NextStep{}
	priority := 1

	switch status {
	case SafeToActStatusSafe:
		steps = append(steps, NextStep{
			Priority:      priority,
			Description:   "Execute the action with standard monitoring",
			ActionType:    "proceed",
			EstimatedTime: 0,
			Blocking:      false,
			Condition:     "Action is safe to execute",
		})

	case SafeToActStatusConditional:
		// First, address any policy restrictions
		if !policyPreview.Allowed {
			steps = append(steps, NextStep{
				Priority:    priority,
				Description: fmt.Sprintf("Address policy restriction: %s", policyPreview.DenialReason),
				ActionType:  "resolve",
				Blocking:    true,
				Condition:   "Policy allows action",
			})
			priority++
		}

		// Then resolve critical conflicts
		for _, conflict := range conflicts {
			if conflict.Severity == ConflictSeverityCritical || conflict.Severity == ConflictSeverityMajor {
				steps = append(steps, NextStep{
					Priority:    priority,
					Description: fmt.Sprintf("Resolve conflict: %s", conflict.Description),
					ActionType:  "resolve",
					Blocking:    true,
					Condition:   fmt.Sprintf("Conflict %s resolved", conflict.ConflictID),
				})
				priority++
			}
		}

		// Add acknowledgment step for high risk
		if riskAssessment.OverallRisk == RiskLevelHigh {
			steps = append(steps, NextStep{
				Priority:    priority,
				Description: "Acknowledge high risk and confirm proceeding",
				ActionType:  "verify",
				Blocking:    true,
				Condition:   "Operator acknowledged high risk",
			})
			priority++
		}

		// Wait for confidence if needed
		if riskAssessment.Confidence < e.minConfidence {
			steps = append(steps, NextStep{
				Priority:      priority,
				Description:   "Wait for additional telemetry to increase confidence",
				ActionType:    "wait",
				EstimatedTime: 30 * time.Second,
				Blocking:      true,
				Condition:     fmt.Sprintf("Confidence reaches %.0f%%", e.minConfidence*100),
			})
			priority++
		}

		// Finally, proceed
		steps = append(steps, NextStep{
			Priority:      priority,
			Description:   "Execute the action with enhanced monitoring",
			ActionType:    "proceed",
			EstimatedTime: 0,
			Blocking:      false,
			Condition:     "All prerequisites satisfied",
		})

	case SafeToActStatusUnsafe:
		steps = append(steps, NextStep{
			Priority:    1,
			Description: "DO NOT EXECUTE this action in current conditions",
			ActionType:  "escalate",
			Blocking:    true,
			Condition:   "Unsafe conditions resolved",
		})

		if policyPreview.Result == AdmissionDenied {
			steps = append(steps, NextStep{
				Priority:    2,
				Description: fmt.Sprintf("Review policy denial: %s", policyPreview.DenialReason),
				ActionType:  "escalate",
				Blocking:    true,
				Condition:   "Policy reviewed by operator",
			})
		}

		// Suggest reviewing alternatives
		steps = append(steps, NextStep{
			Priority:    3,
			Description: "Review Alternative Actions for safer options",
			ActionType:  "verify",
			Blocking:    false,
			Condition:   "Alternative selected or original action approved",
		})

	case SafeToActStatusInsufficientData:
		steps = append(steps, NextStep{
			Priority:    1,
			Description: "Collect additional diagnostic data",
			ActionType:  "verify",
			Blocking:    true,
			Condition:   "Sufficient data available",
		})
		steps = append(steps, NextStep{
			Priority:    2,
			Description: "Re-run simulation with updated data",
			ActionType:  "verify",
			Blocking:    true,
			Condition:   "Simulation complete with sufficient confidence",
		})
	}

	return steps
}

// generateAlternatives suggests safer actions when the proposed one is not safe.
func (e *SafeToActEvaluator) generateAlternatives(
	action control.ControlAction,
	status SafeToActStatus,
	riskAssessment RiskAssessment,
	conflicts []ConflictReport,
) []AlternativeAction {
	// Only generate alternatives if not safe
	if status == SafeToActStatusSafe {
		return nil
	}

	alternatives := []AlternativeAction{}

	// Based on action type, suggest lower-risk alternatives
	switch action.ActionType {
	case control.ActionRestartTransport:
		// Suggest less disruptive options before restart
		alternatives = append(alternatives, AlternativeAction{
			ActionType:            control.ActionTriggerHealthRecheck,
			Description:           "Trigger health recheck before restart",
			RiskReduction:         "Read-only operation with no state changes",
			ExpectedEffectiveness: 0.3,
			WhyRecommended:        "May resolve transient issues without disrupting connectivity",
		})
		alternatives = append(alternatives, AlternativeAction{
			ActionType:            control.ActionResubscribeTransport,
			Description:           "Resubscribe instead of full restart",
			RiskReduction:         "Lower blast radius, faster recovery",
			ExpectedEffectiveness: 0.6,
			WhyRecommended:        "Addresses subscription issues without full reconnect",
		})
		alternatives = append(alternatives, AlternativeAction{
			ActionType:            control.ActionBackoffReset,
			Description:           "Reset backoff timing",
			RiskReduction:         "No connectivity disruption",
			ExpectedEffectiveness: 0.4,
			WhyRecommended:        "May help if transport is stuck in retry loop",
		})

	case control.ActionResubscribeTransport:
		alternatives = append(alternatives, AlternativeAction{
			ActionType:            control.ActionTriggerHealthRecheck,
			Description:           "Verify transport health first",
			RiskReduction:         "Read-only, confirms issue exists",
			ExpectedEffectiveness: 0.4,
			WhyRecommended:        "May reveal that resubscription is unnecessary",
		})

	case control.ActionBackoffIncrease:
		alternatives = append(alternatives, AlternativeAction{
			ActionType:            control.ActionTriggerHealthRecheck,
			Description:           "Check current health before adjusting backoff",
			RiskReduction:         "No state change",
			ExpectedEffectiveness: 0.5,
			WhyRecommended:        "Backoff may not be the root cause",
		})

	case control.ActionTemporarilyDeprioritize:
		alternatives = append(alternatives, AlternativeAction{
			ActionType:            control.ActionTriggerHealthRecheck,
			Description:           "Verify transport health status",
			RiskReduction:         "Read-only assessment",
			ExpectedEffectiveness: 0.4,
			WhyRecommended:        "Deprioritization requires verified alternate path",
		})

	default:
		// Generic alternative
		alternatives = append(alternatives, AlternativeAction{
			ActionType:            control.ActionTriggerHealthRecheck,
			Description:           "Perform health recheck",
			RiskReduction:         "No state changes, diagnostic only",
			ExpectedEffectiveness: 0.3,
			WhyRecommended:        "Gathers more information before taking action",
		})
	}

	// If there are conflicts, suggest waiting
	hasBlockingConflicts := false
	for _, c := range conflicts {
		if c.Severity == ConflictSeverityCritical || c.Severity == ConflictSeverityMajor {
			hasBlockingConflicts = true
			break
		}
	}
	if hasBlockingConflicts {
		alternatives = append(alternatives, AlternativeAction{
			ActionType:            "WAIT",
			Description:           "Wait for conflicts to resolve",
			RiskReduction:         "Avoids compounding issues",
			ExpectedEffectiveness: 0.7,
			WhyRecommended:        "Conflicts indicate action timing is poor",
		})
	}

	return alternatives
}

// requiresAcknowledgment determines if operator acknowledgment is needed.
func (e *SafeToActEvaluator) requiresAcknowledgment(
	riskAssessment RiskAssessment,
	conflicts []ConflictReport,
	policyPreview PolicyPreview,
) bool {
	// High or critical risk always requires acknowledgment
	for _, level := range e.riskLevelsRequiringAck {
		if riskAssessment.OverallRisk == level {
			return true
		}
	}

	// Critical conflicts require acknowledgment
	for _, c := range conflicts {
		if c.Severity == ConflictSeverityCritical {
			return true
		}
	}

	// Policy override requires acknowledgment
	if policyPreview.OverrideAvailable && !policyPreview.Allowed {
		return true
	}

	return false
}

// calculateConfidence computes overall decision confidence.
func (e *SafeToActEvaluator) calculateConfidence(
	riskAssessment RiskAssessment,
	conflicts []ConflictReport,
	policyPreview PolicyPreview,
	blastRadius BlastRadiusPrediction,
) float64 {
	confidence := riskAssessment.Confidence

	// Reduce confidence for conflicts
	conflictPenalty := 0.05 * float64(len(conflicts))
	confidence -= conflictPenalty

	// Reduce confidence for policy issues
	if !policyPreview.Allowed {
		confidence *= 0.7
	}

	// Reduce confidence for uncertain blast radius
	blastPenalty := (1.0 - blastRadius.Confidence) * 0.1
	confidence -= blastPenalty

	// Ensure bounds
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 0.95 {
		confidence = 0.95
	}

	return confidence
}

// Helper methods

func (e *SafeToActEvaluator) extractRiskScore(ra RiskAssessment) float64 {
	// Extract from explanation or use default mapping
	switch ra.OverallRisk {
	case RiskLevelNone:
		return 0.0
	case RiskLevelLow:
		return 0.15
	case RiskLevelMedium:
		return 0.4
	case RiskLevelHigh:
		return 0.7
	case RiskLevelCritical:
		return 0.95
	default:
		return 0.5
	}
}

func (e *SafeToActEvaluator) targetString(action control.ControlAction) string {
	if action.TargetTransport != "" {
		return action.TargetTransport
	}
	if action.TargetSegment != "" {
		return action.TargetSegment
	}
	if action.TargetNode != "" {
		return action.TargetNode
	}
	return "unknown"
}

func (e *SafeToActEvaluator) getPolicyResolution(pp PolicyPreview) string {
	switch pp.DenialCode {
	case control.DenialMode:
		return "Change control mode from disabled/advisory or wait for mode change"
	case control.DenialCooldown:
		return fmt.Sprintf("Wait %d seconds for cooldown to elapse", pp.CooldownInfo.CooldownDuration)
	case control.DenialBudget:
		return "Wait for action budget window to reset"
	case control.DenialConflict:
		return "Resolve conflicting actions before retrying"
	case control.DenialMissingActuator:
		return "Use manual remediation - actuator not implemented"
	default:
		return "Review policy configuration and denial reason"
	}
}

func (e *SafeToActEvaluator) getMitigationSuggestion(rf RiskFactor) string {
	if rf.Mitigatable {
		return fmt.Sprintf("Address %s issue: %s", rf.Category, rf.Impact)
	}
	return fmt.Sprintf("Cannot mitigate %s - consider alternative approach", rf.Category)
}

// ToSafeToActDecision converts EvaluationResult to the standard SafeToActDecision type.
// This provides backward compatibility with existing code.
func (er EvaluationResult) ToSafeToActDecision() SafeToActDecision {
	// Extract alternative action strings
	altStrings := make([]string, len(er.AlternativeActions))
	for i, alt := range er.AlternativeActions {
		altStrings[i] = fmt.Sprintf("%s: %s (%.0f%% effective)",
			alt.ActionType, alt.Description, alt.ExpectedEffectiveness*100)
	}

	// Build recommended action from next steps
	recommendedAction := ""
	for _, step := range er.NextSteps {
		if step.ActionType == "proceed" {
			recommendedAction = step.Description
			break
		}
	}
	if recommendedAction == "" && len(er.NextSteps) > 0 {
		recommendedAction = er.NextSteps[0].Description
	}

	return SafeToActDecision{
		SafeToAct:              er.SafeToAct,
		Decision:               er.Decision,
		PrimaryReason:          er.PrimaryReason,
		SupportingReasons:      er.SupportingReasons,
		OperatorGuidance:       er.OperatorGuidance,
		RequiresAcknowledgment: er.RequiresAcknowledgment,
		RecommendedAction:      recommendedAction,
		AlternativeActions:     altStrings,
		Confidence:             er.Confidence,
	}
}

// EvaluateSimple provides a simplified interface for basic evaluations.
// It takes a SimulationResult and returns an EvaluationResult.
func (e *SafeToActEvaluator) EvaluateSimple(result SimulationResult) EvaluationResult {
	return e.Evaluate(
		SimulationInput{ProposedAction: control.ControlAction{}}.ProposedAction,
		result.RiskAssessment,
		result.Conflicts,
		result.PolicyPreview,
		result.BlastRadius,
		result.OutcomeBranches,
		result.PredictedOutcome,
	)
}

// EvaluateFromInput performs a complete evaluation from simulation input.
// This is a convenience method that would typically be used with an Engine.
func (e *SafeToActEvaluator) EvaluateFromInput(
	input SimulationInput,
	result SimulationResult,
) EvaluationResult {
	return e.Evaluate(
		input.ProposedAction,
		result.RiskAssessment,
		result.Conflicts,
		result.PolicyPreview,
		result.BlastRadius,
		result.OutcomeBranches,
		result.PredictedOutcome,
	)
}

// IsExpired checks if the evaluation has expired and should be refreshed.
func (er EvaluationResult) IsExpired() bool {
	return time.Now().UTC().After(er.ExpiresAt)
}

// GetBlockingSteps returns only the blocking next steps.
func (er EvaluationResult) GetBlockingSteps() []NextStep {
	blocking := []NextStep{}
	for _, step := range er.NextSteps {
		if step.Blocking {
			blocking = append(blocking, step)
		}
	}
	return blocking
}

// CanProceedAutomatically indicates if the action can proceed without operator intervention.
func (er EvaluationResult) CanProceedAutomatically() bool {
	if !er.SafeToAct {
		return false
	}
	if er.RequiresAcknowledgment {
		return false
	}
	if er.Status == SafeToActStatusConditional {
		return false
	}
	return len(er.GetBlockingSteps()) == 0
}

// Summary returns a one-line summary of the evaluation.
func (er EvaluationResult) Summary() string {
	switch er.Status {
	case SafeToActStatusSafe:
		return fmt.Sprintf("SAFE: %s (confidence: %.0f%%)", er.PrimaryReason, er.Confidence*100)
	case SafeToActStatusConditional:
		return fmt.Sprintf("CONDITIONAL: %s (%d prerequisites)", er.PrimaryReason, len(er.MissingPrerequisites))
	case SafeToActStatusUnsafe:
		return fmt.Sprintf("UNSAFE: %s", er.PrimaryReason)
	case SafeToActStatusInsufficientData:
		return "INSUFFICIENT DATA: Cannot evaluate"
	default:
		return fmt.Sprintf("Status: %s", er.Status)
	}
}
