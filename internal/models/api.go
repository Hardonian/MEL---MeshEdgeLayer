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

	// Collaboration / handoff (durable; migration 0020)
	OwnerActorID   string           `json:"owner_actor_id,omitempty"`
	HandoffSummary string           `json:"handoff_summary,omitempty"`
	PendingActions []string         `json:"pending_actions,omitempty"`
	RecentActions  []string         `json:"recent_actions,omitempty"`
	LinkedEvidence []map[string]any `json:"linked_evidence,omitempty"`
	Risks          []string         `json:"risks,omitempty"`

	// LinkedControlActions: rows where control_actions.incident_id = this incident (canonical).
	LinkedControlActions []ActionRecord `json:"linked_control_actions,omitempty"`
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
	TargetNode      string         `json:"target_node,omitempty"`
	TargetSegment   string         `json:"target_segment,omitempty"`
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

	// Trust / provenance (control plane)
	ExecutionMode     string `json:"execution_mode,omitempty"`
	ProposedBy        string `json:"proposed_by,omitempty"`
	ApprovedBy        string `json:"approved_by,omitempty"`
	ApprovedAt        string `json:"approved_at,omitempty"`
	RejectedBy        string `json:"rejected_by,omitempty"`
	RejectedAt        string `json:"rejected_at,omitempty"`
	ApprovalNote      string `json:"approval_note,omitempty"`
	ApprovalExpiresAt string `json:"approval_expires_at,omitempty"`
	BlastRadiusClass  string `json:"blast_radius_class,omitempty"`
	EvidenceBundleID  string `json:"evidence_bundle_id,omitempty"`

	SubmittedBy              string `json:"submitted_by,omitempty"`
	RequiresSeparateApprover bool   `json:"requires_separate_approver,omitempty"`
	IncidentID               string `json:"incident_id,omitempty"`
	ExecutionStartedAt       string `json:"execution_started_at,omitempty"`
	SodBypass                bool   `json:"sod_bypass,omitempty"`
	SodBypassActor           string `json:"sod_bypass_actor,omitempty"`
	SodBypassReason          string `json:"sod_bypass_reason,omitempty"`

	// OperatorView is derived from canonical fields for UI/CLI legibility (not a second source of truth).
	OperatorView map[string]any `json:"operator_view,omitempty"`
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

// StatusResponse is a generic API status message
type StatusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// OperatorBriefingDTO represents a structured briefing for the UI/API
type OperatorBriefingDTO struct {
	OverallStatus       string         `json:"overall_status"`
	TopPriorities       []PriorityItem `json:"top_priorities"`
	LikelyCauses        []string       `json:"likely_causes"`
	RecommendedSequence []RecoveryStep `json:"recommended_sequence"`
	BlastRadiusEstimate string         `json:"blast_radius_estimate"`
	UncertaintyNotes    []string       `json:"uncertainty_notes"`
	GeneratedAt         string         `json:"generated_at"`
}

// PriorityItem is a ranked operational problem for the API
type PriorityItem struct {
	ID                string         `json:"id"`
	Category          string         `json:"category"`
	Severity          string         `json:"severity"`
	Title             string         `json:"title"`
	Summary           string         `json:"summary"`
	Rank              float64        `json:"rank"`
	Confidence        float64        `json:"confidence"`
	EvidenceFreshness string         `json:"evidence_freshness"`
	IsActionable      bool           `json:"is_actionable"`
	BlocksRecovery    bool           `json:"blocks_recovery"`
	Metadata          map[string]any `json:"metadata,omitempty"`
}

