package service

import (
	"testing"
	"time"

	"github.com/mel-project/mel/internal/models"
)

func TestRecordRecommendationOutcome_Persists(t *testing.T) {
	a := newSoDTestApp(t)
	id := "inc-rec-outcome"
	if err := a.DB.UpsertIncident(models.Incident{
		ID:           id,
		Category:     "transport",
		Severity:     "warning",
		Title:        "r",
		Summary:      "s",
		ResourceType: "transport",
		ResourceID:   "mqtt-sod",
		State:        "open",
		OccurredAt:   time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.RecordRecommendationOutcome(id, "op-a", models.IncidentRecommendationOutcomeRequest{
		RecommendationID: "guide-incident-record",
		Outcome:          "accepted",
		Note:             "ok",
	}); err != nil {
		t.Fatal(err)
	}
	rows, err := a.DB.RecommendationOutcomesForIncident(id, 10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows=%v err=%v", rows, err)
	}
}

func TestPatchIncidentWorkflow_ReviewState(t *testing.T) {
	a := newSoDTestApp(t)
	id := "inc-wf-1"
	if err := a.DB.UpsertIncident(models.Incident{
		ID:           id,
		Category:     "transport",
		Severity:     "warning",
		Title:        "r",
		Summary:      "s",
		ResourceType: "transport",
		ResourceID:   "mqtt-sod",
		State:        "open",
		OccurredAt:   time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	rs := "investigating"
	if err := a.PatchIncidentWorkflow(id, "op-b", models.IncidentWorkflowPatch{ReviewState: &rs}); err != nil {
		t.Fatal(err)
	}
	inc, ok, err := a.DB.IncidentByID(id)
	if err != nil || !ok || inc.ReviewState != "investigating" {
		t.Fatalf("got %+v ok=%v err=%v", inc, ok, err)
	}
}

func TestPatchIncidentWorkflow_ReviewState_Mitigated(t *testing.T) {
	a := newSoDTestApp(t)
	id := "inc-wf-mitigated"
	if err := a.DB.UpsertIncident(models.Incident{
		ID:           id,
		Category:     "transport",
		Severity:     "warning",
		Title:        "r",
		Summary:      "s",
		ResourceType: "transport",
		ResourceID:   "mqtt-sod",
		State:        "open",
		OccurredAt:   time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	rs := "mitigated"
	if err := a.PatchIncidentWorkflow(id, "op-c", models.IncidentWorkflowPatch{ReviewState: &rs}); err != nil {
		t.Fatal(err)
	}
	inc, ok, err := a.DB.IncidentByID(id)
	if err != nil || !ok || inc.ReviewState != "mitigated" {
		t.Fatalf("got %+v ok=%v err=%v", inc, ok, err)
	}
}
