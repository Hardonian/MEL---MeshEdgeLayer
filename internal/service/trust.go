package service

// trust.go — Service-layer implementation of the control-plane trust model:
//   - Approval gate enforcement (approval_required execution mode)
//   - Freeze / maintenance-mode check before action execution
//   - Evidence bundle capture on action creation
//   - Approve / Reject action methods
//   - Freeze creation / clearing
//   - Maintenance window management
//   - Operator notes
//   - Timeline query
//   - Action inspect (full evidence + decision bundle)
//   - Approval expiry cleanup loop

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/auth"
	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/operatorlang"
	"github.com/mel-project/mel/internal/selfobs"
)

// ApprovalOpts configures approve/reject behavior beyond the default canonical path.
type ApprovalOpts struct {
	// BreakGlassLegacyCLI is set only by mel control approve|reject after explicit CLI ack.
	// It records durable metadata and allows separation-of-duties override when the human
	// proposer would otherwise match the approver identity.
	BreakGlassLegacyCLI bool
	// BreakGlassHTTP is set by the HTTP API when the client acknowledges break-glass SoD override.
	BreakGlassHTTP bool
	// BreakGlassSodReason is persisted on the control_actions row when SoD bypass applies.
	BreakGlassSodReason string
}

func breakGlassEffective(opts ApprovalOpts) bool {
	return opts.BreakGlassLegacyCLI || opts.BreakGlassHTTP
}

func approvalSodWouldBlock(rec db.ControlActionRecord, actorID string) bool {
	if rec.ExecutionMode != control.ExecutionModeApprovalRequired {
		return false
	}
	if rec.LifecycleState != control.LifecyclePendingApproval {
		return false
	}
	submitter := strings.TrimSpace(rec.SubmittedBy)
	if submitter == "" {
		submitter = strings.TrimSpace(rec.ProposedBy)
	}
	if submitter == "" || strings.EqualFold(submitter, "system") {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(actorID), submitter)
}

func (a *App) mergeControlActionMetadata(actionID string, patch map[string]any) error {
	if a == nil || a.DB == nil || len(patch) == 0 {
		return nil
	}
	rec, ok, err := a.DB.ControlActionByID(actionID)
	if err != nil {
		return fmt.Errorf("could not load action for metadata merge: %w", err)
	}
	if !ok {
		return fmt.Errorf("action not found for metadata merge: %s", actionID)
	}
	if rec.Metadata == nil {
		rec.Metadata = map[string]any{}
	}
	for k, v := range patch {
		rec.Metadata[k] = v
	}
	return a.DB.UpsertControlAction(rec)
}

// ─── Execution mode helpers ───────────────────────────────────────────────────

// resolveExecutionMode returns the effective execution mode for an action.
// Config policy can force approval for specific action types or high blast radius.
func (a *App) resolveExecutionMode(action control.ControlAction) string {
	cfg := a.Cfg.Control

	// Check if action type requires approval
	for _, t := range cfg.RequireApprovalForActionTypes {
		if strings.EqualFold(t, action.ActionType) {
			return control.ExecutionModeApprovalRequired
		}
	}

	// Check blast radius policy
	if cfg.RequireApprovalForHighBlastRadius {
		switch action.BlastRadiusClass {
		case control.BlastRadiusMesh, control.BlastRadiusGlobal:
			return control.ExecutionModeApprovalRequired
		}
	}

	// Default: auto
	return control.ExecutionModeAuto
}

// ─── Freeze and maintenance checks ───────────────────────────────────────────

// isExecutionBlocked checks if the action is blocked by an active freeze or
// maintenance window. Returns (blocked, reason, denial_code).
func (a *App) isExecutionBlocked(action control.ControlAction) (bool, string, string) {
	if a.DB == nil {
		return false, "", ""
	}

	// Check freeze
	frozen, reason, err := a.DB.IsFrozen(action.TargetTransport, action.ActionType)
	if err != nil {
		a.Log.Error("freeze_check_failed", "could not verify freeze status; allowing as fail-open", map[string]any{"error": err.Error()})
	} else if frozen {
		return true, "execution blocked by active freeze: " + reason, control.DenialFreeze
	}

	// Check maintenance window
	inMaintenance, reason, err := a.DB.IsInMaintenance(action.TargetTransport, time.Now().UTC())
	if err != nil {
		a.Log.Error("maintenance_check_failed", "could not verify maintenance window; allowing as fail-open", map[string]any{"error": err.Error()})
	} else if inMaintenance {
		return true, "execution blocked by active maintenance window: " + reason, control.DenialMaintenance
	}

	return false, "", ""
}

// ─── Evidence bundle capture ──────────────────────────────────────────────────

