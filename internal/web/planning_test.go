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
	"github.com/mel-project/mel/internal/topology"
	"github.com/mel-project/mel/internal/transport"
)

func TestPlanningBundleEndpoint(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Features.WebUI = false
	cfg.Topology.Enabled = true
	d, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	srv := New(cfg, logging.New("info", false), d, meshstate.New(), events.New(), func() []transport.Health { return nil }, nil, nil, nil, nil, nil, nil)
	srv.SetTopologyStore(topology.NewStore(d))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/planning/bundle", nil)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["evidence_model"] == nil {
		t.Fatalf("missing evidence_model")
	}
	if payload["evidence_flags"] == nil {
		t.Fatalf("missing evidence_flags")
	}
}

func TestPlanningAdvisoryAlertsEndpointIncludesNoAdvisoriesFlag(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Features.WebUI = false
	cfg.Topology.Enabled = true
	d, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	srv := New(cfg, logging.New("info", false), d, meshstate.New(), events.New(), func() []transport.Health { return nil }, nil, nil, nil, nil, nil, nil)
	srv.SetTopologyStore(topology.NewStore(d))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/planning/advisory-alerts", nil)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	flags, ok := payload["evidence_flags"].(map[string]any)
	if !ok {
		t.Fatalf("missing evidence_flags object")
	}
	if flags["no_advisories"] != true {
		t.Fatalf("expected no_advisories true, got %#v", flags["no_advisories"])
	}
}
