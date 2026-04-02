package db

import (
	"fmt"
	"strings"
	"time"
)

// IncidentIntelSignalOutcomeRecord is operator adjudication for a deterministic assist signal code.
type IncidentIntelSignalOutcomeRecord struct {
	ID         string
	IncidentID string
	SignalCode string
	Outcome    string
	ActorID    string
	Note       string
	CreatedAt  string
}

func (d *DB) InsertIncidentIntelSignalOutcome(rec IncidentIntelSignalOutcomeRecord) error {
	if strings.TrimSpace(rec.ID) == "" || strings.TrimSpace(rec.IncidentID) == "" || strings.TrimSpace(rec.SignalCode) == "" || strings.TrimSpace(rec.Outcome) == "" {
		return fmt.Errorf("id, incident_id, signal_code, and outcome are required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if rec.CreatedAt == "" {
		rec.CreatedAt = now
	}
	if rec.ActorID == "" {
		rec.ActorID = "system"
	}
	sql := fmt.Sprintf(`INSERT INTO incident_intel_signal_outcomes(id,incident_id,signal_code,outcome,actor_id,note,created_at)
VALUES('%s','%s','%s','%s','%s','%s','%s');`,
		esc(rec.ID), esc(rec.IncidentID), esc(rec.SignalCode), esc(rec.Outcome), esc(rec.ActorID), esc(rec.Note), esc(rec.CreatedAt))
	return d.Exec(sql)
}

// LatestIntelSignalOutcomesByIncident returns the most recent row per signal_code for an incident.
func (d *DB) LatestIntelSignalOutcomesByIncident(incidentID string) (map[string]IncidentIntelSignalOutcomeRecord, error) {
	incidentID = strings.TrimSpace(incidentID)
	if incidentID == "" {
		return nil, nil
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id,incident_id,signal_code,outcome,COALESCE(actor_id,'') AS actor_id,COALESCE(note,'') AS note,created_at
FROM incident_intel_signal_outcomes WHERE incident_id='%s' ORDER BY created_at DESC;`, esc(incidentID)))
	if err != nil {
		return nil, err
	}
	out := map[string]IncidentIntelSignalOutcomeRecord{}
	for _, row := range rows {
		code := strings.TrimSpace(asString(row["signal_code"]))
		if code == "" {
			continue
		}
		if _, ok := out[code]; ok {
			continue
		}
		out[code] = IncidentIntelSignalOutcomeRecord{
			ID:         asString(row["id"]),
			IncidentID: asString(row["incident_id"]),
			SignalCode: code,
			Outcome:    asString(row["outcome"]),
			ActorID:    asString(row["actor_id"]),
			Note:       asString(row["note"]),
			CreatedAt:  asString(row["created_at"]),
		}
	}
	return out, nil
}
