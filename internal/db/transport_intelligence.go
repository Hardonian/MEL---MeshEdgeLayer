package db

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type PenaltyRecord struct {
	Reason  string `json:"reason"`
	Penalty int    `json:"penalty"`
	Count   uint64 `json:"count"`
	Window  string `json:"window"`
}

type TransportAlertRecord struct {
	ID                  string          `json:"id"`
	TransportName       string          `json:"transport_name"`
	TransportType       string          `json:"transport_type"`
	Severity            string          `json:"severity"`
	Reason              string          `json:"reason"`
	Summary             string          `json:"summary"`
	FirstTriggeredAt    string          `json:"first_triggered_at"`
	LastUpdatedAt       string          `json:"last_updated_at"`
	ResolvedAt          string          `json:"resolved_at,omitempty"`
	Active              bool            `json:"active"`
	EpisodeID           string          `json:"episode_id,omitempty"`
	ClusterKey          string          `json:"cluster_key"`
	ContributingReasons []string        `json:"contributing_reasons,omitempty"`
	ClusterReference    string          `json:"cluster_reference,omitempty"`
	PenaltySnapshot     []PenaltyRecord `json:"penalty_snapshot,omitempty"`
	TriggerCondition    string          `json:"trigger_condition,omitempty"`
}

type TransportHealthSnapshot struct {
	TransportName              string `json:"transport_name"`
	TransportType              string `json:"transport_type"`
	Score                      int    `json:"score"`
	State                      string `json:"state"`
	SnapshotTime               string `json:"snapshot_time"`
	ActiveAlertCount           int    `json:"active_alert_count"`
	DeadLetterCountWindow      int    `json:"dead_letter_count_window"`
	ObservationDropCountWindow int    `json:"observation_drop_count_window"`
}

type TransportAnomalyHistoryPoint struct {
	BucketStart      string            `json:"bucket_start"`
	TransportName    string            `json:"transport_name"`
	TransportType    string            `json:"transport_type"`
	Reason           string            `json:"reason"`
	Count            uint64            `json:"count"`
	DeadLetters      uint64            `json:"dead_letters"`
	ObservationDrops uint64            `json:"observation_drops"`
	DropCauses       map[string]uint64 `json:"drop_causes,omitempty"`
}

func (d *DB) UpsertTransportAlert(alert TransportAlertRecord) error {
	if strings.TrimSpace(alert.ID) == "" {
		return fmt.Errorf("transport alert id is required")
	}
	if alert.FirstTriggeredAt == "" {
		alert.FirstTriggeredAt = time.Now().UTC().Format(time.RFC3339)
	}
	if alert.LastUpdatedAt == "" {
		alert.LastUpdatedAt = alert.FirstTriggeredAt
	}
	contributingJSON, _ := json.Marshal(alert.ContributingReasons)
	penaltyJSON, _ := json.Marshal(alert.PenaltySnapshot)
	sql := fmt.Sprintf(`INSERT INTO transport_alerts(id,transport_name,transport_type,severity,reason,summary,first_triggered_at,last_updated_at,resolved_at,active,episode_id,cluster_key,contributing_reasons_json,cluster_reference,penalty_snapshot_json,trigger_condition)
VALUES('%s','%s','%s','%s','%s','%s','%s','%s',NULL,%d,%s,'%s','%s','%s','%s','%s')
ON CONFLICT(id) DO UPDATE SET transport_name=excluded.transport_name,transport_type=excluded.transport_type,severity=excluded.severity,reason=excluded.reason,summary=excluded.summary,last_updated_at=excluded.last_updated_at,resolved_at=NULL,active=excluded.active,episode_id=excluded.episode_id,cluster_key=excluded.cluster_key,contributing_reasons_json=excluded.contributing_reasons_json,cluster_reference=excluded.cluster_reference,penalty_snapshot_json=excluded.penalty_snapshot_json,trigger_condition=excluded.trigger_condition;`,
		esc(alert.ID), esc(alert.TransportName), esc(alert.TransportType), esc(alert.Severity), esc(alert.Reason), esc(alert.Summary), esc(alert.FirstTriggeredAt), esc(alert.LastUpdatedAt), boolInt(alert.Active), sqlString(alert.EpisodeID), esc(alert.ClusterKey), esc(string(contributingJSON)), esc(alert.ClusterReference), esc(string(penaltyJSON)), esc(alert.TriggerCondition))
	return d.Exec(sql)
}

