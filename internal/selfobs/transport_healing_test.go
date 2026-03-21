package selfobs

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestTransportHealingControllerCreation tests initialization and embedded HealingController
func TestTransportHealingControllerCreation(t *testing.T) {
	t.Run("initialization", func(t *testing.T) {
		thc := NewTransportHealingController()

		if thc == nil {
			t.Fatal("expected transport healing controller to be created")
		}

		if thc.HealingController == nil {
			t.Error("expected embedded HealingController to be initialized")
		}

		if thc.transportStrategies == nil {
			t.Error("expected transport strategies map to be initialized")
		}

		if thc.transportStates == nil {
			t.Error("expected transport states map to be initialized")
		}

		if thc.failoverInfo == nil {
			t.Error("expected failover info map to be initialized")
		}

		if thc.latencyTracking == nil {
			t.Error("expected latency tracking map to be initialized")
		}

		if thc.failoverCooldown != 5*time.Minute {
			t.Errorf("expected default failover cooldown to be 5m, got %v", thc.failoverCooldown)
		}

		if thc.maxFailoversPerHour != 3 {
			t.Errorf("expected default max failovers per hour to be 3, got %d", thc.maxFailoversPerHour)
		}
	})

	t.Run("embedded_healing_controller", func(t *testing.T) {
		thc := NewTransportHealingController()

		// Test that embedded controller methods work
		strategy := NewRetryStrategy(DefaultRetryPolicy(), func(component string) error {
			return nil
		})

		thc.RegisterStrategy("test-component", strategy)

		components := thc.GetRegisteredComponents()
		found := false
		for _, c := range components {
			if c == "test-component" {
				found = true
				break
			}
		}

		if !found {
			t.Error("expected embedded HealingController to work correctly")
		}
	})
}

// TestMQTTRetryStrategy tests MQTT retry strategy
func TestMQTTRetryStrategy(t *testing.T) {
	t.Run("can_heal_mqtt_transport", func(t *testing.T) {
		policy := DefaultRetryPolicy()
		strategy := NewMQTTRetryStrategy(policy, nil)

		if !strategy.CanHeal(string(TransportMQTT), HealthDegraded) {
			t.Error("expected strategy to heal degraded MQTT transport")
		}

		if !strategy.CanHeal(string(TransportMQTT), HealthFailing) {
			t.Error("expected strategy to heal failing MQTT transport")
		}

		if strategy.CanHeal(string(TransportMQTT), HealthHealthy) {
			t.Error("expected strategy to NOT heal healthy MQTT transport")
		}

		if strategy.CanHeal("direct", HealthFailing) {
			t.Error("expected strategy to NOT heal non-MQTT transport")
		}
	})

	t.Run("execute_with_mock_success", func(t *testing.T) {
		callCount := 0
		reconnectFunc := func(config MQTTConfig) error {
			callCount++
			return nil
		}

		policy := RetryPolicy{
			MaxRetries:       0,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   time.Second,
			Jitter:           false,
		}
		strategy := NewMQTTRetryStrategy(policy, reconnectFunc)

		config := MQTTConfig{
			BrokerURL:    "tcp://localhost:1883",
			ClientID:     "test-client",
			QoS:          1,
			CleanSession: true,
		}

		err := strategy.ExecuteTransport(string(TransportMQTT), config)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if callCount != 1 {
			t.Errorf("expected reconnect function to be called once, got %d", callCount)
		}
	})

	t.Run("execute_with_mock_retry", func(t *testing.T) {
		callCount := 0
		reconnectFunc := func(config MQTTConfig) error {
			callCount++
			if callCount < 2 {
				return errors.New("connection refused")
			}
			return nil
		}

		policy := RetryPolicy{
			MaxRetries:       2,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   time.Second,
			Jitter:           false,
		}
		strategy := NewMQTTRetryStrategy(policy, reconnectFunc)

		config := MQTTConfig{
			BrokerURL:    "tcp://localhost:1883",
			ClientID:     "test-client",
			QoS:          1,
			CleanSession: true,
		}

		err := strategy.ExecuteTransport(string(TransportMQTT), config)
		if err != nil {
			t.Errorf("expected no error after retry, got %v", err)
		}

		if callCount != 2 {
			t.Errorf("expected reconnect function to be called twice, got %d", callCount)
		}
	})

	t.Run("execute_exhausts_retries", func(t *testing.T) {
		callCount := 0
		reconnectFunc := func(config MQTTConfig) error {
			callCount++
			return errors.New("persistent failure")
		}

		policy := RetryPolicy{
			MaxRetries:       1,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   time.Second,
			Jitter:           false,
		}
		strategy := NewMQTTRetryStrategy(policy, reconnectFunc)

		config := MQTTConfig{
			BrokerURL: "tcp://localhost:1883",
			ClientID:  "test-client",
		}

		err := strategy.ExecuteTransport(string(TransportMQTT), config)
		if err == nil {
			t.Error("expected error after exhausting retries")
		}

		if callCount != 2 {
			t.Errorf("expected 2 calls (initial + 1 retry), got %d", callCount)
		}
	})

	t.Run("execute_wrong_config_type", func(t *testing.T) {
		strategy := NewMQTTRetryStrategy(DefaultRetryPolicy(), nil)
		config := DirectConfig{Endpoint: "http://localhost"}

		err := strategy.ExecuteTransport(string(TransportMQTT), config)
		if err == nil {
			t.Error("expected error for wrong config type")
		}
		if !strings.Contains(err.Error(), "invalid config type") {
			t.Errorf("expected 'invalid config type' error, got: %v", err)
		}
	})

	t.Run("execute_no_reconnect_func", func(t *testing.T) {
		strategy := NewMQTTRetryStrategy(DefaultRetryPolicy(), nil)
		config := MQTTConfig{BrokerURL: "tcp://localhost:1883"}

		err := strategy.ExecuteTransport(string(TransportMQTT), config)
		if err == nil {
			t.Error("expected error when reconnect function is nil")
		}
		if !strings.Contains(err.Error(), "no reconnect function configured") {
			t.Errorf("expected 'no reconnect function' error, got: %v", err)
		}
	})

	t.Run("circuit_breaker_integration", func(t *testing.T) {
		thc := NewTransportHealingController()
		callCount := 0
		reconnectFunc := func(config MQTTConfig) error {
			callCount++
			return errors.New("always fails")
		}

		policy := RetryPolicy{
			MaxRetries:       0,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   1 * time.Millisecond,
			Jitter:           false,
		}
		strategy := NewMQTTRetryStrategy(policy, reconnectFunc)

		thc.RegisterTransportStrategy(string(TransportMQTT), strategy)

		config := MQTTConfig{
			BrokerURL: "tcp://localhost:1883",
			ClientID:  "test-client",
		}

		// Trigger multiple failures to open circuit
		for i := 0; i < 6; i++ {
			thc.AttemptTransportHeal(string(TransportMQTT), config)
			thc.RecordTransportFailure(string(TransportMQTT), errors.New("failure"))
			time.Sleep(2 * time.Millisecond)
		}

		if !thc.IsCircuitOpen(string(TransportMQTT)) {
			t.Error("expected circuit to be open after multiple failures")
		}
	})

	t.Run("clean_session_toggle", func(t *testing.T) {
		sessions := []bool{}
		reconnectFunc := func(config MQTTConfig) error {
			sessions = append(sessions, config.CleanSession)
			if len(sessions) < 2 {
				return errors.New("retry")
			}
			return nil
		}

		policy := RetryPolicy{
			MaxRetries:       2,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   time.Second,
			Jitter:           false,
		}
		strategy := NewMQTTRetryStrategy(policy, reconnectFunc)

		config := MQTTConfig{
			BrokerURL:    "tcp://localhost:1883",
			ClientID:     "test-client",
			CleanSession: true,
		}

		strategy.ExecuteTransport(string(TransportMQTT), config)

		if len(sessions) != 2 {
			t.Fatalf("expected 2 reconnect attempts, got %d", len(sessions))
		}

		if sessions[0] != true {
			t.Error("expected initial CleanSession to be true")
		}

		if sessions[1] != false {
			t.Error("expected CleanSession to be toggled to false on retry")
		}
	})

	t.Run("strategy_name_and_type", func(t *testing.T) {
		strategy := NewMQTTRetryStrategy(DefaultRetryPolicy(), nil)

		if strategy.Name() != "mqtt_retry" {
			t.Errorf("expected name 'mqtt_retry', got %s", strategy.Name())
		}

		if strategy.GetTransportType() != string(TransportMQTT) {
			t.Errorf("expected transport type 'mqtt', got %s", strategy.GetTransportType())
		}
	})

	t.Run("set_reconnect_func", func(t *testing.T) {
		strategy := NewMQTTRetryStrategy(DefaultRetryPolicy(), nil)

		called := false
		strategy.SetReconnectFunc(func(config MQTTConfig) error {
			called = true
			return nil
		})

		config := MQTTConfig{BrokerURL: "tcp://localhost:1883"}
		strategy.ExecuteTransport(string(TransportMQTT), config)

		if !called {
			t.Error("expected SetReconnectFunc to update the function")
		}
	})

	t.Run("cooldown_from_policy", func(t *testing.T) {
		policy := RetryPolicy{CooldownWindow: 10 * time.Second}
		strategy := NewMQTTRetryStrategy(policy, nil)

		if strategy.GetCooldown() != 10*time.Second {
			t.Errorf("expected cooldown 10s, got %v", strategy.GetCooldown())
		}
	})
}

