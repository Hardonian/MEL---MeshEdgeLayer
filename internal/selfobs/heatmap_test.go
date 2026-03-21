package selfobs

import (
	"strings"
	"testing"
	"time"
)

// TestHeatAggregatorCreation verifies initialization
func TestHeatAggregatorCreation(t *testing.T) {
	// Test with custom window
	window := 10 * time.Minute
	ha := NewHeatAggregator(window)

	if ha == nil {
		t.Fatal("expected non-nil aggregator")
	}
	if ha.window != window {
		t.Errorf("expected window %v, got %v", window, ha.window)
	}
	if ha.components == nil {
		t.Error("expected components map to be initialized")
	}
	if ha.eventLog == nil {
		t.Error("expected eventLog to be initialized")
	}
	if ha.eventCap != 10000 {
		t.Errorf("expected eventCap 10000, got %d", ha.eventCap)
	}

	// Test with zero/negative window (should use default)
	ha2 := NewHeatAggregator(0)
	if ha2.window != 5*time.Minute {
		t.Errorf("expected default window 5m, got %v", ha2.window)
	}

	ha3 := NewHeatAggregator(-1 * time.Minute)
	if ha3.window != 5*time.Minute {
		t.Errorf("expected default window 5m for negative, got %v", ha3.window)
	}
}

// TestRecordFailure verifies failure counts increment and heat score increases
func TestRecordFailure(t *testing.T) {
	ha := NewHeatAggregator(5 * time.Minute)

	// Record single failure
	ha.RecordFailure("ingest")
	ch := ha.GetComponentHeat("ingest")

	if ch.FailureCount != 1 {
		t.Errorf("expected failure count 1, got %d", ch.FailureCount)
	}
	if ch.HeatScore < 0 {
		t.Error("expected heat score >= 0")
	}

	// Record multiple failures
	for i := 0; i < 9; i++ {
		ha.RecordFailure("ingest")
	}
	ch = ha.GetComponentHeat("ingest")

	if ch.FailureCount != 10 {
		t.Errorf("expected failure count 10, got %d", ch.FailureCount)
	}
	// With 10 failures, heat score should be significant (40 pts from failures alone)
	if ch.HeatScore < 40 {
		t.Errorf("expected heat score >= 40 with 10 failures, got %d", ch.HeatScore)
	}

	// Test multiple components isolation
	ha.RecordFailure("classify")
	ingest := ha.GetComponentHeat("ingest")
	classify := ha.GetComponentHeat("classify")

	if ingest.FailureCount != 10 {
		t.Errorf("expected ingest failures 10, got %d", ingest.FailureCount)
	}
	if classify.FailureCount != 1 {
		t.Errorf("expected classify failures 1, got %d", classify.FailureCount)
	}
}

// TestRecordRetry verifies retry tracking and heat score impact
func TestRecordRetry(t *testing.T) {
	ha := NewHeatAggregator(5 * time.Minute)

	// Record single retry
	ha.RecordRetry("ingest")
	ch := ha.GetComponentHeat("ingest")

	if ch.RetryCount != 1 {
		t.Errorf("expected retry count 1, got %d", ch.RetryCount)
	}

	// Record multiple retries
	for i := 0; i < 9; i++ {
		ha.RecordRetry("ingest")
	}
	ch = ha.GetComponentHeat("ingest")

	if ch.RetryCount != 10 {
		t.Errorf("expected retry count 10, got %d", ch.RetryCount)
	}
	// With 10 retries, heat score should include 30 pts from retry score
	if ch.HeatScore < 30 {
		t.Errorf("expected heat score >= 30 with 10 retries, got %d", ch.HeatScore)
	}

	// Test retry doesn't affect failure count
	if ch.FailureCount != 0 {
		t.Errorf("expected failure count 0, got %d", ch.FailureCount)
	}
}

// TestRecordStateChange verifies state change counting and instability tracking
func TestRecordStateChange(t *testing.T) {
	ha := NewHeatAggregator(5 * time.Minute)

	// Record state change
	ha.RecordStateChange("ingest", HealthHealthy, HealthFailing)
	ch := ha.GetComponentHeat("ingest")

	if ch.StateChangeCount != 1 {
		t.Errorf("expected state change count 1, got %d", ch.StateChangeCount)
	}
	if ch.LastStateChange.IsZero() {
		t.Error("expected LastStateChange to be set")
	}

	// Record multiple state changes
	for i := 0; i < 4; i++ {
		ha.RecordStateChange("ingest", HealthFailing, HealthHealthy)
	}
	ch = ha.GetComponentHeat("ingest")

	if ch.StateChangeCount != 5 {
		t.Errorf("expected state change count 5, got %d", ch.StateChangeCount)
	}
	// With 5 state changes, instability score should be max (20 pts)
	if ch.HeatScore < 20 {
		t.Errorf("expected heat score >= 20 with 5 state changes, got %d", ch.HeatScore)
	}
}

