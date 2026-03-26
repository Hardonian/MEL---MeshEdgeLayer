package meshintel

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mel-project/mel/internal/db"
)

func escapeSQLLiteral(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

const defaultMeshIntelHistory = 120

// SaveSnapshot persists one assessment row (prunes old rows afterward).
func SaveSnapshot(d *db.DB, a Assessment, keep int) error {
	if d == nil {
		return nil
	}
	if keep <= 0 {
		keep = defaultMeshIntelHistory
	}
	b, err := json.Marshal(a)
	if err != nil {
		return err
	}
	sql := fmt.Sprintf(`INSERT INTO mesh_intelligence_snapshots(assessment_id, created_at, scope, graph_hash, payload_json)
		VALUES('%s','%s','mesh','%s','%s');`,
		db.EscString(a.AssessmentID), db.EscString(a.ComputedAt),
		db.EscString(a.GraphHash), escapeSQLLiteral(string(b)))
	if err := d.Exec(sql); err != nil {
		return err
	}
	prune := fmt.Sprintf(`DELETE FROM mesh_intelligence_snapshots WHERE rowid NOT IN (
		SELECT rowid FROM mesh_intelligence_snapshots ORDER BY created_at DESC LIMIT %d
	);`, keep)
	return d.Exec(prune)
}

// RecentSnapshots returns newest assessments (payload decoded).
func RecentSnapshots(d *db.DB, limit int) ([]Assessment, error) {
	if d == nil {
		return nil, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	rows, err := d.QueryRows(fmt.Sprintf(
		`SELECT assessment_id, created_at, COALESCE(graph_hash,'') AS graph_hash, payload_json FROM mesh_intelligence_snapshots ORDER BY created_at DESC LIMIT %d;`, limit))
	if err != nil {
		return nil, err
	}
	var out []Assessment
	for _, row := range rows {
		var a Assessment
		if err := json.Unmarshal([]byte(fmt.Sprint(row["payload_json"])), &a); err != nil {
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

// LatestSnapshot returns the most recent persisted assessment, if any.
func LatestSnapshot(d *db.DB) (Assessment, bool, error) {
	list, err := RecentSnapshots(d, 1)
	if err != nil || len(list) == 0 {
		return Assessment{}, false, err
	}
	return list[0], true, nil
}

// GetAssessmentByID loads a persisted snapshot by assessment_id (for planning validation baselines).
func GetAssessmentByID(d *db.DB, assessmentID string) (Assessment, bool, error) {
	if d == nil || strings.TrimSpace(assessmentID) == "" {
		return Assessment{}, false, nil
	}
	rows, err := d.QueryRows(fmt.Sprintf(
		`SELECT payload_json FROM mesh_intelligence_snapshots WHERE assessment_id='%s' ORDER BY created_at DESC LIMIT 1;`,
		db.EscString(assessmentID)))
	if err != nil {
		return Assessment{}, false, err
	}
	if len(rows) == 0 {
		return Assessment{}, false, nil
	}
	var a Assessment
	if err := json.Unmarshal([]byte(fmt.Sprint(rows[0]["payload_json"])), &a); err != nil {
		return Assessment{}, false, err
	}
	return a, true, nil
}
