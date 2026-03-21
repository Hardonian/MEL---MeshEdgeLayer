// Package selfobs provides self-observation capabilities for the MEL system.
// This file implements SQLite-based persistent storage for audit trails and heat maps.
package selfobs

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// PersistenceStore provides SQLite-based persistent storage for audit trails,
// heat maps, health history, and LLM interactions.
type PersistenceStore struct {
	db              *sql.DB
	mu              sync.RWMutex
	retentionPolicy time.Duration
}

// HeatPoint represents a single point in a heat trend time series.
type HeatPoint struct {
	Timestamp time.Time
	HeatScore int
}

// HealthSnapshot represents a point-in-time health state.
type HealthSnapshot struct {
	ID          int64
	Timestamp   time.Time
	Component   string
	HealthState string
	ErrorRate   float64
	SuccessCount int
	ErrorCount  int
	TotalOps    int
}

// ActionStats aggregates healing action statistics.
type ActionStats struct {
	TotalActions int
	SuccessRate  float64
	AvgRetries   float64
}

// LLMStats aggregates LLM interaction statistics.
type LLMStats struct {
	TotalCalls   int
	SuccessRate  float64
	AvgLatencyMs float64
}

// HealthTrend represents health state transitions over time.
type HealthTrend struct {
	Component string
	Points    []HealthSnapshot
}

// dbHealingAction represents a healing action for database storage.
// This is a DB-specific type to avoid conflicts with the domain type.
type dbHealingAction struct {
	ID            int64
	Timestamp     time.Time
	Component     string
	ActionType    string
	Success       bool
	ErrorMessage  string
	RetryCount    int
	StrategyName  string
	TransportType string
}

// dbComponentHeat represents heat map data for database storage.
// This is a DB-specific type to avoid conflicts with the domain type.
type dbComponentHeat struct {
	ID                int64
	Timestamp         time.Time
	Component         string
	FailureCount      int
	RetryCount        int
	StateChangeCount  int
	RecoveryAttempts  int
	RecoverySuccesses int
	HeatScore         int
}

// dbLLMAuditEntry represents an LLM audit entry for database storage.
// This is a DB-specific type to avoid conflicts with the domain type.
type dbLLMAuditEntry struct {
	ID            int64
	Timestamp     time.Time
	OperationType string
	InputHash     string
	ResponseHash  string
	Success       bool
	LatencyMs     int
	ErrorMessage  string
	ProviderType  string
}

// ComponentState represents the state of a component for persistence.
type ComponentState int

const (
	StateHealthy ComponentState = iota
	StateDegraded
	StateUnhealthy
	StateUnknown
)

const schemaSQL = `
-- Audit trails table
CREATE TABLE IF NOT EXISTS healing_audit (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    component TEXT NOT NULL,
    action_type TEXT NOT NULL,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    retry_count INTEGER,
    strategy_name TEXT,
    transport_type TEXT
);

-- Heat map data table
CREATE TABLE IF NOT EXISTS heat_map_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    component TEXT NOT NULL,
    failure_count INTEGER DEFAULT 0,
    retry_count INTEGER DEFAULT 0,
    state_change_count INTEGER DEFAULT 0,
    recovery_attempts INTEGER DEFAULT 0,
    recovery_successes INTEGER DEFAULT 0,
    heat_score INTEGER DEFAULT 0
);

-- Component health history
CREATE TABLE IF NOT EXISTS health_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    component TEXT NOT NULL,
    health_state TEXT NOT NULL,
    error_rate REAL DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    total_ops INTEGER DEFAULT 0
);

-- LLM interactions audit
CREATE TABLE IF NOT EXISTS llm_audit (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    operation_type TEXT NOT NULL,
    input_hash TEXT NOT NULL,
    response_hash TEXT,
    success BOOLEAN NOT NULL,
    latency_ms INTEGER,
    error_message TEXT,
    provider_type TEXT
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_healing_component ON healing_audit(component);
CREATE INDEX IF NOT EXISTS idx_healing_timestamp ON healing_audit(timestamp);
CREATE INDEX IF NOT EXISTS idx_heat_component ON heat_map_data(component);
CREATE INDEX IF NOT EXISTS idx_heat_timestamp ON heat_map_data(timestamp);
CREATE INDEX IF NOT EXISTS idx_health_component ON health_history(component);
CREATE INDEX IF NOT EXISTS idx_health_timestamp ON health_history(timestamp);
CREATE INDEX IF NOT EXISTS idx_llm_timestamp ON llm_audit(timestamp);
CREATE INDEX IF NOT EXISTS idx_llm_input_hash ON llm_audit(input_hash);
`

