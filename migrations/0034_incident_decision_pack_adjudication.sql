-- Migration: 0034_incident_decision_pack_adjudication.sql
-- Operator adjudication on the canonical Incident Decision Pack (local institutional memory).

CREATE TABLE IF NOT EXISTS incident_decision_pack_adjudication (
    incident_id TEXT PRIMARY KEY,
    reviewed INTEGER NOT NULL DEFAULT 0,
    reviewed_at TEXT NOT NULL DEFAULT '',
    reviewed_by_actor_id TEXT NOT NULL DEFAULT '',
    useful TEXT NOT NULL DEFAULT '',
    operator_note TEXT NOT NULL DEFAULT '',
    cue_outcomes_json TEXT NOT NULL DEFAULT '[]',
    updated_at TEXT NOT NULL,
    FOREIGN KEY (incident_id) REFERENCES incidents(id) ON DELETE CASCADE
);

INSERT OR IGNORE INTO schema_migrations (version, applied_at) VALUES ('0034_incident_decision_pack_adjudication', datetime('now'));
