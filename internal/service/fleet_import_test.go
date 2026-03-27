package service

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/fleet"
)

func TestImportRemoteEvidenceBundle_AcceptedAndAudited(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	cfg.Storage.DataDir = dir
	cfg.Storage.DatabasePath = filepath.Join(dir, "m.db")
	cfg.Scope.SiteID = "site-a"
	app, err := New(cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	b := fleet.RemoteEvidenceBundle{
		SchemaVersion: fleet.RemoteEvidenceBundleSchemaVersion,
		Kind:          fleet.RemoteEvidenceBundleKind,
		Evidence: fleet.EvidenceEnvelope{
			EvidenceClass:       fleet.EvidenceClassPacketObservation,
			OriginInstanceID:    "remote-1",
			OriginSiteID:        "site-a",
			OriginClass:         fleet.OriginRemoteReported,
			ObservedAt:          "2026-01-01T00:00:00Z",
			PhysicalUncertainty: fleet.PhysicalUncertaintyDefault,
		},
	}
	raw, _ := json.Marshal(b)
	out, err := app.ImportRemoteEvidenceBundle(raw, false, "op")
	if err != nil {
		t.Fatal(err)
	}
	if out["status"] != "imported" {
		t.Fatalf("status %+v", out)
	}
	if _, ok := out["item_inspections"]; !ok {
		t.Fatalf("expected item inspections in import output, got %+v", out)
	}
	if _, ok := out["batch_id"]; !ok {
		t.Fatalf("expected batch id in import output, got %+v", out)
	}
	rows, err := app.ListImportedRemoteEvidence(5)
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows %v err %v", len(rows), err)
	}
	rowsJSON, err := app.DB.QueryRows("SELECT event_type, details_json FROM timeline_events ORDER BY event_time DESC LIMIT 10;")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range rowsJSON {
		if row["event_type"] != "remote_evidence_import_item" {
			continue
		}
		found = true
		var details map[string]any
		if err := json.Unmarshal([]byte(row["details_json"].(string)), &details); err != nil {
			t.Fatal(err)
		}
		if len(details) == 0 {
			t.Fatal("expected timeline details for import")
		}
		if _, ok := details["canonical_evidence_envelope"]; !ok {
			t.Fatalf("expected canonical evidence envelope in timeline details, got %+v", details)
		}
		if _, ok := details["local_import_event_envelope"]; !ok {
			t.Fatalf("expected local import event envelope in timeline details, got %+v", details)
		}
		break
	}
	if !found {
		t.Fatal("timeline missing remote_evidence_import_item")
	}
}
