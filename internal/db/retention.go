package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/selfobs"
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
	if err := d.Exec(sql); err != nil {
		return err
	}
	selfobs.MarkFresh("retention")
	return nil
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

// PruneImportedRemoteEvidence removes stale imported remote-evidence rows and enforces
// bounded storage on import batches and item rows.
func (d *DB) PruneImportedRemoteEvidence(cutoff time.Time, maxRows int) error {
	if d == nil {
		return nil
	}
	if maxRows <= 0 {
		maxRows = 50000
	}
	cutoffSQL := cutoff.UTC().Format(time.RFC3339)
	sql := fmt.Sprintf(`BEGIN IMMEDIATE;
DELETE FROM imported_remote_evidence WHERE imported_at < '%s';
DELETE FROM imported_remote_evidence WHERE id IN (
	SELECT id FROM imported_remote_evidence ORDER BY imported_at DESC, id DESC LIMIT -1 OFFSET %d
);
DELETE FROM remote_import_batches WHERE imported_at < '%s' AND id NOT IN (
	SELECT DISTINCT batch_id FROM imported_remote_evidence
);
DELETE FROM remote_import_batches WHERE id IN (
	SELECT id FROM remote_import_batches ORDER BY imported_at DESC, id DESC LIMIT -1 OFFSET %d
) AND id NOT IN (
	SELECT DISTINCT batch_id FROM imported_remote_evidence
);
DELETE FROM timeline_events
WHERE import_id != ''
  AND event_time < '%s'
  AND import_id NOT IN (SELECT id FROM imported_remote_evidence)
  AND import_id NOT IN (SELECT id FROM remote_import_batches);
COMMIT;`, esc(cutoffSQL), maxRows, esc(cutoffSQL), maxRows, esc(cutoffSQL))
	if err := d.Exec(sql); err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil
		}
		return err
	}
	return nil
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