func (d *DB) ResolveTransportAlertsNotIn(transportName string, activeIDs []string, resolvedAt string) error {
	if strings.TrimSpace(transportName) == "" {
		return nil
	}
	clauses := []string{fmt.Sprintf("transport_name='%s'", esc(transportName)), "active=1"}
	if len(activeIDs) > 0 {
		exclusions := make([]string, 0, len(activeIDs))
		for _, id := range activeIDs {
			exclusions = append(exclusions, fmt.Sprintf("'%s'", esc(id)))
		}
		clauses = append(clauses, fmt.Sprintf("id NOT IN (%s)", strings.Join(exclusions, ",")))
	}
	sql := fmt.Sprintf("UPDATE transport_alerts SET active=0, resolved_at='%s', last_updated_at='%s' WHERE %s;", esc(resolvedAt), esc(resolvedAt), strings.Join(clauses, " AND "))
	return d.Exec(sql)
}

func (d *DB) TransportAlerts(activeOnly bool) ([]TransportAlertRecord, error) {
	query := "SELECT id, transport_name, transport_type, severity, reason, summary, first_triggered_at, last_updated_at, COALESCE(resolved_at,'') AS resolved_at, active, COALESCE(episode_id,'') AS episode_id, COALESCE(cluster_key,'') AS cluster_key, COALESCE(contributing_reasons_json,'[]') AS contributing_reasons_json, COALESCE(cluster_reference,'') AS cluster_reference, COALESCE(penalty_snapshot_json,'[]') AS penalty_snapshot_json, COALESCE(trigger_condition,'') AS trigger_condition FROM transport_alerts"
	if activeOnly {
		query += " WHERE active=1"
	}
	query += " ORDER BY active DESC, last_updated_at DESC, first_triggered_at DESC;"
	rows, err := d.QueryRows(query)
	if err != nil {
		return nil, err
	}
	out := make([]TransportAlertRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, alertRecordFromRow(row))
	}
	return out, nil
}

func alertRecordFromRow(row map[string]any) TransportAlertRecord {
	var reasons []string
	var penalties []PenaltyRecord
	_ = json.Unmarshal([]byte(asString(row["contributing_reasons_json"])), &reasons)
	_ = json.Unmarshal([]byte(asString(row["penalty_snapshot_json"])), &penalties)
	return TransportAlertRecord{
		ID:                  asString(row["id"]),
		TransportName:       asString(row["transport_name"]),
		TransportType:       asString(row["transport_type"]),
		Severity:            asString(row["severity"]),
		Reason:              asString(row["reason"]),
		Summary:             asString(row["summary"]),
		FirstTriggeredAt:    asString(row["first_triggered_at"]),
		LastUpdatedAt:       asString(row["last_updated_at"]),
		ResolvedAt:          asString(row["resolved_at"]),
		Active:              asInt(row["active"]) == 1,
		EpisodeID:           asString(row["episode_id"]),
		ClusterKey:          asString(row["cluster_key"]),
		ContributingReasons: reasons,
		ClusterReference:    asString(row["cluster_reference"]),
		PenaltySnapshot:     penalties,
		TriggerCondition:    asString(row["trigger_condition"]),
	}
}

func (d *DB) TransportAlertByID(id string) (TransportAlertRecord, bool, error) {
	rows, err := d.QueryRows(fmt.Sprintf("SELECT id, transport_name, transport_type, severity, reason, summary, first_triggered_at, last_updated_at, COALESCE(resolved_at,'') AS resolved_at, active, COALESCE(episode_id,'') AS episode_id, COALESCE(cluster_key,'') AS cluster_key, COALESCE(contributing_reasons_json,'[]') AS contributing_reasons_json, COALESCE(cluster_reference,'') AS cluster_reference, COALESCE(penalty_snapshot_json,'[]') AS penalty_snapshot_json, COALESCE(trigger_condition,'') AS trigger_condition FROM transport_alerts WHERE id='%s' LIMIT 1;", esc(id)))
	if err != nil {
		return TransportAlertRecord{}, false, err
	}
	if len(rows) == 0 {
		return TransportAlertRecord{}, false, nil
	}
	return alertRecordFromRow(rows[0]), true, nil
}

