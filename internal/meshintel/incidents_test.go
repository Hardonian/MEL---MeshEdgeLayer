package meshintel

import (
	"path/filepath"
	"testing"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
)

func TestEvaluateViabilityRegressionOpensIncident(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "inc.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveMeshIntelState(d, MeshIntelStateRow{
		LastGoodViability: string(ViabilityViableLocalMesh),
		LastGoodReadiness: 0.85,
	}); err != nil {
		t.Fatal(err)
	}
	a := Assessment{
		TopologyEnabled: true,
		GraphHash:       "deadbeef",
		AssessmentID:    "test-assess",
		Bootstrap: BootstrapAssessment{
			Viability:               ViabilityIsolated,
			BootstrapReadinessScore: 0.2,
		},
	}
	sig := MessageSignals{TotalMessages: 50}
	for i := 0; i < 3; i++ {
		if err := EvaluateViabilityRegression(d, a, sig, true); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := d.QueryRows(`SELECT id, category FROM incidents WHERE id='meshintel-viability-regression';`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected incident row, got %v", rows)
	}
}
