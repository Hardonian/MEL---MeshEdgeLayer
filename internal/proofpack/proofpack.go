// Package proofpack assembles incident-scoped evidence bundles for audit,
// export, and operator review. A proofpack is a self-contained, typed
// snapshot of everything MEL knows about an incident at assembly time,
// including the incident itself, linked control actions, timeline events,
// transport health context, dead-letter evidence, operator notes, and
// explicit markers for missing or degraded evidence.
//
// A proofpack is NOT proof of current state. It is a historical evidence
// snapshot assembled at a specific point in time. All timestamps, provenance
// markers, and evidence-gap annotations are preserved so downstream
// consumers (auditors, peer operators, compliance tools) can assess the
// completeness and recency of the evidence without false certainty.
package proofpack

import "time"

// FormatVersion is the canonical proofpack schema version.
// Consumers must check this before parsing.
const FormatVersion = "1.0.0"

// Proofpack is the top-level evidence bundle for a single incident.
type Proofpack struct {
	// Schema
	FormatVersion string `json:"format_version"`

	// Assembly metadata — when, by whom, and under what conditions this
	// proofpack was created.
	Assembly AssemblyMetadata `json:"assembly"`

	// The incident that scopes this proofpack.
	Incident IncidentEvidence `json:"incident"`

	// Control actions linked to this incident (canonical FK).
	LinkedActions []ActionEvidence `json:"linked_actions"`
	// Per-action historical evaluation snapshots associated with this incident signature.
	ActionOutcomeSnapshots []ActionOutcomeSnapshot `json:"action_outcome_snapshots,omitempty"`

	// Chronological timeline events related to this incident.
	// Includes control actions, freezes, notes, handoffs, and other
	// timeline-projected events whose resource_id matches the incident.
	Timeline []TimelineEntry `json:"timeline"`

	// Transport health snapshots closest to the incident time window.
	// These provide context about the transport environment, not proof
	// of current state.
	TransportContext []TransportSnapshot `json:"transport_context"`

	// Dead letters observed during the incident time window, scoped
	// to transports relevant to the incident where possible.
	DeadLetterEvidence []DeadLetterEntry `json:"dead_letter_evidence"`

	// Operator notes attached to the incident.
	OperatorNotes []OperatorNote `json:"operator_notes"`

	// Audit log entries for actions on this incident.
	AuditEntries []AuditEntry `json:"audit_entries"`

	// Explicit evidence gap markers. Every known gap or limitation in
	// the assembled evidence is listed here so consumers do not infer
	// completeness from the absence of gap markers.
	EvidenceGaps []EvidenceGap `json:"evidence_gaps"`
}

// AssemblyMetadata records who assembled the proofpack, when, and what
// evidence was available. This is not an attestation of correctness; it
// is a record of assembly conditions.
type AssemblyMetadata struct {
	AssembledAt    string `json:"assembled_at"`     // RFC3339
	AssembledBy    string `json:"assembled_by"`     // actor ID or "system"
	InstanceID     string `json:"instance_id"`      // MEL instance that assembled
	IncidentID     string `json:"incident_id"`      // scoping incident
	TimeWindowFrom string `json:"time_window_from"` // earliest evidence considered (RFC3339)
	TimeWindowTo   string `json:"time_window_to"`   // latest evidence considered (RFC3339)

	// Counts of evidence items assembled (for quick integrity checks).
	ActionCount                 int    `json:"action_count"`
	ActionOutcomeSnapshotCount  int    `json:"action_outcome_snapshot_count"`
	ActionOutcomeSnapshotStatus string `json:"action_outcome_snapshot_status"` // complete, partial, unavailable
	TimelineCount               int    `json:"timeline_count"`
	TransportCount              int    `json:"transport_count"`
	DeadLetterCount             int    `json:"dead_letter_count"`
	NoteCount                   int    `json:"note_count"`
	AuditEntryCount             int    `json:"audit_entry_count"`
	EvidenceGapCount            int    `json:"evidence_gap_count"`

	// AssemblyDurationMs is the wall-clock time spent assembling.
	AssemblyDurationMs int64 `json:"assembly_duration_ms"`
}

