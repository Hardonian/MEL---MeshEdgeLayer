// Package selfobs provides bounded self-healing capabilities for MEL components.
// The healing controller implements circuit breaker patterns, configurable retry policies,
// and comprehensive audit trails to ensure all healing actions are observable and bounded.
package selfobs

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ActionType represents the type of healing action performed
type ActionType string

const (
	// ActionRetry indicates a retry healing action
	ActionRetry ActionType = "retry"
	// ActionReconnect indicates a reconnect healing action
	ActionReconnect ActionType = "reconnect"
	// ActionReset indicates a reset healing action
	ActionReset ActionType = "reset"
	// ActionCooldown indicates the component is in cooldown
	ActionCooldown ActionType = "cooldown"
)

// CircuitState represents the state of the circuit breaker
type CircuitState string

const (
	// CircuitClosed indicates normal operation - healing allowed
	CircuitClosed CircuitState = "closed"
	// CircuitOpen indicates healing is blocked due to repeated failures
	CircuitOpen CircuitState = "open"
	// CircuitHalfOpen indicates testing if the component has recovered
	CircuitHalfOpen CircuitState = "half-open"
)

// Default values for retry policy
const (
	DefaultMaxRetries      = 3
	DefaultCooldownWindow  = 60 * time.Second
	DefaultCircuitFailures = 5
	DefaultCircuitTimeout  = 30 * time.Second
)

// RetryPolicy defines the configuration for retry behavior
type RetryPolicy struct {
	// MaxRetries is the maximum number of retry attempts (default: 3)
	MaxRetries int
	// BackoffIntervals defines the delay between retry attempts (default: 1s, 5s, 15s)
	BackoffIntervals []time.Duration
	// CooldownWindow is the minimum time between healing attempts (default: 60s)
	CooldownWindow time.Duration
	// Jitter adds randomization to prevent thundering herd (default: true)
	Jitter bool
}

// DefaultRetryPolicy returns a RetryPolicy with sensible defaults
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:       DefaultMaxRetries,
		BackoffIntervals: []time.Duration{1 * time.Second, 5 * time.Second, 15 * time.Second},
		CooldownWindow:   DefaultCooldownWindow,
		Jitter:           true,
	}
}

// getBackoff returns the backoff duration for a specific retry attempt
func (rp *RetryPolicy) getBackoff(attempt int) time.Duration {
	if attempt < 0 || attempt >= len(rp.BackoffIntervals) {
		attempt = len(rp.BackoffIntervals) - 1
		if attempt < 0 {
			return 1 * time.Second
		}
	}
	backoff := rp.BackoffIntervals[attempt]
	if rp.Jitter {
		// Add random jitter of up to 20% of the backoff duration
		jitter := time.Duration(float64(backoff) * (0.8 + 0.4*rand.Float64()))
		backoff = jitter
	}
	return backoff
}

// HealingAction represents a single healing action recorded in the audit trail
type HealingAction struct {
	// ID is the unique identifier for the action
	ID int64 `json:"id,omitempty"`
	// Timestamp is when the action occurred
	Timestamp time.Time `json:"timestamp"`
	// Component is the name of the component being healed
	Component string `json:"component"`
	// ActionType is the type of healing action performed
	ActionType ActionType `json:"action_type"`
	// Success indicates whether the healing action succeeded
	Success bool `json:"success"`
	// ErrorMessage contains the error details if the action failed
	ErrorMessage string `json:"error_message,omitempty"`
	// RetryCount indicates which retry attempt this was (0 = first attempt)
	RetryCount int `json:"retry_count"`
	// StrategyName is the name of the healing strategy used
	StrategyName string `json:"strategy_name"`
	// TransportType is the type of transport for transport-specific actions
	TransportType string `json:"transport_type,omitempty"`
}

// HealingStrategy defines the interface for component-specific healing logic
type HealingStrategy interface {
	// CanHeal returns true if the strategy can heal the given component
	CanHeal(component string, health ComponentHealth) bool
	// Execute performs the healing action and returns any error
	Execute(component string) error
	// GetCooldown returns the cooldown period after this strategy executes
	GetCooldown() time.Duration
	// Name returns the strategy identifier for audit purposes
	Name() string
}

