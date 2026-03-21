package selfobs

import (
	"testing"
	"time"
)

// TestFailureBurstScenario tests rapid failure injection triggers failing state
func TestFailureBurstScenario(t *testing.T) {
	ce := NewChaosEngine()
	result := ce.FailureBurstScenario()

	if !result.Success {
		t.Errorf("FailureBurstScenario failed: %s", result.Error)
	}
	if result.ScenarioName != "FailureBurstScenario" {
		t.Errorf("expected scenario name FailureBurstScenario, got %s", result.ScenarioName)
	}
	if result.FinalState != HealthFailing {
		t.Errorf("expected final state HealthFailing, got %s", result.FinalState)
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}

	// Verify deterministic outcome - 25 failures should always trigger failing
	comp := ce.registry.GetComponent("ingest")
	if comp.ErrorRate() <= 20.0 {
		t.Errorf("expected error rate > 20%%, got %.2f%%", comp.ErrorRate())
	}
}

// TestRecoveryCurveScenario tests gradual recovery from failing to healthy
func TestRecoveryCurveScenario(t *testing.T) {
	ce := NewChaosEngine()
	result := ce.RecoveryCurveScenario()

	if !result.Success {
		t.Errorf("RecoveryCurveScenario failed: %s", result.Error)
	}
	if result.ScenarioName != "RecoveryCurveScenario" {
		t.Errorf("expected scenario name RecoveryCurveScenario, got %s", result.ScenarioName)
	}
	if result.FinalState != HealthHealthy {
		t.Errorf("expected final state HealthHealthy, got %s", result.FinalState)
	}

	// Verify state transitions occurred (initial -> failing -> healthy)
	transitions := ce.GetTransitions()
	if len(transitions) < 1 {
		t.Errorf("expected at least 1 transition, got %d", len(transitions))
	}

	// Verify deterministic outcome
	comp := ce.registry.GetComponent("ingest")
	if comp.ErrorRate() > 1.0 {
		t.Errorf("expected error rate <= 1%%, got %.2f%%", comp.ErrorRate())
	}
}

// TestThresholdPrecisionScenario tests exact boundary conditions
func TestThresholdPrecisionScenario(t *testing.T) {
	ce := NewChaosEngine()
	result := ce.ThresholdPrecisionScenario()

	if !result.Success {
		t.Errorf("ThresholdPrecisionScenario failed: %s", result.Error)
	}
	if result.ScenarioName != "ThresholdPrecisionScenario" {
		t.Errorf("expected scenario name ThresholdPrecisionScenario, got %s", result.ScenarioName)
	}

	// Verify boundary conditions
	comp := ce.registry.GetComponent("ingest")
	if comp.ErrorRate() != 1.0 {
		t.Errorf("expected exactly 1%% error rate for ingest, got %.2f%%", comp.ErrorRate())
	}

	comp = ce.registry.GetComponent("classify")
	if comp.ErrorRate() != 5.0 {
		t.Errorf("expected exactly 5%% error rate for classify, got %.2f%%", comp.ErrorRate())
	}

	comp = ce.registry.GetComponent("alert")
	if comp.ErrorRate() != 10.0 {
		t.Errorf("expected exactly 10%% error rate for alert, got %.2f%%", comp.ErrorRate())
	}

	comp = ce.registry.GetComponent("control")
	if comp.ErrorRate() != 20.0 {
		t.Errorf("expected exactly 20%% error rate for control, got %.2f%%", comp.ErrorRate())
	}
}

// TestStatePersistenceScenario verifies failing state does not de-escalate with more failures
func TestStatePersistenceScenario(t *testing.T) {
	ce := NewChaosEngine()
	result := ce.StatePersistenceScenario()

	if !result.Success {
		t.Errorf("StatePersistenceScenario failed: %s", result.Error)
	}
	if result.ScenarioName != "StatePersistenceScenario" {
		t.Errorf("expected scenario name StatePersistenceScenario, got %s", result.ScenarioName)
	}
	if result.FinalState != HealthFailing {
		t.Errorf("expected final state HealthFailing, got %s", result.FinalState)
	}

	// Verify 80 total failures (30 initial + 50 additional)
	comp := ce.registry.GetComponent("ingest")
	if comp.TotalOps != 80 {
		t.Errorf("expected 80 total operations, got %d", comp.TotalOps)
	}
	if comp.ErrorCount != 80 {
		t.Errorf("expected 80 errors, got %d", comp.ErrorCount)
	}
}

