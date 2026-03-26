package service

import (
	"strings"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/models"
)

// ApprovalPolicyEvaluation is the canonical structured view of approval gating and
// operator authorization for a control action. All approval entry points derive
// denial reasons from this shape; do not duplicate policy strings elsewhere.
type ApprovalPolicyEvaluation struct {
	RequiresApproval bool `json:"requires_approval"`

	ApprovalMode       string `json:"approval_mode"`        // e.g. single_approver
	RequiredApprovals  int    `json:"required_approvals"`   // always 1 today
	CollectedApprovals int    `json:"collected_approvals"`  // 0 until approved, then 1

	ApprovalBasis        []string `json:"approval_basis"`
	ApprovalPolicySource string   `json:"approval_policy_source"` // mel_config

	HighBlastRadius                   bool   `json:"high_blast_radius"`
	BlastRadiusClassification         string `json:"blast_radius_classification"`
	ApprovalEscalatedDueToBlastRadius bool   `json:"approval_escalated_due_to_blast_radius"`

	SubmitterDisqualifiedFromApproval bool   `json:"submitter_disqualified_from_approval"`
	ApproverAllowed                   bool   `json:"approver_allowed"`
	ApproverDenialReason              string `json:"approver_denial_reason,omitempty"`

	ApprovedDoesNotImplyExecution    bool `json:"approved_does_not_imply_execution"`
	BacklogExecutionRequiresExecutor bool `json:"backlog_execution_requires_executor"`
}

const (
	approvalPolicySourceConfig = "mel_config.control"
	approvalModeSingleApprover = "single_approver"
)

// basisActionType is a stable token for API/CLI when action type list forces approval.
const basisPrefixRequireApprovalActionTypes = "require_approval_for_action_types"

// basisHighBlastRadius is a stable token when high-blast policy forces approval.
const basisRequireApprovalHighBlastRadius = "require_approval_for_high_blast_radius"

func blastRadiusHighForApprovalPolicy(class string) bool {
	switch strings.TrimSpace(class) {
	case control.BlastRadiusMesh, control.BlastRadiusGlobal:
		return true
	default:
		return false
	}
}

func actionTypeRequiresApproval(cfg config.ControlConfig, actionType string) bool {
	at := strings.TrimSpace(actionType)
	for _, t := range cfg.RequireApprovalForActionTypes {
		if strings.EqualFold(strings.TrimSpace(t), at) {
			return true
		}
	}
	return false
}

