package policy

import "github.com/mel-project/mel/internal/config"

type Recommendation struct {
	Summary  string `json:"summary"`
	Severity string `json:"severity"`
}

func Explain(cfg config.Config) []Recommendation {
	var out []Recommendation
	if cfg.Privacy.StorePrecisePositions {
		out = append(out, Recommendation{"Disable precise position storage unless required for a tracked incident response workflow.", "high"})
	}
	if !cfg.Privacy.MQTTEncryptionRequired {
		out = append(out, Recommendation{"Require MQTT transport encryption or keep MQTT disabled for privacy-sensitive deployments.", "high"})
	}
	if cfg.Retention.MessagesDays > 30 {
		out = append(out, Recommendation{"Reduce message retention to 30 days or less for local-first posture.", "medium"})
	}
	return out
}
