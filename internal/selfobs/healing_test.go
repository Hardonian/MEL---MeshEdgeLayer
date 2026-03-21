package selfobs

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestHealingControllerCreation verifies controller initializes correctly
func TestHealingControllerCreation(t *testing.T) {
	hc := NewHealingController()

	if hc == nil {
		t.Fatal("expected healing controller to be created")
	}

	if hc.strategies == nil {
		t.Error("expected strategies map to be initialized")
	}

	if hc.auditTrail == nil {
		t.Error("expected audit trail to be initialized")
	}

	if hc.componentStates == nil {
		t.Error("expected component states to be initialized")
	}

	if hc.circuits == nil {
		t.Error("expected circuits map to be initialized")
	}

	if hc.maxAuditEntries != 100 {
		t.Errorf("expected max audit entries to be 100, got %d", hc.maxAuditEntries)
	}

	if hc.circuitFailures != DefaultCircuitFailures {
		t.Errorf("expected circuit failures to be %d, got %d", DefaultCircuitFailures, hc.circuitFailures)
	}

	if hc.circuitTimeout != DefaultCircuitTimeout {
		t.Errorf("expected circuit timeout to be %v, got %v", DefaultCircuitTimeout, hc.circuitTimeout)
	}

	policy := hc.GetPolicy()
	if policy.MaxRetries != DefaultMaxRetries {
		t.Errorf("expected default max retries to be %d, got %d", DefaultMaxRetries, policy.MaxRetries)
	}
}

// TestRetryStrategy tests retry healing behavior
func TestRetryStrategy(t *testing.T) {
	t.Run("heal_failing_components", func(t *testing.T) {
		callCount := 0
		retryFunc := func(component string) error {
			callCount++
			return nil
		}

		policy := DefaultRetryPolicy()
		strategy := NewRetryStrategy(policy, retryFunc)

		canHealDegraded := strategy.CanHeal("test", HealthDegraded)
		if !canHealDegraded {
			t.Error("expected strategy to heal degraded components")
		}

		canHealFailing := strategy.CanHeal("test", HealthFailing)
		if !canHealFailing {
			t.Error("expected strategy to heal failing components")
		}

		canHealHealthy := strategy.CanHeal("test", HealthHealthy)
		if canHealHealthy {
			t.Error("expected strategy to NOT heal healthy components")
		}

		err := strategy.Execute("test-component")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if callCount != 1 {
			t.Errorf("expected retry function to be called once, got %d", callCount)
		}
	})

	t.Run("retry_strategy_executes_once", func(t *testing.T) {
		callCount := 0
		retryFunc := func(component string) error {
			callCount++
			return nil
		}

		strategy := NewRetryStrategy(DefaultRetryPolicy(), retryFunc)

		err := strategy.Execute("test")
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}

		if callCount != 1 {
			t.Errorf("expected strategy to execute once, got %d calls", callCount)
		}
	})

	t.Run("retry_strategy_returns_error", func(t *testing.T) {
		retryFunc := func(component string) error {
			return fmt.Errorf("execution failed")
		}

		strategy := NewRetryStrategy(DefaultRetryPolicy(), retryFunc)

		err := strategy.Execute("test")
		if err == nil {
			t.Error("expected error from strategy")
		}
	})

	t.Run("strategy_name", func(t *testing.T) {
		strategy := NewRetryStrategy(DefaultRetryPolicy(), nil)
		if strategy.Name() != "retry" {
			t.Errorf("expected name 'retry', got %s", strategy.Name())
		}
	})

	t.Run("set_retry_func", func(t *testing.T) {
		strategy := NewRetryStrategy(DefaultRetryPolicy(), nil)

		executed := false
		strategy.SetRetryFunc(func(component string) error {
			executed = true
			return nil
		})

		err := strategy.Execute("test")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if !executed {
			t.Error("expected retry function to be executed after SetRetryFunc")
		}
	})
}

