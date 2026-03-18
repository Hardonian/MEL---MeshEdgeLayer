package main

import (
	"path/filepath"
	"testing"

	"github.com/mel-project/mel/internal/config"
)

func TestDoctorTransportChecksSerialMissing(t *testing.T) {
	cfg := config.Default()
	cfg.Transports = []config.TransportConfig{{Name: "radio", Type: "serial", Enabled: true, SerialDevice: filepath.Join(t.TempDir(), "missing-tty")}}
	checks := doctorTransportChecks(cfg)
	if len(checks) == 0 {
		t.Fatal("expected doctor findings")
	}
}

func TestTransportCapabilitySummary(t *testing.T) {
	cfg := config.Default()
	cfg.Transports = []config.TransportConfig{{Name: "radio", Type: "tcp", Enabled: true, TCPHost: "127.0.0.1", TCPPort: 4403}}
	summary := transportCapabilitySummary(cfg)
	if len(summary) != 1 {
		t.Fatalf("unexpected summary len %d", len(summary))
	}
	caps, ok := summary[0]["capabilities"]
	if !ok || caps == nil {
		t.Fatalf("missing capabilities: %+v", summary[0])
	}
}
