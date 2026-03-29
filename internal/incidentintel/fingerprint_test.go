package incidentintel

import (
	"testing"

	"github.com/mel-project/mel/internal/models"
)

func TestBuildFingerprintV1_DeterministicHash(t *testing.T) {
	inc := models.Incident{
		ID:           "i1",
		Category:     "transport",
		ResourceType: "transport",
		ResourceID:   "mqtt-a",
		OccurredAt:   "2026-03-29T12:00:00Z",
	}
	ev := []models.IncidentEvidenceItem{{Kind: "incident_record", Summary: "x"}}
	fp1 := BuildFingerprintV1(inc, "sig-x", ev, nil, nil)
	fp2 := BuildFingerprintV1(inc, "sig-x", ev, nil, nil)
	if fp1.CanonicalHash != fp2.CanonicalHash {
		t.Fatalf("hash drift: %s vs %s", fp1.CanonicalHash, fp2.CanonicalHash)
	}
}

func TestCompareFingerprints_SameLegacySigCategory(t *testing.T) {
	a := FingerprintV1{LegacySignatureKey: "sig-1", Components: map[string][]string{"anomaly_family": {"transport", "transport"}}}
	b := FingerprintV1{LegacySignatureKey: "sig-1", Components: map[string][]string{"anomaly_family": {"other", "other"}}}
	sim := CompareFingerprints(a, b)
	if sim.Category != "same_recurring_signature_bucket" {
		t.Fatalf("got %s", sim.Category)
	}
}

func TestCompareFingerprints_WeightedOverlap(t *testing.T) {
	a := FingerprintV1{
		LegacySignatureKey: "a",
		Components: map[string][]string{
			"anomaly_family":       {"transport", "transport"},
			"evidence_chain_kinds": {"incident_record", "transport_alert"},
		},
	}
	b := FingerprintV1{
		LegacySignatureKey: "b",
		Components: map[string][]string{
			"anomaly_family":       {"transport", "transport"},
			"evidence_chain_kinds": {"incident_record", "transport_alert"},
		},
	}
	sim := CompareFingerprints(a, b)
	if sim.Score < 0.5 {
		t.Fatalf("expected higher score, got %v", sim.Score)
	}
	if sim.Category == "insufficient_evidence" {
		t.Fatalf("unexpected insufficient: %+v", sim)
	}
}
