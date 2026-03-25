package service

// operator_control_action.go — Operator-submitted control actions (canonical queue path).

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/control"
)

// QueueOperatorControlAction persists a new control action proposed by an operator and
// either enqueues it for execution or holds it pending_approval per policy.
// incidentID is optional; when set, the incident must exist.
func (a *App) QueueOperatorControlAction(actorID string, actionType, targetTransport, targetSegment, targetNode, reason string, confidence float64, incidentID string) (string, error) {
	if a == nil || a.DB == nil {
		return "", fmt.Errorf("service not available")
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return "", fmt.Errorf("operator identity is required to queue a control action")
	}
	actionType = strings.TrimSpace(actionType)
	if actionType == "" {
		return "", fmt.Errorf("action_type is required")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "", fmt.Errorf("reason is required")
	}
	incidentID = strings.TrimSpace(incidentID)
	if incidentID != "" {
		_, ok, err := a.DB.IncidentByID(incidentID)
		if err != nil {
			return "", fmt.Errorf("could not verify incident: %w", err)
		}
		if !ok {
			return "", fmt.Errorf("incident not found: %s", incidentID)
		}
	}

	now := time.Now().UTC()
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	id := "opctl-" + hex.EncodeToString(b)

	candidate := control.ControlAction{
		ID:              id,
		ActionType:      actionType,
		TargetTransport: strings.TrimSpace(targetTransport),
		TargetSegment:   strings.TrimSpace(targetSegment),
		TargetNode:      strings.TrimSpace(targetNode),
		Reason:          reason,
		Confidence:      confidence,
		CreatedAt:       now.Format(time.RFC3339),
		Mode:            a.Cfg.Control.Mode,
		LifecycleState:  control.LifecyclePending,
		ProposedBy:      actorID,
		SubmittedBy:     actorID,
		IncidentID:      incidentID,
	}

	reality, ok, err := a.DB.ControlActionRealityByType(actionType)
	if err == nil && ok {
		candidate.Reversible = reality.Reversible
		candidate.AdvisoryOnly = reality.AdvisoryOnly
		candidate.BlastRadiusClass = reality.BlastRadiusClass
		if candidate.BlastRadiusClass == "" {
			candidate.BlastRadiusClass = control.BlastRadiusUnknown
		}
	} else {
		candidate.BlastRadiusClass = control.BlastRadiusUnknown
	}

	execMode := a.resolveExecutionMode(candidate)
	candidate.ExecutionMode = execMode
	if execMode == control.ExecutionModeApprovalRequired && a.Cfg.Control.RequireSeparateApprover {
		candidate.RequiresSeparateApprover = true
	}

	if execMode == control.ExecutionModeApprovalRequired {
		approvalExpiry := ""
		if a.Cfg.Control.ApprovalTimeoutSeconds > 0 {
			approvalExpiry = now.Add(time.Duration(a.Cfg.Control.ApprovalTimeoutSeconds) * time.Second).Format(time.RFC3339)
		}
		candidate.ApprovalExpiresAt = approvalExpiry
		candidate.LifecycleState = control.LifecyclePendingApproval
		candidate.Result = control.ResultPendingApproval
		thHealth := a.transportHealthJSON(candidate.TargetTransport)
		candidate.EvidenceBundleID = a.captureEvidenceBundle(candidate, thHealth)
		if err := a.DB.UpsertControlAction(controlActionRecord(candidate)); err != nil {
			return "", fmt.Errorf("could not persist action: %w", err)
		}
		return id, nil
	}

	if blocked, blockReason, denialCode := a.isExecutionBlocked(candidate); blocked {
		candidate.ExecutedAt = now.Format(time.RFC3339)
		candidate.CompletedAt = candidate.ExecutedAt
		candidate.LifecycleState = control.LifecycleCompleted
		candidate.DenialCode = denialCode
		candidate.Result = func() string {
			if denialCode == control.DenialFreeze {
				return control.ResultDeniedByFreeze
			}
			return control.ResultDeniedByMaintenance
		}()
		candidate.ClosureState = func() string {
			if denialCode == control.DenialFreeze {
				return control.ClosureBlockedByFreeze
			}
			return control.ClosureBlockedByMaintenance
		}()
		candidate.OutcomeDetail = blockReason
		if err := a.DB.UpsertControlAction(controlActionRecord(candidate)); err != nil {
			return "", fmt.Errorf("could not persist blocked action: %w", err)
		}
		return id, nil
	}

	if err := a.DB.UpsertControlAction(controlActionRecord(candidate)); err != nil {
		return "", fmt.Errorf("could not persist action: %w", err)
	}
	select {
	case a.controlQueue <- candidate:
	default:
		nowStr := now.Format(time.RFC3339)
		candidate.ExecutedAt = nowStr
		candidate.CompletedAt = nowStr
		candidate.LifecycleState = control.LifecycleCompleted
		candidate.Result = control.ResultFailedTransient
		candidate.ClosureState = control.ClosureSuperseded
		candidate.OutcomeDetail = "control queue is full; operator action dropped to preserve bounded execution"
		_ = a.DB.UpsertControlAction(controlActionRecord(candidate))
		return "", fmt.Errorf("control queue full; action was not queued")
	}
	return id, nil
}
