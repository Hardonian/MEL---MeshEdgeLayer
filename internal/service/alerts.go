package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/selfobs"
	statuspkg "github.com/mel-project/mel/internal/status"
)

const defaultIntelligenceInterval = 15 * time.Second

func (a *App) intelligenceInterval() time.Duration {
	if a == nil || a.DB == nil {
		return 0
	}
	if a.intelligenceEvery < 0 {
		return 0
	}
	if a.intelligenceEvery > 0 {
		return a.intelligenceEvery
	}
	return defaultIntelligenceInterval
}

func (a *App) intelligenceWorker(ctx context.Context) {
	interval := a.intelligenceInterval()
	if interval <= 0 {
		return
	}
	a.evaluateTransportIntelligence(time.Now().UTC())
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case tickAt := <-ticker.C:
			a.evaluateTransportIntelligence(tickAt.UTC())
		}
	}
}

func (a *App) evaluateTransportIntelligence(now time.Time) {
	if a == nil || a.DB == nil {
		return
	}
	intelligence, err := statuspkg.EvaluateTransportIntelligence(a.Cfg, a.DB, a.TransportHealth(), now)
	if err != nil {
		a.Log.Error("transport_intelligence_failed", "failed to evaluate transport health intelligence", map[string]any{"error": err.Error()})
		return
	}
	previousSnapshots, err := a.DB.LatestTransportHealthSnapshots()
	if err != nil {
		a.Log.Error("transport_snapshot_query_failed", "failed to query previous transport health snapshots", map[string]any{"error": err.Error()})
		return
	}
	currentAlerts, err := a.DB.TransportAlerts(false)
	if err != nil {
		a.Log.Error("transport_alert_query_failed", "failed to query persisted transport alerts", map[string]any{"error": err.Error()})
		return
	}
	alertByID := map[string]db.TransportAlertRecord{}
	for _, alert := range currentAlerts {
		alertByID[alert.ID] = alert
	}
	for _, tc := range a.Cfg.Transports {
		health, ok := intelligence.HealthByTransport[tc.Name]
		if !ok {
			continue
		}
		alerts := deriveTransportAlerts(a.Cfg, tc.Name, tc.Type, health, intelligence.AnomaliesByTransport[tc.Name], intelligence.ClustersByTransport[tc.Name], previousSnapshots[tc.Name], alertByID, a.DB, now)
		ids := make([]string, 0, len(alerts))
		for _, alert := range alerts {
			ids = append(ids, alert.ID)
			if err := a.DB.UpsertTransportAlert(toAlertRecord(alert)); err != nil {
				a.Log.Error("transport_alert_upsert_failed", "failed to persist transport alert", map[string]any{"transport": alert.TransportName, "reason": alert.Reason, "error": err.Error()})
			}
		}
		if err := a.DB.ResolveTransportAlertsNotIn(tc.Name, ids, now.UTC().Format(time.RFC3339)); err != nil {
			a.Log.Error("transport_alert_resolve_failed", "failed to resolve inactive transport alerts", map[string]any{"transport": tc.Name, "error": err.Error()})
		}
		recent := statuspkg.AnomalyWindow(intelligence.AnomaliesByTransport[tc.Name], recentAnomalyLabel(a.Cfg))
		if err := a.DB.InsertTransportHealthSnapshot(db.TransportHealthSnapshot{
			TransportName:              tc.Name,
			TransportType:              tc.Type,
			Score:                      health.Score,
			State:                      health.State,
			SnapshotTime:               now.UTC().Format(time.RFC3339),
			ActiveAlertCount:           len(alerts),
			DeadLetterCountWindow:      int(recent.DeadLetters),
			ObservationDropCountWindow: int(recent.ObservationDrops),
		}); err != nil {
			a.Log.Error("transport_snapshot_insert_failed", "failed to persist transport health snapshot", map[string]any{"transport": tc.Name, "error": err.Error()})
		}
		for _, snapshot := range anomalySnapshotsForTransport(tc.Name, tc.Type, recent, now) {
			if err := a.DB.UpsertTransportAnomalySnapshot(snapshot); err != nil {
				a.Log.Error("transport_anomaly_snapshot_upsert_failed", "failed to persist transport anomaly snapshot", map[string]any{"transport": tc.Name, "reason": snapshot.Reason, "error": err.Error()})
			}
		}
	}
	a.evaluateControl(now)
	selfobs.MarkFresh("alert")
}

