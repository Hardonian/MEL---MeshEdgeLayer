package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/logging"
	"github.com/mel-project/mel/internal/meshstate"
	"github.com/mel-project/mel/internal/policy"
	"github.com/mel-project/mel/internal/transport"
)

func TestReadyzReturnsSnapshot(t *testing.T) {
	srv := newTestServer(t, []transport.Health{{Name: "tcp", Type: "tcp", State: transport.StateError, Detail: "connect failed"}}, nil)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["ready"] != true {
		t.Fatalf("expected ready=true, got %#v", payload["ready"])
	}
	transports := payload["transports"].([]any)
	if len(transports) != 1 {
		t.Fatalf("expected one transport snapshot, got %#v", payload)
	}
}

func TestStatusReturnsTransportSummary(t *testing.T) {
	insert := func(d *db.DB) {
		stored, err := d.InsertMessage(map[string]any{
			"transport_name": "mqtt",
			"packet_id":      int64(1),
			"dedupe_hash":    "abc123",
			"channel_id":     "test",
			"gateway_id":     "gw",
			"from_node":      int64(10),
			"to_node":        int64(11),
			"portnum":        int64(1),
			"payload_text":   "hello",
			"payload_json":   map[string]any{"payload_text": "hello"},
			"raw_hex":        "00",
			"rx_time":        "2026-03-19T00:00:00Z",
			"hop_limit":      int64(0),
			"relay_node":     int64(0),
		})
		if err != nil {
			t.Fatal(err)
		}
		if !stored {
			t.Fatal("expected seed message to persist")
		}
	}
	srv := newTestServer(t, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateConfiguredNotAttempted}}, insert)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()

	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	status := payload["status"].(map[string]any)
	if status["messages"].(float64) != 1 {
		t.Fatalf("expected persisted message count of 1, got %#v", status["messages"])
	}
	transports := status["transports"].([]any)
	if len(transports) != 1 {
		t.Fatalf("expected one transport report, got %#v", status)
	}
	report := transports[0].(map[string]any)
	if report["effective_state"] != transport.StateHistoricalOnly {
		t.Fatalf("expected historical_only effective state, got %#v", report["effective_state"])
	}
}