// captureEvidenceBundle creates a durable evidence bundle for an action.
// It pulls together the current transport health, recent anomaly summary,
// policy version, and any prior relevant decisions.
func (a *App) captureEvidenceBundle(action control.ControlAction, transportHealthJSON map[string]any) string {
	if a.DB == nil {
		return ""
	}

	bundleID := newTrustID("eb")

	// Build anomaly summary from recent history
	anomalies := []any{}
	if strings.TrimSpace(action.TargetTransport) != "" {
		start := time.Now().Add(-30 * time.Minute).UTC().Format(time.RFC3339)
		rows, err := a.DB.TransportAnomalyHistory(action.TargetTransport, start, "", 20, 0)
		if err == nil {
			for _, r := range rows {
				anomalies = append(anomalies, map[string]any{
					"bucket_start": r.BucketStart,
					"count":        r.Count,
					"dead_letters": r.DeadLetters,
					"obs_drops":    r.ObservationDrops,
				})
			}
		}
	}

	// Explanation from trigger evidence
	explanation := map[string]any{
		"trigger_evidence": action.TriggerEvidence,
		"reason":           action.Reason,
		"confidence":       action.Confidence,
		"policy_rule":      action.PolicyRule,
	}

	// Prior decisions for same transport (last 5)
	priorDecisions := []any{}
	if a.DB != nil && strings.TrimSpace(action.TargetTransport) != "" {
		rows, err := a.DB.ControlDecisions(action.TargetTransport, "", "", "", 5, 0)
		if err == nil {
			for _, r := range rows {
				priorDecisions = append(priorDecisions, map[string]any{
					"id":          r.ID,
					"action_type": r.ActionType,
					"allowed":     r.Allowed,
					"created_at":  r.CreatedAt,
				})
			}
		}
	}

	// Compute integrity hash over deterministic fields
	hashInput := map[string]any{
		"action_id":  action.ID,
		"type":       action.ActionType,
		"transport":  action.TargetTransport,
		"reason":     action.Reason,
		"confidence": action.Confidence,
		"created_at": action.CreatedAt,
	}
	hashBytes, _ := json.Marshal(hashInput)
	sum := sha256.Sum256(hashBytes)
	integrityHash := hex.EncodeToString(sum[:])

	bundle := db.EvidenceBundleRecord{
		ID:              bundleID,
		ActionID:        action.ID,
		DecisionID:      action.DecisionID,
		Anomalies:       anomalies,
		Explanation:     explanation,
		TransportHealth: transportHealthJSON,
		PriorDecisions:  priorDecisions,
		PolicyVersion:   a.Cfg.Control.Mode + "/" + fmt.Sprintf("%d", len(a.Cfg.Control.AllowedActions)),
		IntegrityHash:   integrityHash,
		SourceType:      "system",
	}

	if err := a.DB.UpsertEvidenceBundle(bundle); err != nil {
		a.Log.Error("evidence_bundle_capture_failed", "could not persist evidence bundle", map[string]any{
			"action_id": action.ID,
			"error":     err.Error(),
		})
		return ""
	}

	return bundleID
}

// ─── Approval workflow ────────────────────────────────────────────────────────

func requiresSeparateApproverForRecord(rec db.ControlActionRecord) bool {
	if rec.RequiresSeparateApprover {
		return true
	}
	return rec.ExecutionMode == control.ExecutionModeApprovalRequired
}

// ApproveAction approves a pending_approval action and queues it for execution.
// actorID is the operator performing the approval.
// breakGlassSodAck/breakGlassSodReason are used by API and CLI when separation-of-duties would block same-actor approval.
func (a *App) ApproveAction(actionID, actorID, note string, breakGlassSodAck bool, breakGlassSodReason string) (*models.ApproveActionResponse, error) {
	return a.ApproveActionWithOpts(actionID, actorID, note, ApprovalOpts{
		BreakGlassLegacyCLI: breakGlassSodAck,
		BreakGlassSodReason: breakGlassSodReason,
	})
}