// RetryStrategy implements a simple retry with backoff
type RetryStrategy struct {
	policy     RetryPolicy
	retryFunc  func(component string) error
	strategyMu sync.RWMutex
}

// NewRetryStrategy creates a new retry strategy with the given function
func NewRetryStrategy(policy RetryPolicy, retryFunc func(component string) error) *RetryStrategy {
	return &RetryStrategy{
		policy:    policy,
		retryFunc: retryFunc,
	}
}

// CanHeal returns true for degraded or failing components
func (rs *RetryStrategy) CanHeal(component string, health ComponentHealth) bool {
	return health == HealthDegraded || health == HealthFailing
}

// Execute performs the retry operation
func (rs *RetryStrategy) Execute(component string) error {
	rs.strategyMu.RLock()
	fn := rs.retryFunc
	rs.strategyMu.RUnlock()
	
	if fn == nil {
		return fmt.Errorf("no retry function configured for component %s", component)
	}
	return fn(component)
}

// GetCooldown returns the policy's cooldown window
func (rs *RetryStrategy) GetCooldown() time.Duration {
	rs.strategyMu.RLock()
	defer rs.strategyMu.RUnlock()
	return rs.policy.CooldownWindow
}

// Name returns the strategy identifier
func (rs *RetryStrategy) Name() string {
	return "retry"
}

// SetRetryFunc allows updating the retry function at runtime
func (rs *RetryStrategy) SetRetryFunc(fn func(component string) error) {
	rs.strategyMu.Lock()
	defer rs.strategyMu.Unlock()
	rs.retryFunc = fn
}

// ReconnectStrategy implements healing by reconnecting connection-based components
type ReconnectStrategy struct {
	connectFunc    func(component string) error
	disconnectFunc func(component string) error
	cooldown       time.Duration
	strategyMu     sync.RWMutex
}

// NewReconnectStrategy creates a new reconnect strategy
func NewReconnectStrategy(connectFunc, disconnectFunc func(component string) error, cooldown time.Duration) *ReconnectStrategy {
	return &ReconnectStrategy{
		connectFunc:    connectFunc,
		disconnectFunc: disconnectFunc,
		cooldown:       cooldown,
	}
}

// CanHeal returns true for failing components (reconnect is aggressive)
func (rs *ReconnectStrategy) CanHeal(component string, health ComponentHealth) bool {
	return health == HealthFailing
}

// Execute performs disconnect then reconnect
func (rs *ReconnectStrategy) Execute(component string) error {
	rs.strategyMu.RLock()
	dcFn := rs.disconnectFunc
	connFn := rs.connectFunc
	rs.strategyMu.RUnlock()
	
	if dcFn != nil {
		if err := dcFn(component); err != nil {
			return fmt.Errorf("disconnect failed: %w", err)
		}
	}
	if connFn == nil {
		return fmt.Errorf("no connect function configured for component %s", component)
	}
	if err := connFn(component); err != nil {
		return fmt.Errorf("reconnect failed: %w", err)
	}
	return nil
}

// GetCooldown returns the configured cooldown period
func (rs *ReconnectStrategy) GetCooldown() time.Duration {
	rs.strategyMu.RLock()
	defer rs.strategyMu.RUnlock()
	return rs.cooldown
}

// Name returns the strategy identifier
func (rs *ReconnectStrategy) Name() string {
	return "reconnect"
}

// SetFunctions allows updating the connect/disconnect functions at runtime
func (rs *ReconnectStrategy) SetFunctions(connectFunc, disconnectFunc func(component string) error) {
	rs.strategyMu.Lock()
	defer rs.strategyMu.Unlock()
	rs.connectFunc = connectFunc
	rs.disconnectFunc = disconnectFunc
}

// ResetStrategy implements healing by resetting component state
type ResetStrategy struct {
	resetFunc  func(component string) error
	cooldown   time.Duration
	strategyMu sync.RWMutex
}

// NewResetStrategy creates a new reset strategy
func NewResetStrategy(resetFunc func(component string) error, cooldown time.Duration) *ResetStrategy {
	return &ResetStrategy{
		resetFunc: resetFunc,
		cooldown:  cooldown,
	}
}

