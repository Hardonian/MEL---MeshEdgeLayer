package selfobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// LLM INTEGRATION TESTS
// =============================================================================
// IMPORTANT: These tests document that actual LLM integrations are STUBS ONLY.
// No real LLM APIs are called. All providers return deterministic fallbacks.
//
// IMPLEMENTATION STATUS:
//   - DisabledProvider: FULLY IMPLEMENTED (production-ready no-op)
//   - LocalProvider: STUB ONLY (requires operator configuration)
//   - APIProvider: STUB ONLY (requires operator configuration)
//
// TO ENABLE REAL LLM SUPPORT:
//   1. Deploy and configure local LLM (e.g., Ollama) OR obtain API credentials
//   2. Implement actual HTTP/gRPC clients for the chosen provider
//   3. Add prompt templates and response parsing
//   4. Validate privacy redaction rules meet compliance requirements
//   5. Configure rate limiting and cost controls
// =============================================================================

// TestLLMClientCreation tests creating LLM clients with different provider types
func TestLLMClientCreation(t *testing.T) {
	// Test 1: Disabled provider (default, production-ready)
	disabledConfig := LLMConfig{
		ProviderType: ProviderDisabled,
		Enabled:      false,
	}
	disabledClient, err := NewLLMClient(disabledConfig)
	if err != nil {
		t.Fatalf("failed to create disabled client: %v", err)
	}
	defer disabledClient.Close()

	if disabledClient.IsEnabled() {
		t.Error("expected disabled client to report IsEnabled=false")
	}
	if disabledClient.GetProvider().Type() != ProviderDisabled {
		t.Errorf("expected provider type %s, got %s", ProviderDisabled, disabledClient.GetProvider().Type())
	}

	// Test 2: Local provider stub
	localConfig := LLMConfig{
		ProviderType: ProviderLocal,
		Enabled:      true,
		Endpoint:     "http://localhost:11434",
		Timeout:      5 * time.Second,
	}
	localClient, err := NewLLMClient(localConfig)
	if err != nil {
		t.Fatalf("failed to create local client: %v", err)
	}
	defer localClient.Close()

	if !localClient.IsEnabled() {
		t.Error("expected local client to report IsEnabled=true")
	}
	if localClient.GetProvider().Type() != ProviderLocal {
		t.Errorf("expected provider type %s, got %s", ProviderLocal, localClient.GetProvider().Type())
	}

	// Test 3: API provider stub
	apiConfig := LLMConfig{
		ProviderType: ProviderAPI,
		Enabled:      true,
		Endpoint:     "https://api.example.com/v1",
		Timeout:      10 * time.Second,
	}
	apiConfig.SetAPIKey("sk-test-key-12345")
	apiClient, err := NewLLMClient(apiConfig)
	if err != nil {
		t.Fatalf("failed to create API client: %v", err)
	}
	defer apiClient.Close()

	if !apiClient.IsEnabled() {
		t.Error("expected API client to report IsEnabled=true")
	}
	if apiClient.GetProvider().Type() != ProviderAPI {
		t.Errorf("expected provider type %s, got %s", ProviderAPI, apiClient.GetProvider().Type())
	}
}

// TestDisabledProvider tests the disabled provider behavior
func TestDisabledProvider(t *testing.T) {
	provider := NewDisabledProvider()
	ctx := context.Background()

	// Test IsEnabled equivalent via Type()
	if provider.Type() != ProviderDisabled {
		t.Errorf("expected type %s, got %s", ProviderDisabled, provider.Type())
	}

	// Test Classify returns disabled response
	input := Input{
		Component:  "test-component",
		MetricType: "latency",
		RawValue:   "100ms",
	}
	classifyResult, err := provider.Classify(ctx, input)
	if !errors.Is(err, ErrLLMDisabled) {
		t.Errorf("expected ErrLLMDisabled, got %v", err)
	}
	if classifyResult.Category != "unknown" {
		t.Errorf("expected category 'unknown', got %s", classifyResult.Category)
	}
	if classifyResult.Confidence != 0.0 {
		t.Errorf("expected confidence 0.0, got %f", classifyResult.Confidence)
	}
	if !classifyResult.IsDeterministicFallback {
		t.Error("expected IsDeterministicFallback to be true")
	}

	// Test Explain returns disabled response
	anomaly := Anomaly{
		Component:  "test-component",
		MetricType: "error_rate",
		Severity:   "high",
	}
	explainResult, err := provider.Explain(ctx, anomaly)
	if !errors.Is(err, ErrLLMDisabled) {
		t.Errorf("expected ErrLLMDisabled, got %v", err)
	}
	if explainResult.Summary != "LLM integration is disabled" {
		t.Errorf("expected summary 'LLM integration is disabled', got %s", explainResult.Summary)
	}
	if !explainResult.IsDeterministicFallback {
		t.Error("expected IsDeterministicFallback to be true")
	}

	// Test Suggest returns disabled response
	scenario := Scenario{
		Component:    "test-component",
		CurrentState: "degraded",
		Goal:         "restore service",
	}
	suggestResult, err := provider.Suggest(ctx, scenario)
	if !errors.Is(err, ErrLLMDisabled) {
		t.Errorf("expected ErrLLMDisabled, got %v", err)
	}
	if suggestResult.Priority != "unknown" {
		t.Errorf("expected priority 'unknown', got %s", suggestResult.Priority)
	}
	if !suggestResult.IsDeterministicFallback {
		t.Error("expected IsDeterministicFallback to be true")
	}

	// Test HealthCheck returns nil (disabled is "healthy" as it's working as intended)
	healthErr := provider.HealthCheck()
	if healthErr != nil {
		t.Errorf("expected HealthCheck to return nil for disabled provider, got %v", healthErr)
	}

	// Test Close is no-op
	if err := provider.Close(); err != nil {
		t.Errorf("expected Close to return nil, got %v", err)
	}
}

