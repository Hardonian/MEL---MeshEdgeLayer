package selfobs

import (
	"sync"
	"time"
)

// HeatMap tracks aggregated system health patterns across all components.
// All data stored is anonymized - only counters, rates, and infrastructure
// component identifiers are stored. No PII, no message content, no raw payloads.
type HeatMap struct {
	// ComponentName identifies the infrastructure component (e.g., "ingest", "classify").
	// This is a system identifier, not user-identifiable information.
	ComponentName string

	// FailureDensity tracks how many failures occurred per component.
	// Stored as aggregated count only - no error messages or stack traces.
	FailureDensity map[string]int64

	// RetryBursts tracks retry patterns per component.
	// Shows which components are experiencing transient issues.
	RetryBursts map[string]int64

	// UnstableComponents tracks components with frequent state changes.
	// Count only - no timestamps of specific events or session data.
	UnstableComponents map[string]int64

	// RecoveryRates tracks recovery success per component.
	// Success rate is stored as percentage (0-100).
	RecoveryRates map[string]float64

	// WindowStart marks the beginning of the aggregation window.
	WindowStart time.Time

	// WindowDuration is the size of the rolling aggregation window.
	WindowDuration time.Duration
}

// ComponentHeat represents the aggregated heat data for a single component.
// All fields are statistical aggregates - no individual events are stored.
type ComponentHeat struct {
	// Component is the infrastructure component identifier (e.g., "ingest", "alert").
	// Not PII - this identifies system infrastructure only.
	Component string

	// FailureCount is the number of failures in the current window.
	// Counter only - no error details, messages, or payloads stored.
	FailureCount int64

	// RetryCount is the number of retries in the current window.
	// Counter only - no request content or response data stored.
	RetryCount int64

	// StateChangeCount is the number of health state transitions.
	// Tracks instability - count only, no state history or timing details.
	StateChangeCount int64

	// LastStateChange is the most recent state change timestamp.
	// Used for detecting rapid fluctuations (instability detection).
	LastStateChange time.Time

	// RecoveryAttempts is the total number of recovery attempts.
	RecoveryAttempts int64

	// RecoverySuccesses is the number of successful recoveries.
	RecoverySuccesses int64

	// HeatScore is a computed value (0-100) representing overall component stress.
	// Higher values indicate more stressed components.
	// Calculated deterministically from aggregated metrics only.
	HeatScore int
}

// HeatAggregator is the main type for collecting and aggregating component heat data.
// It maintains thread-safe access to all component heat information.
//
// PRIVACY NOTE: This aggregator stores ONLY the following:
//   - Component names (infrastructure identifiers like "ingest", "classify")
//   - Aggregated counters (failure counts, retry counts, state changes)
//   - Timestamps of last events (for rate calculations)
//   - Recovery success/failure counts (as tallies)
//
// This aggregator does NOT store:
//   - Any raw message payloads
//   - Any user-provided data or content
//   - Any personally identifiable information (PII)
//   - Stack traces, error messages, or failure details
//   - IP addresses, user IDs, session IDs, or request metadata
//   - Any data that could identify individual users or their activities
type HeatAggregator struct {
	mu          sync.RWMutex
	components  map[string]*ComponentHeat
	window      time.Duration
	createdAt   time.Time
	eventLog    []heatEvent // ring buffer for heat score calculation
	eventIdx    int
	eventCap    int
}

// heatEvent is an internal structure for rolling window calculations.
// Contains only timestamps and counters - no payload data.
type heatEvent struct {
	timestamp   time.Time
	component   string
	eventType   string // "failure", "retry", "state_change", "recovery_success", "recovery_failure"
}

// HeatMapSnapshot provides a point-in-time view of the entire heat map.
// Used for trend analysis and external reporting.
type HeatMapSnapshot struct {
	// Timestamp when the snapshot was taken.
	Timestamp time.Time

	// Window is the aggregation window duration used.
	Window time.Duration

	// Components contains heat data for all tracked components.
	Components []ComponentHeat

	// OverallSystemHeat is the average heat score across all components (0-100).
	OverallSystemHeat int

	// HottestComponent identifies the component with the highest heat score.
	// Empty string if no components are tracked.
	HottestComponent string
}

// NewHeatAggregator creates a new heat aggregator with the specified window duration.
// The window determines how long events are considered for heat score calculations.
//
// Example:
//
//	aggregator := NewHeatAggregator(5 * time.Minute)
func NewHeatAggregator(window time.Duration) *HeatAggregator {
	if window <= 0 {
		window = 5 * time.Minute // default window
	}

	return &HeatAggregator{
		components: make(map[string]*ComponentHeat),
		window:     window,
		createdAt:  time.Now(),
		eventLog:   make([]heatEvent, 0, 10000),
		eventCap:   10000,
	}
}