// TestCircuitBreaker tests circuit breaker behavior
func TestCircuitBreaker(t *testing.T) {
	t.Run("circuit_closes_after_failures", func(t *testing.T) {
		hc := NewHealingController()
		hc.SetCircuitThresholds(3, time.Hour)

		component := "test-component"

		initialState := hc.GetCircuitState(component)
		if initialState != CircuitClosed {
			t.Errorf("expected initial state to be closed, got %s", initialState)
		}

		for i := 0; i < 3; i++ {
			hc.updateCircuit(component, false)
		}

		state := hc.GetCircuitState(component)
		if state != CircuitOpen {
			t.Errorf("expected circuit to be open after 3 failures, got %s", state)
		}

		if !hc.IsCircuitOpen(component) {
			t.Error("expected IsCircuitOpen to return true")
		}
	})

	t.Run("circuit_opens_after_threshold", func(t *testing.T) {
		hc := NewHealingController()
		hc.SetCircuitThresholds(2, time.Hour)

		component := "threshold-test"

		hc.updateCircuit(component, false)
		if hc.GetCircuitState(component) != CircuitClosed {
			t.Error("expected circuit to remain closed after 1 failure")
		}

		hc.updateCircuit(component, false)
		if hc.GetCircuitState(component) != CircuitOpen {
			t.Error("expected circuit to open after reaching threshold")
		}
	})

	t.Run("half_open_state", func(t *testing.T) {
		hc := NewHealingController()
		hc.SetCircuitThresholds(1, 50*time.Millisecond)

		component := "halfopen-test"

		hc.updateCircuit(component, false)
		if hc.GetCircuitState(component) != CircuitOpen {
			t.Fatal("expected circuit to be open")
		}

		time.Sleep(60 * time.Millisecond)

		state := hc.GetCircuitState(component)
		if state != CircuitHalfOpen {
			t.Errorf("expected circuit to transition to half-open after timeout, got %s", state)
		}

		if hc.IsCircuitOpen(component) {
			t.Error("expected IsCircuitOpen to return false in half-open state")
		}
	})

	t.Run("recovery_from_half_open", func(t *testing.T) {
		hc := NewHealingController()
		hc.SetCircuitThresholds(1, 10*time.Millisecond)

		component := "recovery-test"

		hc.updateCircuit(component, false)
		time.Sleep(20 * time.Millisecond)

		hc.updateCircuit(component, true)
		hc.updateCircuit(component, true)

		state := hc.GetCircuitState(component)
		if state != CircuitClosed {
			t.Errorf("expected circuit to close after 2 consecutive successes in half-open, got %s", state)
		}
	})

	t.Run("failure_in_half_open_reopens", func(t *testing.T) {
		hc := NewHealingController()
		hc.SetCircuitThresholds(1, 10*time.Millisecond)

		component := "reopen-test"

		hc.updateCircuit(component, false)
		time.Sleep(20 * time.Millisecond)

		hc.updateCircuit(component, true)
		hc.updateCircuit(component, false)

		state := hc.GetCircuitState(component)
		if state != CircuitOpen {
			t.Errorf("expected circuit to re-open after failure in half-open, got %s", state)
		}
	})

	t.Run("reset_circuit", func(t *testing.T) {
		hc := NewHealingController()
		hc.SetCircuitThresholds(1, time.Hour)

		component := "reset-test"

		hc.updateCircuit(component, false)
		if hc.GetCircuitState(component) != CircuitOpen {
			t.Fatal("expected circuit to be open")
		}

		hc.ResetCircuit(component)

		if hc.GetCircuitState(component) != CircuitClosed {
			t.Error("expected circuit to be closed after reset")
		}

		if hc.IsCircuitOpen(component) {
			t.Error("expected IsCircuitOpen to return false after reset")
		}
	})
}

