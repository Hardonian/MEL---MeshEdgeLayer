package selfobs

import (
	"sync"
	"time"
)

// SLODefinition defines an SLO with its target and evaluation window
type SLODefinition struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Target      float64       `json:"target"` // Target percentage (e.g., 99.9 for 99.9%)
	Window      time.Duration `json:"window"` // Evaluation window (e.g., 24h)
	Metric      string        `json:"metric"` // Associated metric name
	Unit        string        `json:"unit"`   // Display unit (e.g., "ms", "%", "count")
}

// SLOStatus represents the current status of an SLO
type SLOStatus struct {
	Name         string    `json:"name"`
	CurrentValue float64   `json:"current_value"`
	Target       float64   `json:"target"`
	Status       string    `json:"status"`      // "healthy", "at_risk", "breached"
	BudgetUsed   float64   `json:"budget_used"` // Percentage of error budget used
	EvaluatedAt  time.Time `json:"evaluated_at"`
	WindowStart  time.Time `json:"window_start"`
	WindowEnd    time.Time `json:"window_end"`
}

// Built-in SLO definitions
var BuiltInSLOs = []SLODefinition{
	{
		Name:        "message_ingest_latency",
		Description: "P99 latency from message receipt to successful ingest",
		Target:      95.0, // 95% of messages ingested within threshold
		Window:      24 * time.Hour,
		Metric:      "ingest_latency_p99",
		Unit:        "ms",
	},
	{
		Name:        "alert_freshness",
		Description: "Percentage of alert cycles completing within expected interval",
		Target:      99.0, // 99% of alert cycles on time
		Window:      24 * time.Hour,
		Metric:      "alert_cycle_success",
		Unit:        "%",
	},
	{
		Name:        "control_success_rate",
		Description: "Percentage of control operations completing successfully",
		Target:      99.5, // 99.5% success rate
		Window:      24 * time.Hour,
		Metric:      "control_operation_success",
		Unit:        "%",
	},
	{
		Name:        "retention_compliance",
		Description: "Percentage of retention runs completing within window",
		Target:      100.0, // Must always complete
		Window:      24 * time.Hour,
		Metric:      "retention_run_success",
		Unit:        "%",
	},
	{
		Name:        "backup_success",
		Description: "Percentage of backup operations completing successfully",
		Target:      99.0, // 99% backup success
		Window:      24 * time.Hour,
		Metric:      "backup_operation_success",
		Unit:        "%",
	},
}

// SLOTacker tracks SLO compliance
type SLOTracker struct {
	mu          sync.RWMutex
	definitions map[string]SLODefinition
	statuses    map[string]*SLOStatus
	// Metric storage (simple in-memory for now)
	metricWindows map[string][]time.Time // component -> success timestamps
}

// NewSLOTracker creates a new SLO tracker with built-in definitions
func NewSLOTracker() *SLOTracker {
	t := &SLOTracker{
		definitions:   make(map[string]SLODefinition),
		statuses:      make(map[string]*SLOStatus),
		metricWindows: make(map[string][]time.Time),
	}
	// Register built-in SLOs
	for _, slo := range BuiltInSLOs {
		t.definitions[slo.Name] = slo
		t.statuses[slo.Name] = &SLOStatus{
			Name:        slo.Name,
			Target:      slo.Target,
			Status:      "unknown",
			EvaluatedAt: time.Now(),
			WindowStart: time.Now().Add(-slo.Window),
			WindowEnd:   time.Now(),
		}
	}
	return t
}

// RecordSuccess records a successful operation for an SLO metric
func (t *SLOTracker) RecordSuccess(metric string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.metricWindows[metric] = append(t.metricWindows[metric], time.Now())
}

// RecordFailure records a failed operation for an SLO metric
func (t *SLOTracker) RecordFailure(metric string) {
	// For failure tracking, we keep track in a separate map or negative entry
	// For simplicity, we'll use a zero time marker
	t.mu.Lock()
	defer t.mu.Unlock()
	// Store a zero time to indicate failure
	t.metricWindows[metric] = append(t.metricWindows[metric], time.Time{})
}