// getOrCreateComponent returns the ComponentHeat for the given component,
// creating it if it doesn't exist. Must be called with lock held.
func (ha *HeatAggregator) getOrCreateComponent(component string) *ComponentHeat {
	ch, ok := ha.components[component]
	if !ok {
		ch = &ComponentHeat{
			Component:       component,
			LastStateChange: time.Time{}, // zero time indicates no state change yet
		}
		ha.components[component] = ch
	}
	return ch
}

// RecordFailure records a failure event for the specified component.
// Increments the failure counter and recalculates the heat score.
//
// PRIVACY: Only the component name and timestamp are stored.
// No error details, messages, or payloads are recorded.
func (ha *HeatAggregator) RecordFailure(component string) {
	ha.mu.Lock()
	defer ha.mu.Unlock()

	ch := ha.getOrCreateComponent(component)
	ch.FailureCount++

	ha.recordEvent(component, "failure")
	ch.HeatScore = ha.calculateHeatScore(ch)
}

// RecordRetry records a retry event for the specified component.
// Tracks retry bursts which may indicate transient issues or instability.
//
// PRIVACY: Only the component name and timestamp are stored.
// No request/response data or retry context is recorded.
func (ha *HeatAggregator) RecordRetry(component string) {
	ha.mu.Lock()
	defer ha.mu.Unlock()

	ch := ha.getOrCreateComponent(component)
	ch.RetryCount++

	ha.recordEvent(component, "retry")
	ch.HeatScore = ha.calculateHeatScore(ch)
}

// RecordStateChange records a health state transition for a component.
// Tracks instability through frequent state changes.
//
// PRIVACY: Only the component name, state transition direction (implied by from/to),
// and timestamp are stored. No session data or context is recorded.
func (ha *HeatAggregator) RecordStateChange(component string, from, to ComponentHealth) {
	ha.mu.Lock()
	defer ha.mu.Unlock()

	ch := ha.getOrCreateComponent(component)
	ch.StateChangeCount++
	ch.LastStateChange = time.Now()

	ha.recordEvent(component, "state_change")
	ch.HeatScore = ha.calculateHeatScore(ch)
}

// RecordRecovery records the outcome of a recovery attempt.
// Tracks recovery success rates to identify components that struggle to recover.
//
// PRIVACY: Only the component name, success boolean, and timestamp are stored.
// No recovery details, logs, or operational data is recorded.
func (ha *HeatAggregator) RecordRecovery(component string, success bool) {
	ha.mu.Lock()
	defer ha.mu.Unlock()

	ch := ha.getOrCreateComponent(component)
	ch.RecoveryAttempts++
	if success {
		ch.RecoverySuccesses++
	}

	if success {
		ha.recordEvent(component, "recovery_success")
	} else {
		ha.recordEvent(component, "recovery_failure")
	}
	ch.HeatScore = ha.calculateHeatScore(ch)
}

// recordEvent adds an event to the internal event log for rolling window calculations.
// Must be called with lock held.
func (ha *HeatAggregator) recordEvent(component, eventType string) {
	event := heatEvent{
		timestamp: time.Now(),
		component: component,
		eventType: eventType,
	}

	if len(ha.eventLog) < ha.eventCap {
		ha.eventLog = append(ha.eventLog, event)
	} else {
		ha.eventLog[ha.eventIdx] = event
		ha.eventIdx = (ha.eventIdx + 1) % ha.eventCap
	}
}

