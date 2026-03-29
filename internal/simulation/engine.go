package simulation

import (
	"fmt"
	"sort"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/diagnostics"
	"github.com/mel-project/mel/internal/status"
	"github.com/mel-project/mel/internal/transport"
)

// Engine provides bounded, deterministic simulation of control actions.
// It implements the SimulationEngine interface defined in types.go.
type Engine struct {
	cfg      config.Config
	database *db.DB
}

// NewEngine creates a new simulation engine with the given configuration and database.
func NewEngine(cfg config.Config, database *db.DB) *Engine {
	return &Engine{
		cfg:      cfg,
		database: database,
	}
}

// Simulate runs a bounded, deterministic simulation of a candidate control action.
// It implements the SimulationEngine interface and returns a comprehensive SimulationResult
// containing predicted outcomes, risk assessments, and explainable reasoning.
func (e *Engine) Simulate(input SimulationInput) (SimulationResult, error) {
	startTime := time.Now()

	// Evaluate preconditions
	preconditionsMet, _ := e.evaluatePreconditions(input)

	// Identify affected components based on action type and current state
	affectedTransports, affectedNodes, affectedSegments := e.identifyAffectedComponents(input)

	// Assess dependencies for the action
	dependencyAssessments := e.assessDependencies(input)

	// Predict the likely outcome based on action type and evidence
	predictedOutcome := e.predictOutcome(input, preconditionsMet)

	// Assess risks
	riskAssessment := e.assessRisk(input, affectedTransports, dependencyAssessments)

	// Check for conflicts
	conflicts := e.detectConflicts(input)

	// Predict blast radius
	blastRadius := e.predictBlastRadius(input, affectedTransports, affectedNodes, affectedSegments)

	// Generate outcome branches
	outcomeBranches := e.generateOutcomeBranches(input, predictedOutcome, riskAssessment)

	// Make final safety decision
	safeToAct := e.makeSafeToActDecision(input, preconditionsMet, riskAssessment, conflicts)

	// Build policy preview
	policyPreview := e.buildPolicyPreview(input, preconditionsMet)

	// Build metadata
	metadata := SimulationMetadata{
		Duration:     time.Since(startTime),
		ModelVersion: "1.0.0",
		Assumptions:  e.getAssumptions(),
		Limitations:  e.getLimitations(),
	}

	return SimulationResult{
		SimulationID:     input.SimulationID,
		CompletedAt:      time.Now(),
		PredictedOutcome: predictedOutcome,
		RiskAssessment:   riskAssessment,
		PolicyPreview:    policyPreview,
		Conflicts:        conflicts,
		BlastRadius:      blastRadius,
		OutcomeBranches:  outcomeBranches,
		SafeToAct:        safeToAct,
		Metadata:         metadata,
	}, nil
}

// SupportsAction indicates if this engine can simulate the action type.
func (e *Engine) SupportsAction(actionType string) bool {
	supportedTypes := []string{
		control.ActionRestartTransport,
		control.ActionResubscribeTransport,
		control.ActionBackoffIncrease,
		control.ActionBackoffReset,
		control.ActionTriggerHealthRecheck,
		control.ActionTemporarilyDeprioritize,
		control.ActionTemporarilySuppressNoisySource,
		control.ActionClearSuppression,
	}
	for _, t := range supportedTypes {
		if t == actionType {
			return true
		}
	}
	return false
}

// Version returns the engine version.
func (e *Engine) Version() string {
	return "1.0.0"
}

