package service

// operator_worklist.go — Deterministic per-operator worklist assembly.
//
// BuildOperatorWorklist composes a bounded, evidence-cited view of what a
// specific operator needs to touch right now. It reads from durable tables
// only — owned incidents, review-state-flagged incidents, pending approvals
// (SoD-aware), decision-pack review backlog, and proposed runbook candidates.
//
// No inference: every list is the projection of rows actually present on disk,
// truncated with explicit disclosure.

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/models"
)

const (
	worklistPerSectionLimit    = 50
	worklistRunbookCandidateCap = 20
	worklistApprovalCap        = 50
	worklistDecisionPackCap    = 50
)

// BuildOperatorWorklist assembles the per-operator worklist from durable rows.
// Pass actor="" to get a team-wide view (owned/handoff sections will be empty
// because they require an explicit actor binding).
func (a *App) BuildOperatorWorklist(actor string) (models.OperatorWorklistDTO, error) {
	if a == nil || a.DB == nil {
		return models.OperatorWorklistDTO{}, fmt.Errorf("service not available")
	}
	actor = strings.TrimSpace(actor)
	out := models.OperatorWorklistDTO{
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		ActorID:       actor,
		EvidenceBasis: []string{"incidents", "control_actions", "incident_decision_pack_adjudication", "incident_runbook_entries"},
	}
	var degraded []string
	var truncation []string

	// Owned open incidents
	if actor != "" {
		owned, err := a.DB.IncidentsOwnedBy(actor, worklistPerSectionLimit)
		if err != nil {
			degraded = append(degraded, "owned_open_incidents: "+err.Error())
		} else {
			out.OwnedOpenIncidents = worklistItemsFromIncidents(owned, "owner_actor_id matches requester")
			if len(owned) == worklistPerSectionLimit {
				truncation = append(truncation, fmt.Sprintf("owned_open_incidents truncated at %d", worklistPerSectionLimit))
			}
		}
	}

	// Review-state-flagged incidents
	pending, err := a.DB.IncidentsByReviewState("pending_review", worklistPerSectionLimit)
	if err != nil {
		degraded = append(degraded, "pending_review: "+err.Error())
	} else {
		out.PendingReview = worklistItemsFromIncidents(pending, "review_state = pending_review")
		if len(pending) == worklistPerSectionLimit {
			truncation = append(truncation, fmt.Sprintf("pending_review truncated at %d", worklistPerSectionLimit))
		}
	}

	followUp, err := a.DB.IncidentsByReviewState("follow_up_needed", worklistPerSectionLimit)
	if err != nil {
		degraded = append(degraded, "follow_up_needed: "+err.Error())
	} else {
		out.FollowUpNeeded = worklistItemsFromIncidents(followUp, "review_state = follow_up_needed")
		if len(followUp) == worklistPerSectionLimit {
			truncation = append(truncation, fmt.Sprintf("follow_up_needed truncated at %d", worklistPerSectionLimit))
		}
	}

	// Pending approvals — SoD-aware. When actor is set we exclude rows where
	// this actor submitted and a separate approver is required.
	approvals, err := a.DB.PendingApprovalControlActions(actor, worklistApprovalCap)
	if err != nil {
		degraded = append(degraded, "pending_approvals: "+err.Error())
	} else {
		out.PendingApprovals = pendingApprovalDTOsFrom(approvals)
		if len(approvals) == worklistApprovalCap {
			truncation = append(truncation, fmt.Sprintf("pending_approvals truncated at %d", worklistApprovalCap))
		}
	}

	// Decision pack review backlog — fetch pending ids, then resolve to compact
	// incident metadata without reloading heavy intelligence.
	ids, err := a.DB.DecisionPackAdjudicationsPending(worklistDecisionPackCap)
	if err != nil {
		degraded = append(degraded, "decision_pack_review: "+err.Error())
	} else if len(ids) > 0 {
		items := make([]models.OperatorWorklistItem, 0, len(ids))
		for _, id := range ids {
			inc, ok, err := a.DB.IncidentByID(id)
			if err != nil || !ok {
				continue
			}
			items = append(items, worklistItemFromIncident(inc, "incident_decision_pack_adjudication.reviewed = 0"))
		}
		out.DecisionPackReview = items
		if len(ids) == worklistDecisionPackCap {
			truncation = append(truncation, fmt.Sprintf("decision_pack_review truncated at %d", worklistDecisionPackCap))
		}
	}

	// Runbook candidates awaiting review (proposed only).
	candidates, err := a.DB.ListRunbookEntries(db.RunbookStatusProposed, "", "", "", worklistRunbookCandidateCap)
	if err != nil {
		degraded = append(degraded, "runbook_candidates: "+err.Error())
	} else {
		out.RunbookCandidates = make([]models.RunbookEntryDTO, 0, len(candidates))
		for _, c := range candidates {
			stats, _ := a.DB.RunbookEntryStatsByID(c.ID)
			out.RunbookCandidates = append(out.RunbookCandidates, runbookEntryDTOFrom(c, stats))
		}
		if len(candidates) == worklistRunbookCandidateCap {
			truncation = append(truncation, fmt.Sprintf("runbook_candidates truncated at %d", worklistRunbookCandidateCap))
		}
	}

	out.Counts = models.OperatorWorklistCounts{
		OwnedOpen:          len(out.OwnedOpenIncidents),
		PendingReview:      len(out.PendingReview),
		FollowUpNeeded:     len(out.FollowUpNeeded),
		PendingApprovals:   len(out.PendingApprovals),
		DecisionPackReview: len(out.DecisionPackReview),
		RunbookCandidates:  len(out.RunbookCandidates),
	}
	out.DegradedSections = degraded
	out.TruncationDisclosure = truncation
	return out, nil
}