// ApprovalPolicyDTO mirrors structured approval policy for API consumers.
type ApprovalPolicyDTO struct {
	RequiresApproval                  bool     `json:"requires_approval"`
	ApprovalMode                      string   `json:"approval_mode"`
	RequiredApprovals                 int      `json:"required_approvals"`
	CollectedApprovals                int      `json:"collected_approvals"`
	ApprovalBasis                     []string `json:"approval_basis,omitempty"`
	ApprovalPolicySource              string   `json:"approval_policy_source"`
	HighBlastRadius                   bool     `json:"high_blast_radius"`
	BlastRadiusClassification         string   `json:"blast_radius_classification"`
	ApprovalEscalatedDueToBlastRadius bool     `json:"approval_escalated_due_to_blast_radius"`
	SubmitterDisqualifiedFromApproval bool   `json:"submitter_disqualified_from_approval"`
	ApproverAllowed                   bool     `json:"approver_allowed"`
	ApproverDenialReason              string   `json:"approver_denial_reason,omitempty"`
	ApprovedDoesNotImplyExecution     bool     `json:"approved_does_not_imply_execution"`
	BacklogExecutionRequiresExecutor  bool     `json:"backlog_execution_requires_executor"`
}

// ApproveActionRequest is the body for POST .../control/actions/{id}/approve.
type ApproveActionRequest struct {
	Note                string `json:"note,omitempty"`
	BreakGlassSodAck    bool   `json:"break_glass_sod_ack,omitempty"`
	BreakGlassSodReason string `json:"break_glass_sod_reason,omitempty"`
}

// ApproveActionResponse is returned after HTTP approve; approval does not imply execution
// and does not drain unrelated queued work.
type ApproveActionResponse struct {
	Status    string `json:"status"`
	ActionID  string `json:"action_id"`
	ActorID   string `json:"actor"`

	LifecycleState string `json:"lifecycle_state"`
	Result         string `json:"result"`

	FullyApprovedSingleStep        bool `json:"fully_approved_single_step"`
	ApprovalDoesNotImplyExecution bool `json:"approval_does_not_imply_execution"`

	QueuedForExecution bool `json:"queued_for_execution"`
	ExecutionOccurred  bool `json:"execution_occurred"`

	HTTPApproveDoesNotDrainQueue           bool `json:"http_approve_does_not_drain_queue"`
	BacklogMayRemain                       bool `json:"backlog_may_remain"`
	BacklogExecutionRequiresActiveExecutor bool `json:"backlog_execution_requires_active_executor"`

	Policy ApprovalPolicyDTO `json:"policy"`
}

// RejectActionRequest is the body for POST .../control/actions/{id}/reject.
type RejectActionRequest struct {
	Note                string `json:"note,omitempty"`
	BreakGlassSodAck    bool   `json:"break_glass_sod_ack,omitempty"`
	BreakGlassSodReason string `json:"break_glass_sod_reason,omitempty"`
}

// RejectActionResponse is returned after HTTP reject.
type RejectActionResponse struct {
	Status         string              `json:"status"`
	ActionID       string              `json:"action_id"`
	ActorID        string              `json:"actor"`
	LifecycleState string              `json:"lifecycle_state"`
	Result         string              `json:"result"`
	Policy         ApprovalPolicyDTO   `json:"policy"`
}

// IncidentHandoffRequest is the body for POST /api/v1/incidents/{id}/handoff.
type IncidentHandoffRequest struct {
	ToOperatorID   string           `json:"to_operator_id"`
	HandoffSummary string           `json:"handoff_summary"`
	PendingActions []string         `json:"pending_actions,omitempty"`
	RecentActions  []string         `json:"recent_actions,omitempty"`
	LinkedEvidence []map[string]any `json:"linked_evidence,omitempty"`
	Risks          []string         `json:"risks,omitempty"`
}

// RecoveryStep represents a single step in a recovery sequence for the API
type RecoveryStep struct {
	Stage         int      `json:"stage"`
	Action        string   `json:"action"`
	Justification string   `json:"justification"`
	Status        string   `json:"status"`
	UnsafeEarly   bool     `json:"unsafe_early"`
	Dependencies  []string `json:"dependencies,omitempty"`
}
