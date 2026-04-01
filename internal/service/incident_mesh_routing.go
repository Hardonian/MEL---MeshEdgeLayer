package service

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/models"
)

// meshRoutingCompanionForIncident attaches bounded mesh routing-pressure context when the incident
// plausibly relates to observed topology (pattern association, not RF/path proof).
func (a *App) meshRoutingCompanionForIncident(inc models.Incident, intel *models.IncidentIntelligence) *models.MeshRoutingIntelCompanion {
	if a == nil {
		return nil
	}
	out := &models.MeshRoutingIntelCompanion{
		Applicable: false,
		Reason:     "service_unavailable",
	}
	if !a.Cfg.Topology.Enabled {
		out.Reason = "topology_disabled_in_config"
		return out
	}
	a.meshIntelMu.RLock()
	snap := a.meshIntelLatest
	hasSnap := a.meshIntelHas
	a.meshIntelMu.RUnlock()
	if !hasSnap {
		out.Reason = "no_mesh_intelligence_snapshot"
		return out
	}
	if !meshRoutingCompanionIncidentRelevant(inc, intel) {
		out.Reason = "incident_not_mesh_topology_scoped"
		return out
	}
	out.Applicable = true
	out.Reason = ""
	out.TopologyEnabled = snap.TopologyEnabled
	out.TransportConnected = snap.MessageSignals.TransportConnected
	out.AssessmentComputedAt = snap.ComputedAt
	out.GraphHash = snap.GraphHash
	out.EvidenceModel = snap.EvidenceModel
	out.MessageWindowDescription = snap.MessageSignals.WindowDescription

	rp := snap.RoutingPressure
	out.RoutingSummaryLines = append([]string(nil), rp.SummaryLines...)

	dup := rp.DuplicateForwardPressureScore.Score
	weak := rp.WeakOnwardPropagationScore.Score
	hop := rp.HopBudgetStressScore.Score
	out.SuspectedRelayHotspot = dup > 0.5 && snap.MessageSignals.TotalMessages >= 10
	out.WeakOnwardPropagationSuspected = weak > 0.55
	out.HopBudgetStressSuspected = hop > 0.55 && snap.MessageSignals.MessagesWithHop > 0

	q := url.Values{}
	q.Set("incident", inc.ID)
	q.Set("filter", "incident_focus")
	if n := incidentTopologyFocusNodeForMeshCompanion(inc); n > 0 {
		q.Set("select", formatInt64(n))
	}
	out.SuggestedTopologySearch = q.Encode()
	return out
}

func meshRoutingCompanionIncidentRelevant(inc models.Incident, intel *models.IncidentIntelligence) bool {
	cat := strings.ToLower(strings.TrimSpace(inc.Category))
	rt := strings.ToLower(strings.TrimSpace(inc.ResourceType))
	if cat == "mesh_topology" || strings.Contains(cat, "mesh") || strings.Contains(cat, "topology") {
		return true
	}
	if rt == "mesh" || rt == "mesh_node" || rt == "node" {
		return true
	}
	if intel != nil {
		for _, d := range intel.ImplicatedDomains {
			if strings.EqualFold(strings.TrimSpace(d.Domain), "topology") {
				return true
			}
		}
	}
	return false
}

func incidentTopologyFocusNodeForMeshCompanion(inc models.Incident) int64 {
	rt := strings.ToLower(strings.TrimSpace(inc.ResourceType))
	rid := strings.TrimSpace(inc.ResourceID)
	if rt == "mesh_node" || rt == "node" {
		var n int64
		for _, r := range rid {
			if r >= '0' && r <= '9' {
				n = n*10 + int64(r-'0')
			}
		}
		if n > 0 {
			return n
		}
	}
	return 0
}

