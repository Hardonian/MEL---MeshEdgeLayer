package db

import (
	"github.com/mel-project/mel/internal/config"
	"path/filepath"
	"testing"
)

func TestOpenAppliesMigration(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := d.QueryJSON("SELECT version FROM schema_migrations ORDER BY version;")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 4 {
		t.Fatalf("expected schema migrations including runtime evidence, got %v", rows)
	}
}

func TestInsertMessageReportsDedupedWrite(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	msg := map[string]any{"transport_name": "radio", "packet_id": int64(1), "dedupe_hash": "abc", "channel_id": "", "gateway_id": "", "from_node": int64(1), "to_node": int64(2), "portnum": int64(1), "payload_text": "hi", "payload_json": map[string]any{"transport_name": "radio"}, "raw_hex": "01", "rx_time": "2026-03-18T00:00:00Z", "hop_limit": int64(3), "relay_node": int64(0)}
	stored, err := d.InsertMessage(msg)
	if err != nil {
		t.Fatal(err)
	}
	if !stored {
		t.Fatal("expected initial insert to store")
	}
	stored, err = d.InsertMessage(msg)
	if err != nil {
		t.Fatal(err)
	}
	if stored {
		t.Fatal("expected duplicate insert to be ignored")
	}
}

func TestInsertDeadLetter(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.InsertDeadLetter(DeadLetter{TransportName: "mqtt", Topic: "msh/test", Reason: "parse failure", PayloadHex: "0102", Details: map[string]any{"error": "boom"}}); err != nil {
		t.Fatal(err)
	}
	rows, err := d.QueryJSON("SELECT transport_name, reason, payload_hex FROM dead_letters;")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["reason"] != "parse failure" {
		t.Fatalf("unexpected dead letter rows: %+v", rows)
	}
}

func TestUpsertTransportRuntimePersistsEvidence(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertTransportRuntime(TransportRuntime{
		Name:            "mqtt",
		Type:            "mqtt",
		Source:          "127.0.0.1:1883",
		Enabled:         true,
		State:           "connected_no_ingest",
		Detail:          "connected; waiting for broker heartbeat or publish",
		LastHeartbeatAt: "2026-03-19T00:00:03Z",
		PacketsDropped:  2,
		Reconnects:      4,
		Timeouts:        1,
	}); err != nil {
		t.Fatal(err)
	}
	rows, err := d.QueryJSON("SELECT transport_name, last_heartbeat_at, packets_dropped, reconnect_attempts, consecutive_timeouts FROM transport_runtime_evidence;")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["last_heartbeat_at"] != "2026-03-19T00:00:03Z" || rows[0]["reconnect_attempts"] != "4" {
		t.Fatalf("unexpected transport runtime evidence rows: %+v", rows)
	}
}
