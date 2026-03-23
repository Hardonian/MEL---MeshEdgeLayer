package security

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type ActorType string

const (
	ActorHuman  ActorType = "human"
	ActorSystem ActorType = "system"
)

type Capability string

const (
	CapReadStatus           Capability = "read_status"
	CapReadIncidents        Capability = "read_incidents"
	CapReadActions          Capability = "read_actions"
	CapAcknowledgeAlerts    Capability = "acknowledge_alerts"
	CapEscalateAlerts       Capability = "escalate_alerts"
	CapSuppressAlerts       Capability = "suppress_alerts"
	CapExportBundle         Capability = "export_support_bundle"
	CapExecuteAction        Capability = "execute_control_action"
	CapApproveControlAction Capability = "approve_control_action"
	CapRejectControlAction  Capability = "reject_control_action"
	CapControlActionWrite   Capability = "control_action_write"
	CapIncidentHandoffWrite Capability = "incident_handoff_write"
	CapIncidentUpdate       Capability = "incident_update"
	CapInspectConfig        Capability = "inspect_config"
	CapAdminSystem          Capability = "admin_system"
)

type Identity struct {
	ActorID      string
	ActorType    ActorType
	DisplayName  string
	Role         string
	Capabilities map[Capability]bool
}

func (i Identity) Can(c Capability) bool {
	if i.Capabilities == nil {
		return false
	}
	// Admin has all capabilities implicitly in this simple model if needed, but let's be explicit
	return i.Capabilities[c]
}

// SystemIdentity represents internal MEL automation
var SystemIdentity = Identity{
	ActorID:     "mel_system",
	ActorType:   ActorSystem,
	DisplayName: "System",
	Role:        "system",
	Capabilities: map[Capability]bool{
		CapReadStatus:           true,
		CapReadIncidents:        true,
		CapReadActions:          true,
		CapExecuteAction:        true,
		CapApproveControlAction: true,
		CapRejectControlAction:  true,
		CapControlActionWrite:   true,
		CapIncidentHandoffWrite: true,
		CapIncidentUpdate:       true,
		CapAcknowledgeAlerts:    true,
		CapEscalateAlerts:       true,
		CapSuppressAlerts:       true,
	},
}

// Ensure the mode checking still remains here
func CheckFileMode(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("%s permissions are broader than 0600", path)
	}
	return nil
}

type contextKey string

const identityKey contextKey = "mel_identity"

func BuildAdminIdentity(username string) Identity {
	return Identity{
		ActorID:      "local_admin_" + username,
		ActorType:    ActorHuman,
		DisplayName:  "Local Admin (" + username + ")",
		Role:         "admin",
		Capabilities: FullAdminCapabilitySet(),
	}
}

func BuildViewerIdentity(username string) Identity {
	return Identity{
		ActorID:     "viewer_" + username,
		ActorType:   ActorHuman,
		DisplayName: "Viewer (" + username + ")",
		Role:        "viewer",
		Capabilities: map[Capability]bool{
			CapReadStatus:    true,
			CapReadIncidents: true,
			CapReadActions:   true,
		},
	}
}

// IdentityFromAPIKey builds the HTTP identity for a validated X-API-Key (shortID is a stable hash prefix).
func IdentityFromAPIKey(shortID string, caps map[Capability]bool) Identity {
	if caps == nil {
		caps = map[Capability]bool{}
	}
	return Identity{
		ActorID:      "apikey:" + shortID,
		ActorType:    ActorHuman,
		DisplayName:  "API key operator",
		Role:         "api_key",
		Capabilities: caps,
	}
}

func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey, id)
}

func GetIdentity(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey).(Identity)
	return id, ok
}

// Require checks if a request has the required capability
func Require(cap Capability, next http.HandlerFunc) http.HandlerFunc {
	return RequireAny([]Capability{cap}, next)
}

// RequireAny allows the request if the identity has any of the listed capabilities.
func RequireAny(caps []Capability, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := GetIdentity(r.Context())
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error": "denied", "reason": "missing identity"}`))
			return
		}
		for _, cap := range caps {
			if id.Can(cap) {
				next(w, r)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		reason := "missing required capability"
		if len(caps) == 1 {
			reason = fmt.Sprintf("missing required capability: %s", caps[0])
		} else if len(caps) > 1 {
			var parts []string
			for _, c := range caps {
				parts = append(parts, string(c))
			}
			reason = fmt.Sprintf("missing one of required capabilities: %s", strings.Join(parts, ", "))
		}
		w.Write([]byte(fmt.Sprintf(`{"error": "denied", "reason": %q}`, reason)))
	}
}
