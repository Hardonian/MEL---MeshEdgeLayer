-- Migration 0023: mesh intelligence incident hook state (streak counters; singleton row)
INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0023_mesh_intel_state', datetime('now'));

CREATE TABLE IF NOT EXISTS mesh_intel_state (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  last_viability TEXT NOT NULL DEFAULT '',
  consecutive_bad INTEGER NOT NULL DEFAULT 0,
  last_good_viability TEXT NOT NULL DEFAULT '',
  last_good_readiness REAL NOT NULL DEFAULT 0,
  last_incident_fingerprint TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO mesh_intel_state(id) VALUES (1);
