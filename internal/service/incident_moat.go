package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/auth"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/models"
)

func (a *App) enrichIncidentIntelligenceMoat(inc models.Incident, intel *models.IncidentIntelligence) {
	if intel == nil {
		return
	}
	var sparse []string
	if len(intel.SimilarIncidents) == 0 {
		sparse = append(sparse, "no_similar_incident_history")
	}
	if len(intel.ActionOutcomeMemory) == 0 {
		sparse = append(sparse, "insufficient_action_outcome_history")
	}
	if len(intel.EvidenceItems) < 2 {
		sparse = append(sparse, "limited_correlated_evidence")
	}
	intel.SparsityMarkers = sparse

	intel.RunbookRecommendations = deriveRunbookRecommendations(intel, inc, a)
	intel.PolicyGovernanceHints = derivePolicyGovernanceHints(inc)
	intel.GovernanceMemory = deriveGovernanceMemory(inc)
	if groups, err := a.DB.CorrelationGroupsForIncident(inc.ID); err == nil {
		for i := range groups {
			ids, _ := a.DB.CorrelatedIncidentIDsForGroup(groups[i].GroupID)
			groups[i].MemberCount = len(ids)
			for _, oid := range ids {
				if oid != inc.ID {
					groups[i].OtherIncidentIDs = append(groups[i].OtherIncidentIDs, oid)
				}
			}
			sort.Strings(groups[i].OtherIncidentIDs)
		}
		intel.CorrelationGroups = groups
	}
	a.syncMultiSignalFaultDomain(inc, intel)
	if assets := a.runbookAssetsForIntel(intel); len(assets) > 0 {
		intel.RunbookAssets = assets
	}
	intel.ReplayHints = buildReplayHints(inc, intel)
	intel.LearningLoopHints = buildLearningLoopHints(intel, inc)
}

func buildLearningLoopHints(intel *models.IncidentIntelligence, inc models.Incident) []string {
	if intel == nil {
		return nil
	}
	var hints []string
	if (intel.SignatureMatchCount) >= 3 {
		hints = append(hints, "Repeated signature: consider capturing resolution_summary and lessons_learned so future runbook strength can cite this incident family.")
	}
	if len(intel.ActionOutcomeMemory) > 0 {
		hints = append(hints, "Action outcome memory present: record recommendation outcomes (accepted/rejected/ineffective) to tighten future ranking without implying automation.")
	}
	if strings.TrimSpace(inc.CloseoutReason) == "" && isResolvedState(inc.State) {
		hints = append(hints, "Incident state is terminal but closeout_reason is empty; add closeout_reason for proofpack and escalation completeness.")
	}
	return hints
}

func deriveRunbookRecommendations(intel *models.IncidentIntelligence, inc models.Incident, a *App) []models.IncidentRunbookRecommendation {
	if intel == nil {
		return nil
	}
	effByID := map[string]db.RecEffectivenessRecord{}
	if a != nil && a.DB != nil && strings.TrimSpace(intel.SignatureKey) != "" {
		if rows, err := a.DB.RecEffectivenessByScope(intel.SignatureKey); err == nil {
			for _, r := range rows {
				effByID[r.RecommendationID] = r
			}
		}
	}
	byType := map[string]models.IncidentActionOutcomeMemory{}
	for _, m := range intel.ActionOutcomeMemory {
		byType[m.ActionType] = m
	}
	out := make([]models.IncidentRunbookRecommendation, 0, len(intel.InvestigateNext)+len(intel.ActionOutcomeMemory))
	seen := map[string]struct{}{}
	for _, g := range intel.InvestigateNext {
		id := strings.TrimSpace(g.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		baseStrength := mapGuidanceConfidenceToStrength(g.Confidence)
		rec := models.IncidentRunbookRecommendation{
			ID:               id,
			Title:            g.Title,
			Rationale:        g.Rationale,
			EvidenceRefs:     append([]string(nil), g.EvidenceRefs...),
			Strength:         baseStrength,
			RequiresApproval: false,
			Reversibility:    "unknown",
			IsCommand:        false,
		}
		applyOutcomeWeighting(&rec, effByID[id], intel, false)
		out = append(out, rec)
	}
	for actionType, mem := range byType {
		if strings.TrimSpace(actionType) == "" {
			continue
		}
		id := "runbook-action-" + actionType
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		strength := strengthFromActionMemory(mem)
		reqAppr, blast, rev := actionGovernanceFromIncident(inc, actionType)
		rec := models.IncidentRunbookRecommendation{
			ID:                  id,
			Title:               "Consider control action pattern: " + firstNonEmpty(mem.ActionLabel, actionType),
			ActionType:          actionType,
			Rationale:           "Historical association from similar signature incidents; temporal association only — not causal proof. Review approval gates before any execution.",
			EvidenceRefs:        append([]string(nil), mem.EvidenceRefs...),
			Strength:            strength,
			RequiresApproval:    reqAppr,
			BlastRadiusClass:    blast,
			Reversibility:       rev,
			PriorOutcomeFraming: mem.OutcomeFraming,
			PriorSampleSize:     mem.SampleSize,
			IsCommand:           true,
		}
		applyOutcomeWeighting(&rec, effByID[id], intel, true)
		out = append(out, rec)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].RankScore == out[j].RankScore {
			return out[i].ID < out[j].ID
		}
		return out[i].RankScore > out[j].RankScore
	})
	return out
}

