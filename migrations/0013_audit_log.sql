-- Migration: 0013_audit_log.sql
-- Description: Add audit_log table for action attribution
-- Created: 2026-03-21

-- Audit log table for tracking all operator actions
-- This establishes the foundation for action attribution and compliance logging

CREATE TABLE IF NOT EXISTS audit_log (
    id TEXT PRIMARY KEY,
    timestamp TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    action_class TEXT NOT NULL,
    action_detail TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    reason TEXT,
    result TEXT NOT NULL,
    details TEXT,
    session_id TEXT,
    remote_addr TEXT,
    created_at TEXT DEFAULT (datetime('now'))
);

-- Indexes for efficient audit log querying
CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp ON audit_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_log_actor_id ON audit_log(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_action_class ON audit_log(action_class);
CREATE INDEX IF NOT EXISTS idx_audit_log_resource ON audit_log(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_session ON audit_log(session_id);

-- Record migration
INSERT OR IGNORE INTO schema_migrations (version, applied_at) VALUES ('0013_audit_log', datetime('now'));
