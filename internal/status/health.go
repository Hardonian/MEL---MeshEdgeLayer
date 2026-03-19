package status

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/transport"
)

type TransportHealth struct {
	TransportName   string                 `json:"transport_name"`
	TransportType   string                 `json:"transport_type"`
	Score           int                    `json:"score"`
	State           string                 `json:"state"`
	LastEvaluatedAt string                 `json:"last_evaluated_at"`
	PrimaryReason   string                 `json:"primary_reason,omitempty"`
	Signals         TransportHealthSignals `json:"signals"`
}

type TransportHealthSignals struct {
	RecentFailures        uint64  `json:"recent_failures"`
	DeadLetterCount       uint64  `json:"dead_letter_count"`
	RetryCount            uint64  `json:"retry_count"`
	LastHeartbeatDeltaSec int64   `json:"last_heartbeat_delta_seconds"`
	AnomalyRate           float64 `json:"anomaly_rate"`
	ObservationDrops      uint64  `json:"observation_drops"`
	ActiveEpisode         bool    `json:"active_episode"`
}

type TransportAnomalySummary struct {
	TransportName    string            `json:"transport_name"`
	Window           string            `json:"window"`
	CountsByReason   map[string]uint64 `json:"counts_by_reason"`
	DeadLetters      uint64            `json:"dead_letters"`
	RetryEvents      uint64            `json:"retry_events"`
	AnomalyRate      float64           `json:"anomaly_rate"`
	ObservationDrops uint64            `json:"observation_drops"`
	ActiveEpisodeIDs []string          `json:"active_episode_ids,omitempty"`
	DropCauses       map[string]uint64 `json:"drop_causes,omitempty"`
}

type FailureCluster struct {
	TransportName            string `json:"transport_name"`
	TransportType            string `json:"transport_type"`
	Reason                   string `json:"reason"`
	Count                    uint64 `json:"count"`
	FirstSeen                string `json:"first_seen"`
	LastSeen                 string `json:"last_seen"`
	Severity                 string `json:"severity"`
	EpisodeID                string `json:"episode_id,omitempty"`
	IncludesDeadLetter       bool   `json:"includes_dead_letter"`
	IncludesObservationDrops bool   `json:"includes_observation_drops"`
	ClusterKey               string `json:"cluster_key"`
}

type TransportAlert struct {
	ID               string `json:"id"`
	TransportName    string `json:"transport_name"`
	TransportType    string `json:"transport_type"`
	Severity         string `json:"severity"`
	Reason           string `json:"reason"`
	Summary          string `json:"summary"`
	FirstTriggeredAt string `json:"first_triggered_at"`
	LastUpdatedAt    string `json:"last_updated_at"`
	Active           bool   `json:"active"`
	EpisodeID        string `json:"episode_id,omitempty"`
	ClusterKey       string `json:"cluster_key"`
}

type TransportIntelligence struct {
	HealthByTransport    map[string]TransportHealth
	AnomaliesByTransport map[string][]TransportAnomalySummary
	ClustersByTransport  map[string][]FailureCluster
	AlertsByTransport    map[string][]TransportAlert
}

type transportEvidenceEvent struct {
	TransportName    string
	TransportType    string
	Reason           string
	EpisodeID        string
	CreatedAt        time.Time
	DeadLetter       bool
	ObservationDrops uint64
	DropCause        string
}

var scoringPenaltyByReason = map[string]int{
	"retry_threshold_exceeded": 30,
	"timeout_failure":          10,
	"timeout_stall":            5,
	"malformed_frame":          5,
	"malformed_publish":        5,
	"decode_failure":           5,
	"topic_mismatch":           3,
	"handler_rejection":        6,
	"rejected_send":            6,
	"rejected_publish":         6,
}

var healthStateOrder = map[string]int{"healthy": 0, "degraded": 1, "unstable": 2, "failed": 3}

