-- Migration 0027: offline imported remote evidence records (explicit provenance; not live federation)
INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0027_imported_remote_evidence', datetime('now'));

CREATE TABLE IF NOT EXISTS imported_remote_evidence (
    id                TEXT PRIMARY KEY,
    imported_at       TEXT NOT NULL,
    local_instance_id TEXT NOT NULL,
    validation_json   TEXT NOT NULL,
    bundle_json         TEXT NOT NULL,
    evidence_json       TEXT NOT NULL,
    origin_instance_id  TEXT NOT NULL,
    origin_site_id      TEXT NOT NULL DEFAULT '',
    evidence_class      TEXT NOT NULL DEFAULT '',
    observation_origin_class TEXT NOT NULL DEFAULT '',
    rejected            INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_imported_remote_evidence_time
    ON imported_remote_evidence(imported_at DESC);
CREATE INDEX IF NOT EXISTS idx_imported_remote_evidence_origin
    ON imported_remote_evidence(origin_instance_id, imported_at DESC);
