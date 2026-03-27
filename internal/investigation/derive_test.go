package investigation

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/fleet"
	"github.com/mel-project/mel/internal/transport"
)

func TestDeriveBuildsCanonicalCasesWithExplicitBoundaries(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Scope.SiteID = "site-a"
	cfg.Scope.ExpectedFleetReporterCount = 3
	cfg.Transports = []config.TransportConfig{{
		Name:     "mqtt",
		Type:     "mqtt",
		Enabled:  true,
		Endpoint: "127.0.0.1:1883",
		Topic:    "msh/test",
		ClientID: "mel-test",
	}}

	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.UpsertTransportRuntime(db.TransportRuntime{
		Name:          "mqtt",
		Type:          "mqtt",
		Source:        "127.0.0.1:1883",
		Enabled:       true,
		State:         transport.StateFailed,
		Detail:        "connect failed",
		LastError:     "broker unreachable",
		FailureCount:  3,
		UpdatedAt:     "2026-03-27T00:00:00Z",
		LastFailureAt: "2026-03-27T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	runtimeStates, err := database.TransportRuntimeStatuses()
	if err != nil {
		t.Fatal(err)
	}
	summary := Derive(cfg, database, []transport.Health{{
		Name:         "mqtt",
		Type:         "mqtt",
		Source:       "127.0.0.1:1883",
		State:        transport.StateFailed,
		LastError:    "broker unreachable",
		FailureCount: 3,
	}}, runtimeStates, time.Date(2026, 3, 27, 1, 0, 0, 0, time.UTC))

	transportCase := findCaseByKind(summary.Cases, CaseTransportDegradation)
	if transportCase == nil {
		t.Fatal("expected transport degradation case")
	}
	if !strings.Contains(transportCase.OutOfScope, "Do not infer fleet-wide outage") {
		t.Fatalf("expected bounded out-of-scope guidance, got %q", transportCase.OutOfScope)
	}
	if len(transportCase.RelatedRecords) == 0 {
		t.Fatal("expected related records on transport case")
	}
	if transportCase.Timing.PrimaryPosture != CaseTimingLocallyOrdered {
		t.Fatalf("expected locally ordered transport case timing, got %+v", transportCase.Timing)
	}

	fleetCase := findCaseByKind(summary.Cases, CasePartialFleetVisibility)
	if fleetCase == nil {
		t.Fatal("expected partial fleet visibility case")
	}
	if !strings.Contains(fleetCase.OutOfScope, "Do not claim fleet-wide health") {
		t.Fatalf("expected no-fake-fleet-certainty guidance, got %q", fleetCase.OutOfScope)
	}
	if summary.CaseCounts.ActiveAttentionCases == 0 {
		t.Fatalf("expected active attention case counts, got %+v", summary.CaseCounts)
	}
}

func TestDeriveBuildsImportedCaseTimelineWithoutFalseLocalConfirmation(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Scope.SiteID = "site-a"

	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	seedInvestigationRemoteImportFixture(t, cfg, database, "batch-investigation-1")

	summary := Derive(cfg, database, nil, nil, time.Date(2026, 3, 27, 1, 0, 0, 0, time.UTC))
	detail, ok := summary.CaseDetail("case:import:historical-context")
	if !ok {
		t.Fatal("expected imported historical case detail")
	}
	if detail.Case.Timing.PrimaryPosture != CaseTimingHistoricalImportNotLive {
		t.Fatalf("expected imported historical timing posture, got %+v", detail.Case.Timing)
	}
	if !caseHasGap(detail.EvidenceGaps, GapNoLocalConfirmation) {
		t.Fatalf("expected no-local-confirmation gap, got %+v", detail.EvidenceGaps)
	}
	if len(detail.LinkedEvents) < 2 {
		t.Fatalf("expected linked import events, got %+v", detail.LinkedEvents)
	}
	foundImportValidation := false
	foundMergeEvolution := false
	for _, event := range detail.LinkedEvents {
		if event.RelationType == CaseEventRelationImportValidation {
			foundImportValidation = true
		}
	}
	for _, entry := range detail.Evolution {
		for _, reason := range entry.ReasonCodes {
			if reason == CaseEvolutionMergeDispositionChanged {
				foundMergeEvolution = true
			}
		}
	}
	if !foundImportValidation {
		t.Fatalf("expected import validation event in case timeline, got %+v", detail.LinkedEvents)
	}
	if !foundMergeEvolution {
		t.Fatalf("expected merge evolution entry, got %+v", detail.Evolution)
	}
}

func findCaseByKind(cases []Case, kind CaseKind) *Case {
	for i := range cases {
		if cases[i].Kind == kind {
			return &cases[i]
		}
	}
	return nil
}

func seedInvestigationRemoteImportFixture(t *testing.T, cfg config.Config, d *db.DB, batchID string) {
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
			CorrelationID:       "corr-investigation-1",
			PhysicalUncertainty: fleet.PhysicalUncertaintyDefault,
		},
		Event: &fleet.EventEnvelope{
			EventID:          "evt-investigation-1",
			EventType:        "packet_observation",
			Summary:          "remote packet observed",
			OriginInstanceID: "remote-1",
			OriginSiteID:     "site-a",
			CorrelationID:    "corr-investigation-1",
			ObservedAt:       "2026-01-01T00:00:00Z",
			RecordedAt:       "2026-01-01T00:00:05Z",
		},
	}

	rawPayload, err := json.Marshal(fleet.RemoteEvidenceBatch{
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
		Reasons:          []fleet.ValidationReasonCode{fleet.CaveatHistoricalImportOnly, fleet.CaveatNotCryptographicallyVerified},
		TrustPosture:     fleet.TrustPostureImportedReadOnly,
		AuthenticityNote: "origin is claimed only",
		OrderingPosture:  fleet.TimingOrderHistoricalImportNotLive,
		Summary:          "accepted with caveats",
	}
	itemValidationJSON, err := json.Marshal(itemValidation)
	if err != nil {
		t.Fatal(err)
	}
	evidenceJSON, err := json.Marshal(bundle.Evidence)
	if err != nil {
		t.Fatal(err)
	}
	eventJSON, err := json.Marshal(bundle.Event)
	if err != nil {
		t.Fatal(err)
	}
	bundleJSON, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	batchValidationJSON, err := json.Marshal(fleet.RemoteEvidenceBatchValidation{
		Outcome:                  fleet.ValidationAcceptedWithCaveats,
		Reasons:                  []fleet.ValidationReasonCode{fleet.CaveatHistoricalImportOnly, fleet.CaveatNotCryptographicallyVerified},
		TrustPosture:             fleet.TrustPostureImportedReadOnly,
		AuthenticityNote:         "origin is claimed only",
		OfflineOnlyNote:          "Offline import only; not live federation.",
		Summary:                  "accepted with caveats",
		StructurallyValid:        true,
		ItemCount:                1,
		AcceptedWithCaveatsCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	capabilityJSON, err := json.Marshal(fleet.DefaultCapabilityPosture())
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
			RawPayload:               rawPayload,
			ItemCount:                1,
			AcceptedWithCaveatsCount: 1,
			Note:                     "Offline remote evidence batch for investigation only.",
		},
		[]db.ImportedRemoteEvidenceRecord{
			{
				ID:                      "imp-investigation-1",
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
				Bundle:                  bundleJSON,
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
				TimingPosture:           string(fleet.TimingOrderHistoricalImportNotLive),
				MergeDisposition:        string(fleet.DedupeConflicting),
				MergeCorrelationID:      bundle.Evidence.CorrelationID,
			},
		},
		[]db.TimelineEvent{
			{
				EventID:          batchID,
				EventTime:        "2026-01-01T00:10:00Z",
				EventType:        "remote_import_batch",
				Summary:          "remote import batch accepted_with_caveats",
				Severity:         "warning",
				ActorID:          "op",
				ResourceID:       batchID,
				ScopePosture:     "remote_import_batch",
				TimingPosture:    string(fleet.TimingOrderHistoricalImportNotLive),
				MergeDisposition: "raw_only",
				ImportID:         batchID,
				Details:          map[string]any{"batch_id": batchID},
			},
			{
				EventID:            "imp-investigation-1:audit",
				EventTime:          "2026-01-01T00:10:00Z",
				EventType:          "remote_evidence_import_item",
				Summary:            "remote evidence import imp-investigation-1",
				Severity:           "warning",
				ActorID:            "op",
				ResourceID:         "imp-investigation-1",
				ScopePosture:       "remote_imported",
				OriginInstanceID:   bundle.Evidence.OriginInstanceID,
				TimingPosture:      string(fleet.TimingOrderHistoricalImportNotLive),
				MergeDisposition:   "raw_only",
				MergeCorrelationID: bundle.Evidence.CorrelationID,
				ImportID:           "imp-investigation-1",
				Details:            map[string]any{"batch_id": batchID, "canonical_evidence_envelope": bundle.Evidence},
			},
			{
				EventID:            "imp-investigation-1:remote_event",
				EventTime:          "2026-01-01T00:00:00Z",
				EventType:          "remote_event_materialized",
				Summary:            "remote event from remote-1: remote packet observed",
				Severity:           "info",
				ActorID:            "remote_import",
				ResourceID:         "imp-investigation-1",
				ScopePosture:       "remote_reported",
				OriginInstanceID:   bundle.Evidence.OriginInstanceID,
				TimingPosture:      string(fleet.TimingOrderHistoricalImportNotLive),
				MergeDisposition:   string(fleet.DedupeConflicting),
				MergeCorrelationID: bundle.Evidence.CorrelationID,
				ImportID:           "imp-investigation-1",
				Details: map[string]any{
					"timing": map[string]any{
						"observed_at": bundle.Evidence.ObservedAt,
						"received_at": bundle.Evidence.ReceivedAt,
						"imported_at": "2026-01-01T00:10:00Z",
					},
					"canonical_evidence_envelope": bundle.Evidence,
					"remote_event_envelope":       bundle.Event,
				},
			},
		},
	); err != nil {
		t.Fatal(err)
	}
}
