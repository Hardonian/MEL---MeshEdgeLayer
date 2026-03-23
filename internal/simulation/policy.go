package simulation

import (
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/status"
	"github.com/mel-project/mel/internal/transport"
)

// AdmissionResultExtended provides additional admission results beyond the basic types.
type AdmissionResultExtended string

const (
	// AdmissionAdmit indicates the action would be admitted and executed.
	AdmissionAdmit AdmissionResultExtended = "admit"
	// AdmissionDeny indicates the action would be denied.
	AdmissionDeny AdmissionResultExtended = "deny"
	// AdmissionConditional indicates the action might be admitted if prerequisites are met.
	AdmissionConditional AdmissionResultExtended = "conditional"
	// AdmissionResultUnknown indicates the admission result cannot be determined.
	AdmissionResultUnknown AdmissionResultExtended = "unknown"
)

// PolicyPreviewResult provides comprehensive preview information for a proposed action.
type PolicyPreviewResult struct {
	// AdmissionResult is the primary outcome: admit, deny, conditional, or unknown.
	AdmissionResult AdmissionResultExtended `json:"admission_result"`

	// Allowed indicates whether the action would be permitted under current conditions.
	Allowed bool `json:"allowed"`

	// WouldBeDenied indicates if policy would deny this action.
	WouldBeDenied bool `json:"would_be_denied"`

	// WouldTriggerCooldown indicates if this action would trigger a cooldown period.
	WouldTriggerCooldown bool `json:"would_trigger_cooldown"`

	// ViolatesGuardrails indicates if this action violates safety guardrails.
	ViolatesGuardrails bool `json:"violates_guardrails"`

	// DenialCode provides the specific denial reason code if denied.
	DenialCode string `json:"denial_code,omitempty"`

	// DenialReason provides human-readable explanation if denied.
	DenialReason string `json:"denial_reason,omitempty"`

	// Mode indicates the active control mode (disabled/advisory/guarded_auto).
	Mode string `json:"mode"`

	// ModeResult describes how the mode affects this action.
	ModeResult ModeImpact `json:"mode_result"`

	// Prerequisites lists conditions that must be met for conditional admission.
	Prerequisites []Prerequisite `json:"prerequisites,omitempty"`

	// ReasonCodes lists all applicable reason codes for the decision.
	ReasonCodes []string `json:"reason_codes,omitempty"`

	// SafetyChecks contains all safety check results.
	SafetyChecks SafetyCheckResults `json:"safety_checks"`

	// CooldownInfo provides details about any applicable cooldown.
	CooldownInfo CooldownPreview `json:"cooldown_info,omitempty"`

	// GuardrailViolations lists specific guardrail violations.
	GuardrailViolations []GuardrailViolation `json:"guardrail_violations,omitempty"`

	// AdvisoryNote provides guidance when action is advisory-only.
	AdvisoryNote string `json:"advisory_note,omitempty"`

	// OverrideAvailable indicates if an operator override could permit this action.
	OverrideAvailable bool `json:"override_available"`

	// GeneratedAt timestamp when this preview was generated.
	GeneratedAt time.Time `json:"generated_at"`

	// Confidence indicates confidence in this preview (0.0-1.0).
	Confidence float64 `json:"confidence"`
}

// ModeImpact describes how the control mode affects an action.
type ModeImpact struct {
	// Mode is the active control mode.
	Mode string `json:"mode"`

	// WouldExecute indicates if the action would be executed in this mode.
	WouldExecute bool `json:"would_execute"`

	// WouldLog indicates if the action would be logged in this mode.
	WouldLog bool `json:"would_log"`

	// WouldAlert indicates if the action would trigger alerts in this mode.
	WouldAlert bool `json:"would_alert"`

	// Description explains the mode's impact on this action.
	Description string `json:"description"`
}

// Prerequisite describes a condition that must be met for conditional admission.
type Prerequisite struct {
	// Code identifies the prerequisite type.
	Code string `json:"code"`

	// Description explains what must be satisfied.
	Description string `json:"description"`

	// CurrentlySatisfied indicates if this prerequisite is currently met.
	CurrentlySatisfied bool `json:"currently_satisfied"`

	// SuggestedAction describes how to satisfy this prerequisite.
	SuggestedAction string `json:"suggested_action,omitempty"`
}

