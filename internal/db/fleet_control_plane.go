package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type WorkspaceRole string

const (
	WorkspaceRoleOwner    WorkspaceRole = "owner"
	WorkspaceRoleAdmin    WorkspaceRole = "admin"
	WorkspaceRoleOperator WorkspaceRole = "operator"
	WorkspaceRoleViewer   WorkspaceRole = "viewer"
)

var workspaceRoleRank = map[WorkspaceRole]int{
	WorkspaceRoleViewer:   1,
	WorkspaceRoleOperator: 2,
	WorkspaceRoleAdmin:    3,
	WorkspaceRoleOwner:    4,
}

type DeviceIdentityInput struct {
	WorkspaceID   string `json:"workspace_id"`
	DeviceID      string `json:"device_id"`
	CanonicalKey  string `json:"canonical_key"`
	DisplayName   string `json:"display_name"`
	IdentityType  string `json:"identity_type"`
	IdentityValue string `json:"identity_value"`
	ActorID       string `json:"actor_id"`
	AllowRelink   bool   `json:"allow_relink"`
}

type DeviceConflict struct {
	State         string `json:"state"`
	ExistingID    string `json:"existing_device_id,omitempty"`
	RequestedID   string `json:"requested_device_id,omitempty"`
	IdentityType  string `json:"identity_type,omitempty"`
	IdentityValue string `json:"identity_value,omitempty"`
}

type RolloutTargetInput struct {
	TargetID   string
	DeviceID   string
	Initial    string
	Offline    bool
	RetryCount int
}

type FleetDashboardDevice struct {
	DeviceID         string  `json:"device_id"`
	DisplayName      string  `json:"display_name,omitempty"`
	Status           string  `json:"status"`
	LastHeard        string  `json:"last_heard,omitempty"`
	BatteryPct       float64 `json:"battery_pct,omitempty"`
	RouteQuality     float64 `json:"route_quality,omitempty"`
	RecentFailures   int     `json:"recent_failures"`
	OpenRollouts     int     `json:"open_rollouts"`
	OpenAlerts       int     `json:"open_alerts"`
	HealthScore      float64 `json:"health_score"`
	HealthScoreBasis string  `json:"health_score_basis"`
}

