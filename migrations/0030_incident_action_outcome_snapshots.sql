-- Migration: 0030_incident_action_outcome_snapshots.sql
-- Description: Persistent per-action outcome evaluation snapshots for incident intelligence traceability.
-- Created: 2026-03-29

CREATE TABLE IF NOT EXISTS incident_action_outcome_snapshots (
    snapshot_id TEXT PRIMARY KEY,
    signature_key TEXT NOT NULL,
    incident_id TEXT NOT NULL,
    action_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    action_label TEXT NOT NULL DEFAULT '',
    derived_classification TEXT NOT NULL,
    evidence_sufficiency TEXT NOT NULL,
    pre_action_summary_json TEXT NOT NULL DEFAULT '{}',
    post_action_summary_json TEXT NOT NULL DEFAULT '{}',
    observed_signal_count INTEGER NOT NULL DEFAULT 0,
    caveats_json TEXT NOT NULL DEFAULT '[]',
    inspect_before_reuse_json TEXT NOT NULL DEFAULT '[]',
    evidence_refs_json TEXT NOT NULL DEFAULT '[]',
    association_only INTEGER NOT NULL DEFAULT 1,
    derivation_window_start TEXT NOT NULL,
    derivation_window_end TEXT NOT NULL,
    derivation_version TEXT NOT NULL DEFAULT 'incident_action_outcome_eval/v1',
    schema_version TEXT NOT NULL DEFAULT '1.0.0',
    derived_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(signature_key, incident_id, action_id)
);

CREATE INDEX IF NOT EXISTS idx_action_outcome_snapshots_signature_action
    ON incident_action_outcome_snapshots(signature_key, action_type, derived_at DESC);

CREATE INDEX IF NOT EXISTS idx_action_outcome_snapshots_incident
    ON incident_action_outcome_snapshots(incident_id, derived_at DESC);

INSERT OR IGNORE INTO schema_migrations(version, applied_at)
    VALUES ('0030_incident_action_outcome_snapshots', datetime('now'));
