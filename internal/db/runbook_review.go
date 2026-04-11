package db

import (
	"fmt"
	"strings"
	"time"
)

// runbook_review.go — DB access for reviewing, promoting, deprecating and applying
// incident runbook entries (migration 0037). Every read is safe-fail if the
// underlying table is missing (pre-migration), every write returns nil on
// "no such table" so callers do not need to guard on migration version.

// Valid runbook lifecycle states persisted on incident_runbook_entries.status.
const (
	RunbookStatusProposed   = "proposed"
	RunbookStatusReviewing  = "reviewing"
	RunbookStatusPromoted   = "promoted"
	RunbookStatusDeprecated = "deprecated"
)

// ValidRunbookStatus returns true when s is a known runbook lifecycle state.
func ValidRunbookStatus(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case RunbookStatusProposed, RunbookStatusReviewing, RunbookStatusPromoted, RunbookStatusDeprecated:
		return true
	}
	return false
}

// RunbookApplicationRecord is one operator attaching a runbook to an incident
// with an explicit outcome. Rows are durable audit truth.
type RunbookApplicationRecord struct {
	ID         string
	RunbookID  string
	IncidentID string
	ActorID    string
	Outcome    string
	Note       string
	CreatedAt  string
}

// Valid runbook application outcomes. "applied" just records that an operator
// used the runbook; "helped" / "did_not_help" / "worsened" are effectiveness
// signals; "superseded" means a newer runbook replaced it mid-incident.
const (
	RunbookOutcomeApplied    = "applied"
	RunbookOutcomeHelped     = "helped"
	RunbookOutcomeDidNotHelp = "did_not_help"
	RunbookOutcomeWorsened   = "worsened"
	RunbookOutcomeSuperseded = "superseded"
)

// ValidRunbookOutcome returns true when s is a known runbook application outcome.
func ValidRunbookOutcome(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case RunbookOutcomeApplied, RunbookOutcomeHelped, RunbookOutcomeDidNotHelp,
		RunbookOutcomeWorsened, RunbookOutcomeSuperseded:
		return true
	}
	return false
}

// RunbookEntryByID returns the runbook row by id.
func (d *DB) RunbookEntryByID(id string) (RunbookEntryRecord, bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return RunbookEntryRecord{}, false, nil
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id,status,source_kind,legacy_signature_key,fingerprint_canonical_hash,title,body,evidence_ref_json,source_incident_ids_json,promotion_basis,created_at,updated_at,reviewed_at,reviewer_actor_id
FROM incident_runbook_entries WHERE id='%s' LIMIT 1;`, esc(id)))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return RunbookEntryRecord{}, false, nil
		}
		return RunbookEntryRecord{}, false, err
	}
	if len(rows) == 0 {
		return RunbookEntryRecord{}, false, nil
	}
	return runbookRows(rows)[0], true, nil
}

// ListRunbookEntries returns runbook entries filtered by status, signature key,
// fingerprint hash, and a free-text title/body match. Empty filters match all.
// Ordering: most recently updated first. Deterministic.
func (d *DB) ListRunbookEntries(status, signatureKey, fingerprintHash, query string, limit int) ([]RunbookEntryRecord, error) {
	limit = clampLimit(limit)
	clauses := []string{"1=1"}
	if s := strings.TrimSpace(status); s != "" && ValidRunbookStatus(s) {
		clauses = append(clauses, fmt.Sprintf("status='%s'", esc(strings.ToLower(s))))
	}
	if k := strings.TrimSpace(signatureKey); k != "" {
		clauses = append(clauses, fmt.Sprintf("legacy_signature_key='%s'", esc(k)))
	}
	if h := strings.TrimSpace(fingerprintHash); h != "" {
		clauses = append(clauses, fmt.Sprintf("fingerprint_canonical_hash='%s'", esc(h)))
	}
	if q := strings.TrimSpace(query); q != "" {
		// bounded LIKE on title/body; callers must pass already-sanitized input.
		clauses = append(clauses, fmt.Sprintf("(title LIKE '%%%s%%' OR body LIKE '%%%s%%')", esc(q), esc(q)))
	}
	sql := fmt.Sprintf(`SELECT id,status,source_kind,legacy_signature_key,fingerprint_canonical_hash,title,body,evidence_ref_json,source_incident_ids_json,promotion_basis,created_at,updated_at,reviewed_at,reviewer_actor_id
FROM incident_runbook_entries WHERE %s ORDER BY updated_at DESC LIMIT %d;`, strings.Join(clauses, " AND "), limit)
	rows, err := d.QueryRows(sql)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil, nil
		}
		return nil, err
	}
	return runbookRows(rows), nil
}

// RunbookEntryStats is the bounded effectiveness/application snapshot surfaced
// alongside a runbook detail.
type RunbookEntryStats struct {
	AppliedCount          int
	UsefulCount           int
	IneffectiveCount      int
	LastAppliedAt         string
	LastAppliedIncidentID string
	PromotedAt            string
	PromotedByActorID     string
	DeprecatedAt          string
	DeprecatedByActorID   string
	DeprecatedReason      string
}

// RunbookEntryStatsByID returns the durable counters and lifecycle timestamps
// for a runbook. Returns zero-value stats if the table is pre-migration.
func (d *DB) RunbookEntryStatsByID(id string) (RunbookEntryStats, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return RunbookEntryStats{}, nil
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT
  COALESCE(applied_count,0) AS applied_count,
  COALESCE(useful_count,0) AS useful_count,
  COALESCE(ineffective_count,0) AS ineffective_count,
  COALESCE(last_applied_at,'') AS last_applied_at,
  COALESCE(last_applied_incident_id,'') AS last_applied_incident_id,
  COALESCE(promoted_at,'') AS promoted_at,
  COALESCE(promoted_by_actor_id,'') AS promoted_by_actor_id,
  COALESCE(deprecated_at,'') AS deprecated_at,
  COALESCE(deprecated_by_actor_id,'') AS deprecated_by_actor_id,
  COALESCE(deprecated_reason,'') AS deprecated_reason
FROM incident_runbook_entries WHERE id='%s' LIMIT 1;`, esc(id)))
	if err != nil {
		if strings.Contains(err.Error(), "no such column") || strings.Contains(err.Error(), "no such table") {
			return RunbookEntryStats{}, nil
		}
		return RunbookEntryStats{}, err
	}
	if len(rows) == 0 {
		return RunbookEntryStats{}, nil
	}
	row := rows[0]
	return RunbookEntryStats{
		AppliedCount:          int(asInt(row["applied_count"])),
		UsefulCount:           int(asInt(row["useful_count"])),
		IneffectiveCount:      int(asInt(row["ineffective_count"])),
		LastAppliedAt:         asString(row["last_applied_at"]),
		LastAppliedIncidentID: asString(row["last_applied_incident_id"]),
		PromotedAt:            asString(row["promoted_at"]),
		PromotedByActorID:     asString(row["promoted_by_actor_id"]),
		DeprecatedAt:          asString(row["deprecated_at"]),
		DeprecatedByActorID:   asString(row["deprecated_by_actor_id"]),
		DeprecatedReason:      asString(row["deprecated_reason"]),
	}, nil
}

