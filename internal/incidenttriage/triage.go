// Package incidenttriage computes deterministic, inspectable triage signals for incident list/detail APIs.
// Signals are evidence-bounded; they do not imply routing, RF, or causal certainty.
package incidenttriage

import (
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"time"

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
		fillQueueOrdering(&out, inc)
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
	if rs := inc.ReplaySummary; rs != nil {
		if strings.TrimSpace(rs.AttentionReason) != "" {
			line := strings.TrimSpace(rs.Summary)
			if line == "" {
				line = "Replay posture needs bounded review; open replay/detail for exact semantics and caveats."
			}
			appendCode(&out, "replay_"+strings.TrimSpace(rs.AttentionReason),
				line,
				[]string{"incident.replay_summary.semantic", "incident.replay_summary.history_pattern", "incident.replay_summary.comparability"})
		}
		if rs.NeedsAttention {
			out.Tier = minTier(out.Tier, tierEvidenceStress)
		}
		if strings.TrimSpace(rs.Comparability) == "not_comparable" || strings.TrimSpace(rs.Comparability) == "unavailable" {
			out.UncertaintyNotes = append(out.UncertaintyNotes,
				"Replay comparison posture is "+strings.TrimSpace(rs.Comparability)+" — avoid trend claims without full replay evidence.")
		}
	}
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

	if intel != nil && intel.MeshRoutingCompanion != nil {
		mc := intel.MeshRoutingCompanion
		if mc.Applicable {
			var meshNotes []string
			if !mc.TopologyEnabled {
				meshNotes = append(meshNotes, "Mesh routing companion: topology model disabled — ingest-graph pressure context may be incomplete.")
			}
			if !mc.TransportConnected {
				meshNotes = append(meshNotes, "Mesh routing companion: transport not connected — routing-pressure lines are not live path proof.")
			}
			if mc.EvidenceModel == "" || mc.AssessmentComputedAt == "" {
				meshNotes = append(meshNotes, "Mesh routing companion: assessment metadata sparse — treat companion lines as bounded diagnostics only.")
			}
			if mc.SuspectedRelayHotspot || mc.WeakOnwardPropagationSuspected || mc.HopBudgetStressSuspected {
				out.Tier = minTier(out.Tier, tierEvidenceStress)
				appendCode(&out, "mesh_routing_pressure_companion",
					"Ingest-graph routing-pressure flags are raised — bounded diagnostics adjacent to replay/topology; not RF delivery or path certainty.",
					[]string{"incident.intelligence.mesh_routing_companion"})
			}
			out.UncertaintyNotes = append(out.UncertaintyNotes, meshNotes...)
		}
	}

	if out.Tier == tierRoutine && len(out.Codes) == 0 {
		appendCode(&out, "open_routine",
			"Open incident without elevated deterministic triage flags — still verify against replay and live queue.",
			[]string{"incident.state"})
	}

	fillQueueOrdering(&out, inc)
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
	if intel.MitigationDurabilityMemory != nil {
		switch intel.MitigationDurabilityMemory.Posture {
		case "reopened_after_resolution_in_family", "deterioration_or_mixed_in_outcome_memory", "family_peer_scan_bounded":
			return true
		}
	}
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

// fillQueueOrdering attaches explicit sort keys so clients need not re-derive tier semantics for open queues.
func fillQueueOrdering(out *models.IncidentTriageSignals, inc models.Incident) {
	if out == nil {
		return
	}
	out.QueueOrderingContract = "open_incident_workbench_v2"
	out.QueueOrderingContractVersion = "2"
	out.QueueSortPrimary = out.Tier
	out.QueueSortSecondary = "updated_at_desc"
	secNs, validity := parseUpdatedAtNanos(inc.UpdatedAt)
	out.QueueSortSecondaryNumeric = secNs
	out.QueueSortSecondaryValidity = validity
	tie := stableIncidentIDHash(inc.ID)
	out.QueueSortTieBreak = "incident_id_fnv1a64_lex_tiebreak"
	out.QueueSortTieBreakNumeric = tie
	recencyInverted := recencyInvertedRank(secNs, validity)
	// Lexicographic ascending sort on this tuple matches: lower tier first, then more-recent updated_at first (smaller inverted rank), then stable id tie-break.
	out.QueueSortTuple = []int64{int64(out.Tier), recencyInverted, tie}
	out.QueueSortKeyLex = queueSortKeyLex(out.Tier, recencyInverted, tie)
	out.OrderingRationale = "open_incident_workbench_v2: prefer ascending queue_sort_key_lex (JSON-safe). Tuple: [tier, recency_inverted_ns, tie_break_hash] — recency_inverted_ns = MaxInt64−updated_at_ns when valid (more recent → smaller rank); missing/invalid timestamps use MaxInt64 (sort after known recency). queue_sort_secondary remains a human hint (updated_at_desc)."
	var ev []string
	ev = append(ev, out.EvidenceRefs...)
	if inc.UpdatedAt != "" {
		ev = append(ev, "incident.updated_at")
	}
	if strings.TrimSpace(inc.ID) != "" {
		ev = append(ev, "incident.id")
	}
	out.OrderingEvidenceRefs = ev
	out.OrderingUncertainty = "Tier is deterministic from this API payload. Recency uses incident.updated_at only; missing or invalid timestamps collapse secondary ordering to 0 — not a team queue, SLA, or hidden score."
	if validity == "valid_rfc3339" {
		out.QueueOrderingPosture = "canonical_v2"
	} else {
		out.QueueOrderingPosture = "degraded_partial_recency"
		out.QueueOrderingDegradedReasons = append(out.QueueOrderingDegradedReasons, "queue_sort_secondary_"+validity)
		out.OrderingUncertainty += " Ordering posture is degraded_partial_recency: recency/tie-break used safe fallbacks — do not assume strict time ordering vs other incidents."
	}
}

func parseUpdatedAtNanos(updatedAt string) (ns int64, validity string) {
	s := strings.TrimSpace(updatedAt)
	if s == "" {
		return 0, "missing"
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
	}
	if err != nil || t.IsZero() {
		return 0, "invalid_timestamp"
	}
	return t.UTC().UnixNano(), "valid_rfc3339"
}

func stableIncidentIDHash(id string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(strings.TrimSpace(id)))
	v := int64(h.Sum64() & ((uint64(1) << 63) - 1))
	if v == 0 {
		return 1
	}
	return v
}

func recencyInvertedRank(secNs int64, validity string) int64 {
	if validity != "valid_rfc3339" {
		return math.MaxInt64
	}
	if secNs < 0 {
		return math.MaxInt64
	}
	return math.MaxInt64 - secNs
}

func queueSortKeyLex(tier int, recencyInverted, tie int64) string {
	return fmt.Sprintf("%d.%020d.%016x", tier, recencyInverted, uint64(tie))
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
