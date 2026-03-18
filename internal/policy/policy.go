package policy

import "github.com/mel-project/mel/internal/config"

type Recommendation struct {
	ID          string   `json:"id"`
	Summary     string   `json:"summary"`
	Severity    string   `json:"severity"`
	Reason      string   `json:"reason"`
	Evidence    []string `json:"evidence,omitempty"`
	Remediation string   `json:"remediation"`
}

func Explain(cfg config.Config) []Recommendation {
	out := make([]Recommendation, 0)
	if cfg.Privacy.StorePrecisePositions {
		out = append(out, Recommendation{
			ID:          "disable-precise-position-storage",
			Summary:     "Disable precise position storage unless your workflow truly requires it.",
			Severity:    "high",
			Reason:      "Position history can be sensitive personal data and is rarely required for local observability.",
			Evidence:    []string{"privacy.store_precise_positions=true"},
			Remediation: "Turn off privacy.store_precise_positions or require operator approval plus storage key management.",
		})
	}
	if !cfg.Privacy.MQTTEncryptionRequired {
		out = append(out, Recommendation{
			ID:          "require-mqtt-encryption",
			Summary:     "Require MQTT transport encryption or disable MQTT for privacy-sensitive deployments.",
			Severity:    "high",
			Reason:      "Broker hops can expose message content and metadata when transport encryption is not enforced.",
			Evidence:    []string{"privacy.mqtt_encryption_required=false"},
			Remediation: "Keep MQTT local, add TLS via a local tunnel, or disable the transport.",
		})
	}
	if cfg.Retention.MessagesDays > 30 {
		out = append(out, Recommendation{
			ID:          "reduce-message-retention",
			Summary:     "Reduce message retention to 30 days or less for the default local-first posture.",
			Severity:    "medium",
			Reason:      "Longer retention increases operator liability and metadata accumulation without improving transport truthfulness.",
			Evidence:    []string{"retention.messages_days>30"},
			Remediation: "Lower the retention period or document the operational justification in your site notes.",
		})
	}
	if cfg.Bind.AllowRemote {
		out = append(out, Recommendation{
			ID:          "keep-local-bind-default",
			Summary:     "Keep MEL on localhost unless you have a reviewed remote-access design.",
			Severity:    "medium",
			Reason:      "Local-only exposure is the safest baseline for an edge operator tool that handles telemetry and node metadata.",
			Evidence:    []string{"bind.allow_remote=true"},
			Remediation: "Use a reverse proxy or VPN if remote access is necessary, and keep auth enabled.",
		})
	}
	return out
}