// TestMQTTBrokerFailoverStrategy tests broker failover logic
func TestMQTTBrokerFailoverStrategy(t *testing.T) {
	t.Run("can_heal_when_backup_available", func(t *testing.T) {
		backups := []string{"tcp://backup1:1883", "tcp://backup2:1883"}
		strategy := NewMQTTBrokerFailoverStrategy(backups, nil)

		if !strategy.CanHeal(string(TransportMQTT), HealthFailing) {
			t.Error("expected strategy to heal when backups available")
		}

		if strategy.CanHeal(string(TransportMQTT), HealthDegraded) {
			t.Error("expected strategy to NOT heal degraded state (only failing)")
		}

		if strategy.CanHeal("direct", HealthFailing) {
			t.Error("expected strategy to NOT heal non-MQTT transport")
		}
	})

	t.Run("cannot_heal_without_backups", func(t *testing.T) {
		strategy := NewMQTTBrokerFailoverStrategy([]string{}, nil)

		if strategy.CanHeal(string(TransportMQTT), HealthFailing) {
			t.Error("expected strategy to NOT heal without backup brokers")
		}
	})

	t.Run("execute_failover", func(t *testing.T) {
		calledWith := ""
		failoverFunc := func(primaryConfig MQTTConfig, backupBroker string) error {
			calledWith = backupBroker
			return nil
		}

		backups := []string{"tcp://backup1:1883", "tcp://backup2:1883"}
		strategy := NewMQTTBrokerFailoverStrategy(backups, failoverFunc)

		config := MQTTConfig{BrokerURL: "tcp://primary:1883"}
		err := strategy.ExecuteTransport(string(TransportMQTT), config)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if calledWith != "tcp://backup1:1883" {
			t.Errorf("expected failover to first backup, got %s", calledWith)
		}
	})

	t.Run("execute_failover_rotates", func(t *testing.T) {
		calledWith := []string{}
		failoverFunc := func(primaryConfig MQTTConfig, backupBroker string) error {
			calledWith = append(calledWith, backupBroker)
			return nil
		}

		backups := []string{"tcp://backup1:1883", "tcp://backup2:1883"}
		strategy := NewMQTTBrokerFailoverStrategy(backups, failoverFunc)

		config := MQTTConfig{BrokerURL: "tcp://primary:1883"}

		// First failover
		strategy.ExecuteTransport(string(TransportMQTT), config)
		// Second failover
		strategy.ExecuteTransport(string(TransportMQTT), config)
		// Third failover (should wrap around)
		strategy.ExecuteTransport(string(TransportMQTT), config)

		if len(calledWith) != 3 {
			t.Fatalf("expected 3 failovers, got %d", len(calledWith))
		}

		if calledWith[0] != "tcp://backup1:1883" {
			t.Errorf("expected first failover to backup1, got %s", calledWith[0])
		}

		if calledWith[1] != "tcp://backup2:1883" {
			t.Errorf("expected second failover to backup2, got %s", calledWith[1])
		}

		if calledWith[2] != "tcp://backup1:1883" {
			t.Errorf("expected third failover to wrap to backup1, got %s", calledWith[2])
		}
	})

	t.Run("cooldown_enforcement", func(t *testing.T) {
		thc := NewTransportHealingController()
		thc.SetFailoverCooldown(100 * time.Millisecond)

		callCount := 0
		failoverFunc := func(primaryConfig MQTTConfig, backupBroker string) error {
			callCount++
			return nil
		}

		backups := []string{"tcp://backup:1883"}
		strategy := NewMQTTBrokerFailoverStrategy(backups, failoverFunc)

		thc.RegisterTransportStrategy(string(TransportMQTT), strategy)

		// Put transport in failing state (need 2 failures to reach failing)
		thc.RecordTransportFailure(string(TransportMQTT), errors.New("failure 1"))
		thc.RecordTransportFailure(string(TransportMQTT), errors.New("failure 2"))

		config := MQTTConfig{BrokerURL: "tcp://primary:1883"}

		// First failover
		thc.AttemptTransportHeal(string(TransportMQTT), config)

		// Put transport back in failing state for second attempt to check cooldown
		thc.RecordTransportFailure(string(TransportMQTT), errors.New("failure 3"))
		thc.RecordTransportFailure(string(TransportMQTT), errors.New("failure 4"))

		// Second attempt (should be blocked by cooldown)
		action := thc.AttemptTransportHeal(string(TransportMQTT), config)

		if callCount != 1 {
			t.Errorf("expected 1 failover during cooldown, got %d", callCount)
		}

		if !strings.Contains(action.ErrorMessage, "cooldown") {
			t.Errorf("expected cooldown error, got: %s", action.ErrorMessage)
		}
	})

	t.Run("max_failover_limit", func(t *testing.T) {
		thc := NewTransportHealingController()
		thc.SetMaxFailoversPerHour(2)
		thc.SetFailoverCooldown(1 * time.Millisecond) // Use controller cooldown, not strategy cooldown

		callCount := 0
		failoverFunc := func(primaryConfig MQTTConfig, backupBroker string) error {
			callCount++
			return nil
		}

		// Use MQTTBrokerFailoverStrategy which counts toward max failovers
		backups := []string{"tcp://backup1:1883", "tcp://backup2:1883"}
		strategy := NewMQTTBrokerFailoverStrategy(backups, failoverFunc)

		thc.RegisterTransportStrategy(string(TransportMQTT), strategy)

		config := MQTTConfig{BrokerURL: "tcp://primary:1883"}

		// First failover (need failing state)
		thc.RecordTransportFailure(string(TransportMQTT), errors.New("failure 1"))
		thc.RecordTransportFailure(string(TransportMQTT), errors.New("failure 2"))
		thc.AttemptTransportHeal(string(TransportMQTT), config)
		time.Sleep(2 * time.Millisecond) // Wait for controller cooldown

		// Second failover (reset to failing state)
		thc.RecordTransportFailure(string(TransportMQTT), errors.New("failure 3"))
		thc.RecordTransportFailure(string(TransportMQTT), errors.New("failure 4"))
		thc.AttemptTransportHeal(string(TransportMQTT), config)
		time.Sleep(2 * time.Millisecond) // Wait for controller cooldown

		// Third attempt (should be blocked by max failovers limit)
		thc.RecordTransportFailure(string(TransportMQTT), errors.New("failure 5"))
		thc.RecordTransportFailure(string(TransportMQTT), errors.New("failure 6"))
		action := thc.AttemptTransportHeal(string(TransportMQTT), config)

		if callCount != 2 {
			t.Errorf("expected 2 failovers max, got %d", callCount)
		}

		if !strings.Contains(action.ErrorMessage, "max failovers") {
			t.Errorf("expected max failovers error, got: %s", action.ErrorMessage)
		}
	})

	t.Run("strategy_name_and_type", func(t *testing.T) {
		strategy := NewMQTTBrokerFailoverStrategy([]string{"tcp://backup:1883"}, nil)

		if strategy.Name() != "mqtt_broker_failover" {
			t.Errorf("expected name 'mqtt_broker_failover', got %s", strategy.Name())
		}

		if strategy.GetTransportType() != string(TransportMQTT) {
			t.Errorf("expected transport type 'mqtt', got %s", strategy.GetTransportType())
		}
	})

	t.Run("execute_wrong_config_type", func(t *testing.T) {
		strategy := NewMQTTBrokerFailoverStrategy([]string{"tcp://backup:1883"}, nil)
		config := DirectConfig{Endpoint: "http://localhost"}

		err := strategy.ExecuteTransport(string(TransportMQTT), config)
		if err == nil {
			t.Error("expected error for wrong config type")
		}
	})

	t.Run("execute_no_failover_func", func(t *testing.T) {
		strategy := NewMQTTBrokerFailoverStrategy([]string{"tcp://backup:1883"}, nil)
		config := MQTTConfig{BrokerURL: "tcp://primary:1883"}

		err := strategy.ExecuteTransport(string(TransportMQTT), config)
		if err == nil {
			t.Error("expected error when failover function is nil")
		}
		if !strings.Contains(err.Error(), "no failover function configured") {
			t.Errorf("expected 'no failover function' error, got: %v", err)
		}
	})

	t.Run("cooldown_duration", func(t *testing.T) {
		strategy := NewMQTTBrokerFailoverStrategy([]string{"tcp://backup:1883"}, nil)

		if strategy.GetCooldown() != 30*time.Second {
			t.Errorf("expected cooldown 30s, got %v", strategy.GetCooldown())
		}
	})

	t.Run("set_failover_func", func(t *testing.T) {
		strategy := NewMQTTBrokerFailoverStrategy([]string{"tcp://backup:1883"}, nil)

		called := false
		strategy.SetFailoverFunc(func(primaryConfig MQTTConfig, backupBroker string) error {
			called = true
			return nil
		})

		config := MQTTConfig{BrokerURL: "tcp://primary:1883"}
		strategy.ExecuteTransport(string(TransportMQTT), config)

		if !called {
			t.Error("expected SetFailoverFunc to update the function")
		}
	})
}