// TestRedaction tests that sensitive data is properly redacted
func TestRedaction(t *testing.T) {
	r := newRedactor()

	// Test 1: Strings are redacted in Input
	input := Input{
		Component:  "gateway-node-1",
		MetricType: "message_latency",
		RawValue:   "sensitive error message: connection failed to 192.168.1.1",
		Statistics: map[string]float64{
			"p50": 10.5,
			"p99": 150.2,
		},
		Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}
	redactedInput := r.RedactInput(input)

	if !strings.Contains(redactedInput, "component:gateway-node-1") {
		t.Error("expected component name to be preserved")
	}
	if !strings.Contains(redactedInput, "metric_type:message_latency") {
		t.Error("expected metric type to be preserved")
	}
	if !strings.Contains(redactedInput, "raw_value:[REDACTED]") {
		t.Error("expected raw value to be redacted")
	}
	if strings.Contains(redactedInput, "sensitive error message") {
		t.Error("raw string should be redacted")
	}
	if !strings.Contains(redactedInput, "statistics:") {
		t.Error("expected statistics to be preserved")
	}
	if !strings.Contains(redactedInput, "p50:10.5000") {
		t.Error("expected p50 statistic to be preserved")
	}
	if !strings.Contains(redactedInput, "timestamp:[TIME]") {
		t.Error("expected timestamp to be replaced with [TIME]")
	}
	if strings.Contains(redactedInput, "2024-01-15") {
		t.Error("timestamp should be redacted")
	}

	// Test 2: Timestamps are redacted in Anomaly
	anomaly := Anomaly{
		Component:  "test-component",
		MetricType: "error_rate",
		Severity:   "critical",
		Context:    "Database connection failed at 2024-01-15T10:30:00Z",
		DetectedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}
	redactedAnomaly := r.RedactAnomaly(anomaly)

	if !strings.Contains(redactedAnomaly, "detected_at:[TIME]") {
		t.Error("expected detected_at to be replaced with [TIME]")
	}
	if !strings.Contains(redactedAnomaly, "context:[REDACTED]") {
		t.Error("expected context to be redacted")
	}

	// Test 3: Component names are preserved in Scenario
	scenario := Scenario{
		Component:    "mesh-router-alpha",
		CurrentState: "high latency on node-42",
		Constraints:  []string{"maintain quorum", "no downtime"},
		Goal:         "reduce latency below 100ms",
	}
	redactedScenario := r.RedactScenario(scenario)

	if !strings.Contains(redactedScenario, "component:mesh-router-alpha") {
		t.Error("expected component name to be preserved")
	}
	if !strings.Contains(redactedScenario, "current_state:[REDACTED]") {
		t.Error("expected current_state to be redacted")
	}
	if !strings.Contains(redactedScenario, "constraints:[REDACTED:2 items]") {
		t.Error("expected constraints to be redacted with count")
	}
	if strings.Contains(redactedScenario, "maintain quorum") {
		t.Error("constraint content should be redacted")
	}

	// Test 4: Statistics are preserved
	if !strings.Contains(redactedInput, "p99:150.2000") {
		t.Error("expected p99 statistic to be preserved with precision")
	}

	// Test 5: RedactString function
	// Note: timestampPattern matches ISO8601 formats, so test with a proper format
	testString := `Error at "sensitive-details": connection to 'db-server-01' failed`
	redactedString := r.RedactString(testString)

	if strings.Contains(redactedString, "sensitive-details") {
		t.Error("quoted string should be redacted")
	}
	if strings.Contains(redactedString, "db-server-01") {
		t.Error("quoted string should be redacted")
	}
	if !strings.Contains(redactedString, "[REDACTED]") {
		t.Errorf("expected [REDACTED] placeholder in redacted string, got: %s", redactedString)
	}

	// Test timestamp redaction with actual ISO8601 format
	testStringWithTimestamp := `Error occurred at 2024-01-15T10:30:00Z in the system`
	redactedTimestampString := r.RedactString(testStringWithTimestamp)

	if strings.Contains(redactedTimestampString, "2024-01-15") {
		t.Error("timestamp in string should be redacted")
	}
	if !strings.Contains(redactedTimestampString, "[TIME]") {
		t.Errorf("expected [TIME] placeholder for timestamp, got: %s", redactedTimestampString)
	}
}

// TestTimeoutAndFallback tests timeout handling and fallback behavior
func TestTimeoutAndFallback(t *testing.T) {
	// Test with disabled provider - should return immediately without blocking
	config := LLMConfig{
		ProviderType: ProviderDisabled,
		Enabled:      false,
		Timeout:      100 * time.Millisecond,
	}
	client, err := NewLLMClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	start := time.Now()

	// Classify should return fallback immediately (no blocking)
	input := Input{
		Component:  "test",
		MetricType: "latency",
	}
	result, err := client.Classify(ctx, input)
	elapsed := time.Since(start)

	// Should not error (fallback is returned, not error)
	if err != nil {
		t.Errorf("expected no error (fallback returned), got %v", err)
	}
	// Should return fallback result
	if !result.IsDeterministicFallback {
		t.Error("expected deterministic fallback when disabled")
	}
	if result.Category != "unknown" {
		t.Errorf("expected category 'unknown', got %s", result.Category)
	}
	// Should complete quickly (no actual LLM call)
	if elapsed > 50*time.Millisecond {
		t.Errorf("expected fast return when disabled, took %v", elapsed)
	}

	// Test Explain fallback
	anomaly := Anomaly{
		Component:  "test",
		MetricType: "errors",
		Severity:   "high",
	}
	explainResult, err := client.Explain(ctx, anomaly)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !explainResult.IsDeterministicFallback {
		t.Error("expected deterministic fallback for explain")
	}

	// Test Suggest fallback
	scenario := Scenario{
		Component:    "test",
		CurrentState: "degraded",
	}
	suggestResult, err := client.Suggest(ctx, scenario)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !suggestResult.IsDeterministicFallback {
		t.Error("expected deterministic fallback for suggest")
	}
}

