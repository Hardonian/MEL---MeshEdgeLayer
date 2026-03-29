-- Migration: 0032_incident_fingerprint_runbook_faultdomain.sql
-- Description: Versioned incident fingerprints, recommendation effectiveness aggregates,
--   runbook evolution entries, multi-signal fault domains, proofpack manifest version column.
-- Created: 2026-03-29

-- Canonical structured fingerprint per incident (deterministic components + versioning).
CREATE TABLE IF NOT EXISTS incident_fingerprints (
    incident_id TEXT PRIMARY KEY,
    fingerprint_schema_version TEXT NOT NULL DEFAULT 'mel.incident_fingerprint/v1',
    profile_version TEXT NOT NULL DEFAULT 'weights/v1',
    legacy_signature_key TEXT NOT NULL DEFAULT '',
    canonical_hash TEXT NOT NULL,
    component_json TEXT NOT NULL DEFAULT '{}',
    sparsity_json TEXT NOT NULL DEFAULT '[]',
    computed_at TEXT NOT NULL,
    FOREIGN KEY (incident_id) REFERENCES incidents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_incident_fingerprints_hash ON incident_fingerprints(canonical_hash);
CREATE INDEX IF NOT EXISTS idx_incident_fingerprints_legacy_sig ON incident_fingerprints(legacy_signature_key);

-- Rolling effectiveness of assistive recommendation ids within a signature scope (deterministic aggregation).
CREATE TABLE IF NOT EXISTS incident_rec_effectiveness (
    scope_key TEXT NOT NULL,
    recommendation_id TEXT NOT NULL,
    total_count INTEGER NOT NULL DEFAULT 0,
    accepted_count INTEGER NOT NULL DEFAULT 0,
    rejected_count INTEGER NOT NULL DEFAULT 0,
    ineffective_count INTEGER NOT NULL DEFAULT 0,
    worsened_count INTEGER NOT NULL DEFAULT 0,
    modified_count INTEGER NOT NULL DEFAULT 0,
    not_attempted_count INTEGER NOT NULL DEFAULT 0,
    unknown_count INTEGER NOT NULL DEFAULT 0,
    last_outcome_at TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL,
    PRIMARY KEY (scope_key, recommendation_id)
);

CREATE INDEX IF NOT EXISTS idx_rec_eff_scope ON incident_rec_effectiveness(scope_key);

-- Runbook / playbook evolution assets (traceable to history; not autonomous generation).
CREATE TABLE IF NOT EXISTS incident_runbook_entries (
    id TEXT PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'proposed',
    source_kind TEXT NOT NULL,
    legacy_signature_key TEXT NOT NULL DEFAULT '',
    fingerprint_canonical_hash TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    evidence_ref_json TEXT NOT NULL DEFAULT '[]',
    source_incident_ids_json TEXT NOT NULL DEFAULT '[]',
    promotion_basis TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    reviewed_at TEXT NOT NULL DEFAULT '',
    reviewer_actor_id TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_runbook_entries_sig ON incident_runbook_entries(legacy_signature_key, status);
CREATE INDEX IF NOT EXISTS idx_runbook_entries_hash ON incident_runbook_entries(fingerprint_canonical_hash, status);

-- Multi-signal fault domain groupings (uncertainty-explicit; not root-cause claims).
CREATE TABLE IF NOT EXISTS incident_fault_domains (
    id TEXT PRIMARY KEY,
    domain_key TEXT NOT NULL UNIQUE,
    basis TEXT NOT NULL,
    uncertainty TEXT NOT NULL,
    rationale_json TEXT NOT NULL DEFAULT '[]',
    evidence_bundle_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS incident_fault_domain_members (
    domain_id TEXT NOT NULL,
    member_kind TEXT NOT NULL,
    member_id TEXT NOT NULL,
    joined_reason TEXT NOT NULL DEFAULT '',
    joined_at TEXT NOT NULL,
    PRIMARY KEY (domain_id, member_kind, member_id),
    FOREIGN KEY (domain_id) REFERENCES incident_fault_domains(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_fault_domain_members_incident ON incident_fault_domain_members(member_kind, member_id);

INSERT OR IGNORE INTO schema_migrations (version, applied_at) VALUES ('0032_incident_fingerprint_runbook_faultdomain', datetime('now'));
