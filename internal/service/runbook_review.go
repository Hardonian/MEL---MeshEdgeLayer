package service

// runbook_review.go — Review, promote, deprecate and apply runbook entries.
//
// This is the operator-facing half of the remediation-memory loop:
// - maybeSyncRunbookCandidate() inserts "proposed" rows from recurring
//   resolutions (see incident_moat.go)
// - This file lets operators list, inspect, promote, deprecate, and record an
//   application of a runbook to a specific incident.
//
// Every write emits an RBAC audit log and a timeline event. Every read is a
// deterministic projection of rows actually in the DB.

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/auth"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/models"
)

const (
	runbookListDefaultLimit         = 50
	runbookApplicationsPerEntryLimit = 20
)

// ListRunbookEntries returns a filtered, deterministic list of runbook entries.
func (a *App) ListRunbookEntries(status, signatureKey, fingerprintHash, query string, limit int) ([]models.RunbookEntryDTO, error) {
	if a == nil || a.DB == nil {
		return nil, fmt.Errorf("service not available")
	}
	if limit <= 0 {
		limit = runbookListDefaultLimit
	}
	rows, err := a.DB.ListRunbookEntries(status, signatureKey, fingerprintHash, query, limit)
	if err != nil {
		return nil, fmt.Errorf("could not list runbook entries: %w", err)
	}
	out := make([]models.RunbookEntryDTO, 0, len(rows))
	for _, r := range rows {
		stats, _ := a.DB.RunbookEntryStatsByID(r.ID)
		out = append(out, runbookEntryDTOFrom(r, stats))
	}
	return out, nil
}

// GetRunbookEntry returns one runbook row with bounded application history.
func (a *App) GetRunbookEntry(id string) (models.RunbookEntryDetailDTO, bool, error) {
	if a == nil || a.DB == nil {
		return models.RunbookEntryDetailDTO{}, false, fmt.Errorf("service not available")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return models.RunbookEntryDetailDTO{}, false, fmt.Errorf("runbook id is required")
	}
	row, ok, err := a.DB.RunbookEntryByID(id)
	if err != nil {
		return models.RunbookEntryDetailDTO{}, false, fmt.Errorf("could not load runbook: %w", err)
	}
	if !ok {
		return models.RunbookEntryDetailDTO{}, false, nil
	}
	stats, _ := a.DB.RunbookEntryStatsByID(id)
	apps, _ := a.DB.RunbookApplicationsForRunbook(id, runbookApplicationsPerEntryLimit)
	detail := models.RunbookEntryDetailDTO{
		Entry:        runbookEntryDTOFrom(row, stats),
		Applications: runbookApplicationDTOsFrom(apps),
	}
	return detail, true, nil
}

// PromoteRunbookEntry moves a runbook from proposed/reviewing to promoted.
// Returns not-found when the id does not exist.
func (a *App) PromoteRunbookEntry(id, actorID, note string) error {
	return a.transitionRunbookStatus(id, actorID, db.RunbookStatusPromoted, note, "runbook_promoted")
}

// DeprecateRunbookEntry moves a runbook to deprecated with a required reason.
func (a *App) DeprecateRunbookEntry(id, actorID, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("deprecation reason is required")
	}
	return a.transitionRunbookStatus(id, actorID, db.RunbookStatusDeprecated, reason, "runbook_deprecated")
}