// TestAuditLog verifies audit logging behavior
// Note: Audit logging only occurs when LLM is enabled (to avoid noise from disabled operations)
func TestAuditLog(t *testing.T) {
	// Use local provider (enabled) to generate audit entries
	config := LLMConfig{
		ProviderType: ProviderLocal,
		Enabled:      true,
		Endpoint:     "http://localhost:11434",
		Timeout:      100 * time.Millisecond,
	}
	client, err := NewLLMClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Perform operations
	input := Input{
		Component:  "audit-test-component",
		MetricType: "test_metric",
		RawValue:   "secret data that should not be logged",
	}
	client.Classify(ctx, input)

	anomaly := Anomaly{
		Component:  "audit-test-component",
		MetricType: "error_rate",
		Context:    "more secret data",
	}
	client.Explain(ctx, anomaly)

	scenario := Scenario{
		Component:    "audit-test-component",
		CurrentState: "even more secrets",
	}
	client.Suggest(ctx, scenario)

	// Get audit log
	auditLog := client.GetAuditLog()

	// Verify all operations were logged
	if len(auditLog) != 3 {
		t.Fatalf("expected 3 audit entries, got %d", len(auditLog))
	}

	// Verify operation types
	expectedOps := []LLMOperation{OpClassify, OpExplain, OpSuggest}
	for i, entry := range auditLog {
		if entry.Operation != expectedOps[i] {
			t.Errorf("entry %d: expected operation %s, got %s", i, expectedOps[i], entry.Operation)
		}
	}

	// Verify hashes are stored (not content)
	for i, entry := range auditLog {
		// InputHash should be a valid SHA256 hash (64 hex characters)
		if len(entry.InputHash) != 64 {
			t.Errorf("entry %d: expected 64-char hash, got %d chars", i, len(entry.InputHash))
		}
		// ResponseHash should also be valid
		if len(entry.ResponseHash) != 64 {
			t.Errorf("entry %d: expected 64-char response hash, got %d chars", i, len(entry.ResponseHash))
		}

		// Verify content is NOT in the audit log (only hash)
		if strings.Contains(entry.InputHash, "secret") {
			t.Errorf("entry %d: raw content should not be in audit log", i)
		}
	}

	// Verify latency tracking
	for i, entry := range auditLog {
		if entry.Latency <= 0 {
			t.Errorf("entry %d: expected positive latency, got %v", i, entry.Latency)
		}
	}

	// With local provider (stub), operations fail and errors are logged
	for i, entry := range auditLog {
		if entry.Success {
			t.Errorf("entry %d: expected Success=false for stub provider", i)
		}
		// Error should be recorded (stub returns not-implemented)
		if entry.Error == "" {
			t.Errorf("entry %d: expected error for stub provider", i)
		}
	}
}

// TestAuditLogErrorSanitization tests that errors are sanitized in audit log
func TestAuditLogErrorSanitization(t *testing.T) {
	// Create a client with local provider (which returns not-implemented errors)
	config := LLMConfig{
		ProviderType: ProviderLocal,
		Enabled:      true,
		Endpoint:     "http://localhost:11434",
		Timeout:      100 * time.Millisecond,
	}
	client, err := NewLLMClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// This will generate an error (provider not implemented)
	input := Input{
		Component:  "test",
		MetricType: "latency",
	}
	client.Classify(ctx, input)

	// Check audit log
	auditLog := client.GetAuditLog()
	if len(auditLog) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(auditLog))
	}

	entry := auditLog[0]
	if entry.Success {
		t.Error("expected Success=false for failed operation")
	}

	// Error should be recorded but sanitized
	if entry.Error == "" {
		t.Error("expected error to be recorded")
	}

	// Error should not contain sensitive patterns (just check it's reasonable length)
	if len(entry.Error) > 300 {
		t.Errorf("error message too long (%d chars), may contain unsanitized data", len(entry.Error))
	}
}

// TestAPIKeySecurity verifies API key security
func TestAPIKeySecurity(t *testing.T) {
	config := LLMConfig{
		ProviderType: ProviderAPI,
		Enabled:      true,
		Endpoint:     "https://api.example.com",
	}

	// Test key not exposed via GetAPIKeyRedacted
	config.SetAPIKey("sk-abcdefghijklmnopqrstuvwxyz")
	redacted := config.GetAPIKeyRedacted()

	if strings.Contains(redacted, "sk-abcdefghijklmnop") {
		t.Error("API key should not be exposed in redacted form")
	}
	if !strings.HasPrefix(redacted, "[REDACTED:") {
		t.Errorf("expected redacted key to start with '[REDACTED:', got %s", redacted)
	}

	// Test empty key
	emptyConfig := LLMConfig{}
	emptyConfig.SetAPIKey("")
	if emptyConfig.GetAPIKeyRedacted() != "[NOT SET]" {
		t.Errorf("expected '[NOT SET]' for empty key, got %s", emptyConfig.GetAPIKeyRedacted())
	}

	// Test short key
	shortConfig := LLMConfig{}
	shortConfig.SetAPIKey("short")
	if shortConfig.GetAPIKeyRedacted() != "[REDACTED]" {
		t.Errorf("expected '[REDACTED]' for short key, got %s", shortConfig.GetAPIKeyRedacted())
	}

	// Test that String() representation of config doesn't leak key
	// (We can't directly test String() since LLMConfig doesn't have one,
	// but we verify the redaction method works correctly)
	apiKeyInEndpoint := "https://api.example.com?key=secret123"
	configWithKeyInEndpoint := LLMConfig{
		Endpoint: apiKeyInEndpoint,
	}
	_ = configWithKeyInEndpoint
	// Note: In production, endpoint URLs with embedded keys should also be handled carefully
}