// IncidentEvidence is the incident record at assembly time, preserved
// exactly as it existed in the database.
type IncidentEvidence struct {
	ID              string                    `json:"id"`
	Category        string                    `json:"category"`
	Severity        string                    `json:"severity"`
	Title           string                    `json:"title"`
	Summary         string                    `json:"summary"`
	ResourceType    string                    `json:"resource_type"`
	ResourceID      string                    `json:"resource_id"`
	State           string                    `json:"state"`
	ActorID         string                    `json:"actor_id,omitempty"`
	OccurredAt      string                    `json:"occurred_at"`
	UpdatedAt       string                    `json:"updated_at,omitempty"`
	ResolvedAt      string                    `json:"resolved_at,omitempty"`
	OwnerActorID    string                    `json:"owner_actor_id,omitempty"`
	HandoffSummary  string                    `json:"handoff_summary,omitempty"`
	PendingActions  []string                  `json:"pending_actions,omitempty"`
	RecentActions   []string                  `json:"recent_actions,omitempty"`
	LinkedEvidence  []map[string]interface{}  `json:"linked_evidence,omitempty"`
	Risks           []string                  `json:"risks,omitempty"`
	Metadata        map[string]interface{}    `json:"metadata,omitempty"`
	WirelessContext *ProofpackWirelessContext `json:"wireless_context,omitempty"`
}

type ProofpackWirelessContext struct {
	Classification    string   `json:"classification"`
	PrimaryDomain     string   `json:"primary_domain"`
	ObservedDomains   []string `json:"observed_domains,omitempty"`
	EvidencePosture   string   `json:"evidence_posture"`
	ConfidencePosture string   `json:"confidence_posture"`
	Summary           string   `json:"summary"`
	EvidenceGaps      []string `json:"evidence_gaps,omitempty"`
	InspectNext       []string `json:"inspect_next,omitempty"`
}

// ActionEvidence is a control action record preserved for the proofpack.
type ActionEvidence struct {
	ID               string   `json:"id"`
	ActionType       string   `json:"action_type"`
	TransportName    string   `json:"transport_name,omitempty"`
	TargetNode       string   `json:"target_node,omitempty"`
	TargetSegment    string   `json:"target_segment,omitempty"`
	LifecycleState   string   `json:"lifecycle_state"`
	Result           string   `json:"result"`
	Reason           string   `json:"reason"`
	OutcomeDetail    string   `json:"outcome_detail,omitempty"`
	ExecutionMode    string   `json:"execution_mode,omitempty"`
	BlastRadiusClass string   `json:"blast_radius_class,omitempty"`
	HighBlastRadius  bool     `json:"high_blast_radius,omitempty"`
	ProposedBy       string   `json:"proposed_by,omitempty"`
	SubmittedBy      string   `json:"submitted_by,omitempty"`
	ApprovedBy       string   `json:"approved_by,omitempty"`
	ApprovedAt       string   `json:"approved_at,omitempty"`
	RejectedBy       string   `json:"rejected_by,omitempty"`
	RejectedAt       string   `json:"rejected_at,omitempty"`
	CreatedAt        string   `json:"created_at"`
	ExecutedAt       string   `json:"executed_at,omitempty"`
	CompletedAt      string   `json:"completed_at,omitempty"`
	IncidentID       string   `json:"incident_id,omitempty"`
	SodBypass        bool     `json:"sod_bypass,omitempty"`
	SodBypassReason  string   `json:"sod_bypass_reason,omitempty"`
	ApprovalBasis    []string `json:"approval_basis,omitempty"`
	ExecutionSource  string   `json:"execution_source,omitempty"`
}

type ActionOutcomeEvidenceSummary struct {
	TransportName        string `json:"transport_name,omitempty"`
	DeadLettersCount     int    `json:"dead_letters_count"`
	TransportAlertsCount int    `json:"transport_alerts_count"`
	IncidentState        string `json:"incident_state,omitempty"`
	ActionResult         string `json:"action_result,omitempty"`
	ActionLifecycle      string `json:"action_lifecycle,omitempty"`
}