func (d *DB) LatestTransportHealthSnapshots() (map[string]TransportHealthSnapshot, error) {
	rows, err := d.QueryRows(`SELECT s.transport_name, s.transport_type, s.score, s.state, s.snapshot_time, s.active_alert_count, s.dead_letter_count_window, s.observation_drop_count_window
FROM transport_health_snapshots s
INNER JOIN (
	SELECT transport_name, MAX(snapshot_time) AS snapshot_time
	FROM transport_health_snapshots
	GROUP BY transport_name
) latest ON latest.transport_name = s.transport_name AND latest.snapshot_time = s.snapshot_time;`)
	if err != nil {
		return nil, err
	}
	out := make(map[string]TransportHealthSnapshot, len(rows))
	for _, row := range rows {
		out[asString(row["transport_name"])] = snapshotFromRow(row)
	}
	return out, nil
}

func (d *DB) InsertTransportHealthSnapshot(snapshot TransportHealthSnapshot) error {
	sql := fmt.Sprintf(`INSERT INTO transport_health_snapshots(transport_name,transport_type,score,state,snapshot_time,active_alert_count,dead_letter_count_window,observation_drop_count_window)
VALUES('%s','%s',%d,'%s','%s',%d,%d,%d);`,
		esc(snapshot.TransportName), esc(snapshot.TransportType), snapshot.Score, esc(snapshot.State), esc(snapshot.SnapshotTime), snapshot.ActiveAlertCount, snapshot.DeadLetterCountWindow, snapshot.ObservationDropCountWindow)
	return d.Exec(sql)
}

func (d *DB) ActiveAlertCounts() (map[string]int, error) {
	rows, err := d.QueryRows("SELECT transport_name, COUNT(*) AS alert_count FROM transport_alerts WHERE active=1 GROUP BY transport_name;")
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, row := range rows {
		out[asString(row["transport_name"])] = int(asInt(row["alert_count"]))
	}
	return out, nil
}

func (d *DB) TransportHealthSnapshots(name, start, end string, limit, offset int) ([]TransportHealthSnapshot, error) {
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT transport_name, transport_type, score, state, snapshot_time, active_alert_count, dead_letter_count_window, observation_drop_count_window
FROM transport_health_snapshots WHERE %s ORDER BY snapshot_time DESC LIMIT %d OFFSET %d;`, historyFilter("transport_name", name, "snapshot_time", start, end), limit, offset))
	if err != nil {
		return nil, err
	}
	out := make([]TransportHealthSnapshot, 0, len(rows))
	for _, row := range rows {
		out = append(out, snapshotFromRow(row))
	}
	return out, nil
}

func snapshotFromRow(row map[string]any) TransportHealthSnapshot {
	return TransportHealthSnapshot{
		TransportName:              asString(row["transport_name"]),
		TransportType:              asString(row["transport_type"]),
		Score:                      int(asInt(row["score"])),
		State:                      asString(row["state"]),
		SnapshotTime:               asString(row["snapshot_time"]),
		ActiveAlertCount:           int(asInt(row["active_alert_count"])),
		DeadLetterCountWindow:      int(asInt(row["dead_letter_count_window"])),
		ObservationDropCountWindow: int(asInt(row["observation_drop_count_window"])),
	}
}

func (d *DB) TransportAlertsHistory(name, start, end string, limit, offset int) ([]TransportAlertRecord, error) {
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id, transport_name, transport_type, severity, reason, summary, first_triggered_at, last_updated_at, COALESCE(resolved_at,'') AS resolved_at, active, COALESCE(episode_id,'') AS episode_id, COALESCE(cluster_key,'') AS cluster_key, COALESCE(contributing_reasons_json,'[]') AS contributing_reasons_json, COALESCE(cluster_reference,'') AS cluster_reference, COALESCE(penalty_snapshot_json,'[]') AS penalty_snapshot_json, COALESCE(trigger_condition,'') AS trigger_condition
FROM transport_alerts WHERE %s ORDER BY last_updated_at DESC, first_triggered_at DESC LIMIT %d OFFSET %d;`, historyFilter("transport_name", name, "last_updated_at", start, end), limit, offset))
	if err != nil {
		return nil, err
	}
	out := make([]TransportAlertRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, alertRecordFromRow(row))
	}
	return out, nil
}

