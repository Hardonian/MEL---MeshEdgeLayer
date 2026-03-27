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
	seedSupportRemoteImportFixture(t, cfg, d, "batch-support-1")

	b, err := Create(cfg, d, "v-test", "", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(b.RemoteImportBatches) != 1 {
		t.Fatalf("expected remote import batch rows, got %d", len(b.RemoteImportBatches))
	}
	if len(b.RemoteImportBatchInspections) != 1 {
		t.Fatalf("expected remote import batch inspections, got %d", len(b.RemoteImportBatchInspections))
	}
	if len(b.ImportedRemoteEvidence) != 1 {
		t.Fatalf("expected imported remote evidence rows, got %d", len(b.ImportedRemoteEvidence))
	}
	if len(b.ImportedRemoteEvidenceInspections) != 1 {
		t.Fatalf("expected imported remote evidence inspections, got %d", len(b.ImportedRemoteEvidenceInspections))
	}
	if b.RemoteEvidenceExchange == nil {
		t.Fatal("expected remote evidence exchange export")
	}
	if b.RemoteEvidenceExchange.Kind != fleet.RemoteEvidenceBatchKind {
		t.Fatalf("unexpected remote evidence exchange kind %+v", b.RemoteEvidenceExchange)
	}
	if len(b.RemoteEvidenceExchange.Items) != 1 {
		t.Fatalf("expected one exported remote evidence item, got %d", len(b.RemoteEvidenceExchange.Items))
	}
	if len(b.RemoteEvidenceTimeline) != 1 {
		t.Fatalf("expected remote evidence timeline rows, got %d", len(b.RemoteEvidenceTimeline))
	}
	if b.RemoteEvidenceTimeline[0].EventType != "remote_evidence_import_item" {
		t.Fatalf("unexpected timeline row %+v", b.RemoteEvidenceTimeline[0])
	}
	if len(b.FullTimeline) < 1 {
		t.Fatalf("expected full timeline to contain at least 1 event, got %d", len(b.FullTimeline))
	}
	foundRemote := false
	for _, ev := range b.FullTimeline {
		if ev.EventType == "remote_evidence_import_item" {
			foundRemote = true
		}
	}
	if !foundRemote {
		t.Fatalf("expected full timeline to contain remote_evidence_import_item event")
	}
}

