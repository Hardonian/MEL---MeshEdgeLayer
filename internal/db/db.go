package db

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
)

type DB struct{ Path string }

type TransportRuntime struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	Source          string `json:"source"`
	Enabled         bool   `json:"enabled"`
	State           string `json:"state"`
	Detail          string `json:"detail"`
	LastAttemptAt   string `json:"last_attempt_at,omitempty"`
	LastConnectedAt string `json:"last_connected_at,omitempty"`
	LastSuccessAt   string `json:"last_success_at,omitempty"`
	LastMessageAt   string `json:"last_message_at,omitempty"`
	LastError       string `json:"last_error,omitempty"`
	TotalMessages   uint64 `json:"total_messages"`
	UpdatedAt       string `json:"updated_at,omitempty"`
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
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		cmd := exec.Command("sqlite3", d.Path)
		cmd.Stdin = strings.NewReader(string(b))
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("sqlite3 migrate %s: %w: %s", name, err, out)
		}
	}
	return nil
}

func (d *DB) Exec(sql string) error {
	cmd := exec.Command("sqlite3", d.Path, sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sqlite exec failed: %w: %s", err, out)
	}
	return nil
}

func (d *DB) QueryRows(sql string) ([]map[string]any, error) {
	cmd := exec.Command("sqlite3", "-json", d.Path, sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("sqlite query failed: %w: %s", err, out)
	}
	rows := make([]map[string]any, 0)
	if len(out) == 0 {
		return rows, nil
	}
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil, err
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

func esc(v string) string { return strings.ReplaceAll(v, "'", "''") }

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

func (d *DB) InsertConfigApply(actor, summary, sha string, diff any) error {
	diffJSON, _ := json.Marshal(diff)
	sql := fmt.Sprintf(`INSERT INTO config_apply_history(actor,summary,applied_at,config_sha256,diff_json) VALUES('%s','%s','%s','%s','%s');`, esc(actor), esc(summary), time.Now().UTC().Format(time.RFC3339), esc(sha), esc(string(diffJSON)))
	return d.Exec(sql)
}

func (d *DB) UpsertTransportRuntime(tr TransportRuntime) error {
	sql := fmt.Sprintf(`INSERT INTO transport_runtime_status(transport_name,transport_type,source,enabled,runtime_state,detail,last_attempt_at,last_connected_at,last_success_at,last_message_at,last_error,total_messages,updated_at) VALUES('%s','%s','%s',%d,'%s','%s',%s,%s,%s,%s,%s,%d,'%s') ON CONFLICT(transport_name) DO UPDATE SET transport_type=excluded.transport_type,source=excluded.source,enabled=excluded.enabled,runtime_state=excluded.runtime_state,detail=excluded.detail,last_attempt_at=excluded.last_attempt_at,last_connected_at=excluded.last_connected_at,last_success_at=excluded.last_success_at,last_message_at=excluded.last_message_at,last_error=excluded.last_error,total_messages=excluded.total_messages,updated_at=excluded.updated_at;`,
		esc(tr.Name), esc(tr.Type), esc(tr.Source), boolInt(tr.Enabled), esc(tr.State), esc(tr.Detail), sqlString(tr.LastAttemptAt), sqlString(tr.LastConnectedAt), sqlString(tr.LastSuccessAt), sqlString(tr.LastMessageAt), sqlString(tr.LastError), tr.TotalMessages, time.Now().UTC().Format(time.RFC3339))
	return d.Exec(sql)
}

func (d *DB) TransportRuntimeStatuses() ([]TransportRuntime, error) {
	rows, err := d.QueryRows("SELECT transport_name, transport_type, source, enabled, runtime_state, detail, COALESCE(last_attempt_at,'') AS last_attempt_at, COALESCE(last_connected_at,'') AS last_connected_at, COALESCE(last_success_at,'') AS last_success_at, COALESCE(last_message_at,'') AS last_message_at, COALESCE(last_error,'') AS last_error, COALESCE(total_messages,0) AS total_messages, COALESCE(updated_at,'') AS updated_at FROM transport_runtime_status ORDER BY transport_name;")
	if err != nil {
		return nil, err
	}
	out := make([]TransportRuntime, 0, len(rows))
	for _, row := range rows {
		out = append(out, TransportRuntime{
			Name:            asString(row["transport_name"]),
			Type:            asString(row["transport_type"]),
			Source:          asString(row["source"]),
			Enabled:         asInt(row["enabled"]) == 1,
			State:           asString(row["runtime_state"]),
			Detail:          asString(row["detail"]),
			LastAttemptAt:   asString(row["last_attempt_at"]),
			LastConnectedAt: asString(row["last_connected_at"]),
			LastSuccessAt:   asString(row["last_success_at"]),
			LastMessageAt:   asString(row["last_message_at"]),
			LastError:       asString(row["last_error"]),
			TotalMessages:   uint64(asInt(row["total_messages"])),
			UpdatedAt:       asString(row["updated_at"]),
		})
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
	return fmt.Sprintf("'%s'", esc(v))
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
