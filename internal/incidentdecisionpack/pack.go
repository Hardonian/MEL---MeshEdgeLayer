// Package incidentdecisionpack assembles the canonical Incident Decision Pack DTO from API-ready incident rows.
// It does not query the database; service layer loads rows and operator adjudication, then calls Build.
package incidentdecisionpack

import (
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/operatorreadiness"
)

const similarIncidentScanCap = 8

var nonClaimStatements = []string{
	"MEL does not prove RF routing, mesh propagation, or transport success from this pack alone.",
	"Queue ordering uses deterministic triage signals — not an opaque global priority score.",
	"Similarity and family history are observational associations from local stored rows, not causal proof.",
}

// Build constructs a versioned IncidentDecisionPack from an incident already finished for API (triage, intel, visibility).
func Build(inc models.Incident, adjudication *models.IncidentDecisionPackAdjudication, readiness operatorreadiness.OperatorReadinessDTO, generatedAt time.Time) models.IncidentDecisionPack {
	out := models.IncidentDecisionPack{
		SchemaVersion: models.IncidentDecisionPackSchemaVersion,
		IncidentID:    inc.ID,
		GeneratedAt:   generatedAt.UTC().Format(time.RFC3339),
	}
	out.Identity = &models.IncidentDecisionPackIdentity{
		Title:          inc.Title,
		State:          inc.State,
		Severity:       inc.Severity,
		Category:       inc.Category,
		ResourceType:   inc.ResourceType,
		ResourceID:     inc.ResourceID,
		OccurredAt:     inc.OccurredAt,
		UpdatedAt:      inc.UpdatedAt,
		ResolvedAt:     inc.ResolvedAt,
		ReviewState:    inc.ReviewState,
		OwnerActorID:   inc.OwnerActorID,
		HandoffSummary: inc.HandoffSummary,
		Summary:        inc.Summary,
	}
	out.Queue = &models.IncidentDecisionPackQueue{
		TriageSignals:       inc.TriageSignals,
		WhySurfacedOneLiner: WhySurfacedOneLiner(inc),
		OrderingNote:        queueOrderingNote(inc),
	}
	out.EvidenceBasis = evidenceBasis(inc)
	out.IntelligenceSummary = intelligenceSummary(inc)
	if inc.Intelligence != nil {
		out.MitigationDurability = inc.Intelligence.MitigationDurabilityMemory
	}
	out.FamilyHistory = familyHistory(inc)
	out.TransportGraph = &models.IncidentDecisionPackTransportGraph{
		MeshRoutingCompanion: nil,
		WirelessContext:      nil,
	}
	if inc.Intelligence != nil {
		out.TransportGraph.MeshRoutingCompanion = inc.Intelligence.MeshRoutingCompanion
		out.TransportGraph.WirelessContext = inc.Intelligence.WirelessContext
	}
	out.LinkedActions = linkedActions(inc)
	out.Readiness = readinessBlock(inc, readiness)
	out.Uncertainty = uncertaintyBlock(inc)
	if inc.AssistSignals != nil {
		as := *inc.AssistSignals
		out.AssistSignals = &as
	}
	if adjudication != nil {
		adj := *adjudication
		out.OperatorAdjudication = &adj
	}
	out.AnalyticsHints = analyticsHints(inc)
	return out
}

func queueOrderingNote(inc models.Incident) string {
	if inc.TriageSignals == nil {
		return ""
	}
	if s := strings.TrimSpace(inc.TriageSignals.QueueOrderingContract); s != "" {
		return "Queue contract: " + s + " — primary sort key matches triage tier when present."
	}
	return ""
}

func evidenceBasis(inc models.Incident) *models.IncidentDecisionPackEvidenceBasis {
	intel := inc.Intelligence
	if intel == nil {
		return &models.IncidentDecisionPackEvidenceBasis{
			Degraded:        true,
			DegradedReasons: []string{"no_intelligence_assembly"},
		}
	}
	const capN = 12
	items := intel.EvidenceItems
	capApplied := 0
	if len(items) > capN {
		items = append([]models.IncidentEvidenceItem(nil), items[:capN]...)
		capApplied = capN
	}
	return &models.IncidentDecisionPackEvidenceBasis{
		EvidenceStrength: intel.EvidenceStrength,
		EvidenceItems:    items,
		ItemCapApplied:   capApplied,
		Degraded:         intel.Degraded,
		DegradedReasons:  append([]string(nil), intel.DegradedReasons...),
		SparsityMarkers:  append([]string(nil), intel.SparsityMarkers...),
	}
}