// calculateHeatScore computes a deterministic heat score (0-100) for a component.
//
// Scoring breakdown:
//   - Failure rate: 0-40 points (based on failures in window vs baseline)
//   - Retry bursts: 0-30 points (based on retry frequency)
//   - Instability: 0-20 points (based on state change frequency)
//   - Recovery health: 0-10 points (inverted - low recovery success = high heat)
//
// The calculation is deterministic: given the same input metrics,
// it will always produce the same heat score.
func (ha *HeatAggregator) calculateHeatScore(ch *ComponentHeat) int {
	now := time.Now()
	windowStart := now.Add(-ha.window)

	// Count events in the current window for this component
	var windowFailures, windowRetries, windowStateChanges int64
	var windowRecoveryAttempts, windowRecoverySuccesses int64

	for _, evt := range ha.eventLog {
		if evt.timestamp.Before(windowStart) {
			continue
		}
		if evt.component != ch.Component {
			continue
		}

		switch evt.eventType {
		case "failure":
			windowFailures++
		case "retry":
			windowRetries++
		case "state_change":
			windowStateChanges++
		case "recovery_success":
			windowRecoveryAttempts++
			windowRecoverySuccesses++
		case "recovery_failure":
			windowRecoveryAttempts++
		}
	}

	// Base: Failure rate (0-40 points)
	// Scale: 0 failures = 0 points, 10+ failures in window = 40 points
	failureScore := int(minInt64(windowFailures, 10) * 4)

	// Retry bursts: Retry rate (0-30 points)
	// Scale: 0 retries = 0 points, 10+ retries in window = 30 points
	retryScore := int(minInt64(windowRetries, 10) * 3)

	// Instability: State change frequency (0-20 points)
	// Scale: 0 changes = 0 points, 5+ changes in window = 20 points
	instabilityScore := int(minInt64(windowStateChanges, 5) * 4)

	// Recovery health: Recovery success rate (0-10 points, inverted)
	// Low recovery success = high heat. Scale: 100% success = 0 points, 0% success = 10 points
	recoveryScore := 0
	if windowRecoveryAttempts > 0 {
		successRate := float64(windowRecoverySuccesses) / float64(windowRecoveryAttempts)
		recoveryScore = int((1.0 - successRate) * 10)
	}

	totalScore := failureScore + retryScore + instabilityScore + recoveryScore
	if totalScore > 100 {
		totalScore = 100
	}

	return totalScore
}

// GetComponentHeat returns the current heat data for a specific component.
// Returns a zero-value ComponentHeat if the component is not tracked.
func (ha *HeatAggregator) GetComponentHeat(component string) ComponentHeat {
	ha.mu.RLock()
	defer ha.mu.RUnlock()

	if ch, ok := ha.components[component]; ok {
		// Return a copy to prevent external mutation
		return *ch
	}

	return ComponentHeat{
		Component: component,
		HeatScore: 0,
	}
}

// GetHotspots returns all components with heat scores at or above the threshold.
// Results are sorted by heat score in descending order (hottest first).
// Threshold should be 0-100; components with score >= threshold are returned.
func (ha *HeatAggregator) GetHotspots(threshold int) []ComponentHeat {
	ha.mu.RLock()
	defer ha.mu.RUnlock()

	var hotspots []ComponentHeat
	for _, ch := range ha.components {
		if ch.HeatScore >= threshold {
			// Create a copy to prevent external mutation
			hotspots = append(hotspots, *ch)
		}
	}

	// Sort by heat score descending
	for i := 0; i < len(hotspots)-1; i++ {
		for j := i + 1; j < len(hotspots); j++ {
			if hotspots[j].HeatScore > hotspots[i].HeatScore {
				hotspots[i], hotspots[j] = hotspots[j], hotspots[i]
			}
		}
	}

	return hotspots
}

// GetHeatMapSnapshot creates a point-in-time snapshot of the entire heat map.
// Useful for trend analysis, reporting, and persistence.
func (ha *HeatAggregator) GetHeatMapSnapshot() HeatMapSnapshot {
	ha.mu.RLock()
	defer ha.mu.RUnlock()

	snapshot := HeatMapSnapshot{
		Timestamp:  time.Now(),
		Window:     ha.window,
		Components: make([]ComponentHeat, 0, len(ha.components)),
	}

	var totalHeat int
	hottestScore := -1
	hottestComponent := ""

	for _, ch := range ha.components {
		// Create a copy to prevent external mutation
		snapshot.Components = append(snapshot.Components, *ch)
		totalHeat += ch.HeatScore

		if ch.HeatScore > hottestScore {
			hottestScore = ch.HeatScore
			hottestComponent = ch.Component
		}
	}

	if len(snapshot.Components) > 0 {
		snapshot.OverallSystemHeat = totalHeat / len(snapshot.Components)
	} else {
		snapshot.OverallSystemHeat = 0
	}

	snapshot.HottestComponent = hottestComponent

	return snapshot
}

// PruneOldData removes all data older than the specified timestamp.
// This includes cleaning up the event log and resetting counters for old events.
// Use this to prevent unbounded memory growth in long-running systems.
func (ha *HeatAggregator) PruneOldData(before time.Time) {
	ha.mu.Lock()
	defer ha.mu.Unlock()

	// Filter event log to remove old events
	newEventLog := make([]heatEvent, 0, len(ha.eventLog))
	for _, evt := range ha.eventLog {
		if !evt.timestamp.Before(before) {
			newEventLog = append(newEventLog, evt)
		}
	}
	ha.eventLog = newEventLog
	ha.eventIdx = 0

	// Reset component counters that are based on the event log
	// The next heat score calculation will naturally reflect the pruned data
	for _, ch := range ha.components {
		// Recalculate heat score after pruning
		ch.HeatScore = ha.calculateHeatScore(ch)
	}
}

// minInt64 returns the minimum of two int64 values.
func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
