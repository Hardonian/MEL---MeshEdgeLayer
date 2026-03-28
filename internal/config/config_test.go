package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestApplyProfileObserve(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs", "profiles"), 0o755); err != nil {
		t.Fatal(err)
	}
	observe := filepath.Join(root, "configs", "profiles", "observe.json")
	if err := os.WriteFile(observe, []byte(`{"control":{"mode":"disabled"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	cfg := Default()
	if err := ApplyProfile(&cfg, "observe"); err != nil {
		t.Fatal(err)
	}
	if cfg.Control.Mode != "disabled" {
		t.Fatalf("expected observe profile to disable control, got %q", cfg.Control.Mode)
	}
}

func TestConfigDiff(t *testing.T) {
	a := Default()
	b := Default()
	b.Control.Mode = "guarded_auto"
	d := Diff(a, b)
	if len(d) == 0 {
		t.Fatal("expected diff entries")
	}
}

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

func TestValidateProductionDeployRequiresStrictAndTransport(t *testing.T) {
	cfg := Default()
	cfg.ProductionDeploy = true
	if err := ValidateProductionDeploy(cfg); err == nil {
		t.Fatal("expected error: no enabled transport")
	}
	cfg.Transports = []TransportConfig{{Name: "mqtt", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/t", ClientID: "x"}}
	if err := normalize(&cfg); err != nil {
		t.Fatal(err)
	}
	if err := Validate(cfg); err != nil {
		t.Fatal(err)
	}
	if err := ValidateProductionDeploy(cfg); err != nil {
		t.Fatalf("expected ok with advisory control and one transport: %v", err)
	}
	cfg.Control.Mode = "guarded_auto"
	if err := ValidateProductionDeploy(cfg); err == nil {
		t.Fatal("expected strict violation for non-advisory control")
	}
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
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("unexpected file mode: %o", info.Mode().Perm())
	}
}

func TestValidateDirectTransports(t *testing.T) {
	cfg := Default()
	cfg.Transports = []TransportConfig{{Name: "serial", Type: "serial", Enabled: true, SerialDevice: "/dev/ttyUSB0", SerialBaud: 115200}, {Name: "tcp", Type: "tcp", Enabled: true, TCPHost: "127.0.0.1", TCPPort: 4403}}
	if err := normalize(&cfg); err != nil {
		t.Fatal(err)
	}
	if err := Validate(cfg); err != nil {
		t.Fatal(err)
	}
	lints := LintConfig(cfg)
	if len(lints) == 0 {
		t.Fatal("expected contention lint for multiple direct transports")
	}
}

func TestValidateMQTTRequiresClientID(t *testing.T) {
	cfg := Default()
	cfg.Transports = []TransportConfig{{Name: "mqtt", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test"}}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected missing client_id validation error")
	}
}

func TestNormalizeSetsTransportReliabilityDefaults(t *testing.T) {
	cfg := Default()
	cfg.Transports = []TransportConfig{{Name: "mqtt", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test", ClientID: "mel-test"}}
	if err := normalize(&cfg); err != nil {
		t.Fatal(err)
	}
	got := cfg.Transports[0]
	if got.MQTTQoS != 1 || got.MQTTKeepAliveSec != 30 || got.ReadTimeoutSec != 15 || got.WriteTimeoutSec != 5 || got.MaxTimeouts != 3 {
		t.Fatalf("unexpected defaults: %+v", got)
	}
}

func TestLintConfigFlagsUnsupportedEnabledTransport(t *testing.T) {
	cfg := Default()
	cfg.Transports = []TransportConfig{{Name: "ble-test", Type: "ble", Enabled: true}}
	lints := LintConfig(cfg)
	if len(lints) == 0 {
		t.Fatal("expected unsupported transport lint")
	}
}

func TestValidateRejectsUnsafeIntelligenceTuning(t *testing.T) {
	cfg := Default()
	cfg.Intelligence.Retention.HealthSnapshotDays = 3
	cfg.Intelligence.Alerts.MinimumStateDurationSeconds = 1
	cfg.Intelligence.Alerts.CooldownSeconds = 1
	cfg.Intelligence.Alerts.RecoveryScoreHealthy = 80
	if err := Validate(cfg); err == nil {
		t.Fatal("expected intelligence validation error")
	}
}

func TestControlApprovalTimeoutNormalization(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{-1, 0},
		{0, 0},
		{300, 300},
		{86400, 86400},
		{86401, 86400},
		{99999, 86400},
	}
	for _, tc := range tests {
		cfg := Default()
		cfg.Control.ApprovalTimeoutSeconds = tc.input
		if err := normalize(&cfg); err != nil {
			t.Fatalf("normalize failed for input %d: %v", tc.input, err)
		}
		if cfg.Control.ApprovalTimeoutSeconds != tc.expected {
			t.Errorf("ApprovalTimeoutSeconds: input=%d expected=%d got=%d",
				tc.input, tc.expected, cfg.Control.ApprovalTimeoutSeconds)
		}
	}
}

func TestControlApprovalTimeoutValidation(t *testing.T) {
	cfg := Default()
	cfg.Control.ApprovalTimeoutSeconds = -5
	// normalize should have clamped it, but test that direct validate rejects raw invalid
	// set to an out-of-range value bypassing normalize
	cfg.Control.ApprovalTimeoutSeconds = 90000
	if err := Validate(cfg); err == nil {
		t.Error("expected validation error for ApprovalTimeoutSeconds > 86400")
	}
}

func TestRequireApprovalForActionTypes(t *testing.T) {
	cfg := Default()
	cfg.Control.RequireApprovalForActionTypes = []string{"restart_transport", "reconfigure_transport"}
	if err := normalize(&cfg); err != nil {
		t.Fatal(err)
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("valid approval config failed: %v", err)
	}
	if len(cfg.Control.RequireApprovalForActionTypes) != 2 {
		t.Errorf("expected 2 approval types, got %d", len(cfg.Control.RequireApprovalForActionTypes))
	}
}

func TestRequireApprovalForHighBlastRadius(t *testing.T) {
	cfg := Default()
	cfg.Control.RequireApprovalForHighBlastRadius = true
	cfg.Control.ApprovalTimeoutSeconds = 300
	if err := Validate(cfg); err != nil {
		t.Fatalf("valid high-blast-radius approval config failed: %v", err)
	}
}
