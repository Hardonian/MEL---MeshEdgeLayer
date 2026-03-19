package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
)

type transportControlState struct {
	mu                 sync.Mutex
	backoffMultiplier  int
	backoffUntil       time.Time
	deprioritizedUntil time.Time
	suppressedUntil    time.Time
	interruptCh        chan struct{}
}

func newTransportControlState() *transportControlState {
	return &transportControlState{interruptCh: make(chan struct{}, 1)}
}

func (s *transportControlState) currentBackoffMultiplier(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.backoffUntil.IsZero() || now.After(s.backoffUntil) {
		s.backoffMultiplier = 1
		return 1
	}
	if s.backoffMultiplier < 1 {
		s.backoffMultiplier = 1
	}
	return s.backoffMultiplier
}

func (s *transportControlState) setBackoff(multiplier int, until time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if multiplier < 1 {
		multiplier = 1
	}
	s.backoffMultiplier = multiplier
	s.backoffUntil = until
}

func (s *transportControlState) clearBackoff() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.backoffMultiplier = 1
	s.backoffUntil = time.Time{}
}

func (s *transportControlState) markDeprioritized(until time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deprioritizedUntil = until
}

func (s *transportControlState) markSuppressed(until time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.suppressedUntil = until
}

func (s *transportControlState) clearSuppression() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.suppressedUntil = time.Time{}
}

func (s *transportControlState) interrupt() {
	select {
	case s.interruptCh <- struct{}{}:
	default:
	}
}

func (a *App) controlExplanation() (map[string]any, error) {
	eval, err := control.Evaluate(a.Cfg, a.DB, a.TransportHealth(), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"mode":               eval.Explanation.Mode,
		"active_actions":     eval.Explanation.ActiveActions,
		"recent_actions":     eval.Explanation.RecentActions,
		"denied_actions":     eval.Explanation.DeniedActions,
		"policy_summary":     eval.Explanation.PolicySummary,
		"reasons_for_denial": eval.Explanation.ReasonsForDenial,
		"emergency_disable":  eval.Explanation.EmergencyDisable,
	}, nil
}

func (a *App) controlHistory(start, end, transportName string, limit, offset int) (map[string]any, error) {
	if a == nil || a.DB == nil {
		return map[string]any{"actions": []db.ControlActionRecord{}, "decisions": []db.ControlDecisionRecord{}}, nil
	}
	actions, err := a.DB.ControlActions(transportName, "", start, end, limit, offset)
	if err != nil {
		return nil, err
	}
	decisions, err := a.DB.ControlDecisions(transportName, "", start, end, limit, offset)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"actions":    actions,
		"decisions":  decisions,
		"transport":  transportName,
		"start":      start,
		"end":        end,
		"pagination": map[string]any{"limit": limit, "offset": offset},
	}, nil
}

func (a *App) controlExecutor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case action := <-a.controlQueue:
			a.executeControlAction(ctx, action)
		}
	}
}

func (a *App) evaluateControl(now time.Time) {
	if a == nil || a.DB == nil {
		return
	}
	eval, err := control.Evaluate(a.Cfg, a.DB, a.TransportHealth(), now)
	if err != nil {
		a.Log.Error("control_evaluation_failed", "failed to evaluate guarded control decisions", map[string]any{"error": err.Error()})
		return
	}
	for _, decision := range eval.Decisions {
		record := controlDecisionRecord(decision)
		if err := a.DB.UpsertControlDecision(record); err != nil {
			a.Log.Error("control_decision_upsert_failed", "failed to persist control decision", map[string]any{"decision_id": decision.ID, "error": err.Error()})
		}
		action := decision.CandidateAction
		action.DecisionID = decision.ID
		if !decision.Allowed {
			action.ExecutedAt = decision.CreatedAt
			action.CompletedAt = decision.CreatedAt
			action.OutcomeDetail = decision.DenialReason
			action.Result = control.ResultDeniedByPolicy
			if strings.Contains(decision.DenialReason, "cooldown") {
				action.Result = control.ResultDeniedByCooldown
			}
			if err := a.DB.UpsertControlAction(controlActionRecord(action)); err != nil {
				a.Log.Error("control_action_upsert_failed", "failed to persist denied control action", map[string]any{"action_id": action.ID, "error": err.Error()})
			}
			continue
		}
		if err := a.DB.UpsertControlAction(controlActionRecord(action)); err != nil {
			a.Log.Error("control_action_upsert_failed", "failed to persist control action", map[string]any{"action_id": action.ID, "error": err.Error()})
			continue
		}
		select {
		case a.controlQueue <- action:
		default:
			action.ExecutedAt = now.UTC().Format(time.RFC3339)
			action.CompletedAt = action.ExecutedAt
			action.Result = control.ResultFailedTransient
			action.OutcomeDetail = "control queue is full; action dropped to preserve bounded execution"
			_ = a.DB.UpsertControlAction(controlActionRecord(action))
			a.Log.Error("control_queue_full", "control action queue is full", map[string]any{"action_id": action.ID, "action_type": action.ActionType})
		}
	}
	a.queueRecoveryActions(now)
}

