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

	// Intelligence is deterministic, evidence-linked incident memory/guidance derived from stored MEL history.
	Intelligence *IncidentIntelligence `json:"intelligence,omitempty"`

	// ActionVisibility is server-computed control/action visibility posture for this API response (capability-aware).
	ActionVisibility *IncidentActionVisibilityPosture `json:"action_visibility,omitempty"`

	// TriageSignals are deterministic, inspectable queue-shaping hints (not ranked black-box scoring).
	TriageSignals *IncidentTriageSignals `json:"triage_signals,omitempty"`
	// AssistSignals are deterministic, inspectable local-intelligence cues — non-canonical vs persisted evidence rows.
	AssistSignals *IncidentAssistSignals `json:"assist_signals,omitempty"`

	// Review / workflow (migration 0031); orthogonal to control lifecycle state.
	ReviewState            string `json:"review_state,omitempty"`
	InvestigationNotes     string `json:"investigation_notes,omitempty"`
	ResolutionSummary      string `json:"resolution_summary,omitempty"`
	CloseoutReason         string `json:"closeout_reason,omitempty"`
	LessonsLearned         string `json:"lessons_learned,omitempty"`
	ReopenedFromIncidentID string `json:"reopened_from_incident_id,omitempty"`
	ReopenedAt             string `json:"reopened_at,omitempty"`

	// DecisionPack is the canonical backend-assembled incident decision object (detail, workbench, export framing).
	// Populated on API responses that run finishIncidentForAPI; not a second source of business truth.
	DecisionPack *IncidentDecisionPack `json:"decision_pack,omitempty"`
}

// IncidentDecisionPackSchemaVersion is bumped when section shapes or semantics change incompatibly.
const IncidentDecisionPackSchemaVersion = "1"

// IncidentDecisionPack is the versioned, backend-owned contract for incident triage, evidence, memory, and next steps.
// Interpretive layers are explicitly labeled; absence of data is honest partial state, not silent health.
type IncidentDecisionPack struct {
	SchemaVersion string `json:"schema_version"`
	IncidentID    string `json:"incident_id"`
	GeneratedAt   string `json:"generated_at"` // RFC3339 UTC from assembly

	Identity             *IncidentDecisionPackIdentity            `json:"identity,omitempty"`
	Queue                *IncidentDecisionPackQueue               `json:"queue,omitempty"`
	Guidance             *IncidentDecisionPackGuidance            `json:"guidance,omitempty"`
	EvidenceBasis        *IncidentDecisionPackEvidenceBasis       `json:"evidence_basis,omitempty"`
	IntelligenceSummary  *IncidentDecisionPackIntelligenceSummary `json:"intelligence_summary,omitempty"`
	MitigationDurability *IncidentMitigationDurabilityMemory      `json:"mitigation_durability,omitempty"`
	FamilyHistory        *IncidentDecisionPackFamilyHistory       `json:"family_history,omitempty"`
	TransportGraph       *IncidentDecisionPackTransportGraph      `json:"transport_graph,omitempty"`
	LinkedActions        *IncidentDecisionPackLinkedActions       `json:"linked_actions,omitempty"`
	Readiness            *IncidentDecisionPackReadiness           `json:"readiness,omitempty"`
	Uncertainty          *IncidentDecisionPackUncertainty         `json:"uncertainty,omitempty"`
	OperatorAdjudication *IncidentDecisionPackAdjudication        `json:"operator_adjudication,omitempty"`
	// AssistSignals re-exports the same bounded cues as incident.assist_signals for pack-first/export surfaces (no second computation).
	AssistSignals  *IncidentAssistSignals              `json:"assist_signals,omitempty"`
	AnalyticsHints *IncidentDecisionPackAnalyticsHints `json:"analytics_hints,omitempty"`
}

// IncidentDecisionPackGuidance is a deterministic operator posture primitive.
// It is bounded guidance (not certainty) and is reused across queue/detail/planning/topology/export surfaces.
type IncidentDecisionPackGuidance struct {
	NeedsAttention bool   `json:"needs_attention"`
	PriorityTier   int    `json:"priority_tier,omitempty"` // deterministic 0..4, lower means sooner review
	WhyNow         string `json:"why_now,omitempty"`

	ReviewRecommended  bool   `json:"review_recommended"`
	VerifyBeforeAction bool   `json:"verify_before_action"`
	EvidencePosture    string `json:"evidence_posture,omitempty"` // strong | moderate | sparse | degraded | unknown

	MitigationFragilityWatch bool `json:"mitigation_fragility_watch"`
	RepeatedFamilyConcern    bool `json:"repeated_family_concern"`

	ActionPosture  string `json:"action_posture,omitempty"`  // available | guarded | unsupported | verify_linkage
	SupportPosture string `json:"support_posture,omitempty"` // ready | partial | blocked | unknown

	TopologyPlanningPosture string   `json:"topology_planning_posture,omitempty"` // useful_non_proving
	EscalationPosture       string   `json:"escalation_posture,omitempty"`        // replay_first | follow_up | bounded_review
	Degraded                bool     `json:"degraded"`
	DegradedReasons         []string `json:"degraded_reasons,omitempty"`
}

