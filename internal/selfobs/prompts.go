// Package selfobs provides prompt templates for LLM interactions in the self-observability system.
package selfobs

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// PromptMessage represents a chat message for LLM completions.
type PromptMessage struct {
	// Role is the message sender role: "system", "user", or "assistant".
	Role string `json:"role"`
	// Content is the message text content.
	Content string `json:"content"`
}

// PromptTemplate defines a reusable prompt template for LLM interactions.
type PromptTemplate struct {
	// Name is the unique identifier for the template.
	Name string `json:"name"`
	// Description explains the template's purpose.
	Description string `json:"description"`
	// SystemPrompt is the system message content.
	SystemPrompt string `json:"system_prompt"`
	// UserPromptTemplate is the user message template with placeholders.
	UserPromptTemplate string `json:"user_prompt_template"`
	// Version is the template version identifier.
	Version string `json:"version"`
	// OutputSchema maps field names to their expected types/descriptions.
	OutputSchema map[string]string `json:"output_schema"`
}

// LLMInput contains data for building classification prompts.
type LLMInput struct {
	Component string
	ErrorRate float64
	State     string
	Timestamp time.Time
	Metrics   map[string]float64
	History   []string
}

// PromptAnomaly represents an anomaly for explanation prompts.
type PromptAnomaly struct {
	Component string
	Metric    string
	Value     float64
	Expected  float64
	Deviation float64
	Timestamp time.Time
}

// PromptScenario contains remediation context for suggestion prompts.
type PromptScenario struct {
	Component   string
	Issue       string
	Context     map[string]string
	Constraints []string
}

// PromptEngine manages prompt templates and executes them.
type PromptEngine struct {
	templates map[string]PromptTemplate
}

// placeholderPattern matches {{VariableName}} syntax.
var placeholderPattern = regexp.MustCompile(`\{\{([A-Za-z_][A-Za-z0-9_]*)\}\}`)

