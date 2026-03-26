-- Migration 0022: Persisted mesh deployment intelligence snapshots (bootstrap, topology advisory,
-- routing-pressure diagnostics, protocol-fit, ranked recommendations). Auditing / trends only;
-- current operator truth is still computed from live graph + message evidence at refresh time.
INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0022_mesh_intelligence', datetime('now'));

CREATE TABLE IF NOT EXISTS mesh_intelligence_snapshots (
  assessment_id TEXT PRIMARY KEY,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  scope TEXT NOT NULL DEFAULT 'mesh',
  graph_hash TEXT,
  payload_json TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_mesh_intel_created ON mesh_intelligence_snapshots(created_at DESC);