// TestAuditTrail verifies audit logging behavior
func TestAuditTrail(t *testing.T) {
	t.Run("all_actions_logged", func(t *testing.T) {
		hc := NewHealingController()

		action1 := HealingAction{
			Timestamp:    time.Now(),
			Component:    "comp1",
			ActionType:   ActionRetry,
			Success:      true,
			RetryCount:   0,
			StrategyName: "retry",
		}

		action2 := HealingAction{
			Timestamp:    time.Now(),
			Component:    "comp1",
			ActionType:   ActionRetry,
			Success:      false,
			ErrorMessage: "failed",
			RetryCount:   1,
			StrategyName: "retry",
		}

		hc.recordAction(action1)
		hc.recordAction(action2)

		trail := hc.GetAuditTrail("comp1")
		if len(trail) != 2 {
			t.Errorf("expected 2 audit entries, got %d", len(trail))
		}
	})

	t.Run("timestamps_recorded", func(t *testing.T) {
		hc := NewHealingController()
		before := time.Now()

		action := HealingAction{
			Timestamp:    time.Now(),
			Component:    "time-test",
			ActionType:   ActionRetry,
			Success:      true,
			StrategyName: "retry",
		}

		hc.recordAction(action)
		after := time.Now()

		trail := hc.GetAuditTrail("time-test")
		if len(trail) != 1 {
			t.Fatal("expected 1 audit entry")
		}

		recorded := trail[0].Timestamp
		if recorded.Before(before) || recorded.After(after) {
			t.Error("timestamp not within expected range")
		}
	})

	t.Run("component_isolation", func(t *testing.T) {
		hc := NewHealingController()

		hc.recordAction(HealingAction{
			Component:  "comp-a",
			ActionType: ActionRetry,
			Success:    true,
		})

		hc.recordAction(HealingAction{
			Component:  "comp-b",
			ActionType: ActionReset,
			Success:    false,
		})

		hc.recordAction(HealingAction{
			Component:  "comp-a",
			ActionType: ActionCooldown,
			Success:    true,
		})

		trailA := hc.GetAuditTrail("comp-a")
		if len(trailA) != 2 {
			t.Errorf("expected 2 entries for comp-a, got %d", len(trailA))
		}

		trailB := hc.GetAuditTrail("comp-b")
		if len(trailB) != 1 {
			t.Errorf("expected 1 entry for comp-b, got %d", len(trailB))
		}

		if trailB[0].ActionType != ActionReset {
			t.Error("comp-b entry has wrong action type")
		}

		trailC := hc.GetAuditTrail("comp-c")
		if len(trailC) != 0 {
			t.Error("expected empty trail for unknown component")
		}
	})

	t.Run("max_entries_enforced", func(t *testing.T) {
		hc := NewHealingController()
		hc.SetMaxAuditEntries(3)

		for i := 0; i < 5; i++ {
			hc.recordAction(HealingAction{
				Component:  "limited",
				ActionType: ActionRetry,
				Success:    true,
				RetryCount: i,
			})
		}

		trail := hc.GetAuditTrail("limited")
		if len(trail) != 3 {
			t.Errorf("expected 3 entries (max), got %d", len(trail))
		}

		if trail[0].RetryCount != 2 {
			t.Errorf("expected oldest entry to be retry 2, got %d", trail[0].RetryCount)
		}

		if trail[2].RetryCount != 4 {
			t.Errorf("expected newest entry to be retry 4, got %d", trail[2].RetryCount)
		}
	})

	t.Run("get_all_audit_trails", func(t *testing.T) {
		hc := NewHealingController()

		hc.recordAction(HealingAction{Component: "a", ActionType: ActionRetry, Success: true})
		hc.recordAction(HealingAction{Component: "b", ActionType: ActionReset, Success: false})

		all := hc.GetAllAuditTrails()
		if len(all) != 2 {
			t.Errorf("expected 2 components in all trails, got %d", len(all))
		}

		if len(all["a"]) != 1 {
			t.Error("expected 1 entry for component a")
		}

		if len(all["b"]) != 1 {
			t.Error("expected 1 entry for component b")
		}
	})

	t.Run("audit_trail_copy", func(t *testing.T) {
		hc := NewHealingController()

		hc.recordAction(HealingAction{Component: "copy-test", ActionType: ActionRetry, Success: true})

		trail1 := hc.GetAuditTrail("copy-test")
		if len(trail1) != 1 {
			t.Fatal("expected 1 entry")
		}

		trail1[0].Success = false

		trail2 := hc.GetAuditTrail("copy-test")
		if !trail2[0].Success {
			t.Error("modifying returned trail should not affect stored data")
		}
	})
}