// TestMQTTQoSAdjustmentStrategy tests QoS adjustment logic
func TestMQTTQoSAdjustmentStrategy(t *testing.T) {
	t.Run("can_heal_degraded_mqtt", func(t *testing.T) {
		strategy := NewMQTTQoSAdjustmentStrategy(0, nil)

		if !strategy.CanHeal(string(TransportMQTT), HealthDegraded) {
			t.Error("expected strategy to heal degraded MQTT transport")
		}

		if strategy.CanHeal(string(TransportMQTT), HealthFailing) {
			t.Error("expected strategy to NOT heal failing transport (only degraded)")
		}

		if strategy.CanHeal("direct", HealthDegraded) {
			t.Error("expected strategy to NOT heal non-MQTT transport")
		}
	})

	t.Run("execute_qos_adjustment", func(t *testing.T) {
		calledWithQoS := byte(255)
		adjustFunc := func(config MQTTConfig, newQoS byte) error {
			calledWithQoS = newQoS
			return nil
		}

		strategy := NewMQTTQoSAdjustmentStrategy(0, adjustFunc)

		config := MQTTConfig{BrokerURL: "tcp://localhost:1883", QoS: 2}
		err := strategy.ExecuteTransport(string(TransportMQTT), config)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if calledWithQoS != 0 {
			t.Errorf("expected QoS adjusted to 0, got %d", calledWithQoS)
		}
	})

	t.Run("execute_stores_original_qos", func(t *testing.T) {
		strategy := NewMQTTQoSAdjustmentStrategy(1, nil)

		config := MQTTConfig{BrokerURL: "tcp://localhost:1883", QoS: 2}
		strategy.ExecuteTransport(string(TransportMQTT), config)

		// Access original QoS through RestoreQoS
		var restoredQoS byte
		strategy.SetAdjustFunc(func(config MQTTConfig, newQoS byte) error {
			restoredQoS = newQoS
			return nil
		})

		strategy.RestoreQoS(config)

		if restoredQoS != 2 {
			t.Errorf("expected original QoS to be stored as 2, got %d", restoredQoS)
		}
	})

	t.Run("revert_after_success", func(t *testing.T) {
		strategy := NewMQTTQoSAdjustmentStrategy(0, func(config MQTTConfig, newQoS byte) error {
			return nil
		})

		config := MQTTConfig{BrokerURL: "tcp://localhost:1883", QoS: 2}
		strategy.ExecuteTransport(string(TransportMQTT), config)

		// Now restore
		restoredQoS := byte(255)
		strategy.SetAdjustFunc(func(config MQTTConfig, newQoS byte) error {
			restoredQoS = newQoS
			return nil
		})

		err := strategy.RestoreQoS(config)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if restoredQoS != 2 {
			t.Errorf("expected QoS restored to 2, got %d", restoredQoS)
		}
	})

	t.Run("revert_no_adjust_func", func(t *testing.T) {
		strategy := NewMQTTQoSAdjustmentStrategy(0, nil)
		strategy.ExecuteTransport(string(TransportMQTT), MQTTConfig{QoS: 2})

		err := strategy.RestoreQoS(MQTTConfig{QoS: 2})
		if err == nil {
			t.Error("expected error when adjust function is nil")
		}
	})

	t.Run("execute_wrong_config_type", func(t *testing.T) {
		strategy := NewMQTTQoSAdjustmentStrategy(0, nil)
		config := DirectConfig{Endpoint: "http://localhost"}

		err := strategy.ExecuteTransport(string(TransportMQTT), config)
		if err == nil {
			t.Error("expected error for wrong config type")
		}
	})

	t.Run("execute_no_adjust_func", func(t *testing.T) {
		strategy := NewMQTTQoSAdjustmentStrategy(0, nil)
		config := MQTTConfig{BrokerURL: "tcp://localhost:1883"}

		err := strategy.ExecuteTransport(string(TransportMQTT), config)
		if err == nil {
			t.Error("expected error when adjust function is nil")
		}
	})

	t.Run("strategy_name_and_type", func(t *testing.T) {
		strategy := NewMQTTQoSAdjustmentStrategy(1, nil)

		if strategy.Name() != "mqtt_qos_adjust" {
			t.Errorf("expected name 'mqtt_qos_adjust', got %s", strategy.Name())
		}

		if strategy.GetTransportType() != string(TransportMQTT) {
			t.Errorf("expected transport type 'mqtt', got %s", strategy.GetTransportType())
		}
	})

	t.Run("cooldown_duration", func(t *testing.T) {
		strategy := NewMQTTQoSAdjustmentStrategy(0, nil)

		if strategy.GetCooldown() != 5*time.Second {
			t.Errorf("expected cooldown 5s, got %v", strategy.GetCooldown())
		}
	})

	t.Run("set_adjust_func", func(t *testing.T) {
		strategy := NewMQTTQoSAdjustmentStrategy(0, nil)

		called := false
		strategy.SetAdjustFunc(func(config MQTTConfig, newQoS byte) error {
			called = true
			return nil
		})

		config := MQTTConfig{BrokerURL: "tcp://localhost:1883"}
		strategy.ExecuteTransport(string(TransportMQTT), config)

		if !called {
			t.Error("expected SetAdjustFunc to update the function")
		}
	})
}

