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

func TestEscalationReplaySupport_CompatibilityProjectionScenarios(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	testCases := []struct {
		name                    string
		incident                models.Incident
		proofpack               map[string]any
		wantStatus              string
		wantReasonCode          string
		wantWarrantsAttention   bool
		wantNeedsReview         bool
		wantDegraded            bool
		wantCannotProveContains string
	}{
		{
			name: "available_healthy",
			incident: models.Incident{
				ID: "inc-rp-healthy",
				ReplaySummary: &models.IncidentReplaySummary{
					SchemaVersion: "incident_replay_summary/v1",
					Semantic:      "quiet_recently",
					Summary:       "quiet bounded replay window",
					RecentCount:   1,
					PriorCount:    2,
					DeltaTotal:    -1,
				},
			},
			proofpack:             map[string]any{"section_statuses": []any{map[string]any{"section": "timeline", "status": "available"}}},
			wantStatus:            "available",
			wantReasonCode:        "replay_bounded_context",
			wantWarrantsAttention: false,
			wantNeedsReview:       false,
			wantDegraded:          true,
		},
		{
			name:                  "summary_missing",
			incident:              models.Incident{ID: "inc-rp-missing"},
			proofpack:             map[string]any{},
			wantStatus:            "unavailable",
			wantReasonCode:        "replay_summary_missing",
			wantWarrantsAttention: true,
			wantNeedsReview:       true,
			wantDegraded:          true,
		},
		{
			name: "timeline_partial_overrides_attention",
			incident: models.Incident{
				ID: "inc-rp-partial",
				ReplaySummary: &models.IncidentReplaySummary{
					SchemaVersion: "incident_replay_summary/v1",
					Semantic:      "quiet_recently",
					Summary:       "bounded replay posture",
				},
			},
			proofpack:             map[string]any{"section_statuses": []any{map[string]any{"section": "timeline", "status": "partial", "reason": "truncated_window"}}},
			wantStatus:            "available",
			wantReasonCode:        "timeline_section_not_complete",
			wantWarrantsAttention: true,
			wantNeedsReview:       true,
			wantDegraded:          true,
		},
		{
			name: "timeline_unavailable",
			incident: models.Incident{
				ID: "inc-rp-unavailable",
				ReplaySummary: &models.IncidentReplaySummary{
					SchemaVersion: "incident_replay_summary/v1",
					Semantic:      "quiet_recently",
					Summary:       "bounded replay posture",
				},
			},
			proofpack:             map[string]any{"section_statuses": []any{map[string]any{"section": "timeline", "status": "unavailable", "reason": "no_timeline_events"}}},
			wantStatus:            "available",
			wantReasonCode:        "timeline_section_not_complete",
			wantWarrantsAttention: true,
			wantNeedsReview:       true,
			wantDegraded:          true,
		},
		{
			name: "window_truncated",
			incident: models.Incident{
				ID: "inc-rp-truncated",
				ReplaySummary: &models.IncidentReplaySummary{
					SchemaVersion:   "incident_replay_summary/v1",
					Semantic:        "active_changing",
					Summary:         "window truncated while activity changed",
					WindowTruncated: true,
				},
			},
			proofpack:               map[string]any{},
			wantStatus:              "available",
			wantReasonCode:          "timeline_window_truncated",
			wantWarrantsAttention:   true,
			wantNeedsReview:         true,
			wantDegraded:            true,
			wantCannotProveContains: "does_not_prove_complete_history_within_response",
		},
		{
			name: "no_history_cannot_prove",
			incident: models.Incident{
				ID: "inc-rp-no-history",
				ReplaySummary: &models.IncidentReplaySummary{
					SchemaVersion: "incident_replay_summary/v1",
					Semantic:      "no_history",
					Summary:       "No persisted replay rows in the bounded incident window.",
					WindowFrom:    now,
					WindowTo:      now,
				},
			},
			proofpack:               map[string]any{},
			wantStatus:              "available",
			wantReasonCode:          "no_persisted_replay_rows_in_window",
			wantWarrantsAttention:   true,
			wantNeedsReview:         true,
			wantDegraded:            true,
			wantCannotProveContains: "does_not_prove_calm_runtime_when_no_rows",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			support := escalationReplaySupport(tc.incident, tc.proofpack)
			posture := replayPostureCompatFromSupport(support)
			attention := replayAttentionCompatFromSupport(support)

			if support.Status != tc.wantStatus {
				t.Fatalf("status=%q", support.Status)
			}
			if support.AttentionReasonCode != tc.wantReasonCode {
				t.Fatalf("attention_reason_code=%q", support.AttentionReasonCode)
			}
			if support.WarrantsAttention != tc.wantWarrantsAttention {
				t.Fatalf("warrants_attention=%v", support.WarrantsAttention)
			}
			if support.NeedsOperatorReview != tc.wantNeedsReview {
				t.Fatalf("needs_operator_review=%v", support.NeedsOperatorReview)
			}
			if support.Degraded != tc.wantDegraded {
				t.Fatalf("degraded=%v reasons=%v", support.Degraded, support.DegradedReasons)
			}
			if posture["status"] != support.Status || posture["status_reason"] != support.StatusReason {
				t.Fatalf("posture status projection mismatch: %#v vs %#v", posture, support)
			}
			if attention["reason_code"] != support.AttentionReasonCode {
				t.Fatalf("attention projection mismatch: %#v vs %#v", attention, support)
			}
			if support.TimelineSection != nil {
				timelineStatus, ok := posture["timeline_section_status"].(map[string]any)
				if !ok {
					t.Fatalf("missing timeline_section_status projection: %#v", posture["timeline_section_status"])
				}
				if timelineStatus["status"] != support.TimelineSection.Status {
					t.Fatalf("timeline status projection mismatch: %#v vs %#v", timelineStatus, support.TimelineSection)
				}
			}
			if tc.wantCannotProveContains != "" {
				found := false
				for _, claim := range support.CannotProve {
					if claim == tc.wantCannotProveContains {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("cannot_prove missing %q: %#v", tc.wantCannotProveContains, support.CannotProve)
				}
			}
		})
	}
}