// NewPersistenceStore creates a new PersistenceStore with the given database path
// and retention policy. It initializes the database schema if it doesn't exist.
func NewPersistenceStore(dbPath string, retention time.Duration) (*PersistenceStore, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("database path cannot be empty")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &PersistenceStore{
		db:              db,
		retentionPolicy: retention,
	}

	if err := store.createSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return store, nil
}

// createSchema initializes the database tables and indexes.
func (s *PersistenceStore) createSchema() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (s *PersistenceStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return nil
	}

	return s.db.Close()
}

// SaveHealingAction persists a healing action to the audit trail.
func (s *PersistenceStore) SaveHealingAction(action HealingAction) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		INSERT INTO healing_audit 
		(timestamp, component, action_type, success, error_message, retry_count, strategy_name, transport_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		action.Timestamp,
		action.Component,
		action.ActionType,
		action.Success,
		action.ErrorMessage,
		action.RetryCount,
		action.StrategyName,
		action.TransportType,
	)
	if err != nil {
		return fmt.Errorf("failed to save healing action: %w", err)
	}

	return nil
}

// GetHealingActions retrieves healing actions for a specific component since the given time.
func (s *PersistenceStore) GetHealingActions(component string, since time.Time) ([]HealingAction, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, timestamp, component, action_type, success, error_message, 
		       retry_count, strategy_name, transport_type
		FROM healing_audit
		WHERE component = ? AND timestamp >= ?
		ORDER BY timestamp DESC
	`

	rows, err := s.db.Query(query, component, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query healing actions: %w", err)
	}
	defer rows.Close()

	return scanHealingActions(rows)
}

// GetAllHealingActions retrieves all healing actions since the given time.
func (s *PersistenceStore) GetAllHealingActions(since time.Time) ([]HealingAction, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, timestamp, component, action_type, success, error_message, 
		       retry_count, strategy_name, transport_type
		FROM healing_audit
		WHERE timestamp >= ?
		ORDER BY timestamp DESC
	`

	rows, err := s.db.Query(query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query healing actions: %w", err)
	}
	defer rows.Close()

	return scanHealingActions(rows)
}