func strengthFromActionMemory(mem models.IncidentActionOutcomeMemory) string {
	if mem.SampleSize < 2 {
		return "unsupported"
	}
	switch mem.EvidenceStrength {
	case "strong":
		if mem.DeteriorationObservedCount > mem.ImprovementObservedCount+1 {
			return "weakly_supported"
		}
		return "historically_proven"
	case "moderate":
		return "historically_promising"
	case "sparse":
		return "weakly_supported"
	default:
		return "plausible"
	}
}

func applyOutcomeWeighting(rec *models.IncidentRunbookRecommendation, eff db.RecEffectivenessRecord, intel *models.IncidentIntelligence, isCommand bool) {
	if rec == nil {
		return
	}
	base := rankScoreFromStrength(rec.Strength)
	explain := []string{fmt.Sprintf("Base rank from evidence-derived strength %q (deterministic rules; not ML).", rec.Strength)}
	if eff.TotalCount > 0 {
		harm := eff.IneffectiveCount + eff.WorsenedCount*2
		help := eff.AcceptedCount*2 + eff.ModifiedCount
		explain = append(explain, fmt.Sprintf("Operator outcomes in signature scope: n=%d accepted=%d rejected=%d ineffective=%d worsened=%d.",
			eff.TotalCount, eff.AcceptedCount, eff.RejectedCount, eff.IneffectiveCount, eff.WorsenedCount))
		base += float64(help - harm - eff.RejectedCount)
		rec.HistoricalOutcomeNote = fmt.Sprintf("Scoped outcomes: total=%d net_help_score=%d", eff.TotalCount, help-harm-eff.RejectedCount)
		if harm >= 3 && eff.AcceptedCount == 0 && isCommand {
			rec.Suppressed = true
			rec.SuppressedReason = "Repeated ineffective or harmful operator outcomes in this signature scope; downgraded to unsupported until reviewed."
			rec.Strength = "unsupported"
			explain = append(explain, rec.SuppressedReason)
			base = -50
		}
	}
	if intel != nil && len(intel.SparsityMarkers) > 2 {
		base -= 1.5
		explain = append(explain, "Sparse history penalty applied (multiple sparsity markers on this incident intelligence view).")
	}
	rec.RankScore = base
	rec.StrengthExplanation = explain
}

func rankScoreFromStrength(s string) float64 {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "historically_proven", "proven_historically":
		return 100
	case "historically_promising":
		return 85
	case "plausible":
		return 70
	case "weakly_supported":
		return 45
	case "unsupported":
		return 10
	default:
		return 40
	}
}

func mapGuidanceConfidenceToStrength(c string) string {
	switch strings.ToLower(strings.TrimSpace(c)) {
	case "medium":
		return "historically_promising"
	default:
		return "weakly_supported"
	}
}

func actionGovernanceFromIncident(inc models.Incident, actionType string) (requiresApproval bool, blastRadius string, reversibility string) {
	reversibility = "unknown"
	for _, ca := range inc.LinkedControlActions {
		if strings.TrimSpace(ca.ActionType) != actionType {
			continue
		}
		if ca.RequiresSeparateApprover {
			requiresApproval = true
		}
		if strings.TrimSpace(ca.BlastRadiusClass) != "" {
			blastRadius = ca.BlastRadiusClass
		}
		switch strings.ToLower(strings.TrimSpace(ca.BlastRadiusClass)) {
		case "low", "segment_local":
			reversibility = "high"
		case "medium", "transport_wide":
			reversibility = "medium"
		case "high", "fleet", "site_wide":
			reversibility = "low"
		}
		break
	}
	if blastRadius == "" {
		blastRadius = "unknown"
	}
	return requiresApproval, blastRadius, reversibility
}

func derivePolicyGovernanceHints(inc models.Incident) []models.IncidentPolicyGovernanceHint {
	if len(inc.LinkedControlActions) == 0 {
		return nil
	}
	approved := 0
	rejected := 0
	sod := 0
	highBlast := 0
	for _, ca := range inc.LinkedControlActions {
		if ca.RequiresSeparateApprover || strings.TrimSpace(ca.ApprovedBy) != "" {
			approved++
		}
		if strings.TrimSpace(ca.RejectedBy) != "" {
			rejected++
		}
		if ca.SodBypass {
			sod++
		}
		if strings.EqualFold(strings.TrimSpace(ca.BlastRadiusClass), "high") || strings.EqualFold(strings.TrimSpace(ca.BlastRadiusClass), "fleet") {
			highBlast++
		}
	}
	summary := fmt.Sprintf("Linked actions: %d rows; separate-approver or approved markers=%d; rejected=%d; SoD bypass flags=%d; high-blast class hints=%d.",
		len(inc.LinkedControlActions), approved, rejected, sod, highBlast)
	return []models.IncidentPolicyGovernanceHint{
		{
			Summary:      summary,
			EvidenceRefs: []string{"incident:" + inc.ID + ":linked_control_actions"},
			Posture:      "observed_from_linked_actions",
		},
	}
}

