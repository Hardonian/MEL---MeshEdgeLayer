package diagnostics

import (
	"time"
)

// DiagnosticReport represents the full output of a diagnostic run
type DiagnosticReport struct {
	GeneratedAt time.Time      `json:"generated_at"`
	Summary     Summary        `json:"summary"`
	Diagnostics []Diagnostic   `json:"diagnostics"`
	RawEvidence map[string]any `json:"raw_evidence"`
}

// Summary provides a high-level overview of diagnostic findings
type Summary struct {
	TotalCount     int `json:"total_count"`
	CriticalCount  int `json:"critical_count"`
	WarningCount   int `json:"warning_count"`
	InfoCount      int `json:"info_count"`
	CanAutoRecover int `json:"can_auto_recover"`
	NeedsOperator  int `json:"needs_operator"`
}

// Diagnostic represents a single diagnostic finding
type Diagnostic struct {
	Code                   string         `json:"code"`
	Severity               string         `json:"severity"`
	Component              string         `json:"component"`
	Title                  string         `json:"title"`
	Explanation            string         `json:"explanation"`
	LikelyCauses           []string       `json:"likely_causes"`
	RecommendedSteps       []string       `json:"recommended_steps"`
	Evidence               map[string]any `json:"evidence"`
	CanAutoRecover         bool           `json:"can_auto_recover"`
	OperatorActionRequired bool           `json:"operator_action_required"`
	AffectedTransport      string         `json:"affected_transport,omitempty"`
	GeneratedAt            string         `json:"generated_at"`
}

// Finding is an alias for Diagnostic for API compatibility
type Finding = Diagnostic

// Constants for diagnostic codes, severities, and components
const (
	SeverityCritical = "critical"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"

	ComponentTransport = "transport"
	ComponentMesh      = "mesh"
	ComponentDatabase  = "database"
	ComponentConfig    = "config"
	ComponentControl   = "control"
	ComponentStorage   = "storage"
	ComponentRetention = "retention"

	CodeTransportUnreachable            = "transport_unreachable"
	CodeTransportDisconnected           = "transport_disconnected"
	CodeTransportReconnecting           = "transport_reconnecting"
	CodeTransportStaleHeartbeat         = "transport_stale_heartbeat"
	CodeTransportHighTimeouts           = "transport_high_timeouts"
	CodeTransportDeadLetterAccumulation = "transport_dead_letter_accumulation"
	CodeTransportObservationDrops       = "transport_observation_drops"
	CodeDatabaseUnreachable             = "database_unreachable"
	CodeDatabaseWriteFailures           = "database_write_failures"
	CodeMeshNoNodesDetected             = "mesh_no_nodes_detected"
	CodeMeshStaleSnapshot               = "mesh_stale_snapshot"
	CodeMeshPartialData                 = "mesh_partial_data"
	CodeMeshNodeSilence                 = "mesh_node_silence"
	CodeConfigUnsafeSettings            = "config_unsafe_settings"
	CodeConfigMissingRequired           = "config_missing_required"
	CodeControlNoSafeguards             = "control_no_safeguards"
	CodeControlModeUnsafe               = "control_mode_unsafe"
	CodeRetentionMisconfigured          = "retention_misconfigured"
	CodeStoragePathMissing              = "storage_path_missing"
)