// TestDirectRetryStrategy tests direct transport retry strategy
func TestDirectRetryStrategy(t *testing.T) {
	t.Run("can_heal_direct_transport", func(t *testing.T) {
		policy := DefaultRetryPolicy()
		strategy := NewDirectRetryStrategy(policy, nil)

		if !strategy.CanHeal(string(TransportDirect), HealthDegraded) {
			t.Error("expected strategy to heal degraded direct transport")
		}

		if !strategy.CanHeal(string(TransportDirect), HealthFailing) {
			t.Error("expected strategy to heal failing direct transport")
		}

		if strategy.CanHeal(string(TransportDirect), HealthHealthy) {
			t.Error("expected strategy to NOT heal healthy direct transport")
		}

		if strategy.CanHeal("mqtt", HealthFailing) {
			t.Error("expected strategy to NOT heal non-direct transport")
		}
	})

	t.Run("execute_with_mock_success", func(t *testing.T) {
		callCount := 0
		retryFunc := func(config DirectConfig) error {
			callCount++
			return nil
		}

		policy := RetryPolicy{
			MaxRetries:       0,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   time.Second,
			Jitter:           false,
		}
		strategy := NewDirectRetryStrategy(policy, retryFunc)

		config := DirectConfig{
			Endpoint:   "http://localhost:8080",
			Timeout:    30 * time.Second,
			BufferSize: 1024,
		}

		err := strategy.ExecuteTransport(string(TransportDirect), config)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if callCount != 1 {
			t.Errorf("expected retry function to be called once, got %d", callCount)
		}
	})

	t.Run("execute_with_retry", func(t *testing.T) {
		callCount := 0
		retryFunc := func(config DirectConfig) error {
			callCount++
			if callCount < 2 {
				return errors.New("connection timeout")
			}
			return nil
		}

		policy := RetryPolicy{
			MaxRetries:       2,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   time.Second,
			Jitter:           false,
		}
		strategy := NewDirectRetryStrategy(policy, retryFunc)

		config := DirectConfig{Endpoint: "http://localhost:8080"}
		err := strategy.ExecuteTransport(string(TransportDirect), config)

		if err != nil {
			t.Errorf("expected no error after retry, got %v", err)
		}

		if callCount != 2 {
			t.Errorf("expected 2 calls, got %d", callCount)
		}
	})

	t.Run("execute_exhausts_retries", func(t *testing.T) {
		callCount := 0
		retryFunc := func(config DirectConfig) error {
			callCount++
			return errors.New("persistent failure")
		}

		policy := RetryPolicy{
			MaxRetries:       1,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   time.Second,
			Jitter:           false,
		}
		strategy := NewDirectRetryStrategy(policy, retryFunc)

		config := DirectConfig{Endpoint: "http://localhost:8080"}
		err := strategy.ExecuteTransport(string(TransportDirect), config)

		if err == nil {
			t.Error("expected error after exhausting retries")
		}

		if callCount != 2 {
			t.Errorf("expected 2 calls (initial + 1 retry), got %d", callCount)
		}
	})

	t.Run("execute_wrong_config_type", func(t *testing.T) {
		strategy := NewDirectRetryStrategy(DefaultRetryPolicy(), nil)
		config := MQTTConfig{BrokerURL: "tcp://localhost:1883"}

		err := strategy.ExecuteTransport(string(TransportDirect), config)
		if err == nil {
			t.Error("expected error for wrong config type")
		}
	})

	t.Run("execute_no_retry_func", func(t *testing.T) {
		strategy := NewDirectRetryStrategy(DefaultRetryPolicy(), nil)
		config := DirectConfig{Endpoint: "http://localhost:8080"}

		err := strategy.ExecuteTransport(string(TransportDirect), config)
		if err == nil {
			t.Error("expected error when retry function is nil")
		}
	})

	t.Run("strategy_name_and_type", func(t *testing.T) {
		strategy := NewDirectRetryStrategy(DefaultRetryPolicy(), nil)

		if strategy.Name() != "direct_retry" {
			t.Errorf("expected name 'direct_retry', got %s", strategy.Name())
		}

		if strategy.GetTransportType() != string(TransportDirect) {
			t.Errorf("expected transport type 'direct', got %s", strategy.GetTransportType())
		}
	})

	t.Run("set_retry_func", func(t *testing.T) {
		strategy := NewDirectRetryStrategy(DefaultRetryPolicy(), nil)

		called := false
		strategy.SetRetryFunc(func(config DirectConfig) error {
			called = true
			return nil
		})

		config := DirectConfig{Endpoint: "http://localhost:8080"}
		strategy.ExecuteTransport(string(TransportDirect), config)

		if !called {
			t.Error("expected SetRetryFunc to update the function")
		}
	})

	t.Run("cooldown_from_policy", func(t *testing.T) {
		policy := RetryPolicy{CooldownWindow: 15 * time.Second}
		strategy := NewDirectRetryStrategy(policy, nil)

		if strategy.GetCooldown() != 15*time.Second {
			t.Errorf("expected cooldown 15s, got %v", strategy.GetCooldown())
		}
	})
}

// TestDirectCircuitResetStrategy tests circuit breaker reset for direct transport
func TestDirectCircuitResetStrategy(t *testing.T) {
	t.Run("can_heal_failing_direct", func(t *testing.T) {
		strategy := NewDirectCircuitResetStrategy(nil)

		if !strategy.CanHeal(string(TransportDirect), HealthFailing) {
			t.Error("expected strategy to heal failing direct transport")
		}

		if strategy.CanHeal(string(TransportDirect), HealthDegraded) {
			t.Error("expected strategy to NOT heal degraded transport (only failing)")
		}

		if strategy.CanHeal("mqtt", HealthFailing) {
			t.Error("expected strategy to NOT heal non-direct transport")
		}
	})

	t.Run("execute_circuit_reset", func(t *testing.T) {
		called := false
		resetFunc := func(config DirectConfig) error {
			called = true
			return nil
		}

		strategy := NewDirectCircuitResetStrategy(resetFunc)

		config := DirectConfig{Endpoint: "http://localhost:8080"}
		err := strategy.ExecuteTransport(string(TransportDirect), config)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if !called {
			t.Error("expected reset function to be called")
		}
	})

	t.Run("circuit_breaker_reset_integration", func(t *testing.T) {
		thc := NewTransportHealingController()

		// Open the circuit first
		for i := 0; i < 6; i++ {
			thc.updateCircuit(string(TransportDirect), false)
		}

		if !thc.IsCircuitOpen(string(TransportDirect)) {
			t.Fatal("expected circuit to be open")
		}

		// Put transport in failing state (need 2 failures)
		thc.RecordTransportFailure(string(TransportDirect), errors.New("failure 1"))
		thc.RecordTransportFailure(string(TransportDirect), errors.New("failure 2"))

		resetCalled := false
		resetFunc := func(config DirectConfig) error {
			resetCalled = true
			return nil
		}

		strategy := NewDirectCircuitResetStrategy(resetFunc)
		thc.RegisterTransportStrategy(string(TransportDirect), strategy)

		config := DirectConfig{Endpoint: "http://localhost:8080"}
		thc.AttemptTransportHeal(string(TransportDirect), config)

		if !resetCalled {
			t.Error("expected reset function to be called")
		}
	})

	t.Run("subsequent_behavior_after_reset", func(t *testing.T) {
		thc := NewTransportHealingController()

		// Open circuit
		for i := 0; i < 6; i++ {
			thc.updateCircuit(string(TransportDirect), false)
		}

		resetFunc := func(config DirectConfig) error {
			return nil
		}

		strategy := NewDirectCircuitResetStrategy(resetFunc)
		thc.RegisterTransportStrategy(string(TransportDirect), strategy)

		// Record health as failing so healing is attempted
		thc.RecordTransportFailure(string(TransportDirect), errors.New("test failure"))

		config := DirectConfig{Endpoint: "http://localhost:8080"}
		action := thc.AttemptTransportHeal(string(TransportDirect), config)

		if !action.Success {
			t.Errorf("expected healing to succeed, got: %s", action.ErrorMessage)
		}
	})

	t.Run("execute_wrong_config_type", func(t *testing.T) {
		strategy := NewDirectCircuitResetStrategy(nil)
		config := MQTTConfig{BrokerURL: "tcp://localhost:1883"}

		err := strategy.ExecuteTransport(string(TransportDirect), config)
		if err == nil {
			t.Error("expected error for wrong config type")
		}
	})

	t.Run("execute_no_reset_func", func(t *testing.T) {
		strategy := NewDirectCircuitResetStrategy(nil)
		config := DirectConfig{Endpoint: "http://localhost:8080"}

		err := strategy.ExecuteTransport(string(TransportDirect), config)
		if err == nil {
			t.Error("expected error when reset function is nil")
		}
	})

	t.Run("strategy_name_and_type", func(t *testing.T) {
		strategy := NewDirectCircuitResetStrategy(nil)

		if strategy.Name() != "direct_circuit_reset" {
			t.Errorf("expected name 'direct_circuit_reset', got %s", strategy.Name())
		}

		if strategy.GetTransportType() != string(TransportDirect) {
			t.Errorf("expected transport type 'direct', got %s", strategy.GetTransportType())
		}
	})

	t.Run("cooldown_duration", func(t *testing.T) {
		strategy := NewDirectCircuitResetStrategy(nil)

		if strategy.GetCooldown() != 10*time.Second {
			t.Errorf("expected cooldown 10s, got %v", strategy.GetCooldown())
		}
	})

	t.Run("set_reset_func", func(t *testing.T) {
		strategy := NewDirectCircuitResetStrategy(nil)

		called := false
		strategy.SetResetFunc(func(config DirectConfig) error {
			called = true
			return nil
		})

		config := DirectConfig{Endpoint: "http://localhost:8080"}
		strategy.ExecuteTransport(string(TransportDirect), config)

		if !called {
			t.Error("expected SetResetFunc to update the function")
		}
	})
}