// ApproveActionWithOpts approves a pending_approval action with optional break-glass behavior.
func (a *App) ApproveActionWithOpts(actionID, actorID, note string, opts ApprovalOpts) (*models.ApproveActionResponse, error) {
	if a == nil || a.DB == nil {
		return nil, fmt.Errorf("service not available")
	}
	if strings.TrimSpace(actionID) == "" {
		return nil, fmt.Errorf("action_id is required")
	}
	if strings.TrimSpace(actorID) == "" {
		actorID = "system"
	}

	// Verify the action exists and is pending approval
	rec, ok, err := a.DB.ControlActionByID(actionID)
	if err != nil {
		return nil, fmt.Errorf("could not load action: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("action not found: %s", actionID)
	}
	if rec.LifecycleState != control.LifecyclePendingApproval {
		return nil, fmt.Errorf("action %s is not pending approval (state: %s)", actionID, rec.LifecycleState)
	}

	submitter := strings.TrimSpace(rec.SubmittedBy)
	if submitter == "" {
		submitter = strings.TrimSpace(rec.ProposedBy)
	}
	if submitter == "" {
		submitter = "system"
	}

	// Check if approval has already expired
	if rec.ApprovalExpiresAt != "" {
		exp, err2 := time.Parse(time.RFC3339, rec.ApprovalExpiresAt)
		if err2 == nil && time.Now().UTC().After(exp) {
			return nil, fmt.Errorf("approval window for action %s has expired", actionID)
		}
	}

	if approvalSodWouldBlock(rec, actorID) && opts.BreakGlassHTTP && strings.TrimSpace(opts.BreakGlassSodReason) == "" {
		return nil, fmt.Errorf("break_glass_sod_reason is required when break_glass_sod_ack is true for same-submitter approval")
	}

	if approvalSodWouldBlock(rec, actorID) && !breakGlassEffective(opts) {
		pol := a.EvaluateApprovalPolicyForRecord(rec, actorID)
		_ = a.DB.InsertRBACAuditLog(auth.AuditEntry{
			ID:           newTrustID("aud"),
			ActorID:      auth.OperatorID(actorID),
			ActionClass:  auth.ActionControl,
			ActionDetail: "approve_action_denied",
			ResourceType: "control_action",
			ResourceID:   actionID,
			Reason:       pol.ApproverDenialReason,
			Result:       auth.AuditResultDenied,
			Timestamp:    time.Now().UTC(),
		})
		return nil, fmt.Errorf("separation of duties: submitter and approver cannot be the same operator (%s); use a different operator identity, HTTP break_glass_sod_ack with reason, or mel control approve with --i-understand-break-glass-sod (emergency only)", strings.TrimSpace(rec.SubmittedBy))
	}

	sodBypass := opts.BreakGlassLegacyCLI && approvalSodWouldBlock(rec, actorID)
	sodReason := strings.TrimSpace(opts.BreakGlassSodReason)
	if sodBypass && sodReason == "" {
		sodReason = strings.TrimSpace(note)
	}

	if err := a.DB.ApproveControlAction(actionID, actorID, note, sodBypass, actorID, sodReason); err != nil {
		return nil, fmt.Errorf("could not approve action: %w", err)
	}

	if opts.BreakGlassLegacyCLI {
		now := time.Now().UTC().Format(time.RFC3339)
		patch := map[string]any{
			"mel_break_glass_approval": true,
			"mel_break_glass_path":     "mel_control_legacy",
			"mel_break_glass_at":       now,
			"mel_break_glass_actor":    actorID,
		}
		if approvalSodWouldBlock(rec, actorID) {
			patch["mel_break_glass_sod_override"] = true
		}
		if err := a.mergeControlActionMetadata(actionID, patch); err != nil {
			return nil, fmt.Errorf("approved action but could not persist break-glass metadata: %w", err)
		}
	}
	if opts.BreakGlassHTTP && approvalSodWouldBlock(rec, actorID) {
		now := time.Now().UTC().Format(time.RFC3339)
		patch := map[string]any{
			"mel_break_glass_approval":     true,
			"mel_break_glass_path":         "http_api",
			"mel_break_glass_at":           now,
			"mel_break_glass_actor":        actorID,
			"mel_break_glass_sod_override": true,
		}
		if err := a.mergeControlActionMetadata(actionID, patch); err != nil {
			return nil, fmt.Errorf("approved action but could not persist break-glass metadata: %w", err)
		}
	}

	// Audit the approval
	_ = a.DB.InsertRBACAuditLog(auth.AuditEntry{
		ID:           newTrustID("aud"),
		ActorID:      auth.OperatorID(actorID),
		ActionClass:  auth.ActionControl,
		ActionDetail: "approve_action",
		ResourceType: "control_action",
		ResourceID:   actionID,
		Reason: func() string {
			if opts.BreakGlassLegacyCLI {
				if strings.TrimSpace(note) != "" {
					return "break_glass_legacy_cli: " + note
				}
				return "break_glass_legacy_cli"
			}
			if opts.BreakGlassHTTP {
				r := strings.TrimSpace(opts.BreakGlassSodReason)
				if r != "" {
					return "break_glass_http_api: " + r
				}
				return "break_glass_http_api"
			}
			return note
		}(),
		Result:    auth.AuditResultSuccess,
		Timestamp: time.Now().UTC(),
	})

	a.Log.Info("action_approved", "operator approved pending control action", map[string]any{
		"action_id":    actionID,
		"actor":        actorID,
		"sod_bypass":   sodBypass,
		"submitted_by": submitter,
	})
	a.integrationForwardControlAction("action approved: "+actionID, map[string]any{
		"action_id": actionID,
		"actor":     actorID,
		"decision":  "approved",
	})
	_ = a.DB.InsertTimelineEvent(db.TimelineEvent{
		EventID:    newTrustID("tl"),
		EventType:  "action_approved",
		Summary:    "action approved: " + actionID,
		Severity:   "info",
		ActorID:    actorID,
		ResourceID: actionID,
		Details: map[string]any{
			"action_id":              actionID,
			"note":                   note,
			"break_glass_legacy_cli": opts.BreakGlassLegacyCLI,
			"break_glass_http":       opts.BreakGlassHTTP,
		},
	})

	// Re-load the action and queue for execution
	updated, ok, err := a.DB.ControlActionByID(actionID)
	if err != nil || !ok {
		return nil, fmt.Errorf("could not reload approved action: %v", err)
	}

	action := db_ControlActionRecordToControlAction(updated)
	queued := false
	select {
	case a.controlQueue <- action:
		queued = true
	default:
		a.Log.Error("control_queue_full", "approved action could not be queued", map[string]any{"action_id": actionID})
		return nil, fmt.Errorf("control queue full; action approved but could not be queued immediately")
	}

	out := models.ApproveActionResponse{
		Status:                                 "approved",
		ActionID:                               actionID,
		ActorID:                                actorID,
		LifecycleState:                         updated.LifecycleState,
		Result:                                 updated.Result,
		FullyApprovedSingleStep:                true,
		ApprovalDoesNotImplyExecution:          true,
		QueuedForExecution:                     queued,
		ExecutionOccurred:                      false,
		HTTPApproveDoesNotDrainQueue:           true,
		BacklogMayRemain:                       true,
		BacklogExecutionRequiresActiveExecutor: true,
		Policy:                                 approvalPolicyToDTO(a.EvaluateApprovalPolicyForRecord(updated, "")),
	}
	return &out, nil
}

// RejectAction rejects a pending_approval action and marks it closed (CLI / internal).
func (a *App) RejectAction(actionID, actorID, note string) error {
	_, err := a.RejectActionWithOpts(actionID, actorID, note, ApprovalOpts{})
	return err
}

// RejectActionHTTP is wired to the HTTP API (break-glass flags from JSON body).
func (a *App) RejectActionHTTP(actionID, actorID, note string, breakGlassSodAck bool, breakGlassSodReason string) (*models.RejectActionResponse, error) {
	return a.RejectActionWithOpts(actionID, actorID, note, ApprovalOpts{
		BreakGlassHTTP:      breakGlassSodAck,
		BreakGlassSodReason: breakGlassSodReason,
	})
}

// RejectActionWithOpts rejects a pending_approval action with optional break-glass metadata.
func (a *App) RejectActionWithOpts(actionID, actorID, note string, opts ApprovalOpts) (*models.RejectActionResponse, error) {
	if a == nil || a.DB == nil {
		return nil, fmt.Errorf("service not available")
	}
	if strings.TrimSpace(actionID) == "" {
		return nil, fmt.Errorf("action_id is required")
	}
	if strings.TrimSpace(actorID) == "" {
		actorID = "system"
	}

	rec, ok, err := a.DB.ControlActionByID(actionID)
	if err != nil {
		return nil, fmt.Errorf("could not load action: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("action not found: %s", actionID)
	}
	if rec.LifecycleState != control.LifecyclePendingApproval {
		return nil, fmt.Errorf("action %s is not pending approval (state: %s)", actionID, rec.LifecycleState)
	}

	if approvalSodWouldBlock(rec, actorID) && opts.BreakGlassHTTP && strings.TrimSpace(opts.BreakGlassSodReason) == "" {
		return nil, fmt.Errorf("break_glass_sod_reason is required when break_glass_sod_ack is true for same-submitter rejection")
	}

	if approvalSodWouldBlock(rec, actorID) && !breakGlassEffective(opts) {
		pol := a.EvaluateApprovalPolicyForRecord(rec, actorID)
		_ = a.DB.InsertRBACAuditLog(auth.AuditEntry{
			ID:           newTrustID("aud"),
			ActorID:      auth.OperatorID(actorID),
			ActionClass:  auth.ActionControl,
			ActionDetail: "reject_action_denied",
			ResourceType: "control_action",
			ResourceID:   actionID,
			Reason:       pol.ApproverDenialReason,
			Result:       auth.AuditResultDenied,
			Timestamp:    time.Now().UTC(),
		})
		return nil, fmt.Errorf("separation of duties: submitter and rejector cannot be the same operator (%s); use a different operator identity, HTTP break_glass_sod_ack with reason, or mel control reject with --i-understand-break-glass-sod (emergency only)", strings.TrimSpace(rec.SubmittedBy))
	}

	if err := a.DB.RejectControlAction(actionID, actorID, note); err != nil {
		return nil, fmt.Errorf("could not reject action: %w", err)
	}

	if opts.BreakGlassLegacyCLI {
		now := time.Now().UTC().Format(time.RFC3339)
		patch := map[string]any{
			"mel_break_glass_reject": true,
			"mel_break_glass_path":   "mel_control_legacy",
			"mel_break_glass_at":     now,
			"mel_break_glass_actor":  actorID,
		}
		if approvalSodWouldBlock(rec, actorID) {
			patch["mel_break_glass_sod_override"] = true
		}
		if err := a.mergeControlActionMetadata(actionID, patch); err != nil {
			return nil, fmt.Errorf("rejected action but could not persist break-glass metadata: %w", err)
		}
	}
	if opts.BreakGlassHTTP && approvalSodWouldBlock(rec, actorID) {
		now := time.Now().UTC().Format(time.RFC3339)
		patch := map[string]any{
			"mel_break_glass_reject":       true,
			"mel_break_glass_path":         "http_api",
			"mel_break_glass_at":           now,
			"mel_break_glass_actor":        actorID,
			"mel_break_glass_sod_override": true,
		}
		if err := a.mergeControlActionMetadata(actionID, patch); err != nil {
			return nil, fmt.Errorf("rejected action but could not persist break-glass metadata: %w", err)
		}
	}

	_ = a.DB.InsertRBACAuditLog(auth.AuditEntry{
		ID:           newTrustID("aud"),
		ActorID:      auth.OperatorID(actorID),
		ActionClass:  auth.ActionControl,
		ActionDetail: "reject_action",
		ResourceType: "control_action",
		ResourceID:   actionID,
		Reason: func() string {
			if opts.BreakGlassLegacyCLI {
				if strings.TrimSpace(note) != "" {
					return "break_glass_legacy_cli: " + note
				}
				return "break_glass_legacy_cli"
			}
			if opts.BreakGlassHTTP {
				r := strings.TrimSpace(opts.BreakGlassSodReason)
				if r != "" {
					return "break_glass_http_api: " + r
				}
				return "break_glass_http_api"
			}
			return note
		}(),
		Result:    auth.AuditResultSuccess,
		Timestamp: time.Now().UTC(),
	})

	a.Log.Info("action_rejected", "operator rejected pending control action", map[string]any{
		"action_id": actionID,
		"actor":     actorID,
	})
	a.integrationForwardControlAction("action rejected: "+actionID, map[string]any{
		"action_id": actionID,
		"actor":     actorID,
		"decision":  "rejected",
	})
	_ = a.DB.InsertTimelineEvent(db.TimelineEvent{
		EventID:    newTrustID("tl"),
		EventType:  "action_rejected",
		Summary:    "action rejected: " + actionID,
		Severity:   "warning",
		ActorID:    actorID,
		ResourceID: actionID,
		Details: map[string]any{
			"action_id":              actionID,
			"note":                   note,
			"break_glass_legacy_cli": opts.BreakGlassLegacyCLI,
			"break_glass_http":       opts.BreakGlassHTTP,
		},
	})
	updated, ok2, err2 := a.DB.ControlActionByID(actionID)
	if err2 != nil || !ok2 {
		return nil, fmt.Errorf("could not reload rejected action: %v", err2)
	}
	return &models.RejectActionResponse{
		Status:         "rejected",
		ActionID:       actionID,
		ActorID:        actorID,
		LifecycleState: updated.LifecycleState,
		Result:         updated.Result,
		Policy:         approvalPolicyToDTO(a.EvaluateApprovalPolicyForRecord(updated, "")),
	}, nil
}

// ─── Freeze management ────────────────────────────────────────────────────────

// CreateFreeze creates a new control freeze record.
func (a *App) CreateFreeze(scopeType, scopeValue, reason, createdBy string, expiresAt string) (string, error) {
	if a == nil || a.DB == nil {
		return "", fmt.Errorf("service not available")
	}
	id := newTrustID("frz")
	rec := db.FreezeRecord{
		ID:         id,
		ScopeType:  scopeType,
		ScopeValue: scopeValue,
		Reason:     reason,
		CreatedBy:  createdBy,
		ExpiresAt:  expiresAt,
		Active:     true,
	}
	if err := a.DB.CreateFreeze(rec); err != nil {
		return "", fmt.Errorf("could not create freeze: %w", err)
	}
	_ = a.DB.InsertRBACAuditLog(auth.AuditEntry{
		ID:           newTrustID("aud"),
		ActorID:      auth.OperatorID(createdBy),
		ActionClass:  auth.ActionControl,
		ActionDetail: "create_freeze",
		ResourceType: "control_freeze",
		ResourceID:   id,
		Reason:       reason,
		Result:       auth.AuditResultSuccess,
		Timestamp:    time.Now().UTC(),
	})
	a.Log.Info("control_freeze_created", "automation freeze created", map[string]any{
		"freeze_id":   id,
		"scope_type":  scopeType,
		"scope_value": scopeValue,
		"reason":      reason,
	})
	_ = a.DB.InsertTimelineEvent(db.TimelineEvent{
		EventID:    newTrustID("tl"),
		EventType:  "freeze_created",
		Summary:    "freeze created: " + scopeType + " " + scopeValue,
		Severity:   "warning",
		ActorID:    createdBy,
		ResourceID: id,
		Details:    map[string]any{"freeze_id": id, "scope_type": scopeType, "scope_value": scopeValue, "reason": reason},
	})
	return id, nil
}

// ClearFreeze removes an active freeze.
func (a *App) ClearFreeze(freezeID, clearedBy string) error {
	if a == nil || a.DB == nil {
		return fmt.Errorf("service not available")
	}
	if err := a.DB.ClearFreeze(freezeID, clearedBy); err != nil {
		return fmt.Errorf("could not clear freeze: %w", err)
	}
	_ = a.DB.InsertRBACAuditLog(auth.AuditEntry{
		ID:           newTrustID("aud"),
		ActorID:      auth.OperatorID(clearedBy),
		ActionClass:  auth.ActionControl,
		ActionDetail: "clear_freeze",
		ResourceType: "control_freeze",
		ResourceID:   freezeID,
		Result:       auth.AuditResultSuccess,
		Timestamp:    time.Now().UTC(),
	})
	a.Log.Info("control_freeze_cleared", "automation freeze cleared", map[string]any{
		"freeze_id":  freezeID,
		"cleared_by": clearedBy,
	})
	_ = a.DB.InsertTimelineEvent(db.TimelineEvent{
		EventID:    newTrustID("tl"),
		EventType:  "freeze_cleared",
		Summary:    "freeze cleared: " + freezeID,
		Severity:   "info",
		ActorID:    clearedBy,
		ResourceID: freezeID,
		Details:    map[string]any{"freeze_id": freezeID},
	})
	return nil
}

// ─── Maintenance windows ──────────────────────────────────────────────────────

// CreateMaintenanceWindow creates a new maintenance window record.
func (a *App) CreateMaintenanceWindow(title, reason, scopeType, scopeValue, createdBy, startsAt, endsAt string) (string, error) {
	if a == nil || a.DB == nil {
		return "", fmt.Errorf("service not available")
	}
	if strings.TrimSpace(startsAt) == "" || strings.TrimSpace(endsAt) == "" {
		return "", fmt.Errorf("starts_at and ends_at are required")
	}
	id := newTrustID("mw")
	rec := db.MaintenanceWindowRecord{
		ID:         id,
		Title:      title,
		Reason:     reason,
		ScopeType:  scopeType,
		ScopeValue: scopeValue,
		StartsAt:   startsAt,
		EndsAt:     endsAt,
		CreatedBy:  createdBy,
		Active:     true,
	}
	if err := a.DB.CreateMaintenanceWindow(rec); err != nil {
		return "", fmt.Errorf("could not create maintenance window: %w", err)
	}
	_ = a.DB.InsertRBACAuditLog(auth.AuditEntry{
		ID:           newTrustID("aud"),
		ActorID:      auth.OperatorID(createdBy),
		ActionClass:  auth.ActionControl,
		ActionDetail: "create_maintenance_window",
		ResourceType: "maintenance_window",
		ResourceID:   id,
		Reason:       reason,
		Result:       auth.AuditResultSuccess,
		Timestamp:    time.Now().UTC(),
	})
	a.Log.Info("maintenance_window_created", "maintenance window created", map[string]any{
		"window_id": id,
		"starts_at": startsAt,
		"ends_at":   endsAt,
	})
	_ = a.DB.InsertTimelineEvent(db.TimelineEvent{
		EventID:    newTrustID("tl"),
		EventType:  "maintenance_created",
		Summary:    "maintenance window created: " + title,
		Severity:   "info",
		ActorID:    createdBy,
		ResourceID: id,
		Details:    map[string]any{"window_id": id, "title": title, "starts_at": startsAt, "ends_at": endsAt},
	})
	return id, nil
}

// CancelMaintenanceWindow cancels an active maintenance window.
func (a *App) CancelMaintenanceWindow(windowID, cancelledBy string) error {
	if a == nil || a.DB == nil {
		return fmt.Errorf("service not available")
	}
	if err := a.DB.CancelMaintenanceWindow(windowID, cancelledBy); err != nil {
		return fmt.Errorf("could not cancel maintenance window: %w", err)
	}
	_ = a.DB.InsertRBACAuditLog(auth.AuditEntry{
		ID:           newTrustID("aud"),
		ActorID:      auth.OperatorID(cancelledBy),
		ActionClass:  auth.ActionControl,
		ActionDetail: "cancel_maintenance_window",
		ResourceType: "maintenance_window",
		ResourceID:   windowID,
		Result:       auth.AuditResultSuccess,
		Timestamp:    time.Now().UTC(),
	})
	_ = a.DB.InsertTimelineEvent(db.TimelineEvent{
		EventID:    newTrustID("tl"),
		EventType:  "maintenance_cancelled",
		Summary:    "maintenance window cancelled: " + windowID,
		Severity:   "info",
		ActorID:    cancelledBy,
		ResourceID: windowID,
		Details:    map[string]any{"window_id": windowID},
	})
	return nil
}

// ─── Operator notes ───────────────────────────────────────────────────────────

// AddOperatorNote attaches a note to any resource reference.
func (a *App) AddOperatorNote(refType, refID, actorID, content string) (string, error) {
	if a == nil || a.DB == nil {
		return "", fmt.Errorf("service not available")
	}
	if strings.TrimSpace(refType) == "" || strings.TrimSpace(refID) == "" {
		return "", fmt.Errorf("ref_type and ref_id are required")
	}
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("note content is required")
	}
	id := newTrustID("note")
	note := db.OperatorNoteRecord{
		ID:      id,
		RefType: refType,
		RefID:   refID,
		ActorID: actorID,
		Content: content,
	}
	if err := a.DB.CreateOperatorNote(note); err != nil {
		return "", fmt.Errorf("could not create operator note: %w", err)
	}
	return id, nil
}

// ─── Timeline ─────────────────────────────────────────────────────────────────

// Timeline returns a unified chronological event feed.
func (a *App) Timeline(start, end string, limit int) ([]db.TimelineEvent, error) {
	if a == nil || a.DB == nil {
		return []db.TimelineEvent{}, nil
	}
	return a.DB.TimelineEvents(start, end, limit)
}

// ─── Action inspect (full evidence + decision bundle) ─────────────────────────

// InspectAction returns the full evidence bundle, decision record, and action
// record for a given action ID. This is the operator-grade reconstruction path.
func (a *App) InspectAction(actionID string) (map[string]any, error) {
	if a == nil || a.DB == nil {
		return nil, fmt.Errorf("service not available")
	}

	action, ok, err := a.DB.ControlActionByID(actionID)
	if err != nil {
		return nil, fmt.Errorf("could not load action: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("action not found: %s", actionID)
	}

	var decision *db.ControlDecisionRecord
	if strings.TrimSpace(action.DecisionID) != "" {
		d, ok2, err2 := a.DB.ControlDecisionByID(action.DecisionID)
		if err2 == nil && ok2 {
			decision = &d
		}
	}

	var evidenceBundle *db.EvidenceBundleRecord
	if strings.TrimSpace(action.EvidenceBundleID) != "" {
		b, ok2, err2 := a.DB.EvidenceBundleByID(action.EvidenceBundleID)
		if err2 == nil && ok2 {
			evidenceBundle = &b
		}
	} else {
		b, ok2, err2 := a.DB.EvidenceBundleByActionID(actionID)
		if err2 == nil && ok2 {
			evidenceBundle = &b
		}
	}

	notes, _ := a.DB.OperatorNotesByRef("action", actionID, 50)

	out := map[string]any{
		"action":          action,
		"decision":        decision,
		"evidence_bundle": evidenceBundle,
		"notes":           notes,
		"inspected_at":    time.Now().UTC().Format(time.RFC3339),
		"operator_view":   operatorlang.ActionOperatorLabels(action),
		"approval_policy": approvalPolicyToDTO(a.EvaluateApprovalPolicyForRecord(action, "")),
	}
	return out, nil
}

// ─── Control plane operational state ─────────────────────────────────────────

// OperationalState returns the current control plane operational posture:
// freeze status, maintenance mode, approval backlog, and automation mode.
func (a *App) OperationalState() (map[string]any, error) {
	if a == nil || a.DB == nil {
		return map[string]any{"status": "degraded", "reason": db.ErrDatabaseUnavailable}, nil
	}
	state, err := a.DB.ControlPlaneStateSnapshot(time.Now().UTC())
	if err != nil {
		return nil, err
	}
	raw, _ := state["pending_approvals"].([]db.ControlActionRecord)
	if len(raw) == 0 {
		return state, nil
	}
	enriched := make([]map[string]any, 0, len(raw))
	for _, row := range raw {
		b, err := json.Marshal(row)
		if err != nil {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			continue
		}
		m["operator_view"] = operatorlang.ActionOperatorLabels(row)
		enriched = append(enriched, m)
	}
	state["pending_approvals"] = enriched
	return state, nil
}

// ─── Approval expiry cleanup ──────────────────────────────────────────────────

// approvalBacklogWarnThreshold is the number of pending-approval actions above
// which a warning timeline event is emitted.
const approvalBacklogWarnThreshold = 5

// cleanupExpiredApprovals is called periodically to expire stale pending-approval
// actions and time-expired freezes. It also emits self-observability signals and
// trust-layer warning events if the approval backlog is high.
func (a *App) cleanupExpiredApprovals() {
	if a == nil || a.DB == nil {
		return
	}
	now := time.Now().UTC()
	failed := false
	if err := a.DB.ExpireStaleApprovalActions(now); err != nil {
		a.Log.Error("approval_expiry_cleanup_failed", "could not expire stale approval actions", map[string]any{"error": err.Error()})
		selfobs.GetGlobalRegistry().RecordFailure("trust")
		failed = true
	}
	if err := a.DB.ExpireOldFreezes(now); err != nil {
		a.Log.Error("freeze_expiry_cleanup_failed", "could not expire old freezes", map[string]any{"error": err.Error()})
		selfobs.GetGlobalRegistry().RecordFailure("trust")
		failed = true
	}
	if !failed {
		selfobs.GetGlobalRegistry().RecordSuccess("trust")
		selfobs.MarkFresh("trust")

		// Check for high approval backlog and emit a warning timeline event.
		pending, err := a.DB.PendingApprovalActions(approvalBacklogWarnThreshold + 1)
		if err == nil && len(pending) >= approvalBacklogWarnThreshold {
			a.Log.Warn("approval_backlog_high", "pending-approval action backlog is high", map[string]any{
				"count": len(pending),
			})
			_ = a.DB.InsertTimelineEvent(db.TimelineEvent{
				EventID:   newTrustID("tl"),
				EventType: "approval_backlog_warn",
				Summary:   fmt.Sprintf("approval backlog: %d actions awaiting operator approval", len(pending)),
				Severity:  "warning",
				ActorID:   "system",
				Details:   map[string]any{"backlog_count": len(pending)},
			})
		}
	}
}

// ─── Health snapshot helper ───────────────────────────────────────────────────

// transportHealthJSON returns the current transport health as a map, suitable
// for evidence bundle capture. Returns an empty map if unavailable.
func (a *App) transportHealthJSON(transportName string) map[string]any {
	if a == nil {
		return map[string]any{}
	}
	healthList := a.TransportHealth()
	result := map[string]any{}
	for _, h := range healthList {
		if strings.TrimSpace(transportName) == "" || h.Name == transportName {
			result[h.Name] = map[string]any{
				"state":  h.State,
				"ok":     h.OK,
				"detail": h.Detail,
			}
		}
	}
	return result
}

// ─── Utility ──────────────────────────────────────────────────────────────────

func newTrustID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

// ControlActionRecordToControlAction is a bridge function to re-hydrate a DB
// record back into a ControlAction for re-queuing after approval.
// This is defined here to keep db package free of service imports.
func db_ControlActionRecordToControlAction(r db.ControlActionRecord) control.ControlAction {
	return control.ControlAction{
		ID:                                r.ID,
		DecisionID:                        r.DecisionID,
		ActionType:                        r.ActionType,
		TargetTransport:                   r.TargetTransport,
		TargetSegment:                     r.TargetSegment,
		TargetNode:                        r.TargetNode,
		Reason:                            r.Reason,
		Confidence:                        r.Confidence,
		TriggerEvidence:                   append([]string(nil), r.TriggerEvidence...),
		EpisodeID:                         r.EpisodeID,
		CreatedAt:                         r.CreatedAt,
		ExecutedAt:                        r.ExecutedAt,
		CompletedAt:                       r.CompletedAt,
		Result:                            r.Result,
		Reversible:                        r.Reversible,
		ExpiresAt:                         r.ExpiresAt,
		OutcomeDetail:                     r.OutcomeDetail,
		Mode:                              r.Mode,
		PolicyRule:                        r.PolicyRule,
		LifecycleState:                    r.LifecycleState,
		AdvisoryOnly:                      r.AdvisoryOnly,
		DenialCode:                        r.DenialCode,
		ClosureState:                      r.ClosureState,
		Metadata:                          r.Metadata,
		ExecutionMode:                     r.ExecutionMode,
		ProposedBy:                        r.ProposedBy,
		ApprovedBy:                        r.ApprovedBy,
		ApprovedAt:                        r.ApprovedAt,
		RejectedBy:                        r.RejectedBy,
		RejectedAt:                        r.RejectedAt,
		ApprovalNote:                      r.ApprovalNote,
		ApprovalExpiresAt:                 r.ApprovalExpiresAt,
		BlastRadiusClass:                  r.BlastRadiusClass,
		EvidenceBundleID:                  r.EvidenceBundleID,
		SubmittedBy:                       r.SubmittedBy,
		RequiresSeparateApprover:          r.RequiresSeparateApprover,
		IncidentID:                        r.IncidentID,
		ExecutionStartedAt:                r.ExecutionStartedAt,
		SodBypass:                         r.SodBypass,
		SodBypassActor:                    r.SodBypassActor,
		SodBypassReason:                   r.SodBypassReason,
		ApprovalMode:                      r.ApprovalMode,
		RequiredApprovals:                 r.RequiredApprovals,
		CollectedApprovals:                r.CollectedApprovals,
		ApprovalBasis:                     append([]string(nil), r.ApprovalBasis...),
		ApprovalPolicySource:              r.ApprovalPolicySource,
		HighBlastRadius:                   r.HighBlastRadius,
		ApprovalEscalatedDueToBlastRadius: r.ApprovalEscalatedDueToBlastRadius,
		ExecutionSource:                   r.ExecutionSource,
	}
}