type IncidentDecisionPackIdentity struct {
	Title          string `json:"title,omitempty"`
	State          string `json:"state,omitempty"`
	Severity       string `json:"severity,omitempty"`
	Category       string `json:"category,omitempty"`
	ResourceType   string `json:"resource_type,omitempty"`
	ResourceID     string `json:"resource_id,omitempty"`
	OccurredAt     string `json:"occurred_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
	ResolvedAt     string `json:"resolved_at,omitempty"`
	ReviewState    string `json:"review_state,omitempty"`
	OwnerActorID   string `json:"owner_actor_id,omitempty"`
	HandoffSummary string `json:"handoff_summary,omitempty"`
	Summary        string `json:"summary,omitempty"`
}

type IncidentDecisionPackQueue struct {
	TriageSignals       *IncidentTriageSignals `json:"triage_signals,omitempty"`
	WhySurfacedOneLiner string                 `json:"why_surfaced_one_liner,omitempty"`
	OrderingNote        string                 `json:"ordering_note,omitempty"`
}

type IncidentDecisionPackEvidenceBasis struct {
	EvidenceStrength string                 `json:"evidence_strength,omitempty"`
	EvidenceItems    []IncidentEvidenceItem `json:"evidence_items,omitempty"`
	ItemCapApplied   int                    `json:"item_cap_applied,omitempty"`
	Degraded         bool                   `json:"degraded"`
	DegradedReasons  []string               `json:"degraded_reasons,omitempty"`
	SparsityMarkers  []string               `json:"sparsity_markers,omitempty"`
}

type IncidentDecisionPackIntelligenceSummary struct {
	SignatureLabel      string   `json:"signature_label,omitempty"`
	SignatureMatchCount int      `json:"signature_match_count,omitempty"`
	SummaryLines        []string `json:"summary_lines,omitempty"`
	InvestigateNextIDs  []string `json:"investigate_next_ids,omitempty"`
	RunbookRecIDs       []string `json:"runbook_recommendation_ids,omitempty"`
	LearningLoopHints   []string `json:"learning_loop_hints,omitempty"`
}

type IncidentDecisionPackFamilyHistory struct {
	SignatureFamily  *IncidentSignatureFamilyResolvedHistory `json:"signature_family,omitempty"`
	SimilarIncidents []IncidentSimilarityRecord              `json:"similar_incidents,omitempty"`
	SimilarScanCap   int                                     `json:"similar_scan_cap,omitempty"`
}

type IncidentDecisionPackTransportGraph struct {
	MeshRoutingCompanion *MeshRoutingIntelCompanion `json:"mesh_routing_companion,omitempty"`
	WirelessContext      *IncidentWirelessContext   `json:"wireless_context,omitempty"`
}

type IncidentDecisionPackLinkedActions struct {
	ActionVisibility        *IncidentActionVisibilityPosture `json:"action_visibility,omitempty"`
	OperatorSuggestedAction []OperatorSuggestedAction        `json:"operator_suggested_actions,omitempty"`
	LinkedControlActionIDs  []string                         `json:"linked_control_action_ids,omitempty"`
}

type IncidentDecisionPackReadiness struct {
	ExportPolicySemantic    string   `json:"export_policy_semantic,omitempty"`
	ExportPolicySummary     string   `json:"export_policy_summary,omitempty"`
	ExportArtifactStrength  string   `json:"export_artifact_strength,omitempty"`
	ExportBlockerCodes      []string `json:"export_blocker_codes,omitempty"`
	ProofpackPath           string   `json:"proofpack_path,omitempty"`
	EscalationBundlePath    string   `json:"escalation_bundle_path,omitempty"`
	HandoffPostureNote      string   `json:"handoff_posture_note,omitempty"`
	EvidenceSufficiencyNote string   `json:"evidence_sufficiency_note,omitempty"`
}

type IncidentDecisionPackUncertainty struct {
	NonClaims              []string `json:"non_claims,omitempty"`
	InterpretationLabels   []string `json:"interpretation_labels,omitempty"`
	DegradedSections       []string `json:"degraded_sections,omitempty"`
	BoundedScanDisclosures []string `json:"bounded_scan_disclosures,omitempty"`
}

// IncidentDecisionPackAdjudication is operator feedback on the pack (durable; separate from assistive rec outcomes).
type IncidentDecisionPackAdjudication struct {
	Reviewed          bool                             `json:"reviewed"`
	ReviewedAt        string                           `json:"reviewed_at,omitempty"`
	ReviewedByActorID string                           `json:"reviewed_by_actor_id,omitempty"`
	Useful            string                           `json:"useful,omitempty"` // useful | not_useful | ""
	OperatorNote      string                           `json:"operator_note,omitempty"`
	CueOutcomes       []IncidentDecisionPackCueOutcome `json:"cue_outcomes,omitempty"`
	UpdatedAt         string                           `json:"updated_at,omitempty"`
}

type IncidentDecisionPackCueOutcome struct {
	CueID      string `json:"cue_id"` // stable id: matches assist signal code
	Outcome    string `json:"outcome"`
	Note       string `json:"note,omitempty"`
	ReasonCode string `json:"reason_code,omitempty"` // optional taxonomy: false_positive | needs_more_evidence | defer | other
}

// IncidentDecisionPackAnalyticsHints carries stable keys for future compounding analytics (no aggregation here).
type IncidentDecisionPackAnalyticsHints struct {
	SignatureKey                string `json:"signature_key,omitempty"`
	FingerprintCanonicalHash    string `json:"fingerprint_canonical_hash,omitempty"`
	TriageTier                  int    `json:"triage_tier,omitempty"`
	MitigationDurabilityPosture string `json:"mitigation_durability_posture,omitempty"`
	EvidenceStrength            string `json:"evidence_strength,omitempty"`
	IntelDegraded               bool   `json:"intel_degraded"`
}

// IncidentDecisionPackAdjudicationPatch is the body for PATCH .../decision-pack.
type IncidentDecisionPackAdjudicationPatch struct {
	Reviewed           *bool                            `json:"reviewed,omitempty"`
	Useful             *string                          `json:"useful,omitempty"`
	OperatorNote       *string                          `json:"operator_note,omitempty"`
	CueOutcomes        []IncidentDecisionPackCueOutcome `json:"cue_outcomes,omitempty"`
	ReplaceCueOutcomes bool                             `json:"replace_cue_outcomes,omitempty"`
}

// IncidentActionVisibilityPosture encodes deterministic visibility truth for operators (no implied workflow certainty).
type IncidentActionVisibilityPosture struct {
	Kind                     string `json:"action_visibility_kind"`
	Reason                   string `json:"action_visibility_reason,omitempty"`
	Summary                  string `json:"action_visibility_summary"`
	SuggestControlQueue      bool   `json:"action_context_should_open_control_queue"`
	HasMaterialPriorAttempts bool   `json:"action_context_has_material_prior_attempts"`
	HasPendingRelatedWork    bool   `json:"action_context_has_pending_related_work"`
	IsPartial                bool   `json:"action_context_is_partial"`
	LinkedRowCount           int    `json:"linked_control_row_count,omitempty"`
	PendingRefCount          int    `json:"pending_action_ref_count,omitempty"`
	RecentActionRefCount     int    `json:"recent_action_ref_count,omitempty"`
}

// IncidentIntelligence is an evidence-bounded operator aid for recurring signatures and investigation flow.
type IncidentIntelligence struct {
	SignatureKey            string                          `json:"signature_key,omitempty"`
	SignatureLabel          string                          `json:"signature_label,omitempty"`
	SignatureMatchCount     int                             `json:"signature_match_count,omitempty"`
	Fingerprint             *IncidentFingerprint            `json:"fingerprint,omitempty"`
	EvidenceStrength        string                          `json:"evidence_strength"` // sparse, moderate, strong
	EvidenceItems           []IncidentEvidenceItem          `json:"evidence_items,omitempty"`
	ImplicatedDomains       []IncidentDomainHint            `json:"implicated_domains,omitempty"`
	WirelessContext         *IncidentWirelessContext        `json:"wireless_context,omitempty"`
	InvestigateNext         []IncidentGuidanceItem          `json:"investigate_next,omitempty"`
	SimilarIncidents        []IncidentSimilarityRecord      `json:"similar_incidents,omitempty"`
	HistoricallyUsedActions []IncidentActionPattern         `json:"historically_used_actions,omitempty"`
	ActionOutcomeMemory     []IncidentActionOutcomeMemory   `json:"action_outcome_memory,omitempty"`
	ActionOutcomeSnapshots  []IncidentActionOutcomeSnapshot `json:"action_outcome_snapshots,omitempty"`
	ActionOutcomeTrace      *IncidentActionOutcomeTrace     `json:"action_outcome_trace,omitempty"`
	Degraded                bool                            `json:"degraded"`
	DegradedReasons         []string                        `json:"degraded_reasons,omitempty"`
	SparsityMarkers         []string                        `json:"sparsity_markers,omitempty"`
	RunbookRecommendations  []IncidentRunbookRecommendation `json:"runbook_recommendations,omitempty"`
	RunbookAssets           []IncidentRunbookAsset          `json:"runbook_assets,omitempty"`
	PolicyGovernanceHints   []IncidentPolicyGovernanceHint  `json:"policy_governance_hints,omitempty"`
	GovernanceMemory        []IncidentGovernanceMemory      `json:"governance_memory,omitempty"`
	DriftFingerprints       []IncidentDriftFingerprint      `json:"drift_fingerprints,omitempty"`
	CorrelationGroups       []IncidentCorrelationGroup      `json:"correlation_groups,omitempty"`
	FaultDomains            []IncidentFaultDomain           `json:"fault_domains,omitempty"`
	ReplayHints             *IncidentReplayHints            `json:"replay_hints,omitempty"`
	LearningLoopHints       []string                        `json:"learning_loop_hints,omitempty"`
	GeneratedAt             string                          `json:"generated_at,omitempty"`
	// MeshRoutingCompanion is optional ingest-graph routing-pressure context (never RF/path proof).
	MeshRoutingCompanion *MeshRoutingIntelCompanion `json:"mesh_routing_companion,omitempty"`
	// OperatorSuggestedActions are deterministic, reviewable next checks — not ranked black-box scoring.
	OperatorSuggestedActions []OperatorSuggestedAction `json:"operator_suggested_actions,omitempty"`
	// SignatureFamilyResolvedHistory summarizes resolved/reopened peers sharing this incident's signature (local DB only).
	SignatureFamilyResolvedHistory *IncidentSignatureFamilyResolvedHistory `json:"signature_family_resolved_history,omitempty"`
	// MitigationDurabilityMemory is deterministic local-history posture for this incident/family — not prediction of success.
	MitigationDurabilityMemory *IncidentMitigationDurabilityMemory `json:"mitigation_durability_memory,omitempty"`
}

// IncidentMitigationDurabilityMemory summarizes stored outcome/family rows only — association, not causal proof.
// SchemaVersion documents the DTO shape for export/API parity (mitigation_durability_memory_v2).
type IncidentMitigationDurabilityMemory struct {
	SchemaVersion string `json:"schema_version,omitempty"`
	// Posture is the primary normalized posture code (stable enum string for operators and export).
	Posture string `json:"posture"`
	// ReasonCodes are normalized taxonomy entries for why this posture was chosen (subset of posture-specific codes).
	ReasonCodes []string `json:"reason_codes,omitempty"`
	Summary     string   `json:"summary"`
	// Scope bounds what the summary claims (instance_record, signature_family_scan, outcome_memory_aggregate, etc.).
	Scope *IncidentDurabilityScope `json:"scope,omitempty"`
	// Basis lists deterministic derivation steps (field paths), not narrative proof.
	Basis *IncidentDurabilityBasis `json:"basis,omitempty"`
	// NonClaims explicitly states what this memory does not assert.
	NonClaims    []string `json:"non_claims,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
	Uncertainty  string   `json:"uncertainty"`
}

