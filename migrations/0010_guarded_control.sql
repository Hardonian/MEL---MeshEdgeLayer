CREATE TABLE IF NOT EXISTS control_actions (
  id TEXT PRIMARY KEY,
  decision_id TEXT,
  action_type TEXT NOT NULL,
  target_transport TEXT,
  target_segment TEXT,
  target_node TEXT,
  reason TEXT NOT NULL,
  confidence REAL NOT NULL DEFAULT 0,
  trigger_evidence_json TEXT NOT NULL DEFAULT '[]',
  episode_id TEXT,
  created_at TEXT NOT NULL,
  executed_at TEXT,
  completed_at TEXT,
  result TEXT,
  reversible INTEGER NOT NULL DEFAULT 0,
  expires_at TEXT,
  outcome_detail TEXT,
  mode TEXT NOT NULL DEFAULT 'advisory',
  policy_rule TEXT,
  metadata_json TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_control_actions_created_at ON control_actions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_control_actions_target_transport ON control_actions(target_transport, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_control_actions_action_type ON control_actions(action_type, created_at DESC);

CREATE TABLE IF NOT EXISTS control_decisions (
  id TEXT PRIMARY KEY,
  candidate_action_id TEXT NOT NULL,
  action_type TEXT NOT NULL,
  target_transport TEXT,
  target_segment TEXT,
  reason TEXT NOT NULL,
  confidence REAL NOT NULL DEFAULT 0,
  allowed INTEGER NOT NULL DEFAULT 0,
  denial_reason TEXT,
  safety_checks_json TEXT NOT NULL DEFAULT '{}',
  decision_inputs_json TEXT NOT NULL DEFAULT '{}',
  policy_summary_json TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  mode TEXT NOT NULL DEFAULT 'advisory',
  operator_override INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_control_decisions_created_at ON control_decisions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_control_decisions_target_transport ON control_decisions(target_transport, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_control_decisions_action_type ON control_decisions(action_type, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_messages_transport_rx_time ON messages(transport_name, rx_time DESC);
CREATE INDEX IF NOT EXISTS idx_messages_from_node_rx_time ON messages(from_node, rx_time DESC);
CREATE INDEX IF NOT EXISTS idx_nodes_last_gateway_updated ON nodes(last_gateway_id, updated_at DESC);

INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0010_guarded_control', datetime('now'));
