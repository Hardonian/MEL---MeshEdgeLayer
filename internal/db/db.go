package db

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/auth"
	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/logging"
	"github.com/mel-project/mel/internal/models"
)

const (
	MaxRows          = 10000
	MaxFieldLength   = 4096
	MaxIdentifierLen = 128
)

type DB struct{ Path string }

type TransportRuntime struct {
	Name                string `json:"name"`
	Type                string `json:"type"`
	Source              string `json:"source"`
	Enabled             bool   `json:"enabled"`
	State               string `json:"state"`
	Detail              string `json:"detail"`
	LastAttemptAt       string `json:"last_attempt_at,omitempty"`
	LastConnectedAt     string `json:"last_connected_at,omitempty"`
	LastSuccessAt       string `json:"last_success_at,omitempty"`
	LastMessageAt       string `json:"last_message_at,omitempty"`
	LastHeartbeatAt     string `json:"last_heartbeat_at,omitempty"`
	LastFailureAt       string `json:"last_failure_at,omitempty"`
	LastObservationDrop string `json:"last_observation_drop_at,omitempty"`
	LastError           string `json:"last_error,omitempty"`
	EpisodeID           string `json:"episode_id,omitempty"`
	TotalMessages       uint64 `json:"total_messages"`
	PacketsDropped      uint64 `json:"packets_dropped"`
	Reconnects          uint64 `json:"reconnect_attempts"`
	Timeouts            uint64 `json:"consecutive_timeouts"`
	FailureCount        uint64 `json:"failure_count"`
	ObservationDrops    uint64 `json:"observation_drops"`
	UpdatedAt           string `json:"updated_at,omitempty"`
}

type DeadLetter struct {
	TransportName string         `json:"transport_name"`
	TransportType string         `json:"transport_type"`
	Topic         string         `json:"topic"`
	Reason        string         `json:"reason"`
	PayloadHex    string         `json:"payload_hex"`
	Details       map[string]any `json:"details,omitempty"`
}

func Open(cfg config.Config) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.Storage.DatabasePath), 0o755); err != nil {
		return nil, err
	}
	db := &DB{Path: cfg.Storage.DatabasePath}
	if err := db.ApplyMigrations(migrationDir()); err != nil {
		return nil, err
	}
	return db, nil
}

func (d *DB) ApplyMigrations(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	applied, err := d.appliedMigrations()
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		version := strings.TrimSuffix(name, filepath.Ext(name))
		if applied[version] {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		cmd := exec.Command("sqlite3", "-cmd", ".timeout 5000", d.Path)
		cmd.Stdin = strings.NewReader(string(b))
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("sqlite3 migrate %s: %w: %s", name, err, out)
		}
	}
	return nil
}

func (d *DB) appliedMigrations() (map[string]bool, error) {
	out := map[string]bool{}
	rows, err := d.QueryRows("SELECT name FROM sqlite_master WHERE type='table' AND name='schema_migrations';")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return out, nil
	}
	rows, err = d.QueryRows("SELECT version FROM schema_migrations;")
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[asString(row["version"])] = true
	}
	return out, nil
}

func (d *DB) Exec(sql string) error {
	cmd := exec.Command("sqlite3", "-cmd", ".timeout 5000", d.Path, sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return logging.NewSafeError("database operation failed", fmt.Errorf("sqlite exec failed: %w: %s", err, out), "database", isTransientDBError(err))
	}
	return nil
}

func (d *DB) QueryRows(sql string) ([]map[string]any, error) {
	cmd := exec.Command("sqlite3", "-cmd", ".timeout 5000", "-json", d.Path, sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, logging.NewSafeError("database query failed", fmt.Errorf("sqlite query failed: %w: %s", err, out), "database", isTransientDBError(err))
	}
	rows := make([]map[string]any, 0)
	if len(out) == 0 {
		return rows, nil
	}
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil, logging.NewSafeError("failed to parse query results", err, "database", false)
	}
	return rows, nil
}