func deriveGovernanceMemory(inc models.Incident) []models.IncidentGovernanceMemory {
	if len(inc.LinkedControlActions) == 0 {
		return nil
	}
	byType := map[string]*models.IncidentGovernanceMemory{}
	for _, ca := range inc.LinkedControlActions {
		t := strings.TrimSpace(ca.ActionType)
		if t == "" {
			t = "unknown_action_type"
		}
		if byType[t] == nil {
			byType[t] = &models.IncidentGovernanceMemory{
				ActionType: t,
				EvidenceRefs: []string{
					"incident:" + inc.ID + ":linked_control_actions",
				},
			}
		}
		g := byType[t]
		g.LinkedActionCount++
		if strings.TrimSpace(ca.ApprovedBy) != "" || ca.RequiresSeparateApprover {
			g.ApprovedOrPassedCount++
		}
		if strings.TrimSpace(ca.RejectedBy) != "" {
			g.RejectedCount++
		}
		if ca.HighBlastRadius || strings.EqualFold(strings.TrimSpace(ca.BlastRadiusClass), "high") || strings.EqualFold(strings.TrimSpace(ca.BlastRadiusClass), "fleet") {
			g.HighBlastCount++
		}
		if ca.RequiresSeparateApprover {
			g.SeparateApproverCount++
		}
	}
	out := make([]models.IncidentGovernanceMemory, 0, len(byType))
	for _, g := range byType {
		g.Summary = fmt.Sprintf("Observed linked actions for type %s: n=%d approved_or_gated=%d rejected=%d high_blast=%d separate_approver=%d (association only; not policy truth).",
			g.ActionType, g.LinkedActionCount, g.ApprovedOrPassedCount, g.RejectedCount, g.HighBlastCount, g.SeparateApproverCount)
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ActionType < out[j].ActionType })
	return out
}

func (a *App) runbookAssetsForIntel(intel *models.IncidentIntelligence) []models.IncidentRunbookAsset {
	if a == nil || a.DB == nil || intel == nil {
		return nil
	}
	var rows []db.RunbookEntryRecord
	var err error
	if intel.Fingerprint != nil && strings.TrimSpace(intel.Fingerprint.CanonicalHash) != "" {
		rows, err = a.DB.RunbookEntriesForFingerprintHash(intel.Fingerprint.CanonicalHash, 20)
	}
	if err != nil || len(rows) == 0 {
		rows, _ = a.DB.RunbookEntriesForSignature(intel.SignatureKey, 20)
	}
	out := make([]models.IncidentRunbookAsset, 0, len(rows))
	for _, r := range rows {
		var refs, srcIDs []string
		_ = json.Unmarshal([]byte(r.EvidenceRefJSON), &refs)
		_ = json.Unmarshal([]byte(r.SourceIncidentIDsJSON), &srcIDs)
		out = append(out, models.IncidentRunbookAsset{
			ID:                 r.ID,
			Status:             r.Status,
			SourceKind:         r.SourceKind,
			Title:              r.Title,
			Body:               r.Body,
			EvidenceRefs:       refs,
			SourceIncidentIDs:  srcIDs,
			LegacySignatureKey: r.LegacySignatureKey,
			FingerprintHash:    r.FingerprintCanonicalHash,
			PromotionBasis:     r.PromotionBasis,
			CreatedAt:          r.CreatedAt,
			UpdatedAt:          r.UpdatedAt,
		})
	}
	return out
}

