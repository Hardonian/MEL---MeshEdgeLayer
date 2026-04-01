// Package actionvisibility computes deterministic incident × control visibility posture for API responses.
// It encodes only what the backend can observe — no implied team workflow or hidden linkage certainty.
package actionvisibility

import (
	"strconv"
	"strings"

	"github.com/mel-project/mel/internal/models"
)

// FromIncident derives posture from the incident payload as the API will serialize it
// (linked rows already stripped when canReadLinked is false). canReadLinked reflects read_actions capability.
func FromIncident(inc models.Incident, canReadLinked bool) models.IncidentActionVisibilityPosture {
	linked := inc.LinkedControlActions
	pendingRefs := nonEmptyStrings(inc.PendingActions)
	recentIDs := nonEmptyStrings(inc.RecentActions)
	intel := inc.Intelligence

	awaiting, inFlight := lifecycleCounts(linked)

	hasPriorAttempts := hasHistoricalActionSignals(intel)
	isPartial := false
	reason := ""
	summary := ""
	kind := ""
	suggestQueue := false

	if !canReadLinked {
		kind = "visibility_limited"
		reason = "capability_limited"
		isPartial = true
		suggestQueue = true
		summary = "Linked control rows are omitted for this identity (read_actions not granted). Absence of linked rows in this response does not prove the control queue is empty."
		return models.IncidentActionVisibilityPosture{
			Kind:                     kind,
			Reason:                   reason,
			Summary:                  summary,
			SuggestControlQueue:      suggestQueue,
			HasMaterialPriorAttempts: hasPriorAttempts,
			HasPendingRelatedWork:    len(pendingRefs) > 0 || len(recentIDs) > 0,
			IsPartial:                isPartial,
			LinkedRowCount:           0,
			PendingRefCount:          len(pendingRefs),
			RecentActionRefCount:     len(recentIDs),
		}
	}

	if len(linked) > 0 {
		kind = "linked_observed"
		reason = ""
		parts := []string{formatLinkedSummary(len(linked), awaiting, inFlight)}
		summary = strings.Join(parts, " ")
		summary += " Approval, dispatch, execution, and audit are distinct; verify lifecycle on each row."
		suggestQueue = awaiting > 0 || inFlight > 0
		return models.IncidentActionVisibilityPosture{
			Kind:                     kind,
			Reason:                   reason,
			Summary:                  summary,
			SuggestControlQueue:      suggestQueue,
			HasMaterialPriorAttempts: hasPriorAttempts || len(linked) > 0,
			HasPendingRelatedWork:    awaiting > 0 || inFlight > 0 || len(pendingRefs) > 0,
			IsPartial:                actionTraceDegraded(intel),
			LinkedRowCount:           len(linked),
			PendingRefCount:          len(pendingRefs),
			RecentActionRefCount:     len(recentIDs),
		}
	}

	if len(pendingRefs) > 0 || len(recentIDs) > 0 {
		kind = "references_only"
		reason = "partial_payload_only"
		var bits []string
		if len(pendingRefs) > 0 {
			bits = append(bits, formatRefs(len(pendingRefs), "pending action ID", "pending action IDs"))
		}
		if len(recentIDs) > 0 {
			bits = append(bits, formatRefs(len(recentIDs), "recent action ID", "recent action IDs"))
		}
		summary = strings.Join(bits, " · ") + " on the incident record — no FK-linked control rows in this response. Match IDs in the control queue; durable linkage requires incident_id on the action."
		suggestQueue = true
		isPartial = true
		return models.IncidentActionVisibilityPosture{
			Kind:                     kind,
			Reason:                   reason,
			Summary:                  summary,
			SuggestControlQueue:      suggestQueue,
			HasMaterialPriorAttempts: hasPriorAttempts,
			HasPendingRelatedWork:    len(pendingRefs) > 0,
			IsPartial:                isPartial,
			LinkedRowCount:           0,
			PendingRefCount:          len(pendingRefs),
			RecentActionRefCount:     len(recentIDs),
		}
	}

	if actionTraceDegraded(intel) {
		kind = "action_context_degraded"
		reason = traceReason(intel)
		if reason == "" {
			reason = "unknown"
		}
		summary = "Action outcome snapshot/trace for this incident is partial or unavailable — do not read “no linked actions” as proof nothing ran; verify the control queue and replay."
		suggestQueue = true
		isPartial = true
		return models.IncidentActionVisibilityPosture{
			Kind:                     kind,
			Reason:                   reason,
			Summary:                  summary,
			SuggestControlQueue:      suggestQueue,
			HasMaterialPriorAttempts: hasPriorAttempts,
			HasPendingRelatedWork:    false,
			IsPartial:                true,
			LinkedRowCount:           0,
			PendingRefCount:          0,
			RecentActionRefCount:     0,
		}
	}

	if hasPriorAttempts {
		kind = "no_linked_historical_signals"
		reason = ""
		summary = "No FK-linked control rows in this response; intelligence still carries historical action or governance signals — association only; confirm live queue state before acting."
		suggestQueue = true
		isPartial = intel != nil && (intel.EvidenceStrength == "sparse" || intel.Degraded)
		return models.IncidentActionVisibilityPosture{
			Kind:                     kind,
			Reason:                   reason,
			Summary:                  summary,
			SuggestControlQueue:      suggestQueue,
			HasMaterialPriorAttempts: true,
			HasPendingRelatedWork:    false,
			IsPartial:                isPartial,
			LinkedRowCount:           0,
			PendingRefCount:          0,
			RecentActionRefCount:     0,
		}
	}

	kind = "no_linked_observed"
	reason = ""
	summary = "No linked control rows, action references, or degraded trace flags in this view — if you still expect work in flight, check the control queue."
	suggestQueue = false
	return models.IncidentActionVisibilityPosture{
		Kind:                     kind,
		Reason:                   reason,
		Summary:                  summary,
		SuggestControlQueue:      suggestQueue,
		HasMaterialPriorAttempts: false,
		HasPendingRelatedWork:    false,
		IsPartial:                false,
		LinkedRowCount:           0,
		PendingRefCount:          0,
		RecentActionRefCount:     0,
	}
}

