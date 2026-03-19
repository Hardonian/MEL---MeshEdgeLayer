CREATE INDEX IF NOT EXISTS idx_messages_transport_rx_time ON messages(transport_name, rx_time);
CREATE INDEX IF NOT EXISTS idx_messages_portnum ON messages(portnum);
CREATE INDEX IF NOT EXISTS idx_messages_from_node ON messages(from_node);
CREATE INDEX IF NOT EXISTS idx_nodes_last_seen ON nodes(last_seen);
INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0002_runtime_truth', datetime('now'));
