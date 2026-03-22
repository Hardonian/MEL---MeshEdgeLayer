package control

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	statuspkg "github.com/mel-project/mel/internal/status"
	"github.com/mel-project/mel/internal/transport"
)

var (
	startupTime     = time.Now().UTC()
	startupTimeOnce sync.Once
)

const (
	MaxActionsPerWindowCap    = 100
	MinCooldownPerTargetSecs  = 10
	MaxActionTimeoutSeconds   = 300
	StartupGracePeriodSeconds = 30
)

const (
	ModeDisabled    = "disabled"
	ModeAdvisory    = "advisory"
	ModeGuardedAuto = "guarded_auto"

	ActionRestartTransport               = "restart_transport"
	ActionResubscribeTransport           = "resubscribe_transport"
	ActionBackoffIncrease                = "backoff_increase"
	ActionBackoffReset                   = "backoff_reset"
	ActionTemporarilyDeprioritize        = "temporarily_deprioritize_transport"
	ActionTemporarilySuppressNoisySource = "temporarily_suppress_noisy_source"
	ActionClearSuppression               = "clear_suppression"
	ActionTriggerHealthRecheck           = "trigger_health_recheck"

	ResultExecutedSuccessfully = "executed_successfully"
	ResultExecutedNoop         = "executed_noop"
	ResultDeniedByPolicy       = "denied_by_policy"
	ResultDeniedByCooldown     = "denied_by_cooldown"
	ResultFailedTransient      = "failed_transient"
	ResultFailedTerminal       = "failed_terminal"
	ResultExpired              = "expired"

	LifecyclePending   = "pending"
	LifecycleRunning   = "running"
	LifecycleCompleted = "completed"
	LifecycleRecovered = "recovered"

	ClosureRecoveredAndClosed = "recovered_and_closed"
	ClosureExpiredAndReverted = "expired_and_reverted"
	ClosureSuperseded         = "superseded"
	ClosureCanceledByOperator = "canceled_by_operator"

	DenialPolicy             = "policy"
	DenialMode               = "mode"
	DenialOverride           = "override"
	DenialLowConfidence      = "low_confidence"
	DenialTransient          = "transient"
	DenialCooldown           = "cooldown"
	DenialBudget             = "budget"
	DenialMissingActuator    = "missing_actuator"
	DenialUnknownBlastRadius = "unknown_blast_radius"
	DenialNoAlternatePath    = "no_alternate_path"
	DenialIrreversible       = "irreversible"
	DenialConflict           = "conflict"
	DenialAttributionWeak    = "attribution_weak"

	// ─── Execution modes ───────────────────────────────────────────────────────
	// ExecutionModeAuto: system executes automatically when safety checks pass.
	ExecutionModeAuto = "auto"
	// ExecutionModeApprovalRequired: action is held in pending_approval state
	// until an operator explicitly approves it via API/CLI.
	ExecutionModeApprovalRequired = "approval_required"
	// ExecutionModeManualOnly: action is never executed autonomously; operator
	// must execute it through an explicit API/CLI call.
	ExecutionModeManualOnly = "manual_only"
	// ExecutionModeDryRun: action goes through the full lifecycle but is never
	// sent to the actuator; used for testing policy and evidence capture.
	ExecutionModeDryRun = "dry_run"

	// ─── Additional lifecycle states ──────────────────────────────────────────
	// LifecyclePendingApproval: action proposed, waiting for operator approval.
	LifecyclePendingApproval = "pending_approval"

	// ─── Additional result codes ──────────────────────────────────────────────
	ResultPendingApproval     = "pending_approval"
	ResultApproved            = "approved"
	ResultRejected            = "rejected"
	ResultDeniedByFreeze      = "denied_by_freeze"
	ResultDeniedByMaintenance = "denied_by_maintenance"
	ResultApprovalExpired     = "approval_expired"
	ResultDryRun              = "dry_run_only"

	// ─── Additional denial codes ──────────────────────────────────────────────
	DenialFreeze           = "freeze"
	DenialMaintenance      = "maintenance_window"
	DenialApprovalRequired = "approval_required"
	DenialApprovalExpired  = "approval_expired"
	DenialManualOnly       = "manual_only"
	DenialDryRun           = "dry_run"

	// ─── Blast radius classes ─────────────────────────────────────────────────
	BlastRadiusLocal     = "local"
	BlastRadiusTransport = "transport"
	BlastRadiusMesh      = "mesh"
	BlastRadiusGlobal    = "global"
	BlastRadiusUnknown   = "unknown"

	// ─── Additional closure states ────────────────────────────────────────────
	ClosureRejectedByOperator   = "rejected_by_operator"
	ClosureApprovalExpired      = "approval_expired"
	ClosureBlockedByFreeze      = "blocked_by_freeze"
	ClosureBlockedByMaintenance = "blocked_by_maintenance"
	ClosureDryRun               = "dry_run_completed"
)

