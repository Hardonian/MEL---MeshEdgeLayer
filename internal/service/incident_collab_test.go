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

func TestIncidentIntelligence_ActionOutcomeMemory_ClassifiesMixedAndImprovement(t *testing.T) {
	a := newSoDTestApp(t)
	base := time.Now().UTC().Add(-6 * time.Hour)
	makeIncident := func(id string, occurred time.Time, state string) {
		t.Helper()
		err := a.DB.UpsertIncident(models.Incident{
			ID:           id,
			Category:     "transport",
			Severity:     "warning",
			Title:        "MQTT timeout",
			Summary:      "timeout burst",
			ResourceType: "transport",
			ResourceID:   "mqtt-sod",
			State:        state,
			OccurredAt:   occurred.Format(time.RFC3339),
			Metadata:     map[string]any{"reason": "timeout_stall"},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	makeIncident("inc-hist-1", base, "resolved")
	makeIncident("inc-hist-2", base.Add(2*time.Hour), "open")
	makeIncident("inc-current", base.Add(5*time.Hour), "open")

	// Seed signatures first (build intelligence mutates signature tables).
	if _, ok, err := a.IncidentByID("inc-hist-1"); err != nil || !ok {
		t.Fatalf("seed intelligence 1: ok=%v err=%v", ok, err)
	}
	if _, ok, err := a.IncidentByID("inc-hist-2"); err != nil || !ok {
		t.Fatalf("seed intelligence 2: ok=%v err=%v", ok, err)
	}

	// Same action type across similar incidents with opposite signal trends -> mixed.
	if err := a.DB.UpsertControlAction(db.ControlActionRecord{
		ID:              "act-hist-1",
		ActionType:      "trigger_health_recheck",
		TargetTransport: "mqtt-sod",
		IncidentID:      "inc-hist-1",
		CreatedAt:       base.Add(30 * time.Minute).Format(time.RFC3339),
		CompletedAt:     base.Add(30 * time.Minute).Format(time.RFC3339),
		Result:          "completed",
		LifecycleState:  "completed",
		Mode:            "operator",
		Reason:          "manual",
		ExecutionSource: "test",
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.DB.UpsertControlAction(db.ControlActionRecord{
		ID:              "act-hist-2",
		ActionType:      "trigger_health_recheck",
		TargetTransport: "mqtt-sod",
		IncidentID:      "inc-hist-2",
		CreatedAt:       base.Add(150 * time.Minute).Format(time.RFC3339),
		CompletedAt:     base.Add(150 * time.Minute).Format(time.RFC3339),
		Result:          "failed",
		LifecycleState:  "failed",
		Mode:            "operator",
		Reason:          "manual",
		ExecutionSource: "test",
	}); err != nil {
		t.Fatal(err)
	}

	got, ok, err := a.IncidentByID("inc-current")
	if err != nil || !ok || got.Intelligence == nil {
		t.Fatalf("incident intelligence: ok=%v err=%v", ok, err)
	}
	if len(got.Intelligence.ActionOutcomeMemory) == 0 {
		t.Fatalf("expected action outcome memory")
	}
	if len(got.Intelligence.ActionOutcomeSnapshots) == 0 {
		t.Fatalf("expected persisted action outcome snapshots")
	}
	if got.Intelligence.ActionOutcomeTrace == nil {
		t.Fatalf("expected action outcome trace metadata")
	}
	if got.Intelligence.ActionOutcomeTrace.SnapshotRetrievalStatus != "available" {
		t.Fatalf("snapshot retrieval status=%q", got.Intelligence.ActionOutcomeTrace.SnapshotRetrievalStatus)
	}
	if got.Intelligence.ActionOutcomeTrace.SnapshotRetrievalReason != "historical_snapshots_loaded" {
		t.Fatalf("snapshot retrieval reason=%q", got.Intelligence.ActionOutcomeTrace.SnapshotRetrievalReason)
	}
	if got.Intelligence.ActionOutcomeTrace.Completeness != "complete" {
		t.Fatalf("trace completeness=%q", got.Intelligence.ActionOutcomeTrace.Completeness)
	}
	found := false
	for _, m := range got.Intelligence.ActionOutcomeMemory {
		if m.ActionType != "trigger_health_recheck" {
			continue
		}
		found = true
		if m.OutcomeFraming != "mixed_historical_evidence" {
			t.Fatalf("outcome framing=%q", m.OutcomeFraming)
		}
		if m.ImprovementObservedCount == 0 || m.DeteriorationObservedCount == 0 {
			t.Fatalf("expected mixed counts, got improvement=%d deterioration=%d", m.ImprovementObservedCount, m.DeteriorationObservedCount)
		}
		if len(m.SnapshotRefs) < 2 {
			t.Fatalf("expected snapshot refs for traceability, got %v", m.SnapshotRefs)
		}
		if m.SnapshotTraceStatus != "complete" {
			t.Fatalf("snapshot trace status=%q", m.SnapshotTraceStatus)
		}
		if m.SnapshotCoveragePosture != "matched" {
			t.Fatalf("snapshot coverage posture=%q", m.SnapshotCoveragePosture)
		}
		if m.SnapshotCoveragePercent < 100 {
			t.Fatalf("snapshot coverage percent=%v", m.SnapshotCoveragePercent)
		}
	}
	if !found {
		t.Fatalf("expected trigger_health_recheck memory")
	}
}

func TestIncidentIntelligence_ActionOutcomeMemory_DegradesOnSparseHistory(t *testing.T) {
	a := newSoDTestApp(t)
	now := time.Now().UTC().Add(-2 * time.Hour)
	for _, id := range []string{"inc-single-hist", "inc-single-current"} {
		if err := a.DB.UpsertIncident(models.Incident{
			ID:           id,
			Category:     "transport",
			Severity:     "warning",
			Title:        "Sparse",
			Summary:      "sparse",
			ResourceType: "transport",
			ResourceID:   "mqtt-sod",
			State:        "open",
			OccurredAt:   now.Format(time.RFC3339),
			Metadata:     map[string]any{"reason": "timeout_stall"},
		}); err != nil {
			t.Fatal(err)
		}
		now = now.Add(time.Hour)
	}
	if _, ok, err := a.IncidentByID("inc-single-hist"); err != nil || !ok {
		t.Fatalf("seed intelligence: ok=%v err=%v", ok, err)
	}
	if err := a.DB.UpsertControlAction(db.ControlActionRecord{
		ID:              "act-single",
		ActionType:      "reset_transport_session",
		TargetTransport: "mqtt-sod",
		IncidentID:      "inc-single-hist",
		CreatedAt:       time.Now().UTC().Add(-90 * time.Minute).Format(time.RFC3339),
		CompletedAt:     time.Now().UTC().Add(-90 * time.Minute).Format(time.RFC3339),
		Result:          "completed",
		LifecycleState:  "completed",
		Mode:            "operator",
		Reason:          "manual",
		ExecutionSource: "test",
	}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := a.IncidentByID("inc-single-current")
	if err != nil || !ok || got.Intelligence == nil {
		t.Fatalf("incident intelligence: ok=%v err=%v", ok, err)
	}
	if len(got.Intelligence.ActionOutcomeMemory) == 0 {
		t.Fatalf("expected action outcome memory")
	}
	if got.Intelligence.ActionOutcomeMemory[0].OutcomeFraming != "insufficient_evidence" {
		t.Fatalf("outcome framing=%q", got.Intelligence.ActionOutcomeMemory[0].OutcomeFraming)
	}
	if got.Intelligence.ActionOutcomeTrace == nil {
		t.Fatalf("expected action outcome trace")
	}
	if got.Intelligence.ActionOutcomeTrace.SnapshotRetrievalReason != "historical_snapshots_loaded" {
		t.Fatalf("snapshot retrieval reason=%q", got.Intelligence.ActionOutcomeTrace.SnapshotRetrievalReason)
	}
	if got.Intelligence.ActionOutcomeMemory[0].SnapshotTraceStatus != "complete" {
		t.Fatalf("snapshot trace status=%q", got.Intelligence.ActionOutcomeMemory[0].SnapshotTraceStatus)
	}
}
