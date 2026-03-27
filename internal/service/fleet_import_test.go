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
	rows, err := app.ListImportedRemoteEvidence(5)
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows %v err %v", len(rows), err)
	}
	evs, err := app.Timeline("", "", 20)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range evs {
		if e.EventType == "remote_evidence_import" {
			found = true
			if len(e.Details) == 0 {
				t.Fatal("expected timeline details for import")
			}
			break
		}
	}
	if !found {
		t.Fatal("timeline missing remote_evidence_import")
	}
}
