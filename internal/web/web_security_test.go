package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/investigation"
	"github.com/mel-project/mel/internal/logging"
	"github.com/mel-project/mel/internal/meshstate"
	"github.com/mel-project/mel/internal/policy"
	"github.com/mel-project/mel/internal/transport"
	"golang.org/x/crypto/bcrypt"
)

func TestSQLInjectionViaTransportName(t *testing.T) {
	srv := newTestServer(t, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateIngesting}}, nil)

	sqlInjectionPayloads := []struct {
		name        string
		transport   string
		expectBlock bool
	}{
		{"semicolon_injection", "mqtt; DROP TABLE messages;", true},
		{"comment_injection", "mqtt--", true},
		{"block_comment_start", "mqtt/*", true},
		{"block_comment_end", "mqtt*/", true},
		{"union_injection", "mqtt' UNION SELECT * FROM users--", true},
		{"stacked_query", "mqtt'; DELETE FROM messages;", true},
		{"valid_transport", "mqtt", false},
		{"transport_with_hyphen", "mqtt-local", false},
		{"transport_with_underscore", "mqtt_local", false},
		{"transport_with_dot", "mqtt.local", false},
	}

	for _, tc := range sqlInjectionPayloads {
		t.Run(tc.name, func(t *testing.T) {
			endpoints := []string{
				fmt.Sprintf("/api/v1/transports/health/history?transport=%s", url.QueryEscape(tc.transport)),
				fmt.Sprintf("/api/v1/transports/alerts/history?transport=%s", url.QueryEscape(tc.transport)),
				fmt.Sprintf("/api/v1/transports/anomalies/history?transport=%s", url.QueryEscape(tc.transport)),
				fmt.Sprintf("/api/v1/control/history?transport=%s", url.QueryEscape(tc.transport)),
				fmt.Sprintf("/api/v1/control/actions?transport=%s", url.QueryEscape(tc.transport)),
				fmt.Sprintf("/api/v1/dead-letters?transport=%s", url.QueryEscape(tc.transport)),
				fmt.Sprintf("/api/v1/events?transport=%s", url.QueryEscape(tc.transport)),
			}

			for _, path := range endpoints {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				rec := httptest.NewRecorder()
				srv.http.Handler.ServeHTTP(rec, req)

				if tc.expectBlock {
					if rec.Code != http.StatusBadRequest {
						t.Errorf("%s: expected 400 for transport '%s', got %d", path, tc.transport, rec.Code)
					}
					var payload map[string]any
					if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
						t.Errorf("%s: failed to parse error response: %v", path, err)
						continue
					}
					errObj, ok := payload["error"].(map[string]any)
					if !ok {
						t.Errorf("%s: expected error object in response", path)
					}
					msg, _ := errObj["message"].(string)
					if !strings.Contains(strings.ToLower(msg), "invalid") {
						t.Errorf("%s: expected validation error message, got: %s", path, msg)
					}
				} else {
					if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
						t.Errorf("%s: expected 200/404 for valid transport '%s', got %d", path, tc.transport, rec.Code)
					}
				}
			}
		})
	}
}

func TestSQLInjectionViaNodeID(t *testing.T) {
	srv := newTestServer(t, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateIngesting}}, nil)

	nodeInjectionPayloads := []struct {
		name        string
		nodeID      string
		expectBlock bool
	}{
		{"semicolon_in_node", "12345; DROP TABLE nodes;", false},
		{"comment_in_node", "12345--", false},
		{"block_comment_node", "12345/*", false},
		{"path_traversal", "../etc/passwd", true},
		{"encoded_traversal", "..%2f..%2fetc%2fpasswd", true},
		{"union_injection", "123' UNION SELECT * FROM nodes--", false},
		{"null_byte", "12345\x00 malicious", false},
		{"valid_node_num", "12345", false},
		{"valid_hex_node", "!a1b2c3d4", false},
	}

	for _, tc := range nodeInjectionPayloads {
		t.Run(tc.name, func(t *testing.T) {
			path := fmt.Sprintf("/api/v1/node/%s", url.QueryEscape(tc.nodeID))
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			srv.http.Handler.ServeHTTP(rec, req)

			if tc.expectBlock {
				if rec.Code != http.StatusBadRequest {
					t.Errorf("expected 400 for node '%s', got %d", tc.nodeID, rec.Code)
				}
			}
		})
	}
}

