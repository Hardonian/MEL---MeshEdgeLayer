package selfobs

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// TEST: OpenAIProvider Creation
// =============================================================================

func TestOpenAIProviderCreation(t *testing.T) {
	t.Run("valid configuration", func(t *testing.T) {
		provider, err := NewOpenAIProvider("sk-test-key", "https://api.openai.com/v1", "gpt-4", 30*time.Second)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if provider == nil {
			t.Fatal("expected provider, got nil")
		}
		if provider.apiKey != "sk-test-key" {
			t.Errorf("expected apiKey to be set")
		}
		if provider.baseURL != "https://api.openai.com/v1" {
			t.Errorf("expected baseURL to be 'https://api.openai.com/v1', got %s", provider.baseURL)
		}
		if provider.model != "gpt-4" {
			t.Errorf("expected model to be 'gpt-4', got %s", provider.model)
		}
		if provider.timeout != 30*time.Second {
			t.Errorf("expected timeout to be 30s, got %v", provider.timeout)
		}
		defer provider.Close()
	})

	t.Run("empty API key", func(t *testing.T) {
		_, err := NewOpenAIProvider("", "https://api.openai.com/v1", "gpt-4", 30*time.Second)
		if err == nil {
			t.Fatal("expected error for empty API key")
		}
		if !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("expected ErrInvalidConfig, got %v", err)
		}
	})

	t.Run("invalid URL with trailing slash", func(t *testing.T) {
		provider, err := NewOpenAIProvider("sk-test", "https://api.openai.com/v1/", "gpt-4", 30*time.Second)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		// Trailing slash should be trimmed
		if provider.baseURL != "https://api.openai.com/v1" {
			t.Errorf("expected trailing slash to be trimmed, got %s", provider.baseURL)
		}
		defer provider.Close()
	})

	t.Run("default baseURL", func(t *testing.T) {
		provider, err := NewOpenAIProvider("sk-test", "", "gpt-4", 30*time.Second)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if provider.baseURL != DefaultOpenAIBaseURL {
			t.Errorf("expected default baseURL %s, got %s", DefaultOpenAIBaseURL, provider.baseURL)
		}
		defer provider.Close()
	})

	t.Run("default model", func(t *testing.T) {
		provider, err := NewOpenAIProvider("sk-test", "https://api.openai.com/v1", "", 30*time.Second)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if provider.model != DefaultOpenAIModel {
			t.Errorf("expected default model %s, got %s", DefaultOpenAIModel, provider.model)
		}
		defer provider.Close()
	})

	t.Run("default timeout", func(t *testing.T) {
		provider, err := NewOpenAIProvider("sk-test", "https://api.openai.com/v1", "gpt-4", 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if provider.timeout != DefaultRequestTimeout {
			t.Errorf("expected default timeout %v, got %v", DefaultRequestTimeout, provider.timeout)
		}
		defer provider.Close()
	})
}

// =============================================================================
// TEST: HealthCheck
// =============================================================================

func TestOpenAIHealthCheck(t *testing.T) {
	t.Run("mock server returning 200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/models" {
				t.Errorf("expected path /models, got %s", r.URL.Path)
			}
			if r.Header.Get("Authorization") != "Bearer sk-test" {
				t.Errorf("expected Authorization header")
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]string{},
			})
		}))
		defer server.Close()

		provider, err := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer provider.Close()

		if err := provider.HealthCheck(); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("mock server returning 401", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		provider, err := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer provider.Close()

		err = provider.HealthCheck()
		if err == nil {
			t.Fatal("expected error for 401 response")
		}
		if !strings.Contains(err.Error(), "401") {
			t.Errorf("expected error to contain '401', got %v", err)
		}
	})

	t.Run("mock server returning 429", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer server.Close()

		provider, err := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer provider.Close()

		err = provider.HealthCheck()
		if err == nil {
			t.Fatal("expected error for 429 response")
		}
	})

	t.Run("unreachable server", func(t *testing.T) {
		provider, err := NewOpenAIProvider("sk-test", "http://localhost:99999", "gpt-4", 1*time.Second)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer provider.Close()

		err = provider.HealthCheck()
		if err == nil {
			t.Fatal("expected error for unreachable server")
		}
	})

	t.Run("circuit breaker open", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		provider, err := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer provider.Close()

		// Open the circuit
		provider.circuitMu.Lock()
		provider.circuitState = circuitOpen
		provider.lastFailureTime = time.Now()
		provider.circuitMu.Unlock()

		err = provider.HealthCheck()
		if !errors.Is(err, ErrCircuitBreakerOpen) {
			t.Errorf("expected ErrCircuitBreakerOpen, got %v", err)
		}
	})
}

