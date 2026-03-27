package status

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/runtime"
	"github.com/mel-project/mel/internal/transport"
)

type Snapshot struct {
	GeneratedAt              string            `json:"generated_at"`
	Bind                     string            `json:"bind,omitempty"`
	BindLocalOnly            bool              `json:"bind_local_only"`
	SchemaVersion            string            `json:"schema_version,omitempty"`
	ConfiguredTransportModes []string          `json:"configured_transport_modes"`
	Messages                 int64             `json:"messages"`
	Nodes                    int64             `json:"nodes"`
	LastSuccessfulIngest     string            `json:"last_successful_ingest,omitempty"`
	Transports               []TransportReport `json:"transports"`
	RecentIncidents          []models.Incident `json:"recent_incidents,omitempty"`
	ActiveTransportAlerts    []TransportAlert  `json:"active_transport_alerts,omitempty"`
	Mesh                     MeshDrilldown     `json:"mesh"`
	// Product is build-time capability and honest deployment scope (single gateway).
	Product runtime.ProductEnvelope `json:"product"`
	// Instance is durable SQLite identity plus optional live process fields when status is assembled inside mel serve.
	Instance runtime.InstanceTruth `json:"instance"`
}

type TransportReport struct {
	Name                string                     `json:"name"`
	Type                string                     `json:"type"`
	Source              string                     `json:"source"`
	Enabled             bool                       `json:"enabled"`
	EffectiveState      string                     `json:"effective_state"`
	RuntimeState        string                     `json:"runtime_state"`
	PersistedState      string                     `json:"persisted_state"`
	StatusScope         string                     `json:"status_scope"`
	Detail              string                     `json:"detail"`
	Guidance            string                     `json:"guidance"`
	LastAttemptAt       string                     `json:"last_attempt_at,omitempty"`
	LastIngestAt        string                     `json:"last_ingest_at,omitempty"`
	LastHeartbeatAt     string                     `json:"last_heartbeat_at,omitempty"`
	LastError           string                     `json:"last_error,omitempty"`
	LastFailureAt       string                     `json:"last_failure_at,omitempty"`
	EpisodeID           string                     `json:"episode_id,omitempty"`
	FailureCount        uint64                     `json:"failure_count"`
	ObservationDrops    uint64                     `json:"observation_drops"`
	TotalMessages       uint64                     `json:"total_messages"`
	PersistedMessages   uint64                     `json:"persisted_messages"`
	ErrorCount          uint64                     `json:"error_count"`
	DroppedCount        uint64                     `json:"dropped_count"`
	ReconnectAttempts   uint64                     `json:"reconnect_attempts"`
	ConsecutiveTimeouts uint64                     `json:"consecutive_timeouts"`
	DeadLetters         uint64                     `json:"dead_letters"`
	RetryStatus         string                     `json:"retry_status"`
	Capabilities        transport.CapabilityMatrix `json:"capabilities"`
	Health              TransportHealth            `json:"health"`
	ActiveAlerts        []TransportAlert           `json:"active_alerts,omitempty"`
	RecentAnomalies     []TransportAnomalySummary  `json:"recent_anomalies,omitempty"`
	FailureClusters     []FailureCluster           `json:"failure_clusters,omitempty"`
}

type persistEvidence struct {
	Count      uint64
	LastIngest string
}

