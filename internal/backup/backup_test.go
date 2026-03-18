package backup

import (
	"path/filepath"
	"testing"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
)

func TestCreateAndValidateRestore(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.Default()
	cfg.Storage.DataDir = tmp
	cfg.Storage.DatabasePath = filepath.Join(tmp, "mel.db")
	cfgPath := filepath.Join(tmp, "mel.json")
	if _, err := config.WriteInit(cfgPath); err != nil {
		t.Fatal(err)
	}
	d, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.InsertAuditLog("test", "info", "hello", map[string]any{"ok": true}); err != nil {
		t.Fatal(err)
	}
	bundlePath := filepath.Join(tmp, "bundle.tgz")
	manifest, err := Create(cfg, cfgPath, bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion == "" {
		t.Fatal("expected schema version")
	}
	report, err := ValidateRestore(bundlePath, filepath.Join(tmp, "restore"))
	if err != nil {
		t.Fatal(err)
	}
	if !report.Valid {
		t.Fatalf("expected valid report: %#v", report)
	}
}