func scanHealingActions(rows *sql.Rows) ([]HealingAction, error) {
	var actions []HealingAction

	for rows.Next() {
		var a HealingAction
		err := rows.Scan(
			&a.ID,
			&a.Timestamp,
			&a.Component,
			&a.ActionType,
			&a.Success,
			&a.ErrorMessage,
			&a.RetryCount,
			&a.StrategyName,
			&a.TransportType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan healing action: %w", err)
		}
		actions = append(actions, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return actions, nil
}

// GetHealingActionStats retrieves aggregated statistics for healing actions
// within the specified time window.
func (s *PersistenceStore) GetHealingActionStats(component string, window time.Duration) (ActionStats, error) {
	if s.db == nil {
		return ActionStats{}, fmt.Errorf("database not initialized")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	since := time.Now().Add(-window)

	query := `
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN success THEN 1 ELSE 0 END) as successes,
			AVG(CASE WHEN retry_count IS NOT NULL THEN retry_count ELSE 0 END) as avg_retries
		FROM healing_audit
		WHERE component = ? AND timestamp >= ?
	`

	var stats ActionStats
	var successes int
	err := s.db.QueryRow(query, component, since).Scan(&stats.TotalActions, &successes, &stats.AvgRetries)
	if err != nil {
		return ActionStats{}, fmt.Errorf("failed to get healing action stats: %w", err)
	}

	if stats.TotalActions > 0 {
		stats.SuccessRate = float64(successes) / float64(stats.TotalActions)
	}

	return stats, nil
}

// SaveHeatData persists heat map data for a component.
func (s *PersistenceStore) SaveHeatData(component string, heat ComponentHeat) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		INSERT INTO heat_map_data 
		(timestamp, component, failure_count, retry_count, state_change_count, 
		 recovery_attempts, recovery_successes, heat_score)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		heat.Timestamp,
		component,
		heat.FailureCount,
		heat.RetryCount,
		heat.StateChangeCount,
		heat.RecoveryAttempts,
		heat.RecoverySuccesses,
		heat.HeatScore,
	)
	if err != nil {
		return fmt.Errorf("failed to save heat data: %w", err)
	}

	return nil
}

// GetHeatData retrieves heat map data for a component since the given time.
func (s *PersistenceStore) GetHeatData(component string, since time.Time) ([]ComponentHeat, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, timestamp, component, failure_count, retry_count, 
		       state_change_count, recovery_attempts, recovery_successes, heat_score
		FROM heat_map_data
		WHERE component = ? AND timestamp >= ?
		ORDER BY timestamp DESC
	`

	rows, err := s.db.Query(query, component, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query heat data: %w", err)
	}
	defer rows.Close()

	return scanComponentHeat(rows)
}

func scanComponentHeat(rows *sql.Rows) ([]ComponentHeat, error) {
	var heats []ComponentHeat

	for rows.Next() {
		var h ComponentHeat
		err := rows.Scan(
			&h.ID,
			&h.Timestamp,
			&h.Component,
			&h.FailureCount,
			&h.RetryCount,
			&h.StateChangeCount,
			&h.RecoveryAttempts,
			&h.RecoverySuccesses,
			&h.HeatScore,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan heat data: %w", err)
		}
		heats = append(heats, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return heats, nil
}

// GetHotspotsFromDB retrieves components with heat scores above the threshold
// within the specified time window.
func (s *PersistenceStore) GetHotspotsFromDB(threshold int, window time.Duration) ([]ComponentHeat, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	since := time.Now().Add(-window)

	query := `
		SELECT id, timestamp, component, failure_count, retry_count, 
		       state_change_count, recovery_attempts, recovery_successes, heat_score
		FROM heat_map_data
		WHERE heat_score >= ? AND timestamp >= ?
		ORDER BY heat_score DESC, timestamp DESC
	`

	rows, err := s.db.Query(query, threshold, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query hotspots: %w", err)
	}
	defer rows.Close()

	return scanComponentHeat(rows)
}

// GetHeatTrend retrieves the heat score trend for a component over time.
func (s *PersistenceStore) GetHeatTrend(component string, window time.Duration) ([]HeatPoint, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	since := time.Now().Add(-window)

	query := `
		SELECT timestamp, heat_score
		FROM heat_map_data
		WHERE component = ? AND timestamp >= ?
		ORDER BY timestamp ASC
	`

	rows, err := s.db.Query(query, component, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query heat trend: %w", err)
	}
	defer rows.Close()

	var points []HeatPoint
	for rows.Next() {
		var p HeatPoint
		err := rows.Scan(&p.Timestamp, &p.HeatScore)
		if err != nil {
			return nil, fmt.Errorf("failed to scan heat point: %w", err)
		}
		points = append(points, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return points, nil
}

// SaveHealthState persists the current health state of a component.
func (s *PersistenceStore) SaveHealthState(comp *InternalComponent) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	if comp == nil {
		return fmt.Errorf("component cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		INSERT INTO health_history 
		(timestamp, component, health_state, error_rate, success_count, error_count, total_ops)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	var errorRate float64
	if comp.TotalOps > 0 {
		errorRate = float64(comp.ErrorCount) / float64(comp.TotalOps)
	}

	_, err := s.db.Exec(query,
		time.Now(),
		comp.Name,
		string(comp.Health),
		errorRate,
		comp.SuccessCount,
		comp.ErrorCount,
		comp.TotalOps,
	)
	if err != nil {
		return fmt.Errorf("failed to save health state: %w", err)
	}

	return nil
}

// GetHealthHistory retrieves the health history for a component since the given time.
func (s *PersistenceStore) GetHealthHistory(component string, since time.Time) ([]HealthSnapshot, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, timestamp, component, health_state, error_rate, success_count, error_count, total_ops
		FROM health_history
		WHERE component = ? AND timestamp >= ?
		ORDER BY timestamp DESC
	`

	rows, err := s.db.Query(query, component, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query health history: %w", err)
	}
	defer rows.Close()

	return scanHealthSnapshots(rows)
}

func scanHealthSnapshots(rows *sql.Rows) ([]HealthSnapshot, error) {
	var snapshots []HealthSnapshot

	for rows.Next() {
		var h HealthSnapshot
		err := rows.Scan(
			&h.ID,
			&h.Timestamp,
			&h.Component,
			&h.HealthState,
			&h.ErrorRate,
			&h.SuccessCount,
			&h.ErrorCount,
			&h.TotalOps,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan health snapshot: %w", err)
		}
		snapshots = append(snapshots, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return snapshots, nil
}

// GetHealthTrend retrieves the health trend for a component over the specified window.
func (s *PersistenceStore) GetHealthTrend(component string, window time.Duration) (HealthTrend, error) {
	if s.db == nil {
		return HealthTrend{}, fmt.Errorf("database not initialized")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	since := time.Now().Add(-window)

	query := `
		SELECT id, timestamp, component, health_state, error_rate, success_count, error_count, total_ops
		FROM health_history
		WHERE component = ? AND timestamp >= ?
		ORDER BY timestamp ASC
	`

	rows, err := s.db.Query(query, component, since)
	if err != nil {
		return HealthTrend{}, fmt.Errorf("failed to query health trend: %w", err)
	}
	defer rows.Close()

	snapshots, err := scanHealthSnapshots(rows)
	if err != nil {
		return HealthTrend{}, err
	}

	return HealthTrend{
		Component: component,
		Points:    snapshots,
	}, nil
}

// SaveLLMAudit persists an LLM interaction audit entry.
func (s *PersistenceStore) SaveLLMAudit(entry LLMAuditEntry) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		INSERT INTO llm_audit 
		(timestamp, operation_type, input_hash, response_hash, success, latency_ms, error_message, provider_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		entry.Timestamp,
		entry.OperationType,
		entry.InputHash,
		entry.ResponseHash,
		entry.Success,
		entry.LatencyMs,
		entry.ErrorMessage,
		entry.ProviderType,
	)
	if err != nil {
		return fmt.Errorf("failed to save LLM audit: %w", err)
	}

	return nil
}

// GetLLMAudit retrieves LLM audit entries since the given time.
func (s *PersistenceStore) GetLLMAudit(since time.Time) ([]LLMAuditEntry, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, timestamp, operation_type, input_hash, response_hash, 
		       success, latency_ms, error_message, provider_type
		FROM llm_audit
		WHERE timestamp >= ?
		ORDER BY timestamp DESC
	`

	rows, err := s.db.Query(query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query LLM audit: %w", err)
	}
	defer rows.Close()

	return scanLLMAuditEntries(rows)
}

func scanLLMAuditEntries(rows *sql.Rows) ([]LLMAuditEntry, error) {
	var entries []LLMAuditEntry

	for rows.Next() {
		var e LLMAuditEntry
		err := rows.Scan(
			&e.ID,
			&e.Timestamp,
			&e.OperationType,
			&e.InputHash,
			&e.ResponseHash,
			&e.Success,
			&e.LatencyMs,
			&e.ErrorMessage,
			&e.ProviderType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan LLM audit entry: %w", err)
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return entries, nil
}

// GetLLMStats retrieves aggregated statistics for LLM interactions within the
// specified time window.
func (s *PersistenceStore) GetLLMStats(window time.Duration) (LLMStats, error) {
	if s.db == nil {
		return LLMStats{}, fmt.Errorf("database not initialized")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	since := time.Now().Add(-window)

	query := `
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN success THEN 1 ELSE 0 END) as successes,
			AVG(CASE WHEN latency_ms IS NOT NULL THEN latency_ms ELSE 0 END) as avg_latency
		FROM llm_audit
		WHERE timestamp >= ?
	`

	var stats LLMStats
	var successes int
	err := s.db.QueryRow(query, since).Scan(&stats.TotalCalls, &successes, &stats.AvgLatencyMs)
	if err != nil {
		return LLMStats{}, fmt.Errorf("failed to get LLM stats: %w", err)
	}

	if stats.TotalCalls > 0 {
		stats.SuccessRate = float64(successes) / float64(stats.TotalCalls)
	}

	return stats, nil
}

// PruneOldData removes all data older than the specified cutoff time.
func (s *PersistenceStore) PruneOldData(before time.Time) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	tables := []string{
		"healing_audit",
		"heat_map_data",
		"health_history",
		"llm_audit",
	}

	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s WHERE timestamp < ?", table)
		_, err := tx.Exec(query, before)
		if err != nil {
			return fmt.Errorf("failed to prune %s: %w", table, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit prune transaction: %w", err)
	}

	return nil
}

// PruneAuditTrail removes audit trail entries older than the specified cutoff time.
func (s *PersistenceStore) PruneAuditTrail(before time.Time) error {
	return s.pruneTable("healing_audit", before)
}

// PruneHeatData removes heat map data older than the specified cutoff time.
func (s *PersistenceStore) PruneHeatData(before time.Time) error {
	return s.pruneTable("heat_map_data", before)
}

// PruneHealthHistory removes health history entries older than the specified cutoff time.
func (s *PersistenceStore) PruneHealthHistory(before time.Time) error {
	return s.pruneTable("health_history", before)
}

// PruneLLMAudit removes LLM audit entries older than the specified cutoff time.
func (s *PersistenceStore) PruneLLMAudit(before time.Time) error {
	return s.pruneTable("llm_audit", before)
}

func (s *PersistenceStore) pruneTable(tableName string, before time.Time) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	query := fmt.Sprintf("DELETE FROM %s WHERE timestamp < ?", tableName)
	_, err := s.db.Exec(query, before)
	if err != nil {
		return fmt.Errorf("failed to prune %s: %w", tableName, err)
	}

	return nil
}

// String returns the string representation of a component state.
// This helper is used for persistence-related state strings.
func componentStateString(cs ComponentState) string {
	switch cs {
	case StateHealthy:
		return "healthy"
	case StateDegraded:
		return "degraded"
	case StateUnhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}
