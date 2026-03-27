package web

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/fleet"
	"github.com/mel-project/mel/internal/investigation"
	"github.com/mel-project/mel/internal/logging"
	"github.com/mel-project/mel/internal/meshstate"
	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/policy"
	"github.com/mel-project/mel/internal/support"
	"github.com/mel-project/mel/internal/transport"
)

func TestReadyzNotReadyWithoutIngest(t *testing.T) {
	srv := newTestServer(t, []transport.Health{{Name: "tcp", Type: "tcp", State: transport.StateError, Detail: "connect failed"}}, nil)
	for _, path := range []string{"/readyz", "/api/v1/readyz"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.http.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s: unexpected status: %d body=%s", path, rec.Code, rec.Body.String())
		}
		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if payload["ready"] != false {
			t.Fatalf("%s: expected ready=false, got %#v", path, payload["ready"])
		}
		if payload["status"] != "not_ready" {
			t.Fatalf("%s: expected status not_ready, got %#v", path, payload["status"])
		}
		rc, ok := payload["reason_codes"].([]any)
		if !ok || len(rc) == 0 {
			t.Fatalf("%s: expected reason_codes, got %#v", path, payload["reason_codes"])
		}
		transports := payload["transports"].([]any)
		if len(transports) != 1 {
			t.Fatalf("%s: expected one transport snapshot, got %#v", path, payload)
		}
	}
}

func TestReadyzIdleWhenNoTransportsEnabled(t *testing.T) {
	srv := newTestServer(t, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/readyz", nil)
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
		t.Fatalf("expected ready=true for idle, got %#v", payload["ready"])
	}
}

func TestReadyzReadyWhenIngesting(t *testing.T) {
	srv := newTestServer(t, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateIngesting, Detail: "live"}}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/readyz", nil)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["ingest_ready"] != true {
		t.Fatalf("expected ingest_ready, got %#v", payload["ingest_ready"])
	}
}

func TestFleetTruthEndpoint(t *testing.T) {
	srv := newTestServer(t, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fleet/truth", nil)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["instance_id"] == nil || payload["instance_id"] == "" {
		t.Fatalf("expected instance_id, got %#v", payload["instance_id"])
	}
	if payload["truth_posture"] == nil {
		t.Fatalf("expected truth_posture")
	}
}

func TestGlobalTopologyStubNotFakeHealth(t *testing.T) {
	srv := newTestServer(t, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/topology/global", nil)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["global_health"] != nil {
		t.Fatalf("expected no fake global_health, got %#v", payload["global_health"])
	}
	if payload["global_topology_posture"] != "unsupported_without_federation_handlers" {
		t.Fatalf("unexpected stub: %#v", payload)
	}
}

