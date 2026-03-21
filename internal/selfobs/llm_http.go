package selfobs

// =============================================================================
// OPENAI HTTP PROVIDER - PRODUCTION-READY LLM CLIENT
// =============================================================================
//
// This file implements a production-ready HTTP client for OpenAI-compatible
// LLM APIs. It provides robust error handling, retry logic, circuit breaker
// pattern, and comprehensive safety features.
//
// SAFETY FEATURES:
//   - Input redaction before sending (reuses existing redactor)
//   - Response validation (check for empty/invalid responses)
//   - Timeout enforcement via context
//   - Retry with exponential backoff (max 3 retries)
//   - Circuit breaker pattern for failing endpoints
//   - API key never logged (redacted in errors)
//
// ERROR HANDLING:
//   - Distinguishes between transient (retryable) and permanent errors
//   - HTTP status code handling (429 rate limit, 5xx server errors, etc.)
//   - Structured error messages
//
// USAGE:
//   provider, err := NewOpenAIProvider(apiKey, "https://api.openai.com/v1", "gpt-4", 30*time.Second)
//   if err != nil {
//       // handle error
//   }
//   defer provider.Close()
//
//   classification, err := provider.Classify(ctx, input)
//
// =============================================================================

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// DefaultOpenAIBaseURL is the default OpenAI API base URL
	DefaultOpenAIBaseURL = "https://api.openai.com/v1"

	// DefaultOpenAIModel is the default model to use
	DefaultOpenAIModel = "gpt-4"

	// MaxRetries is the maximum number of retries for failed requests
	MaxRetries = 3

	// CircuitBreakerThreshold is the number of consecutive failures before opening circuit
	CircuitBreakerThreshold = 5

	// CircuitBreakerTimeout is how long the circuit stays open
	CircuitBreakerTimeout = 30 * time.Second

	// DefaultRequestTimeout is the default HTTP request timeout
	DefaultRequestTimeout = 30 * time.Second
)

// =============================================================================
// CIRCUIT BREAKER STATE
// =============================================================================

type circuitState int

const (
	circuitClosed circuitState = iota
	circuitOpen
	circuitHalfOpen
)