func (a *App) syncMultiSignalFaultDomain(inc models.Incident, intel *models.IncidentIntelligence) {
	if a == nil || a.DB == nil || intel == nil {
		return
	}
	parts := []string{intel.SignatureKey}
	if intel.Fingerprint != nil && strings.TrimSpace(intel.Fingerprint.CanonicalHash) != "" {
		parts = append(parts, intel.Fingerprint.CanonicalHash)
	}
	if inc.ResourceType == "transport" && strings.TrimSpace(inc.ResourceID) != "" {
		parts = append(parts, "transport:"+inc.ResourceID)
	}
	raw := strings.Join(parts, "|")
	sum := sha256.Sum256([]byte(raw))
	domainKey := "fd:" + hex.EncodeToString(sum[:10])
	domainID := "ifd-" + hex.EncodeToString(sum[:8])
	rationale := []string{
		"Grouping joins signature bucket, structured fingerprint hash, and transport resource when present.",
		"Uncertainty is explicit: shared symptoms and correlated persistence only — not verified single root cause.",
	}
	evidence := map[string]string{
		"legacy_signature": intel.SignatureKey,
	}
	if intel.Fingerprint != nil {
		evidence["fingerprint_canonical_hash"] = intel.Fingerprint.CanonicalHash
	}
	if inc.ResourceType == "transport" {
		evidence["transport_resource"] = inc.ResourceID
	}
	rj, _ := json.Marshal(rationale)
	ej, _ := json.Marshal(evidence)
	uncertainty := "possibly_related"
	if len(intel.EvidenceItems) >= 4 && intel.SignatureMatchCount >= 2 {
		uncertainty = "likely_related"
	}
	if len(intel.EvidenceItems) < 2 {
		uncertainty = "inconclusive"
	}
	_ = a.DB.UpsertFaultDomain(db.FaultDomainRecord{
		ID:                 domainID,
		DomainKey:          domainKey,
		Basis:              "multi_signal_fingerprint_transport",
		Uncertainty:        uncertainty,
		RationaleJSON:      string(rj),
		EvidenceBundleJSON: string(ej),
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:          time.Now().UTC().Format(time.RFC3339),
	})
	members := []db.FaultDomainMember{{Kind: "incident", ID: inc.ID, Reason: "scoped_incident"}}
	if inc.ResourceType == "transport" && strings.TrimSpace(inc.ResourceID) != "" {
		members = append(members, db.FaultDomainMember{Kind: "transport", ID: inc.ResourceID, Reason: "incident_resource"})
	}
	for _, cg := range intel.CorrelationGroups {
		if strings.TrimSpace(cg.GroupID) != "" {
			members = append(members, db.FaultDomainMember{Kind: "correlation_group", ID: cg.GroupID, Reason: "structural_correlation"})
		}
	}
	for _, it := range intel.EvidenceItems {
		if it.Kind == "transport_alert" && strings.HasPrefix(it.ReferenceID, "transport_alert:") {
			id := strings.TrimPrefix(it.ReferenceID, "transport_alert:")
			members = append(members, db.FaultDomainMember{Kind: "transport_alert", ID: id, Reason: "incident_window_evidence"})
		}
	}
	_ = a.DB.ReplaceFaultDomainMembers(domainID, members)
	if doms, err := a.DB.FaultDomainsForIncident(inc.ID); err == nil && len(doms) > 0 {
		intel.FaultDomains = doms
	}
}

func (a *App) maybeSyncRunbookCandidate(inc models.Incident, intel *models.IncidentIntelligence) {
	if a == nil || a.DB == nil || intel == nil {
		return
	}
	if intel.SignatureMatchCount < 3 {
		return
	}
	rs := strings.TrimSpace(inc.ResolutionSummary)
	ll := strings.TrimSpace(inc.LessonsLearned)
	if rs == "" && ll == "" {
		return
	}
	fpHash := ""
	if intel.Fingerprint != nil {
		fpHash = intel.Fingerprint.CanonicalHash
	}
	sum := sha256.Sum256([]byte(intel.SignatureKey + "|" + fpHash + "|" + rs + "|" + ll))
	id := "irb-" + hex.EncodeToString(sum[:10])
	title := "Candidate guidance from repeated incidents"
	if rs != "" {
		title = "Candidate: " + truncateRunbookTitle(rs, 80)
	}
	body := rs
	if ll != "" {
		if body != "" {
			body += "\n\nLessons learned:\n" + ll
		} else {
			body = "Lessons learned:\n" + ll
		}
	}
	evidenceRefs := []string{"incident:" + inc.ID, "incident_signatures:" + intel.SignatureKey}
	if fpHash != "" {
		evidenceRefs = append(evidenceRefs, "incident_fingerprint:"+fpHash)
	}
	srcIDs := []string{inc.ID}
	refsJ, _ := json.Marshal(evidenceRefs)
	srcJ, _ := json.Marshal(srcIDs)
	_ = a.DB.InsertRunbookEntry(db.RunbookEntryRecord{
		ID:                       id,
		Status:                   "proposed",
		SourceKind:               "repeated_incident_resolution_text",
		LegacySignatureKey:       intel.SignatureKey,
		FingerprintCanonicalHash: fpHash,
		Title:                    title,
		Body:                     body,
		EvidenceRefJSON:          string(refsJ),
		SourceIncidentIDsJSON:    string(srcJ),
		PromotionBasis:           fmt.Sprintf("signature_match_count=%d with non-empty resolution or lessons on this incident", intel.SignatureMatchCount),
	})
}