// TestOscillationScenario tests rapid state changes for stability
func TestOscillationScenario(t *testing.T) {
	ce := NewChaosEngine()
	result := ce.OscillationScenario()

	if !result.Success {
		t.Errorf("OscillationScenario failed: %s", result.Error)
	}
	if result.ScenarioName != "OscillationScenario" {
		t.Errorf("expected scenario name OscillationScenario, got %s", result.ScenarioName)
	}

	// Verify final state is reasonable (degraded or healthy)
	if result.FinalState != HealthDegraded && result.FinalState != HealthHealthy {
		t.Errorf("expected final state to be degraded or healthy, got %s", result.FinalState)
	}

	// Verify deterministic outcome: 25 failures, 250 successes = 9.1% error rate
	comp := ce.registry.GetComponent("ingest")
	expectedRate := 25.0 / 275.0 * 100
	if comp.ErrorRate() != expectedRate {
		t.Errorf("expected error rate %.2f%%, got %.2f%%", expectedRate, comp.ErrorRate())
	}
}

// TestZeroStateScenario tests behavior with zero initial state
func TestZeroStateScenario(t *testing.T) {
	ce := NewChaosEngine()
	result := ce.ZeroStateScenario()

	if !result.Success {
		t.Errorf("ZeroStateScenario failed: %s", result.Error)
	}
	if result.ScenarioName != "ZeroStateScenario" {
		t.Errorf("expected scenario name ZeroStateScenario, got %s", result.ScenarioName)
	}

	// Verify ingest component transitions to healthy
	comp := ce.registry.GetComponent("ingest")
	if comp.TotalOps != 1 {
		t.Errorf("expected 1 total op for ingest, got %d", comp.TotalOps)
	}
	if comp.Health != HealthHealthy {
		t.Errorf("expected ingest to be healthy, got %s", comp.Health)
	}

	// Verify classify component has 100% error rate after single failure
	comp = ce.registry.GetComponent("classify")
	if comp.TotalOps != 1 {
		t.Errorf("expected 1 total op for classify, got %d", comp.TotalOps)
	}
	if comp.ErrorCount != 1 {
		t.Errorf("expected 1 error for classify, got %d", comp.ErrorCount)
	}
	if comp.ErrorRate() != 100.0 {
		t.Errorf("expected 100%% error rate for classify, got %.2f%%", comp.ErrorRate())
	}
}

// TestConcurrencyScenario tests thread safety of health registry
func TestConcurrencyScenario(t *testing.T) {
	ce := NewChaosEngine()
	result := ce.ConcurrencyScenario()

	if !result.Success {
		t.Errorf("ConcurrencyScenario failed: %s", result.Error)
	}
	if result.ScenarioName != "ConcurrencyScenario" {
		t.Errorf("expected scenario name ConcurrencyScenario, got %s", result.ScenarioName)
	}

	// Verify totals are correct (4 workers * 25 ops = 100 total)
	comp := ce.registry.GetComponent("ingest")
	if comp.TotalOps != 100 {
		t.Errorf("expected 100 total operations, got %d", comp.TotalOps)
	}
	if comp.SuccessCount != 50 {
		t.Errorf("expected 50 successes, got %d", comp.SuccessCount)
	}
	if comp.ErrorCount != 50 {
		t.Errorf("expected 50 failures, got %d", comp.ErrorCount)
	}

	// Verify error rate is exactly 50%
	if comp.ErrorRate() != 50.0 {
		t.Errorf("expected 50%% error rate, got %.2f%%", comp.ErrorRate())
	}
}

// TestChaosEngineCreation tests creating a new chaos engine
func TestChaosEngineCreation(t *testing.T) {
	ce := NewChaosEngine()

	if ce == nil {
		t.Fatal("expected non-nil ChaosEngine")
	}
	if ce.registry == nil {
		t.Error("expected registry to be initialized")
	}
	if ce.results == nil {
		t.Error("expected results slice to be initialized")
	}
	if ce.transitions == nil {
		t.Error("expected transitions slice to be initialized")
	}

	// Verify initial component state is unknown
	state := ce.GetComponentState("ingest")
	if state != HealthUnknown {
		t.Errorf("expected initial state HealthUnknown, got %s", state)
	}
}