// evaluatePreconditions checks if required preconditions for an action are met.
// Returns true if preconditions are satisfied, along with explanatory notes.
func (e *Engine) evaluatePreconditions(input SimulationInput) (bool, []string) {
	notes := []string{}
	met := true
	action := input.ProposedAction

	// Check action reality - does the actuator exist?
	reality := e.getActionReality(action.ActionType)
	if !reality.ActuatorExists {
		notes = append(notes, fmt.Sprintf("Actuator for %s does not exist in this build", action.ActionType))
		met = false
	}

	// Check if action is advisory-only
	if reality.AdvisoryOnly {
		notes = append(notes, fmt.Sprintf("Action %s is advisory-only: %s", action.ActionType, reality.Notes))
		met = false
	}

	// Check mode-based preconditions
	if e.cfg.Control.Mode == control.ModeDisabled {
		notes = append(notes, "Control mode is disabled")
		met = false
	}

	if e.cfg.Control.Mode == control.ModeAdvisory && !reality.SafeForGuardedAuto {
		notes = append(notes, "Control mode is advisory; action not safe for guarded auto")
		met = false
	}

	// Check emergency disable
	if e.cfg.Control.EmergencyDisable {
		notes = append(notes, "Emergency disable is active")
		met = false
	}

	// Check confidence threshold
	if action.Confidence < e.cfg.Control.RequireMinConfidence {
		notes = append(notes, fmt.Sprintf("Action confidence %.2f below required threshold %.2f",
			action.Confidence, e.cfg.Control.RequireMinConfidence))
		met = false
	}

	// Check target transport exists and is accessible
	if action.TargetTransport != "" {
		transportHealth := e.findTransportHealth(action.TargetTransport, input.TransportHealth)
		if transportHealth.Name == "" {
			// Check if it's in config at least
			found := false
			for _, tc := range e.cfg.Transports {
				if tc.Name == action.TargetTransport {
					found = true
					break
				}
			}
			if !found {
				notes = append(notes, fmt.Sprintf("Target transport %s not found in configuration", action.TargetTransport))
				met = false
			} else {
				notes = append(notes, fmt.Sprintf("Target transport %s configured but not currently connected", action.TargetTransport))
			}
		} else {
			notes = append(notes, fmt.Sprintf("Target transport %s found with state %s",
				action.TargetTransport, transportHealth.State))
		}
	}

	// Check for conflicting active actions
	if e.hasConflictingActiveAction(action, input.MeshTopology) {
		notes = append(notes, "Conflicting action already active for this target")
		met = false
	}

	// Check blast radius is known for safety
	if !reality.BlastRadiusKnown {
		notes = append(notes, "Blast radius unknown; action not safe for automated execution")
		met = false
	}

	if met && len(notes) == 0 {
		notes = append(notes, "All preconditions satisfied")
	}

	return met, notes
}

// identifyAffectedComponents determines which components would be affected by an action.
// Returns lists of affected transports, nodes, and segments.
func (e *Engine) identifyAffectedComponents(input SimulationInput) ([]string, []string, []string) {
	action := input.ProposedAction
	mesh := input.MeshTopology

	affectedTransports := []string{}
	affectedNodes := []string{}
	affectedSegments := []string{}

	// Always include the target transport
	if action.TargetTransport != "" {
		affectedTransports = append(affectedTransports, action.TargetTransport)
	}

	// For mesh-level actions, identify related transports
	if action.TargetSegment != "" {
		for _, segment := range mesh.DegradedSegments {
			if segment.SegmentID == action.TargetSegment {
				affectedSegments = append(affectedSegments, segment.SegmentID)
				for _, transportName := range segment.Transports {
					if transportName != action.TargetTransport {
						affectedTransports = appendUnique(affectedTransports, transportName)
					}
				}
				for _, nodeID := range segment.Nodes {
					affectedNodes = appendUnique(affectedNodes, nodeID)
				}
			}
		}
	}

	// Identify correlated transports for correlated failure actions
	for _, failure := range mesh.CorrelatedFailures {
		for _, transportName := range failure.Transports {
			affectedTransports = appendUnique(affectedTransports, transportName)
		}
		for _, nodeID := range failure.NodeIDs {
			affectedNodes = appendUnique(affectedNodes, nodeID)
		}
	}

	// Sort for determinism
	sort.Strings(affectedTransports)
	sort.Strings(affectedNodes)
	sort.Strings(affectedSegments)

	return affectedTransports, affectedNodes, affectedSegments
}

