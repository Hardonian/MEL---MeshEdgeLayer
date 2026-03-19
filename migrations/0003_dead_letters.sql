CREATE TABLE IF NOT EXISTS dead_letters (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  transport_name TEXT NOT NULL,
  topic TEXT,
  reason TEXT NOT NULL,
  payload_hex TEXT NOT NULL,
  details_json TEXT,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_dead_letters_transport_created_at ON dead_letters(transport_name, created_at);
INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0003_dead_letters', datetime('now'));
