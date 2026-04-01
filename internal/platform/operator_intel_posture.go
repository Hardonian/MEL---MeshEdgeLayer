package platform

import "github.com/mel-project/mel/internal/config"

// OperatorIntelligencePosture is machine-visible truth for what intelligence layers are allowed
// and what they are allowed to claim. Canonical deterministic intel stays viable when assist is off.
type OperatorIntelligencePosture struct {
	DeterministicIncidentIntel string `json:"deterministic_incident_intel"` // enabled | degraded_unavailable
	DeterministicBasis         string `json:"deterministic_basis"`
	TelemetryOutbound          bool   `json:"telemetry_outbound"`
	TelemetryRequireExplicit   bool   `json:"telemetry_require_explicit_opt_in"`
	AssistiveInferenceLayer    string `json:"assistive_inference_layer"`  // disabled | unavailable_not_configured | degraded | available
	AssistCapabilityStrategy   string `json:"assist_capability_strategy"` // explicit contract for UI: disabled | enabled_local_deterministic_only | enabled_bounded_local_assist | unavailable | not_configured | unsupported | remote_assist_unsupported_or_absent
	AssistNonCanonicalTruth    bool   `json:"assist_non_canonical_truth"`
	RemoteAssistSupported      bool   `json:"remote_assist_supported"` // false: no cloud assist path in base product
	ReviewRecommended          bool   `json:"review_recommended_for_assist_output"`
}

// BuildOperatorIntelligencePosture summarizes consent/capability for operator-facing intelligence surfaces.
func BuildOperatorIntelligencePosture(cfg config.Config, inferenceDegraded, runtimeReady bool) OperatorIntelligencePosture {
	p := OperatorIntelligencePosture{
		DeterministicIncidentIntel: "enabled",
		DeterministicBasis:         "local_persisted_records_and_config_bounded_calculators_only",
		TelemetryOutbound:          cfg.Platform.Telemetry.AllowOutbound,
		TelemetryRequireExplicit:   cfg.Platform.Telemetry.RequireExplicit,
		AssistNonCanonicalTruth:    true,
		RemoteAssistSupported:      false,
		ReviewRecommended:          true,
	}
	if !cfg.Platform.Inference.Enabled {
		p.AssistiveInferenceLayer = "disabled"
		// Bounded assist is off; deterministic incident intelligence on persisted records remains the product path.
		p.AssistCapabilityStrategy = "enabled_local_deterministic_only"
		return p
	}
	if !runtimeReady {
		p.AssistiveInferenceLayer = "unavailable_not_configured"
		p.AssistCapabilityStrategy = "not_configured"
		return p
	}
	if inferenceDegraded {
		p.AssistiveInferenceLayer = "degraded"
		p.AssistCapabilityStrategy = "unavailable"
		return p
	}
	p.AssistiveInferenceLayer = "available"
	p.AssistCapabilityStrategy = "enabled_bounded_local_assist"
	return p
}
