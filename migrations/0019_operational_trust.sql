-- Migration: 0019_operational_trust.sql
-- Operational trust plane: instance metadata, backup provenance, audit hash chain columns.

CREATE TABLE IF NOT EXISTS instance_metadata (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Recorded when mel backup create succeeds (used by upgrade preflight).
CREATE TABLE IF NOT EXISTS backup_metadata (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  backup_time TEXT NOT NULL,
  bundle_path TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Tamper-evident chain for audit_logs (NULL on rows created before this migration).
ALTER TABLE audit_logs ADD COLUMN chain_prev_hash TEXT;
ALTER TABLE audit_logs ADD COLUMN content_hash TEXT;
ALTER TABLE audit_logs ADD COLUMN chain_hash TEXT;

INSERT OR IGNORE INTO schema_migrations (version, applied_at) VALUES ('0019_operational_trust', datetime('now'));