func deriveTransportAlerts(cfg config.Config, name, typ string, health statuspkg.TransportHealth, anomalies []statuspkg.TransportAnomalySummary, clusters []statuspkg.FailureCluster, previous db.TransportHealthSnapshot, persisted map[string]db.TransportAlertRecord, database *db.DB, now time.Time) []statuspkg.TransportAlert {
	alerts := []statuspkg.TransportAlert{}
	recentLabel := recentAnomalyLabel(cfg)
	fiveMinute := statuspkg.AnomalyWindow(anomalies, recentLabel)
	penalties := topPenaltyRecords(health.Explanation.TopPenalties)
	if shouldKeepHealthAlert(cfg, previous, health, persisted, name, database, now) {
		id := fmt.Sprintf("%s|health_state_transition|%s", name, health.State)
		alerts = append(alerts, newAlert(cfg, persisted[id], statuspkg.TransportAlert{
			ID:                  id,
			TransportName:       name,
			TransportType:       typ,
			Severity:            healthStateSeverity(health.State),
			Reason:              "health_state_transition",
			Summary:             fmt.Sprintf("health is %s with score=%d", health.State, health.Score),
			EpisodeID:           health.Explanation.ActiveEpisodeID,
			ClusterKey:          health.Explanation.ActiveClusterReason,
			ContributingReasons: collectContributingReasons(health, fiveMinute),
			PenaltySnapshot:     health.Explanation.TopPenalties,
			TriggerCondition:    fmt.Sprintf("score=%d state=%s recovery_threshold=%d", health.Score, health.State, recoveryThreshold(cfg, health.State)),
		}, now))
	}
	if fiveMinute.CountsByReason["retry_threshold_exceeded"] > 0 && alertAllowed(cfg, persisted[fmt.Sprintf("%s|retry_threshold_exceeded|retry-threshold", name)], persisted, name, "retry_threshold_exceeded", now) {
		alerts = append(alerts, newAlert(cfg, persisted[fmt.Sprintf("%s|retry_threshold_exceeded|retry-threshold", name)], statuspkg.TransportAlert{ID: fmt.Sprintf("%s|retry_threshold_exceeded|retry-threshold", name), TransportName: name, TransportType: typ, Severity: "critical", Reason: "retry_threshold_exceeded", Summary: "retry threshold exceeded within the active anomaly window", EpisodeID: health.Explanation.ActiveEpisodeID, ClusterKey: "retry-threshold", ContributingReasons: []string{"retry_threshold_exceeded"}, PenaltySnapshot: penalties, TriggerCondition: fmt.Sprintf("retry_threshold_exceeded_count=%d", fiveMinute.CountsByReason["retry_threshold_exceeded"])}, now))
	}
	if fiveMinute.DeadLetters >= 2 && alertAllowed(cfg, persisted[fmt.Sprintf("%s|dead_letter_burst|dead-letter-burst", name)], persisted, name, "dead_letter_burst", now) {
		alerts = append(alerts, newAlert(cfg, persisted[fmt.Sprintf("%s|dead_letter_burst|dead-letter-burst", name)], statuspkg.TransportAlert{ID: fmt.Sprintf("%s|dead_letter_burst|dead-letter-burst", name), TransportName: name, TransportType: typ, Severity: severityForBurst(fiveMinute.DeadLetters), Reason: "dead_letter_burst", Summary: fmt.Sprintf("%d dead letters recorded within the active anomaly window", fiveMinute.DeadLetters), ClusterKey: "dead-letter-burst", ContributingReasons: []string{"dead_letter_burst"}, PenaltySnapshot: penalties, TriggerCondition: fmt.Sprintf("dead_letters=%d", fiveMinute.DeadLetters)}, now))
	}
	if health.Signals.LastHeartbeatDeltaSec >= 120 && alertAllowed(cfg, persisted[fmt.Sprintf("%s|heartbeat_loss|heartbeat-loss", name)], persisted, name, "heartbeat_loss", now) {
		alerts = append(alerts, newAlert(cfg, persisted[fmt.Sprintf("%s|heartbeat_loss|heartbeat-loss", name)], statuspkg.TransportAlert{ID: fmt.Sprintf("%s|heartbeat_loss|heartbeat-loss", name), TransportName: name, TransportType: typ, Severity: healthStateSeverity(health.State), Reason: "heartbeat_loss", Summary: fmt.Sprintf("heartbeat gap is %ds", health.Signals.LastHeartbeatDeltaSec), ContributingReasons: []string{fmt.Sprintf("heartbeat_gap_%ds", health.Signals.LastHeartbeatDeltaSec)}, PenaltySnapshot: penalties, TriggerCondition: fmt.Sprintf("last_heartbeat_delta_seconds=%d", health.Signals.LastHeartbeatDeltaSec)}, now))
	}
	if fiveMinute.ObservationDrops > 0 && alertAllowed(cfg, persisted[fmt.Sprintf("%s|evidence_loss|evidence-loss|%s", name, dominantDropCause(fiveMinute))], persisted, name, "evidence_loss", now) {
		dropCause := dominantDropCause(fiveMinute)
		alerts = append(alerts, newAlert(cfg, persisted[fmt.Sprintf("%s|evidence_loss|evidence-loss|%s", name, dropCause)], statuspkg.TransportAlert{ID: fmt.Sprintf("%s|evidence_loss|evidence-loss|%s", name, dropCause), TransportName: name, TransportType: typ, Severity: severityForObservationDrops(fiveMinute.ObservationDrops), Reason: "evidence_loss", Summary: fmt.Sprintf("%d observation drops recorded within the active anomaly window (%s)", fiveMinute.ObservationDrops, dropCause), ClusterKey: dropCause, ContributingReasons: []string{dropCause}, ClusterReference: dropCause, PenaltySnapshot: penalties, TriggerCondition: fmt.Sprintf("observation_drops=%d dominant_drop_cause=%s", fiveMinute.ObservationDrops, dropCause)}, now))
	}
	if health.Signals.ActiveEpisode && health.Signals.RecentFailures >= 2 && alertAllowed(cfg, persisted[fmt.Sprintf("%s|active_failure_episode|active-episode", name)], persisted, name, "active_failure_episode", now) {
		alerts = append(alerts, newAlert(cfg, persisted[fmt.Sprintf("%s|active_failure_episode|active-episode", name)], statuspkg.TransportAlert{ID: fmt.Sprintf("%s|active_failure_episode|active-episode", name), TransportName: name, TransportType: typ, Severity: healthStateSeverity(health.State), Reason: "active_failure_episode", Summary: fmt.Sprintf("failure episode remains active with %d recent failures", health.Signals.RecentFailures), EpisodeID: health.Explanation.ActiveEpisodeID, ContributingReasons: collectContributingReasons(health, fiveMinute), PenaltySnapshot: penalties, TriggerCondition: fmt.Sprintf("active_episode=true recent_failures=%d", health.Signals.RecentFailures)}, now))
	}
	for _, cluster := range clusters {
		if cluster.Severity == "info" {
			continue
		}
		id := fmt.Sprintf("%s|cluster|%s", name, cluster.ClusterKey)
		if !alertAllowed(cfg, persisted[id], persisted, name, cluster.Reason, now) {
			continue
		}
		alerts = append(alerts, newAlert(cfg, persisted[id], statuspkg.TransportAlert{ID: id, TransportName: name, TransportType: typ, Severity: cluster.Severity, Reason: cluster.Reason, Summary: fmt.Sprintf("cluster %s repeated %d times", cluster.Reason, cluster.Count), EpisodeID: cluster.EpisodeID, ClusterKey: cluster.ClusterKey, ClusterReference: cluster.ClusterKey, ContributingReasons: []string{cluster.Reason}, PenaltySnapshot: penalties, TriggerCondition: fmt.Sprintf("cluster_count=%d severity=%s", cluster.Count, cluster.Severity)}, now))
	}
	dedup := map[string]statuspkg.TransportAlert{}
	for _, alert := range alerts {
		dedup[alert.ID] = alert
	}
	out := make([]statuspkg.TransportAlert, 0, len(dedup))
	for _, alert := range dedup {
		out = append(out, alert)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Severity == out[j].Severity {
			return out[i].Reason < out[j].Reason
		}
		return alertSeverityRank(out[i].Severity) > alertSeverityRank(out[j].Severity)
	})
	return out
}