func TestSupportBundleZipIncludesDoctorJSON(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Features.WebUI = false
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(t.TempDir(), "mel.json")
	if err := os.WriteFile(cfgPath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := support.Create(cfg, database, "test-version", cfgPath, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	zb, err := b.ToZip()
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(zb), int64(len(zb)))
	if err != nil {
		t.Fatal(err)
	}
	var sawDoctor, sawBundle bool
	for _, f := range zr.File {
		switch f.Name {
		case "doctor.json":
			sawDoctor = true
		case "bundle.json":
			sawBundle = true
		}
	}
	if !sawBundle {
		t.Fatal("expected bundle.json in zip")
	}
	if !sawDoctor {
		t.Fatal("expected doctor.json in zip")
	}
}

func TestInvestigationsCaseEndpoints(t *testing.T) {
	summary := investigation.Summary{
		GeneratedAt: "2026-03-27T00:00:00Z",
		Cases: []investigation.Case{{
			ID:                "case:evidence:freshness",
			Kind:              investigation.CaseEvidenceFreshnessGap,
			Status:            investigation.CaseStatusActiveAttention,
			Attention:         investigation.AttentionHigh,
			Certainty:         0.6,
			Title:             "Current live evidence is not proven",
			Summary:           "No enabled transport is proving live ingest.",
			AttentionReason:   "No transport is actively ingesting",
			WhyItMatters:      "Missing fresh evidence makes silence ambiguous.",
			Scope:             investigation.ScopeLocal,
			FindingIDs:        []string{"no_active_ingest:transport"},
			EvidenceGapIDs:    []string{"missing_expected_reporters:local:transports"},
			RecommendationIDs: []string{"rec_verify_transports"},
			SafeToConsider:    "Treat current state as unconfirmed until ingest resumes.",
			OutOfScope:        "Do not conclude the mesh is quiet from missing ingest.",
			ObservedAt:        "2026-03-27T00:00:00Z",
			UpdatedAt:         "2026-03-27T00:00:00Z",
		}},
		Findings: []investigation.Finding{{
			ID:           "no_active_ingest:transport",
			Code:         "no_active_ingest",
			Category:     investigation.CategoryTransport,
			Attention:    investigation.AttentionHigh,
			Certainty:    0.6,
			Title:        "No transport is actively ingesting",
			Explanation:  "Enabled transports exist but none are proving live ingest.",
			WhyItMatters: "Missing fresh evidence makes silence ambiguous.",
			Scope:        investigation.ScopeLocal,
			ObservedAt:   "2026-03-27T00:00:00Z",
			GeneratedAt:  "2026-03-27T00:00:00Z",
			Source:       "test",
		}},
		EvidenceGaps: []investigation.EvidenceGap{{
			ID:          "missing_expected_reporters:local:transports",
			Reason:      investigation.GapMissingExpectedReporters,
			Title:       "No active transport reporters",
			Explanation: "Enabled transports are not reporting live ingest.",
			Impact:      "Current-state certainty is limited.",
			Scope:       investigation.ScopeLocal,
			GeneratedAt: "2026-03-27T00:00:00Z",
		}},
		Recommendations: []investigation.Recommendation{{
			ID:              "rec_verify_transports",
			Code:            investigation.RecInspectTransport,
			Action:          "Verify transport connectivity",
			Rationale:       "No transport is proving live ingest.",
			ActionAuthority: "operator_only",
			Scope:           investigation.ScopeLocal,
			GeneratedAt:     "2026-03-27T00:00:00Z",
		}},
		CaseCounts: investigation.CaseCounts{
			TotalCases:           1,
			ActiveAttentionCases: 1,
		},
	}

	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Features.WebUI = false
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	srv := New(cfg, logging.New("info", false), database, meshstate.New(), events.New(),
		func() []transport.Health { return nil },
		func() []policy.Recommendation { return nil },
		nil, nil, nil, nil, nil,
		func() investigation.Summary { return summary })

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/investigations/cases", nil)
	listRec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("unexpected list status: %d body=%s", listRec.Code, listRec.Body.String())
	}
	var listPayload map[string]any
	if err := json.Unmarshal(listRec.Body.Bytes(), &listPayload); err != nil {
		t.Fatal(err)
	}
	if int(listPayload["count"].(float64)) != 1 {
		t.Fatalf("expected one case, got %#v", listPayload)
	}

	showReq := httptest.NewRequest(http.MethodGet, "/api/v1/investigations/cases/case:evidence:freshness", nil)
	showRec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(showRec, showReq)
	if showRec.Code != http.StatusOK {
		t.Fatalf("unexpected detail status: %d body=%s", showRec.Code, showRec.Body.String())
	}
	var detailPayload map[string]any
	if err := json.Unmarshal(showRec.Body.Bytes(), &detailPayload); err != nil {
		t.Fatal(err)
	}
	casePayload, ok := detailPayload["case"].(map[string]any)
	if !ok || casePayload["id"] != "case:evidence:freshness" {
		t.Fatalf("unexpected case payload: %#v", detailPayload["case"])
	}
}

