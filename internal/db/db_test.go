package db

import (
	"github.com/mel-project/mel/internal/config"
	"path/filepath"
	"testing"
)

func TestOpenAppliesMigration(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := d.QueryJSON("SELECT version FROM schema_migrations ORDER BY version;")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 5 {
		t.Fatalf("expected schema migrations including runtime evidence, got %v", rows)
	}
}

// TestRepairMigration0022BackfillsMissingRow simulates legacy DBs where 0022 ALTERs ran
// without a schema_migrations row; ApplyMigrations must not fail with duplicate columns.
func TestRepairMigration0035BackfillsWhenOnly0034PackRow(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Exec(`DELETE FROM schema_migrations WHERE version='0035_incident_decision_pack_adjudication';`); err != nil {
		t.Fatal(err)
	}
	if err := d.Exec(`INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES ('0034_incident_decision_pack_adjudication', datetime('now'));`); err != nil {
		t.Fatal(err)
	}
	if err := d.ApplyMigrations(migrationDir()); err != nil {
		t.Fatal(err)
	}
	got, err := d.Scalar(`SELECT COUNT(*) FROM schema_migrations WHERE version='0035_incident_decision_pack_adjudication';`)
	if err != nil || got != "1" {
		t.Fatalf("expected 0035 row after repair, got %q err=%v", got, err)
	}
	n, err := d.HighestMigrationNumeric()
	if err != nil || n != 37 {
		t.Fatalf("HighestMigrationNumeric=%d err=%v", n, err)
	}
}

func TestRepairMigration0022BackfillsMissingRow(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Exec(`DELETE FROM schema_migrations WHERE version='0022_control_action_policy_truth';`); err != nil {
		t.Fatal(err)
	}
	if err := d.ApplyMigrations(migrationDir()); err != nil {
		t.Fatal(err)
	}
	got, err := d.Scalar(`SELECT COUNT(*) FROM schema_migrations WHERE version='0022_control_action_policy_truth';`)
	if err != nil || got != "1" {
		t.Fatalf("expected backfilled migration row, got %q err=%v", got, err)
	}
}

func TestInsertMessageReportsDedupedWrite(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	msg := map[string]any{"transport_name": "radio", "packet_id": int64(1), "dedupe_hash": "abc", "channel_id": "", "gateway_id": "", "from_node": int64(1), "to_node": int64(2), "portnum": int64(1), "payload_text": "hi", "payload_json": map[string]any{"transport_name": "radio"}, "raw_hex": "01", "rx_time": "2026-03-18T00:00:00Z", "hop_limit": int64(3), "relay_node": int64(0)}
	stored, err := d.InsertMessage(msg)
	if err != nil {
		t.Fatal(err)
	}
	if !stored {
		t.Fatal("expected initial insert to store")
	}
	stored, err = d.InsertMessage(msg)
	if err != nil {
		t.Fatal(err)
	}
	if stored {
		t.Fatal("expected duplicate insert to be ignored")
	}
}

func TestInsertDeadLetter(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.InsertDeadLetter(DeadLetter{TransportName: "mqtt", TransportType: "mqtt", Topic: "msh/test", Reason: "parse failure", PayloadHex: "0102", Details: map[string]any{"error": "boom"}}); err != nil {
		t.Fatal(err)
	}
	rows, err := d.QueryJSON("SELECT transport_name, transport_type, reason, payload_hex FROM dead_letters;")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["reason"] != "parse failure" || rows[0]["transport_type"] != "mqtt" {
		t.Fatalf("unexpected dead letter rows: %+v", rows)
	}
}

func TestUpsertTransportRuntimePersistsEvidence(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertTransportRuntime(TransportRuntime{
		Name:            "mqtt",
		Type:            "mqtt",
		Source:          "127.0.0.1:1883",
		Enabled:         true,
		State:           "idle",
		Detail:          "connected; waiting for broker heartbeat or publish",
		LastHeartbeatAt: "2026-03-19T00:00:03Z",
		PacketsDropped:  2,
		Reconnects:      4,
		Timeouts:        1,
	}); err != nil {
		t.Fatal(err)
	}
	rows, err := d.QueryJSON("SELECT transport_name, last_heartbeat_at, packets_dropped, reconnect_attempts, consecutive_timeouts FROM transport_runtime_evidence;")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["last_heartbeat_at"] != "2026-03-19T00:00:03Z" || rows[0]["reconnect_attempts"] != "4" {
		t.Fatalf("unexpected transport runtime evidence rows: %+v", rows)
	}
}

