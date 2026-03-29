-- Migration: 0029_incident_intelligence.sql
-- Description: Persistent recurring-incident signatures and incident/signature linkage.
-- Created: 2026-03-29

CREATE TABLE IF NOT EXISTS incident_signatures (
    signature_key TEXT PRIMARY KEY,
    signature_label TEXT NOT NULL,
    category TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    reason_key TEXT NOT NULL DEFAULT '',
    first_seen_at TEXT NOT NULL,
    last_seen_at TEXT NOT NULL,
    match_count INTEGER NOT NULL DEFAULT 1,
    example_incident_id TEXT NOT NULL,
    last_summary TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_incident_signatures_last_seen
    ON incident_signatures(last_seen_at DESC);

CREATE TABLE IF NOT EXISTS incident_signature_incidents (
    signature_key TEXT NOT NULL,
    incident_id TEXT NOT NULL,
    linked_at TEXT NOT NULL,
    PRIMARY KEY (signature_key, incident_id)
);

CREATE INDEX IF NOT EXISTS idx_incident_signature_incidents_incident
    ON incident_signature_incidents(incident_id, linked_at DESC);

INSERT OR IGNORE INTO schema_migrations(version, applied_at)
    VALUES ('0029_incident_intelligence', datetime('now'));