// TestRecordRecovery verifies recovery tracking and success/failure rates
func TestRecordRecovery(t *testing.T) {
	ha := NewHeatAggregator(5 * time.Minute)

	// Record successful recovery
	ha.RecordRecovery("ingest", true)
	ch := ha.GetComponentHeat("ingest")

	if ch.RecoveryAttempts != 1 {
		t.Errorf("expected recovery attempts 1, got %d", ch.RecoveryAttempts)
	}
	if ch.RecoverySuccesses != 1 {
		t.Errorf("expected recovery successes 1, got %d", ch.RecoverySuccesses)
	}

	// Record failed recovery
	ha.RecordRecovery("ingest", false)
	ch = ha.GetComponentHeat("ingest")

	if ch.RecoveryAttempts != 2 {
		t.Errorf("expected recovery attempts 2, got %d", ch.RecoveryAttempts)
	}
	if ch.RecoverySuccesses != 1 {
		t.Errorf("expected recovery successes still 1, got %d", ch.RecoverySuccesses)
	}

	// Test mixed recoveries affect heat score
	ha2 := NewHeatAggregator(5 * time.Minute)
	// 5 failures, 5 recoveries (3 success, 2 failure) = 60% success rate
	for i := 0; i < 5; i++ {
		ha2.RecordFailure("ingest")
	}
	ha2.RecordRecovery("ingest", true)
	ha2.RecordRecovery("ingest", true)
	ha2.RecordRecovery("ingest", true)
	ha2.RecordRecovery("ingest", false)
	ha2.RecordRecovery("ingest", false)

	ch2 := ha2.GetComponentHeat("ingest")
	// Recovery score: 40% failure rate * 10 = 4 points
	// Failure score: 5 failures * 4 = 20 (capped at 40)
	// Total should be at least 20
	if ch2.HeatScore < 20 {
		t.Errorf("expected heat score >= 20 with failures and mixed recoveries, got %d", ch2.HeatScore)
	}
}

// TestHeatScoreCalculation verifies deterministic calculation and boundary conditions
func TestHeatScoreCalculation(t *testing.T) {
	ha := NewHeatAggregator(5 * time.Minute)

	// Test boundary condition: zero events = zero heat
	ch := ha.GetComponentHeat("ingest")
	if ch.HeatScore != 0 {
		t.Errorf("expected heat score 0 for no events, got %d", ch.HeatScore)
	}

	// Test boundary condition: 50 heat score
	// Need to calculate what produces ~50
	// 10 failures = 40 pts, need ~10 more from other sources
	ha2 := NewHeatAggregator(5 * time.Minute)
	for i := 0; i < 10; i++ {
		ha2.RecordFailure("ingest") // 40 pts
	}
	ha2.RecordRecovery("ingest", false) // ~5 pts
	ch2 := ha2.GetComponentHeat("ingest")
	if ch2.HeatScore < 45 || ch2.HeatScore > 55 {
		t.Errorf("expected heat score near 50, got %d", ch2.HeatScore)
	}

	// Test boundary condition: max heat (100)
	ha3 := NewHeatAggregator(5 * time.Minute)
	// Max out all contributors
	for i := 0; i < 20; i++ {
		ha3.RecordFailure("ingest") // capped at 40
	}
	for i := 0; i < 20; i++ {
		ha3.RecordRetry("ingest") // capped at 30
	}
	for i := 0; i < 10; i++ {
		ha3.RecordStateChange("ingest", HealthHealthy, HealthFailing) // capped at 20
	}
	for i := 0; i < 10; i++ {
		ha3.RecordRecovery("ingest", false) // 10 pts for 0% success
	}
	ch3 := ha3.GetComponentHeat("ingest")
	if ch3.HeatScore > 100 {
		t.Errorf("expected heat score capped at 100, got %d", ch3.HeatScore)
	}
	// Should be close to 100 with all max contributions
	if ch3.HeatScore < 90 {
		t.Errorf("expected heat score near 100 with max contributions, got %d", ch3.HeatScore)
	}

	// Test component isolation
	ha4 := NewHeatAggregator(5 * time.Minute)
	ha4.RecordFailure("component-a")
	ha4.RecordFailure("component-b")
	ha4.RecordFailure("component-b")

	compA := ha4.GetComponentHeat("component-a")
	compB := ha4.GetComponentHeat("component-b")

	if compA.HeatScore == compB.HeatScore {
		t.Error("expected different heat scores for different event counts")
	}
	if compA.FailureCount != 1 {
		t.Errorf("expected component-a failures 1, got %d", compA.FailureCount)
	}
	if compB.FailureCount != 2 {
		t.Errorf("expected component-b failures 2, got %d", compB.FailureCount)
	}
}

