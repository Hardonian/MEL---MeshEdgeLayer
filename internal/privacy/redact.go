package privacy

import "github.com/mel-project/mel/internal/config"

// RedactConfig returns a copy of the configuration with sensitive fields masked.
func RedactConfig(cfg config.Config) config.Config {
	cloned := cfg
	if cloned.Auth.SessionSecret != "" {
		cloned.Auth.SessionSecret = "[redacted]"
	}
	if cloned.Auth.UIPassword != "" {
		cloned.Auth.UIPassword = "[redacted]"
	}
	for i := range cloned.Transports {
		if cloned.Transports[i].Password != "" {
			cloned.Transports[i].Password = "[redacted]"
		}
	}
	return cloned
}

// RedactMessages returns a copy of row maps with sensitive fields (text, hex) masked.
func RedactMessages(rows []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		cloned := make(map[string]any)
		for k, v := range row {
			cloned[k] = v
		}
		if _, ok := cloned["payload_text"]; ok {
			cloned["payload_text"] = "[redacted]"
		}
		if _, ok := cloned["raw_hex"]; ok {
			cloned["raw_hex"] = "[redacted]"
		}
		// Also redact payloads in JSON if they look like plaintext
		if _, ok := cloned["payload_json"]; ok {
			cloned["payload_json"] = `{"redacted": true, "reason": "privacy_policy"}`
		}
		out = append(out, cloned)
	}
	return out
}
