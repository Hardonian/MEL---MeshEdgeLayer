CREATE TABLE IF NOT EXISTS transport_alerts (
  id TEXT PRIMARY KEY,
  transport_name TEXT NOT NULL,
  transport_type TEXT NOT NULL,
  severity TEXT NOT NULL,
  reason TEXT NOT NULL,
  summary TEXT NOT NULL,
  first_triggered_at TEXT NOT NULL,
  last_updated_at TEXT NOT NULL,
  resolved_at TEXT,
  active INTEGER NOT NULL DEFAULT 1,
  episode_id TEXT,
  cluster_key TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_transport_alerts_transport_active ON transport_alerts(transport_name, active, last_updated_at);

CREATE TABLE IF NOT EXISTS transport_health_snapshots (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  transport_name TEXT NOT NULL,
  transport_type TEXT NOT NULL,
  score INTEGER NOT NULL,
  state TEXT NOT NULL,
  snapshot_time TEXT NOT NULL,
  active_alert_count INTEGER NOT NULL DEFAULT 0,
  dead_letter_count_window INTEGER NOT NULL DEFAULT 0,
  observation_drop_count_window INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_transport_health_snapshots_transport_time ON transport_health_snapshots(transport_name, snapshot_time DESC);

INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0007_transport_intelligence', datetime('now'));