func (d *DB) EnsureWorkspaceMembership(workspaceID, actorID string, minRole WorkspaceRole) error {
	if d == nil {
		return fmt.Errorf(ErrDatabaseUnavailable)
	}
	if strings.TrimSpace(workspaceID) == "" || strings.TrimSpace(actorID) == "" {
		return fmt.Errorf("workspace_id and actor_id are required")
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT role FROM workspace_memberships WHERE workspace_id=%s AND actor_id=%s LIMIT 1;`, sqlq(workspaceID), sqlq(actorID)))
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("workspace membership not found")
	}
	role := WorkspaceRole(asString(rows[0]["role"]))
	if workspaceRoleRank[role] < workspaceRoleRank[minRole] {
		return fmt.Errorf("workspace role %s is not sufficient", role)
	}
	return nil
}

func (d *DB) UpsertDeviceIdentity(in DeviceIdentityInput) (*DeviceConflict, error) {
	if d == nil {
		return nil, fmt.Errorf(ErrDatabaseUnavailable)
	}
	if strings.TrimSpace(in.WorkspaceID) == "" || strings.TrimSpace(in.DeviceID) == "" || strings.TrimSpace(in.CanonicalKey) == "" || strings.TrimSpace(in.ActorID) == "" {
		return nil, fmt.Errorf("workspace_id, device_id, canonical_key, and actor_id are required")
	}
	if err := d.EnsureWorkspaceMembership(in.WorkspaceID, in.ActorID, WorkspaceRoleOperator); err != nil {
		return nil, err
	}
	if in.IdentityType != "" && in.IdentityType != "public_key" && in.IdentityType != "hardware_id" && in.IdentityType != "node_id" {
		return nil, fmt.Errorf("identity_type must be public_key, hardware_id, or node_id")
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id FROM devices WHERE workspace_id=%s AND canonical_key=%s LIMIT 1;`, sqlq(in.WorkspaceID), sqlq(in.CanonicalKey)))
	if err != nil {
		return nil, err
	}
	if len(rows) > 0 && asString(rows[0]["id"]) != in.DeviceID {
		return &DeviceConflict{State: "canonical_conflict", ExistingID: asString(rows[0]["id"]), RequestedID: in.DeviceID}, nil
	}
	if in.IdentityType != "" && strings.TrimSpace(in.IdentityValue) != "" {
		rows, err = d.QueryRows(fmt.Sprintf(`SELECT device_id FROM device_hardware_identities WHERE workspace_id=%s AND identity_type=%s AND identity_value=%s AND active=1 LIMIT 1;`, sqlq(in.WorkspaceID), sqlq(in.IdentityType), sqlq(in.IdentityValue)))
		if err != nil {
			return nil, err
		}
		if len(rows) > 0 && asString(rows[0]["device_id"]) != in.DeviceID {
			if !in.AllowRelink {
				return &DeviceConflict{State: "identity_conflict", ExistingID: asString(rows[0]["device_id"]), RequestedID: in.DeviceID, IdentityType: in.IdentityType, IdentityValue: in.IdentityValue}, nil
			}
			if err := d.Exec(fmt.Sprintf(`UPDATE device_hardware_identities SET active=0, unbound_at=datetime('now') WHERE workspace_id=%s AND identity_type=%s AND identity_value=%s AND active=1;`, sqlq(in.WorkspaceID), sqlq(in.IdentityType), sqlq(in.IdentityValue))); err != nil {
				return nil, err
			}
		}
	}
	if err := d.Exec(fmt.Sprintf(`
INSERT INTO devices(id,workspace_id,canonical_key,display_name,status,created_at,updated_at)
VALUES (%s,%s,%s,%s,'active',datetime('now'),datetime('now'))
ON CONFLICT(id) DO UPDATE SET canonical_key=excluded.canonical_key, display_name=excluded.display_name, updated_at=datetime('now');
`, sqlq(in.DeviceID), sqlq(in.WorkspaceID), sqlq(in.CanonicalKey), sqlq(in.DisplayName))); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.DisplayName) != "" {
		if err := d.Exec(fmt.Sprintf(`INSERT INTO device_alias_history(workspace_id, device_id, alias, changed_by, changed_at) VALUES (%s,%s,%s,%s,datetime('now'));`, sqlq(in.WorkspaceID), sqlq(in.DeviceID), sqlq(in.DisplayName), sqlq(in.ActorID))); err != nil {
			return nil, err
		}
	}
	if in.IdentityType != "" && strings.TrimSpace(in.IdentityValue) != "" {
		if err := d.Exec(fmt.Sprintf(`INSERT OR REPLACE INTO device_hardware_identities(workspace_id,device_id,identity_type,identity_value,active,bound_at) VALUES (%s,%s,%s,%s,1,datetime('now'));`, sqlq(in.WorkspaceID), sqlq(in.DeviceID), sqlq(in.IdentityType), sqlq(in.IdentityValue))); err != nil {
			return nil, err
		}
	}
	_ = d.InsertWorkspaceAuditLog(in.WorkspaceID, in.ActorID, "device.identity.upsert", "device", in.DeviceID, map[string]any{"canonical_key": in.CanonicalKey, "display_name": in.DisplayName, "identity_type": in.IdentityType, "allow_relink": in.AllowRelink})
	return nil, nil
}

func (d *DB) InsertWorkspaceAuditLog(workspaceID, actorID, actionType, targetType, targetID string, payload map[string]any) error {
	if d == nil {
		return fmt.Errorf(ErrDatabaseUnavailable)
	}
	enc := "{}"
	if payload != nil {
		b, _ := json.Marshal(payload)
		enc = string(b)
	}
	id := fmt.Sprintf("wal_%d", time.Now().UTC().UnixNano())
	return d.Exec(fmt.Sprintf(`INSERT INTO workspace_audit_logs(id,workspace_id,actor_id,action_type,target_type,target_id,payload_json,created_at) VALUES (%s,%s,%s,%s,%s,%s,%s,datetime('now'));`, sqlq(id), sqlq(workspaceID), sqlq(actorID), sqlq(actionType), sqlq(targetType), sqlq(targetID), sqlq(enc)))
}

func (d *DB) CreateRolloutJob(workspaceID, actorID, jobID, templateID, action, scope, scheduledFor string, targets []RolloutTargetInput) error {
	if err := d.EnsureWorkspaceMembership(workspaceID, actorID, WorkspaceRoleOperator); err != nil {
		return err
	}
	if action != "apply_template" && action != "rollback" {
		return fmt.Errorf("unsupported rollout action")
	}
	if scope == "" {
		scope = "selected_devices"
	}
	if err := d.Exec(fmt.Sprintf(`INSERT INTO rollout_jobs(id,workspace_id,template_id,action,state,target_scope,scheduled_for,created_by,created_at,updated_at) VALUES (%s,%s,%s,%s,'draft',%s,%s,%s,datetime('now'),datetime('now'));`,
		sqlq(jobID), sqlq(workspaceID), sqlq(templateID), sqlq(action), sqlq(scope), sqlq(scheduledFor), sqlq(actorID))); err != nil {
		return err
	}
	for _, t := range targets {
		state := t.Initial
		if state == "" {
			state = "queued"
		}
		if t.Offline {
			state = "skipped_offline"
		}
		if err := d.Exec(fmt.Sprintf(`INSERT INTO rollout_job_targets(id,workspace_id,rollout_job_id,device_id,state,retry_count,created_at,updated_at) VALUES (%s,%s,%s,%s,%s,%d,datetime('now'),datetime('now'));`,
			sqlq(t.TargetID), sqlq(workspaceID), sqlq(jobID), sqlq(t.DeviceID), sqlq(state), t.RetryCount)); err != nil {
			return err
		}
	}
	_ = d.UpdateRolloutJobState(workspaceID, jobID)
	_ = d.InsertWorkspaceAuditLog(workspaceID, actorID, "rollout.created", "rollout_job", jobID, map[string]any{"target_count": len(targets), "action": action})
	return nil
}