type ActionOutcomeSnapshot struct {
	SnapshotID            string                       `json:"snapshot_id"`
	SignatureKey          string                       `json:"signature_key"`
	IncidentID            string                       `json:"incident_id"`
	ActionID              string                       `json:"action_id"`
	ActionType            string                       `json:"action_type"`
	ActionLabel           string                       `json:"action_label,omitempty"`
	DerivedClassification string                       `json:"derived_classification"`
	EvidenceSufficiency   string                       `json:"evidence_sufficiency"`
	WindowStart           string                       `json:"window_start"`
	WindowEnd             string                       `json:"window_end"`
	PreActionEvidence     ActionOutcomeEvidenceSummary `json:"pre_action_evidence"`
	PostActionEvidence    ActionOutcomeEvidenceSummary `json:"post_action_evidence"`
	ObservedSignalCount   int                          `json:"observed_signal_count"`
	Caveats               []string                     `json:"caveats,omitempty"`
	InspectBeforeReuse    []string                     `json:"inspect_before_reuse,omitempty"`
	EvidenceRefs          []string                     `json:"evidence_refs,omitempty"`
	AssociationOnly       bool                         `json:"association_only"`
	DerivationVersion     string                       `json:"derivation_version,omitempty"`
	SchemaVersion         string                       `json:"schema_version,omitempty"`
	DerivedAt             string                       `json:"derived_at"`
}

// TimelineEntry is a single event in the incident-scoped timeline.
type TimelineEntry struct {
	EventTime  string                 `json:"event_time"`
	EventType  string                 `json:"event_type"`
	EventID    string                 `json:"event_id"`
	Summary    string                 `json:"summary"`
	Severity   string                 `json:"severity,omitempty"`
	ActorID    string                 `json:"actor_id,omitempty"`
	ResourceID string                 `json:"resource_id,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	// Provenance
	ScopePosture     string `json:"scope_posture,omitempty"`
	TimingPosture    string `json:"timing_posture,omitempty"`
	MergeDisposition string `json:"merge_disposition,omitempty"`
}

// TransportSnapshot records transport health at a point in time.
type TransportSnapshot struct {
	TransportName              string `json:"transport_name"`
	TransportType              string `json:"transport_type"`
	Score                      int    `json:"score"`
	State                      string `json:"state"`
	SnapshotTime               string `json:"snapshot_time"`
	ActiveAlertCount           int    `json:"active_alert_count"`
	DeadLetterCountWindow      int    `json:"dead_letter_count_window"`
	ObservationDropCountWindow int    `json:"observation_drop_count_window"`
}

// DeadLetterEntry is a dead letter observed during the incident window.
type DeadLetterEntry struct {
	TransportName string                 `json:"transport_name"`
	TransportType string                 `json:"transport_type"`
	Topic         string                 `json:"topic"`
	Reason        string                 `json:"reason"`
	CreatedAt     string                 `json:"created_at"`
	Details       map[string]interface{} `json:"details,omitempty"`
}

// OperatorNote is a note attached to the incident by an operator.
type OperatorNote struct {
	ID        string `json:"id"`
	ActorID   string `json:"actor_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// AuditEntry is an RBAC audit log entry related to the incident.
type AuditEntry struct {
	ID           string `json:"id"`
	Timestamp    string `json:"timestamp"`
	ActorID      string `json:"actor_id"`
	ActionClass  string `json:"action_class"`
	ActionDetail string `json:"action_detail"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Result       string `json:"result"`
}

// EvidenceGap marks a known gap or limitation in the assembled evidence.
// Every proofpack should include at least one gap marker (even if it is
// "no_known_gaps") so that consumers can distinguish "no gaps" from
// "gaps not evaluated".
type EvidenceGap struct {
	Category    string `json:"category"`    // e.g. "timeline", "transport_health", "dead_letters", "actions", "audit"
	Severity    string `json:"severity"`    // info, warning, critical
	Description string `json:"description"` // human-readable explanation
}

// GapCategoryTimeline indicates timeline evidence gaps.
const GapCategoryTimeline = "timeline"

// GapCategoryTransportHealth indicates transport health evidence gaps.
const GapCategoryTransportHealth = "transport_health"

// GapCategoryDeadLetters indicates dead letter evidence gaps.
const GapCategoryDeadLetters = "dead_letters"

// GapCategoryActions indicates action evidence gaps.
const GapCategoryActions = "actions"

// GapCategoryAudit indicates audit log evidence gaps.
const GapCategoryAudit = "audit"

// GapCategoryIncident indicates incident record gaps.
const GapCategoryIncident = "incident"

// TimeWindowPadding is the duration added before/after the incident
// occurrence and resolution times to capture surrounding context.
var TimeWindowPadding = 30 * time.Minute
