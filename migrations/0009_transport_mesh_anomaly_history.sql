CREATE TABLE IF NOT EXISTS transport_anomaly_snapshots (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  bucket_start TEXT NOT NULL,
  transport_name TEXT NOT NULL,
  transport_type TEXT NOT NULL,
  reason TEXT NOT NULL,
  count INTEGER NOT NULL DEFAULT 0,
  dead_letters INTEGER NOT NULL DEFAULT 0,
  observation_drops INTEGER NOT NULL DEFAULT 0,
  drop_causes_json TEXT NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_transport_anomaly_snapshots_bucket_transport_reason ON transport_anomaly_snapshots(bucket_start, transport_name, reason);
CREATE INDEX IF NOT EXISTS idx_transport_anomaly_snapshots_transport_time ON transport_anomaly_snapshots(transport_name, bucket_start DESC);
CREATE INDEX IF NOT EXISTS idx_transport_anomaly_snapshots_time ON transport_anomaly_snapshots(bucket_start DESC, transport_name);

INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0009_transport_mesh_anomaly_history', datetime('now'));
