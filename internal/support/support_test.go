package support

import (
	"path/filepath"
	"testing"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
)

func TestCreateOmitsDoctorWhenNoConfigPath(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	d, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Create(cfg, d, "v-test", "")
	if err != nil {
		t.Fatal(err)
	}
	if b.DoctorJSON != nil {
		t.Fatalf("expected nil doctor without path, got %#v", b.DoctorJSON)
	}
}
