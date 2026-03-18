package privacy

import (
	"sort"
	"strings"

	"github.com/mel-project/mel/internal/config"
)

type Finding struct {
	ID          string   `json:"id"`
	Severity    string   `json:"severity"`
	Message     string   `json:"message"`
	Remediation string   `json:"remediation"`
	Evidence    []string `json:"evidence,omitempty"`
}

func Audit(cfg config.Config) []Finding {
	out := make([]Finding, 0)
	for _, lint := range config.LintConfig(cfg) {
		out = append(out, Finding{ID: lint.ID, Severity: lint.Severity, Message: lint.Message, Remediation: lint.Remediation})
	}
	if !cfg.Privacy.RedactExports {
		out = append(out, Finding{
			ID:          "export-redaction-disabled",
			Severity:    "medium",
			Message:     "Exports are configured without redaction.",
			Remediation: "Enable privacy.redact_exports for operator-safe exports.",
			Evidence:    []string{"privacy.redact_exports=false"},
		})
	}
	if len(cfg.Privacy.TrustList) == 0 {
		out = append(out, Finding{
			ID:          "empty-trust-list",
			Severity:    "info",
			Message:     "No trust list is configured.",
			Remediation: "Leave it empty if intentional, or add known node IDs for stricter export and policy workflows.",
			Evidence:    []string{"privacy.trust_list=[]"},
		})
	}
	for _, transport := range cfg.Transports {
		if !transport.Enabled || transport.Type != "mqtt" {
			continue
		}
		if transport.Username == "" {
			out = append(out, Finding{
				ID:          "mqtt-anonymous",
				Severity:    "medium",
				Message:     "MQTT transport is enabled without broker credentials in MEL config.",
				Remediation: "Use a local tunnel or authenticated broker account when your deployment needs it.",
				Evidence:    []string{"transport=" + transport.Name},
			})
		}
		if strings.Contains(transport.Topic, "/json/") {
			out = append(out, Finding{
				ID:          "mqtt-json-topic",
				Severity:    "high",
				Message:     "MQTT topic appears to use JSON payload routing, which can widen metadata exposure.",
				Remediation: "Prefer protobuf payload topics unless JSON output is a deliberate operator choice.",
				Evidence:    []string{"topic=" + transport.Topic},
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return severityRank(out[i].Severity) > severityRank(out[j].Severity)
	})
	return out
}

func Summary(findings []Finding) map[string]int {
	out := map[string]int{"critical": 0, "high": 0, "medium": 0, "low": 0, "info": 0}
	for _, finding := range findings {
		out[finding.Severity]++
	}
	return out
}

func severityRank(level string) int {
	switch level {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	default:
		return 1
	}
}
