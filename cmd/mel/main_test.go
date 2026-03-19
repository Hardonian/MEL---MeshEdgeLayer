package main

import (
	"path/filepath"
	"testing"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	status "github.com/mel-project/mel/internal/status"
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
	inserted, err := database.InsertMessage(map[string]any{
		"transport_name": "radio",
		"packet_id":      int64(1),
		"dedupe_hash":    "abc",
		"channel_id":     "",
		"gateway_id":     "",
		"from_node":      int64(1),
		"to_node":        int64(2),
		"portnum":        int64(1),
		"payload_text":   "hi",
		"payload_json":   map[string]any{"transport_name": "radio", "message_type": "text"},
		"raw_hex":        "01",
		"rx_time":        "2026-03-18T00:00:00Z",
		"hop_limit":      int64(3),
		"relay_node":     int64(0),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !inserted {
		t.Fatal("expected insert")
	}
	snap, err := status.Collect(cfg, database, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Transports) != 1 || snap.Transports[0].EffectiveState != "historical_only" {
		t.Fatalf("unexpected status snapshot: %+v", snap.Transports)
	}
}

func TestStatusCollectsCapabilitySummary(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	cfg.Transports = []config.TransportConfig{{Name: "radio", Type: "tcp", Enabled: true, TCPHost: "127.0.0.1", TCPPort: 4403}}
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := status.Collect(cfg, database, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Transports) != 1 {
		t.Fatalf("unexpected summary len %d", len(snap.Transports))
	}
	if snap.Transports[0].Capabilities.ImplementationStatus == "" {
		t.Fatalf("missing capabilities: %+v", snap.Transports[0])
	}
	if snap.Transports[0].RuntimeState != "configured_not_attempted" {
		t.Fatalf("expected truthful offline state, got %+v", snap.Transports[0])
	}
}