type ControlAction struct {
	ID              string         `json:"id"`
	DecisionID      string         `json:"decision_id,omitempty"`
	ActionType      string         `json:"action_type"`
	TargetTransport string         `json:"target_transport,omitempty"`
	TargetSegment   string         `json:"target_segment,omitempty"`
	TargetNode      string         `json:"target_node,omitempty"`
	Reason          string         `json:"reason"`
	Confidence      float64        `json:"confidence"`
	TriggerEvidence []string       `json:"trigger_evidence,omitempty"`
	EpisodeID       string         `json:"episode_id,omitempty"`
	CreatedAt       string         `json:"created_at"`
	ExecutedAt      string         `json:"executed_at,omitempty"`
	CompletedAt     string         `json:"completed_at,omitempty"`
	Result          string         `json:"result,omitempty"`
	Reversible      bool           `json:"reversible"`
	ExpiresAt       string         `json:"expires_at,omitempty"`
	OutcomeDetail   string         `json:"outcome_detail,omitempty"`
	Mode            string         `json:"mode"`
	PolicyRule      string         `json:"policy_rule,omitempty"`
	LifecycleState  string         `json:"lifecycle_state,omitempty"`
	AdvisoryOnly    bool           `json:"advisory_only,omitempty"`
	DenialCode      string         `json:"denial_code,omitempty"`
	ClosureState    string         `json:"closure_state,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`

	// Trust / approval fields (populated from migration 0017)
	ExecutionMode     string `json:"execution_mode,omitempty"`
	ProposedBy        string `json:"proposed_by,omitempty"`
	ApprovedBy        string `json:"approved_by,omitempty"`
	ApprovedAt        string `json:"approved_at,omitempty"`
	RejectedBy        string `json:"rejected_by,omitempty"`
	RejectedAt        string `json:"rejected_at,omitempty"`
	ApprovalNote      string `json:"approval_note,omitempty"`
	ApprovalExpiresAt string `json:"approval_expires_at,omitempty"`
	BlastRadiusClass  string `json:"blast_radius_class,omitempty"`
	BeforeStateJSON   string `json:"before_state_json,omitempty"`
	AfterStateJSON    string `json:"after_state_json,omitempty"`
	EvidenceBundleID  string `json:"evidence_bundle_id,omitempty"`
}

type ControlPolicy struct {
	Mode                   string   `json:"mode"`
	AllowedActions         []string `json:"allowed_actions"`
	MaxActionsPerWindow    int      `json:"max_actions_per_window"`
	CooldownPerTarget      int      `json:"cooldown_per_target_seconds"`
	RequireMinConfidence   float64  `json:"require_min_confidence"`
	AllowMeshLevelActions  bool     `json:"allow_mesh_level_actions"`
	AllowTransportRestart  bool     `json:"allow_transport_restart"`
	AllowSourceSuppression bool     `json:"allow_source_suppression"`
	ActionWindowSeconds    int      `json:"action_window_seconds"`
	RestartCapPerWindow    int      `json:"restart_cap_per_window"`
}

type ControlDecision struct {
	ID                 string         `json:"id"`
	CandidateAction    ControlAction  `json:"candidate_action"`
	Allowed            bool           `json:"allowed"`
	DenialReason       string         `json:"denial_reason,omitempty"`
	DenialCode         string         `json:"denial_code,omitempty"`
	Confidence         float64        `json:"confidence"`
	SafetyChecksPassed []string       `json:"safety_checks_passed,omitempty"`
	SafetyChecks       map[string]any `json:"safety_checks"`
	CreatedAt          string         `json:"created_at"`
	Mode               string         `json:"mode"`
	OperatorOverride   bool           `json:"operator_override"`
	InputSummary       map[string]any `json:"input_summary,omitempty"`
	PolicySummary      map[string]any `json:"policy_summary,omitempty"`
}

type ActionReality struct {
	ActionType         string `json:"action_type"`
	ActuatorExists     bool   `json:"actuator_exists"`
	Reversible         bool   `json:"reversible"`
	BlastRadiusKnown   bool   `json:"blast_radius_known"`
	BlastRadiusClass   string `json:"blast_radius_class"`
	SafeForGuardedAuto bool   `json:"safe_for_guarded_auto"`
	AdvisoryOnly       bool   `json:"advisory_only"`
	DenialCode         string `json:"denial_code,omitempty"`
	Notes              string `json:"notes,omitempty"`
}

type ControlExplanation struct {
	Mode             string            `json:"mode"`
	ActiveActions    []ControlAction   `json:"active_actions,omitempty"`
	RecentActions    []ControlAction   `json:"recent_actions,omitempty"`
	PendingActions   []ControlAction   `json:"pending_actions,omitempty"`
	DeniedActions    []ControlDecision `json:"denied_actions,omitempty"`
	PolicySummary    ControlPolicy     `json:"policy_summary"`
	RealityMatrix    []ActionReality   `json:"reality_matrix,omitempty"`
	ReasonsForDenial []string          `json:"reasons_for_denial,omitempty"`
	EmergencyDisable bool              `json:"emergency_disable"`
}

type Evaluation struct {
	Policy      ControlPolicy      `json:"policy"`
	Decisions   []ControlDecision  `json:"decisions"`
	Explanation ControlExplanation `json:"explanation"`
}

type runtimeSignals struct {
	connectedCount int
	byTransport    map[string]transport.Health
}