// TestRetryPolicy tests retry policy behavior
func TestRetryPolicy(t *testing.T) {
	t.Run("backoff_intervals", func(t *testing.T) {
		policy := RetryPolicy{
			BackoffIntervals: []time.Duration{
				100 * time.Millisecond,
				200 * time.Millisecond,
				500 * time.Millisecond,
			},
			Jitter: false,
		}

		backoff0 := policy.getBackoff(0)
		if backoff0 != 100*time.Millisecond {
			t.Errorf("expected backoff 100ms for attempt 0, got %v", backoff0)
		}

		backoff1 := policy.getBackoff(1)
		if backoff1 != 200*time.Millisecond {
			t.Errorf("expected backoff 200ms for attempt 1, got %v", backoff1)
		}

		backoff2 := policy.getBackoff(2)
		if backoff2 != 500*time.Millisecond {
			t.Errorf("expected backoff 500ms for attempt 2, got %v", backoff2)
		}

		backoff5 := policy.getBackoff(5)
		if backoff5 != 500*time.Millisecond {
			t.Errorf("expected last backoff for out-of-range attempt, got %v", backoff5)
		}
	})

	t.Run("jitter_behavior", func(t *testing.T) {
		policy := RetryPolicy{
			BackoffIntervals: []time.Duration{100 * time.Millisecond},
			Jitter:           true,
		}

		var sum int64
		iterations := 100

		for i := 0; i < iterations; i++ {
			backoff := policy.getBackoff(0)
			sum += int64(backoff)

			if backoff < 80*time.Millisecond || backoff > 120*time.Millisecond {
				t.Errorf("jittered backoff %v outside expected range [80ms, 120ms]", backoff)
			}
		}

		average := time.Duration(sum / int64(iterations))
		if average < 95*time.Millisecond || average > 105*time.Millisecond {
			t.Errorf("average jittered backoff %v deviates too much from 100ms", average)
		}
	})

	t.Run("jitter_disabled", func(t *testing.T) {
		policy := RetryPolicy{
			BackoffIntervals: []time.Duration{100 * time.Millisecond},
			Jitter:           false,
		}

		for i := 0; i < 10; i++ {
			backoff := policy.getBackoff(0)
			if backoff != 100*time.Millisecond {
				t.Errorf("expected exact backoff without jitter, got %v", backoff)
			}
		}
	})

	t.Run("cooldown_enforcement", func(t *testing.T) {
		hc := NewHealingController()
		hc.SetPolicy(RetryPolicy{
			MaxRetries:       0,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   100 * time.Millisecond,
			Jitter:           false,
		})
		registry := NewHealthRegistry()
		registry.SetHealth("cooldown-test", HealthFailing)
		SetGlobalRegistry(registry)
		defer SetGlobalRegistry(NewHealthRegistry())

		callCount := 0
		strategy := NewRetryStrategy(RetryPolicy{
			MaxRetries:     0,
			CooldownWindow: 100 * time.Millisecond,
			Jitter:         false,
		}, func(component string) error {
			callCount++
			return fmt.Errorf("heal fails to trigger cooldown test")
		})

		hc.RegisterStrategy("cooldown-test", strategy)

		hc.AttemptHeal("cooldown-test")
		if callCount != 1 {
			t.Fatalf("expected 1 call, got %d", callCount)
		}

		action := hc.AttemptHeal("cooldown-test")
		if action.ActionType != ActionCooldown {
			t.Errorf("expected cooldown action, got %s", action.ActionType)
		}

		if !strings.Contains(action.ErrorMessage, "cooldown") {
			t.Errorf("expected cooldown error message, got: %s", action.ErrorMessage)
		}

		time.Sleep(110 * time.Millisecond)

		registry.SetHealth("cooldown-test", HealthFailing)
		action = hc.AttemptHeal("cooldown-test")
		if action.ActionType == ActionCooldown {
			t.Error("expected healing to proceed after cooldown expires")
		}
	})

	t.Run("default_retry_policy", func(t *testing.T) {
		policy := DefaultRetryPolicy()

		if policy.MaxRetries != DefaultMaxRetries {
			t.Errorf("expected MaxRetries %d, got %d", DefaultMaxRetries, policy.MaxRetries)
		}

		if policy.CooldownWindow != DefaultCooldownWindow {
			t.Errorf("expected CooldownWindow %v, got %v", DefaultCooldownWindow, policy.CooldownWindow)
		}

		if !policy.Jitter {
			t.Error("expected Jitter to be true by default")
		}

		expectedIntervals := []time.Duration{1 * time.Second, 5 * time.Second, 15 * time.Second}
		if len(policy.BackoffIntervals) != len(expectedIntervals) {
			t.Errorf("expected %d backoff intervals, got %d", len(expectedIntervals), len(policy.BackoffIntervals))
		}

		for i, expected := range expectedIntervals {
			if policy.BackoffIntervals[i] != expected {
				t.Errorf("expected interval %d to be %v, got %v", i, expected, policy.BackoffIntervals[i])
			}
		}
	})

	t.Run("negative_attempt_fallback", func(t *testing.T) {
		policy := RetryPolicy{
			BackoffIntervals: []time.Duration{100 * time.Millisecond},
			Jitter:           false,
		}

		backoff := policy.getBackoff(-1)
		if backoff != 100*time.Millisecond {
			t.Errorf("expected fallback to last interval for negative attempt, got %v", backoff)
		}
	})

	t.Run("empty_intervals_fallback", func(t *testing.T) {
		policy := RetryPolicy{
			BackoffIntervals: []time.Duration{},
			Jitter:           false,
		}

		backoff := policy.getBackoff(0)
		if backoff != 1*time.Second {
			t.Errorf("expected 1s fallback for empty intervals, got %v", backoff)
		}
	})
}

