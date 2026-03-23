package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/logging"
	"github.com/mel-project/mel/internal/meshstate"
	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/policy"
	"github.com/mel-project/mel/internal/transport"
)

func writeTestAuthConfig(t *testing.T, dbPath string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "mel.json")
	raw := []byte(`{
  "bind": {"api": "127.0.0.1:8080", "metrics": "", "allow_remote": false},
  "auth": {
    "enabled": true,
    "session_secret": "0123456789abcdef0123456789abcdef",
    "ui_user": "admin",
    "ui_password": "adminpw",
    "operator_keys": [
      {
        "key": "viewer-test-key",
        "capabilities": ["read_status", "read_incidents", "read_actions"]
      },
      {
        "key": "approver-test-key",
        "capabilities": ["read_status", "read_incidents", "read_actions", "approve_control_action", "reject_control_action"]
      },
      {
        "key": "approve-only-key",
        "capabilities": ["read_status", "read_incidents", "read_actions", "approve_control_action"]
      }
    ]
  },
  "storage": {
    "data_dir": "` + filepath.ToSlash(filepath.Dir(dbPath)) + `",
    "database_path": "` + filepath.ToSlash(dbPath) + `",
    "encryption_key_env": "MEL_STORAGE_KEY",
    "encryption_required": false
  },
  "logging": {"level": "info", "format": "json"},
  "retention": {"enabled": true, "messages_days": 30, "telemetry_days": 30, "audit_days": 90, "precise_position_days": 7},
  "privacy": {
    "store_precise_positions": false,
    "mqtt_encryption_required": true,
    "map_reporting_allowed": false,
    "redact_exports": true,
    "trust_list": []
  },
  "transports": [],
  "features": {"web_ui": false, "metrics": false, "ble_experimental": false},
  "rate_limits": {"http_rps": 20, "transport_reconnect_seconds": 10},
  "control": {"mode": "advisory", "emergency_disable": false},
  "strict_mode": false
}`)
	if err := os.WriteFile(p, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func newAuthedTrustServer(t *testing.T) (*Server, string) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "mel.db")
	cfgPath := writeTestAuthConfig(t, dbPath)
	cfg, _, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	srv := New(cfg, logging.New("error", false), database, meshstate.New(), events.New(),
		func() []transport.Health { return nil },
		func() []policy.Recommendation { return nil },
		nil, nil, nil, nil, nil)
	// Direct DB approve/reject is enough to prove HTTP authz; avoids import cycle with service → web.
	srv.SetTrustFuncs(
		func(id, actor, note string) error { return database.ApproveControlAction(id, actor, note) },
		func(id, actor, note string) error { return database.RejectControlAction(id, actor, note) },
		nil, nil, nil, nil, nil, nil, nil, nil,
	)
	safe := strings.ReplaceAll(t.Name(), "/", "_")
	return srv, "act-http-" + safe
}

func insertPendingForHTTP(t *testing.T, d *db.DB, id string) {
	t.Helper()
	if err := d.UpsertControlAction(db.ControlActionRecord{
		ID:               id,
		ActionType:       control.ActionRestartTransport,
		TargetTransport:  "mqtt-test",
		Reason:           "test",
		Confidence:       0.9,
		ExecutionMode:    control.ExecutionModeApprovalRequired,
		LifecycleState:   control.LifecyclePendingApproval,
		ProposedBy:       "system",
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		BlastRadiusClass: "transport",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPApprove_ViewerKeyForbidden(t *testing.T) {
	srv, actionID := newAuthedTrustServer(t)
	insertPendingForHTTP(t, srv.db, actionID)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/control/actions/"+actionID+"/approve", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("X-API-Key", "viewer-test-key")
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHTTPApprove_ApproverKeyOK(t *testing.T) {
	srv, actionID := newAuthedTrustServer(t)
	insertPendingForHTTP(t, srv.db, actionID)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/control/actions/"+actionID+"/approve", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("X-API-Key", "approver-test-key")
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHTTPReject_ApproveOnlyKeyForbidden(t *testing.T) {
	srv, actionID := newAuthedTrustServer(t)
	insertPendingForHTTP(t, srv.db, actionID)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/control/actions/"+actionID+"/reject", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("X-API-Key", "approve-only-key")
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestIncidentsListIncludesHandoffFields(t *testing.T) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "mel.db")
	cfgPath := writeTestAuthConfig(t, dbPath)
	cfg, _, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.UpsertIncident(models.Incident{
		ID:             "inc-handoff-1",
		Category:       "transport",
		Severity:       "warning",
		Title:          "Test",
		Summary:        "Summary",
		ResourceType:   "transport",
		ResourceID:     "mqtt",
		State:          "open",
		OccurredAt:     "2026-03-19T12:00:00Z",
		OwnerActorID:   "op-alice",
		HandoffSummary: "Shift B: check broker",
		PendingActions: []string{"act-1", "act-2"},
		Metadata:       map[string]any{"k": "v"},
	}); err != nil {
		t.Fatal(err)
	}
	srv := New(cfg, logging.New("error", false), database, meshstate.New(), events.New(),
		func() []transport.Health { return nil },
		func() []policy.Recommendation { return nil },
		nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil)
	req.Header.Set("X-API-Key", "viewer-test-key")
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Recent []struct {
			ID             string   `json:"id"`
			OwnerActorID   string   `json:"owner_actor_id"`
			HandoffSummary string   `json:"handoff_summary"`
			PendingActions []string `json:"pending_actions"`
			State          string   `json:"state"`
		} `json:"recent_incidents"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Recent) != 1 {
		t.Fatalf("expected 1 incident, got %+v", payload)
	}
	if payload.Recent[0].OwnerActorID != "op-alice" || payload.Recent[0].HandoffSummary != "Shift B: check broker" {
		t.Fatalf("unexpected handoff fields: %+v", payload.Recent[0])
	}
	if len(payload.Recent[0].PendingActions) != 2 || payload.Recent[0].PendingActions[0] != "act-1" {
		t.Fatalf("unexpected pending_actions: %+v", payload.Recent[0].PendingActions)
	}
}