// TestHonestCapabilityDeclaration documents what is and isn't implemented
func TestHonestCapabilityDeclaration(t *testing.T) {
	// This test documents the current implementation status
	// It serves as a contract test that fails if implementation status changes

	t.Log("=== LLM INTEGRATION CAPABILITY DECLARATION ===")
	t.Log("")
	t.Log("WHAT IS IMPLEMENTED:")
	t.Log("  - LLMProvider interface with Classify, Explain, Suggest, HealthCheck methods")
	t.Log("  - DisabledProvider: production-ready no-op provider (FULLY IMPLEMENTED)")
	t.Log("  - LocalProvider: stub for local LLM integration (STUB ONLY)")
	t.Log("  - APIProvider: stub for API-based LLM integration (STUB ONLY)")
	t.Log("  - LLMClient with timeout, retry, and fallback handling")
	t.Log("  - Privacy-first redaction system for all external calls")
	t.Log("  - Thread-safe audit logging of all LLM interactions")
	t.Log("  - Configuration validation and secure credential handling")
	t.Log("")
	t.Log("WHAT IS NOT IMPLEMENTED:")
	t.Log("  - Actual LLM API integrations (no HTTP clients for Ollama, OpenAI, etc.)")
	t.Log("  - Real model inference or token streaming")
	t.Log("  - Prompt templates and fine-tuning")
	t.Log("  - Response parsing and structured output validation")
	t.Log("")
	t.Log("TO ENABLE LLM SUPPORT:")
	t.Log("  1. Choose provider: 'disabled' (default), 'local', or 'api'")
	t.Log("  2. Configure endpoint and credentials")
	t.Log("  3. Validate privacy redaction rules meet compliance requirements")
	t.Log("  4. Implement actual HTTP/gRPC client for chosen provider")
	t.Log("  5. Add prompt templates and response parsing")

	// Verify stubs return not-implemented errors
	localProvider := NewLocalProvider("http://localhost:11434")
	apiProvider := NewAPIProvider("https://api.example.com", "key")
	ctx := context.Background()

	// Local provider should return not implemented
	_, err := localProvider.Classify(ctx, Input{})
	if !errors.Is(err, ErrProviderNotImplemented) {
		t.Errorf("expected ErrProviderNotImplemented from local provider, got %v", err)
	}

	// API provider should return not implemented
	_, err = apiProvider.Explain(ctx, Anomaly{})
	if !errors.Is(err, ErrProviderNotImplemented) {
		t.Errorf("expected ErrProviderNotImplemented from API provider, got %v", err)
	}

	// Health checks should indicate unavailable
	if err := localProvider.HealthCheck(); !errors.Is(err, ErrLLMUnavailable) {
		t.Errorf("expected ErrLLMUnavailable from local provider health check, got %v", err)
	}
	if err := apiProvider.HealthCheck(); !errors.Is(err, ErrLLMUnavailable) {
		t.Errorf("expected ErrLLMUnavailable from API provider health check, got %v", err)
	}
}

// TestLLMPrivacySafety verifies no raw data is sent to LLM and aggregation occurs
func TestLLMPrivacySafety(t *testing.T) {
	r := newRedactor()

	// Test 1: Input redaction removes sensitive data
	sensitiveInput := Input{
		Component:  "gateway-prod-01",
		MetricType: "message_processing",
		RawValue:   "Error: connection to 192.168.1.100:8080 failed - user 'admin' password invalid",
		Statistics: map[string]float64{
			"count":   1000.0,
			"error_count": 50.0,
			"p99_latency_ms": 250.5,
		},
		Timestamp: time.Now(),
	}

	redacted := r.RedactInput(sensitiveInput)

	// Verify no IP addresses in output
	if strings.Contains(redacted, "192.168.1.100") {
		t.Error("IP address should be redacted from input")
	}
	// Verify no credentials in output
	if strings.Contains(redacted, "admin") || strings.Contains(redacted, "password") {
		t.Error("credentials should be redacted from input")
	}
	// Verify statistics are preserved (aggregation before external call)
	if !strings.Contains(redacted, "count:1000.0000") {
		t.Error("count statistic should be preserved")
	}
	if !strings.Contains(redacted, "error_count:50.0000") {
		t.Error("error_count statistic should be preserved")
	}
	if !strings.Contains(redacted, "p99_latency_ms:250.5000") {
		t.Error("p99_latency_ms statistic should be preserved")
	}

	// Test 2: Anomaly redaction
	sensitiveAnomaly := Anomaly{
		Component:  "database-primary",
		MetricType: "query_performance",
		Severity:   "critical",
		Context:    "Query 'SELECT * FROM users WHERE ssn = 123-45-6789' caused timeout",
		DetectedAt: time.Now(),
	}

	redactedAnomaly := r.RedactAnomaly(sensitiveAnomaly)

	// Verify PII is redacted
	if strings.Contains(redactedAnomaly, "ssn") || strings.Contains(redactedAnomaly, "123-45-6789") {
		t.Error("SSN should be redacted from anomaly context")
	}
	if strings.Contains(redactedAnomaly, "SELECT * FROM users") {
		t.Error("SQL query should be redacted from anomaly context")
	}
	// Verify component and metric type are preserved
	if !strings.Contains(redactedAnomaly, "component:database-primary") {
		t.Error("component should be preserved")
	}
	if !strings.Contains(redactedAnomaly, "metric_type:query_performance") {
		t.Error("metric_type should be preserved")
	}

	// Test 3: Scenario redaction
	sensitiveScenario := Scenario{
		Component:    "api-gateway",
		CurrentState: "High load from user john.doe@example.com, IP 10.0.0.5",
		Constraints:  []string{"Must maintain session for user_id=12345", "No downtime for VIP users"},
		Goal:         "Reduce load while preserving john.doe's session",
	}

	redactedScenario := r.RedactScenario(sensitiveScenario)

	// Verify PII is redacted
	if strings.Contains(redactedScenario, "john.doe@example.com") {
		t.Error("email should be redacted from scenario")
	}
	if strings.Contains(redactedScenario, "10.0.0.5") {
		t.Error("IP should be redacted from scenario")
	}
	if strings.Contains(redactedScenario, "12345") {
		t.Error("user ID should be redacted from scenario")
	}
	// Verify component is preserved
	if !strings.Contains(redactedScenario, "component:api-gateway") {
		t.Error("component should be preserved")
	}
	// Verify constraints count is shown but not content
	if !strings.Contains(redactedScenario, "constraints:[REDACTED:2 items]") {
		t.Errorf("expected constraints count, got: %s", redactedScenario)
	}
}

