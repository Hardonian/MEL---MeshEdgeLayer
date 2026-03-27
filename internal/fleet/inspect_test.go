package fleet

import (
	"encoding/json"
	"testing"

	"github.com/mel-project/mel/internal/db"
)

func TestInspectImportedRemoteEvidenceRecord_BuildsTimingAndRelatedEvidence(t *testing.T) {
	truth := FleetTruthSummary{
		InstanceID: "local-inst",
		SiteID:     "site-a",
		FleetID:    "fleet-a",
	}
	current := importedEvidenceRecordForTest(t, "imp-current", "2026-01-02T00:10:00Z", RemoteEvidenceBundle{
		SchemaVersion: RemoteEvidenceBundleSchemaVersion,
		Kind:          RemoteEvidenceBundleKind,
		Evidence: EvidenceEnvelope{
			EvidenceClass:       EvidenceClassPacketObservation,
			OriginInstanceID:    "remote-1",
			OriginSiteID:        "site-a",
			ObserverInstanceID:  "observer-1",
			OriginClass:         OriginRemoteReported,
			ObservedAt:          "2026-01-02T00:00:00Z",
			ReceivedAt:          "2026-01-02T00:00:05Z",
			RecordedAt:          "2026-01-02T00:00:06Z",
			EventTimeSrc:        "remote_gateway_clock",
			CorrelationID:       "corr-1",
			PhysicalUncertainty: PhysicalUncertaintyDefault,
			Details:             map[string]any{"hop_limit": 3},
		},
		Event: &EventEnvelope{
			EventID:          "evt-1",
			EventType:        "packet_observation",
			Summary:          "remote packet observed",
			OriginInstanceID: "remote-1",
			OriginSiteID:     "site-a",
			CorrelationID:    "corr-1",
			ObservedAt:       "2026-01-02T00:00:00Z",
			RecordedAt:       "2026-01-02T00:00:06Z",
		},
	}, RemoteEvidenceValidation{
		Outcome:         ValidationAcceptedWithCaveats,
		Reasons:         []ValidationReasonCode{CaveatNotCryptographicallyVerified, CaveatPartialObservationOnly},
		TrustPosture:    TrustPostureImportedReadOnly,
		OrderingPosture: TimingOrderReceiveDiffersFromObserved,
		Summary:         "accepted with caveats",
	}, false)
	prior := importedEvidenceRecordForTest(t, "imp-prior", "2026-01-02T00:05:00Z", RemoteEvidenceBundle{
		SchemaVersion: RemoteEvidenceBundleSchemaVersion,
		Kind:          RemoteEvidenceBundleKind,
		Evidence: EvidenceEnvelope{
			EvidenceClass:       EvidenceClassPacketObservation,
			OriginInstanceID:    "remote-1",
			OriginSiteID:        "site-a",
			ObserverInstanceID:  "observer-1",
			OriginClass:         OriginRemoteReported,
			ObservedAt:          "2026-01-02T00:00:07Z",
			ReceivedAt:          "2026-01-02T00:00:09Z",
			RecordedAt:          "2026-01-02T00:00:10Z",
			EventTimeSrc:        "remote_gateway_clock",
			CorrelationID:       "corr-1",
			PhysicalUncertainty: PhysicalUncertaintyDefault,
			Details:             map[string]any{"hop_limit": 9},
		},
		Event: &EventEnvelope{
			EventID:          "evt-1",
			EventType:        "packet_observation",
			Summary:          "remote packet observed",
			OriginInstanceID: "remote-1",
			OriginSiteID:     "site-a",
			CorrelationID:    "corr-1",
			ObservedAt:       "2026-01-02T00:00:07Z",
			RecordedAt:       "2026-01-02T00:00:10Z",
		},
	}, RemoteEvidenceValidation{
		Outcome:         ValidationAcceptedWithCaveats,
		Reasons:         []ValidationReasonCode{CaveatNotCryptographicallyVerified},
		TrustPosture:    TrustPostureImportedReadOnly,
		OrderingPosture: TimingOrderReceiveDiffersFromObserved,
		Summary:         "accepted with caveats",
	}, false)

	inspection, err := InspectImportedRemoteEvidenceRecord(truth, current, []db.ImportedRemoteEvidenceRecord{current, prior})
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Provenance.LocalInstanceID != "local-inst" {
		t.Fatalf("expected local provenance, got %+v", inspection.Provenance)
	}
	if inspection.RemoteEvent == nil || inspection.RemoteEvent.EventID != "evt-1" {
		t.Fatalf("expected remote event envelope, got %+v", inspection.RemoteEvent)
	}
	if inspection.LocalImportEvent.OriginInstanceID != "local-inst" {
		t.Fatalf("expected local import event origin, got %+v", inspection.LocalImportEvent)
	}
	if inspection.Timing.PrimaryPosture != TimingOrderReceiveDiffersFromObserved {
		t.Fatalf("expected receive/observed posture, got %s", inspection.Timing.PrimaryPosture)
	}
	if len(inspection.RelatedEvidence) != 1 {
		t.Fatalf("expected one related row, got %d", len(inspection.RelatedEvidence))
	}
	if inspection.RelatedEvidence[0].Classification.Disposition != DedupeConflicting {
		t.Fatalf("expected conflicting related row, got %+v", inspection.RelatedEvidence[0].Classification)
	}
	if inspection.MergeInspection.Classification.Disposition != DedupeConflicting {
		t.Fatalf("expected conflicting aggregate merge inspection, got %+v", inspection.MergeInspection)
	}
	if len(inspection.Unknowns) == 0 {
		t.Fatal("expected unknowns to remain explicit")
	}
}

func importedEvidenceRecordForTest(t *testing.T, id, importedAt string, bundle RemoteEvidenceBundle, validation RemoteEvidenceValidation, rejected bool) db.ImportedRemoteEvidenceRecord {
	t.Helper()
	bundleJSON, err := json.Marshal(bundle)
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
	return db.ImportedRemoteEvidenceRecord{
		ID:                     id,
		ImportedAt:             importedAt,
		LocalInstanceID:        "local-inst",
		Validation:             validationJSON,
		Bundle:                 bundleJSON,
		Evidence:               evidenceJSON,
		OriginInstanceID:       bundle.Evidence.OriginInstanceID,
		OriginSiteID:           bundle.Evidence.OriginSiteID,
		EvidenceClass:          string(bundle.Evidence.EvidenceClass),
		ObservationOriginClass: string(bundle.Evidence.OriginClass),
		Rejected:               rejected,
	}
}
