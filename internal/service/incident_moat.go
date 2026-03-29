package service

import (
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

	intel.RunbookRecommendations = deriveRunbookRecommendations(intel, inc)
	intel.PolicyGovernanceHints = derivePolicyGovernanceHints(inc)
	intel.DriftFingerprints = a.deriveDriftFingerprints(inc)
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

func deriveRunbookRecommendations(intel *models.IncidentIntelligence, inc models.Incident) []models.IncidentRunbookRecommendation {
	if intel == nil {
		return nil
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
		out = append(out, models.IncidentRunbookRecommendation{
			ID:               id,
			Title:            g.Title,
			Rationale:        g.Rationale,
			EvidenceRefs:     append([]string(nil), g.EvidenceRefs...),
			Strength:         mapGuidanceConfidenceToStrength(g.Confidence),
			RequiresApproval: false,
			Reversibility:    "unknown",
			IsCommand:        false,
		})
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
		strength := "weakly_supported"
		switch mem.EvidenceStrength {
		case "strong":
			strength = "proven_historically"
		case "moderate":
			strength = "plausible"
		case "sparse":
			strength = "weakly_supported"
		}
		if mem.SampleSize < 2 {
			strength = "unsupported"
		}
		reqAppr, blast, rev := actionGovernanceFromIncident(inc, actionType)
		out = append(out, models.IncidentRunbookRecommendation{
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
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func mapGuidanceConfidenceToStrength(c string) string {
	switch strings.ToLower(strings.TrimSpace(c)) {
	case "medium":
		return "plausible"
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
	return &models.IncidentReplayHints{
		Statement:          "Use timeline, linked actions, and proofpack export to reconstruct what MEL had persisted before each operator step.",
		EvidenceAtTimeRefs: refs,
		CounterfactualNote: note,
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
	case "open", "investigating", "pending_review", "resolved_review", "closed_review":
		return true
	default:
		return false
	}
}

// IncidentReplayView returns a static reconstruction payload for post-incident learning (no simulation).
func (a *App) IncidentReplayView(incidentID string) (map[string]any, error) {
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
	return map[string]any{
		"kind":                    "incident_replay_view/v1",
		"incident_id":             inc.ID,
		"incident":                inc,
		"recommendation_outcomes": omo,
		"truth_note":              "Derived from persisted rows at query time; not a live simulation.",
		"generated_at":            time.Now().UTC().Format(time.RFC3339),
	}, nil
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
	return map[string]any{
		"kind":               "escalation_bundle/v1",
		"incident_id":        inc.ID,
		"narrative":          narrative,
		"proofpack_summary":  pack["assembly"],
		"section_statuses":   pack["section_statuses"],
		"evidence_gap_count": gapCount,
		"privacy_note":       "Redaction follows platform export policy; safe-share consumers should use redacted export mode when enabled.",
		"generated_at":       time.Now().UTC().Format(time.RFC3339),
	}, nil
}