func intelligenceSummary(inc models.Incident) *models.IncidentDecisionPackIntelligenceSummary {
	intel := inc.Intelligence
	if intel == nil {
		return nil
	}
	var lines []string
	if strings.TrimSpace(intel.SignatureLabel) != "" {
		lines = append(lines, fmt.Sprintf("Signature label: %s (match count=%d on this instance).", intel.SignatureLabel, intel.SignatureMatchCount))
	}
	if len(intel.LearningLoopHints) > 0 {
		lines = append(lines, intel.LearningLoopHints...)
	}
	var invNext []string
	for _, g := range intel.InvestigateNext {
		if strings.TrimSpace(g.ID) != "" {
			invNext = append(invNext, g.ID)
		}
	}
	var rb []string
	for _, r := range intel.RunbookRecommendations {
		if strings.TrimSpace(r.ID) != "" {
			rb = append(rb, r.ID)
		}
	}
	return &models.IncidentDecisionPackIntelligenceSummary{
		SignatureLabel:      intel.SignatureLabel,
		SignatureMatchCount: intel.SignatureMatchCount,
		SummaryLines:        lines,
		InvestigateNextIDs:  invNext,
		RunbookRecIDs:       rb,
		LearningLoopHints:   append([]string(nil), intel.LearningLoopHints...),
	}
}

func familyHistory(inc models.Incident) *models.IncidentDecisionPackFamilyHistory {
	intel := inc.Intelligence
	if intel == nil {
		return nil
	}
	return &models.IncidentDecisionPackFamilyHistory{
		SignatureFamily:  intel.SignatureFamilyResolvedHistory,
		SimilarIncidents: append([]models.IncidentSimilarityRecord(nil), intel.SimilarIncidents...),
		SimilarScanCap:   similarIncidentScanCap,
	}
}

func linkedActions(inc models.Incident) *models.IncidentDecisionPackLinkedActions {
	var ids []string
	for _, a := range inc.LinkedControlActions {
		if strings.TrimSpace(a.ID) != "" {
			ids = append(ids, a.ID)
		}
	}
	var sug []models.OperatorSuggestedAction
	if inc.Intelligence != nil {
		sug = append([]models.OperatorSuggestedAction(nil), inc.Intelligence.OperatorSuggestedActions...)
	}
	return &models.IncidentDecisionPackLinkedActions{
		ActionVisibility:        inc.ActionVisibility,
		OperatorSuggestedAction: sug,
		LinkedControlActionIDs:  ids,
	}
}

func readinessBlock(inc models.Incident, readiness operatorreadiness.OperatorReadinessDTO) *models.IncidentDecisionPackReadiness {
	var blockers []string
	for _, b := range readiness.Blockers {
		if strings.TrimSpace(b.Code) != "" {
			blockers = append(blockers, b.Code)
		}
	}
	suff := ""
	if inc.Intelligence != nil {
		switch inc.Intelligence.EvidenceStrength {
		case "sparse":
			suff = "Evidence strength is sparse — exports and proofpacks remain assembly-time snapshots; review gaps before handoff."
		case "moderate":
			suff = "Evidence strength is moderate — include replay and linked control context when exporting."
		case "strong":
			suff = "Evidence strength is strong — still review export redaction and policy gates before external handoff."
		}
	}
	return &models.IncidentDecisionPackReadiness{
		ExportPolicySemantic:    string(readiness.Semantic),
		ExportPolicySummary:     readiness.Summary,
		ExportArtifactStrength:  string(readiness.ArtifactStrength),
		ExportBlockerCodes:      blockers,
		ProofpackPath:           fmt.Sprintf("/api/v1/incidents/%s/proofpack", inc.ID),
		EscalationBundlePath:    fmt.Sprintf("/api/v1/incidents/%s/escalation-bundle", inc.ID),
		HandoffPostureNote:      "Handoff and workflow fields are operator-owned; export routes may redact identifiers per policy.",
		EvidenceSufficiencyNote: suff,
	}
}