func EvaluateTransportIntelligence(cfg config.Config, database *db.DB, runtime []transport.Health, now time.Time) (TransportIntelligence, error) {
	result := TransportIntelligence{
		HealthByTransport:    map[string]TransportHealth{},
		AnomaliesByTransport: map[string][]TransportAnomalySummary{},
		ClustersByTransport:  map[string][]FailureCluster{},
		AlertsByTransport:    map[string][]TransportAlert{},
	}
	runtimeMap := map[string]transport.Health{}
	for _, item := range runtime {
		runtimeMap[item.Name] = item
	}
	persistedRuntime := map[string]db.TransportRuntime{}
	if database != nil {
		rows, err := database.TransportRuntimeStatuses()
		if err != nil {
			return result, err
		}
		for _, row := range rows {
			persistedRuntime[row.Name] = row
		}
	}
	eventsByTransport, err := queryEvidenceEvents(database, now.Add(-15*time.Minute))
	if err != nil {
		return result, err
	}
	for _, tc := range cfg.Transports {
		h := runtimeMap[tc.Name]
		if h.Name == "" {
			if persisted, ok := persistedRuntime[tc.Name]; ok {
				h = transport.Health{
					Name:                  persisted.Name,
					Type:                  persisted.Type,
					Source:                persisted.Source,
					State:                 persisted.State,
					Detail:                persisted.Detail,
					LastAttemptAt:         persisted.LastAttemptAt,
					LastConnectedAt:       persisted.LastConnectedAt,
					LastSuccessAt:         persisted.LastSuccessAt,
					LastIngestAt:          persisted.LastMessageAt,
					LastHeartbeatAt:       persisted.LastHeartbeatAt,
					LastFailureAt:         persisted.LastFailureAt,
					LastObservationDropAt: persisted.LastObservationDrop,
					LastError:             persisted.LastError,
					EpisodeID:             persisted.EpisodeID,
					TotalMessages:         persisted.TotalMessages,
					PacketsDropped:        persisted.PacketsDropped,
					ReconnectAttempts:     persisted.Reconnects,
					ConsecutiveTimeouts:   persisted.Timeouts,
					FailureCount:          persisted.FailureCount,
					ObservationDrops:      persisted.ObservationDrops,
				}
			}
		}
		events := eventsByTransport[tc.Name]
		anomalies := summarizeTransportAnomalies(tc.Name, events, h, now)
		clusters := clusterTransportFailures(tc.Name, tc.Type, events, h, now)
		health := scoreTransportHealth(tc.Name, tc.Type, h, anomalies, now)
		result.HealthByTransport[tc.Name] = health
		result.AnomaliesByTransport[tc.Name] = anomalies
		result.ClustersByTransport[tc.Name] = clusters
	}
	return result, nil
}

func queryEvidenceEvents(database *db.DB, since time.Time) (map[string][]transportEvidenceEvent, error) {
	out := map[string][]transportEvidenceEvent{}
	if database == nil {
		return out, nil
	}
	cutoff := since.UTC().Format(time.RFC3339)
	rows, err := database.QueryRows(fmt.Sprintf(`SELECT transport_name, transport_type, reason, episode_id, dead_letter, created_at, observation_drops, drop_cause FROM (
	SELECT COALESCE(json_extract(details_json,'$.transport'), '') AS transport_name,
	       COALESCE(json_extract(details_json,'$.type'), '') AS transport_type,
	       message AS reason,
	       COALESCE(json_extract(details_json,'$.episode_id'), '') AS episode_id,
	       COALESCE(json_extract(details_json,'$.dead_letter'), 0) AS dead_letter,
	       created_at,
	       COALESCE(json_extract(details_json,'$.drop_count'), json_extract(details_json,'$.details.drop_count'), 0) AS observation_drops,
	       COALESCE(json_extract(details_json,'$.drop_cause'), json_extract(details_json,'$.details.drop_cause'), '') AS drop_cause
	FROM audit_logs
	WHERE category='transport' AND created_at >= '%s'
	UNION ALL
	SELECT transport_name,
	       transport_type,
	       reason,
	       COALESCE(json_extract(details_json,'$.episode_id'), '') AS episode_id,
	       1 AS dead_letter,
	       created_at,
	       0 AS observation_drops,
	       COALESCE(json_extract(details_json,'$.drop_cause'), '') AS drop_cause
	FROM dead_letters
	WHERE created_at >= '%s'
) evidence WHERE transport_name != '';`, sqlEscape(cutoff), sqlEscape(cutoff)))
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		reason := asString(row["reason"])
		if !knownTransportReason(reason) {
			continue
		}
		createdAt, ok := parseTimestamp(asString(row["created_at"]))
		if !ok {
			continue
		}
		evt := transportEvidenceEvent{
			TransportName:    asString(row["transport_name"]),
			TransportType:    asString(row["transport_type"]),
			Reason:           reason,
			EpisodeID:        asString(row["episode_id"]),
			CreatedAt:        createdAt,
			DeadLetter:       asInt(row["dead_letter"]) == 1,
			ObservationDrops: uint64(asInt(row["observation_drops"])),
			DropCause:        asString(row["drop_cause"]),
		}
		out[evt.TransportName] = append(out[evt.TransportName], evt)
	}
	for name := range out {
		sort.Slice(out[name], func(i, j int) bool { return out[name][i].CreatedAt.Before(out[name][j].CreatedAt) })
	}
	return out, nil
}