// assessDependencies evaluates dependency confidence for an action.
func (e *Engine) assessDependencies(input SimulationInput) []DependencyAssessment {
	action := input.ProposedAction
	mesh := input.MeshTopology
	runtime := input.TransportHealth

	assessments := []DependencyAssessment{}

	// Database dependency
	dbConfidence := 0.95
	dbEvidence := "Database required for action persistence"
	if e.database == nil {
		dbConfidence = 0.5
		dbEvidence = "Database not available; action will have limited persistence"
	}
	assessments = append(assessments, DependencyAssessment{
		DependencyID:   "database",
		DependencyType: "storage",
		Confidence:     dbConfidence,
		Evidence:       dbEvidence,
		IsCritical:     true,
	})

	// Transport dependency (if applicable)
	if action.TargetTransport != "" {
		health := e.findTransportHealth(action.TargetTransport, runtime)
		transportConfidence := 0.9
		if health.State == transport.StateFailed || health.State == transport.StateDisconnected {
			transportConfidence = 0.4
		}
		assessments = append(assessments, DependencyAssessment{
			DependencyID:   action.TargetTransport,
			DependencyType: "transport",
			Confidence:     transportConfidence,
			Evidence:       fmt.Sprintf("Transport state: %s", health.State),
			IsCritical:     true,
		})
	}

	// Configuration dependency
	configConfidence := 0.95
	configEvidence := "Configuration validated"
	if e.cfg.Control.Mode == control.ModeDisabled {
		configConfidence = 0.0
		configEvidence = "Control mode disabled in configuration"
	}
	assessments = append(assessments, DependencyAssessment{
		DependencyID:   "configuration",
		DependencyType: "config",
		Confidence:     configConfidence,
		Evidence:       configEvidence,
		IsCritical:     true,
	})

	// Alternate path dependency (for deprioritize actions)
	if action.ActionType == control.ActionTemporarilyDeprioritize {
		alternateExists := false
		for _, route := range mesh.RoutingRecommendations {
			if route.Action == "suggest_alternate_ingest_path" && route.TargetTransport == action.TargetTransport {
				alternateExists = true
				break
			}
		}
		alternateConfidence := 0.7
		if !alternateExists {
			alternateConfidence = 0.3
		}
		assessments = append(assessments, DependencyAssessment{
			DependencyID:   "alternate_path",
			DependencyType: "routing",
			Confidence:     alternateConfidence,
			Evidence:       fmt.Sprintf("Alternate path available: %v", alternateExists),
			IsCritical:     false,
		})
	}

	// Sort for determinism
	sort.Slice(assessments, func(i, j int) bool {
		return assessments[i].DependencyID < assessments[j].DependencyID
	})

	return assessments
}

// predictOutcome predicts likely state transitions based on action type and current state.
func (e *Engine) predictOutcome(input SimulationInput, preconditionsMet bool) PredictedOutcome {
	action := input.ProposedAction
	runtime := input.TransportHealth

	if !preconditionsMet {
		return PredictedOutcome{
			SuccessProbability: 0.0,
			ExpectedState:      "unchanged",
			Explanation:        "Preconditions not met; action will not execute",
			RollbackCapability: RollbackInfo{
				Reversible: true,
				Automatic:  false,
				Notes:      "No action taken, no rollback needed",
			},
		}
	}

	transportHealth := e.findTransportHealth(action.TargetTransport, runtime)
	currentState := transportHealth.State
	if currentState == "" {
		currentState = transport.StateConfigured
	}

	switch action.ActionType {
	case control.ActionRestartTransport:
		return e.predictRestartOutcome(action, currentState, transportHealth)
	case control.ActionResubscribeTransport:
		return e.predictResubscribeOutcome(action, currentState, transportHealth)
	case control.ActionBackoffIncrease:
		return e.predictBackoffIncreaseOutcome(action, currentState, transportHealth)
	case control.ActionBackoffReset:
		return e.predictBackoffResetOutcome(action, currentState, transportHealth)
	case control.ActionTriggerHealthRecheck:
		return e.predictHealthRecheckOutcome(action, currentState, transportHealth)
	case control.ActionTemporarilyDeprioritize:
		return e.predictDeprioritizeOutcome(action, currentState, transportHealth, input.MeshTopology)
	case control.ActionTemporarilySuppressNoisySource:
		return e.predictSuppressOutcome(action, currentState, transportHealth, input.MeshTopology)
	case control.ActionClearSuppression:
		return e.predictClearSuppressionOutcome(action, currentState, transportHealth)
	default:
		return PredictedOutcome{
			SuccessProbability: 0.5,
			ExpectedState:      "unknown",
			Explanation:        "Unknown action type; outcome cannot be predicted",
			RollbackCapability: RollbackInfo{
				Reversible: false,
				Automatic:  false,
				Notes:      "Unknown action, rollback capability uncertain",
			},
		}
	}
}

