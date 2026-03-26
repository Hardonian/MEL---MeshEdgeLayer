package meshintel

import (
	"fmt"

	"github.com/mel-project/mel/internal/db"
)

// MeshIntelStateRow is persisted streak / baseline for conservative incident hooks.
type MeshIntelStateRow struct {
	LastViability           string
	ConsecutiveBad          int
	LastGoodViability       string
	LastGoodReadiness       float64
	LastIncidentFingerprint string
}

// LoadMeshIntelState reads the singleton mesh_intel_state row.
func LoadMeshIntelState(d *db.DB) (MeshIntelStateRow, error) {
	var out MeshIntelStateRow
	if d == nil {
		return out, nil
	}
	rows, err := d.QueryRows(`SELECT COALESCE(last_viability,'') AS last_viability, COALESCE(consecutive_bad,0) AS consecutive_bad,
		COALESCE(last_good_viability,'') AS last_good_viability, COALESCE(last_good_readiness,0) AS last_good_readiness,
		COALESCE(last_incident_fingerprint,'') AS last_incident_fingerprint FROM mesh_intel_state WHERE id=1;`)
	if err != nil || len(rows) == 0 {
		return out, err
	}
	r := rows[0]
	out.LastViability = fmt.Sprint(r["last_viability"])
	out.ConsecutiveBad = int(dbAsInt64(r["consecutive_bad"]))
	out.LastGoodViability = fmt.Sprint(r["last_good_viability"])
	out.LastGoodReadiness = dbAsFloat(r["last_good_readiness"])
	out.LastIncidentFingerprint = fmt.Sprint(r["last_incident_fingerprint"])
	return out, nil
}

// SaveMeshIntelState persists streak / baseline.
func SaveMeshIntelState(d *db.DB, s MeshIntelStateRow) error {
	if d == nil {
		return nil
	}
	sql := fmt.Sprintf(`UPDATE mesh_intel_state SET last_viability='%s', consecutive_bad=%d, last_good_viability='%s', last_good_readiness=%f, last_incident_fingerprint='%s', updated_at=datetime('now') WHERE id=1;`,
		db.EscString(s.LastViability), s.ConsecutiveBad,
		db.EscString(s.LastGoodViability), s.LastGoodReadiness,
		db.EscString(s.LastIncidentFingerprint))
	return d.Exec(sql)
}
