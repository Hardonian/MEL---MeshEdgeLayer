package planning

import (
	"path/filepath"
	"testing"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
)

func TestSaveAndGetPlan(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "plan.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	p := DeploymentPlan{Title: "t", Intent: "i", Status: "draft"}
	if err := SavePlan(database, &p); err != nil {
		t.Fatal(err)
	}
	id := p.PlanID
	got, ok, err := GetPlan(database, id)
	if err != nil || !ok || got.Title != "t" {
		t.Fatalf("get plan: ok=%v err=%v %+v", ok, err, got)
	}
}