func (e *Engine) predictRestartOutcome(action control.ControlAction, currentState string, health transport.Health) PredictedOutcome {
	finalState := transport.StateLive
	if health.FailureCount > 5 {
		finalState = transport.StateRetrying
	}

	sideEffects := []PredictedEffect{
		{
			Component:  action.TargetTransport,
			Effect:     "Transport connection interrupted",
			Severity:   RiskLevelMedium,
			Likelihood: 1.0,
			Mitigation: "Connection will be automatically re-established",
		},
		{
			Component:  "message_ingest",
			Effect:     "Temporary ingestion pause",
			Severity:   RiskLevelLow,
			Likelihood: 0.9,
			Mitigation: "Buffered messages will be processed after reconnect",
		},
	}

	return PredictedOutcome{
		SuccessProbability: 0.85,
		ExpectedState:      finalState,
		ExpectedDuration:   5 * time.Second,
		SideEffects:        sideEffects,
		Explanation:        fmt.Sprintf("Transport restart will transition through %s -> %s -> %s", currentState, transport.StateDisconnected, finalState),
		RollbackCapability: RollbackInfo{
			Reversible:       true,
			Automatic:        false,
			RollbackDuration: 5 * time.Second,
			Notes:            "Restart is reversible by allowing normal reconnect",
		},
	}
}

func (e *Engine) predictResubscribeOutcome(action control.ControlAction, currentState string, health transport.Health) PredictedOutcome {
	sideEffects := []PredictedEffect{
		{
			Component:  action.TargetTransport,
			Effect:     "Subscription renewed",
			Severity:   RiskLevelLow,
			Likelihood: 0.88,
			Mitigation: "Messages may be missed during brief resubscribe window",
		},
	}

	return PredictedOutcome{
		SuccessProbability: 0.88,
		ExpectedState:      transport.StateLive,
		ExpectedDuration:   3 * time.Second,
		SideEffects:        sideEffects,
		Explanation:        fmt.Sprintf("Resubscribe will transition through %s -> %s", currentState, transport.StateLive),
		RollbackCapability: RollbackInfo{
			Reversible:       true,
			Automatic:        true,
			RollbackDuration: 1 * time.Second,
			Notes:            "Resubscribe is automatically reversible",
		},
	}
}

func (e *Engine) predictBackoffIncreaseOutcome(action control.ControlAction, currentState string, health transport.Health) PredictedOutcome {
	sideEffects := []PredictedEffect{
		{
			Component:  "retry_timing",
			Effect:     "Reconnection attempts will be delayed",
			Severity:   RiskLevelLow,
			Likelihood: 0.95,
			Mitigation: "Backoff can be reset with backoff_reset action",
		},
	}

	return PredictedOutcome{
		SuccessProbability: 0.95,
		ExpectedState:      currentState,
		ExpectedDuration:   0,
		SideEffects:        sideEffects,
		Explanation:        "Backoff increase affects retry timing without changing transport state",
		RollbackCapability: RollbackInfo{
			Reversible:       true,
			Automatic:        false,
			RollbackDuration: 0,
			Notes:            "Use backoff_reset to reverse",
		},
	}
}

func (e *Engine) predictBackoffResetOutcome(action control.ControlAction, currentState string, health transport.Health) PredictedOutcome {
	sideEffects := []PredictedEffect{
		{
			Component:  "retry_timing",
			Effect:     "Normal retry timing restored",
			Severity:   RiskLevelNone,
			Likelihood: 0.95,
		},
	}

	return PredictedOutcome{
		SuccessProbability: 0.95,
		ExpectedState:      currentState,
		ExpectedDuration:   0,
		SideEffects:        sideEffects,
		Explanation:        "Backoff reset restores normal retry timing without changing transport state",
		RollbackCapability: RollbackInfo{
			Reversible:       true,
			Automatic:        false,
			RollbackDuration: 0,
			Notes:            "Backoff can be increased again if needed",
		},
	}
}