// =============================================================================
// TEST: Classify
// =============================================================================

func TestClassify(t *testing.T) {
	t.Run("successful classification", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/chat/completions" {
				t.Errorf("expected path /chat/completions, got %s", r.URL.Path)
			}

			var req ChatCompletionRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("failed to decode request: %v", err)
			}

			if req.Model != "gpt-4" {
				t.Errorf("expected model gpt-4, got %s", req.Model)
			}

			resp := ChatCompletionResponse{
				ID:      "test-id",
				Object:  "chat.completion",
				Created: time.Now().Unix(),
				Model:   "gpt-4",
				Choices: []Choice{
					{
						Index: 0,
						Message: Message{
							Role:    "assistant",
							Content: `{"category": "warning", "confidence": 0.85, "reasoning": "High latency detected"}`,
						},
						FinishReason: "stop",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		provider, err := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer provider.Close()

		input := Input{
			Component:  "database",
			MetricType: "latency",
			RawValue:   "500ms",
			Statistics: map[string]float64{"p99": 500},
		}

		ctx := context.Background()
		classification, err := provider.Classify(ctx, input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if classification.Category != "warning" {
			t.Errorf("expected category 'warning', got %s", classification.Category)
		}
		if classification.Confidence != 0.85 {
			t.Errorf("expected confidence 0.85, got %f", classification.Confidence)
		}
		if classification.IsDeterministicFallback {
			t.Error("expected IsDeterministicFallback to be false")
		}
	})

	t.Run("response parsing with markdown code block", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ChatCompletionResponse{
				ID: "test-id",
				Choices: []Choice{
					{
						Message: Message{
							Content: "```json\n{\"category\": \"critical\", \"confidence\": 0.95, \"reasoning\": \"System failure\"}\n```",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		provider, _ := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		defer provider.Close()

		classification, _ := provider.Classify(context.Background(), Input{})

		if classification.Category != "critical" {
			t.Errorf("expected category 'critical', got %s", classification.Category)
		}
		if classification.Confidence != 0.95 {
			t.Errorf("expected confidence 0.95, got %f", classification.Confidence)
		}
	})

	t.Run("timeout handling", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		provider, _ := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		defer provider.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		classification, _ := provider.Classify(ctx, Input{})

		if !classification.IsDeterministicFallback {
			t.Error("expected fallback due to timeout")
		}
		if classification.Category != "unknown" {
			t.Errorf("expected category 'unknown', got %s", classification.Category)
		}
	})

	t.Run("retry behavior with rate limit", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			if attemptCount < 3 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			resp := ChatCompletionResponse{
				Choices: []Choice{
					{
						Message: Message{
							Content: `{"category": "normal", "confidence": 0.9, "reasoning": "All good"}`,
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		provider, _ := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		defer provider.Close()

		classification, _ := provider.Classify(context.Background(), Input{})

		if attemptCount < 3 {
			t.Errorf("expected at least 3 attempts, got %d", attemptCount)
		}
		if classification.Category != "normal" {
			t.Errorf("expected category 'normal', got %s", classification.Category)
		}
	})

	t.Run("circuit breaker after failures", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		provider, _ := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		defer provider.Close()

		// Trigger failures to open circuit
		for i := 0; i < CircuitBreakerThreshold+1; i++ {
			provider.Classify(context.Background(), Input{})
		}

		// Check circuit is open
		if provider.CircuitStatus() != "open" {
			t.Errorf("expected circuit to be open, got %s", provider.CircuitStatus())
		}

		// Next request should use fallback due to open circuit
		classification, _ := provider.Classify(context.Background(), Input{})
		if !classification.IsDeterministicFallback {
			t.Error("expected fallback when circuit is open")
		}
	})

	t.Run("empty response fallback", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ChatCompletionResponse{
				Choices: []Choice{},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		provider, _ := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		defer provider.Close()

		classification, _ := provider.Classify(context.Background(), Input{})

		if !classification.IsDeterministicFallback {
			t.Error("expected fallback for empty response")
		}
	})
}

// =============================================================================
// TEST: Explain
// =============================================================================

func TestExplain(t *testing.T) {
	t.Run("successful explanation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ChatCompletionResponse{
				Choices: []Choice{
					{
						Message: Message{
							Content: `{"summary": "Database connection pool exhausted", "possible_causes": ["Too many concurrent connections", "Connection leak"], "confidence": 0.8}`,
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		provider, _ := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		defer provider.Close()

		anomaly := Anomaly{
			Component:  "database",
			MetricType: "connections",
			Severity:   "critical",
			Context:    "Connection pool at 100%",
		}

		explanation, err := provider.Explain(context.Background(), anomaly)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if explanation.Summary != "Database connection pool exhausted" {
			t.Errorf("unexpected summary: %s", explanation.Summary)
		}
		if len(explanation.PossibleCauses) != 2 {
			t.Errorf("expected 2 possible causes, got %d", len(explanation.PossibleCauses))
		}
		if explanation.Confidence != 0.8 {
			t.Errorf("expected confidence 0.8, got %f", explanation.Confidence)
		}
		if explanation.IsDeterministicFallback {
			t.Error("expected IsDeterministicFallback to be false")
		}
	})

	t.Run("error handling", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		provider, _ := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		defer provider.Close()

		anomaly := Anomaly{Component: "test"}
		explanation, err := provider.Explain(context.Background(), anomaly)

		// Explain returns nil error with fallback on failure
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if !explanation.IsDeterministicFallback {
			t.Error("expected fallback for error case")
		}
		if explanation.Confidence != 0.0 {
			t.Errorf("expected confidence 0.0, got %f", explanation.Confidence)
		}
	})
}

// =============================================================================
// TEST: Suggest
// =============================================================================

func TestSuggest(t *testing.T) {
	t.Run("successful suggestion", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ChatCompletionResponse{
				Choices: []Choice{
					{
						Message: Message{
							Content: `{"actions": ["Increase connection pool size", "Add connection timeout"], "priority": "high", "risks": ["Temporary performance impact"], "confidence": 0.85}`,
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		provider, _ := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		defer provider.Close()

		scenario := Scenario{
			Component:    "database",
			CurrentState: "High load",
			Goal:         "Reduce latency",
		}

		suggestion, err := provider.Suggest(context.Background(), scenario)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(suggestion.Actions) != 2 {
			t.Errorf("expected 2 actions, got %d", len(suggestion.Actions))
		}
		if suggestion.Priority != "high" {
			t.Errorf("expected priority 'high', got %s", suggestion.Priority)
		}
		if len(suggestion.Risks) != 1 {
			t.Errorf("expected 1 risk, got %d", len(suggestion.Risks))
		}
		if suggestion.Confidence != 0.85 {
			t.Errorf("expected confidence 0.85, got %f", suggestion.Confidence)
		}
		if suggestion.IsDeterministicFallback {
			t.Error("expected IsDeterministicFallback to be false")
		}
	})

	t.Run("error handling", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		provider, _ := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		defer provider.Close()

		scenario := Scenario{Component: "test"}
		suggestion, err := provider.Suggest(context.Background(), scenario)

		// Suggest returns nil error with fallback on failure
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if !suggestion.IsDeterministicFallback {
			t.Error("expected fallback for error case")
		}
		if suggestion.Confidence != 0.0 {
			t.Errorf("expected confidence 0.0, got %f", suggestion.Confidence)
		}
	})
}

// =============================================================================
// TEST: Circuit Breaker
// =============================================================================

func TestOpenAICircuitBreaker(t *testing.T) {
	t.Run("circuit closes initially", func(t *testing.T) {
		provider, _ := NewOpenAIProvider("sk-test", "http://localhost", "gpt-4", 30*time.Second)
		defer provider.Close()

		if provider.CircuitStatus() != "closed" {
			t.Errorf("expected circuit to be closed initially, got %s", provider.CircuitStatus())
		}
	})

	t.Run("circuit opens after threshold", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		provider, _ := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		defer provider.Close()

		// Record failures up to threshold
		for i := 0; i < CircuitBreakerThreshold; i++ {
			provider.recordFailure()
		}

		if provider.CircuitStatus() != "open" {
			t.Errorf("expected circuit to be open after %d failures, got %s", CircuitBreakerThreshold, provider.CircuitStatus())
		}
	})

	t.Run("circuit half-open state", func(t *testing.T) {
		provider, _ := NewOpenAIProvider("sk-test", "http://localhost", "gpt-4", 30*time.Second)
		defer provider.Close()

		// Open the circuit with an old failure time
		provider.circuitMu.Lock()
		provider.circuitState = circuitOpen
		provider.lastFailureTime = time.Now().Add(-CircuitBreakerTimeout - time.Second)
		provider.circuitMu.Unlock()

		// Check circuit should transition to half-open
		err := provider.checkCircuit()
		if err != nil {
			t.Errorf("expected no error when transitioning to half-open, got %v", err)
		}

		if provider.CircuitStatus() != "half-open" {
			t.Errorf("expected circuit to be half-open, got %s", provider.CircuitStatus())
		}
	})

	t.Run("manual reset", func(t *testing.T) {
		provider, _ := NewOpenAIProvider("sk-test", "http://localhost", "gpt-4", 30*time.Second)
		defer provider.Close()

		// Open the circuit
		for i := 0; i < CircuitBreakerThreshold; i++ {
			provider.recordFailure()
		}

		if provider.CircuitStatus() != "open" {
			t.Fatal("expected circuit to be open")
		}

		// Reset manually
		provider.ResetCircuit()

		if provider.CircuitStatus() != "closed" {
			t.Errorf("expected circuit to be closed after reset, got %s", provider.CircuitStatus())
		}
		if provider.FailureCount() != 0 {
			t.Errorf("expected failure count to be 0, got %d", provider.FailureCount())
		}
	})

	t.Run("success resets failure count", func(t *testing.T) {
		provider, _ := NewOpenAIProvider("sk-test", "http://localhost", "gpt-4", 30*time.Second)
		defer provider.Close()

		// Record some failures
		provider.recordFailure()
		provider.recordFailure()

		if provider.FailureCount() != 2 {
			t.Errorf("expected failure count 2, got %d", provider.FailureCount())
		}

		// Record success
		provider.recordSuccess()

		if provider.FailureCount() != 0 {
			t.Errorf("expected failure count reset to 0, got %d", provider.FailureCount())
		}
		if provider.CircuitStatus() != "closed" {
			t.Errorf("expected circuit to be closed, got %s", provider.CircuitStatus())
		}
	})
}

// =============================================================================
// TEST: Retry Behavior
// =============================================================================

func TestRetryBehavior(t *testing.T) {
	t.Run("exponential backoff", func(t *testing.T) {
		provider, _ := NewOpenAIProvider("sk-test", "http://localhost", "gpt-4", 30*time.Second)
		defer provider.Close()

		// Test backoff durations
		d1 := provider.backoffDuration(0)
		d2 := provider.backoffDuration(1)
		d3 := provider.backoffDuration(2)

		// Should be increasing (100ms, 200ms, 400ms with jitter)
		if d1 >= d2 {
			t.Errorf("expected increasing backoff: d1=%v, d2=%v", d1, d2)
		}
		if d2 >= d3 {
			t.Errorf("expected increasing backoff: d2=%v, d3=%v", d2, d3)
		}

		// Check approximate values
		if d1 < 100*time.Millisecond || d1 > 120*time.Millisecond {
			t.Errorf("expected d1 around 100ms, got %v", d1)
		}
		if d2 < 200*time.Millisecond || d2 > 230*time.Millisecond {
			t.Errorf("expected d2 around 200ms, got %v", d2)
		}
		if d3 < 400*time.Millisecond || d3 > 430*time.Millisecond {
			t.Errorf("expected d3 around 400ms, got %v", d3)
		}
	})

	t.Run("max retry limit", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		provider, _ := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		defer provider.Close()

		// Make request that will fail
		provider.Classify(context.Background(), Input{})

		// Should be initial request + MaxRetries
		expectedAttempts := 1 + MaxRetries
		if attemptCount != expectedAttempts {
			t.Errorf("expected %d attempts, got %d", expectedAttempts, attemptCount)
		}
	})

	t.Run("non-retryable errors don't retry", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		provider, _ := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		defer provider.Close()

		// Make request that will fail with 401
		_, err := provider.makeChatCompletion(context.Background(), []Message{})

		if err == nil {
			t.Fatal("expected error")
		}

		// Should only be 1 attempt (no retries for 401)
		if attemptCount != 1 {
			t.Errorf("expected 1 attempt for non-retryable error, got %d", attemptCount)
		}
	})

	t.Run("isRetryableError for rate limited", func(t *testing.T) {
		if !isRetryableError(ErrRateLimited) {
			t.Error("expected ErrRateLimited to be retryable")
		}
	})

	t.Run("isRetryableError for context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if isRetryableError(ctx.Err()) {
			t.Error("expected context.Canceled to not be retryable")
		}
	})

	t.Run("isRetryableError for deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond)

		if isRetryableError(ctx.Err()) {
			t.Error("expected context.DeadlineExceeded to not be retryable")
		}
	})

	t.Run("isRetryableError for API errors", func(t *testing.T) {
		tests := []struct {
			code      string
			retryable bool
		}{
			{"rate_limit_exceeded", true},
			{"timeout", true},
			{"temporarily_unavailable", true},
			{"invalid_api_key", false},
			{"insufficient_quota", false},
			{"invalid_request_error", false},
		}

		for _, tc := range tests {
			apiErr := &APIError{Code: tc.code, Message: "test"}
			result := isRetryableError(apiErr)
			if result != tc.retryable {
				t.Errorf("code=%s: expected retryable=%v, got %v", tc.code, tc.retryable, result)
			}
		}
	})
}

// =============================================================================
// TEST: Redaction
// =============================================================================

func TestOpenAIRedaction(t *testing.T) {
	t.Run("API key redacted in errors", func(t *testing.T) {
		apiKey := "sk-secret-key-12345"
		provider, _ := NewOpenAIProvider(apiKey, "http://localhost:1", "gpt-4", 100*time.Millisecond)
		defer provider.Close()

		// Trigger an error
		_, err := provider.makeChatCompletion(context.Background(), []Message{{Role: "user", Content: "test"}})

		if err == nil {
			t.Fatal("expected error")
		}

		errStr := err.Error()
		if strings.Contains(errStr, apiKey) {
			t.Errorf("error message contains unredacted API key: %s", errStr)
		}
		if strings.Contains(errStr, "[REDACTED]") {
			// API key was redacted - good
		}
	})

	t.Run("input redaction before sending", func(t *testing.T) {
		var receivedBody string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req ChatCompletionRequest
			json.NewDecoder(r.Body).Decode(&req)
			bodyBytes, _ := json.Marshal(req)
			receivedBody = string(bodyBytes)

			resp := ChatCompletionResponse{
				Choices: []Choice{{Message: Message{Content: `{"category": "normal"}`}}},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		provider, _ := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		defer provider.Close()

		input := Input{
			Component:  "database",
			MetricType: "latency",
			RawValue:   "sensitive-value-123",
			Statistics: map[string]float64{"p99": 500},
		}

		provider.Classify(context.Background(), input)

		// Raw value should be redacted in the request
		if strings.Contains(receivedBody, "sensitive-value-123") {
			t.Error("raw value was not redacted in request")
		}
		if !strings.Contains(receivedBody, "[REDACTED]") {
			t.Error("expected [REDACTED] in request body")
		}
	})

	t.Run("redactor redacts strings", func(t *testing.T) {
		r := newRedactor()
		input := "error: \"sensitive data\" at 2024-01-15T10:30:00Z"
		result := r.RedactString(input)

		if strings.Contains(result, "sensitive data") {
			t.Error("string was not redacted")
		}
		if !strings.Contains(result, "[REDACTED]") {
			t.Error("expected [REDACTED] in result")
		}
		if !strings.Contains(result, "[TIME]") {
			t.Error("expected [TIME] for timestamp")
		}
	})

	t.Run("redactor redacts input", func(t *testing.T) {
		r := newRedactor()
		input := Input{
			Component:  "db",
			MetricType: "cpu",
			RawValue:   "secret",
			Statistics: map[string]float64{"usage": 0.9},
			Timestamp:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		}
		result := r.RedactInput(input)

		if strings.Contains(result, "secret") {
			t.Error("raw value was not redacted")
		}
		if !strings.Contains(result, "component:db") {
			t.Error("component should be preserved")
		}
		if !strings.Contains(result, "raw_value:[REDACTED]") {
			t.Error("expected raw_value to be redacted")
		}
		if !strings.Contains(result, "timestamp:[TIME]") {
			t.Error("expected timestamp to be replaced")
		}
	})

	t.Run("redactor redacts anomaly", func(t *testing.T) {
		r := newRedactor()
		anomaly := Anomaly{
			Component:  "api",
			MetricType: "errors",
			Severity:   "high",
			Context:    "sensitive context info",
			DetectedAt: time.Now(),
		}
		result := r.RedactAnomaly(anomaly)

		if strings.Contains(result, "sensitive context info") {
			t.Error("context was not redacted")
		}
		if !strings.Contains(result, "context:[REDACTED]") {
			t.Error("expected context to be redacted")
		}
		if !strings.Contains(result, "severity:high") {
			t.Error("severity should be preserved")
		}
	})

	t.Run("redactor redacts scenario", func(t *testing.T) {
		r := newRedactor()
		scenario := Scenario{
			Component:    "service",
			CurrentState: "secret state",
			Constraints:  []string{"constraint1", "constraint2"},
			Goal:         "secret goal",
		}
		result := r.RedactScenario(scenario)

		if strings.Contains(result, "secret state") || strings.Contains(result, "secret goal") {
			t.Error("state/goal was not redacted")
		}
		if !strings.Contains(result, "current_state:[REDACTED]") {
			t.Error("expected current_state to be redacted")
		}
		if !strings.Contains(result, "constraints:[REDACTED:2 items]") {
			t.Error("expected constraints count")
		}
	})
}

// =============================================================================
// TEST: Error Types
// =============================================================================

func TestOpenAIErrorTypes(t *testing.T) {
	tests := []struct {
		name     error
		expected string
	}{
		{ErrCircuitBreakerOpen, "circuit breaker is open"},
		{ErrInvalidResponse, "invalid or empty response from API"},
		{ErrRateLimited, "rate limited by API"},
		{ErrAuthenticationFailed, "API authentication failed"},
		{ErrInvalidRequest, "invalid request"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			if tc.name.Error() != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, tc.name.Error())
			}
		})
	}

	t.Run("APIError error message", func(t *testing.T) {
		apiErr := &APIError{
			Message: "something went wrong",
			Type:    "server_error",
			Code:    "internal_error",
		}
		expected := "API error (type=server_error, code=internal_error): something went wrong"
		if apiErr.Error() != expected {
			t.Errorf("expected %q, got %q", expected, apiErr.Error())
		}
	})

	t.Run("nil APIError", func(t *testing.T) {
		var apiErr *APIError
		expected := "unknown API error"
		if apiErr.Error() != expected {
			t.Errorf("expected %q, got %q", expected, apiErr.Error())
		}
	})

	t.Run("error messages don't contain sensitive data", func(t *testing.T) {
		apiKey := "sk-abc123-secret"
		provider, _ := NewOpenAIProvider(apiKey, "http://invalid:99999", "gpt-4", 100*time.Millisecond)
		defer provider.Close()

		_, err := provider.makeChatCompletion(context.Background(), []Message{{Role: "user", Content: "test"}})

		if err != nil && strings.Contains(err.Error(), apiKey) {
			t.Error("error contains sensitive API key")
		}
	})
}