// TestBoundedBehavior verifies healing doesn't runaway
func TestBoundedBehavior(t *testing.T) {
	t.Run("max_retries_enforced", func(t *testing.T) {
		hc := NewHealingController()
		hc.SetPolicy(RetryPolicy{
			MaxRetries:       2,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   time.Second,
			Jitter:           false,
		})
		registry := NewHealthRegistry()
		registry.SetHealth("bounded-test", HealthFailing)
		SetGlobalRegistry(registry)
		defer SetGlobalRegistry(NewHealthRegistry())

		callCount := 0
		strategy := NewRetryStrategy(RetryPolicy{
			MaxRetries:       2,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   time.Second,
			Jitter:           false,
		}, func(component string) error {
			callCount++
			return fmt.Errorf("always fails")
		})

		hc.RegisterStrategy("bounded-test", strategy)

		action := hc.AttemptHeal("bounded-test")

		if action.Success {
			t.Error("expected healing to fail")
		}

		expectedCalls := 3
		if callCount != expectedCalls {
			t.Errorf("expected %d calls (initial + 2 retries), got %d", expectedCalls, callCount)
		}

		if action.RetryCount != 2 {
			t.Errorf("expected RetryCount to be 2 (last attempt), got %d", action.RetryCount)
		}
	})

	t.Run("no_infinite_loops", func(t *testing.T) {
		hc := NewHealingController()
		hc.SetPolicy(RetryPolicy{
			MaxRetries:       0,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   5 * time.Millisecond,
			Jitter:           false,
		})
		registry := NewHealthRegistry()
		registry.SetHealth("loop-test", HealthFailing)
		SetGlobalRegistry(registry)
		defer SetGlobalRegistry(NewHealthRegistry())

		callCount := 0
		strategy := NewRetryStrategy(RetryPolicy{
			MaxRetries:       0,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   5 * time.Millisecond,
			Jitter:           false,
		}, func(component string) error {
			callCount++
			return fmt.Errorf("failure %d", callCount)
		})

		hc.RegisterStrategy("loop-test", strategy)

		start := time.Now()
		for i := 0; i < 5; i++ {
			hc.AttemptHeal("loop-test")
			time.Sleep(2 * time.Millisecond)
		}
		elapsed := time.Since(start)

		if elapsed > 500*time.Millisecond {
			t.Errorf("took too long (%v), possible infinite loop", elapsed)
		}

		if callCount > 10 {
			t.Errorf("too many calls (%d), possible runaway", callCount)
		}
	})

	t.Run("cooldown_prevents_rapid_retry", func(t *testing.T) {
		hc := NewHealingController()
		hc.SetPolicy(RetryPolicy{
			MaxRetries:       0,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   50 * time.Millisecond,
			Jitter:           false,
		})
		registry := NewHealthRegistry()
		registry.SetHealth("rapid-test", HealthFailing)
		SetGlobalRegistry(registry)
		defer SetGlobalRegistry(NewHealthRegistry())

		callCount := 0
		strategy := NewRetryStrategy(RetryPolicy{
			MaxRetries:       0,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   50 * time.Millisecond,
			Jitter:           false,
		}, func(component string) error {
			callCount++
			return fmt.Errorf("always fails to keep component failing")
		})

		hc.RegisterStrategy("rapid-test", strategy)

		hc.AttemptHeal("rapid-test")

		for i := 0; i < 10; i++ {
			hc.AttemptHeal("rapid-test")
		}

		if callCount != 1 {
			t.Errorf("expected only 1 heal during cooldown, got %d", callCount)
		}

		time.Sleep(60 * time.Millisecond)
		registry.SetHealth("rapid-test", HealthFailing)
		hc.AttemptHeal("rapid-test")

		if callCount != 2 {
			t.Errorf("expected 2 heals after cooldown, got %d", callCount)
		}
	})

	t.Run("circuit_breaker_bounds_failures", func(t *testing.T) {
		hc := NewHealingController()
		hc.SetCircuitThresholds(2, time.Hour)
		hc.SetPolicy(RetryPolicy{
			MaxRetries:       0,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   1 * time.Millisecond,
			Jitter:           false,
		})
		registry := NewHealthRegistry()
		registry.SetHealth("circuit-bounded", HealthFailing)
		SetGlobalRegistry(registry)
		defer SetGlobalRegistry(NewHealthRegistry())

		callCount := 0
		strategy := NewRetryStrategy(RetryPolicy{
			MaxRetries:       0,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   1 * time.Millisecond,
			Jitter:           false,
		}, func(component string) error {
			callCount++
			return fmt.Errorf("always fails")
		})

		hc.RegisterStrategy("circuit-bounded", strategy)

		for i := 0; i < 10; i++ {
			hc.AttemptHeal("circuit-bounded")
			registry.SetHealth("circuit-bounded", HealthFailing)
			time.Sleep(5 * time.Millisecond)
		}

		if callCount > 5 {
			t.Errorf("expected circuit to limit calls, got %d calls", callCount)
		}

		if !hc.IsCircuitOpen("circuit-bounded") {
			t.Error("expected circuit to be open")
		}
	})
}

