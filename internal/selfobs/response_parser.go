// Package selfobs provides structured output parsing and validation for LLM responses.
// It supports parsing classifications, explanations, and suggestions from raw LLM output,
// with built-in validation, safety features, and multiple parsing strategies.
package selfobs

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	// DefaultMinConfidence is the minimum acceptable confidence score (0-1 scale)
	DefaultMinConfidence = 0.5

	// DefaultMaxStringLength is the maximum allowed length for string fields
	DefaultMaxStringLength = 10000
)

// ParseError indicates a malformed response that could not be parsed
type ParseError struct {
	Message string
	Cause   error
}

func (e ParseError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("parse error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("parse error: %s", e.Message)
}

func (e ParseError) Unwrap() error {
	return e.Cause
}

// ValidationError indicates a response that does not match the expected schema
type ValidationError struct {
	Message string
	Field   string
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error: field %q: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// LowConfidenceError indicates a response with confidence below the threshold
type LowConfidenceError struct {
	Confidence float64
	Threshold  float64
}

func (e LowConfidenceError) Error() string {
	return fmt.Sprintf("confidence %.2f below threshold %.2f", e.Confidence, e.Threshold)
}

// FieldType represents the expected type of a field for validation purposes
type FieldType string

const (
	FieldTypeString  FieldType = "string"
	FieldTypeInt     FieldType = "int"
	FieldTypeFloat   FieldType = "float"
	FieldTypeBool    FieldType = "bool"
	FieldTypeArray   FieldType = "array"
	FieldTypeObject  FieldType = "object"
	FieldTypeMap     FieldType = "map"
)

// ValidationRule defines a single field validation constraint
type ValidationRule struct {
	Field         string
	Type          FieldType
	Required      bool
	AllowedValues []string
}

// SchemaValidator validates parsed output against a schema defined by validation rules
type SchemaValidator struct {
	rules []ValidationRule
}

// NewSchemaValidator creates a validator for the given rules
func NewSchemaValidator(rules []ValidationRule) *SchemaValidator {
	return &SchemaValidator{rules: rules}
}

// Validate checks the provided data against the schema rules
func (sv *SchemaValidator) Validate(data map[string]interface{}) error {
	for _, rule := range sv.rules {
		value, exists := data[rule.Field]

		if rule.Required && !exists {
			return ValidationError{Field: rule.Field, Message: "required field missing"}
		}

		if !exists {
			continue
		}

		if err := sv.validateType(rule.Field, value, rule.Type); err != nil {
			return err
		}

		if len(rule.AllowedValues) > 0 {
			strValue, ok := value.(string)
			if !ok {
				return ValidationError{Field: rule.Field, Message: "expected string for allowed values check"}
			}
			found := false
			for _, allowed := range rule.AllowedValues {
				if strings.EqualFold(strValue, allowed) {
					found = true
					break
				}
			}
			if !found {
				return ValidationError{
					Field:   rule.Field,
					Message: fmt.Sprintf("value %q not in allowed values %v", strValue, rule.AllowedValues),
				}
			}
		}
	}

	return nil
}

func (sv *SchemaValidator) validateType(field string, value interface{}, expectedType FieldType) error {
	switch expectedType {
	case FieldTypeString:
		if _, ok := value.(string); !ok {
			return ValidationError{Field: field, Message: fmt.Sprintf("expected string, got %T", value)}
		}
	case FieldTypeInt:
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			// ok
		case float64:
			// JSON numbers unmarshal as float64
		default:
			return ValidationError{Field: field, Message: fmt.Sprintf("expected int, got %T", value)}
		}
	case FieldTypeFloat:
		switch value.(type) {
		case float32, float64:
			// ok
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			// ok, will be converted
		default:
			return ValidationError{Field: field, Message: fmt.Sprintf("expected float, got %T", value)}
		}
	case FieldTypeBool:
		if _, ok := value.(bool); !ok {
			return ValidationError{Field: field, Message: fmt.Sprintf("expected bool, got %T", value)}
		}
	case FieldTypeArray:
		if _, ok := value.([]interface{}); !ok {
			return ValidationError{Field: field, Message: fmt.Sprintf("expected array, got %T", value)}
		}
	case FieldTypeObject:
		if _, ok := value.(map[string]interface{}); !ok {
			return ValidationError{Field: field, Message: fmt.Sprintf("expected object, got %T", value)}
		}
	case FieldTypeMap:
		if _, ok := value.(map[string]interface{}); !ok {
			return ValidationError{Field: field, Message: fmt.Sprintf("expected map, got %T", value)}
		}
	}
	return nil
}

// ParsedClassification represents a classification result from an LLM response
type ParsedClassification struct {
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
	Severity   string  `json:"severity"`
}

// ParsedExplanation represents an explanation result from an LLM response
type ParsedExplanation struct {
	Summary             string   `json:"summary"`
	RootCause           string   `json:"root_cause"`
	ContributingFactors []string `json:"contributing_factors"`
	Confidence          float64  `json:"confidence"`
}

// SuggestedAction represents a single action within a suggestion
type SuggestedAction struct {
	Type       string            `json:"type"`
	Target     string            `json:"target"`
	Parameters map[string]string `json:"parameters"`
	Automated  bool              `json:"automated"`
}

// ParsedSuggestion represents a suggestion result from an LLM response
type ParsedSuggestion struct {
	Actions         []SuggestedAction `json:"actions"`
	Priority        string            `json:"priority"`
	RiskLevel       string            `json:"risk_level"`
	ExpectedOutcome string            `json:"expected_outcome"`
}

// ResponseParser handles structured output parsing with schema validation
type ResponseParser struct {
	schemas           map[string][]ValidationRule
	minConfidence     float64
	maxStringLength   int
	jsonCodeBlockRe   *regexp.Regexp
	keyValueRe        *regexp.Regexp
}

// NewResponseParser creates a new ResponseParser with default settings
func NewResponseParser() *ResponseParser {
	rp := &ResponseParser{
		schemas:         make(map[string][]ValidationRule),
		minConfidence:   DefaultMinConfidence,
		maxStringLength: DefaultMaxStringLength,
		jsonCodeBlockRe: regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\})\\s*```"),
		keyValueRe:      regexp.MustCompile(`(?m)^\s*([^:]+)\s*:\s*(.+?)\s*$`),
	}

	rp.registerDefaultSchemas()

	return rp
}

// registerDefaultSchemas registers built-in schemas for known output types
func (rp *ResponseParser) registerDefaultSchemas() {
	rp.RegisterSchema("classification", []ValidationRule{
		{Field: "category", Type: FieldTypeString, Required: true, AllowedValues: []string{"infrastructure", "transient", "configuration", "unknown"}},
		{Field: "confidence", Type: FieldTypeFloat, Required: true},
		{Field: "reasoning", Type: FieldTypeString, Required: false},
		{Field: "severity", Type: FieldTypeString, Required: false, AllowedValues: []string{"critical", "high", "medium", "low"}},
	})

	rp.RegisterSchema("explanation", []ValidationRule{
		{Field: "summary", Type: FieldTypeString, Required: true},
		{Field: "root_cause", Type: FieldTypeString, Required: false},
		{Field: "contributing_factors", Type: FieldTypeArray, Required: false},
		{Field: "confidence", Type: FieldTypeFloat, Required: true},
	})

	rp.RegisterSchema("suggestion", []ValidationRule{
		{Field: "actions", Type: FieldTypeArray, Required: true},
		{Field: "priority", Type: FieldTypeString, Required: false, AllowedValues: []string{"critical", "high", "medium", "low"}},
		{Field: "risk_level", Type: FieldTypeString, Required: false, AllowedValues: []string{"critical", "high", "medium", "low"}},
		{Field: "expected_outcome", Type: FieldTypeString, Required: false},
	})
}

// RegisterSchema registers a validation schema for an output type
func (rp *ResponseParser) RegisterSchema(outputType string, schema []ValidationRule) {
	rp.schemas[outputType] = schema
}

// SetMinConfidence sets the minimum acceptable confidence threshold
func (rp *ResponseParser) SetMinConfidence(threshold float64) {
	rp.minConfidence = threshold
}

// SetMaxStringLength sets the maximum allowed string length
func (rp *ResponseParser) SetMaxStringLength(length int) {
	rp.maxStringLength = length
}

// ExtractJSONFromText extracts JSON content from markdown code blocks or plain text
func (rp *ResponseParser) ExtractJSONFromText(text string) (string, error) {
	// Try to extract from markdown code blocks first
	matches := rp.jsonCodeBlockRe.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1]), nil
	}

	// Look for raw JSON object
	text = strings.TrimSpace(text)
	startIdx := strings.Index(text, "{")
	endIdx := strings.LastIndex(text, "}")

	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return "", ParseError{Message: "no JSON object found in text"}
	}

	return text[startIdx : endIdx+1], nil
}

// ValidateOutput validates parsed data against the registered schema for the output type
func (rp *ResponseParser) ValidateOutput(outputType string, data map[string]interface{}) error {
	schema, exists := rp.schemas[outputType]
	if !exists {
		return ValidationError{Message: fmt.Sprintf("no schema registered for output type %q", outputType)}
	}

	validator := NewSchemaValidator(schema)
	return validator.Validate(data)
}

// SanitizeAndValidate extracts, parses, and validates raw response against a schema
func (rp *ResponseParser) SanitizeAndValidate(raw string, outputType string) (map[string]interface{}, error) {
	jsonStr, err := rp.ExtractJSONFromText(raw)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, ParseError{Message: "failed to parse JSON", Cause: err}
	}

	if err := rp.ValidateOutput(outputType, data); err != nil {
		return nil, err
	}

	return data, nil
}

// ParseClassification parses a raw LLM response into a ParsedClassification
func (rp *ResponseParser) ParseClassification(rawResponse string) (ParsedClassification, error) {
	data, err := rp.SanitizeAndValidate(rawResponse, "classification")
	if err != nil {
		// Try key-value fallback parsing
		data, err = rp.parseKeyValueFallback(rawResponse, "classification")
		if err != nil {
			return ParsedClassification{}, err
		}
	}

	result := ParsedClassification{}

	if category, ok := data["category"].(string); ok {
		result.Category = normalizeCategory(category)
	}

	if confidence, ok := data["confidence"].(float64); ok {
		result.Confidence = normalizeConfidence(confidence)
	} else if confidence, ok := data["confidence"].(int); ok {
		result.Confidence = normalizeConfidence(float64(confidence))
	}

	if result.Confidence < rp.minConfidence {
		return ParsedClassification{}, LowConfidenceError{Confidence: result.Confidence, Threshold: rp.minConfidence}
	}

	if reasoning, ok := data["reasoning"].(string); ok {
		result.Reasoning = truncateString(reasoning, rp.maxStringLength)
	}

	if severity, ok := data["severity"].(string); ok {
		result.Severity = normalizeSeverity(severity)
	}

	return result, nil
}

// ParseExplanation parses a raw LLM response into a ParsedExplanation
func (rp *ResponseParser) ParseExplanation(rawResponse string) (ParsedExplanation, error) {
	data, err := rp.SanitizeAndValidate(rawResponse, "explanation")
	if err != nil {
		data, err = rp.parseKeyValueFallback(rawResponse, "explanation")
		if err != nil {
			return ParsedExplanation{}, err
		}
	}

	result := ParsedExplanation{}

	if summary, ok := data["summary"].(string); ok {
		result.Summary = truncateString(summary, rp.maxStringLength)
	}

	if rootCause, ok := data["root_cause"].(string); ok {
		result.RootCause = truncateString(rootCause, rp.maxStringLength)
	}

	if factors, ok := data["contributing_factors"].([]interface{}); ok {
		result.ContributingFactors = make([]string, 0, len(factors))
		for _, f := range factors {
			if str, ok := f.(string); ok {
				result.ContributingFactors = append(result.ContributingFactors, truncateString(str, rp.maxStringLength))
			}
		}
	}

	if confidence, ok := data["confidence"].(float64); ok {
		result.Confidence = normalizeConfidence(confidence)
	} else if confidence, ok := data["confidence"].(int); ok {
		result.Confidence = normalizeConfidence(float64(confidence))
	}

	if result.Confidence < rp.minConfidence {
		return ParsedExplanation{}, LowConfidenceError{Confidence: result.Confidence, Threshold: rp.minConfidence}
	}

	return result, nil
}

// ParseSuggestion parses a raw LLM response into a ParsedSuggestion
func (rp *ResponseParser) ParseSuggestion(rawResponse string) (ParsedSuggestion, error) {
	data, err := rp.SanitizeAndValidate(rawResponse, "suggestion")
	if err != nil {
		data, err = rp.parseKeyValueFallback(rawResponse, "suggestion")
		if err != nil {
			return ParsedSuggestion{}, err
		}
	}

	result := ParsedSuggestion{}

	if actions, ok := data["actions"].([]interface{}); ok {
		result.Actions = make([]SuggestedAction, 0, len(actions))
		for _, a := range actions {
			actionMap, ok := a.(map[string]interface{})
			if !ok {
				continue
			}
			action := SuggestedAction{}

			if actionType, ok := actionMap["type"].(string); ok {
				action.Type = normalizeActionType(actionType)
			}
			if target, ok := actionMap["target"].(string); ok {
				action.Target = truncateString(target, rp.maxStringLength)
			}
			if automated, ok := actionMap["automated"].(bool); ok {
				action.Automated = automated
			}
			if params, ok := actionMap["parameters"].(map[string]interface{}); ok {
				action.Parameters = make(map[string]string)
				for k, v := range params {
					if str, ok := v.(string); ok {
						action.Parameters[k] = truncateString(str, rp.maxStringLength)
					}
				}
			}

			result.Actions = append(result.Actions, action)
		}
	}

	if priority, ok := data["priority"].(string); ok {
		result.Priority = normalizePriority(priority)
	}

	if riskLevel, ok := data["risk_level"].(string); ok {
		result.RiskLevel = normalizeSeverity(riskLevel)
	}

	if expectedOutcome, ok := data["expected_outcome"].(string); ok {
		result.ExpectedOutcome = truncateString(expectedOutcome, rp.maxStringLength)
	}

	return result, nil
}

// parseKeyValueFallback attempts to parse key-value formatted text when JSON parsing fails
func (rp *ResponseParser) parseKeyValueFallback(text string, outputType string) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	// Normalize key names for consistency
	keyMap := map[string]string{
		"category":             "category",
		"confidence":           "confidence",
		"reasoning":            "reasoning",
		"severity":             "severity",
		"summary":              "summary",
		"root_cause":           "root_cause",
		"contributing_factors": "contributing_factors",
		"priority":             "priority",
		"risk_level":           "risk_level",
		"expected_outcome":     "expected_outcome",
		"actions":              "actions",
	}

	matches := rp.keyValueRe.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(match[1]))
		value := strings.TrimSpace(match[2])

		// Map common key variations
		if normalizedKey, ok := keyMap[key]; ok {
			key = normalizedKey
		}
		if normalizedKey, ok := keyMap[strings.ReplaceAll(key, " ", "_")]; ok {
			key = normalizedKey
		}

		// Type conversion based on field
		switch key {
		case "confidence":
			if f, err := strconv.ParseFloat(value, 64); err == nil {
				data[key] = f
			} else {
				data[key] = value
			}
		case "contributing_factors", "actions":
			// Try to parse as JSON array first
			var arr []interface{}
			if err := json.Unmarshal([]byte(value), &arr); err == nil {
				data[key] = arr
			} else {
				// Fall back to comma-separated list
				parts := strings.Split(value, ",")
				arr := make([]interface{}, len(parts))
				for i, p := range parts {
					arr[i] = strings.TrimSpace(p)
				}
				data[key] = arr
			}
		case "automated":
			if b, err := strconv.ParseBool(value); err == nil {
				data[key] = b
			} else {
				data[key] = value
			}
		default:
			data[key] = value
		}
	}

	if len(data) == 0 {
		return nil, ParseError{Message: "key-value fallback parsing produced no results"}
	}

	if err := rp.ValidateOutput(outputType, data); err != nil {
		return nil, err
	}

	return data, nil
}

// normalizeCategory normalizes category values to lowercase and validates
func normalizeCategory(category string) string {
	category = strings.ToLower(strings.TrimSpace(category))
	validCategories := map[string]bool{
		"infrastructure": true,
		"transient":      true,
		"configuration":  true,
		"unknown":        true,
	}
	if !validCategories[category] {
		return "unknown"
	}
	return category
}

// normalizeSeverity normalizes severity values to lowercase and validates
func normalizeSeverity(severity string) string {
	severity = strings.ToLower(strings.TrimSpace(severity))
	validSeverities := map[string]bool{
		"critical": true,
		"high":     true,
		"medium":   true,
		"low":      true,
	}
	if !validSeverities[severity] {
		return "medium"
	}
	return severity
}

// normalizeActionType normalizes action type values to lowercase and validates
func normalizeActionType(actionType string) string {
	actionType = strings.ToLower(strings.TrimSpace(actionType))
	validTypes := map[string]bool{
		"restart":   true,
		"reconnect": true,
		"scale":     true,
		"alert":     true,
		"ignore":    true,
	}
	if !validTypes[actionType] {
		return "alert"
	}
	return actionType
}

// normalizePriority normalizes priority values to lowercase and validates
func normalizePriority(priority string) string {
	return normalizeSeverity(priority)
}

// normalizeConfidence ensures confidence is within valid range (0-1)
func normalizeConfidence(confidence float64) float64 {
	if confidence < 0 {
		return 0
	}
	if confidence > 1 {
		return 1
	}
	return confidence
}

// truncateString truncates a string to the specified maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// IsParseError checks if an error is a ParseError
func IsParseError(err error) bool {
	var parseErr ParseError
	return errors.As(err, &parseErr)
}

// IsValidationError checks if an error is a ValidationError
func IsValidationError(err error) bool {
	var validationErr ValidationError
	return errors.As(err, &validationErr)
}

// IsLowConfidenceError checks if an error is a LowConfidenceError
func IsLowConfidenceError(err error) bool {
	var lowConfErr LowConfidenceError
	return errors.As(err, &lowConfErr)
}
