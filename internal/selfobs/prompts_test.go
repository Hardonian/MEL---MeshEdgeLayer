package selfobs

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPromptEngineCreation(t *testing.T) {
	engine := NewPromptEngine()
	if engine == nil {
		t.Fatal("NewPromptEngine() returned nil")
	}
	if engine.templates == nil {
		t.Fatal("templates map not initialized")
	}

	// Test built-in templates registered
	expectedTemplates := []string{"classification", "explanation", "suggestion", "anomaly_detection"}
	for _, name := range expectedTemplates {
		template, ok := engine.GetTemplate(name)
		if !ok {
			t.Errorf("built-in template '%s' not registered", name)
		}
		if template.Name != name {
			t.Errorf("template name mismatch: got %s, want %s", template.Name, name)
		}
	}
}

func TestRegisterTemplate(t *testing.T) {
	engine := NewPromptEngine()

	// Test valid template registration
	customTemplate := PromptTemplate{
		Name:               "custom",
		Description:        "A custom template",
		SystemPrompt:       "System prompt",
		UserPromptTemplate: "User {{Variable}}",
		Version:            "1.0",
		OutputSchema: map[string]string{
			"result": "string",
		},
	}

	err := engine.RegisterTemplate(customTemplate)
	if err != nil {
		t.Errorf("RegisterTemplate() error = %v", err)
	}

	template, ok := engine.GetTemplate("custom")
	if !ok {
		t.Error("custom template not found after registration")
	}
	if template.Name != "custom" {
		t.Errorf("template name = %v, want custom", template.Name)
	}

	// Test duplicate name error
	err = engine.RegisterTemplate(customTemplate)
	if err == nil {
		t.Error("RegisterTemplate() should return error for duplicate name")
	}
	if !strings.Contains(err.Error(), "already registered") {
		t.Errorf("error message should contain 'already registered', got: %v", err)
	}

	// Test invalid template - empty name
	invalidTemplate := PromptTemplate{
		Name: "",
	}
	err = engine.RegisterTemplate(invalidTemplate)
	if err == nil {
		t.Error("RegisterTemplate() should return error for empty name")
	}
}

func TestExecuteTemplate(t *testing.T) {
	engine := NewPromptEngine()

	// Test variable substitution
	variables := map[string]interface{}{
		"Component": "api-gateway",
		"ErrorRate": 5.5,
		"State":     "degraded",
	}

	result, err := engine.ExecuteTemplate("classification", variables)
	if err != nil {
		t.Errorf("ExecuteTemplate() error = %v", err)
	}

	if !strings.Contains(result, "api-gateway") {
		t.Error("result should contain Component value")
	}
	if !strings.Contains(result, "5.5000") {
		t.Error("result should contain formatted ErrorRate")
	}
	if !strings.Contains(result, "degraded") {
		t.Error("result should contain State value")
	}

	// Test all placeholders
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	fullVars := map[string]interface{}{
		"Component": "test-service",
		"ErrorRate": 10.5,
		"State":     "critical",
		"Timestamp": testTime,
		"Metrics": map[string]float64{
			"cpu":    85.5,
			"memory": 90.0,
		},
		"History": []string{"error1", "error2"},
	}

	result, err = engine.ExecuteTemplate("classification", fullVars)
	if err != nil {
		t.Errorf("ExecuteTemplate() error = %v", err)
	}

	if !strings.Contains(result, "test-service") {
		t.Error("result should contain Component")
	}
	if !strings.Contains(result, "2024-01-15T10:30:00Z") {
		t.Error("result should contain formatted Timestamp")
	}
	if !strings.Contains(result, "cpu=85.5000") {
		t.Error("result should contain formatted Metrics")
	}
	if !strings.Contains(result, "error1; error2") {
		t.Error("result should contain formatted History")
	}

	// Test missing variables - placeholder should remain
	partialVars := map[string]interface{}{
		"Component": "partial",
	}
	result, err = engine.ExecuteTemplate("classification", partialVars)
	if err != nil {
		t.Errorf("ExecuteTemplate() error = %v", err)
	}
	if !strings.Contains(result, "{{State}}") {
		t.Error("missing variable placeholder should remain in output")
	}

	// Test non-existent template
	_, err = engine.ExecuteTemplate("nonexistent", variables)
	if err == nil {
		t.Error("ExecuteTemplate() should return error for non-existent template")
	}

	// Test special character escaping
	maliciousVars := map[string]interface{}{
		"Component": "test\x00\x01\x02service",
	}
	result, err = engine.ExecuteTemplate("classification", maliciousVars)
	if err != nil {
		t.Errorf("ExecuteTemplate() error = %v", err)
	}
	if strings.Contains(result, "\x00") {
		t.Error("control characters should be sanitized")
	}
}

