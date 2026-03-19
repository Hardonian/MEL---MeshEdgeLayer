ALTER TABLE control_actions ADD COLUMN lifecycle_state TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE control_actions ADD COLUMN advisory_only INTEGER NOT NULL DEFAULT 0;
ALTER TABLE control_actions ADD COLUMN denial_code TEXT;
ALTER TABLE control_actions ADD COLUMN closure_state TEXT;

ALTER TABLE control_decisions ADD COLUMN denial_code TEXT;

CREATE INDEX IF NOT EXISTS idx_control_actions_lifecycle_target ON control_actions(lifecycle_state, target_transport, action_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_control_actions_expires_at ON control_actions(expires_at, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_control_actions_result_created_at ON control_actions(result, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_control_decisions_denial_code ON control_decisions(denial_code, created_at DESC);

CREATE TABLE IF NOT EXISTS control_action_reality (
  action_type TEXT PRIMARY KEY,
  actuator_exists INTEGER NOT NULL DEFAULT 0,
  reversible INTEGER NOT NULL DEFAULT 0,
  blast_radius_known INTEGER NOT NULL DEFAULT 0,
  blast_radius_class TEXT NOT NULL DEFAULT 'unknown',
  safe_for_guarded_auto INTEGER NOT NULL DEFAULT 0,
  advisory_only INTEGER NOT NULL DEFAULT 0,
  denial_code TEXT,
  notes TEXT,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_control_action_reality_advisory_only ON control_action_reality(advisory_only, action_type);

INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0011_control_closure', datetime('now'));