// SafetyCheckResults aggregates all safety check outcomes.
type SafetyCheckResults struct {
	// EvidencePass indicates if sufficient evidence exists.
	EvidencePass bool `json:"evidence_pass"`

	// ConfidencePass indicates if confidence threshold is met.
	ConfidencePass bool `json:"confidence_pass"`

	// PolicyPass indicates if the action is in the allowed list.
	PolicyPass bool `json:"policy_pass"`

	// CooldownPass indicates if cooldown period has elapsed.
	CooldownPass bool `json:"cooldown_pass"`

	// OverridePass indicates no override is blocking this action.
	OverridePass bool `json:"override_pass"`

	// ConflictPass indicates no conflicting actions exist.
	ConflictPass bool `json:"conflict_pass"`

	// ReversibilityPass indicates the action is reversible.
	ReversibilityPass bool `json:"reversibility_pass"`

	// AlternatePathExists indicates a healthy alternate path exists (for deprioritization).
	AlternatePathExists bool `json:"alternate_path_exists"`

	// BudgetPass indicates action budget has not been exceeded.
	BudgetPass bool `json:"budget_pass"`

	// ActuatorExists indicates the actuator for this action is implemented.
	ActuatorExists bool `json:"actuator_exists"`

	// BlastRadiusKnown indicates the blast radius is bounded.
	BlastRadiusKnown bool `json:"blast_radius_known"`

	// SafeForGuardedAuto indicates the action is safe for automated execution.
	SafeForGuardedAuto bool `json:"safe_for_guarded_auto"`

	// AdvisoryOnly indicates this action remains advisory (not fully implemented).
	AdvisoryOnly bool `json:"advisory_only"`

	// AttributionStrong indicates strong attribution for source suppression.
	AttributionStrong bool `json:"attribution_strong"`

	// AttributionBestEffort indicates best-effort attribution status.
	AttributionBestEffort bool `json:"attribution_best_effort"`

	// AllPassed indicates if all required checks passed.
	AllPassed bool `json:"all_passed"`

	// FailedChecks lists the names of checks that failed.
	FailedChecks []string `json:"failed_checks,omitempty"`

	// PassedChecks lists the names of checks that passed.
	PassedChecks []string `json:"passed_checks,omitempty"`
}

// CooldownPreview provides cooldown-related preview information.
type CooldownPreview struct {
	// WouldTrigger indicates if this action would trigger cooldown.
	WouldTrigger bool `json:"would_trigger"`

	// CurrentlyInCooldown indicates if target is currently in cooldown.
	CurrentlyInCooldown bool `json:"currently_in_cooldown"`

	// RemainingSeconds indicates seconds remaining in current cooldown.
	RemainingSeconds int `json:"remaining_seconds,omitempty"`

	// CooldownDuration is the configured cooldown period.
	CooldownDuration int `json:"cooldown_duration_seconds"`

	// Target identifies the cooldown target.
	Target string `json:"target,omitempty"`
}

// GuardrailViolation describes a specific guardrail violation.
type GuardrailViolation struct {
	// Code identifies the violated guardrail.
	Code string `json:"code"`

	// Description explains the violation.
	Description string `json:"description"`

	// Severity of the violation.
	Severity string `json:"severity"`

	// Remediation suggests how to address the violation.
	Remediation string `json:"remediation,omitempty"`
}

// PolicyPreviewInput collects all inputs needed for policy preview generation.
type PolicyPreviewInput struct {
	// Action is the proposed control action to evaluate.
	Action control.ControlAction

	// Config is the system configuration.
	Config config.Config

	// Database provides access to historical action data.
	Database *db.DB

	// MeshState provides current mesh topology and health.
	MeshState status.MeshDrilldown

	// RuntimeHealth provides current transport health status.
	RuntimeHealth []transport.Health

	// Now is the reference time for the preview.
	Now time.Time

	// HistoryCache can optionally provide pre-loaded action history.
	HistoryCache map[string][]db.ControlActionRecord
}

// PolicyPreviewGenerator generates policy impact previews without side effects.
type PolicyPreviewGenerator struct {
	// config holds the system configuration.
	config config.Config

	// policy holds the derived control policy.
	policy control.ControlPolicy

	// realityByType provides action reality metadata.
	realityByType map[string]control.ActionReality
}

// NewPolicyPreviewGenerator creates a new policy preview generator.
func NewPolicyPreviewGenerator(cfg config.Config) *PolicyPreviewGenerator {
	return &PolicyPreviewGenerator{
		config:        cfg,
		policy:        control.PolicyFromConfig(cfg),
		realityByType: control.ActionRealityByType(),
	}
}

