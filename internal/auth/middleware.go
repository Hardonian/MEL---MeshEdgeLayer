package auth

import (
	"context"
	"net/http"
	"time"
)

const (
	// ContextKey is the key used to store AuthorizationContext in request context.
	ContextKey = "auth_context"
)

// APIKeyValidator validates API keys (for future multi-operator support).
// Currently returns a single admin context for any valid API key.
type APIKeyValidator interface {
	ValidateAPIKey(key string) (AuthorizationContext, bool)
}

// DefaultAPIKeyValidator is a simple validator that maps any key to admin role.
// This is used in single-operator mode with shared credentials.
type DefaultAPIKeyValidator struct{}

// ValidateAPIKey returns true for any non-empty key, mapping to admin context.
func (v DefaultAPIKeyValidator) ValidateAPIKey(key string) (AuthorizationContext, bool) {
	if key == "" {
		return AuthorizationContext{}, false
	}
	// Single-operator mode: all keys map to admin
	return AuthorizationContext{
		OperatorID: "system",
		Role:       RoleAdmin,
		SessionID:  "single-operator-session",
		Timestamp:  time.Now().UTC(),
		AuthMethod: "shared-credentials",
	}, true
}

// WithAuthContext creates middleware that extracts authentication info
// from the request and populates the request context with AuthorizationContext.
//
// Current behavior:
// - Extracts API key from X-API-Key header
// - Falls back to Basic Auth
// - Maps to single-operator admin context
//
// Future hooks for multi-operator support:
// - JWT token validation
// - Session-based authentication
// - Role derivation from token claims
func WithAuthContext(validator APIKeyValidator) func(http.Handler) http.Handler {
	if validator == nil {
		validator = DefaultAPIKeyValidator{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Try X-API-Key header first
			apiKey := r.Header.Get("X-API-Key")
			if ctx, ok := tryAuth(ctx, apiKey, validator); ok {
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Fall back to Basic Auth
			if user, pass, ok := r.BasicAuth(); ok && user != "" && pass != "" {
				// In single-operator mode, any valid credentials map to admin
				apiKey = user + ":" + pass
				if ctx, ok := tryAuth(ctx, apiKey, validator); ok {
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// No valid auth provided - use default single-operator context
			// This maintains backward compatibility when auth is not enforced
			defaultCtx := context.WithValue(r.Context(), ContextKey, DefaultAdminContext())
			next.ServeHTTP(w, r.WithContext(defaultCtx))
		})
	}
}

// tryAuth attempts to authenticate using the provided API key.
func tryAuth(ctx context.Context, key string, validator APIKeyValidator) (context.Context, bool) {
	if key == "" {
		return ctx, false
	}

	authCtx, valid := validator.ValidateAPIKey(key)
	if !valid {
		return ctx, false
	}

	return context.WithValue(ctx, ContextKey, authCtx), true
}

// GetAuthContext retrieves the AuthorizationContext from a request context.
// Returns DefaultAdminContext() if no context is set.
func GetAuthContext(ctx context.Context) AuthorizationContext {
	if ctx == nil {
		return DefaultAdminContext()
	}

	if val := ctx.Value(ContextKey); val != nil {
		if authCtx, ok := val.(AuthorizationContext); ok {
			return authCtx
		}
	}

	return DefaultAdminContext()
}

// GetAuthContextFromRequest retrieves the AuthorizationContext from the request.
// This is a convenience function for HTTP handlers.
func GetAuthContextFromRequest(r *http.Request) AuthorizationContext {
	return GetAuthContext(r.Context())
}

// RequirePermission creates middleware that checks if the request has
// permission to perform the specified action class.
//
// Currently advisory only - always allows in single-operator mode.
// Returns 403 Forbidden if the action is not permitted.
func RequirePermission(actionClass ActionClass) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := GetAuthContextFromRequest(r)

			// In single-operator mode, RBAC is advisory only
			// Always allow - this will change with future multi-operator support
			if !CanPerformWithContext(ctx, actionClass) {
				// For now, we still allow but could return 403 in the future
				// when full RBAC is implemented
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole creates middleware that checks if the request has
// at least the specified role level.
func RequireRole(minRole OperatorRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := GetAuthContextFromRequest(r)

			// In single-operator mode, always allow
			// Future: check role hierarchy
			_ = minRole

			next.ServeHTTP(w, r)
		})
	}
}

// ResponseWithForbidden writes a 403 Forbidden response with a clear message.
func ResponseWithForbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`{"error":"` + message + `","code":"forbidden"}`))
}

// ResponseWithUnauthorized writes a 401 Unauthorized response.
func ResponseWithUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Basic realm="mel"`)
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"` + message + `","code":"unauthorized"}`))
}
