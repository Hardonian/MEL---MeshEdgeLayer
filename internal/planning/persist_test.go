package planning

import (
	"path/filepath"
	"testing"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
)

func TestSaveArtifact_prunesToMaxKeep(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	d, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if err := SaveArtifact(d, "compare", "gh", "aid", map[string]int{"i": i}, 3); err != nil {
			t.Fatal(err)
		}
	}
	count, err := d.Scalar("SELECT COUNT(*) FROM planning_artifacts;")
	if err != nil || count != "3" {
		t.Fatalf("expected 3 artifacts after retention, got %s err=%v", count, err)
	}
}
