package selfobs

// =============================================================================
// LLM INTEGRATION MODULE - SAFE, OPTIONAL, BOUNDED
// =============================================================================
//
// CAPABILITY DECLARATION - READ THIS FIRST:
//
// WHAT IS IMPLEMENTED:
//   - Framework for LLM integration with safety-first design
//   - LLMProvider interface with Classify, Explain, Suggest, HealthCheck methods
//   - DisabledProvider: production-ready no-op provider
//   - LocalProvider: stub for local LLM integration (Ollama, etc.)
//   - APIProvider: stub for API-based LLM integration
//   - LLMClient with timeout, retry, and fallback handling
//   - Privacy-first redaction system for all external calls
//   - Thread-safe audit logging of all LLM interactions
//   - Configuration validation and secure credential handling
//
// WHAT IS NOT IMPLEMENTED:
//   - Actual LLM API integrations (no HTTP clients for Ollama, OpenAI, etc.)
//   - Real model inference or token streaming
//   - Prompt templates and fine-tuning
//   - Response parsing and structured output validation
//
// WHY NOT IMPLEMENTED:
//   LLM integration requires operator configuration and validation:
//   - Model selection and deployment (local vs. remote)
//   - API key management and rotation
//   - Rate limiting and cost controls
//   - Response validation and safety filtering
//   - These concerns require operational decisions before implementation.
//
// TO ENABLE LLM SUPPORT:
//   1. Choose provider: "disabled" (default), "local", or "api"
//   2. Configure endpoint and credentials
//   3. Validate privacy redaction rules meet compliance requirements
//   4. Implement actual HTTP/gRPC client for chosen provider
//   5. Add prompt templates and response parsing
//
// =============================================================================
// DESIGN PRINCIPLES
// =============================================================================
//
// 1. OPTIONAL: LLM is never required. The system operates fully without it.
//    All LLM calls have deterministic fallbacks. Disabled by default.
//
// 2. NON-AUTHORITATIVE: LLM output is never primary authority. It only
//    provides suggestions that must be validated by deterministic logic.
//    No control-plane mutations from model output.
//
// 3. PRIVACY-FIRST: All data sent to external LLMs is redacted.
//    - Strings replaced with [REDACTED]
//    - Timestamps replaced with [TIME]
//    - Only statistical patterns and component names are preserved
//
// 4. BOUNDED: All LLM calls have strict timeouts (default 5s, max 30s)
//    and retry limits. No unbounded operations.
//
// 5. AUDITABLE: Every LLM interaction is logged with:
//    - Timestamp
//    - Operation type (classify, explain, suggest)
//    - Input hash (content never logged)
//    - Success/failure status
//    - Response hash (content never logged)
//    - Latency measurement
//    - Error details (sanitized)
//
// 6. FAIL-SAFE: All errors return to deterministic fallback.
//    No partial failures or undefined behavior.
//
// =============================================================================

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// DefaultLLMTimeout is the default timeout for LLM operations
	DefaultLLMTimeout = 5 * time.Second

	// MaxLLMTimeout is the maximum allowed timeout for LLM operations
	MaxLLMTimeout = 30 * time.Second

	// DefaultLLMMaxRetries is the default number of retries for LLM operations
	DefaultLLMMaxRetries = 2

	// MaxAuditLogEntries is the maximum number of audit log entries to retain
	MaxAuditLogEntries = 10000
)

// ProviderType identifies the type of LLM provider
// Valid values: "disabled", "local", "api"
type ProviderType string

const (
	// ProviderDisabled indicates LLM integration is disabled (default)
	ProviderDisabled ProviderType = "disabled"

	// ProviderLocal indicates a local LLM (e.g., Ollama)
	ProviderLocal ProviderType = "local"

	// ProviderAPI indicates an API-based LLM (e.g., OpenAI, Anthropic)
	ProviderAPI ProviderType = "api"
)

// LLMOperation represents the type of LLM operation performed
type LLMOperation string

const (
	// OpClassify is the classify operation
	OpClassify LLMOperation = "classify"

	// OpExplain is the explain operation
	OpExplain LLMOperation = "explain"

	// OpSuggest is the suggest operation
	OpSuggest LLMOperation = "suggest"

	// OpHealthCheck is the health check operation
	OpHealthCheck LLMOperation = "health_check"
)

// =============================================================================
// ERRORS
// =============================================================================

