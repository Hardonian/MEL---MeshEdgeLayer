-- Migration 0019: Canonical topology model
-- Adds first-class node health, link/edge, topology snapshot, source trust,
-- bookmark/preference, and scoring tables.
INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0019_topology_model', datetime('now'));

-- ============================================================
-- Source trust classification
-- ============================================================
CREATE TABLE IF NOT EXISTS source_trust (
  connector_id TEXT PRIMARY KEY,
  connector_name TEXT NOT NULL,
  connector_type TEXT NOT NULL,  -- local_direct, trusted_broker, partial_relay, imported_external, unknown
  trust_class TEXT NOT NULL DEFAULT 'unknown',  -- direct_local, trusted, partial, untrusted, unknown
  trust_level REAL NOT NULL DEFAULT 0.5,  -- 0.0 to 1.0
  first_seen_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_seen_at TEXT,
  observation_count INTEGER NOT NULL DEFAULT 0,
  contradiction_count INTEGER NOT NULL DEFAULT 0,
  stale_count INTEGER NOT NULL DEFAULT 0,
  operator_notes TEXT,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Extended node health model
-- ============================================================
ALTER TABLE nodes ADD COLUMN first_seen_at TEXT;
ALTER TABLE nodes ADD COLUMN last_direct_seen_at TEXT;
ALTER TABLE nodes ADD COLUMN last_broker_seen_at TEXT;
ALTER TABLE nodes ADD COLUMN trust_class TEXT NOT NULL DEFAULT 'unknown';
ALTER TABLE nodes ADD COLUMN location_state TEXT NOT NULL DEFAULT 'unknown';  -- exact, approximate, stale, unknown, redacted
ALTER TABLE nodes ADD COLUMN mobility_state TEXT NOT NULL DEFAULT 'unknown';  -- static, likely_mobile, unknown
ALTER TABLE nodes ADD COLUMN health_state TEXT NOT NULL DEFAULT 'unknown';    -- healthy, degraded, unstable, stale, isolated, unknown
ALTER TABLE nodes ADD COLUMN health_score REAL NOT NULL DEFAULT 0.0;
ALTER TABLE nodes ADD COLUMN health_factors_json TEXT;  -- JSON: factor breakdown
ALTER TABLE nodes ADD COLUMN source_connector_id TEXT;
ALTER TABLE nodes ADD COLUMN stale INTEGER NOT NULL DEFAULT 0;
ALTER TABLE nodes ADD COLUMN quarantined INTEGER NOT NULL DEFAULT 0;
ALTER TABLE nodes ADD COLUMN quarantine_reason TEXT;

CREATE INDEX IF NOT EXISTS idx_nodes_health_state ON nodes(health_state);
CREATE INDEX IF NOT EXISTS idx_nodes_stale ON nodes(stale);
CREATE INDEX IF NOT EXISTS idx_nodes_last_seen ON nodes(last_seen);

-- ============================================================
-- Canonical link/edge model
-- ============================================================
CREATE TABLE IF NOT EXISTS topology_links (
  edge_id TEXT PRIMARY KEY,
  src_node_num INTEGER NOT NULL,
  dst_node_num INTEGER NOT NULL,
  observed INTEGER NOT NULL DEFAULT 1,    -- 1=observed, 0=inferred
  directional INTEGER NOT NULL DEFAULT 0, -- 1=directional, 0=undirected
  transport_path TEXT,                     -- which transport observed this
  first_observed_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_observed_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  quality_score REAL NOT NULL DEFAULT 0.0,
  reliability REAL NOT NULL DEFAULT 0.0,
  intermittence_count INTEGER NOT NULL DEFAULT 0,
  source_trust_level REAL NOT NULL DEFAULT 0.5,
  source_connector_id TEXT,
  stale INTEGER NOT NULL DEFAULT 0,
  contradiction INTEGER NOT NULL DEFAULT 0,
  contradiction_detail TEXT,
  relay_dependent INTEGER NOT NULL DEFAULT 0,
  quality_factors_json TEXT,  -- JSON: factor breakdown
  observation_count INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(src_node_num, dst_node_num, transport_path)
);

CREATE INDEX IF NOT EXISTS idx_topology_links_src ON topology_links(src_node_num);
CREATE INDEX IF NOT EXISTS idx_topology_links_dst ON topology_links(dst_node_num);
CREATE INDEX IF NOT EXISTS idx_topology_links_stale ON topology_links(stale);
CREATE INDEX IF NOT EXISTS idx_topology_links_last_observed ON topology_links(last_observed_at);

-- ============================================================
-- Topology snapshots
-- ============================================================
CREATE TABLE IF NOT EXISTS topology_snapshots (
  snapshot_id TEXT PRIMARY KEY,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  node_count INTEGER NOT NULL DEFAULT 0,
  edge_count INTEGER NOT NULL DEFAULT 0,
  direct_edge_count INTEGER NOT NULL DEFAULT 0,
  inferred_edge_count INTEGER NOT NULL DEFAULT 0,
  healthy_nodes INTEGER NOT NULL DEFAULT 0,
  degraded_nodes INTEGER NOT NULL DEFAULT 0,
  stale_nodes INTEGER NOT NULL DEFAULT 0,
  isolated_nodes INTEGER NOT NULL DEFAULT 0,
  graph_hash TEXT,
  source_coverage_json TEXT,   -- JSON: which connectors contributed
  confidence_summary_json TEXT, -- JSON: overall confidence breakdown
  explanation_json TEXT,        -- JSON: human-readable summary factors
  region_summary_json TEXT      -- JSON: per-region/cluster summaries
);

CREATE INDEX IF NOT EXISTS idx_topology_snapshots_created ON topology_snapshots(created_at);

-- ============================================================
-- Node observations (raw source-attributed sightings)
-- ============================================================
CREATE TABLE IF NOT EXISTS node_observations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  node_num INTEGER NOT NULL,
  connector_id TEXT NOT NULL,
  source_type TEXT NOT NULL,       -- direct, broker, relay, import
  trust_level REAL NOT NULL DEFAULT 0.5,
  observed_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  snr REAL,
  rssi INTEGER,
  lat REAL,
  lon REAL,
  altitude INTEGER,
  hop_count INTEGER,
  via_mqtt INTEGER NOT NULL DEFAULT 0,
  gateway_id TEXT,
  metadata_json TEXT,
  dedupe_key TEXT,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_node_observations_node ON node_observations(node_num, observed_at);
CREATE INDEX IF NOT EXISTS idx_node_observations_connector ON node_observations(connector_id);
CREATE INDEX IF NOT EXISTS idx_node_observations_dedupe ON node_observations(dedupe_key);

-- ============================================================
-- Operator bookmarks and preferences
-- ============================================================
CREATE TABLE IF NOT EXISTS node_bookmarks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  node_num INTEGER NOT NULL,
  bookmark_type TEXT NOT NULL DEFAULT 'bookmark',  -- bookmark, preferred_anchor, suspect, critical_infra, avoid, suppress
  label TEXT,
  notes TEXT,
  actor_id TEXT NOT NULL DEFAULT 'system',
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  active INTEGER NOT NULL DEFAULT 1,
  UNIQUE(node_num, bookmark_type)
);

