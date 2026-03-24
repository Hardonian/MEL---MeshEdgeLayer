// Package operatorlang holds user-facing, operator-oriented labels derived from
// canonical backend fields. It avoids inventing protocol-specific claims.
package operatorlang

import (
	"strings"

	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
)

// IncidentIDFromMetadata returns a linked incident id when present in action metadata.
func IncidentIDFromMetadata(meta map[string]any) string {
	if meta == nil {
		return ""
	}
	for _, k := range []string{"incident_id", "mel_incident_id", "linked_incident_id"} {
		if v, ok := meta[k]; ok {
			if s, ok2 := v.(string); ok2 && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

// ActionOperatorLabels derives short operator-facing strings from a control action row.
func ActionOperatorLabels(r db.ControlActionRecord) map[string]any {
	proposer := strings.TrimSpace(r.ProposedBy)
	if proposer == "" {
		proposer = "system"
	}
	incidentID := IncidentIDFromMetadata(r.Metadata)

	targetSummary := TargetSummary(r.ActionType, r.TargetTransport, r.TargetSegment, r.TargetNode)
	queueStatus, approvalStatus, executionStatus := lifecycleOperatorLabels(r)

	out := map[string]any{
		"target_summary":         targetSummary,
		"queue_status":           queueStatus,
		"approval_status":        approvalStatus,
		"execution_status":       executionStatus,
		"mesh_side_effect":       MeshSideEffectHint(r.ActionType, r.TargetTransport, r.TargetNode),
		"second_operator_note":   SecondOperatorNote(r.ExecutionMode, r.LifecycleState, proposer),
		"sod_blocks_self":        SodBlocksSelfApproval(r.ExecutionMode, r.LifecycleState, proposer),
		"break_glass_in_history": BreakGlassUsed(r.Metadata),
	}
	if incidentID != "" {
		out["linked_incident_id"] = incidentID
	}
	return out
}

// TargetSummary is a single-line description of what the action affects.
func TargetSummary(actionType, targetTransport, targetSegment, targetNode string) string {
	tt := strings.TrimSpace(targetTransport)
	tn := strings.TrimSpace(targetNode)
	ts := strings.TrimSpace(targetSegment)
	at := strings.TrimSpace(actionType)

	parts := []string{}
	if tn != "" {
		parts = append(parts, "node "+tn)
	}
	if tt != "" {
		lower := strings.ToLower(tt)
		switch {
		case strings.Contains(lower, "mqtt"):
			parts = append(parts, "MQTT bridge / gateway path (\""+tt+"\")")
		default:
			parts = append(parts, "link / transport (\""+tt+"\")")
		}
	}
	if ts != "" && ts != tt {
		parts = append(parts, "segment "+ts)
	}
	if len(parts) == 0 {
		if at != "" {
			return "Platform / MEL (" + at + ")"
		}
		return "Target not specified in row"
	}
	s := strings.Join(parts, " · ")
	if at != "" {
		return s + " — " + at
	}
	return s
}

// MeshSideEffectHint is true when the action type plausibly affects mesh/radio ingress/egress
// (transport or node target present). It is conservative and non-protocol-specific.
func MeshSideEffectHint(actionType, targetTransport, targetNode string) bool {
	if strings.TrimSpace(targetNode) != "" {
		return true
	}
	if strings.TrimSpace(targetTransport) == "" {
		return false
	}
	at := strings.TrimSpace(actionType)
	switch at {
	case control.ActionRestartTransport, control.ActionResubscribeTransport,
		control.ActionBackoffIncrease, control.ActionBackoffReset,
		control.ActionTemporarilyDeprioritize, control.ActionTriggerHealthRecheck:
		return true
	default:
		return true // transport-scoped actions are treated as mesh-adjacent
	}
}

// SecondOperatorNote explains approval expectations in operator language.
func SecondOperatorNote(executionMode, lifecycleState, proposedBy string) string {
	if executionMode != control.ExecutionModeApprovalRequired {
		return ""
	}
	if lifecycleState != control.LifecyclePendingApproval {
		return ""
	}
	pb := strings.TrimSpace(proposedBy)
	if pb == "" || strings.EqualFold(pb, "system") {
		return "Needs operator approval before it runs on targets."
	}
	return "Needs approval from a different operator than " + pb + " (separation of duties)."
}

// SodBlocksSelfApproval is true when policy would block the proposer from approving.
func SodBlocksSelfApproval(executionMode, lifecycleState, proposedBy string) bool {
	if executionMode != control.ExecutionModeApprovalRequired {
		return false
	}
	if lifecycleState != control.LifecyclePendingApproval {
		return false
	}
	pb := strings.TrimSpace(proposedBy)
	if pb == "" || strings.EqualFold(pb, "system") {
		return false
	}
	return true
}

// BreakGlassUsed reports whether durable metadata records an emergency approval path.
func BreakGlassUsed(meta map[string]any) bool {
	if meta == nil {
		return false
	}
	v, ok := meta["mel_break_glass_approval"]
	if !ok {
		v, ok = meta["mel_break_glass_reject"]
	}
	if !ok {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return strings.EqualFold(x, "true") || x == "1"
	default:
		return false
	}
}

func lifecycleOperatorLabels(r db.ControlActionRecord) (queue, approval, execution string) {
	ls := r.LifecycleState
	res := r.Result
	em := r.ExecutionMode

	// Rejected / expired terminal paths
	if ls == control.LifecycleCompleted {
		switch res {
		case control.ResultRejected:
			return "Closed", "Rejected", "Not run"
		case control.ResultApprovalExpired:
			return "Closed", "Approval expired", "Not run"
		case control.ResultDeniedByPolicy, control.ResultDeniedByFreeze, control.ResultDeniedByMaintenance:
			return "Closed", "N/A", "Blocked before run"
		case control.ResultExecutedSuccessfully, control.ResultExecutedNoop:
			return "Done", "N/A", "Completed"
		case control.ResultFailedTerminal, control.ResultFailedTransient:
			return "Done", "N/A", "Failed"
		case control.ResultApproved:
			// Approved but executor has not finished — distinguish using execution timestamps
			if strings.TrimSpace(r.ExecutedAt) == "" {
				return "Executor queue", "Approved", "Waiting for executor"
			}
			if strings.TrimSpace(r.CompletedAt) == "" {
				return "Running", "Approved", "Executing on target"
			}
			return "Done", "Approved", "Finished"
		default:
			return "Closed", "—", humanResult(res)
		}
	}

	switch ls {
	case control.LifecyclePendingApproval:
		return "Approver inbox", "Awaiting approver", "Not started"
	case control.LifecyclePending:
		if em == control.ExecutionModeApprovalRequired && res == control.ResultApproved {
			if strings.TrimSpace(r.ExecutedAt) == "" {
				return "Executor queue", "Approved", "Waiting for executor"
			}
		}
		if strings.TrimSpace(r.ExecutedAt) == "" {
			return "Executor queue", "—", "Queued"
		}
		return "Running", "—", "Executing on target"
	case control.LifecycleRunning:
		return "Running", "—", "Executing on target"
	case control.LifecycleRecovered:
		return "Recovered", "—", humanResult(res)
	default:
		return ls, "—", humanResult(res)
	}
}

func humanResult(res string) string {
	if res == "" {
		return "Unknown"
	}
	return strings.ReplaceAll(res, "_", " ")
}
