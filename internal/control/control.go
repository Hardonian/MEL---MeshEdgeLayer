package control

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	statuspkg "github.com/mel-project/mel/internal/status"
	"github.com/mel-project/mel/internal/transport"
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
	Metadata        map[string]any `json:"metadata,omitempty"`
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
	ID                 string          `json:"id"`
	CandidateAction    ControlAction   `json:"candidate_action"`
	Allowed            bool            `json:"allowed"`
	DenialReason       string          `json:"denial_reason,omitempty"`
	Confidence         float64         `json:"confidence"`
	SafetyChecksPassed []string        `json:"safety_checks_passed,omitempty"`
	SafetyChecks       map[string]bool `json:"safety_checks"`
	CreatedAt          string          `json:"created_at"`
	Mode               string          `json:"mode"`
	OperatorOverride   bool            `json:"operator_override"`
	InputSummary       map[string]any  `json:"input_summary,omitempty"`
	PolicySummary      map[string]any  `json:"policy_summary,omitempty"`
}

type ControlExplanation struct {
	Mode             string            `json:"mode"`
	ActiveActions    []ControlAction   `json:"active_actions,omitempty"`
	RecentActions    []ControlAction   `json:"recent_actions,omitempty"`
	DeniedActions    []ControlDecision `json:"denied_actions,omitempty"`
	PolicySummary    ControlPolicy     `json:"policy_summary"`
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

	for _, candidate := range candidateActions(cfg, mesh, now) {
		decision := evaluateCandidate(cfg, database, policy, candidate, mesh, signals, activeActions, historyCache, now)
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
		DeniedActions:    denied,
		PolicySummary:    policy,
		ReasonsForDenial: dedupeStrings(denialReasons),
		EmergencyDisable: cfg.Control.EmergencyDisable,
	}
	return Evaluation{Policy: policy, Decisions: decisions, Explanation: explanation}, nil
}

func PolicyFromConfig(cfg config.Config) ControlPolicy {
	return ControlPolicy{
		Mode:                   cfg.Control.Mode,
		AllowedActions:         append([]string(nil), cfg.Control.AllowedActions...),
		MaxActionsPerWindow:    cfg.Control.MaxActionsPerWindow,
		CooldownPerTarget:      cfg.Control.CooldownPerTargetSeconds,
		RequireMinConfidence:   cfg.Control.RequireMinConfidence,
		AllowMeshLevelActions:  cfg.Control.AllowMeshLevelActions,
		AllowTransportRestart:  cfg.Control.AllowTransportRestart,
		AllowSourceSuppression: cfg.Control.AllowSourceSuppression,
		ActionWindowSeconds:    cfg.Control.ActionWindowSeconds,
		RestartCapPerWindow:    cfg.Control.RestartCapPerWindow,
	}
}