// GeneratePreview generates a policy impact preview for the proposed action.
// This method performs all safety checks from control.Evaluate but in preview mode
// with no side effects - no actions are recorded, no state is modified.
func (g *PolicyPreviewGenerator) GeneratePreview(input PolicyPreviewInput) PolicyPreviewResult {
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	candidate := input.Action
	cfg := input.Config
	policy := g.policy

	// Get action reality metadata
	reality, ok := g.realityByType[candidate.ActionType]
	if !ok {
		reality = control.ActionReality{
			ActionType:       candidate.ActionType,
			BlastRadiusClass: "unknown",
			DenialCode:       control.DenialMissingActuator,
		}
	}

	// Build runtime signals
	signals := buildRuntimeSignals(input.RuntimeHealth)

	// Perform all safety checks (preview mode - no side effects)
	checks := g.performSafetyChecks(input, reality, signals, now)

	// Determine mode impact
	modeResult := g.determineModeImpact(cfg, candidate, checks)

	// Determine admission result
	admissionResult, allowed, denialCode, denialReason := g.determineAdmissionResult(cfg, candidate, policy, reality, checks, modeResult)

	// Identify guardrail violations
	violations := g.identifyGuardrailViolations(checks, reality)

	// Build prerequisites for conditional admission
	prerequisites := g.buildPrerequisites(checks, candidate, input)

	// Build cooldown info
	cooldownInfo := g.buildCooldownPreview(input, candidate, policy, now)

	// Generate reason codes
	reasonCodes := g.generateReasonCodes(checks, violations, admissionResult)

	// Generate advisory note
	advisoryNote := g.generateAdvisoryNote(admissionResult, modeResult, violations, reality)

	// Calculate confidence
	confidence := g.calculateConfidence(checks, violations)

	return PolicyPreviewResult{
		AdmissionResult:      admissionResult,
		Allowed:              allowed,
		WouldBeDenied:        admissionResult == AdmissionDeny,
		WouldTriggerCooldown: cooldownInfo.WouldTrigger,
		ViolatesGuardrails:   len(violations) > 0,
		DenialCode:           denialCode,
		DenialReason:         denialReason,
		Mode:                 policy.Mode,
		ModeResult:           modeResult,
		Prerequisites:        prerequisites,
		ReasonCodes:          reasonCodes,
		SafetyChecks:         checks,
		CooldownInfo:         cooldownInfo,
		GuardrailViolations:  violations,
		AdvisoryNote:         advisoryNote,
		OverrideAvailable:    cfg.Control.EmergencyDisable || g.overrideWouldBeNeeded(input, candidate),
		GeneratedAt:          now,
		Confidence:           confidence,
	}
}

// performSafetyChecks executes all safety checks without side effects.
func (g *PolicyPreviewGenerator) performSafetyChecks(input PolicyPreviewInput, reality control.ActionReality, signals runtimeSignals, now time.Time) SafetyCheckResults {
	candidate := input.Action
	policy := g.policy
	database := input.Database

	// Attribution check (simulated, no DB writes)
	attribution := g.simulateAttribution(database, candidate.TargetTransport, now)

	// Evidence check
	evidencePass := g.checkEvidence(database, candidate, now, signals)

	// Confidence check
	confidencePass := candidate.Confidence >= policy.RequireMinConfidence

	// Policy allow check
	policyPass := g.policyAllows(policy, candidate.ActionType)

	// Specific policy flags
	if candidate.ActionType == control.ActionRestartTransport && !policy.AllowTransportRestart {
		policyPass = false
	}
	if candidate.ActionType == control.ActionTemporarilySuppressNoisySource && !policy.AllowSourceSuppression {
		policyPass = false
	}
	if g.isMeshLevelAction(candidate) && !policy.AllowMeshLevelActions {
		policyPass = false
	}

	// Override check
	overridePass := !input.Config.Control.EmergencyDisable && !g.checkOverride(input.Config, candidate)

	// Conflict check
	conflictPass := g.checkConflicts(database, policy, candidate, now)

	// Cooldown check
	cooldownPass := g.checkCooldown(database, policy, candidate, input.HistoryCache, now)

	// Budget check
	budgetPass := g.checkBudget(database, policy, candidate, now)

	// Alternate path check
	alternatePathExists := candidate.ActionType != control.ActionTemporarilyDeprioritize || g.healthyAlternateExists(input.MeshState, candidate.TargetTransport)

	// Reversibility check
	reversibilityPass := reality.Reversible

	// Blast radius check
	blastRadiusKnown := reality.BlastRadiusKnown && strings.TrimSpace(reality.BlastRadiusClass) != "" && reality.BlastRadiusClass != "unknown"

	// Collect failed checks
	failedChecks := []string{}
	passedChecks := []string{}

	if evidencePass {
		passedChecks = append(passedChecks, "evidence")
	} else {
		failedChecks = append(failedChecks, "evidence")
	}
	if confidencePass {
		passedChecks = append(passedChecks, "confidence")
	} else {
		failedChecks = append(failedChecks, "confidence")
	}
	if policyPass {
		passedChecks = append(passedChecks, "policy")
	} else {
		failedChecks = append(failedChecks, "policy")
	}
	if cooldownPass {
		passedChecks = append(passedChecks, "cooldown")
	} else {
		failedChecks = append(failedChecks, "cooldown")
	}
	if overridePass {
		passedChecks = append(passedChecks, "override")
	} else {
		failedChecks = append(failedChecks, "override")
	}
	if conflictPass {
		passedChecks = append(passedChecks, "conflict")
	} else {
		failedChecks = append(failedChecks, "conflict")
	}
	if reversibilityPass {
		passedChecks = append(passedChecks, "reversibility")
	} else {
		failedChecks = append(failedChecks, "reversibility")
	}
	if budgetPass {
		passedChecks = append(passedChecks, "budget")
	} else {
		failedChecks = append(failedChecks, "budget")
	}
	if blastRadiusKnown {
		passedChecks = append(passedChecks, "blast_radius")
	} else {
		failedChecks = append(failedChecks, "blast_radius")
	}
	if reality.ActuatorExists {
		passedChecks = append(passedChecks, "actuator")
	} else {
		failedChecks = append(failedChecks, "actuator")
	}

	allPassed := len(failedChecks) == 0

	return SafetyCheckResults{
		EvidencePass:          evidencePass,
		ConfidencePass:        confidencePass,
		PolicyPass:            policyPass,
		CooldownPass:          cooldownPass,
		OverridePass:          overridePass,
		ConflictPass:          conflictPass,
		ReversibilityPass:     reversibilityPass,
		AlternatePathExists:   alternatePathExists,
		BudgetPass:            budgetPass,
		ActuatorExists:        reality.ActuatorExists,
		BlastRadiusKnown:      blastRadiusKnown,
		SafeForGuardedAuto:    reality.SafeForGuardedAuto,
		AdvisoryOnly:          reality.AdvisoryOnly,
		AttributionStrong:     attribution.Strong,
		AttributionBestEffort: attribution.BestEffort,
		AllPassed:             allPassed,
		FailedChecks:          failedChecks,
		PassedChecks:          passedChecks,
	}
}

