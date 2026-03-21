package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOperatorRoleString(t *testing.T) {
	tests := []struct {
		role     OperatorRole
		expected string
	}{
		{RoleViewer, "viewer"},
		{RoleResponder, "responder"},
		{RoleOperator, "operator"},
		{RoleAdmin, "admin"},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if got := tt.role.String(); got != tt.expected {
				t.Errorf("OperatorRole.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestActionClassString(t *testing.T) {
	tests := []struct {
		action   ActionClass
		expected string
	}{
		{ActionRead, "read"},
		{ActionAck, "ack"},
		{ActionSuppress, "suppress"},
		{ActionControl, "control"},
		{ActionConfig, "config"},
		{ActionExport, "export"},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			if got := tt.action.String(); got != tt.expected {
				t.Errorf("ActionClass.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDefaultAdminContext(t *testing.T) {
	ctx := DefaultAdminContext()

	if ctx.OperatorID != "system" {
		t.Errorf("DefaultAdminContext().OperatorID = %v, want system", ctx.OperatorID)
	}

	if ctx.Role != RoleAdmin {
		t.Errorf("DefaultAdminContext().Role = %v, want admin", ctx.Role)
	}

	if ctx.SessionID != "single-operator-session" {
		t.Errorf("DefaultAdminContext().SessionID = %v, want single-operator-session", ctx.SessionID)
	}

	if ctx.AuthMethod != "shared-credentials" {
		t.Errorf("DefaultAdminContext().AuthMethod = %v, want shared-credentials", ctx.AuthMethod)
	}
}

func TestAuthorizationContextIsSingleOperatorMode(t *testing.T) {
	tests := []struct {
		name     string
		ctx      AuthorizationContext
		expected bool
	}{
		{
			name: "system operator",
			ctx: AuthorizationContext{
				OperatorID: "system",
				Role:       RoleAdmin,
			},
			expected: true,
		},
		{
			name: "shared credentials",
			ctx: AuthorizationContext{
				OperatorID: "operator-1",
				Role:       RoleAdmin,
				AuthMethod: "shared-credentials",
			},
			expected: true,
		},
		{
			name: "session-based auth",
			ctx: AuthorizationContext{
				OperatorID: "operator-1",
				Role:       RoleAdmin,
				AuthMethod: "session",
			},
			expected: false,
		},
		{
			name: "JWT auth",
			ctx: AuthorizationContext{
				OperatorID: "operator-1",
				Role:       RoleAdmin,
				AuthMethod: "jwt",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ctx.IsSingleOperatorMode(); got != tt.expected {
				t.Errorf("AuthorizationContext.IsSingleOperatorMode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCanPerform(t *testing.T) {
	tests := []struct {
		name       string
		role       OperatorRole
		action     ActionClass
		wantAllowed bool
	}{
		// Admin can perform all actions
		{"admin_read", RoleAdmin, ActionRead, true},
		{"admin_ack", RoleAdmin, ActionAck, true},
		{"admin_suppress", RoleAdmin, ActionSuppress, true},
		{"admin_control", RoleAdmin, ActionControl, true},
		{"admin_config", RoleAdmin, ActionConfig, true},
		{"admin_export", RoleAdmin, ActionExport, true},

		// Operator can perform most actions except config
		{"operator_read", RoleOperator, ActionRead, true},
		{"operator_ack", RoleOperator, ActionAck, true},
		{"operator_suppress", RoleOperator, ActionSuppress, true},
		{"operator_control", RoleOperator, ActionControl, true},
		{"operator_config", RoleOperator, ActionConfig, false},
		{"operator_export", RoleOperator, ActionExport, true},

		// Responder can read and ack
		{"responder_read", RoleResponder, ActionRead, true},
		{"responder_ack", RoleResponder, ActionAck, true},
		{"responder_control", RoleResponder, ActionControl, false},
		{"responder_config", RoleResponder, ActionConfig, false},

		// Viewer can only read
		{"viewer_read", RoleViewer, ActionRead, true},
		{"viewer_ack", RoleViewer, ActionAck, false},
		{"viewer_control", RoleViewer, ActionControl, false},
		{"viewer_config", RoleViewer, ActionConfig, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanPerform(tt.role, tt.action); got != tt.wantAllowed {
				t.Errorf("CanPerform(%v, %v) = %v, want %v", tt.role, tt.action, got, tt.wantAllowed)
			}
		})
	}
}

func TestCanPerformWithContext(t *testing.T) {
	// In single-operator mode, all actions should be allowed
	ctx := DefaultAdminContext()
	ctx2 := AuthorizationContext{
		OperatorID: "test-operator",
		Role:       RoleViewer,
		AuthMethod: "session",
	}

	// Single operator mode allows everything
	if !CanPerformWithContext(ctx, ActionControl) {
		t.Error("Single-operator mode should allow control action")
	}

	// But the check passes for non-single-operator in advisory mode
	// (currently always returns true)
	if !CanPerformWithContext(ctx2, ActionControl) {
		t.Error("RBAC is currently advisory, should allow")
	}
}

func TestGetAllowedActions(t *testing.T) {
	tests := []struct {
		role          OperatorRole
		minActions    int
		contains      []ActionClass
		notContains   []ActionClass
	}{
		{
			role:       RoleViewer,
			minActions:  1,
			contains:    []ActionClass{ActionRead},
			notContains: []ActionClass{ActionControl, ActionConfig},
		},
		{
			role:       RoleResponder,
			minActions:  2,
			contains:    []ActionClass{ActionRead, ActionAck},
			notContains: []ActionClass{ActionControl, ActionConfig},
		},
		{
			role:       RoleOperator,
			minActions:  4,
			contains:    []ActionClass{ActionRead, ActionAck, ActionSuppress, ActionControl, ActionExport},
			notContains: []ActionClass{ActionConfig},
		},
		{
			role:       RoleAdmin,
			minActions:  6,
			contains:    []ActionClass{ActionRead, ActionAck, ActionSuppress, ActionControl, ActionConfig, ActionExport},
			notContains: []ActionClass{},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			actions := GetAllowedActions(tt.role)
			if len(actions) < tt.minActions {
				t.Errorf("GetAllowedActions(%v) returned %d actions, want at least %d", tt.role, len(actions), tt.minActions)
			}

			actionSet := make(map[ActionClass]bool)
			for _, a := range actions {
				actionSet[a] = true
			}

			for _, a := range tt.contains {
				if !actionSet[a] {
					t.Errorf("GetAllowedActions(%v) should contain %v", tt.role, a)
				}
			}

			for _, a := range tt.notContains {
				if actionSet[a] {
					t.Errorf("GetAllowedActions(%v) should NOT contain %v", tt.role, a)
				}
			}
		})
	}
}

func TestRoleAllowsFunctions(t *testing.T) {
	tests := []struct {
		name     string
		role     OperatorRole
		control  bool
		config   bool
		export   bool
		ack      bool
		suppress bool
	}{
		{RoleViewer.String(), RoleViewer, false, false, false, false, false},
		{RoleResponder.String(), RoleResponder, false, false, false, true, false},
		{RoleOperator.String(), RoleOperator, true, false, true, true, true},
		{RoleAdmin.String(), RoleAdmin, true, true, true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RoleAllowsControl(tt.role); got != tt.control {
				t.Errorf("RoleAllowsControl(%v) = %v, want %v", tt.role, got, tt.control)
			}
			if got := RoleAllowsConfig(tt.role); got != tt.config {
				t.Errorf("RoleAllowsConfig(%v) = %v, want %v", tt.role, got, tt.config)
			}
			if got := RoleAllowsExport(tt.role); got != tt.export {
				t.Errorf("RoleAllowsExport(%v) = %v, want %v", tt.role, got, tt.export)
			}
			if got := RoleCanAcknowledge(tt.role); got != tt.ack {
				t.Errorf("RoleCanAcknowledge(%v) = %v, want %v", tt.role, got, tt.ack)
			}
			if got := RoleCanSuppress(tt.role); got != tt.suppress {
				t.Errorf("RoleCanSuppress(%v) = %v, want %v", tt.role, got, tt.suppress)
			}
		})
	}
}

func TestNewDeniedActionReason(t *testing.T) {
	reason := NewDeniedActionReason(RoleViewer, ActionControl, "viewer cannot control")

	if reason.Reason != "action_not_permitted_by_role" {
		t.Errorf("DeniedActionReason.Reason = %v, want action_not_permitted_by_role", reason.Reason)
	}
	if reason.Role != RoleViewer {
		t.Errorf("DeniedActionReason.Role = %v, want viewer", reason.Role)
	}
	if reason.Action != ActionControl {
		t.Errorf("DeniedActionReason.Action = %v, want control", reason.Action)
	}
	if reason.Details != "viewer cannot control" {
		t.Errorf("DeniedActionReason.Details = %v, want 'viewer cannot control'", reason.Details)
	}
}

func TestAuditEntryCreation(t *testing.T) {
	params := AuditEntryParams{
		ActorID:      "operator-1",
		ActionClass:  ActionControl,
		ActionDetail: "restart_transport",
		ResourceType: "transport",
		ResourceID:   "mqtt-primary",
		Reason:       "transport not responding",
		Result:       AuditResultSuccess,
		Details:      map[string]any{"timeout": 30},
		SessionID:    "session-123",
		RemoteAddr:   "192.168.1.100",
	}

	entry := NewAuditEntry(params)

	if entry.ActorID != params.ActorID {
		t.Errorf("AuditEntry.ActorID = %v, want %v", entry.ActorID, params.ActorID)
	}
	if entry.ActionClass != params.ActionClass {
		t.Errorf("AuditEntry.ActionClass = %v, want %v", entry.ActionClass, params.ActionClass)
	}
	if entry.ResourceID != params.ResourceID {
		t.Errorf("AuditEntry.ResourceID = %v, want %v", entry.ResourceID, params.ResourceID)
	}
	if entry.Result != params.Result {
		t.Errorf("AuditEntry.Result = %v, want %v", entry.Result, params.Result)
	}
	if entry.Timestamp.IsZero() {
		t.Error("AuditEntry.Timestamp should not be zero")
	}
}

func TestLogControlAction(t *testing.T) {
	ctx := DefaultAdminContext()
	details := map[string]any{"transport": "mqtt-primary"}

	entry := LogControlAction(ctx, "restart_transport", "transport", "mqtt-primary", "transport unhealthy", AuditResultSuccess, details)

	if entry.ActionClass != ActionControl {
		t.Errorf("AuditEntry.ActionClass = %v, want control", entry.ActionClass)
	}
	if entry.ActorID != ctx.OperatorID {
		t.Errorf("AuditEntry.ActorID = %v, want %v", entry.ActorID, ctx.OperatorID)
	}
}

func TestLogDeniedAction(t *testing.T) {
	ctx := AuthorizationContext{
		OperatorID: "viewer-1",
		Role:       RoleViewer,
		SessionID:  "session-456",
	}

	entry := LogDeniedAction(ctx, ActionControl, "restart_transport", "transport", "mqtt-primary", "permission denied")

	if entry.Result != AuditResultDenied {
		t.Errorf("AuditEntry.Result = %v, want denied", entry.Result)
	}
	if entry.ActionClass != ActionControl {
		t.Errorf("AuditEntry.ActionClass = %v, want control", entry.ActionClass)
	}
}

func TestWithAuthContext(t *testing.T) {
	// Test that WithAuthContext sets up the context properly
	middleware := WithAuthContext(nil)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := GetAuthContextFromRequest(r)
		if ctx.OperatorID != "system" {
			t.Errorf("Expected system operator, got %v", ctx.OperatorID)
		}
		if ctx.Role != RoleAdmin {
			t.Errorf("Expected admin role, got %v", ctx.Role)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
}

func TestWithAuthContextWithAPIKey(t *testing.T) {
	middleware := WithAuthContext(DefaultAPIKeyValidator{})

	var capturedCtx AuthorizationContext
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = GetAuthContextFromRequest(r)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// DefaultAPIKeyValidator should return system context
	if capturedCtx.OperatorID != "system" {
		t.Errorf("Expected system operator, got %v", capturedCtx.OperatorID)
	}
}

func TestGetAuthContext(t *testing.T) {
	// Test with nil context
	ctx := GetAuthContext(nil)
	if ctx.OperatorID != "system" {
		t.Errorf("Expected system operator for nil context, got %v", ctx.OperatorID)
	}

	// Test with empty context
	ctx = GetAuthContext(context.Background())
	if ctx.OperatorID != "system" {
		t.Errorf("Expected system operator for empty context, got %v", ctx.OperatorID)
	}

	// Test with populated context
	authCtx := AuthorizationContext{
		OperatorID: "test-operator",
		Role:       RoleOperator,
		SessionID:  "session-789",
		Timestamp:  time.Now().UTC(),
		AuthMethod: "session",
	}
	ctx = GetAuthContext(context.WithValue(context.Background(), ContextKey, authCtx))
	if ctx.OperatorID != "test-operator" {
		t.Errorf("Expected test-operator, got %v", ctx.OperatorID)
	}
}

func TestRequirePermission(t *testing.T) {
	middleware := RequirePermission(ActionControl)

	var called bool
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should always call next in single-operator mode
	if !called {
		t.Error("Expected next handler to be called")
	}
}

func TestResponseWithForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	ResponseWithForbidden("action not permitted")

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestResponseWithUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	ResponseWithUnauthorized("authentication required")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	// Check WWW-Authenticate header
	if w.Header().Get("WWW-Authenticate") == "" {
		t.Error("Expected WWW-Authenticate header")
	}
}
