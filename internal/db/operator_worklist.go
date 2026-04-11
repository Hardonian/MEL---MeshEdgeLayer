package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/models"
)

// operator_worklist.go — deterministic repository queries that back the
// per-operator worklist, shift handoff packet, and runbook candidate review
// surfaces. These queries never invent state: every row shown is a row on disk
// in this instance's SQLite database.

// IncidentsOwnedBy returns open incidents where owner_actor_id = actor.
// "Open" means state NOT IN ('resolved','closed') — review_state is orthogonal.
func (d *DB) IncidentsOwnedBy(actor string, limit int) ([]models.Incident, error) {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return nil, nil
	}
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id, category, severity, title, summary, resource_type, resource_id, state, COALESCE(actor_id,'') AS actor_id, occurred_at, updated_at, COALESCE(resolved_at,'') AS resolved_at, COALESCE(metadata_json,'{}') AS metadata_json,
			COALESCE(owner_actor_id,'') AS owner_actor_id, COALESCE(handoff_summary,'') AS handoff_summary,
			COALESCE(pending_actions_json,'[]') AS pending_actions_json, COALESCE(recent_actions_json,'[]') AS recent_actions_json,
			COALESCE(linked_evidence_json,'[]') AS linked_evidence_json, COALESCE(risks_json,'[]') AS risks_json,
			COALESCE(review_state,'open') AS review_state, COALESCE(investigation_notes,'') AS investigation_notes, COALESCE(resolution_summary,'') AS resolution_summary,
			COALESCE(closeout_reason,'') AS closeout_reason, COALESCE(lessons_learned,'') AS lessons_learned,
			COALESCE(reopened_from_incident_id,'') AS reopened_from_incident_id, COALESCE(reopened_at,'') AS reopened_at
FROM incidents WHERE LOWER(state) NOT IN ('resolved','closed') AND owner_actor_id='%s' ORDER BY occurred_at DESC LIMIT %d;`,
		esc(actor), limit))
	if err != nil {
		return nil, err
	}
	out := make([]models.Incident, 0, len(rows))
	for _, row := range rows {
		out = append(out, incidentFromRow(row))
	}
	return out, nil
}

// IncidentsByReviewState returns open incidents whose review_state matches the
// supplied state (e.g. "follow_up_needed", "pending_review"). Returns ordered
// by last updated.
func (d *DB) IncidentsByReviewState(reviewState string, limit int) ([]models.Incident, error) {
	reviewState = strings.ToLower(strings.TrimSpace(reviewState))
	if reviewState == "" {
		return nil, nil
	}
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id, category, severity, title, summary, resource_type, resource_id, state, COALESCE(actor_id,'') AS actor_id, occurred_at, updated_at, COALESCE(resolved_at,'') AS resolved_at, COALESCE(metadata_json,'{}') AS metadata_json,
			COALESCE(owner_actor_id,'') AS owner_actor_id, COALESCE(handoff_summary,'') AS handoff_summary,
			COALESCE(pending_actions_json,'[]') AS pending_actions_json, COALESCE(recent_actions_json,'[]') AS recent_actions_json,
			COALESCE(linked_evidence_json,'[]') AS linked_evidence_json, COALESCE(risks_json,'[]') AS risks_json,
			COALESCE(review_state,'open') AS review_state, COALESCE(investigation_notes,'') AS investigation_notes, COALESCE(resolution_summary,'') AS resolution_summary,
			COALESCE(closeout_reason,'') AS closeout_reason, COALESCE(lessons_learned,'') AS lessons_learned,
			COALESCE(reopened_from_incident_id,'') AS reopened_from_incident_id, COALESCE(reopened_at,'') AS reopened_at
FROM incidents WHERE LOWER(state) NOT IN ('resolved','closed') AND LOWER(COALESCE(review_state,'open'))='%s' ORDER BY updated_at DESC LIMIT %d;`,
		esc(reviewState), limit))
	if err != nil {
		return nil, err
	}
	out := make([]models.Incident, 0, len(rows))
	for _, row := range rows {
		out = append(out, incidentFromRow(row))
	}
	return out, nil
}

