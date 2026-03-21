package selfobs

import (
	"context"
	"testing"
	"time"
)

// TestHealthRegistryTransitions tests health registry transitions
func TestHealthRegistryTransitions(t *testing.T) {
	// Create a fresh registry for testing
	registry := NewHealthRegistry()

	// Test initial state - should be unknown
	comp := registry.GetComponent("ingest")
	if comp.Health != HealthUnknown {
		t.Errorf("expected initial health to be unknown, got %s", comp.Health)
	}

	// Test: unknown -> healthy (after some successes)
	registry.RecordSuccess("ingest")
	comp = registry.GetComponent("ingest")
	if comp.Health != HealthHealthy {
		t.Errorf("expected healthy after success, got %s", comp.Health)
	}
	if comp.SuccessCount != 1 {
		t.Errorf("expected 1 success, got %d", comp.SuccessCount)
	}

	// Test: healthy -> failing (after failures cause >20% error rate)
	// With 1 success, 1 failure = 50% error rate > 20% = failing
	registry.RecordFailure("ingest")
	comp = registry.GetComponent("ingest")
	if comp.Health != HealthFailing {
		t.Errorf("expected failing after 50%% error rate, got %s", comp.Health)
	}

	// Test: failing -> degraded (after more successes reduce error rate)
	// With 10 more successes: 11 success, 1 failure = 8.3% error rate > 1% = degraded
	// (error rate > 1% but <= 10%, so degraded, not healthy)
	for i := 0; i < 10; i++ {
		registry.RecordSuccess("ingest")
	}
	comp = registry.GetComponent("ingest")
	if comp.Health != HealthDegraded {
		t.Errorf("expected degraded after ~8.3%% error rate, got %s", comp.Health)
	}

	// Test: healthy -> degraded (error rate > 1% but < 10%)
	// Need 99 more successes, 1 failure: 110 success, 2 failure = 1.8% error rate > 1% = degraded
	for i := 0; i < 99; i++ {
		registry.RecordSuccess("ingest")
	}
	registry.RecordFailure("ingest")
	comp = registry.GetComponent("ingest")
	if comp.Health != HealthDegraded {
		t.Errorf("expected degraded after ~1.8%% error rate, got %s", comp.Health)
	}

	// Test: healthy -> failing (after many failures)
	for i := 0; i < 20; i++ {
		registry.RecordFailure("classify")
	}
	comp = registry.GetComponent("classify")
	// 20 failures, 0 successes = 100% error rate > 20%
	if comp.Health != HealthFailing {
		t.Errorf("expected failing after high error rate, got %s", comp.Health)
	}

	// Test GetOverallHealth
	overall := registry.GetOverallHealth()
	if overall != HealthFailing {
		t.Errorf("expected overall health to be failing, got %s", overall)
	}

	// Test GetAllComponents
	all := registry.GetAllComponents()
	if len(all) == 0 {
		t.Error("expected components to be returned")
	}
}