func (d *DB) QueryJSON(sql string) ([]map[string]string, error) {
	rows, err := d.QueryRows(sql)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		converted := map[string]string{}
		for k, v := range row {
			converted[k] = fmt.Sprint(v)
		}
		out = append(out, converted)
	}
	return out, nil
}

func (d *DB) Scalar(sql string) (string, error) {
	rows, err := d.QueryRows(sql)
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", nil
	}
	for _, v := range rows[0] {
		return fmt.Sprint(v), nil
	}
	return "", nil
}

func (d *DB) SchemaVersion() (string, error) {
	return d.Scalar("SELECT version FROM schema_migrations ORDER BY applied_at DESC, version DESC LIMIT 1;")
}

var safeIdentifierRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
var safeNodeIDRegex = regexp.MustCompile(`^!?[a-zA-Z0-9]{4,12}$`)

// IsSafeIdentifier returns true if the input string is a valid, safe SQL identifier
// (alphanumeric and underscore only, limited length).
func IsSafeIdentifier(v string) bool {
	if v == "" || len(v) > MaxIdentifierLen {
		return false
	}
	return safeIdentifierRegex.MatchString(v)
}

// IsSafeNodeID returns true if the input is a valid Meshtastic Node ID.
func IsSafeNodeID(v string) bool {
	if v == "" || len(v) > 16 {
		return false
	}
	return safeNodeIDRegex.MatchString(v)
}

type SQLValidationError struct {
	Input   string
	Reason  string
	Blocked bool
}

func (e SQLValidationError) Error() string {
	return fmt.Sprintf("SQL validation failed: %s (input length: %d)", e.Reason, len(e.Input))
}

// ValidateSQLInput provides a baseline for sanitizing strings meant for SQL queries.
// It returns a string escaped for SQLite single-quote context or an error if the input
// contains suspicious characters like null bytes or comment sequences.
func ValidateSQLInput(v string) (string, error) {
	if len(v) > MaxFieldLength {
		return "", SQLValidationError{Input: v, Reason: fmt.Sprintf("input exceeds max length of %d", MaxFieldLength), Blocked: true}
	}
	if strings.ContainsRune(v, 0x00) {
		return "", SQLValidationError{Input: v, Reason: "input contains NULL byte", Blocked: true}
	}
	// Common SQL injection markers
	if strings.Contains(v, "--") {
		return "", SQLValidationError{Input: v, Reason: "input contains SQL comment marker (--)", Blocked: true}
	}
	if strings.Contains(v, "/*") || strings.Contains(v, "*/") {
		return "", SQLValidationError{Input: v, Reason: "input contains block comment marker (/* or */)", Blocked: true}
	}
	if strings.Contains(v, ";") {
		return "", SQLValidationError{Input: v, Reason: "input contains statement terminator (;)", Blocked: true}
	}
	// Basic single quote escaping for SQLite
	return strings.ReplaceAll(v, "'", "''"), nil
}

// EscString returns a sanitized version of the string for use in SQL single quotes.
// If validation fails, it returns an empty string and logs a warning.
func EscString(v string) string {
	sanitized, err := ValidateSQLInput(v)
	if err != nil {
		logSuspiciousSQL(v, err.Error())
		return ""
	}
	return sanitized
}

// EscIdentifier returns a sanitized identifier (table or column name).
// If the identifier is not safe, it returns an empty string.
func EscIdentifier(v string) string {
	if !IsSafeIdentifier(v) {
		return ""
	}
	return v
}

func esc(v string) string { return EscString(v) }

func logSuspiciousSQL(input, reason string) {
	slog.Warn("suspicious_sql_input_rejected",
		"reason", reason,
		"input_preview", truncateForLog(input, 200),
		"timestamp", time.Now().UTC().Format(time.RFC3339),
	)
}

func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func sqlStringNonNull(v string) string {
	sanitized, err := ValidateSQLInput(v)
	if err != nil {
		logSuspiciousSQL(v, err.Error())
		return "''"
	}
	return fmt.Sprintf("'%s'", sanitized)
}

