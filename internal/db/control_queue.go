package db

import (
	"fmt"
	"strings"
	"time"
)

const (
	ControlPlaneKeyExecutorHeartbeat = "control_executor_last_heartbeat_at"
	ControlPlaneKeyExecutorKind      = "control_executor_last_kind"
)

// TouchControlExecutorHeartbeat records that an in-process control executor loop
// is active (mel serve). kind is e.g. serve_loop.
func (d *DB) TouchControlExecutorHeartbeat(kind string, now time.Time) error {
	if d == nil {
		return nil
	}
	ts := now.UTC().Format(time.RFC3339)
	k := strings.TrimSpace(kind)
	if k == "" {
		k = "serve_loop"
	}
	if err := d.UpsertControlPlaneState(ControlPlaneKeyExecutorHeartbeat, ts, "executor", ts); err != nil {
		return err
	}
	return d.UpsertControlPlaneState(ControlPlaneKeyExecutorKind, k, "executor", ts)
}

// ControlActionQueueMetrics returns counts and oldest-created timestamps for queue insight.
func (d *DB) ControlActionQueueMetrics() (map[string]any, error) {
	if d == nil {
		return map[string]any{}, nil
	}
	out := map[string]any{
		"queued_lifecycle_pending_count":              0,
		"awaiting_approval_count":                       0,
		"approved_waiting_executor_count":               0,
		"oldest_queued_pending_created_at":              "",
		"oldest_approved_waiting_executor_created_at":   "",
	}
	type rowAgg struct {
		key   string
		query string
	}
	queries := []rowAgg{
		{"queued_lifecycle_pending_count", `SELECT COUNT(*) AS c FROM control_actions WHERE lifecycle_state='pending' AND result != 'approved';`},
		{"awaiting_approval_count", `SELECT COUNT(*) AS c FROM control_actions WHERE lifecycle_state='pending_approval';`},
		{"approved_waiting_executor_count", `SELECT COUNT(*) AS c FROM control_actions WHERE lifecycle_state='pending' AND result='approved';`},
	}
	for _, q := range queries {
		rows, err := d.QueryRows(q.query)
		if err != nil {
			return nil, err
		}
		if len(rows) > 0 {
			out[q.key] = int(asInt(rows[0]["c"]))
		}
	}
	oldestPending, err := d.QueryRows(`SELECT COALESCE(MIN(created_at),'') AS t FROM control_actions WHERE lifecycle_state='pending' AND result != 'approved';`)
	if err != nil {
		return nil, err
	}
	if len(oldestPending) > 0 {
		out["oldest_queued_pending_created_at"] = asString(oldestPending[0]["t"])
	}
	oldestApprovedWait, err := d.QueryRows(`SELECT COALESCE(MIN(created_at),'') AS t FROM control_actions WHERE lifecycle_state='pending' AND result='approved';`)
	if err != nil {
		return nil, err
	}
	if len(oldestApprovedWait) > 0 {
		out["oldest_approved_waiting_executor_created_at"] = asString(oldestApprovedWait[0]["t"])
	}
	return out, nil
}

// ExecutorPresence returns heartbeat and inferred activity state for operators.
func (d *DB) ExecutorPresence(now time.Time) (map[string]any, error) {
	if d == nil {
		return map[string]any{
			"executor_activity": "unknown",
			"note":                "database unavailable",
		}, nil
	}
	hb, err := d.GetControlPlaneState(ControlPlaneKeyExecutorHeartbeat)
	if err != nil {
		return nil, err
	}
	kind, _ := d.GetControlPlaneState(ControlPlaneKeyExecutorKind)
	activity := "unknown"
	note := ""
	if strings.TrimSpace(hb) != "" {
		t, perr := time.Parse(time.RFC3339, hb)
		if perr == nil && now.Sub(t) < 2*time.Minute {
			activity = "active"
			note = "recent executor heartbeat from mel serve (or equivalent loop)"
		} else if perr == nil {
			activity = "inactive"
			note = fmt.Sprintf("last executor heartbeat %s is stale relative to now", hb)
		}
	} else {
		note = "no executor heartbeat recorded; run mel serve for continuous queue draining or use CLI one-shot processing"
	}
	return map[string]any{
		"executor_activity":              activity,
		"executor_last_heartbeat_at":     hb,
		"executor_last_reported_kind":    kind,
		"executor_heartbeat_basis":       "control_plane_state",
		"executor_presence_note":         note,
		"backlog_requires_active_executor": true,
	}, nil
}
