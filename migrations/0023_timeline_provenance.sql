-- Migration: 0023_timeline_provenance.sql
-- Description: Adds canonical provenance, merge, and timing posture to operator events,
--   allowing MEL to become a truthful federated evidence substrate without
--   losing origin or timing context.
-- Created: 2026-03-27

ALTER TABLE timeline_events ADD COLUMN scope_posture TEXT NOT NULL DEFAULT 'local';
ALTER TABLE timeline_events ADD COLUMN origin_instance_id TEXT NOT NULL DEFAULT '';
ALTER TABLE timeline_events ADD COLUMN timing_posture TEXT NOT NULL DEFAULT 'local_ordered';
ALTER TABLE timeline_events ADD COLUMN merge_disposition TEXT NOT NULL DEFAULT 'raw_only';
ALTER TABLE timeline_events ADD COLUMN merge_correlation_id TEXT NOT NULL DEFAULT '';
ALTER TABLE timeline_events ADD COLUMN import_id TEXT NOT NULL DEFAULT '';

INSERT OR IGNORE INTO schema_migrations(version, applied_at)
    VALUES ('0023_timeline_provenance', datetime('now'));
