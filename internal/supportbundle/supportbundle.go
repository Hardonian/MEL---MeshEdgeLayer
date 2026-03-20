// Package supportbundle provides support bundle generation with redaction capabilities.
package supportbundle

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/diagnostics"
	"github.com/mel-project/mel/internal/transport"
)

// SensitiveFields contains field names that should be redacted
var SensitiveFields = []string{
	"password",
	"secret",
	"token",
	"api_key",
	"apikey",
	"private_key",
	"credential",
	"auth",
	"basic_auth",
	"access_token",
	"refresh_token",
}

// BundleOptions defines what to include in the support bundle
type BundleOptions struct {
	IncludeConfig      bool
	IncludeTransport   bool
	IncludeNodes       bool
	IncludeMessages    bool
	IncludeDiagnostics bool
	IncludeRecentLogs bool
	MaxMessages       int
	MaxLogLines       int
}

// DefaultBundleOptions returns sensible defaults
func DefaultBundleOptions() BundleOptions {
	return BundleOptions{
		IncludeConfig:      true,
		IncludeTransport:   true,
		IncludeNodes:       true,
		IncludeMessages:   false,
		IncludeDiagnostics: true,
		IncludeRecentLogs: false,
		MaxMessages:       100,
		MaxLogLines:       500,
	}
}

// Bundle represents a support bundle
type Bundle struct {
	GeneratedAt     time.Time    `json:"generated_at"`
	Version        string       `json:"version"`
	Options        BundleOptions `json:"options"`
	Summary        BundleSummary `json:"summary"`
	Config         *RedactedConfig `json:"config,omitempty"`
	TransportState []transport.Health `json:"transport_state,omitempty"`
	Nodes          []NodeSummary `json:"nodes,omitempty"`
	Diagnostics    []diagnostics.Diagnostic `json:"diagnostics,omitempty"`
	RedactedNotice string `json:"redacted_notice"`
}

// BundleSummary provides a quick overview
type BundleSummary struct {
	ConfigRedacted      bool `json:"config_redacted"`
	TransportRedacted   bool `json:"transport_redacted"`
	NodesIncluded       int  `json:"nodes_included"`
	DiagnosticsFound   int  `json:"diagnostics_found"`
	MessagesIncluded   int  `json:"messages_included,omitempty"`
	LogsIncluded       int  `json:"logs_included,omitempty"`
}

// RedactedConfig is a config with sensitive fields redacted
type RedactedConfig struct {
	Storage        interface{} `json:"storage"`
	Database       interface{} `json:"database"`
	Control        interface{} `json:"control"`
	Features       interface{} `json:"features"`
	Transports     interface{} `json:"transports"`
	Retention      interface{} `json:"retention"`
	Intelligence   interface{} `json:"intelligence"`
	SensitiveCount int        `json:"sensitive_fields_redacted"`
}

// NodeSummary provides a summary of a node (not full details)
type NodeSummary struct {
	NodeNum      int64  `json:"node_num"`
	UserName    string `json:"user_name,omitempty"`
	ShortName   string `json:"short_name,omitempty"`
	LastSeen    string `json:"last_seen"`
	HasPosition bool   `json:"has_position"`
}

// Generate creates a support bundle with appropriate redaction
func Generate(
	cfg config.Config,
	database *db.DB,
	runtimeTransports func() []transport.Health,
	diagnosticsRun func(config.Config, *db.DB) []diagnostics.Finding,
	opts BundleOptions,
	version string,
) (*Bundle, error) {
	bundle := &Bundle{
		GeneratedAt: time.Now().UTC(),
		Version:    version,
		Options:    opts,
		RedactedNotice: "This bundle has been processed to redact sensitive information. " +
			"Sensitive fields (passwords, tokens, keys, credentials) are redacted.",
		Summary: BundleSummary{
			ConfigRedacted:    true,
			TransportRedacted: true,
		},
	}

	// Include redacted config
	if opts.IncludeConfig {
		bundle.Config = redactConfig(cfg)
	}

	// Include transport state
	if opts.IncludeTransport && runtimeTransports != nil {
		bundle.TransportState = runtimeTransports()
	}

	// Include node summaries
	if opts.IncludeNodes && database != nil {
		nodes, err := getNodeSummaries(database, 100)
		if err == nil {
			bundle.Nodes = nodes
			bundle.Summary.NodesIncluded = len(nodes)
		}
	}

	// Include diagnostics
	if opts.IncludeDiagnostics && diagnosticsRun != nil {
		findings := diagnosticsRun(cfg, database)
		// Convert findings to diagnostics
		for _, f := range findings {
			bundle.Diagnostics = append(bundle.Diagnostics, diagnostics.Diagnostic{
				Code:                   f.Code,
				Severity:               f.Severity,
				Component:              f.Component,
				Title:                  f.Title,
				Explanation:            f.Explanation,
				LikelyCauses:           f.LikelyCauses,
				RecommendedSteps:       f.RecommendedSteps,
				Evidence:               f.Evidence,
				CanAutoRecover:         f.CanAutoRecover,
				OperatorActionRequired: f.OperatorActionRequired,
				GeneratedAt:           f.GeneratedAt,
			})
		}
		bundle.Summary.DiagnosticsFound = len(bundle.Diagnostics)
	}

	return bundle, nil
}