func (e *Engine) predictHealthRecheckOutcome(action control.ControlAction, currentState string, health transport.Health) PredictedOutcome {
	sideEffects := []PredictedEffect{
		{
			Component:  "health_status",
			Effect:     "Health metrics refreshed",
			Severity:   RiskLevelNone,
			Likelihood: 0.90,
		},
	}

	return PredictedOutcome{
		SuccessProbability: 0.95,
		ExpectedState:      currentState,
		ExpectedDuration:   2 * time.Second,
		SideEffects:        sideEffects,
		Explanation:        "Health recheck updates status without changing transport state",
		RollbackCapability: RollbackInfo{
			Reversible:       true,
			Automatic:        true,
			RollbackDuration: 0,
			Notes:            "No state change to rollback",
		},
	}
}

func (e *Engine) predictDeprioritizeOutcome(action control.ControlAction, currentState string, health transport.Health, mesh status.MeshDrilldown) PredictedOutcome {
	return PredictedOutcome{
		SuccessProbability: 0.85,
		ExpectedState:      currentState,
		ExpectedDuration:   0,
		SideEffects: []PredictedEffect{
			{Component: "ingest", Effect: "Adds bounded per-packet delay in MEL ingest workers for the target transport until expiry"},
		},
		Explanation: "MEL applies ingest-side deprioritization only; Meshtastic RF routing is unchanged",
		RollbackCapability: RollbackInfo{
			Reversible:       true,
			Automatic:        true,
			RollbackDuration: 0,
			Notes:            "Expires automatically; clear_suppression or transport restart also clears state",
		},
	}
}

func (e *Engine) predictSuppressOutcome(action control.ControlAction, currentState string, health transport.Health, mesh status.MeshDrilldown) PredictedOutcome {
	return PredictedOutcome{
		SuccessProbability: 0.8,
		ExpectedState:      currentState,
		ExpectedDuration:   0,
		SideEffects: []PredictedEffect{
			{Component: "ingest", Effect: "Decoded packets from target_node on the named transport are dropped until expiry (no SQLite write)"},
		},
		Explanation: "Ingest-level drop for one attributed node; mesh RF and other observers may still see traffic",
		RollbackCapability: RollbackInfo{
			Reversible:       true,
			Automatic:        true,
			RollbackDuration: 0,
			Notes:            "Expires automatically; clear_suppression clears windows",
		},
	}
}

func (e *Engine) predictClearSuppressionOutcome(action control.ControlAction, currentState string, health transport.Health) PredictedOutcome {
	return PredictedOutcome{
		SuccessProbability: 0.95,
		ExpectedState:      currentState,
		ExpectedDuration:   0,
		SideEffects: []PredictedEffect{
			{Component: "ingest", Effect: "Clears deprioritization and per-node suppression windows for the transport"},
		},
		Explanation: "Restores normal ingest timing and stops dropping suppressed node packets",
		RollbackCapability: RollbackInfo{
			Reversible: true,
			Automatic:  true,
			Notes:      "Minimal risk; only clears local MEL ingest actuator state",
		},
	}
}

// assessRisk evaluates overall risk for the proposed action.
func (e *Engine) assessRisk(input SimulationInput, affectedTransports []string, dependencyAssessments []DependencyAssessment) RiskAssessment {
	action := input.ProposedAction
	mesh := input.MeshTopology

	// Determine overall risk level
	riskLevel := RiskLevelLow
	safetyLevel := SafetyLevelSafe

	switch action.ActionType {
	case control.ActionRestartTransport:
		riskLevel = RiskLevelMedium
		safetyLevel = SafetyLevelCaution
	case control.ActionResubscribeTransport:
		riskLevel = RiskLevelLow
		safetyLevel = SafetyLevelSafe
	case control.ActionTemporarilyDeprioritize:
		riskLevel = RiskLevelLow
		safetyLevel = SafetyLevelSafe
	case control.ActionTemporarilySuppressNoisySource:
		riskLevel = RiskLevelMedium
		safetyLevel = SafetyLevelCaution
	}

	// Build risk factors
	riskFactors := []RiskFactor{}

	// Check reality
	reality := e.getActionReality(action.ActionType)
	if !reality.Reversible {
		riskFactors = append(riskFactors, RiskFactor{
			Category:    "action",
			Description: "Action is not reversible",
			Level:       RiskLevelMedium,
			Likelihood:  1.0,
			Impact:      "Cannot undo action if outcome is negative",
			Mitigatable: false,
		})
	}

	if !reality.BlastRadiusKnown {
		riskFactors = append(riskFactors, RiskFactor{
			Category:    "action",
			Description: "Blast radius unknown",
			Level:       RiskLevelHigh,
			Likelihood:  1.0,
			Impact:      "Cannot predict full scope of impact",
			Mitigatable: false,
		})
	}

	// Check mesh health
	if mesh.MeshHealth.Score < 50 {
		riskFactors = append(riskFactors, RiskFactor{
			Category:    "mesh",
			Description: "Mesh health is poor",
			Level:       RiskLevelMedium,
			Likelihood:  0.8,
			Impact:      "Action may exacerbate existing issues",
			Mitigatable: true,
		})
	}

	// Calculate confidence
	confidence := action.Confidence
	for _, dep := range dependencyAssessments {
		if dep.IsCritical {
			confidence *= dep.Confidence
		}
	}

	return RiskAssessment{
		OverallRisk:      riskLevel,
		SafetyLevel:      safetyLevel,
		RiskFactors:      riskFactors,
		Confidence:       confidence,
		Explanation:      fmt.Sprintf("Risk assessment based on action type %s and current mesh state", action.ActionType),
		EvidenceLossRisk: action.ActionType == control.ActionRestartTransport || action.ActionType == control.ActionResubscribeTransport,
		ConnectivityRisk: action.ActionType == control.ActionRestartTransport,
	}
}