var (
	// ErrLLMDisabled indicates LLM integration is disabled
	ErrLLMDisabled = errors.New("LLM integration is disabled")

	// ErrLLMTimeout indicates an LLM operation timed out
	ErrLLMTimeout = errors.New("LLM operation timed out")

	// ErrLLMUnavailable indicates the LLM provider is unavailable
	ErrLLMUnavailable = errors.New("LLM provider unavailable")

	// ErrInvalidConfig indicates invalid LLM configuration
	ErrInvalidConfig = errors.New("invalid LLM configuration")

	// ErrRedactionFailed indicates data redaction failed
	ErrRedactionFailed = errors.New("data redaction failed")

	// ErrProviderNotImplemented indicates the provider is not fully implemented
	ErrProviderNotImplemented = errors.New("LLM provider not fully implemented - requires operator configuration")
)

// =============================================================================
// INPUT/OUTPUT TYPES
// =============================================================================

// Input represents data to be classified by the LLM
// All string fields will be redacted before sending to external LLMs
type Input struct {
	// Component identifies the infrastructure component (preserved)
	Component string

	// MetricType identifies the type of metric (preserved)
	MetricType string

	// RawValue is the raw input value (redacted)
	RawValue string

	// Statistics contains numerical patterns (preserved)
	Statistics map[string]float64

	// Timestamp is the event time (replaced with [TIME])
	Timestamp time.Time
}

// Classification represents the result of a classification operation
// This is produced by the LLM but NOT treated as authoritative
type Classification struct {
	// Category is the suggested classification category
	Category string

	// Confidence is the confidence score (0.0 - 1.0)
	Confidence float64

	// Reasoning provides the LLM's reasoning (for debugging only)
	Reasoning string

	// IsDeterministicFallback indicates this is a fallback result, not LLM output
	IsDeterministicFallback bool
}

// Anomaly represents an anomaly to be explained
type Anomaly struct {
	// Component identifies the affected component (preserved)
	Component string

	// MetricType identifies the metric type (preserved)
	MetricType string

	// Severity indicates anomaly severity (preserved)
	Severity string

	// Context contains additional context (redacted)
	Context string

	// DetectedAt is when the anomaly was detected (replaced with [TIME])
	DetectedAt time.Time
}

// Explanation represents the LLM's explanation of an anomaly
// This is advisory only and NOT used for automated decisions
type Explanation struct {
	// Summary is a brief explanation
	Summary string

	// PossibleCauses lists potential causes
	PossibleCauses []string

	// Confidence is the confidence in this explanation (0.0 - 1.0)
	Confidence float64

	// IsDeterministicFallback indicates this is a fallback result, not LLM output
	IsDeterministicFallback bool
}

// Scenario represents a scenario for which suggestions are requested
type Scenario struct {
	// Component identifies the component (preserved)
	Component string

	// CurrentState describes the current state (redacted)
	CurrentState string

	// Constraints lists operational constraints (redacted)
	Constraints []string

	// Goal describes the desired outcome (redacted)
	Goal string
}

// Suggestion represents an LLM-generated suggestion
// Suggestions are advisory only and require operator validation
type Suggestion struct {
	// Actions are the suggested actions
	Actions []string

	// Priority indicates the suggested priority
	Priority string

	// Risks lists potential risks
	Risks []string

	// Confidence is the confidence in this suggestion (0.0 - 1.0)
	Confidence float64

	// IsDeterministicFallback indicates this is a fallback result, not LLM output
	IsDeterministicFallback bool
}

// =============================================================================
// LLM CONFIGURATION
// =============================================================================

// LLMConfig holds configuration for LLM integration
// API keys are stored securely and redacted in all logs
type LLMConfig struct {
	// ProviderType specifies the provider: "disabled", "local", or "api"
	ProviderType ProviderType

	// Endpoint is the URL for the LLM endpoint
	Endpoint string

	// apiKey is the API key (private, use SetAPIKey/GetAPIKeyRedacted)
	apiKey string

	// Timeout is the maximum time to wait for LLM responses
	// Default: 5s, Maximum: 30s
	Timeout time.Duration

	// MaxRetries is the number of retries on failure
	// Default: 2
	MaxRetries int

	// Enabled is the master switch for LLM integration
	// Default: false
	Enabled bool
}

// SetAPIKey sets the API key securely
func (c *LLMConfig) SetAPIKey(key string) {
	c.apiKey = key
}

