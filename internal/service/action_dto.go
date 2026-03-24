package service

import (
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/models"
)

// ActionRecordFromDB maps a persisted control action row to the API DTO.
func ActionRecordFromDB(r db.ControlActionRecord) models.ActionRecord {
	return models.ActionRecord{
		ID:                       r.ID,
		TransportName:            r.TargetTransport,
		ActionType:               r.ActionType,
		LifecycleState:           r.LifecycleState,
		Result:                   r.Result,
		Reason:                   r.Reason,
		OutcomeDetail:            r.OutcomeDetail,
		CreatedAt:                r.CreatedAt,
		ExecutedAt:               r.ExecutedAt,
		CompletedAt:              r.CompletedAt,
		ExpiresAt:                r.ExpiresAt,
		TriggerEvidence:          append([]string(nil), r.TriggerEvidence...),
		Details:                  r.Metadata,
		ExecutionMode:            r.ExecutionMode,
		ProposedBy:               r.ProposedBy,
		ApprovedBy:               r.ApprovedBy,
		ApprovedAt:               r.ApprovedAt,
		RejectedBy:               r.RejectedBy,
		RejectedAt:               r.RejectedAt,
		ApprovalNote:             r.ApprovalNote,
		ApprovalExpiresAt:        r.ApprovalExpiresAt,
		BlastRadiusClass:         r.BlastRadiusClass,
		EvidenceBundleID:         r.EvidenceBundleID,
		SubmittedBy:              r.SubmittedBy,
		RequiresSeparateApprover: r.RequiresSeparateApprover,
		IncidentID:               r.IncidentID,
		ExecutionStartedAt:       r.ExecutionStartedAt,
		SodBypass:                r.SodBypass,
		SodBypassActor:           r.SodBypassActor,
		SodBypassReason:          r.SodBypassReason,
		TargetSegment:            r.TargetSegment,
		TargetNode:               r.TargetNode,
	}
}
