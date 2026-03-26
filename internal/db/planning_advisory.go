package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// UpsertPlanningAdvisoryAlert persists a synthetic transport alert row for planning advisories.
// Uses transport_name="planning" and transport_type="advisory" to distinguish from real transports.
func (d *DB) UpsertPlanningAdvisoryAlert(id, severity, reason, summary, clusterKey string, contributing []string, trigger string) error {
	if d == nil || strings.TrimSpace(id) == "" {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	contribJSON, _ := json.Marshal(contributing)
	sql := fmt.Sprintf(`INSERT INTO transport_alerts(id,transport_name,transport_type,severity,reason,summary,first_triggered_at,last_updated_at,resolved_at,active,episode_id,cluster_key,contributing_reasons_json,cluster_reference,penalty_snapshot_json,trigger_condition)
VALUES('%s','planning','advisory','%s','%s','%s','%s','%s',NULL,1,NULL,'%s','%s','','[]','%s')
ON CONFLICT(id) DO UPDATE SET severity=excluded.severity,reason=excluded.reason,summary=excluded.summary,last_updated_at=excluded.last_updated_at,resolved_at=NULL,active=1,cluster_key=excluded.cluster_key,contributing_reasons_json=excluded.contributing_reasons_json,trigger_condition=excluded.trigger_condition;`,
		esc(id), esc(severity), esc(reason), esc(summary), esc(now), esc(now), esc(clusterKey), esc(string(contribJSON)), esc(trigger))
	return d.Exec(sql)
}

// ResolvePlanningAdvisoryAlertsNotIn marks inactive advisory alerts not in the active set.
// ListPlanningAdvisoryAlerts returns active synthetic planning advisories.
func (d *DB) ListPlanningAdvisoryAlerts(activeOnly bool) ([]TransportAlertRecord, error) {
	if d == nil {
		return nil, nil
	}
	activeClause := ""
	if activeOnly {
		activeClause = " AND active=1"
	}
	rows, err := d.QueryRows(fmt.Sprintf(
		`SELECT id, transport_name, transport_type, severity, reason, summary, first_triggered_at, last_updated_at, resolved_at, active, episode_id, cluster_key, contributing_reasons_json, cluster_reference, penalty_snapshot_json, trigger_condition FROM transport_alerts WHERE transport_name='planning' AND transport_type='advisory'%s ORDER BY last_updated_at DESC LIMIT 200;`,
		activeClause))
	if err != nil {
		return nil, err
	}
	var out []TransportAlertRecord
	for _, row := range rows {
		out = append(out, alertRecordFromRow(row))
	}
	return out, nil
}

func (d *DB) ResolvePlanningAdvisoryAlertsNotIn(activeIDs []string, resolvedAt string) error {
	if d == nil {
		return nil
	}
	if resolvedAt == "" {
		resolvedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if len(activeIDs) == 0 {
		sql := fmt.Sprintf(`UPDATE transport_alerts SET active=0, resolved_at='%s', last_updated_at='%s' WHERE transport_name='planning' AND transport_type='advisory' AND active=1;`,
			esc(resolvedAt), esc(resolvedAt))
		return d.Exec(sql)
	}
	var parts []string
	for _, id := range activeIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		parts = append(parts, "'"+esc(id)+"'")
	}
	if len(parts) == 0 {
		return nil
	}
	inClause := strings.Join(parts, ",")
	sql := fmt.Sprintf(`UPDATE transport_alerts SET active=0, resolved_at='%s', last_updated_at='%s' WHERE transport_name='planning' AND transport_type='advisory' AND active=1 AND id NOT IN (%s);`,
		esc(resolvedAt), esc(resolvedAt), inClause)
	return d.Exec(sql)
}