// TestRunAllScenarios tests running all chaos scenarios
func TestRunAllScenarios(t *testing.T) {
	ce := NewChaosEngine()
	results := ce.RunAllScenarios()

	if len(results) != 7 {
		t.Errorf("expected 7 results, got %d", len(results))
	}

	expectedNames := []string{
		"FailureBurstScenario",
		"RecoveryCurveScenario",
		"ThresholdPrecisionScenario",
		"StatePersistenceScenario",
		"OscillationScenario",
		"ZeroStateScenario",
		"ConcurrencyScenario",
	}

	for i, expectedName := range expectedNames {
		if i >= len(results) {
			t.Errorf("missing result for %s", expectedName)
			continue
		}
		if results[i].ScenarioName != expectedName {
			t.Errorf("expected result[%d].ScenarioName to be %s, got %s",
				i, expectedName, results[i].ScenarioName)
		}
		if !results[i].Success {
			t.Errorf("scenario %s failed: %s", expectedName, results[i].Error)
		}
	}

	// Verify GetResults returns the same results
	storedResults := ce.GetResults()
	if len(storedResults) != len(results) {
		t.Errorf("expected %d stored results, got %d", len(results), len(storedResults))
	}

	// Verify summary is generated
	summary := ce.Summary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}
}

// TestChaosValidation runs all scenarios and validates comprehensive behavior
func TestChaosValidation(t *testing.T) {
	ce := NewChaosEngine()
	results := ce.RunAllScenarios()

	// Validate: All scenarios pass
	passed := 0
	failed := 0
	for _, r := range results {
		if r.Success {
			passed++
		} else {
			failed++
			t.Errorf("scenario %s failed: %s", r.ScenarioName, r.Error)
		}
	}

	if failed > 0 {
		t.Errorf("expected all scenarios to pass, %d failed", failed)
	}
	if passed != 7 {
		t.Errorf("expected 7 passed scenarios, got %d", passed)
	}

	// Validate: No panics occurred (errors would indicate panics caught by recovery)
	for _, r := range results {
		if r.Error != "" {
			t.Errorf("scenario %s has error (possible panic): %s", r.ScenarioName, r.Error)
		}
	}

	// Validate: States are valid
	validStates := map[ComponentHealth]bool{
		HealthUnknown:  true,
		HealthHealthy:  true,
		HealthDegraded: true,
		HealthFailing:  true,
	}

	for _, r := range results {
		if !validStates[r.InitialState] {
			t.Errorf("scenario %s has invalid initial state: %s", r.ScenarioName, r.InitialState)
		}
		if !validStates[r.FinalState] {
			t.Errorf("scenario %s has invalid final state: %s", r.ScenarioName, r.FinalState)
		}
	}

	// Validate: Results are deterministic by running again with fresh engine
	ce2 := NewChaosEngine()
	results2 := ce2.RunAllScenarios()

	if len(results) != len(results2) {
		t.Fatalf("result count mismatch: %d vs %d", len(results), len(results2))
	}

	for i := range results {
		if results[i].ScenarioName != results2[i].ScenarioName {
			t.Errorf("scenario name mismatch at %d: %s vs %s",
				i, results[i].ScenarioName, results2[i].ScenarioName)
		}
		if results[i].Success != results2[i].Success {
			t.Errorf("success mismatch at %s: %v vs %v",
				results[i].ScenarioName, results[i].Success, results2[i].Success)
		}
		if results[i].FinalState != results2[i].FinalState {
			t.Errorf("final state mismatch at %s: %s vs %s",
				results[i].ScenarioName, results[i].FinalState, results2[i].FinalState)
		}
	}

	// Validate: Individual scenarios record their own transitions
	// Note: ResetRegistry() clears transitions between scenarios, so we check
	// that at least some scenarios recorded transitions
	hasTransitions := false
	for _, r := range results {
		if len(r.Transitions) > 0 {
			hasTransitions = true
			// Validate transition fields
			for j, tr := range r.Transitions {
				if !validStates[tr.From] {
					t.Errorf("%s transition %d has invalid from state: %s", r.ScenarioName, j, tr.From)
				}
				if !validStates[tr.To] {
					t.Errorf("%s transition %d has invalid to state: %s", r.ScenarioName, j, tr.To)
				}
				if tr.Timestamp.IsZero() {
					t.Errorf("%s transition %d has zero timestamp", r.ScenarioName, j)
				}
				if tr.Trigger == "" {
					t.Errorf("%s transition %d has empty trigger", r.ScenarioName, j)
				}
			}
		}
	}
	if !hasTransitions {
		t.Error("expected at least some scenarios to record transitions")
	}

	// Validate: Duration is reasonable (positive and not excessively long)
	maxDuration := 5 * time.Second
	totalDuration := time.Duration(0)
	for _, r := range results {
		if r.Duration <= 0 {
			t.Errorf("scenario %s has non-positive duration: %v", r.ScenarioName, r.Duration)
		}
		if r.Duration > maxDuration {
			t.Errorf("scenario %s took too long: %v", r.ScenarioName, r.Duration)
		}
		totalDuration += r.Duration
	}

	t.Logf("Chaos validation complete: %d passed, %d failed, total duration: %v",
		passed, failed, totalDuration)
}