func seedSupportRemoteImportFixture(t *testing.T, cfg config.Config, d *db.DB, batchID string) {
	t.Helper()
	localID, err := d.EnsureInstanceID()
	if err != nil {
		t.Fatal(err)
	}
	bundle := fleet.RemoteEvidenceBundle{
		SchemaVersion:           fleet.RemoteEvidenceBundleSchemaVersion,
		Kind:                    fleet.RemoteEvidenceBundleKind,
		ClaimedOriginInstanceID: "remote-1",
		ClaimedOriginSiteID:     "site-a",
		ClaimedFleetID:          "fleet-remote",
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
		Event: &fleet.EventEnvelope{
			EventID:          "evt-support-1",
			EventType:        "packet_observation",
			Summary:          "remote packet observed",
			OriginInstanceID: "remote-1",
			OriginSiteID:     "site-a",
			CorrelationID:    "corr-1",
			ObservedAt:       "2026-01-01T00:00:00Z",
			RecordedAt:       "2026-01-01T00:00:05Z",
		},
	}
	raw, err := json.Marshal(fleet.RemoteEvidenceBatch{
		SchemaVersion:     fleet.RemoteEvidenceBatchSchemaVersion,
		Kind:              fleet.RemoteEvidenceBatchKind,
		ExportedAt:        "2026-01-01T00:09:00Z",
		ClaimedOrigin:     fleet.RemoteEvidenceBatchClaimedOrigin{InstanceID: "remote-1", SiteID: "site-a", FleetID: "fleet-remote"},
		CapabilityPosture: fleet.DefaultCapabilityPosture(),
		SourceContext:     fleet.RemoteEvidenceImportSource{SourceType: "file", SourceName: "remote-evidence.json", SourcePath: "/tmp/remote-evidence.json"},
		Items:             []fleet.RemoteEvidenceBundle{bundle},
	})
	if err != nil {
		t.Fatal(err)
	}
	itemValidation := fleet.RemoteEvidenceValidation{
		Outcome:          fleet.ValidationAcceptedWithCaveats,
		Reasons:          []fleet.ValidationReasonCode{fleet.CaveatNotCryptographicallyVerified, fleet.CaveatHistoricalImportOnly, fleet.CaveatPartialObservationOnly, fleet.CaveatReceiveDiffersFromObserved},
		TrustPosture:     fleet.TrustPostureImportedReadOnly,
		AuthenticityNote: "Import authenticity is not cryptographically verified in core MEL; treat claimed origin as unverified unless checked outside MEL.",
		OrderingPosture:  fleet.TimingOrderReceiveDiffersFromObserved,
		Summary:          "accepted with caveats",
	}
	evidenceJSON, err := json.Marshal(bundle.Evidence)
	if err != nil {
		t.Fatal(err)
	}
	eventJSON, err := json.Marshal(bundle.Event)
	if err != nil {
		t.Fatal(err)
	}
	itemValidationJSON, err := json.Marshal(itemValidation)
	if err != nil {
		t.Fatal(err)
	}
	capabilityJSON, err := json.Marshal(fleet.DefaultCapabilityPosture())
	if err != nil {
		t.Fatal(err)
	}
	batchValidationJSON, err := json.Marshal(fleet.RemoteEvidenceBatchValidation{
		Outcome:                  fleet.ValidationAcceptedWithCaveats,
		Reasons:                  []fleet.ValidationReasonCode{fleet.CaveatNotCryptographicallyVerified, fleet.CaveatHistoricalImportOnly, fleet.CaveatUnverifiedOrigin},
		TrustPosture:             fleet.TrustPostureImportedReadOnly,
		AuthenticityNote:         "Import authenticity is not cryptographically verified in core MEL; treat claimed origin as unverified unless checked outside MEL.",
		OfflineOnlyNote:          "Remote evidence import is offline/file-scoped in core MEL; it does not establish live federation, remote execution, or fleet-wide authority.",
		Summary:                  "Accepted 1 item with caveats: offline import remains read-only, historical, and authenticity-unverified.",
		StructurallyValid:        true,
		ItemCount:                1,
		AcceptedWithCaveatsCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	itemBundleJSON, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.PersistRemoteImportBatch(
		db.RemoteImportBatchRecord{
			ID:                       batchID,
			ImportedAt:               "2026-01-01T00:10:00Z",
			LocalInstanceID:          localID,
			LocalSiteID:              cfg.Scope.SiteID,
			SourceType:               "file",
			SourceName:               "remote-evidence.json",
			SourcePath:               "/tmp/remote-evidence.json",
			FormatKind:               fleet.RemoteEvidenceBatchKind,
			SchemaVersion:            fleet.RemoteEvidenceBatchSchemaVersion,
			ClaimedOriginInstanceID:  "remote-1",
			ClaimedOriginSiteID:      "site-a",
			ClaimedFleetID:           "fleet-remote",
			ExportedAt:               "2026-01-01T00:09:00Z",
			CapabilityPosture:        capabilityJSON,
			Validation:               batchValidationJSON,
			RawPayload:               raw,
			ItemCount:                1,
			AcceptedWithCaveatsCount: 1,
			Note:                     "Offline remote evidence batch for audit and investigation only.",
		},
		[]db.ImportedRemoteEvidenceRecord{
			{
				ID:                      "imp-support-1",
				BatchID:                 batchID,
				ItemID:                  batchID + ":001",
				SequenceNo:              1,
				ImportedAt:              "2026-01-01T00:10:00Z",
				LocalInstanceID:         localID,
				LocalSiteID:             cfg.Scope.SiteID,
				SourceType:              "file",
				SourceName:              "remote-evidence.json",
				SourcePath:              "/tmp/remote-evidence.json",
				ValidationStatus:        string(fleet.ValidationAcceptedWithCaveats),
				Validation:              itemValidationJSON,
				Bundle:                  itemBundleJSON,
				Evidence:                evidenceJSON,
				Event:                   eventJSON,
				ClaimedOriginInstanceID: "remote-1",
				ClaimedOriginSiteID:     "site-a",
				ClaimedFleetID:          "fleet-remote",
				OriginInstanceID:        bundle.Evidence.OriginInstanceID,
				OriginSiteID:            bundle.Evidence.OriginSiteID,
				EvidenceClass:           string(bundle.Evidence.EvidenceClass),
				ObservationOriginClass:  string(bundle.Evidence.OriginClass),
				CorrelationID:           bundle.Evidence.CorrelationID,
				ObservedAt:              bundle.Evidence.ObservedAt,
				ReceivedAt:              bundle.Evidence.ReceivedAt,
				TimingPosture:           string(itemValidation.OrderingPosture),
				MergeDisposition:        "raw_only",
				MergeCorrelationID:      bundle.Evidence.CorrelationID,
			},
		},
		[]db.TimelineEvent{
			{
				EventID:            "imp-support-1:audit",
				EventTime:          "2026-01-01T00:10:00Z",
				EventType:          "remote_evidence_import_item",
				Summary:            "remote evidence import imp-support-1",
				Severity:           "info",
				ActorID:            "op",
				ResourceID:         "imp-support-1",
				ScopePosture:       "remote_imported",
				OriginInstanceID:   bundle.Evidence.OriginInstanceID,
				TimingPosture:      string(itemValidation.OrderingPosture),
				MergeDisposition:   "raw_only",
				MergeCorrelationID: bundle.Evidence.CorrelationID,
				ImportID:           "imp-support-1",
				Details:            map[string]any{"batch_id": batchID, "canonical_evidence_envelope": bundle.Evidence},
			},
		},
	); err != nil {
		t.Fatal(err)
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

func TestBundleZipIncludesRemoteEvidenceExportWhenImportsPresent(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Scope.SiteID = "site-a"
	d, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	seedSupportRemoteImportFixture(t, cfg, d, "batch-support-zip")

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
	if !fileNames["imported_evidence.json"] {
		t.Fatal("imported_evidence.json missing from support bundle zip")
	}
	if !fileNames["remote_evidence_export.json"] {
		t.Fatal("remote_evidence_export.json missing from support bundle zip")
	}
}
