package service

import (
	"testing"
	"time"

	"github.com/mel-project/mel/internal/models"
)

func TestIncidentHandoff_PersistsOwnerAndSummary(t *testing.T) {
	a := newTrustTestApp(t)
	inc := models.Incident{
		ID:           "inc-handoff-1",
		Category:     "transport",
		Severity:     "high",
		Title:        "Test incident",
		Summary:      "summary",
		ResourceType: "transport",
		ResourceID:   "t1",
		State:        "open",
		OccurredAt:   time.Now().UTC().Format(time.RFC3339),
	}
	if err := a.DB.UpsertIncident(inc); err != nil {
		t.Fatal(err)
	}
	req := models.IncidentHandoffRequest{
		ToOperatorID:   "operator-b",
		HandoffSummary: "Seeing elevated drops on t1; pending restart approval act-1.",
		PendingActions: []string{"act-1"},
		Risks:          []string{"restart may flap if broker unstable"},
	}
	if err := a.IncidentHandoff("inc-handoff-1", "operator-a", req); err != nil {
		t.Fatal(err)
	}
	got, ok, err := a.DB.IncidentByID("inc-handoff-1")
	if err != nil || !ok {
		t.Fatalf("reload: err=%v ok=%v", err, ok)
	}
	if got.OwnerActorID != "operator-b" {
		t.Errorf("owner: want operator-b, got %q", got.OwnerActorID)
	}
	if got.HandoffSummary != req.HandoffSummary {
		t.Errorf("handoff summary mismatch")
	}
	if len(got.PendingActions) != 1 || got.PendingActions[0] != "act-1" {
		t.Errorf("pending actions: %+v", got.PendingActions)
	}
}
