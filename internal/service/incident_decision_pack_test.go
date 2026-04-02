package service

import (
	"testing"
	"time"

	"github.com/mel-project/mel/internal/models"
)

func TestIncidentByIDForAPI_IncludesDecisionPack(t *testing.T) {
	a := newTrustTestApp(t)
	inc := models.Incident{
		ID:           "inc-pack-1",
		Category:     "transport",
		Severity:     "high",
		Title:        "Pack test",
		Summary:      "summary",
		ResourceType: "transport",
		ResourceID:   "t-pack",
		State:        "open",
		OccurredAt:   time.Now().UTC().Format(time.RFC3339),
	}
	if err := a.DB.UpsertIncident(inc); err != nil {
		t.Fatal(err)
	}
	got, ok, err := a.IncidentByIDForAPI(inc.ID, true)
	if err != nil || !ok {
		t.Fatalf("IncidentByIDForAPI: err=%v ok=%v", err, ok)
	}
	if got.DecisionPack == nil {
		t.Fatal("expected decision_pack on API incident")
	}
	if got.DecisionPack.SchemaVersion != models.IncidentDecisionPackSchemaVersion {
		t.Fatalf("schema: %q", got.DecisionPack.SchemaVersion)
	}
	if got.DecisionPack.Queue == nil || got.DecisionPack.Queue.WhySurfacedOneLiner == "" {
		t.Fatalf("expected queue why line in pack: %#v", got.DecisionPack.Queue)
	}
	if got.DecisionPack.Readiness == nil || got.DecisionPack.Readiness.ProofpackPath == "" {
		t.Fatalf("expected readiness in pack")
	}
}

func TestPatchIncidentDecisionPackAdjudication_Persists(t *testing.T) {
	a := newTrustTestApp(t)
	inc := models.Incident{
		ID:           "inc-pack-adj-1",
		Category:     "transport",
		Severity:     "high",
		Title:        "Adj test",
		Summary:      "s",
		ResourceType: "transport",
		ResourceID:   "t-adj",
		State:        "open",
		OccurredAt:   time.Now().UTC().Format(time.RFC3339),
	}
	if err := a.DB.UpsertIncident(inc); err != nil {
		t.Fatal(err)
	}
	reviewed := true
	useful := "useful"
	if err := a.PatchIncidentDecisionPackAdjudication(inc.ID, "op-a", models.IncidentDecisionPackAdjudicationPatch{
		Reviewed: &reviewed,
		Useful:   &useful,
	}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := a.DB.GetIncidentDecisionPackAdjudication(inc.ID)
	if err != nil || !ok {
		t.Fatalf("load adjudication: err=%v ok=%v", err, ok)
	}
	if !got.Reviewed || got.Useful != "useful" {
		t.Fatalf("unexpected row: %+v", got)
	}
}