// worklistItemFromIncident builds a compact worklist item with a specific
// rationale (why this incident ended up in this section).
func worklistItemFromIncident(inc models.Incident, rationale string) models.OperatorWorklistItem {
	return models.OperatorWorklistItem{
		IncidentID:   inc.ID,
		Title:        inc.Title,
		Severity:     inc.Severity,
		Category:     inc.Category,
		State:        inc.State,
		ReviewState:  inc.ReviewState,
		OwnerActorID: inc.OwnerActorID,
		UpdatedAt:    inc.UpdatedAt,
		OccurredAt:   inc.OccurredAt,
		Rationale:    rationale,
	}
}

func worklistItemsFromIncidents(incs []models.Incident, rationale string) []models.OperatorWorklistItem {
	out := make([]models.OperatorWorklistItem, 0, len(incs))
	for _, inc := range incs {
		out = append(out, worklistItemFromIncident(inc, rationale))
	}
	// Stable ordering: occurred_at DESC then id — helps clients diff snapshots.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].OccurredAt == out[j].OccurredAt {
			return out[i].IncidentID < out[j].IncidentID
		}
		return out[i].OccurredAt > out[j].OccurredAt
	})
	return out
}

func pendingApprovalDTOsFrom(rows []db.ControlActionRecord) []models.OperatorPendingApproval {
	out := make([]models.OperatorPendingApproval, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.OperatorPendingApproval{
			ActionID:                 r.ID,
			ActionType:               r.ActionType,
			TargetTransport:          r.TargetTransport,
			TargetSegment:            r.TargetSegment,
			TargetNode:               r.TargetNode,
			Reason:                   r.Reason,
			Confidence:               r.Confidence,
			BlastRadiusClass:         r.BlastRadiusClass,
			SubmittedBy:              r.SubmittedBy,
			RequiresSeparateApprover: r.RequiresSeparateApprover,
			HighBlastRadius:          r.HighBlastRadius,
			IncidentID:               r.IncidentID,
			CreatedAt:                r.CreatedAt,
		})
	}
	return out
}
