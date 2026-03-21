package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/selfobs"
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
	reality, _ := a.DB.ControlActionRealities()
	return map[string]any{
		"mode":               eval.Explanation.Mode,
		"active_actions":     eval.Explanation.ActiveActions,
		"pending_actions":    eval.Explanation.PendingActions,
		"recent_actions":     eval.Explanation.RecentActions,
		"denied_actions":     eval.Explanation.DeniedActions,
		"policy_summary":     eval.Explanation.PolicySummary,
		"reality_matrix":     reality,
		"queue_depth":        len(a.controlQueue),
		"queue_capacity":     cap(a.controlQueue),
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
	inFlight, err := a.DB.IncompleteControlActions(limit)
	if err != nil {
		return nil, err
	}
	reality, err := a.DB.ControlActionRealities()
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"actions":        actions,
		"decisions":      decisions,
		"in_flight":      inFlight,
		"reality_matrix": reality,
		"transport":      transportName,
		"start":          start,
		"end":            end,
		"pagination":     map[string]any{"limit": limit, "offset": offset},
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
	if control.WithinStartupGracePeriod(now) {
		a.Log.Info("control_startup_grace_period", "skipping control evaluation during startup grace period", map[string]any{
			"grace_period_remaining_sec": int(control.StartupGracePeriodSeconds - now.Sub(control.StartupTime()).Seconds()),
		})
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
		action.LifecycleState = control.LifecyclePending
		action.AdvisoryOnly = !decision.Allowed
		action.DenialCode = decision.DenialCode
		if !decision.Allowed {
			action.ExecutedAt = decision.CreatedAt
			action.CompletedAt = decision.CreatedAt
			action.OutcomeDetail = decision.DenialReason
			action.LifecycleState = control.LifecycleCompleted
			action.Result = control.ResultDeniedByPolicy
			if decision.DenialCode == control.DenialCooldown {
				action.Result = control.ResultDeniedByCooldown
			}
			if decision.DenialCode == control.DenialOverride {
				action.ClosureState = control.ClosureCanceledByOperator
			}
			if err := a.DB.UpsertControlAction(controlActionRecord(action)); err != nil {
				a.Log.Error("control_action_upsert_failed", "failed to persist denied control action", map[string]any{"action_id": action.ID, "error": err.Error()})
			}
			continue
		}

		// ── Trust gate: check freeze and maintenance window before enqueuing ──
		if blocked, reason, denialCode := a.isExecutionBlocked(action); blocked {
			action.ExecutedAt = now.UTC().Format(time.RFC3339)
			action.CompletedAt = action.ExecutedAt
			action.LifecycleState = control.LifecycleCompleted
			action.Result = func() string {
				if denialCode == control.DenialFreeze {
					return control.ResultDeniedByFreeze
				}
				return control.ResultDeniedByMaintenance
			}()
			action.DenialCode = denialCode
			action.ClosureState = func() string {
				if denialCode == control.DenialFreeze {
					return control.ClosureBlockedByFreeze
				}
				return control.ClosureBlockedByMaintenance
			}()
			action.OutcomeDetail = reason
			if err := a.DB.UpsertControlAction(controlActionRecord(action)); err != nil {
				a.Log.Error("control_action_upsert_failed", "failed to persist freeze-blocked control action", map[string]any{"action_id": action.ID, "error": err.Error()})
			}
			a.Log.Info("control_action_blocked", "control action blocked by trust gate", map[string]any{
				"action_id":   action.ID,
				"action_type": action.ActionType,
				"denial_code": denialCode,
				"reason":      reason,
			})
			continue
		}

		// ── Trust gate: resolve execution mode and check for approval_required ──
		execMode := a.resolveExecutionMode(action)
		action.ExecutionMode = execMode
		action.ProposedBy = "system"

		if execMode == control.ExecutionModeApprovalRequired {
			// Compute approval expiry
			approvalExpiry := ""
			if a.Cfg.Control.ApprovalTimeoutSeconds > 0 {
				approvalExpiry = now.UTC().Add(time.Duration(a.Cfg.Control.ApprovalTimeoutSeconds) * time.Second).Format(time.RFC3339)
			}
			action.ApprovalExpiresAt = approvalExpiry
			action.LifecycleState = control.LifecyclePendingApproval
			action.Result = control.ResultPendingApproval

			// Capture evidence bundle immediately so operator can review it
			thHealth := a.transportHealthJSON(action.TargetTransport)
			bundleID := a.captureEvidenceBundle(action, thHealth)
			action.EvidenceBundleID = bundleID

			if err := a.DB.UpsertControlAction(controlActionRecord(action)); err != nil {
				a.Log.Error("control_action_upsert_failed", "failed to persist approval-required action", map[string]any{"action_id": action.ID, "error": err.Error()})
			}
			a.Log.Info("control_action_pending_approval", "control action held pending operator approval", map[string]any{
				"action_id":          action.ID,
				"action_type":        action.ActionType,
				"approval_expires":   approvalExpiry,
				"evidence_bundle_id": bundleID,
			})
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
			action.LifecycleState = control.LifecycleCompleted
			action.Result = control.ResultFailedTransient
			action.ClosureState = control.ClosureSuperseded
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
		if row.Result != control.ResultExecutedSuccessfully || row.ExpiresAt == "" || row.ClosureState != "" {
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
			followup = control.ControlAction{ID: fmt.Sprintf("%s-reset", row.ID), DecisionID: row.DecisionID, ActionType: control.ActionBackoffReset, TargetTransport: row.TargetTransport, Reason: "backoff increase window expired after quiet recovery evidence", Confidence: 0.9, TriggerEvidence: []string{"anomaly snapshots quiet after expiry"}, CreatedAt: now.UTC().Format(time.RFC3339), Mode: a.Cfg.Control.Mode, PolicyRule: "evidence_based_backoff_reset", LifecycleState: control.LifecyclePending}
		default:
			continue
		}
		if _, ok := a.seenControlAction(followup.ID); ok {
			continue
		}
		row.ClosureState = control.ClosureExpiredAndReverted
		_ = a.DB.UpsertControlAction(row)
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

func advisoryDenied(actionType string) bool {
	for _, item := range control.DefaultActionRealityMatrix() {
		if item.ActionType == actionType {
			return item.AdvisoryOnly || !item.ActuatorExists || !item.SafeForGuardedAuto
		}
	}
	return true
}

func (a *App) enqueueHealthRecheckFollowup(action control.ControlAction) {
	followup := control.ControlAction{
		ID:              fmt.Sprintf("%s-health-recheck", action.ID),
		DecisionID:      action.DecisionID,
		ActionType:      control.ActionTriggerHealthRecheck,
		TargetTransport: action.TargetTransport,
		TargetSegment:   action.TargetSegment,
		Reason:          "post-action health recheck for guarded control closure",
		Confidence:      0.9,
		TriggerEvidence: []string{"prior guarded control action executed successfully"},
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
		Mode:            a.Cfg.Control.Mode,
		PolicyRule:      "post_action_health_recheck",
		LifecycleState:  control.LifecyclePending,
	}
	if _, ok := a.seenControlAction(followup.ID); ok {
		return
	}
	if err := a.DB.UpsertControlAction(controlActionRecord(followup)); err != nil {
		return
	}
	select {
	case a.controlQueue <- followup:
	default:
	}
}

func (a *App) syncControlReality() {
	if a == nil || a.DB == nil {
		return
	}
	for _, item := range control.DefaultActionRealityMatrix() {
		_ = a.DB.UpsertControlActionReality(db.ControlActionRealityRecord{
			ActionType:         item.ActionType,
			ActuatorExists:     item.ActuatorExists,
			Reversible:         item.Reversible,
			BlastRadiusKnown:   item.BlastRadiusKnown,
			BlastRadiusClass:   item.BlastRadiusClass,
			SafeForGuardedAuto: item.SafeForGuardedAuto,
			AdvisoryOnly:       item.AdvisoryOnly,
			DenialCode:         item.DenialCode,
			Notes:              item.Notes,
		})
	}
}

func (a *App) recoverIncompleteControlActions(now time.Time) {
	if a == nil || a.DB == nil {
		return
	}
	rows, err := a.DB.IncompleteControlActions(a.Cfg.Intelligence.Queries.MaxLimit)
	if err != nil {
		return
	}
	for _, row := range rows {
		if strings.TrimSpace(row.ExecutedAt) == "" {
			row.ExecutedAt = now.UTC().Format(time.RFC3339)
		}
		row.CompletedAt = now.UTC().Format(time.RFC3339)
		row.LifecycleState = control.LifecycleRecovered
		row.Result = control.ResultFailedTransient
		row.OutcomeDetail = "process restarted before action completion; safe recovery closed the in-flight action without blind re-execution"
		row.ClosureState = control.ClosureSuperseded
		if row.ActionType == control.ActionBackoffIncrease {
			row.Result = control.ResultExpired
			row.OutcomeDetail = "process restarted before rollback; volatile backoff state reverted to baseline during recovery"
			row.ClosureState = control.ClosureExpiredAndReverted
		}
		_ = a.DB.UpsertControlAction(row)
		if row.ActionType == control.ActionRestartTransport || row.ActionType == control.ActionResubscribeTransport {
			a.enqueueHealthRecheckFollowup(control.ControlAction{
				ID:              row.ID,
				DecisionID:      row.DecisionID,
				ActionType:      row.ActionType,
				TargetTransport: row.TargetTransport,
				TargetSegment:   row.TargetSegment,
			})
		}
	}
}

func (a *App) seenControlAction(id string) (db.ControlActionRecord, bool) {
	row, ok, err := a.DB.ControlActionByID(id)
	if err != nil {
		return db.ControlActionRecord{}, false
	}
	return row, ok
}

func (a *App) executeControlAction(ctx context.Context, action control.ControlAction) {
	if a == nil || a.DB == nil {
		return
	}

	// Re-check freeze and maintenance window at execution time
	// (state may have changed between proposal and execution)
	if blocked, reason, denialCode := a.isExecutionBlocked(action); blocked {
		now := time.Now().UTC().Format(time.RFC3339)
		action.ExecutedAt = now
		action.CompletedAt = now
		action.LifecycleState = control.LifecycleCompleted
		action.DenialCode = denialCode
		action.Result = func() string {
			if denialCode == control.DenialFreeze {
				return control.ResultDeniedByFreeze
			}
			return control.ResultDeniedByMaintenance
		}()
		action.ClosureState = func() string {
			if denialCode == control.DenialFreeze {
				return control.ClosureBlockedByFreeze
			}
			return control.ClosureBlockedByMaintenance
		}()
		action.OutcomeDetail = reason
		_ = a.DB.UpsertControlAction(controlActionRecord(action))
		a.Log.Info("control_action_blocked_at_execution", "action blocked by freeze at execution time", map[string]any{
			"action_id":   action.ID,
			"denial_code": denialCode,
		})
		return
	}

	if advisoryDenied(action.ActionType) {
		now := time.Now().UTC().Format(time.RFC3339)
		action.ExecutedAt = now
		action.CompletedAt = now
		action.LifecycleState = control.LifecycleCompleted
		action.AdvisoryOnly = true
		action.DenialCode = control.DenialMissingActuator
		action.Result = control.ResultDeniedByPolicy
		action.OutcomeDetail = "action remains advisory because the actuator reality matrix marks it unsupported for guarded_auto"
		_ = a.DB.UpsertControlAction(controlActionRecord(action))
		return
	}
	startedAt := time.Now().UTC().Format(time.RFC3339)
	action.ExecutedAt = startedAt
	action.LifecycleState = control.LifecycleRunning
	if err := a.DB.UpsertControlAction(controlActionRecord(action)); err != nil {
		a.Log.Error("control_action_insert_failed", "failed to insert control action", map[string]any{"action_id": action.ID, "error": err.Error()})
		return
	}

	timeout := time.Duration(a.Cfg.Control.ActionTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	maxTimeout := time.Duration(control.MaxActionTimeoutSeconds) * time.Second
	if timeout > maxTimeout {
		timeout = maxTimeout
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
		result, detail = control.ResultDeniedByPolicy, "routing selector is not implemented; deprioritization stays advisory-only"
	case control.ActionTemporarilySuppressNoisySource:
		result, detail = control.ResultDeniedByPolicy, "source suppression actuator is not implemented; suppression stays advisory-only"
	case control.ActionClearSuppression:
		result, detail = control.ResultDeniedByPolicy, "clear_suppression is unavailable because suppression is not implemented as a real actuator"
	case control.ActionTriggerHealthRecheck:
		go a.evaluateTransportIntelligence(time.Now().UTC())
		result, detail = control.ResultExecutedSuccessfully, "scheduled asynchronous health recheck outside the ingest hot path"
	default:
		result, detail = control.ResultFailedTerminal, "unsupported control action type"
	}

	if runCtx.Err() == context.DeadlineExceeded {
		result = control.ResultFailedTransient
		detail = "action timed out after " + timeout.String()
		action.ClosureState = control.ClosureSuperseded
	}

	completedAt := time.Now().UTC().Format(time.RFC3339)
	action.CompletedAt = completedAt
	action.LifecycleState = control.LifecycleCompleted
	action.Result = result
	action.OutcomeDetail = detail
	switch action.ActionType {
	case control.ActionBackoffReset, control.ActionTriggerHealthRecheck:
		if result == control.ResultExecutedSuccessfully {
			action.ClosureState = control.ClosureRecoveredAndClosed
		}
	}
	_ = a.DB.UpsertControlAction(controlActionRecord(action))
	if result == control.ResultExecutedSuccessfully && (action.ActionType == control.ActionRestartTransport || action.ActionType == control.ActionResubscribeTransport) {
		a.enqueueHealthRecheckFollowup(action)
	}
	if result == control.ResultExecutedSuccessfully {
		selfobs.MarkFresh("control")
	}
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
	execMode := action.ExecutionMode
	if execMode == "" {
		execMode = control.ExecutionModeAuto
	}
	proposedBy := action.ProposedBy
	if proposedBy == "" {
		proposedBy = "system"
	}
	blastClass := action.BlastRadiusClass
	if blastClass == "" {
		blastClass = control.BlastRadiusUnknown
	}
	return db.ControlActionRecord{
		ID:                action.ID,
		DecisionID:        action.DecisionID,
		ActionType:        action.ActionType,
		TargetTransport:   action.TargetTransport,
		TargetSegment:     action.TargetSegment,
		TargetNode:        action.TargetNode,
		Reason:            action.Reason,
		Confidence:        action.Confidence,
		TriggerEvidence:   append([]string(nil), action.TriggerEvidence...),
		EpisodeID:         action.EpisodeID,
		CreatedAt:         action.CreatedAt,
		ExecutedAt:        action.ExecutedAt,
		CompletedAt:       action.CompletedAt,
		Result:            action.Result,
		Reversible:        action.Reversible,
		ExpiresAt:         action.ExpiresAt,
		OutcomeDetail:     action.OutcomeDetail,
		Mode:              action.Mode,
		PolicyRule:        action.PolicyRule,
		LifecycleState:    action.LifecycleState,
		AdvisoryOnly:      action.AdvisoryOnly,
		DenialCode:        action.DenialCode,
		ClosureState:      action.ClosureState,
		Metadata:          action.Metadata,
		ExecutionMode:     execMode,
		ProposedBy:        proposedBy,
		ApprovedBy:        action.ApprovedBy,
		ApprovedAt:        action.ApprovedAt,
		RejectedBy:        action.RejectedBy,
		RejectedAt:        action.RejectedAt,
		ApprovalNote:      action.ApprovalNote,
		ApprovalExpiresAt: action.ApprovalExpiresAt,
		BlastRadiusClass:  blastClass,
		EvidenceBundleID:  action.EvidenceBundleID,
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
		DenialCode:        decision.DenialCode,
		SafetyChecks:      decision.SafetyChecks,
		DecisionInputs:    decision.InputSummary,
		PolicySummary:     decision.PolicySummary,
		CreatedAt:         decision.CreatedAt,
		Mode:              decision.Mode,
		OperatorOverride:  decision.OperatorOverride,
	}
}