func DefaultActionRealityMatrix() []ActionReality {
	return []ActionReality{
		{ActionType: ActionBackoffIncrease, ActuatorExists: true, Reversible: true, BlastRadiusKnown: true, BlastRadiusClass: "local_transport", SafeForGuardedAuto: true, Notes: "Raises only the local reconnect backoff multiplier until expiry or reset."},
		{ActionType: ActionBackoffReset, ActuatorExists: true, Reversible: true, BlastRadiusKnown: true, BlastRadiusClass: "local_transport", SafeForGuardedAuto: true, Notes: "Restores the local reconnect backoff multiplier to baseline."},
		{ActionType: ActionClearSuppression, ActuatorExists: false, Reversible: false, BlastRadiusKnown: false, BlastRadiusClass: "unknown", AdvisoryOnly: true, DenialCode: DenialMissingActuator, Notes: "Suppression is not shipped as a real actuator in this build, so clear_suppression stays advisory-only."},
		{ActionType: ActionRestartTransport, ActuatorExists: true, Reversible: true, BlastRadiusKnown: true, BlastRadiusClass: "local_transport", SafeForGuardedAuto: true, Notes: "Interrupts only the named transport so the bounded reconnect loop can re-enter connect/subscribe."},
		{ActionType: ActionResubscribeTransport, ActuatorExists: true, Reversible: true, BlastRadiusKnown: true, BlastRadiusClass: "local_transport", SafeForGuardedAuto: true, Notes: "Interrupts only the named transport subscription path and relies on the existing reconnect/subscribe loop."},
		{ActionType: ActionTemporarilyDeprioritize, ActuatorExists: false, Reversible: false, BlastRadiusKnown: false, BlastRadiusClass: "unknown", AdvisoryOnly: true, DenialCode: DenialMissingActuator, Notes: "MEL does not currently own a verified live routing selector, so routing changes remain advisory."},
		{ActionType: ActionTemporarilySuppressNoisySource, ActuatorExists: false, Reversible: false, BlastRadiusKnown: false, BlastRadiusClass: "unknown", AdvisoryOnly: true, DenialCode: DenialMissingActuator, Notes: "MEL does not currently ship a verified source suppression actuator or metrics-backed suppression path."},
		{ActionType: ActionTriggerHealthRecheck, ActuatorExists: true, Reversible: true, BlastRadiusKnown: true, BlastRadiusClass: "local_process", SafeForGuardedAuto: true, Notes: "Schedules a bounded asynchronous health recheck without changing transport routing."},
	}
}

func ActionRealityByType() map[string]ActionReality {
	out := map[string]ActionReality{}
	for _, item := range DefaultActionRealityMatrix() {
		out[item.ActionType] = item
	}
	return out
}

func Evaluate(cfg config.Config, database *db.DB, runtime []transport.Health, now time.Time) (Evaluation, error) {
	mesh, err := statuspkg.InspectMesh(cfg, database, runtime, now)
	if err != nil {
		return Evaluation{}, err
	}
	policy := PolicyFromConfig(cfg)
	activeActions := []db.ControlActionRecord{}
	if database != nil {
		activeActions, _ = database.ControlActions("", "", now.Add(-time.Duration(policy.ActionWindowSeconds)*time.Second).Format(time.RFC3339), "", cfg.Intelligence.Queries.MaxLimit, 0)
	}
	decisions := make([]ControlDecision, 0)
	denialReasons := []string{}
	signals := buildRuntimeSignals(runtime)
	historyCache := map[string][]db.ControlActionRecord{}
	realityByType := ActionRealityByType()

	for _, candidate := range candidateActions(cfg, mesh, now) {
		decision := evaluateCandidate(cfg, database, policy, candidate, mesh, signals, activeActions, historyCache, realityByType, now)
		decisions = append(decisions, decision)
		if !decision.Allowed && decision.DenialReason != "" {
			denialReasons = append(denialReasons, decision.DenialReason)
		}
	}

	recent := []db.ControlActionRecord{}
	if database != nil {
		recent, _ = database.ControlActions("", "", now.Add(-24*time.Hour).Format(time.RFC3339), "", minPositive(cfg.Intelligence.Queries.DefaultLimit, 50), 0)
	}
	active := filterActiveActions(activeActions, now)
	pending := []db.ControlActionRecord{}
	if database != nil {
		pending, _ = database.IncompleteControlActions(minPositive(cfg.Intelligence.Queries.DefaultLimit, 50))
	}
	denied := make([]ControlDecision, 0)
	for _, decision := range decisions {
		if !decision.Allowed {
			denied = append(denied, decision)
		}
	}
	explanation := ControlExplanation{
		Mode:             policy.Mode,
		ActiveActions:    controlActionsFromRecords(active),
		RecentActions:    controlActionsFromRecords(recent),
		PendingActions:   controlActionsFromRecords(pending),
		DeniedActions:    denied,
		PolicySummary:    policy,
		RealityMatrix:    DefaultActionRealityMatrix(),
		ReasonsForDenial: dedupeStrings(denialReasons),
		EmergencyDisable: cfg.Control.EmergencyDisable,
	}
	return Evaluation{Policy: policy, Decisions: decisions, Explanation: explanation}, nil
}

func PolicyFromConfig(cfg config.Config) ControlPolicy {
	maxActions := cfg.Control.MaxActionsPerWindow
	if maxActions > MaxActionsPerWindowCap {
		maxActions = MaxActionsPerWindowCap
	}
	cooldown := cfg.Control.CooldownPerTargetSeconds
	if cooldown > 0 && cooldown < MinCooldownPerTargetSecs {
		cooldown = MinCooldownPerTargetSecs
	}
	timeout := cfg.Control.ActionTimeoutSeconds
	if timeout > MaxActionTimeoutSeconds {
		timeout = MaxActionTimeoutSeconds
	}
	return ControlPolicy{
		Mode:                   cfg.Control.Mode,
		AllowedActions:         append([]string(nil), cfg.Control.AllowedActions...),
		MaxActionsPerWindow:    maxActions,
		CooldownPerTarget:      cooldown,
		RequireMinConfidence:   cfg.Control.RequireMinConfidence,
		AllowMeshLevelActions:  cfg.Control.AllowMeshLevelActions,
		AllowTransportRestart:  cfg.Control.AllowTransportRestart,
		AllowSourceSuppression: cfg.Control.AllowSourceSuppression,
		ActionWindowSeconds:    cfg.Control.ActionWindowSeconds,
		RestartCapPerWindow:    cfg.Control.RestartCapPerWindow,
	}
}

