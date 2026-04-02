package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/models"
)

// IncidentDecisionPackAdjudicationRecord mirrors incident_decision_pack_adjudication.
type IncidentDecisionPackAdjudicationRecord struct {
	IncidentID        string
	Reviewed          bool
	ReviewedAt        string
	ReviewedByActorID string
	Useful            string
	OperatorNote      string
	CueOutcomesJSON   string
	UpdatedAt         string
}

func (d *DB) GetIncidentDecisionPackAdjudication(incidentID string) (IncidentDecisionPackAdjudicationRecord, bool, error) {
	if d == nil {
		return IncidentDecisionPackAdjudicationRecord{}, false, fmt.Errorf(ErrDatabaseUnavailable)
	}
	id := strings.TrimSpace(incidentID)
	if id == "" {
		return IncidentDecisionPackAdjudicationRecord{}, false, nil
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT incident_id, reviewed, reviewed_at, reviewed_by_actor_id, useful, operator_note, cue_outcomes_json, updated_at
		FROM incident_decision_pack_adjudication WHERE incident_id='%s' LIMIT 1;`, esc(id)))
	if err != nil {
		return IncidentDecisionPackAdjudicationRecord{}, false, err
	}
	if len(rows) == 0 {
		return IncidentDecisionPackAdjudicationRecord{}, false, nil
	}
	row := rows[0]
	return IncidentDecisionPackAdjudicationRecord{
		IncidentID:        asString(row["incident_id"]),
		Reviewed:          asInt(row["reviewed"]) != 0,
		ReviewedAt:        asString(row["reviewed_at"]),
		ReviewedByActorID: asString(row["reviewed_by_actor_id"]),
		Useful:            asString(row["useful"]),
		OperatorNote:      asString(row["operator_note"]),
		CueOutcomesJSON:   asString(row["cue_outcomes_json"]),
		UpdatedAt:         asString(row["updated_at"]),
	}, true, nil
}

func (d *DB) UpsertIncidentDecisionPackAdjudication(rec IncidentDecisionPackAdjudicationRecord) error {
	if d == nil {
		return fmt.Errorf(ErrDatabaseUnavailable)
	}
	if strings.TrimSpace(rec.IncidentID) == "" {
		return fmt.Errorf("incident id is required")
	}
	if rec.CueOutcomesJSON == "" {
		rec.CueOutcomesJSON = "[]"
	}
	if rec.UpdatedAt == "" {
		rec.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	rv := 0
	if rec.Reviewed {
		rv = 1
	}
	sql := fmt.Sprintf(`INSERT INTO incident_decision_pack_adjudication(incident_id, reviewed, reviewed_at, reviewed_by_actor_id, useful, operator_note, cue_outcomes_json, updated_at)
		VALUES('%s', %d, '%s', '%s', '%s', '%s', '%s', '%s')
		ON CONFLICT(incident_id) DO UPDATE SET reviewed=excluded.reviewed, reviewed_at=excluded.reviewed_at, reviewed_by_actor_id=excluded.reviewed_by_actor_id,
		useful=excluded.useful, operator_note=excluded.operator_note, cue_outcomes_json=excluded.cue_outcomes_json, updated_at=excluded.updated_at;`,
		esc(rec.IncidentID), rv, esc(rec.ReviewedAt), esc(rec.ReviewedByActorID), esc(rec.Useful), esc(rec.OperatorNote), esc(rec.CueOutcomesJSON), esc(rec.UpdatedAt))
	return d.Exec(sql)
}

func DecodeCueOutcomesJSON(s string) ([]models.IncidentDecisionPackCueOutcome, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var out []models.IncidentDecisionPackCueOutcome
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, err
	}
	return out, nil
}