// =============================================================================
// TEST: Close
// =============================================================================

func TestClose(t *testing.T) {
	t.Run("cleanup", func(t *testing.T) {
		provider, _ := NewOpenAIProvider("sk-test", "http://localhost", "gpt-4", 30*time.Second)

		err := provider.Close()
		if err != nil {
			t.Errorf("expected no error on close, got %v", err)
		}

		// Should be able to close multiple times
		err = provider.Close()
		if err != nil {
			t.Errorf("expected no error on second close, got %v", err)
		}
	})

	t.Run("subsequent operations fail after close", func(t *testing.T) {
		provider, _ := NewOpenAIProvider("sk-test", "http://localhost", "gpt-4", 30*time.Second)
		provider.Close()

		// HealthCheck should fail
		if err := provider.HealthCheck(); err == nil {
			t.Error("expected HealthCheck to fail after close")
		}

		// Classify should return fallback
		classification, _ := provider.Classify(context.Background(), Input{})
		if !classification.IsDeterministicFallback {
			t.Error("expected fallback classification after close")
		}
		if classification.Reasoning != "provider is closed" {
			t.Errorf("expected 'provider is closed' reasoning, got %s", classification.Reasoning)
		}

		// Explain should return fallback
		explanation, _ := provider.Explain(context.Background(), Anomaly{})
		if !explanation.IsDeterministicFallback {
			t.Error("expected fallback explanation after close")
		}

		// Suggest should return fallback
		suggestion, _ := provider.Suggest(context.Background(), Scenario{})
		if !suggestion.IsDeterministicFallback {
			t.Error("expected fallback suggestion after close")
		}
	})

	t.Run("makeChatCompletion fails after close", func(t *testing.T) {
		provider, _ := NewOpenAIProvider("sk-test", "http://localhost", "gpt-4", 30*time.Second)
		provider.Close()

		_, err := provider.makeChatCompletion(context.Background(), []Message{})
		if err == nil {
			t.Error("expected makeChatCompletion to fail after close")
		}
		if !strings.Contains(err.Error(), "closed") {
			t.Errorf("expected 'closed' in error, got %v", err)
		}
	})
}

