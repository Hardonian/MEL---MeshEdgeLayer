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
	Explanation     HealthExplanation      `json:"explanation"`
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

type HealthPenalty struct {
	Reason  string `json:"reason"`
	Penalty int    `json:"penalty"`
	Count   uint64 `json:"count"`
	Window  string `json:"window"`
}

type HealthExplanation struct {
	TransportName       string          `json:"transport_name"`
	Score               int             `json:"score"`
	State               string          `json:"state"`
	TopPenalties        []HealthPenalty `json:"top_penalties"`
	ActiveClusterReason string          `json:"active_cluster_reason,omitempty"`
	ActiveClusterCount  uint64          `json:"active_cluster_count,omitempty"`
	ActiveEpisodeID     string          `json:"active_episode_id,omitempty"`
	FailureCount        uint64          `json:"failure_count"`
	ObservationDrops    uint64          `json:"observation_drops"`
	DeadLetterCount     uint64          `json:"dead_letter_count"`
	RecoveryBlockers    []string        `json:"recovery_blockers,omitempty"`
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
	ID                  string          `json:"id"`
	TransportName       string          `json:"transport_name"`
	TransportType       string          `json:"transport_type"`
	Severity            string          `json:"severity"`
	Reason              string          `json:"reason"`
	Summary             string          `json:"summary"`
	FirstTriggeredAt    string          `json:"first_triggered_at"`
	LastUpdatedAt       string          `json:"last_updated_at"`
	Active              bool            `json:"active"`
	EpisodeID           string          `json:"episode_id,omitempty"`
	ClusterKey          string          `json:"cluster_key"`
	ContributingReasons []string        `json:"contributing_reasons,omitempty"`
	ClusterReference    string          `json:"cluster_reference,omitempty"`
	PenaltySnapshot     []HealthPenalty `json:"penalty_snapshot,omitempty"`
	TriggerCondition    string          `json:"trigger_condition,omitempty"`
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

type scoreContribution struct {
	Reason  string
	Penalty int
	Count   uint64
	Window  string
}

type healthScoreBreakdown struct {
	Score         int
	PrimaryReason string
	Signals       TransportHealthSignals
	Penalties     []scoreContribution
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
	maxWindow := maxWindowDuration(cfg)
	eventsByTransport, err := queryEvidenceEvents(database, now.Add(-maxWindow))
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
		anomalies := summarizeTransportAnomalies(cfg, tc.Name, events, h, now)
		clusters := clusterTransportFailures(tc.Name, tc.Type, events, h, now)
		health := scoreTransportHealth(cfg, tc.Name, tc.Type, h, anomalies, clusters, now)
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

func summarizeTransportAnomalies(cfg config.Config, name string, events []transportEvidenceEvent, runtime transport.Health, now time.Time) []TransportAnomalySummary {
	windows := anomalyWindows(cfg)
	out := make([]TransportAnomalySummary, 0, len(windows))
	for _, window := range windows {
		summary := TransportAnomalySummary{TransportName: name, Window: window.label, CountsByReason: map[string]uint64{}, ActiveEpisodeIDs: []string{}, DropCauses: map[string]uint64{}}
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
		if runtime.ObservationDrops > 0 && window.duration == windows[len(windows)-1].duration {
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

func deriveClusterSeverity(cluster FailureCluster, runtime transport.Health, _ time.Time) string {
	if cluster.IncludesDeadLetter || cluster.Reason == transport.ReasonRetryThresholdExceeded || runtime.State == transport.StateFailed {
		return "critical"
	}
	if cluster.IncludesObservationDrops || cluster.Count >= 4 || (runtime.EpisodeID != "" && runtime.FailureCount >= 2) {
		return "warn"
	}
	return "info"
}

func scoreTransportHealth(cfg config.Config, name, typ string, runtime transport.Health, anomalies []TransportAnomalySummary, clusters []FailureCluster, now time.Time) TransportHealth {
	breakdown := buildHealthScoreBreakdown(cfg, runtime, anomalies, now)
	health := TransportHealth{
		TransportName:   name,
		TransportType:   typ,
		Score:           breakdown.Score,
		State:           healthStateForScore(breakdown.Score),
		LastEvaluatedAt: now.UTC().Format(time.RFC3339),
		PrimaryReason:   breakdown.PrimaryReason,
		Signals:         breakdown.Signals,
	}
	health.Explanation = buildHealthExplanation(name, health, clusters, runtime, anomalies, breakdown)
	return health
}

func buildHealthScoreBreakdown(cfg config.Config, runtime transport.Health, anomalies []TransportAnomalySummary, now time.Time) healthScoreBreakdown {
	recentLabel := anomalyWindows(cfg)[1].label
	residualLabel := anomalyWindows(cfg)[2].label
	fiveMinute := AnomalyWindow(anomalies, recentLabel)
	fifteenMinute := AnomalyWindow(anomalies, residualLabel)
	score := 100
	primaryReason := ""
	contributions := make([]scoreContribution, 0)
	weights := cfg.Intelligence.Scoring.ReasonWeights
	for _, reason := range sortedReasonKeys(fiveMinute.CountsByReason) {
		count := fiveMinute.CountsByReason[reason]
		penalty := weights[reason] * int(count)
		if penalty <= 0 {
			continue
		}
		score -= penalty
		contributions = append(contributions, scoreContribution{Reason: reason, Penalty: penalty, Count: count, Window: recentLabel})
		if primaryReason == "" {
			primaryReason = reason
		}
	}
	if fiveMinute.DeadLetters > 0 {
		penalty := int(fiveMinute.DeadLetters) * cfg.Intelligence.Scoring.DeadLetterPenalty
		score -= penalty
		contributions = append(contributions, scoreContribution{Reason: "dead_letter_burst", Penalty: penalty, Count: fiveMinute.DeadLetters, Window: recentLabel})
		if primaryReason == "" {
			primaryReason = "dead_letter_burst"
		}
	}
	if fifteenMinute.DeadLetters > fiveMinute.DeadLetters {
		residual := fifteenMinute.DeadLetters - fiveMinute.DeadLetters
		penalty := minInt(cfg.Intelligence.Scoring.ResidualPenaltyCap, int(residual)*cfg.Intelligence.Scoring.HistoricalDeadLetterWeight)
		score -= penalty
		contributions = append(contributions, scoreContribution{Reason: "historical_dead_letters", Penalty: penalty, Count: residual, Window: residualLabel})
	}
	for _, reason := range sortedReasonKeys(fifteenMinute.CountsByReason) {
		residual := fifteenMinute.CountsByReason[reason]
		if recent := fiveMinute.CountsByReason[reason]; residual > recent {
			residual -= recent
		} else {
			residual = 0
		}
		if residual == 0 {
			continue
		}
		penalty := minInt(cfg.Intelligence.Scoring.ResidualPenaltyCap, residualPenaltyForReason(cfg, reason, residual))
		score -= penalty
		contributions = append(contributions, scoreContribution{Reason: reason, Penalty: penalty, Count: residual, Window: residualLabel})
	}
	if runtime.EpisodeID != "" && runtime.FailureCount > 0 {
		penalty := minInt(cfg.Intelligence.Scoring.ActiveEpisodePenaltyCap, int(runtime.FailureCount)*cfg.Intelligence.Scoring.ActiveEpisodePenalty)
		score -= penalty
		contributions = append(contributions, scoreContribution{Reason: "active_failure_episode", Penalty: penalty, Count: runtime.FailureCount, Window: "runtime"})
		if primaryReason == "" {
			primaryReason = "active_failure_episode"
		}
	}
	if fiveMinute.ObservationDrops > 0 {
		penalty := progressiveObservationPenalty(cfg, fiveMinute.ObservationDrops)
		score -= penalty
		contributions = append(contributions, scoreContribution{Reason: dominantDropCause(fiveMinute), Penalty: penalty, Count: fiveMinute.ObservationDrops, Window: recentLabel})
		if primaryReason == "" {
			primaryReason = "observation_drops"
		}
	}
	if gapPenalty, heartbeatDelta := heartbeatPenalty(cfg, runtime, now); gapPenalty > 0 {
		score -= gapPenalty
		contributions = append(contributions, scoreContribution{Reason: fmt.Sprintf("heartbeat_gap_%ds", heartbeatDelta), Penalty: gapPenalty, Count: uint64(heartbeatDelta), Window: "runtime"})
		if primaryReason == "" {
			primaryReason = fmt.Sprintf("heartbeat_gap_%ds", heartbeatDelta)
		}
	}
	if runtime.State == transport.StateFailed {
		score -= cfg.Intelligence.Scoring.RuntimeFailedPenalty
		contributions = append(contributions, scoreContribution{Reason: "runtime_failed_state", Penalty: cfg.Intelligence.Scoring.RuntimeFailedPenalty, Count: 1, Window: "runtime"})
	} else if runtime.State == transport.StateRetrying {
		score -= cfg.Intelligence.Scoring.RuntimeRetryingPenalty
		contributions = append(contributions, scoreContribution{Reason: "runtime_retrying_state", Penalty: cfg.Intelligence.Scoring.RuntimeRetryingPenalty, Count: 1, Window: "runtime"})
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return healthScoreBreakdown{
		Score:         score,
		PrimaryReason: primaryReason,
		Signals: TransportHealthSignals{
			RecentFailures:        totalReasonCount(fiveMinute.CountsByReason),
			DeadLetterCount:       fiveMinute.DeadLetters,
			RetryCount:            fiveMinute.RetryEvents + runtime.ReconnectAttempts,
			LastHeartbeatDeltaSec: heartbeatDeltaSeconds(runtime, now),
			AnomalyRate:           fiveMinute.AnomalyRate,
			ObservationDrops:      fiveMinute.ObservationDrops,
			ActiveEpisode:         runtime.EpisodeID != "" && runtime.FailureCount > 0,
		},
		Penalties: contributions,
	}
}

func buildHealthExplanation(name string, health TransportHealth, clusters []FailureCluster, runtime transport.Health, anomalies []TransportAnomalySummary, breakdown healthScoreBreakdown) HealthExplanation {
	explanation := HealthExplanation{
		TransportName:    name,
		Score:            health.Score,
		State:            health.State,
		TopPenalties:     topPenaltySnapshot(breakdown.Penalties, 5),
		ActiveEpisodeID:  runtime.EpisodeID,
		FailureCount:     runtime.FailureCount,
		ObservationDrops: health.Signals.ObservationDrops,
		DeadLetterCount:  health.Signals.DeadLetterCount,
	}
	if len(clusters) > 0 {
		explanation.ActiveClusterReason = clusters[0].Reason
		explanation.ActiveClusterCount = clusters[0].Count
	}
	recent := TransportAnomalySummary{DropCauses: map[string]uint64{}}
	if len(anomalies) > 1 {
		recent = anomalies[1]
	} else if len(anomalies) > 0 {
		recent = anomalies[0]
	}
	explanation.RecoveryBlockers = recoveryBlockers(health, runtime, clusters, breakdown, recent)
	return explanation
}

func topPenaltySnapshot(in []scoreContribution, limit int) []HealthPenalty {
	contributions := append([]scoreContribution(nil), in...)
	sort.Slice(contributions, func(i, j int) bool {
		if contributions[i].Penalty == contributions[j].Penalty {
			if contributions[i].Reason == contributions[j].Reason {
				return contributions[i].Window < contributions[j].Window
			}
			return contributions[i].Reason < contributions[j].Reason
		}
		return contributions[i].Penalty > contributions[j].Penalty
	})
	if len(contributions) > limit {
		contributions = contributions[:limit]
	}
	out := make([]HealthPenalty, 0, len(contributions))
	for _, item := range contributions {
		out = append(out, HealthPenalty{Reason: item.Reason, Penalty: item.Penalty, Count: item.Count, Window: item.Window})
	}
	return out
}

func recoveryBlockers(health TransportHealth, runtime transport.Health, clusters []FailureCluster, breakdown healthScoreBreakdown, recent TransportAnomalySummary) []string {
	blockers := []string{}
	if runtime.State == transport.StateFailed || runtime.State == transport.StateRetrying {
		blockers = append(blockers, fmt.Sprintf("runtime_state:%s", runtime.State))
	}
	if runtime.EpisodeID != "" && runtime.FailureCount > 0 {
		blockers = append(blockers, fmt.Sprintf("active_failure_episode:%s (%d failures)", runtime.EpisodeID, runtime.FailureCount))
	}
	if health.Signals.LastHeartbeatDeltaSec >= 120 {
		blockers = append(blockers, fmt.Sprintf("heartbeat_gap:%ds", health.Signals.LastHeartbeatDeltaSec))
	}
	if health.Signals.DeadLetterCount > 0 {
		blockers = append(blockers, fmt.Sprintf("dead_letters:%d", health.Signals.DeadLetterCount))
	}
	if health.Signals.ObservationDrops > 0 {
		blockers = append(blockers, fmt.Sprintf("observation_drops:%d", health.Signals.ObservationDrops))
	}
	for _, cluster := range clusters {
		if cluster.Severity == "info" {
			continue
		}
		blockers = append(blockers, fmt.Sprintf("cluster:%s x%d", cluster.Reason, cluster.Count))
	}
	if len(recent.DropCauses) > 0 {
		for _, cause := range sortedReasonKeys(recent.DropCauses) {
			blockers = append(blockers, fmt.Sprintf("drop_cause:%s x%d", cause, recent.DropCauses[cause]))
		}
	}
	for _, penalty := range topPenaltySnapshot(breakdown.Penalties, 3) {
		blockers = append(blockers, fmt.Sprintf("penalty:%s=%d", penalty.Reason, penalty.Penalty))
	}
	return dedupeStrings(blockers)
}

func AnomalyWindow(windows []TransportAnomalySummary, label string) TransportAnomalySummary {
	for _, window := range windows {
		if window.Window == label {
			return window
		}
	}
	return TransportAnomalySummary{CountsByReason: map[string]uint64{}, DropCauses: map[string]uint64{}}
}

func heartbeatPenalty(cfg config.Config, runtime transport.Health, now time.Time) (int, int64) {
	delta := heartbeatDeltaSeconds(runtime, now)
	switch {
	case delta >= 900:
		return cfg.Intelligence.Scoring.HeartbeatPenalty900, delta
	case delta >= 300:
		return cfg.Intelligence.Scoring.HeartbeatPenalty300, delta
	case delta >= 120:
		return cfg.Intelligence.Scoring.HeartbeatPenalty120, delta
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

func progressiveObservationPenalty(cfg config.Config, count uint64) int {
	if count == 0 {
		return 0
	}
	penalty := cfg.Intelligence.Scoring.ObservationDropBasePenalty
	if count > 2 {
		penalty += minInt(cfg.Intelligence.Scoring.ObservationDropPenaltyCap-cfg.Intelligence.Scoring.ObservationDropBasePenalty, int((count-2)*uint64(cfg.Intelligence.Scoring.ObservationDropStepPenalty)))
	}
	return minInt(cfg.Intelligence.Scoring.ObservationDropPenaltyCap, penalty)
}

func residualPenaltyForReason(cfg config.Config, reason string, count uint64) int {
	switch reason {
	case transport.ReasonRetryThresholdExceeded:
		return int(count) * 12
	case transport.ReasonTimeoutFailure, transport.ReasonHandlerRejection:
		return int(count) * 8
	default:
		weight := cfg.Intelligence.Scoring.ReasonWeights[reason]
		if weight == 0 {
			weight = 4
		}
		return int(count) * minInt(4, maxInt(1, weight))
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

func anomalyWindows(cfg config.Config) []struct {
	label    string
	duration time.Duration
} {
	windows := make([]struct {
		label    string
		duration time.Duration
	}, 0, len(cfg.Intelligence.Scoring.AnomalyWindowsSeconds))
	for _, seconds := range cfg.Intelligence.Scoring.AnomalyWindowsSeconds {
		d := time.Duration(seconds) * time.Second
		windows = append(windows, struct {
			label    string
			duration time.Duration
		}{label: durationLabel(d), duration: d})
	}
	if len(windows) == 0 {
		windows = []struct {
			label    string
			duration time.Duration
		}{{"1m", time.Minute}, {"5m", 5 * time.Minute}, {"15m", 15 * time.Minute}}
	}
	return windows
}

func maxWindowDuration(cfg config.Config) time.Duration {
	maxWindow := 15 * time.Minute
	for _, window := range anomalyWindows(cfg) {
		if window.duration > maxWindow {
			maxWindow = window.duration
		}
	}
	return maxWindow
}

func durationLabel(d time.Duration) string {
	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	return fmt.Sprintf("%ds", int(d/time.Second))
}

func dominantDropCause(summary TransportAnomalySummary) string {
	if len(summary.DropCauses) == 0 {
		return "unspecified_drop_cause"
	}
	pairs := sortedReasonKeys(summary.DropCauses)
	best := pairs[0]
	bestCount := summary.DropCauses[best]
	for _, cause := range pairs[1:] {
		if summary.DropCauses[cause] > bestCount {
			best = cause
			bestCount = summary.DropCauses[cause]
		}
	}
	return best
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func sortedReasonKeys[V ~uint64](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func dedupeStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func sqlEscape(v string) string {
	return strings.ReplaceAll(v, "'", "''")
}