// redactConfig removes sensitive information from config
func redactConfig(cfg config.Config) *RedactedConfig {
	redacted := &RedactedConfig{
		Storage:    cfg.Storage,
		Database:   cfg.Database,
		Control:   cfg.Control,
		Features:   cfg.Features,
		Transports: redactTransports(cfg.Transports),
		Retention:  cfg.Retention,
		Intelligence: cfg.Intelligence,
	}

	// Count redacted sensitive fields
	count := 0
	for _, t := range cfg.Transports {
		if containsSensitive(t.Password) {
			count++
		}
		if containsSensitive(t.APIKey) {
			count++
		}
		if containsSensitive(t.ClientID) && isSensitiveFieldName(t.ClientID) {
			count++
		}
	}
	redacted.SensitiveCount = count

	return redacted
}

// redactTransports redacts sensitive fields in transport configs
func redactTransports(transports []config.TransportConfig) []interface{} {
	result := make([]interface{}, len(transports))
	for i, t := range transports {
		redacted := map[string]interface{}{
			"name":        t.Name,
			"type":        t.Type,
			"enabled":     t.Enabled,
			"endpoint":    redactIfSensitive(t.Endpoint, "endpoint"),
			"topic":       t.Topic,
			"mqtt_enabled": t.MQTTEnabled,
		}
		
		if t.Password != "" {
			redacted["password"] = "[REDACTED]"
		}
		if t.APIKey != "" {
			redacted["api_key"] = "[REDACTED]"
		}
		
		result[i] = redacted
	}
	return result
}

// containsSensitive checks if a value looks like it contains sensitive data
func containsSensitive(value string) bool {
	if value == "" {
		return false
	}
	lower := strings.ToLower(value)
	sensitivePatterns := []string{"sk-", "akia", "sq0c", "token", "key", "secret", "password"}
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// isSensitiveFieldName checks if the field name indicates sensitive data
func isSensitiveFieldName(name string) bool {
	lower := strings.ToLower(name)
	for _, field := range SensitiveFields {
		if strings.Contains(lower, field) {
			return true
		}
	}
	return false
}

// redactIfSensitive redacts a value if it appears to contain sensitive data
func redactIfSensitive(value, fieldName string) string {
	if value == "" {
		return ""
	}
	if isSensitiveFieldName(fieldName) || containsSensitive(value) {
		return "[REDACTED]"
	}
	return value
}

// getNodeSummaries retrieves node summaries from database
func getNodeSummaries(database *db.DB, limit int) ([]NodeSummary, error) {
	rows, err := database.Query("SELECT node_num, user_name, short_name, last_seen, has_position FROM nodes ORDER BY last_seen DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []NodeSummary
	for rows.Next() {
		var n NodeSummary
		var userName, shortName sql.NullString
		var lastSeen interface{}
		var hasPos bool
		
		err := rows.Scan(&n.NodeNum, &userName, &shortName, &lastSeen, &hasPos)
		if err != nil {
			continue
		}
		
		if userName.Valid {
			n.UserName = userName.String
		}
		if shortName.Valid {
			n.ShortName = shortName.String
		}
		if lastSeen != nil {
			n.LastSeen = fmt.Sprintf("%v", lastSeen)
		}
		n.HasPosition = hasPos
		
		nodes = append(nodes, n)
	}
	
	return nodes, nil
}

// ToJSON converts the bundle to JSON
func (b *Bundle) ToJSON() (string, error) {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal bundle: %w", err)
	}
	return string(data), nil
}
