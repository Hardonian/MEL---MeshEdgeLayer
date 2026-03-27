package fleet

import (
	"encoding/json"
	"testing"
)

func TestValidateRemoteEvidenceBundle_Accepted(t *testing.T) {
	b := RemoteEvidenceBundle{
		SchemaVersion: RemoteEvidenceBundleSchemaVersion,
		Kind:          RemoteEvidenceBundleKind,
		Evidence: EvidenceEnvelope{
			EvidenceClass:       EvidenceClassPacketObservation,
			OriginInstanceID:    "remote-inst-1",
			OriginSiteID:        "site-a",
			OriginClass:         OriginRemoteReported,
			ObservedAt:          "2026-01-02T00:00:00Z",
			PhysicalUncertainty: PhysicalUncertaintyDefault,
		},
	}
	raw, _ := json.Marshal(b)
	got, val, err := ValidateRemoteEvidenceBundle(raw, "site-a", "f1", IngestValidateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if val.Outcome != ValidationAccepted && val.Outcome != ValidationAcceptedWithCaveats {
		t.Fatalf("outcome %s reasons %v", val.Outcome, val.Reasons)
	}
	if got.Evidence.OriginInstanceID != "remote-inst-1" {
		t.Fatal("bundle mismatch")
	}
}

func TestValidateRemoteEvidenceBundle_RejectSiteConflict(t *testing.T) {
	b := RemoteEvidenceBundle{
		SchemaVersion: RemoteEvidenceBundleSchemaVersion,
		Kind:          RemoteEvidenceBundleKind,
		Evidence: EvidenceEnvelope{
			EvidenceClass:       EvidenceClassPacketObservation,
			OriginInstanceID:    "remote-inst-1",
			OriginSiteID:        "site-other",
			OriginClass:         OriginRemoteReported,
			ObservedAt:          "2026-01-02T00:00:00Z",
			PhysicalUncertainty: PhysicalUncertaintyDefault,
		},
	}
	raw, _ := json.Marshal(b)
	_, val, err := ValidateRemoteEvidenceBundle(raw, "site-a", "", IngestValidateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if val.Outcome != ValidationRejected {
		t.Fatalf("expected rejected, got %+v", val)
	}
}

func TestValidateRemoteEvidenceBundle_StrictOrigin(t *testing.T) {
	b := RemoteEvidenceBundle{
		SchemaVersion:           RemoteEvidenceBundleSchemaVersion,
		Kind:                    RemoteEvidenceBundleKind,
		ClaimedOriginInstanceID: "wrong",
		Evidence: EvidenceEnvelope{
			EvidenceClass:       EvidenceClassPacketObservation,
			OriginInstanceID:    "right",
			OriginSiteID:        "site-a",
			OriginClass:         OriginRemoteReported,
			ObservedAt:          "2026-01-02T00:00:00Z",
			PhysicalUncertainty: PhysicalUncertaintyDefault,
		},
	}
	raw, _ := json.Marshal(b)
	_, val, err := ValidateRemoteEvidenceBundle(raw, "site-a", "", IngestValidateOptions{StrictOriginMatch: true})
	if err != nil {
		t.Fatal(err)
	}
	if val.Outcome != ValidationRejected {
		t.Fatalf("expected strict reject, got %+v", val)
	}
}

func TestValidateRemoteEvidenceBundle_RejectsMismatchedEventOrigin(t *testing.T) {
	b := RemoteEvidenceBundle{
		SchemaVersion: RemoteEvidenceBundleSchemaVersion,
		Kind:          RemoteEvidenceBundleKind,
		Evidence: EvidenceEnvelope{
			EvidenceClass:       EvidenceClassPacketObservation,
			OriginInstanceID:    "remote-inst-1",
			OriginSiteID:        "site-a",
			OriginClass:         OriginRemoteReported,
			CorrelationID:       "corr-1",
			ObservedAt:          "2026-01-02T00:00:00Z",
			PhysicalUncertainty: PhysicalUncertaintyDefault,
		},
		Event: &EventEnvelope{
			EventID:          "evt-1",
			EventType:        "packet_observation",
			Summary:          "remote packet observed",
			OriginInstanceID: "remote-inst-2",
			OriginSiteID:     "site-a",
			CorrelationID:    "corr-1",
			ObservedAt:       "2026-01-02T00:00:00Z",
		},
	}
	raw, _ := json.Marshal(b)
	_, val, err := ValidateRemoteEvidenceBundle(raw, "site-a", "", IngestValidateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if val.Outcome != ValidationRejected {
		t.Fatalf("expected rejected event mismatch, got %+v", val)
	}
	if len(val.Reasons) == 0 || val.Reasons[0] != ReasonEventOriginMismatch {
		t.Fatalf("expected event origin mismatch reason, got %+v", val.Reasons)
	}
}

func TestValidateRemoteEvidenceImportPayload_AcceptsPartialBatch(t *testing.T) {
	payload := RemoteEvidenceBatch{
		SchemaVersion: RemoteEvidenceBatchSchemaVersion,
		Kind:          RemoteEvidenceBatchKind,
		Items: []RemoteEvidenceBundle{
			{
				SchemaVersion: RemoteEvidenceBundleSchemaVersion,
				Kind:          RemoteEvidenceBundleKind,
				Evidence: EvidenceEnvelope{
					EvidenceClass:       EvidenceClassPacketObservation,
					OriginInstanceID:    "remote-good",
					OriginSiteID:        "site-a",
					OriginClass:         OriginRemoteReported,
					ObservedAt:          "2026-01-02T00:00:00Z",
					PhysicalUncertainty: PhysicalUncertaintyDefault,
				},
			},
			{
				SchemaVersion: RemoteEvidenceBundleSchemaVersion,
				Kind:          RemoteEvidenceBundleKind,
				Evidence: EvidenceEnvelope{
					EvidenceClass:       EvidenceClassPacketObservation,
					OriginSiteID:        "site-a",
					OriginClass:         OriginRemoteReported,
					ObservedAt:          "2026-01-02T00:01:00Z",
					PhysicalUncertainty: PhysicalUncertaintyDefault,
				},
			},
		},
	}
	raw, _ := json.Marshal(payload)
	got, err := ValidateRemoteEvidenceImportPayload(raw, "site-a", "", IngestValidateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Validation.Outcome != ValidationAcceptedPartial {
		t.Fatalf("expected partial batch acceptance, got %+v", got.Validation)
	}
	if got.Validation.ItemCount != 2 || got.Validation.RejectedCount != 1 {
		t.Fatalf("unexpected batch counts %+v", got.Validation)
	}
	if len(got.Items) != 2 {
		t.Fatalf("expected 2 item results, got %d", len(got.Items))
	}
	if got.Items[0].Validation.Outcome == ValidationRejected {
		t.Fatalf("expected first item accepted, got %+v", got.Items[0].Validation)
	}
	if got.Items[1].Validation.Outcome != ValidationRejected {
		t.Fatalf("expected second item rejected, got %+v", got.Items[1].Validation)
	}
}