// CanHeal returns true for any component that isn't healthy
func (rs *ResetStrategy) CanHeal(component string, health ComponentHealth) bool {
	return health != HealthHealthy && health != HealthUnknown
}

// Execute performs the reset operation
func (rs *ResetStrategy) Execute(component string) error {
	rs.strategyMu.RLock()
	fn := rs.resetFunc
	rs.strategyMu.RUnlock()
	
	if fn == nil {
		return fmt.Errorf("no reset function configured for component %s", component)
	}
	return fn(component)
}

// GetCooldown returns the configured cooldown period
func (rs *ResetStrategy) GetCooldown() time.Duration {
	rs.strategyMu.RLock()
	defer rs.strategyMu.RUnlock()
	return rs.cooldown
}

// Name returns the strategy identifier
func (rs *ResetStrategy) Name() string {
	return "reset"
}

// SetResetFunc allows updating the reset function at runtime
func (rs *ResetStrategy) SetResetFunc(fn func(component string) error) {
	rs.strategyMu.Lock()
	defer rs.strategyMu.Unlock()
	rs.resetFunc = fn
}

// circuitInfo tracks circuit breaker state for a component
type circuitInfo struct {
	state          CircuitState
	failureCount   int
	lastFailure    time.Time
	lastAttempt    time.Time
	openedAt       time.Time
	consecutiveSuccesses int
}

// componentState tracks per-component healing state
type componentState struct {
	retryCount  int
	lastAttempt time.Time
	auditTrail  []HealingAction
}

// HealingController manages bounded self-healing for MEL components
type HealingController struct {
	// strategies maps component names to their healing strategies
	strategies map[string]HealingStrategy
	
	// policy is the default retry policy (can be overridden per component)
	policy RetryPolicy
	
	// auditTrail stores all healing actions keyed by component
	auditTrail map[string][]HealingAction
	
	// componentStates tracks per-component healing state
	componentStates map[string]*componentState
	
	// circuits tracks circuit breaker state per component
	circuits map[string]*circuitInfo
	
	// maxAuditEntries limits the size of audit trails per component
	maxAuditEntries int
	
	// circuitFailures before opening circuit (default: 5)
	circuitFailures int
	
	// circuitTimeout before attempting to half-open circuit (default: 30s)
	circuitTimeout time.Duration
	
	mu sync.RWMutex
}

// NewHealingController creates a new healing controller with default settings
func NewHealingController() *HealingController {
	healing := &HealingController{
		strategies:       make(map[string]HealingStrategy),
		policy:           DefaultRetryPolicy(),
		auditTrail:       make(map[string][]HealingAction),
		componentStates:  make(map[string]*componentState),
		circuits:         make(map[string]*circuitInfo),
		maxAuditEntries:  100,
		circuitFailures:  DefaultCircuitFailures,
		circuitTimeout:   DefaultCircuitTimeout,
	}
	
	// Seed the random number generator for jitter
	rand.Seed(time.Now().UnixNano())
	
	return healing
}

// RegisterStrategy registers a healing strategy for a specific component
func (hc *HealingController) RegisterStrategy(component string, strategy HealingStrategy) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.strategies[component] = strategy
}

// SetPolicy sets the default retry policy for all healing operations
func (hc *HealingController) SetPolicy(policy RetryPolicy) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.policy = policy
}

// GetPolicy returns the current default retry policy
func (hc *HealingController) GetPolicy() RetryPolicy {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.policy
}

// getOrCreateComponentState returns or creates the state for a component
func (hc *HealingController) getOrCreateComponentState(component string) *componentState {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	if state, ok := hc.componentStates[component]; ok {
		return state
	}
	
	state := &componentState{
		retryCount: 0,
		auditTrail: make([]HealingAction, 0),
	}
	hc.componentStates[component] = state
	return state
}

// getOrCreateCircuit returns or creates the circuit breaker info for a component
func (hc *HealingController) getOrCreateCircuit(component string) *circuitInfo {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	if circuit, ok := hc.circuits[component]; ok {
		return circuit
	}
	
	circuit := &circuitInfo{
		state: CircuitClosed,
	}
	hc.circuits[component] = circuit
	return circuit
}