// determineModeImpact calculates how the control mode affects the action.
func (g *PolicyPreviewGenerator) determineModeImpact(cfg config.Config, candidate control.ControlAction, checks SafetyCheckResults) ModeImpact {
	mode := cfg.Control.Mode

	switch mode {
	case control.ModeDisabled:
		return ModeImpact{
			Mode:         mode,
			WouldExecute: false,
			WouldLog:     false,
			WouldAlert:   false,
			Description:  "Control mode is disabled. No actions will be executed, logged, or alerted.",
		}
	case control.ModeAdvisory:
		return ModeImpact{
			Mode:         mode,
			WouldExecute: false,
			WouldLog:     true,
			WouldAlert:   true,
			Description:  "Control mode is advisory. Actions will be logged and alerted but not executed.",
		}
	case control.ModeGuardedAuto:
		wouldExecute := checks.AllPassed && !checks.AdvisoryOnly
		return ModeImpact{
			Mode:         mode,
			WouldExecute: wouldExecute,
			WouldLog:     true,
			WouldAlert:   true,
			Description:  g.buildGuardedAutoDescription(wouldExecute, checks),
		}
	default:
		return ModeImpact{
			Mode:         mode,
			WouldExecute: false,
			WouldLog:     false,
			WouldAlert:   false,
			Description:  fmt.Sprintf("Unknown control mode: %s", mode),
		}
	}
}

// determineAdmissionResult calculates the final admission result.
func (g *PolicyPreviewGenerator) determineAdmissionResult(
	cfg config.Config,
	candidate control.ControlAction,
	policy control.ControlPolicy,
	reality control.ActionReality,
	checks SafetyCheckResults,
	modeResult ModeImpact,
) (AdmissionResultExtended, bool, string, string) {
	// Check for emergency disable
	if cfg.Control.EmergencyDisable {
		return AdmissionDeny, false, control.DenialOverride, "control disabled by emergency_disable"
	}

	// Mode-based determination
	switch policy.Mode {
	case control.ModeDisabled:
		return AdmissionDeny, false, control.DenialMode, "control mode is disabled"
	case control.ModeAdvisory:
		return AdmissionConditional, false, control.DenialMode, "control mode is advisory - action would be logged but not executed"
	}

	// Guarded auto mode - evaluate all checks
	denialCode, denialReason := g.findPrimaryDenial(checks, candidate, policy, reality)

	if denialCode != "" {
		return AdmissionDeny, false, denialCode, denialReason
	}

	// Check if action is advisory-only
	if checks.AdvisoryOnly || !reality.SafeForGuardedAuto {
		return AdmissionConditional, false, control.DenialMissingActuator, "action remains advisory because MEL does not ship a verified actuator for it"
	}

	// Check if all prerequisites are met
	if !checks.AllPassed {
		return AdmissionConditional, false, "", "some safety checks did not pass"
	}

	return AdmissionAdmit, true, "", ""
}

