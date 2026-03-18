package privacy

import "github.com/mel-project/mel/internal/config"

type Finding struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Remediation string `json:"remediation"`
}

func Audit(cfg config.Config) []Finding {
	var out []Finding
	if cfg.Bind.AllowRemote {
		out = append(out, Finding{"remote-bind", "high", "API/UI is allowed to bind beyond localhost.", "Keep bind local or enable strong auth and network controls."})
	}
	if !cfg.Privacy.MQTTEncryptionRequired {
		out = append(out, Finding{"mqtt-encryption", "high", "MQTT encryption requirement is disabled.", "Use TLS-capable brokers or keep MQTT disabled."})
	}
	if cfg.Privacy.MapReportingAllowed {
		out = append(out, Finding{"map-reporting", "high", "Map reporting can leak node metadata and locations.", "Disable map reporting unless operators explicitly opt in."})
	}
	if cfg.Retention.MessagesDays > 90 {
		out = append(out, Finding{"retention-long", "medium", "Message retention exceeds 90 days.", "Shorten retention windows or justify them in ops docs."})
	}
	if cfg.Privacy.StorePrecisePositions {
		out = append(out, Finding{"precise-position", "high", "Precise location storage is enabled.", "Disable precise position storage or ensure a 32-byte storage key is configured."})
	}
	if cfg.Bind.AllowRemote && !cfg.Auth.Enabled {
		out = append(out, Finding{"remote-auth", "critical", "Remote bind without auth is unsafe.", "Enable auth before allowing remote access."})
	}
	return out
}
