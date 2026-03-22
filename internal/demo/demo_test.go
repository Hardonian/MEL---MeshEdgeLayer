package demo

import (
	"path/filepath"
	"testing"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
)

func TestScenarioCatalogIDsUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range Scenarios() {
		if s.ID == "" {
			t.Fatal("empty scenario id")
		}
		if seen[s.ID] {
			t.Fatalf("duplicate scenario id %q", s.ID)
		}
		seen[s.ID] = true
	}
}

func TestReplayForUnknown(t *testing.T) {
	_, ok := ReplayFor("no-such-scenario")
	if ok {
		t.Fatal("expected false for unknown scenario")
	}
}

func TestSeedHealthyPrivateMesh(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = t.TempDir()
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "demo_sandbox", "demo_sandbox.db")
	sc := ScenarioByID("healthy-private-mesh")
	if sc == nil {
		t.Fatal("missing scenario")
	}
	cfg.Transports = transportsForScenario(sc, cfg)
	_, err := Execute(cfg, "healthy-private-mesh", SeedOptions{Force: true})
	if err != nil {
		t.Fatal(err)
	}
	d, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	n, err := d.Scalar("SELECT COUNT(*) FROM nodes;")
	if err != nil {
		t.Fatal(err)
	}
	if n != "3" {
		t.Fatalf("expected 3 nodes, got %s", n)
	}
}
