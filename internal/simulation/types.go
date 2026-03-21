// Package simulation provides the core simulation engine for MEL control actions.
// It enables pre-execution what-if analysis to predict outcomes, assess risks,
// and provide operator guidance before taking automated or manual actions.
package simulation

import (
	"time"

	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/diagnostics"
	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/selfobs"
	"github.com/mel-project/mel/internal/status"
	"github.com/mel-project/mel/internal/transport"
)

// RiskLevel represents the severity of identified risks.
type RiskLevel string

const (
	RiskLevelNone     RiskLevel = "none"
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// SafetyLevel represents the assessed safety of proceeding with an action.
type SafetyLevel string

const (
	SafetyLevelSafe      SafetyLevel = "safe"
	SafetyLevelCaution   SafetyLevel = "caution"
	SafetyLevelAtRisk    SafetyLevel = "at_risk"
	SafetyLevelUnsafe    SafetyLevel = "unsafe"
	SafetyLevelForbidden SafetyLevel = "forbidden"
)

// AdmissionResult represents the policy admission check outcome.
type AdmissionResult string

const (
	AdmissionAllowed    AdmissionResult = "allowed"
	AdmissionDenied     AdmissionResult = "denied"
	AdmissionAdvisory   AdmissionResult = "advisory"
	AdmissionPending    AdmissionResult = "pending_review"
	AdmissionUnknown    AdmissionResult = "unknown"
)

// ConflictSeverity represents the severity of detected conflicts.
type ConflictSeverity string

const (
	ConflictSeverityNone     ConflictSeverity = "none"
	ConflictSeverityMinor    ConflictSeverity = "minor"
	ConflictSeverityModerate ConflictSeverity = "moderate"
	ConflictSeverityMajor    ConflictSeverity = "major"
	ConflictSeverityCritical ConflictSeverity = "critical"
)

// SimulationInput collects all available inputs for simulating a control action.
// It provides a comprehensive snapshot of system state at simulation time.
type SimulationInput struct {
	// SimulationID uniquely identifies this simulation run
	SimulationID string `json:"simulation_id"`

	// Timestamp when the simulation was initiated
	Timestamp time.Time `json:"timestamp"`

	// ProposedAction is the control action being evaluated
	ProposedAction control.ControlAction `json:"proposed_action"`

	// ActiveIncidents lists currently active system incidents
	ActiveIncidents []models.Incident `json:"active_incidents,omitempty"`

	// DiagnosticReport contains recent diagnostic findings
	DiagnosticReport diagnostics.DiagnosticReport `json:"diagnostic_report,omitempty"`

	// TransportHealth contains current health status of all transports
	TransportHealth []transport.Health `json:"transport_health,omitempty"`

	// MeshTopology provides the current mesh state and drilldown
	MeshTopology status.MeshDrilldown `json:"mesh_topology,omitempty"`

	// ActionHistory contains recent control actions for context
	ActionHistory []models.ActionRecord `json:"action_history,omitempty"`

	// PolicyConfig contains the active control policy
	PolicyConfig control.ControlPolicy `json:"policy_config"`

	// FreshnessSignals tracks data freshness for all components
	FreshnessSignals []selfobs.FreshnessMarker `json:"freshness_signals,omitempty"`

	// ContextData holds additional simulation context (can be extended)
	ContextData map[string]any `json:"context_data,omitempty"`
}

// SimulationResult provides the comprehensive output of a simulation run.
// It contains all predictions, assessments, and recommendations for operator guidance.
type SimulationResult struct {
	// SimulationID matches the input simulation ID
	SimulationID string `json:"simulation_id"`

	// Timestamp when the simulation completed
	CompletedAt time.Time `json:"completed_at"`

	// PredictedOutcome contains the expected result if action is taken
	PredictedOutcome PredictedOutcome `json:"predicted_outcome"`

	// RiskAssessment provides explicit risk evaluation
	RiskAssessment RiskAssessment `json:"risk_assessment"`

	// PolicyPreview contains the pre-execution admission check
	PolicyPreview PolicyPreview `json:"policy_preview"`

	// Conflicts lists any detected conflicts
	Conflicts []ConflictReport `json:"conflicts,omitempty"`

	// BlastRadius predicts the scope of impact
	BlastRadius BlastRadiusPrediction `json:"blast_radius"`

	// OutcomeBranches describes best/expected/worst case scenarios
	OutcomeBranches []OutcomeBranch `json:"outcome_branches,omitempty"`

	// SafeToAct provides the final operator guidance decision
	SafeToAct SafeToActDecision `json:"safe_to_act"`

	// Metadata contains additional simulation metadata
	Metadata SimulationMetadata `json:"metadata"`
}

// PredictedOutcome describes what will likely happen if the proposed action is taken.
type PredictedOutcome struct {
	// SuccessProbability estimates likelihood of successful execution (0.0-1.0)
	SuccessProbability float64 `json:"success_probability"`

	// ExpectedState describes the anticipated resulting system state
	ExpectedState string `json:"expected_state"`

	// ExpectedDuration estimates how long the action will take to complete
	ExpectedDuration time.Duration `json:"expected_duration,omitempty"`

	// SideEffects lists anticipated secondary effects
	SideEffects []PredictedEffect `json:"side_effects,omitempty"`

	// RollbackCapability indicates if the action can be reversed
	RollbackCapability RollbackInfo `json:"rollback_capability"`

	// Explanation provides human-readable reasoning
	Explanation string `json:"explanation"`
}

// PredictedEffect describes a single anticipated side effect.
type PredictedEffect struct {
	// Component affected by this effect
	Component string `json:"component"`

	// Effect describes what will happen
	Effect string `json:"effect"`

	// Severity of the effect
	Severity RiskLevel `json:"severity"`

	// Likelihood of this effect occurring (0.0-1.0)
	Likelihood float64 `json:"likelihood"`

	// Mitigation suggests how to address this effect
	Mitigation string `json:"mitigation,omitempty"`
}

// RollbackInfo describes the reversibility of an action.
type RollbackInfo struct {
	// Reversible indicates if the action can be undone
	Reversible bool `json:"reversible"`

	// Automatic indicates if rollback happens automatically
	Automatic bool `json:"automatic"`

	// RollbackDuration estimates time to complete rollback
	RollbackDuration time.Duration `json:"rollback_duration,omitempty"`

	// RollbackWindow defines the time window for rollback availability
	RollbackWindow time.Duration `json:"rollback_window,omitempty"`

	// Notes provides additional rollback context
	Notes string `json:"notes,omitempty"`
}

// RiskAssessment provides explicit risk evaluation with detailed explanations.
type RiskAssessment struct {
	// OverallRisk is the aggregated risk level
	OverallRisk RiskLevel `json:"overall_risk"`

	// SafetyLevel indicates the safety of proceeding
	SafetyLevel SafetyLevel `json:"safety_level"`

	// RiskFactors lists individual risk factors identified
	RiskFactors []RiskFactor `json:"risk_factors,omitempty"`

	// Mitigations lists available risk mitigations
	Mitigations []string `json:"mitigations,omitempty"`

	// Confidence in this assessment (0.0-1.0)
	Confidence float64 `json:"confidence"`

	// Explanation provides human-readable risk reasoning
	Explanation string `json:"explanation"`

	// EvidenceLossRisk specifically flags risk of losing observability data
	EvidenceLossRisk bool `json:"evidence_loss_risk"`

	// ConnectivityRisk flags risk of breaking mesh connectivity
	ConnectivityRisk bool `json:"connectivity_risk"`
}

// RiskFactor describes a single identified risk.
type RiskFactor struct {
	// Category of risk (e.g., "transport", "mesh", "data")
	Category string `json:"category"`

	// Description of the specific risk
	Description string `json:"description"`

	// Level of this risk
	Level RiskLevel `json:"level"`

	// Likelihood of this risk materializing (0.0-1.0)
	Likelihood float64 `json:"likelihood"`

	// Impact describes consequences if risk occurs
	Impact string `json:"impact"`

	// Mitigatable indicates if this risk can be mitigated
	Mitigatable bool `json:"mitigatable"`
}

// PolicyPreview contains the pre-execution admission check result.
type PolicyPreview struct {
	// Result of the admission check
	Result AdmissionResult `json:"result"`

	// Allowed indicates if the action would be permitted
	Allowed bool `json:"allowed"`

	// DenialCode provides the specific denial reason code (if denied)
	DenialCode string `json:"denial_code,omitempty"`

	// DenialReason provides human-readable denial explanation
	DenialReason string `json:"denial_reason,omitempty"`

	// Mode indicates the active control mode
	Mode string `json:"mode"`

	// ChecksPassed lists all safety checks that passed
	ChecksPassed []string `json:"checks_passed,omitempty"`

	// ChecksFailed lists all safety checks that failed
	ChecksFailed []string `json:"checks_failed,omitempty"`

	// OverrideAvailable indicates if an operator override is possible
	OverrideAvailable bool `json:"override_available"`

	// AdvisoryNote provides guidance when action is advisory-only
	AdvisoryNote string `json:"advisory_note,omitempty"`
}

// ConflictReport describes a detected conflict with other actions or state.
type ConflictReport struct {
	// ConflictID uniquely identifies this conflict
	ConflictID string `json:"conflict_id"`

	// Severity of the conflict
	Severity ConflictSeverity `json:"severity"`

	// Type of conflict (e.g., "active_action", "cooldown", "resource")
	Type string `json:"type"`

	// Description explains the conflict
	Description string `json:"description"`

	// ConflictingAction references the action in conflict (if applicable)
	ConflictingActionID string `json:"conflicting_action_id,omitempty"`

	// Resource identifies the resource in conflict
	Resource string `json:"resource,omitempty"`

	// Resolution suggests how to resolve the conflict
	Resolution string `json:"resolution,omitempty"`

	// AutoResolvable indicates if the conflict can be auto-resolved
	AutoResolvable bool `json:"auto_resolvable"`
}

// BlastRadiusPrediction predicts the scope of impact from an action.
type BlastRadiusPrediction struct {
	// Score represents the blast radius magnitude (0.0-1.0)
	Score float64 `json:"score"`

	// Classification categorizes the blast radius
	Classification string `json:"classification"`

	// Description provides human-readable impact summary
	Description string `json:"description"`

	// AffectedTransports lists transports likely to be impacted
	AffectedTransports []string `json:"affected_transports,omitempty"`

	// AffectedNodes lists nodes likely to be impacted
	AffectedNodes []string `json:"affected_nodes,omitempty"`

	// AffectedSegments lists mesh segments likely to be impacted
	AffectedSegments []string `json:"affected_segments,omitempty"`

	// ServiceImpact describes impact on services
	ServiceImpact []ServiceImpact `json:"service_impact,omitempty"`

	// Confidence in this prediction (0.0-1.0)
	Confidence float64 `json:"confidence"`
}

// ServiceImpact describes impact on a specific service.
type ServiceImpact struct {
	// ServiceName identifies the affected service
	ServiceName string `json:"service_name"`

	// ImpactLevel describes severity of impact
	ImpactLevel RiskLevel `json:"impact_level"`

	// Description explains the expected impact
	Description string `json:"description"`

	// DurationEstimate predicts how long impact will last
	DurationEstimate time.Duration `json:"duration_estimate,omitempty"`
}

// OutcomeBranch describes a possible scenario outcome.
type OutcomeBranch struct {
	// Scenario identifies this outcome (e.g., "best_case", "expected", "worst_case")
	Scenario string `json:"scenario"`

	// Probability of this scenario occurring (0.0-1.0)
	Probability float64 `json:"probability"`

	// Description provides human-readable scenario explanation
	Description string `json:"description"`

	// SystemState describes the resulting system state
	SystemState string `json:"system_state"`

	// HealthScore predicts resulting mesh health score (0-100)
	HealthScore int `json:"health_score,omitempty"`

	// RecoveryTime estimates time to recover if this outcome occurs
	RecoveryTime time.Duration `json:"recovery_time,omitempty"`

	// TriggeringConditions lists conditions that would lead to this outcome
	TriggeringConditions []string `json:"triggering_conditions,omitempty"`
}

// SafeToActDecision provides clear operator guidance with supporting reasons.
type SafeToActDecision struct {
	// SafeToAct indicates if the action is deemed safe to execute
	SafeToAct bool `json:"safe_to_act"`

	// Decision provides the categorical decision
	Decision SafetyLevel `json:"decision"`

	// PrimaryReason is the main factor in this decision
	PrimaryReason string `json:"primary_reason"`

	// SupportingReasons lists additional factors considered
	SupportingReasons []string `json:"supporting_reasons,omitempty"`

	// OperatorGuidance provides actionable guidance
	OperatorGuidance string `json:"operator_guidance"`

	// RequiresAcknowledgment indicates if operator acknowledgment is required
	RequiresAcknowledgment bool `json:"requires_acknowledgment"`

	// RecommendedAction suggests the best course of action
	RecommendedAction string `json:"recommended_action,omitempty"`

	// AlternativeActions suggests safer alternatives if primary is not safe
	AlternativeActions []string `json:"alternative_actions,omitempty"`

	// Confidence in this decision (0.0-1.0)
	Confidence float64 `json:"confidence"`
}

// SimulationMetadata contains additional information about the simulation.
type SimulationMetadata struct {
	// Duration indicates how long the simulation took
	Duration time.Duration `json:"duration"`

	// ModelVersion identifies the simulation model version
	ModelVersion string `json:"model_version"`

	// InputHash provides a hash of input data for integrity
	InputHash string `json:"input_hash,omitempty"`

	// Warnings lists any non-fatal warnings
	Warnings []string `json:"warnings,omitempty"`

	// Assumptions lists assumptions made during simulation
	Assumptions []string `json:"assumptions,omitempty"`

	// Limitations notes any simulation limitations
	Limitations []string `json:"limitations,omitempty"`
}

// SimulationEngine defines the interface for simulation implementations.
type SimulationEngine interface {
	// Simulate runs a simulation for the given input
	Simulate(input SimulationInput) (SimulationResult, error)

	// SupportsAction indicates if this engine can simulate the action type
	SupportsAction(actionType string) bool

	// Version returns the engine version
	Version() string
}

// SimulationRequest wraps a simulation request with additional context.
type SimulationRequest struct {
	// Input is the simulation input data
	Input SimulationInput `json:"input"`

	// RequestedAt when the simulation was requested
	RequestedAt time.Time `json:"requested_at"`

	// RequestedBy identifies who/what requested the simulation
	RequestedBy string `json:"requested_by,omitempty"`

	// Priority indicates simulation priority (higher = more urgent)
	Priority int `json:"priority,omitempty"`

	// Timeout for simulation completion
	Timeout time.Duration `json:"timeout,omitempty"`

	// Options contains simulation-specific options
	Options SimulationOptions `json:"options,omitempty"`
}

// SimulationOptions provides configuration for simulation behavior.
type SimulationOptions struct {
	// Depth controls simulation depth (1=basic, 2=standard, 3=deep)
	Depth int `json:"depth,omitempty"`

	// IncludeBranches indicates if outcome branches should be calculated
	IncludeBranches bool `json:"include_branches,omitempty"`

	// MaxBranches limits the number of outcome branches
	MaxBranches int `json:"max_branches,omitempty"`

	// HistoricalLookback defines how far back to look for patterns
	HistoricalLookback time.Duration `json:"historical_lookback,omitempty"`

	// ConfidenceThreshold minimum confidence for predictions
	ConfidenceThreshold float64 `json:"confidence_threshold,omitempty"`

	// ExtraContext allows passing additional simulation parameters
	ExtraContext map[string]any `json:"extra_context,omitempty"`
}
