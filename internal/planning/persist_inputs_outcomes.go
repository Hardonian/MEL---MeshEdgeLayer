package planning

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/db"
)

// --- Input sets ---

func SaveInputSet(d *db.DB, id, title string) (string, error) {
	if d == nil {
		return "", fmt.Errorf("database unavailable")
	}
	if strings.TrimSpace(id) == "" {
		id = "pis-" + randomID()
	}
	now := nowRFC3339()
	title = strings.TrimSpace(title)
	if title == "" {
		title = "input set"
	}
	sql := fmt.Sprintf(`INSERT INTO planning_input_sets(input_set_id, title, created_at, updated_at) VALUES('%s','%s','%s','%s')
		ON CONFLICT(input_set_id) DO UPDATE SET title=excluded.title, updated_at=excluded.updated_at;`,
		db.EscString(id), db.EscString(title), db.EscString(now), db.EscString(now))
	if err := d.Exec(sql); err != nil {
		return "", err
	}
	return id, nil
}

func nextInputVersionNum(d *db.DB, inputSetID string) (int, error) {
	rows, err := d.QueryRows(fmt.Sprintf(
		`SELECT COALESCE(MAX(version_num),0) AS m FROM planning_input_versions WHERE input_set_id='%s';`,
		db.EscString(inputSetID)))
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 1, nil
	}
	m := int(asInt64(rows[0]["m"]))
	return m + 1, nil
}

// SaveInputVersion creates a new version row; returns version_id.
func SaveInputVersion(d *db.DB, inputSetID string, payload PlanningInputVersionPayload) (string, error) {
	if d == nil {
		return "", fmt.Errorf("database unavailable")
	}
	inputSetID = strings.TrimSpace(inputSetID)
	if inputSetID == "" {
		return "", fmt.Errorf("input_set_id required")
	}
	n, err := nextInputVersionNum(d, inputSetID)
	if err != nil {
		return "", err
	}
	payload.VersionNum = n
	payload.InputSetID = inputSetID
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	vid := "piv-" + randomID()
	now := nowRFC3339()
	sql := fmt.Sprintf(`INSERT INTO planning_input_versions(version_id, input_set_id, version_num, created_at, payload_json)
		VALUES('%s','%s',%d,'%s','%s');`,
		db.EscString(vid), db.EscString(inputSetID), n, db.EscString(now), db.EscString(string(b)))
	if err := d.Exec(sql); err != nil {
		return "", err
	}
	_ = d.Exec(fmt.Sprintf(`UPDATE planning_input_sets SET updated_at='%s' WHERE input_set_id='%s';`, db.EscString(now), db.EscString(inputSetID)))
	return vid, nil
}

func GetInputVersion(d *db.DB, versionID string) (PlanningInputVersionPayload, bool, error) {
	if d == nil {
		return PlanningInputVersionPayload{}, false, fmt.Errorf("database unavailable")
	}
	versionID = strings.TrimSpace(versionID)
	if versionID == "" {
		return PlanningInputVersionPayload{}, false, nil
	}
	rows, err := d.QueryRows(fmt.Sprintf(
		`SELECT payload_json FROM planning_input_versions WHERE version_id='%s' LIMIT 1;`, db.EscString(versionID)))
	if err != nil {
		return PlanningInputVersionPayload{}, false, err
	}
	if len(rows) == 0 {
		return PlanningInputVersionPayload{}, false, nil
	}
	var p PlanningInputVersionPayload
	if err := json.Unmarshal([]byte(fmt.Sprint(rows[0]["payload_json"])), &p); err != nil {
		return PlanningInputVersionPayload{}, false, err
	}
	return p, true, nil
}

