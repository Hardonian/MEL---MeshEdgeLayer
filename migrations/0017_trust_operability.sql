-- Migration: 0017_trust_operability.sql
-- Description: Control-plane trust, approval gates, evidence bundles,
--   freeze/maintenance mode, operator notes, global control state
-- Created: 2026-03-21

-- ─── 1. Extend control_actions with approval lifecycle fields ────────────────
-- execution_mode: auto | approval_required | manual_only | dry_run
-- proposed_by: actor ID that proposed the action
-- approved_by / rejected_by: actor who approved or rejected
-- approval_expires_at: hard deadline for pending-approval state
-- blast_radius_class: local | transport | mesh | global
-- before_state_json / after_state_json: point-in-time snapshots
-- evidence_bundle_id: FK reference to evidence_bundles
ALTER TABLE control_actions ADD COLUMN execution_mode TEXT NOT NULL DEFAULT 'auto';
ALTER TABLE control_actions ADD COLUMN proposed_by TEXT NOT NULL DEFAULT 'system';
ALTER TABLE control_actions ADD COLUMN approved_by TEXT;
ALTER TABLE control_actions ADD COLUMN approved_at TEXT;
ALTER TABLE control_actions ADD COLUMN rejected_by TEXT;
ALTER TABLE control_actions ADD COLUMN rejected_at TEXT;
ALTER TABLE control_actions ADD COLUMN approval_note TEXT;
ALTER TABLE control_actions ADD COLUMN approval_expires_at TEXT;
ALTER TABLE control_actions ADD COLUMN blast_radius_class TEXT NOT NULL DEFAULT 'unknown';
ALTER TABLE control_actions ADD COLUMN before_state_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE control_actions ADD COLUMN after_state_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE control_actions ADD COLUMN evidence_bundle_id TEXT;

CREATE INDEX IF NOT EXISTS idx_control_actions_exec_mode
    ON control_actions(execution_mode, lifecycle_state);