func TestStatusUsesPersistedRuntimeEvidenceWhenNoLiveRuntimeIsPresent(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Features.WebUI = false
	cfg.Transports = []config.TransportConfig{{Name: "mqtt", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test", ClientID: "mel-test"}}
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.UpsertTransportRuntime(db.TransportRuntime{
		Name:            "mqtt",
		Type:            "mqtt",
		Source:          "127.0.0.1:1883",
		Enabled:         true,
		State:           transport.StateConnectedNoIngest,
		Detail:          "connected; waiting for broker heartbeat or publish",
		LastAttemptAt:   "2026-03-19T00:00:00Z",
		LastHeartbeatAt: "2026-03-19T00:00:03Z",
		Reconnects:      2,
		Timeouts:        1,
	}); err != nil {
		t.Fatal(err)
	}
	if err := database.InsertDeadLetter(db.DeadLetter{TransportName: "mqtt", TransportType: "mqtt", Topic: "msh/test", Reason: "parse failure", PayloadHex: "aa"}); err != nil {
		t.Fatal(err)
	}
	srv := New(cfg, logging.New("info", false), database, meshstate.New(), events.New(), func() []transport.Health { return nil }, func() []policy.Recommendation { return nil }, nil, nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()

	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	status := payload["status"].(map[string]any)
	report := status["transports"].([]any)[0].(map[string]any)
	if report["last_heartbeat_at"] != "2026-03-19T00:00:03Z" {
		t.Fatalf("expected persisted heartbeat evidence, got %#v", report["last_heartbeat_at"])
	}
	if report["consecutive_timeouts"].(float64) != 1 {
		t.Fatalf("expected timeout evidence, got %#v", report["consecutive_timeouts"])
	}
	if report["dead_letters"].(float64) != 1 {
		t.Fatalf("expected dead letter count, got %#v", report["dead_letters"])
	}
}

func TestDeadLettersEndpointFiltersByTransport(t *testing.T) {
	srv := newTestServer(t, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateIdle}}, func(database *db.DB) {
		if err := database.InsertDeadLetter(db.DeadLetter{TransportName: "mqtt", TransportType: "mqtt", Topic: "msh/test", Reason: "retry_threshold_exceeded", PayloadHex: "aa"}); err != nil {
			t.Fatal(err)
		}
		if err := database.InsertDeadLetter(db.DeadLetter{TransportName: "direct", TransportType: "tcp", Topic: "", Reason: "timeout_failure", PayloadHex: "bb"}); err != nil {
			t.Fatal(err)
		}
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dead-letters?transport=mqtt", nil)
	rec := httptest.NewRecorder()

	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	rows := payload["dead_letters"].([]any)
	if len(rows) != 1 {
		t.Fatalf("expected one filtered dead letter, got %#v", payload)
	}
	row := rows[0].(map[string]any)
	if row["transport_name"] != "mqtt" || row["transport_type"] != "mqtt" {
		t.Fatalf("unexpected dead letter row: %#v", row)
	}
}

func TestPanelEndpointExposesCompactOperatorView(t *testing.T) {
	srv := newTestServer(t, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateIngesting, Detail: "live ingest confirmed by SQLite writes"}}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/panel", nil)
	rec := httptest.NewRecorder()

	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["operator_state"] != "ready" {
		t.Fatalf("expected ready operator state, got %#v", payload["operator_state"])
	}
	if len(payload["short_commands"].([]any)) == 0 {
		t.Fatalf("expected short commands, got %#v", payload["short_commands"])
	}
}

func newTestServer(t *testing.T, health []transport.Health, seed func(*db.DB)) *Server {
	t.Helper()
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Features.WebUI = false
	cfg.Transports = make([]config.TransportConfig, 0, len(health))
	for _, h := range health {
		cfg.Transports = append(cfg.Transports, config.TransportConfig{Name: h.Name, Type: h.Type, Enabled: true, Endpoint: h.Source, Topic: "msh/test"})
	}
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if seed != nil {
		seed(database)
	}
	return New(cfg, logging.New("info", false), database, meshstate.New(), events.New(), func() []transport.Health { return health }, func() []policy.Recommendation { return nil }, nil, nil, nil, nil, nil)
}

func TestIncidentsEndpointReturnsGroupedTransportIncidents(t *testing.T) {
	srv := newTestServer(t, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateRetrying}}, func(database *db.DB) {
		if err := database.InsertAuditLog("transport", "warning", "timeout_stall", map[string]any{"transport": "mqtt", "dead_letter": false}); err != nil {
			t.Fatal(err)
		}
		if err := database.InsertAuditLog("transport", "warning", "timeout_stall", map[string]any{"transport": "mqtt", "dead_letter": false}); err != nil {
			t.Fatal(err)
		}
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil)
	rec := httptest.NewRecorder()

	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	incidents := payload["recent_transport_incidents"].([]any)
	if len(incidents) == 0 {
		t.Fatalf("expected grouped incidents, got %#v", payload)
	}
}

func TestTransportHealthEndpointsExposeDerivedHealthAndAlerts(t *testing.T) {
	srv := newTestServer(t, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateRetrying, EpisodeID: "ep-1", FailureCount: 2, ObservationDrops: 2, LastHeartbeatAt: "2026-03-19T00:00:00Z"}}, func(database *db.DB) {
		if err := database.InsertAuditLog("transport", "warning", transport.ReasonRetryThresholdExceeded, map[string]any{"transport": "mqtt", "type": "mqtt", "episode_id": "ep-1"}); err != nil {
			t.Fatal(err)
		}
		if err := database.UpsertTransportAlert(db.TransportAlertRecord{ID: "mqtt|retry_threshold_exceeded|retry-threshold", TransportName: "mqtt", TransportType: "mqtt", Severity: "critical", Reason: "retry_threshold_exceeded", Summary: "retry threshold exceeded", FirstTriggeredAt: "2026-03-19T00:00:00Z", LastUpdatedAt: "2026-03-19T00:00:00Z", Active: true, EpisodeID: "ep-1", ClusterKey: "retry-threshold"}); err != nil {
			t.Fatal(err)
		}
	})
	for _, path := range []string{"/api/v1/transports/health", "/api/v1/transports/alerts", "/api/v1/transports/anomalies"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.http.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status for %s: %d", path, rec.Code)
		}
	}
}

