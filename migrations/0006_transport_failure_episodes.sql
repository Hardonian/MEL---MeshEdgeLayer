ALTER TABLE transport_runtime_evidence ADD COLUMN last_failure_at TEXT NOT NULL DEFAULT '';
ALTER TABLE transport_runtime_evidence ADD COLUMN episode_id TEXT NOT NULL DEFAULT '';
ALTER TABLE transport_runtime_evidence ADD COLUMN failure_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE transport_runtime_evidence ADD COLUMN observation_drops INTEGER NOT NULL DEFAULT 0;
ALTER TABLE transport_runtime_evidence ADD COLUMN last_observation_drop_at TEXT NOT NULL DEFAULT '';
INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0006_transport_failure_episodes', datetime('now'));