// IncidentDurabilityScope bounds mitigation durability statements to stored evidence classes.
type IncidentDurabilityScope struct {
	Primary string   `json:"primary"` // e.g. instance_record | signature_family_bounded_scan | action_outcome_memory
	Detail  []string `json:"detail,omitempty"`
}

// IncidentDurabilityBasis ties the memory object to deterministic inputs (evidence paths, counts).
type IncidentDurabilityBasis struct {
	Inputs      []string       `json:"inputs,omitempty"`       // e.g. incident.signature_family_resolved_history
	Counts      map[string]int `json:"counts,omitempty"`       // e.g. resolved_peer_count -> 2
	ScanPosture string         `json:"scan_posture,omitempty"` // full_family | bounded_recent_peers
}

// IncidentAssistSignals groups bounded assistive incident-intelligence outputs (deterministic, not LLM).
type IncidentAssistSignals struct {
	SchemaVersion string                 `json:"schema_version"` // incident_assist_signals_v1
	Signals       []IncidentAssistSignal `json:"signals,omitempty"`
	Uncertainty   string                 `json:"uncertainty,omitempty"`
	EvidenceBasis string                 `json:"evidence_basis,omitempty"` // high-level note on inputs used
}

// IncidentAssistSignal is one inspectable assist cue with explicit codes and uncertainty.
type IncidentAssistSignal struct {
	Code         string   `json:"code"`
	Severity     string   `json:"severity,omitempty"` // info | watch | review
	Title        string   `json:"title"`
	Rationale    string   `json:"rationale"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
	Uncertainty  string   `json:"uncertainty,omitempty"`
	// OperatorState summarizes durable adjudication for this signal code when rows exist (non-authoritative snapshot).
	OperatorState *IncidentAssistSignalOperatorState `json:"operator_state,omitempty"`
}

// IncidentAssistSignalOperatorState is derived from persisted incident_intel_signal_outcomes (latest per signal_code).
type IncidentAssistSignalOperatorState struct {
	LatestOutcome string `json:"latest_outcome,omitempty"`
	LatestAt      string `json:"latest_at,omitempty"`
	ActorID       string `json:"actor_id,omitempty"`
}

// IncidentIntelSignalOutcomeRequest records operator adjudication for a deterministic assist signal code.
type IncidentIntelSignalOutcomeRequest struct {
	SignalCode string `json:"signal_code"`
	Outcome    string `json:"outcome"` // dismissed | accepted | reviewed | snoozed
	Note       string `json:"note,omitempty"`
}

// IncidentSignatureFamilyResolvedHistory is observational recurrence context from stored incidents — not prediction.
// IncidentTriageSignals mirrors internal/incidenttriage (JSON-stable for operators).
type IncidentTriageSignals struct {
	Tier             int      `json:"tier"`
	Codes            []string `json:"codes,omitempty"`
	RationaleLines   []string `json:"rationale_lines,omitempty"`
	EvidenceRefs     []string `json:"evidence_refs,omitempty"`
	UncertaintyNotes []string `json:"uncertainty_notes,omitempty"`
	// Queue ordering contract: same semantics as Tier for open-incident work ordering; explicit for cross-surface parity.
	QueueOrderingContract string `json:"queue_ordering_contract,omitempty"`
	// QueueOrderingContractVersion distinguishes contract upgrades (open_incident_workbench_v2 adds full tuple).
	QueueOrderingContractVersion string `json:"queue_ordering_contract_version,omitempty"`
	QueueSortPrimary             int    `json:"queue_sort_primary,omitempty"`
	// QueueSortSecondary is a human hint; use QueueSortSecondaryNumeric + tuple for machine ordering.
	QueueSortSecondary         string  `json:"queue_sort_secondary,omitempty"`
	QueueSortSecondaryNumeric  int64   `json:"queue_sort_secondary_numeric,omitempty"`  // nanoseconds since epoch when valid; 0 when missing/invalid
	QueueSortSecondaryValidity string  `json:"queue_sort_secondary_validity,omitempty"` // valid_rfc3339 | missing | invalid_timestamp
	QueueSortTieBreak          string  `json:"queue_sort_tie_break,omitempty"`          // incident_id_lex_asc
	QueueSortTieBreakNumeric   int64   `json:"queue_sort_tie_break_numeric,omitempty"`  // stable hash for id tie-break when present
	QueueSortTuple             []int64 `json:"queue_sort_tuple,omitempty"`              // [primary, recency_inverted_ns, tie_break] — prefer queue_sort_key_lex in JSON clients (int64 ns may exceed JS safe integer).
	// QueueSortKeyLex sorts lexicographically ascending with the same order as the tuple (tier, recency, tie-break).
	QueueSortKeyLex      string   `json:"queue_sort_key_lex,omitempty"`
	OrderingRationale    string   `json:"ordering_rationale,omitempty"`
	OrderingEvidenceRefs []string `json:"ordering_evidence_refs,omitempty"`
	OrderingUncertainty  string   `json:"ordering_uncertainty,omitempty"`
	// QueueOrderingPosture is explicit client truth: canonical when full v2 keys present; degraded when recency/tie-break had to use safe fallbacks.
	QueueOrderingPosture string `json:"queue_ordering_posture,omitempty"` // canonical_v2 | degraded_partial_recency
	// QueueOrderingDegradedReasons lists machine-readable reasons when posture is degraded (e.g. invalid_timestamp, missing_updated_at).
	QueueOrderingDegradedReasons []string `json:"queue_ordering_degraded_reasons,omitempty"`
}

type IncidentSignatureFamilyResolvedHistory struct {
	FamilyMatchTotal     int    `json:"family_match_total"`
	ResolvedPeerCount    int    `json:"resolved_peer_count"`
	ReopenedPeerCount    int    `json:"reopened_peer_count"`
	Basis                string `json:"basis"`
	Uncertainty          string `json:"uncertainty"`
	PeerSampleIncidentID string `json:"peer_sample_incident_id,omitempty"`
	// When true, resolved/reopened counts are computed only over the most recent PeerScanWindow linked peers — not the full family. FamilyMatchTotal remains an exact COUNT.
	PeerHistoryScanTruncated bool `json:"peer_history_scan_truncated,omitempty"`
	PeerScanWindow           int  `json:"peer_scan_window,omitempty"`
}

// MeshRoutingIntelCompanion surfaces bounded mesh routing-pressure diagnostics next to mesh-scoped incidents.
type MeshRoutingIntelCompanion struct {
	Applicable                     bool     `json:"applicable"`
	Reason                         string   `json:"reason,omitempty"`
	TopologyEnabled                bool     `json:"topology_enabled,omitempty"`
	TransportConnected             bool     `json:"transport_connected,omitempty"`
	AssessmentComputedAt           string   `json:"assessment_computed_at,omitempty"`
	GraphHash                      string   `json:"graph_hash,omitempty"`
	EvidenceModel                  string   `json:"evidence_model,omitempty"`
	MessageWindowDescription       string   `json:"message_window_description,omitempty"`
	RoutingSummaryLines            []string `json:"routing_summary_lines,omitempty"`
	SuspectedRelayHotspot          bool     `json:"suspected_relay_hotspot"`
	WeakOnwardPropagationSuspected bool     `json:"weak_onward_propagation_suspected"`
	HopBudgetStressSuspected       bool     `json:"hop_budget_stress_suspected"`
	// SuggestedTopologySearch is URL query (no leading ?) for /topology deep link — incident_focus filter when incident id known.
	SuggestedTopologySearch string `json:"suggested_topology_search,omitempty"`
}

// OperatorSuggestedAction is a single deterministic operator affordance with explicit evidence basis.
type OperatorSuggestedAction struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Rationale    string   `json:"rationale"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
	Uncertainty  string   `json:"uncertainty,omitempty"`
	Href         string   `json:"href,omitempty"`
	Kind         string   `json:"kind"` // inspect_surface | correlation_memory
	DisableHint  string   `json:"disable_hint,omitempty"`
}

