package service

// shift_handoff.go — Deterministic shift handoff packet composer.
//
// BuildShiftHandoffPacket projects a bounded time window of durable rows into a
// single JSON-shaped briefing: incidents opened, incidents resolved, still-open
// incidents owned by the operator, handoffs to this operator in the window,
// pending approvals, runbook candidates created, runbook promotions, and
// runbook applications. Every section cites its table of origin; nothing is
// derived or inferred.

import (
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/models"
)

const (
	shiftHandoffSectionLimit        = 200
	shiftHandoffDefaultWindowHours  = 8
	shiftHandoffMaxWindowHours      = 72
)

// BuildShiftHandoffPacket composes the shift handoff. windowHours defaults to 8
// and is capped at 72. Pass actor="" to get a team-wide view (owned / handoff
// sections will be empty because they require an explicit binding).
func (a *App) BuildShiftHandoffPacket(actor string, windowHours int) (models.ShiftHandoffPacketDTO, error) {
	if a == nil || a.DB == nil {
		return models.ShiftHandoffPacketDTO{}, fmt.Errorf("service not available")
	}
	actor = strings.TrimSpace(actor)
	if windowHours <= 0 {
		windowHours = shiftHandoffDefaultWindowHours
	}
	if windowHours > shiftHandoffMaxWindowHours {
		windowHours = shiftHandoffMaxWindowHours
	}
	start, end := db.ShiftWindowNow(windowHours)

	out := models.ShiftHandoffPacketDTO{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		ActorID:     actor,
		WindowHours: windowHours,
		WindowStart: start,
		WindowEnd:   end,
		EvidenceBasis: []string{
			"incidents.occurred_at",
			"incidents.resolved_at",
			"incidents.owner_actor_id",
			"incidents.handoff_summary",
			"control_actions.lifecycle_state",
			"incident_runbook_entries.created_at",
			"incident_runbook_entries.promoted_at",
			"incident_runbook_applications.created_at",
		},
	}
	var degraded []string
	var truncation []string

	opened, err := a.DB.IncidentsOpenedInWindow(start, end, shiftHandoffSectionLimit)
	if err != nil {
		degraded = append(degraded, "opened_incidents: "+err.Error())
	} else {
		out.OpenedIncidents = worklistItemsFromIncidents(opened, "incidents.occurred_at in window")
		if len(opened) == shiftHandoffSectionLimit {
			truncation = append(truncation, fmt.Sprintf("opened_incidents truncated at %d", shiftHandoffSectionLimit))
		}
	}

	resolved, err := a.DB.IncidentsResolvedInWindow(start, end, shiftHandoffSectionLimit)
	if err != nil {
		degraded = append(degraded, "resolved_incidents: "+err.Error())
	} else {
		out.ResolvedIncidents = worklistItemsFromIncidents(resolved, "incidents.resolved_at in window")
		if len(resolved) == shiftHandoffSectionLimit {
			truncation = append(truncation, fmt.Sprintf("resolved_incidents truncated at %d", shiftHandoffSectionLimit))
		}
	}

	if actor != "" {
		stillOpen, err := a.DB.IncidentsOwnedBy(actor, shiftHandoffSectionLimit)
		if err != nil {
			degraded = append(degraded, "still_open_owned: "+err.Error())
		} else {
			out.StillOpenOwned = worklistItemsFromIncidents(stillOpen, "owner_actor_id matches requester, state not resolved/closed")
			if len(stillOpen) == shiftHandoffSectionLimit {
				truncation = append(truncation, fmt.Sprintf("still_open_owned truncated at %d", shiftHandoffSectionLimit))
			}
		}

		handoffs, err := a.DB.IncidentsHandedOffToInWindow(actor, start, end, shiftHandoffSectionLimit)
		if err != nil {
			degraded = append(degraded, "handoffs_to_me: "+err.Error())
		} else {
			out.HandoffsToMe = worklistItemsFromIncidents(handoffs, "handoff_summary set and updated_at in window, owner_actor_id matches requester")
			if len(handoffs) == shiftHandoffSectionLimit {
				truncation = append(truncation, fmt.Sprintf("handoffs_to_me truncated at %d", shiftHandoffSectionLimit))
			}
		}
	}

	approvals, err := a.DB.PendingApprovalControlActions(actor, shiftHandoffSectionLimit)
	if err != nil {
		degraded = append(degraded, "pending_approvals: "+err.Error())
	} else {
		out.PendingApprovals = pendingApprovalDTOsFrom(approvals)
		if len(approvals) == shiftHandoffSectionLimit {
			truncation = append(truncation, fmt.Sprintf("pending_approvals truncated at %d", shiftHandoffSectionLimit))
		}
	}

	candidates, err := a.DB.RunbookCandidatesCreatedInWindow(start, end, shiftHandoffSectionLimit)
	if err != nil {
		degraded = append(degraded, "runbook_candidates_created: "+err.Error())
	} else {
		out.RunbookCandidates = make([]models.RunbookEntryDTO, 0, len(candidates))
		for _, c := range candidates {
			stats, _ := a.DB.RunbookEntryStatsByID(c.ID)
			out.RunbookCandidates = append(out.RunbookCandidates, runbookEntryDTOFrom(c, stats))
		}
		if len(candidates) == shiftHandoffSectionLimit {
			truncation = append(truncation, fmt.Sprintf("runbook_candidates_created truncated at %d", shiftHandoffSectionLimit))
		}
	}

	promos, err := a.DB.RunbookEntriesPromotedInWindow(start, end, shiftHandoffSectionLimit)
	if err != nil {
		degraded = append(degraded, "runbook_promotions: "+err.Error())
	} else {
		out.RunbookPromotions = make([]models.RunbookEntryDTO, 0, len(promos))
		for _, p := range promos {
			stats, _ := a.DB.RunbookEntryStatsByID(p.ID)
			out.RunbookPromotions = append(out.RunbookPromotions, runbookEntryDTOFrom(p, stats))
		}
		if len(promos) == shiftHandoffSectionLimit {
			truncation = append(truncation, fmt.Sprintf("runbook_promotions truncated at %d", shiftHandoffSectionLimit))
		}
	}

	applications, err := a.DB.RunbookApplicationsInWindow(start, end, shiftHandoffSectionLimit)
	if err != nil {
		degraded = append(degraded, "runbook_applications: "+err.Error())
	} else {
		out.RunbookApplications = runbookApplicationDTOsFrom(applications)
		if len(applications) == shiftHandoffSectionLimit {
			truncation = append(truncation, fmt.Sprintf("runbook_applications truncated at %d", shiftHandoffSectionLimit))
		}
	}

	out.Counts = models.ShiftHandoffCounts{
		OpenedIncidents:     len(out.OpenedIncidents),
		ResolvedIncidents:   len(out.ResolvedIncidents),
		StillOpenOwned:      len(out.StillOpenOwned),
		HandoffsToMe:        len(out.HandoffsToMe),
		PendingApprovals:    len(out.PendingApprovals),
		RunbookCandidates:   len(out.RunbookCandidates),
		RunbookPromotions:   len(out.RunbookPromotions),
		RunbookApplications: len(out.RunbookApplications),
	}
	out.DegradedSections = degraded
	out.TruncationDisclosure = truncation
	return out, nil
}
