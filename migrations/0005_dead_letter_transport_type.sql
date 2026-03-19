ALTER TABLE dead_letters ADD COLUMN transport_type TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_dead_letters_transport_type_created_at ON dead_letters(transport_type, created_at);
INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0005_dead_letter_transport_type', datetime('now'));
