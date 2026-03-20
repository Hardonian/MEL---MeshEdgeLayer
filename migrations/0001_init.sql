CREATE TABLE IF NOT EXISTS schema_migrations(version TEXT PRIMARY KEY, applied_at TEXT NOT NULL);
INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0001_init', datetime('now'));

CREATE TABLE IF NOT EXISTS nodes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  node_num INTEGER NOT NULL UNIQUE,
  node_id TEXT,
  long_name TEXT,
  short_name TEXT,
  role TEXT,
  hardware_model TEXT,
  last_seen TEXT,
  last_snr REAL,
  last_rssi INTEGER,
  last_gateway_id TEXT,
  precise_lat_ciphertext TEXT,
  precise_lon_ciphertext TEXT,
  lat_redacted REAL,
  lon_redacted REAL,
  altitude INTEGER,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Table exists for future use but is not currently populated
CREATE TABLE IF NOT EXISTS channels (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  channel_id TEXT NOT NULL UNIQUE,
  fingerprint TEXT NOT NULL,
  first_seen TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_seen TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  transport_name TEXT NOT NULL,
  packet_id INTEGER,
  dedupe_hash TEXT NOT NULL UNIQUE,
  channel_id TEXT,
  gateway_id TEXT,
  from_node INTEGER,
  to_node INTEGER,
  portnum INTEGER,
  payload_text TEXT,
  payload_json TEXT,
  raw_hex TEXT NOT NULL,
  rx_time TEXT,
  hop_limit INTEGER,
  relay_node INTEGER,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS telemetry_samples (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  node_num INTEGER NOT NULL,
  sample_type TEXT NOT NULL,
  value_json TEXT NOT NULL,
  observed_at TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Table exists for future use but is not currently populated
CREATE TABLE IF NOT EXISTS topology_edges (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source_node INTEGER NOT NULL,
  target_node INTEGER NOT NULL,
  confidence REAL NOT NULL,
  source_type TEXT NOT NULL,
  observed_at TEXT NOT NULL,
  UNIQUE(source_node, target_node, source_type)
);

-- Table exists for future use but is not currently populated
CREATE TABLE IF NOT EXISTS trust_records (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  node_num INTEGER,
  node_id TEXT,
  state TEXT NOT NULL,
  reason TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS retention_jobs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  job_name TEXT NOT NULL,
  last_run TEXT,
  last_status TEXT,
  details TEXT
);

CREATE TABLE IF NOT EXISTS audit_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  category TEXT NOT NULL,
  level TEXT NOT NULL,
  message TEXT NOT NULL,
  details_json TEXT,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS config_apply_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  actor TEXT NOT NULL,
  summary TEXT NOT NULL,
  applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  config_sha256 TEXT NOT NULL,
  diff_json TEXT
);

CREATE TABLE IF NOT EXISTS transport_runtime_status (
  transport_name TEXT PRIMARY KEY,
  transport_type TEXT NOT NULL,
  source TEXT,
  enabled INTEGER NOT NULL DEFAULT 0,
  runtime_state TEXT NOT NULL,
  detail TEXT NOT NULL,
  last_attempt_at TEXT,
  last_connected_at TEXT,
  last_success_at TEXT,
  last_message_at TEXT,
  last_error TEXT,
  total_messages INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