func summarizeTransportAnomalies(name string, events []transportEvidenceEvent, runtime transport.Health, now time.Time) []TransportAnomalySummary {
	windows := []struct {
		label    string
		duration time.Duration
	}{
		{label: "1m", duration: time.Minute},
		{label: "5m", duration: 5 * time.Minute},
		{label: "15m", duration: 15 * time.Minute},
	}
	out := make([]TransportAnomalySummary, 0, len(windows))
	for _, window := range windows {
		summary := TransportAnomalySummary{
			TransportName:    name,
			Window:           window.label,
			CountsByReason:   map[string]uint64{},
			ActiveEpisodeIDs: []string{},
			DropCauses:       map[string]uint64{},
		}
		episodeSet := map[string]struct{}{}
		for _, evt := range events {
			if now.Sub(evt.CreatedAt) > window.duration {
				continue
			}
			summary.CountsByReason[evt.Reason]++
			if evt.DeadLetter {
				summary.DeadLetters++
			}
			if evt.Reason == transport.ReasonRetryThresholdExceeded {
				summary.RetryEvents++
			}
			if evt.ObservationDrops > 0 {
				summary.ObservationDrops += evt.ObservationDrops
			}
			if evt.DropCause != "" {
				summary.DropCauses[evt.DropCause] += maxUint64(evt.ObservationDrops, 1)
			}
			if evt.EpisodeID != "" {
				episodeSet[evt.EpisodeID] = struct{}{}
			}
		}
		if runtime.ObservationDrops > 0 && window.label == "15m" {
			summary.ObservationDrops = maxUint64(summary.ObservationDrops, runtime.ObservationDrops)
		}
		for episodeID := range episodeSet {
			summary.ActiveEpisodeIDs = append(summary.ActiveEpisodeIDs, episodeID)
		}
		sort.Strings(summary.ActiveEpisodeIDs)
		total := uint64(0)
		for _, count := range summary.CountsByReason {
			total += count
		}
		if window.duration > 0 {
			summary.AnomalyRate = float64(total) / window.duration.Minutes()
		}
		out = append(out, summary)
	}
	return out
}