func (a *App) transitionRunbookStatus(id, actorID, targetStatus, reason, timelineKind string) error {
	if a == nil || a.DB == nil {
		return fmt.Errorf("service not available")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("runbook id is required")
	}
	if strings.TrimSpace(actorID) == "" {
		actorID = "system"
	}
	row, ok, err := a.DB.RunbookEntryByID(id)
	if err != nil {
		return fmt.Errorf("could not load runbook: %w", err)
	}
	if !ok {
		return fmt.Errorf("runbook not found: %s", id)
	}
	updated, err := a.DB.SetRunbookStatus(id, targetStatus, actorID, reason)
	if err != nil {
		return fmt.Errorf("could not update runbook: %w", err)
	}
	if !updated {
		return fmt.Errorf("runbook not found: %s", id)
	}

	_ = a.DB.InsertRBACAuditLog(auth.AuditEntry{
		ID:           newTrustID("aud"),
		ActorID:      auth.OperatorID(actorID),
		ActionClass:  auth.ActionControl,
		ActionDetail: "runbook_" + targetStatus,
		ResourceType: "runbook",
		ResourceID:   id,
		Reason:       reason,
		Result:       auth.AuditResultSuccess,
		Timestamp:    time.Now().UTC(),
	})
	_ = a.DB.InsertTimelineEvent(db.TimelineEvent{
		EventID:    newTrustID("tl"),
		EventType:  timelineKind,
		Summary:    fmt.Sprintf("runbook %s → %s: %s", id, targetStatus, row.Title),
		Severity:   "info",
		ActorID:    actorID,
		ResourceID: id,
		Details: map[string]any{
			"runbook_id":      id,
			"previous_status": row.Status,
			"new_status":      targetStatus,
			"reason":          reason,
			"title":           row.Title,
		},
	})
	return nil
}

