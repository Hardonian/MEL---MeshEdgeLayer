package models

import "time"

// Node represents a Mesh Node for the API
type Node struct {
	NodeNum       int64   `json:"node_num"`
	NodeID        string  `json:"node_id"`
	LongName      string  `json:"long_name"`
	ShortName     string  `json:"short_name"`
	LastSeen      string  `json:"last_seen"` // RFC3339
	LastGatewayID string  `json:"last_gateway_id"`
	LatRedacted   float64 `json:"lat_redacted"`
	LonRedacted   float64 `json:"lon_redacted"`
	Altitude      int64   `json:"altitude"`
	LastSNR       float64 `json:"last_snr"`
	LastRSSI      int64   `json:"last_rssi"`
	MessageCount  int64   `json:"message_count"`
}

// TransportSummary represents a transport's health and alert status for the list view
type TransportSummary struct {
	Name              string   `json:"name"`
	Type              string   `json:"type"`
	RuntimeState      string   `json:"runtime_state"`
	EffectiveState    string   `json:"effective_state"`
	Health            int      `json:"health"`
	ActiveAlerts      []string `json:"active_alerts"`
	RecentAnomalies   int      `json:"recent_anomalies"`
	LastFailureAt     string   `json:"last_failure_at"`
}

// Incident represents a system incident or alert
type Incident struct {
	ID           string         `json:"id"`
	Category     string         `json:"category"`
	Severity     string         `json:"severity"`
	Title        string         `json:"title"`
	Summary      string         `json:"summary"`
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id"`
	State        string         `json:"state"`
	ActorID      string         `json:"actor_id,omitempty"`
	OccurredAt   string         `json:"occurred_at"`
	UpdatedAt    string         `json:"updated_at,omitempty"`
	ResolvedAt   string         `json:"resolved_at,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// SupportManifest defines the inventory of a support bundle
type SupportManifest struct {
	GeneratedAt time.Time      `json:"generated_at"`
	Version     string         `json:"version"`
	Fingerprint string         `json:"fingerprint"`
	Contents    []string       `json:"contents"`
	Redactions  []string       `json:"redactions"`
}

// ActionRecord represents a control action in history
type ActionRecord struct {
	ID             string         `json:"id"`
	TransportName  string         `json:"transport_name"`
	ActionType     string         `json:"action_type"`
	LifecycleState string         `json:"lifecycle_state"`
	Result         string         `json:"result"`
	CreatedAt      string         `json:"created_at"`
	ExecutedAt     string         `json:"executed_at,omitempty"`
	CompletedAt    string         `json:"completed_at,omitempty"`
	Details        map[string]any `json:"details,omitempty"`
}