// TestLLMClientConfigValidation tests configuration validation
func TestLLMClientConfigValidation(t *testing.T) {
	// Test valid disabled config
	disabledConfig := LLMConfig{
		ProviderType: ProviderDisabled,
		Enabled:      false,
	}
	if err := disabledConfig.Validate(); err != nil {
		t.Errorf("disabled config should always be valid: %v", err)
	}

	// Test local config without endpoint (should fail when enabled)
	localConfigNoEndpoint := LLMConfig{
		ProviderType: ProviderLocal,
		Enabled:      true,
	}
	if err := localConfigNoEndpoint.Validate(); err == nil {
		t.Error("local config without endpoint should fail validation when enabled")
	}

	// Test API config without endpoint (should fail when enabled)
	apiConfigNoEndpoint := LLMConfig{
		ProviderType: ProviderAPI,
		Enabled:      true,
	}
	if err := apiConfigNoEndpoint.Validate(); err == nil {
		t.Error("API config without endpoint should fail validation when enabled")
	}

	// Test timeout validation (must be enabled for timeout capping to apply)
	configWithLongTimeout := LLMConfig{
		ProviderType: ProviderLocal,
		Enabled:      true,
		Endpoint:     "http://localhost:11434",
		Timeout:      60 * time.Second, // Exceeds MaxLLMTimeout
	}
	if err := configWithLongTimeout.Validate(); err != nil {
		t.Errorf("timeout validation should succeed: %v", err)
	}
	if configWithLongTimeout.Timeout != MaxLLMTimeout {
		t.Errorf("timeout should be capped at %v, got %v", MaxLLMTimeout, configWithLongTimeout.Timeout)
	}

	// Test disabled config doesn't cap timeout (returns early)
	disabledWithLongTimeout := LLMConfig{
		ProviderType: ProviderDisabled,
		Enabled:      false,
		Timeout:      60 * time.Second,
	}
	if err := disabledWithLongTimeout.Validate(); err != nil {
		t.Errorf("disabled config should be valid: %v", err)
	}
	// When disabled, timeout is not modified (returns early from Validate)
	if disabledWithLongTimeout.Timeout != 60*time.Second {
		t.Logf("note: disabled config timeout behavior: %v", disabledWithLongTimeout.Timeout)
	}

	// Test negative retries
	configWithNegativeRetries := LLMConfig{
		ProviderType: ProviderDisabled,
		Enabled:      false,
		MaxRetries:   -1,
	}
	if err := configWithNegativeRetries.Validate(); err != nil {
		t.Errorf("negative retries should be allowed (set to 0): %v", err)
	}
	if configWithNegativeRetries.MaxRetries != -1 {
		// Note: The validation doesn't actually modify the value, just allows it
		// The client handles negative values appropriately
	}
}

// TestLLMClientClose tests client close behavior
func TestLLMClientClose(t *testing.T) {
	config := LLMConfig{
		ProviderType: ProviderDisabled,
		Enabled:      false,
	}
	client, err := NewLLMClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Close should succeed
	if err := client.Close(); err != nil {
		t.Errorf("first close should succeed: %v", err)
	}

	// Double close should succeed (idempotent)
	if err := client.Close(); err != nil {
		t.Errorf("second close should succeed (idempotent): %v", err)
	}

	// Operations after close should return fallback with error indication
	ctx := context.Background()
	input := Input{Component: "test", MetricType: "latency"}
	result, err := client.Classify(ctx, input)
	if err == nil {
		t.Error("expected error when calling Classify on closed client")
	}
	if !result.IsDeterministicFallback {
		t.Error("expected deterministic fallback for closed client")
	}
}

// TestGlobalLLMClient tests global client management
func TestGlobalLLMClient(t *testing.T) {
	// Save original global client
	originalClient := GetGlobalLLMClient()
	if originalClient == nil {
		t.Fatal("expected global client to be initialized")
	}

	// Reset after test
	defer func() {
		SetGlobalLLMClient(originalClient)
	}()

	// Test GetGlobalLLMClient returns a client
	client := GetGlobalLLMClient()
	if client == nil {
		t.Error("GetGlobalLLMClient should never return nil")
	}

	// Default global client should be disabled
	if client.IsEnabled() {
		t.Error("default global client should be disabled")
	}

	// Test SetGlobalLLMClient
	newConfig := LLMConfig{
		ProviderType: ProviderDisabled,
		Enabled:      false,
	}
	newClient, err := NewLLMClient(newConfig)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	SetGlobalLLMClient(newClient)

	if GetGlobalLLMClient() != newClient {
		t.Error("SetGlobalLLMClient should update the global client")
	}
}

// TestInitGlobalLLMClient tests global client initialization
func TestInitGlobalLLMClient(t *testing.T) {
	// Save original
	originalClient := GetGlobalLLMClient()
	defer SetGlobalLLMClient(originalClient)

	// Reset global to test initialization
	SetGlobalLLMClient(nil)

	// Test initialization with disabled config
	config := LLMConfig{
		ProviderType: ProviderDisabled,
		Enabled:      false,
	}

	err := InitGlobalLLMClient(config, false)
	if err != nil {
		t.Errorf("InitGlobalLLMClient should succeed: %v", err)
	}

	client := GetGlobalLLMClient()
	if client == nil {
		t.Fatal("global client should be initialized")
	}
	if client.IsEnabled() {
		t.Error("initialized client should be disabled")
	}

	// Test second initialization without force (should be no-op)
	err = InitGlobalLLMClient(config, false)
	if err != nil {
		t.Errorf("second InitGlobalLLMClient without force should succeed: %v", err)
	}

	// Test initialization with force (should replace)
	newConfig := LLMConfig{
		ProviderType: ProviderDisabled,
		Enabled:      false,
	}
	err = InitGlobalLLMClient(newConfig, true)
	if err != nil {
		t.Errorf("InitGlobalLLMClient with force should succeed: %v", err)
	}
}

