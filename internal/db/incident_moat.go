package db

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/models"
)

// --- Recommendation outcomes (operator feedback on assistive guidance) ---

type IncidentRecommendationOutcomeRecord struct {
	ID               string
	IncidentID       string
	RecommendationID string
	Outcome          string
	ActorID          string
	Note             string
	CreatedAt        string
}

func (d *DB) InsertIncidentRecommendationOutcome(rec IncidentRecommendationOutcomeRecord) error {
	if strings.TrimSpace(rec.ID) == "" || strings.TrimSpace(rec.IncidentID) == "" || strings.TrimSpace(rec.RecommendationID) == "" || strings.TrimSpace(rec.Outcome) == "" {
		return fmt.Errorf("id, incident_id, recommendation_id, and outcome are required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if rec.CreatedAt == "" {
		rec.CreatedAt = now
	}
	if rec.ActorID == "" {
		rec.ActorID = "system"
	}
	sql := fmt.Sprintf(`INSERT INTO incident_recommendation_outcomes(id,incident_id,recommendation_id,outcome,actor_id,note,created_at)
VALUES('%s','%s','%s','%s','%s','%s','%s');`,
		esc(rec.ID), esc(rec.IncidentID), esc(rec.RecommendationID), esc(rec.Outcome), esc(rec.ActorID), esc(rec.Note), esc(rec.CreatedAt))
	return d.Exec(sql)
}

func (d *DB) RecommendationOutcomesForIncident(incidentID string, limit int) ([]IncidentRecommendationOutcomeRecord, error) {
	incidentID = strings.TrimSpace(incidentID)
	if incidentID == "" {
		return nil, nil
	}
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id,incident_id,recommendation_id,outcome,COALESCE(actor_id,'') AS actor_id,COALESCE(note,'') AS note,created_at
FROM incident_recommendation_outcomes WHERE incident_id='%s' ORDER BY created_at DESC LIMIT %d;`, esc(incidentID), limit))
	if err != nil {
		return nil, err
	}
	out := make([]IncidentRecommendationOutcomeRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, IncidentRecommendationOutcomeRecord{
			ID:               asString(row["id"]),
			IncidentID:       asString(row["incident_id"]),
			RecommendationID: asString(row["recommendation_id"]),
			Outcome:          asString(row["outcome"]),
			ActorID:          asString(row["actor_id"]),
			Note:             asString(row["note"]),
			CreatedAt:        asString(row["created_at"]),
		})
	}
	return out, nil
}

// --- Cross-incident correlation (shared signature key; association only) ---

// EnsureSignatureCorrelationGroup syncs the correlation group for a signature with all linked incidents.
func (d *DB) EnsureSignatureCorrelationGroup(signatureKey string) error {
	signatureKey = strings.TrimSpace(signatureKey)
	if signatureKey == "" {
		return nil
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT incident_id FROM incident_signature_incidents WHERE signature_key='%s' ORDER BY linked_at ASC;`, esc(signatureKey)))
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	incidentIDs := make([]string, 0, len(rows))
	seen := map[string]struct{}{}
	for _, row := range rows {
		id := strings.TrimSpace(asString(row["incident_id"]))
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		incidentIDs = append(incidentIDs, id)
	}
	if len(incidentIDs) < 2 {
		return nil
	}
	sum := sha1.Sum([]byte("corr|shared_signature|" + signatureKey))
	groupID := fmt.Sprintf("cg-%x", sum[:6])
	correlationKey := "shared_signature:" + signatureKey
	now := time.Now().UTC().Format(time.RFC3339)
	rationale, _ := json.Marshal([]string{
		"Incidents linked to the same deterministic incident_signature row in MEL storage.",
		"Resemblance is by shared signature key only; not proof of identical root cause.",
	})
	evidenceRefs, _ := json.Marshal([]string{"incident_signatures:" + signatureKey, "incident_signature_incidents"})
	uncertainty := "Correlation is structural (shared persisted signature). Verify timeline and evidence before treating as one operational failure domain."

	sql := fmt.Sprintf(`INSERT INTO incident_correlation_groups(id,correlation_key,basis,created_at,updated_at,rationale_json,evidence_refs_json,uncertainty_note)
VALUES('%s','%s','shared_signature_key','%s','%s','%s','%s','%s')
ON CONFLICT(correlation_key) DO UPDATE SET updated_at=excluded.updated_at,rationale_json=excluded.rationale_json,evidence_refs_json=excluded.evidence_refs_json,uncertainty_note=excluded.uncertainty_note;`,
		esc(groupID), esc(correlationKey), esc(now), esc(now), esc(string(rationale)), esc(string(evidenceRefs)), esc(uncertainty))
	if err := d.Exec(sql); err != nil {
		return err
	}
	for _, incID := range incidentIDs {
		q := fmt.Sprintf(`INSERT OR IGNORE INTO incident_correlation_members(group_id,incident_id,joined_at) VALUES('%s','%s','%s');`,
			esc(groupID), esc(incID), esc(now))
		if err := d.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) CorrelationGroupsForIncident(incidentID string) ([]models.IncidentCorrelationGroup, error) {
	incidentID = strings.TrimSpace(incidentID)
	if incidentID == "" {
		return nil, nil
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT g.id,g.correlation_key,g.basis,g.created_at,g.updated_at,g.rationale_json,g.evidence_refs_json,g.uncertainty_note
FROM incident_correlation_groups g
JOIN incident_correlation_members m ON m.group_id=g.id
WHERE m.incident_id='%s'
ORDER BY g.updated_at DESC;`, esc(incidentID)))
	if err != nil {
		return nil, err
	}
	out := make([]models.IncidentCorrelationGroup, 0, len(rows))
	for _, row := range rows {
		var rationale, refs []string
		_ = json.Unmarshal([]byte(asString(row["rationale_json"])), &rationale)
		_ = json.Unmarshal([]byte(asString(row["evidence_refs_json"])), &refs)
		out = append(out, models.IncidentCorrelationGroup{
			GroupID:         asString(row["id"]),
			CorrelationKey:  asString(row["correlation_key"]),
			Basis:           asString(row["basis"]),
			CreatedAt:       asString(row["created_at"]),
			UpdatedAt:       asString(row["updated_at"]),
			Rationale:       rationale,
			EvidenceRefs:    refs,
			UncertaintyNote: asString(row["uncertainty_note"]),
		})
	}
	return out, nil
}

func (d *DB) CorrelatedIncidentIDsForGroup(groupID string) ([]string, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, nil
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT incident_id FROM incident_correlation_members WHERE group_id='%s' ORDER BY joined_at ASC;`, esc(groupID)))
	if err != nil {
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
