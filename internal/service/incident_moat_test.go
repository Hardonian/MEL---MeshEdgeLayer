package service

import (
	"strings"
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
		ID:              "act-replay-1",
		ActionType:      "restart_transport",
		TargetTransport: "mqtt-sod",
		LifecycleState:  "completed",
		IncidentID:      id,
		Reason:          "test",
		Confidence:      0.5,
		CreatedAt:       ts,
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
	view, err := a.IncidentReplayView(id, true)
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
	delta, ok := meta["delta_last_10m"].(map[string]any)
	if !ok {
		t.Fatalf("missing delta_last_10m: %#v", meta["delta_last_10m"])
	}
	if got, _ := delta["window_minutes"].(int); got != 10 {
		t.Fatalf("window_minutes=%v", delta["window_minutes"])
	}
	if _, ok := delta["delta_total"]; !ok {
		t.Fatalf("delta_total missing: %#v", delta)
	}
}

func TestBuildEscalationBundle_IncludesReplayPosture(t *testing.T) {
	a := newSoDTestApp(t)
	now := time.Now().UTC()
	inc := models.Incident{
		ID:           "inc-escalation-replay",
		Category:     "transport",
		Severity:     "warning",
		Title:        "Escalation replay",
		Summary:      "check portability",
		ResourceType: "transport",
		ResourceID:   "mqtt-sod",
		State:        "open",
		OccurredAt:   now.Add(-30 * time.Minute).Format(time.RFC3339),
		UpdatedAt:    now.Format(time.RFC3339),
	}
	if err := a.DB.UpsertIncident(inc); err != nil {
		t.Fatal(err)
	}
	if err := a.DB.InsertTimelineEvent(db.TimelineEvent{
		EventID:    "tl-escalation-replay",
		EventTime:  now.Add(-3 * time.Minute).Format(time.RFC3339),
		EventType:  "incident_workflow",
		Summary:    "incident workflow updated",
		Severity:   "info",
		ResourceID: inc.ID,
		Details:    map[string]any{"incident_id": inc.ID},
	}); err != nil {
		t.Fatal(err)
	}

	b, err := a.BuildEscalationBundle(inc.ID, "op-escalate")
	if err != nil {
		t.Fatal(err)
	}
	rp, ok := b["replay_posture"].(map[string]any)
	if !ok {
		t.Fatalf("missing replay_posture: %#v", b["replay_posture"])
	}
	if rp["status"] != "available" {
		t.Fatalf("status=%v", rp["status"])
	}
	if strings.TrimSpace(asString(rp["semantic"])) == "" {
		t.Fatalf("missing replay semantic: %#v", rp)
	}
	att, ok := b["replay_attention"].(map[string]any)
	if !ok {
		t.Fatalf("missing replay_attention: %#v", b["replay_attention"])
	}
	if strings.TrimSpace(asString(att["reason_code"])) == "" {
		t.Fatalf("missing reason_code: %#v", att)
	}
	rs, ok := b["replay_support"].(models.IncidentEscalationReplaySupport)
	if !ok {
		t.Fatalf("missing replay_support: %#v", b["replay_support"])
	}
	if rs.Status != "available" {
		t.Fatalf("replay_support status=%v", rs.Status)
	}
	if strings.TrimSpace(rs.AttentionReasonCode) == "" {
		t.Fatalf("missing replay_support attention_reason_code: %#v", rs)
	}
}

func TestEscalationReplaySupport_TimelineUnavailableMarksDegraded(t *testing.T) {
	inc := models.Incident{
		ID: "inc-rp-unavailable",
		ReplaySummary: &models.IncidentReplaySummary{
			SchemaVersion: "incident_replay_summary/v1",
			Semantic:      "quiet_recently",
			Summary:       "Replay posture quiet recently: 0 recent vs 1 prior rows (Δ -1) in deterministic 10-minute delta (bounded incident window).",
		},
	}
	support := escalationReplaySupport(inc, map[string]any{
		"section_statuses": []any{
			map[string]any{
				"section": "timeline",
				"status":  "unavailable",
				"reason":  "no_timeline_events",
			},
		},
	})
	if !support.Degraded {
		t.Fatalf("expected degraded replay support when timeline section unavailable: %#v", support)
	}
	if !support.WarrantsAttention {
		t.Fatalf("expected attention when timeline section unavailable: %#v", support)
	}
	if support.TimelineSection == nil || support.TimelineSection.Status != "unavailable" {
		t.Fatalf("expected timeline section reference: %#v", support.TimelineSection)
	}
	found := false
	for _, reason := range support.DegradedReasons {
		if reason == "timeline_section_unavailable" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected timeline_section_unavailable degraded reason: %#v", support.DegradedReasons)
	}
}