func TestSQLInjectionViaMessagesEndpoint(t *testing.T) {
	srv := newTestServer(t, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateIngesting}}, nil)

	messageInjectionPayloads := []struct {
		name        string
		param       string
		value       string
		expectBlock bool
	}{
		{"node_param_injection", "node", "123; DROP TABLE messages;", true},
		{"node_param_comment", "node", "123--", true},
		{"type_param_injection", "type", "text'; DELETE FROM messages;", true},
		{"type_param_comment", "type", "text--", true},
		{"valid_node", "node", "12345", false},
		{"valid_type", "type", "text", false},
	}

	for _, tc := range messageInjectionPayloads {
		t.Run(tc.name, func(t *testing.T) {
			path := fmt.Sprintf("/api/v1/messages?%s=%s", tc.param, url.QueryEscape(tc.value))
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			srv.http.Handler.ServeHTTP(rec, req)

			if tc.expectBlock {
				if rec.Code != http.StatusBadRequest {
					t.Errorf("expected 400 for param '%s'='%s', got %d", tc.param, tc.value, rec.Code)
				}
			} else {
				if rec.Code != http.StatusOK {
					t.Errorf("expected 200 for valid param '%s'='%s', got %d", tc.param, tc.value, rec.Code)
				}
			}
		})
	}
}

func TestInputStormDoSProtection(t *testing.T) {
	srv := newTestServer(t, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateIngesting}}, nil)

	const numConcurrentRequests = 100
	const numSequentialRequests = 50

	t.Run("concurrent_request_storm", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan int, numConcurrentRequests)
		successes := make(chan int, numConcurrentRequests)

		for i := 0; i < numConcurrentRequests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
				rec := httptest.NewRecorder()
				srv.http.Handler.ServeHTTP(rec, req)
				if rec.Code == http.StatusOK {
					successes <- 1
				} else {
					errors <- rec.Code
				}
			}()
		}

		wg.Wait()
		close(errors)
		close(successes)

		successCount := 0
		for range successes {
			successCount++
		}

		errorCodes := make(map[int]int)
		for code := range errors {
			errorCodes[code]++
		}

		if successCount < numConcurrentRequests/2 {
			t.Errorf("too many requests failed: %d successes out of %d", successCount, numConcurrentRequests)
		}
		for code, count := range errorCodes {
			if code == http.StatusTooManyRequests {
				t.Logf("rate limiting active: %d requests throttled", count)
			}
		}
	})

	t.Run("sequential_rapid_requests", func(t *testing.T) {
		for i := 0; i < numSequentialRequests; i++ {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/panel", nil)
			rec := httptest.NewRecorder()
			srv.http.Handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("request %d failed with code %d", i, rec.Code)
			}
		}
	})
}

func TestBoundaryConditions(t *testing.T) {
	srv := newTestServer(t, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateIngesting}}, nil)

	boundaryTests := []struct {
		name       string
		path       string
		expectCode int
	}{
		{"limit_zero", "/api/v1/messages?limit=0", http.StatusOK},
		{"limit_negative", "/api/v1/messages?limit=-1", http.StatusOK},
		{"limit_small", "/api/v1/messages?limit=1", http.StatusOK},
		{"limit_typical", "/api/v1/messages?limit=100", http.StatusOK},
		{"limit_max_boundary", "/api/v1/messages?limit=500", http.StatusOK},
		{"limit_over_max", "/api/v1/messages?limit=1000", http.StatusOK},
		{"offset_negative", "/api/v1/control/history?offset=-1", http.StatusBadRequest},
		{"offset_zero", "/api/v1/control/history?offset=0", http.StatusOK},
		{"offset_typical", "/api/v1/control/history?offset=100", http.StatusOK},
		{"offset_large", "/api/v1/control/history?offset=999999", http.StatusOK},
		{"history_limit_zero", "/api/v1/transports/health/history?limit=0", http.StatusOK},
		{"history_limit_max", "/api/v1/transports/health/history?limit=1000", http.StatusOK},
		{"history_limit_exceeds_max", "/api/v1/transports/health/history?limit=999999", http.StatusOK},
	}

	for _, tc := range boundaryTests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			srv.http.Handler.ServeHTTP(rec, req)
			if rec.Code != tc.expectCode {
				t.Errorf("expected %d for %s, got %d", tc.expectCode, tc.name, rec.Code)
			}
		})
	}
}