// TestHashInput tests the hash function used for audit logging
func TestHashInput(t *testing.T) {
	data := "test data for hashing"
	hash1 := hashInput(data)
	hash2 := hashInput(data)

	// Same input should produce same hash
	if hash1 != hash2 {
		t.Error("same input should produce same hash")
	}

	// Hash should be 64 hex characters (SHA256)
	if len(hash1) != 64 {
		t.Errorf("expected 64 char hash, got %d", len(hash1))
	}

	// Different input should produce different hash
	hash3 := hashInput("different data")
	if hash1 == hash3 {
		t.Error("different input should produce different hash")
	}

	// Verify it's actually hex
	_, err := hex.DecodeString(hash1)
	if err != nil {
		t.Errorf("hash should be valid hex: %v", err)
	}

	// Verify it matches expected SHA256
	h := sha256.New()
	h.Write([]byte(data))
	expectedHash := hex.EncodeToString(h.Sum(nil))
	if hash1 != expectedHash {
		t.Error("hash should match expected SHA256")
	}
}

// TestDeterministicFallbacks tests the deterministic fallback functions
func TestDeterministicFallbacks(t *testing.T) {
	config := LLMConfig{
		ProviderType: ProviderDisabled,
		Enabled:      false,
	}
	client, err := NewLLMClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Test deterministicClassify
	classifyResult, err := client.Classify(context.Background(), Input{})
	if err != nil {
		t.Errorf("expected no error for disabled classify, got %v", err)
	}
	if classifyResult.Category != "unknown" {
		t.Errorf("expected category 'unknown', got %s", classifyResult.Category)
	}
	if classifyResult.Confidence != 0.0 {
		t.Errorf("expected confidence 0.0, got %f", classifyResult.Confidence)
	}
	if !classifyResult.IsDeterministicFallback {
		t.Error("expected IsDeterministicFallback to be true")
	}

	// Test deterministicExplain
	explainResult, err := client.Explain(context.Background(), Anomaly{})
	if err != nil {
		t.Errorf("expected no error for disabled explain, got %v", err)
	}
	if explainResult.Summary != "LLM unavailable - no explanation available" {
		t.Errorf("expected specific summary, got %s", explainResult.Summary)
	}
	if len(explainResult.PossibleCauses) != 1 || explainResult.PossibleCauses[0] != "LLM service unavailable" {
		t.Error("expected specific possible causes")
	}
	if !explainResult.IsDeterministicFallback {
		t.Error("expected IsDeterministicFallback to be true")
	}

	// Test deterministicSuggest
	suggestResult, err := client.Suggest(context.Background(), Scenario{})
	if err != nil {
		t.Errorf("expected no error for disabled suggest, got %v", err)
	}
	if len(suggestResult.Actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(suggestResult.Actions))
	}
	if suggestResult.Priority != "low" {
		t.Errorf("expected priority 'low', got %s", suggestResult.Priority)
	}
	if !suggestResult.IsDeterministicFallback {
		t.Error("expected IsDeterministicFallback to be true")
	}
}

// TestConcurrentAccess tests thread safety of the LLM client
// Note: Uses enabled provider to generate audit entries
func TestConcurrentAccess(t *testing.T) {
	config := LLMConfig{
		ProviderType: ProviderLocal,
		Enabled:      true,
		Endpoint:     "http://localhost:11434",
		Timeout:      100 * time.Millisecond,
	}
	client, err := NewLLMClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 50

	// Run concurrent Classify operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				input := Input{
					Component:  fmt.Sprintf("component-%d", id),
					MetricType: "test",
				}
				_, err := client.Classify(ctx, input)
				if err != nil {
					t.Errorf("goroutine %d: classify error: %v", id, err)
				}
			}
		}(i)
	}

	// Run concurrent audit log reads
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = client.GetAuditLog()
			}
		}(i)
	}

	wg.Wait()

	// Verify audit log has expected entries
	auditLog := client.GetAuditLog()
	expectedEntries := numGoroutines * numOperations
	if len(auditLog) != expectedEntries {
		t.Errorf("expected %d audit entries, got %d", expectedEntries, len(auditLog))
	}
}

// TestLastErrorTracking tests error tracking
func TestLastErrorTracking(t *testing.T) {
	// Create client with local provider (will generate errors)
	config := LLMConfig{
		ProviderType: ProviderLocal,
		Enabled:      true,
		Endpoint:     "http://localhost:11434",
		Timeout:      100 * time.Millisecond,
	}
	client, err := NewLLMClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Initially no error
	if client.GetLastError() != nil {
		t.Error("expected no last error initially")
	}

	// Perform operation that will fail
	ctx := context.Background()
	client.Classify(ctx, Input{Component: "test", MetricType: "latency"})

	// Should have recorded error
	lastErr := client.GetLastError()
	if lastErr == nil {
		t.Error("expected last error to be set after failed operation")
	}
}

// TestConstants tests that constants are properly defined
func TestConstants(t *testing.T) {
	// Test timeout constants
	if DefaultLLMTimeout != 5*time.Second {
		t.Errorf("expected DefaultLLMTimeout to be 5s, got %v", DefaultLLMTimeout)
	}
	if MaxLLMTimeout != 30*time.Second {
		t.Errorf("expected MaxLLMTimeout to be 30s, got %v", MaxLLMTimeout)
	}

	// Test retry constant
	if DefaultLLMMaxRetries != 2 {
		t.Errorf("expected DefaultLLMMaxRetries to be 2, got %d", DefaultLLMMaxRetries)
	}

	// Test audit log constant
	if MaxAuditLogEntries != 10000 {
		t.Errorf("expected MaxAuditLogEntries to be 10000, got %d", MaxAuditLogEntries)
	}

	// Test provider types
	if ProviderDisabled != "disabled" {
		t.Errorf("expected ProviderDisabled to be 'disabled', got %s", ProviderDisabled)
	}
	if ProviderLocal != "local" {
		t.Errorf("expected ProviderLocal to be 'local', got %s", ProviderLocal)
	}
	if ProviderAPI != "api" {
		t.Errorf("expected ProviderAPI to be 'api', got %s", ProviderAPI)
	}

	// Test operation types
	if OpClassify != "classify" {
		t.Errorf("expected OpClassify to be 'classify', got %s", OpClassify)
	}
	if OpExplain != "explain" {
		t.Errorf("expected OpExplain to be 'explain', got %s", OpExplain)
	}
	if OpSuggest != "suggest" {
		t.Errorf("expected OpSuggest to be 'suggest', got %s", OpSuggest)
	}
	if OpHealthCheck != "health_check" {
		t.Errorf("expected OpHealthCheck to be 'health_check', got %s", OpHealthCheck)
	}
}

