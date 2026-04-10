CREATE TABLE IF NOT EXISTS organizations (
  id TEXT PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS workspaces (
  id TEXT PRIMARY KEY,
  organization_id TEXT NOT NULL,
  slug TEXT NOT NULL,
  name TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(organization_id, slug),
  FOREIGN KEY (organization_id) REFERENCES organizations(id)
);

CREATE TABLE IF NOT EXISTS workspace_memberships (
  workspace_id TEXT NOT NULL,
  actor_id TEXT NOT NULL,
  role TEXT NOT NULL CHECK(role IN ('owner', 'admin', 'operator', 'viewer')),
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (workspace_id, actor_id),
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id)
);

CREATE TABLE IF NOT EXISTS devices (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  canonical_key TEXT NOT NULL,
  display_name TEXT,
  status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','relinked','replaced','merged','retired')),
  replacement_for_device_id TEXT,
  merged_into_device_id TEXT,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(workspace_id, canonical_key),
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id)
);

CREATE TABLE IF NOT EXISTS device_alias_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  workspace_id TEXT NOT NULL,
  device_id TEXT NOT NULL,
  alias TEXT NOT NULL,
  changed_by TEXT NOT NULL,
  changed_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (device_id) REFERENCES devices(id)
);

CREATE TABLE IF NOT EXISTS device_hardware_identities (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  workspace_id TEXT NOT NULL,
  device_id TEXT NOT NULL,
  identity_type TEXT NOT NULL CHECK(identity_type IN ('public_key','hardware_id','node_id')),
  identity_value TEXT NOT NULL,
  active INTEGER NOT NULL DEFAULT 1,
  bound_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  unbound_at TEXT,
  UNIQUE(workspace_id, identity_type, identity_value, active),
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (device_id) REFERENCES devices(id)
);

CREATE TABLE IF NOT EXISTS config_templates (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  name TEXT NOT NULL,
  version INTEGER NOT NULL DEFAULT 1,
  template_json TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','deprecated','archived')),
  created_by TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(workspace_id, name, version),
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id)
);

CREATE TABLE IF NOT EXISTS device_template_assignments (
  workspace_id TEXT NOT NULL,
  device_id TEXT NOT NULL,
  template_id TEXT NOT NULL,
  assigned_by TEXT NOT NULL,
  assigned_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (workspace_id, device_id),
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (device_id) REFERENCES devices(id),
  FOREIGN KEY (template_id) REFERENCES config_templates(id)
);

CREATE TABLE IF NOT EXISTS device_backups (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  device_id TEXT NOT NULL,
  backup_reason TEXT NOT NULL,
  snapshot_json TEXT NOT NULL,
  created_by TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (device_id) REFERENCES devices(id)
);

CREATE TABLE IF NOT EXISTS rollout_jobs (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  template_id TEXT,
  action TEXT NOT NULL CHECK(action IN ('apply_template','rollback')),
  state TEXT NOT NULL CHECK(state IN ('draft','scheduled','in_progress','succeeded','partial_failure','rolled_back','failed','cancelled')),
  target_scope TEXT NOT NULL CHECK(target_scope IN ('workspace','tag','selected_devices')),
  scheduled_for TEXT,
  created_by TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (template_id) REFERENCES config_templates(id)
);

CREATE TABLE IF NOT EXISTS rollout_job_targets (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  rollout_job_id TEXT NOT NULL,
  device_id TEXT NOT NULL,
  state TEXT NOT NULL CHECK(state IN ('queued','sent','acknowledged','skipped_offline','expired','retryable','failed','rolled_back')),
  retry_count INTEGER NOT NULL DEFAULT 0,
  failure_reason TEXT,
  ack_at TEXT,
  last_attempt_at TEXT,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(rollout_job_id, device_id),
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (rollout_job_id) REFERENCES rollout_jobs(id),
  FOREIGN KEY (device_id) REFERENCES devices(id)
);

CREATE TABLE IF NOT EXISTS alerts (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  device_id TEXT,
  rule_kind TEXT NOT NULL CHECK(rule_kind IN ('offline_too_long','degraded_quality','low_battery','rollout_failure')),
  severity TEXT NOT NULL CHECK(severity IN ('info','warning','critical')),
  state TEXT NOT NULL CHECK(state IN ('open','acknowledged','resolved')),
  title TEXT NOT NULL,
  detail TEXT,
  triggered_at TEXT NOT NULL,
  acknowledged_by TEXT,
  acknowledged_at TEXT,
  resolved_at TEXT,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (device_id) REFERENCES devices(id)
);

CREATE TABLE IF NOT EXISTS alert_events (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  alert_id TEXT NOT NULL,
  event_type TEXT NOT NULL CHECK(event_type IN ('triggered','acknowledged','reopened','resolved')),
  actor_id TEXT,
  detail TEXT,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id),
  FOREIGN KEY (alert_id) REFERENCES alerts(id)
);

CREATE TABLE IF NOT EXISTS workspace_audit_logs (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  actor_id TEXT NOT NULL,
  action_type TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id TEXT NOT NULL,
  payload_json TEXT,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (workspace_id) REFERENCES workspaces(id)
);

CREATE INDEX IF NOT EXISTS idx_workspaces_org ON workspaces(organization_id);
CREATE INDEX IF NOT EXISTS idx_memberships_actor ON workspace_memberships(actor_id);
CREATE INDEX IF NOT EXISTS idx_devices_workspace_status ON devices(workspace_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_alias_history_device ON device_alias_history(workspace_id, device_id, changed_at DESC);
CREATE INDEX IF NOT EXISTS idx_device_hw_identity ON device_hardware_identities(workspace_id, identity_type, identity_value);
CREATE INDEX IF NOT EXISTS idx_templates_workspace ON config_templates(workspace_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_rollouts_workspace_state ON rollout_jobs(workspace_id, state, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_rollout_targets_job_state ON rollout_job_targets(rollout_job_id, state, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_backups_device ON device_backups(workspace_id, device_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_workspace_state ON alerts(workspace_id, state, triggered_at DESC);
CREATE INDEX IF NOT EXISTS idx_alert_events_alert ON alert_events(alert_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_workspace_audit_workspace_created ON workspace_audit_logs(workspace_id, created_at DESC);

INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0036_fleet_control_plane', datetime('now'));
