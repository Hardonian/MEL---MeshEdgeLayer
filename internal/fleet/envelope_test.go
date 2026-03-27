package fleet

import (
	"path/filepath"
	"testing"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
)

func TestBuildTruthSummaryWithDB(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	cfg.Storage.DataDir = dir
	cfg.Storage.DatabasePath = filepath.Join(dir, "t.db")
	d, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Scope.SiteID = "s1"
	cfg.Scope.FleetID = "f1"
	cfg.Scope.ExpectedFleetReporterCount = 2
	_ = SyncScopeMetadata(cfg, d)
	s, err := BuildTruthSummary(cfg, d)
	if err != nil {
		t.Fatal(err)
	}
	if s.InstanceID == "" {
		t.Fatal("expected instance id")
	}
	if s.SiteID != "s1" || s.FleetID != "f1" {
		t.Fatalf("site/fleet: %+v", s)
	}
	if s.TruthPosture != TruthPosturePartialFleet {
		t.Fatalf("posture %s", s.TruthPosture)
	}
}