// IncidentFingerprint is a versioned structured fingerprint derived from persisted and correlated evidence.
type IncidentFingerprint struct {
	SchemaVersion      string              `json:"schema_version"`
	ProfileVersion     string              `json:"profile_version"`
	LegacySignatureKey string              `json:"legacy_signature_key,omitempty"`
	CanonicalHash      string              `json:"canonical_hash"`
	Components         map[string][]string `json:"components,omitempty"`
	SparsityMarkers    []string            `json:"sparsity_markers,omitempty"`
	ComputedAt         string              `json:"computed_at,omitempty"`
}

// IncidentRunbookAsset is a durable runbook/playbook unit traceable to operational history.
type IncidentRunbookAsset struct {
	ID                 string   `json:"id"`
	Status             string   `json:"status"` // proposed, approved, deprecated, historical_only
	SourceKind         string   `json:"source_kind"`
	Title              string   `json:"title"`
	Body               string   `json:"body,omitempty"`
	EvidenceRefs       []string `json:"evidence_refs,omitempty"`
	SourceIncidentIDs  []string `json:"source_incident_ids,omitempty"`
	LegacySignatureKey string   `json:"legacy_signature_key,omitempty"`
	FingerprintHash    string   `json:"fingerprint_canonical_hash,omitempty"`
	PromotionBasis     string   `json:"promotion_basis,omitempty"`
	CreatedAt          string   `json:"created_at,omitempty"`
	UpdatedAt          string   `json:"updated_at,omitempty"`
}

