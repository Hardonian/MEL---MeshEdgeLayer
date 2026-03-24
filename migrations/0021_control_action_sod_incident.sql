-- Migration: 0021_control_action_sod_incident.sql
-- Description: Separation-of-duties signals, submitted-by identity, incident linkage,
--   execution-started timestamp, optional break-glass SoD bypass audit fields.
-- Created: 2026-03-23

-- submitted_by: canonical actor who queued/submitted the action (operator or system)
ALTER TABLE control_actions ADD COLUMN submitted_by TEXT NOT NULL DEFAULT 'system';

-- requires_separate_approver: persisted policy bit; true when SoD applies at approval time
ALTER TABLE control_actions ADD COLUMN requires_separate_approver INTEGER NOT NULL DEFAULT 0;

-- incident_id: optional FK-style reference to incidents.id for first-class linkage
ALTER TABLE control_actions ADD COLUMN incident_id TEXT;

-- execution_started_at: when executor began running (distinct from executed_at / completed_at)
ALTER TABLE control_actions ADD COLUMN execution_started_at TEXT;

-- sod_bypass: explicit break-glass same-actor approval (auditable; not set on normal path)
ALTER TABLE control_actions ADD COLUMN sod_bypass INTEGER NOT NULL DEFAULT 0;
ALTER TABLE control_actions ADD COLUMN sod_bypass_actor TEXT;
ALTER TABLE control_actions ADD COLUMN sod_bypass_reason TEXT;

CREATE INDEX IF NOT EXISTS idx_control_actions_incident
    ON control_actions(incident_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_control_actions_submitted_by
    ON control_actions(submitted_by, created_at DESC);

-- Backfill: historical approval-gated rows should show SoD requirement
UPDATE control_actions
SET requires_separate_approver = 1
WHERE execution_mode = 'approval_required'
  AND requires_separate_approver = 0;

INSERT OR IGNORE INTO schema_migrations(version, applied_at)
    VALUES ('0021_control_action_sod_incident', datetime('now'));