func uncertaintyBlock(inc models.Incident) *models.IncidentDecisionPackUncertainty {
	var bounded []string
	bounded = append(bounded, fmt.Sprintf("Similar incidents list is bounded (cap=%d signature peers).", similarIncidentScanCap))
	if inc.Intelligence != nil && inc.Intelligence.SignatureFamilyResolvedHistory != nil {
		fh := inc.Intelligence.SignatureFamilyResolvedHistory
		if fh.PeerHistoryScanTruncated {
			bounded = append(bounded, fmt.Sprintf("Family resolved/reopened peer scan truncated to last %d linked peers (family total match count is exact).", fh.PeerScanWindow))
		}
	}
	labels := []string{
		"investigate_next: assistive guidance",
		"runbook_recommendations: assistive, non-command",
		"operator_suggested_actions: deterministic affordances",
	}
	var degraded []string
	if inc.Intelligence != nil && inc.Intelligence.Degraded {
		degraded = append(degraded, "incident_intelligence")
	}
	if inc.ActionVisibility != nil && inc.ActionVisibility.IsPartial {
		degraded = append(degraded, "action_visibility_partial")
	}
	return &models.IncidentDecisionPackUncertainty{
		NonClaims:              append([]string(nil), nonClaimStatements...),
		InterpretationLabels:   labels,
		DegradedSections:       degraded,
		BoundedScanDisclosures: bounded,
	}
}

func analyticsHints(inc models.Incident) *models.IncidentDecisionPackAnalyticsHints {
	h := &models.IncidentDecisionPackAnalyticsHints{}
	if inc.TriageSignals != nil {
		h.TriageTier = inc.TriageSignals.Tier
	}
	if inc.Intelligence != nil {
		h.SignatureKey = inc.Intelligence.SignatureKey
		h.EvidenceStrength = inc.Intelligence.EvidenceStrength
		h.IntelDegraded = inc.Intelligence.Degraded
		if inc.Intelligence.Fingerprint != nil {
			h.FingerprintCanonicalHash = inc.Intelligence.Fingerprint.CanonicalHash
		}
		if inc.Intelligence.MitigationDurabilityMemory != nil {
			h.MitigationDurabilityPosture = inc.Intelligence.MitigationDurabilityMemory.Posture
		}
	}
	return h
}

// WhySurfacedOneLiner mirrors the incident workbench “why” contract (deterministic, bounded).
func WhySurfacedOneLiner(inc models.Incident) string {
	rs := strings.ToLower(strings.TrimSpace(inc.ReviewState))
	followUp := rs == "follow_up_needed" || rs == "pending_review" || rs == "mitigated"
	if followUp {
		return fmt.Sprintf("Review state %q — explicit follow-up or review posture in MEL.", inc.ReviewState)
	}
	if inc.TriageSignals != nil && len(inc.TriageSignals.Codes) > 0 {
		pick := []string{"governance_friction_memory", "mitigation_durability_weak_in_family", "sparse_or_degraded_intel"}
		for _, code := range pick {
			for i, c := range inc.TriageSignals.Codes {
				if c == code && i < len(inc.TriageSignals.RationaleLines) && strings.TrimSpace(inc.TriageSignals.RationaleLines[i]) != "" {
					return inc.TriageSignals.RationaleLines[i] + " (triage code: " + strings.ReplaceAll(code, "_", " ") + ")."
				}
			}
		}
	}
	if inc.Intelligence != nil && inc.Intelligence.MitigationDurabilityMemory != nil {
		md := inc.Intelligence.MitigationDurabilityMemory
		if md.Posture == "reopened_after_resolution_in_family" || md.Posture == "deterioration_or_mixed_in_outcome_memory" {
			return md.Summary + " (" + strings.ReplaceAll(md.Uncertainty, "_", " ") + ")."
		}
	}
	if inc.ActionVisibility != nil {
		switch inc.ActionVisibility.Kind {
		case "visibility_limited", "action_context_degraded", "no_linked_historical_signals":
			return inc.ActionVisibility.Summary
		}
	}
	if inc.Intelligence != nil {
		if inc.Intelligence.EvidenceStrength == "sparse" || inc.Intelligence.Degraded {
			return "Sparse or degraded intelligence — keep conclusions bounded; gather replay, topology, and control context."
		}
		if inc.Intelligence.SignatureMatchCount > 1 {
			return "Recurring signature on this instance — pattern memory, not proof of repeating root cause."
		}
	}
	if strings.TrimSpace(inc.ReopenedFromIncidentID) != "" {
		return "Reopened incident — compare replay and outcomes before reusing the same control pattern."
	}
	return "Open in workflow — verify state against replay and exports before stronger claims."
}