// IncidentFaultDomain groups cross-signal members with explicit uncertainty (not root-cause proof).
type IncidentFaultDomain struct {
	DomainID       string            `json:"domain_id"`
	DomainKey      string            `json:"domain_key"`
	Basis          string            `json:"basis"`
	Uncertainty    string            `json:"uncertainty"` // likely_related, possibly_related, shared_symptoms_only, inconclusive
	Rationale      []string          `json:"rationale,omitempty"`
	EvidenceBundle map[string]string `json:"evidence_bundle,omitempty"`
	MemberKinds    []string          `json:"member_kinds,omitempty"`
}

// IncidentGovernanceMemory summarizes historical adjudication posture for action classes (observational).
type IncidentGovernanceMemory struct {
	ActionType            string   `json:"action_type"`
	Summary               string   `json:"summary"`
	LinkedActionCount     int      `json:"linked_action_count"`
	ApprovedOrPassedCount int      `json:"approved_or_passed_count"`
	RejectedCount         int      `json:"rejected_count"`
	HighBlastCount        int      `json:"high_blast_count"`
	SeparateApproverCount int      `json:"separate_approver_count"`
	EvidenceRefs          []string `json:"evidence_refs,omitempty"`
}

type IncidentActionOutcomeTrace struct {
	ExpectedSnapshotWrites  int      `json:"expected_snapshot_writes"`
	SnapshotWriteFailures   int      `json:"snapshot_write_failures"`
	SnapshotWriteFailureIDs []string `json:"snapshot_write_failure_ids,omitempty"`
	SnapshotRetrievalStatus string   `json:"snapshot_retrieval_status"` // available, unavailable, error
	SnapshotRetrievalReason string   `json:"snapshot_retrieval_reason,omitempty"`
	SnapshotRetrievalError  string   `json:"snapshot_retrieval_error,omitempty"`
	PersistedSnapshotCount  int      `json:"persisted_snapshot_count"`
	Completeness            string   `json:"completeness"` // complete, partial, unavailable
}