func TestPersistIngestWritesMessageNodeAndTelemetryAtomically(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := d.PersistIngest(IngestRecord{
		Message:        map[string]any{"transport_name": "mqtt", "packet_id": int64(1), "dedupe_hash": "ingest-1", "channel_id": "test", "gateway_id": "gw", "from_node": int64(10), "to_node": int64(11), "portnum": int64(1), "payload_text": "hello", "payload_json": map[string]any{"payload_text": "hello"}, "raw_hex": "00", "rx_time": "2026-03-19T00:00:00Z", "hop_limit": int64(0), "relay_node": int64(0)},
		Node:           map[string]any{"node_num": int64(10), "node_id": "!abcd", "long_name": "Node A", "short_name": "A", "last_seen": "2026-03-19T00:00:00Z", "last_gateway_id": "gw", "last_snr": float64(7.5), "last_rssi": int64(-20), "lat_redacted": float64(1.23), "lon_redacted": float64(4.56), "altitude": int64(123)},
		TelemetryType:  "position",
		TelemetryValue: map[string]any{"lat_redacted": 1.23},
		ObservedAt:     "2026-03-19T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !stored {
		t.Fatal("expected first atomic ingest write to store")
	}
	messageCount, err := d.Scalar("SELECT COUNT(*) FROM messages WHERE dedupe_hash='ingest-1';")
	if err != nil || messageCount != "1" {
		t.Fatalf("expected message row, got count=%s err=%v", messageCount, err)
	}
	nodeCount, err := d.Scalar("SELECT COUNT(*) FROM nodes WHERE node_num=10;")
	if err != nil || nodeCount != "1" {
		t.Fatalf("expected node row, got count=%s err=%v", nodeCount, err)
	}
	telemetryCount, err := d.Scalar("SELECT COUNT(*) FROM telemetry_samples WHERE node_num=10;")
	if err != nil || telemetryCount != "1" {
		t.Fatalf("expected telemetry row, got count=%s err=%v", telemetryCount, err)
	}
}

func TestPersistIngestValidationFailureLeavesNoPartialWrites(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := d.PersistIngest(IngestRecord{
		Message: map[string]any{"transport_name": "", "packet_id": int64(1), "dedupe_hash": "broken", "raw_hex": "00", "rx_time": "2026-03-19T00:00:00Z"},
		Node:    map[string]any{"node_num": int64(10)},
	})
	if err == nil {
		t.Fatal("expected ingest validation or sqlite error")
	}
	if stored {
		t.Fatal("expected failed ingest to report not stored")
	}
	messageCount, _ := d.Scalar("SELECT COUNT(*) FROM messages;")
	nodeCount, _ := d.Scalar("SELECT COUNT(*) FROM nodes;")
	telemetryCount, _ := d.Scalar("SELECT COUNT(*) FROM telemetry_samples;")
	if messageCount != "0" || nodeCount != "0" || telemetryCount != "0" {
		t.Fatalf("expected no partial writes, got messages=%s nodes=%s telemetry=%s", messageCount, nodeCount, telemetryCount)
	}
}

func TestAuditLogChainVerify(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.InsertAuditLog("test", "info", "one", map[string]any{"n": 1}); err != nil {
		t.Fatal(err)
	}
	if err := d.InsertAuditLog("test", "info", "two", map[string]any{"n": 2}); err != nil {
		t.Fatal(err)
	}
	rep, err := d.VerifyAuditLogChain()
	if err != nil {
		t.Fatal(err)
	}
	if !rep.OK || rep.VerifiedRows != 2 {
		t.Fatalf("expected chain ok with 2 rows, got %+v", rep)
	}
	if rep.HeadChainHash == "" {
		t.Fatal("expected head chain hash")
	}
}