func (s circuitState) String() string {
	switch s {
	case circuitClosed:
		return "closed"
	case circuitOpen:
		return "open"
	case circuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// =============================================================================
// REQUEST/RESPONSE TYPES
// =============================================================================

// Message represents a message in the chat completion format
type Message struct {
	// Role is the message role: "system", "user", "assistant", "tool"
	Role string `json:"role"`

	// Content is the message content
	Content string `json:"content"`
}

// ChatCompletionRequest represents an OpenAI chat completion request
type ChatCompletionRequest struct {
	// Model is the model to use for completion
	Model string `json:"model"`

	// Messages is the list of messages for the conversation
	Messages []Message `json:"messages"`

	// Temperature controls randomness (0.0 - 2.0)
	Temperature float64 `json:"temperature,omitempty"`

	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int `json:"max_tokens,omitempty"`

	// TopP controls diversity via nucleus sampling
	TopP float64 `json:"top_p,omitempty"`

	// Stream indicates whether to stream the response
	Stream bool `json:"stream,omitempty"`
}

// Choice represents a response choice from the LLM
type Choice struct {
	// Index is the choice index
	Index int `json:"index"`

	// Message is the generated message
	Message Message `json:"message"`

	// FinishReason is why the generation stopped
	FinishReason string `json:"finish_reason"`
}

// ChatCompletionResponse represents an OpenAI chat completion response
type ChatCompletionResponse struct {
	// ID is the unique identifier for the completion
	ID string `json:"id"`

	// Object is the object type
	Object string `json:"object"`

	// Created is the Unix timestamp when the completion was created
	Created int64 `json:"created"`

	// Model is the model used for completion
	Model string `json:"model"`

	// Choices is the list of completion choices
	Choices []Choice `json:"choices"`

	// Usage contains token usage information
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`

	// Error contains error information if the request failed
	Error *APIError `json:"error,omitempty"`
}

// APIError represents an error response from the API
type APIError struct {
	// Message is the error message
	Message string `json:"message"`

	// Type is the error type
	Type string `json:"type"`

	// Param is the parameter that caused the error
	Param string `json:"param,omitempty"`

	// Code is the error code
	Code string `json:"code,omitempty"`
}

// Error implements the error interface for APIError
func (e *APIError) Error() string {
	if e == nil {
		return "unknown API error"
	}
	return fmt.Sprintf("API error (type=%s, code=%s): %s", e.Type, e.Code, e.Message)
}

// =============================================================================
// ERRORS
// =============================================================================

var (
	// ErrCircuitBreakerOpen indicates the circuit breaker is open
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")

	// ErrInvalidResponse indicates an invalid or empty response from the API
	ErrInvalidResponse = errors.New("invalid or empty response from API")

	// ErrRateLimited indicates the request was rate limited
	ErrRateLimited = errors.New("rate limited by API")

	// ErrAuthenticationFailed indicates authentication failed
	ErrAuthenticationFailed = errors.New("API authentication failed")

	// ErrInvalidRequest indicates the request was invalid
	ErrInvalidRequest = errors.New("invalid request")
)

// =============================================================================
// OPENAI PROVIDER
// =============================================================================

// OpenAIProvider implements LLMProvider for OpenAI-compatible APIs.
// It provides a production-ready HTTP client with retry logic,
// circuit breaker pattern, and comprehensive error handling.
type OpenAIProvider struct {
	// apiKey is the API key for authentication (never logged)
	apiKey string

	// baseURL is the API base URL
	baseURL string

	// model is the model to use for completions
	model string

	// timeout is the HTTP client timeout
	timeout time.Duration

	// httpClient is the HTTP client for making requests
	httpClient *http.Client

	// redactor handles privacy-preserving redaction
	redactor *redactor

	// circuit breaker state
	circuitState    circuitState
	circuitMu       sync.RWMutex
	failureCount    int
	lastFailureTime time.Time

	// closed indicates if the provider is closed
	closed  bool
	closeMu sync.RWMutex
}

// NewOpenAIProvider creates a new OpenAI-compatible LLM provider.
//
// Parameters:
//   - apiKey: API key for authentication (required)
//   - baseURL: API base URL (empty string uses default OpenAI URL)
//   - model: Model name (empty string uses default "gpt-4")
//   - timeout: HTTP client timeout (zero uses default 30s)
//
// Returns an error if the API key is empty.
//
// Example:
//
//	provider, err := NewOpenAIProvider(
//	    "sk-...",
//	    "https://api.openai.com/v1",
//	    "gpt-4",
//	    30*time.Second,
//	)
func NewOpenAIProvider(apiKey, baseURL, model string, timeout time.Duration) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("%w: API key is required", ErrInvalidConfig)
	}

	if baseURL == "" {
		baseURL = DefaultOpenAIBaseURL
	}

	// Ensure baseURL doesn't have trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	if model == "" {
		model = DefaultOpenAIModel
	}

	if timeout == 0 {
		timeout = DefaultRequestTimeout
	}

	return &OpenAIProvider{
		apiKey:       apiKey,
		baseURL:      baseURL,
		model:        model,
		timeout:      timeout,
		httpClient:   &http.Client{Timeout: timeout},
		redactor:     newRedactor(),
		circuitState: circuitClosed,
	}, nil
}

// Type returns the provider type.
// Returns ProviderAPI for OpenAI-compatible providers.
func (p *OpenAIProvider) Type() ProviderType {
	return ProviderAPI
}

// Close cleans up resources held by the provider.
// After calling Close, the provider should not be used.
func (p *OpenAIProvider) Close() error {
	p.closeMu.Lock()
	defer p.closeMu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	p.httpClient.CloseIdleConnections()
	return nil
}

// isClosed returns whether the provider is closed
func (p *OpenAIProvider) isClosed() bool {
	p.closeMu.RLock()
	defer p.closeMu.RUnlock()
	return p.closed
}

// =============================================================================
// CIRCUIT BREAKER
// =============================================================================

// checkCircuit checks if the circuit breaker allows the request
func (p *OpenAIProvider) checkCircuit() error {
	p.circuitMu.Lock()
	defer p.circuitMu.Unlock()

	switch p.circuitState {
	case circuitClosed:
		return nil
	case circuitOpen:
		if time.Since(p.lastFailureTime) > CircuitBreakerTimeout {
			p.circuitState = circuitHalfOpen
			p.failureCount = 0
			return nil
		}
		return fmt.Errorf("%w: circuit has been open since %s", ErrCircuitBreakerOpen, p.lastFailureTime.Format(time.RFC3339))
	case circuitHalfOpen:
		return nil
	}

	return nil
}

// recordSuccess records a successful request
func (p *OpenAIProvider) recordSuccess() {
	p.circuitMu.Lock()
	defer p.circuitMu.Unlock()

	p.failureCount = 0
	p.circuitState = circuitClosed
}

// recordFailure records a failed request
func (p *OpenAIProvider) recordFailure() {
	p.circuitMu.Lock()
	defer p.circuitMu.Unlock()

	p.failureCount++
	p.lastFailureTime = time.Now()

	if p.failureCount >= CircuitBreakerThreshold {
		p.circuitState = circuitOpen
	}
}

// =============================================================================
// HTTP REQUEST HANDLING
// =============================================================================

// isRetryableError determines if an error is transient and can be retried
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific error types
	if errors.Is(err, ErrRateLimited) {
		return true
	}

	if errors.Is(err, ErrCircuitBreakerOpen) {
		return true
	}

	// Check for context errors (not retryable)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}

	// Check for HTTP status codes that indicate retryability
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case "rate_limit_exceeded", "timeout", "temporarily_unavailable":
			return true
		case "invalid_api_key", "insufficient_quota", "invalid_request_error":
			return false
		}
	}

	return false
}

// doRequest performs an HTTP request with retry logic
func (p *OpenAIProvider) doRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		// Check circuit breaker
		if err := p.checkCircuit(); err != nil {
			lastErr = err
			if !isRetryableError(err) {
				return nil, err
			}
			// Wait before retrying
			if attempt < MaxRetries {
				time.Sleep(p.backoffDuration(attempt))
				continue
			}
			return nil, err
		}

		// Clone request for retry
		reqClone := req.Clone(ctx)

		// Perform request
		resp, err := p.httpClient.Do(reqClone)
		if err != nil {
			lastErr = err
			p.recordFailure()
			if !isRetryableError(err) || attempt == MaxRetries {
				return nil, err
			}
			time.Sleep(p.backoffDuration(attempt))
			continue
		}

		// Handle response status
		switch resp.StatusCode {
		case http.StatusOK:
			p.recordSuccess()
			return resp, nil
		case http.StatusTooManyRequests:
			p.recordFailure()
			lastErr = ErrRateLimited
			resp.Body.Close()
			if attempt < MaxRetries {
				time.Sleep(p.backoffDuration(attempt))
				continue
			}
			return nil, ErrRateLimited
		case http.StatusUnauthorized:
			p.recordFailure()
			resp.Body.Close()
			return nil, ErrAuthenticationFailed
		case http.StatusBadRequest:
			p.recordFailure()
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("%w: %s", ErrInvalidRequest, string(body))
		case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			p.recordFailure()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			resp.Body.Close()
			if attempt < MaxRetries {
				time.Sleep(p.backoffDuration(attempt))
				continue
			}
			return nil, lastErr
		default:
			p.recordFailure()
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected status code: %d - %s", resp.StatusCode, string(body))
		}
	}

	return nil, lastErr
}

// backoffDuration calculates the backoff duration for a given attempt
func (p *OpenAIProvider) backoffDuration(attempt int) time.Duration {
	// Exponential backoff: 100ms, 200ms, 400ms
	duration := time.Duration(1<<attempt) * 100 * time.Millisecond
	// Add jitter
	return duration + time.Duration(attempt)*10*time.Millisecond
}

// makeChatCompletion makes a chat completion request
func (p *OpenAIProvider) makeChatCompletion(ctx context.Context, messages []Message) (*ChatCompletionResponse, error) {
	if p.isClosed() {
		return nil, errors.New("provider is closed")
	}

	// Build request body
	reqBody := ChatCompletionRequest{
		Model:       p.model,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   500,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/chat/completions", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("User-Agent", "mel-llm-client/1.0")

	// Perform request with retries
	resp, err := p.doRequest(ctx, req)
	if err != nil {
		// Redact API key from error message
		errMsg := err.Error()
		errMsg = strings.ReplaceAll(errMsg, p.apiKey, "[REDACTED]")
		return nil, errors.New(errMsg)
	}
	defer resp.Body.Close()

	// Read and parse response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API error in response body
	if chatResp.Error != nil {
		return nil, chatResp.Error
	}

	// Validate response
	if len(chatResp.Choices) == 0 {
		return nil, ErrInvalidResponse
	}

	return &chatResp, nil
}

// =============================================================================
// LLMProvider INTERFACE IMPLEMENTATION
// =============================================================================

// Classify requests a classification from the LLM.
// The input is redacted before sending to the provider.
// Returns a Classification with the LLM's assessment.
//
// On error, returns a fallback classification with IsDeterministicFallback=true.
func (p *OpenAIProvider) Classify(ctx context.Context, input Input) (Classification, error) {
	if p.isClosed() {
		return Classification{
			Category:                "unknown",
			Confidence:              0.0,
			Reasoning:               "provider is closed",
			IsDeterministicFallback: true,
		}, nil
	}

	// Redact input for privacy
	redactedInput := p.redactor.RedactInput(input)

	// Build prompt
	systemPrompt := `You are a classification assistant. Analyze the input and classify it into one of: "normal", "warning", "critical", or "unknown".
Respond with JSON only: {"category": "...", "confidence": 0.0-1.0, "reasoning": "brief explanation"}`

	userPrompt := fmt.Sprintf("Classify this input: %s", redactedInput)

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	// Make request
	resp, err := p.makeChatCompletion(ctx, messages)
	if err != nil {
		return Classification{
			Category:                "unknown",
			Confidence:              0.0,
			Reasoning:               fmt.Sprintf("API error: %v", err),
			IsDeterministicFallback: true,
		}, nil
	}

	// Parse response content as JSON
	content := resp.Choices[0].Message.Content
	content = strings.TrimSpace(content)

	// Try to extract JSON from markdown code blocks
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

	var result struct {
		Category   string  `json:"category"`
		Confidence float64 `json:"confidence"`
		Reasoning  string  `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// Fallback: try to parse as plain text
		category := "unknown"
		contentLower := strings.ToLower(content)
		if strings.Contains(contentLower, "critical") {
			category = "critical"
		} else if strings.Contains(contentLower, "warning") {
			category = "warning"
		} else if strings.Contains(contentLower, "normal") {
			category = "normal"
		}

		return Classification{
			Category:                category,
			Confidence:              0.5,
			Reasoning:               content,
			IsDeterministicFallback: true,
		}, nil
	}

	// Validate confidence
	if result.Confidence < 0.0 || result.Confidence > 1.0 {
		result.Confidence = 0.5
	}

	return Classification{
		Category:                result.Category,
		Confidence:              result.Confidence,
		Reasoning:               result.Reasoning,
		IsDeterministicFallback: false,
	}, nil
}

// Explain requests an explanation for an anomaly from the LLM.
// The anomaly is redacted before sending to the provider.
// Returns an Explanation with the LLM's analysis.
//
// On error, returns a fallback explanation with IsDeterministicFallback=true.
func (p *OpenAIProvider) Explain(ctx context.Context, anomaly Anomaly) (Explanation, error) {
	if p.isClosed() {
		return Explanation{
			Summary:                 "provider is closed",
			PossibleCauses:          []string{},
			Confidence:              0.0,
			IsDeterministicFallback: true,
		}, nil
	}

	// Redact anomaly for privacy
	redactedAnomaly := p.redactor.RedactAnomaly(anomaly)

	// Build prompt
	systemPrompt := `You are an anomaly explanation assistant. Analyze the anomaly and provide a brief explanation and possible causes.
Respond with JSON only: {"summary": "brief explanation", "possible_causes": ["cause1", "cause2"], "confidence": 0.0-1.0}`

	userPrompt := fmt.Sprintf("Explain this anomaly: %s", redactedAnomaly)

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	// Make request
	resp, err := p.makeChatCompletion(ctx, messages)
	if err != nil {
		return Explanation{
			Summary:                 fmt.Sprintf("API error: %v", err),
			PossibleCauses:          []string{},
			Confidence:              0.0,
			IsDeterministicFallback: true,
		}, nil
	}

	// Parse response content as JSON
	content := resp.Choices[0].Message.Content
	content = strings.TrimSpace(content)

	// Try to extract JSON from markdown code blocks
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

	var result struct {
		Summary        string   `json:"summary"`
		PossibleCauses []string `json:"possible_causes"`
		Confidence     float64  `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// Fallback: use content as summary
		return Explanation{
			Summary:                 content,
			PossibleCauses:          []string{"unable to parse structured response"},
			Confidence:              0.5,
			IsDeterministicFallback: true,
		}, nil
	}

	// Validate confidence
	if result.Confidence < 0.0 || result.Confidence > 1.0 {
		result.Confidence = 0.5
	}

	return Explanation{
		Summary:                 result.Summary,
		PossibleCauses:          result.PossibleCauses,
		Confidence:              result.Confidence,
		IsDeterministicFallback: false,
	}, nil
}

// Suggest requests suggestions for a scenario from the LLM.
// The scenario is redacted before sending to the provider.
// Returns a Suggestion with the LLM's recommendations.
//
// On error, returns a fallback suggestion with IsDeterministicFallback=true.
func (p *OpenAIProvider) Suggest(ctx context.Context, scenario Scenario) (Suggestion, error) {
	if p.isClosed() {
		return Suggestion{
			Actions:                 []string{},
			Priority:                "unknown",
			Risks:                   []string{},
			Confidence:              0.0,
			IsDeterministicFallback: true,
		}, nil
	}

	// Redact scenario for privacy
	redactedScenario := p.redactor.RedactScenario(scenario)

	// Build prompt
	systemPrompt := `You are a suggestions assistant. Analyze the scenario and suggest actions to achieve the goal.
Respond with JSON only: {"actions": ["action1", "action2"], "priority": "low|medium|high", "risks": ["risk1"], "confidence": 0.0-1.0}`

	userPrompt := fmt.Sprintf("Provide suggestions for this scenario: %s", redactedScenario)

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	// Make request
	resp, err := p.makeChatCompletion(ctx, messages)
	if err != nil {
		return Suggestion{
			Actions:                 []string{},
			Priority:                "unknown",
			Risks:                   []string{fmt.Sprintf("API error: %v", err)},
			Confidence:              0.0,
			IsDeterministicFallback: true,
		}, nil
	}

	// Parse response content as JSON
	content := resp.Choices[0].Message.Content
	content = strings.TrimSpace(content)

	// Try to extract JSON from markdown code blocks
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

	var result struct {
		Actions    []string `json:"actions"`
		Priority   string   `json:"priority"`
		Risks      []string `json:"risks"`
		Confidence float64  `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// Fallback: use content as single action
		return Suggestion{
			Actions:                 []string{content},
			Priority:                "unknown",
			Risks:                   []string{"unable to parse structured response"},
			Confidence:              0.5,
			IsDeterministicFallback: true,
		}, nil
	}

	// Validate confidence
	if result.Confidence < 0.0 || result.Confidence > 1.0 {
		result.Confidence = 0.5
	}

	return Suggestion{
		Actions:                 result.Actions,
		Priority:                result.Priority,
		Risks:                   result.Risks,
		Confidence:              result.Confidence,
		IsDeterministicFallback: false,
	}, nil
}

// HealthCheck verifies the LLM provider is available.
// Returns nil if healthy, error otherwise.
func (p *OpenAIProvider) HealthCheck() error {
	if p.isClosed() {
		return errors.New("provider is closed")
	}

	// Check circuit breaker state
	p.circuitMu.RLock()
	state := p.circuitState
	p.circuitMu.RUnlock()

	if state == circuitOpen {
		return ErrCircuitBreakerOpen
	}

	// Perform a simple models list request to verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/models", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// =============================================================================
// CIRCUIT BREAKER STATUS (PUBLIC)
// =============================================================================

// CircuitStatus returns the current state of the circuit breaker
func (p *OpenAIProvider) CircuitStatus() string {
	p.circuitMu.RLock()
	defer p.circuitMu.RUnlock()
	return p.circuitState.String()
}

// FailureCount returns the current consecutive failure count
func (p *OpenAIProvider) FailureCount() int {
	p.circuitMu.RLock()
	defer p.circuitMu.RUnlock()
	return p.failureCount
}

// ResetCircuit resets the circuit breaker to closed state
func (p *OpenAIProvider) ResetCircuit() {
	p.circuitMu.Lock()
	defer p.circuitMu.Unlock()
	p.circuitState = circuitClosed
	p.failureCount = 0
}