func clusterTransportFailures(name, typ string, events []transportEvidenceEvent, runtime transport.Health, now time.Time) []FailureCluster {
	cutoff := now.Add(-15 * time.Minute)
	grouped := map[string]*FailureCluster{}
	for _, evt := range events {
		if evt.CreatedAt.Before(cutoff) {
			continue
		}
		if evt.Reason == transport.ReasonUnsupportedControlPath {
			continue
		}
		clusterReason := evt.Reason
		if evt.Reason == transport.ReasonObservationDropped && evt.DropCause != "" {
			clusterReason = evt.DropCause
		}
		episodeKey := evt.EpisodeID
		if clusterReason == transport.ReasonTimeoutFailure || clusterReason == transport.ReasonRetryThresholdExceeded || clusterReason == transport.ReasonTimeoutStall {
			episodeKey = firstNonEmpty(evt.EpisodeID, runtime.EpisodeID)
		}
		key := strings.Join([]string{name, clusterReason, episodeKey}, "|")
		cluster := grouped[key]
		if cluster == nil {
			cluster = &FailureCluster{TransportName: name, TransportType: typ, Reason: clusterReason, FirstSeen: evt.CreatedAt.UTC().Format(time.RFC3339), LastSeen: evt.CreatedAt.UTC().Format(time.RFC3339), EpisodeID: episodeKey, ClusterKey: key}
			grouped[key] = cluster
		}
		cluster.Count++
		if evt.CreatedAt.UTC().Format(time.RFC3339) < cluster.FirstSeen {
			cluster.FirstSeen = evt.CreatedAt.UTC().Format(time.RFC3339)
		}
		if evt.CreatedAt.UTC().Format(time.RFC3339) > cluster.LastSeen {
			cluster.LastSeen = evt.CreatedAt.UTC().Format(time.RFC3339)
		}
		cluster.IncludesDeadLetter = cluster.IncludesDeadLetter || evt.DeadLetter
		cluster.IncludesObservationDrops = cluster.IncludesObservationDrops || evt.ObservationDrops > 0 || evt.Reason == transport.ReasonObservationDropped
	}
	out := make([]FailureCluster, 0, len(grouped))
	for _, cluster := range grouped {
		if cluster.Count < 2 && !cluster.IncludesDeadLetter && !cluster.IncludesObservationDrops && cluster.EpisodeID == "" {
			continue
		}
		cluster.Severity = deriveClusterSeverity(*cluster, runtime, now)
		out = append(out, *cluster)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Severity == out[j].Severity {
			return out[i].LastSeen > out[j].LastSeen
		}
		return severityRank(out[i].Severity) > severityRank(out[j].Severity)
	})
	return out
}

func deriveClusterSeverity(cluster FailureCluster, runtime transport.Health, now time.Time) string {
	if cluster.IncludesDeadLetter || cluster.Reason == transport.ReasonRetryThresholdExceeded || runtime.State == transport.StateFailed {
		return "critical"
	}
	if cluster.IncludesObservationDrops || cluster.Count >= 4 || (runtime.EpisodeID != "" && runtime.FailureCount >= 2) {
		return "warn"
	}
	return "info"
}