func TestInvestigationCaseTimelineEndpoint(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Features.WebUI = false
	cfg.Transports = []config.TransportConfig{{
		Name:     "mqtt",
		Type:     "mqtt",
		Enabled:  true,
		Endpoint: "127.0.0.1:1883",
		Topic:    "msh/test",
		ClientID: "mel-test",
	}}
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.UpsertTransportRuntime(db.TransportRuntime{
		Name:          "mqtt",
		Type:          "mqtt",
		Source:        "127.0.0.1:1883",
		Enabled:       true,
		State:         transport.StateFailed,
		Detail:        "connect failed",
		LastError:     "broker unreachable",
		FailureCount:  2,
		UpdatedAt:     "2026-03-27T00:00:00Z",
		LastFailureAt: "2026-03-27T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	if err := database.InsertTimelineEvent(db.TimelineEvent{
		EventID:        "evt-transport-1",
		EventTime:      "2026-03-27T00:00:00Z",
		EventType:      "transport_runtime_note",
		Summary:        "mqtt failed to connect",
		Severity:       "warning",
		ActorID:        "system",
		ResourceID:     "mqtt",
		ScopePosture:   "local",
		TimingPosture:  "local_ordered",
		Details:        map[string]any{"transport": "mqtt"},
	}); err != nil {
		t.Fatal(err)
	}
	runtimeStates, err := database.TransportRuntimeStatuses()
	if err != nil {
		t.Fatal(err)
	}
	srv := New(cfg, logging.New("info", false), database, meshstate.New(), events.New(),
		func() []transport.Health {
			return []transport.Health{{
				Name:         "mqtt",
				Type:         "mqtt",
				Source:       "127.0.0.1:1883",
				State:        transport.StateFailed,
				LastError:    "broker unreachable",
				FailureCount: 2,
			}}
		},
		func() []policy.Recommendation { return nil },
		nil, nil, nil, nil, nil,
		func() investigation.Summary {
			return investigation.Derive(cfg, database, []transport.Health{{
				Name:         "mqtt",
				Type:         "mqtt",
				Source:       "127.0.0.1:1883",
				State:        transport.StateFailed,
				LastError:    "broker unreachable",
				FailureCount: 2,
			}}, runtimeStates, time.Date(2026, 3, 27, 1, 0, 0, 0, time.UTC))
		})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/investigations/cases/case:transport:mqtt/timeline", nil)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected timeline status: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["case_id"] != "case:transport:mqtt" {
		t.Fatalf("unexpected case id payload: %#v", payload)
	}
	timing, ok := payload["timing"].(map[string]any)
	if !ok || timing["primary_posture"] == "" {
		t.Fatalf("expected timing posture in payload, got %#v", payload["timing"])
	}
	if payload["linked_event_count"].(float64) < 1 {
		t.Fatalf("expected linked events in payload, got %#v", payload)
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
	srv := New(cfg, logging.New("info", false), database, meshstate.New(), events.New(),
		func() []transport.Health { return nil },
		func() []policy.Recommendation { return nil },
		nil, nil, nil, nil, nil,
		func() investigation.Summary { return investigation.Summary{} })
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

func TestFleetImportBatchEndpointsExposeOfflineAuditDetails(t *testing.T) {
	const batchID = "batch-web-1"
	srv := newTestServer(t, nil, func(database *db.DB) {
		seedRemoteImportBatchFixture(t, database, batchID)
	})

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/fleet/imports?limit=10", nil)
	listRec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("unexpected list status: %d body=%s", listRec.Code, listRec.Body.String())
	}
	var listPayload map[string]any
	if err := json.Unmarshal(listRec.Body.Bytes(), &listPayload); err != nil {
		t.Fatal(err)
	}
	if int(listPayload["count"].(float64)) != 1 {
		t.Fatalf("expected one batch, got %#v", listPayload)
	}
	if note, _ := listPayload["note"].(string); note == "" {
		t.Fatalf("expected offline-audit note, got %#v", listPayload["note"])
	}
	summaries, ok := listPayload["summaries"].([]any)
	if !ok || len(summaries) != 1 {
		t.Fatalf("expected one batch summary, got %#v", listPayload["summaries"])
	}
	summary := summaries[0].(map[string]any)
	if summary["batch_id"] != batchID {
		t.Fatalf("unexpected batch summary %#v", summary)
	}
	if summary["partial_success"] != false {
		t.Fatalf("expected full accepted batch summary, got %#v", summary)
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/api/v1/fleet/imports/"+batchID, nil)
	detailRec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(detailRec, detailReq)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("unexpected detail status: %d body=%s", detailRec.Code, detailRec.Body.String())
	}
	var detailPayload map[string]any
	if err := json.Unmarshal(detailRec.Body.Bytes(), &detailPayload); err != nil {
		t.Fatal(err)
	}
	batch := detailPayload["batch"].(map[string]any)
	if batch["id"] != batchID {
		t.Fatalf("unexpected batch row %#v", batch)
	}
	items, ok := detailPayload["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one batch item, got %#v", detailPayload["items"])
	}
	item := items[0].(map[string]any)
	if item["batch_id"] != batchID {
		t.Fatalf("expected item to retain batch id, got %#v", item)
	}
	inspection := detailPayload["inspection"].(map[string]any)
	inspectionBatch := inspection["batch"].(map[string]any)
	validation := inspectionBatch["validation"].(map[string]any)
	if validation["outcome"] != string(fleet.ValidationAcceptedWithCaveats) {
		t.Fatalf("expected accepted_with_caveats batch validation, got %#v", validation)
	}
	itemInspections, ok := inspection["item_inspections"].([]any)
	if !ok || len(itemInspections) != 1 {
		t.Fatalf("expected one item inspection, got %#v", inspection["item_inspections"])
	}
	itemInspection := itemInspections[0].(map[string]any)
	source := itemInspection["source"].(map[string]any)
	if source["source_type"] != "file" {
		t.Fatalf("expected file source type, got %#v", source)
	}
	provenance := itemInspection["provenance"].(map[string]any)
	if provenance["origin_instance_id"] != "remote-1" {
		t.Fatalf("expected origin instance to survive inspection, got %#v", provenance)
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
	return New(cfg, logging.New("info", false), database, meshstate.New(), events.New(),
		func() []transport.Health { return health },
		func() []policy.Recommendation { return nil },
		nil, nil, nil, nil, nil,
		func() investigation.Summary { return investigation.Summary{} })
}

