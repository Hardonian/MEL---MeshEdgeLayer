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

func TestReadyzSeparatesProcessAndIngestTruth(t *testing.T) {
	srv := newTestServer(t, []transport.Health{{Name: "tcp", Type: "tcp", State: "error", Detail: "connect failed"}}, nil)
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
	if payload["process_ready"] != true {
		t.Fatalf("expected process_ready=true, got %#v", payload["process_ready"])
	}
	if payload["ingest_ready"] != false {
		t.Fatalf("expected ingest_ready=false, got %#v", payload["ingest_ready"])
	}
	if payload["operator_state"] != "degraded" {
		t.Fatalf("expected degraded operator_state, got %#v", payload["operator_state"])
	}
}

func TestStatusSeparatesRuntimeAndPersistedState(t *testing.T) {
	insert := func(d *db.DB) {
		if _, err := d.InsertMessage(map[string]any{
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
		}); err != nil {
			t.Fatal(err)
		}
	}
	srv := newTestServer(t, []transport.Health{{Name: "mqtt", Type: "mqtt", State: "configured_not_attempted"}}, insert)
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
	runtimeSnapshot := payload["runtime_snapshot"].(map[string]any)
	if runtimeSnapshot["messages"].(float64) != 0 {
		t.Fatalf("expected empty runtime snapshot, got %#v", runtimeSnapshot["messages"])
	}
	persisted := payload["persisted_summary"].(map[string]any)
	if persisted["messages"] != "1" {
		t.Fatalf("expected persisted message count of 1, got %#v", persisted["messages"])
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
	return New(cfg, logging.New("info", false), database, meshstate.New(), events.New(), func() []transport.Health { return health }, func() []policy.Recommendation { return nil })
}
