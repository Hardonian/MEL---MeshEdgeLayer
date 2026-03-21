// Package selfobs provides bounded self-healing capabilities for MEL components.
// This file implements transport-specific healing strategies for MQTT and Direct transports.
package selfobs

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// TransportActionType represents transport-specific healing action types
type TransportActionType string

const (
	// TransportActionReconnect re-establishes connection
	TransportActionReconnect TransportActionType = "reconnect"
	// TransportActionFailover switches to backup endpoint
	TransportActionFailover TransportActionType = "failover"
	// TransportActionQoSAdjust changes QoS settings
	TransportActionQoSAdjust TransportActionType = "qos_adjust"
	// TransportActionBufferReset clears and resets buffers
	TransportActionBufferReset TransportActionType = "buffer_reset"
	// TransportActionCircuitReset resets circuit breaker
	TransportActionCircuitReset TransportActionType = "circuit_reset"
)

// TransportType represents the type of transport
type TransportType string

const (
	// TransportMQTT represents MQTT transport
	TransportMQTT TransportType = "mqtt"
	// TransportDirect represents direct transport
	TransportDirect TransportType = "direct"
)

// TransportConfig is the base interface for transport-specific configurations
type TransportConfig interface {
	GetTransportType() TransportType
}

// MQTTConfig holds configuration for MQTT transport
type MQTTConfig struct {
	BrokerURL    string
	ClientID     string
	QoS          byte
	CleanSession bool
	KeepAlive    time.Duration
	RetryPolicy  RetryPolicy
}

// GetTransportType returns the transport type for MQTT
func (m MQTTConfig) GetTransportType() TransportType {
	return TransportMQTT
}

// DirectConfig holds configuration for Direct transport
type DirectConfig struct {
	Endpoint    string
	Timeout     time.Duration
	BufferSize  int
	RetryPolicy RetryPolicy
}

// GetTransportType returns the transport type for Direct
func (d DirectConfig) GetTransportType() TransportType {
	return TransportDirect
}

// TransportHealth tracks the health state of a transport
type TransportHealth struct {
	TransportType string
	State         ComponentHealth
	LastFailure   error
	FailoverCount int
	LastFailover  time.Time
	Latency       time.Duration
}

// TransportHealer extends HealingStrategy with transport-specific capabilities
type TransportHealer interface {
	HealingStrategy
	// CanHeal returns true if the strategy can heal the given transport
	CanHeal(transportType string, state ComponentHealth) bool
	// ExecuteTransport performs the healing action for the transport
	ExecuteTransport(transportType string, config TransportConfig) error
	// GetTransportType returns the transport type this healer handles
	GetTransportType() string
}

// TransportHealingStrategy wraps a HealingStrategy for transport use
type TransportHealingStrategy struct {
	strategy      HealingStrategy
	transportType TransportType
	canHealFunc   func(transportType string, state ComponentHealth) bool
	executeFunc   func(transportType string, config TransportConfig) error
}

// CanHeal returns true if the strategy can heal the given transport
func (ths *TransportHealingStrategy) CanHeal(transportType string, state ComponentHealth) bool {
	if ths.canHealFunc != nil {
		return ths.canHealFunc(transportType, state)
	}
	return ths.strategy.CanHeal(transportType, state)
}

// Execute performs the healing action for the transport
func (ths *TransportHealingStrategy) ExecuteTransport(transportType string, config TransportConfig) error {
	if ths.executeFunc != nil {
		return ths.executeFunc(transportType, config)
	}
	return ths.strategy.Execute(transportType)
}

// GetCooldown returns the cooldown period from the underlying strategy
func (ths *TransportHealingStrategy) GetCooldown() time.Duration {
	return ths.strategy.GetCooldown()
}

// Name returns the strategy identifier
func (ths *TransportHealingStrategy) Name() string {
	return ths.strategy.Name()
}

// GetTransportType returns the transport type
func (ths *TransportHealingStrategy) GetTransportType() string {
	return string(ths.transportType)
}

// MQTTRetryStrategy reconnects with backoff and clean session toggle
type MQTTRetryStrategy struct {
	policy        RetryPolicy
	reconnectFunc func(config MQTTConfig) error
	strategyMu    sync.RWMutex
}

