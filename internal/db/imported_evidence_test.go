package db

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/mel-project/mel/internal/config"
)

func TestImportedRemoteEvidenceRoundTrip(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	val := map[string]any{"outcome": "accepted", "summary": "ok"}
	bundle := map[string]any{
		"schema_version": "1.0",
		"kind":           "mel_remote_evidence_bundle",
		"evidence": map[string]any{
			"evidence_class":               "packet_observation",
			"origin_instance_id":           "o1",
			"observation_origin_class":     "remote_reported",
			"physical_uncertainty_posture": "partial_observation_clock_skew_duplication_delay",
		},
	}
	valJ, _ := json.Marshal(val)
	bundleJ, _ := json.Marshal(bundle)
	evJ, _ := json.Marshal(bundle["evidence"])
	rec := ImportedRemoteEvidenceRecord{
		ID:                     "imp-test-1",
		ImportedAt:             "2026-01-01T00:00:00Z",
		LocalInstanceID:        "local-1",
		Validation:             valJ,
		Bundle:                 bundleJ,
		Evidence:               evJ,
		OriginInstanceID:       "o1",
		EvidenceClass:          "packet_observation",
		ObservationOriginClass: "remote_reported",
	}
	if err := d.InsertImportedRemoteEvidence(rec); err != nil {
		t.Fatal(err)
	}
	list, err := d.ListImportedRemoteEvidence(10)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v len=%d", err, len(list))
	}
	got, ok, err := d.GetImportedRemoteEvidence("imp-test-1")
	if err != nil || !ok || got.ID != "imp-test-1" {
		t.Fatalf("get: ok=%v err=%v id=%s", ok, err, got.ID)
	}
}