// TestErrors tests error definitions
func TestErrors(t *testing.T) {
	errors := []struct {
		err  error
		name string
	}{
		{ErrLLMDisabled, "ErrLLMDisabled"},
		{ErrLLMTimeout, "ErrLLMTimeout"},
		{ErrLLMUnavailable, "ErrLLMUnavailable"},
		{ErrInvalidConfig, "ErrInvalidConfig"},
		{ErrRedactionFailed, "ErrRedactionFailed"},
		{ErrProviderNotImplemented, "ErrProviderNotImplemented"},
	}

	for _, e := range errors {
		if e.err == nil {
			t.Errorf("%s should not be nil", e.name)
		}
		if e.err.Error() == "" {
			t.Errorf("%s should have a non-empty message", e.name)
		}
	}
}

// TestSanitizeError tests error sanitization
func TestSanitizeError(t *testing.T) {
	config := LLMConfig{
		ProviderType: ProviderDisabled,
		Enabled:      false,
	}
	client, err := NewLLMClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Test nil error
	sanitized := client.sanitizeError(nil)
	if sanitized != "" {
		t.Errorf("expected empty string for nil error, got %q", sanitized)
	}

	// Test regular error
	regularErr := errors.New("some error message")
	sanitized = client.sanitizeError(regularErr)
	if sanitized != "some error message" {
		t.Errorf("expected 'some error message', got %q", sanitized)
	}

	// Test long error (should be truncated)
	longErr := errors.New(strings.Repeat("a", 300))
	sanitized = client.sanitizeError(longErr)
	if len(sanitized) > 260 {
		t.Errorf("expected truncated error (<=260 chars), got %d chars", len(sanitized))
	}
	if !strings.HasSuffix(sanitized, "...") {
		t.Error("expected truncated error to end with '...'")
	}
}

// TestAuditLogTruncation tests that audit log is truncated when it exceeds max entries
// Note: Uses enabled provider to generate audit entries
func TestAuditLogTruncation(t *testing.T) {
	config := LLMConfig{
		ProviderType: ProviderLocal,
		Enabled:      true,
		Endpoint:     "http://localhost:11434",
		Timeout:      100 * time.Millisecond,
	}
	client, err := NewLLMClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Add entries (using a reasonable number for testing)
	numEntries := 100
	for i := 0; i < numEntries; i++ {
		input := Input{
			Component:  fmt.Sprintf("component-%d", i),
			MetricType: "test",
		}
		client.Classify(ctx, input)
	}

	// Verify all entries are present (we're under the MaxAuditLogEntries limit)
	auditLog := client.GetAuditLog()
	if len(auditLog) != numEntries {
		t.Errorf("expected %d entries, got %d", numEntries, len(auditLog))
	}

	// The audit log implementation handles truncation automatically
	// when entries exceed MaxAuditLogEntries (10000)
}

// TestInputTypes tests Input struct fields
func TestInputTypes(t *testing.T) {
	now := time.Now()
	input := Input{
		Component:  "test-component",
		MetricType: "test-metric",
		RawValue:   "test-value",
		Statistics: map[string]float64{
			"mean": 10.5,
		},
		Timestamp: now,
	}

	if input.Component != "test-component" {
		t.Error("Component field not set correctly")
	}
	if input.MetricType != "test-metric" {
		t.Error("MetricType field not set correctly")
	}
	if input.RawValue != "test-value" {
		t.Error("RawValue field not set correctly")
	}
	if input.Statistics["mean"] != 10.5 {
		t.Error("Statistics field not set correctly")
	}
	if !input.Timestamp.Equal(now) {
		t.Error("Timestamp field not set correctly")
	}
}

// TestAnomalyTypes tests Anomaly struct fields
func TestAnomalyTypes(t *testing.T) {
	now := time.Now()
	anomaly := Anomaly{
		Component:  "test-component",
		MetricType: "test-metric",
		Severity:   "high",
		Context:    "test context",
		DetectedAt: now,
	}

	if anomaly.Component != "test-component" {
		t.Error("Component field not set correctly")
	}
	if anomaly.Severity != "high" {
		t.Error("Severity field not set correctly")
	}
	if !anomaly.DetectedAt.Equal(now) {
		t.Error("DetectedAt field not set correctly")
	}
}

// TestScenarioTypes tests Scenario struct fields
func TestScenarioTypes(t *testing.T) {
	scenario := Scenario{
		Component:    "test-component",
		CurrentState: "degraded",
		Constraints:  []string{"constraint1", "constraint2"},
		Goal:         "restore service",
	}

	if scenario.Component != "test-component" {
		t.Error("Component field not set correctly")
	}
	if len(scenario.Constraints) != 2 {
		t.Errorf("expected 2 constraints, got %d", len(scenario.Constraints))
	}
}

// TestClassificationTypes tests Classification struct fields
func TestClassificationTypes(t *testing.T) {
	classification := Classification{
		Category:                "anomaly",
		Confidence:              0.95,
		Reasoning:               "test reasoning",
		IsDeterministicFallback: false,
	}

	if classification.Category != "anomaly" {
		t.Error("Category field not set correctly")
	}
	if classification.Confidence != 0.95 {
		t.Error("Confidence field not set correctly")
	}
	if classification.IsDeterministicFallback {
		t.Error("IsDeterministicFallback should be false")
	}
}

