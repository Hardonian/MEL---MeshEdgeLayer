-- Migration: 0015_incidents.sql
-- Description: Add incidents table for persistent alert management
-- Created: 2026-03-21

CREATE TABLE IF NOT EXISTS incidents (
    id TEXT PRIMARY KEY,
    category TEXT NOT NULL, -- 'transport', 'system', 'security', 'control'
    severity TEXT NOT NULL, -- 'critical', 'high', 'medium', 'low'
    title TEXT NOT NULL,
    summary TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'open', -- 'open', 'acknowledged', 'resolved', 'suppressed'
    actor_id TEXT, -- Who acknowledged/resolved it
    occurred_at TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at TEXT,
    metadata_json TEXT DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_incidents_state ON incidents(state);
CREATE INDEX IF NOT EXISTS idx_incidents_resource ON incidents(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_incidents_occurred ON incidents(occurred_at);

-- Record migration
INSERT OR IGNORE INTO schema_migrations (version, applied_at) VALUES ('0015_incidents', datetime('now'));
