-- Migration 0028: canonical offline remote-evidence import batches and per-item provenance closure.
-- This keeps raw payloads, normalized items, and import-driven timeline correlation durable and inspectable.

CREATE TABLE IF NOT EXISTS remote_import_batches (
    id                           TEXT PRIMARY KEY,
    imported_at                  TEXT NOT NULL,
    local_instance_id            TEXT NOT NULL,
    local_site_id                TEXT NOT NULL DEFAULT '',
    local_fleet_id               TEXT NOT NULL DEFAULT '',
    source_type                  TEXT NOT NULL DEFAULT '',
    source_name                  TEXT NOT NULL DEFAULT '',
    source_path                  TEXT NOT NULL DEFAULT '',
    support_bundle_id            TEXT NOT NULL DEFAULT '',
    format_kind                  TEXT NOT NULL DEFAULT '',
    schema_version               TEXT NOT NULL DEFAULT '',
    claimed_origin_instance_id   TEXT NOT NULL DEFAULT '',
    claimed_origin_site_id       TEXT NOT NULL DEFAULT '',
    claimed_fleet_id             TEXT NOT NULL DEFAULT '',
    exported_at                  TEXT NOT NULL DEFAULT '',
    capability_posture_json      TEXT NOT NULL DEFAULT '{}',
    validation_json              TEXT NOT NULL DEFAULT '{}',
    raw_payload_json             TEXT NOT NULL DEFAULT '{}',
    item_count                   INTEGER NOT NULL DEFAULT 0,
    accepted_count               INTEGER NOT NULL DEFAULT 0,
    accepted_with_caveats_count  INTEGER NOT NULL DEFAULT 0,
    rejected_count               INTEGER NOT NULL DEFAULT 0,
    partial_success              INTEGER NOT NULL DEFAULT 0,
    note                         TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_remote_import_batches_time
    ON remote_import_batches(imported_at DESC);
CREATE INDEX IF NOT EXISTS idx_remote_import_batches_local_scope
    ON remote_import_batches(local_instance_id, imported_at DESC);
CREATE INDEX IF NOT EXISTS idx_remote_import_batches_claimed_origin
    ON remote_import_batches(claimed_origin_instance_id, claimed_origin_site_id, imported_at DESC);

ALTER TABLE imported_remote_evidence ADD COLUMN batch_id TEXT NOT NULL DEFAULT '';
ALTER TABLE imported_remote_evidence ADD COLUMN item_id TEXT NOT NULL DEFAULT '';
ALTER TABLE imported_remote_evidence ADD COLUMN sequence_no INTEGER NOT NULL DEFAULT 0;
ALTER TABLE imported_remote_evidence ADD COLUMN local_site_id TEXT NOT NULL DEFAULT '';
ALTER TABLE imported_remote_evidence ADD COLUMN local_fleet_id TEXT NOT NULL DEFAULT '';
ALTER TABLE imported_remote_evidence ADD COLUMN source_type TEXT NOT NULL DEFAULT '';
ALTER TABLE imported_remote_evidence ADD COLUMN source_name TEXT NOT NULL DEFAULT '';
ALTER TABLE imported_remote_evidence ADD COLUMN source_path TEXT NOT NULL DEFAULT '';
ALTER TABLE imported_remote_evidence ADD COLUMN validation_status TEXT NOT NULL DEFAULT '';
ALTER TABLE imported_remote_evidence ADD COLUMN event_json TEXT NOT NULL DEFAULT '';
ALTER TABLE imported_remote_evidence ADD COLUMN normalized_json TEXT NOT NULL DEFAULT '';
ALTER TABLE imported_remote_evidence ADD COLUMN claimed_origin_instance_id TEXT NOT NULL DEFAULT '';
ALTER TABLE imported_remote_evidence ADD COLUMN claimed_origin_site_id TEXT NOT NULL DEFAULT '';
ALTER TABLE imported_remote_evidence ADD COLUMN claimed_fleet_id TEXT NOT NULL DEFAULT '';
ALTER TABLE imported_remote_evidence ADD COLUMN correlation_id TEXT NOT NULL DEFAULT '';
ALTER TABLE imported_remote_evidence ADD COLUMN observed_at TEXT NOT NULL DEFAULT '';
ALTER TABLE imported_remote_evidence ADD COLUMN received_at TEXT NOT NULL DEFAULT '';
ALTER TABLE imported_remote_evidence ADD COLUMN recorded_at TEXT NOT NULL DEFAULT '';
ALTER TABLE imported_remote_evidence ADD COLUMN timing_posture TEXT NOT NULL DEFAULT 'imported_preserved_order';
ALTER TABLE imported_remote_evidence ADD COLUMN merge_disposition TEXT NOT NULL DEFAULT 'raw_only';
ALTER TABLE imported_remote_evidence ADD COLUMN merge_correlation_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_imported_remote_evidence_batch
    ON imported_remote_evidence(batch_id, sequence_no);
CREATE INDEX IF NOT EXISTS idx_imported_remote_evidence_correlation
    ON imported_remote_evidence(correlation_id, imported_at DESC);
CREATE INDEX IF NOT EXISTS idx_imported_remote_evidence_validation
    ON imported_remote_evidence(validation_status, imported_at DESC);
CREATE INDEX IF NOT EXISTS idx_imported_remote_evidence_merge
    ON imported_remote_evidence(merge_correlation_id, imported_at DESC);

CREATE INDEX IF NOT EXISTS idx_timeline_events_import_id
    ON timeline_events(import_id, event_time DESC);
CREATE INDEX IF NOT EXISTS idx_timeline_events_scope_posture
    ON timeline_events(scope_posture, event_time DESC);

INSERT OR IGNORE INTO schema_migrations(version, applied_at)
    VALUES ('0028_remote_import_batches', datetime('now'));