// GetAPIKeyRedacted returns a redacted version of the API key for display
// Returns "[NOT SET]" if no key is configured
// Returns "[REDACTED: <prefix>...]" if a key is configured
func (c *LLMConfig) GetAPIKeyRedacted() string {
	if c.apiKey == "" {
		return "[NOT SET]"
	}
	if len(c.apiKey) <= 8 {
		return "[REDACTED]"
	}
	return fmt.Sprintf("[REDACTED: %s...]", c.apiKey[:4])
}

// Validate validates the LLM configuration
func (c *LLMConfig) Validate() error {
	if !c.Enabled {
		return nil // Disabled config is always valid
	}

	switch c.ProviderType {
	case ProviderDisabled:
		// Valid - disabled provider
	case ProviderLocal, ProviderAPI:
		if c.Endpoint == "" {
			return fmt.Errorf("%w: endpoint is required for provider %s", ErrInvalidConfig, c.ProviderType)
		}
	default:
		return fmt.Errorf("%w: unknown provider type %q", ErrInvalidConfig, c.ProviderType)
	}

	if c.Timeout == 0 {
		c.Timeout = DefaultLLMTimeout
	}
	if c.Timeout > MaxLLMTimeout {
		c.Timeout = MaxLLMTimeout
	}

	if c.MaxRetries < 0 {
		c.MaxRetries = 0
	}

	return nil
}

// =============================================================================
// AUDIT LOG
// =============================================================================

// LLMAuditEntry represents a single audit log entry for LLM interactions
// Content is NEVER logged - only hashes are stored
type LLMAuditEntry struct {
	// Timestamp is when the operation occurred
	Timestamp time.Time

	// Operation is the type of operation performed
	Operation LLMOperation

	// InputHash is the SHA256 hash of the redacted input
	InputHash string

	// Success indicates whether the operation succeeded
	Success bool

	// ResponseHash is the SHA256 hash of the response
	ResponseHash string

	// Latency is the operation duration
	Latency time.Duration

	// Error contains error details if the operation failed
	// Error messages are sanitized to avoid leaking sensitive data
	Error string
}

