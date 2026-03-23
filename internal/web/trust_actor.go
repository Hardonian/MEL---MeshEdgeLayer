package web

// trust_actor.go — Bind durable operator attribution to authenticated identity.

import (
	"net/http"
	"strings"

	"github.com/mel-project/mel/internal/security"
)

// actorFromTrustContext returns the actor id recorded in audit and control_actions.
// When auth is disabled, local shell/API users still get a stable default identity.
// Optional X-Operator-ID is only honored when auth is enabled (API key or Basic),
// so unauthenticated clients cannot spoof a stronger identity than the server assigned.
func (s *Server) actorFromTrustContext(r *http.Request) string {
	id, ok := security.GetIdentity(r.Context())
	if !ok {
		return "system"
	}
	if s.cfg.Auth.Enabled {
		if hdr := strings.TrimSpace(r.Header.Get("X-Operator-ID")); hdr != "" {
			return hdr
		}
	}
	return id.ActorID
}
