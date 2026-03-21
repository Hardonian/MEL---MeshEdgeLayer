package models

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
	Name               string   `json:"name"`
	Type               string   `json:"type"`
	RuntimeState       string   `json:"runtime_state"`
	EffectiveState     string   `json:"effective_state"`
	Health             int      `json:"health"`
	ActiveAlertCount   int      `json:"active_alert_count"`
	RecentAnomalyCount int      `json:"recent_anomaly_count"`
	LastFailureAt      string   `json:"last_failure_at"`
	ActiveAlertReasons []string `json:"active_alert_reasons,omitempty"`
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
	ID        string         `json:"id"`
	Version   string         `json:"version"`
	Platform  string         `json:"platform"`
	CreatedAt string         `json:"created_at"`
	Features  []string       `json:"features"`
	Checklist map[string]any `json:"checklist"`
}

// ActionRecord represents a control action in history
type ActionRecord struct {
	ID              string         `json:"id"`
	TransportName   string         `json:"transport_name"`
	ActionType      string         `json:"action_type"`
	LifecycleState  string         `json:"lifecycle_state"`
	Result          string         `json:"result"`
	Reason          string         `json:"reason"`
	OutcomeDetail   string         `json:"outcome_detail"`
	CreatedAt       string         `json:"created_at"`
	ExecutedAt      string         `json:"executed_at,omitempty"`
	CompletedAt     string         `json:"completed_at,omitempty"`
	ExpiresAt       string         `json:"expires_at,omitempty"`
	TriggerEvidence []string       `json:"trigger_evidence,omitempty"`
	Details         map[string]any `json:"details,omitempty"`
}

// DecisionRecord represents a control decision in history
type DecisionRecord struct {
	ID                string         `json:"id"`
	CandidateActionID string         `json:"candidate_action_id"`
	ActionType        string         `json:"action_type"`
	TargetTransport   string         `json:"target_transport"`
	Reason            string         `json:"reason"`
	Confidence        float64        `json:"confidence"`
	Allowed           bool           `json:"allowed"`
	DenialReason      string         `json:"denial_reason,omitempty"`
	CreatedAt         string         `json:"created_at"`
	Mode              string         `json:"mode"`
	PolicySummary     map[string]any `json:"policy_summary,omitempty"`
}

// FreshnessReport represents the freshness of a system component
type FreshnessReport struct {
	Component       string `json:"component"`
	LastUpdate      string `json:"last_update"`
	IntervalSeconds int    `json:"expected_interval_seconds"`
	StaleThreshold  int    `json:"stale_threshold_seconds"`
	Status          string `json:"status"` // fresh, stale, unknown
	AgeSeconds      int    `json:"age_seconds"`
}
