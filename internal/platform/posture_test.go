package platform

import (
	"testing"

	"github.com/mel-project/mel/internal/config"
)

func TestBuildPostureDefaults(t *testing.T) {
	cfg := config.Default()
	p := BuildPosture(cfg)
	if p.TelemetryEnabled {
		t.Fatalf("expected telemetry disabled by default")
	}
	if p.InferenceEnabled {
		t.Fatalf("expected inference disabled by default")
	}
	if len(p.AssistPolicies) == 0 {
		t.Fatalf("expected assist policy rows")
	}
	if p.AssistPolicies[0].Availability != AssistUnavailable {
		t.Fatalf("expected unavailable assist when providers disabled, got %s", p.AssistPolicies[0].Availability)
	}
	if p.TelemetryStatus != "disabled" {
		t.Fatalf("expected telemetry disabled status, got %q", p.TelemetryStatus)
	}
	if p.InferenceRuntimeReady {
		t.Fatalf("expected inference runtime not ready by default")
	}
	if p.InferenceDegraded {
		t.Fatalf("expected inference not degraded when globally disabled")
	}
}

func TestBuildPostureDeleteDisabled(t *testing.T) {
	cfg := config.Default()
	cfg.Platform.Retention.AllowDelete = false
	p := BuildPosture(cfg)
	if p.EvidenceExportDelete.DeleteEnabled {
		t.Fatalf("expected delete disabled")
	}
	if len(p.EvidenceExportDelete.DeleteScope) != 0 {
		t.Fatalf("expected no delete scope when disabled")
	}
}

func TestBuildPostureInferenceDegradedWhenEnabledWithoutReadyProvider(t *testing.T) {
	cfg := config.Default()
	cfg.Platform.Inference.Enabled = true
	cfg.Platform.Inference.Ollama.Enabled = false
	cfg.Platform.Inference.LlamaCPP.Enabled = false
	p := BuildPosture(cfg)
	if !p.InferenceDegraded {
		t.Fatalf("expected inference degraded posture")
	}
	if p.InferenceRuntimeReady {
		t.Fatalf("expected runtime not ready")
	}
	if p.InferenceCaveat == "" {
		t.Fatalf("expected inference caveat text")
	}
}
