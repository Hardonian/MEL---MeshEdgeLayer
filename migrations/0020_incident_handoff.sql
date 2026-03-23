-- Migration: 0020_incident_handoff.sql
-- Description: Durable incident collaboration fields (ownership, handoff, risk notes).
-- Created: 2026-03-23

ALTER TABLE incidents ADD COLUMN owner_actor_id TEXT;
ALTER TABLE incidents ADD COLUMN handoff_summary TEXT;
ALTER TABLE incidents ADD COLUMN pending_actions_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE incidents ADD COLUMN recent_actions_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE incidents ADD COLUMN linked_evidence_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE incidents ADD COLUMN risks_json TEXT NOT NULL DEFAULT '[]';

CREATE INDEX IF NOT EXISTS idx_incidents_owner ON incidents(owner_actor_id);

INSERT OR IGNORE INTO schema_migrations (version, applied_at) VALUES ('0020_incident_handoff', datetime('now'));
