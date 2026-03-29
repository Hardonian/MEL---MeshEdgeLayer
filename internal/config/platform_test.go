package config

import (
	"strings"
	"testing"
)

func TestPlatformDefaultsNormalize(t *testing.T) {
	cfg := Default()
	cfg.Platform = PlatformConfig{}
	if err := normalize(&cfg); err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if cfg.Platform.Mode != "self_hosted" {
		t.Fatalf("expected self_hosted mode, got %q", cfg.Platform.Mode)
	}
	if cfg.Platform.EventBus.Provider == "" {
		t.Fatalf("expected event bus provider default")
	}
}

func TestPlatformValidationRejectsHiddenTelemetry(t *testing.T) {
	cfg := Default()
	cfg.Platform.Telemetry.Enabled = true
	cfg.Platform.Telemetry.AllowOutbound = false
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "allow_outbound") {
		t.Fatalf("expected allow_outbound validation error, got %v", err)
	}
}

func TestPlatformValidationRequiresInferenceProvider(t *testing.T) {
	cfg := Default()
	cfg.Platform.Inference.Enabled = true
	cfg.Platform.Inference.DefaultProvider = "ollama"
	cfg.Platform.Inference.Ollama.Enabled = false
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "default_provider=ollama") {
		t.Fatalf("expected ollama provider validation error, got %v", err)
	}
}
