-- Performance indexes identified in persistence audit
-- Created: 2026-03-20

-- messages table: deduplication lookups (used in ingest path)
-- Note: dedupe_hash has UNIQUE constraint which implicitly creates an index,
-- but we make it explicit here for documentation purposes
CREATE INDEX IF NOT EXISTS idx_messages_dedupe_hash ON messages(dedupe_hash);

-- messages table: filtering by recipient node (broadcast vs direct)
-- Partial index excludes broadcast messages (to_node = 0) which are majority
CREATE INDEX IF NOT EXISTS idx_messages_to_node ON messages(to_node) WHERE to_node != 0;

-- messages table: ordering by id is already optimized via INTEGER PRIMARY KEY implicit index
-- No additional index needed for ORDER BY id queries

-- telemetry_samples table: retention queries filter by observed_at
CREATE INDEX IF NOT EXISTS idx_telemetry_samples_observed_at ON telemetry_samples(observed_at);

-- telemetry_samples table: join performance with nodes table
CREATE INDEX IF NOT EXISTS idx_telemetry_samples_node_num ON telemetry_samples(node_num);

-- audit_logs table: retention queries filter by created_at
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);

-- audit_logs table: filtering by category (common query pattern)
CREATE INDEX IF NOT EXISTS idx_audit_logs_category ON audit_logs(category);

-- audit_logs table: composite for category + time range queries (common in dashboards)
CREATE INDEX IF NOT EXISTS idx_audit_logs_category_created_at ON audit_logs(category, created_at DESC);

-- control_actions table: composite index for active actions dashboard query
-- Supports: lifecycle_state filtering with result and reversibility checks
CREATE INDEX IF NOT EXISTS idx_control_actions_lifecycle_result_reversible ON control_actions(lifecycle_state, result, reversible, created_at DESC);

-- control_actions table: index for reversible actions with expiration
-- Partial index for rows that could be active reversible actions
CREATE INDEX IF NOT EXISTS idx_control_actions_reversible_active ON control_actions(result, reversible, expires_at) WHERE reversible = 1;

-- Record migration
INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0012_performance_indexes', datetime('now'));