// identifyGuardrailViolations identifies specific guardrail violations.
func (g *PolicyPreviewGenerator) identifyGuardrailViolations(checks SafetyCheckResults, reality control.ActionReality) []GuardrailViolation {
	violations := []GuardrailViolation{}

	if !checks.BlastRadiusKnown {
		violations = append(violations, GuardrailViolation{
			Code:        "unknown_blast_radius",
			Description: "Blast radius is not bounded strongly enough for guarded automation",
			Severity:    "high",
			Remediation: "Ensure action has well-defined impact scope",
		})
	}

	if !checks.ReversibilityPass {
		violations = append(violations, GuardrailViolation{
			Code:        "irreversible_action",
			Description: "Action is not reversible or expiry-backed",
			Severity:    "medium",
			Remediation: "Consider adding reversibility mechanism or expiry",
		})
	}

	if !checks.ActuatorExists {
		violations = append(violations, GuardrailViolation{
			Code:        "missing_actuator",
			Description: "Action actuator is not implemented in this build",
			Severity:    "critical",
			Remediation: "Verify action type is supported or wait for implementation",
		})
	}

	if !checks.ConfidencePass {
		violations = append(violations, GuardrailViolation{
			Code:        "low_confidence",
			Description: "Candidate confidence is below the guarded_auto threshold",
			Severity:    "medium",
			Remediation: "Gather more evidence or increase confidence threshold",
		})
	}

	if !checks.AlternatePathExists {
		violations = append(violations, GuardrailViolation{
			Code:        "no_alternate_path",
			Description: "No healthy alternate transport exists for deprioritization",
			Severity:    "high",
			Remediation: "Ensure redundant transport paths are available",
		})
	}

	if !checks.BudgetPass {
		violations = append(violations, GuardrailViolation{
			Code:        "budget_exceeded",
			Description: "Action budget exceeded for current control window",
			Severity:    "medium",
			Remediation: "Wait for control window to reset or increase budget",
		})
	}

	return violations
}

// buildPrerequisites constructs prerequisites list for conditional admission.
func (g *PolicyPreviewGenerator) buildPrerequisites(checks SafetyCheckResults, candidate control.ControlAction, input PolicyPreviewInput) []Prerequisite {
	prereqs := []Prerequisite{}

	if !checks.EvidencePass {
		prereqs = append(prereqs, Prerequisite{
			Code:               "sufficient_evidence",
			Description:        "Persistent evidence of the issue must exist in history",
			CurrentlySatisfied: false,
			SuggestedAction:    "Wait for more telemetry or trigger diagnostic collection",
		})
	}

	if !checks.ConfidencePass {
		prereqs = append(prereqs, Prerequisite{
			Code:               "confidence_threshold",
			Description:        fmt.Sprintf("Confidence must meet minimum threshold (%.2f)", g.policy.RequireMinConfidence),
			CurrentlySatisfied: false,
			SuggestedAction:    "Gather additional evidence or wait for issue to persist",
		})
	}

	if !checks.CooldownPass {
		prereqs = append(prereqs, Prerequisite{
			Code:               "cooldown_elapsed",
			Description:        "Cooldown period for target must have elapsed",
			CurrentlySatisfied: false,
			SuggestedAction:    fmt.Sprintf("Wait %d seconds before retrying", g.policy.CooldownPerTarget),
		})
	}

	if !checks.ConflictPass {
		prereqs = append(prereqs, Prerequisite{
			Code:               "no_conflicts",
			Description:        "No conflicting active actions may exist for target",
			CurrentlySatisfied: false,
			SuggestedAction:    "Wait for existing actions to complete or resolve conflicts",
		})
	}

	if !checks.BudgetPass {
		prereqs = append(prereqs, Prerequisite{
			Code:               "budget_available",
			Description:        "Action budget must have available capacity",
			CurrentlySatisfied: false,
			SuggestedAction:    "Wait for action window to reset",
		})
	}

	if !checks.ActuatorExists {
		prereqs = append(prereqs, Prerequisite{
			Code:               "actuator_implemented",
			Description:        "Actuator for this action type must be implemented",
			CurrentlySatisfied: false,
			SuggestedAction:    "Use manual remediation or wait for feature implementation",
		})
	}

	return prereqs
}

