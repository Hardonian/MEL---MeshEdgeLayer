package main

import (
	"path/filepath"
	"testing"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/transport"
)

func TestDoctorTransportChecksSerialMissing(t *testing.T) {
	cfg := config.Default()
	cfg.Transports = []config.TransportConfig{{Name: "radio", Type: "serial", Enabled: true, SerialDevice: filepath.Join(t.TempDir(), "missing-tty")}}
	checks := doctorTransportChecks(cfg, nil)
	if len(checks) == 0 {
		t.Fatal("expected doctor findings")
	}
	if checks[0]["guidance"] == "" {
		t.Fatalf("expected operator guidance, got %+v", checks[0])
	}
}

func TestDoctorTransportObservationsHistoricalIngest(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	cfg.Transports = []config.TransportConfig{{Name: "radio", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/US/2/e/#", ClientID: "mel-test"}}
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.InsertMessage(map[string]any{"transport_name": "radio", "packet_id": int64(1), "dedupe_hash": "abc", "channel_id": "", "gateway_id": "", "from_node": int64(1), "to_node": int64(2), "portnum": int64(1), "payload_text": "hi", "payload_json": map[string]any{"transport_name": "radio"}, "raw_hex": "01", "rx_time": "2026-03-18T00:00:00Z", "hop_limit": int64(3), "relay_node": int64(0)}); err != nil {
		t.Fatal(err)
	}
	obs := doctorTransportObservations(cfg, database)
	if len(obs) != 1 || obs[0]["status"] != "historical_ingest_seen" {
		t.Fatalf("unexpected observations: %+v", obs)
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
	if summary[0]["state"] != transport.StateConfigured {
		t.Fatalf("expected truthful offline state, got %+v", summary[0])
	}
}
