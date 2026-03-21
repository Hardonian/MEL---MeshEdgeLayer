-- Self-Observability Tables
-- Phase 10: Internal Health, Freshness, and SLO Tracking

-- Internal component health tracking
CREATE TABLE IF NOT EXISTS internal_health (
    component TEXT PRIMARY KEY,
    health TEXT NOT NULL DEFAULT 'unknown',
    last_success TEXT,
    last_failure TEXT,
    error_count INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    total_ops INTEGER NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_internal_health_updated ON internal_health(updated_at);

-- Freshness markers for internal components
CREATE TABLE IF NOT EXISTS freshness_markers (
    component TEXT PRIMARY KEY,
    last_update TEXT NOT NULL,
    expected_interval_seconds INTEGER NOT NULL DEFAULT 60,
    stale_threshold_seconds INTEGER NOT NULL DEFAULT 300,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_freshness_markers_updated ON freshness_markers(updated_at);

-- SLO status tracking
CREATE TABLE IF NOT EXISTS slo_status (
    slo_name TEXT PRIMARY KEY,
    current_value REAL NOT NULL DEFAULT 0,
    target REAL NOT NULL DEFAULT 99.0,
    status TEXT NOT NULL DEFAULT 'unknown',
    budget_used REAL NOT NULL DEFAULT 0,
    window_start TEXT,
    window_end TEXT,
    evaluated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_slo_status_evaluated ON slo_status(evaluated_at);

-- Correlation ID tracking for end-to-end tracing
CREATE TABLE IF NOT EXISTS correlation_events (
    correlation_id TEXT NOT NULL,
    component TEXT NOT NULL,
    stage TEXT NOT NULL,
    event_time TEXT NOT NULL DEFAULT (datetime('now')),
    details TEXT,
    PRIMARY KEY (correlation_id, component, stage)
);

CREATE INDEX IF NOT EXISTS idx_correlation_events_time ON correlation_events(event_time);
CREATE INDEX IF NOT EXISTS idx_correlation_events_id ON correlation_events(correlation_id);

-- Internal metrics snapshot (for historical analysis)
CREATE TABLE IF NOT EXISTS internal_metrics_snapshot (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    snapshot_time TEXT NOT NULL DEFAULT (datetime('now')),
    pipeline_latency_ingest_to_classify_p99 INTEGER,
    pipeline_latency_classify_to_alert_p99 INTEGER,
    pipeline_latency_alert_to_action_p99 INTEGER,
    worker_heartbeat_ingest TEXT,
    worker_heartbeat_classify TEXT,
    worker_heartbeat_alert TEXT,
    worker_heartbeat_control TEXT,
    queue_depth_ingest INTEGER,
    queue_depth_classify INTEGER,
    queue_depth_alert INTEGER,
    error_rate_ingest REAL,
    error_rate_classify REAL,
    error_rate_alert REAL,
    error_rate_control REAL,
    memory_used_bytes INTEGER,
    goroutines INTEGER
);

CREATE INDEX IF NOT EXISTS idx_metrics_snapshot_time ON internal_metrics_snapshot(snapshot_time);