// NewMQTTRetryStrategy creates a new MQTT retry strategy
func NewMQTTRetryStrategy(policy RetryPolicy, reconnectFunc func(config MQTTConfig) error) *MQTTRetryStrategy {
	return &MQTTRetryStrategy{
		policy:        policy,
		reconnectFunc: reconnectFunc,
	}
}

// CanHeal returns true for degraded or failing MQTT transports
func (mrs *MQTTRetryStrategy) CanHeal(transportType string, state ComponentHealth) bool {
	if transportType != string(TransportMQTT) {
		return false
	}
	return state == HealthDegraded || state == HealthFailing
}

// Execute performs the MQTT reconnection with backoff
func (mrs *MQTTRetryStrategy) ExecuteTransport(transportType string, config TransportConfig) error {
	mqttConfig, ok := config.(MQTTConfig)
	if !ok {
		return fmt.Errorf("invalid config type for MQTT transport: %T", config)
	}

	mrs.strategyMu.RLock()
	fn := mrs.reconnectFunc
	mrs.strategyMu.RUnlock()

	if fn == nil {
		return errors.New("no reconnect function configured for MQTT strategy")
	}

	var lastErr error
	for attempt := 0; attempt <= mrs.policy.MaxRetries; attempt++ {
		if err := fn(mqttConfig); err != nil {
			lastErr = err
			if attempt < mrs.policy.MaxRetries {
				backoff := mrs.policy.getBackoff(attempt)
				time.Sleep(backoff)
				// Toggle clean session on retry
				mqttConfig.CleanSession = !mqttConfig.CleanSession
			}
		} else {
			return nil
		}
	}

	return fmt.Errorf("MQTT retry failed after %d attempts: %w", mrs.policy.MaxRetries+1, lastErr)
}

// GetCooldown returns the policy's cooldown window
func (mrs *MQTTRetryStrategy) GetCooldown() time.Duration {
	mrs.strategyMu.RLock()
	defer mrs.strategyMu.RUnlock()
	return mrs.policy.CooldownWindow
}

// Name returns the strategy identifier
func (mrs *MQTTRetryStrategy) Name() string {
	return "mqtt_retry"
}

// GetTransportType returns the transport type
func (mrs *MQTTRetryStrategy) GetTransportType() string {
	return string(TransportMQTT)
}

// Execute implements HealingStrategy.Execute for compatibility
func (mrs *MQTTRetryStrategy) Execute(component string) error {
	return errors.New("use ExecuteTransport for transport strategies")
}

// SetReconnectFunc allows updating the reconnect function at runtime
func (mrs *MQTTRetryStrategy) SetReconnectFunc(fn func(config MQTTConfig) error) {
	mrs.strategyMu.Lock()
	defer mrs.strategyMu.Unlock()
	mrs.reconnectFunc = fn
}

// MQTTBrokerFailoverStrategy fails over to backup broker
type MQTTBrokerFailoverStrategy struct {
	backupBrokers []string
	failoverFunc  func(primaryConfig MQTTConfig, backupBroker string) error
	currentBroker int
	strategyMu    sync.RWMutex
}

// NewMQTTBrokerFailoverStrategy creates a new broker failover strategy
func NewMQTTBrokerFailoverStrategy(backupBrokers []string, failoverFunc func(primaryConfig MQTTConfig, backupBroker string) error) *MQTTBrokerFailoverStrategy {
	return &MQTTBrokerFailoverStrategy{
		backupBrokers: backupBrokers,
		failoverFunc:  failoverFunc,
		currentBroker: 0,
	}
}

// CanHeal returns true for failing MQTT transports when backup brokers are available
func (mfs *MQTTBrokerFailoverStrategy) CanHeal(transportType string, state ComponentHealth) bool {
	if transportType != string(TransportMQTT) {
		return false
	}
	mfs.strategyMu.RLock()
	defer mfs.strategyMu.RUnlock()
	return state == HealthFailing && len(mfs.backupBrokers) > 0
}

// Execute performs the broker failover
func (mfs *MQTTBrokerFailoverStrategy) ExecuteTransport(transportType string, config TransportConfig) error {
	mqttConfig, ok := config.(MQTTConfig)
	if !ok {
		return fmt.Errorf("invalid config type for MQTT transport: %T", config)
	}

	mfs.strategyMu.Lock()
	fn := mfs.failoverFunc
	if mfs.currentBroker >= len(mfs.backupBrokers) {
		mfs.currentBroker = 0
	}
	backupBroker := mfs.backupBrokers[mfs.currentBroker]
	mfs.currentBroker++
	mfs.strategyMu.Unlock()

	if fn == nil {
		return errors.New("no failover function configured for MQTT broker failover strategy")
	}

	return fn(mqttConfig, backupBroker)
}

