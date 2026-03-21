package selfobs

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestResponseParserCreation(t *testing.T) {
	rp := NewResponseParser()
	if rp == nil {
		t.Fatal("NewResponseParser() returned nil")
	}

	if rp.minConfidence != DefaultMinConfidence {
		t.Errorf("expected minConfidence %v, got %v", DefaultMinConfidence, rp.minConfidence)
	}

	if rp.maxStringLength != DefaultMaxStringLength {
		t.Errorf("expected maxStringLength %v, got %v", DefaultMaxStringLength, rp.maxStringLength)
	}

	if len(rp.schemas) == 0 {
		t.Error("expected default schemas to be registered, got none")
	}

	expectedSchemas := []string{"classification", "explanation", "suggestion"}
	for _, schemaName := range expectedSchemas {
		if _, ok := rp.schemas[schemaName]; !ok {
			t.Errorf("expected schema %q to be registered", schemaName)
		}
	}
}

func TestParseClassification(t *testing.T) {
	rp := NewResponseParser()

	t.Run("valid JSON response", func(t *testing.T) {
		raw := `{"category": "infrastructure", "confidence": 0.8, "reasoning": "test reasoning", "severity": "high"}`
		result, err := rp.ParseClassification(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Category != "infrastructure" {
			t.Errorf("expected category 'infrastructure', got %q", result.Category)
		}
		if result.Confidence != 0.8 {
			t.Errorf("expected confidence 0.8, got %v", result.Confidence)
		}
		if result.Reasoning != "test reasoning" {
			t.Errorf("expected reasoning 'test reasoning', got %q", result.Reasoning)
		}
		if result.Severity != "high" {
			t.Errorf("expected severity 'high', got %q", result.Severity)
		}
	})

	t.Run("response with markdown code blocks", func(t *testing.T) {
		raw := "```json\n{\"category\": \"transient\", \"confidence\": 0.75, \"reasoning\": \"network issue\"}\n```"
		result, err := rp.ParseClassification(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Category != "transient" {
			t.Errorf("expected category 'transient', got %q", result.Category)
		}
		if result.Confidence != 0.75 {
			t.Errorf("expected confidence 0.75, got %v", result.Confidence)
		}
	})

	t.Run("invalid JSON handling", func(t *testing.T) {
		raw := `{invalid json}`
		_, err := rp.ParseClassification(raw)
		if err == nil {
			t.Error("expected error for invalid JSON, got nil")
		}
		if !IsParseError(err) {
			t.Errorf("expected ParseError, got %T", err)
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		raw := `{"category": "infrastructure"}`
		_, err := rp.ParseClassification(raw)
		if err == nil {
			t.Error("expected error for missing required fields, got nil")
		}
	})

	t.Run("confidence normalization", func(t *testing.T) {
		tests := []struct {
			input    float64
			expected float64
		}{
			{1.5, 1.0},
			{0.8, 0.8},
			{1.0, 1.0},
		}

		for _, tc := range tests {
			raw := `{"category": "infrastructure", "confidence": ` + formatFloat(tc.input) + `}`
			result, err := rp.ParseClassification(raw)
			if err != nil {
				t.Fatalf("unexpected error for confidence %v: %v", tc.input, err)
			}
			if result.Confidence != tc.expected {
				t.Errorf("confidence normalization: input %v, expected %v, got %v", tc.input, tc.expected, result.Confidence)
			}
		}
	})

	t.Run("category normalization", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"INFRASTRUCTURE", "infrastructure"},
			{"Transient", "transient"},
			{"Configuration", "configuration"},
			{"UNKNOWN", "unknown"},
		}

		for _, tc := range tests {
			raw := `{"category": "` + tc.input + `", "confidence": 0.8}`
			result, err := rp.ParseClassification(raw)
			if err != nil {
				t.Fatalf("unexpected error for category %q: %v", tc.input, err)
			}
			if result.Category != tc.expected {
				t.Errorf("category normalization: input %q, expected %q, got %q", tc.input, tc.expected, result.Category)
			}
		}
	})
}

