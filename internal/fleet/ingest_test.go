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
