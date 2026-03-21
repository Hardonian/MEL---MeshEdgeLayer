package status

import (
	"fmt"
	"sort"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/transport"
)

type TransportDrilldown struct {
	TransportName      string                            `json:"transport_name"`
	TransportType      string                            `json:"transport_type"`
	Health             TransportHealth                   `json:"health"`
	HealthExplanation  HealthExplanation                 `json:"health_explanation"`
	RecentClusters     []FailureCluster                  `json:"recent_clusters"`
	RecentAlerts       []TransportAlert                  `json:"recent_alerts"`
	AnomalySummary     []TransportAnomalySummary         `json:"anomaly_summary"`
	LastIncidents      []models.Incident                 `json:"last_incidents"`
	EpisodeHistory     []db.TransportAlertRecord         `json:"episode_history"`
	HealthHistory      []db.TransportHealthSnapshot      `json:"health_history"`
	AlertHistory       []db.TransportAlertRecord         `json:"alert_history"`
	AnomalyHistory     []db.TransportAnomalyHistoryPoint `json:"anomaly_history"`
	TransportConnected bool                              `json:"transport_connected"`
}

func InspectTransport(cfg config.Config, database *db.DB, runtime []transport.Health, name string, now time.Time) (TransportDrilldown, error) {
	intel, err := EvaluateTransportIntelligence(cfg, database, runtime, now)
	if err != nil {
		return TransportDrilldown{}, err
	}
	transportType := ""
	found := false
	for _, tc := range cfg.Transports {
		if tc.Name == name {
			transportType = tc.Type
			found = true
			break
		}
	}
	if !found {
		return TransportDrilldown{}, fmt.Errorf("transport %s not found", name)
	}
	drilldown := TransportDrilldown{TransportName: name, TransportType: transportType, Health: intel.HealthByTransport[name], HealthExplanation: intel.HealthByTransport[name].Explanation, RecentClusters: intel.ClustersByTransport[name], RecentAlerts: intel.AlertsByTransport[name], AnomalySummary: intel.AnomaliesByTransport[name]}
	if database == nil {
		return drilldown, nil
	}
	if incidents, err := database.RecentIncidents(100); err == nil {
		for _, incident := range incidents {
			if incident.ResourceID == name {
				drilldown.LastIncidents = append(drilldown.LastIncidents, incident)
			}
			if len(drilldown.LastIncidents) >= 5 {
				break
			}
		}
	}
	if alerts, err := database.TransportAlertsHistory(name, now.Add(-7*24*time.Hour).Format(time.RFC3339), now.Format(time.RFC3339), cfg.Intelligence.Queries.DefaultLimit, 0); err == nil {
		drilldown.AlertHistory = alerts
		drilldown.EpisodeHistory = filterEpisodeAlerts(alerts)
		for _, alert := range alerts {
			drilldown.RecentAlerts = append(drilldown.RecentAlerts, TransportAlert{ID: alert.ID, TransportName: alert.TransportName, TransportType: alert.TransportType, Severity: alert.Severity, Reason: alert.Reason, Summary: alert.Summary, FirstTriggeredAt: alert.FirstTriggeredAt, LastUpdatedAt: alert.LastUpdatedAt, Active: alert.Active, EpisodeID: alert.EpisodeID, ClusterKey: alert.ClusterKey, ContributingReasons: alert.ContributingReasons, ClusterReference: alert.ClusterReference, TriggerCondition: alert.TriggerCondition})
			if len(drilldown.RecentAlerts) >= 5 {
				break
			}
		}
	}
	if history, err := database.TransportHealthSnapshots(name, now.Add(-24*time.Hour).Format(time.RFC3339), now.Format(time.RFC3339), cfg.Intelligence.Queries.DefaultLimit, 0); err == nil {
		drilldown.HealthHistory = history
	}
	if anomalies, err := database.TransportAnomalyHistory(name, now.Add(-24*time.Hour).Format(time.RFC3339), now.Format(time.RFC3339), cfg.Intelligence.Queries.DefaultLimit, 0); err == nil {
		drilldown.AnomalyHistory = anomalies
	}
	drilldown.TransportConnected = drilldown.Health.State != "" && drilldown.Health.State != "failed"
	sort.Slice(drilldown.RecentAlerts, func(i, j int) bool {
		return drilldown.RecentAlerts[i].LastUpdatedAt > drilldown.RecentAlerts[j].LastUpdatedAt
	})
	return drilldown, nil
}

func filterEpisodeAlerts(alerts []db.TransportAlertRecord) []db.TransportAlertRecord {
	out := make([]db.TransportAlertRecord, 0)
	for _, alert := range alerts {
		if alert.EpisodeID != "" {
			out = append(out, alert)
		}
		if len(out) >= 10 {
			break
		}
	}
	return out
}