func (d *DB) InsertMessage(m map[string]any) (bool, error) {
	payloadJSON, _ := json.Marshal(m["payload_json"])
	rows, err := d.QueryRows(fmt.Sprintf(`INSERT OR IGNORE INTO messages(transport_name,packet_id,dedupe_hash,channel_id,gateway_id,from_node,to_node,portnum,payload_text,payload_json,raw_hex,rx_time,hop_limit,relay_node) VALUES('%s',%d,'%s','%s','%s',%d,%d,%d,'%s','%s','%s','%s',%d,%d); SELECT changes() AS changes;`,
		esc(asString(m["transport_name"])), asInt(m["packet_id"]), esc(asString(m["dedupe_hash"])), esc(asString(m["channel_id"])), esc(asString(m["gateway_id"])), asInt(m["from_node"]), asInt(m["to_node"]), asInt(m["portnum"]), esc(asString(m["payload_text"])), esc(string(payloadJSON)), esc(asString(m["raw_hex"])), esc(asString(m["rx_time"])), asInt(m["hop_limit"]), asInt(m["relay_node"])))
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	return asInt(rows[0]["changes"]) > 0, nil
}

func (d *DB) UpsertNode(m map[string]any) error {
	sql := fmt.Sprintf(`INSERT INTO nodes(node_num,node_id,long_name,short_name,last_seen,last_gateway_id,last_snr,last_rssi,lat_redacted,lon_redacted,altitude,updated_at) VALUES(%d,'%s','%s','%s','%s','%s',%f,%d,%f,%f,%d,'%s') ON CONFLICT(node_num) DO UPDATE SET node_id=excluded.node_id,long_name=excluded.long_name,short_name=excluded.short_name,last_seen=excluded.last_seen,last_gateway_id=excluded.last_gateway_id,last_snr=excluded.last_snr,last_rssi=excluded.last_rssi,lat_redacted=excluded.lat_redacted,lon_redacted=excluded.lon_redacted,altitude=excluded.altitude,updated_at=excluded.updated_at;`,
		asInt(m["node_num"]), esc(asString(m["node_id"])), esc(asString(m["long_name"])), esc(asString(m["short_name"])), esc(asString(m["last_seen"])), esc(asString(m["last_gateway_id"])), asFloat(m["last_snr"]), asInt(m["last_rssi"]), asFloat(m["lat_redacted"]), asFloat(m["lon_redacted"]), asInt(m["altitude"]), time.Now().UTC().Format(time.RFC3339))
	return d.Exec(sql)
}

func (d *DB) InsertTelemetrySample(nodeNum int64, sampleType string, value any, observedAt string) error {
	valueJSON, _ := json.Marshal(value)
	sql := fmt.Sprintf(`INSERT INTO telemetry_samples(node_num,sample_type,value_json,observed_at) VALUES(%d,'%s','%s','%s');`, nodeNum, esc(sampleType), esc(string(valueJSON)), esc(observedAt))
	return d.Exec(sql)
}

func (d *DB) InsertAuditLog(category, level, message string, details any) error {
	detailJSON, _ := json.Marshal(details)
	sql := fmt.Sprintf(`INSERT INTO audit_logs(category,level,message,details_json,created_at) VALUES('%s','%s','%s','%s','%s');`, esc(category), esc(level), esc(message), esc(string(detailJSON)), time.Now().UTC().Format(time.RFC3339))
	return d.Exec(sql)
}

func (d *DB) InsertDeadLetter(dl DeadLetter) error {
	detailJSON, _ := json.Marshal(dl.Details)
	sql := fmt.Sprintf(`INSERT INTO dead_letters(transport_name,transport_type,topic,reason,payload_hex,details_json,created_at) VALUES('%s','%s','%s','%s','%s','%s','%s');`,
		esc(dl.TransportName), esc(dl.TransportType), esc(dl.Topic), esc(dl.Reason), esc(dl.PayloadHex), esc(string(detailJSON)), time.Now().UTC().Format(time.RFC3339))
	return d.Exec(sql)
}