// TestDirectBufferFlushStrategy tests buffer flush for direct transport
func TestDirectBufferFlushStrategy(t *testing.T) {
	t.Run("can_heal_degraded_or_failing", func(t *testing.T) {
		strategy := NewDirectBufferFlushStrategy(nil)

		if !strategy.CanHeal(string(TransportDirect), HealthDegraded) {
			t.Error("expected strategy to heal degraded direct transport")
		}

		if !strategy.CanHeal(string(TransportDirect), HealthFailing) {
			t.Error("expected strategy to heal failing direct transport")
		}

		if strategy.CanHeal(string(TransportDirect), HealthHealthy) {
			t.Error("expected strategy to NOT heal healthy transport")
		}

		if strategy.CanHeal("mqtt", HealthFailing) {
			t.Error("expected strategy to NOT heal non-direct transport")
		}
	})

	t.Run("execute_buffer_flush", func(t *testing.T) {
		called := false
		flushFunc := func(config DirectConfig) error {
			called = true
			return nil
		}

		strategy := NewDirectBufferFlushStrategy(flushFunc)

		config := DirectConfig{
			Endpoint:   "http://localhost:8080",
			BufferSize: 1024,
		}
		err := strategy.ExecuteTransport(string(TransportDirect), config)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if !called {
			t.Error("expected flush function to be called")
		}
	})

	t.Run("state_reset_after_flush", func(t *testing.T) {
		thc := NewTransportHealingController()

		// Degrade the transport
		thc.RecordTransportFailure(string(TransportDirect), errors.New("buffer overflow"))
		health := thc.GetTransportHealth(string(TransportDirect))
		if health.State != HealthDegraded {
			t.Fatalf("expected degraded state, got %s", health.State)
		}

		flushFunc := func(config DirectConfig) error {
			return nil
		}

		strategy := NewDirectBufferFlushStrategy(flushFunc)
		thc.RegisterTransportStrategy(string(TransportDirect), strategy)

		config := DirectConfig{Endpoint: "http://localhost:8080"}
		action := thc.AttemptTransportHeal(string(TransportDirect), config)

		if !action.Success {
			t.Errorf("expected healing to succeed, got: %s", action.ErrorMessage)
		}
	})

	t.Run("execute_wrong_config_type", func(t *testing.T) {
		strategy := NewDirectBufferFlushStrategy(nil)
		config := MQTTConfig{BrokerURL: "tcp://localhost:1883"}

		err := strategy.ExecuteTransport(string(TransportDirect), config)
		if err == nil {
			t.Error("expected error for wrong config type")
		}
	})

	t.Run("execute_no_flush_func", func(t *testing.T) {
		strategy := NewDirectBufferFlushStrategy(nil)
		config := DirectConfig{Endpoint: "http://localhost:8080"}

		err := strategy.ExecuteTransport(string(TransportDirect), config)
		if err == nil {
			t.Error("expected error when flush function is nil")
		}
	})

	t.Run("strategy_name_and_type", func(t *testing.T) {
		strategy := NewDirectBufferFlushStrategy(nil)

		if strategy.Name() != "direct_buffer_flush" {
			t.Errorf("expected name 'direct_buffer_flush', got %s", strategy.Name())
		}

		if strategy.GetTransportType() != string(TransportDirect) {
			t.Errorf("expected transport type 'direct', got %s", strategy.GetTransportType())
		}
	})

	t.Run("cooldown_duration", func(t *testing.T) {
		strategy := NewDirectBufferFlushStrategy(nil)

		if strategy.GetCooldown() != 5*time.Second {
			t.Errorf("expected cooldown 5s, got %v", strategy.GetCooldown())
		}
	})

	t.Run("set_flush_func", func(t *testing.T) {
		strategy := NewDirectBufferFlushStrategy(nil)

		called := false
		strategy.SetFlushFunc(func(config DirectConfig) error {
			called = true
			return nil
		})

		config := DirectConfig{Endpoint: "http://localhost:8080"}
		strategy.ExecuteTransport(string(TransportDirect), config)

		if !called {
			t.Error("expected SetFlushFunc to update the function")
		}
	})
}

// TestTransportHealthTracking tests health state tracking
func TestTransportHealthTracking(t *testing.T) {
	t.Run("record_transport_failure", func(t *testing.T) {
		thc := NewTransportHealingController()

		testErr := errors.New("connection timeout")
		thc.RecordTransportFailure(string(TransportMQTT), testErr)

		health := thc.GetTransportHealth(string(TransportMQTT))

		if health.State != HealthDegraded {
			t.Errorf("expected state degraded after first failure, got %s", health.State)
		}

		if health.LastFailure != testErr {
			t.Error("expected LastFailure to be recorded")
		}

		// Second failure should transition to failing
		thc.RecordTransportFailure(string(TransportMQTT), errors.New("another error"))
		health = thc.GetTransportHealth(string(TransportMQTT))

		if health.State != HealthFailing {
			t.Errorf("expected state failing after second failure, got %s", health.State)
		}
	})

	t.Run("record_transport_success", func(t *testing.T) {
		thc := NewTransportHealingController()

		// First degrade the transport
		thc.RecordTransportFailure(string(TransportMQTT), errors.New("failure"))
		thc.RecordTransportFailure(string(TransportMQTT), errors.New("failure"))

		health := thc.GetTransportHealth(string(TransportMQTT))
		if health.State != HealthFailing {
			t.Fatal("expected failing state")
		}

		// Now record success
		thc.RecordTransportSuccess(string(TransportMQTT))

		health = thc.GetTransportHealth(string(TransportMQTT))
		if health.State != HealthDegraded {
			t.Errorf("expected state degraded after success from failing, got %s", health.State)
		}

		if health.LastFailure != nil {
			t.Error("expected LastFailure to be cleared")
		}

		// Another success should make it healthy
		thc.RecordTransportSuccess(string(TransportMQTT))
		health = thc.GetTransportHealth(string(TransportMQTT))

		if health.State != HealthHealthy {
			t.Errorf("expected state healthy after second success, got %s", health.State)
		}
	})

	t.Run("get_transport_health_unknown", func(t *testing.T) {
		thc := NewTransportHealingController()

		health := thc.GetTransportHealth("unknown-transport")

		if health.TransportType != "unknown-transport" {
			t.Errorf("expected transport type to be set, got %s", health.TransportType)
		}

		if health.State != HealthUnknown {
			t.Errorf("expected state unknown for new transport, got %s", health.State)
		}
	})

	t.Run("health_state_transitions", func(t *testing.T) {
		thc := NewTransportHealingController()

		// Start unknown -> degraded
		thc.RecordTransportFailure(string(TransportDirect), errors.New("err1"))
		health := thc.GetTransportHealth(string(TransportDirect))
		if health.State != HealthDegraded {
			t.Errorf("expected degraded, got %s", health.State)
		}

		// Degraded -> failing
		thc.RecordTransportFailure(string(TransportDirect), errors.New("err2"))
		health = thc.GetTransportHealth(string(TransportDirect))
		if health.State != HealthFailing {
			t.Errorf("expected failing, got %s", health.State)
		}

		// Failing -> degraded
		thc.RecordTransportSuccess(string(TransportDirect))
		health = thc.GetTransportHealth(string(TransportDirect))
		if health.State != HealthDegraded {
			t.Errorf("expected degraded, got %s", health.State)
		}

		// Degraded -> healthy
		thc.RecordTransportSuccess(string(TransportDirect))
		health = thc.GetTransportHealth(string(TransportDirect))
		if health.State != HealthHealthy {
			t.Errorf("expected healthy, got %s", health.State)
		}
	})

	t.Run("concurrent_health_tracking", func(t *testing.T) {
		thc := NewTransportHealingController()

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(success bool) {
				defer wg.Done()
				if success {
					thc.RecordTransportSuccess(string(TransportMQTT))
				} else {
					thc.RecordTransportFailure(string(TransportMQTT), errors.New("concurrent error"))
				}
			}(i%2 == 0)
		}

		wg.Wait()

		// Should not panic or deadlock
		health := thc.GetTransportHealth(string(TransportMQTT))
		if health.TransportType != string(TransportMQTT) {
			t.Error("expected transport type to be recorded")
		}
	})
}