// Collect assembles operator-facing status. When processStartedAt is non-nil (mel serve), Instance includes PID, start time, and uptime.
// configPath is non-empty when the caller knows the effective config file path (e.g. mel serve); pass empty for CLI-only snapshots.
func Collect(cfg config.Config, database *db.DB, transportHealth []transport.Health, processStartedAt *time.Time, configPath string) (Snapshot, error) {
	snap := Snapshot{
		GeneratedAt:              time.Now().UTC().Format(time.RFC3339),
		Bind:                     cfg.Bind.API,
		BindLocalOnly:            !cfg.Bind.AllowRemote,
		ConfiguredTransportModes: enabledModes(cfg),
		Transports:               make([]TransportReport, 0, len(cfg.Transports)),
		Product:                  runtime.BuildProductEnvelope(cfg),
	}
	snap.Instance.BindAPI = cfg.Bind.API
	if strings.TrimSpace(configPath) != "" {
		snap.Instance.ConfigPath = strings.TrimSpace(configPath)
	}
	if database != nil {
		if id, err := database.EnsureInstanceID(); err == nil {
			snap.Instance.InstanceID = id
		}
		snap.Instance.DataDir = cfg.Storage.DataDir
		snap.Instance.DatabasePath = cfg.Storage.DatabasePath
		schemaVersion, _ := database.SchemaVersion()
		snap.SchemaVersion = schemaVersion
		snap.LastSuccessfulIngest, _ = database.Scalar("SELECT COALESCE(MAX(rx_time), '') FROM messages;")
		snap.Messages = scalarInt(database, "SELECT COUNT(*) FROM messages;")
		snap.Nodes = scalarInt(database, "SELECT COUNT(*) FROM nodes;")
	}
	if processStartedAt != nil {
		p := runtime.NewProcessIdentity(*processStartedAt)
		snap.Instance.Process = &p
		snap.Instance.UptimeSeconds = int64(time.Since(*processStartedAt).Seconds())
	}
	persisted, err := persistedEvidenceByTransport(database)
	if err != nil {
		return snap, err
	}
	persistedRuntime := map[string]transport.Health{}
	if database != nil {
		rows, err := database.TransportRuntimeStatuses()
		if err != nil {
			return snap, err
		}
		for _, row := range rows {
			persistedRuntime[row.Name] = transport.Health{
				Name:                row.Name,
				Type:                row.Type,
				Source:              row.Source,
				State:               row.State,
				Detail:              row.Detail,
				LastAttemptAt:       row.LastAttemptAt,
				LastConnectedAt:     row.LastConnectedAt,
				LastSuccessAt:       row.LastSuccessAt,
				LastIngestAt:        row.LastMessageAt,
				LastHeartbeatAt:     row.LastHeartbeatAt,
				LastError:           row.LastError,
				LastFailureAt:       row.LastFailureAt,
				EpisodeID:           row.EpisodeID,
				TotalMessages:       row.TotalMessages,
				PacketsDropped:      row.PacketsDropped,
				ReconnectAttempts:   row.Reconnects,
				ConsecutiveTimeouts: row.Timeouts,
				FailureCount:        row.FailureCount,
				ObservationDrops:    row.ObservationDrops,
			}
		}
	}
	deadLetters, err := deadLetterEvidenceByTransport(database)
	if err != nil {
		return snap, err
	}
	if database != nil {
		snap.RecentIncidents, _ = database.RecentIncidents(20)
	}
	intelligence, err := EvaluateTransportIntelligence(cfg, database, transportHealth, time.Now().UTC())
	if err != nil {
		return snap, err
	}
	if database != nil {
		if alerts, err := database.TransportAlerts(true); err == nil {
			snap.ActiveTransportAlerts = make([]TransportAlert, 0, len(alerts))
			for _, alert := range alerts {
				penalties := make([]HealthPenalty, 0, len(alert.PenaltySnapshot))
				for _, penalty := range alert.PenaltySnapshot {
					penalties = append(penalties, HealthPenalty{Reason: penalty.Reason, Penalty: penalty.Penalty, Count: penalty.Count, Window: penalty.Window})
				}
				converted := TransportAlert{
					ID:                  alert.ID,
					TransportName:       alert.TransportName,
					TransportType:       alert.TransportType,
					Severity:            alert.Severity,
					Reason:              alert.Reason,
					Summary:             alert.Summary,
					FirstTriggeredAt:    alert.FirstTriggeredAt,
					LastUpdatedAt:       alert.LastUpdatedAt,
					Active:              alert.Active,
					EpisodeID:           alert.EpisodeID,
					ClusterKey:          alert.ClusterKey,
					ContributingReasons: alert.ContributingReasons,
					ClusterReference:    alert.ClusterReference,
					PenaltySnapshot:     penalties,
					TriggerCondition:    alert.TriggerCondition,
				}
				snap.ActiveTransportAlerts = append(snap.ActiveTransportAlerts, converted)
				intelligence.AlertsByTransport[alert.TransportName] = append(intelligence.AlertsByTransport[alert.TransportName], converted)
			}
		}
	}
	runtimeMap := map[string]transport.Health{}
	for _, h := range transportHealth {
		runtimeMap[h.Name] = h
	}
	snap.Mesh = buildMeshDrilldown(cfg, database, intelligence, snap.ActiveTransportAlerts, time.Now().UTC())
	for _, tc := range cfg.Transports {
		h := runtimeMap[tc.Name]
		if h.Name == "" {
			h = persistedRuntime[tc.Name]
		}
		if h.Name == "" {
			built, err := transport.Build(tc, nil, nil)
			if err == nil {
				h = built.Health()
			} else {
				h = transport.Health{Name: tc.Name, Type: tc.Type, Source: tc.SourceLabel(), State: transport.StateError, Detail: err.Error()}
			}
		} else if h.Capabilities == (transport.CapabilityMatrix{}) {
			if built, err := transport.Build(tc, nil, nil); err == nil {
				h.Capabilities = built.Capabilities()
				if h.Source == "" {
					h.Source = built.Health().Source
				}
			}
		}
		evidence := persisted[tc.Name]
		report := TransportReport{
			Name:                tc.Name,
			Type:                tc.Type,
			Source:              tc.SourceLabel(),
			Enabled:             tc.Enabled,
			EffectiveState:      EffectiveState(tc.Enabled, h.State, evidence.Count),
			RuntimeState:        h.State,
			PersistedState:      persistedState(evidence.Count),
			StatusScope:         statusScope(h.State, evidence.Count),
			Detail:              detailFor(tc.Enabled, h, evidence),
			Guidance:            guidanceFor(tc.Enabled, EffectiveState(tc.Enabled, h.State, evidence.Count), tc.Type),
			LastAttemptAt:       h.LastAttemptAt,
			LastIngestAt:        firstNonEmpty(h.LastIngestAt, evidence.LastIngest),
			LastHeartbeatAt:     h.LastHeartbeatAt,
			LastError:           h.LastError,
			LastFailureAt:       h.LastFailureAt,
			EpisodeID:           h.EpisodeID,
			FailureCount:        h.FailureCount,
			ObservationDrops:    h.ObservationDrops,
			TotalMessages:       h.TotalMessages,
			PersistedMessages:   evidence.Count,
			ErrorCount:          h.ErrorCount,
			DroppedCount:        h.PacketsDropped,
			ReconnectAttempts:   h.ReconnectAttempts,
			ConsecutiveTimeouts: h.ConsecutiveTimeouts,
			DeadLetters:         deadLetters[tc.Name],
			RetryStatus:         retryStatus(EffectiveState(tc.Enabled, h.State, evidence.Count), h.ReconnectAttempts, h.ConsecutiveTimeouts),
			Capabilities:        h.Capabilities,
			Health:              intelligence.HealthByTransport[tc.Name],
			ActiveAlerts:        intelligence.AlertsByTransport[tc.Name],
			RecentAnomalies:     intelligence.AnomaliesByTransport[tc.Name],
			FailureClusters:     intelligence.ClustersByTransport[tc.Name],
		}
		snap.Transports = append(snap.Transports, report)
	}
	sort.Slice(snap.Transports, func(i, j int) bool { return snap.Transports[i].Name < snap.Transports[j].Name })
	return snap, nil
}