// sanitizeString removes potentially harmful characters from input.
func sanitizeString(s string) string {
	// Remove control characters except newlines and tabs
	var result strings.Builder
	for _, r := range s {
		if (r >= 32 && r < 127) || r == '\n' || r == '\t' || r >= 160 {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// formatMetrics converts metrics map to a sanitized string representation.
func formatMetrics(metrics map[string]float64) string {
	if len(metrics) == 0 {
		return "none"
	}
	var parts []string
	for k, v := range metrics {
		safeKey := sanitizeString(k)
		parts = append(parts, fmt.Sprintf("%s=%.4f", safeKey, v))
	}
	return strings.Join(parts, ", ")
}

// formatHistory converts history slice to a sanitized string representation.
func formatHistory(history []string) string {
	if len(history) == 0 {
		return "none"
	}
	var parts []string
	for _, h := range history {
		parts = append(parts, sanitizeString(h))
	}
	return strings.Join(parts, "; ")
}

// formatTimestamp returns an RFC3339 formatted timestamp.
func formatTimestamp(t time.Time) string {
	return t.Format(time.RFC3339)
}

// NewPromptEngine creates a new prompt engine with built-in templates registered.
func NewPromptEngine() *PromptEngine {
	engine := &PromptEngine{
		templates: make(map[string]PromptTemplate),
	}

	engine.RegisterTemplate(ClassificationTemplate)
	engine.RegisterTemplate(ExplanationTemplate)
	engine.RegisterTemplate(SuggestionTemplate)
	engine.RegisterTemplate(AnomalyDetectionTemplate)

	return engine
}

// RegisterTemplate adds a template to the engine's registry.
// Returns an error if a template with the same name already exists.
func (pe *PromptEngine) RegisterTemplate(template PromptTemplate) error {
	if template.Name == "" {
		return fmt.Errorf("template name cannot be empty")
	}
	if _, exists := pe.templates[template.Name]; exists {
		return fmt.Errorf("template '%s' already registered", template.Name)
	}
	pe.templates[template.Name] = template
	return nil
}

// GetTemplate retrieves a template by name.
// Returns the template and true if found, otherwise empty and false.
func (pe *PromptEngine) GetTemplate(name string) (PromptTemplate, bool) {
	template, ok := pe.templates[name]
	return template, ok
}

// ExecuteTemplate renders a template with the provided variables.
// Substitutes placeholders like {{Variable}} with sanitized values from the map.
func (pe *PromptEngine) ExecuteTemplate(name string, variables map[string]interface{}) (string, error) {
	template, ok := pe.templates[name]
	if !ok {
		return "", fmt.Errorf("template '%s' not found", name)
	}

	result := placeholderPattern.ReplaceAllStringFunc(template.UserPromptTemplate, func(match string) string {
		varName := match[2 : len(match)-2]
		value, exists := variables[varName]
		if !exists {
			return match
		}

		switch v := value.(type) {
		case string:
			return sanitizeString(v)
		case float64, float32:
			return fmt.Sprintf("%.4f", v)
		case int, int64, int32:
			return fmt.Sprintf("%d", v)
		case time.Time:
			return formatTimestamp(v)
		case map[string]float64:
			return formatMetrics(v)
		case []string:
			return formatHistory(v)
		default:
			return sanitizeString(fmt.Sprintf("%v", v))
		}
	})

	return result, nil
}

// ValidateOutput validates LLM output against the expected schema.
// Attempts to parse JSON and verify required fields are present.
func (pe *PromptEngine) ValidateOutput(name string, output string) (map[string]string, error) {
	template, ok := pe.templates[name]
	if !ok {
		return nil, fmt.Errorf("template '%s' not found", name)
	}

	safeOutput := sanitizeString(output)

	// Try to parse as JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(safeOutput), &parsed); err != nil {
		return nil, fmt.Errorf("output is not valid JSON: %w", err)
	}

	// Validate required fields
	result := make(map[string]string)
	for field, expectedType := range template.OutputSchema {
		value, exists := parsed[field]
		if !exists {
			return nil, fmt.Errorf("missing required field: %s", field)
		}
		result[field] = fmt.Sprintf("%v", value)

		// Type checking
		switch expectedType {
		case "string":
			if _, ok := value.(string); !ok {
				return nil, fmt.Errorf("field %s expected to be string", field)
			}
		case "number":
			if _, ok := value.(float64); !ok {
				return nil, fmt.Errorf("field %s expected to be number", field)
			}
		case "boolean":
			if _, ok := value.(bool); !ok {
				return nil, fmt.Errorf("field %s expected to be boolean", field)
			}
		case "array":
			if _, ok := value.([]interface{}); !ok {
				return nil, fmt.Errorf("field %s expected to be array", field)
			}
		case "object":
			if _, ok := value.(map[string]interface{}); !ok {
				return nil, fmt.Errorf("field %s expected to be object", field)
			}
		}
	}

	return result, nil
}

// BuildClassificationPrompt creates messages for component health classification.
// Returns system and user messages ready for LLM completion.
func (pe *PromptEngine) BuildClassificationPrompt(input LLMInput) ([]PromptMessage, error) {
	variables := map[string]interface{}{
		"Component": sanitizeString(input.Component),
		"ErrorRate": input.ErrorRate,
		"State":     sanitizeString(input.State),
		"Timestamp": input.Timestamp,
		"Metrics":   input.Metrics,
		"History":   input.History,
	}

	userContent, err := pe.ExecuteTemplate("classification", variables)
	if err != nil {
		return nil, fmt.Errorf("failed to execute classification template: %w", err)
	}

	template, _ := pe.templates["classification"]
	return []PromptMessage{
		{Role: "system", Content: template.SystemPrompt},
		{Role: "user", Content: userContent},
	}, nil
}

// BuildExplanationPrompt creates messages for anomaly explanation.
// Returns system and user messages ready for LLM completion.
func (pe *PromptEngine) BuildExplanationPrompt(anomaly PromptAnomaly) ([]PromptMessage, error) {
	variables := map[string]interface{}{
		"Component": sanitizeString(anomaly.Component),
		"Metric":    sanitizeString(anomaly.Metric),
		"Value":     anomaly.Value,
		"Expected":  anomaly.Expected,
		"Deviation": anomaly.Deviation,
		"Timestamp": anomaly.Timestamp,
	}

	userContent, err := pe.ExecuteTemplate("explanation", variables)
	if err != nil {
		return nil, fmt.Errorf("failed to execute explanation template: %w", err)
	}

	template, _ := pe.templates["explanation"]
	return []PromptMessage{
		{Role: "system", Content: template.SystemPrompt},
		{Role: "user", Content: userContent},
	}, nil
}

// BuildSuggestionPrompt creates messages for remediation suggestions.
// Returns system and user messages ready for LLM completion.
func (pe *PromptEngine) BuildSuggestionPrompt(scenario PromptScenario) ([]PromptMessage, error) {
	// Build context string
	var contextParts []string
	for k, v := range scenario.Context {
		contextParts = append(contextParts, fmt.Sprintf("%s=%s", sanitizeString(k), sanitizeString(v)))
	}
	contextStr := strings.Join(contextParts, ", ")
	if contextStr == "" {
		contextStr = "none"
	}

	// Build constraints string
	var constraintsStr string
	if len(scenario.Constraints) == 0 {
		constraintsStr = "none"
	} else {
		var safeConstraints []string
		for _, c := range scenario.Constraints {
			safeConstraints = append(safeConstraints, sanitizeString(c))
		}
		constraintsStr = strings.Join(safeConstraints, ", ")
	}

	variables := map[string]interface{}{
		"Component":   sanitizeString(scenario.Component),
		"Issue":       sanitizeString(scenario.Issue),
		"Context":     contextStr,
		"Constraints": constraintsStr,
	}

	userContent, err := pe.ExecuteTemplate("suggestion", variables)
	if err != nil {
		return nil, fmt.Errorf("failed to execute suggestion template: %w", err)
	}

	template, _ := pe.templates["suggestion"]
	return []PromptMessage{
		{Role: "system", Content: template.SystemPrompt},
		{Role: "user", Content: userContent},
	}, nil
}

// Built-in templates for common self-observability tasks.

// ClassificationTemplate classifies component health issues.
var ClassificationTemplate = PromptTemplate{
	Name:        "classification",
	Description: "Classify the severity and type of a component health issue",
	SystemPrompt: `You are a site reliability engineer analyzing component health.
Your task is to classify health issues based on metrics and state information.
Respond with a JSON object containing:
- severity: "critical", "warning", or "info"
- category: "resource", "dependency", "configuration", or "unknown"
- confidence: number between 0 and 1
- summary: brief human-readable description`,
	UserPromptTemplate: `Component: {{Component}}
Error Rate: {{ErrorRate}}%
Current State: {{State}}
Timestamp: {{Timestamp}}
Metrics: {{Metrics}}
History: {{History}}

Classify this health issue and provide the analysis in the requested JSON format.`,
	Version: "1.0",
	OutputSchema: map[string]string{
		"severity":   "string",
		"category":   "string",
		"confidence": "number",
		"summary":    "string",
	},
}

// ExplanationTemplate explains anomalies in system behavior.
var ExplanationTemplate = PromptTemplate{
	Name:        "explanation",
	Description: "Explain the root cause of an anomaly",
	SystemPrompt: `You are a systems analyst explaining anomalies.
Analyze the provided anomaly data and explain the likely cause.
Respond with a JSON object containing:
- cause: brief description of the probable cause
- likelihood: number between 0 and 1
- related_factors: array of contributing factors
- recommendation: suggested immediate action`,
	UserPromptTemplate: `Anomaly Detected:
Component: {{Component}}
Metric: {{Metric}}
Observed Value: {{Value}}
Expected Value: {{Expected}}
Deviation: {{Deviation}}%
Timestamp: {{Timestamp}}

Explain this anomaly with your analysis in the requested JSON format.`,
	Version: "1.0",
	OutputSchema: map[string]string{
		"cause":           "string",
		"likelihood":      "number",
		"related_factors": "array",
		"recommendation":  "string",
	},
}

// SuggestionTemplate suggests remediation actions for issues.
var SuggestionTemplate = PromptTemplate{
	Name:        "suggestion",
	Description: "Suggest remediation actions for identified issues",
	SystemPrompt: `You are an SRE providing remediation suggestions.
Based on the issue description and context, suggest actionable steps.
Respond with a JSON object containing:
- actions: array of recommended actions (each with description, priority, risk)
- rollback_plan: brief rollback strategy if applicable
- verification: how to verify the fix`,
	UserPromptTemplate: `Remediation Request:
Component: {{Component}}
Issue: {{Issue}}
Context: {{Context}}
Constraints: {{Constraints}}

Suggest remediation actions in the requested JSON format.`,
	Version: "1.0",
	OutputSchema: map[string]string{
		"actions":       "array",
		"rollback_plan": "string",
		"verification":  "string",
	},
}

// AnomalyDetectionTemplate detects anomalies in metric data.
var AnomalyDetectionTemplate = PromptTemplate{
	Name:        "anomaly_detection",
	Description: "Detect anomalies in time-series metrics",
	SystemPrompt: `You are an anomaly detection system analyzing metric patterns.
Identify outliers and unusual patterns in the provided metrics.
Respond with a JSON object containing:
- is_anomaly: boolean indicating if anomaly detected
- anomalies: array of detected anomalies with metric names and descriptions
- severity: "high", "medium", or "low"
- pattern: description of the observed pattern`,
	UserPromptTemplate: `Analyze the following metrics for anomalies:
Component: {{Component}}
Timestamp: {{Timestamp}}
Metrics: {{Metrics}}
History: {{History}}

Return your anomaly analysis in the requested JSON format.`,
	Version: "1.0",
	OutputSchema: map[string]string{
		"is_anomaly": "boolean",
		"anomalies":  "array",
		"severity":   "string",
		"pattern":    "string",
	},
}