// GetCooldown returns a fixed cooldown for failover operations
func (mfs *MQTTBrokerFailoverStrategy) GetCooldown() time.Duration {
	return 30 * time.Second
}

// Name returns the strategy identifier
func (mfs *MQTTBrokerFailoverStrategy) Name() string {
	return "mqtt_broker_failover"
}

// GetTransportType returns the transport type
func (mfs *MQTTBrokerFailoverStrategy) GetTransportType() string {
	return string(TransportMQTT)
}

// Execute implements HealingStrategy.Execute for compatibility
func (mfs *MQTTBrokerFailoverStrategy) Execute(component string) error {
	return errors.New("use ExecuteTransport for transport strategies")
}

// SetFailoverFunc allows updating the failover function at runtime
func (mfs *MQTTBrokerFailoverStrategy) SetFailoverFunc(fn func(primaryConfig MQTTConfig, backupBroker string) error) {
	mfs.strategyMu.Lock()
	defer mfs.strategyMu.Unlock()
	mfs.failoverFunc = fn
}

// MQTTQoSAdjustmentStrategy adjusts QoS level temporarily
type MQTTQoSAdjustmentStrategy struct {
	originalQoS byte
	adjustedQoS byte
	adjustFunc  func(config MQTTConfig, newQoS byte) error
	strategyMu  sync.RWMutex
}

// NewMQTTQoSAdjustmentStrategy creates a new QoS adjustment strategy
func NewMQTTQoSAdjustmentStrategy(adjustedQoS byte, adjustFunc func(config MQTTConfig, newQoS byte) error) *MQTTQoSAdjustmentStrategy {
	return &MQTTQoSAdjustmentStrategy{
		adjustedQoS: adjustedQoS,
		adjustFunc:  adjustFunc,
	}
}

// CanHeal returns true for degraded MQTT transports
func (mqs *MQTTQoSAdjustmentStrategy) CanHeal(transportType string, state ComponentHealth) bool {
	if transportType != string(TransportMQTT) {
		return false
	}
	return state == HealthDegraded
}

// Execute performs the QoS adjustment
func (mqs *MQTTQoSAdjustmentStrategy) ExecuteTransport(transportType string, config TransportConfig) error {
	mqttConfig, ok := config.(MQTTConfig)
	if !ok {
		return fmt.Errorf("invalid config type for MQTT transport: %T", config)
	}

	mqs.strategyMu.Lock()
	mqs.originalQoS = mqttConfig.QoS
	fn := mqs.adjustFunc
	mqs.strategyMu.Unlock()

	if fn == nil {
		return errors.New("no adjust function configured for MQTT QoS strategy")
	}

	return fn(mqttConfig, mqs.adjustedQoS)
}

// GetCooldown returns a short cooldown for QoS adjustments
func (mqs *MQTTQoSAdjustmentStrategy) GetCooldown() time.Duration {
	return 5 * time.Second
}

// Name returns the strategy identifier
func (mqs *MQTTQoSAdjustmentStrategy) Name() string {
	return "mqtt_qos_adjust"
}

// GetTransportType returns the transport type
func (mqs *MQTTQoSAdjustmentStrategy) GetTransportType() string {
	return string(TransportMQTT)
}

// Execute implements HealingStrategy.Execute for compatibility
func (mqs *MQTTQoSAdjustmentStrategy) Execute(component string) error {
	return errors.New("use ExecuteTransport for transport strategies")
}

// SetAdjustFunc allows updating the adjust function at runtime
func (mqs *MQTTQoSAdjustmentStrategy) SetAdjustFunc(fn func(config MQTTConfig, newQoS byte) error) {
	mqs.strategyMu.Lock()
	defer mqs.strategyMu.Unlock()
	mqs.adjustFunc = fn
}

// RestoreQoS restores the original QoS level
func (mqs *MQTTQoSAdjustmentStrategy) RestoreQoS(config MQTTConfig) error {
	mqs.strategyMu.RLock()
	fn := mqs.adjustFunc
	originalQoS := mqs.originalQoS
	mqs.strategyMu.RUnlock()

	if fn == nil {
		return errors.New("no adjust function configured")
	}

	return fn(config, originalQoS)
}