// OpenIncidentsForQueue returns open incidents in occurred_at DESC order,
// suitable for triage tier ranking at the service layer. review_state is
// included so the service can surface "pending_review" etc.
func (d *DB) OpenIncidentsForQueue(limit int) ([]models.Incident, error) {
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id, category, severity, title, summary, resource_type, resource_id, state, COALESCE(actor_id,'') AS actor_id, occurred_at, updated_at, COALESCE(resolved_at,'') AS resolved_at, COALESCE(metadata_json,'{}') AS metadata_json,
			COALESCE(owner_actor_id,'') AS owner_actor_id, COALESCE(handoff_summary,'') AS handoff_summary,
			COALESCE(pending_actions_json,'[]') AS pending_actions_json, COALESCE(recent_actions_json,'[]') AS recent_actions_json,
			COALESCE(linked_evidence_json,'[]') AS linked_evidence_json, COALESCE(risks_json,'[]') AS risks_json,
			COALESCE(review_state,'open') AS review_state, COALESCE(investigation_notes,'') AS investigation_notes, COALESCE(resolution_summary,'') AS resolution_summary,
			COALESCE(closeout_reason,'') AS closeout_reason, COALESCE(lessons_learned,'') AS lessons_learned,
			COALESCE(reopened_from_incident_id,'') AS reopened_from_incident_id, COALESCE(reopened_at,'') AS reopened_at
FROM incidents WHERE LOWER(state) NOT IN ('resolved','closed') ORDER BY occurred_at DESC LIMIT %d;`, limit))
	if err != nil {
		return nil, err
	}
	out := make([]models.Incident, 0, len(rows))
	for _, row := range rows {
		out = append(out, incidentFromRow(row))
	}
	return out, nil
}

// IncidentsOpenedInWindow returns incidents with occurred_at in [start,end] (inclusive UTC RFC3339).
func (d *DB) IncidentsOpenedInWindow(startRFC3339, endRFC3339 string, limit int) ([]models.Incident, error) {
	start := strings.TrimSpace(startRFC3339)
	end := strings.TrimSpace(endRFC3339)
	if start == "" || end == "" {
		return nil, fmt.Errorf("window start and end are required")
	}
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id, category, severity, title, summary, resource_type, resource_id, state, COALESCE(actor_id,'') AS actor_id, occurred_at, updated_at, COALESCE(resolved_at,'') AS resolved_at, COALESCE(metadata_json,'{}') AS metadata_json,
			COALESCE(owner_actor_id,'') AS owner_actor_id, COALESCE(handoff_summary,'') AS handoff_summary,
			COALESCE(pending_actions_json,'[]') AS pending_actions_json, COALESCE(recent_actions_json,'[]') AS recent_actions_json,
			COALESCE(linked_evidence_json,'[]') AS linked_evidence_json, COALESCE(risks_json,'[]') AS risks_json,
			COALESCE(review_state,'open') AS review_state, COALESCE(investigation_notes,'') AS investigation_notes, COALESCE(resolution_summary,'') AS resolution_summary,
			COALESCE(closeout_reason,'') AS closeout_reason, COALESCE(lessons_learned,'') AS lessons_learned,
			COALESCE(reopened_from_incident_id,'') AS reopened_from_incident_id, COALESCE(reopened_at,'') AS reopened_at
FROM incidents WHERE occurred_at >= '%s' AND occurred_at <= '%s' ORDER BY occurred_at ASC LIMIT %d;`,
		esc(start), esc(end), limit))
	if err != nil {
		return nil, err
	}
	out := make([]models.Incident, 0, len(rows))
	for _, row := range rows {
		out = append(out, incidentFromRow(row))
	}
	return out, nil
}

