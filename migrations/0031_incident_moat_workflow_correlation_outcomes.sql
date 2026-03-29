-- Migration: 0031_incident_moat_workflow_correlation_outcomes.sql
-- Description: Incident workflow lock-in fields, operator recommendation outcomes, and persisted cross-incident correlation groups.
-- Created: 2026-03-29

-- Workflow / review state (operator-owned; not control execution)
ALTER TABLE incidents ADD COLUMN review_state TEXT NOT NULL DEFAULT 'open';
ALTER TABLE incidents ADD COLUMN investigation_notes TEXT NOT NULL DEFAULT '';
ALTER TABLE incidents ADD COLUMN resolution_summary TEXT NOT NULL DEFAULT '';
ALTER TABLE incidents ADD COLUMN closeout_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE incidents ADD COLUMN lessons_learned TEXT NOT NULL DEFAULT '';
ALTER TABLE incidents ADD COLUMN reopened_from_incident_id TEXT NOT NULL DEFAULT '';
ALTER TABLE incidents ADD COLUMN reopened_at TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_incidents_review_state ON incidents(review_state);

-- Operator adjudication of assistive recommendations (non-canonical; does not execute control)
CREATE TABLE IF NOT EXISTS incident_recommendation_outcomes (
    id TEXT PRIMARY KEY,
    incident_id TEXT NOT NULL,
    recommendation_id TEXT NOT NULL,
    outcome TEXT NOT NULL,
    actor_id TEXT NOT NULL DEFAULT 'system',
    note TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    FOREIGN KEY (incident_id) REFERENCES incidents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rec_outcomes_incident ON incident_recommendation_outcomes(incident_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_rec_outcomes_rec_id ON incident_recommendation_outcomes(recommendation_id);

-- Deterministic correlation: incidents sharing the same persisted signature key (evidence: signature linkage table)
CREATE TABLE IF NOT EXISTS incident_correlation_groups (
    id TEXT PRIMARY KEY,
    correlation_key TEXT NOT NULL UNIQUE,
    basis TEXT NOT NULL DEFAULT 'shared_signature_key',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    rationale_json TEXT NOT NULL DEFAULT '[]',
    evidence_refs_json TEXT NOT NULL DEFAULT '[]',
    uncertainty_note TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS incident_correlation_members (
    group_id TEXT NOT NULL,
    incident_id TEXT NOT NULL,
    joined_at TEXT NOT NULL,
    PRIMARY KEY (group_id, incident_id),
    FOREIGN KEY (group_id) REFERENCES incident_correlation_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (incident_id) REFERENCES incidents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_corr_members_incident ON incident_correlation_members(incident_id);

INSERT OR IGNORE INTO schema_migrations (version, applied_at) VALUES ('0031_incident_moat_workflow_correlation_outcomes', datetime('now'));
