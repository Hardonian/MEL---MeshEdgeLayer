package config

import (
	"os"
	"path/filepath"
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

func TestLintConfig(t *testing.T) {
	cfg := Default()
	cfg.Bind.AllowRemote = true
	cfg.Privacy.StorePrecisePositions = true
	cfg.Transports = []TransportConfig{{Name: "a", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/default"}, {Name: "b", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1884", Topic: "msh/public"}}
	lints := LintConfig(cfg)
	if len(lints) < 3 {
		t.Fatalf("expected multiple lints, got %d", len(lints))
	}
}

func TestWriteInit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mel.json")
	cfg, err := WriteInit(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Auth.SessionSecret == "" {
		t.Fatal("expected generated secret")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("unexpected file mode: %o", info.Mode().Perm())
	}
}

func TestValidateDirectTransports(t *testing.T) {
	cfg := Default()
	cfg.Transports = []TransportConfig{{Name: "serial", Type: "serial", Enabled: true, SerialDevice: "/dev/ttyUSB0", SerialBaud: 115200}, {Name: "tcp", Type: "tcp", Enabled: true, TCPHost: "127.0.0.1", TCPPort: 4403}}
	if err := Validate(cfg); err != nil {
		t.Fatal(err)
	}
	lints := LintConfig(cfg)
	if len(lints) == 0 {
		t.Fatal("expected contention lint for multiple direct transports")
	}
}

func TestValidateRejectsInvalidMQTTEndpointAndMissingClientID(t *testing.T) {
	cfg := Default()
	cfg.Transports = []TransportConfig{{Name: "mqtt", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1", Topic: "msh/test"}}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error")
	}
}
