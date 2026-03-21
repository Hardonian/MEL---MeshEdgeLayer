package config

import (
	"encoding/json"
	"strings"
)

type EffectiveConfig struct {
	Fingerprint string            `json:"fingerprint"`
	Values      any               `json:"values"`
	Violations  []SafetyViolation `json:"violations"`
}

func Inspect(cfg Config, loadedFromFile []byte) EffectiveConfig {
	b, _ := json.Marshal(cfg)

	fingerprint := ""
	if len(loadedFromFile) > 0 {
		fingerprint = SHA256(loadedFromFile)
	} else {
		fingerprint = SHA256(b)
	}

	var raw map[string]any
	_ = json.Unmarshal(b, &raw)

	redactMap(raw, "")

	return EffectiveConfig{
		Fingerprint: fingerprint,
		Values:      raw,
		Violations:  ValidateSafeDefaults(cfg),
	}
}

func redactMap(m map[string]any, prefix string) {
	sensitiveKeys := []string{"password", "secret", "key"}
	for k, v := range m {
		kLower := strings.ToLower(k)
		isSensitive := false
		for _, sk := range sensitiveKeys {
			if strings.Contains(kLower, sk) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			if s, ok := v.(string); ok && s != "" {
				m[k] = "***REDACTED***"
			}
		} else if nextMap, ok := v.(map[string]any); ok {
			redactMap(nextMap, prefix+k+".")
		} else if nextSlice, ok := v.([]any); ok {
			for _, item := range nextSlice {
				if itemMap, okMap := item.(map[string]any); okMap {
					redactMap(itemMap, prefix+k+".[]")
				}
			}
		}
	}
}
