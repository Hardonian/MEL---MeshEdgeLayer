package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/models"
)

const replaySummaryTimelineCap = 80
const replaySummaryOutcomeCap = 40

func (a *App) buildIncidentReplaySummary(inc models.Incident) *models.IncidentReplaySummary {
	if a == nil || a.DB == nil || strings.TrimSpace(inc.ID) == "" {
		return &models.IncidentReplaySummary{
			SchemaVersion:   "incident_replay_summary/v1",
			Semantic:        "unavailable",
			Degraded:        true,
			DegradedReasons: []string{"replay_service_unavailable"},
			Summary:         "Replay posture unavailable in this API context.",
		}
	}
	from, to := incidentEvidenceWindow(inc)
	out := &models.IncidentReplaySummary{
		SchemaVersion: "incident_replay_summary/v1",
		WindowFrom:    from,
		WindowTo:      to,
	}

	timeline, err := a.DB.TimelineEventsForIncidentResource(inc.ID, from, to, replaySummaryTimelineCap)
	if err != nil {
		out.Semantic = "unavailable"
		out.Degraded = true
		out.DegradedReasons = append(out.DegradedReasons, "timeline_query_failed")
		out.Summary = "Replay unavailable: could not query persisted timeline rows for this incident window."
		out.Uncertainty = "Replay posture missing due to query failure; verify timeline/replay endpoint before deriving queue urgency."
		return out
	}
	out.WindowTruncated = len(timeline) >= replaySummaryTimelineCap
	if out.WindowTruncated {
		out.Degraded = true
		out.DegradedReasons = append(out.DegradedReasons, "timeline_window_truncated")
	}
	outcomes, _ := a.DB.RecommendationOutcomesForIncident(inc.ID, replaySummaryOutcomeCap)
	segments := mergeReplaySegmentsChronologically(
		replaySegmentsFromTimeline(timeline, inc),
		replaySegmentsFromRecommendationOutcomes(outcomes, inc),
	)
	if len(segments) == 0 {
		out.Semantic = "no_history"
		out.ActivityPosture = "no_persisted_activity"
		out.Degraded = true
		out.DegradedReasons = append(out.DegradedReasons, "no_replay_rows")
		out.Summary = "No persisted replay rows in the bounded incident window."
		out.Uncertainty = "Absence of rows is not proof of calm runtime; evidence may be outside window or retention bounds."
		out.RecommendationRef = "/api/v1/incidents/" + inc.ID + "/replay"
		return out
	}
	if ts := latestReplaySegmentTime(segments); !ts.IsZero() {
		out.LastEventAt = ts.Format(time.RFC3339)
	}
	delta := replayDeltaLast10Minutes(segments, to)
	if len(delta) == 0 {
		out.Semantic = "partial"
		out.Degraded = true
		out.DegradedReasons = append(out.DegradedReasons, "delta_unavailable")
		out.Summary = "Replay rows found, but deterministic now-vs-10m delta is unavailable for this response."
		out.Uncertainty = "Timestamp quality in persisted rows may be incomplete; open full replay for manual review."
		out.RecommendationRef = "/api/v1/incidents/" + inc.ID + "/replay"
		return out
	}
	out.ActivityPosture = asString(delta["activity_posture"])
	out.AnchorTime = asString(delta["anchor_time"])
	out.RecentCount = asInt(delta["recent_segment_count"])
	out.PriorCount = asInt(delta["prior_segment_count"])
	out.DeltaTotal = asInt(delta["delta_total"])
	out.SparseEvidence = asBool(delta["sparse_evidence"])
	switch out.ActivityPosture {
	case "activity_increasing":
		out.Semantic = "active_changing"
	case "activity_lower_recently":
		out.Semantic = "cooling_off"
	case "quiet_recently":
		out.Semantic = "quiet_recently"
	default:
		out.Semantic = "partial"
	}
	if out.SparseEvidence {
		out.Semantic = "sparse"
		out.Degraded = true
		out.DegradedReasons = append(out.DegradedReasons, "sparse_replay_rows")
	}
	if u := strings.TrimSpace(asString(delta["uncertainty"])); u != "" {
		out.Uncertainty = u
	}
	if out.Summary == "" {
		out.Summary = fmt.Sprintf("Replay posture %s: %d recent vs %d prior rows (Δ %d) in deterministic 10-minute delta (bounded incident window).",
			strings.ReplaceAll(out.Semantic, "_", " "), out.RecentCount, out.PriorCount, out.DeltaTotal)
	}
	if out.WindowTruncated {
		out.Summary += " Timeline rows reached cap; additional history may exist outside this response."
	}
	out.RecommendationRef = "/api/v1/incidents/" + inc.ID + "/replay"
	return out
}

func latestReplaySegmentTime(segments []replaySegment) time.Time {
	var latest time.Time
	for _, seg := range segments {
		t := parseReplayTime(seg.EventTime)
		if t.After(latest) {
			latest = t
		}
	}
	return latest
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func asInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func asBool(v any) bool {
	b, _ := v.(bool)
	return b
}