type IncidentEvidenceItem struct {
	Kind         string `json:"kind"`
	ReferenceID  string `json:"reference_id,omitempty"`
	Summary      string `json:"summary"`
	ObservedAt   string `json:"observed_at,omitempty"`
	SupportsOnly string `json:"supports_only,omitempty"` // association, chronology, recurring_pattern
}

type IncidentDomainHint struct {
	Domain       string   `json:"domain"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
	Note         string   `json:"note,omitempty"`
}

type IncidentGuidanceItem struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Rationale    string   `json:"rationale"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
	Confidence   string   `json:"confidence"` // low, medium
}

// IncidentRunbookRecommendation is assistive, non-command guidance ranked from stored history and governance fields.
type IncidentRunbookRecommendation struct {
	ID                    string   `json:"id"`
	Title                 string   `json:"title"`
	ActionType            string   `json:"action_type,omitempty"`
	Rationale             string   `json:"rationale"`
	EvidenceRefs          []string `json:"evidence_refs,omitempty"`
	Strength              string   `json:"strength"` // historically_proven, historically_promising, plausible, weakly_supported, unsupported
	StrengthExplanation   []string `json:"strength_explanation,omitempty"`
	RankScore             float64  `json:"rank_score,omitempty"`
	RequiresApproval      bool     `json:"requires_approval"`
	BlastRadiusClass      string   `json:"blast_radius_class,omitempty"`
	Reversibility         string   `json:"reversibility"` // high, medium, low, unknown
	PriorOutcomeFraming   string   `json:"prior_outcome_framing,omitempty"`
	PriorSampleSize       int      `json:"prior_sample_size,omitempty"`
	HistoricalOutcomeNote string   `json:"historical_outcome_note,omitempty"`
	Suppressed            bool     `json:"suppressed,omitempty"`
	SuppressedReason      string   `json:"suppressed_reason,omitempty"`
	IsCommand             bool     `json:"is_command"`
}

