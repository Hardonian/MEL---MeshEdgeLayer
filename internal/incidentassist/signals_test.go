package incidentassist

import (
	"testing"

	"github.com/mel-project/mel/internal/models"
)

func TestCompute_EvidenceThin(t *testing.T) {
	inc := models.Incident{ID: "i1", State: "open"}
	intel := &models.IncidentIntelligence{
		EvidenceStrength: "sparse",
		Degraded:         true,
	}
	out := Compute(inc, intel)
	if out == nil || len(out.Signals) == 0 {
		t.Fatal("expected evidence_thin signal")
	}
	found := false
	for _, s := range out.Signals {
		if s.Code == "evidence_thin_review_needed" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("signals %#v", out.Signals)
	}
}

func TestCompute_IngestGraphPressure(t *testing.T) {
	inc := models.Incident{ID: "i2", State: "open"}
	intel := &models.IncidentIntelligence{
		EvidenceStrength: "moderate",
		MeshRoutingCompanion: &models.MeshRoutingIntelCompanion{
			Applicable:            true,
			SuspectedRelayHotspot: true,
		},
	}
	out := Compute(inc, intel)
	found := false
	for _, s := range out.Signals {
		if s.Code == "ingest_graph_pressure_advisory" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("signals %#v", out.Signals)
	}
}