func truncateRunbookTitle(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func (a *App) deriveDriftFingerprints(inc models.Incident) []models.IncidentDriftFingerprint {
	if a == nil || a.DB == nil || inc.ResourceType != "transport" || strings.TrimSpace(inc.ResourceID) == "" {
		return nil
	}
	from, to := incidentEvidenceWindow(inc)
	tFrom, err1 := time.Parse(time.RFC3339, strings.TrimSpace(from))
	tTo, err2 := time.Parse(time.RFC3339, strings.TrimSpace(to))
	if err1 != nil || err2 != nil || !tTo.After(tFrom) {
		return nil
	}
	duration := tTo.Sub(tFrom)
	priorTo := tFrom
	priorFrom := tFrom.Add(-duration)
	cur, err := a.DB.TransportAnomalyHistory(inc.ResourceID, from, to, 50, 0)
	if err != nil || len(cur) == 0 {
		return nil
	}
	prior, _ := a.DB.TransportAnomalyHistory(inc.ResourceID, priorFrom.Format(time.RFC3339), priorTo.Format(time.RFC3339), 50, 0)
	priorByReason := map[string]int{}
	for _, p := range prior {
		priorByReason[strings.TrimSpace(p.Reason)] += int(p.Count)
	}
	out := make([]models.IncidentDriftFingerprint, 0, 3)
	reasonCounts := map[string]int{}
	for _, c := range cur {
		reasonCounts[strings.TrimSpace(c.Reason)] += int(c.Count)
	}
	type pair struct {
		reason string
		cur    int
		prior  int
	}
	var pairs []pair
	for r, c := range reasonCounts {
		if r == "" {
			continue
		}
		pairs = append(pairs, pair{reason: r, cur: c, prior: priorByReason[r]})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].cur == pairs[j].cur {
			return pairs[i].reason < pairs[j].reason
		}
		return pairs[i].cur > pairs[j].cur
	})
	for _, p := range pairs {
		if len(out) >= 3 {
			break
		}
		stmt := fmt.Sprintf("Transport %s: reason %q appears in the incident window with count sum=%d; prior window sum=%d. Resembles recurring anomaly family only if counts are stable across windows — verify raw dead letters and alerts.",
			inc.ResourceID, p.reason, p.cur, p.prior)
		out = append(out, models.IncidentDriftFingerprint{
			Kind:              "transport_anomaly_reason_recurring",
			TransportName:     inc.ResourceID,
			Reason:            p.reason,
			Statement:         stmt,
			CurrentBucketHits: p.cur,
			PriorBucketHits:   p.prior,
			SupportsOnly:      "association",
		})
	}
	return out
}

func buildReplayHints(inc models.Incident, intel *models.IncidentIntelligence) *models.IncidentReplayHints {
	if intel == nil {
		return nil
	}
	refs := []string{"incident:" + inc.ID}
	for _, it := range intel.EvidenceItems {
		if strings.TrimSpace(it.ReferenceID) != "" {
			refs = append(refs, it.ReferenceID)
		}
	}
	note := "Replay is evidence ordering review only. Runbook rows marked is_command are not executed by this surface; use the control plane with approvals."
	if len(intel.ActionOutcomeSnapshots) > 0 {
		note += " Historical action-outcome snapshots show bounded pre/post windows around past actions on similar signatures."
	}
	rankNote := "Counterfactual ranking is bounded: given current deterministic recommendation ranking (rules + stored outcomes), order may differ from what an operator saw historically; this does not imply a better outcome would have occurred."
	return &models.IncidentReplayHints{
		Statement:          "Use timeline, linked actions, and proofpack export to reconstruct what MEL had persisted before each operator step.",
		EvidenceAtTimeRefs: refs,
		CounterfactualNote: note,
		RankingModelNote:   rankNote,
	}
}

// RecordRecommendationOutcome persists operator adjudication for assistive recommendations.
func (a *App) RecordRecommendationOutcome(incidentID, actorID string, req models.IncidentRecommendationOutcomeRequest) error {
	if a == nil || a.DB == nil {
		return fmt.Errorf("service not available")
	}
	incidentID = strings.TrimSpace(incidentID)
	recID := strings.TrimSpace(req.RecommendationID)
	outcome := strings.TrimSpace(req.Outcome)
	if incidentID == "" || recID == "" || outcome == "" {
		return fmt.Errorf("incident_id, recommendation_id, and outcome are required")
	}
	if !validRecommendationOutcome(outcome) {
		return fmt.Errorf("unknown outcome %q", outcome)
	}
	if strings.TrimSpace(actorID) == "" {
		actorID = "system"
	}
	_, ok, err := a.DB.IncidentByID(incidentID)
	if err != nil {
		return fmt.Errorf("could not load incident: %w", err)
	}
	if !ok {
		return fmt.Errorf("incident not found: %s", incidentID)
	}
	rec := db.IncidentRecommendationOutcomeRecord{
		ID:               newTrustID("iro"),
		IncidentID:       incidentID,
		RecommendationID: recID,
		Outcome:          outcome,
		ActorID:          actorID,
		Note:             strings.TrimSpace(req.Note),
	}
	if err := a.DB.InsertIncidentRecommendationOutcome(rec); err != nil {
		return fmt.Errorf("could not persist outcome: %w", err)
	}
	sigKey, _ := a.DB.SignatureKeyForIncident(incidentID)
	if strings.TrimSpace(sigKey) != "" {
		_ = a.DB.AccumulateRecommendationEffectiveness(sigKey, recID, outcome)
	}
	_ = a.DB.InsertRBACAuditLog(auth.AuditEntry{
		ID:           newTrustID("aud"),
		ActorID:      auth.OperatorID(actorID),
		ActionClass:  auth.ActionControl,
		ActionDetail: "incident_recommendation_outcome",
		ResourceType: "incident",
		ResourceID:   incidentID,
		Reason:       fmt.Sprintf("recommendation_id=%s outcome=%s", recID, outcome),
		Result:       auth.AuditResultSuccess,
		Timestamp:    time.Now().UTC(),
	})
	return nil
}

func validRecommendationOutcome(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "accepted", "rejected", "modified", "not_attempted", "ineffective", "worsened", "resolved_incident", "unknown":
		return true
	default:
		return false
	}
}

