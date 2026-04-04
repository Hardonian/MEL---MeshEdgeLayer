package db

import (
	"path/filepath"
	"testing"

	"github.com/mel-project/mel/internal/config"
)

func TestOperationalDigestSnapshotAndWindow(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = t.TempDir()
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := d.OperationalDigestSnapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snap.OpenIncidents != 0 || snap.ControlActionsTotal != 0 {
		t.Fatalf("expected empty db counts, got %+v", snap)
	}
	win, err := d.OperationalDigestWindow("2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("window: %v", err)
	}
	if win.IncidentsOpened != 0 {
		t.Fatalf("expected zero window opens, got %d", win.IncidentsOpened)
	}
}