func shouldKeepHealthAlert(cfg config.Config, previous db.TransportHealthSnapshot, health statuspkg.TransportHealth, persisted map[string]db.TransportAlertRecord, name string, database *db.DB, now time.Time) bool {
	if transportRecentlyRecovered(cfg, persisted, name, now) {
		return false
	}
	if health.State == "healthy" && health.Score >= cfg.Intelligence.Alerts.RecoveryScoreHealthy {
		return false
	}
	if previous.State == "" && health.State == "healthy" {
		return false
	}
	if previous.State == health.State && health.Score >= recoveryThreshold(cfg, health.State) {
		return false
	}
	stateSince := now
	if database != nil {
		before := now.Add(-time.Duration(cfg.Intelligence.Alerts.MinimumStateDurationSeconds) * time.Second).Format(time.RFC3339)
		if snap, ok, err := database.LatestTransportSnapshotBefore(name, before); err == nil && ok && snap.State == health.State {
			if parsed, ok := parseRFC3339(snap.SnapshotTime); ok {
				stateSince = parsed
			}
		}
	}
	if now.Sub(stateSince) < time.Duration(cfg.Intelligence.Alerts.MinimumStateDurationSeconds)*time.Second {
		return false
	}
	id := fmt.Sprintf("%s|health_state_transition|%s", name, health.State)
	if cooldownBlocksReason(cfg, persisted, name, "health_state_transition", now) {
		return false
	}
	if alert, ok := persisted[id]; ok && alert.Active {
		return health.Score < recoveryThreshold(cfg, health.State)
	}
	return previous.State != health.State || health.Score < recoveryThreshold(cfg, health.State)
}

