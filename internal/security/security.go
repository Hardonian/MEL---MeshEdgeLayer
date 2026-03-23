package security

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

type ActorType string

const (
	ActorHuman  ActorType = "human"
	ActorSystem ActorType = "system"
)

type Capability string

const (
	CapReadStatus           Capability = "read_status"
	CapAcknowledgeAlerts    Capability = "acknowledge_alerts"
	CapEscalateAlerts       Capability = "escalate_alerts"
	CapSuppressAlerts       Capability = "suppress_alerts"
	CapExportBundle         Capability = "export_support_bundle"
	CapExecuteAction        Capability = "execute_control_action"
	CapApproveControlAction Capability = "approve_control_action"
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
		CapExecuteAction:        true,
		CapApproveControlAction: true,
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
		ActorID:     "local_admin_" + username,
		ActorType:   ActorHuman,
		DisplayName: "Local Admin (" + username + ")",
		Role:        "admin",
		Capabilities: map[Capability]bool{
			CapReadStatus:           true,
			CapAcknowledgeAlerts:    true,
			CapEscalateAlerts:       true,
			CapSuppressAlerts:       true,
			CapExportBundle:         true,
			CapExecuteAction:        true,
			CapApproveControlAction: true,
			CapInspectConfig:        true,
			CapAdminSystem:          true,
		},
	}
}

func BuildViewerIdentity(username string) Identity {
	// A simple scaffolding viewer identity
	return Identity{
		ActorID:     "viewer_" + username,
		ActorType:   ActorHuman,
		DisplayName: "Viewer (" + username + ")",
		Role:        "viewer",
		Capabilities: map[Capability]bool{
			CapReadStatus: true,
		},
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
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := GetIdentity(r.Context())
		if !ok || !id.Can(cap) {

			// For JSON vs HTML response we'd do better, but simple 403 for now.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			// Return a generic JSON error since this might be an API endpoint
			w.Write([]byte(fmt.Sprintf(`{"error": "denied", "reason": "missing required capability: %s"}`, cap)))
			return
		}
		next(w, r)
	}
}
