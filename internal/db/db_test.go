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
	rows, err := d.QueryJSON("SELECT version FROM schema_migrations;")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("expected schema migration row")
	}
}