// buildCooldownPreview constructs cooldown preview information.
func (g *PolicyPreviewGenerator) buildCooldownPreview(input PolicyPreviewInput, candidate control.ControlAction, policy control.ControlPolicy, now time.Time) CooldownPreview {
	preview := CooldownPreview{
		WouldTrigger:     true,
		CooldownDuration: policy.CooldownPerTarget,
		Target:           candidate.TargetTransport,
	}

	if policy.CooldownPerTarget <= 0 || candidate.TargetTransport == "" {
		preview.WouldTrigger = false
		return preview
	}

	// Check if currently in cooldown
	if input.Database != nil {
		start := now.Add(-time.Duration(policy.CooldownPerTarget) * time.Second).UTC().Format(time.RFC3339)
		rows, _ := input.Database.ControlActions(candidate.TargetTransport, "", start, "", "", 50, 0)

		for _, row := range rows {
			if row.Result != control.ResultExecutedSuccessfully && row.Result != control.ResultExecutedNoop {
				continue
			}

			executedAt, hasExecuted := parseRFC3339(row.ExecutedAt)
			completedAt, hasCompleted := parseRFC3339(row.CompletedAt)

			var ts time.Time
			if hasCompleted {
				ts = completedAt
			} else if hasExecuted {
				ts = executedAt
			} else {
				continue
			}

			elapsed := now.Sub(ts)
			cooldownDuration := time.Duration(policy.CooldownPerTarget) * time.Second

			if elapsed < cooldownDuration {
				preview.CurrentlyInCooldown = true
				preview.RemainingSeconds = int((cooldownDuration - elapsed).Seconds())
				break
			}
		}
	}

	return preview
}

// Helper methods

func (g *PolicyPreviewGenerator) policyAllows(policy control.ControlPolicy, actionType string) bool {
	for _, allowed := range policy.AllowedActions {
		if allowed == actionType {
			return true
		}
	}
	return false
}

func (g *PolicyPreviewGenerator) isMeshLevelAction(action control.ControlAction) bool {
	return action.TargetSegment != "" || action.ActionType == control.ActionTemporarilyDeprioritize
}

func (g *PolicyPreviewGenerator) checkOverride(cfg config.Config, candidate control.ControlAction) bool {
	if candidate.TargetTransport == "" {
		return false
	}
	for _, tc := range cfg.Transports {
		if tc.Name != candidate.TargetTransport {
			continue
		}
		if tc.ManualOnly || tc.SuppressAutoActions {
			return true
		}
		if candidate.ActionType == control.ActionTemporarilyDeprioritize && tc.FreezeRouting {
			return true
		}
	}
	return false
}

func (g *PolicyPreviewGenerator) overrideWouldBeNeeded(input PolicyPreviewInput, candidate control.ControlAction) bool {
	if input.Config.Control.EmergencyDisable {
		return true
	}
	return g.checkOverride(input.Config, candidate)
}

func (g *PolicyPreviewGenerator) checkEvidence(database *db.DB, candidate control.ControlAction, now time.Time, signals runtimeSignals) bool {
	if database == nil || candidate.TargetTransport == "" {
		return candidate.ActionType == control.ActionTriggerHealthRecheck
	}

	start := now.Add(-15 * time.Minute).UTC().Format(time.RFC3339)
	rows, err := database.TransportAnomalyHistory(candidate.TargetTransport, start, now.UTC().Format(time.RFC3339), 50, 0)
	if err != nil || len(rows) == 0 {
		return false
	}

	buckets := map[string]uint64{}
	for _, row := range rows {
		if row.Count == 0 {
			continue
		}
		switch candidate.ActionType {
		case control.ActionRestartTransport:
			if row.Reason == transport.ReasonRetryThresholdExceeded {
				buckets[row.BucketStart] += row.Count
			}
		case control.ActionResubscribeTransport:
			if row.Reason == transport.ReasonSubscribeFailure {
				buckets[row.BucketStart] += row.Count
			}
		case control.ActionBackoffIncrease:
			if row.ObservationDrops > 0 || row.Reason == transport.ReasonMalformedFrame || row.Reason == transport.ReasonMalformedPublish {
				buckets[row.BucketStart] += maxUint64(row.Count, row.ObservationDrops)
			}
		default:
			if row.Count > 0 {
				buckets[row.BucketStart] += row.Count
			}
		}
	}

	return len(buckets) >= 2
}

func (g *PolicyPreviewGenerator) checkConflicts(database *db.DB, policy control.ControlPolicy, candidate control.ControlAction, now time.Time) bool {
	if database == nil {
		return true
	}

	start := now.Add(-time.Duration(policy.ActionWindowSeconds) * time.Second).UTC().Format(time.RFC3339)
	rows, _ := database.ControlActions(candidate.TargetTransport, "", start, "", "", policy.MaxActionsPerWindow+10, 0)

	for _, action := range rows {
		if action.TargetTransport != "" && action.TargetTransport != candidate.TargetTransport {
			continue
		}
		if action.ActionType == candidate.ActionType && g.isEffectivelyActive(action, now) {
			return false
		}
		if action.ActionType == control.ActionRestartTransport && candidate.ActionType == control.ActionResubscribeTransport && g.isEffectivelyActive(action, now) {
			return false
		}
	}
	return true
}