// =============================================================================
// TEST: Provider Type
// =============================================================================

func TestProviderType(t *testing.T) {
	provider, _ := NewOpenAIProvider("sk-test", "http://localhost", "gpt-4", 30*time.Second)
	defer provider.Close()

	if provider.Type() != ProviderAPI {
		t.Errorf("expected type ProviderAPI, got %v", provider.Type())
	}
}

// =============================================================================
// TEST: Circuit State String
// =============================================================================

func TestCircuitStateString(t *testing.T) {
	tests := []struct {
		state    circuitState
		expected string
	}{
		{circuitClosed, "closed"},
		{circuitOpen, "open"},
		{circuitHalfOpen, "half-open"},
		{circuitState(99), "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			if tc.state.String() != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, tc.state.String())
			}
		})
	}
}

// =============================================================================
// TEST: Response Validation
// =============================================================================

func TestResponseValidation(t *testing.T) {
	t.Run("confidence clamping high", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ChatCompletionResponse{
				Choices: []Choice{
					{
						Message: Message{
							Content: `{"category": "normal", "confidence": 1.5, "reasoning": "test"}`,
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		provider, _ := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		defer provider.Close()

		classification, _ := provider.Classify(context.Background(), Input{})

		// Confidence should be clamped to 0.5
		if classification.Confidence != 0.5 {
			t.Errorf("expected confidence clamped to 0.5, got %f", classification.Confidence)
		}
	})

	t.Run("confidence clamping low", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ChatCompletionResponse{
				Choices: []Choice{
					{
						Message: Message{
							Content: `{"category": "normal", "confidence": -0.5, "reasoning": "test"}`,
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		provider, _ := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		defer provider.Close()

		classification, _ := provider.Classify(context.Background(), Input{})

		// Confidence should be clamped to 0.5
		if classification.Confidence != 0.5 {
			t.Errorf("expected confidence clamped to 0.5, got %f", classification.Confidence)
		}
	})
}

// =============================================================================
// TEST: Plain Text Fallback
// =============================================================================

func TestPlainTextFallback(t *testing.T) {
	t.Run("classify plain text fallback", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ChatCompletionResponse{
				Choices: []Choice{
					{
						Message: Message{
							Content: "This looks like a critical issue based on the metrics",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		provider, _ := NewOpenAIProvider("sk-test", server.URL, "gpt-4", 30*time.Second)
		defer provider.Close()

		classification, _ := provider.Classify(context.Background(), Input{})

		if classification.Category != "critical" {
			t.Errorf("expected category 'critical' from plain text, got %s", classification.Category)
		}
		if classification.Confidence != 0.5 {
			t.Errorf("expected confidence 0.5 for fallback, got %f", classification.Confidence)
		}
		if !classification.IsDeterministicFallback {
			t.Error("expected IsDeterministicFallback to be true for plain text")
		}
	})
}
