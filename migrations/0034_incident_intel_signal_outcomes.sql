-- Operator adjudication for deterministic assist signal codes (non-canonical; complements recommendation_outcomes).
CREATE TABLE IF NOT EXISTS incident_intel_signal_outcomes (
    id TEXT PRIMARY KEY,
    incident_id TEXT NOT NULL,
    signal_code TEXT NOT NULL,
    outcome TEXT NOT NULL,
    actor_id TEXT NOT NULL DEFAULT 'system',
    note TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    FOREIGN KEY (incident_id) REFERENCES incidents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_intel_sig_out_incident ON incident_intel_signal_outcomes(incident_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_intel_sig_out_code ON incident_intel_signal_outcomes(incident_id, signal_code, created_at DESC);

INSERT OR IGNORE INTO schema_migrations (version, applied_at) VALUES ('0034_incident_intel_signal_outcomes', datetime('now'));