func (g *PolicyPreviewGenerator) checkCooldown(database *db.DB, policy control.ControlPolicy, candidate control.ControlAction, historyCache map[string][]db.ControlActionRecord, now time.Time) bool {
	if database == nil || candidate.TargetTransport == "" || policy.CooldownPerTarget <= 0 {
		return true
	}

	key := candidate.TargetTransport
	rows, ok := historyCache[key]
	if !ok {
		start := now.Add(-time.Duration(policy.CooldownPerTarget) * time.Second).UTC().Format(time.RFC3339)
		rows, _ = database.ControlActions(candidate.TargetTransport, "", start, "", "", 50, 0)
	}

	for _, row := range rows {
		if row.Result != control.ResultExecutedSuccessfully && row.Result != control.ResultExecutedNoop {
			continue
		}
		executedAt, hasExecuted := parseRFC3339(row.ExecutedAt)
		completedAt, hasCompleted := parseRFC3339(row.CompletedAt)

		var ts time.Time
		if hasCompleted {
			ts = completedAt
		} else if hasExecuted {
			ts = executedAt
		} else {
			continue
		}

		if now.Sub(ts) < time.Duration(policy.CooldownPerTarget)*time.Second {
			return false
		}
	}
	return true
}

func (g *PolicyPreviewGenerator) checkBudget(database *db.DB, policy control.ControlPolicy, candidate control.ControlAction, now time.Time) bool {
	if database == nil || policy.MaxActionsPerWindow <= 0 {
		return true
	}

	start := now.Add(-time.Duration(policy.ActionWindowSeconds) * time.Second).UTC().Format(time.RFC3339)
	rows, err := database.ControlActions("", "", start, "", "", policy.MaxActionsPerWindow+10, 0)
	if err != nil {
		return false
	}

	executedCount := 0
	for _, row := range rows {
		if row.Result == control.ResultExecutedSuccessfully || row.Result == control.ResultExecutedNoop {
			executedCount++
		}
	}

	if executedCount >= policy.MaxActionsPerWindow {
		return false
	}

	if candidate.ActionType != control.ActionRestartTransport || policy.RestartCapPerWindow <= 0 {
		return true
	}

	restartCount := 0
	for _, row := range rows {
		if row.ActionType == control.ActionRestartTransport && (row.Result == control.ResultExecutedSuccessfully || row.Result == control.ResultExecutedNoop) {
			restartCount++
		}
	}

	return restartCount < policy.RestartCapPerWindow
}

func (g *PolicyPreviewGenerator) healthyAlternateExists(mesh status.MeshDrilldown, degraded string) bool {
	for _, route := range mesh.RoutingRecommendations {
		if route.Action == "suggest_alternate_ingest_path" && route.TargetTransport == degraded {
			return true
		}
	}
	return false
}

func (g *PolicyPreviewGenerator) isEffectivelyActive(row db.ControlActionRecord, now time.Time) bool {
	if row.Result != control.ResultExecutedSuccessfully {
		return false
	}
	if row.ExpiresAt == "" {
		return row.Reversible
	}
	expires, ok := parseRFC3339(row.ExpiresAt)
	return ok && now.Before(expires)
}

func (g *PolicyPreviewGenerator) findPrimaryDenial(checks SafetyCheckResults, candidate control.ControlAction, policy control.ControlPolicy, reality control.ActionReality) (string, string) {
	if !checks.OverridePass {
		return control.DenialOverride, "operator override suppresses automation for target"
	}
	if !checks.PolicyPass {
		switch {
		case candidate.ActionType == control.ActionRestartTransport && !policy.AllowTransportRestart:
			return control.DenialPolicy, "policy disables transport restart actions"
		case candidate.ActionType == control.ActionTemporarilySuppressNoisySource && !policy.AllowSourceSuppression:
			return control.DenialPolicy, "policy disables source suppression"
		case g.isMeshLevelAction(candidate) && !policy.AllowMeshLevelActions:
			return control.DenialPolicy, "policy disables mesh-level actions"
		default:
			return control.DenialPolicy, "policy does not allow action type"
		}
	}
	if !checks.ConfidencePass {
		return control.DenialLowConfidence, "candidate confidence is below the guarded_auto threshold"
	}
	if !checks.AlternatePathExists {
		return control.DenialNoAlternatePath, "no healthy alternate transport exists for deprioritization"
	}
	if !checks.EvidencePass {
		return control.DenialTransient, "evidence remains transient or ambiguous in persisted history"
	}
	if !checks.CooldownPass {
		return control.DenialCooldown, "cooldown window still active for target"
	}
	if !checks.BudgetPass {
		return control.DenialBudget, "action budget exceeded for current control window"
	}
	if !checks.ConflictPass {
		return control.DenialConflict, "conflicting active or in-flight action already exists for target"
	}
	if !checks.BlastRadiusKnown {
		return control.DenialUnknownBlastRadius, "blast radius is not bounded strongly enough for guarded automation"
	}
	if !checks.ReversibilityPass {
		return control.DenialIrreversible, "action is not reversible or expiry-backed"
	}
	if !checks.ActuatorExists || reality.AdvisoryOnly || !reality.SafeForGuardedAuto {
		code := reality.DenialCode
		if code == "" {
			code = control.DenialMissingActuator
		}
		return code, firstNonEmpty(reality.Notes, "action remains advisory because MEL does not ship a verified actuator for it")
	}
	return "", ""
}