func formatInt64(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [32]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// operatorSuggestedActions builds reviewable, deterministic next checks (not black-box ranking).
func operatorSuggestedActions(cfg config.Config, inc models.Incident, intel *models.IncidentIntelligence) []models.OperatorSuggestedAction {
	if intel == nil {
		return nil
	}
	byID := map[string]models.OperatorSuggestedAction{}
	add := func(a models.OperatorSuggestedAction) {
		if strings.TrimSpace(a.ID) == "" {
			return
		}
		if _, ok := byID[a.ID]; ok {
			return
		}
		byID[a.ID] = a
	}

	add(models.OperatorSuggestedAction{
		ID:           "inspect-replay",
		Title:        "Review incident replay",
		Rationale:    "Replay orders persisted timeline events for this incident id; use it before reusing control patterns.",
		EvidenceRefs: []string{"incident:" + inc.ID},
		Uncertainty:  "Window may truncate or redact by policy; replay completeness is not guaranteed.",
		Href:         "/incidents/" + inc.ID + "/replay",
		Kind:         "inspect_surface",
		DisableHint:  "Replay is always available when you can read incidents; there is no assist toggle for deterministic replay.",
	})

	if cfg.Topology.Enabled {
		href := "/topology"
		if c := intel.MeshRoutingCompanion; c != nil && c.Applicable && strings.TrimSpace(c.SuggestedTopologySearch) != "" {
			href = href + "?" + strings.TrimSpace(c.SuggestedTopologySearch)
		}
		rationale := "Topology shows packet-derived graph and routing-pressure proxies from recent ingest — not RF coverage proof."
		if c := intel.MeshRoutingCompanion; c != nil && c.Applicable {
			parts := []string{rationale}
			if c.SuspectedRelayHotspot {
				parts = append(parts, "Current snapshot flags suspected duplicate-forward / relay hotspot (proxy).")
			}
			if c.WeakOnwardPropagationSuspected {
				parts = append(parts, "Current snapshot flags weak onward propagation in observed edges (proxy).")
			}
			if c.HopBudgetStressSuspected {
				parts = append(parts, "Current snapshot flags hop-limit stress in recent message rollup (proxy).")
			}
			if !c.TransportConnected {
				parts = append(parts, "Transport was disconnected when the companion snapshot was taken — treat as possibly stale.")
			}
			rationale = strings.Join(parts, " ")
		}
		add(models.OperatorSuggestedAction{
			ID:           "inspect-topology",
			Title:        "Open topology with graph context",
			Rationale:    rationale,
			EvidenceRefs: []string{"mesh_intelligence:routing_pressure"},
			Uncertainty:  "Graph is observation-only; does not prove live routing outcomes.",
			Href:         href,
			Kind:         "inspect_surface",
		})
	}

	if len(intel.SimilarIncidents) > 0 {
		peer := intel.SimilarIncidents[0]
		add(models.OperatorSuggestedAction{
			ID:        "compare-similar-incident",
			Title:     "Open highest-similarity prior incident",
			Rationale: "Fingerprint/signature correlation found another incident row on this instance — compare outcomes before repeating the same mitigation.",
			EvidenceRefs: []string{
				"similar_incident:" + peer.IncidentID,
				"incident:" + inc.ID,
			},
			Uncertainty: "Similarity is bounded association; shared signature does not prove shared root cause.",
			Href:        "/incidents/" + peer.IncidentID + "#similar-prior-incidents",
			Kind:        "correlation_memory",
		})
	}

	if len(intel.ActionOutcomeMemory) > 0 {
		mem := intel.ActionOutcomeMemory[0]
		label := strings.TrimSpace(mem.ActionLabel)
		if label == "" {
			label = strings.ReplaceAll(strings.TrimSpace(mem.ActionType), "_", " ")
		}
		rationale := fmt.Sprintf("Prior %s on this signature: framing=%s, sample_size=%d, evidence_strength=%s.",
			label, strings.TrimSpace(mem.OutcomeFraming), mem.SampleSize, strings.TrimSpace(mem.EvidenceStrength))
		add(models.OperatorSuggestedAction{
			ID:        "review-action-outcome-memory",
			Title:     "Review historical action outcome memory",
			Rationale: rationale,
			EvidenceRefs: append([]string{
				"action_outcome_memory:" + strings.TrimSpace(mem.ActionType),
				"incident:" + inc.ID,
			}, mem.EvidenceRefs...),
			Uncertainty: "Memory aggregates prior linked actions on the same signature bucket; association only.",
			Href:        "/incidents/" + inc.ID + "#linked-control-actions",
			Kind:        "correlation_memory",
		})
	}

	out := make([]models.OperatorSuggestedAction, 0, len(byID))
	order := []string{"inspect-replay", "inspect-topology", "compare-similar-incident", "review-action-outcome-memory"}
	for _, id := range order {
		if a, ok := byID[id]; ok {
			out = append(out, a)
			delete(byID, id)
		}
	}
	for _, a := range byID {
		out = append(out, a)
	}
	return out
}
