package db

import (
	"fmt"
	"time"
)

func (d *DB) PruneTransportIntelligence(cutoff time.Time, maxRows int) error {
	if d == nil {
		return nil
	}
	if maxRows <= 0 {
		maxRows = 50000
	}
	cutoffSQL := cutoff.UTC().Format(time.RFC3339)
	sql := fmt.Sprintf(`BEGIN IMMEDIATE;
DELETE FROM transport_health_snapshots WHERE snapshot_time < '%s';
DELETE FROM transport_health_snapshots WHERE id IN (
	SELECT id FROM transport_health_snapshots ORDER BY snapshot_time DESC, id DESC LIMIT -1 OFFSET %d
);
DELETE FROM transport_anomaly_snapshots WHERE bucket_start < '%s';
DELETE FROM transport_anomaly_snapshots WHERE id IN (
	SELECT id FROM transport_anomaly_snapshots ORDER BY bucket_start DESC, id DESC LIMIT -1 OFFSET %d
);
DELETE FROM transport_alerts WHERE active=0 AND resolved_at != '' AND resolved_at < '%s';
COMMIT;`, esc(cutoffSQL), maxRows, esc(cutoffSQL), maxRows, esc(cutoffSQL))
	return d.Exec(sql)
}

func (d *DB) PruneControlHistory(cutoff time.Time, maxRows int) error {
	if d == nil {
		return nil
	}
	if maxRows <= 0 {
		maxRows = 50000
	}
	cutoffSQL := cutoff.UTC().Format(time.RFC3339)
	sql := fmt.Sprintf(`BEGIN IMMEDIATE;
DELETE FROM control_actions WHERE created_at < '%s';
DELETE FROM control_actions WHERE id IN (
	SELECT id FROM control_actions ORDER BY created_at DESC, id DESC LIMIT -1 OFFSET %d
);
DELETE FROM control_decisions WHERE created_at < '%s';
DELETE FROM control_decisions WHERE id IN (
	SELECT id FROM control_decisions ORDER BY created_at DESC, id DESC LIMIT -1 OFFSET %d
);
COMMIT;`, esc(cutoffSQL), maxRows, esc(cutoffSQL), maxRows)
	return d.Exec(sql)
}
func (d *DB) PruneAuditLogs(cutoff time.Time) error {
	if d == nil {
		return nil
	}
	return d.Exec(fmt.Sprintf("DELETE FROM audit_logs WHERE created_at < '%s';", cutoff.UTC().Format(time.RFC3339)))
}

func (d *DB) PruneMessages(cutoff time.Time) error {
	if d == nil {
		return nil
	}
	return d.Exec(fmt.Sprintf("DELETE FROM messages WHERE created_at < '%s';", cutoff.UTC().Format(time.RFC3339)))
}

func (d *DB) PruneTelemetry(cutoff time.Time) error {
	if d == nil {
		return nil
	}
	return d.Exec(fmt.Sprintf("DELETE FROM telemetry_samples WHERE observed_at < '%s';", cutoff.UTC().Format(time.RFC3339)))
}
func (d *DB) PruneDeadLetters(cutoff time.Time) error {
	if d == nil {
		return nil
	}
	return d.Exec(fmt.Sprintf("DELETE FROM dead_letters WHERE created_at < '%s';", cutoff.UTC().Format(time.RFC3339)))
}
