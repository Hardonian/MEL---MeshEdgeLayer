// Package incidenttriage computes deterministic, inspectable triage signals for incident list/detail APIs.
// Signals are evidence-bounded; they do not imply routing, RF, or causal certainty.
package incidenttriage

import (
	"strings"

	"github.com/mel-project/mel/internal/models"
)

const (
	tierFollowUp       = 0
	tierControlGate    = 1
	tierEvidenceStress = 2
	tierRecurrence     = 3
	tierRoutine        = 4
)

// ComputeForIncident derives triage signals from the incident as serialized (after intelligence + action_visibility).
func ComputeForIncident(inc models.Incident) models.IncidentTriageSignals {
	out := models.IncidentTriageSignals{
		Tier: tierRoutine,
	}
	rs := strings.ToLower(strings.TrimSpace(inc.ReviewState))
	if rs == "follow_up_needed" || rs == "pending_review" || rs == "mitigated" {
		out.Tier = tierFollowUp
		appendCode(&out, "explicit_follow_up_review", "Review state on record calls for follow-up or review — not proof of live fault.",
			[]string{"incident.review_state:" + rs})
		return out
	}

	av := inc.ActionVisibility
	canReadLinked := av == nil || av.Kind != "visibility_limited"
	awaiting := 0
	if canReadLinked {
		for _, ca := range inc.LinkedControlActions {
			if strings.EqualFold(strings.TrimSpace(ca.LifecycleState), "pending_approval") {
				awaiting++
			}
		}
	}
	pendRefs := len(nonEmpty(inc.PendingActions))
	if awaiting > 0 {
		out.Tier = tierControlGate
		appendCode(&out, "governance_pending_approval",
			"Linked control actions await approval on this incident — approval ≠ execution; verify queue state.",
			[]string{"incident.linked_control_actions.lifecycle_state"})
	}
	if pendRefs > 0 && awaiting == 0 {
		out.Tier = minTier(out.Tier, tierControlGate)
		appendCode(&out, "pending_action_refs_on_record",
			"Incident record lists pending action id(s) without FK-linked rows in this response — match IDs in the control queue.",
			[]string{"incident.pending_actions"})
	}

	intel := inc.Intelligence
	if av != nil && (av.Kind == "visibility_limited" || av.Kind == "references_only" || av.Kind == "action_context_degraded") {
		out.Tier = minTier(out.Tier, tierEvidenceStress)
		if av.Kind == "visibility_limited" {
			appendCode(&out, "capability_limited_control_view",
				"Linked control rows omitted for this identity — absence here does not prove the queue is empty.",
				[]string{"incident.action_visibility"})
			out.UncertaintyNotes = append(out.UncertaintyNotes, "Triage uses pending_action refs and intelligence only when read_actions is denied.")
		}
		if av.Kind == "references_only" {
			appendCode(&out, "partial_action_linkage_payload",
				"Only action id references on the incident row — durable linkage requires incident_id on control actions.",
				[]string{"incident.pending_actions", "incident.recent_actions"})
		}
		if av.Kind == "action_context_degraded" {
			appendCode(&out, "action_outcome_trace_degraded",
				"Action outcome memory or snapshot trace is incomplete — reuse prior patterns with extra verification.",
				[]string{"incident.intelligence.action_outcome_trace"})
		}
	}

	if intel != nil {
		if intel.EvidenceStrength == "sparse" || intel.Degraded {
			out.Tier = minTier(out.Tier, tierEvidenceStress)
			appendCode(&out, "sparse_or_degraded_intel",
				"Intelligence is sparse or degraded — conclusions stay bounded; add replay/topology/control context.",
				[]string{"incident.intelligence.evidence_strength", "incident.intelligence.degraded"})
		}
		if governanceFriction(intel) {
			out.Tier = minTier(out.Tier, tierEvidenceStress)
			appendCode(&out, "governance_friction_memory",
				"Historical governance pattern for action types on this incident shows repeated rejection or high blast — stall risk, not a verdict.",
				[]string{"incident.intelligence.governance_memory"})
		}
		if mitigationDidNotHold(intel) {
			out.Tier = minTier(out.Tier, tierEvidenceStress)
			appendCode(&out, "mitigation_durability_weak_in_family",
				"Prior incidents in this signature family often showed deterioration or mixed signals after similar actions — association only, not prediction.",
				[]string{"incident.intelligence.action_outcome_memory", "incident.signature_family_resolved_history"})
		}
		if intel.SignatureMatchCount > 1 {
			out.Tier = minTier(out.Tier, tierRecurrence)
			appendCode(&out, "recurring_signature_bucket",
				"Signature match count >1 on this instance — recurring operational bucket, not proof of repeating root cause.",
				[]string{"incident.intelligence.signature_match_count"})
		}
		if strings.TrimSpace(inc.ReopenedFromIncidentID) != "" {
			out.Tier = minTier(out.Tier, tierRecurrence)
			appendCode(&out, "reopened_incident",
				"Incident was reopened from a prior case — compare replay and outcomes before reusing the same control pattern.",
				[]string{"incident.reopened_from_incident_id"})
		}
	}

	if out.Tier == tierRoutine && len(out.Codes) == 0 {
		appendCode(&out, "open_routine",
			"Open incident without elevated deterministic triage flags — still verify against replay and live queue.",
			[]string{"incident.state"})
	}

	return out
}

func governanceFriction(intel *models.IncidentIntelligence) bool {
	for _, g := range intel.GovernanceMemory {
		if g.RejectedCount >= 2 {
			return true
		}
		if g.HighBlastCount >= 2 {
			return true
		}
		if g.LinkedActionCount >= 3 && g.ApprovedOrPassedCount == 0 && g.RejectedCount > 0 {
			return true
		}
	}
	return false
}

func mitigationDidNotHold(intel *models.IncidentIntelligence) bool {
	for _, m := range intel.ActionOutcomeMemory {
		if m.OutcomeFraming == "deterioration_observed" && m.SampleSize >= 2 {
			return true
		}
		if m.OutcomeFraming == "mixed_historical_evidence" && m.SampleSize >= 3 {
			return true
		}
	}
	if intel.SignatureFamilyResolvedHistory != nil {
		h := intel.SignatureFamilyResolvedHistory
		if h.ResolvedPeerCount >= 2 && h.ReopenedPeerCount >= 1 {
			return true
		}
	}
	return false
}

func appendCode(out *models.IncidentTriageSignals, code, rationale string, refs []string) {
	for _, c := range out.Codes {
		if c == code {
			return
		}
	}
	out.Codes = append(out.Codes, code)
	out.RationaleLines = append(out.RationaleLines, rationale)
	out.EvidenceRefs = append(out.EvidenceRefs, refs...)
}

func minTier(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func nonEmpty(ss []string) []string {
	var out []string
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}