// TestRegisterStrategy tests strategy registration
func TestRegisterStrategy(t *testing.T) {
	t.Run("strategy_registration", func(t *testing.T) {
		hc := NewHealingController()

		strategy := NewRetryStrategy(DefaultRetryPolicy(), func(component string) error {
			return nil
		})

		hc.RegisterStrategy("my-component", strategy)

		hc.mu.RLock()
		registered, ok := hc.strategies["my-component"]
		hc.mu.RUnlock()

		if !ok {
			t.Fatal("expected strategy to be registered")
		}

		if registered != strategy {
			t.Error("registered strategy does not match")
		}
	})

	t.Run("component_strategy_mapping", func(t *testing.T) {
		hc := NewHealingController()

		retryStrategy := NewRetryStrategy(DefaultRetryPolicy(), nil)
		reconnectStrategy := NewReconnectStrategy(nil, nil, time.Second)
		resetStrategy := NewResetStrategy(nil, time.Second)

		hc.RegisterStrategy("comp-a", retryStrategy)
		hc.RegisterStrategy("comp-b", reconnectStrategy)
		hc.RegisterStrategy("comp-c", resetStrategy)

		components := hc.GetRegisteredComponents()
		if len(components) != 3 {
			t.Errorf("expected 3 registered components, got %d", len(components))
		}

		found := make(map[string]bool)
		for _, c := range components {
			found[c] = true
		}

		if !found["comp-a"] || !found["comp-b"] || !found["comp-c"] {
			t.Error("not all components found in registered list")
		}
	})

	t.Run("strategy_overwrite", func(t *testing.T) {
		hc := NewHealingController()

		strategy1 := NewRetryStrategy(DefaultRetryPolicy(), nil)
		strategy2 := NewResetStrategy(nil, time.Second)

		hc.RegisterStrategy("overwrite", strategy1)
		hc.RegisterStrategy("overwrite", strategy2)

		hc.mu.RLock()
		registered := hc.strategies["overwrite"]
		hc.mu.RUnlock()

		if registered.Name() != "reset" {
			t.Error("expected second registration to overwrite first")
		}
	})

	t.Run("unregister_strategy", func(t *testing.T) {
		hc := NewHealingController()

		strategy := NewRetryStrategy(DefaultRetryPolicy(), nil)
		hc.RegisterStrategy("unregister", strategy)

		hc.UnregisterStrategy("unregister")

		components := hc.GetRegisteredComponents()
		for _, c := range components {
			if c == "unregister" {
				t.Error("expected unregistered component to not be in list")
			}
		}
	})

	t.Run("reconnect_strategy", func(t *testing.T) {
		disconnectCalled := false
		connectCalled := false

		strategy := NewReconnectStrategy(
			func(component string) error {
				connectCalled = true
				return nil
			},
			func(component string) error {
				disconnectCalled = true
				return nil
			},
			time.Second,
		)

		if strategy.Name() != "reconnect" {
			t.Errorf("expected name 'reconnect', got %s", strategy.Name())
		}

		canHealFailing := strategy.CanHeal("test", HealthFailing)
		if !canHealFailing {
			t.Error("expected reconnect to heal failing components")
		}

		canHealDegraded := strategy.CanHeal("test", HealthDegraded)
		if canHealDegraded {
			t.Error("expected reconnect to NOT heal degraded components")
		}

		err := strategy.Execute("test")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if !disconnectCalled {
			t.Error("expected disconnect to be called")
		}

		if !connectCalled {
			t.Error("expected connect to be called")
		}
	})

	t.Run("reset_strategy", func(t *testing.T) {
		resetCalled := false

		strategy := NewResetStrategy(func(component string) error {
			resetCalled = true
			return nil
		}, time.Second)

		if strategy.Name() != "reset" {
			t.Errorf("expected name 'reset', got %s", strategy.Name())
		}

		canHealFailing := strategy.CanHeal("test", HealthFailing)
		if !canHealFailing {
			t.Error("expected reset to heal failing components")
		}

		canHealDegraded := strategy.CanHeal("test", HealthDegraded)
		if !canHealDegraded {
			t.Error("expected reset to heal degraded components")
		}

		canHealHealthy := strategy.CanHeal("test", HealthHealthy)
		if canHealHealthy {
			t.Error("expected reset to NOT heal healthy components")
		}

		err := strategy.Execute("test")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if !resetCalled {
			t.Error("expected reset function to be called")
		}
	})

	t.Run("strategy_set_functions", func(t *testing.T) {
		reconnect := NewReconnectStrategy(nil, nil, time.Second)

		connectCalled := false
		disconnectCalled := false

		reconnect.SetFunctions(
			func(component string) error {
				connectCalled = true
				return nil
			},
			func(component string) error {
				disconnectCalled = true
				return nil
			},
		)

		reconnect.Execute("test")

		if !connectCalled {
			t.Error("expected connect function to be called after SetFunctions")
		}

		if !disconnectCalled {
			t.Error("expected disconnect function to be called after SetFunctions")
		}

		reset := NewResetStrategy(nil, time.Second)
		resetCalled := false
		reset.SetResetFunc(func(component string) error {
			resetCalled = true
			return nil
		})
		reset.Execute("test")

		if !resetCalled {
			t.Error("expected reset function to be called after SetResetFunc")
		}
	})

	t.Run("nil_function_errors", func(t *testing.T) {
		retry := NewRetryStrategy(DefaultRetryPolicy(), nil)
		err := retry.Execute("test")
		if err == nil {
			t.Error("expected error when retry function is nil")
		}

		reconnect := NewReconnectStrategy(nil, nil, time.Second)
		err = reconnect.Execute("test")
		if err == nil {
			t.Error("expected error when connect function is nil")
		}

		reset := NewResetStrategy(nil, time.Second)
		err = reset.Execute("test")
		if err == nil {
			t.Error("expected error when reset function is nil")
		}
	})
}