func TestBuildClassificationPrompt(t *testing.T) {
	engine := NewPromptEngine()

	input := LLMInput{
		Component: "test-component",
		ErrorRate: 15.5,
		State:     "failing",
		Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Metrics: map[string]float64{
			"cpu": 95.0,
		},
		History: []string{"timeout", "error"},
	}

	messages, err := engine.BuildClassificationPrompt(input)
	if err != nil {
		t.Errorf("BuildClassificationPrompt() error = %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}

	// Test system message
	if messages[0].Role != "system" {
		t.Errorf("first message role = %v, want system", messages[0].Role)
	}
	if messages[0].Content == "" {
		t.Error("system message content should not be empty")
	}

	// Test user message
	if messages[1].Role != "user" {
		t.Errorf("second message role = %v, want user", messages[1].Role)
	}
	if messages[1].Content == "" {
		t.Error("user message content should not be empty")
	}

	// Test input redaction - verify sensitive data is handled
	if !strings.Contains(messages[1].Content, "test-component") {
		t.Error("user content should contain sanitized component name")
	}

	// Test valid output structure
	if !strings.Contains(messages[1].Content, "15.5000") {
		t.Error("user content should contain formatted error rate")
	}
}

func TestBuildExplanationPrompt(t *testing.T) {
	engine := NewPromptEngine()

	anomaly := PromptAnomaly{
		Component: "database",
		Metric:    "latency",
		Value:     500.0,
		Expected:  100.0,
		Deviation: 400.0,
		Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	messages, err := engine.BuildExplanationPrompt(anomaly)
	if err != nil {
		t.Errorf("BuildExplanationPrompt() error = %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}

	// Test prompt structure
	if messages[0].Role != "system" {
		t.Errorf("system message role = %v, want system", messages[0].Role)
	}
	if messages[1].Role != "user" {
		t.Errorf("user message role = %v, want user", messages[1].Role)
	}

	// Test anomaly data inclusion
	content := messages[1].Content
	if !strings.Contains(content, "database") {
		t.Error("content should contain component")
	}
	if !strings.Contains(content, "latency") {
		t.Error("content should contain metric")
	}
	if !strings.Contains(content, "500.0000") {
		t.Error("content should contain value")
	}
	if !strings.Contains(content, "100.0000") {
		t.Error("content should contain expected value")
	}

	// Test sanitization
	maliciousAnomaly := PromptAnomaly{
		Component: "db\x00malicious",
		Metric:    "test",
		Value:     1.0,
	}
	messages, err = engine.BuildExplanationPrompt(maliciousAnomaly)
	if err != nil {
		t.Errorf("BuildExplanationPrompt() error = %v", err)
	}
	if strings.Contains(messages[1].Content, "\x00") {
		t.Error("content should be sanitized")
	}
}

func TestBuildSuggestionPrompt(t *testing.T) {
	engine := NewPromptEngine()

	scenario := PromptScenario{
		Component: "cache-service",
		Issue:     "high memory usage",
		Context: map[string]string{
			"threshold": "90%",
			"current":   "95%",
		},
		Constraints: []string{"no downtime", "budget limit"},
	}

	messages, err := engine.BuildSuggestionPrompt(scenario)
	if err != nil {
		t.Errorf("BuildSuggestionPrompt() error = %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}

	// Test prompt structure
	if messages[0].Role != "system" {
		t.Errorf("system message role = %v, want system", messages[0].Role)
	}
	if messages[1].Role != "user" {
		t.Errorf("user message role = %v, want user", messages[1].Role)
	}

	// Test scenario data inclusion
	content := messages[1].Content
	if !strings.Contains(content, "cache-service") {
		t.Error("content should contain component")
	}
	if !strings.Contains(content, "high memory usage") {
		t.Error("content should contain issue")
	}
	if !strings.Contains(content, "90%") {
		t.Error("content should contain context")
	}
	if !strings.Contains(content, "no downtime") {
		t.Error("content should contain constraints")
	}

	// Test sanitization
	maliciousScenario := PromptScenario{
		Component: "service\x01test",
		Issue:     "issue\x02test",
		Context: map[string]string{
			"key\x03": "value\x04",
		},
	}
	messages, err = engine.BuildSuggestionPrompt(maliciousScenario)
	if err != nil {
		t.Errorf("BuildSuggestionPrompt() error = %v", err)
	}
	if strings.Contains(messages[1].Content, "\x01") || strings.Contains(messages[1].Content, "\x02") {
		t.Error("content should be sanitized")
	}

	// Test empty context and constraints
	emptyScenario := PromptScenario{
		Component:   "empty-test",
		Issue:       "test",
		Context:     map[string]string{},
		Constraints: []string{},
	}
	messages, err = engine.BuildSuggestionPrompt(emptyScenario)
	if err != nil {
		t.Errorf("BuildSuggestionPrompt() error = %v", err)
	}
	if !strings.Contains(messages[1].Content, "none") {
		t.Error("empty context/constraints should show 'none'")
	}
}

func TestValidateOutput(t *testing.T) {
	engine := NewPromptEngine()

	// Test valid JSON output
	validOutput := `{"severity":"critical","category":"resource","confidence":0.95,"summary":"High CPU usage detected"}`
	result, err := engine.ValidateOutput("classification", validOutput)
	if err != nil {
		t.Errorf("ValidateOutput() error = %v", err)
	}
	if result["severity"] != "critical" {
		t.Errorf("severity = %v, want critical", result["severity"])
	}
	if result["confidence"] != "0.95" {
		t.Errorf("confidence = %v, want 0.95", result["confidence"])
	}

	// Test invalid JSON handling
	invalidOutput := `{invalid json}`
	_, err = engine.ValidateOutput("classification", invalidOutput)
	if err == nil {
		t.Error("ValidateOutput() should return error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "not valid JSON") {
		t.Errorf("error should mention JSON, got: %v", err)
	}

	// Test missing required fields
	missingFields := `{"severity":"critical","category":"resource"}`
	_, err = engine.ValidateOutput("classification", missingFields)
	if err == nil {
		t.Error("ValidateOutput() should return error for missing fields")
	}
	if !strings.Contains(err.Error(), "missing required field") {
		t.Errorf("error should mention missing field, got: %v", err)
	}

	// Test type validation - string field with number
	typeMismatch := `{"severity":123,"category":"resource","confidence":0.5,"summary":"test"}`
	_, err = engine.ValidateOutput("classification", typeMismatch)
	if err == nil {
		t.Error("ValidateOutput() should return error for type mismatch")
	}

	// Test non-existent template
	_, err = engine.ValidateOutput("nonexistent", validOutput)
	if err == nil {
		t.Error("ValidateOutput() should return error for non-existent template")
	}

	// Test number type validation
	explanationOutput := `{"cause":"network","likelihood":0.8,"related_factors":["factor1"],"recommendation":"restart"}`
	result, err = engine.ValidateOutput("explanation", explanationOutput)
	if err != nil {
		t.Errorf("ValidateOutput() error = %v", err)
	}
	if result["likelihood"] != "0.8" {
		t.Errorf("likelihood = %v, want 0.8", result["likelihood"])
	}

	// Test array type validation
	if _, ok := result["related_factors"]; !ok {
		t.Error("related_factors should be present")
	}

	// Test boolean type validation
	anomalyOutput := `{"is_anomaly":true,"anomalies":[],"severity":"high","pattern":"spike"}`
	result, err = engine.ValidateOutput("anomaly_detection", anomalyOutput)
	if err != nil {
		t.Errorf("ValidateOutput() error = %v", err)
	}
	if result["is_anomaly"] != "true" {
		t.Errorf("is_anomaly = %v, want true", result["is_anomaly"])
	}
}

func TestSanitization(t *testing.T) {
	// Test string sanitization - control characters
	tests := []struct {
		input    string
		expected string
	}{
		{"hello world", "hello world"},
		{"hello\x00world", "helloworld"},
		{"hello\x01\x02world", "helloworld"},
		{"hello\nworld", "hello\nworld"},
		{"hello\tworld", "hello\tworld"},
		{"normal text 123", "normal text 123"},
	}

	for _, tt := range tests {
		result := sanitizeString(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeString(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}

	// Test special character removal
	withControl := "test\x00\x01\x02\x03\x04\x05end"
	result := sanitizeString(withControl)
	if result != "testend" {
		t.Errorf("control characters not removed: got %q", result)
	}

	// Test length limits (sanitization should handle long strings)
	longString := strings.Repeat("a", 10000)
	result = sanitizeString(longString)
	if len(result) != 10000 {
		t.Errorf("long string length = %d, want 10000", len(result))
	}

	// Test unicode handling
	unicodeString := "Hello 世界 🌍"
	result = sanitizeString(unicodeString)
	if result != unicodeString {
		t.Errorf("unicode string altered: got %q", result)
	}
}

func TestGetTemplate(t *testing.T) {
	engine := NewPromptEngine()

	// Test retrieving existing template
	template, ok := engine.GetTemplate("classification")
	if !ok {
		t.Error("GetTemplate() should return true for existing template")
	}
	if template.Name != "classification" {
		t.Errorf("template name = %v, want classification", template.Name)
	}
	if template.SystemPrompt == "" {
		t.Error("template system prompt should not be empty")
	}

	// Test retrieving non-existent template
	template, ok = engine.GetTemplate("nonexistent")
	if ok {
		t.Error("GetTemplate() should return false for non-existent template")
	}
	if template.Name != "" {
		t.Error("template should be empty for non-existent")
	}
}

func TestMessageTypes(t *testing.T) {
	// Test PromptMessage struct creation
	msg := PromptMessage{
		Role:    "user",
		Content: "test message",
	}
	if msg.Role != "user" {
		t.Errorf("Role = %v, want user", msg.Role)
	}
	if msg.Content != "test message" {
		t.Errorf("Content = %v, want 'test message'", msg.Content)
	}

	// Test JSON marshaling
	msg = PromptMessage{
		Role:    "system",
		Content: "system prompt",
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Errorf("json.Marshal() error = %v", err)
	}

	expected := `{"role":"system","content":"system prompt"}`
	if string(data) != expected {
		t.Errorf("JSON = %s, want %s", string(data), expected)
	}

	// Test JSON unmarshaling
	jsonData := `{"role":"assistant","content":"response"}`
	var parsed PromptMessage
	err = json.Unmarshal([]byte(jsonData), &parsed)
	if err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
	}
	if parsed.Role != "assistant" {
		t.Errorf("Role = %v, want assistant", parsed.Role)
	}
	if parsed.Content != "response" {
		t.Errorf("Content = %v, want response", parsed.Content)
	}

	// Test array of messages marshaling
	messages := []PromptMessage{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "user msg"},
	}
	data, err = json.Marshal(messages)
	if err != nil {
		t.Errorf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(data), "system") {
		t.Error("JSON should contain system role")
	}
	if !strings.Contains(string(data), "user") {
		t.Error("JSON should contain user role")
	}

	// Test unmarshaling array
	var parsedMessages []PromptMessage
	err = json.Unmarshal(data, &parsedMessages)
	if err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
	}
	if len(parsedMessages) != 2 {
		t.Errorf("len = %d, want 2", len(parsedMessages))
	}
}
