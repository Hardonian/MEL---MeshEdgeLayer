package db

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
)

func TestIncidentActionOutcomeSnapshots_UpsertAndQuery(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err = d.UpsertIncidentActionOutcomeSnapshot(IncidentActionOutcomeSnapshotRecord{
		SignatureKey:          "sig-test",
		IncidentID:            "inc-1",
		ActionID:              "act-1",
		ActionType:            "restart_transport",
		DerivedClassification: "improvement_observed",
		EvidenceSufficiency:   "sufficient",
		PreActionSummary:      map[string]any{"dead_letters_count": 3},
		PostActionSummary:     map[string]any{"dead_letters_count": 1},
		ObservedSignalCount:   2,
		Caveats:               []string{"Temporal association only; this is not causal proof."},
		WindowStart:           now,
		WindowEnd:             now,
		DerivedAt:             now,
	})
	if err != nil {
		t.Fatalf("upsert snapshot: %v", err)
	}
	rows, err := d.ActionOutcomeSnapshotsBySignature("sig-test", "inc-current", 10)
	if err != nil {
		t.Fatalf("query snapshots: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(rows))
	}
	if rows[0].ActionID != "act-1" || rows[0].DerivedClassification != "improvement_observed" {
		t.Fatalf("unexpected snapshot row: %+v", rows[0])
	}
	if got := rows[0].PreActionSummary["dead_letters_count"]; got == nil {
		t.Fatalf("expected pre_action_summary to be populated")
	}
}
