package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/mel-project/mel/internal/db"
	statuspkg "github.com/mel-project/mel/internal/status"
)

const defaultIntelligenceInterval = 15 * time.Second

func (a *App) intelligenceInterval() time.Duration {
	if a == nil || a.DB == nil {
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
	for _, tc := range a.Cfg.Transports {
		health, ok := intelligence.HealthByTransport[tc.Name]
		if !ok {
			continue
		}
		alerts := deriveTransportAlerts(tc.Name, tc.Type, health, intelligence.AnomaliesByTransport[tc.Name], intelligence.ClustersByTransport[tc.Name], previousSnapshots[tc.Name], now)
		ids := make([]string, 0, len(alerts))
		for _, alert := range alerts {
			ids = append(ids, alert.ID)
			if err := a.DB.UpsertTransportAlert(db.TransportAlertRecord{
				ID:               alert.ID,
				TransportName:    alert.TransportName,
				TransportType:    alert.TransportType,
				Severity:         alert.Severity,
				Reason:           alert.Reason,
				Summary:          alert.Summary,
				FirstTriggeredAt: alert.FirstTriggeredAt,
				LastUpdatedAt:    alert.LastUpdatedAt,
				Active:           true,
				EpisodeID:        alert.EpisodeID,
				ClusterKey:       alert.ClusterKey,
			}); err != nil {
				a.Log.Error("transport_alert_upsert_failed", "failed to persist transport alert", map[string]any{"transport": alert.TransportName, "reason": alert.Reason, "error": err.Error()})
			}
		}
		if err := a.DB.ResolveTransportAlertsNotIn(tc.Name, ids, now.UTC().Format(time.RFC3339)); err != nil {
			a.Log.Error("transport_alert_resolve_failed", "failed to resolve inactive transport alerts", map[string]any{"transport": tc.Name, "error": err.Error()})
		}
		if err := a.DB.InsertTransportHealthSnapshot(db.TransportHealthSnapshot{
			TransportName:              tc.Name,
			TransportType:              tc.Type,
			Score:                      health.Score,
			State:                      health.State,
			SnapshotTime:               now.UTC().Format(time.RFC3339),
			ActiveAlertCount:           len(alerts),
			DeadLetterCountWindow:      int(statuspkg.AnomalyWindow(intelligence.AnomaliesByTransport[tc.Name], "5m").DeadLetters),
			ObservationDropCountWindow: int(statuspkg.AnomalyWindow(intelligence.AnomaliesByTransport[tc.Name], "5m").ObservationDrops),
		}); err != nil {
			a.Log.Error("transport_snapshot_insert_failed", "failed to persist transport health snapshot", map[string]any{"transport": tc.Name, "error": err.Error()})
		}
	}
}

func deriveTransportAlerts(name, typ string, health statuspkg.TransportHealth, anomalies []statuspkg.TransportAnomalySummary, clusters []statuspkg.FailureCluster, previous db.TransportHealthSnapshot, now time.Time) []statuspkg.TransportAlert {
	alerts := []statuspkg.TransportAlert{}
	fiveMinute := statuspkg.AnomalyWindow(anomalies, "5m")
	if previous.State != "" && statuspkg.WorseHealthState(previous.State, health.State) {
		alerts = append(alerts, newAlert(name, typ, "health_state_transition", health.Signals, health.PrimaryReason, now, healthStateSeverity(health.State), fmt.Sprintf("health moved from %s to %s (score=%d)", previous.State, health.State, health.Score), fmt.Sprintf("health-state|%s|%s", previous.State, health.State)))
	}
	if fiveMinute.CountsByReason["retry_threshold_exceeded"] > 0 {
		alerts = append(alerts, newAlert(name, typ, "retry_threshold_exceeded", health.Signals, health.PrimaryReason, now, "critical", "retry threshold exceeded within the last 5 minutes", "retry-threshold"))
	}
	if fiveMinute.DeadLetters >= 2 {
		alerts = append(alerts, newAlert(name, typ, "dead_letter_burst", health.Signals, health.PrimaryReason, now, severityForBurst(fiveMinute.DeadLetters), fmt.Sprintf("%d dead letters recorded within the last 5 minutes", fiveMinute.DeadLetters), "dead-letter-burst"))
	}
	if health.Signals.LastHeartbeatDeltaSec >= 120 {
		alerts = append(alerts, newAlert(name, typ, "heartbeat_loss", health.Signals, health.PrimaryReason, now, healthStateSeverity(health.State), fmt.Sprintf("heartbeat gap is %ds", health.Signals.LastHeartbeatDeltaSec), "heartbeat-loss"))
	}
	if fiveMinute.ObservationDrops > 0 {
		dropCause := dominantDropCause(fiveMinute)
		alerts = append(alerts, newAlert(name, typ, "evidence_loss", health.Signals, dropCause, now, severityForObservationDrops(fiveMinute.ObservationDrops), fmt.Sprintf("%d observation drops recorded within the last 5 minutes (%s)", fiveMinute.ObservationDrops, dropCause), fmt.Sprintf("evidence-loss|%s", dropCause)))
	}
	if health.Signals.ActiveEpisode && health.Signals.RecentFailures >= 2 {
		alerts = append(alerts, newAlert(name, typ, "active_failure_episode", health.Signals, health.PrimaryReason, now, healthStateSeverity(health.State), fmt.Sprintf("failure episode remains active with %d recent failures", health.Signals.RecentFailures), "active-episode"))
	}
	for _, cluster := range clusters {
		if cluster.Severity == "info" {
			continue
		}
		alerts = append(alerts, statuspkg.TransportAlert{
			ID:               fmt.Sprintf("%s|cluster|%s", name, cluster.ClusterKey),
			TransportName:    name,
			TransportType:    typ,
			Severity:         cluster.Severity,
			Reason:           cluster.Reason,
			Summary:          fmt.Sprintf("cluster %s repeated %d times", cluster.Reason, cluster.Count),
			FirstTriggeredAt: now.UTC().Format(time.RFC3339),
			LastUpdatedAt:    now.UTC().Format(time.RFC3339),
			Active:           true,
			EpisodeID:        cluster.EpisodeID,
			ClusterKey:       cluster.ClusterKey,
		})
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

func newAlert(name, typ, reason string, _ statuspkg.TransportHealthSignals, clusterKey string, now time.Time, severity, summary, suffix string) statuspkg.TransportAlert {
	return statuspkg.TransportAlert{
		ID:               fmt.Sprintf("%s|%s|%s", name, reason, suffix),
		TransportName:    name,
		TransportType:    typ,
		Severity:         severity,
		Reason:           reason,
		Summary:          summary,
		FirstTriggeredAt: now.UTC().Format(time.RFC3339),
		LastUpdatedAt:    now.UTC().Format(time.RFC3339),
		Active:           true,
		ClusterKey:       clusterKey,
	}
}

func healthStateSeverity(state string) string {
	switch state {
	case "failed":
		return "critical"
	case "unstable":
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