func EffectiveState(enabled bool, runtimeState string, persistedCount uint64) string {
	if !enabled {
		return transport.StateDisabled
	}
	if persistedCount > 0 && (runtimeState == "" || runtimeState == transport.StateConfiguredNotAttempted || runtimeState == transport.StateConfiguredOffline || runtimeState == transport.StateError) {
		return transport.StateHistoricalOnly
	}
	if runtimeState == "" {
		return transport.StateConfiguredNotAttempted
	}
	return runtimeState
}

func persistedState(count uint64) string {
	if count > 0 {
		return transport.StateHistoricalOnly
	}
	return transport.StateConfiguredNotAttempted
}

func statusScope(runtimeState string, persistedCount uint64) string {
	if runtimeState == transport.StateLive || runtimeState == transport.StateIdle || runtimeState == transport.StateConnecting || runtimeState == transport.StateRetrying || runtimeState == transport.StateFailed {
		return "runtime+persisted"
	}
	if persistedCount > 0 {
		return "persisted_only"
	}
	return "config+persisted"
}

func detailFor(enabled bool, h transport.Health, evidence persistEvidence) string {
	if !enabled {
		return "disabled by config"
	}
	if evidence.Count > 0 && (h.State == "" || h.State == transport.StateConfiguredNotAttempted || h.State == transport.StateConfiguredOffline || h.State == transport.StateError) {
		return fmt.Sprintf("historical ingest exists (%d stored messages); current runtime state is not proven live", evidence.Count)
	}
	if h.Detail != "" {
		return h.Detail
	}
	if evidence.Count > 0 {
		return "historical ingest exists, but no runtime process has updated this transport in the current command"
	}
	return "configured transport has not been proven by a stored message yet"
}