// TestChaosEngineResetRegistry tests registry reset functionality
func TestChaosEngineResetRegistry(t *testing.T) {
	ce := NewChaosEngine()

	// Add some data
	ce.registry.RecordFailure("ingest")
	ce.RecordTransition(HealthUnknown, HealthFailing, "test")

	if ce.GetComponentState("ingest") != HealthFailing {
		t.Error("expected failing state before reset")
	}
	if len(ce.GetTransitions()) == 0 {
		t.Error("expected transitions before reset")
	}

	// Reset registry
	ce.ResetRegistry()

	// Verify reset
	if ce.GetComponentState("ingest") != HealthUnknown {
		t.Error("expected unknown state after reset")
	}
	if len(ce.GetTransitions()) != 0 {
		t.Error("expected no transitions after reset")
	}
}

// TestChaosEngineRecordTransition tests transition recording
func TestChaosEngineRecordTransition(t *testing.T) {
	ce := NewChaosEngine()

	// Record same state - should not create transition
	ce.RecordTransition(HealthHealthy, HealthHealthy, "no change")
	if len(ce.GetTransitions()) != 0 {
		t.Error("expected no transition for same state")
	}

	// Record different state - should create transition
	ce.RecordTransition(HealthHealthy, HealthDegraded, "degraded")
	transitions := ce.GetTransitions()
	if len(transitions) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(transitions))
	}
	if transitions[0].From != HealthHealthy {
		t.Errorf("expected from HealthHealthy, got %s", transitions[0].From)
	}
	if transitions[0].To != HealthDegraded {
		t.Errorf("expected to HealthDegraded, got %s", transitions[0].To)
	}
	if transitions[0].Trigger != "degraded" {
		t.Errorf("expected trigger 'degraded', got %s", transitions[0].Trigger)
	}
	if transitions[0].Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

// TestChaosResultFields validates ChaosResult struct fields
func TestChaosResultFields(t *testing.T) {
	ce := NewChaosEngine()
	result := ce.FailureBurstScenario()

	if result.ScenarioName == "" {
		t.Error("expected non-empty scenario name")
	}
	if result.Duration < 0 {
		t.Error("expected non-negative duration")
	}

	// Verify transitions slice is not nil
	if result.Transitions == nil {
		t.Error("expected non-nil transitions slice")
	}
}

// TestStateTransitionStruct validates StateTransition struct
func TestStateTransitionStruct(t *testing.T) {
	tr := StateTransition{
		From:      HealthHealthy,
		To:        HealthFailing,
		Timestamp: time.Now(),
		Trigger:   "test trigger",
	}

	if tr.From != HealthHealthy {
		t.Errorf("expected from HealthHealthy, got %s", tr.From)
	}
	if tr.To != HealthFailing {
		t.Errorf("expected to HealthFailing, got %s", tr.To)
	}
	if tr.Trigger != "test trigger" {
		t.Errorf("expected trigger 'test trigger', got %s", tr.Trigger)
	}
	if tr.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}
