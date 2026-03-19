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
}

type TransportReport struct {
	Name              string                     `json:"name"`
	Type              string                     `json:"type"`
	Source            string                     `json:"source"`
	Enabled           bool                       `json:"enabled"`
	EffectiveState    string                     `json:"effective_state"`
	RuntimeState      string                     `json:"runtime_state"`
	PersistedState    string                     `json:"persisted_state"`
	StatusScope       string                     `json:"status_scope"`
	Detail            string                     `json:"detail"`
	Guidance          string                     `json:"guidance"`
	LastAttemptAt     string                     `json:"last_attempt_at,omitempty"`
	LastIngestAt      string                     `json:"last_ingest_at,omitempty"`
	LastError         string                     `json:"last_error,omitempty"`
	TotalMessages     uint64                     `json:"total_messages"`
	PersistedMessages uint64                     `json:"persisted_messages"`
	ErrorCount        uint64                     `json:"error_count"`
	DroppedCount      uint64                     `json:"dropped_count"`
	ReconnectAttempts uint64                     `json:"reconnect_attempts"`
	Capabilities      transport.CapabilityMatrix `json:"capabilities"`
}

type persistEvidence struct {
	Count      uint64
	LastIngest string
}

func Collect(cfg config.Config, database *db.DB, runtime []transport.Health) (Snapshot, error) {
	snap := Snapshot{
		GeneratedAt:              time.Now().UTC().Format(time.RFC3339),
		Bind:                     cfg.Bind.API,
		BindLocalOnly:            !cfg.Bind.AllowRemote,
		ConfiguredTransportModes: enabledModes(cfg),
		Transports:               make([]TransportReport, 0, len(cfg.Transports)),
	}
	if database != nil {
		schemaVersion, _ := database.SchemaVersion()
		snap.SchemaVersion = schemaVersion
		snap.LastSuccessfulIngest, _ = database.Scalar("SELECT COALESCE(MAX(rx_time), '') FROM messages;")
		snap.Messages = scalarInt(database, "SELECT COUNT(*) FROM messages;")
		snap.Nodes = scalarInt(database, "SELECT COUNT(*) FROM nodes;")
	}
	persisted, err := persistedEvidenceByTransport(database)
	if err != nil {
		return snap, err
	}
	runtimeMap := map[string]transport.Health{}
	for _, h := range runtime {
		runtimeMap[h.Name] = h
	}
	for _, tc := range cfg.Transports {
		h := runtimeMap[tc.Name]
		if h.Name == "" {
			built, err := transport.Build(tc, nil, nil)
			if err == nil {
				h = built.Health()
			} else {
				h = transport.Health{Name: tc.Name, Type: tc.Type, Source: tc.SourceLabel(), State: transport.StateError, Detail: err.Error()}
			}
		}
		evidence := persisted[tc.Name]
		report := TransportReport{
			Name:              tc.Name,
			Type:              tc.Type,
			Source:            tc.SourceLabel(),
			Enabled:           tc.Enabled,
			EffectiveState:    EffectiveState(tc.Enabled, h.State, evidence.Count),
			RuntimeState:      h.State,
			PersistedState:    persistedState(evidence.Count),
			StatusScope:       statusScope(h.State, evidence.Count),
			Detail:            detailFor(tc.Enabled, h, evidence),
			Guidance:          guidanceFor(tc.Enabled, EffectiveState(tc.Enabled, h.State, evidence.Count), tc.Type),
			LastAttemptAt:     h.LastAttemptAt,
			LastIngestAt:      firstNonEmpty(h.LastIngestAt, evidence.LastIngest),
			LastError:         h.LastError,
			TotalMessages:     h.TotalMessages,
			PersistedMessages: evidence.Count,
			ErrorCount:        h.ErrorCount,
			DroppedCount:      h.PacketsDropped,
			ReconnectAttempts: h.ReconnectAttempts,
			Capabilities:      h.Capabilities,
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
	if runtimeState == transport.StateIngesting || runtimeState == transport.StateConnectedNoIngest || runtimeState == transport.StateAttempting || runtimeState == transport.StateError {
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
	case transport.StateConfiguredNotAttempted:
		return "Start mel serve, then verify this transport transitions to ingesting only after messages are written to SQLite."
	case transport.StateAttempting:
		return "Verify endpoint reachability and credentials if this state does not clear quickly."
	case transport.StateConfiguredOffline:
		return "Check cable, host, port, topic, and node ownership; MEL is configured but not currently connected."
	case transport.StateConnectedNoIngest:
		return "Connection is up but no payload has been stored yet; generate real mesh traffic before trusting the path."
	case transport.StateIngesting:
		return "Live ingest is confirmed by successful database writes."
	case transport.StateHistoricalOnly:
		return "Past messages exist, but this command cannot prove current live connectivity."
	case transport.StateError:
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
