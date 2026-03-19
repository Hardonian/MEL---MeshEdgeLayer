ALTER TABLE transport_alerts ADD COLUMN contributing_reasons_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE transport_alerts ADD COLUMN cluster_reference TEXT NOT NULL DEFAULT '';
ALTER TABLE transport_alerts ADD COLUMN penalty_snapshot_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE transport_alerts ADD COLUMN trigger_condition TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_audit_logs_transport_created_at ON audit_logs(COALESCE(json_extract(details_json,'$.transport'), ''), created_at DESC);
CREATE INDEX IF NOT EXISTS idx_dead_letters_transport_created_at_desc ON dead_letters(transport_name, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transport_alerts_transport_history ON transport_alerts(transport_name, last_updated_at DESC, active);
CREATE INDEX IF NOT EXISTS idx_transport_health_snapshots_time ON transport_health_snapshots(snapshot_time DESC, transport_name);

INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0008_transport_intelligence_operability', datetime('now'));
