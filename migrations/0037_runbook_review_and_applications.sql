-- Migration: 0037_runbook_review_and_applications.sql
-- Description: Close the remediation-memory loop by making incident_runbook_entries
--   reviewable, promotable, deprecatable, and by recording every time a runbook is
--   applied to an incident. Application rows are durable audit truth that feed
--   runbook effectiveness aggregates and shift handoff packets.
-- Created: 2026-04-11

-- Reviewable lifecycle fields on incident_runbook_entries
ALTER TABLE incident_runbook_entries ADD COLUMN applied_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE incident_runbook_entries ADD COLUMN last_applied_at TEXT NOT NULL DEFAULT '';
ALTER TABLE incident_runbook_entries ADD COLUMN last_applied_incident_id TEXT NOT NULL DEFAULT '';
ALTER TABLE incident_runbook_entries ADD COLUMN deprecated_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE incident_runbook_entries ADD COLUMN promoted_at TEXT NOT NULL DEFAULT '';
ALTER TABLE incident_runbook_entries ADD COLUMN promoted_by_actor_id TEXT NOT NULL DEFAULT '';
ALTER TABLE incident_runbook_entries ADD COLUMN deprecated_at TEXT NOT NULL DEFAULT '';
ALTER TABLE incident_runbook_entries ADD COLUMN deprecated_by_actor_id TEXT NOT NULL DEFAULT '';
ALTER TABLE incident_runbook_entries ADD COLUMN useful_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE incident_runbook_entries ADD COLUMN ineffective_count INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_runbook_entries_status ON incident_runbook_entries(status, updated_at DESC);

-- Runbook application history. One row per operator attaching a runbook to a specific
-- incident, with an explicit outcome. This is the compounding remediation-memory trail.
CREATE TABLE IF NOT EXISTS incident_runbook_applications (
    id TEXT PRIMARY KEY,
    runbook_id TEXT NOT NULL,
    incident_id TEXT NOT NULL,
    actor_id TEXT NOT NULL DEFAULT 'system',
    outcome TEXT NOT NULL DEFAULT 'applied',
    note TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    FOREIGN KEY (incident_id) REFERENCES incidents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_runbook_apps_runbook
    ON incident_runbook_applications(runbook_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_runbook_apps_incident
    ON incident_runbook_applications(incident_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_runbook_apps_outcome
    ON incident_runbook_applications(runbook_id, outcome, created_at DESC);

INSERT OR IGNORE INTO schema_migrations (version, applied_at)
    VALUES ('0037_runbook_review_and_applications', datetime('now'));