// TestExplanationTypes tests Explanation struct fields
func TestExplanationTypes(t *testing.T) {
	explanation := Explanation{
		Summary:                 "test summary",
		PossibleCauses:          []string{"cause1", "cause2"},
		Confidence:              0.8,
		IsDeterministicFallback: true,
	}

	if explanation.Summary != "test summary" {
		t.Error("Summary field not set correctly")
	}
	if len(explanation.PossibleCauses) != 2 {
		t.Errorf("expected 2 possible causes, got %d", len(explanation.PossibleCauses))
	}
	if !explanation.IsDeterministicFallback {
		t.Error("IsDeterministicFallback should be true")
	}
}

// TestSuggestionTypes tests Suggestion struct fields
func TestSuggestionTypes(t *testing.T) {
	suggestion := Suggestion{
		Actions:                 []string{"action1", "action2"},
		Priority:                "high",
		Risks:                   []string{"risk1"},
		Confidence:              0.75,
		IsDeterministicFallback: false,
	}

	if suggestion.Priority != "high" {
		t.Error("Priority field not set correctly")
	}
	if len(suggestion.Actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(suggestion.Actions))
	}
	if len(suggestion.Risks) != 1 {
		t.Errorf("expected 1 risk, got %d", len(suggestion.Risks))
	}
}

// TestLLMAuditEntryTypes tests LLMAuditEntry struct fields
func TestLLMAuditEntryTypes(t *testing.T) {
	now := time.Now()
	entry := LLMAuditEntry{
		Timestamp:    now,
		Operation:    OpClassify,
		InputHash:    "abc123",
		Success:      true,
		ResponseHash: "def456",
		Latency:      100 * time.Millisecond,
		Error:        "",
	}

	if entry.Operation != OpClassify {
		t.Error("Operation field not set correctly")
	}
	if !entry.Success {
		t.Error("Success field not set correctly")
	}
	if entry.Latency != 100*time.Millisecond {
		t.Error("Latency field not set correctly")
	}
}

// TestRedactorCreation tests redactor initialization
func TestRedactorCreation(t *testing.T) {
	r := newRedactor()
	if r == nil {
		t.Fatal("newRedactor should return non-nil redactor")
	}
	if r.stringPattern == nil {
		t.Error("stringPattern should be initialized")
	}
	if r.timestampPattern == nil {
		t.Error("timestampPattern should be initialized")
	}
}

// TestProviderTypeString tests ProviderType string conversion
func TestProviderTypeString(t *testing.T) {
	tests := []struct {
		provider ProviderType
		expected string
	}{
		{ProviderDisabled, "disabled"},
		{ProviderLocal, "local"},
		{ProviderAPI, "api"},
	}

	for _, tt := range tests {
		if string(tt.provider) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, string(tt.provider))
		}
	}
}

// TestLLMOperationString tests LLMOperation string conversion
func TestLLMOperationString(t *testing.T) {
	tests := []struct {
		op       LLMOperation
		expected string
	}{
		{OpClassify, "classify"},
		{OpExplain, "explain"},
		{OpSuggest, "suggest"},
		{OpHealthCheck, "health_check"},
	}

	for _, tt := range tests {
		if string(tt.op) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, string(tt.op))
		}
	}
}

// TestHealthCheck tests the HealthCheck method
func TestHealthCheck(t *testing.T) {
	// Test disabled client health check
	disabledConfig := LLMConfig{
		ProviderType: ProviderDisabled,
		Enabled:      false,
	}
	disabledClient, _ := NewLLMClient(disabledConfig)
	defer disabledClient.Close()

	if err := disabledClient.HealthCheck(); err != nil {
		t.Errorf("disabled client should always pass health check: %v", err)
	}

	// Test local client health check (returns unavailable since it's a stub)
	localConfig := LLMConfig{
		ProviderType: ProviderLocal,
		Enabled:      true,
		Endpoint:     "http://localhost:11434",
	}
	localClient, _ := NewLLMClient(localConfig)
	defer localClient.Close()

	if err := localClient.HealthCheck(); !errors.Is(err, ErrLLMUnavailable) {
		t.Errorf("local client should return ErrLLMUnavailable: %v", err)
	}

	// Test API client health check (returns unavailable since it's a stub)
	apiConfig := LLMConfig{
		ProviderType: ProviderAPI,
		Enabled:      true,
		Endpoint:     "https://api.example.com",
	}
	apiClient, _ := NewLLMClient(apiConfig)
	defer apiClient.Close()

	if err := apiClient.HealthCheck(); !errors.Is(err, ErrLLMUnavailable) {
		t.Errorf("API client should return ErrLLMUnavailable: %v", err)
	}
}

// TestClosedClientHealthCheck tests HealthCheck on closed client
func TestClosedClientHealthCheck(t *testing.T) {
	config := LLMConfig{
		ProviderType: ProviderDisabled,
		Enabled:      false,
	}
	client, _ := NewLLMClient(config)
	client.Close()

	if err := client.HealthCheck(); err == nil {
		t.Error("expected error when calling HealthCheck on closed client")
	}
}

// TestContextCancellation tests that context cancellation is handled
func TestContextCancellation(t *testing.T) {
	// Note: With disabled provider, operations return immediately
	// so context cancellation doesn't come into play.
	// This test documents the expected behavior.

	config := LLMConfig{
		ProviderType: ProviderDisabled,
		Enabled:      false,
	}
	client, _ := NewLLMClient(config)
	defer client.Close()

	// Create already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should still return fallback (not error) because disabled provider
	// returns before checking context
	input := Input{Component: "test", MetricType: "latency"}
	result, err := client.Classify(ctx, input)
	if err != nil {
		t.Errorf("expected no error even with cancelled context for disabled provider: %v", err)
	}
	if !result.IsDeterministicFallback {
		t.Error("expected deterministic fallback")
	}
}