// DirectRetryStrategy retries with exponential backoff
type DirectRetryStrategy struct {
	policy     RetryPolicy
	retryFunc  func(config DirectConfig) error
	strategyMu sync.RWMutex
}

// NewDirectRetryStrategy creates a new direct retry strategy
func NewDirectRetryStrategy(policy RetryPolicy, retryFunc func(config DirectConfig) error) *DirectRetryStrategy {
	return &DirectRetryStrategy{
		policy:    policy,
		retryFunc: retryFunc,
	}
}

// CanHeal returns true for degraded or failing direct transports
func (drs *DirectRetryStrategy) CanHeal(transportType string, state ComponentHealth) bool {
	if transportType != string(TransportDirect) {
		return false
	}
	return state == HealthDegraded || state == HealthFailing
}

// Execute performs the retry with exponential backoff
func (drs *DirectRetryStrategy) ExecuteTransport(transportType string, config TransportConfig) error {
	directConfig, ok := config.(DirectConfig)
	if !ok {
		return fmt.Errorf("invalid config type for Direct transport: %T", config)
	}

	drs.strategyMu.RLock()
	fn := drs.retryFunc
	drs.strategyMu.RUnlock()

	if fn == nil {
		return errors.New("no retry function configured for Direct strategy")
	}

	var lastErr error
	for attempt := 0; attempt <= drs.policy.MaxRetries; attempt++ {
		if err := fn(directConfig); err != nil {
			lastErr = err
			if attempt < drs.policy.MaxRetries {
				backoff := drs.policy.getBackoff(attempt)
				time.Sleep(backoff)
			}
		} else {
			return nil
		}
	}

	return fmt.Errorf("direct retry failed after %d attempts: %w", drs.policy.MaxRetries+1, lastErr)
}

// GetCooldown returns the policy's cooldown window
func (drs *DirectRetryStrategy) GetCooldown() time.Duration {
	drs.strategyMu.RLock()
	defer drs.strategyMu.RUnlock()
	return drs.policy.CooldownWindow
}

// Name returns the strategy identifier
func (drs *DirectRetryStrategy) Name() string {
	return "direct_retry"
}

// GetTransportType returns the transport type
func (drs *DirectRetryStrategy) GetTransportType() string {
	return string(TransportDirect)
}

// Execute implements HealingStrategy.Execute for compatibility
func (drs *DirectRetryStrategy) Execute(component string) error {
	return errors.New("use ExecuteTransport for transport strategies")
}

// SetRetryFunc allows updating the retry function at runtime
func (drs *DirectRetryStrategy) SetRetryFunc(fn func(config DirectConfig) error) {
	drs.strategyMu.Lock()
	defer drs.strategyMu.Unlock()
	drs.retryFunc = fn
}

// DirectCircuitResetStrategy forces circuit breaker reset
type DirectCircuitResetStrategy struct {
	resetFunc  func(config DirectConfig) error
	cooldown   time.Duration
	strategyMu sync.RWMutex
}

// NewDirectCircuitResetStrategy creates a new circuit reset strategy
func NewDirectCircuitResetStrategy(resetFunc func(config DirectConfig) error) *DirectCircuitResetStrategy {
	return &DirectCircuitResetStrategy{
		resetFunc: resetFunc,
		cooldown:  10 * time.Second,
	}
}

// CanHeal returns true for failing direct transports
func (dcr *DirectCircuitResetStrategy) CanHeal(transportType string, state ComponentHealth) bool {
	if transportType != string(TransportDirect) {
		return false
	}
	return state == HealthFailing
}

// Execute performs the circuit breaker reset
func (dcr *DirectCircuitResetStrategy) ExecuteTransport(transportType string, config TransportConfig) error {
	directConfig, ok := config.(DirectConfig)
	if !ok {
		return fmt.Errorf("invalid config type for Direct transport: %T", config)
	}

	dcr.strategyMu.RLock()
	fn := dcr.resetFunc
	dcr.strategyMu.RUnlock()

	if fn == nil {
		return errors.New("no reset function configured for Direct circuit reset strategy")
	}

	return fn(directConfig)
}

// GetCooldown returns the configured cooldown
func (dcr *DirectCircuitResetStrategy) GetCooldown() time.Duration {
	dcr.strategyMu.RLock()
	defer dcr.strategyMu.RUnlock()
	return dcr.cooldown
}

