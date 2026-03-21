package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestUIStateResiliency verifies that key frontend-consumed endpoints
// behave truthfully when underlying data is empty or partial, ensuring
// the frontend never receives unexpected nulls or malformed JSON that
// would cause a blank state or false UI rendering.
func TestUIStateResiliency(t *testing.T) {
	// Setup minimalist server with no active mesh
	srv := newTestServer(t, nil, nil)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		checkBody  func(t *testing.T, body []byte)
	}{
		{
			name:       "Diagnostics empty state",
			path:       "/api/v1/diagnostics",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var res map[string]interface{}
				if err := json.Unmarshal(body, &res); err != nil {
					t.Fatalf("Failed to parse JSON: %v", err)
				}
				// Even if empty, it should be an explicit clear structure
				if findings, ok := res["findings"]; !ok || findings == nil {
					t.Errorf("Expected 'findings' to be present and non-null, got: %v", res)
				}
			},
		},
		{
			name:       "Nodes empty state",
			path:       "/api/v1/nodes",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var res map[string]interface{}
				if err := json.Unmarshal(body, &res); err != nil {
					t.Fatalf("Failed to parse JSON: %v", err)
				}
				if nodes, ok := res["nodes"]; !ok || nodes == nil {
					t.Errorf("Expected 'nodes' to be present and non-null, got: %v", res)
				}
			},
		},
		{
			name:       "Control history empty state",
			path:       "/api/v1/control/history",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var res map[string]interface{}
				if err := json.Unmarshal(body, &res); err != nil {
					t.Fatalf("Failed to parse JSON: %v", err)
				}
				if actions, ok := res["actions"]; !ok || actions == nil {
					t.Errorf("Expected 'actions' to be present and non-null, got: %v", res)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("Expected status %d, got %d", tc.wantStatus, rr.Code)
			}
			tc.checkBody(t, rr.Body.Bytes())
		})
	}
}
