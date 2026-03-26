-- Migration 0025: planning input sets (versioned), execution/validation loop, recommendation outcomes
INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0025_planning_inputs_outcomes_advisory', datetime('now'));

-- Optional link from a saved plan to a versioned input set
ALTER TABLE deployment_plans ADD COLUMN input_set_version_id TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS planning_input_sets (
  input_set_id TEXT PRIMARY KEY,
  title TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_planning_input_sets_updated ON planning_input_sets(updated_at DESC);

CREATE TABLE IF NOT EXISTS planning_input_versions (
  version_id TEXT PRIMARY KEY,
  input_set_id TEXT NOT NULL,
  version_num INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  payload_json TEXT NOT NULL,
  UNIQUE(input_set_id, version_num)
);

CREATE INDEX IF NOT EXISTS idx_planning_input_versions_set ON planning_input_versions(input_set_id, version_num DESC);

CREATE TABLE IF NOT EXISTS plan_executions (
  execution_id TEXT PRIMARY KEY,
  plan_id TEXT NOT NULL,
  plan_graph_hash TEXT NOT NULL DEFAULT '',
  mesh_assessment_id TEXT NOT NULL DEFAULT '',
  baseline_metrics_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'attempted',
  started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  observation_horizon_hours INTEGER NOT NULL DEFAULT 0,
  notes TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_plan_executions_plan ON plan_executions(plan_id, started_at DESC);

CREATE TABLE IF NOT EXISTS plan_step_executions (
  step_execution_id TEXT PRIMARY KEY,
  execution_id TEXT NOT NULL,
  step_id TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'attempted',
  attempted_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  operator_note TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_plan_step_exec ON plan_step_executions(execution_id);

CREATE TABLE IF NOT EXISTS plan_validations (
  validation_id TEXT PRIMARY KEY,
  execution_id TEXT NOT NULL,
  validated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  graph_hash_after TEXT NOT NULL DEFAULT '',
  mesh_assessment_id_after TEXT NOT NULL DEFAULT '',
  verdict TEXT NOT NULL,
  payload_json TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_plan_validations_exec ON plan_validations(execution_id);

CREATE TABLE IF NOT EXISTS recommendation_outcomes (
  outcome_id TEXT PRIMARY KEY,
  recommendation_key TEXT NOT NULL,
  graph_hash TEXT NOT NULL DEFAULT '',
  mesh_assessment_id TEXT NOT NULL DEFAULT '',
  verdict TEXT NOT NULL,
  recorded_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  payload_json TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_recommendation_outcomes_key ON recommendation_outcomes(recommendation_key, recorded_at DESC);