func generateActionID(prefix string) string {
	nanos := time.Now().UTC().UnixNano()
	b := make([]byte, 8)
	rand.Read(b)
	randomPart := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%d-%s", prefix, nanos, randomPart[:16])
}

func StartupTime() time.Time {
	return startupTime
}

func WithinStartupGracePeriod(now time.Time) bool {
	delta := now.Sub(startupTime)
	if delta < 0 {
		// Historical / synthetic evaluation times (before process start) must not
		// inherit startup grace; otherwise guarded control never runs in tests or replay.
		return false
	}
	return delta < StartupGracePeriodSeconds*time.Second
}

func candidateActions(cfg config.Config, mesh statuspkg.MeshDrilldown, now time.Time) []ControlAction {
	actions := []ControlAction{}
	mode := cfg.Control.Mode
	createdAt := now.UTC().Format(time.RFC3339)
	for _, correlated := range mesh.CorrelatedFailures {
		if correlated.Reason == transport.ReasonRetryThresholdExceeded {
			for _, target := range correlated.Transports {
				actions = append(actions, ControlAction{
					ID:              generateActionID(fmt.Sprintf("ca-restart-%s", target)),
					ActionType:      ActionRestartTransport,
					TargetTransport: target,
					Reason:          "retry threshold exceeded with no proven recovery",
					Confidence:      0.96,
					TriggerEvidence: []string{correlated.Explanation, mesh.RootCauseAnalysis.Explanation},
					CreatedAt:       createdAt,
					Reversible:      false,
					Mode:            mode,
					PolicyRule:      "retry_threshold_exceeded",
				})
			}
			continue
		}
		if correlated.Reason == transport.ReasonSubscribeFailure {
			for _, target := range correlated.Transports {
				actions = append(actions, ControlAction{
					ID:              generateActionID(fmt.Sprintf("ca-resub-%s", target)),
					ActionType:      ActionResubscribeTransport,
					TargetTransport: target,
					Reason:          "subscription failure cluster persists across the active window",
					Confidence:      0.9,
					TriggerEvidence: []string{correlated.Explanation},
					CreatedAt:       createdAt,
					Mode:            mode,
					PolicyRule:      "subscription_failure_cluster",
				})
			}
		}
		if correlated.Reason == transport.ReasonMalformedFrame || correlated.Reason == transport.ReasonMalformedPublish || correlated.Reason == transport.ReasonObservationDropped {
			for _, target := range correlated.Transports {
				actions = append(actions, ControlAction{
					ID:              generateActionID(fmt.Sprintf("ca-backoff-%s-%s", target, correlated.Reason)),
					ActionType:      ActionBackoffIncrease,
					TargetTransport: target,
					Reason:          fmt.Sprintf("%s cluster suggests saturation or malformed flood pressure", correlated.Reason),
					Confidence:      0.86,
					TriggerEvidence: []string{correlated.Explanation},
					CreatedAt:       createdAt,
					Reversible:      true,
					ExpiresAt:       now.Add(10 * time.Minute).UTC().Format(time.RFC3339),
					Mode:            mode,
					PolicyRule:      "failure_storm_backoff_increase",
				})
			}
		}
	}
	for _, route := range mesh.RoutingRecommendations {
		if route.Action != "deprioritize_degraded_transport" {
			continue
		}
		actions = append(actions, ControlAction{
			ID:              generateActionID(fmt.Sprintf("ca-deprioritize-%s", route.TargetTransport)),
			ActionType:      ActionTemporarilyDeprioritize,
			TargetTransport: route.TargetTransport,
			Reason:          route.Reason,
			Confidence:      confidenceScore(route.Confidence),
			TriggerEvidence: []string{route.Reason},
			CreatedAt:       createdAt,
			Reversible:      true,
			ExpiresAt:       now.Add(15 * time.Minute).UTC().Format(time.RFC3339),
			Mode:            mode,
			PolicyRule:      "mesh_relative_degradation",
		})
	}
	for _, route := range mesh.RoutingRecommendations {
		if route.Action != "temporarily_suppress_noisy_source" {
			continue
		}
		actions = append(actions, ControlAction{
			ID:              generateActionID(fmt.Sprintf("ca-suppress-%s", route.TargetTransport)),
			ActionType:      ActionTemporarilySuppressNoisySource,
			TargetTransport: route.TargetTransport,
			Reason:          route.Reason,
			Confidence:      confidenceScore(route.Confidence),
			TriggerEvidence: []string{route.Reason},
			CreatedAt:       createdAt,
			Reversible:      true,
			ExpiresAt:       now.Add(10 * time.Minute).UTC().Format(time.RFC3339),
			Mode:            mode,
			PolicyRule:      "high_confidence_noisy_source_attribution",
		})
	}
	for _, alert := range mesh.ActiveAlerts {
		switch alert.Reason {
		case transport.ReasonRetryThresholdExceeded:
			actions = append(actions, ControlAction{
				ID:              generateActionID(fmt.Sprintf("ca-restart-alert-%s", alert.TransportName)),
				ActionType:      ActionRestartTransport,
				TargetTransport: alert.TransportName,
				Reason:          "retry threshold exceeded on an active transport alert",
				Confidence:      0.95,
				TriggerEvidence: []string{alert.Summary, alert.TriggerCondition},
				EpisodeID:       alert.EpisodeID,
				CreatedAt:       createdAt,
				Mode:            mode,
				PolicyRule:      "retry_threshold_alert",
			})
		case transport.ReasonSubscribeFailure:
			actions = append(actions, ControlAction{
				ID:              generateActionID(fmt.Sprintf("ca-resub-alert-%s", alert.TransportName)),
				ActionType:      ActionResubscribeTransport,
				TargetTransport: alert.TransportName,
				Reason:          "subscription failure remains active in transport alerts",
				Confidence:      0.88,
				TriggerEvidence: []string{alert.Summary, alert.TriggerCondition},
				EpisodeID:       alert.EpisodeID,
				CreatedAt:       createdAt,
				Mode:            mode,
				PolicyRule:      "subscription_alert",
			})
		case transport.ReasonEvidenceLoss:
			actions = append(actions, ControlAction{
				ID:              generateActionID(fmt.Sprintf("ca-backoff-alert-%s", alert.TransportName)),
				ActionType:      ActionBackoffIncrease,
				TargetTransport: alert.TransportName,
				Reason:          "active evidence-loss alert indicates ingest saturation pressure",
				Confidence:      0.84,
				TriggerEvidence: []string{alert.Summary, alert.TriggerCondition},
				CreatedAt:       createdAt,
				Reversible:      true,
				ExpiresAt:       now.Add(10 * time.Minute).UTC().Format(time.RFC3339),
				Mode:            mode,
				PolicyRule:      "evidence_loss_alert",
			})
		}
	}
	for _, segment := range mesh.DegradedSegments {
		if segment.Severity != "critical" {
			continue
		}
		actions = append(actions, ControlAction{
			ID:              generateActionID(fmt.Sprintf("ca-recheck-%s", sanitizeID(segment.SegmentID))),
			ActionType:      ActionTriggerHealthRecheck,
			TargetSegment:   segment.SegmentID,
			Reason:          "critical degraded segment requires explicit post-action health recheck",
			Confidence:      0.8,
			TriggerEvidence: []string{segment.Explanation},
			CreatedAt:       createdAt,
			Mode:            mode,
			PolicyRule:      "critical_segment_recheck",
		})
	}
	return dedupeCandidateActions(actions)
}