func candidateActions(cfg config.Config, mesh statuspkg.MeshDrilldown, now time.Time) []ControlAction {
	actions := []ControlAction{}
	mode := cfg.Control.Mode
	createdAt := now.UTC().Format(time.RFC3339)
	for _, correlated := range mesh.CorrelatedFailures {
		if correlated.Reason == transport.ReasonRetryThresholdExceeded {
			for _, target := range correlated.Transports {
				actions = append(actions, ControlAction{
					ID:              fmt.Sprintf("ca-%s-restart-%s", sanitizeID(createdAt), target),
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
					ID:              fmt.Sprintf("ca-%s-resub-%s", sanitizeID(createdAt), target),
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
					ID:              fmt.Sprintf("ca-%s-backoff-%s-%s", sanitizeID(createdAt), target, correlated.Reason),
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
			ID:              fmt.Sprintf("ca-%s-deprioritize-%s", sanitizeID(createdAt), route.TargetTransport),
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
			ID:              fmt.Sprintf("ca-%s-suppress-%s", sanitizeID(createdAt), route.TargetTransport),
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
				ID:              fmt.Sprintf("ca-%s-restart-alert-%s", sanitizeID(createdAt), alert.TransportName),
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
				ID:              fmt.Sprintf("ca-%s-resub-alert-%s", sanitizeID(createdAt), alert.TransportName),
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
		case "evidence_loss":
			actions = append(actions, ControlAction{
				ID:              fmt.Sprintf("ca-%s-backoff-alert-%s", sanitizeID(createdAt), alert.TransportName),
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
			ID:              fmt.Sprintf("ca-%s-recheck-%s", sanitizeID(createdAt), sanitizeID(segment.SegmentID)),
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

func evaluateCandidate(cfg config.Config, database *db.DB, policy ControlPolicy, candidate ControlAction, mesh statuspkg.MeshDrilldown, signals runtimeSignals, active []db.ControlActionRecord, historyCache map[string][]db.ControlActionRecord, now time.Time) ControlDecision {
	checks := map[string]bool{
		"confidence_threshold_met":     candidate.Confidence >= policy.RequireMinConfidence,
		"policy_allows_action":         policyAllows(policy, candidate.ActionType),
		"cooldown_satisfied":           true,
		"no_conflicting_active_action": true,
		"persistent_evidence":          true,
		"action_budget_not_exceeded":   true,
		"operator_override_not_active": true,
	}
	passed := []string{}
	denial := ""
	if cfg.Control.EmergencyDisable {
		checks["operator_override_not_active"] = false
		denial = "control disabled by emergency_disable"
	}
	if overrideActive(cfg, candidate) {
		checks["operator_override_not_active"] = false
		denial = "operator override suppresses automation for target"
	}
	if !policyAllows(policy, candidate.ActionType) {
		checks["policy_allows_action"] = false
		denial = "policy does not allow action type"
	}
	if candidate.ActionType == ActionRestartTransport && !policy.AllowTransportRestart {
		checks["policy_allows_action"] = false
		denial = "policy disables transport restart actions"
	}
	if candidate.ActionType == ActionTemporarilySuppressNoisySource && !policy.AllowSourceSuppression {
		checks["policy_allows_action"] = false
		denial = "policy disables source suppression"
	}
	if isMeshLevelAction(candidate) && !policy.AllowMeshLevelActions {
		checks["policy_allows_action"] = false
		denial = "policy disables mesh-level actions"
	}
	if !persistentEvidence(database, candidate, now) {
		checks["persistent_evidence"] = false
		denial = "evidence remains transient in anomaly snapshot history"
	}
	if conflictingActiveAction(active, candidate, now) {
		checks["no_conflicting_active_action"] = false
		denial = "conflicting active action already exists for target"
	}
	if !cooldownSatisfied(database, policy, candidate, historyCache, now) {
		checks["cooldown_satisfied"] = false
		denial = "cooldown window still active for target"
	}
	if !budgetSatisfied(database, policy, candidate, now) {
		checks["action_budget_not_exceeded"] = false
		denial = "action budget exceeded for current control window"
	}
	if candidate.ActionType == ActionTemporarilyDeprioritize && !healthyAlternateExists(mesh, candidate.TargetTransport) {
		checks["persistent_evidence"] = false
		denial = "no healthy alternate transport exists for deprioritization"
	}
	if candidate.ActionType == ActionTemporarilySuppressNoisySource {
		checks["policy_allows_action"] = false
		if denial == "" {
			denial = "source suppression remains advisory until attribution is stronger than current transport-level evidence"
		}
	}
	if candidate.ActionType == ActionTemporarilyDeprioritize {
		checks["policy_allows_action"] = false
		if denial == "" {
			denial = "routing changes remain advisory because MEL does not yet own a reversible live routing selector"
		}
	}
	if candidate.ActionType == ActionRestartTransport && signals.byTransport[candidate.TargetTransport].State != transport.StateFailed && signals.byTransport[candidate.TargetTransport].FailureCount == 0 {
		checks["persistent_evidence"] = false
		denial = "restart target is not currently failed or inside an active failure episode"
	}
	for name, ok := range checks {
		if ok {
			passed = append(passed, name)
		}
	}
	sort.Strings(passed)
	allowed := denial == "" && cfg.Control.Mode == ModeGuardedAuto
	if cfg.Control.Mode == ModeDisabled {
		allowed = false
		if denial == "" {
			denial = "control mode is disabled"
		}
	}
	if cfg.Control.Mode == ModeAdvisory {
		allowed = false
		if denial == "" {
			denial = "control mode is advisory"
		}
	}
	return ControlDecision{
		ID:                 fmt.Sprintf("cd-%s-%s", sanitizeID(now.UTC().Format(time.RFC3339Nano)), sanitizeID(candidate.ID)),
		CandidateAction:    candidate,
		Allowed:            allowed,
		DenialReason:       denial,
		Confidence:         candidate.Confidence,
		SafetyChecksPassed: passed,
		SafetyChecks:       checks,
		CreatedAt:          now.UTC().Format(time.RFC3339),
		Mode:               cfg.Control.Mode,
		OperatorOverride:   !checks["operator_override_not_active"],
		InputSummary: map[string]any{
			"mesh_state":              mesh.MeshHealth.State,
			"mesh_score":              mesh.MeshHealth.Score,
			"dominant_failure":        mesh.MeshHealth.DominantFailureReason,
			"correlated_failures":     len(mesh.CorrelatedFailures),
			"degraded_segments":       len(mesh.DegradedSegments),
			"routing_recommendations": len(mesh.RoutingRecommendations),
		},
		PolicySummary: map[string]any{
			"mode":                    policy.Mode,
			"max_actions_per_window":  policy.MaxActionsPerWindow,
			"cooldown_per_target_sec": policy.CooldownPerTarget,
			"min_confidence":          policy.RequireMinConfidence,
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
		if row.Count == 0 {
			continue
		}
		switch candidate.ActionType {
		case ActionRestartTransport:
			if row.Reason == transport.ReasonRetryThresholdExceeded {
				buckets[row.BucketStart] += row.Count
			}
		case ActionResubscribeTransport:
			if row.Reason == transport.ReasonSubscribeFailure {
				buckets[row.BucketStart] += row.Count
			}
		case ActionBackoffIncrease:
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
		if action.ActionType == candidate.ActionType && isEffectivelyActive(action, now) {
			return true
		}
		if action.ActionType == ActionRestartTransport && candidate.ActionType == ActionResubscribeTransport && isEffectivelyActive(action, now) {
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
		ts, ok := parseRFC3339(firstNonEmpty(row.CompletedAt, row.ExecutedAt, row.CreatedAt))
		if !ok {
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
	rows, err := database.ControlActions("", "", start, "", policy.MaxActionsPerWindow+1, 0)
	if err != nil {
		return false
	}
	if len(rows) >= policy.MaxActionsPerWindow {
		return false
	}
	if candidate.ActionType != ActionRestartTransport || policy.RestartCapPerWindow <= 0 {
		return true
	}
	count := 0
	for _, row := range rows {
		if row.ActionType == ActionRestartTransport {
			count++
		}
	}
	return count < policy.RestartCapPerWindow
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

func MarshalJSONMap(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