func nonEmptyStrings(a []string) []string {
	if len(a) == 0 {
		return nil
	}
	out := make([]string, 0, len(a))
	for _, s := range a {
		if strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

func lifecycleCounts(linked []models.ActionRecord) (awaitingApproval, inFlight int) {
	for _, a := range linked {
		ls := strings.ToLower(strings.TrimSpace(a.LifecycleState))
		switch ls {
		case "pending_approval":
			awaitingApproval++
		case "pending", "running":
			inFlight++
		}
	}
	return awaitingApproval, inFlight
}

func actionTraceDegraded(intel *models.IncidentIntelligence) bool {
	if intel == nil {
		return false
	}
	t := intel.ActionOutcomeTrace
	if t == nil {
		return false
	}
	if t.SnapshotRetrievalStatus == "error" || t.SnapshotRetrievalStatus == "unavailable" {
		return true
	}
	if t.SnapshotWriteFailures > 0 {
		return true
	}
	if t.Completeness == "unavailable" {
		return true
	}
	return false
}

func traceReason(intel *models.IncidentIntelligence) string {
	if intel == nil || intel.ActionOutcomeTrace == nil {
		return "trace_missing"
	}
	t := intel.ActionOutcomeTrace
	if t.SnapshotWriteFailures > 0 {
		return "partial_payload_only"
	}
	if t.SnapshotRetrievalStatus == "error" {
		return "runtime_load_failed"
	}
	if t.SnapshotRetrievalStatus == "unavailable" {
		return "trace_missing"
	}
	if t.Completeness == "unavailable" {
		return "trace_missing"
	}
	return "unknown"
}

func hasHistoricalActionSignals(intel *models.IncidentIntelligence) bool {
	if intel == nil {
		return false
	}
	if len(intel.ActionOutcomeMemory) > 0 {
		return true
	}
	if len(intel.GovernanceMemory) > 0 {
		return true
	}
	if len(intel.HistoricallyUsedActions) > 0 {
		return true
	}
	return false
}

func formatLinkedSummary(n, awaiting, inFlight int) string {
	var parts []string
	parts = append(parts, plural(n, "FK-linked control row", "FK-linked control rows"))
	if awaiting > 0 {
		if awaiting == 1 {
			parts = append(parts, "1 awaiting approval")
		} else {
			parts = append(parts, formatInt(awaiting)+" awaiting approval")
		}
	}
	if inFlight > 0 {
		if inFlight == 1 {
			parts = append(parts, "1 queued or in flight")
		} else {
			parts = append(parts, formatInt(inFlight)+" queued or in flight")
		}
	}
	return strings.Join(parts, " · ") + "."
}

func formatRefs(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	return formatInt(n) + " " + plural
}

func plural(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	return formatInt(n) + " " + plural
}

func formatInt(n int) string {
	return strconv.Itoa(n)
}
