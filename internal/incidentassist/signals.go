// Package incidentassist computes deterministic, inspectable assist cues from incident API payloads.
// Outputs are non-canonical; they do not imply routing certainty, RF proof, or predictive success.
package incidentassist

import (
	"fmt"
	"strings"

	"github.com/mel-project/mel/internal/models"
)

const SchemaVersion = "incident_assist_signals_v1"

// Compute returns bounded assist signals derived from the incident + intelligence payload (caller sets incident.assist_signals).
func Compute(inc models.Incident, intel *models.IncidentIntelligence) *models.IncidentAssistSignals {
	if intel == nil {
		return nil
	}
	var sigs []models.IncidentAssistSignal
	evidenceBasis := []string{
		"incident.intelligence fields",
		"incident.triage_signals when present",
		"incident row fields (review_state, reopened_from_incident_id)",
	}

	// Recurrence review — signature match / family / reopen stress (association only).
	if intel.SignatureMatchCount > 1 || strings.TrimSpace(inc.ReopenedFromIncidentID) != "" {
		reason := "signature_match_count>1 on this instance"
		if strings.TrimSpace(inc.ReopenedFromIncidentID) != "" {
			reason = "reopened_from_incident_id present on record"
		}
		if intel.SignatureMatchCount > 1 && strings.TrimSpace(inc.ReopenedFromIncidentID) != "" {
			reason = "signature_match_count>1 and reopened_from_incident_id present"
		}
		sigs = append(sigs, models.IncidentAssistSignal{
			Code:         "recurrence_review_recommended",
			Severity:     "review",
			Title:        "Recurrence review recommended",
			Rationale:    fmt.Sprintf("Local evidence: %s — compare timeline and outcomes before reusing the same playbook.", reason),
			EvidenceRefs: []string{"incident.intelligence.signature_match_count", "incident.reopened_from_incident_id"},
			Uncertainty:  "pattern_and_state_association_only_not_root_cause_proof",
		})
	}

	// Mitigation fragility — durability memory or weak outcome memory.
	if intel.MitigationDurabilityMemory != nil {
		switch intel.MitigationDurabilityMemory.Posture {
		case "deterioration_or_mixed_in_outcome_memory", "reopened_after_resolution_in_family", "reopened_incident_on_record":
			sigs = append(sigs, models.IncidentAssistSignal{
				Code:         "mitigation_fragility_watch",
				Severity:     "watch",
				Title:        "Mitigation fragility watch",
				Rationale:    "Local history shows deterioration, mixed outcomes, reopen-on-record, or family reopen stress — verify before repeating the same mitigation.",
				EvidenceRefs: []string{"incident.intelligence.mitigation_durability_memory"},
				Uncertainty:  intel.MitigationDurabilityMemory.Uncertainty,
			})
		}
	}

	// Repeated reopen in family — explicit peer counts when history block exists.
	if h := intel.SignatureFamilyResolvedHistory; h != nil && h.ResolvedPeerCount >= 2 && h.ReopenedPeerCount >= 1 {
		sigs = append(sigs, models.IncidentAssistSignal{
			Code:     "reopen_family_signal",
			Severity: "watch",
			Title:    "Reopen stress in signature family (bounded scan)",
			Rationale: fmt.Sprintf("Among linked peers in scanned window: %d resolved/closed rows and %d reopened-from-prior rows — association only.",
				h.ResolvedPeerCount, h.ReopenedPeerCount),
			EvidenceRefs: []string{"incident.intelligence.signature_family_resolved_history"},
			Uncertainty:  "state_chronology_on_scanned_peers_not_causal",
		})
		if h.PeerHistoryScanTruncated {
			sigs[len(sigs)-1].Uncertainty = "state_chronology_on_recent_peer_window_only_not_full_family"
		}
	}

	// Evidence thin / review needed.
	if intel.EvidenceStrength == "sparse" || intel.Degraded || len(intel.SparsityMarkers) > 1 {
		sigs = append(sigs, models.IncidentAssistSignal{
			Code:         "evidence_thin_review_needed",
			Severity:     "review",
			Title:        "Evidence thin — gather context before strong claims",
			Rationale:    "Intelligence is sparse, degraded, or carries multiple sparsity markers — conclusions stay bounded.",
			EvidenceRefs: []string{"incident.intelligence.evidence_strength", "incident.intelligence.degraded", "incident.intelligence.sparsity_markers"},
			Uncertainty:  "absence_of_dense_evidence_not_proof_of_absence_of_fault",
		})
	}

	// Ingest / graph stress — only when mesh routing companion flags are raised (ingest-graph diagnostics, not RF path proof).
	if mc := intel.MeshRoutingCompanion; mc != nil && mc.Applicable && (mc.SuspectedRelayHotspot || mc.WeakOnwardPropagationSuspected || mc.HopBudgetStressSuspected) {
		sigs = append(sigs, models.IncidentAssistSignal{
			Code:         "ingest_graph_pressure_advisory",
			Severity:     "info",
			Title:        "Ingest-graph pressure flags raised",
			Rationale:    "Mesh routing companion flags suggest ingest-graph pressure patterns — bounded diagnostics adjacent to replay/topology; not RF delivery or path certainty.",
			EvidenceRefs: []string{"incident.intelligence.mesh_routing_companion"},
			Uncertainty:  "ingest_model_only_not_live_path_proof",
		})
	}

	if len(sigs) == 0 {
		return nil
	}

	return &models.IncidentAssistSignals{
		SchemaVersion: SchemaVersion,
		Signals:       sigs,
		Uncertainty:   "assist_signals_are_deterministic_heuristics_from_this_payload_not_canonical_truth",
		EvidenceBasis: strings.Join(evidenceBasis, "; "),
	}
}