func guidanceFor(enabled bool, state, transportType string) string {
	if !enabled {
		return "Enable the transport before expecting live ingest."
	}
	switch state {
	case transport.StateConfigured:
		return "Start mel serve, then verify this transport transitions to live only after messages are written to SQLite."
	case transport.StateConnecting:
		return "Verify endpoint reachability and credentials if this state does not clear quickly."
	case transport.StateRetrying:
		return "Check cable, host, port, topic, and node ownership; MEL is configured but not currently connected."
	case transport.StateIdle:
		return "Connection is up but no payload has been stored yet; generate real mesh traffic before trusting the path."
	case transport.StateLive:
		return "Live ingest is confirmed by successful database writes."
	case transport.StateHistoricalOnly:
		return "Past messages exist, but this command cannot prove current live connectivity."
	case transport.StateFailed:
		if transportType == "mqtt" {
			return "Inspect broker reachability, topic alignment, and credentials; errors are surfaced directly in doctor and logs."
		}
		return "Inspect transport errors, ownership contention, and malformed input."
	default:
		return "Inspect transport status and logs."
	}
}

func persistedEvidenceByTransport(database *db.DB) (map[string]persistEvidence, error) {
	out := map[string]persistEvidence{}
	if database == nil {
		return out, nil
	}
	rows, err := database.QueryRows("SELECT transport_name, COUNT(*) AS message_count, COALESCE(MAX(rx_time), '') AS last_ingest_at FROM messages GROUP BY transport_name;")
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[fmt.Sprint(row["transport_name"])] = persistEvidence{Count: uint64(asInt(row["message_count"])), LastIngest: fmt.Sprint(row["last_ingest_at"])}
	}
	return out, nil
}

func deadLetterEvidenceByTransport(database *db.DB) (map[string]uint64, error) {
	if database == nil {
		return map[string]uint64{}, nil
	}
	return database.DeadLetterCounts()
}

func scalarInt(database *db.DB, sql string) int64 {
	if database == nil {
		return 0
	}
	v, _ := database.Scalar(sql)
	return asInt(v)
}

func enabledModes(cfg config.Config) []string {
	out := make([]string, 0)
	for _, t := range cfg.Transports {
		if t.Enabled {
			out = append(out, t.Type)
		}
	}
	if len(out) == 0 {
		return []string{"none"}
	}
	sort.Strings(out)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func asInt(v any) int64 {
	switch x := v.(type) {
	case int:
		return int64(x)
	case int64:
		return x
	case float64:
		return int64(x)
	case string:
		var parsed int64
		fmt.Sscan(x, &parsed)
		return parsed
	}
	var parsed int64
	fmt.Sscan(fmt.Sprint(v), &parsed)
	return parsed
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func retryStatus(state string, reconnectAttempts, consecutiveTimeouts uint64) string {
	switch state {
	case transport.StateConnecting:
		return fmt.Sprintf("retrying connection (attempt %d)", reconnectAttempts)
	case transport.StateRetrying, transport.StateFailed:
		if reconnectAttempts > 0 {
			return fmt.Sprintf("backoff armed after %d reconnect attempts", reconnectAttempts)
		}
		return "retry pending after surfaced error"
	case transport.StateIdle, transport.StateLive:
		if consecutiveTimeouts > 0 {
			return fmt.Sprintf("connected with %d consecutive read timeout(s)", consecutiveTimeouts)
		}
		if state == transport.StateLive {
			return "live evidence present; no retry pending"
		}
		return "idle evidence present; no retry pending"
	case transport.StateHistoricalOnly:
		return "no live retry state in this process; inspect stored evidence"
	case transport.StateDisabled:
		return "disabled"
	default:
		return "no retry evidence"
	}
}