// detectConflicts identifies conflicts with other actions or system state.
func (e *Engine) detectConflicts(input SimulationInput) []ConflictReport {
	action := input.ProposedAction
	mesh := input.MeshTopology
	conflicts := []ConflictReport{}

	// Check for active alerts on same transport
	for _, alert := range mesh.ActiveAlerts {
		if alert.TransportName == action.TargetTransport {
			// Check if action addresses the alert
			switch action.ActionType {
			case control.ActionRestartTransport:
				if alert.Reason != transport.ReasonRetryThresholdExceeded {
					conflicts = append(conflicts, ConflictReport{
						ConflictID:     fmt.Sprintf("alert_%s", alert.ID),
						Severity:       ConflictSeverityMinor,
						Type:           "active_alert",
						Description:    fmt.Sprintf("Active alert %s exists on target transport", alert.Reason),
						Resource:       action.TargetTransport,
						Resolution:     "Verify action addresses the alert condition",
						AutoResolvable: false,
					})
				}
			}
		}
	}

	return conflicts
}

// predictBlastRadius estimates the scope of impact from an action.
func (e *Engine) predictBlastRadius(input SimulationInput, affectedTransports, affectedNodes, affectedSegments []string) BlastRadiusPrediction {
	action := input.ProposedAction

	score := 0.1
	classification := "local_transport"

	// Adjust based on action type
	switch action.ActionType {
	case control.ActionRestartTransport:
		score = 0.3
		classification = "local_transport"
	case control.ActionTemporarilyDeprioritize:
		score = 0.4
		classification = "mesh_level"
	case control.ActionTemporarilySuppressNoisySource:
		score = 0.2
		classification = "source_specific"
	}

	// Adjust based on number of affected components
	score += float64(len(affectedTransports)) * 0.05
	score += float64(len(affectedNodes)) * 0.02

	if score > 1.0 {
		score = 1.0
	}

	return BlastRadiusPrediction{
		Score:              score,
		Classification:     classification,
		Description:        fmt.Sprintf("Action affects %d transports, %d nodes", len(affectedTransports), len(affectedNodes)),
		AffectedTransports: affectedTransports,
		AffectedNodes:      affectedNodes,
		AffectedSegments:   affectedSegments,
		Confidence:         0.85,
	}
}

// generateOutcomeBranches describes best/expected/worst case scenarios.
func (e *Engine) generateOutcomeBranches(input SimulationInput, predictedOutcome PredictedOutcome, riskAssessment RiskAssessment) []OutcomeBranch {
	branches := []OutcomeBranch{
		{
			Scenario:    "expected",
			Probability: predictedOutcome.SuccessProbability,
			Description: predictedOutcome.Explanation,
			SystemState: predictedOutcome.ExpectedState,
			HealthScore: 75,
		},
		{
			Scenario:    "best_case",
			Probability: predictedOutcome.SuccessProbability * 1.1,
			Description: "Action succeeds faster than expected with no side effects",
			SystemState: "healthy",
			HealthScore: 90,
		},
		{
			Scenario:     "worst_case",
			Probability:  1.0 - predictedOutcome.SuccessProbability,
			Description:  "Action fails or causes additional issues",
			SystemState:  "degraded",
			HealthScore:  40,
			RecoveryTime: 30 * time.Second,
		},
	}

	// Normalize probabilities
	totalProb := 0.0
	for _, b := range branches {
		totalProb += b.Probability
	}
	if totalProb > 0 {
		for i := range branches {
			branches[i].Probability /= totalProb
		}
	}

	return branches
}