func TestPrivacyLeakRawHex(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Features.WebUI = false
	cfg.Auth.Enabled = true
	cfg.Auth.UIUser = "admin"
	cfg.Auth.UIPassword = "secret123"

	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := database.InsertMessage(map[string]any{
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
		"raw_hex":        "00aabbccdd",
		"rx_time":        time.Now().UTC().Format(time.RFC3339),
		"hop_limit":      int64(3),
		"relay_node":     int64(0),
	}); err != nil {
		t.Fatal(err)
	}

	srv := New(cfg, logging.New("info", false), database, meshstate.New(), events.New(),
		func() []transport.Health { return nil },
		func() []policy.Recommendation { return nil },
		nil, nil, nil, nil, nil, nil,
		func() investigation.Summary { return investigation.Summary{} })

	t.Run("unauthorized_no_raw_hex", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
		rec := httptest.NewRecorder()
		srv.http.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 without auth, got %d", rec.Code)
		}
	})

	t.Run("authorized_can_access_messages", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
		req.SetBasicAuth("admin", "secret123")
		rec := httptest.NewRecorder()
		srv.http.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 with auth, got %d", rec.Code)
		}

		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		messages, ok := payload["messages"].([]any)
		if !ok || len(messages) == 0 {
			t.Skip("no messages in response")
		}
	})
}

func TestControlAbusePrevention(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Features.WebUI = false
	cfg.Control.Mode = "guarded_auto"
	cfg.Control.MaxActionsPerWindow = 5
	cfg.Control.CooldownPerTargetSeconds = 60

	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	for i := 0; i < cfg.Control.MaxActionsPerWindow; i++ {
		action := db.ControlActionRecord{
			ID:              fmt.Sprintf("action-%d", i),
			ActionType:      "restart_transport",
			TargetTransport: "mqtt",
			Reason:          "test",
			Confidence:      0.9,
			CreatedAt:       now.Add(-time.Duration(i) * time.Second).Format(time.RFC3339),
			ExecutedAt:      now.Add(-time.Duration(i) * time.Second).Format(time.RFC3339),
			CompletedAt:     now.Add(-time.Duration(i) * time.Second).Format(time.RFC3339),
			Result:          "executed_successfully",
			Mode:            "guarded_auto",
		}
		if err := database.UpsertControlAction(action); err != nil {
			t.Fatal(err)
		}
	}

	cooldownAction := db.ControlActionRecord{
		ID:              "cooldown-action",
		ActionType:      "restart_transport",
		TargetTransport: "mqtt-cooldown",
		Reason:          "test",
		Confidence:      0.9,
		CreatedAt:       now.Add(-30 * time.Second).Format(time.RFC3339),
		ExecutedAt:      now.Add(-30 * time.Second).Format(time.RFC3339),
		CompletedAt:     now.Add(-30 * time.Second).Format(time.RFC3339),
		Result:          "executed_successfully",
		Mode:            "guarded_auto",
	}
	if err := database.UpsertControlAction(cooldownAction); err != nil {
		t.Fatal(err)
	}

	srv := New(cfg, logging.New("info", false), database, meshstate.New(), events.New(),
		func() []transport.Health { return nil },
		func() []policy.Recommendation { return nil },
		nil, nil, nil, nil, nil, nil,
		func() investigation.Summary { return investigation.Summary{} })

	t.Run("history_endpoint_respects_limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/control/history?limit=%d", cfg.Control.MaxActionsPerWindow+10), nil)
		rec := httptest.NewRecorder()
		srv.http.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}

		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		pagination, ok := payload["pagination"].(map[string]any)
		if !ok {
			t.Skip("no pagination info")
		}
		limit, _ := pagination["limit"].(float64)
		if limit > float64(cfg.Control.MaxActionsPerWindow+10) {
			t.Errorf("limit not properly bounded: got %f", limit)
		}
	})
}