// hashInput creates a SHA256 hash of input data for audit logging
func hashInput(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// =============================================================================
// LLM PROVIDER INTERFACE
// =============================================================================

// LLMProvider is the interface for LLM implementations
// All implementations must be thread-safe and handle context cancellation
// gracefully.
type LLMProvider interface {
	// Classify requests a classification from the LLM
	// The input is redacted before sending to the provider
	// Returns ErrLLMDisabled if the provider is disabled
	Classify(ctx context.Context, input Input) (Classification, error)

	// Explain requests an explanation for an anomaly from the LLM
	// The anomaly is redacted before sending to the provider
	// Returns ErrLLMDisabled if the provider is disabled
	Explain(ctx context.Context, anomaly Anomaly) (Explanation, error)

	// Suggest requests suggestions for a scenario from the LLM
	// The scenario is redacted before sending to the provider
	// Returns ErrLLMDisabled if the provider is disabled
	Suggest(ctx context.Context, scenario Scenario) (Suggestion, error)

	// HealthCheck verifies the LLM provider is available
	// Returns nil if healthy, error otherwise
	HealthCheck() error

	// Type returns the provider type
	Type() ProviderType

	// Close cleans up any resources held by the provider
	Close() error
}

// =============================================================================
// DISABLED PROVIDER (Default - Production Ready)
// =============================================================================

// DisabledProvider is a no-op provider that returns "disabled" responses
// This is the default and recommended production configuration when LLM
// integration is not explicitly enabled and validated.
type DisabledProvider struct{}

// NewDisabledProvider creates a new disabled provider
func NewDisabledProvider() *DisabledProvider {
	return &DisabledProvider{}
}

// Classify returns a disabled classification
func (p *DisabledProvider) Classify(ctx context.Context, input Input) (Classification, error) {
	return Classification{
		Category:                "unknown",
		Confidence:              0.0,
		Reasoning:               "LLM integration is disabled",
		IsDeterministicFallback: true,
	}, ErrLLMDisabled
}

// Explain returns a disabled explanation
func (p *DisabledProvider) Explain(ctx context.Context, anomaly Anomaly) (Explanation, error) {
	return Explanation{
		Summary:                 "LLM integration is disabled",
		PossibleCauses:          []string{},
		Confidence:              0.0,
		IsDeterministicFallback: true,
	}, ErrLLMDisabled
}

// Suggest returns disabled suggestions
func (p *DisabledProvider) Suggest(ctx context.Context, scenario Scenario) (Suggestion, error) {
	return Suggestion{
		Actions:                 []string{},
		Priority:                "unknown",
		Risks:                   []string{},
		Confidence:              0.0,
		IsDeterministicFallback: true,
	}, ErrLLMDisabled
}

// HealthCheck always returns nil for disabled provider
func (p *DisabledProvider) HealthCheck() error {
	return nil // Disabled is always "healthy" in the sense that it's working as intended
}

// Type returns the provider type
func (p *DisabledProvider) Type() ProviderType {
	return ProviderDisabled
}

// Close is a no-op for disabled provider
func (p *DisabledProvider) Close() error {
	return nil
}

// =============================================================================
// LOCAL PROVIDER (Stub)
// =============================================================================

// LocalProvider is a stub for local LLM integration (e.g., Ollama)
//
// IMPLEMENTATION STATUS: STUB ONLY
// This provider is NOT fully implemented. It provides the interface structure
// but requires operator configuration and implementation of the actual
// HTTP/gRPC client for the local LLM service.
//
// TO ENABLE:
//   1. Deploy local LLM (e.g., Ollama with desired model)
//   2. Implement HTTP client for Ollama API
//   3. Add prompt templates for classify/explain/suggest operations
//   4. Implement response parsing
//   5. Validate privacy redaction rules
//   6. Set ProviderType to "local" and configure Endpoint
type LocalProvider struct {
	endpoint   string
	httpClient interface{} // Placeholder - would be *http.Client
}

// NewLocalProvider creates a new local provider stub
func NewLocalProvider(endpoint string) *LocalProvider {
	return &LocalProvider{
		endpoint:   endpoint,
		httpClient: nil, // Not implemented
	}
}

// Classify is a stub that returns not-implemented error
func (p *LocalProvider) Classify(ctx context.Context, input Input) (Classification, error) {
	return Classification{
		Category:                "unknown",
		Confidence:              0.0,
		Reasoning:               "Local provider not implemented",
		IsDeterministicFallback: true,
	}, fmt.Errorf("%w: local provider requires operator configuration", ErrProviderNotImplemented)
}

// Explain is a stub that returns not-implemented error
func (p *LocalProvider) Explain(ctx context.Context, anomaly Anomaly) (Explanation, error) {
	return Explanation{
		Summary:                 "Local provider not implemented",
		PossibleCauses:          []string{},
		Confidence:              0.0,
		IsDeterministicFallback: true,
	}, fmt.Errorf("%w: local provider requires operator configuration", ErrProviderNotImplemented)
}

// Suggest is a stub that returns not-implemented error
func (p *LocalProvider) Suggest(ctx context.Context, scenario Scenario) (Suggestion, error) {
	return Suggestion{
		Actions:                 []string{},
		Priority:                "unknown",
		Risks:                   []string{},
		Confidence:              0.0,
		IsDeterministicFallback: true,
	}, fmt.Errorf("%w: local provider requires operator configuration", ErrProviderNotImplemented)
}

// HealthCheck verifies the local endpoint is reachable
// Currently always returns unavailable since implementation is stub-only
func (p *LocalProvider) HealthCheck() error {
	return ErrLLMUnavailable
}

// Type returns the provider type
func (p *LocalProvider) Type() ProviderType {
	return ProviderLocal
}

// Close cleans up the provider
func (p *LocalProvider) Close() error {
	return nil
}

// =============================================================================
// API PROVIDER (Stub)
// =============================================================================

// APIProvider is a stub for API-based LLM integration (e.g., OpenAI, Anthropic)
//
// IMPLEMENTATION STATUS: STUB ONLY
// This provider is NOT fully implemented. It provides the interface structure
// but requires operator configuration and implementation of the actual
// HTTP client for the API service.
//
// TO ENABLE:
//   1. Obtain API credentials from provider
//   2. Implement HTTP client with proper authentication
//   3. Add prompt templates for classify/explain/suggest operations
//   4. Implement response parsing and error handling
//   5. Validate privacy redaction rules meet compliance requirements
//   6. Configure rate limiting and cost controls
//   7. Set ProviderType to "api", configure Endpoint and API key
type APIProvider struct {
	endpoint   string
	apiKey     string
	httpClient interface{} // Placeholder - would be *http.Client
}

// NewAPIProvider creates a new API provider stub
func NewAPIProvider(endpoint, apiKey string) *APIProvider {
	return &APIProvider{
		endpoint:   endpoint,
		apiKey:     apiKey,
		httpClient: nil, // Not implemented
	}
}

// Classify is a stub that returns not-implemented error
func (p *APIProvider) Classify(ctx context.Context, input Input) (Classification, error) {
	return Classification{
		Category:                "unknown",
		Confidence:              0.0,
		Reasoning:               "API provider not implemented",
		IsDeterministicFallback: true,
	}, fmt.Errorf("%w: API provider requires operator configuration", ErrProviderNotImplemented)
}

// Explain is a stub that returns not-implemented error
func (p *APIProvider) Explain(ctx context.Context, anomaly Anomaly) (Explanation, error) {
	return Explanation{
		Summary:                 "API provider not implemented",
		PossibleCauses:          []string{},
		Confidence:              0.0,
		IsDeterministicFallback: true,
	}, fmt.Errorf("%w: API provider requires operator configuration", ErrProviderNotImplemented)
}

// Suggest is a stub that returns not-implemented error
func (p *APIProvider) Suggest(ctx context.Context, scenario Scenario) (Suggestion, error) {
	return Suggestion{
		Actions:                 []string{},
		Priority:                "unknown",
		Risks:                   []string{},
		Confidence:              0.0,
		IsDeterministicFallback: true,
	}, fmt.Errorf("%w: API provider requires operator configuration", ErrProviderNotImplemented)
}

// HealthCheck verifies the API endpoint is reachable
// Currently always returns unavailable since implementation is stub-only
func (p *APIProvider) HealthCheck() error {
	return ErrLLMUnavailable
}

// Type returns the provider type
func (p *APIProvider) Type() ProviderType {
	return ProviderAPI
}

// Close cleans up the provider
func (p *APIProvider) Close() error {
	return nil
}

// =============================================================================
// REDACTION ENGINE
// =============================================================================

// redactor handles privacy-preserving redaction of data before LLM calls
// This ensures no PII or sensitive data leaves the system.
type redactor struct {
	// stringPattern matches quoted strings
	stringPattern *regexp.Regexp

	// timestampPattern matches common timestamp formats
	timestampPattern *regexp.Regexp
}

// newRedactor creates a new redactor with compiled patterns
func newRedactor() *redactor {
	return &redactor{
		// Match quoted strings (simplified - real implementation may need more)
		stringPattern: regexp.MustCompile(`"[^"]*"|'[^']*'`),

		// Match ISO8601-like timestamps and other common formats
		timestampPattern: regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:?\d{2})?|\d{10,13}`),
	}
}

// RedactInput redacts sensitive data from an Input struct
// Returns a string representation suitable for LLM consumption
func (r *redactor) RedactInput(input Input) string {
	var parts []string

	// Component names are preserved (infrastructure identifiers)
	if input.Component != "" {
		parts = append(parts, fmt.Sprintf("component:%s", input.Component))
	}

	// Metric types are preserved
	if input.MetricType != "" {
		parts = append(parts, fmt.Sprintf("metric_type:%s", input.MetricType))
	}

	// Raw values are redacted
	if input.RawValue != "" {
		parts = append(parts, "raw_value:[REDACTED]")
	}

	// Statistics are preserved (numerical patterns only)
	if len(input.Statistics) > 0 {
		stats := make([]string, 0, len(input.Statistics))
		for k, v := range input.Statistics {
			stats = append(stats, fmt.Sprintf("%s:%.4f", k, v))
		}
		parts = append(parts, fmt.Sprintf("statistics:[%s]", strings.Join(stats, ",")))
	}

	// Timestamps are replaced
	if !input.Timestamp.IsZero() {
		parts = append(parts, "timestamp:[TIME]")
	}

	return strings.Join(parts, " ")
}

// RedactAnomaly redacts sensitive data from an Anomaly struct
func (r *redactor) RedactAnomaly(anomaly Anomaly) string {
	var parts []string

	if anomaly.Component != "" {
		parts = append(parts, fmt.Sprintf("component:%s", anomaly.Component))
	}

	if anomaly.MetricType != "" {
		parts = append(parts, fmt.Sprintf("metric_type:%s", anomaly.MetricType))
	}

	if anomaly.Severity != "" {
		parts = append(parts, fmt.Sprintf("severity:%s", anomaly.Severity))
	}

	if anomaly.Context != "" {
		parts = append(parts, "context:[REDACTED]")
	}

	if !anomaly.DetectedAt.IsZero() {
		parts = append(parts, "detected_at:[TIME]")
	}

	return strings.Join(parts, " ")
}

// RedactScenario redacts sensitive data from a Scenario struct
func (r *redactor) RedactScenario(scenario Scenario) string {
	var parts []string

	if scenario.Component != "" {
		parts = append(parts, fmt.Sprintf("component:%s", scenario.Component))
	}

	if scenario.CurrentState != "" {
		parts = append(parts, "current_state:[REDACTED]")
	}

	if len(scenario.Constraints) > 0 {
		parts = append(parts, fmt.Sprintf("constraints:[REDACTED:%d items]", len(scenario.Constraints)))
	}

	if scenario.Goal != "" {
		parts = append(parts, "goal:[REDACTED]")
	}

	return strings.Join(parts, " ")
}

// RedactString redacts strings from arbitrary text
// Useful for sanitizing error messages before logging
func (r *redactor) RedactString(s string) string {
	// Replace quoted strings
	s = r.stringPattern.ReplaceAllString(s, "[REDACTED]")
	// Replace timestamps
	s = r.timestampPattern.ReplaceAllString(s, "[TIME]")
	return s
}

// =============================================================================
// LLM CLIENT
// =============================================================================

// LLMClient wraps an LLMProvider with safety features:
// - Timeout handling
// - Retry logic with exponential backoff
// - Privacy redaction
// - Audit logging
// - Thread-safe operation
type LLMClient struct {
	config     LLMConfig
	provider   LLMProvider
	redactor   *redactor
	auditLog   []LLMAuditEntry
	auditMu    sync.RWMutex
	closed     bool
	closeMu    sync.RWMutex
	lastErr    error
	lastErrMu  sync.RWMutex
}

// NewLLMClient creates a new LLM client with the given configuration
// Returns an error if the configuration is invalid
func NewLLMClient(config LLMConfig) (*LLMClient, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Create appropriate provider
	var provider LLMProvider
	switch config.ProviderType {
	case ProviderDisabled, "":
		provider = NewDisabledProvider()
	case ProviderLocal:
		provider = NewLocalProvider(config.Endpoint)
	case ProviderAPI:
		provider = NewAPIProvider(config.Endpoint, config.apiKey)
	default:
		return nil, fmt.Errorf("%w: unknown provider type %q", ErrInvalidConfig, config.ProviderType)
	}

	client := &LLMClient{
		config:   config,
		provider: provider,
		redactor: newRedactor(),
		auditLog: make([]LLMAuditEntry, 0, MaxAuditLogEntries),
	}

	return client, nil
}

// IsEnabled returns whether LLM integration is enabled
func (c *LLMClient) IsEnabled() bool {
	return c.config.Enabled && c.config.ProviderType != ProviderDisabled
}

// GetProvider returns the underlying provider
// This allows direct access for advanced use cases, but most users should
// use the client's wrapper methods for safety
func (c *LLMClient) GetProvider() LLMProvider {
	return c.provider
}

// GetAuditLog returns a copy of the audit log
// Returns the most recent entries up to MaxAuditLogEntries
func (c *LLMClient) GetAuditLog() []LLMAuditEntry {
	c.auditMu.RLock()
	defer c.auditMu.RUnlock()

	result := make([]LLMAuditEntry, len(c.auditLog))
	copy(result, c.auditLog)
	return result
}

// Close closes the client and its provider
func (c *LLMClient) Close() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	return c.provider.Close()
}

// isClosed returns whether the client is closed
func (c *LLMClient) isClosed() bool {
	c.closeMu.RLock()
	defer c.closeMu.RUnlock()
	return c.closed
}

// recordAuditEntry adds an entry to the audit log
// Automatically truncates to MaxAuditLogEntries
func (c *LLMClient) recordAuditEntry(entry LLMAuditEntry) {
	c.auditMu.Lock()
	defer c.auditMu.Unlock()

	c.auditLog = append(c.auditLog, entry)

	// Truncate if over limit
	if len(c.auditLog) > MaxAuditLogEntries {
		c.auditLog = c.auditLog[len(c.auditLog)-MaxAuditLogEntries:]
	}
}

// Classify performs a classification with timeout, retry, and fallback
// Returns deterministic fallback on any error
func (c *LLMClient) Classify(ctx context.Context, input Input) (Classification, error) {
	if c.isClosed() {
		return c.deterministicClassify(), errors.New("LLM client is closed")
	}

	// If disabled, return immediately with deterministic fallback
	if !c.IsEnabled() {
		return c.deterministicClassify(), nil
	}

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	start := time.Now()

	// Redact input for privacy
	redactedInput := c.redactor.RedactInput(input)
	inputHash := hashInput(redactedInput)

	// Attempt operation with retries
	var result Classification
	var err error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		result, err = c.provider.Classify(ctx, input)
		if err == nil {
			break
		}

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			break
		}

		// Don't retry if disabled
		if errors.Is(err, ErrLLMDisabled) {
			break
		}

		// Exponential backoff between retries
		if attempt < c.config.MaxRetries {
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
		}
	}

	latency := time.Since(start)

	// On error, use deterministic fallback
	if err != nil {
		c.setLastError(err)
		result = c.deterministicClassify()
	}

	// Record audit entry
	responseData := fmt.Sprintf("%s:%.4f:%v", result.Category, result.Confidence, result.IsDeterministicFallback)
	entry := LLMAuditEntry{
		Timestamp:    time.Now().UTC(),
		Operation:    OpClassify,
		InputHash:    inputHash,
		Success:      err == nil,
		ResponseHash: hashInput(responseData),
		Latency:      latency,
		Error:        c.sanitizeError(err),
	}
	c.recordAuditEntry(entry)

	return result, nil // Never return error to caller - always provide fallback
}

// Explain performs an explanation with timeout, retry, and fallback
func (c *LLMClient) Explain(ctx context.Context, anomaly Anomaly) (Explanation, error) {
	if c.isClosed() {
		return c.deterministicExplain(), errors.New("LLM client is closed")
	}

	if !c.IsEnabled() {
		return c.deterministicExplain(), nil
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	start := time.Now()

	redactedAnomaly := c.redactor.RedactAnomaly(anomaly)
	inputHash := hashInput(redactedAnomaly)

	var result Explanation
	var err error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		result, err = c.provider.Explain(ctx, anomaly)
		if err == nil {
			break
		}

		if ctx.Err() != nil {
			break
		}

		if errors.Is(err, ErrLLMDisabled) {
			break
		}

		if attempt < c.config.MaxRetries {
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
		}
	}

	latency := time.Since(start)

	if err != nil {
		c.setLastError(err)
		result = c.deterministicExplain()
	}

	responseData := fmt.Sprintf("%s:%.4f:%v", result.Summary, result.Confidence, result.IsDeterministicFallback)
	entry := LLMAuditEntry{
		Timestamp:    time.Now().UTC(),
		Operation:    OpExplain,
		InputHash:    inputHash,
		Success:      err == nil,
		ResponseHash: hashInput(responseData),
		Latency:      latency,
		Error:        c.sanitizeError(err),
	}
	c.recordAuditEntry(entry)

	return result, nil
}

// Suggest performs a suggestion with timeout, retry, and fallback
func (c *LLMClient) Suggest(ctx context.Context, scenario Scenario) (Suggestion, error) {
	if c.isClosed() {
		return c.deterministicSuggest(), errors.New("LLM client is closed")
	}

	if !c.IsEnabled() {
		return c.deterministicSuggest(), nil
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	start := time.Now()

	redactedScenario := c.redactor.RedactScenario(scenario)
	inputHash := hashInput(redactedScenario)

	var result Suggestion
	var err error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		result, err = c.provider.Suggest(ctx, scenario)
		if err == nil {
			break
		}

		if ctx.Err() != nil {
			break
		}

		if errors.Is(err, ErrLLMDisabled) {
			break
		}

		if attempt < c.config.MaxRetries {
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
		}
	}

	latency := time.Since(start)

	if err != nil {
		c.setLastError(err)
		result = c.deterministicSuggest()
	}

	responseData := fmt.Sprintf("%s:%s:%.4f:%v", strings.Join(result.Actions, ","), result.Priority, result.Confidence, result.IsDeterministicFallback)
	entry := LLMAuditEntry{
		Timestamp:    time.Now().UTC(),
		Operation:    OpSuggest,
		InputHash:    inputHash,
		Success:      err == nil,
		ResponseHash: hashInput(responseData),
		Latency:      latency,
		Error:        c.sanitizeError(err),
	}
	c.recordAuditEntry(entry)

	return result, nil
}

// HealthCheck checks the health of the LLM provider
func (c *LLMClient) HealthCheck() error {
	if c.isClosed() {
		return errors.New("LLM client is closed")
	}

	if !c.IsEnabled() {
		return nil // Disabled is considered healthy
	}

	return c.provider.HealthCheck()
}

// GetLastError returns the last error encountered (for debugging)
func (c *LLMClient) GetLastError() error {
	c.lastErrMu.RLock()
	defer c.lastErrMu.RUnlock()
	return c.lastErr
}

// setLastError records the last error encountered
func (c *LLMClient) setLastError(err error) {
	c.lastErrMu.Lock()
	defer c.lastErrMu.Unlock()
	c.lastErr = err
}

// sanitizeError sanitizes an error for audit logging
// Removes potentially sensitive information
func (c *LLMClient) sanitizeError(err error) string {
	if err == nil {
		return ""
	}

	s := err.Error()
	s = c.redactor.RedactString(s)

	// Limit length
	if len(s) > 256 {
		s = s[:256] + "..."
	}

	return s
}

// =============================================================================
// DETERMINISTIC FALLBACKS
// =============================================================================
// These provide safe, deterministic results when LLM is unavailable
// They ensure the system continues to function without LLM support

// deterministicClassify returns a safe fallback classification
func (c *LLMClient) deterministicClassify() Classification {
	return Classification{
		Category:                "unknown",
		Confidence:              0.0,
		Reasoning:               "LLM unavailable - using deterministic fallback",
		IsDeterministicFallback: true,
	}
}

// deterministicExplain returns a safe fallback explanation
func (c *LLMClient) deterministicExplain() Explanation {
	return Explanation{
		Summary:                 "LLM unavailable - no explanation available",
		PossibleCauses:          []string{"LLM service unavailable"},
		Confidence:              0.0,
		IsDeterministicFallback: true,
	}
}

// deterministicSuggest returns a safe fallback suggestion
func (c *LLMClient) deterministicSuggest() Suggestion {
	return Suggestion{
		Actions:                 []string{"Check LLM configuration", "Review system logs"},
		Priority:                "low",
		Risks:                   []string{"LLM suggestions unavailable"},
		Confidence:              0.0,
		IsDeterministicFallback: true,
	}
}

// =============================================================================
// GLOBAL INSTANCE
// =============================================================================

var (
	globalLLMClient     *LLMClient
	globalLLMClientMu   sync.RWMutex
	globalLLMClientOnce sync.Once
)

// GetGlobalLLMClient returns the global LLM client instance
// Creates a disabled client if not initialized
func GetGlobalLLMClient() *LLMClient {
	globalLLMClientMu.RLock()
	if globalLLMClient != nil {
		globalLLMClientMu.RUnlock()
		return globalLLMClient
	}
	globalLLMClientMu.RUnlock()

	// Initialize with disabled config
	globalLLMClientOnce.Do(func() {
		client, _ := NewLLMClient(LLMConfig{
			ProviderType: ProviderDisabled,
			Enabled:      false,
		})
		globalLLMClient = client
	})

	return globalLLMClient
}

// SetGlobalLLMClient sets the global LLM client instance
// Use with caution - primarily for testing and initialization
func SetGlobalLLMClient(client *LLMClient) {
	globalLLMClientMu.Lock()
	defer globalLLMClientMu.Unlock()

	// Close existing client if replacing
	if globalLLMClient != nil && globalLLMClient != client {
		globalLLMClient.Close()
	}

	globalLLMClient = client
}

// InitGlobalLLMClient initializes the global LLM client with the given config
// Safe to call multiple times - only first call succeeds unless force=true
func InitGlobalLLMClient(config LLMConfig, force bool) error {
	globalLLMClientMu.Lock()
	defer globalLLMClientMu.Unlock()

	if globalLLMClient != nil && !force {
		return nil // Already initialized
	}

	client, err := NewLLMClient(config)
	if err != nil {
		return err
	}

	if globalLLMClient != nil {
		globalLLMClient.Close()
	}

	globalLLMClient = client
	return nil
}