func (d *DB) TransportAnomalyHistory(name, start, end string, limit, offset int) ([]TransportAnomalyHistoryPoint, error) {
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT bucket_start, transport_name, transport_type, reason, SUM(count) AS count, SUM(dead_letters) AS dead_letters, SUM(observation_drops) AS observation_drops, json_group_object(drop_cause, drop_count) AS drop_causes_json
FROM (
	SELECT strftime('%%Y-%%m-%%dT%%H:%%M:00Z', created_at) AS bucket_start,
	       COALESCE(json_extract(details_json,'$.transport'), '') AS transport_name,
	       COALESCE(json_extract(details_json,'$.type'), '') AS transport_type,
	       message AS reason,
	       1 AS count,
	       0 AS dead_letters,
	       COALESCE(json_extract(details_json,'$.drop_count'), json_extract(details_json,'$.details.drop_count'), 0) AS observation_drops,
	       COALESCE(json_extract(details_json,'$.drop_cause'), json_extract(details_json,'$.details.drop_cause'), '') AS drop_cause,
	       COALESCE(json_extract(details_json,'$.drop_count'), json_extract(details_json,'$.details.drop_count'), 0) AS drop_count,
	       created_at
	FROM audit_logs WHERE category='transport'
	UNION ALL
	SELECT strftime('%%Y-%%m-%%dT%%H:%%M:00Z', created_at) AS bucket_start,
	       transport_name,
	       transport_type,
	       reason,
	       1 AS count,
	       1 AS dead_letters,
	       0 AS observation_drops,
	       '' AS drop_cause,
	       0 AS drop_count,
	       created_at
	FROM dead_letters
) evidence
WHERE %s AND transport_name != ''
GROUP BY bucket_start, transport_name, transport_type, reason
ORDER BY bucket_start DESC, transport_name, reason LIMIT %d OFFSET %d;`, historyFilter("transport_name", name, "created_at", start, end), limit, offset))
	if err != nil {
		return nil, err
	}
	out := make([]TransportAnomalyHistoryPoint, 0, len(rows))
	for _, row := range rows {
		point := TransportAnomalyHistoryPoint{
			BucketStart:      asString(row["bucket_start"]),
			TransportName:    asString(row["transport_name"]),
			TransportType:    asString(row["transport_type"]),
			Reason:           asString(row["reason"]),
			Count:            uint64(asInt(row["count"])),
			DeadLetters:      uint64(asInt(row["dead_letters"])),
			ObservationDrops: uint64(asInt(row["observation_drops"])),
			DropCauses:       map[string]uint64{},
		}
		_ = json.Unmarshal([]byte(asString(row["drop_causes_json"])), &point.DropCauses)
		out = append(out, point)
	}
	return out, nil
}

func historyFilter(nameColumn, name, timeColumn, start, end string) string {
	clauses := []string{"1=1"}
	if strings.TrimSpace(name) != "" {
		clauses = append(clauses, fmt.Sprintf("%s='%s'", nameColumn, esc(name)))
	}
	if strings.TrimSpace(start) != "" {
		clauses = append(clauses, fmt.Sprintf("%s >= '%s'", timeColumn, esc(start)))
	}
	if strings.TrimSpace(end) != "" {
		clauses = append(clauses, fmt.Sprintf("%s <= '%s'", timeColumn, esc(end)))
	}
	return strings.Join(clauses, " AND ")
}

func (d *DB) LatestTransportSnapshotBefore(name, before string) (TransportHealthSnapshot, bool, error) {
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT transport_name, transport_type, score, state, snapshot_time, active_alert_count, dead_letter_count_window, observation_drop_count_window
FROM transport_health_snapshots WHERE transport_name='%s' AND snapshot_time <= '%s' ORDER BY snapshot_time DESC LIMIT 1;`, esc(name), esc(before)))
	if err != nil {
		return TransportHealthSnapshot{}, false, err
	}
	if len(rows) == 0 {
		return TransportHealthSnapshot{}, false, nil
	}
	return snapshotFromRow(rows[0]), true, nil
}

func SortedAlertIDs(alerts []TransportAlertRecord) []string {
	ids := make([]string, 0, len(alerts))
	for _, alert := range alerts {
		ids = append(ids, alert.ID)
	}
	sort.Strings(ids)
	return ids
}