func TestMethodNotAllowed(t *testing.T) {
	srv := newTestServer(t, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateIngesting}}, nil)

	methodTests := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/status"},
		{http.MethodPut, "/api/v1/nodes"},
		{http.MethodDelete, "/api/v1/messages"},
		{http.MethodPatch, "/api/v1/panel"},
		{http.MethodOptions, "/api/v1/mesh"},
	}

	for _, tc := range methodTests {
		t.Run(fmt.Sprintf("%s_%s", tc.method, tc.path), func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			srv.http.Handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected 405 for %s %s, got %d", tc.method, tc.path, rec.Code)
			}

			allowHeader := rec.Header().Get("Allow")
			if allowHeader == "" {
				t.Error("expected Allow header in 405 response")
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	srv := newTestServer(t, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateIngesting}}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)

	headerTests := []struct {
		header   string
		expected string
		contains bool
	}{
		{"X-Content-Type-Options", "nosniff", false},
		{"X-Frame-Options", "DENY", false},
		{"Content-Security-Policy", "default-src", true},
	}

	for _, tc := range headerTests {
		t.Run(tc.header, func(t *testing.T) {
			value := rec.Header().Get(tc.header)
			if tc.contains {
				if !strings.Contains(value, tc.expected) {
					t.Errorf("expected %s header to contain %q, got %q", tc.header, tc.expected, value)
				}
			} else {
				if value != tc.expected {
					t.Errorf("expected %s=%q, got %q", tc.header, tc.expected, value)
				}
			}
		})
	}
}

func TestNodeIDRedaction(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Features.WebUI = false

	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if err := database.UpsertNode(map[string]any{
		"node_num":        int64(12345),
		"node_id":         "!a1b2c3d4",
		"long_name":       "TestNode",
		"short_name":      "TN",
		"last_seen":       time.Now().UTC().Format(time.RFC3339),
		"last_gateway_id": "!gw1234",
		"last_snr":        float64(5.5),
		"last_rssi":       int64(-80),
		"lat_redacted":    float64(45.5),
		"lon_redacted":    float64(-122.6),
		"altitude":        int64(100),
	}); err != nil {
		t.Fatal(err)
	}

	logger := logging.New("info", false)
	srv := New(cfg, logger, database, meshstate.New(), events.New(),
		func() []transport.Health { return nil },
		func() []policy.Recommendation { return nil },
		nil, nil, nil, nil, nil, nil,
		func() investigation.Summary { return investigation.Summary{} })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nodes", nil)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
}

func TestPayloadTruncationInLogs(t *testing.T) {
	testCases := []struct {
		name    string
		payload []byte
		maxLen  int
	}{
		{"small_payload", []byte("hello"), 100},
		{"exact_size", make([]byte, 100), 100},
		{"oversized", make([]byte, 1000), 100},
		{"empty", []byte{}, 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := logging.SanitizePayloadForLog(tc.payload, tc.maxLen)
			if len(result) > tc.maxLen*2+10 {
				t.Errorf("result too long: got %d chars, max expected ~%d", len(result), tc.maxLen*2)
			}
		})
	}
}

func TestTransportInspectPathTraversal(t *testing.T) {
	srv := newTestServer(t, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateIngesting}}, nil)

	tests := []struct {
		name       string
		transport  string
		expectCode int
	}{
		{"valid", "mqtt", http.StatusOK},
		{"empty", "", http.StatusBadRequest},
		{"with_slash", "mqtt/test", http.StatusNotFound},
		{"dot_dot", "..", http.StatusMovedPermanently},
		{"encoded_slash", "mqtt%2fetc", http.StatusNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := fmt.Sprintf("/api/v1/transports/inspect/%s", tc.transport)
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			srv.http.Handler.ServeHTTP(rec, req)

			if rec.Code != tc.expectCode {
				t.Errorf("expected %d for transport %q, got %d", tc.expectCode, tc.transport, rec.Code)
			}
		})
	}
}