CREATE INDEX IF NOT EXISTS idx_node_bookmarks_node ON node_bookmarks(node_num);
CREATE INDEX IF NOT EXISTS idx_node_bookmarks_type ON node_bookmarks(bookmark_type);

-- ============================================================
-- Link preferences
-- ============================================================
CREATE TABLE IF NOT EXISTS link_preferences (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  src_node_num INTEGER NOT NULL,
  dst_node_num INTEGER NOT NULL,
  preference TEXT NOT NULL DEFAULT 'neutral',  -- preferred, avoid, neutral
  reason TEXT,
  actor_id TEXT NOT NULL DEFAULT 'system',
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  active INTEGER NOT NULL DEFAULT 1,
  UNIQUE(src_node_num, dst_node_num)
);

-- ============================================================
-- Topology scoring history (time-series for trends)
-- ============================================================
CREATE TABLE IF NOT EXISTS node_score_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  node_num INTEGER NOT NULL,
  health_score REAL NOT NULL,
  health_state TEXT NOT NULL,
  factors_json TEXT,
  recorded_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_node_score_history_node ON node_score_history(node_num, recorded_at);

-- ============================================================
-- Recovery state tracking
-- ============================================================
CREATE TABLE IF NOT EXISTS recovery_state (
  id INTEGER PRIMARY KEY CHECK (id = 1),  -- singleton row
  last_clean_shutdown_at TEXT,
  last_startup_at TEXT,
  unclean_shutdown INTEGER NOT NULL DEFAULT 0,
  recovered_jobs_json TEXT,
  pending_actions_json TEXT,
  startup_mode TEXT NOT NULL DEFAULT 'normal',  -- normal, recovery, degraded
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO recovery_state(id, startup_mode) VALUES (1, 'normal');