// Name returns the strategy identifier
func (dcr *DirectCircuitResetStrategy) Name() string {
	return "direct_circuit_reset"
}

// GetTransportType returns the transport type
func (dcr *DirectCircuitResetStrategy) GetTransportType() string {
	return string(TransportDirect)
}

// Execute implements HealingStrategy.Execute for compatibility
func (dcr *DirectCircuitResetStrategy) Execute(component string) error {
	return errors.New("use ExecuteTransport for transport strategies")
}

// SetResetFunc allows updating the reset function at runtime
func (dcr *DirectCircuitResetStrategy) SetResetFunc(fn func(config DirectConfig) error) {
	dcr.strategyMu.Lock()
	defer dcr.strategyMu.Unlock()
	dcr.resetFunc = fn
}

// DirectBufferFlushStrategy flushes and resets buffers
type DirectBufferFlushStrategy struct {
	flushFunc  func(config DirectConfig) error
	cooldown   time.Duration
	strategyMu sync.RWMutex
}

// NewDirectBufferFlushStrategy creates a new buffer flush strategy
func NewDirectBufferFlushStrategy(flushFunc func(config DirectConfig) error) *DirectBufferFlushStrategy {
	return &DirectBufferFlushStrategy{
		flushFunc: flushFunc,
		cooldown:  5 * time.Second,
	}
}

// CanHeal returns true for degraded or failing direct transports with buffer issues
func (dbf *DirectBufferFlushStrategy) CanHeal(transportType string, state ComponentHealth) bool {
	if transportType != string(TransportDirect) {
		return false
	}
	return state == HealthDegraded || state == HealthFailing
}

// Execute performs the buffer flush and reset
func (dbf *DirectBufferFlushStrategy) ExecuteTransport(transportType string, config TransportConfig) error {
	directConfig, ok := config.(DirectConfig)
	if !ok {
		return fmt.Errorf("invalid config type for Direct transport: %T", config)
	}

	dbf.strategyMu.RLock()
	fn := dbf.flushFunc
	dbf.strategyMu.RUnlock()

	if fn == nil {
		return errors.New("no flush function configured for Direct buffer flush strategy")
	}

	return fn(directConfig)
}

// GetCooldown returns the configured cooldown
func (dbf *DirectBufferFlushStrategy) GetCooldown() time.Duration {
	dbf.strategyMu.RLock()
	defer dbf.strategyMu.RUnlock()
	return dbf.cooldown
}

// Name returns the strategy identifier
func (dbf *DirectBufferFlushStrategy) Name() string {
	return "direct_buffer_flush"
}

// GetTransportType returns the transport type
func (dbf *DirectBufferFlushStrategy) GetTransportType() string {
	return string(TransportDirect)
}

// Execute implements HealingStrategy.Execute for compatibility
func (dbf *DirectBufferFlushStrategy) Execute(component string) error {
	return errors.New("use ExecuteTransport for transport strategies")
}

// SetFlushFunc allows updating the flush function at runtime
func (dbf *DirectBufferFlushStrategy) SetFlushFunc(fn func(config DirectConfig) error) {
	dbf.strategyMu.Lock()
	defer dbf.strategyMu.Unlock()
	dbf.flushFunc = fn
}

// transportFailoverInfo tracks failover state for a transport
type transportFailoverInfo struct {
	failoverCount  int
	lastFailover   time.Time
	failoverWindow time.Time
}

// TransportHealingController extends HealingController with transport-specific capabilities
type TransportHealingController struct {
	*HealingController
	transportStrategies map[string]TransportHealer
	transportStates     map[string]*TransportHealth
	failoverInfo        map[string]*transportFailoverInfo
	failoverCooldown    time.Duration
	maxFailoversPerHour int
	latencyTracking     map[string][]time.Duration
	mu                  sync.RWMutex
}

// NewTransportHealingController creates a new transport healing controller
func NewTransportHealingController() *TransportHealingController {
	return &TransportHealingController{
		HealingController:   NewHealingController(),
		transportStrategies: make(map[string]TransportHealer),
		transportStates:     make(map[string]*TransportHealth),
		failoverInfo:        make(map[string]*transportFailoverInfo),
		failoverCooldown:    5 * time.Minute,
		maxFailoversPerHour: 3,
		latencyTracking:     make(map[string][]time.Duration),
	}
}