// IncidentPolicyGovernanceHint summarizes observable approval / risk posture from linked actions (not policy mutation).
type IncidentPolicyGovernanceHint struct {
	Summary      string   `json:"summary"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
	Posture      string   `json:"posture"` // informational
}

// IncidentDriftFingerprint compares bounded transport anomaly history to the incident window (association only).
type IncidentDriftFingerprint struct {
	Kind              string `json:"kind"` // transport_anomaly_reason_recurring
	TransportName     string `json:"transport_name,omitempty"`
	Reason            string `json:"reason,omitempty"`
	Statement         string `json:"statement"`
	CurrentBucketHits int    `json:"current_bucket_hits"`
	PriorBucketHits   int    `json:"prior_bucket_hits"`
	SupportsOnly      string `json:"supports_only"`
}

// IncidentCorrelationGroup is a persisted structural grouping; not causal proof.
type IncidentCorrelationGroup struct {
	GroupID          string   `json:"group_id"`
	CorrelationKey   string   `json:"correlation_key"`
	Basis            string   `json:"basis"`
	CreatedAt        string   `json:"created_at,omitempty"`
	UpdatedAt        string   `json:"updated_at,omitempty"`
	Rationale        []string `json:"rationale,omitempty"`
	EvidenceRefs     []string `json:"evidence_refs,omitempty"`
	UncertaintyNote  string   `json:"uncertainty_note,omitempty"`
	MemberCount      int      `json:"member_count,omitempty"`
	OtherIncidentIDs []string `json:"other_incident_ids,omitempty"`
}

// IncidentReplayHints are pointers for post-incident review; not simulation output.
type IncidentReplayHints struct {
	Statement          string   `json:"statement"`
	EvidenceAtTimeRefs []string `json:"evidence_at_time_refs,omitempty"`
	CounterfactualNote string   `json:"counterfactual_note,omitempty"`
	RankingModelNote   string   `json:"ranking_model_note,omitempty"`
}

// IncidentRecommendationOutcomeRecord is persisted operator feedback on a recommendation id.
type IncidentRecommendationOutcomeRecord struct {
	ID               string `json:"id"`
	IncidentID       string `json:"incident_id"`
	RecommendationID string `json:"recommendation_id"`
	Outcome          string `json:"outcome"`
	ActorID          string `json:"actor_id,omitempty"`
	Note             string `json:"note,omitempty"`
	CreatedAt        string `json:"created_at"`
}

// IncidentWorkflowPatch is the body for PATCH /api/v1/incidents/{id}/workflow.
type IncidentWorkflowPatch struct {
	ReviewState            *string `json:"review_state,omitempty"`
	InvestigationNotes     *string `json:"investigation_notes,omitempty"`
	ResolutionSummary      *string `json:"resolution_summary,omitempty"`
	CloseoutReason         *string `json:"closeout_reason,omitempty"`
	LessonsLearned         *string `json:"lessons_learned,omitempty"`
	ReopenedFromIncidentID *string `json:"reopened_from_incident_id,omitempty"`
}

// IncidentRecommendationOutcomeRequest records how an operator adjudicated assistive guidance.
type IncidentRecommendationOutcomeRequest struct {
	RecommendationID string `json:"recommendation_id"`
	Outcome          string `json:"outcome"`
	Note             string `json:"note,omitempty"`
}

type IncidentWirelessContext struct {
	Classification    string                        `json:"classification"` // lora_mesh_pressure, bluetooth_onboarding_issue, wifi_backhaul_instability, mixed_path_degradation, sparse_evidence_incident, recurring_unknown_pattern, unsupported_wireless_domain_observed
	PrimaryDomain     string                        `json:"primary_domain"` // lora, bluetooth, wifi, mixed, unknown
	ObservedDomains   []string                      `json:"observed_domains,omitempty"`
	EvidencePosture   string                        `json:"evidence_posture"`   // live, historical, partial, sparse, imported, unsupported
	ConfidencePosture string                        `json:"confidence_posture"` // evidence_backed, mixed, sparse, inconclusive
	Summary           string                        `json:"summary"`
	Reasons           []IncidentWirelessReason      `json:"reasons,omitempty"`
	EvidenceGaps      []string                      `json:"evidence_gaps,omitempty"`
	InspectNext       []string                      `json:"inspect_next,omitempty"`
	Unsupported       []IncidentWirelessUnsupported `json:"unsupported,omitempty"`
}

type IncidentWirelessReason struct {
	Code         string   `json:"code"`
	Statement    string   `json:"statement"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type IncidentWirelessUnsupported struct {
	Domain string `json:"domain"`
	Scope  string `json:"scope"`
	Note   string `json:"note"`
}

type IncidentSimilarityRecord struct {
	IncidentID           string   `json:"incident_id"`
	Title                string   `json:"title,omitempty"`
	State                string   `json:"state,omitempty"`
	OccurredAt           string   `json:"occurred_at,omitempty"`
	SimilarityReason     []string `json:"similarity_reason,omitempty"`
	MatchCategory        string   `json:"match_category,omitempty"`
	WeightedScore        float64  `json:"weighted_score,omitempty"`
	MatchedDimensions    []string `json:"matched_dimensions,omitempty"`
	UnmatchedDimensions  []string `json:"unmatched_dimensions,omitempty"`
	WeakSparseDimensions []string `json:"weak_sparse_dimensions,omitempty"`
	InsufficientEvidence bool     `json:"insufficient_evidence,omitempty"`
	MatchExplanation     []string `json:"match_explanation,omitempty"`
}

type IncidentActionPattern struct {
	ActionType string `json:"action_type"`
	Count      int    `json:"count"`
}

type IncidentActionOutcomeMemory struct {
	ActionType                 string   `json:"action_type"`
	ActionLabel                string   `json:"action_label,omitempty"`
	OccurrenceCount            int      `json:"occurrence_count"`
	SampleSize                 int      `json:"sample_size"`
	OutcomeFraming             string   `json:"outcome_framing"`   // improvement_observed, deterioration_observed, mixed_historical_evidence, insufficient_evidence, no_clear_post_action_signal
	EvidenceStrength           string   `json:"evidence_strength"` // sparse, moderate, strong
	ObservedPostActionStatus   string   `json:"observed_post_action_status"`
	ImprovementObservedCount   int      `json:"improvement_observed_count"`
	DeteriorationObservedCount int      `json:"deterioration_observed_count"`
	InconclusiveCount          int      `json:"inconclusive_count"`
	Caveats                    []string `json:"caveats,omitempty"`
	InspectBeforeReuse         []string `json:"inspect_before_reuse,omitempty"`
	EvidenceRefs               []string `json:"evidence_refs,omitempty"`
	SnapshotRefs               []string `json:"snapshot_refs,omitempty"`
	SnapshotTraceStatus        string   `json:"snapshot_trace_status"`     // complete, partial, unavailable
	SnapshotCoveragePosture    string   `json:"snapshot_coverage_posture"` // matched, sparse, missing
	SnapshotCoveragePercent    float64  `json:"snapshot_coverage_percent"` // 0..100
}

type IncidentActionEvidenceSummary struct {
	TransportName        string `json:"transport_name,omitempty"`
	DeadLettersCount     int    `json:"dead_letters_count"`
	TransportAlertsCount int    `json:"transport_alerts_count"`
	IncidentState        string `json:"incident_state,omitempty"`
	ActionResult         string `json:"action_result,omitempty"`
	ActionLifecycle      string `json:"action_lifecycle,omitempty"`
}

type IncidentActionOutcomeSnapshot struct {
	SnapshotID            string                        `json:"snapshot_id"`
	SignatureKey          string                        `json:"signature_key"`
	IncidentID            string                        `json:"incident_id"`
	ActionID              string                        `json:"action_id"`
	ActionType            string                        `json:"action_type"`
	ActionLabel           string                        `json:"action_label,omitempty"`
	DerivedClassification string                        `json:"derived_classification"` // improvement_observed, deterioration_observed, mixed_historical_evidence, inconclusive, insufficient_evidence
	EvidenceSufficiency   string                        `json:"evidence_sufficiency"`   // sufficient, partial, insufficient
	WindowStart           string                        `json:"window_start"`
	WindowEnd             string                        `json:"window_end"`
	PreActionEvidence     IncidentActionEvidenceSummary `json:"pre_action_evidence"`
	PostActionEvidence    IncidentActionEvidenceSummary `json:"post_action_evidence"`
	ObservedSignalCount   int                           `json:"observed_signal_count"`
	Caveats               []string                      `json:"caveats,omitempty"`
	InspectBeforeReuse    []string                      `json:"inspect_before_reuse,omitempty"`
	EvidenceRefs          []string                      `json:"evidence_refs,omitempty"`
	AssociationOnly       bool                          `json:"association_only"`
	DerivationVersion     string                        `json:"derivation_version,omitempty"`
	SchemaVersion         string                        `json:"schema_version,omitempty"`
	DerivedAt             string                        `json:"derived_at"`
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

	// Policy / approval truth (aligned with control_actions; migration 0022)
	ApprovalMode                      string   `json:"approval_mode,omitempty"`
	RequiredApprovals                 int      `json:"required_approvals,omitempty"`
	CollectedApprovals                int      `json:"collected_approvals,omitempty"`
	ApprovalBasis                     []string `json:"approval_basis,omitempty"`
	ApprovalPolicySource              string   `json:"approval_policy_source,omitempty"`
	HighBlastRadius                   bool     `json:"high_blast_radius,omitempty"`
	ApprovalEscalatedDueToBlastRadius bool     `json:"approval_escalated_due_to_blast_radius,omitempty"`
	ExecutionSource                   string   `json:"execution_source,omitempty"`

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
	SubmitterDisqualifiedFromApproval bool     `json:"submitter_disqualified_from_approval"`
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
	Status   string `json:"status"`
	ActionID string `json:"action_id"`
	ActorID  string `json:"actor"`

	LifecycleState string `json:"lifecycle_state"`
	Result         string `json:"result"`

	FullyApprovedSingleStep       bool `json:"fully_approved_single_step"`
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
	Status         string            `json:"status"`
	ActionID       string            `json:"action_id"`
	ActorID        string            `json:"actor"`
	LifecycleState string            `json:"lifecycle_state"`
	Result         string            `json:"result"`
	Policy         ApprovalPolicyDTO `json:"policy"`
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