func TestControlEndpointsExposeStatusAndHistory(t *testing.T) {
	srv := newTestServer(t, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateRetrying}}, nil)
	for _, path := range []string{"/api/v1/control/status", "/api/v1/control/actions", "/api/v1/control/history"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.http.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status for %s: %d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestTransportHistoryEndpointsAndInspect(t *testing.T) {
	srv := newTestServer(t, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateRetrying, EpisodeID: "ep-1", FailureCount: 2, ObservationDrops: 3, LastHeartbeatAt: "2026-03-19T00:00:00Z"}}, func(database *db.DB) {
		if err := database.InsertTransportHealthSnapshot(db.TransportHealthSnapshot{TransportName: "mqtt", TransportType: "mqtt", Score: 42, State: "unstable", SnapshotTime: "2026-03-19T00:00:00Z", ActiveAlertCount: 1}); err != nil {
			t.Fatal(err)
		}
		if err := database.UpsertTransportAlert(db.TransportAlertRecord{ID: "mqtt|retry_threshold_exceeded|retry-threshold", TransportName: "mqtt", TransportType: "mqtt", Severity: "critical", Reason: "retry_threshold_exceeded", Summary: "retry threshold exceeded", FirstTriggeredAt: "2026-03-19T00:00:00Z", LastUpdatedAt: "2026-03-19T00:00:00Z", Active: true, EpisodeID: "ep-1", ClusterKey: "retry-threshold", ContributingReasons: []string{"retry_threshold_exceeded"}, PenaltySnapshot: []db.PenaltyRecord{{Reason: "retry_threshold_exceeded", Penalty: 30, Count: 1, Window: "5m"}}, TriggerCondition: "retry_threshold_exceeded_count=1"}); err != nil {
			t.Fatal(err)
		}
		if err := database.InsertAuditLog("transport", "warning", transport.ReasonObservationDropped, map[string]any{"transport": "mqtt", "type": "mqtt", "drop_count": 3, "drop_cause": "observation_queue_saturation"}); err != nil {
			t.Fatal(err)
		}
	})
	for _, path := range []string{
		"/api/v1/transports/health/history?transport=mqtt&limit=10",
		"/api/v1/transports/alerts/history?transport=mqtt&limit=10",
		"/api/v1/transports/anomalies/history?transport=mqtt&limit=10",
		"/api/v1/transports/inspect/mqtt",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.http.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status for %s: %d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestMeshEndpointsExposeMeshDrilldown(t *testing.T) {
	srv := newTestServer(t, []transport.Health{
		{Name: "mqtt-a", Type: "mqtt", State: transport.StateRetrying, EpisodeID: "ep-a", FailureCount: 2, LastHeartbeatAt: "2026-03-19T00:00:00Z"},
		{Name: "mqtt-b", Type: "mqtt", State: transport.StateRetrying, EpisodeID: "ep-b", FailureCount: 2, LastHeartbeatAt: "2026-03-19T00:00:00Z"},
	}, func(database *db.DB) {
		for _, name := range []string{"mqtt-a", "mqtt-b"} {
			if err := database.InsertAuditLog("transport", "warning", transport.ReasonRetryThresholdExceeded, map[string]any{"transport": name, "type": "mqtt", "episode_id": "ep-" + name}); err != nil {
				t.Fatal(err)
			}
			if err := database.UpsertTransportAlert(db.TransportAlertRecord{ID: name + "|retry_threshold_exceeded|retry-threshold", TransportName: name, TransportType: "mqtt", Severity: "critical", Reason: "retry_threshold_exceeded", Summary: "retry threshold exceeded", FirstTriggeredAt: "2026-03-19T00:00:00Z", LastUpdatedAt: "2026-03-19T00:00:00Z", Active: true}); err != nil {
				t.Fatal(err)
			}
		}
	})
	for _, path := range []string{"/api/v1/mesh", "/api/v1/mesh/inspect"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.http.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status for %s: %d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}