// ApplyRunbookToIncident records one operator attaching a runbook to an incident
// with an explicit outcome. The runbook and incident must both exist. Updates
// bounded effectiveness counters on the runbook row.
func (a *App) ApplyRunbookToIncident(incidentID, actorID string, req models.ApplyRunbookRequest) (models.RunbookApplicationDTO, error) {
	if a == nil || a.DB == nil {
		return models.RunbookApplicationDTO{}, fmt.Errorf("service not available")
	}
	incidentID = strings.TrimSpace(incidentID)
	runbookID := strings.TrimSpace(req.RunbookID)
	if incidentID == "" || runbookID == "" {
		return models.RunbookApplicationDTO{}, fmt.Errorf("incident_id and runbook_id are required")
	}
	outcome := strings.ToLower(strings.TrimSpace(req.Outcome))
	if outcome == "" {
		outcome = db.RunbookOutcomeApplied
	}
	if !db.ValidRunbookOutcome(outcome) {
		return models.RunbookApplicationDTO{}, fmt.Errorf("invalid outcome %q", req.Outcome)
	}
	if strings.TrimSpace(actorID) == "" {
		actorID = "system"
	}

	// Verify both rows exist so we do not create orphan audit trails.
	inc, incOk, err := a.DB.IncidentByID(incidentID)
	if err != nil {
		return models.RunbookApplicationDTO{}, fmt.Errorf("could not load incident: %w", err)
	}
	if !incOk {
		return models.RunbookApplicationDTO{}, fmt.Errorf("incident not found: %s", incidentID)
	}
	runbook, rbOk, err := a.DB.RunbookEntryByID(runbookID)
	if err != nil {
		return models.RunbookApplicationDTO{}, fmt.Errorf("could not load runbook: %w", err)
	}
	if !rbOk {
		return models.RunbookApplicationDTO{}, fmt.Errorf("runbook not found: %s", runbookID)
	}

	rec := db.RunbookApplicationRecord{
		ID:         newTrustID("rba"),
		RunbookID:  runbookID,
		IncidentID: incidentID,
		ActorID:    actorID,
		Outcome:    outcome,
		Note:       strings.TrimSpace(req.Note),
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	if err := a.DB.InsertRunbookApplication(rec); err != nil {
		return models.RunbookApplicationDTO{}, fmt.Errorf("could not persist runbook application: %w", err)
	}

	_ = a.DB.InsertRBACAuditLog(auth.AuditEntry{
		ID:           newTrustID("aud"),
		ActorID:      auth.OperatorID(actorID),
		ActionClass:  auth.ActionControl,
		ActionDetail: "runbook_applied",
		ResourceType: "incident",
		ResourceID:   incidentID,
		Reason:       fmt.Sprintf("runbook_id=%s outcome=%s", runbookID, outcome),
		Result:       auth.AuditResultSuccess,
		Timestamp:    time.Now().UTC(),
	})
	_ = a.DB.InsertTimelineEvent(db.TimelineEvent{
		EventID:    newTrustID("tl"),
		EventType:  "runbook_applied",
		Summary:    fmt.Sprintf("runbook %s applied to incident %s (%s)", runbookID, incidentID, outcome),
		Severity:   "info",
		ActorID:    actorID,
		ResourceID: incidentID,
		Details: map[string]any{
			"runbook_id":    runbookID,
			"runbook_title": runbook.Title,
			"incident_id":   incidentID,
			"outcome":       outcome,
			"note":          rec.Note,
		},
	})

	// Ensure the incident row's updated_at reflects the application without
	// mutating workflow fields; UpsertIncident handles this deterministically.
	inc.UpdatedAt = rec.CreatedAt
	_ = a.DB.UpsertIncident(inc)

	return models.RunbookApplicationDTO{
		ID:         rec.ID,
		RunbookID:  rec.RunbookID,
		IncidentID: rec.IncidentID,
		ActorID:    rec.ActorID,
		Outcome:    rec.Outcome,
		Note:       rec.Note,
		CreatedAt:  rec.CreatedAt,
	}, nil
}

// RunbookApplicationsForIncidentDTO returns bounded runbook application history
// for an incident id.
func (a *App) RunbookApplicationsForIncidentDTO(incidentID string, limit int) ([]models.RunbookApplicationDTO, error) {
	if a == nil || a.DB == nil {
		return nil, fmt.Errorf("service not available")
	}
	if limit <= 0 {
		limit = runbookApplicationsPerEntryLimit
	}
	rows, err := a.DB.RunbookApplicationsForIncident(strings.TrimSpace(incidentID), limit)
	if err != nil {
		return nil, err
	}
	return runbookApplicationDTOsFrom(rows), nil
}

// runbookEntryDTOFrom maps a repo row + stats to the API DTO.
func runbookEntryDTOFrom(row db.RunbookEntryRecord, stats db.RunbookEntryStats) models.RunbookEntryDTO {
	evidence := parseJSONStringArray(row.EvidenceRefJSON)
	sources := parseJSONStringArray(row.SourceIncidentIDsJSON)
	return models.RunbookEntryDTO{
		ID:                    row.ID,
		Status:                row.Status,
		SourceKind:            row.SourceKind,
		Title:                 row.Title,
		Body:                  row.Body,
		LegacySignatureKey:    row.LegacySignatureKey,
		FingerprintHash:       row.FingerprintCanonicalHash,
		EvidenceRefs:          evidence,
		SourceIncidentIDs:     sources,
		PromotionBasis:        row.PromotionBasis,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
		ReviewedAt:            row.ReviewedAt,
		ReviewerActorID:       row.ReviewerActorID,
		AppliedCount:          stats.AppliedCount,
		UsefulCount:           stats.UsefulCount,
		IneffectiveCount:      stats.IneffectiveCount,
		LastAppliedAt:         stats.LastAppliedAt,
		LastAppliedIncidentID: stats.LastAppliedIncidentID,
		PromotedAt:            stats.PromotedAt,
		PromotedByActorID:     stats.PromotedByActorID,
		DeprecatedAt:          stats.DeprecatedAt,
		DeprecatedByActorID:   stats.DeprecatedByActorID,
		DeprecatedReason:      stats.DeprecatedReason,
	}
}

func runbookApplicationDTOsFrom(rows []db.RunbookApplicationRecord) []models.RunbookApplicationDTO {
	out := make([]models.RunbookApplicationDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.RunbookApplicationDTO{
			ID:         r.ID,
			RunbookID:  r.RunbookID,
			IncidentID: r.IncidentID,
			ActorID:    r.ActorID,
			Outcome:    r.Outcome,
			Note:       r.Note,
			CreatedAt:  r.CreatedAt,
		})
	}
	return out
}

func parseJSONStringArray(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "[]" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil
	}
	return out
}