func (a *App) queueRecoveryActions(now time.Time) {
	if a == nil || a.DB == nil {
		return
	}
	rows, err := a.DB.ControlActions("", "", now.AddDate(0, 0, -1).UTC().Format(time.RFC3339), "", a.Cfg.Intelligence.Queries.MaxLimit, 0)
	if err != nil {
		return
	}
	for _, row := range rows {
		if row.Result != control.ResultExecutedSuccessfully || row.ExpiresAt == "" {
			continue
		}
		expiresAt, ok := parseRFC3339(row.ExpiresAt)
		if !ok || now.Before(expiresAt) {
			continue
		}
		if !quietSinceExpiry(a.DB, row.TargetTransport, now) {
			continue
		}
		var followup control.ControlAction
		switch row.ActionType {
		case control.ActionBackoffIncrease:
			followup = control.ControlAction{ID: fmt.Sprintf("%s-reset", row.ID), DecisionID: row.DecisionID, ActionType: control.ActionBackoffReset, TargetTransport: row.TargetTransport, Reason: "backoff increase window expired after quiet recovery evidence", Confidence: 0.9, TriggerEvidence: []string{"anomaly snapshots quiet after expiry"}, CreatedAt: now.UTC().Format(time.RFC3339), Mode: a.Cfg.Control.Mode, PolicyRule: "evidence_based_backoff_reset"}
		case control.ActionTemporarilySuppressNoisySource:
			followup = control.ControlAction{ID: fmt.Sprintf("%s-clear", row.ID), DecisionID: row.DecisionID, ActionType: control.ActionClearSuppression, TargetTransport: row.TargetTransport, Reason: "suppression window expired after quiet recovery evidence", Confidence: 0.85, TriggerEvidence: []string{"anomaly snapshots quiet after expiry"}, CreatedAt: now.UTC().Format(time.RFC3339), Mode: a.Cfg.Control.Mode, PolicyRule: "evidence_based_clear_suppression"}
		default:
			continue
		}
		if _, ok := a.seenControlAction(followup.ID); ok {
			continue
		}
		_ = a.DB.UpsertControlAction(controlActionRecord(followup))
		select {
		case a.controlQueue <- followup:
		default:
		}
	}
}

func quietSinceExpiry(database *db.DB, transportName string, now time.Time) bool {
	if database == nil || strings.TrimSpace(transportName) == "" {
		return false
	}
	start := now.Add(-5 * time.Minute).UTC().Format(time.RFC3339)
	rows, err := database.TransportAnomalyHistory(transportName, start, now.UTC().Format(time.RFC3339), 20, 0)
	if err != nil {
		return false
	}
	for _, row := range rows {
		if row.Count > 0 || row.DeadLetters > 0 || row.ObservationDrops > 0 {
			return false
		}
	}
	return true
}

func (a *App) seenControlAction(id string) (db.ControlActionRecord, bool) {
	rows, err := a.DB.ControlActions("", "", "", "", 500, 0)
	if err != nil {
		return db.ControlActionRecord{}, false
	}
	for _, row := range rows {
		if row.ID == id {
			return row, true
		}
	}
	return db.ControlActionRecord{}, false
}