// PatchIncidentWorkflow updates durable review fields; does not execute control actions.
func (a *App) PatchIncidentWorkflow(incidentID, actorID string, patch models.IncidentWorkflowPatch) error {
	if a == nil || a.DB == nil {
		return fmt.Errorf("service not available")
	}
	incidentID = strings.TrimSpace(incidentID)
	if incidentID == "" {
		return fmt.Errorf("incident id is required")
	}
	inc, ok, err := a.DB.IncidentByID(incidentID)
	if err != nil {
		return fmt.Errorf("could not load incident: %w", err)
	}
	if !ok {
		return fmt.Errorf("incident not found: %s", incidentID)
	}
	if patch.ReviewState != nil {
		rs := strings.TrimSpace(*patch.ReviewState)
		if rs != "" && !validReviewState(rs) {
			return fmt.Errorf("invalid review_state %q", rs)
		}
		if rs != "" {
			inc.ReviewState = rs
		}
	}
	if patch.InvestigationNotes != nil {
		inc.InvestigationNotes = strings.TrimSpace(*patch.InvestigationNotes)
	}
	if patch.ResolutionSummary != nil {
		inc.ResolutionSummary = strings.TrimSpace(*patch.ResolutionSummary)
	}
	if patch.CloseoutReason != nil {
		inc.CloseoutReason = strings.TrimSpace(*patch.CloseoutReason)
	}
	if patch.LessonsLearned != nil {
		inc.LessonsLearned = strings.TrimSpace(*patch.LessonsLearned)
	}
	if patch.ReopenedFromIncidentID != nil {
		v := strings.TrimSpace(*patch.ReopenedFromIncidentID)
		inc.ReopenedFromIncidentID = v
		if v != "" {
			inc.ReopenedAt = time.Now().UTC().Format(time.RFC3339)
		}
	}
	if err := a.DB.UpsertIncident(inc); err != nil {
		return fmt.Errorf("could not persist workflow: %w", err)
	}
	if strings.TrimSpace(actorID) == "" {
		actorID = "system"
	}
	_ = a.DB.InsertRBACAuditLog(auth.AuditEntry{
		ID:           newTrustID("aud"),
		ActorID:      auth.OperatorID(actorID),
		ActionClass:  auth.ActionControl,
		ActionDetail: "incident_workflow_patch",
		ResourceType: "incident",
		ResourceID:   incidentID,
		Reason:       "workflow fields updated",
		Result:       auth.AuditResultSuccess,
		Timestamp:    time.Now().UTC(),
	})
	_ = a.DB.InsertTimelineEvent(db.TimelineEvent{
		EventID:    newTrustID("tl"),
		EventType:  "incident_workflow",
		Summary:    "incident workflow updated: " + incidentID,
		Severity:   "info",
		ActorID:    actorID,
		ResourceID: incidentID,
		Details: map[string]any{
			"review_state": inc.ReviewState,
		},
	})
	return nil
}

func validReviewState(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "open", "acknowledged", "investigating", "mitigated", "resolved", "follow_up_needed",
		"pending_review", "resolved_review", "closed_review":
		return true
	default:
		return false
	}
}