func evaluateCandidate(cfg config.Config, database *db.DB, policy ControlPolicy, candidate ControlAction, mesh statuspkg.MeshDrilldown, signals runtimeSignals, active []db.ControlActionRecord, historyCache map[string][]db.ControlActionRecord, realityByType map[string]ActionReality, now time.Time) ControlDecision {
	reality, ok := realityByType[candidate.ActionType]
	if !ok {
		reality = ActionReality{ActionType: candidate.ActionType, BlastRadiusClass: "unknown", DenialCode: DenialMissingActuator}
	}
	attribution := suppressionAttribution(database, candidate.TargetTransport, now)
	evidencePass := persistentEvidence(database, candidate, now)
	if candidate.ActionType == ActionRestartTransport && signals.byTransport[candidate.TargetTransport].State != transport.StateFailed && signals.byTransport[candidate.TargetTransport].FailureCount == 0 {
		evidencePass = false
	}
	if candidate.ActionType == ActionTemporarilySuppressNoisySource && !attribution.Strong {
		evidencePass = false
	}
	if candidate.ActionType == ActionTemporarilyDeprioritize && signals.connectedCount == 0 {
		evidencePass = false
	}
	confidencePass := candidate.Confidence >= policy.RequireMinConfidence
	policyPass := policyAllows(policy, candidate.ActionType)
	overridePass := !cfg.Control.EmergencyDisable && !overrideActive(cfg, candidate)
	conflictPass := !conflictingActiveAction(active, candidate, now)
	cooldownPass := cooldownSatisfied(database, policy, candidate, historyCache, now)
	budgetPass := budgetSatisfied(database, policy, candidate, now)
	alternatePathExists := candidate.ActionType != ActionTemporarilyDeprioritize || healthyAlternateExists(mesh, candidate.TargetTransport)
	reversibilityPass := reality.Reversible
	blastKnown := reality.BlastRadiusKnown && strings.TrimSpace(reality.BlastRadiusClass) != "" && reality.BlastRadiusClass != "unknown"

	if candidate.ActionType == ActionRestartTransport && !policy.AllowTransportRestart {
		policyPass = false
	}
	if candidate.ActionType == ActionTemporarilySuppressNoisySource && !policy.AllowSourceSuppression {
		policyPass = false
	}
	if isMeshLevelAction(candidate) && !policy.AllowMeshLevelActions {
		policyPass = false
	}

	safetyChecks := map[string]any{
		"evidence_pass":           evidencePass,
		"confidence_pass":         confidencePass,
		"policy_pass":             policyPass,
		"cooldown_pass":           cooldownPass,
		"override_pass":           overridePass,
		"conflict_pass":           conflictPass,
		"reversibility_pass":      reversibilityPass,
		"alternate_path_exists":   alternatePathExists,
		"blast_radius_class":      reality.BlastRadiusClass,
		"budget_pass":             budgetPass,
		"actuator_exists":         reality.ActuatorExists,
		"blast_radius_known":      blastKnown,
		"safe_for_guarded_auto":   reality.SafeForGuardedAuto,
		"advisory_only":           reality.AdvisoryOnly,
		"attribution_strong":      attribution.Strong,
		"attribution_best_effort": attribution.BestEffort,
	}

	passed := passedSafetyChecks(safetyChecks)
	denialCode, denial := determineDenial(cfg, candidate, policy, reality, attribution, safetyChecks)
	allowed := denialCode == "" && cfg.Control.Mode == ModeGuardedAuto
	if cfg.Control.Mode == ModeDisabled {
		allowed = false
		if denialCode == "" {
			denialCode, denial = DenialMode, "control mode is disabled"
		}
	}
	if cfg.Control.Mode == ModeAdvisory {
		allowed = false
		if denialCode == "" {
			denialCode, denial = DenialMode, "control mode is advisory"
		}
	}
	candidate.AdvisoryOnly = !allowed
	candidate.DenialCode = denialCode
	candidate.LifecycleState = LifecyclePending
	return ControlDecision{
		ID:                 fmt.Sprintf("cd-%s-%s", sanitizeID(now.UTC().Format(time.RFC3339Nano)), sanitizeID(candidate.ID)),
		CandidateAction:    candidate,
		Allowed:            allowed,
		DenialReason:       denial,
		DenialCode:         denialCode,
		Confidence:         candidate.Confidence,
		SafetyChecksPassed: passed,
		SafetyChecks:       safetyChecks,
		CreatedAt:          now.UTC().Format(time.RFC3339),
		Mode:               cfg.Control.Mode,
		OperatorOverride:   !overridePass,
		InputSummary: map[string]any{
			"mesh_state":              mesh.MeshHealth.State,
			"mesh_score":              mesh.MeshHealth.Score,
			"dominant_failure":        mesh.MeshHealth.DominantFailureReason,
			"correlated_failures":     len(mesh.CorrelatedFailures),
			"degraded_segments":       len(mesh.DegradedSegments),
			"routing_recommendations": len(mesh.RoutingRecommendations),
			"attribution_confidence":  attribution.Confidence,
			"attribution_best_effort": attribution.BestEffort,
			"attributed_node_id":      attribution.NodeID,
			"blast_radius_class":      reality.BlastRadiusClass,
			"safe_for_guarded_auto":   reality.SafeForGuardedAuto,
			"actuator_exists":         reality.ActuatorExists,
			"alternate_path_exists":   alternatePathExists,
		},
		PolicySummary: map[string]any{
			"mode":                    policy.Mode,
			"max_actions_per_window":  policy.MaxActionsPerWindow,
			"cooldown_per_target_sec": policy.CooldownPerTarget,
			"min_confidence":          policy.RequireMinConfidence,
			"restart_cap_per_window":  policy.RestartCapPerWindow,
		},
	}
}

