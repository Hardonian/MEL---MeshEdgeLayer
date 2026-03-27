-- Migration 0026: optional persisted operator scope keys in instance_metadata (site/fleet boundaries for truthful partial-fleet semantics)
INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0026_fleet_scope_metadata', datetime('now'));

-- No DDL: instance_metadata already exists; keys are written at runtime from config.