// IncidentsResolvedInWindow returns incidents whose resolved_at falls in [start,end].
func (d *DB) IncidentsResolvedInWindow(startRFC3339, endRFC3339 string, limit int) ([]models.Incident, error) {
	start := strings.TrimSpace(startRFC3339)
	end := strings.TrimSpace(endRFC3339)
	if start == "" || end == "" {
		return nil, fmt.Errorf("window start and end are required")
	}
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id, category, severity, title, summary, resource_type, resource_id, state, COALESCE(actor_id,'') AS actor_id, occurred_at, updated_at, COALESCE(resolved_at,'') AS resolved_at, COALESCE(metadata_json,'{}') AS metadata_json,
			COALESCE(owner_actor_id,'') AS owner_actor_id, COALESCE(handoff_summary,'') AS handoff_summary,
			COALESCE(pending_actions_json,'[]') AS pending_actions_json, COALESCE(recent_actions_json,'[]') AS recent_actions_json,
			COALESCE(linked_evidence_json,'[]') AS linked_evidence_json, COALESCE(risks_json,'[]') AS risks_json,
			COALESCE(review_state,'open') AS review_state, COALESCE(investigation_notes,'') AS investigation_notes, COALESCE(resolution_summary,'') AS resolution_summary,
			COALESCE(closeout_reason,'') AS closeout_reason, COALESCE(lessons_learned,'') AS lessons_learned,
			COALESCE(reopened_from_incident_id,'') AS reopened_from_incident_id, COALESCE(reopened_at,'') AS reopened_at
FROM incidents WHERE resolved_at IS NOT NULL AND resolved_at != '' AND resolved_at >= '%s' AND resolved_at <= '%s' ORDER BY resolved_at ASC LIMIT %d;`,
		esc(start), esc(end), limit))
	if err != nil {
		return nil, err
	}
	out := make([]models.Incident, 0, len(rows))
	for _, row := range rows {
		out = append(out, incidentFromRow(row))
	}
	return out, nil
}

// PendingApprovalControlActions returns control actions currently awaiting
// approval. When excludeSubmitter != "", rows where submitted_by matches that
// actor AND requires_separate_approver = 1 are filtered out (SoD self-approval
// is disallowed). This is surfaced on the per-operator worklist.
func (d *DB) PendingApprovalControlActions(excludeSubmitter string, limit int) ([]ControlActionRecord, error) {
	limit = clampLimit(limit)
	clauses := []string{"lifecycle_state='pending_approval'"}
	if actor := strings.TrimSpace(excludeSubmitter); actor != "" {
		clauses = append(clauses, fmt.Sprintf("NOT (COALESCE(requires_separate_approver,0)=1 AND COALESCE(submitted_by,'')='%s')", esc(actor)))
	}
	sql := fmt.Sprintf(`SELECT %s FROM control_actions WHERE %s ORDER BY created_at ASC LIMIT %d;`,
		sqlControlActionSelectList, strings.Join(clauses, " AND "), limit)
	rows, err := d.QueryRows(sql)
	if err != nil {
		return nil, err
	}
	out := make([]ControlActionRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, controlActionFromRow(row))
	}
	return out, nil
}

// ControlActionsCreatedInWindow returns all control actions whose created_at
// is in [start,end], ordered ascending.
func (d *DB) ControlActionsCreatedInWindow(startRFC3339, endRFC3339 string, limit int) ([]ControlActionRecord, error) {
	start := strings.TrimSpace(startRFC3339)
	end := strings.TrimSpace(endRFC3339)
	if start == "" || end == "" {
		return nil, fmt.Errorf("window start and end are required")
	}
	limit = clampLimit(limit)
	sql := fmt.Sprintf(`SELECT %s FROM control_actions WHERE created_at >= '%s' AND created_at <= '%s' ORDER BY created_at ASC LIMIT %d;`,
		sqlControlActionSelectList, esc(start), esc(end), limit)
	rows, err := d.QueryRows(sql)
	if err != nil {
		return nil, err
	}
	out := make([]ControlActionRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, controlActionFromRow(row))
	}
	return out, nil
}

// IncidentsHandedOffToInWindow returns incidents where handoff_summary was set
// and updated_at is in the window AND owner_actor_id = actor. This is a bounded
// view of "recent handoffs to me" — the handoff trail lives in timeline_events.
func (d *DB) IncidentsHandedOffToInWindow(actor, startRFC3339, endRFC3339 string, limit int) ([]models.Incident, error) {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return nil, nil
	}
	start := strings.TrimSpace(startRFC3339)
	end := strings.TrimSpace(endRFC3339)
	if start == "" || end == "" {
		return nil, fmt.Errorf("window start and end are required")
	}
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id, category, severity, title, summary, resource_type, resource_id, state, COALESCE(actor_id,'') AS actor_id, occurred_at, updated_at, COALESCE(resolved_at,'') AS resolved_at, COALESCE(metadata_json,'{}') AS metadata_json,
			COALESCE(owner_actor_id,'') AS owner_actor_id, COALESCE(handoff_summary,'') AS handoff_summary,
			COALESCE(pending_actions_json,'[]') AS pending_actions_json, COALESCE(recent_actions_json,'[]') AS recent_actions_json,
			COALESCE(linked_evidence_json,'[]') AS linked_evidence_json, COALESCE(risks_json,'[]') AS risks_json,
			COALESCE(review_state,'open') AS review_state, COALESCE(investigation_notes,'') AS investigation_notes, COALESCE(resolution_summary,'') AS resolution_summary,
			COALESCE(closeout_reason,'') AS closeout_reason, COALESCE(lessons_learned,'') AS lessons_learned,
			COALESCE(reopened_from_incident_id,'') AS reopened_from_incident_id, COALESCE(reopened_at,'') AS reopened_at
FROM incidents WHERE owner_actor_id='%s' AND handoff_summary != '' AND updated_at >= '%s' AND updated_at <= '%s' ORDER BY updated_at DESC LIMIT %d;`,
		esc(actor), esc(start), esc(end), limit))
	if err != nil {
		return nil, err
	}
	out := make([]models.Incident, 0, len(rows))
	for _, row := range rows {
		out = append(out, incidentFromRow(row))
	}
	return out, nil
}

