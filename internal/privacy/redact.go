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