// TestFailoverCooldown tests cooldown prevents rapid failover
func TestFailoverCooldown(t *testing.T) {
	t.Run("cooldown_prevents_rapid_failover", func(t *testing.T) {
		thc := NewTransportHealingController()
		thc.SetFailoverCooldown(100 * time.Millisecond)

		callCount := 0
		reconnectFunc := func(config MQTTConfig) error {
			callCount++
			return nil
		}

		policy := RetryPolicy{
			MaxRetries:       0,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   1 * time.Millisecond,
			Jitter:           false,
		}
		strategy := NewMQTTRetryStrategy(policy, reconnectFunc)

		thc.RegisterTransportStrategy(string(TransportMQTT), strategy)

		// Put transport in degraded state
		thc.RecordTransportFailure(string(TransportMQTT), errors.New("failure"))

		config := MQTTConfig{BrokerURL: "tcp://localhost:1883"}

		// First attempt
		thc.AttemptTransportHeal(string(TransportMQTT), config)

		// Multiple attempts during cooldown
		for i := 0; i < 5; i++ {
			action := thc.AttemptTransportHeal(string(TransportMQTT), config)
			if action.Success && strings.Contains(action.ErrorMessage, "cooldown") {
				// Expected
			}
		}

		if callCount != 1 {
			t.Errorf("expected 1 call during cooldown, got %d", callCount)
		}
	})

	t.Run("cooldown_expiration", func(t *testing.T) {
		thc := NewTransportHealingController()
		thc.SetFailoverCooldown(50 * time.Millisecond)

		callCount := 0
		reconnectFunc := func(config MQTTConfig) error {
			callCount++
			return nil
		}

		policy := RetryPolicy{
			MaxRetries:       0,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   1 * time.Millisecond,
			Jitter:           false,
		}
		strategy := NewMQTTRetryStrategy(policy, reconnectFunc)

		thc.RegisterTransportStrategy(string(TransportMQTT), strategy)

		config := MQTTConfig{BrokerURL: "tcp://localhost:1883"}

		// First attempt (put in degraded state)
		thc.RecordTransportFailure(string(TransportMQTT), errors.New("failure"))
		thc.AttemptTransportHeal(string(TransportMQTT), config)

		// Wait for cooldown to expire
		time.Sleep(60 * time.Millisecond)

		// Second attempt should succeed (put back in degraded state)
		thc.RecordTransportFailure(string(TransportMQTT), errors.New("failure 2"))
		thc.AttemptTransportHeal(string(TransportMQTT), config)

		if callCount != 2 {
			t.Errorf("expected 2 calls after cooldown expiration, got %d", callCount)
		}
	})

	t.Run("set_failover_cooldown", func(t *testing.T) {
		thc := NewTransportHealingController()

		thc.SetFailoverCooldown(10 * time.Minute)

		if thc.failoverCooldown != 10*time.Minute {
			t.Errorf("expected failover cooldown 10m, got %v", thc.failoverCooldown)
		}
	})

	t.Run("set_max_failovers_per_hour", func(t *testing.T) {
		thc := NewTransportHealingController()

		thc.SetMaxFailoversPerHour(10)

		if thc.maxFailoversPerHour != 10 {
			t.Errorf("expected max failovers 10, got %d", thc.maxFailoversPerHour)
		}
	})
}

// TestTransportHealingIntegration tests full healing flow
func TestTransportHealingIntegration(t *testing.T) {
	t.Run("full_mqtt_healing_flow", func(t *testing.T) {
		thc := NewTransportHealingController()

		reconnectCount := 0
		reconnectFunc := func(config MQTTConfig) error {
			reconnectCount++
			return nil
		}

		policy := RetryPolicy{
			MaxRetries:       0,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   1 * time.Millisecond,
			Jitter:           false,
		}
		strategy := NewMQTTRetryStrategy(policy, reconnectFunc)

		thc.RegisterTransportStrategy(string(TransportMQTT), strategy)

		// Simulate failure
		thc.RecordTransportFailure(string(TransportMQTT), errors.New("connection lost"))

		config := MQTTConfig{
			BrokerURL: "tcp://localhost:1883",
			ClientID:  "test-client",
		}

		// Attempt healing
		action := thc.AttemptTransportHeal(string(TransportMQTT), config)

		if !action.Success {
			t.Errorf("expected healing to succeed, got: %s", action.ErrorMessage)
		}

		if reconnectCount != 1 {
			t.Errorf("expected 1 reconnect, got %d", reconnectCount)
		}

		// Verify health was updated
		health := thc.GetTransportHealth(string(TransportMQTT))
		if health.State != HealthHealthy {
			t.Errorf("expected healthy state after healing, got %s", health.State)
		}
	})

	t.Run("full_direct_healing_flow", func(t *testing.T) {
		thc := NewTransportHealingController()

		flushCount := 0
		flushFunc := func(config DirectConfig) error {
			flushCount++
			return nil
		}

		strategy := NewDirectBufferFlushStrategy(flushFunc)
		thc.RegisterTransportStrategy(string(TransportDirect), strategy)

		// Simulate degradation
		thc.RecordTransportFailure(string(TransportDirect), errors.New("buffer full"))

		config := DirectConfig{
			Endpoint:   "http://localhost:8080",
			BufferSize: 1024,
		}

		// Attempt healing
		action := thc.AttemptTransportHeal(string(TransportDirect), config)

		if !action.Success {
			t.Errorf("expected healing to succeed, got: %s", action.ErrorMessage)
		}

		if flushCount != 1 {
			t.Errorf("expected 1 flush, got %d", flushCount)
		}
	})

	t.Run("audit_trail_integration", func(t *testing.T) {
		thc := NewTransportHealingController()

		reconnectFunc := func(config MQTTConfig) error {
			return nil
		}

		strategy := NewMQTTRetryStrategy(DefaultRetryPolicy(), reconnectFunc)
		thc.RegisterTransportStrategy(string(TransportMQTT), strategy)

		config := MQTTConfig{BrokerURL: "tcp://localhost:1883"}

		// Perform healing multiple times
		thc.AttemptTransportHeal(string(TransportMQTT), config)
		thc.AttemptTransportHeal(string(TransportMQTT), config)

		// Check audit trail
		trail := thc.GetAuditTrail(string(TransportMQTT))

		if len(trail) < 2 {
			t.Errorf("expected at least 2 audit entries, got %d", len(trail))
		}

		for _, action := range trail {
			if action.Component != string(TransportMQTT) {
				t.Errorf("expected component to be mqtt, got %s", action.Component)
			}
		}
	})

	t.Run("audit_trail_records_failures", func(t *testing.T) {
		thc := NewTransportHealingController()

		reconnectFunc := func(config MQTTConfig) error {
			return errors.New("connection refused")
		}

		policy := RetryPolicy{
			MaxRetries:       0,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   1 * time.Millisecond,
			Jitter:           false,
		}
		strategy := NewMQTTRetryStrategy(policy, reconnectFunc)
		thc.RegisterTransportStrategy(string(TransportMQTT), strategy)

		// Put transport in degraded state
		thc.RecordTransportFailure(string(TransportMQTT), errors.New("failure"))

		config := MQTTConfig{BrokerURL: "tcp://localhost:1883"}

		action := thc.AttemptTransportHeal(string(TransportMQTT), config)

		if action.Success {
			t.Error("expected healing to fail")
		}

		if action.ErrorMessage == "" {
			t.Error("expected error message in audit trail")
		}

		trail := thc.GetAuditTrail(string(TransportMQTT))
		if len(trail) == 0 {
			t.Fatal("expected audit trail entries")
		}

		if trail[len(trail)-1].Success {
			t.Error("expected last action to be recorded as failure")
		}
	})

	t.Run("latency_tracking", func(t *testing.T) {
		thc := NewTransportHealingController()

		reconnectFunc := func(config MQTTConfig) error {
			time.Sleep(5 * time.Millisecond)
			return nil
		}

		strategy := NewMQTTRetryStrategy(DefaultRetryPolicy(), reconnectFunc)
		thc.RegisterTransportStrategy(string(TransportMQTT), strategy)

		// Put transport in degraded state
		thc.RecordTransportFailure(string(TransportMQTT), errors.New("failure"))

		config := MQTTConfig{BrokerURL: "tcp://localhost:1883"}

		// Perform healing
		thc.AttemptTransportHeal(string(TransportMQTT), config)

		avgLatency := thc.GetAverageLatency(string(TransportMQTT))
		if avgLatency == 0 {
			t.Error("expected latency to be tracked")
		}

		health := thc.GetTransportHealth(string(TransportMQTT))
		if health.Latency == 0 {
			t.Error("expected latency in health state")
		}
	})

	t.Run("concurrent_transport_healing", func(t *testing.T) {
		thc := NewTransportHealingController()

		var mu sync.Mutex
		callCount := 0
		reconnectFunc := func(config MQTTConfig) error {
			mu.Lock()
			callCount++
			mu.Unlock()
			return nil
		}

		policy := RetryPolicy{
			MaxRetries:       0,
			BackoffIntervals: []time.Duration{1 * time.Millisecond},
			CooldownWindow:   1 * time.Millisecond,
			Jitter:           false,
		}
		strategy := NewMQTTRetryStrategy(policy, reconnectFunc)
		thc.RegisterTransportStrategy(string(TransportMQTT), strategy)

		config := MQTTConfig{BrokerURL: "tcp://localhost:1883"}

		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				thc.AttemptTransportHeal(string(TransportMQTT), config)
			}()
		}

		wg.Wait()

		mu.Lock()
		if callCount > 20 {
			t.Errorf("expected at most 20 calls, got %d", callCount)
		}
		mu.Unlock()
	})

	t.Run("no_strategy_registered", func(t *testing.T) {
		thc := NewTransportHealingController()

		config := MQTTConfig{BrokerURL: "tcp://localhost:1883"}
		action := thc.AttemptTransportHeal(string(TransportMQTT), config)

		if action.Success {
			t.Error("expected healing to fail without strategy")
		}

		if !strings.Contains(action.ErrorMessage, "no transport healing strategy registered") {
			t.Errorf("expected 'no strategy registered' error, got: %s", action.ErrorMessage)
		}
	})

	t.Run("transport_does_not_require_healing", func(t *testing.T) {
		thc := NewTransportHealingController()

		reconnectFunc := func(config MQTTConfig) error {
			return nil
		}

		strategy := NewMQTTRetryStrategy(DefaultRetryPolicy(), reconnectFunc)
		thc.RegisterTransportStrategy(string(TransportMQTT), strategy)

		// Set health to healthy (no healing needed)
		thc.RecordTransportSuccess(string(TransportMQTT))
		thc.RecordTransportSuccess(string(TransportMQTT))

		config := MQTTConfig{BrokerURL: "tcp://localhost:1883"}
		action := thc.AttemptTransportHeal(string(TransportMQTT), config)

		if !action.Success {
			t.Error("expected action to succeed when no healing needed")
		}

		if !strings.Contains(action.ErrorMessage, "does not require healing") {
			t.Errorf("expected 'does not require healing' message, got: %s", action.ErrorMessage)
		}
	})
}

