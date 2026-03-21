-- Migration: 0018_timeline_events.sql
-- Description: Explicit timeline_events table for operator-visible control-plane
--   events that cannot be derived purely from other table queries.
--   Complements the UNION-query view in TimelineEvents() with durable
--   first-class event rows that survive retention pruning of source tables.
-- Created: 2026-03-21

-- ─── Timeline events ─────────────────────────────────────────────────────────
-- event_type values (non-exhaustive; stored as text for extensibility):
--   action_approved       — operator approved a pending-approval action
--   action_rejected       — operator rejected a pending-approval action
--   action_expired        — pending-approval action expired without approval
--   action_executed       — control action executed successfully
--   action_failed         — control action execution failed
--   action_rolled_back    — action was rolled back
--   freeze_created        — operator created an automation freeze
--   freeze_cleared        — operator cleared an automation freeze
--   freeze_expired        — freeze expired automatically
--   maintenance_created   — maintenance window created
--   maintenance_cancelled — maintenance window cancelled
--   control_stale         — control loop detected as stale
--   approval_backlog_warn — approval backlog exceeded warning threshold
--   system_started        — MEL control plane started
--   operator_note         — operator added a note (mirrored for timeline)
CREATE TABLE IF NOT EXISTS timeline_events (
    id          TEXT PRIMARY KEY,
    event_time  TEXT NOT NULL DEFAULT (datetime('now')),
    event_type  TEXT NOT NULL,
    summary     TEXT NOT NULL DEFAULT '',
    severity    TEXT NOT NULL DEFAULT 'info',  -- info | warning | error | critical
    actor_id    TEXT NOT NULL DEFAULT 'system',
    resource_id TEXT NOT NULL DEFAULT '',       -- action_id / freeze_id / window_id etc.
    details_json TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_timeline_events_time
    ON timeline_events(event_time DESC);
CREATE INDEX IF NOT EXISTS idx_timeline_events_type
    ON timeline_events(event_type, event_time DESC);
CREATE INDEX IF NOT EXISTS idx_timeline_events_resource
    ON timeline_events(resource_id, event_time DESC);

-- ─── Record migration ─────────────────────────────────────────────────────────
INSERT OR IGNORE INTO schema_migrations(version, applied_at)
    VALUES ('0018_timeline_events', datetime('now'));
