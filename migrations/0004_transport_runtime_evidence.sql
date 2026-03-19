CREATE TABLE IF NOT EXISTS transport_runtime_evidence (
  transport_name TEXT PRIMARY KEY,
  last_heartbeat_at TEXT,
  packets_dropped INTEGER NOT NULL DEFAULT 0,
  reconnect_attempts INTEGER NOT NULL DEFAULT 0,
  consecutive_timeouts INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0004_transport_runtime_evidence', datetime('now'));