// makeSafeToActDecision provides final operator guidance.
func (e *Engine) makeSafeToActDecision(input SimulationInput, preconditionsMet bool, riskAssessment RiskAssessment, conflicts []ConflictReport) SafeToActDecision {
	action := input.ProposedAction

	safeToAct := preconditionsMet && riskAssessment.SafetyLevel != SafetyLevelUnsafe

	decision := SafetyLevelSafe
	if !safeToAct {
		decision = SafetyLevelUnsafe
	} else if riskAssessment.SafetyLevel == SafetyLevelCaution {
		decision = SafetyLevelCaution
	}

	primaryReason := "All preconditions satisfied and risk acceptable"
	if !preconditionsMet {
		primaryReason = "Preconditions not met"
	} else if riskAssessment.SafetyLevel == SafetyLevelUnsafe {
		primaryReason = "Risk assessment indicates unsafe conditions"
	}

	guidance := fmt.Sprintf("Action %s is %s to execute", action.ActionType, decision)
	if decision == SafetyLevelCaution {
		guidance = "Review risk factors before proceeding"
	}

	return SafeToActDecision{
		SafeToAct:              safeToAct,
		Decision:               decision,
		PrimaryReason:          primaryReason,
		OperatorGuidance:       guidance,
		RequiresAcknowledgment: riskAssessment.OverallRisk == RiskLevelHigh || riskAssessment.OverallRisk == RiskLevelCritical,
		Confidence:             riskAssessment.Confidence,
	}
}

// buildPolicyPreview creates the policy admission check result.
func (e *Engine) buildPolicyPreview(input SimulationInput, preconditionsMet bool) PolicyPreview {
	result := AdmissionAllowed
	allowed := preconditionsMet
	denialCode := ""
	denialReason := ""

	if !preconditionsMet {
		result = AdmissionDenied
		allowed = false
		denialCode = control.DenialPolicy
		denialReason = "Simulation preconditions not satisfied"
	}

	if e.cfg.Control.Mode == control.ModeAdvisory {
		result = AdmissionAdvisory
		allowed = false
	}

	return PolicyPreview{
		Result:            result,
		Allowed:           allowed,
		DenialCode:        denialCode,
		DenialReason:      denialReason,
		Mode:              e.cfg.Control.Mode,
		ChecksPassed:      []string{"simulation_run"},
		OverrideAvailable: e.cfg.Control.Mode != control.ModeDisabled,
	}
}

// Helper methods

func (e *Engine) getActionReality(actionType string) control.ActionReality {
	realityMap := control.ActionRealityByType()
	if reality, ok := realityMap[actionType]; ok {
		return reality
	}
	return control.ActionReality{
		ActionType:       actionType,
		ActuatorExists:   false,
		BlastRadiusKnown: false,
		BlastRadiusClass: "unknown",
	}
}

func (e *Engine) findTransportHealth(name string, runtime []transport.Health) transport.Health {
	for _, h := range runtime {
		if h.Name == name {
			return h
		}
	}
	return transport.Health{}
}

func (e *Engine) hasConflictingActiveAction(action control.ControlAction, mesh status.MeshDrilldown) bool {
	for _, alert := range mesh.ActiveAlerts {
		if alert.TransportName == action.TargetTransport {
			switch action.ActionType {
			case control.ActionRestartTransport:
				if alert.Reason == transport.ReasonRetryThresholdExceeded {
					return false
				}
			case control.ActionResubscribeTransport:
				if alert.Reason == transport.ReasonSubscribeFailure {
					return false
				}
			}
		}
	}
	return false
}

func (e *Engine) getAssumptions() []string {
	return []string{
		"Transport health signals are fresh within 30 seconds",
		"Mesh topology reflects current state",
		"Configuration has been validated",
	}
}