func persistentEvidence(database *db.DB, candidate ControlAction, now time.Time) bool {
	if database == nil || candidate.TargetTransport == "" {
		return candidate.ActionType == ActionTriggerHealthRecheck
	}
	start := now.Add(-15 * time.Minute).UTC().Format(time.RFC3339)
	rows, err := database.TransportAnomalyHistory(candidate.TargetTransport, start, now.UTC().Format(time.RFC3339), 50, 0)
	if err != nil || len(rows) == 0 {
		return false
	}
	buckets := map[string]uint64{}
	for _, row := range rows {
		switch candidate.ActionType {
		case ActionRestartTransport:
			if row.Count == 0 {
				continue
			}
			if row.Reason == transport.ReasonRetryThresholdExceeded {
				buckets[row.BucketStart] += row.Count
			}
		case ActionResubscribeTransport:
			if row.Count == 0 {
				continue
			}
			if row.Reason == transport.ReasonSubscribeFailure {
				buckets[row.BucketStart] += row.Count
			}
		case ActionBackoffIncrease:
			// Saturation / loss evidence may appear as observation_drops with count=0, or as evidence_loss rows.
			if row.Count == 0 && row.ObservationDrops == 0 {
				continue
			}
			if row.ObservationDrops > 0 ||
				row.Reason == transport.ReasonMalformedFrame ||
				row.Reason == transport.ReasonMalformedPublish ||
				row.Reason == transport.ReasonEvidenceLoss {
				buckets[row.BucketStart] += maxUint64(row.Count, row.ObservationDrops)
			}
		default:
			if row.Count == 0 {
				continue
			}
			buckets[row.BucketStart] += row.Count
		}
	}
	return len(buckets) >= 2
}

type attributionSummary struct {
	NodeID      string
	Confidence  float64
	BestEffort  bool
	Strong      bool
	MessageSpan int
}

func suppressionAttribution(database *db.DB, transportName string, now time.Time) attributionSummary {
	if database == nil || strings.TrimSpace(transportName) == "" {
		return attributionSummary{BestEffort: true}
	}
	start := now.Add(-15 * time.Minute).UTC().Format(time.RFC3339)
	rows, err := database.QueryRows(fmt.Sprintf(`SELECT COALESCE(NULLIF(n.node_id,''), CAST(m.from_node AS TEXT)) AS attributed_node_id,
COUNT(*) AS message_count
FROM messages m
LEFT JOIN nodes n ON n.node_num = m.from_node
WHERE m.transport_name='%s' AND m.from_node > 0 AND m.rx_time >= '%s' AND m.rx_time <= '%s'
GROUP BY attributed_node_id
ORDER BY message_count DESC, attributed_node_id ASC
LIMIT 3;`, sqlSafe(transportName), sqlSafe(start), sqlSafe(now.UTC().Format(time.RFC3339))))
	if err != nil || len(rows) == 0 {
		return attributionSummary{BestEffort: true}
	}
	total := 0
	for _, row := range rows {
		total += int(asFloatValue(row["message_count"]))
	}
	top := rows[0]
	topCount := int(asFloatValue(top["message_count"]))
	confidence := 0.0
	if total > 0 {
		confidence = float64(topCount) / float64(total)
	}
	strong := len(rows) == 1 && topCount >= 3 && confidence >= 0.85
	return attributionSummary{
		NodeID:      fmt.Sprint(top["attributed_node_id"]),
		Confidence:  confidence,
		BestEffort:  len(rows) > 1 || !strong,
		Strong:      strong,
		MessageSpan: total,
	}
}