// RegisterTransportStrategy registers a healing strategy for a specific transport
func (thc *TransportHealingController) RegisterTransportStrategy(transportType string, strategy TransportHealer) {
	thc.mu.Lock()
	defer thc.mu.Unlock()
	thc.transportStrategies[transportType] = strategy
}

// AttemptTransportHeal attempts to heal a transport using its registered strategy
func (thc *TransportHealingController) AttemptTransportHeal(transportType string, config TransportConfig) HealingAction {
	now := time.Now()
	action := HealingAction{
		Timestamp: now,
		Component: transportType,
		Success:   false,
	}

	thc.mu.RLock()
	strategy, hasStrategy := thc.transportStrategies[transportType]
	thc.mu.RUnlock()

	if !hasStrategy {
		action.ActionType = ActionType(ActionCooldown)
		action.ErrorMessage = "no transport healing strategy registered"
		thc.recordAction(action)
		return action
	}

	action.StrategyName = strategy.Name()

	state := thc.GetTransportHealth(transportType)

	if !strategy.CanHeal(transportType, state.State) {
		action.ActionType = ActionType(ActionCooldown)
		action.ErrorMessage = "transport does not require healing"
		action.Success = true
		thc.recordAction(action)
		return action
	}

	// Check max failovers first (global rate limit)
	if !thc.canFailover(transportType) {
		action.ActionType = ActionType(ActionCooldown)
		action.ErrorMessage = "max failovers per hour exceeded"
		thc.recordAction(action)
		return action
	}

	// Check cooldown using the controller's failover cooldown (allows test configuration)
	thc.mu.RLock()
	cooldown := thc.failoverCooldown
	info, hasInfo := thc.failoverInfo[transportType]
	thc.mu.RUnlock()

	if hasInfo && !info.lastFailover.IsZero() && now.Sub(info.lastFailover) < cooldown {
		action.ActionType = ActionType(ActionCooldown)
		action.ErrorMessage = fmt.Sprintf("transport in failover cooldown, next attempt after %v", info.lastFailover.Add(cooldown))
		thc.recordAction(action)
		return action
	}

	startTime := time.Now()
	err := strategy.ExecuteTransport(transportType, config)
	latency := time.Since(startTime)

	thc.recordLatency(transportType, latency)

	if err != nil {
		action.ErrorMessage = fmt.Sprintf("transport healing failed: %v", err)
		thc.RecordTransportFailure(transportType, err)
		thc.updateCircuit(transportType, false)
	} else {
		action.Success = true
		thc.RecordTransportSuccess(transportType)
		thc.updateCircuit(transportType, true)
	}

	actionType := thc.determineActionType(strategy)
	action.ActionType = ActionType(actionType)

	if actionType == TransportActionFailover {
		thc.incrementFailoverCount(transportType)
	}

	thc.recordAction(action)
	return action
}

// determineActionType maps strategy names to action types
func (thc *TransportHealingController) determineActionType(strategy TransportHealer) TransportActionType {
	switch strategy.Name() {
	case "mqtt_retry", "direct_retry":
		return TransportActionReconnect
	case "mqtt_broker_failover":
		return TransportActionFailover
	case "mqtt_qos_adjust":
		return TransportActionQoSAdjust
	case "direct_circuit_reset":
		return TransportActionCircuitReset
	case "direct_buffer_flush":
		return TransportActionBufferReset
	default:
		return TransportActionReconnect
	}
}

// canFailover checks if failover is allowed based on rate limits
func (thc *TransportHealingController) canFailover(transportType string) bool {
	thc.mu.RLock()
	defer thc.mu.RUnlock()

	info, ok := thc.failoverInfo[transportType]
	if !ok {
		return true
	}

	// Reset count if we're outside the hour window
	if time.Since(info.failoverWindow) > time.Hour {
		return true
	}

	return info.failoverCount < thc.maxFailoversPerHour
}

// incrementFailoverCount increments the failover count for a transport
func (thc *TransportHealingController) incrementFailoverCount(transportType string) {
	thc.mu.Lock()
	defer thc.mu.Unlock()

	info, ok := thc.failoverInfo[transportType]
	if !ok {
		info = &transportFailoverInfo{}
		thc.failoverInfo[transportType] = info
	}

	// Reset if outside the hour window
	if time.Since(info.failoverWindow) > time.Hour {
		info.failoverCount = 0
		info.failoverWindow = time.Now()
	}

	info.failoverCount++
	info.lastFailover = time.Now()

	// Also update the TransportHealth for visibility
	if state, ok := thc.transportStates[transportType]; ok {
		state.FailoverCount = info.failoverCount
		state.LastFailover = info.lastFailover
	}
}

