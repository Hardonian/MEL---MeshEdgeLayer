package config

import (
	"os"
	"testing"
)

func TestLoadAndValidate(t *testing.T) {
	t.Setenv("MEL_BIND_API", "127.0.0.1:18080")
	cfg, _, err := Load("../../configs/mel.example.json")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Bind.API != "127.0.0.1:18080" {
		t.Fatalf("env override failed: %s", cfg.Bind.API)
	}
	if err := Validate(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsRemoteWithoutAuth(t *testing.T) {
	cfg := Default()
	cfg.Bind.AllowRemote = true
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error")
	}
	_ = os.Unsetenv("MEL_BIND_API")
}