func (d *DB) InsertConfigApply(actor, summary, sha string, diff any) error {
	diffJSON, _ := json.Marshal(diff)
	sql := fmt.Sprintf(`INSERT INTO config_apply_history(actor,summary,applied_at,config_sha256,diff_json) VALUES('%s','%s','%s','%s','%s');`, esc(actor), esc(summary), time.Now().UTC().Format(time.RFC3339), esc(sha), esc(string(diffJSON)))
	return d.Exec(sql)
}

// InsertRBACAuditLog inserts an RBAC audit log entry into the audit_log table.
// This provides action attribution for control actions and configuration changes.
func (d *DB) InsertRBACAuditLog(entry auth.AuditEntry) error {
	detailsJSON := ""
	if entry.Details != nil {
		detailsBytes, _ := json.Marshal(entry.Details)
		detailsJSON = string(detailsBytes)
	}

	sql := fmt.Sprintf(`INSERT INTO audit_log(id,timestamp,actor_id,action_class,action_detail,resource_type,resource_id,reason,result,details,session_id,remote_addr)
		VALUES('%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s');`,
		esc(entry.ID),
		esc(entry.Timestamp.Format(time.RFC3339)),
		esc(string(entry.ActorID)),
		esc(string(entry.ActionClass)),
		esc(entry.ActionDetail),
		esc(entry.ResourceType),
		esc(entry.ResourceID),
		esc(entry.Reason),
		esc(string(entry.Result)),
		esc(detailsJSON),
		esc(entry.SessionID),
		esc(entry.RemoteAddr))
	return d.Exec(sql)
}

func (d *DB) UpsertTransportRuntime(tr TransportRuntime) error {
	now := time.Now().UTC().Format(time.RFC3339)
	sql := fmt.Sprintf(`BEGIN IMMEDIATE;
INSERT INTO transport_runtime_status(transport_name,transport_type,source,enabled,runtime_state,detail,last_attempt_at,last_connected_at,last_success_at,last_message_at,last_error,total_messages,updated_at) VALUES('%s','%s','%s',%d,'%s','%s',%s,%s,%s,%s,%s,%d,'%s')
ON CONFLICT(transport_name) DO UPDATE SET transport_type=excluded.transport_type,source=excluded.source,enabled=excluded.enabled,runtime_state=excluded.runtime_state,detail=excluded.detail,last_attempt_at=excluded.last_attempt_at,last_connected_at=excluded.last_connected_at,last_success_at=excluded.last_success_at,last_message_at=excluded.last_message_at,last_error=excluded.last_error,total_messages=excluded.total_messages,updated_at=excluded.updated_at;
INSERT INTO transport_runtime_evidence(transport_name,last_heartbeat_at,packets_dropped,reconnect_attempts,consecutive_timeouts,last_failure_at,episode_id,failure_count,observation_drops,last_observation_drop_at,updated_at) VALUES('%s',%s,%d,%d,%d,%s,%s,%d,%d,%s,'%s')
ON CONFLICT(transport_name) DO UPDATE SET last_heartbeat_at=excluded.last_heartbeat_at,packets_dropped=excluded.packets_dropped,reconnect_attempts=excluded.reconnect_attempts,consecutive_timeouts=excluded.consecutive_timeouts,last_failure_at=excluded.last_failure_at,episode_id=excluded.episode_id,failure_count=excluded.failure_count,observation_drops=excluded.observation_drops,last_observation_drop_at=excluded.last_observation_drop_at,updated_at=excluded.updated_at;
COMMIT;`,
		esc(tr.Name), esc(tr.Type), esc(tr.Source), boolInt(tr.Enabled), esc(tr.State), esc(tr.Detail), sqlString(tr.LastAttemptAt), sqlString(tr.LastConnectedAt), sqlString(tr.LastSuccessAt), sqlString(tr.LastMessageAt), sqlString(tr.LastError), tr.TotalMessages, now,
		esc(tr.Name), sqlString(tr.LastHeartbeatAt), tr.PacketsDropped, tr.Reconnects, tr.Timeouts, sqlStringNonNull(tr.LastFailureAt), sqlStringNonNull(tr.EpisodeID), tr.FailureCount, tr.ObservationDrops, sqlStringNonNull(tr.LastObservationDrop), now)
	return d.Exec(sql)
}