func TestParseExplanation(t *testing.T) {
	rp := NewResponseParser()

	t.Run("valid response parsing", func(t *testing.T) {
		raw := `{"summary": "system overload", "root_cause": "high traffic", "confidence": 0.85}`
		result, err := rp.ParseExplanation(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Summary != "system overload" {
			t.Errorf("expected summary 'system overload', got %q", result.Summary)
		}
		if result.RootCause != "high traffic" {
			t.Errorf("expected root_cause 'high traffic', got %q", result.RootCause)
		}
		if result.Confidence != 0.85 {
			t.Errorf("expected confidence 0.85, got %v", result.Confidence)
		}
	})

	t.Run("contributing factors extraction", func(t *testing.T) {
		raw := `{"summary": "test", "contributing_factors": ["factor1", "factor2", "factor3"], "confidence": 0.8}`
		result, err := rp.ParseExplanation(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.ContributingFactors) != 3 {
			t.Errorf("expected 3 contributing factors, got %d", len(result.ContributingFactors))
		}

		expected := []string{"factor1", "factor2", "factor3"}
		for i, exp := range expected {
			if result.ContributingFactors[i] != exp {
				t.Errorf("contributing factor %d: expected %q, got %q", i, exp, result.ContributingFactors[i])
			}
		}
	})

	t.Run("confidence validation", func(t *testing.T) {
		raw := `{"summary": "test", "confidence": 0.3}`
		_, err := rp.ParseExplanation(raw)
		if err == nil {
			t.Error("expected error for low confidence, got nil")
		}
		if !IsLowConfidenceError(err) {
			t.Errorf("expected LowConfidenceError, got %T", err)
		}
	})
}

func TestParseSuggestion(t *testing.T) {
	rp := NewResponseParser()

	t.Run("valid response parsing", func(t *testing.T) {
		raw := `{"actions": [{"type": "restart", "target": "service1"}], "priority": "high", "risk_level": "medium"}`
		result, err := rp.ParseSuggestion(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Actions) != 1 {
			t.Errorf("expected 1 action, got %d", len(result.Actions))
		}
		if result.Priority != "high" {
			t.Errorf("expected priority 'high', got %q", result.Priority)
		}
		if result.RiskLevel != "medium" {
			t.Errorf("expected risk_level 'medium', got %q", result.RiskLevel)
		}
	})

	t.Run("actions extraction", func(t *testing.T) {
		raw := `{"actions": [{"type": "restart", "target": "service1", "parameters": {"timeout": "30"}}, {"type": "scale", "target": "deployment", "automated": false}], "priority": "critical"}`
		result, err := rp.ParseSuggestion(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Actions) != 2 {
			t.Fatalf("expected 2 actions, got %d", len(result.Actions))
		}

		action1 := result.Actions[0]
		if action1.Type != "restart" {
			t.Errorf("expected action1 type 'restart', got %q", action1.Type)
		}
		if action1.Target != "service1" {
			t.Errorf("expected action1 target 'service1', got %q", action1.Target)
		}
		if action1.Parameters["timeout"] != "30" {
			t.Errorf("expected action1 parameter timeout='30', got %q", action1.Parameters["timeout"])
		}
	})

	t.Run("automated flag parsing", func(t *testing.T) {
		raw := `{"actions": [{"type": "restart", "target": "service1", "automated": true}], "priority": "high"}`
		result, err := rp.ParseSuggestion(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Actions) != 1 {
			t.Fatalf("expected 1 action, got %d", len(result.Actions))
		}

		if !result.Actions[0].Automated {
			t.Error("expected automated flag to be true")
		}
	})

	t.Run("priority validation", func(t *testing.T) {
		tests := []struct {
			priority string
			expected string
		}{
			{"critical", "critical"},
			{"HIGH", "high"},
			{"Medium", "medium"},
			{"LOW", "low"},
		}

		for _, tc := range tests {
			raw := `{"actions": [{"type": "alert", "target": "test"}], "priority": "` + tc.priority + `"}`
			result, err := rp.ParseSuggestion(raw)
			if err != nil {
				t.Fatalf("unexpected error for priority %q: %v", tc.priority, err)
			}
			if result.Priority != tc.expected {
				t.Errorf("priority validation: input %q, expected %q, got %q", tc.priority, tc.expected, result.Priority)
			}
		}
	})
}

func TestExtractJSONFromText(t *testing.T) {
	rp := NewResponseParser()

	t.Run("extraction from markdown blocks", func(t *testing.T) {
		text := "Some text\n```json\n{\"key\": \"value\"}\n```\nMore text"
		result, err := rp.ExtractJSONFromText(text)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := `{"key": "value"}`
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("extraction from inline JSON", func(t *testing.T) {
		text := `Here is the result: {"key": "value"}`
		result, err := rp.ExtractJSONFromText(text)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := `{"key": "value"}`
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("handling non-JSON text", func(t *testing.T) {
		text := "This is just plain text with no JSON"
		_, err := rp.ExtractJSONFromText(text)
		if err == nil {
			t.Error("expected error for non-JSON text, got nil")
		}
	})

	t.Run("multiple JSON blocks take first", func(t *testing.T) {
		text := "```json\n{\"first\": true}\n```\nSome text\n```json\n{\"second\": true}\n```"
		result, err := rp.ExtractJSONFromText(text)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := `{"first": true}`
		if result != expected {
			t.Errorf("expected %q (first JSON block), got %q", expected, result)
		}
	})
}

func TestResponseParserValidateOutput(t *testing.T) {
	rp := NewResponseParser()

	t.Run("valid output against schema", func(t *testing.T) {
		data := map[string]interface{}{
			"category":   "infrastructure",
			"confidence": 0.8,
			"severity":   "high",
		}
		err := rp.ValidateOutput("classification", data)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("missing required fields", func(t *testing.T) {
		data := map[string]interface{}{
			"category": "infrastructure",
		}
		err := rp.ValidateOutput("classification", data)
		if err == nil {
			t.Error("expected error for missing required field, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("expected ValidationError, got %T", err)
		}
	})

	t.Run("invalid field types", func(t *testing.T) {
		data := map[string]interface{}{
			"category":   123,
			"confidence": "not a number",
		}
		err := rp.ValidateOutput("classification", data)
		if err == nil {
			t.Error("expected error for invalid field types, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("expected ValidationError, got %T", err)
		}
	})

	t.Run("disallowed values", func(t *testing.T) {
		data := map[string]interface{}{
			"category":   "invalid_category",
			"confidence": 0.8,
		}
		err := rp.ValidateOutput("classification", data)
		if err == nil {
			t.Error("expected error for disallowed category value, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("expected ValidationError, got %T", err)
		}
	})
}

func TestConfidenceThreshold(t *testing.T) {
	rp := NewResponseParser()

	tests := []struct {
		confidence float64
		shouldPass bool
	}{
		{0.9, true},
		{0.6, true},
		{0.5, true},
		{0.49, false},
		{0.1, false},
		{0.0, false},
		{1.0, true},
	}

	for _, tc := range tests {
		raw := `{"category": "infrastructure", "confidence": ` + formatFloat(tc.confidence) + `}`
		_, err := rp.ParseClassification(raw)

		if tc.shouldPass {
			if err != nil {
				t.Errorf("confidence %v should pass, but got error: %v", tc.confidence, err)
			}
		} else {
			if err == nil {
				t.Errorf("confidence %v should fail, but got no error", tc.confidence)
			} else if !IsLowConfidenceError(err) {
				t.Errorf("confidence %v: expected LowConfidenceError, got %T", tc.confidence, err)
			}
		}
	}
}

func TestErrorTypes(t *testing.T) {
	t.Run("ParseError", func(t *testing.T) {
		err := ParseError{Message: "test parse error", Cause: errors.New("cause")}
		if err.Error() != "parse error: test parse error: cause" {
			t.Errorf("unexpected error message: %v", err.Error())
		}
		if !IsParseError(err) {
			t.Error("IsParseError should return true")
		}
		if IsParseError(errors.New("not a parse error")) {
			t.Error("IsParseError should return false for non-ParseError")
		}
	})

	t.Run("ValidationError", func(t *testing.T) {
		err := ValidationError{Message: "test validation error", Field: "testField"}
		if !strings.Contains(err.Error(), "testField") {
			t.Errorf("error message should contain field name: %v", err.Error())
		}
		if !IsValidationError(err) {
			t.Error("IsValidationError should return true")
		}
		if IsValidationError(errors.New("not a validation error")) {
			t.Error("IsValidationError should return false for non-ValidationError")
		}
	})

	t.Run("LowConfidenceError", func(t *testing.T) {
		err := LowConfidenceError{Confidence: 0.3, Threshold: 0.5}
		if !strings.Contains(err.Error(), "0.30") {
			t.Errorf("error message should contain confidence: %v", err.Error())
		}
		if !strings.Contains(err.Error(), "0.50") {
			t.Errorf("error message should contain threshold: %v", err.Error())
		}
		if !IsLowConfidenceError(err) {
			t.Error("IsLowConfidenceError should return true")
		}
		if IsLowConfidenceError(errors.New("not a low confidence error")) {
			t.Error("IsLowConfidenceError should return false for non-LowConfidenceError")
		}
	})
}

func TestSanitizeAndValidate(t *testing.T) {
	rp := NewResponseParser()

	t.Run("valid input", func(t *testing.T) {
		raw := `{"category": "transient", "confidence": 0.7}`
		data, err := rp.SanitizeAndValidate(raw, "classification")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if data["category"] != "transient" {
			t.Errorf("expected category 'transient', got %v", data["category"])
		}
	})

	t.Run("truncation", func(t *testing.T) {
		rp.SetMaxStringLength(10)
		defer rp.SetMaxStringLength(DefaultMaxStringLength)

		longReasoning := strings.Repeat("a", 100)
		raw := `{"category": "transient", "confidence": 0.7, "reasoning": "` + longReasoning + `"}`
		_, err := rp.ParseClassification(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("field validation", func(t *testing.T) {
		raw := `{"invalid_field": "value"}`
		_, err := rp.SanitizeAndValidate(raw, "classification")
		if err == nil {
			t.Error("expected error for invalid field, got nil")
		}
	})
}

func TestSchemaRegistration(t *testing.T) {
	rp := NewResponseParser()

	t.Run("custom schema registration", func(t *testing.T) {
		customSchema := []ValidationRule{
			{Field: "custom_field", Type: FieldTypeString, Required: true},
			{Field: "count", Type: FieldTypeInt, Required: false},
		}

		rp.RegisterSchema("custom_type", customSchema)

		if _, ok := rp.schemas["custom_type"]; !ok {
			t.Error("custom schema should be registered")
		}
	})

	t.Run("schema retrieval via validation", func(t *testing.T) {
		customSchema := []ValidationRule{
			{Field: "name", Type: FieldTypeString, Required: true},
		}

		rp.RegisterSchema("test_type", customSchema)

		validData := map[string]interface{}{"name": "test"}
		err := rp.ValidateOutput("test_type", validData)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		invalidData := map[string]interface{}{}
		err = rp.ValidateOutput("test_type", invalidData)
		if err == nil {
			t.Error("expected error for missing required field")
		}
	})
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
