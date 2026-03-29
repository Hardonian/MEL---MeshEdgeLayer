package service

import (
	"testing"
	"time"

	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
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

func TestIncidentByID_BuildsIncidentIntelligence(t *testing.T) {
	a := newSoDTestApp(t)
	base := time.Now().UTC()
	inc := models.Incident{
		ID:           "inc-intel-1",
		Category:     "transport",
		Severity:     "high",
		Title:        "MQTT timeout cluster",
		Summary:      "timeouts and drops observed",
		ResourceType: "transport",
		ResourceID:   "mqtt-sod",
		State:        "open",
		OccurredAt:   base.Format(time.RFC3339),
		Metadata:     map[string]any{"reason": "timeout_stall"},
	}
	if err := a.DB.UpsertIncident(inc); err != nil {
		t.Fatal(err)
	}
	if err := a.DB.InsertDeadLetter(db.DeadLetter{
		TransportName: "mqtt-sod",
		TransportType: "mqtt",
		Topic:         "topic/a",
		Reason:        "decode_failed",
		PayloadHex:    "01",
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.DB.UpsertTransportAlert(db.TransportAlertRecord{
		ID:               "alert-intel-1",
		TransportName:    "mqtt-sod",
		TransportType:    "mqtt",
		Severity:         "warning",
		Reason:           "timeout_stall",
		Summary:          "timeout stall",
		FirstTriggeredAt: base.Format(time.RFC3339),
		LastUpdatedAt:    base.Format(time.RFC3339),
		Active:           true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.QueueOperatorControlAction("alice", control.ActionTriggerHealthRecheck, "mqtt-sod", "", "", "intel link", 0.8, inc.ID); err != nil {
		t.Fatal(err)
	}

	got, ok, err := a.IncidentByID(inc.ID)
	if err != nil || !ok {
		t.Fatalf("IncidentByID: err=%v ok=%v", err, ok)
	}
	if got.Intelligence == nil {
		t.Fatalf("expected intelligence payload")
	}
	if got.Intelligence.SignatureKey == "" {
		t.Fatalf("expected signature key")
	}
	if len(got.Intelligence.EvidenceItems) < 2 {
		t.Fatalf("expected evidence items, got %d", len(got.Intelligence.EvidenceItems))
	}
}

func TestRecentIncidentsWithLinkedActions_IntelligenceIncludesSimilarity(t *testing.T) {
	a := newSoDTestApp(t)
	now := time.Now().UTC()
	seed := func(id string, occurred time.Time) {
		t.Helper()
		err := a.DB.UpsertIncident(models.Incident{
			ID:           id,
			Category:     "transport",
			Severity:     "warning",
			Title:        "MQTT timeout",
			Summary:      "timeout burst",
			ResourceType: "transport",
			ResourceID:   "mqtt-sod",
			State:        "open",
			OccurredAt:   occurred.Format(time.RFC3339),
			Metadata:     map[string]any{"reason": "timeout_stall"},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	seed("inc-sim-old", now.Add(-2*time.Hour))
	seed("inc-sim-new", now.Add(-1*time.Hour))

	_, err := a.RecentIncidentsWithLinkedActions(10)
	if err != nil {
		t.Fatal(err)
	}
	incs, err := a.RecentIncidentsWithLinkedActions(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(incs) < 2 {
		t.Fatalf("expected incidents, got %d", len(incs))
	}
	found := false
	for _, inc := range incs {
		if inc.ID != "inc-sim-new" || inc.Intelligence == nil {
			continue
		}
		if len(inc.Intelligence.SimilarIncidents) == 0 {
			t.Fatalf("expected similar incidents for %s", inc.ID)
		}
		found = true
	}
	if !found {
		t.Fatalf("did not find inc-sim-new")
	}
}
