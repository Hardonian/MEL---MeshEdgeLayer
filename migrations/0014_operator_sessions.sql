-- Migration: 0014_operator_sessions.sql
-- Description: Add operator_sessions table for session tracking (future multi-operator support)
-- Created: 2026-03-21

-- Operator sessions table for future multi-operator authentication
-- Currently not actively used - MEL operates in single-operator mode
-- This table provides the foundation for future session-based authentication

CREATE TABLE IF NOT EXISTS operator_sessions (
    session_id TEXT PRIMARY KEY,
    operator_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    last_activity_at TEXT NOT NULL,
    auth_method TEXT NOT NULL,
    remote_addr TEXT,
    user_agent TEXT,
    active INTEGER DEFAULT 1,
    created_at_internal TEXT DEFAULT (datetime('now'))
);

-- Indexes for efficient session querying
CREATE INDEX IF NOT EXISTS idx_operator_sessions_operator ON operator_sessions(operator_id);
CREATE INDEX IF NOT EXISTS idx_operator_sessions_expires ON operator_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_operator_sessions_active ON operator_sessions(active);

-- Cleanup old expired sessions periodically
-- This is a placeholder for future session management

-- Record migration
INSERT OR IGNORE INTO schema_migrations (version, applied_at) VALUES ('0014_operator_sessions', datetime('now'));