func (d *DB) TransportRuntimeStatuses() ([]TransportRuntime, error) {
	rows, err := d.QueryRows("SELECT transport_name, transport_type, source, enabled, runtime_state, detail, COALESCE(last_attempt_at,'') AS last_attempt_at, COALESCE(last_connected_at,'') AS last_connected_at, COALESCE(last_success_at,'') AS last_success_at, COALESCE(last_message_at,'') AS last_message_at, COALESCE(last_error,'') AS last_error, COALESCE(total_messages,0) AS total_messages, COALESCE(updated_at,'') AS updated_at FROM transport_runtime_status ORDER BY transport_name;")
	if err != nil {
		return nil, err
	}
	evidenceByName, err := d.transportRuntimeEvidenceMap()
	if err != nil {
		return nil, err
	}
	out := make([]TransportRuntime, 0, len(rows))
	for _, row := range rows {
		evidence := evidenceByName[asString(row["transport_name"])]
		out = append(out, TransportRuntime{
			Name:                asString(row["transport_name"]),
			Type:                asString(row["transport_type"]),
			Source:              asString(row["source"]),
			Enabled:             asInt(row["enabled"]) == 1,
			State:               asString(row["runtime_state"]),
			Detail:              asString(row["detail"]),
			LastAttemptAt:       asString(row["last_attempt_at"]),
			LastConnectedAt:     asString(row["last_connected_at"]),
			LastSuccessAt:       asString(row["last_success_at"]),
			LastMessageAt:       asString(row["last_message_at"]),
			LastHeartbeatAt:     evidence.LastHeartbeatAt,
			LastFailureAt:       evidence.LastFailureAt,
			LastObservationDrop: evidence.LastObservationDrop,
			LastError:           asString(row["last_error"]),
			EpisodeID:           evidence.EpisodeID,
			TotalMessages:       uint64(asInt(row["total_messages"])),
			PacketsDropped:      evidence.PacketsDropped,
			Reconnects:          evidence.Reconnects,
			Timeouts:            evidence.Timeouts,
			FailureCount:        evidence.FailureCount,
			ObservationDrops:    evidence.ObservationDrops,
			UpdatedAt:           asString(row["updated_at"]),
		})
	}
	return out, nil
}

func (d *DB) transportRuntimeEvidenceMap() (map[string]TransportRuntime, error) {
	rows, err := d.QueryRows("SELECT transport_name, COALESCE(last_heartbeat_at,'') AS last_heartbeat_at, COALESCE(packets_dropped,0) AS packets_dropped, COALESCE(reconnect_attempts,0) AS reconnect_attempts, COALESCE(consecutive_timeouts,0) AS consecutive_timeouts, COALESCE(last_failure_at,'') AS last_failure_at, COALESCE(episode_id,'') AS episode_id, COALESCE(failure_count,0) AS failure_count, COALESCE(observation_drops,0) AS observation_drops, COALESCE(last_observation_drop_at,'') AS last_observation_drop_at FROM transport_runtime_evidence;")
	if err != nil {
		return nil, err
	}
	out := make(map[string]TransportRuntime, len(rows))
	for _, row := range rows {
		out[asString(row["transport_name"])] = TransportRuntime{
			LastHeartbeatAt:     asString(row["last_heartbeat_at"]),
			PacketsDropped:      uint64(asInt(row["packets_dropped"])),
			Reconnects:          uint64(asInt(row["reconnect_attempts"])),
			Timeouts:            uint64(asInt(row["consecutive_timeouts"])),
			LastFailureAt:       asString(row["last_failure_at"]),
			EpisodeID:           asString(row["episode_id"]),
			FailureCount:        uint64(asInt(row["failure_count"])),
			ObservationDrops:    uint64(asInt(row["observation_drops"])),
			LastObservationDrop: asString(row["last_observation_drop_at"]),
		}
	}
	return out, nil
}