func seedRemoteImportBatchFixture(t *testing.T, database *db.DB, batchID string) {
	t.Helper()
	localID, err := database.EnsureInstanceID()
	if err != nil {
		t.Fatal(err)
	}
	bundle := fleet.RemoteEvidenceBundle{
		SchemaVersion:           fleet.RemoteEvidenceBundleSchemaVersion,
		Kind:                    fleet.RemoteEvidenceBundleKind,
		ClaimedOriginInstanceID: "remote-1",
		ClaimedOriginSiteID:     "site-a",
		ClaimedFleetID:          "fleet-remote",
		Evidence: fleet.EvidenceEnvelope{
			EvidenceClass:       fleet.EvidenceClassPacketObservation,
			OriginInstanceID:    "remote-1",
			OriginSiteID:        "site-a",
			OriginClass:         fleet.OriginRemoteReported,
			CorrelationID:       "corr-web-1",
			ObservedAt:          "2026-01-01T00:00:00Z",
			ReceivedAt:          "2026-01-01T00:00:05Z",
			PhysicalUncertainty: fleet.PhysicalUncertaintyDefault,
		},
		Event: &fleet.EventEnvelope{
			EventID:          "evt-web-1",
			EventType:        "packet_observation",
			Summary:          "remote packet observed",
			OriginInstanceID: "remote-1",
			OriginSiteID:     "site-a",
			CorrelationID:    "corr-web-1",
			ObservedAt:       "2026-01-01T00:00:00Z",
			RecordedAt:       "2026-01-01T00:00:05Z",
		},
	}
	rawPayload, err := json.Marshal(fleet.RemoteEvidenceBatch{
		SchemaVersion:     fleet.RemoteEvidenceBatchSchemaVersion,
		Kind:              fleet.RemoteEvidenceBatchKind,
		ExportedAt:        "2026-01-01T00:09:00Z",
		ClaimedOrigin:     fleet.RemoteEvidenceBatchClaimedOrigin{InstanceID: "remote-1", SiteID: "site-a", FleetID: "fleet-remote"},
		CapabilityPosture: fleet.DefaultCapabilityPosture(),
		SourceContext:     fleet.RemoteEvidenceImportSource{SourceType: "file", SourceName: "remote-evidence.json", SourcePath: "/tmp/remote-evidence.json"},
		Items:             []fleet.RemoteEvidenceBundle{bundle},
	})
	if err != nil {
		t.Fatal(err)
	}
	batchValidation, err := json.Marshal(fleet.RemoteEvidenceBatchValidation{
		Outcome:                  fleet.ValidationAcceptedWithCaveats,
		Reasons:                  []fleet.ValidationReasonCode{fleet.CaveatNotCryptographicallyVerified, fleet.CaveatHistoricalImportOnly, fleet.CaveatUnverifiedOrigin},
		TrustPosture:             fleet.TrustPostureImportedReadOnly,
		AuthenticityNote:         "Import authenticity is not cryptographically verified in core MEL; treat claimed origin as unverified unless checked outside MEL.",
		OfflineOnlyNote:          "Remote evidence import is offline/file-scoped in core MEL; it does not establish live federation, remote execution, or fleet-wide authority.",
		Summary:                  "Accepted 1 item with caveats: offline import remains read-only, historical, and authenticity-unverified.",
		StructurallyValid:        true,
		ItemCount:                1,
		AcceptedWithCaveatsCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	capabilityJSON, err := json.Marshal(fleet.DefaultCapabilityPosture())
	if err != nil {
		t.Fatal(err)
	}
	batch := db.RemoteImportBatchRecord{
		ID:                       batchID,
		ImportedAt:               "2026-01-01T00:10:00Z",
		LocalInstanceID:          localID,
		SourceType:               "file",
		SourceName:               "remote-evidence.json",
		SourcePath:               "/tmp/remote-evidence.json",
		FormatKind:               fleet.RemoteEvidenceBatchKind,
		SchemaVersion:            fleet.RemoteEvidenceBatchSchemaVersion,
		ClaimedOriginInstanceID:  "remote-1",
		ClaimedOriginSiteID:      "site-a",
		ClaimedFleetID:           "fleet-remote",
		ExportedAt:               "2026-01-01T00:09:00Z",
		CapabilityPosture:        capabilityJSON,
		Validation:               batchValidation,
		RawPayload:               rawPayload,
		ItemCount:                1,
		AcceptedWithCaveatsCount: 1,
		Note:                     "Offline remote evidence batch for audit and investigation only.",
	}
	bundleJSON, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	evidenceJSON, err := json.Marshal(bundle.Evidence)
	if err != nil {
		t.Fatal(err)
	}
	eventJSON, err := json.Marshal(bundle.Event)
	if err != nil {
		t.Fatal(err)
	}
	itemValidation, err := json.Marshal(fleet.RemoteEvidenceValidation{
		Outcome:          fleet.ValidationAcceptedWithCaveats,
		Reasons:          []fleet.ValidationReasonCode{fleet.CaveatNotCryptographicallyVerified, fleet.CaveatHistoricalImportOnly, fleet.CaveatPartialObservationOnly, fleet.CaveatReceiveDiffersFromObserved},
		TrustPosture:     fleet.TrustPostureImportedReadOnly,
		AuthenticityNote: "Import authenticity is not cryptographically verified in core MEL; treat claimed origin as unverified unless checked outside MEL.",
		OrderingPosture:  fleet.TimingOrderReceiveDiffersFromObserved,
		Summary:          "Accepted with caveats: structurally valid offline remote evidence; authenticity and authority are not verified; import remains historical/read-only.",
	})
	if err != nil {
		t.Fatal(err)
	}
	item := db.ImportedRemoteEvidenceRecord{
		ID:                      "imp-web-1",
		BatchID:                 batchID,
		ItemID:                  batchID + ":001",
		SequenceNo:              1,
		ImportedAt:              "2026-01-01T00:10:00Z",
		LocalInstanceID:         localID,
		SourceType:              "file",
		SourceName:              "remote-evidence.json",
		SourcePath:              "/tmp/remote-evidence.json",
		ValidationStatus:        string(fleet.ValidationAcceptedWithCaveats),
		Validation:              itemValidation,
		Bundle:                  bundleJSON,
		Evidence:                evidenceJSON,
		Event:                   eventJSON,
		ClaimedOriginInstanceID: "remote-1",
		ClaimedOriginSiteID:     "site-a",
		ClaimedFleetID:          "fleet-remote",
		OriginInstanceID:        "remote-1",
		OriginSiteID:            "site-a",
		EvidenceClass:           string(fleet.EvidenceClassPacketObservation),
		ObservationOriginClass:  string(fleet.OriginRemoteReported),
		CorrelationID:           "corr-web-1",
		ObservedAt:              "2026-01-01T00:00:00Z",
		ReceivedAt:              "2026-01-01T00:00:05Z",
		TimingPosture:           string(fleet.TimingOrderReceiveDiffersFromObserved),
		MergeDisposition:        "raw_only",
		MergeCorrelationID:      "corr-web-1",
		Rejected:                false,
	}
	timelineEvents := []db.TimelineEvent{
		{
			EventID:            batchID,
			EventTime:          "2026-01-01T00:10:00Z",
			EventType:          "remote_import_batch",
			Summary:            "remote import batch accepted_with_caveats",
			Severity:           "info",
			ActorID:            "tester",
			ResourceID:         batchID,
			ScopePosture:       "remote_import_batch",
			TimingPosture:      string(fleet.TimingOrderLocalOrdered),
			MergeDisposition:   "raw_only",
			MergeCorrelationID: "corr-web-1",
			ImportID:           batchID,
			Details:            map[string]any{"batch_id": batchID, "offline_only_federation": true},
		},
	}
	if err := database.PersistRemoteImportBatch(batch, []db.ImportedRemoteEvidenceRecord{item}, timelineEvents); err != nil {
		t.Fatal(err)
	}
}

func TestIncidentsEndpointReturnsGroupedTransportIncidents(t *testing.T) {
	srv := newTestServer(t, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateRetrying}}, func(database *db.DB) {
		if err := database.UpsertIncident(models.Incident{
			ID:           "inc-mqtt-timeout",
			Category:     "transport",
			Severity:     "warning",
			Title:        "Transport stall",
			Summary:      "timeout_stall on mqtt",
			ResourceType: "transport",
			ResourceID:   "mqtt",
			State:        "open",
			OccurredAt:   "2026-03-19T12:00:00Z",
			Metadata:     map[string]any{"reason": "timeout_stall"},
		}); err != nil {
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
	raw, ok := payload["recent_incidents"]
	if !ok {
		t.Fatalf("missing recent_incidents in %#v", payload)
	}
	incidents, ok := raw.([]any)
	if !ok {
		t.Fatalf("recent_incidents type %T, want []any", raw)
	}
	if len(incidents) == 0 {
		t.Fatalf("expected incidents, got %#v", payload)
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