CREATE INDEX IF NOT EXISTS idx_control_actions_proposed_by
    ON control_actions(proposed_by, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_control_actions_evidence_bundle
    ON control_actions(evidence_bundle_id);

-- ─── 2. Evidence bundles ─────────────────────────────────────────────────────
-- Ties together all evidence that led to an action or decision so operators can
-- replay and verify the reasoning path.
CREATE TABLE IF NOT EXISTS evidence_bundles (
    id                    TEXT PRIMARY KEY,
    action_id             TEXT,           -- linked control action (nullable: bundles may exist before action)
    alert_id              TEXT,           -- linked alert/incident
    decision_id           TEXT,           -- linked control decision
    observations_json     TEXT NOT NULL DEFAULT '[]',  -- relevant observation window summary
    anomalies_json        TEXT NOT NULL DEFAULT '[]',  -- anomaly cluster references
    health_snapshots_json TEXT NOT NULL DEFAULT '[]',  -- health snapshot IDs / inline
    policy_version        TEXT NOT NULL DEFAULT '',    -- config version / policy fingerprint at decision time
    explanation_json      TEXT NOT NULL DEFAULT '{}',  -- intelligence explanation output
    transport_health_json TEXT NOT NULL DEFAULT '{}',  -- relevant transport health at decision time
    prior_decisions_json  TEXT NOT NULL DEFAULT '[]',  -- recent relevant prior actions/decisions
    operator_annotations  TEXT NOT NULL DEFAULT '[]',  -- operator notes attached at capture time
    execution_result_json TEXT NOT NULL DEFAULT '{}',  -- execution outcome (filled after execution)
    integrity_hash        TEXT NOT NULL DEFAULT '',    -- sha256 of deterministic canonical fields
    source_type           TEXT NOT NULL DEFAULT 'system', -- system | operator | replay | test
    captured_at           TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at            TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_evidence_bundles_action
    ON evidence_bundles(action_id);
CREATE INDEX IF NOT EXISTS idx_evidence_bundles_decision
    ON evidence_bundles(decision_id);
CREATE INDEX IF NOT EXISTS idx_evidence_bundles_alert
    ON evidence_bundles(alert_id);
CREATE INDEX IF NOT EXISTS idx_evidence_bundles_captured
    ON evidence_bundles(captured_at DESC);

-- ─── 3. Control freezes ──────────────────────────────────────────────────────
-- Runtime-writable global or scoped freeze records.
-- scope_type: global | transport | action_type
-- scope_value: transport name, action type string, or empty for global
-- Cleared by setting cleared_at and cleared_by.
CREATE TABLE IF NOT EXISTS control_freezes (
    id           TEXT PRIMARY KEY,
    scope_type   TEXT NOT NULL DEFAULT 'global', -- global | transport | action_type
    scope_value  TEXT NOT NULL DEFAULT '',        -- '' for global, else transport name / action type
    reason       TEXT NOT NULL DEFAULT '',
    created_by   TEXT NOT NULL DEFAULT 'system',
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    expires_at   TEXT,                            -- NULL = indefinite
    cleared_by   TEXT,
    cleared_at   TEXT,
    active       INTEGER NOT NULL DEFAULT 1       -- 1=active, 0=cleared
);

CREATE INDEX IF NOT EXISTS idx_control_freezes_active
    ON control_freezes(active, scope_type);
CREATE INDEX IF NOT EXISTS idx_control_freezes_created
    ON control_freezes(created_at DESC);

-- ─── 4. Maintenance windows ──────────────────────────────────────────────────
-- Time-bounded suppression of autonomous control actions.
-- Actions during an active maintenance window are logged but not executed
-- unless explicitly marked emergency_override.
CREATE TABLE IF NOT EXISTS maintenance_windows (
    id               TEXT PRIMARY KEY,
    title            TEXT NOT NULL DEFAULT '',
    reason           TEXT NOT NULL DEFAULT '',
    scope_type       TEXT NOT NULL DEFAULT 'global', -- global | transport
    scope_value      TEXT NOT NULL DEFAULT '',
    starts_at        TEXT NOT NULL,
    ends_at          TEXT NOT NULL,
    created_by       TEXT NOT NULL DEFAULT 'system',
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    cancelled_by     TEXT,
    cancelled_at     TEXT,
    active           INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_maintenance_windows_active
    ON maintenance_windows(active, starts_at, ends_at);

-- ─── 5. Operator notes ───────────────────────────────────────────────────────
-- Freeform annotations operators can attach to any ref (action, incident,
-- node, transport, segment, bundle).
CREATE TABLE IF NOT EXISTS operator_notes (
    id          TEXT PRIMARY KEY,
    ref_type    TEXT NOT NULL,  -- action | incident | node | transport | segment | bundle
    ref_id      TEXT NOT NULL,
    actor_id    TEXT NOT NULL DEFAULT 'system',
    content     TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_operator_notes_ref
    ON operator_notes(ref_type, ref_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_operator_notes_actor
    ON operator_notes(actor_id, created_at DESC);

-- ─── 6. Global control plane state ───────────────────────────────────────────
-- Single-row keyed state for top-level operational posture.
-- automation_enabled: master switch readable by all sub-systems.
-- approval_backlog:   count of pending-approval actions.
-- last_action_at:     last time any action executed (freshness signal).
CREATE TABLE IF NOT EXISTS control_plane_state (
    key            TEXT PRIMARY KEY,
    value_text     TEXT,
    value_int      INTEGER,
    value_json     TEXT,
    updated_by     TEXT NOT NULL DEFAULT 'system',
    updated_at     TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Seed initial state rows so reads never fail on a fresh install
INSERT OR IGNORE INTO control_plane_state(key, value_text, updated_at)
    VALUES ('automation_mode', 'normal', datetime('now'));
INSERT OR IGNORE INTO control_plane_state(key, value_int, updated_at)
    VALUES ('approval_backlog', 0, datetime('now'));
INSERT OR IGNORE INTO control_plane_state(key, value_text, updated_at)
    VALUES ('last_action_at', '', datetime('now'));
INSERT OR IGNORE INTO control_plane_state(key, value_text, updated_at)
    VALUES ('freeze_summary', '', datetime('now'));

-- ─── 7. Record migration ─────────────────────────────────────────────────────
INSERT OR IGNORE INTO schema_migrations(version, applied_at)
    VALUES ('0017_trust_operability', datetime('now'));