// TestFreshnessTracking tests freshness tracking
func TestFreshnessTracking(t *testing.T) {
	// Create a fresh tracker for testing
	tracker := NewFreshnessTracker()

	// Test initial state - should not be fresh
	if tracker.IsFresh("ingest") {
		t.Error("expected initial state to not be fresh")
	}

	// Test MarkFresh
	tracker.MarkFresh("ingest")
	if !tracker.IsFresh("ingest") {
		t.Error("expected ingest to be fresh after MarkFresh")
	}

	// Test marker properties
	marker := tracker.GetMarker("ingest")
	if marker.Component != "ingest" {
		t.Errorf("expected component name to be 'ingest', got %s", marker.Component)
	}
	if marker.LastUpdate.IsZero() {
		t.Error("expected LastUpdate to be set")
	}

	// Test stale detection - use the FreshnessMarker directly
	tracker.MarkFresh("classify")
	marker = tracker.GetMarker("classify")
	// Verify IsStale on FreshnessMarker works
	if marker.IsStale() {
		t.Error("expected classify to not be stale immediately after MarkFresh")
	}

	// Manually set LastUpdate to past to simulate stale
	marker.LastUpdate = time.Now().Add(-10 * time.Minute)
	if !marker.IsStale() {
		t.Error("expected classify to be stale after 10 minutes")
	}
	if marker.IsFresh() {
		t.Error("expected classify to not be fresh after 10 minutes")
	}

	// Test GetStaleComponents
	tracker.MarkFresh("alert") // mark alert as fresh
	stale := tracker.GetStaleComponents()
	// At this point, classify should be stale
	foundClassify := false
	for _, m := range stale {
		if m.Component == "classify" {
			foundClassify = true
			break
		}
	}
	if !foundClassify {
		t.Error("expected classify to be in stale components")
	}

	// Test GetAllMarkers
	all := tracker.GetAllMarkers()
	if len(all) == 0 {
		t.Error("expected markers to be returned")
	}

	// Test fresh components
	fresh := tracker.GetFreshComponents()
	foundAlert := false
	for _, m := range fresh {
		if m.Component == "alert" {
			foundAlert = true
			break
		}
	}
	if !foundAlert {
		t.Error("expected alert to be in fresh components")
	}
}

// TestSLOEvaluation tests SLO evaluation
func TestSLOEvaluation(t *testing.T) {
	// Create a fresh tracker for testing
	tracker := NewSLOTracker()

	// Test initial state
	status := tracker.EvaluateSLO("message_ingest_latency")
	if status.Status != "healthy" {
		t.Errorf("expected initial status to be healthy, got %s", status.Status)
	}

	// Test recording successes
	for i := 0; i < 100; i++ {
		tracker.RecordSuccess("ingest_latency_p99")
	}

	status = tracker.EvaluateSLO("message_ingest_latency")
	if status.CurrentValue != 100.0 {
		t.Errorf("expected 100%% success rate, got %f", status.CurrentValue)
	}
	if status.Status != "healthy" {
		t.Errorf("expected healthy status, got %s", status.Status)
	}

	// Test recording some failures (at_risk)
	for i := 0; i < 10; i++ {
		tracker.RecordFailure("ingest_latency_p99")
	}
	// 100 success, 10 failure = 90.9% success rate
	// Target is 95%, so error budget is 5%
	// Used error is 9.1%, budget used = 9.1/5 = 182% > 100%
	// This should be breached

	status = tracker.EvaluateSLO("message_ingest_latency")
	if status.Status != "breached" {
		t.Errorf("expected breached status, got %s (value: %f)", status.Status, status.CurrentValue)
	}

	// Test GetAllSLOStatuses
	statuses := tracker.GetAllSLOStatuses()
	if len(statuses) == 0 {
		t.Error("expected SLO statuses to be returned")
	}

	// Test GetSLODefinition
	def := tracker.GetSLODefinition("message_ingest_latency")
	if def == nil {
		t.Error("expected definition to be returned")
	}
	if def.Target != 95.0 {
		t.Errorf("expected target 95.0, got %f", def.Target)
	}
}

