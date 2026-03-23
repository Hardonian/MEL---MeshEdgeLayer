package security

import (
	"fmt"
	"strings"
)

// KnownCapabilities returns every capability the server understands, for session introspection and tests.
func KnownCapabilities() []Capability {
	return []Capability{
		CapReadStatus,
		CapReadIncidents,
		CapReadActions,
		CapApproveControlAction,
		CapRejectControlAction,
		CapExecuteAction,
		CapControlActionWrite,
		CapIncidentHandoffWrite,
		CapIncidentUpdate,
		CapAcknowledgeAlerts,
		CapEscalateAlerts,
		CapSuppressAlerts,
		CapExportBundle,
		CapInspectConfig,
		CapAdminSystem,
	}
}

// ParseCapability normalizes a config string to a Capability.
func ParseCapability(s string) (Capability, error) {
	c := Capability(strings.TrimSpace(s))
	switch c {
	case CapReadStatus,
		CapReadIncidents,
		CapReadActions,
		CapApproveControlAction,
		CapRejectControlAction,
		CapExecuteAction,
		CapControlActionWrite,
		CapIncidentHandoffWrite,
		CapIncidentUpdate,
		CapAcknowledgeAlerts,
		CapEscalateAlerts,
		CapSuppressAlerts,
		CapExportBundle,
		CapInspectConfig,
		CapAdminSystem:
		return c, nil
	default:
		return "", fmt.Errorf("unknown capability %q", s)
	}
}

// ParseCapabilities parses a list of capability strings into a set map.
func ParseCapabilities(names []string) (map[Capability]bool, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("capabilities list is empty")
	}
	out := make(map[Capability]bool, len(names))
	for _, raw := range names {
		c, err := ParseCapability(raw)
		if err != nil {
			return nil, err
		}
		out[c] = true
	}
	return out, nil
}

// FullAdminCapabilitySet is the superset granted to UI basic-auth admins and legacy env-only API keys.
func FullAdminCapabilitySet() map[Capability]bool {
	m := make(map[Capability]bool)
	for _, c := range KnownCapabilities() {
		m[c] = true
	}
	return m
}
