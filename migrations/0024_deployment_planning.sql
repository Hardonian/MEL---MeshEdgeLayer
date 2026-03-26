-- Migration 0024: deployment planning / what-if artifacts (bounded retention)
INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0024_deployment_planning', datetime('now'));

CREATE TABLE IF NOT EXISTS deployment_plans (
  plan_id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  intent TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  payload_json TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_deployment_plans_updated ON deployment_plans(updated_at DESC);

CREATE TABLE IF NOT EXISTS planning_artifacts (
  artifact_id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  graph_hash TEXT NOT NULL DEFAULT '',
  assessment_id TEXT NOT NULL DEFAULT '',
  payload_json TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_planning_artifacts_created ON planning_artifacts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_planning_artifacts_kind ON planning_artifacts(kind, created_at DESC);