// TestCorrelationIDGeneration tests correlation ID generation and propagation
func TestCorrelationIDGeneration(t *testing.T) {
	// Test NewCorrelationID
	corr1 := NewCorrelationID("test")
	if corr1.ID == "" {
		t.Error("expected non-empty correlation ID")
	}
	if corr1.Source != "test" {
		t.Errorf("expected source 'test', got %s", corr1.Source)
	}

	// Test uniqueness
	corr2 := NewCorrelationID("test")
	if corr1.ID == corr2.ID {
		t.Error("expected unique correlation IDs")
	}

	// Test String() method
	if corr1.String() != corr1.ID {
		t.Error("expected String() to return ID")
	}

	// Test context propagation with ContextWithCorrelationID
	ctx := context.Background()
	ctx = ContextWithCorrelationID(ctx, corr1)

	// Test retrieving from context with FromContext
	retrieved, ok := FromContext(ctx)
	if !ok {
		t.Error("expected to find correlation ID in context")
	}
	if retrieved.ID != corr1.ID {
		t.Errorf("expected %s, got %s", corr1.ID, retrieved.ID)
	}

	// Test ContextWithNewCorrelationID
	ctx2, corr3 := ContextWithNewCorrelationID(context.Background(), "new-source")
	if corr3.ID == "" {
		t.Error("expected non-empty correlation ID")
	}
	if corr3.Source != "new-source" {
		t.Errorf("expected source 'new-source', got %s", corr3.Source)
	}
	retrieved2, ok := FromContext(ctx2)
	if !ok {
		t.Error("expected to find correlation ID in new context")
	}
	if retrieved2.ID != corr3.ID {
		t.Error("expected correlation ID to match")
	}
}

// TestCorrelationIDPool tests the correlation ID pool
func TestCorrelationIDPool(t *testing.T) {
	pool := NewCorrelationIDPool(5)

	// Test Get returns pre-allocated IDs
	corr1 := pool.Get("test")
	if corr1.ID == "" {
		t.Error("expected non-empty correlation ID")
	}

	// Test Put returns ID to pool
	pool.Put(corr1)

	// Get again should reuse or create new
	corr2 := pool.Get("test2")
	if corr2.ID == "" {
		t.Error("expected non-empty correlation ID")
	}
}

// TestPackageLevelFunctions tests package-level convenience functions
func TestPackageLevelFunctions(t *testing.T) {
	// Reset global tracker and registry for isolated testing
	oldTracker := globalFreshnessTracker
	oldRegistry := globalRegistry
	oldSLOTracker := globalSLOTracker

	defer func() {
		globalFreshnessTracker = oldTracker
		globalRegistry = oldRegistry
		globalSLOTracker = oldSLOTracker
	}()

	// Test MarkFresh
	MarkFresh("ingest")
	if !IsFresh("ingest") {
		t.Error("expected ingest to be fresh after MarkFresh")
	}

	// Test GetStaleComponents
	stale := GetStaleComponents()
	// ingest should be fresh, others might be stale
	_ = stale

	// Test SLO package functions
	RecordSLOSuccess("test_metric")
	EvaluateSLO("message_ingest_latency")
	EvaluateAllSLOs()
}

// TestSLOBuiltInDefinitions tests that built-in SLOs are registered
func TestSLOBuiltInDefinitions(t *testing.T) {
	tracker := NewSLOTracker()

	defs := tracker.GetAllDefinitions()
	if len(defs) == 0 {
		t.Error("expected built-in SLO definitions")
	}

	// Verify expected built-in SLOs exist
	found := make(map[string]bool)
	for _, def := range defs {
		found[def.Name] = true
	}

	expected := []string{
		"message_ingest_latency",
		"alert_freshness",
		"control_success_rate",
		"retention_compliance",
		"backup_success",
	}

	for _, name := range expected {
		if !found[name] {
			t.Errorf("expected SLO %s to be defined", name)
		}
	}
}

// TestHealthRegistrySetHealth tests direct health state setting
func TestHealthRegistrySetHealth(t *testing.T) {
	registry := NewHealthRegistry()

	// Test SetHealth
	registry.SetHealth("ingest", HealthDegraded)
	comp := registry.GetComponent("ingest")
	if comp.Health != HealthDegraded {
		t.Errorf("expected degraded, got %s", comp.Health)
	}

	// Test setting health for unknown component
	registry.SetHealth("unknown_component", HealthFailing)
	comp = registry.GetComponent("unknown_component")
	if comp.Health != HealthFailing {
		t.Errorf("expected failing, got %s", comp.Health)
	}
}