// RunbookCandidatesCreatedInWindow returns proposed runbook candidates with
// created_at in [start,end]. Used in shift handoff to surface what MEL proposed
// during the shift.
func (d *DB) RunbookCandidatesCreatedInWindow(startRFC3339, endRFC3339 string, limit int) ([]RunbookEntryRecord, error) {
	start := strings.TrimSpace(startRFC3339)
	end := strings.TrimSpace(endRFC3339)
	if start == "" || end == "" {
		return nil, fmt.Errorf("window start and end are required")
	}
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id,status,source_kind,legacy_signature_key,fingerprint_canonical_hash,title,body,evidence_ref_json,source_incident_ids_json,promotion_basis,created_at,updated_at,reviewed_at,reviewer_actor_id
FROM incident_runbook_entries WHERE status='proposed' AND created_at >= '%s' AND created_at <= '%s' ORDER BY created_at DESC LIMIT %d;`,
		esc(start), esc(end), limit))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil, nil
		}
		return nil, err
	}
	return runbookRows(rows), nil
}

// RunbookEntriesPromotedInWindow returns runbook rows promoted in [start,end].
func (d *DB) RunbookEntriesPromotedInWindow(startRFC3339, endRFC3339 string, limit int) ([]RunbookEntryRecord, error) {
	start := strings.TrimSpace(startRFC3339)
	end := strings.TrimSpace(endRFC3339)
	if start == "" || end == "" {
		return nil, fmt.Errorf("window start and end are required")
	}
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id,status,source_kind,legacy_signature_key,fingerprint_canonical_hash,title,body,evidence_ref_json,source_incident_ids_json,promotion_basis,created_at,updated_at,reviewed_at,reviewer_actor_id
FROM incident_runbook_entries WHERE status='promoted' AND promoted_at >= '%s' AND promoted_at <= '%s' ORDER BY promoted_at DESC LIMIT %d;`,
		esc(start), esc(end), limit))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") || strings.Contains(err.Error(), "no such column") {
			return nil, nil
		}
		return nil, err
	}
	return runbookRows(rows), nil
}

// RunbookApplicationsInWindow returns runbook applications attached to any
// incident during [start,end]. Surfaced on the shift handoff.
func (d *DB) RunbookApplicationsInWindow(startRFC3339, endRFC3339 string, limit int) ([]RunbookApplicationRecord, error) {
	start := strings.TrimSpace(startRFC3339)
	end := strings.TrimSpace(endRFC3339)
	if start == "" || end == "" {
		return nil, fmt.Errorf("window start and end are required")
	}
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id,runbook_id,incident_id,COALESCE(actor_id,'system') AS actor_id,outcome,COALESCE(note,'') AS note,created_at
FROM incident_runbook_applications WHERE created_at >= '%s' AND created_at <= '%s' ORDER BY created_at ASC LIMIT %d;`,
		esc(start), esc(end), limit))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil, nil
		}
		return nil, err
	}
	return runbookApplicationRows(rows), nil
}

// DecisionPackAdjudicationsPendingByActor returns the incident IDs whose
// decision pack has been opened (adjudication row exists) but reviewed=0,
// optionally filtered to rows where reviewed_by_actor_id is blank or matches
// the actor. Used in the worklist.
func (d *DB) DecisionPackAdjudicationsPending(limit int) ([]string, error) {
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT incident_id FROM incident_decision_pack_adjudication WHERE COALESCE(reviewed,0)=0 ORDER BY updated_at DESC LIMIT %d;`, limit))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil, nil
		}
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		id := strings.TrimSpace(asString(row["incident_id"]))
		if id != "" {
			out = append(out, id)
		}
	}
	return out, nil
}

// ShiftWindowNow returns [now-windowHours, now] in UTC RFC3339. Defaults to
// 8h when windowHours <= 0. Capped at 72h to keep responses bounded.
func ShiftWindowNow(windowHours int) (string, string) {
	if windowHours <= 0 {
		windowHours = 8
	}
	if windowHours > 72 {
		windowHours = 72
	}
	now := time.Now().UTC()
	start := now.Add(-time.Duration(windowHours) * time.Hour)
	return start.Format(time.RFC3339), now.Format(time.RFC3339)
}
