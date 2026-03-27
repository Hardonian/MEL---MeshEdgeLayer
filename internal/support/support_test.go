package support

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/fleet"
)

func TestCreateOmitsDoctorWhenNoConfigPath(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	d, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Create(cfg, d, "v-test", "", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if b.DoctorJSON != nil {
		t.Fatalf("expected nil doctor without path, got %#v", b.DoctorJSON)
	}
}

func TestCreateIncludesImportedEvidenceInspectionAndTimeline(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Scope.SiteID = "site-a"
	d, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	bundle := fleet.RemoteEvidenceBundle{
		SchemaVersion: fleet.RemoteEvidenceBundleSchemaVersion,
		Kind:          fleet.RemoteEvidenceBundleKind,
		Evidence: fleet.EvidenceEnvelope{
			EvidenceClass:       fleet.EvidenceClassPacketObservation,
			OriginInstanceID:    "remote-1",
			OriginSiteID:        "site-a",
			OriginClass:         fleet.OriginRemoteReported,
			ObservedAt:          "2026-01-01T00:00:00Z",
			ReceivedAt:          "2026-01-01T00:00:05Z",
			CorrelationID:       "corr-1",
			PhysicalUncertainty: fleet.PhysicalUncertaintyDefault,
		},
	}
	validation := fleet.RemoteEvidenceValidation{
		Outcome:         fleet.ValidationAcceptedWithCaveats,
		Reasons:         []fleet.ValidationReasonCode{fleet.CaveatNotCryptographicallyVerified, fleet.CaveatPartialObservationOnly},
		TrustPosture:    fleet.TrustPostureImportedReadOnly,
		OrderingPosture: fleet.TimingOrderReceiveDiffersFromObserved,
		Summary:         "accepted with caveats",
	}
	raw, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	evidenceJSON, err := json.Marshal(bundle.Evidence)
	if err != nil {
		t.Fatal(err)
	}
	validationJSON, err := json.Marshal(validation)
	if err != nil {
		t.Fatal(err)
	}
	localID, err := d.EnsureInstanceID()
	if err != nil {
		t.Fatal(err)
	}
	if err := d.InsertImportedRemoteEvidence(db.ImportedRemoteEvidenceRecord{
		ID:                     "imp-1",
		ImportedAt:             "2026-01-01T00:10:00Z",
		LocalInstanceID:        localID,
		Validation:             validationJSON,
		Bundle:                 raw,
		Evidence:               evidenceJSON,
		OriginInstanceID:       bundle.Evidence.OriginInstanceID,
		OriginSiteID:           bundle.Evidence.OriginSiteID,
		EvidenceClass:          string(bundle.Evidence.EvidenceClass),
		ObservationOriginClass: string(bundle.Evidence.OriginClass),
	}); err != nil {
		t.Fatal(err)
	}
	if err := d.InsertTimelineEvent(db.TimelineEvent{
		EventID:          "imp-1",
		EventTime:        "2026-01-01T00:10:00Z",
		EventType:        "remote_evidence_import",
		Summary:          "remote evidence import imp-1",
		Severity:         "info",
		ActorID:          "op",
		ResourceID:       "imp-1",
		ScopePosture:     "remote_imported",
		OriginInstanceID: bundle.Evidence.OriginInstanceID,
		TimingPosture:    string(validation.OrderingPosture),
		ImportID:         "imp-1",
		Details:          map[string]any{"canonical_evidence_envelope": bundle.Evidence},
	}); err != nil {
		t.Fatal(err)
	}

	b, err := Create(cfg, d, "v-test", "", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(b.ImportedRemoteEvidence) != 1 {
		t.Fatalf("expected imported remote evidence rows, got %d", len(b.ImportedRemoteEvidence))
	}
	if len(b.ImportedRemoteEvidenceInspections) != 1 {
		t.Fatalf("expected imported remote evidence inspections, got %d", len(b.ImportedRemoteEvidenceInspections))
	}
	if len(b.RemoteEvidenceTimeline) != 1 {
		t.Fatalf("expected remote evidence timeline rows, got %d", len(b.RemoteEvidenceTimeline))
	}
	if b.RemoteEvidenceTimeline[0].EventType != "remote_evidence_import" {
		t.Fatalf("unexpected timeline row %+v", b.RemoteEvidenceTimeline[0])
	}
	// Contract: FullTimeline must include all events (not just remote imports).
	if len(b.FullTimeline) < 1 {
		t.Fatalf("expected full timeline to contain at least 1 event, got %d", len(b.FullTimeline))
	}
	foundRemote := false
	for _, ev := range b.FullTimeline {
		if ev.EventType == "remote_evidence_import" {
			foundRemote = true
		}
	}
	if !foundRemote {
		t.Fatalf("expected full timeline to contain remote_evidence_import event")
	}
}

func TestBundleZipContainsManifestAndSections(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	d, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Insert a timeline event so timeline.json is generated.
	if err := d.InsertTimelineEvent(db.TimelineEvent{
		EventID:          "test-ev-1",
		EventTime:        "2026-01-01T00:00:00Z",
		EventType:        "control_action",
		Summary:          "test control action",
		Severity:         "info",
		ActorID:          "op",
		ResourceID:       "act-1",
		ScopePosture:     "local",
		OriginInstanceID: "local",
		TimingPosture:    "local_ordered",
	}); err != nil {
		t.Fatal(err)
	}

	b, err := Create(cfg, d, "v-test", "", time.Time{})
	if err != nil {
		t.Fatal(err)
	}

	zipBytes, err := b.ToZip()
	if err != nil {
		t.Fatalf("ToZip failed: %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("could not open zip: %v", err)
	}

	fileNames := make(map[string]bool)
	for _, f := range reader.File {
		fileNames[f.Name] = true
	}

	// Contract: MANIFEST.md must always be present.
	if !fileNames["MANIFEST.md"] {
		t.Fatal("MANIFEST.md missing from support bundle zip")
	}
	// Contract: bundle.json must always be present.
	if !fileNames["bundle.json"] {
		t.Fatal("bundle.json missing from support bundle zip")
	}
	// Contract: timeline.json must be present when there are timeline events.
	if !fileNames["timeline.json"] {
		t.Fatal("timeline.json missing from support bundle zip (expected because we inserted a timeline event)")
	}
}