// TestAttemptHealIntegration tests the full healing flow
func TestAttemptHealIntegration(t *testing.T) {
	t.Run("successful_heal_updates_registry", func(t *testing.T) {
		hc := NewHealingController()
		registry := NewHealthRegistry()
		registry.SetHealth("heal-test", HealthFailing)
		SetGlobalRegistry(registry)
		defer SetGlobalRegistry(NewHealthRegistry())

		strategy := NewRetryStrategy(RetryPolicy{
			MaxRetries:     0,
			CooldownWindow: time.Second,
			Jitter:         false,
		}, func(component string) error {
			return nil
		})

		hc.RegisterStrategy("heal-test", strategy)

		action := hc.AttemptHeal("heal-test")

		if !action.Success {
			t.Errorf("expected healing to succeed, got error: %s", action.ErrorMessage)
		}

		if action.ActionType != ActionRetry {
			t.Errorf("expected action type retry, got %s", action.ActionType)
		}

		comp := registry.GetComponent("heal-test")
		if comp.Health != HealthHealthy {
			t.Errorf("expected component to be healthy after heal, got %s", comp.Health)
		}
	})

	t.Run("no_strategy_returns_error", func(t *testing.T) {
		hc := NewHealingController()

		action := hc.AttemptHeal("no-strategy")

		if action.Success {
			t.Error("expected healing to fail without strategy")
		}

		if action.ActionType != ActionCooldown {
			t.Errorf("expected cooldown action, got %s", action.ActionType)
		}

		if !strings.Contains(action.ErrorMessage, "no healing strategy") {
			t.Errorf("expected 'no healing strategy' error, got: %s", action.ErrorMessage)
		}
	})

	t.Run("healthy_component_no_heal_needed", func(t *testing.T) {
		hc := NewHealingController()
		registry := NewHealthRegistry()
		registry.SetHealth("healthy-comp", HealthHealthy)
		SetGlobalRegistry(registry)
		defer SetGlobalRegistry(NewHealthRegistry())

		callCount := 0
		strategy := NewRetryStrategy(DefaultRetryPolicy(), func(component string) error {
			callCount++
			return nil
		})

		hc.RegisterStrategy("healthy-comp", strategy)

		action := hc.AttemptHeal("healthy-comp")

		if !action.Success {
			t.Error("expected action to succeed (no heal needed is not a failure)")
		}

		if action.ActionType != ActionCooldown {
			t.Errorf("expected cooldown action for healthy component, got %s", action.ActionType)
		}

		if callCount != 0 {
			t.Errorf("expected strategy not to be called for healthy component, got %d calls", callCount)
		}
	})

	t.Run("concurrent_healing_safe", func(t *testing.T) {
		hc := NewHealingController()
		registry := NewHealthRegistry()
		registry.SetHealth("concurrent", HealthFailing)
		SetGlobalRegistry(registry)
		defer SetGlobalRegistry(NewHealthRegistry())

		var mu sync.Mutex
		callCount := 0
		strategy := NewRetryStrategy(RetryPolicy{
			MaxRetries:       0,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   50 * time.Millisecond,
			Jitter:           false,
		}, func(component string) error {
			mu.Lock()
			callCount++
			mu.Unlock()
			return nil
		})

		hc.RegisterStrategy("concurrent", strategy)

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				hc.AttemptHeal("concurrent")
			}()
		}

		wg.Wait()

		mu.Lock()
		if callCount > 10 {
			t.Errorf("expected at most 10 calls, got %d", callCount)
		}
		mu.Unlock()
	})
}