func alertAllowed(cfg config.Config, persisted db.TransportAlertRecord, all map[string]db.TransportAlertRecord, transportName, reason string, now time.Time) bool {
	if cooldownBlocksReason(cfg, all, transportName, reason, now) {
		return false
	}
	if persisted.ID == "" {
		return true
	}
	if persisted.Active {
		return true
	}
	resolvedAt, ok := parseRFC3339(persisted.ResolvedAt)
	if !ok {
		return true
	}
	return now.Sub(resolvedAt) >= time.Duration(cfg.Intelligence.Alerts.CooldownSeconds)*time.Second
}

func transportRecentlyRecovered(cfg config.Config, persisted map[string]db.TransportAlertRecord, transportName string, now time.Time) bool {
	for _, alert := range persisted {
		if alert.TransportName != transportName || alert.Active {
			continue
		}
		if resolvedAt, ok := parseRFC3339(alert.ResolvedAt); ok && now.Sub(resolvedAt) < time.Duration(cfg.Intelligence.Alerts.CooldownSeconds)*time.Second {
			return true
		}
	}
	return false
}

func cooldownBlocksReason(cfg config.Config, persisted map[string]db.TransportAlertRecord, transportName, reason string, now time.Time) bool {
	for _, alert := range persisted {
		if alert.TransportName != transportName || alert.Reason != reason || alert.Active {
			continue
		}
		if resolvedAt, ok := parseRFC3339(alert.ResolvedAt); ok && now.Sub(resolvedAt) < time.Duration(cfg.Intelligence.Alerts.CooldownSeconds)*time.Second {
			return true
		}
	}
	return false
}

func recoveryThreshold(cfg config.Config, state string) int {
	switch state {
	case "failed":
		return cfg.Intelligence.Alerts.RecoveryScoreUnstable
	case "unstable":
		return cfg.Intelligence.Alerts.RecoveryScoreDegraded
	case "degraded":
		return cfg.Intelligence.Alerts.RecoveryScoreHealthy
	default:
		return 100
	}
}

func newAlert(_ config.Config, persisted db.TransportAlertRecord, base statuspkg.TransportAlert, now time.Time) statuspkg.TransportAlert {
	first := now.UTC().Format(time.RFC3339)
	if persisted.FirstTriggeredAt != "" {
		first = persisted.FirstTriggeredAt
	}
	base.FirstTriggeredAt = first
	base.LastUpdatedAt = now.UTC().Format(time.RFC3339)
	base.Active = true
	if base.ClusterReference == "" {
		base.ClusterReference = base.ClusterKey
	}
	return base
}

func toAlertRecord(alert statuspkg.TransportAlert) db.TransportAlertRecord {
	penalties := make([]db.PenaltyRecord, 0, len(alert.PenaltySnapshot))
	for _, penalty := range alert.PenaltySnapshot {
		penalties = append(penalties, db.PenaltyRecord{Reason: penalty.Reason, Penalty: penalty.Penalty, Count: penalty.Count, Window: penalty.Window})
	}
	return db.TransportAlertRecord{ID: alert.ID, TransportName: alert.TransportName, TransportType: alert.TransportType, Severity: alert.Severity, Reason: alert.Reason, Summary: alert.Summary, FirstTriggeredAt: alert.FirstTriggeredAt, LastUpdatedAt: alert.LastUpdatedAt, Active: alert.Active, EpisodeID: alert.EpisodeID, ClusterKey: alert.ClusterKey, ContributingReasons: alert.ContributingReasons, ClusterReference: alert.ClusterReference, PenaltySnapshot: penalties, TriggerCondition: alert.TriggerCondition}
}