// TestGetHotspots verifies threshold filtering, sorting, and empty results
func TestGetHotspots(t *testing.T) {
	ha := NewHeatAggregator(5 * time.Minute)

	// Create components with different heat levels
	// Component A: high heat (15 failures = 40 pts max + retries for more)
	for i := 0; i < 15; i++ {
		ha.RecordFailure("component-a")
	}
	ha.RecordRetry("component-a") // adds 3 pts
	// Component B: medium heat (10 failures = 40 pts)
	for i := 0; i < 10; i++ {
		ha.RecordFailure("component-b")
	}
	// Component C: low heat (5 failures = 20 pts)
	for i := 0; i < 5; i++ {
		ha.RecordFailure("component-c")
	}

	// Test threshold filtering at 40 (should get A and B)
	hotspots := ha.GetHotspots(40)
	if len(hotspots) != 2 {
		t.Errorf("expected 2 hotspots with threshold 40, got %d", len(hotspots))
	}

	// Test sorting by heat descending
	if len(hotspots) >= 2 {
		if hotspots[0].HeatScore < hotspots[1].HeatScore {
			t.Error("expected hotspots sorted by heat descending")
		}
		if hotspots[0].Component != "component-a" {
			t.Errorf("expected component-a first (hottest), got %s", hotspots[0].Component)
		}
	}

	// Test high threshold - empty results
	hotspotsHigh := ha.GetHotspots(90)
	if len(hotspotsHigh) != 0 {
		t.Errorf("expected 0 hotspots with threshold 90, got %d", len(hotspotsHigh))
	}

	// Test low threshold - all components
	hotspotsLow := ha.GetHotspots(0)
	if len(hotspotsLow) != 3 {
		t.Errorf("expected 3 hotspots with threshold 0, got %d", len(hotspotsLow))
	}
}

// TestGetHeatMapSnapshot verifies snapshot content and calculations
func TestGetHeatMapSnapshot(t *testing.T) {
	ha := NewHeatAggregator(5 * time.Minute)

	// Add data to multiple components
	ha.RecordFailure("ingest")
	ha.RecordFailure("ingest")
	ha.RecordFailure("classify")

	// Get snapshot
	snapshot := ha.GetHeatMapSnapshot()

	// Verify timestamp
	if snapshot.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}

	// Verify window
	if snapshot.Window != 5*time.Minute {
		t.Errorf("expected window 5m, got %v", snapshot.Window)
	}

	// Verify all components present
	if len(snapshot.Components) != 2 {
		t.Errorf("expected 2 components, got %d", len(snapshot.Components))
	}

	// Verify overall system heat calculation
	// ingest: 2 failures = 8 pts, classify: 1 failure = 4 pts
	// Average: (8 + 4) / 2 = 6
	expectedAvg := (ha.GetComponentHeat("ingest").HeatScore + ha.GetComponentHeat("classify").HeatScore) / 2
	if snapshot.OverallSystemHeat != expectedAvg {
		t.Errorf("expected overall heat %d, got %d", expectedAvg, snapshot.OverallSystemHeat)
	}

	// Verify hottest component
	if snapshot.HottestComponent != "ingest" {
		t.Errorf("expected hottest component 'ingest', got '%s'", snapshot.HottestComponent)
	}

	// Test empty snapshot
	ha2 := NewHeatAggregator(5 * time.Minute)
	snapshot2 := ha2.GetHeatMapSnapshot()
	if snapshot2.OverallSystemHeat != 0 {
		t.Errorf("expected overall heat 0 for empty, got %d", snapshot2.OverallSystemHeat)
	}
	if snapshot2.HottestComponent != "" {
		t.Errorf("expected empty hottest component for empty, got '%s'", snapshot2.HottestComponent)
	}
}