// TestTransportHealingStrategyWrapper tests the wrapper
func TestTransportHealingStrategyWrapper(t *testing.T) {
	t.Run("wrapper_delegates_to_strategy", func(t *testing.T) {
		baseStrategy := NewRetryStrategy(DefaultRetryPolicy(), func(component string) error {
			return nil
		})

		wrapper := &TransportHealingStrategy{
			strategy:      baseStrategy,
			transportType: TransportMQTT,
		}

		if wrapper.Name() != "retry" {
			t.Errorf("expected name 'retry', got %s", wrapper.Name())
		}

		if wrapper.GetTransportType() != string(TransportMQTT) {
			t.Errorf("expected transport type 'mqtt', got %s", wrapper.GetTransportType())
		}

		cooldown := wrapper.GetCooldown()
		if cooldown != DefaultCooldownWindow {
			t.Errorf("expected cooldown %v, got %v", DefaultCooldownWindow, cooldown)
		}
	})

	t.Run("wrapper_with_custom_functions", func(t *testing.T) {
		baseStrategy := NewRetryStrategy(DefaultRetryPolicy(), nil)

		canHealCalled := false
		executeCalled := false

		wrapper := &TransportHealingStrategy{
			strategy:    baseStrategy,
			canHealFunc: func(transportType string, state ComponentHealth) bool { canHealCalled = true; return true },
			executeFunc: func(transportType string, config TransportConfig) error { executeCalled = true; return nil },
		}

		wrapper.CanHeal("mqtt", HealthFailing)
		if !canHealCalled {
			t.Error("expected custom CanHeal to be called")
		}

		wrapper.ExecuteTransport("mqtt", MQTTConfig{})
		if !executeCalled {
			t.Error("expected custom Execute to be called")
		}
	})
}

// TestTransportConfigTypes tests config type methods
func TestTransportConfigTypes(t *testing.T) {
	t.Run("mqtt_config_type", func(t *testing.T) {
		config := MQTTConfig{}
		if config.GetTransportType() != TransportMQTT {
			t.Errorf("expected transport type MQTT, got %s", config.GetTransportType())
		}
	})

	t.Run("direct_config_type", func(t *testing.T) {
		config := DirectConfig{}
		if config.GetTransportType() != TransportDirect {
			t.Errorf("expected transport type Direct, got %s", config.GetTransportType())
		}
	})
}

// TestTransportActionTypes tests action type constants
func TestTransportActionTypes(t *testing.T) {
	cases := []struct {
		action   TransportActionType
		expected string
	}{
		{TransportActionReconnect, "reconnect"},
		{TransportActionFailover, "failover"},
		{TransportActionQoSAdjust, "qos_adjust"},
		{TransportActionBufferReset, "buffer_reset"},
		{TransportActionCircuitReset, "circuit_reset"},
	}

	for _, tc := range cases {
		t.Run(string(tc.action), func(t *testing.T) {
			if string(tc.action) != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, tc.action)
			}
		})
	}
}

// TestTransportTypeConstants tests transport type constants
func TestTransportTypeConstants(t *testing.T) {
	if TransportMQTT != "mqtt" {
		t.Errorf("expected TransportMQTT to be 'mqtt', got %s", TransportMQTT)
	}

	if TransportDirect != "direct" {
		t.Errorf("expected TransportDirect to be 'direct', got %s", TransportDirect)
	}
}

// TestAverageLatency tests average latency calculation
func TestAverageLatency(t *testing.T) {
	t.Run("average_latency_calculation", func(t *testing.T) {
		thc := NewTransportHealingController()

		// Record several latencies
		thc.recordLatency(string(TransportMQTT), 10*time.Millisecond)
		thc.recordLatency(string(TransportMQTT), 20*time.Millisecond)
		thc.recordLatency(string(TransportMQTT), 30*time.Millisecond)

		avg := thc.GetAverageLatency(string(TransportMQTT))
		expected := 20 * time.Millisecond

		if avg != expected {
			t.Errorf("expected average latency %v, got %v", expected, avg)
		}
	})

	t.Run("average_latency_empty", func(t *testing.T) {
		thc := NewTransportHealingController()

		avg := thc.GetAverageLatency(string(TransportMQTT))
		if avg != 0 {
			t.Errorf("expected 0 for unknown transport, got %v", avg)
		}
	})

	t.Run("average_latency_max_100", func(t *testing.T) {
		thc := NewTransportHealingController()

		// Record more than 100 latencies
		for i := 0; i < 150; i++ {
			thc.recordLatency(string(TransportMQTT), time.Duration(i)*time.Millisecond)
		}

		latencies := thc.latencyTracking[string(TransportMQTT)]
		if len(latencies) != 100 {
			t.Errorf("expected max 100 latencies, got %d", len(latencies))
		}
	})
}

// TestGracefulDegrade tests graceful degradation
func TestGracefulDegrade(t *testing.T) {
	t.Run("mqtt_graceful_degrade", func(t *testing.T) {
		thc := NewTransportHealingController()

		config := MQTTConfig{
			BrokerURL: "tcp://localhost:1883",
			QoS:       2,
		}

		// Note: This will fail since no adjustFunc is set, but tests the path
		err := thc.GracefulDegrade(string(TransportMQTT), config)
		// Error expected since no adjust function is configured
		if err == nil {
			t.Error("expected error when no adjust function configured")
		}
	})

	t.Run("direct_graceful_degrade", func(t *testing.T) {
		thc := NewTransportHealingController()

		config := DirectConfig{
			Endpoint:   "http://localhost:8080",
			BufferSize: 2048,
		}

		// Note: This will fail since no flushFunc is set, but tests the path
		err := thc.GracefulDegrade(string(TransportDirect), config)
		// Error expected since no flush function is configured
		if err == nil {
			t.Error("expected error when no flush function configured")
		}
	})

	t.Run("mqtt_qos_decrements", func(t *testing.T) {
		thc := NewTransportHealingController()

		// Use QoS 0 which can't be decremented
		config := MQTTConfig{
			BrokerURL: "tcp://localhost:1883",
			QoS:       0,
		}

		// Should return nil since QoS is already at minimum
		err := thc.GracefulDegrade(string(TransportMQTT), config)
		if err != nil {
			t.Errorf("expected no error for QoS 0, got %v", err)
		}
	})

	t.Run("direct_buffer_small", func(t *testing.T) {
		thc := NewTransportHealingController()

		// Use small buffer
		config := DirectConfig{
			Endpoint:   "http://localhost:8080",
			BufferSize: 512,
		}

		// Should return nil since buffer is already small
		err := thc.GracefulDegrade(string(TransportDirect), config)
		if err != nil {
			t.Errorf("expected no error for small buffer, got %v", err)
		}
	})
}