// recordLatency records latency for a transport
func (thc *TransportHealingController) recordLatency(transportType string, latency time.Duration) {
	thc.mu.Lock()
	defer thc.mu.Unlock()

	latencies := thc.latencyTracking[transportType]
	latencies = append(latencies, latency)

	// Keep last 100 latency measurements
	if len(latencies) > 100 {
		latencies = latencies[len(latencies)-100:]
	}

	thc.latencyTracking[transportType] = latencies

	if state, ok := thc.transportStates[transportType]; ok {
		state.Latency = latency
	}
}

// GetTransportHealth returns the health state for a transport
func (thc *TransportHealingController) GetTransportHealth(transportType string) TransportHealth {
	thc.mu.RLock()
	defer thc.mu.RUnlock()

	if state, ok := thc.transportStates[transportType]; ok {
		return *state
	}

	return TransportHealth{
		TransportType: transportType,
		State:         HealthUnknown,
	}
}

// RecordTransportFailure records a failure for a transport
func (thc *TransportHealingController) RecordTransportFailure(transportType string, err error) {
	thc.mu.Lock()
	defer thc.mu.Unlock()

	state, ok := thc.transportStates[transportType]
	if !ok {
		state = &TransportHealth{
			TransportType: transportType,
		}
		thc.transportStates[transportType] = state
	}

	state.LastFailure = err

	if state.State == HealthHealthy || state.State == HealthUnknown || state.State == "" {
		state.State = HealthDegraded
	} else if state.State == HealthDegraded {
		state.State = HealthFailing
	}

	thc.updateCircuit(transportType, false)
}

// RecordTransportSuccess records a success for a transport
func (thc *TransportHealingController) RecordTransportSuccess(transportType string) {
	thc.mu.Lock()
	defer thc.mu.Unlock()

	state, ok := thc.transportStates[transportType]
	if !ok {
		state = &TransportHealth{
			TransportType: transportType,
		}
		thc.transportStates[transportType] = state
	}

	state.LastFailure = nil

	if state.State == HealthFailing {
		state.State = HealthDegraded
	} else if state.State == HealthDegraded {
		state.State = HealthHealthy
	} else if state.State == "" || state.State == HealthUnknown {
		state.State = HealthHealthy
	}

	thc.updateCircuit(transportType, true)
}

// SetFailoverCooldown sets the cooldown period between failovers
func (thc *TransportHealingController) SetFailoverCooldown(cooldown time.Duration) {
	thc.mu.Lock()
	defer thc.mu.Unlock()
	thc.failoverCooldown = cooldown
}

// SetMaxFailoversPerHour sets the maximum number of failovers allowed per hour
func (thc *TransportHealingController) SetMaxFailoversPerHour(max int) {
	thc.mu.Lock()
	defer thc.mu.Unlock()
	thc.maxFailoversPerHour = max
}

// GetAverageLatency returns the average latency for a transport
func (thc *TransportHealingController) GetAverageLatency(transportType string) time.Duration {
	thc.mu.RLock()
	defer thc.mu.RUnlock()

	latencies := thc.latencyTracking[transportType]
	if len(latencies) == 0 {
		return 0
	}

	var total time.Duration
	for _, lat := range latencies {
		total += lat
	}

	return total / time.Duration(len(latencies))
}

// GracefulDegrade attempts to gracefully degrade transport before disconnecting
func (thc *TransportHealingController) GracefulDegrade(transportType string, config TransportConfig) error {
	switch cfg := config.(type) {
	case MQTTConfig:
		if cfg.QoS > 0 {
			cfg.QoS--
			strategy := NewMQTTQoSAdjustmentStrategy(cfg.QoS, nil)
			return strategy.ExecuteTransport(transportType, cfg)
		}
	case DirectConfig:
		if cfg.BufferSize > 1024 {
			cfg.BufferSize = cfg.BufferSize / 2
			strategy := NewDirectBufferFlushStrategy(nil)
			return strategy.ExecuteTransport(transportType, cfg)
		}
	}

	return nil
}
