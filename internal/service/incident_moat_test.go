package service

import (
	"testing"
	"time"

	"github.com/mel-project/mel/internal/db"
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

func TestIncidentReplayView_MergesOutcomesAndTypedSegments(t *testing.T) {
	a := newSoDTestApp(t)
	id := "inc-replay-merge"
	ts := time.Now().UTC().Format(time.RFC3339)
	if err := a.DB.UpsertIncident(models.Incident{
		ID:           id,
		Category:     "transport",
		Severity:     "warning",
		Title:        "replay test",
		Summary:      "s",
		ResourceType: "transport",
		ResourceID:   "mqtt-sod",
		State:        "open",
		OccurredAt:   ts,
		UpdatedAt:    ts,
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.DB.UpsertControlAction(db.ControlActionRecord{
		ID:             "act-replay-1",
		ActionType:     "restart_transport",
		TargetTransport: "mqtt-sod",
		LifecycleState: "completed",
		IncidentID:     id,
		Reason:         "test",
		Confidence:     0.5,
		CreatedAt:      ts,
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.RecordRecommendationOutcome(id, "op-x", models.IncidentRecommendationOutcomeRequest{
		RecommendationID: "guide-incident-record",
		Outcome:          "accepted",
		Note:             "verified",
	}); err != nil {
		t.Fatal(err)
	}
	view, err := a.IncidentReplayView(id)
	if err != nil {
		t.Fatal(err)
	}
	if view["kind"] != "incident_replay_view/v3" {
		t.Fatalf("unexpected kind: %v", view["kind"])
	}
	meta, ok := view["replay_meta"].(map[string]any)
	if !ok {
		t.Fatalf("missing replay_meta: %#v", view["replay_meta"])
	}
	segs, ok := view["replay_segments"].([]replaySegment)
	if !ok {
		t.Fatalf("replay_segments type %T", view["replay_segments"])
	}
	if len(segs) < 2 {
		t.Fatalf("expected merged segments, got %d", len(segs))
	}
	hasRec := false
	hasCtl := false
	for _, s := range segs {
		if s.EventType == "recommendation_outcome" {
			hasRec = true
		}
		if s.EventType == "control_action" {
			hasCtl = true
		}
		if s.EventClass == "" {
			t.Errorf("empty event_class on segment %+v", s)
		}
	}
	if !hasRec || !hasCtl {
		t.Fatalf("merge missing types: rec=%v ctl=%v segs=%d", hasRec, hasCtl, len(segs))
	}
	if sparse, _ := meta["sparse_timeline"].(bool); sparse {
		t.Fatal("unexpected sparse_timeline")
	}
}