func topPenaltyRecords(in []statuspkg.HealthPenalty) []statuspkg.HealthPenalty {
	return append([]statuspkg.HealthPenalty(nil), in...)
}

func collectContributingReasons(health statuspkg.TransportHealth, recent statuspkg.TransportAnomalySummary) []string {
	reasons := []string{}
	for reason := range recent.CountsByReason {
		reasons = append(reasons, reason)
	}
	for cause := range recent.DropCauses {
		reasons = append(reasons, cause)
	}
	for _, penalty := range health.Explanation.TopPenalties {
		reasons = append(reasons, penalty.Reason)
	}
	reasons = dedupe(reasons)
	sort.Strings(reasons)
	return reasons
}

func recentAnomalyLabel(cfg config.Config) string {
	if len(cfg.Intelligence.Scoring.AnomalyWindowsSeconds) > 1 {
		return durationLabel(time.Duration(cfg.Intelligence.Scoring.AnomalyWindowsSeconds[1]) * time.Second)
	}
	return "5m"
}

func durationLabel(d time.Duration) string {
	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	return fmt.Sprintf("%ds", int(d/time.Second))
}

func parseRFC3339(v string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, strings.TrimSpace(v)); err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}

func dedupe(in []string) []string {
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

func healthStateSeverity(state string) string {
	switch state {
	case "failed":
		return "critical"
	case "unstable", "degraded":
		return "warn"
	default:
		return "info"
	}
}

func severityForBurst(count uint64) string {
	if count >= 4 {
		return "critical"
	}
	return "warn"
}

func severityForObservationDrops(count uint64) string {
	if count >= 5 {
		return "critical"
	}
	return "warn"
}

func dominantDropCause(summary statuspkg.TransportAnomalySummary) string {
	if len(summary.DropCauses) == 0 {
		return "unspecified_drop_cause"
	}
	type pair struct {
		cause string
		count uint64
	}
	pairs := make([]pair, 0, len(summary.DropCauses))
	for cause, count := range summary.DropCauses {
		pairs = append(pairs, pair{cause: cause, count: count})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count == pairs[j].count {
			return pairs[i].cause < pairs[j].cause
		}
		return pairs[i].count > pairs[j].count
	})
	return pairs[0].cause
}

func alertSeverityRank(severity string) int {
	switch severity {
	case "critical":
		return 3
	case "warn":
		return 2
	default:
		return 1
	}
}

func anomalySnapshotsForTransport(name, typ string, summary statuspkg.TransportAnomalySummary, now time.Time) []db.TransportAnomalySnapshot {
	bucket := now.UTC().Truncate(time.Minute).Format(time.RFC3339)
	out := make([]db.TransportAnomalySnapshot, 0, len(summary.CountsByReason)+2)
	for _, reason := range sortedReasonKeys(summary.CountsByReason) {
		out = append(out, db.TransportAnomalySnapshot{BucketStart: bucket, TransportName: name, TransportType: typ, Reason: reason, Count: summary.CountsByReason[reason]})
	}
	if summary.DeadLetters > 0 {
		out = append(out, db.TransportAnomalySnapshot{BucketStart: bucket, TransportName: name, TransportType: typ, Reason: "dead_letter_burst", Count: summary.DeadLetters, DeadLetters: summary.DeadLetters})
	}
	if summary.ObservationDrops > 0 {
		out = append(out, db.TransportAnomalySnapshot{BucketStart: bucket, TransportName: name, TransportType: typ, Reason: dominantDropCause(summary), Count: summary.ObservationDrops, ObservationDrops: summary.ObservationDrops, DropCauses: summary.DropCauses})
	}
	if len(out) == 0 {
		out = append(out, db.TransportAnomalySnapshot{BucketStart: bucket, TransportName: name, TransportType: typ, Reason: "no_recent_anomalies", Count: 0})
	}
	return out
}

func sortedReasonKeys[V ~uint64](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