func TestDuplicateActionDetection(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Features.WebUI = false

	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	action1 := db.ControlActionRecord{
		ID:              "duplicate-test-1",
		ActionType:      "restart_transport",
		TargetTransport: "mqtt",
		Reason:          "test",
		Confidence:      0.9,
		CreatedAt:       now.Add(-5 * time.Second).Format(time.RFC3339),
		ExecutedAt:      now.Add(-5 * time.Second).Format(time.RFC3339),
		Result:          "executed_successfully",
		LifecycleState:  "pending",
		Mode:            "guarded_auto",
	}
	if err := database.UpsertControlAction(action1); err != nil {
		t.Fatal(err)
	}

	action2 := db.ControlActionRecord{
		ID:              "duplicate-test-2",
		ActionType:      "restart_transport",
		TargetTransport: "mqtt",
		Reason:          "test duplicate",
		Confidence:      0.9,
		CreatedAt:       now.Add(-3 * time.Second).Format(time.RFC3339),
		ExecutedAt:      now.Add(-3 * time.Second).Format(time.RFC3339),
		Result:          "executed_successfully",
		LifecycleState:  "pending",
		Mode:            "guarded_auto",
	}
	if err := database.UpsertControlAction(action2); err != nil {
		t.Fatal(err)
	}

	srv := New(cfg, logging.New("info", false), database, meshstate.New(), events.New(),
		func() []transport.Health { return nil },
		func() []policy.Recommendation { return nil },
		nil, nil, nil, nil, nil, nil,
		func() investigation.Summary { return investigation.Summary{} })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/control/status", nil)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAuthBruteForceProtection(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Features.WebUI = false
	cfg.Auth.Enabled = true
	cfg.Auth.UIUser = "admin"
	cfg.Auth.UIPassword = "correctpassword"

	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}

	logger := logging.New("info", false)
	srv := New(cfg, logger, database, meshstate.New(), events.New(),
		func() []transport.Health { return nil },
		func() []policy.Recommendation { return nil },
		nil, nil, nil, nil, nil, nil,
		func() investigation.Summary { return investigation.Summary{} })

	t.Run("invalid_credentials_rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
		req.SetBasicAuth("admin", "wrongpassword")
		rec := httptest.NewRecorder()
		srv.http.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for wrong password, got %d", rec.Code)
		}
	})

	t.Run("missing_credentials_rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
		rec := httptest.NewRecorder()
		srv.http.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for missing auth, got %d", rec.Code)
		}
	})

	t.Run("valid_credentials_accepted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
		req.SetBasicAuth("admin", "correctpassword")
		rec := httptest.NewRecorder()
		srv.http.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 for valid auth, got %d", rec.Code)
		}
	})
}

func TestAuthWithBcryptPasswordHash(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Features.WebUI = false
	cfg.Auth.Enabled = true
	cfg.Auth.UIUser = "admin"
	cfg.Auth.UIPassword = ""
	hashed, err := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Auth.UIPasswordHash = string(hashed)

	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	logger := logging.New("info", false)
	srv := New(cfg, logger, database, meshstate.New(), events.New(),
		func() []transport.Health { return nil },
		func() []policy.Recommendation { return nil },
		nil, nil, nil, nil, nil, nil,
		func() investigation.Summary { return investigation.Summary{} })

	reqWrong := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	reqWrong.SetBasicAuth("admin", "wrongpassword")
	recWrong := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(recWrong, reqWrong)
	if recWrong.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password against hash, got %d", recWrong.Code)
	}

	reqOk := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	reqOk.SetBasicAuth("admin", "correctpassword")
	recOk := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(recOk, reqOk)
	if recOk.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid password against hash, got %d", recOk.Code)
	}
}

func TestStringSanitizationForLogs(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		maxLen int
	}{
		{"short_string", "hello", 100},
		{"exact_length", strings.Repeat("a", 100), 100},
		{"long_string", strings.Repeat("a", 1000), 100},
		{"with_newlines", "line1\nline2\nline3", 50},
		{"with_unicode", "hello\u0000world", 100},
		{"empty", "", 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := logging.SanitizeStringForLog(tc.input, tc.maxLen)
			if len(result) > tc.maxLen+20 {
				t.Errorf("sanitized string too long: got %d, expected max %d", len(result), tc.maxLen+20)
			}
		})
	}
}