func (d *DB) UpdateRolloutTargetState(workspaceID, actorID, targetID, newState, reason string, incrementRetry bool) error {
	if err := d.EnsureWorkspaceMembership(workspaceID, actorID, WorkspaceRoleOperator); err != nil {
		return err
	}
	if newState == "" {
		return fmt.Errorf("state required")
	}
	retrySQL := "retry_count"
	if incrementRetry {
		retrySQL = "retry_count + 1"
	}
	if err := d.Exec(fmt.Sprintf(`UPDATE rollout_job_targets SET state=%s, failure_reason=%s, retry_count=%s, ack_at=CASE WHEN %s='acknowledged' THEN datetime('now') ELSE ack_at END, updated_at=datetime('now') WHERE workspace_id=%s AND id=%s;`,
		sqlq(newState), sqlq(reason), retrySQL, sqlq(newState), sqlq(workspaceID), sqlq(targetID))); err != nil {
		return err
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT rollout_job_id FROM rollout_job_targets WHERE workspace_id=%s AND id=%s LIMIT 1;`, sqlq(workspaceID), sqlq(targetID)))
	if err == nil && len(rows) > 0 {
		_ = d.UpdateRolloutJobState(workspaceID, asString(rows[0]["rollout_job_id"]))
	}
	return nil
}

func (d *DB) UpdateRolloutJobState(workspaceID, jobID string) error {
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT state, COUNT(*) AS c FROM rollout_job_targets WHERE workspace_id=%s AND rollout_job_id=%s GROUP BY state;`, sqlq(workspaceID), sqlq(jobID)))
	if err != nil {
		return err
	}
	counts := map[string]int64{}
	total := int64(0)
	for _, r := range rows {
		c := asInt(r["c"])
		counts[asString(r["state"])] = c
		total += c
	}
	jobState := "draft"
	switch {
	case total == 0:
		jobState = "draft"
	case counts["acknowledged"] == total:
		jobState = "succeeded"
	case counts["rolled_back"] == total:
		jobState = "rolled_back"
	case counts["failed"] > 0 || counts["retryable"] > 0 || counts["skipped_offline"] > 0:
		if counts["acknowledged"] > 0 {
			jobState = "partial_failure"
		} else {
			jobState = "failed"
		}
	case counts["sent"] > 0 || counts["queued"] > 0:
		jobState = "in_progress"
	default:
		jobState = "scheduled"
	}
	return d.Exec(fmt.Sprintf(`UPDATE rollout_jobs SET state=%s, updated_at=datetime('now') WHERE workspace_id=%s AND id=%s;`, sqlq(jobState), sqlq(workspaceID), sqlq(jobID)))
}