// TestSetPolicy tests policy configuration
func TestSetPolicy(t *testing.T) {
	hc := NewHealingController()

	customPolicy := RetryPolicy{
		MaxRetries:       10,
		BackoffIntervals: []time.Duration{1 * time.Second},
		CooldownWindow:   5 * time.Minute,
		Jitter:           false,
	}

	hc.SetPolicy(customPolicy)

	retrieved := hc.GetPolicy()
	if retrieved.MaxRetries != 10 {
		t.Errorf("expected MaxRetries 10, got %d", retrieved.MaxRetries)
	}

	if retrieved.CooldownWindow != 5*time.Minute {
		t.Errorf("expected CooldownWindow 5m, got %v", retrieved.CooldownWindow)
	}

	if retrieved.Jitter {
		t.Error("expected Jitter to be false")
	}
}

// TestHealingActionTypes tests action type constants
func TestHealingActionTypes(t *testing.T) {
	if ActionRetry != "retry" {
		t.Errorf("expected ActionRetry to be 'retry', got %s", ActionRetry)
	}

	if ActionReconnect != "reconnect" {
		t.Errorf("expected ActionReconnect to be 'reconnect', got %s", ActionReconnect)
	}

	if ActionReset != "reset" {
		t.Errorf("expected ActionReset to be 'reset', got %s", ActionReset)
	}

	if ActionCooldown != "cooldown" {
		t.Errorf("expected ActionCooldown to be 'cooldown', got %s", ActionCooldown)
	}
}

// TestCircuitStateConstants tests circuit state constants
func TestCircuitStateConstants(t *testing.T) {
	if CircuitClosed != "closed" {
		t.Errorf("expected CircuitClosed to be 'closed', got %s", CircuitClosed)
	}

	if CircuitOpen != "open" {
		t.Errorf("expected CircuitOpen to be 'open', got %s", CircuitOpen)
	}

	if CircuitHalfOpen != "half-open" {
		t.Errorf("expected CircuitHalfOpen to be 'half-open', got %s", CircuitHalfOpen)
	}
}

// TestReconnectStrategyErrors tests error handling in reconnect strategy
func TestReconnectStrategyErrors(t *testing.T) {
	t.Run("disconnect_error", func(t *testing.T) {
		strategy := NewReconnectStrategy(
			func(component string) error {
				return nil
			},
			func(component string) error {
				return fmt.Errorf("disconnect failed")
			},
			time.Second,
		)

		err := strategy.Execute("test")
		if err == nil {
			t.Error("expected error when disconnect fails")
		}

		if !strings.Contains(err.Error(), "disconnect failed") {
			t.Errorf("expected disconnect error message, got: %v", err)
		}
	})

	t.Run("connect_error", func(t *testing.T) {
		strategy := NewReconnectStrategy(
			func(component string) error {
				return fmt.Errorf("connect failed")
			},
			func(component string) error {
				return nil
			},
			time.Second,
		)

		err := strategy.Execute("test")
		if err == nil {
			t.Error("expected error when connect fails")
		}

		if !strings.Contains(err.Error(), "connect failed") {
			t.Errorf("expected connect error message, got: %v", err)
		}
	})
}

// TestGetCooldown tests strategy cooldown methods
func TestGetCooldown(t *testing.T) {
	t.Run("retry_strategy_cooldown", func(t *testing.T) {
		policy := RetryPolicy{CooldownWindow: 5 * time.Second}
		strategy := NewRetryStrategy(policy, nil)

		cooldown := strategy.GetCooldown()
		if cooldown != 5*time.Second {
			t.Errorf("expected cooldown 5s, got %v", cooldown)
		}
	})

	t.Run("reconnect_strategy_cooldown", func(t *testing.T) {
		strategy := NewReconnectStrategy(nil, nil, 10*time.Second)

		cooldown := strategy.GetCooldown()
		if cooldown != 10*time.Second {
			t.Errorf("expected cooldown 10s, got %v", cooldown)
		}
	})

	t.Run("reset_strategy_cooldown", func(t *testing.T) {
		strategy := NewResetStrategy(nil, 15*time.Second)

		cooldown := strategy.GetCooldown()
		if cooldown != 15*time.Second {
			t.Errorf("expected cooldown 15s, got %v", cooldown)
		}
	})
}

// TestDefaultConstants tests default value constants
func TestDefaultConstants(t *testing.T) {
	if DefaultMaxRetries != 3 {
		t.Errorf("expected DefaultMaxRetries = 3, got %d", DefaultMaxRetries)
	}

	if DefaultCooldownWindow != 60*time.Second {
		t.Errorf("expected DefaultCooldownWindow = 60s, got %v", DefaultCooldownWindow)
	}

	if DefaultCircuitFailures != 5 {
		t.Errorf("expected DefaultCircuitFailures = 5, got %d", DefaultCircuitFailures)
	}

	if DefaultCircuitTimeout != 30*time.Second {
		t.Errorf("expected DefaultCircuitTimeout = 30s, got %v", DefaultCircuitTimeout)
	}
}