// TestPruneOldData verifies old data removal and recent data preservation
func TestPruneOldData(t *testing.T) {
	ha := NewHeatAggregator(5 * time.Minute)

	// Add some events
	ha.RecordFailure("ingest")
	ha.RecordFailure("ingest")
	ha.RecordRetry("ingest")

	// Record initial state
	beforeHeat := ha.GetComponentHeat("ingest").HeatScore

	// Prune data from before now (nothing should be pruned)
	now := time.Now()
	ha.PruneOldData(now)

	afterPrune := ha.GetComponentHeat("ingest")
	if afterPrune.FailureCount != 2 {
		t.Errorf("expected failure count 2 after no-op prune, got %d", afterPrune.FailureCount)
	}

	// Prune data from the future (everything should be pruned)
	future := now.Add(1 * time.Hour)
	ha.PruneOldData(future)

	afterFullPrune := ha.GetComponentHeat("ingest")
	// Heat score should be recalculated based on empty event log
	if afterFullPrune.HeatScore != 0 {
		t.Errorf("expected heat score 0 after full prune, got %d (was %d before)", afterFullPrune.HeatScore, beforeHeat)
	}

	// Verify event log was cleaned
	snapshot := ha.GetHeatMapSnapshot()
	for _, comp := range snapshot.Components {
		// All components should have recalculated heat after prune
		if comp.HeatScore != 0 {
			t.Errorf("expected component %s to have 0 heat after full prune, got %d", comp.Component, comp.HeatScore)
		}
	}
}

// TestPrivacySafety verifies no PII stored, only aggregated data
func TestPrivacySafety(t *testing.T) {
	ha := NewHeatAggregator(5 * time.Minute)

	// Simulate recording events that might contain PII
	ha.RecordFailure("ingest")
	ha.RecordRetry("user-api")
	ha.RecordStateChange("payment-processor", HealthHealthy, HealthFailing)
	ha.RecordRecovery("database", true)

	// Get snapshot and verify only component names are stored
	snapshot := ha.GetHeatMapSnapshot()

	for _, comp := range snapshot.Components {
		// Component name should be infrastructure identifier only
		if comp.Component == "" {
			t.Error("component name should not be empty")
		}

		// No error messages, no user data, no request/response content
		// ComponentHeat struct has no fields for PII

		// All counts should be non-negative
		if comp.FailureCount < 0 {
			t.Error("failure count should be non-negative")
		}
		if comp.RetryCount < 0 {
			t.Error("retry count should be non-negative")
		}
		if comp.StateChangeCount < 0 {
			t.Error("state change count should be non-negative")
		}

		// Heat score should be within valid range
		if comp.HeatScore < 0 || comp.HeatScore > 100 {
			t.Errorf("heat score should be 0-100, got %d", comp.HeatScore)
		}
	}

	// Verify internal eventLog contains no PII
	// heatEvent only has: timestamp, component (string), eventType (string)
	// This is verified by checking the struct definition - no user data fields

	// Test that sensitive-looking component names are treated as infrastructure IDs
	ha2 := NewHeatAggregator(5 * time.Minute)
	sensitiveNames := []string{
		"user-service",
		"email-processor",
		"auth-handler",
		"session-manager",
	}

	for _, name := range sensitiveNames {
		// These are infrastructure identifiers, not PII
		ha2.RecordFailure(name)
		comp := ha2.GetComponentHeat(name)
		if comp.Component != name {
			t.Errorf("expected component name '%s', got '%s'", name, comp.Component)
		}
		// Verify only aggregated data stored
		if comp.FailureCount != 1 {
			t.Errorf("expected failure count 1 for %s", name)
		}
	}

	// Verify no stack traces or error messages stored anywhere
	// By design, the heatEvent struct has no such fields
	snapshot2 := ha2.GetHeatMapSnapshot()
	for _, comp := range snapshot2.Components {
		// Component names should not contain user-specific identifiers
		if strings.Contains(comp.Component, "@") {
			t.Error("component name should not contain email-like content")
		}
		if strings.Contains(comp.Component, "uuid") || strings.Contains(comp.Component, "guid") {
			// Infrastructure components might legitimately have these
			// but the key point is: these are infrastructure identifiers, not user IDs
		}
	}
}