// TestCanFailover tests the canFailover method
func TestCanFailover(t *testing.T) {
	t.Run("can_failover_initial", func(t *testing.T) {
		thc := NewTransportHealingController()

		if !thc.canFailover(string(TransportMQTT)) {
			t.Error("expected canFailover to return true for new transport")
		}
	})

	t.Run("failover_window_resets", func(t *testing.T) {
		thc := NewTransportHealingController()
		thc.SetMaxFailoversPerHour(2)

		// Simulate failover
		thc.incrementFailoverCount(string(TransportMQTT))
		thc.incrementFailoverCount(string(TransportMQTT))

		if thc.canFailover(string(TransportMQTT)) {
			t.Error("expected canFailover to return false after max reached")
		}
	})
}

// TestDetermineActionType tests the determineActionType method
func TestDetermineActionType(t *testing.T) {
	thc := NewTransportHealingController()

	cases := []struct {
		strategyName string
		expected     TransportActionType
	}{
		{"mqtt_retry", TransportActionReconnect},
		{"direct_retry", TransportActionReconnect},
		{"mqtt_broker_failover", TransportActionFailover},
		{"mqtt_qos_adjust", TransportActionQoSAdjust},
		{"direct_circuit_reset", TransportActionCircuitReset},
		{"direct_buffer_flush", TransportActionBufferReset},
		{"unknown_strategy", TransportActionReconnect}, // default
	}

	for _, tc := range cases {
		t.Run(tc.strategyName, func(t *testing.T) {
			strategy := &mockTransportHealer{name: tc.strategyName}
			actionType := thc.determineActionType(strategy)
			if actionType != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, actionType)
			}
		})
	}
}

// mockTransportHealer is a mock implementation for testing
type mockTransportHealer struct {
	name string
}

func (m *mockTransportHealer) CanHeal(transportType string, state ComponentHealth) bool {
	return true
}

func (m *mockTransportHealer) Execute(component string) error {
	return nil
}

func (m *mockTransportHealer) ExecuteTransport(transportType string, config TransportConfig) error {
	return nil
}

func (m *mockTransportHealer) GetCooldown() time.Duration {
	return time.Second
}

func (m *mockTransportHealer) Name() string {
	return m.name
}

func (m *mockTransportHealer) GetTransportType() string {
	return "mock"
}

// TestTransportHealthStruct tests TransportHealth fields
func TestTransportHealthStruct(t *testing.T) {
	now := time.Now()
	health := TransportHealth{
		TransportType: string(TransportMQTT),
		State:         HealthDegraded,
		LastFailure:   errors.New("test error"),
		FailoverCount: 2,
		LastFailover:  now,
		Latency:       50 * time.Millisecond,
	}

	if health.TransportType != "mqtt" {
		t.Error("TransportType not set correctly")
	}
	if health.State != HealthDegraded {
		t.Error("State not set correctly")
	}
	if health.LastFailure == nil {
		t.Error("LastFailure not set correctly")
	}
	if health.FailoverCount != 2 {
		t.Error("FailoverCount not set correctly")
	}
	if health.LastFailover != now {
		t.Error("LastFailover not set correctly")
	}
	if health.Latency != 50*time.Millisecond {
		t.Error("Latency not set correctly")
	}
}

// TestRegisterTransportStrategy tests strategy registration
func TestRegisterTransportStrategy(t *testing.T) {
	thc := NewTransportHealingController()

	strategy := NewMQTTRetryStrategy(DefaultRetryPolicy(), nil)
	thc.RegisterTransportStrategy(string(TransportMQTT), strategy)

	thc.mu.RLock()
	registered, ok := thc.transportStrategies[string(TransportMQTT)]
	thc.mu.RUnlock()

	if !ok {
		t.Fatal("expected strategy to be registered")
	}

	if registered != strategy {
		t.Error("registered strategy does not match")
	}
}

// TestRecordTransportFailurePrivate tests the private method
func TestRecordTransportFailurePrivate(t *testing.T) {
	thc := NewTransportHealingController()

	// Test the private recordTransportFailure (called within AttemptTransportHeal on error)
	reconnectFunc := func(config MQTTConfig) error {
		return errors.New("always fails")
	}

	policy := RetryPolicy{
		MaxRetries:       0,
		BackoffIntervals: []time.Duration{1 * time.Millisecond},
		CooldownWindow:   1 * time.Millisecond,
		Jitter:           false,
	}
	strategy := NewMQTTRetryStrategy(policy, reconnectFunc)
	thc.RegisterTransportStrategy(string(TransportMQTT), strategy)

	// Put transport in degraded state so healing will be attempted
	thc.RecordTransportFailure(string(TransportMQTT), errors.New("initial failure"))

	config := MQTTConfig{BrokerURL: "tcp://localhost:1883"}
	thc.AttemptTransportHeal(string(TransportMQTT), config)

	health := thc.GetTransportHealth(string(TransportMQTT))
	if health.LastFailure == nil {
		t.Error("expected failure to be recorded")
	}
}

// TestRecordTransportSuccessPrivate tests the private method
func TestRecordTransportSuccessPrivate(t *testing.T) {
	thc := NewTransportHealingController()

	reconnectFunc := func(config MQTTConfig) error {
		return nil
	}

	policy := RetryPolicy{
		MaxRetries:       0,
		BackoffIntervals: []time.Duration{1 * time.Millisecond},
		CooldownWindow:   1 * time.Millisecond, // Short cooldown for testing
		Jitter:           false,
	}
	strategy := NewMQTTRetryStrategy(policy, reconnectFunc)
	thc.RegisterTransportStrategy(string(TransportMQTT), strategy)

	// First create a failure state
	thc.RecordTransportFailure(string(TransportMQTT), errors.New("test"))
	thc.RecordTransportFailure(string(TransportMQTT), errors.New("test"))

	health := thc.GetTransportHealth(string(TransportMQTT))
	if health.State != HealthFailing {
		t.Fatalf("expected failing state, got %s", health.State)
	}

	// Wait for cooldown to ensure healing executes
	time.Sleep(2 * time.Millisecond)

	// Now succeed
	config := MQTTConfig{BrokerURL: "tcp://localhost:1883"}
	thc.AttemptTransportHeal(string(TransportMQTT), config)

	health = thc.GetTransportHealth(string(TransportMQTT))
	// State transitions: Failing -> Degraded -> Healthy
	// After one success from Failing state, should be Degraded
	if health.State != HealthDegraded {
		t.Errorf("expected degraded state after success from failing, got %s", health.State)
	}
}

// TestIncrementFailoverCountPrivate tests failover counting
func TestIncrementFailoverCountPrivate(t *testing.T) {
	thc := NewTransportHealingController()

	// Create transport state first (so incrementFailoverCount can update it)
	thc.RecordTransportFailure(string(TransportMQTT), errors.New("initial"))

	// Increment multiple times
	thc.incrementFailoverCount(string(TransportMQTT))
	thc.incrementFailoverCount(string(TransportMQTT))
	thc.incrementFailoverCount(string(TransportMQTT))

	health := thc.GetTransportHealth(string(TransportMQTT))
	if health.FailoverCount != 3 {
		t.Errorf("expected 3 failovers, got %d", health.FailoverCount)
	}

	if health.LastFailover.IsZero() {
		t.Error("expected LastFailover to be set")
	}
}

// TestFailoverInfoReset tests failover window reset
func TestFailoverInfoReset(t *testing.T) {
	thc := NewTransportHealingController()

	// Create failover info with old window
	info := &transportFailoverInfo{
		failoverCount:  5,
		lastFailover:   time.Now().Add(-2 * time.Hour),
		failoverWindow: time.Now().Add(-2 * time.Hour),
	}

	thc.mu.Lock()
	thc.failoverInfo[string(TransportMQTT)] = info
	thc.mu.Unlock()

	// Check that canFailover returns true (window has passed)
	if !thc.canFailover(string(TransportMQTT)) {
		t.Error("expected canFailover to return true after window reset")
	}

	// Now increment and verify count reset
	thc.incrementFailoverCount(string(TransportMQTT))

	thc.mu.RLock()
	updatedInfo := thc.failoverInfo[string(TransportMQTT)]
	thc.mu.RUnlock()

	if updatedInfo.failoverCount != 1 {
		t.Errorf("expected count to reset to 1, got %d", updatedInfo.failoverCount)
	}
}