func (a *App) executeControlAction(ctx context.Context, action control.ControlAction) {
	if a == nil || a.DB == nil {
		return
	}
	startedAt := time.Now().UTC().Format(time.RFC3339)
	action.ExecutedAt = startedAt
	_ = a.DB.UpsertControlAction(controlActionRecord(action))

	timeout := time.Duration(a.Cfg.Control.ActionTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, detail := control.ResultExecutedNoop, "action did not change runtime state"
	switch action.ActionType {
	case control.ActionRestartTransport, control.ActionResubscribeTransport:
		if tr, st := a.findTransportAndControl(action.TargetTransport); tr != nil {
			_ = tr.Close(runCtx)
			st.interrupt()
			result, detail = control.ResultExecutedSuccessfully, "transport interrupted so the bounded reconnect loop can re-enter connect/subscribe"
		} else {
			result, detail = control.ResultFailedTerminal, "target transport not found"
		}
	case control.ActionBackoffIncrease:
		if _, st := a.findTransportAndControl(action.TargetTransport); st != nil {
			until, _ := parseRFC3339(action.ExpiresAt)
			if until.IsZero() {
				until = time.Now().UTC().Add(10 * time.Minute)
				action.ExpiresAt = until.Format(time.RFC3339)
			}
			st.setBackoff(2, until)
			result, detail = control.ResultExecutedSuccessfully, "transport reconnect backoff multiplier raised to 2x until evidence-based reset"
		} else {
			result, detail = control.ResultFailedTerminal, "target transport not found"
		}
	case control.ActionBackoffReset:
		if _, st := a.findTransportAndControl(action.TargetTransport); st != nil {
			st.clearBackoff()
			result, detail = control.ResultExecutedSuccessfully, "transport reconnect backoff multiplier restored to baseline"
		} else {
			result, detail = control.ResultFailedTerminal, "target transport not found"
		}
	case control.ActionTemporarilyDeprioritize:
		if _, st := a.findTransportAndControl(action.TargetTransport); st != nil {
			if until, ok := parseRFC3339(action.ExpiresAt); ok {
				st.markDeprioritized(until)
			}
			result, detail = control.ResultExecutedNoop, "routing selector is not implemented; deprioritization is persisted as advisory state only"
		}
	case control.ActionTemporarilySuppressNoisySource:
		if _, st := a.findTransportAndControl(action.TargetTransport); st != nil {
			if until, ok := parseRFC3339(action.ExpiresAt); ok {
				st.markSuppressed(until)
			}
			result, detail = control.ResultExecutedNoop, "source suppression actuator is not implemented; suppression is recorded as advisory state only"
		}
	case control.ActionClearSuppression:
		if _, st := a.findTransportAndControl(action.TargetTransport); st != nil {
			st.clearSuppression()
			result, detail = control.ResultExecutedSuccessfully, "suppression advisory state cleared after quiet recovery evidence"
		}
	case control.ActionTriggerHealthRecheck:
		go a.evaluateTransportIntelligence(time.Now().UTC())
		result, detail = control.ResultExecutedSuccessfully, "scheduled asynchronous health recheck outside the ingest hot path"
	default:
		result, detail = control.ResultFailedTerminal, "unsupported control action type"
	}
	completedAt := time.Now().UTC().Format(time.RFC3339)
	action.CompletedAt = completedAt
	action.Result = result
	action.OutcomeDetail = detail
	_ = a.DB.UpsertControlAction(controlActionRecord(action))
}

func (a *App) findTransportAndControl(name string) (anyTransport, *transportControlState) {
	if strings.TrimSpace(name) == "" {
		return nil, nil
	}
	for _, tr := range a.Transports {
		if tr.Name() == name {
			return tr, a.transportControls[name]
		}
	}
	return nil, a.transportControls[name]
}

type anyTransport interface {
	Close(context.Context) error
	Name() string
}

func (a *App) effectiveBackoff(base time.Duration, transportName string, now time.Time) time.Duration {
	state := a.transportControls[transportName]
	if state == nil {
		return base
	}
	multiplier := state.currentBackoffMultiplier(now)
	if multiplier <= 1 {
		return base
	}
	return time.Duration(multiplier) * base
}

func (a *App) interruptCh(transportName string) <-chan struct{} {
	if state := a.transportControls[transportName]; state != nil {
		return state.interruptCh
	}
	return nil
}

func controlActionRecord(action control.ControlAction) db.ControlActionRecord {
	return db.ControlActionRecord{
		ID:              action.ID,
		DecisionID:      action.DecisionID,
		ActionType:      action.ActionType,
		TargetTransport: action.TargetTransport,
		TargetSegment:   action.TargetSegment,
		TargetNode:      action.TargetNode,
		Reason:          action.Reason,
		Confidence:      action.Confidence,
		TriggerEvidence: append([]string(nil), action.TriggerEvidence...),
		EpisodeID:       action.EpisodeID,
		CreatedAt:       action.CreatedAt,
		ExecutedAt:      action.ExecutedAt,
		CompletedAt:     action.CompletedAt,
		Result:          action.Result,
		Reversible:      action.Reversible,
		ExpiresAt:       action.ExpiresAt,
		OutcomeDetail:   action.OutcomeDetail,
		Mode:            action.Mode,
		PolicyRule:      action.PolicyRule,
		Metadata:        action.Metadata,
	}
}

func controlDecisionRecord(decision control.ControlDecision) db.ControlDecisionRecord {
	return db.ControlDecisionRecord{
		ID:                decision.ID,
		CandidateActionID: decision.CandidateAction.ID,
		ActionType:        decision.CandidateAction.ActionType,
		TargetTransport:   decision.CandidateAction.TargetTransport,
		TargetSegment:     decision.CandidateAction.TargetSegment,
		Reason:            decision.CandidateAction.Reason,
		Confidence:        decision.Confidence,
		Allowed:           decision.Allowed,
		DenialReason:      decision.DenialReason,
		SafetyChecks:      decision.SafetyChecks,
		DecisionInputs:    decision.InputSummary,
		PolicySummary:     decision.PolicySummary,
		CreatedAt:         decision.CreatedAt,
		Mode:              decision.Mode,
		OperatorOverride:  decision.OperatorOverride,
	}
}