func scoreTransportHealth(name, typ string, runtime transport.Health, anomalies []TransportAnomalySummary, now time.Time) TransportHealth {
	fiveMinute := AnomalyWindow(anomalies, "5m")
	fifteenMinute := AnomalyWindow(anomalies, "15m")
	score := 100
	primaryReason := ""
	for reason, count := range fiveMinute.CountsByReason {
		penalty := scoringPenaltyByReason[reason] * int(count)
		if penalty > 0 {
			score -= penalty
			if primaryReason == "" {
				primaryReason = reason
			}
		}
	}
	if fiveMinute.DeadLetters > 0 {
		score -= int(fiveMinute.DeadLetters) * 25
		if primaryReason == "" {
			primaryReason = "dead_letter_burst"
		}
	}
	if fifteenMinute.DeadLetters > fiveMinute.DeadLetters {
		score -= minInt(15, int(fifteenMinute.DeadLetters-fiveMinute.DeadLetters)*5)
	}
	for reason, count := range fifteenMinute.CountsByReason {
		residual := count
		if recent := fiveMinute.CountsByReason[reason]; residual > recent {
			residual -= recent
		} else {
			residual = 0
		}
		if residual == 0 {
			continue
		}
		score -= minInt(15, residualPenaltyForReason(reason, residual))
	}
	if runtime.EpisodeID != "" && runtime.FailureCount > 0 {
		score -= minInt(25, int(runtime.FailureCount)*5)
		if primaryReason == "" {
			primaryReason = "active_failure_episode"
		}
	}
	if fiveMinute.ObservationDrops > 0 {
		score -= progressiveObservationPenalty(fiveMinute.ObservationDrops)
		if primaryReason == "" {
			primaryReason = "observation_drops"
		}
	}
	if gapPenalty, heartbeatDelta := heartbeatPenalty(runtime, now); gapPenalty > 0 {
		score -= gapPenalty
		if primaryReason == "" {
			primaryReason = fmt.Sprintf("heartbeat_gap_%ds", heartbeatDelta)
		}
	}
	if runtime.State == transport.StateFailed {
		score -= 20
	} else if runtime.State == transport.StateRetrying {
		score -= 12
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return TransportHealth{
		TransportName:   name,
		TransportType:   typ,
		Score:           score,
		State:           healthStateForScore(score),
		LastEvaluatedAt: now.UTC().Format(time.RFC3339),
		PrimaryReason:   primaryReason,
		Signals: TransportHealthSignals{
			RecentFailures:        totalReasonCount(fiveMinute.CountsByReason),
			DeadLetterCount:       fiveMinute.DeadLetters,
			RetryCount:            fiveMinute.RetryEvents + runtime.ReconnectAttempts,
			LastHeartbeatDeltaSec: heartbeatDeltaSeconds(runtime, now),
			AnomalyRate:           fiveMinute.AnomalyRate,
			ObservationDrops:      fiveMinute.ObservationDrops,
			ActiveEpisode:         runtime.EpisodeID != "" && runtime.FailureCount > 0,
		},
	}
}

func AnomalyWindow(windows []TransportAnomalySummary, label string) TransportAnomalySummary {
	for _, window := range windows {
		if window.Window == label {
			return window
		}
	}
	return TransportAnomalySummary{CountsByReason: map[string]uint64{}, DropCauses: map[string]uint64{}}
}

func heartbeatPenalty(runtime transport.Health, now time.Time) (int, int64) {
	delta := heartbeatDeltaSeconds(runtime, now)
	switch {
	case delta >= 900:
		return 30, delta
	case delta >= 300:
		return 20, delta
	case delta >= 120:
		return 10, delta
	default:
		return 0, delta
	}
}

func heartbeatDeltaSeconds(runtime transport.Health, now time.Time) int64 {
	lastHeartbeat := firstNonEmpty(runtime.LastHeartbeatAt, runtime.LastIngestAt, runtime.LastSuccessAt, runtime.LastConnectedAt)
	if lastHeartbeat == "" {
		if runtime.State == transport.StateConfigured || runtime.State == transport.StateDisabled {
			return 0
		}
		return int64((15 * time.Minute).Seconds())
	}
	parsed, ok := parseTimestamp(lastHeartbeat)
	if !ok {
		return 0
	}
	return int64(now.Sub(parsed).Seconds())
}

func progressiveObservationPenalty(count uint64) int {
	if count == 0 {
		return 0
	}
	penalty := 6
	if count > 2 {
		penalty += minInt(24, int((count-2)*3))
	}
	return penalty
}

func residualPenaltyForReason(reason string, count uint64) int {
	switch reason {
	case transport.ReasonRetryThresholdExceeded:
		return int(count) * 12
	case transport.ReasonTimeoutFailure, transport.ReasonHandlerRejection:
		return int(count) * 8
	default:
		return int(count) * 4
	}
}

func healthStateForScore(score int) string {
	switch {
	case score >= 90:
		return "healthy"
	case score >= 70:
		return "degraded"
	case score >= 40:
		return "unstable"
	default:
		return "failed"
	}
}

func knownTransportReason(reason string) bool {
	switch reason {
	case transport.ReasonMalformedFrame, transport.ReasonDecodeFailure, transport.ReasonRejectedSend, transport.ReasonUnsupportedControlPath,
		transport.ReasonTimeoutStall, transport.ReasonTimeoutFailure, transport.ReasonMalformedPublish, transport.ReasonTopicMismatch,
		transport.ReasonHandlerRejection, transport.ReasonRejectedPublish, transport.ReasonStreamFailure, transport.ReasonSubscribeFailure,
		transport.ReasonRetryThresholdExceeded, transport.ReasonObservationDropped:
		return true
	default:
		return false
	}
}

func parseTimestamp(v string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, strings.TrimSpace(v)); err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}

func totalReasonCount(counts map[string]uint64) uint64 {
	var total uint64
	for _, count := range counts {
		total += count
	}
	return total
}

func severityRank(severity string) int {
	switch severity {
	case "critical":
		return 3
	case "warn":
		return 2
	default:
		return 1
	}
}

func WorseHealthState(previous, current string) bool {
	return healthStateOrder[current] > healthStateOrder[previous]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func sqlEscape(v string) string {
	return strings.ReplaceAll(v, "'", "''")
}
