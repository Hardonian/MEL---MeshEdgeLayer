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
	if len(rows) < 2 {
		t.Fatalf("expected both schema migrations, got %v", rows)
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