func (d *DB) ListFleetDashboard(workspaceID, actorID string, limit, offset int) ([]FleetDashboardDevice, error) {
	if err := d.EnsureWorkspaceMembership(workspaceID, actorID, WorkspaceRoleViewer); err != nil {
		return nil, err
	}
	rows, err := d.QueryRows(fmt.Sprintf(`
SELECT
  d.id,
  COALESCE(d.display_name,'') AS display_name,
  d.status,
  COALESCE(d.updated_at,'') AS last_heard,
  COALESCE((SELECT CAST(json_extract(ts.value_json, '$.battery_level') AS REAL)
            FROM telemetry_samples ts
            WHERE ts.node_num = (SELECT node_num FROM nodes n WHERE n.node_id=d.id LIMIT 1)
            ORDER BY ts.observed_at DESC LIMIT 1), -1) AS battery_pct,
  COALESCE((SELECT CAST(json_extract(ts.value_json, '$.snr') AS REAL)
            FROM telemetry_samples ts
            WHERE ts.node_num = (SELECT node_num FROM nodes n WHERE n.node_id=d.id LIMIT 1)
            ORDER BY ts.observed_at DESC LIMIT 1), -1) AS route_quality,
  (SELECT COUNT(*) FROM rollout_job_targets t WHERE t.workspace_id=d.workspace_id AND t.device_id=d.id AND t.state IN ('failed','retryable','skipped_offline')) AS recent_failures,
  (SELECT COUNT(*) FROM rollout_job_targets t WHERE t.workspace_id=d.workspace_id AND t.device_id=d.id AND t.state IN ('queued','sent','retryable')) AS open_rollouts,
  (SELECT COUNT(*) FROM alerts a WHERE a.workspace_id=d.workspace_id AND a.device_id=d.id AND a.state='open') AS open_alerts
FROM devices d
WHERE d.workspace_id=%s
ORDER BY d.updated_at DESC
LIMIT %d OFFSET %d;
`, sqlq(workspaceID), clamp(limit, 1, 500), max(offset, 0)))
	if err != nil {
		return nil, err
	}
	out := make([]FleetDashboardDevice, 0, len(rows))
	for _, r := range rows {
		battery := asFloat(r["battery_pct"])
		route := asFloat(r["route_quality"])
		failures := asInt(r["recent_failures"])
		score := 1.0
		basis := []string{"baseline=1.0"}
		if battery >= 0 {
			score -= maxf(0, (25.0-battery)/100.0)
			basis = append(basis, fmt.Sprintf("battery=%.1f", battery))
		}
		if route >= 0 {
			score -= maxf(0, (5.0-route)/20.0)
			basis = append(basis, fmt.Sprintf("route=%.1f", route))
		}
		score -= float64(min64(failures, 5)) * 0.08
		if score < 0 {
			score = 0
		}
		out = append(out, FleetDashboardDevice{
			DeviceID:         asString(r["id"]),
			DisplayName:      asString(r["display_name"]),
			Status:           asString(r["status"]),
			LastHeard:        asString(r["last_heard"]),
			BatteryPct:       battery,
			RouteQuality:     route,
			RecentFailures:   int(failures),
			OpenRollouts:     int(asInt(r["open_rollouts"])),
			OpenAlerts:       int(asInt(r["open_alerts"])),
			HealthScore:      score,
			HealthScoreBasis: strings.Join(basis, ";"),
		})
	}
	return out, nil
}

func (d *DB) TriggerAlert(workspaceID, actorID, alertID, deviceID, ruleKind, severity, title, detail string) error {
	if err := d.EnsureWorkspaceMembership(workspaceID, actorID, WorkspaceRoleOperator); err != nil {
		return err
	}
	if err := d.Exec(fmt.Sprintf(`INSERT INTO alerts(id,workspace_id,device_id,rule_kind,severity,state,title,detail,triggered_at) VALUES (%s,%s,%s,%s,%s,'open',%s,%s,datetime('now'));`, sqlq(alertID), sqlq(workspaceID), sqlq(deviceID), sqlq(ruleKind), sqlq(severity), sqlq(title), sqlq(detail))); err != nil {
		return err
	}
	if err := d.Exec(fmt.Sprintf(`INSERT INTO alert_events(id,workspace_id,alert_id,event_type,actor_id,detail,created_at) VALUES (%s,%s,%s,'triggered',%s,%s,datetime('now'));`, sqlq(fmt.Sprintf("ae_%d", time.Now().UTC().UnixNano())), sqlq(workspaceID), sqlq(alertID), sqlq(actorID), sqlq(detail))); err != nil {
		return err
	}
	return d.InsertWorkspaceAuditLog(workspaceID, actorID, "alert.triggered", "alert", alertID, map[string]any{"rule_kind": ruleKind, "severity": severity})
}

func (d *DB) AcknowledgeAlert(workspaceID, actorID, alertID string) error {
	if err := d.EnsureWorkspaceMembership(workspaceID, actorID, WorkspaceRoleOperator); err != nil {
		return err
	}
	if err := d.Exec(fmt.Sprintf(`UPDATE alerts SET state='acknowledged', acknowledged_by=%s, acknowledged_at=datetime('now') WHERE workspace_id=%s AND id=%s AND state='open';`, sqlq(actorID), sqlq(workspaceID), sqlq(alertID))); err != nil {
		return err
	}
	if err := d.Exec(fmt.Sprintf(`INSERT INTO alert_events(id,workspace_id,alert_id,event_type,actor_id,created_at) VALUES (%s,%s,%s,'acknowledged',%s,datetime('now'));`, sqlq(fmt.Sprintf("ae_%d", time.Now().UTC().UnixNano())), sqlq(workspaceID), sqlq(alertID), sqlq(actorID))); err != nil {
		return err
	}
	return d.InsertWorkspaceAuditLog(workspaceID, actorID, "alert.acknowledged", "alert", alertID, nil)
}

func sqlq(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "''") + "'"
}

func clamp(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}
func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