func (g *PolicyPreviewGenerator) buildGuardedAutoDescription(wouldExecute bool, checks SafetyCheckResults) string {
	if wouldExecute {
		return "Action meets all safety criteria and would be executed automatically."
	}
	if checks.AdvisoryOnly {
		return "Action remains advisory-only - actuator not yet verified."
	}
	if !checks.AllPassed {
		return fmt.Sprintf("Action blocked - %d safety check(s) failed: %v", len(checks.FailedChecks), checks.FailedChecks)
	}
	return "Action would not be executed due to guardrails."
}

func (g *PolicyPreviewGenerator) generateReasonCodes(checks SafetyCheckResults, violations []GuardrailViolation, admission AdmissionResultExtended) []string {
	codes := []string{}

	// Add admission result code
	switch admission {
	case AdmissionAdmit:
		codes = append(codes, "admit")
	case AdmissionDeny:
		codes = append(codes, "deny")
	case AdmissionConditional:
		codes = append(codes, "conditional")
	case AdmissionResultUnknown:
		codes = append(codes, "unknown")
	}

	// Add mode code
	codes = append(codes, g.policy.Mode)

	// Add violation codes
	for _, v := range violations {
		codes = append(codes, v.Code)
	}

	// Add check failure codes
	for _, check := range checks.FailedChecks {
		codes = append(codes, check+"_failed")
	}

	return codes
}

func (g *PolicyPreviewGenerator) generateAdvisoryNote(admission AdmissionResultExtended, mode ModeImpact, violations []GuardrailViolation, reality control.ActionReality) string {
	if admission == AdmissionAdmit {
		return "Action is cleared for automatic execution."
	}

	if admission == AdmissionDeny {
		return mode.Description
	}

	if len(violations) > 0 {
		return fmt.Sprintf("Action is conditional: %s. %s", violations[0].Description, violations[0].Remediation)
	}

	if reality.AdvisoryOnly {
		return firstNonEmpty(reality.Notes, "This action type remains advisory-only in the current build.")
	}

	return mode.Description
}

func (g *PolicyPreviewGenerator) calculateConfidence(checks SafetyCheckResults, violations []GuardrailViolation) float64 {
	base := 1.0

	// Reduce confidence for each failed check
	penalty := 0.05 * float64(len(checks.FailedChecks))
	base -= penalty

	// Reduce confidence for guardrail violations
	violationPenalty := 0.1 * float64(len(violations))
	base -= violationPenalty

	// Ensure non-negative
	if base < 0 {
		base = 0
	}

	return base
}

// simulateAttribution simulates attribution analysis without DB writes.
func (g *PolicyPreviewGenerator) simulateAttribution(database *db.DB, transportName string, now time.Time) struct {
	NodeID      string
	Confidence  float64
	BestEffort  bool
	Strong      bool
	MessageSpan int
} {
	// This simulates the attribution check without requiring DB access
	// In preview mode, we return best-effort if no database is available
	if database == nil || strings.TrimSpace(transportName) == "" {
		return struct {
			NodeID      string
			Confidence  float64
			BestEffort  bool
			Strong      bool
			MessageSpan int
		}{BestEffort: true}
	}

	// Query would be performed here in real execution
	// For preview, we assume best-effort attribution
	return struct {
		NodeID      string
		Confidence  float64
		BestEffort  bool
		Strong      bool
		MessageSpan int
	}{BestEffort: true, Confidence: 0.5}
}

// runtimeSignals mirrors the internal type from control package.
type runtimeSignals struct {
	connectedCount int
	byTransport    map[string]transport.Health
}

func buildRuntimeSignals(runtime []transport.Health) runtimeSignals {
	out := runtimeSignals{byTransport: map[string]transport.Health{}}
	for _, item := range runtime {
		out.byTransport[item.Name] = item
		if item.State == transport.StateLive || item.State == transport.StateIdle {
			out.connectedCount++
		}
	}
	return out
}

func parseRFC3339(v string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if ts, err := time.Parse(layout, strings.TrimSpace(v)); err == nil {
			return ts.UTC(), true
		}
	}
	return time.Time{}, false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func maxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