// evaluateSLOInternal calculates the current SLO compliance for a specific SLO
// Caller must hold t.mu lock.
func (t *SLOTracker) evaluateSLOInternal(sloName string) *SLOStatus {
	def, ok := t.definitions[sloName]
	if !ok {
		return &SLOStatus{
			Name:   sloName,
			Status: "unknown",
			Target: 0,
		}
	}

	windowStart := time.Now().Add(-def.Window)
	events := t.metricWindows[def.Metric]

	var successCount, totalCount int64
	for _, eventTime := range events {
		if eventTime.IsZero() {
			// This was a failure
			totalCount++
		} else if eventTime.After(windowStart) {
			totalCount++
			successCount++
		}
	}

	var currentValue float64
	if totalCount == 0 {
		currentValue = 100.0 // No data, assume healthy
	} else {
		currentValue = float64(successCount) / float64(totalCount) * 100
	}

	status := "healthy"
	budgetUsed := 0.0

	if currentValue < def.Target {
		// Calculate error budget used
		errorBudget := 100.0 - def.Target
		currentError := 100.0 - currentValue
		if errorBudget > 0 {
			budgetUsed = (currentError / errorBudget) * 100
		}

		if budgetUsed >= 100 {
			status = "breached"
		} else if budgetUsed >= 50 {
			status = "at_risk"
		} else {
			status = "healthy"
		}
	}

	return &SLOStatus{
		Name:         sloName,
		CurrentValue: currentValue,
		Target:       def.Target,
		Status:       status,
		BudgetUsed:   budgetUsed,
		EvaluatedAt:  time.Now(),
		WindowStart:  windowStart,
		WindowEnd:    time.Now(),
	}
}

// EvaluateSLO calculates the current SLO compliance for a specific SLO
func (t *SLOTracker) EvaluateSLO(sloName string) *SLOStatus {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.evaluateSLOInternal(sloName)
}

// EvaluateAllSLOs calculates compliance for all registered SLOs
func (t *SLOTracker) EvaluateAllSLOs() []*SLOStatus {
	t.mu.Lock()
	defer t.mu.Unlock()

	var results []*SLOStatus
	for name := range t.definitions {
		results = append(results, t.evaluateSLOInternal(name))
	}
	return results
}

// GetSLOStatus returns the current status of a specific SLO
func (t *SLOTracker) GetSLOStatus(sloName string) *SLOStatus {
	return t.EvaluateSLO(sloName)
}

// GetAllSLOStatuses returns all SLO statuses
func (t *SLOTracker) GetAllSLOStatuses() []*SLOStatus {
	return t.EvaluateAllSLOs()
}

// GetSLODefinition returns the definition for a specific SLO
func (t *SLOTracker) GetSLODefinition(sloName string) *SLODefinition {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if def, ok := t.definitions[sloName]; ok {
		return &def
	}
	return nil
}

// GetAllDefinitions returns all SLO definitions
func (t *SLOTracker) GetAllDefinitions() []SLODefinition {
	t.mu.RLock()
	defer t.mu.RUnlock()
	defs := make([]SLODefinition, 0, len(t.definitions))
	for _, def := range t.definitions {
		defs = append(defs, def)
	}
	return defs
}

// PruneOldData removes data outside the evaluation windows
func (t *SLOTracker) PruneOldData() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Find the maximum window across all SLOs
	maxWindow := 24 * time.Hour
	for _, def := range t.definitions {
		if def.Window > maxWindow {
			maxWindow = def.Window
		}
	}

	cutoff := time.Now().Add(-maxWindow)
	for metric, events := range t.metricWindows {
		var valid []time.Time
		for _, eventTime := range events {
			if !eventTime.IsZero() && eventTime.After(cutoff) {
				valid = append(valid, eventTime)
			}
		}
		t.metricWindows[metric] = valid
	}
}

// Global SLO tracker instance
var globalSLOTracker = NewSLOTracker()

// GetGlobalSLOTracker returns the global SLO tracker
func GetGlobalSLOTracker() *SLOTracker {
	return globalSLOTracker
}

// SetGlobalSLOTracker allows replacing the global tracker (useful for testing)
func SetGlobalSLOTracker(tracker *SLOTracker) {
	globalSLOTracker = tracker
}

// Package-level convenience functions

// RecordSLOSuccess records a success for a metric
func RecordSLOSuccess(metric string) {
	globalSLOTracker.RecordSuccess(metric)
}

// RecordSLOFailure records a failure for a metric
func RecordSLOFailure(metric string) {
	globalSLOTracker.RecordFailure(metric)
}

// EvaluateSLO evaluates a specific SLO
func EvaluateSLO(sloName string) *SLOStatus {
	return globalSLOTracker.EvaluateSLO(sloName)
}

// EvaluateAllSLOs evaluates all SLOs
func EvaluateAllSLOs() []*SLOStatus {
	return globalSLOTracker.EvaluateAllSLOs()
}
