-- Bounded signature-family peer scans: index supports capped JOIN counts on large fleets.
-- Created: 2026-04-01

CREATE INDEX IF NOT EXISTS idx_incident_signature_incidents_key_incident
    ON incident_signature_incidents(signature_key, incident_id);

INSERT OR IGNORE INTO schema_migrations(version, applied_at)
    VALUES ('0033_signature_family_peer_index', datetime('now'));