// EvaluateApprovalPolicyForRecord computes policy truth for a persisted row. When
// actorID is non-empty, SoD / approver eligibility is evaluated for approve/reject.
func (a *App) EvaluateApprovalPolicyForRecord(rec db.ControlActionRecord, actorID string) ApprovalPolicyEvaluation {
	ev := ApprovalPolicyEvaluation{
		ApprovalMode:                   approvalModeSingleApprover,
		RequiredApprovals:              1,
		BlastRadiusClassification:      strings.TrimSpace(rec.BlastRadiusClass),
		ApprovedDoesNotImplyExecution:  true,
		BacklogExecutionRequiresExecutor: true,
		ApprovalPolicySource:           strings.TrimSpace(rec.ApprovalPolicySource),
		ApproverAllowed:                true,
	}
	if ev.BlastRadiusClassification == "" {
		ev.BlastRadiusClassification = control.BlastRadiusUnknown
	}
	if ev.ApprovalPolicySource == "" {
		ev.ApprovalPolicySource = approvalPolicySourceConfig
	}

	cfg := a.Cfg.Control
	typeMatch := actionTypeRequiresApproval(cfg, rec.ActionType)
	high := blastRadiusHighForApprovalPolicy(rec.BlastRadiusClass)
	ev.HighBlastRadius = high

	if len(rec.ApprovalBasis) > 0 {
		ev.ApprovalBasis = append(ev.ApprovalBasis, rec.ApprovalBasis...)
	} else {
		if cfg.RequireApprovalForHighBlastRadius && high {
			ev.ApprovalBasis = append(ev.ApprovalBasis, basisRequireApprovalHighBlastRadius)
		}
		if typeMatch {
			ev.ApprovalBasis = append(ev.ApprovalBasis, basisPrefixRequireApprovalActionTypes+":"+rec.ActionType)
		}
	}

	ev.RequiresApproval = rec.ExecutionMode == control.ExecutionModeApprovalRequired
	if rec.HighBlastRadius {
		ev.HighBlastRadius = true
	}
	if rec.ApprovalEscalatedDueToBlastRadius {
		ev.ApprovalEscalatedDueToBlastRadius = true
	} else {
		needsByConfig := typeMatch || (cfg.RequireApprovalForHighBlastRadius && high)
		ev.ApprovalEscalatedDueToBlastRadius = cfg.RequireApprovalForHighBlastRadius && high && !typeMatch && needsByConfig
	}

	if rec.RequiredApprovals > 0 {
		ev.RequiredApprovals = rec.RequiredApprovals
	}
	if strings.TrimSpace(rec.ApprovalMode) != "" {
		ev.ApprovalMode = rec.ApprovalMode
	}

	switch {
	case rec.LifecycleState == control.LifecyclePendingApproval:
		ev.CollectedApprovals = 0
	case rec.ApprovedBy != "" && rec.ExecutionMode == control.ExecutionModeApprovalRequired:
		ev.CollectedApprovals = 1
	default:
		ev.CollectedApprovals = rec.CollectedApprovals
	}

	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return ev
	}

	if !ev.RequiresApproval || rec.LifecycleState != control.LifecyclePendingApproval {
		return ev
	}

	if approvalSodWouldBlock(rec, actorID) {
		ev.SubmitterDisqualifiedFromApproval = true
		ev.ApproverAllowed = false
		ev.ApproverDenialReason = "separation_of_duties_submitter_may_not_approve"
	}

	return ev
}

// ApplyApprovalGateMetadata sets durable fields on a record before first insert
// when an action is held for approval (pending_approval).
func applyApprovalGateMetadata(cfg config.ControlConfig, rec *db.ControlActionRecord) {
	if rec == nil {
		return
	}
	typeMatch := actionTypeRequiresApproval(cfg, rec.ActionType)
	high := blastRadiusHighForApprovalPolicy(rec.BlastRadiusClass)
	rec.ApprovalMode = approvalModeSingleApprover
	rec.RequiredApprovals = 1
	rec.CollectedApprovals = 0
	rec.ApprovalPolicySource = approvalPolicySourceConfig
	rec.HighBlastRadius = high
	rec.ApprovalEscalatedDueToBlastRadius = cfg.RequireApprovalForHighBlastRadius && high && !typeMatch
	var basis []string
	if typeMatch {
		basis = append(basis, basisPrefixRequireApprovalActionTypes+":"+rec.ActionType)
	}
	if cfg.RequireApprovalForHighBlastRadius && high {
		basis = append(basis, basisRequireApprovalHighBlastRadius)
	}
	rec.ApprovalBasis = basis
}

func approvalPolicyToDTO(ev ApprovalPolicyEvaluation) models.ApprovalPolicyDTO {
	return models.ApprovalPolicyDTO{
		RequiresApproval:                   ev.RequiresApproval,
		ApprovalMode:                       ev.ApprovalMode,
		RequiredApprovals:                  ev.RequiredApprovals,
		CollectedApprovals:                 ev.CollectedApprovals,
		ApprovalBasis:                      append([]string(nil), ev.ApprovalBasis...),
		ApprovalPolicySource:               ev.ApprovalPolicySource,
		HighBlastRadius:                    ev.HighBlastRadius,
		BlastRadiusClassification:          ev.BlastRadiusClassification,
		ApprovalEscalatedDueToBlastRadius:  ev.ApprovalEscalatedDueToBlastRadius,
		SubmitterDisqualifiedFromApproval:  ev.SubmitterDisqualifiedFromApproval,
		ApproverAllowed:                    ev.ApproverAllowed,
		ApproverDenialReason:               ev.ApproverDenialReason,
		ApprovedDoesNotImplyExecution:      ev.ApprovedDoesNotImplyExecution,
		BacklogExecutionRequiresExecutor:   ev.BacklogExecutionRequiresExecutor,
	}
}