// updateCircuit updates the circuit breaker state based on healing result
func (hc *HealingController) updateCircuit(component string, success bool) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	circuit, ok := hc.circuits[component]
	if !ok {
		circuit = &circuitInfo{state: CircuitClosed}
		hc.circuits[component] = circuit
	}
	
	circuit.lastAttempt = time.Now()
	
	if success {
		circuit.consecutiveSuccesses++
		if circuit.state == CircuitHalfOpen && circuit.consecutiveSuccesses >= 2 {
			// Close circuit after consecutive successes in half-open state
			circuit.state = CircuitClosed
			circuit.failureCount = 0
			circuit.consecutiveSuccesses = 0
		} else if circuit.state == CircuitOpen {
			// Transition to half-open after timeout
			if time.Since(circuit.openedAt) > hc.circuitTimeout {
				circuit.state = CircuitHalfOpen
				circuit.consecutiveSuccesses = 1
			}
		}
	} else {
		circuit.consecutiveSuccesses = 0
		circuit.failureCount++
		circuit.lastFailure = time.Now()
		
		if circuit.state == CircuitHalfOpen {
			// Re-open circuit immediately on failure in half-open state
			circuit.state = CircuitOpen
			circuit.openedAt = time.Now()
		} else if circuit.state == CircuitClosed && circuit.failureCount >= hc.circuitFailures {
			// Open circuit after repeated failures
			circuit.state = CircuitOpen
			circuit.openedAt = time.Now()
		}
	}
}

// IsCircuitOpen returns true if the circuit breaker is open for the component
func (hc *HealingController) IsCircuitOpen(component string) bool {
	hc.mu.RLock()
	circuit, ok := hc.circuits[component]
	hc.mu.RUnlock()
	
	if !ok {
		return false
	}
	
	// Check if we should transition from open to half-open
	if circuit.state == CircuitOpen && time.Since(circuit.openedAt) > hc.circuitTimeout {
		hc.mu.Lock()
		circuit.state = CircuitHalfOpen
		hc.mu.Unlock()
		return false // Allow attempt in half-open state
	}
	
	return circuit.state == CircuitOpen
}

// GetCircuitState returns the current circuit breaker state for a component
func (hc *HealingController) GetCircuitState(component string) CircuitState {
	hc.mu.RLock()
	circuit, ok := hc.circuits[component]
	hc.mu.RUnlock()
	
	if !ok {
		return CircuitClosed
	}
	
	// Check for timeout transition
	if circuit.state == CircuitOpen && time.Since(circuit.openedAt) > hc.circuitTimeout {
		return CircuitHalfOpen
	}
	
	return circuit.state
}