func passedSafetyChecks(checks map[string]any) []string {
	out := make([]string, 0, len(checks))
	for name, value := range checks {
		if ok, isBool := value.(bool); isBool && ok {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func determineDenial(cfg config.Config, candidate ControlAction, policy ControlPolicy, reality ActionReality, attribution attributionSummary, checks map[string]any) (string, string) {
	if cfg.Control.EmergencyDisable {
		return DenialOverride, "control disabled by emergency_disable"
	}
	if overrideActive(cfg, candidate) {
		return DenialOverride, "operator override suppresses automation for target"
	}
	if cfg.Control.Mode == ModeDisabled {
		return DenialMode, "control mode is disabled"
	}
	if cfg.Control.Mode == ModeAdvisory {
		return DenialMode, "control mode is advisory"
	}
	if pass, _ := checks["policy_pass"].(bool); !pass {
		switch {
		case candidate.ActionType == ActionRestartTransport && !policy.AllowTransportRestart:
			return DenialPolicy, "policy disables transport restart actions"
		case candidate.ActionType == ActionTemporarilySuppressNoisySource && !policy.AllowSourceSuppression:
			return DenialPolicy, "policy disables source suppression"
		case isMeshLevelAction(candidate) && !policy.AllowMeshLevelActions:
			return DenialPolicy, "policy disables mesh-level actions"
		default:
			return DenialPolicy, "policy does not allow action type"
		}
	}
	if pass, _ := checks["confidence_pass"].(bool); !pass {
		return DenialLowConfidence, "candidate confidence is below the guarded_auto threshold"
	}
	if pass, _ := checks["alternate_path_exists"].(bool); !pass {
		return DenialNoAlternatePath, "no healthy alternate transport exists for deprioritization"
	}
	if candidate.ActionType == ActionTemporarilySuppressNoisySource && !attribution.Strong {
		return DenialAttributionWeak, "source suppression remains advisory because attribution is still best-effort"
	}
	if pass, _ := checks["evidence_pass"].(bool); !pass {
		return DenialTransient, "evidence remains transient or ambiguous in persisted history"
	}
	if pass, _ := checks["cooldown_pass"].(bool); !pass {
		return DenialCooldown, "cooldown window still active for target"
	}
	if pass, _ := checks["budget_pass"].(bool); !pass {
		return DenialBudget, "action budget exceeded for current control window"
	}
	if pass, _ := checks["conflict_pass"].(bool); !pass {
		return DenialConflict, "conflicting active or in-flight action already exists for target"
	}
	if pass, _ := checks["blast_radius_known"].(bool); !pass {
		return DenialUnknownBlastRadius, "blast radius is not bounded strongly enough for guarded automation"
	}
	if pass, _ := checks["reversibility_pass"].(bool); !pass {
		return DenialIrreversible, "action is not reversible or expiry-backed"
	}
	if pass, _ := checks["actuator_exists"].(bool); !pass || reality.AdvisoryOnly || !reality.SafeForGuardedAuto {
		code := reality.DenialCode
		if code == "" {
			code = DenialMissingActuator
		}
		return code, firstNonEmpty(reality.Notes, "action remains advisory because MEL does not ship a verified actuator for it")
	}
	return "", ""
}

func healthyAlternateExists(mesh statuspkg.MeshDrilldown, degraded string) bool {
	for _, route := range mesh.RoutingRecommendations {
		if route.Action == "suggest_alternate_ingest_path" && route.TargetTransport == degraded {
			return true
		}
	}
	return false
}

func conflictingActiveAction(active []db.ControlActionRecord, candidate ControlAction, now time.Time) bool {
	for _, action := range active {
		if action.TargetTransport != "" && action.TargetTransport != candidate.TargetTransport {
			continue
		}
		if action.ActionType == candidate.ActionType && (isEffectivelyActive(action, now) || isInFlight(action)) {
			return true
		}
		if action.ActionType == ActionRestartTransport && candidate.ActionType == ActionResubscribeTransport && (isEffectivelyActive(action, now) || isInFlight(action)) {
			return true
		}
	}
	return false
}

func cooldownSatisfied(database *db.DB, policy ControlPolicy, candidate ControlAction, historyCache map[string][]db.ControlActionRecord, now time.Time) bool {
	if database == nil || candidate.TargetTransport == "" || policy.CooldownPerTarget <= 0 {
		return true
	}
	key := candidate.TargetTransport
	rows, ok := historyCache[key]
	if !ok {
		start := now.Add(-time.Duration(policy.CooldownPerTarget) * time.Second).UTC().Format(time.RFC3339)
		rows, _ = database.ControlActions(candidate.TargetTransport, "", start, "", 50, 0)
		historyCache[key] = rows
	}
	for _, row := range rows {
		if row.Result != ResultExecutedSuccessfully && row.Result != ResultExecutedNoop {
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

func budgetSatisfied(database *db.DB, policy ControlPolicy, candidate ControlAction, now time.Time) bool {
	if database == nil || policy.MaxActionsPerWindow <= 0 {
		return true
	}
	start := now.Add(-time.Duration(policy.ActionWindowSeconds) * time.Second).UTC().Format(time.RFC3339)
	rows, err := database.ControlActions("", "", start, "", policy.MaxActionsPerWindow+10, 0)
	if err != nil {
		return false
	}
	executedCount := 0
	for _, row := range rows {
		if row.Result == ResultExecutedSuccessfully || row.Result == ResultExecutedNoop {
			executedCount++
		}
	}
	if executedCount >= policy.MaxActionsPerWindow {
		return false
	}
	if candidate.ActionType != ActionRestartTransport || policy.RestartCapPerWindow <= 0 {
		return true
	}
	restartCount := 0
	for _, row := range rows {
		if row.ActionType == ActionRestartTransport && (row.Result == ResultExecutedSuccessfully || row.Result == ResultExecutedNoop) {
			restartCount++
		}
	}
	return restartCount < policy.RestartCapPerWindow
}

func overrideActive(cfg config.Config, candidate ControlAction) bool {
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
		if candidate.ActionType == ActionTemporarilyDeprioritize && tc.FreezeRouting {
			return true
		}
	}
	return false
}

func policyAllows(policy ControlPolicy, actionType string) bool {
	for _, allowed := range policy.AllowedActions {
		if allowed == actionType {
			return true
		}
	}
	return false
}

func isMeshLevelAction(action ControlAction) bool {
	return action.TargetSegment != "" || action.ActionType == ActionTemporarilyDeprioritize
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

func filterActiveActions(in []db.ControlActionRecord, now time.Time) []db.ControlActionRecord {
	out := make([]db.ControlActionRecord, 0, len(in))
	for _, row := range in {
		if isEffectivelyActive(row, now) {
			out = append(out, row)
		}
	}
	return out
}

func isEffectivelyActive(row db.ControlActionRecord, now time.Time) bool {
	if row.Result != ResultExecutedSuccessfully {
		return false
	}
	if row.ExpiresAt == "" {
		return row.Reversible
	}
	expires, ok := parseRFC3339(row.ExpiresAt)
	return ok && now.Before(expires)
}

func isInFlight(row db.ControlActionRecord) bool {
	return row.LifecycleState == LifecyclePending || row.LifecycleState == LifecycleRunning
}

func controlActionsFromRecords(records []db.ControlActionRecord) []ControlAction {
	out := make([]ControlAction, 0, len(records))
	for _, row := range records {
		out = append(out, ControlAction{
			ID:              row.ID,
			DecisionID:      row.DecisionID,
			ActionType:      row.ActionType,
			TargetTransport: row.TargetTransport,
			TargetSegment:   row.TargetSegment,
			TargetNode:      row.TargetNode,
			Reason:          row.Reason,
			Confidence:      row.Confidence,
			TriggerEvidence: append([]string(nil), row.TriggerEvidence...),
			EpisodeID:       row.EpisodeID,
			CreatedAt:       row.CreatedAt,
			ExecutedAt:      row.ExecutedAt,
			CompletedAt:     row.CompletedAt,
			Result:          row.Result,
			Reversible:      row.Reversible,
			ExpiresAt:       row.ExpiresAt,
			OutcomeDetail:   row.OutcomeDetail,
			Mode:            row.Mode,
			PolicyRule:      row.PolicyRule,
			LifecycleState:  row.LifecycleState,
			AdvisoryOnly:    row.AdvisoryOnly,
			DenialCode:      row.DenialCode,
			ClosureState:    row.ClosureState,
			Metadata:        row.Metadata,
		})
	}
	return out
}

func dedupeCandidateActions(in []ControlAction) []ControlAction {
	seen := map[string]ControlAction{}
	for _, action := range in {
		key := strings.Join([]string{action.ActionType, action.TargetTransport, action.TargetSegment, action.PolicyRule}, "|")
		if existing, ok := seen[key]; ok && existing.Confidence >= action.Confidence {
			continue
		}
		seen[key] = action
	}
	out := make([]ControlAction, 0, len(seen))
	for _, action := range seen {
		out = append(out, action)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Confidence == out[j].Confidence {
			if out[i].TargetTransport == out[j].TargetTransport {
				return out[i].ActionType < out[j].ActionType
			}
			return out[i].TargetTransport < out[j].TargetTransport
		}
		return out[i].Confidence > out[j].Confidence
	})
	return out
}

func confidenceScore(label string) float64 {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "high":
		return 0.92
	case "medium":
		return 0.78
	default:
		return 0.6
	}
}

func sanitizeID(v string) string {
	v = strings.ToLower(v)
	replacer := strings.NewReplacer(":", "-", ".", "-", "+", "-", "/", "-", "|", "-", " ", "-", "_", "-")
	return replacer.Replace(v)
}

func parseRFC3339(v string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if ts, err := time.Parse(layout, strings.TrimSpace(v)); err == nil {
			return ts.UTC(), true
		}
	}
	return time.Time{}, false
}

func dedupeStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func minPositive(a, b int) int {
	if a <= 0 {
		return b
	}
	if a < b {
		return a
	}
	return b
}

func maxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func sqlSafe(v string) string {
	return strings.ReplaceAll(v, "'", "''")
}

func asFloatValue(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		ts := strings.TrimSpace(t)
		if ts == "" {
			return 0
		}
		var parsed float64
		fmt.Sscanf(ts, "%f", &parsed)
		return parsed
	default:
		return 0
	}
}

func MarshalJSONMap(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// ResetStartupTimeForTests sets the startup time to a specific value for testing.
// This should only be used in tests to bypass the startup grace period.
func ResetStartupTimeForTests(t time.Time) {
	startupTime = t
}
