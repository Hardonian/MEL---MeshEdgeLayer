package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ApplyProfile merges a named profile from configs/profiles/<name>.json over cfg.
// Only JSON keys present in the profile file are applied (omitted keys leave cfg unchanged).
func ApplyProfile(cfg *Config, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		if v := strings.TrimSpace(os.Getenv("MEL_CONFIG_PROFILE")); v != "" {
			name = v
		}
	}
	if name == "" {
		return nil
	}
	path := filepath.Join("configs", "profiles", name+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("profile %q: %w", name, err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("profile %q: %w", name, err)
	}
	for key, fragment := range raw {
		if err := mergeProfileKey(cfg, key, fragment); err != nil {
			return fmt.Errorf("profile %q key %q: %w", name, key, err)
		}
	}
	return nil
}

func mergeProfileKey(cfg *Config, key string, fragment json.RawMessage) error {
	switch key {
	case "bind":
		var v BindConfig
		if err := json.Unmarshal(fragment, &v); err != nil {
			return err
		}
		cfg.Bind = v
	case "auth":
		var v AuthConfig
		if err := json.Unmarshal(fragment, &v); err != nil {
			return err
		}
		cfg.Auth = v
	case "storage":
		var v StorageConfig
		if err := json.Unmarshal(fragment, &v); err != nil {
			return err
		}
		cfg.Storage = v
	case "logging":
		var v LoggingConfig
		if err := json.Unmarshal(fragment, &v); err != nil {
			return err
		}
		cfg.Logging = v
	case "retention":
		var v RetentionConfig
		if err := json.Unmarshal(fragment, &v); err != nil {
			return err
		}
		cfg.Retention = v
	case "privacy":
		var v PrivacyConfig
		if err := json.Unmarshal(fragment, &v); err != nil {
			return err
		}
		cfg.Privacy = v
	case "transports":
		var v []TransportConfig
		if err := json.Unmarshal(fragment, &v); err != nil {
			return err
		}
		cfg.Transports = v
	case "features":
		var v FeatureConfig
		if err := json.Unmarshal(fragment, &v); err != nil {
			return err
		}
		cfg.Features = v
	case "rate_limits":
		var v RateLimitConfig
		if err := json.Unmarshal(fragment, &v); err != nil {
			return err
		}
		cfg.RateLimits = v
	case "intelligence":
		var v IntelligenceConfig
		if err := json.Unmarshal(fragment, &v); err != nil {
			return err
		}
		cfg.Intelligence = v
	case "control":
		var v ControlConfig
		if err := json.Unmarshal(fragment, &v); err != nil {
			return err
		}
		cfg.Control = v
	case "federation":
		var v FederationConfig
		if err := json.Unmarshal(fragment, &v); err != nil {
			return err
		}
		cfg.Federation = v
	case "integration":
		var v IntegrationConfig
		if err := json.Unmarshal(fragment, &v); err != nil {
			return err
		}
		cfg.Integration = v
	case "strict_mode":
		var v bool
		if err := json.Unmarshal(fragment, &v); err != nil {
			return err
		}
		cfg.StrictMode = v
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	return nil
}