// AttemptHeal attempts to heal a component using its registered strategy
func (hc *HealingController) AttemptHeal(component string) HealingAction {
	now := time.Now()
	action := HealingAction{
		Timestamp: now,
		Component: component,
		Success:   false,
	}
	
	// Check if circuit breaker is open
	if hc.IsCircuitOpen(component) {
		action.ActionType = ActionCooldown
		action.ErrorMessage = "circuit breaker is open"
		hc.recordAction(action)
		return action
	}
	
	// Get the healing strategy for this component
	hc.mu.RLock()
	strategy, hasStrategy := hc.strategies[component]
	hc.mu.RUnlock()
	
	if !hasStrategy {
		action.ActionType = ActionCooldown
		action.ErrorMessage = "no healing strategy registered"
		hc.recordAction(action)
		return action
	}
	
	action.StrategyName = strategy.Name()
	
	// Get component health from global registry
	registry := GetGlobalRegistry()
	compHealth := registry.GetComponent(component)
	
	// Check if healing is needed
	if !strategy.CanHeal(component, compHealth.Health) {
		action.ActionType = ActionCooldown
		action.ErrorMessage = "component does not require healing"
		action.Success = true // Not a failure, just no action needed
		hc.recordAction(action)
		return action
	}
	
	// Get component state
	state := hc.getOrCreateComponentState(component)
	
	// Check cooldown period
	hc.mu.RLock()
	cooldown := hc.policy.CooldownWindow
	if strategy.GetCooldown() > cooldown {
		cooldown = strategy.GetCooldown()
	}
	hc.mu.RUnlock()
	
	if !state.lastAttempt.IsZero() && now.Sub(state.lastAttempt) < cooldown {
		action.ActionType = ActionCooldown
		action.ErrorMessage = fmt.Sprintf("component in cooldown, next attempt after %v", state.lastAttempt.Add(cooldown))
		hc.recordAction(action)
		return action
	}
	
	// Update state
	state.lastAttempt = now
	action.RetryCount = state.retryCount
	
	// Determine action type from strategy name
	switch strategy.Name() {
	case "retry":
		action.ActionType = ActionRetry
	case "reconnect":
		action.ActionType = ActionReconnect
	case "reset":
		action.ActionType = ActionReset
	default:
		action.ActionType = ActionRetry
	}
	
	// Execute healing strategy with retry logic
	hc.mu.RLock()
	maxRetries := hc.policy.MaxRetries
	hc.mu.RUnlock()
	
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		action.RetryCount = attempt
		
		if err := strategy.Execute(component); err != nil {
			lastErr = err
			
			// Don't sleep on last attempt
			if attempt < maxRetries {
				hc.mu.RLock()
				backoff := hc.policy.getBackoff(attempt)
				hc.mu.RUnlock()
				time.Sleep(backoff)
			}
		} else {
			// Success!
			action.Success = true
			state.retryCount = 0
			hc.updateCircuit(component, true)
			hc.recordAction(action)
			
			// Update component health in registry
			registry.SetHealth(component, HealthHealthy)
			
			return action
		}
	}
	
	// All retries exhausted
	action.ErrorMessage = fmt.Sprintf("healing failed after %d retries: %v", maxRetries+1, lastErr)
	state.retryCount++
	hc.updateCircuit(component, false)
	hc.recordAction(action)
	
	return action
}

// recordAction adds a healing action to the audit trail
func (hc *HealingController) recordAction(action HealingAction) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	// Get existing trail
	trail, ok := hc.auditTrail[action.Component]
	if !ok {
		trail = make([]HealingAction, 0)
	}
	
	// Add new action
	trail = append(trail, action)
	
	// Trim to max entries
	if len(trail) > hc.maxAuditEntries {
		trail = trail[len(trail)-hc.maxAuditEntries:]
	}
	
	hc.auditTrail[action.Component] = trail
	
	// Also update component state audit trail
	if state, ok := hc.componentStates[action.Component]; ok {
		state.auditTrail = trail
	}
}

// GetAuditTrail returns the audit trail for a specific component
func (hc *HealingController) GetAuditTrail(component string) []HealingAction {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	if trail, ok := hc.auditTrail[component]; ok {
		// Return a copy to prevent external modification
		result := make([]HealingAction, len(trail))
		copy(result, trail)
		return result
	}
	
	return []HealingAction{}
}

// GetAllAuditTrails returns audit trails for all components
func (hc *HealingController) GetAllAuditTrails() map[string][]HealingAction {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	result := make(map[string][]HealingAction)
	for component, trail := range hc.auditTrail {
		trailCopy := make([]HealingAction, len(trail))
		copy(trailCopy, trail)
		result[component] = trailCopy
	}
	
	return result
}

// ResetCircuit manually resets the circuit breaker for a component
func (hc *HealingController) ResetCircuit(component string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	hc.circuits[component] = &circuitInfo{
		state: CircuitClosed,
	}
	
	if state, ok := hc.componentStates[component]; ok {
		state.retryCount = 0
	}
}

// UnregisterStrategy removes a healing strategy for a component
func (hc *HealingController) UnregisterStrategy(component string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	delete(hc.strategies, component)
}

// GetRegisteredComponents returns all components with registered strategies
func (hc *HealingController) GetRegisteredComponents() []string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	components := make([]string, 0, len(hc.strategies))
	for component := range hc.strategies {
		components = append(components, component)
	}
	
	return components
}

// SetMaxAuditEntries configures the maximum number of audit entries per component
func (hc *HealingController) SetMaxAuditEntries(max int) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.maxAuditEntries = max
}

// SetCircuitThresholds configures the circuit breaker thresholds
func (hc *HealingController) SetCircuitThresholds(failures int, timeout time.Duration) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.circuitFailures = failures
	hc.circuitTimeout = timeout
}
