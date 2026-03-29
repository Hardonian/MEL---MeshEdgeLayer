package diagnostics

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/transport"
)

// Run is a simple entry point that runs all diagnostic checks with default thresholds
// This is suitable for API handlers that don't have access to runtime transport state
func Run(cfg config.Config, database *db.DB) []Finding {
	report := RunAllChecks(cfg, database, nil, nil, time.Now())
	return report.Diagnostics
}

// DiagnosticThresholds defines thresholds for various diagnostic checks
type DiagnosticThresholds struct {
	// Transport thresholds
	StaleHeartbeatSeconds         int64
	MaxConsecutiveTimeouts        uint64
	HighDeadLetterThreshold       uint64
	HighObservationDropsThreshold uint64

	// Mesh thresholds
	StaleNodeSeconds  int64
	SilentNodeSeconds int64

	// Database thresholds
	StaleWriteSeconds int64

	// Storage thresholds
	DiskPressurePercent uint64
}

// DefaultThresholds returns sensible defaults for diagnostic checks
func DefaultThresholds() DiagnosticThresholds {
	return DiagnosticThresholds{
		StaleHeartbeatSeconds:         120, // 2 minutes
		MaxConsecutiveTimeouts:        3,
		HighDeadLetterThreshold:       10,
		HighObservationDropsThreshold: 10,
		StaleNodeSeconds:              300,  // 5 minutes
		SilentNodeSeconds:             1800, // 30 minutes
		StaleWriteSeconds:             60,   // 1 minute
		DiskPressurePercent:           90,
	}
}

func maxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func parseRFC3339OrZero(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func pickNewerRFC3339(a, b string) string {
	ta, tb := parseRFC3339OrZero(a), parseRFC3339OrZero(b)
	switch {
	case ta.IsZero() && tb.IsZero():
		return ""
	case ta.IsZero():
		return b
	case tb.IsZero():
		return a
	case tb.After(ta):
		return b
	default:
		return a
	}
}

func pickNonEmpty(preferred, fallback string) string {
	if strings.TrimSpace(preferred) != "" {
		return preferred
	}
	return fallback
}

// runtimeTransportFromHealth maps in-process transport health into the shared TransportRuntime shape.
func runtimeTransportFromHealth(h transport.Health) db.TransportRuntime {
	return db.TransportRuntime{
		Name:                h.Name,
		Type:                h.Type,
		Source:              h.Source,
		State:               h.State,
		Detail:              h.Detail,
		LastAttemptAt:       h.LastAttemptAt,
		LastConnectedAt:     h.LastConnectedAt,
		LastSuccessAt:       h.LastSuccessAt,
		LastMessageAt:       h.LastIngestAt,
		LastHeartbeatAt:     h.LastHeartbeatAt,
		LastFailureAt:       h.LastFailureAt,
		LastObservationDrop: h.LastObservationDropAt,
		LastError:           h.LastError,
		EpisodeID:           h.EpisodeID,
		TotalMessages:       h.TotalMessages,
		PacketsDropped:      h.PacketsDropped,
		Reconnects:          h.ReconnectAttempts,
		Timeouts:            h.ConsecutiveTimeouts,
		FailureCount:        h.FailureCount,
		ObservationDrops:    h.ObservationDrops,
	}
}

// mergeTransportRuntimeEvidence prefers live connection fields from the process while taking the
// maximum of cumulative counters and the newest RFC3339 timestamps across DB and runtime.
func mergeTransportRuntimeEvidence(live, persisted db.TransportRuntime) db.TransportRuntime {
	out := persisted
	if strings.TrimSpace(live.Name) != "" {
		out.Name = live.Name
	}
	out.Type = pickNonEmpty(live.Type, out.Type)
	out.Source = pickNonEmpty(live.Source, out.Source)
	out.State = pickNonEmpty(live.State, out.State)
	out.Detail = pickNonEmpty(live.Detail, out.Detail)
	out.LastAttemptAt = pickNewerRFC3339(live.LastAttemptAt, out.LastAttemptAt)
	out.LastConnectedAt = pickNewerRFC3339(live.LastConnectedAt, out.LastConnectedAt)
	out.LastSuccessAt = pickNewerRFC3339(live.LastSuccessAt, out.LastSuccessAt)
	out.LastMessageAt = pickNewerRFC3339(live.LastMessageAt, out.LastMessageAt)
	out.LastHeartbeatAt = pickNewerRFC3339(live.LastHeartbeatAt, out.LastHeartbeatAt)
	out.LastFailureAt = pickNewerRFC3339(live.LastFailureAt, out.LastFailureAt)
	out.LastObservationDrop = pickNewerRFC3339(live.LastObservationDrop, out.LastObservationDrop)
	out.LastError = pickNonEmpty(live.LastError, out.LastError)
	out.EpisodeID = pickNonEmpty(live.EpisodeID, out.EpisodeID)
	out.TotalMessages = maxUint64(live.TotalMessages, out.TotalMessages)
	out.PacketsDropped = maxUint64(live.PacketsDropped, out.PacketsDropped)
	out.Reconnects = maxUint64(live.Reconnects, out.Reconnects)
	out.Timeouts = maxUint64(live.Timeouts, out.Timeouts)
	out.FailureCount = maxUint64(live.FailureCount, out.FailureCount)
	out.ObservationDrops = maxUint64(live.ObservationDrops, out.ObservationDrops)
	out.UpdatedAt = pickNewerRFC3339(live.UpdatedAt, out.UpdatedAt)
	if persisted.Name == "" && strings.TrimSpace(live.Name) != "" {
		out.Enabled = true
	} else {
		out.Enabled = persisted.Enabled
	}
	return out
}

// mergeResolvedTransportStates combines live transport.Health with persisted sqlite rows so checks
// see both process truth and cumulative counters (dead letters, observation drops, reconnects).
func mergeResolvedTransportStates(live []transport.Health, persisted []db.TransportRuntime) []db.TransportRuntime {
	byName := make(map[string]db.TransportRuntime)
	for _, p := range persisted {
		if strings.TrimSpace(p.Name) == "" {
			continue
		}
		byName[p.Name] = p
	}
	merged := make(map[string]db.TransportRuntime)
	order := make([]string, 0, len(byName)+len(live))
	seen := make(map[string]struct{})
	for _, h := range live {
		name := strings.TrimSpace(h.Name)
		if name == "" {
			continue
		}
		liveRow := runtimeTransportFromHealth(h)
		combined := mergeTransportRuntimeEvidence(liveRow, byName[name])
		merged[name] = combined
		seen[name] = struct{}{}
		order = append(order, name)
	}
	for name, p := range byName {
		if _, ok := seen[name]; ok {
			continue
		}
		merged[name] = p
		order = append(order, name)
	}
	sort.Strings(order)
	out := make([]db.TransportRuntime, 0, len(merged))
	for _, name := range order {
		out = append(out, merged[name])
	}
	return out
}

// RunAllChecks runs all diagnostic checks and returns a diagnostic report
func RunAllChecks(
	cfg config.Config,
	database *db.DB,
	runtimeTransports []transport.Health,
	transportStates []db.TransportRuntime,
	now time.Time,
) DiagnosticReport {
	thresholds := DefaultThresholds()
	diagnostics := []Diagnostic{}

	persisted := transportStates
	if len(persisted) == 0 && database != nil {
		if rows, err := database.TransportRuntimeStatuses(); err == nil {
			persisted = rows
		}
	}
	resolvedTransport := mergeResolvedTransportStates(runtimeTransports, persisted)

	// Run transport checks
	diagnostics = append(diagnostics, checkTransports(cfg, runtimeTransports, resolvedTransport, thresholds, now)...)

	// Run database checks
	diagnostics = append(diagnostics, checkDatabase(cfg, database, thresholds, now)...)

	// Run mesh checks
	diagnostics = append(diagnostics, checkMesh(cfg, database, thresholds, now)...)

	// Run config checks
	diagnostics = append(diagnostics, checkConfig(cfg)...)

	// Run control checks
	diagnostics = append(diagnostics, checkControl(cfg)...)

	// Run storage checks
	diagnostics = append(diagnostics, checkStorage(cfg)...)

	// Build summary
	summary := buildSummary(diagnostics)

	// Build raw evidence
	rawEvidence := map[string]any{
		"thresholds":      thresholds,
		"config_mode":     cfg.Control.Mode,
		"transport_count": len(resolvedTransport),
	}

	return DiagnosticReport{
		GeneratedAt: now,
		Summary:     summary,
		Diagnostics: diagnostics,
		RawEvidence: rawEvidence,
	}
}

// Using types from diagnostics.go: Diagnostic, DiagnosticReport, Summary

// checkTransports runs all transport-related diagnostic checks
func checkTransports(
	cfg config.Config,
	runtimeTransports []transport.Health,
	transportStates []db.TransportRuntime,
	thresholds DiagnosticThresholds,
	now time.Time,
) []Diagnostic {
	diagnostics := []Diagnostic{}

	runtimeMap := make(map[string]transport.Health)
	for _, t := range runtimeTransports {
		runtimeMap[t.Name] = t
	}

	// Index transport states by name
	stateMap := make(map[string]db.TransportRuntime)
	for _, t := range transportStates {
		stateMap[t.Name] = t
	}

	// Get enabled transports
	enabledTransports := make(map[string]config.TransportConfig)
	for _, t := range cfg.Transports {
		if t.Enabled {
			enabledTransports[t.Name] = t
		}
	}

	// Check each configured transport
	for name, tcfg := range enabledTransports {
		runtime, hasRuntime := runtimeMap[name]
		state, hasState := stateMap[name]

		// Skip if no runtime data (transport not initialized yet)
		if !hasRuntime && !hasState {
			continue
		}

		var stateStr string
		var lastHeartbeat time.Time
		var timeouts, failures, deadLetters, obsDrops uint64
		var lastError string
		var episodeID string

		if hasState {
			stateStr = state.State
			if state.LastHeartbeatAt != "" {
				lastHeartbeat, _ = time.Parse(time.RFC3339, state.LastHeartbeatAt)
			}
			timeouts = state.Timeouts
			failures = state.FailureCount
			deadLetters = state.PacketsDropped
			obsDrops = state.ObservationDrops
			lastError = state.LastError
			episodeID = state.EpisodeID
		} else if hasRuntime {
			stateStr = runtime.State
			if runtime.LastHeartbeatAt != "" {
				lastHeartbeat, _ = time.Parse(time.RFC3339, runtime.LastHeartbeatAt)
			}
			timeouts = runtime.ConsecutiveTimeouts
			failures = runtime.FailureCount
			deadLetters = runtime.PacketsDropped
			obsDrops = runtime.ObservationDrops
			lastError = runtime.LastError
			episodeID = runtime.EpisodeID
		}

		heartbeatAge := now.Sub(lastHeartbeat).Seconds()

		// Check: transport disconnected/failed
		if stateStr == transport.StateFailed {
			diagnostics = append(diagnostics, Diagnostic{
				Code:                   "transport_failed",
				Severity:               SeverityCritical,
				Component:              ComponentTransport,
				Title:                  fmt.Sprintf("Transport %s is in failed state", name),
				Explanation:            fmt.Sprintf("Transport %s (%s) has entered failed state", name, tcfg.Type),
				LikelyCauses:           []string{"Hardware connection lost", "Network unreachable", "Configuration error", "Serial device not available"},
				RecommendedSteps:       []string{"Check physical connection", "Verify device is powered on", "Check configuration matches hardware", "Review last error: " + lastError},
				Evidence:               map[string]any{"state": stateStr, "last_error": lastError, "type": tcfg.Type},
				CanAutoRecover:         true,
				OperatorActionRequired: true,
				AffectedTransport:      name,
				GeneratedAt:            now.Format(time.RFC3339),
			})
			continue
		}

		// Check: transport reconnecting
		if stateStr == transport.StateRetrying {
			diagnostics = append(diagnostics, Diagnostic{
				Code:                   "transport_reconnecting",
				Severity:               SeverityWarning,
				Component:              ComponentTransport,
				Title:                  fmt.Sprintf("Transport %s is reconnecting", name),
				Explanation:            fmt.Sprintf("Transport %s is attempting to reconnect after failures", name),
				LikelyCauses:           []string{"Network instability", "Broker unavailable", "Serial port issues", "Authentication failure"},
				RecommendedSteps:       []string{"Wait for reconnection", "Check network connectivity", "Verify broker/device availability", "Review failure count: " + strconv.FormatUint(failures, 10)},
				Evidence:               map[string]any{"state": stateStr, "failure_count": failures, "episode_id": episodeID},
				CanAutoRecover:         true,
				OperatorActionRequired: false,
				AffectedTransport:      name,
				GeneratedAt:            now.Format(time.RFC3339),
			})
		}

		// Check: stale heartbeat
		if heartbeatAge > float64(thresholds.StaleHeartbeatSeconds) && !lastHeartbeat.IsZero() {
			diagnostics = append(diagnostics, Diagnostic{
				Code:                   "transport_stale_heartbeat",
				Severity:               SeverityWarning,
				Component:              ComponentTransport,
				Title:                  fmt.Sprintf("Transport %s has stale heartbeat", name),
				Explanation:            fmt.Sprintf("No heartbeat received for %.0f seconds (threshold: %ds)", heartbeatAge, thresholds.StaleHeartbeatSeconds),
				LikelyCauses:           []string{"Transport not receiving data", "Network latency", "Device idle", "Connection silently dropped"},
				RecommendedSteps:       []string{"Verify transport is still running", "Check for network issues", "Review device status", "Consider restart if stale persists"},
				Evidence:               map[string]any{"last_heartbeat": lastHeartbeat.Format(time.RFC3339), "age_seconds": heartbeatAge},
				CanAutoRecover:         true,
				OperatorActionRequired: heartbeatAge > float64(thresholds.StaleHeartbeatSeconds)*3,
				AffectedTransport:      name,
				GeneratedAt:            now.Format(time.RFC3339),
			})
		}

		// Check: high timeouts
		if timeouts >= thresholds.MaxConsecutiveTimeouts {
			diagnostics = append(diagnostics, Diagnostic{
				Code:                   "transport_high_timeouts",
				Severity:               SeverityWarning,
				Component:              ComponentTransport,
				Title:                  fmt.Sprintf("Transport %s has high timeout count", name),
				Explanation:            fmt.Sprintf("Consecutive timeouts: %d (threshold: %d)", timeouts, thresholds.MaxConsecutiveTimeouts),
				LikelyCauses:           []string{"Device not responding", "Baud rate mismatch", "Flow control issue", "Cable quality"},
				RecommendedSteps:       []string{"Check device is responsive", "Verify serial/TCP settings", "Try different cable", "Check device logs"},
				Evidence:               map[string]any{"consecutive_timeouts": timeouts, "type": tcfg.Type},
				CanAutoRecover:         true,
				OperatorActionRequired: true,
				AffectedTransport:      name,
				GeneratedAt:            now.Format(time.RFC3339),
			})
		}

		// Check: dead letter accumulation
		if deadLetters >= thresholds.HighDeadLetterThreshold {
			diagnostics = append(diagnostics, Diagnostic{
				Code:                   "transport_dead_letter_accumulation",
				Severity:               SeverityWarning,
				Component:              ComponentTransport,
				Title:                  fmt.Sprintf("Transport %s has accumulated dead letters", name),
				Explanation:            fmt.Sprintf("Dead letter count: %d (threshold: %d)", deadLetters, thresholds.HighDeadLetterThreshold),
				LikelyCauses:           []string{"Message parsing failures", "Schema incompatibility", "Topic mismatch", "Invalid payload format"},
				RecommendedSteps:       []string{"Check dead letter details in /dead-letters", "Review message formats being received", "Verify topic configuration"},
				Evidence:               map[string]any{"dead_letter_count": deadLetters, "type": tcfg.Type},
				CanAutoRecover:         true,
				OperatorActionRequired: false,
				AffectedTransport:      name,
				GeneratedAt:            now.Format(time.RFC3339),
			})
		}

		// Check: observation drops
		if obsDrops >= thresholds.HighObservationDropsThreshold {
			diagnostics = append(diagnostics, Diagnostic{
				Code:                   "transport_observation_drops",
				Severity:               SeverityWarning,
				Component:              ComponentTransport,
				Title:                  fmt.Sprintf("Transport %s has observation drops", name),
				Explanation:            fmt.Sprintf("Observation drops: %d (threshold: %d)", obsDrops, thresholds.HighObservationDropsThreshold),
				LikelyCauses:           []string{"Processing backlog", "Memory pressure", "Rate limiting", "Invalid observations"},
				RecommendedSteps:       []string{"Check system resources", "Review processing latency", "Consider reducing message rate"},
				Evidence:               map[string]any{"observation_drops": obsDrops},
				CanAutoRecover:         true,
				OperatorActionRequired: false,
				AffectedTransport:      name,
				GeneratedAt:            now.Format(time.RFC3339),
			})
		}
	}

	// Check for configured but not initialized transports
	for name, tcfg := range enabledTransports {
		if _, exists := runtimeMap[name]; !exists {
			if _, stateExists := stateMap[name]; !stateExists {
				diagnostics = append(diagnostics, Diagnostic{
					Code:                   "transport_not_initialized",
					Severity:               SeverityInfo,
					Component:              ComponentTransport,
					Title:                  fmt.Sprintf("Transport %s not initialized", name),
					Explanation:            fmt.Sprintf("Transport %s (%s) is configured but has not reported any runtime state", name, tcfg.Type),
					LikelyCauses:           []string{"Transport disabled at runtime", "Failed to initialize", "Configuration error"},
					RecommendedSteps:       []string{"Verify transport configuration", "Check MEL startup logs", "Ensure device/endpoint is accessible"},
					Evidence:               map[string]any{"type": tcfg.Type, "configured": true, "initialized": false},
					CanAutoRecover:         true,
					OperatorActionRequired: true,
					AffectedTransport:      name,
					GeneratedAt:            now.Format(time.RFC3339),
				})
			}
		}
	}

	return diagnostics
}

// checkDatabase runs database-related diagnostic checks
func checkDatabase(
	cfg config.Config,
	database *db.DB,
	thresholds DiagnosticThresholds,
	now time.Time,
) []Diagnostic {
	diagnostics := []Diagnostic{}

	if database == nil {
		diagnostics = append(diagnostics, Diagnostic{
			Code:                   "database_unavailable",
			Severity:               SeverityCritical,
			Component:              ComponentDatabase,
			Title:                  "Database is not available",
			Explanation:            "Database connection is nil - MEL may be running in read-only or degraded mode",
			LikelyCauses:           []string{"Database file missing", "Permission denied", "Database locked", "Storage failure"},
			RecommendedSteps:       []string{"Check database file exists", "Verify file permissions", "Check storage availability", "Review MEL logs"},
			Evidence:               map[string]any{"database_path": cfg.Storage.DatabasePath},
			CanAutoRecover:         false,
			OperatorActionRequired: true,
			GeneratedAt:            now.Format(time.RFC3339),
		})
		return diagnostics
	}

	// Check database accessibility by trying to read
	_, err := database.TransportRuntimeStatuses()
	if err != nil {
		diagnostics = append(diagnostics, Diagnostic{
			Code:                   "database_unreachable",
			Severity:               SeverityCritical,
			Component:              ComponentDatabase,
			Title:                  "Database is not accessible",
			Explanation:            fmt.Sprintf("Failed to query database: %v", err),
			LikelyCauses:           []string{"Database file corrupted", "Permission denied", "Disk I/O error", "Schema mismatch"},
			RecommendedSteps:       []string{"Check file permissions", "Run database integrity check", "Verify disk health", "Consider backup restore"},
			Evidence:               map[string]any{"error": err.Error(), "database_path": cfg.Storage.DatabasePath},
			CanAutoRecover:         false,
			OperatorActionRequired: true,
			GeneratedAt:            now.Format(time.RFC3339),
		})
		return diagnostics
	}

	// Get basic database stats using Scalar
	messageCount, _ := database.Scalar("SELECT COUNT(*) FROM messages;")
	nodeCount, _ := database.Scalar("SELECT COUNT(*) FROM nodes;")
	deadLetterCount, _ := database.Scalar("SELECT COUNT(*) FROM dead_letters;")
	lastWrite, _ := database.Scalar("SELECT COALESCE(MAX(rx_time), '') FROM messages;")

	msgCount, _ := strconv.ParseInt(messageCount, 10, 64)
	nodeCnt, _ := strconv.ParseInt(nodeCount, 10, 64)
	dlCount, _ := strconv.ParseInt(deadLetterCount, 10, 64)

	// Check for stale writes
	if lastWrite != "" {
		lastWriteTime, err := time.Parse(time.RFC3339, lastWrite)
		if err == nil {
			writeAge := now.Sub(lastWriteTime).Seconds()
			if writeAge > float64(thresholds.StaleWriteSeconds) {
				diagnostics = append(diagnostics, Diagnostic{
					Code:                   "database_stale_writes",
					Severity:               SeverityWarning,
					Component:              ComponentDatabase,
					Title:                  "Database writes may be stalled",
					Explanation:            fmt.Sprintf("Last write was %.0f seconds ago (threshold: %ds)", writeAge, thresholds.StaleWriteSeconds),
					LikelyCauses:           []string{"Ingest pipeline stalled", "Database locked", "Disk I/O bottleneck", "No new messages"},
					RecommendedSteps:       []string{"Check transport status", "Verify disk I/O", "Review MEL logs for errors"},
					Evidence:               map[string]any{"last_write": lastWrite, "age_seconds": writeAge, "message_count": msgCount},
					CanAutoRecover:         true,
					OperatorActionRequired: false,
					GeneratedAt:            now.Format(time.RFC3339),
				})
			}
		}
	}

	// Check for high dead letter accumulation
	if dlCount >= 50 {
		diagnostics = append(diagnostics, Diagnostic{
			Code:                   "high_dead_letter_accumulation",
			Severity:               SeverityWarning,
			Component:              ComponentDatabase,
			Title:                  "High dead letter accumulation",
			Explanation:            fmt.Sprintf("Total dead letters in database: %d", dlCount),
			LikelyCauses:           []string{"Multiple transports with parsing failures", "Schema incompatibility", "Corrupted messages"},
			RecommendedSteps:       []string{"Review dead letters in UI", "Check transport configurations", "Investigate message sources"},
			Evidence:               map[string]any{"dead_letter_count": dlCount, "message_count": msgCount, "node_count": nodeCnt},
			CanAutoRecover:         true,
			OperatorActionRequired: false,
			GeneratedAt:            now.Format(time.RFC3339),
		})
	}

	return diagnostics
}

// checkMesh runs mesh-related diagnostic checks
func checkMesh(
	cfg config.Config,
	database *db.DB,
	thresholds DiagnosticThresholds,
	now time.Time,
) []Diagnostic {
	diagnostics := []Diagnostic{}

	if database == nil {
		return diagnostics
	}

	// Get node data directly from database
	nodeRows, err := database.QueryRows("SELECT node_num, node_id, long_name, last_seen FROM nodes;")
	if err != nil {
		return diagnostics
	}

	if len(nodeRows) == 0 {
		diagnostics = append(diagnostics, Diagnostic{
			Code:                   "mesh_no_nodes",
			Severity:               SeverityWarning,
			Component:              ComponentMesh,
			Title:                  "No mesh nodes detected",
			Explanation:            "No nodes have been observed from any transport",
			LikelyCauses:           []string{"No active mesh devices", "Transports not receiving data", "Wrong topic configuration", "Mesh is out of range"},
			RecommendedSteps:       []string{"Verify mesh devices are transmitting", "Check transport status", "Review topic configuration for MQTT", "Check serial/TCP connection"},
			Evidence:               map[string]any{"node_count": 0},
			CanAutoRecover:         true,
			OperatorActionRequired: true,
			GeneratedAt:            now.Format(time.RFC3339),
		})
		return diagnostics
	}

	// Check for stale/silent nodes
	staleCount := 0
	silentCount := 0

	for _, row := range nodeRows {
		lastSeen := row["last_seen"]
		if lastSeen == "" {
			continue
		}

		lastSeenStr, ok := lastSeen.(string)
		if !ok {
			continue // skip invalid type
		}
		lastSeenTime, err := time.Parse(time.RFC3339, lastSeenStr)
		if err != nil {
			continue
		}

		age := now.Sub(lastSeenTime).Seconds()

		if age > float64(thresholds.StaleNodeSeconds) {
			staleCount++
		}

		if age > float64(thresholds.SilentNodeSeconds) {
			silentCount++
		}
	}

	// Check: stale snapshot (no recent updates at all)
	lastMsgTime, _ := database.Scalar("SELECT COALESCE(MAX(rx_time), '') FROM messages;")
	if lastMsgTime != "" {
		lastMsg, err := time.Parse(time.RFC3339, lastMsgTime)
		if err == nil && now.Sub(lastMsg).Seconds() > float64(thresholds.StaleNodeSeconds) {
			diagnostics = append(diagnostics, Diagnostic{
				Code:                   "mesh_stale_snapshot",
				Severity:               SeverityWarning,
				Component:              ComponentMesh,
				Title:                  "Mesh snapshot is stale",
				Explanation:            fmt.Sprintf("No mesh updates received in >%d seconds", thresholds.StaleNodeSeconds),
				LikelyCauses:           []string{"No new mesh messages", "Transports stalled", "Mesh devices idle"},
				RecommendedSteps:       []string{"Check transport status", "Verify mesh devices are active", "Review connectivity"},
				Evidence:               map[string]any{"last_message": lastMsgTime, "node_count": len(nodeRows)},
				CanAutoRecover:         true,
				OperatorActionRequired: false,
				GeneratedAt:            now.Format(time.RFC3339),
			})
		}
	}

	// Check: partial data (some stale nodes but not all)
	if staleCount > 0 && staleCount < len(nodeRows) {
		diagnostics = append(diagnostics, Diagnostic{
			Code:                   "mesh_partial_connectivity",
			Severity:               SeverityInfo,
			Component:              ComponentMesh,
			Title:                  "Partial mesh connectivity",
			Explanation:            fmt.Sprintf("%d of %d nodes are stale (>%ds)", staleCount, len(nodeRows), thresholds.StaleNodeSeconds),
			LikelyCauses:           []string{"Some nodes out of range", "Intermittent connectivity", "Node sleep cycles"},
			RecommendedSteps:       []string{"Monitor for patterns", "Check node power status", "Review mesh topology"},
			Evidence:               map[string]any{"total_nodes": len(nodeRows), "stale_nodes": staleCount, "silent_nodes": silentCount},
			CanAutoRecover:         true,
			OperatorActionRequired: false,
			GeneratedAt:            now.Format(time.RFC3339),
		})
	}

	// Check: silent nodes
	if silentCount > 0 {
		diagnostics = append(diagnostics, Diagnostic{
			Code:                   "mesh_node_silence",
			Severity:               SeverityWarning,
			Component:              ComponentMesh,
			Title:                  fmt.Sprintf("%d node(s) have gone silent", silentCount),
			Explanation:            fmt.Sprintf("%d nodes have not been heard from in >%d seconds", silentCount, thresholds.SilentNodeSeconds),
			LikelyCauses:           []string{"Node powered off", "Node out of range", "Node malfunction", "Antenna issue"},
			RecommendedSteps:       []string{"Verify node status physically", "Check node power", "Review node locations"},
			Evidence:               map[string]any{"silent_node_count": silentCount, "total_nodes": len(nodeRows), "threshold_seconds": thresholds.SilentNodeSeconds},
			CanAutoRecover:         false,
			OperatorActionRequired: true,
			GeneratedAt:            now.Format(time.RFC3339),
		})
	}

	return diagnostics
}

// checkConfig runs configuration-related diagnostic checks
func checkConfig(cfg config.Config) []Diagnostic {
	diagnostics := []Diagnostic{}

	// Check for unsafe configurations using existing validation
	violations := config.ValidateSafeDefaults(cfg)

	if len(violations) > 0 {
		for _, v := range violations {
			severity := SeverityWarning
			if strings.Contains(v.Issue, "remote bind without") || strings.Contains(v.Issue, "insecure") {
				severity = SeverityCritical
			}

			diagnostics = append(diagnostics, Diagnostic{
				Code:                   "config_unsafe_setting",
				Severity:               severity,
				Component:              ComponentConfig,
				Title:                  fmt.Sprintf("Unsafe configuration: %s", v.Field),
				Explanation:            fmt.Sprintf("%s: %s", v.Issue, v.Current),
				LikelyCauses:           []string{"Configuration drift", "Default values not changed", "Testing configuration left in place"},
				RecommendedSteps:       []string{"Update to safe value: " + v.Safe, "Review security configuration guide"},
				Evidence:               map[string]any{"field": v.Field, "current": v.Current, "safe": v.Safe, "issue": v.Issue},
				CanAutoRecover:         false,
				OperatorActionRequired: true,
				GeneratedAt:            time.Now().Format(time.RFC3339),
			})
		}
	}

	// Check for missing required transports
	if len(cfg.Transports) == 0 {
		diagnostics = append(diagnostics, Diagnostic{
			Code:                   "config_no_transports",
			Severity:               SeverityWarning,
			Component:              ComponentConfig,
			Title:                  "No transports configured",
			Explanation:            "MEL has no transport configuration - it will not ingest any data",
			LikelyCauses:           []string{"Initial configuration incomplete", "Configuration not loaded"},
			RecommendedSteps:       []string{"Configure at least one transport (serial, TCP, or MQTT)", "See config example: configs/mel.example.json"},
			Evidence:               map[string]any{"transport_count": 0},
			CanAutoRecover:         false,
			OperatorActionRequired: true,
			GeneratedAt:            time.Now().Format(time.RFC3339),
		})
	}

	// Check for enabled transports with missing required fields
	for _, t := range cfg.Transports {
		if !t.Enabled {
			continue
		}

		switch t.Type {
		case "serial":
			if t.SerialDevice == "" {
				diagnostics = append(diagnostics, Diagnostic{
					Code:                   "config_missing_serial_device",
					Severity:               SeverityCritical,
					Component:              ComponentConfig,
					Title:                  fmt.Sprintf("Transport %s missing required field", t.Name),
					Explanation:            "Serial transport enabled but serial_device not configured",
					LikelyCauses:           []string{"Incomplete transport configuration"},
					RecommendedSteps:       []string{"Set serial_device in configuration", "Disable transport if not using serial"},
					Evidence:               map[string]any{"transport": t.Name, "type": t.Type, "missing_field": "serial_device"},
					CanAutoRecover:         false,
					OperatorActionRequired: true,
					GeneratedAt:            time.Now().Format(time.RFC3339),
				})
			}
		case "tcp", "serialtcp":
			if t.TCPHost == "" || t.TCPPort == 0 {
				diagnostics = append(diagnostics, Diagnostic{
					Code:                   "config_missing_tcp_fields",
					Severity:               SeverityCritical,
					Component:              ComponentConfig,
					Title:                  fmt.Sprintf("Transport %s missing required fields", t.Name),
					Explanation:            "TCP transport enabled but tcp_host or tcp_port not configured",
					LikelyCauses:           []string{"Incomplete transport configuration"},
					RecommendedSteps:       []string{"Set tcp_host and tcp_port in configuration", "Disable transport if not using TCP"},
					Evidence:               map[string]any{"transport": t.Name, "type": t.Type, "missing_fields": []string{"tcp_host", "tcp_port"}},
					CanAutoRecover:         false,
					OperatorActionRequired: true,
					GeneratedAt:            time.Now().Format(time.RFC3339),
				})
			}
		case "mqtt":
			if t.Endpoint == "" {
				diagnostics = append(diagnostics, Diagnostic{
					Code:                   "config_missing_mqtt_endpoint",
					Severity:               SeverityCritical,
					Component:              ComponentConfig,
					Title:                  fmt.Sprintf("Transport %s missing required field", t.Name),
					Explanation:            "MQTT transport enabled but endpoint not configured",
					LikelyCauses:           []string{"Incomplete transport configuration"},
					RecommendedSteps:       []string{"Set endpoint (host:port) in configuration", "Disable transport if not using MQTT"},
					Evidence:               map[string]any{"transport": t.Name, "type": t.Type, "missing_field": "endpoint"},
					CanAutoRecover:         false,
					OperatorActionRequired: true,
					GeneratedAt:            time.Now().Format(time.RFC3339),
				})
			}
		}
	}

	return diagnostics
}

// checkControl runs control-plane diagnostic checks
func checkControl(cfg config.Config) []Diagnostic {
	diagnostics := []Diagnostic{}

	// Check: control mode unsafe
	if cfg.Control.Mode == "guarded_auto" {
		// Check if safeguards are configured
		hasSafeguards := cfg.Control.MaxQueue > 0 ||
			cfg.Control.MaxActionsPerWindow > 0 ||
			cfg.Control.CooldownPerTargetSeconds > 0

		if !hasSafeguards {
			diagnostics = append(diagnostics, Diagnostic{
				Code:                   "control_no_safeguards",
				Severity:               SeverityCritical,
				Component:              ComponentControl,
				Title:                  "Control mode enabled without safeguards",
				Explanation:            "guarded_auto mode is enabled but no safety limits are configured",
				LikelyCauses:           []string{"Default configuration left unchanged", "Safeguard limits not set"},
				RecommendedSteps:       []string{"Configure control.max_queue", "Configure control.max_actions_per_window", "Configure control.cooldown_per_target_seconds"},
				Evidence:               map[string]any{"mode": cfg.Control.Mode},
				CanAutoRecover:         false,
				OperatorActionRequired: true,
				GeneratedAt:            time.Now().Format(time.RFC3339),
			})
		}

		// Check for overly permissive settings
		if cfg.Control.MaxQueue > 100 {
			diagnostics = append(diagnostics, Diagnostic{
				Code:                   "control_queue_limit_high",
				Severity:               SeverityWarning,
				Component:              ComponentControl,
				Title:                  "Control queue limit too high",
				Explanation:            fmt.Sprintf("Max queue set to %d, recommended <= 64", cfg.Control.MaxQueue),
				LikelyCauses:           []string{"Default configuration not reviewed"},
				RecommendedSteps:       []string{"Reduce control.max_queue to <= 64"},
				Evidence:               map[string]any{"max_queue": cfg.Control.MaxQueue},
				CanAutoRecover:         false,
				OperatorActionRequired: true,
				GeneratedAt:            time.Now().Format(time.RFC3339),
			})
		}
	}

	// Check for disabled retention when control is active
	if cfg.Control.Mode != "disabled" && !cfg.Retention.Enabled {
		diagnostics = append(diagnostics, Diagnostic{
			Code:                   "retention_disabled_with_control",
			Severity:               SeverityWarning,
			Component:              ComponentRetention,
			Title:                  "Control active with retention disabled",
			Explanation:            "Control plane is active but retention is disabled - action history will be lost",
			LikelyCauses:           []string{"Retention disabled for testing", "Configuration error"},
			RecommendedSteps:       []string{"Enable retention.enabled", "Configure appropriate retention periods"},
			Evidence:               map[string]any{"control_mode": cfg.Control.Mode, "retention_enabled": false},
			CanAutoRecover:         false,
			OperatorActionRequired: true,
			GeneratedAt:            time.Now().Format(time.RFC3339),
		})
	}

	return diagnostics
}

// checkStorage runs storage-related diagnostic checks
func checkStorage(cfg config.Config) []Diagnostic {
	diagnostics := []Diagnostic{}

	// Check: storage path missing
	if cfg.Storage.DataDir == "" && cfg.Storage.DatabasePath == "" {
		diagnostics = append(diagnostics, Diagnostic{
			Code:                   "storage_no_path",
			Severity:               SeverityCritical,
			Component:              ComponentStorage,
			Title:                  "No storage path configured",
			Explanation:            "Neither data_dir nor database_path is configured",
			LikelyCauses:           []string{"Configuration incomplete", "Default values not set"},
			RecommendedSteps:       []string{"Set storage.data_dir or storage.database_path"},
			Evidence:               map[string]any{},
			CanAutoRecover:         false,
			OperatorActionRequired: true,
			GeneratedAt:            time.Now().Format(time.RFC3339),
		})
		return diagnostics
	}

	// Check if database path directory exists and is writable
	if cfg.Storage.DatabasePath != "" {
		dir := filepath.Dir(cfg.Storage.DatabasePath)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			// Directory exists, check if we can write
			testFile := filepath.Join(dir, ".mel-write-test")
			f, err := os.Create(testFile)
			if err != nil {
				diagnostics = append(diagnostics, Diagnostic{
					Code:                   "storage_not_writable",
					Severity:               SeverityCritical,
					Component:              ComponentStorage,
					Title:                  "Storage directory not writable",
					Explanation:            fmt.Sprintf("Cannot write to storage directory: %s", dir),
					LikelyCauses:           []string{"Permission denied", "Filesystem read-only"},
					RecommendedSteps:       []string{"Fix directory permissions", "Check filesystem is writable"},
					Evidence:               map[string]any{"directory": dir, "error": err.Error()},
					CanAutoRecover:         false,
					OperatorActionRequired: true,
					GeneratedAt:            time.Now().Format(time.RFC3339),
				})
			} else {
				f.Close()
				os.Remove(testFile)
			}
		}
	}

	return diagnostics
}

// buildSummary creates a summary of diagnostic counts
func buildSummary(diagnostics []Diagnostic) Summary {
	summary := Summary{
		TotalCount: len(diagnostics),
	}

	for _, d := range diagnostics {
		switch d.Severity {
		case SeverityCritical:
			summary.CriticalCount++
		case SeverityWarning:
			summary.WarningCount++
		case SeverityInfo:
			summary.InfoCount++
		}

		if d.CanAutoRecover {
			summary.CanAutoRecover++
		}

		if d.OperatorActionRequired {
			summary.NeedsOperator++
		}
	}

	return summary
}