// SetRunbookStatus transitions a runbook to promoted or deprecated (or back to
// reviewing). Records actor and timestamp for the matching lifecycle column.
// Returns false if the row was not found.
func (d *DB) SetRunbookStatus(id, status, actorID, reason string) (bool, error) {
	id = strings.TrimSpace(id)
	status = strings.ToLower(strings.TrimSpace(status))
	if id == "" {
		return false, fmt.Errorf("runbook id required")
	}
	if !ValidRunbookStatus(status) {
		return false, fmt.Errorf("invalid runbook status %q", status)
	}
	// confirm the row exists to distinguish not-found vs. success
	_, ok, err := d.RunbookEntryByID(id)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	if strings.TrimSpace(actorID) == "" {
		actorID = "system"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	sets := []string{
		fmt.Sprintf("status='%s'", esc(status)),
		fmt.Sprintf("updated_at='%s'", esc(now)),
		fmt.Sprintf("reviewed_at='%s'", esc(now)),
		fmt.Sprintf("reviewer_actor_id='%s'", esc(actorID)),
	}
	switch status {
	case RunbookStatusPromoted:
		sets = append(sets,
			fmt.Sprintf("promoted_at='%s'", esc(now)),
			fmt.Sprintf("promoted_by_actor_id='%s'", esc(actorID)),
		)
	case RunbookStatusDeprecated:
		sets = append(sets,
			fmt.Sprintf("deprecated_at='%s'", esc(now)),
			fmt.Sprintf("deprecated_by_actor_id='%s'", esc(actorID)),
			fmt.Sprintf("deprecated_reason='%s'", esc(strings.TrimSpace(reason))),
		)
	}
	sql := fmt.Sprintf(`UPDATE incident_runbook_entries SET %s WHERE id='%s';`, strings.Join(sets, ","), esc(id))
	if err := d.Exec(sql); err != nil {
		if strings.Contains(err.Error(), "no such column") || strings.Contains(err.Error(), "no such table") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// InsertRunbookApplication persists a runbook application row and updates the
// parent runbook's bounded counters. Caller provides an actor id (must be a
// real operator or "system"). When outcome is empty, defaults to "applied".
func (d *DB) InsertRunbookApplication(rec RunbookApplicationRecord) error {
	if strings.TrimSpace(rec.ID) == "" {
		return fmt.Errorf("application id required")
	}
	if strings.TrimSpace(rec.RunbookID) == "" || strings.TrimSpace(rec.IncidentID) == "" {
		return fmt.Errorf("runbook_id and incident_id are required")
	}
	outcome := strings.ToLower(strings.TrimSpace(rec.Outcome))
	if outcome == "" {
		outcome = RunbookOutcomeApplied
	}
	if !ValidRunbookOutcome(outcome) {
		return fmt.Errorf("invalid outcome %q", rec.Outcome)
	}
	rec.Outcome = outcome
	if strings.TrimSpace(rec.ActorID) == "" {
		rec.ActorID = "system"
	}
	if rec.CreatedAt == "" {
		rec.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	insertSQL := fmt.Sprintf(`INSERT INTO incident_runbook_applications(id,runbook_id,incident_id,actor_id,outcome,note,created_at)
VALUES('%s','%s','%s','%s','%s','%s','%s');`,
		esc(rec.ID), esc(rec.RunbookID), esc(rec.IncidentID), esc(rec.ActorID), esc(rec.Outcome), esc(rec.Note), esc(rec.CreatedAt))
	if err := d.Exec(insertSQL); err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil
		}
		return err
	}
	// update parent counters on incident_runbook_entries (bounded accumulators)
	var extra string
	switch rec.Outcome {
	case RunbookOutcomeHelped:
		extra = ", useful_count = COALESCE(useful_count,0) + 1"
	case RunbookOutcomeDidNotHelp, RunbookOutcomeWorsened:
		extra = ", ineffective_count = COALESCE(ineffective_count,0) + 1"
	}
	upd := fmt.Sprintf(`UPDATE incident_runbook_entries SET
  applied_count = COALESCE(applied_count,0) + 1,
  last_applied_at = '%s',
  last_applied_incident_id = '%s',
  updated_at = '%s'%s
WHERE id='%s';`,
		esc(rec.CreatedAt), esc(rec.IncidentID), esc(rec.CreatedAt), extra, esc(rec.RunbookID))
	if err := d.Exec(upd); err != nil {
		if strings.Contains(err.Error(), "no such column") || strings.Contains(err.Error(), "no such table") {
			return nil
		}
		return err
	}
	return nil
}

// RunbookApplicationsForRunbook returns the most recent applications for a runbook.
func (d *DB) RunbookApplicationsForRunbook(runbookID string, limit int) ([]RunbookApplicationRecord, error) {
	runbookID = strings.TrimSpace(runbookID)
	if runbookID == "" {
		return nil, nil
	}
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id,runbook_id,incident_id,COALESCE(actor_id,'system') AS actor_id,outcome,COALESCE(note,'') AS note,created_at
FROM incident_runbook_applications WHERE runbook_id='%s' ORDER BY created_at DESC LIMIT %d;`, esc(runbookID), limit))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil, nil
		}
		return nil, err
	}
	return runbookApplicationRows(rows), nil
}

// RunbookApplicationsForIncident returns applications attached to an incident
// (e.g. to render "runbooks operators applied on this incident" on the detail page).
func (d *DB) RunbookApplicationsForIncident(incidentID string, limit int) ([]RunbookApplicationRecord, error) {
	incidentID = strings.TrimSpace(incidentID)
	if incidentID == "" {
		return nil, nil
	}
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id,runbook_id,incident_id,COALESCE(actor_id,'system') AS actor_id,outcome,COALESCE(note,'') AS note,created_at
FROM incident_runbook_applications WHERE incident_id='%s' ORDER BY created_at DESC LIMIT %d;`, esc(incidentID), limit))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil, nil
		}
		return nil, err
	}
	return runbookApplicationRows(rows), nil
}

func runbookApplicationRows(rows []map[string]any) []RunbookApplicationRecord {
	out := make([]RunbookApplicationRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, RunbookApplicationRecord{
			ID:         asString(row["id"]),
			RunbookID:  asString(row["runbook_id"]),
			IncidentID: asString(row["incident_id"]),
			ActorID:    asString(row["actor_id"]),
			Outcome:    asString(row["outcome"]),
			Note:       asString(row["note"]),
			CreatedAt:  asString(row["created_at"]),
		})
	}
	return out
}

// RunbookEntriesProposedCount returns the number of runbook candidates awaiting
// review. Used by the operator worklist to surface "N candidates to review".
func (d *DB) RunbookEntriesProposedCount() (int, error) {
	rows, err := d.QueryRows(`SELECT COUNT(*) AS c FROM incident_runbook_entries WHERE status='proposed';`)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return 0, nil
		}
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return int(asInt(rows[0]["c"])), nil
}
