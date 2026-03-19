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
DELETE FROM transport_alerts WHERE active=0 AND resolved_at != '' AND resolved_at < '%s';
COMMIT;`, esc(cutoffSQL), maxRows, esc(cutoffSQL))
	return d.Exec(sql)
}