func (e *Engine) getLimitations() []string {
	return []string{
		"Simulation does not account for network latency variations",
		"Predictions based on historical patterns may not reflect real-time conditions",
		"Advisory-only actions have no actual effect in current implementation",
	}
}

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

// DependencyAssessment evaluates confidence in a dependency relationship.
type DependencyAssessment struct {
	DependencyID   string  `json:"dependency_id"`
	DependencyType string  `json:"dependency_type"`
	Confidence     float64 `json:"confidence"`
	Evidence       string  `json:"evidence"`
	IsCritical     bool    `json:"is_critical"`
}

// SimulateSimple provides a simplified simulation interface for basic use cases.
// It takes a candidate action and returns a SimulationResult.
func (e *Engine) SimulateSimple(candidateAction control.ControlAction, mesh status.MeshDrilldown, runtime []transport.Health) (SimulationResult, error) {
	input := SimulationInput{
		SimulationID:    fmt.Sprintf("sim-%d", time.Now().UnixNano()),
		Timestamp:       time.Now(),
		ProposedAction:  candidateAction,
		TransportHealth: runtime,
		MeshTopology:    mesh,
		PolicyConfig:    control.PolicyFromConfig(e.cfg),
	}
	return e.Simulate(input)
}

// SimulateBatch runs simulations for multiple candidate actions efficiently.
func (e *Engine) SimulateBatch(actions []control.ControlAction, mesh status.MeshDrilldown, runtime []transport.Health) ([]SimulationResult, error) {
	results := make([]SimulationResult, 0, len(actions))

	for _, action := range actions {
		result, err := e.SimulateSimple(action, mesh, runtime)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

// GenerateDiagnosticReport creates a diagnostic report based on simulation results.
func (e *Engine) GenerateDiagnosticReport(results []SimulationResult) diagnostics.DiagnosticReport {
	now := time.Now()
	diagnosticsList := []diagnostics.Diagnostic{}
	canAutoRecover := 0
	needsOperator := 0

	for _, result := range results {
		if result.SafeToAct.SafeToAct && result.PolicyPreview.Allowed {
			canAutoRecover++
		} else {
			needsOperator++
		}

		severity := diagnostics.SeverityInfo
		if result.RiskAssessment.OverallRisk == RiskLevelHigh {
			severity = diagnostics.SeverityWarning
		}
		if result.RiskAssessment.OverallRisk == RiskLevelCritical {
			severity = diagnostics.SeverityCritical
		}

		diag := diagnostics.Diagnostic{
			Code:                   fmt.Sprintf("sim_%s", result.SimulationID),
			Severity:               severity,
			Component:              diagnostics.ComponentControl,
			Title:                  fmt.Sprintf("Simulation: %s", result.PredictedOutcome.ExpectedState),
			Explanation:            result.PredictedOutcome.Explanation,
			LikelyCauses:           []string{result.PolicyPreview.DenialReason},
			RecommendedSteps:       []string{result.SafeToAct.RecommendedAction},
			Evidence:               map[string]any{"confidence": result.SafeToAct.Confidence, "risk": result.RiskAssessment.OverallRisk},
			CanAutoRecover:         result.SafeToAct.SafeToAct,
			OperatorActionRequired: !result.SafeToAct.SafeToAct,
			AffectedTransport:      "",
			GeneratedAt:            result.CompletedAt.Format(time.RFC3339),
		}
		diagnosticsList = append(diagnosticsList, diag)
	}

	// Count by severity
	criticalCount := 0
	warningCount := 0
	infoCount := 0
	for _, d := range diagnosticsList {
		switch d.Severity {
		case diagnostics.SeverityCritical:
			criticalCount++
		case diagnostics.SeverityWarning:
			warningCount++
		case diagnostics.SeverityInfo:
			infoCount++
		}
	}

	return diagnostics.DiagnosticReport{
		GeneratedAt: now,
		Summary: diagnostics.Summary{
			TotalCount:     len(diagnosticsList),
			CriticalCount:  criticalCount,
			WarningCount:   warningCount,
			InfoCount:      infoCount,
			CanAutoRecover: canAutoRecover,
			NeedsOperator:  needsOperator,
		},
		Diagnostics: diagnosticsList,
		RawEvidence: map[string]any{
			"simulation_count": len(results),
			"timestamp":        now.Format(time.RFC3339),
		},
	}
}