// IncidentReplayView returns a static reconstruction payload for post-incident learning (no simulation).
// When canReadLinked is false (identity lacks read_actions), the nested incident matches GET detail: linked rows omitted and intelligence rebuilt.
func (a *App) IncidentReplayView(incidentID string, canReadLinked bool) (map[string]any, error) {
	if a == nil || a.DB == nil {
		return nil, fmt.Errorf("service not available")
	}
	incidentID = strings.TrimSpace(incidentID)
	if incidentID == "" {
		return nil, fmt.Errorf("incident id is required")
	}
	inc, ok, err := a.IncidentByIDForAPI(incidentID, canReadLinked)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("incident not found: %s", incidentID)
	}
	outcomes, _ := a.DB.RecommendationOutcomesForIncident(incidentID, 100)
	omo := make([]map[string]any, 0, len(outcomes))
	for _, o := range outcomes {
		omo = append(omo, map[string]any{
			"id":                o.ID,
			"recommendation_id": o.RecommendationID,
			"outcome":           o.Outcome,
			"actor_id":          o.ActorID,
			"note":              o.Note,
			"created_at":        o.CreatedAt,
		})
	}
	from, to := incidentEvidenceWindow(inc)
	timeline, _ := a.DB.TimelineEventsForIncidentResource(incidentID, from, to, 200)
	timelineSegs := replaySegmentsFromTimeline(timeline, inc)
	outcomeSegs := replaySegmentsFromRecommendationOutcomes(outcomes, inc)
	segments := mergeReplaySegmentsChronologically(timelineSegs, outcomeSegs)
	knowledge := []map[string]any{}
	for _, seg := range segments {
		knowledge = append(knowledge, map[string]any{
			"event_time":        seg.EventTime,
			"event_type":        seg.EventType,
			"summary":           seg.Summary,
			"knowledge_posture": seg.Posture,
			"evidence_refs":     seg.EvidenceRefs,
			"event_class":       seg.EventClass,
			"actor_id":          seg.ActorID,
			"severity":          seg.Severity,
			"scope_posture":     seg.ScopePosture,
			"timing_posture":    seg.TimingPosture,
			"resource_id":       seg.ResourceID,
		})
	}
	var counterfactual map[string]any
	if inc.Intelligence != nil && len(inc.Intelligence.RunbookRecommendations) > 1 {
		top := inc.Intelligence.RunbookRecommendations[0]
		second := inc.Intelligence.RunbookRecommendations[1]
		counterfactual = map[string]any{
			"statement": "Given today's deterministic rank_score ordering on this incident, top two recommendation ids are ordered as below. This is not a claim about historical operator UI order or outcome optimality.",
			"top":       []map[string]any{{"id": top.ID, "rank_score": top.RankScore, "strength": top.Strength}},
			"second":    []map[string]any{{"id": second.ID, "rank_score": second.RankScore, "strength": second.Strength}},
		}
	}
	replayMeta := map[string]any{
		"schema_version":               "incident_replay_view/v3",
		"window_from":                  from,
		"window_to":                    to,
		"timeline_event_count":         len(timeline),
		"recommendation_outcome_count": len(outcomes),
		"combined_segment_count":       len(segments),
		"ordering":                     "ascending_event_time",
		"sparse_timeline":              len(timeline) == 0,
		"ordering_posture_note":        "Sequence is instance-local persisted time ordering only; imported or federated rows keep their declared timing_posture — not a claim of global causality.",
		"window_truncated":             len(timeline) >= 200,
		"interpretation_posture":       replayInterpretationPosture(len(timeline), len(segments)),
	}
	if !canReadLinked {
		replayMeta["linked_control_redacted"] = true
		replayMeta["visibility_note"] = "Incident object omits FK-linked control rows for this identity (read_actions). Timeline rows are bounded by window and retention; filtered views are not globally representative."
	}
	return map[string]any{
		"kind":                           "incident_replay_view/v3",
		"incident_id":                    inc.ID,
		"incident":                       inc,
		"recommendation_outcomes":        omo,
		"replay_segments":                segments,
		"knowledge_timeline":             knowledge,
		"replay_meta":                    replayMeta,
		"bounded_counterfactual_ranking": counterfactual,
		"truth_note":                     "Derived from persisted rows at query time; not a live simulation. event_class groups rows for filtering; knowledge_posture describes observation vs control-plane vs operator-recorded layers — not root cause.",
		"generated_at":                   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

type replaySegment struct {
	EventTime     string         `json:"event_time"`
	EventType     string         `json:"event_type"`
	EventID       string         `json:"event_id,omitempty"`
	Summary       string         `json:"summary"`
	Posture       string         `json:"knowledge_posture"`
	EventClass    string         `json:"event_class,omitempty"`
	ActorID       string         `json:"actor_id,omitempty"`
	Severity      string         `json:"severity,omitempty"`
	ScopePosture  string         `json:"scope_posture,omitempty"`
	TimingPosture string         `json:"timing_posture,omitempty"`
	ResourceID    string         `json:"resource_id,omitempty"`
	Details       map[string]any `json:"details,omitempty"`
	EvidenceRefs  []string       `json:"evidence_refs,omitempty"`
}

func replayEventClass(eventType string) string {
	switch eventType {
	case "incident":
		return "incident_record"
	case "control_action":
		return "control_action"
	case "operator_note":
		return "operator_annotation"
	case "incident_workflow":
		return "workflow"
	case "incident_handoff":
		return "handoff"
	case "proofpack_export":
		return "evidence_export"
	case "recommendation_outcome":
		return "operator_adjudication"
	case "remote_evidence_import", "remote_evidence_item", "remote_materialized_event":
		return "imported_evidence"
	default:
		if strings.HasPrefix(eventType, "action_") {
			return "control_lifecycle"
		}
		return "timeline_event"
	}
}

func replaySegmentsFromRecommendationOutcomes(rows []db.IncidentRecommendationOutcomeRecord, inc models.Incident) []replaySegment {
	out := make([]replaySegment, 0, len(rows))
	for _, o := range rows {
		if strings.TrimSpace(o.ID) == "" {
			continue
		}
		summary := fmt.Sprintf("Runbook / guidance outcome %q for recommendation %s", strings.TrimSpace(o.Outcome), strings.TrimSpace(o.RecommendationID))
		if strings.TrimSpace(o.Note) != "" {
			summary += " — " + strings.TrimSpace(o.Note)
		}
		out = append(out, replaySegment{
			EventTime:    o.CreatedAt,
			EventType:    "recommendation_outcome",
			EventID:      o.ID,
			Summary:      summary,
			Posture:      "observed_operator_or_system_event",
			EventClass:   "operator_adjudication",
			ActorID:      o.ActorID,
			EvidenceRefs: []string{"recommendation_outcome:" + o.ID, "incident:" + inc.ID},
			ResourceID:   inc.ID,
			Details: map[string]any{
				"recommendation_id": o.RecommendationID,
				"outcome":           o.Outcome,
			},
		})
	}
	return out
}

func mergeReplaySegmentsChronologically(a, b []replaySegment) []replaySegment {
	n := len(a) + len(b)
	if n == 0 {
		return nil
	}
	out := make([]replaySegment, 0, n)
	out = append(out, a...)
	out = append(out, b...)
	sort.SliceStable(out, func(i, j int) bool {
		ti := parseReplayTime(out[i].EventTime)
		tj := parseReplayTime(out[j].EventTime)
		if ti.Equal(tj) {
			if out[i].EventType == out[j].EventType {
				return out[i].EventID < out[j].EventID
			}
			return out[i].EventType < out[j].EventType
		}
		return ti.Before(tj)
	})
	return out
}

// replayInterpretationPosture is a deterministic hint for operators; not a completeness proof.
func replayInterpretationPosture(timelineCount, segmentCount int) string {
	switch {
	case timelineCount >= 200:
		return "timeline_query_capped"
	case timelineCount == 0:
		return "no_timeline_rows_in_window"
	case timelineCount < 3 && segmentCount < 3:
		return "sparse_evidence_window"
	default:
		return "bounded_persistence_view"
	}
}

func parseReplayTime(v string) time.Time {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func replaySegmentsFromTimeline(events []db.TimelineEvent, inc models.Incident) []replaySegment {
	out := make([]replaySegment, 0, len(events))
	for _, ev := range events {
		posture := "observed_persisted_event"
		switch ev.EventType {
		case "proofpack_export", "incident_workflow", "incident_handoff":
			posture = "observed_operator_or_system_event"
		case "control_action":
			posture = "observed_control_plane_event"
		case "operator_note":
			posture = "observed_operator_or_system_event"
		default:
			if strings.HasPrefix(ev.EventType, "action_") {
				posture = "observed_control_lifecycle_event"
			}
		}
		refs := []string{"timeline_event:" + ev.EventID}
		rid := strings.TrimSpace(ev.ResourceID)
		if rid != "" && rid == inc.ID {
			refs = append(refs, "incident:"+inc.ID)
		}
		if ev.EventType == "control_action" && rid != "" && rid != inc.ID {
			refs = append(refs, "control_action:"+ev.EventID)
		}
		details := ev.Details
		if details == nil {
			details = map[string]any{}
		}
		out = append(out, replaySegment{
			EventTime:     ev.EventTime,
			EventType:     ev.EventType,
			EventID:       ev.EventID,
			Summary:       ev.Summary,
			Posture:       posture,
			EventClass:    replayEventClass(ev.EventType),
			ActorID:       ev.ActorID,
			Severity:      ev.Severity,
			ScopePosture:  ev.ScopePosture,
			TimingPosture: ev.TimingPosture,
			ResourceID:    ev.ResourceID,
			Details:       details,
			EvidenceRefs:  refs,
		})
	}
	return out
}

// BuildEscalationBundle returns a concise machine-readable bundle for support handoff.
func (a *App) BuildEscalationBundle(incidentID, actorID string) (map[string]any, error) {
	if a == nil || a.DB == nil {
		return nil, fmt.Errorf("service not available")
	}
	incidentID = strings.TrimSpace(incidentID)
	if incidentID == "" {
		return nil, fmt.Errorf("incident id is required")
	}
	inc, ok, err := a.IncidentByID(incidentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("incident not found: %s", incidentID)
	}
	pack, err := a.AssembleProofpack(incidentID, actorID)
	if err != nil {
		return nil, err
	}
	var gapCount any
	if asm, ok := pack["assembly"].(map[string]any); ok {
		gapCount = asm["evidence_gap_count"]
	}
	narrative := map[string]any{
		"incident_id":   inc.ID,
		"title":         inc.Title,
		"state":         inc.State,
		"review_state":  inc.ReviewState,
		"severity":      inc.Severity,
		"resource":      fmt.Sprintf("%s/%s", inc.ResourceType, inc.ResourceID),
		"summary":       inc.Summary,
		"owner":         inc.OwnerActorID,
		"handoff":       inc.HandoffSummary,
		"investigation": inc.InvestigationNotes,
		"resolution":    inc.ResolutionSummary,
		"closeout":      inc.CloseoutReason,
		"lessons":       inc.LessonsLearned,
	}
	actions, _ := a.DB.ControlActionsByIncidentID(incidentID, 50)
	linked := make([]map[string]any, 0, len(actions))
	for _, act := range actions {
		linked = append(linked, map[string]any{
			"id":               act.ID,
			"action_type":      act.ActionType,
			"lifecycle_state":  act.LifecycleState,
			"target_transport": act.TargetTransport,
			"result":           act.Result,
			"created_at":       act.CreatedAt,
			"reversible":       act.Reversible,
		})
	}
	return map[string]any{
		"kind":                   "escalation_bundle/v1",
		"incident_id":            inc.ID,
		"narrative":              narrative,
		"linked_control_actions": linked,
		"proofpack_summary":      pack["assembly"],
		"section_statuses":       pack["section_statuses"],
		"evidence_gap_count":     gapCount,
		"continuity_note":        "linked_control_actions are incident_id-linked rows only; use proofpack for full evidence chain.",
		"privacy_note":           "Redaction follows platform export policy; safe-share consumers should use redacted export mode when enabled.",
		"generated_at":           time.Now().UTC().Format(time.RFC3339),
	}, nil
}