func (d *DB) DeadLetterCounts() (map[string]uint64, error) {
	rows, err := d.QueryRows("SELECT transport_name, COUNT(*) AS dead_letter_count FROM dead_letters GROUP BY transport_name;")
	if err != nil {
		return nil, err
	}
	out := map[string]uint64{}
	for _, row := range rows {
		out[asString(row["transport_name"])] = uint64(asInt(row["dead_letter_count"]))
	}
	return out, nil
}

func (d *DB) MessageStatsByTransport(name string) (uint64, string, error) {
	rows, err := d.QueryRows(fmt.Sprintf("SELECT COUNT(*) AS message_count, COALESCE(MAX(rx_time), '') AS last_rx_time FROM messages WHERE transport_name='%s';", esc(name)))
	if err != nil {
		return 0, "", err
	}
	if len(rows) == 0 {
		return 0, "", nil
	}
	return uint64(asInt(rows[0]["message_count"])), asString(rows[0]["last_rx_time"]), nil
}

func (d *DB) UpsertIncident(record models.Incident) error {
	if strings.TrimSpace(record.ID) == "" {
		return fmt.Errorf("incident id is required")
	}
	metadataJSON, _ := json.Marshal(record.Metadata)
	if record.OccurredAt == "" {
		record.OccurredAt = time.Now().UTC().Format(time.RFC3339)
	}
	record.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	sql := fmt.Sprintf(`INSERT INTO incidents(id,category,severity,title,summary,resource_type,resource_id,state,actor_id,occurred_at,updated_at,resolved_at,metadata_json)
		VALUES('%s','%s','%s','%s','%s','%s','%s','%s',%s,'%s','%s',%s,'%s')
		ON CONFLICT(id) DO UPDATE SET category=excluded.category,severity=excluded.severity,title=excluded.title,summary=excluded.summary,resource_type=excluded.resource_type,resource_id=excluded.resource_id,state=excluded.state,actor_id=excluded.actor_id,occurred_at=excluded.occurred_at,updated_at=excluded.updated_at,resolved_at=excluded.resolved_at,metadata_json=excluded.metadata_json;`,
		esc(record.ID), esc(record.Category), esc(record.Severity), esc(record.Title), esc(record.Summary), esc(record.ResourceType), esc(record.ResourceID), esc(record.State), sqlString(record.ActorID), esc(record.OccurredAt), esc(record.UpdatedAt), sqlString(record.ResolvedAt), esc(string(metadataJSON)))
	return d.Exec(sql)
}

func (d *DB) RecentIncidents(limit int) ([]models.Incident, error) {
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id, category, severity, title, summary, resource_type, resource_id, state, COALESCE(actor_id,'') AS actor_id, occurred_at, updated_at, COALESCE(resolved_at,'') AS resolved_at, COALESCE(metadata_json,'{}') AS metadata_json
		FROM incidents ORDER BY occurred_at DESC LIMIT %d;`, limit))
	if err != nil {
		return nil, err
	}
	out := make([]models.Incident, 0, len(rows))
	for _, row := range rows {
		item := models.Incident{
			ID:           asString(row["id"]),
			Category:     asString(row["category"]),
			Severity:     asString(row["severity"]),
			Title:        asString(row["title"]),
			Summary:      asString(row["summary"]),
			ResourceType: asString(row["resource_type"]),
			ResourceID:   asString(row["resource_id"]),
			State:        asString(row["state"]),
			ActorID:      asString(row["actor_id"]),
			OccurredAt:   asString(row["occurred_at"]),
			UpdatedAt:    asString(row["updated_at"]),
			ResolvedAt:   asString(row["resolved_at"]),
			Metadata:     map[string]any{},
		}
		_ = json.Unmarshal([]byte(asString(row["metadata_json"])), &item.Metadata)
		out = append(out, item)
	}
	return out, nil
}

func (d *DB) IncidentByID(id string) (models.Incident, bool, error) {
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id, category, severity, title, summary, resource_type, resource_id, state, COALESCE(actor_id,'') AS actor_id, occurred_at, updated_at, COALESCE(resolved_at,'') AS resolved_at, COALESCE(metadata_json,'{}') AS metadata_json
		FROM incidents WHERE id='%s' LIMIT 1;`, esc(id)))
	if err != nil {
		return models.Incident{}, false, err
	}
	if len(rows) == 0 {
		return models.Incident{}, false, nil
	}
	row := rows[0]
	item := models.Incident{
		ID:           asString(row["id"]),
		Category:     asString(row["category"]),
		Severity:     asString(row["severity"]),
		Title:        asString(row["title"]),
		Summary:      asString(row["summary"]),
		ResourceType: asString(row["resource_type"]),
		ResourceID:   asString(row["resource_id"]),
		State:        asString(row["state"]),
		ActorID:      asString(row["actor_id"]),
		OccurredAt:   asString(row["occurred_at"]),
		UpdatedAt:    asString(row["updated_at"]),
		ResolvedAt:   asString(row["resolved_at"]),
		Metadata:     map[string]any{},
	}
	_ = json.Unmarshal([]byte(asString(row["metadata_json"])), &item.Metadata)
	return item, true, nil
}