func ListInputSets(d *db.DB, limit int) ([]PlanningInputSetMeta, error) {
	if d == nil {
		return nil, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := d.QueryRows(fmt.Sprintf(
		`SELECT input_set_id, title, updated_at FROM planning_input_sets ORDER BY updated_at DESC LIMIT %d;`, limit))
	if err != nil {
		return nil, err
	}
	var out []PlanningInputSetMeta
	for _, row := range rows {
		out = append(out, PlanningInputSetMeta{
			InputSetID: fmt.Sprint(row["input_set_id"]),
			Title:      fmt.Sprint(row["title"]),
			UpdatedAt:  fmt.Sprint(row["updated_at"]),
		})
	}
	return out, nil
}

func asInt64(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	default:
		return 0
	}
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// --- Plan execution / validation ---

func StartPlanExecution(d *db.DB, planID, graphHash, assessmentID string, baseline PostChangeMetricsSnapshot, observeHours int, notes string) (string, error) {
	if d == nil {
		return "", fmt.Errorf("database unavailable")
	}
	planID = strings.TrimSpace(planID)
	if planID == "" {
		return "", fmt.Errorf("plan_id required")
	}
	if !baseline.Captured {
		baseline.Captured = true
	}
	bj, _ := json.Marshal(baseline)
	eid := "pex-" + randomID()
	now := nowRFC3339()
	sql := fmt.Sprintf(`INSERT INTO plan_executions(execution_id, plan_id, plan_graph_hash, mesh_assessment_id, baseline_metrics_json, status, started_at, updated_at, observation_horizon_hours, notes)
		VALUES('%s','%s','%s','%s','%s','attempted','%s','%s',%d,'%s');`,
		db.EscString(eid), db.EscString(planID), db.EscString(graphHash), db.EscString(assessmentID), db.EscString(string(bj)), db.EscString(now), db.EscString(now), observeHours, db.EscString(notes))
	if err := d.Exec(sql); err != nil {
		return "", err
	}
	return eid, nil
}

func MarkStepExecuted(d *db.DB, executionID, stepID, note string) (string, error) {
	if d == nil {
		return "", fmt.Errorf("database unavailable")
	}
	sid := "pst-" + randomID()
	now := nowRFC3339()
	sql := fmt.Sprintf(`INSERT INTO plan_step_executions(step_execution_id, execution_id, step_id, status, attempted_at, operator_note)
		VALUES('%s','%s','%s','attempted','%s','%s');`,
		db.EscString(sid), db.EscString(executionID), db.EscString(stepID), db.EscString(now), db.EscString(note))
	if err := d.Exec(sql); err != nil {
		return "", err
	}
	_ = d.Exec(fmt.Sprintf(`UPDATE plan_executions SET updated_at='%s', status='in_observation' WHERE execution_id='%s';`,
		db.EscString(now), db.EscString(executionID)))
	return sid, nil
}

func SaveValidation(d *db.DB, executionID string, vr ValidationResult) (string, error) {
	if d == nil {
		return "", fmt.Errorf("database unavailable")
	}
	vr.ValidationID = "val-" + randomID()
	vr.ValidatedAt = nowRFC3339()
	b, err := json.Marshal(vr)
	if err != nil {
		return "", err
	}
	sql := fmt.Sprintf(`INSERT INTO plan_validations(validation_id, execution_id, validated_at, graph_hash_after, mesh_assessment_id_after, verdict, payload_json)
		VALUES('%s','%s','%s','%s','%s','%s','%s');`,
		db.EscString(vr.ValidationID), db.EscString(executionID), db.EscString(vr.ValidatedAt),
		db.EscString(vr.GraphHashAfter), db.EscString(vr.MeshAssessmentIDAfter), db.EscString(string(vr.Verdict)), db.EscString(string(b)))
	if err := d.Exec(sql); err != nil {
		return "", err
	}
	_ = d.Exec(fmt.Sprintf(`UPDATE plan_executions SET updated_at='%s', status='completed' WHERE execution_id='%s';`,
		db.EscString(vr.ValidatedAt), db.EscString(executionID)))
	return vr.ValidationID, nil
}

func ListValidationsForExecution(d *db.DB, executionID string) ([]ValidationResult, error) {
	if d == nil {
		return nil, nil
	}
	rows, err := d.QueryRows(fmt.Sprintf(
		`SELECT payload_json FROM plan_validations WHERE execution_id='%s' ORDER BY validated_at DESC;`,
		db.EscString(executionID)))
	if err != nil {
		return nil, err
	}
	var out []ValidationResult
	for _, row := range rows {
		var vr ValidationResult
		if err := json.Unmarshal([]byte(fmt.Sprint(row["payload_json"])), &vr); err != nil {
			continue
		}
		out = append(out, vr)
	}
	return out, nil
}

func GetPlanExecution(d *db.DB, executionID string) (PlanExecutionRecord, bool, error) {
	if d == nil {
		return PlanExecutionRecord{}, false, fmt.Errorf("database unavailable")
	}
	rows, err := d.QueryRows(fmt.Sprintf(
		`SELECT execution_id, plan_id, plan_graph_hash, mesh_assessment_id, COALESCE(baseline_metrics_json,'{}') AS baseline_metrics_json, status, started_at, updated_at, observation_horizon_hours, notes FROM plan_executions WHERE execution_id='%s' LIMIT 1;`,
		db.EscString(executionID)))
	if err != nil {
		return PlanExecutionRecord{}, false, err
	}
	if len(rows) == 0 {
		return PlanExecutionRecord{}, false, nil
	}
	row := rows[0]
	var base PostChangeMetricsSnapshot
	_ = json.Unmarshal([]byte(fmt.Sprint(row["baseline_metrics_json"])), &base)
	return PlanExecutionRecord{
		ExecutionID:             fmt.Sprint(row["execution_id"]),
		PlanID:                  fmt.Sprint(row["plan_id"]),
		PlanGraphHash:           fmt.Sprint(row["plan_graph_hash"]),
		MeshAssessmentID:        fmt.Sprint(row["mesh_assessment_id"]),
		BaselineMetrics:         base,
		Status:                  fmt.Sprint(row["status"]),
		StartedAt:               fmt.Sprint(row["started_at"]),
		UpdatedAt:                 fmt.Sprint(row["updated_at"]),
		ObservationHorizonHours: int(asInt64(row["observation_horizon_hours"])),
		Notes:                   fmt.Sprint(row["notes"]),
	}, true, nil
}

func ListPlanExecutions(d *db.DB, planID string, limit int) ([]PlanExecutionRecord, error) {
	if d == nil {
		return nil, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := d.QueryRows(fmt.Sprintf(
		`SELECT execution_id, plan_id, plan_graph_hash, mesh_assessment_id, COALESCE(baseline_metrics_json,'{}') AS baseline_metrics_json, status, started_at, updated_at, observation_horizon_hours, notes FROM plan_executions WHERE plan_id='%s' ORDER BY started_at DESC LIMIT %d;`,
		db.EscString(planID), limit))
	if err != nil {
		return nil, err
	}
	var out []PlanExecutionRecord
	for _, row := range rows {
		var base PostChangeMetricsSnapshot
		_ = json.Unmarshal([]byte(fmt.Sprint(row["baseline_metrics_json"])), &base)
		out = append(out, PlanExecutionRecord{
			ExecutionID:             fmt.Sprint(row["execution_id"]),
			PlanID:                  fmt.Sprint(row["plan_id"]),
			PlanGraphHash:           fmt.Sprint(row["plan_graph_hash"]),
			MeshAssessmentID:        fmt.Sprint(row["mesh_assessment_id"]),
			BaselineMetrics:         base,
			Status:                  fmt.Sprint(row["status"]),
			StartedAt:               fmt.Sprint(row["started_at"]),
			UpdatedAt:                 fmt.Sprint(row["updated_at"]),
			ObservationHorizonHours: int(asInt64(row["observation_horizon_hours"])),
			Notes:                   fmt.Sprint(row["notes"]),
		})
	}
	return out, nil
}

// RecordRecommendationOutcome stores a compact outcome row for retrospectives.
func RecordRecommendationOutcome(d *db.DB, key, graphHash, assessmentID string, verdict OutcomeVerdict, payload any) error {
	if d == nil {
		return fmt.Errorf("database unavailable")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("recommendation_key required")
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	oid := "rout-" + randomID()
	now := nowRFC3339()
	sql := fmt.Sprintf(`INSERT INTO recommendation_outcomes(outcome_id, recommendation_key, graph_hash, mesh_assessment_id, verdict, recorded_at, payload_json)
		VALUES('%s','%s','%s','%s','%s','%s','%s');`,
		db.EscString(oid), db.EscString(key), db.EscString(graphHash), db.EscString(assessmentID), db.EscString(string(verdict)), db.EscString(now), db.EscString(string(b)))
	return d.Exec(sql)
}

func RecommendationRetrospectiveForKey(d *db.DB, key string) (RecommendationRetrospective, error) {
	out := RecommendationRetrospective{RecommendationKey: key}
	if d == nil {
		return out, nil
	}
	rows, err := d.QueryRows(fmt.Sprintf(
		`SELECT verdict, COUNT(*) AS c FROM recommendation_outcomes WHERE recommendation_key='%s' GROUP BY verdict;`,
		db.EscString(key)))
	if err != nil {
		return out, err
	}
	for _, row := range rows {
		v := fmt.Sprint(row["verdict"])
		c := int(asInt64(row["c"]))
		out.TotalRecorded += c
		switch OutcomeVerdict(v) {
		case OutcomeVerdictSupported:
			out.SuccessCount += c
		case OutcomeVerdictContradicted:
			out.ContradictedCount += c
		default:
			out.InconclusiveCount += c
		}
	}
	return out, nil
}

// PruneRecommendationOutcomes keeps the newest maxRows rows globally.
func PruneRecommendationOutcomes(d *db.DB, maxRows int) error {
	if d == nil {
		return nil
	}
	if maxRows <= 0 {
		maxRows = 500
	}
	sql := fmt.Sprintf(`DELETE FROM recommendation_outcomes WHERE outcome_id NOT IN (
		SELECT outcome_id FROM recommendation_outcomes ORDER BY recorded_at DESC LIMIT %d
	);`, maxRows)
	return d.Exec(sql)
}

// PrunePlanExecutions keeps newest maxRows execution records.
func PrunePlanExecutions(d *db.DB, maxRows int) error {
	if d == nil {
		return nil
	}
	if maxRows <= 0 {
		maxRows = 300
	}
	sql := fmt.Sprintf(`DELETE FROM plan_executions WHERE execution_id NOT IN (
		SELECT execution_id FROM plan_executions ORDER BY started_at DESC LIMIT %d
	);`, maxRows)
	return d.Exec(sql)
}
