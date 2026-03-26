package planning

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/db"
)

const (
	defaultPlanRetention  = 200
	defaultArtifactRetention = 300
)

// SavePlan upserts a deployment plan. If PlanID is empty, one is generated and written back into p.
func SavePlan(d *db.DB, p *DeploymentPlan) error {
	if d == nil {
		return fmt.Errorf("database unavailable")
	}
	if strings.TrimSpace(p.PlanID) == "" {
		p.PlanID = "plan-" + randomID()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if p.CreatedAt == "" {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	title := db.EscString(p.Title)
	status := db.EscString(p.Status)
	if status == "" {
		status = "draft"
	}
	intent := db.EscString(p.Intent)
	inpVer := db.EscString(strings.TrimSpace(p.InputSetVersionID))
	sql := fmt.Sprintf(`INSERT INTO deployment_plans(plan_id, title, status, intent, created_at, updated_at, payload_json, input_set_version_id)
		VALUES('%s','%s','%s','%s','%s','%s','%s','%s')
		ON CONFLICT(plan_id) DO UPDATE SET title=excluded.title, status=excluded.status, intent=excluded.intent, updated_at=excluded.updated_at, payload_json=excluded.payload_json, input_set_version_id=excluded.input_set_version_id;`,
		db.EscString(p.PlanID), title, status, intent, db.EscString(p.CreatedAt), db.EscString(p.UpdatedAt), db.EscString(string(b)), inpVer)
	return d.Exec(sql)
}

// ListPlans returns recent plans (newest first).
func ListPlans(d *db.DB, limit int) ([]DeploymentPlan, error) {
	if d == nil {
		return nil, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := d.QueryRows(fmt.Sprintf(
		`SELECT plan_id, title, status, intent, created_at, updated_at, payload_json FROM deployment_plans ORDER BY updated_at DESC LIMIT %d;`, limit))
	if err != nil {
		return nil, err
	}
	var out []DeploymentPlan
	for _, row := range rows {
		var p DeploymentPlan
		if err := json.Unmarshal([]byte(fmt.Sprint(row["payload_json"])), &p); err != nil {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

// GetPlan loads one plan by id.
func GetPlan(d *db.DB, planID string) (DeploymentPlan, bool, error) {
	if d == nil {
		return DeploymentPlan{}, false, fmt.Errorf("database unavailable")
	}
	planID = strings.TrimSpace(planID)
	if planID == "" {
		return DeploymentPlan{}, false, nil
	}
	rows, err := d.QueryRows(fmt.Sprintf(
		`SELECT payload_json FROM deployment_plans WHERE plan_id='%s' LIMIT 1;`, db.EscString(planID)))
	if err != nil {
		return DeploymentPlan{}, false, err
	}
	if len(rows) == 0 {
		return DeploymentPlan{}, false, nil
	}
	var p DeploymentPlan
	if err := json.Unmarshal([]byte(fmt.Sprint(rows[0]["payload_json"])), &p); err != nil {
		return DeploymentPlan{}, false, err
	}
	return p, true, nil
}

// PrunePlans keeps the newest maxRows plans.
func PrunePlans(d *db.DB, maxRows int) error {
	if d == nil {
		return nil
	}
	if maxRows <= 0 {
		maxRows = defaultPlanRetention
	}
	sql := fmt.Sprintf(`DELETE FROM deployment_plans WHERE plan_id NOT IN (
		SELECT plan_id FROM deployment_plans ORDER BY updated_at DESC LIMIT %d
	);`, maxRows)
	return d.Exec(sql)
}

// SaveArtifact stores a scenario or comparison result with bounded retention.
func SaveArtifact(d *db.DB, kind string, graphHash, assessmentID string, payload any, maxKeep int) error {
	if d == nil {
		return nil
	}
	kind = strings.TrimSpace(kind)
	if kind == "" {
		kind = "unknown"
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	id := "pa-" + randomID()
	now := time.Now().UTC().Format(time.RFC3339)
	sql := fmt.Sprintf(`INSERT INTO planning_artifacts(artifact_id, kind, created_at, graph_hash, assessment_id, payload_json)
		VALUES('%s','%s','%s','%s','%s','%s');`,
		db.EscString(id), db.EscString(kind), db.EscString(now), db.EscString(graphHash), db.EscString(assessmentID), db.EscString(string(b)))
	if err := d.Exec(sql); err != nil {
		return err
	}
	if maxKeep <= 0 {
		maxKeep = defaultArtifactRetention
	}
	prune := fmt.Sprintf(`DELETE FROM planning_artifacts WHERE artifact_id NOT IN (
		SELECT artifact_id FROM planning_artifacts ORDER BY created_at DESC LIMIT %d
	);`, maxKeep)
	return d.Exec(prune)
}

// PruneArtifacts keeps the newest maxKeep planning artifact rows (scenario/compare outputs).
func PruneArtifacts(d *db.DB, maxKeep int) error {
	if d == nil {
		return nil
	}
	if maxKeep <= 0 {
		maxKeep = defaultArtifactRetention
	}
	prune := fmt.Sprintf(`DELETE FROM planning_artifacts WHERE artifact_id NOT IN (
		SELECT artifact_id FROM planning_artifacts ORDER BY created_at DESC LIMIT %d
	);`, maxKeep)
	return d.Exec(prune)
}

func randomID() string {
	b := make([]byte, 10)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