func (d *DB) VerifyWriteRead() error {
	probe := fmt.Sprintf("doctor-%d", time.Now().UnixNano())
	sql := fmt.Sprintf("CREATE TEMP TABLE IF NOT EXISTS doctor_write_check(v TEXT); DELETE FROM doctor_write_check; INSERT INTO doctor_write_check(v) VALUES('%s'); SELECT v FROM doctor_write_check LIMIT 1;", esc(probe))
	got, err := d.Scalar(sql)
	if err != nil {
		return err
	}
	if got != probe {
		return fmt.Errorf("db write/read verification mismatch: wrote %q got %q", probe, got)
	}
	return nil
}

func (d *DB) Vacuum() error { return d.Exec("VACUUM;") }

func asString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func asInt(v any) int64 {
	switch x := v.(type) {
	case int:
		return int64(x)
	case int64:
		return x
	case uint32:
		return int64(x)
	case float64:
		return int64(x)
	case string:
		parsed, _ := strconv.ParseInt(x, 10, 64)
		return parsed
	}
	var i int64
	fmt.Sscan(fmt.Sprint(v), &i)
	return i
}

func asFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case string:
		parsed, _ := strconv.ParseFloat(x, 64)
		return parsed
	}
	var f float64
	fmt.Sscan(fmt.Sprint(v), &f)
	return f
}

func migrationDir() string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	return filepath.Join(root, "migrations")
}

func sqlString(v string) string {
	if v == "" {
		return "NULL"
	}
	sanitized, err := ValidateSQLInput(v)
	if err != nil {
		logSuspiciousSQL(v, err.Error())
		return "NULL"
	}
	return fmt.Sprintf("'%s'", sanitized)
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return 100
	}
	if limit > MaxRows {
		return MaxRows
	}
	return limit
}

func validateOrderBy(orderBy string) string {
	if !IsSafeIdentifier(orderBy) {
		return ""
	}
	return orderBy
}

func buildSafeWhereClause(column, value string) (string, error) {
	if !IsSafeIdentifier(column) {
		return "", logging.NewSafeError("invalid query parameters", fmt.Errorf("invalid column name: %s", column), "validation", false)
	}
	sanitized, err := ValidateSQLInput(value)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s='%s'", column, sanitized), nil
}

func isTransientDBError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	transient := []string{
		"locked", "busy", "timeout", "temporary", "retry",
		"database is locked", "busy timeout",
	}
	for _, pattern := range transient {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}
func (d *DB) LastAuditTime() string {
	rows, err := d.QueryRows("SELECT created_at FROM audit_logs ORDER BY created_at DESC LIMIT 1;")
	if err != nil || len(rows) == 0 {
		return ""
	}
	return asString(rows[0]["created_at"])
}
