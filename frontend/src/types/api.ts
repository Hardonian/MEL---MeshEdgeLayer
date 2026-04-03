// MEL API Types - Derived from Go backend structure

/** GET /api/v1/version — build and schema metadata from the running binary */
export interface VersionResponse {
  version: string
  /** Topology ingest + graph store enabled in running config (not “live mesh proof”). */
  topology_model_enabled?: boolean
  git_commit?: string
  build_time?: string
  go_version?: string
  db_schema_version?: number
  db_actual_version?: string
  db_migration_numeric?: number
  schema_matches_binary?: boolean
  compatibility_level?: string
  config_canonical_fingerprint?: string
  boot_metadata?: Record<string, unknown>
  /** Canonical product boundary (single-gateway operator scope). */
  product?: ProductEnvelope
  instance_id?: string
  /** Canonical instance/site/fleet boundary when DB is available (same shape as GET /api/v1/fleet/truth). */
  fleet_truth?: FleetTruthSummary
  process?: { pid: number; started_at: string }
  uptime_seconds?: number
  platform_posture?: PlatformPosture
  /** Canonical operator export/support/proofpack readiness from backend (GET /api/v1/version). */
  operator_readiness?: OperatorReadinessDTO
}

export interface OperatorIntelligencePosture {
  deterministic_incident_intel: string
  deterministic_basis: string
  telemetry_outbound: boolean
  telemetry_require_explicit_opt_in: boolean
  assistive_inference_layer: string
  /** Explicit contract for future assist UI — does not imply remote/cloud assist. */
  assist_capability_strategy?: string
  assist_non_canonical_truth: boolean
  remote_assist_supported: boolean
  review_recommended_for_assist_output: boolean
  /** Stable field names bounded assist may read; deterministic API remains canonical. */
  assist_input_contracts?: string[]
  assist_disable_semantics?: string
  assist_audit_expectation?: string
}

export interface OperatorReadinessBlocker {
  code: string
  summary: string
}

export interface OperatorReadinessDTO {
  semantic: OperatorReadinessSemantic
  summary: string
  artifact_strength: OperatorArtifactStrengthDTO
  blockers?: OperatorReadinessBlocker[]
  evidence_basis?: string[]
  generated_from_note?: string
}

export type OperatorReadinessSemantic =
  | 'available'
  | 'degraded'
  | 'gated'
  | 'unsupported'
  | 'unavailable'
  | 'unknown_partial'
  | 'sparse'
  | 'partial'
  | 'capability_limited'
  | 'policy_limited'
  | 'stale'

export type OperatorArtifactStrengthDTO =
  | 'useful_now'
  | 'usable_degraded'
  | 'weaker_until_runtime_checked'
  | 'blocked'

export interface PlatformPosture {
  mode: string
  telemetry_enabled: boolean
  telemetry_outbound: boolean
  telemetry_require_explicit_opt_in: boolean
  retention_default_days: number
  retention: {
    enabled: boolean
    messages_days: number
    telemetry_days: number
    audit_days: number
    precise_position_days: number
  }
  evidence_export_delete: {
    export_enabled: boolean
    delete_enabled: boolean
    delete_scope: string[]
    delete_caveat?: string
  }
  inference_enabled: boolean
  inference_runtime_ready?: boolean
  inference_degraded?: boolean
  inference_caveat?: string
  export_redaction_enabled?: boolean
  inference_providers: Array<{
    name: string
    enabled: boolean
    endpoint_configured: boolean
    available_by_config: boolean
  }>
  assist_policies: Array<{
    task_class: string
    availability: 'available' | 'queued' | 'partial' | 'unavailable'
    execution_mode: 'inline' | 'queued' | 'scheduled' | 'disabled'
    provider: string
    hardware: 'cpu' | 'gpu'
    compression: string
    concurrency: string
    fallback_reason?: string
    latency_budget_ms: number
    context_token_budget: number
    non_canonical_truth: boolean
  }>
  operator_intelligence_posture?: OperatorIntelligencePosture
}

/** Honest capability envelope for this build (backend internal/runtime). */
export interface ProductEnvelope {
  product_name: string
  product_scope: string
  deployment_mode: string
  multi_site_fleet_supported: boolean
  notes: string
  transport_kinds: Array<{
    kind: string
    ingest_implemented: boolean
    send_implemented: boolean
    implementation_status: string
    notes?: string
  }>
  integration_enabled: boolean
  capability_posture?: FleetCapabilityPosture
}

/** Honest federation and cross-instance boundaries (backend internal/fleet). */
export interface FleetCapabilityPosture {
  federation_mode: string
  federation_read_only_evidence_ingest: string
  cross_instance_action_execution: string
  fleet_aggregation_supported: boolean
  notes: string
}

/** Instance/site/fleet boundary truth — does not imply global health. */
export interface FleetTruthSummary {
  instance_id: string
  site_id?: string
  fleet_id?: string
  fleet_label?: string
  gateway_label?: string
  truth_posture: string
  visibility_posture: string
  expected_fleet_reporters: number
  reporting_instances_known: number
  partial_visibility_reasons?: string[]
  ordering_posture: string
  capability_posture: FleetCapabilityPosture
  physics_network_note: string
}

/** Offline remote evidence bundle import audit row (local DB; not live federation). */
export interface ImportedRemoteEvidenceRecord {
  id: string
  imported_at: string
  local_instance_id: string
  validation: Record<string, unknown>
  bundle: Record<string, unknown>
  evidence: Record<string, unknown>
  origin_instance_id: string
  origin_site_id?: string
  evidence_class: string
  observation_origin_class: string
  rejected: boolean
}

/** Durable SQLite identity + optional live process fields (mel serve). */
export interface InstanceTruth {
  instance_id?: string
  process?: { pid: number; started_at: string }
  uptime_seconds?: number
  data_dir?: string
  database_path?: string
  config_path?: string
  bind_api?: string
}

export interface StatusResponse {
  configured_transport_modes: string[]
  messages: number
  nodes: NodeInfo[]
  transports: TransportHealth[]
  product?: ProductEnvelope
  instance?: InstanceTruth
  fleet_truth?: FleetTruthSummary
  /** When present, full GET /api/v1/status JSON envelope (panel, privacy, etc.). */
  _apiEnvelope?: Record<string, unknown>
}

export interface NodeInfo {
  node_num: number
  node_id: string
  long_name: string
  short_name: string
  last_seen: string
  last_gateway_id: string
  // Additional fields from mesh endpoint
  user?: {
    id: string
    long_name: string
    short_name: string
    macaddr: string
    hw_model: string
  }
  position?: {
    latitude_i: number
    longitude_i: number
    altitude: number
    time: string
  }
}

export interface TransportHealth {
  name: string
  source?: string
  type: string
  effective_state: string
  runtime_state: string
  health: HealthScore
  active_alerts: string[]
  status_scope: string
  detail: string
  guidance: string
  total_messages: number
  persisted_messages: number
  last_heartbeat_at: string
  consecutive_timeouts: number
  retry_status: string
  dead_letters: number
  observation_drops: number
  last_attempt_at: string
  last_ingest_at: string
  last_error: string
}

export interface HealthScore {
  score: number
  state: string
  primary_reason: string
  explanation: string[]
}

export interface Message {
  transport_name: string
  packet_id: string
  from_node: string
  to_node: string
  portnum: string
  payload_text: string
  rx_time: string
}

export interface DeadLetter {
  transport_name: string
  transport_type: string
  topic: string
  reason: string
  created_at: string
}

/** Incident row (list/detail); handoff fields optional on older DB rows */
export interface Incident {
  id: string
  category?: string
  severity?: string
  title?: string
  summary?: string
  resource_type?: string
  resource_id?: string
  state?: string
  actor_id?: string
  occurred_at?: string
  updated_at?: string
  resolved_at?: string
  metadata?: Record<string, unknown>
  owner_actor_id?: string
  handoff_summary?: string
  pending_actions?: string[]
  recent_actions?: string[]
  linked_evidence?: Record<string, unknown>[]
  risks?: string[]
  review_state?: string
  investigation_notes?: string
  resolution_summary?: string
  closeout_reason?: string
  lessons_learned?: string
  reopened_from_incident_id?: string
  reopened_at?: string
  /** Canonical FK-linked control actions (when backend enriches list/detail) */
  linked_control_actions?: ControlActionRecord[]
  intelligence?: IncidentIntelligence
  /** Server-computed control visibility for this response (capability-aware); prefer over pure frontend inference when present. */
  action_visibility?: IncidentActionVisibilityPosture
  /** Compact backend-owned replay posture for queue/workbench/detail/export surfaces. */
  replay_summary?: IncidentReplaySummary
  /** Server-computed deterministic triage hints (inspectable codes, not opaque ranking). */
  triage_signals?: IncidentTriageSignals
  /** Deterministic assist cues (same payload mirrored on decision_pack.assist_signals when pack is present). */
  assist_signals?: IncidentAssistSignals
  /** Canonical backend-assembled decision object (detail/workbench/export framing). */
  decision_pack?: IncidentDecisionPack
}

export interface IncidentDecisionPack {
  schema_version: string
  incident_id: string
  generated_at: string
  identity?: IncidentDecisionPackIdentity
  queue?: IncidentDecisionPackQueue
  guidance?: IncidentDecisionPackGuidance
  evidence_basis?: IncidentDecisionPackEvidenceBasis
  intelligence_summary?: IncidentDecisionPackIntelligenceSummary
  mitigation_durability?: IncidentMitigationDurabilityMemory
  family_history?: IncidentDecisionPackFamilyHistory
  transport_graph?: IncidentDecisionPackTransportGraph
  linked_actions?: IncidentDecisionPackLinkedActions
  readiness?: IncidentDecisionPackReadiness
  uncertainty?: IncidentDecisionPackUncertainty
  operator_adjudication?: IncidentDecisionPackAdjudication
  assist_signals?: IncidentAssistSignals
  analytics_hints?: IncidentDecisionPackAnalyticsHints
}

export interface IncidentDecisionPackIdentity {
  title?: string
  state?: string
  severity?: string
  category?: string
  resource_type?: string
  resource_id?: string
  occurred_at?: string
  updated_at?: string
  resolved_at?: string
  review_state?: string
  owner_actor_id?: string
  handoff_summary?: string
  summary?: string
}

export interface IncidentDecisionPackQueue {
  triage_signals?: IncidentTriageSignals
  why_surfaced_one_liner?: string
  ordering_note?: string
}

export interface IncidentDecisionPackGuidance {
  needs_attention?: boolean
  priority_tier?: number
  why_now?: string
  review_recommended?: boolean
  verify_before_action?: boolean
  evidence_posture?: 'strong' | 'moderate' | 'sparse' | 'degraded' | 'unknown'
  mitigation_fragility_watch?: boolean
  repeated_family_concern?: boolean
  action_posture?: 'available' | 'guarded' | 'unsupported' | 'verify_linkage'
  support_posture?: 'ready' | 'partial' | 'blocked' | 'unknown'
  topology_planning_posture?: 'useful_non_proving'
  escalation_posture?: 'replay_first' | 'follow_up' | 'bounded_review'
  replay_semantic?: string
  replay_history_pattern?: string
  replay_comparability?: string
  replay_attention_reason?: string
  replay_not_comparable_reasons?: string[]
  replay_summary?: string
  degraded?: boolean
  degraded_reasons?: string[]
}

export interface IncidentReplaySummary {
  schema_version?: string
  semantic?: 'active_changing' | 'cooling_off' | 'quiet_recently' | 'sparse' | 'no_history' | 'unavailable' | 'partial'
  history_pattern?: 'worsening' | 'recovering' | 'stable' | 'thin_history' | 'unavailable' | 'partial'
  comparability?: 'comparable' | 'not_comparable' | 'unavailable'
  activity_posture?: string
  window_from?: string
  window_to?: string
  anchor_time?: string
  last_event_at?: string
  recent_count?: number
  prior_count?: number
  delta_total?: number
  sparse_evidence?: boolean
  window_truncated?: boolean
  degraded?: boolean
  degraded_reasons?: string[]
  needs_attention?: boolean
  attention_reason?: string
  not_comparable_reasons?: string[]
  summary?: string
  uncertainty?: string
  recommendation_ref?: string
}

export interface IncidentDecisionPackEvidenceBasis {
  evidence_strength?: string
  evidence_items?: IncidentEvidenceItem[]
  item_cap_applied?: number
  degraded?: boolean
  degraded_reasons?: string[]
  sparsity_markers?: string[]
}

export interface IncidentDecisionPackIntelligenceSummary {
  signature_label?: string
  signature_match_count?: number
  summary_lines?: string[]
  investigate_next_ids?: string[]
  runbook_recommendation_ids?: string[]
  learning_loop_hints?: string[]
}

export interface IncidentDecisionPackFamilyHistory {
  signature_family?: IncidentSignatureFamilyResolvedHistory
  similar_incidents?: IncidentSimilarityRecord[]
  similar_scan_cap?: number
}

export interface IncidentDecisionPackTransportGraph {
  mesh_routing_companion?: MeshRoutingIntelCompanion
  wireless_context?: IncidentWirelessContext
}

export interface IncidentDecisionPackLinkedActions {
  action_visibility?: IncidentActionVisibilityPosture
  operator_suggested_actions?: OperatorSuggestedAction[]
  linked_control_action_ids?: string[]
}

export interface IncidentDecisionPackReadiness {
  export_policy_semantic?: string
  export_policy_summary?: string
  export_artifact_strength?: string
  export_blocker_codes?: string[]
  proofpack_path?: string
  escalation_bundle_path?: string
  handoff_posture_note?: string
  evidence_sufficiency_note?: string
}

export interface IncidentDecisionPackUncertainty {
  non_claims?: string[]
  interpretation_labels?: string[]
  degraded_sections?: string[]
  bounded_scan_disclosures?: string[]
}

export interface IncidentDecisionPackCueOutcome {
  cue_id: string
  outcome?: string
  note?: string
  reason_code?: string
}

export interface IncidentDecisionPackAdjudication {
  reviewed?: boolean
  reviewed_at?: string
  reviewed_by_actor_id?: string
  useful?: string
  operator_note?: string
  cue_outcomes?: IncidentDecisionPackCueOutcome[]
  updated_at?: string
}

export interface IncidentDecisionPackAnalyticsHints {
  signature_key?: string
  fingerprint_canonical_hash?: string
  triage_tier?: number
  mitigation_durability_posture?: string
  evidence_strength?: string
  intel_degraded?: boolean
}

/** Mirrors GET incident JSON; emitted when backend computes operator control visibility. */
export interface IncidentActionVisibilityPosture {
  action_visibility_kind: IncidentActionVisibilityKindAPI
  action_visibility_reason?: string
  action_visibility_summary: string
  action_context_should_open_control_queue: boolean
  action_context_has_material_prior_attempts: boolean
  action_context_has_pending_related_work: boolean
  action_context_is_partial: boolean
  linked_control_row_count?: number
  pending_action_ref_count?: number
  recent_action_ref_count?: number
}

export type IncidentActionVisibilityKindAPI =
  | 'visibility_limited'
  | 'linked_observed'
  | 'references_only'
  | 'action_context_degraded'
  | 'no_linked_historical_signals'
  | 'no_linked_observed'

export interface IncidentIntelligence {
  signature_key?: string
  signature_label?: string
  signature_match_count?: number
  fingerprint?: IncidentFingerprint
  evidence_strength: 'sparse' | 'moderate' | 'strong'
  evidence_items?: IncidentEvidenceItem[]
  implicated_domains?: IncidentDomainHint[]
  wireless_context?: IncidentWirelessContext
  investigate_next?: IncidentGuidanceItem[]
  similar_incidents?: IncidentSimilarityRecord[]
  historically_used_actions?: IncidentActionPattern[]
  action_outcome_memory?: IncidentActionOutcomeMemory[]
  action_outcome_snapshots?: IncidentActionOutcomeSnapshot[]
  action_outcome_trace?: IncidentActionOutcomeTrace
  degraded?: boolean
  degraded_reasons?: string[]
  sparsity_markers?: string[]
  runbook_recommendations?: IncidentRunbookRecommendation[]
  runbook_assets?: IncidentRunbookAsset[]
  policy_governance_hints?: IncidentPolicyGovernanceHint[]
  governance_memory?: IncidentGovernanceMemory[]
  drift_fingerprints?: IncidentDriftFingerprint[]
  correlation_groups?: IncidentCorrelationGroup[]
  fault_domains?: IncidentFaultDomain[]
  replay_hints?: IncidentReplayHints
  learning_loop_hints?: string[]
  generated_at?: string
  mesh_routing_companion?: MeshRoutingIntelCompanion
  operator_suggested_actions?: OperatorSuggestedAction[]
  signature_family_resolved_history?: IncidentSignatureFamilyResolvedHistory
  mitigation_durability_memory?: IncidentMitigationDurabilityMemory
}

export interface IncidentMitigationDurabilityMemory {
  schema_version?: string
  posture: string
  reason_codes?: string[]
  summary: string
  scope?: { primary: string; detail?: string[] }
  basis?: { inputs?: string[]; counts?: Record<string, number>; scan_posture?: string }
  non_claims?: string[]
  evidence_refs?: string[]
  uncertainty: string
}

export interface IncidentAssistSignals {
  schema_version: string
  signals?: IncidentAssistSignal[]
  uncertainty?: string
  evidence_basis?: string
}

export interface IncidentAssistSignal {
  code: string
  severity?: 'info' | 'watch' | 'review'
  title: string
  rationale: string
  evidence_refs?: string[]
  uncertainty?: string
  operator_state?: {
    latest_outcome?: string
    latest_at?: string
    actor_id?: string
  }
}

export interface IncidentTriageSignals {
  tier: number
  codes?: string[]
  rationale_lines?: string[]
  evidence_refs?: string[]
  uncertainty_notes?: string[]
  queue_ordering_contract?: string
  queue_ordering_contract_version?: string
  queue_sort_primary?: number
  queue_sort_secondary?: string
  queue_sort_secondary_numeric?: number
  queue_sort_secondary_validity?: string
  queue_sort_tie_break?: string
  queue_sort_tie_break_numeric?: number
  queue_sort_tuple?: number[]
  queue_sort_key_lex?: string
  ordering_rationale?: string
  ordering_evidence_refs?: string[]
  ordering_uncertainty?: string
  /** canonical_v2 = full tuple+lex from valid updated_at; degraded_partial_recency = missing/invalid timestamp fallbacks */
  queue_ordering_posture?: string
  queue_ordering_degraded_reasons?: string[]
}

export interface IncidentSignatureFamilyResolvedHistory {
  family_match_total: number
  resolved_peer_count: number
  reopened_peer_count: number
  basis: string
  uncertainty: string
  peer_sample_incident_id?: string
  peer_history_scan_truncated?: boolean
  peer_scan_window?: number
}

export interface MeshRoutingIntelCompanion {
  applicable: boolean
  reason?: string
  topology_enabled?: boolean
  transport_connected?: boolean
  assessment_computed_at?: string
  graph_hash?: string
  evidence_model?: string
  message_window_description?: string
  routing_summary_lines?: string[]
  suspected_relay_hotspot?: boolean
  weak_onward_propagation_suspected?: boolean
  hop_budget_stress_suspected?: boolean
  suggested_topology_search?: string
}

export interface OperatorSuggestedAction {
  id: string
  title: string
  rationale: string
  evidence_refs?: string[]
  uncertainty?: string
  href?: string
  kind: string
  disable_hint?: string
}

export interface IncidentFingerprint {
  schema_version: string
  profile_version: string
  legacy_signature_key?: string
  canonical_hash: string
  components?: Record<string, string[]>
  sparsity_markers?: string[]
  computed_at?: string
}

export interface IncidentRunbookAsset {
  id: string
  status: string
  source_kind: string
  title: string
  body?: string
  evidence_refs?: string[]
  source_incident_ids?: string[]
  legacy_signature_key?: string
  fingerprint_canonical_hash?: string
  promotion_basis?: string
  created_at?: string
  updated_at?: string
}

export interface IncidentFaultDomain {
  domain_id: string
  domain_key: string
  basis: string
  uncertainty: string
  rationale?: string[]
  evidence_bundle?: Record<string, string>
  member_kinds?: string[]
}

export interface IncidentGovernanceMemory {
  action_type: string
  summary: string
  linked_action_count: number
  approved_or_passed_count: number
  rejected_count: number
  high_blast_count: number
  separate_approver_count: number
  evidence_refs?: string[]
}

export interface IncidentRunbookRecommendation {
  id: string
  title: string
  action_type?: string
  rationale: string
  evidence_refs?: string[]
  strength:
    | 'historically_proven'
    | 'historically_promising'
    | 'plausible'
    | 'weakly_supported'
    | 'unsupported'
    | 'proven_historically'
  strength_explanation?: string[]
  rank_score?: number
  requires_approval: boolean
  blast_radius_class?: string
  reversibility: 'high' | 'medium' | 'low' | 'unknown'
  prior_outcome_framing?: string
  prior_sample_size?: number
  historical_outcome_note?: string
  suppressed?: boolean
  suppressed_reason?: string
  is_command: boolean
}

export interface IncidentPolicyGovernanceHint {
  summary: string
  evidence_refs?: string[]
  posture: string
}

export interface IncidentDriftFingerprint {
  kind: string
  transport_name?: string
  reason?: string
  statement: string
  current_bucket_hits: number
  prior_bucket_hits: number
  supports_only?: string
}

export interface IncidentCorrelationGroup {
  group_id: string
  correlation_key: string
  basis: string
  created_at?: string
  updated_at?: string
  rationale?: string[]
  evidence_refs?: string[]
  uncertainty_note?: string
  member_count?: number
  other_incident_ids?: string[]
}

export interface IncidentReplayHints {
  statement: string
  evidence_at_time_refs?: string[]
  counterfactual_note?: string
  ranking_model_note?: string
}

export interface IncidentActionOutcomeTrace {
  expected_snapshot_writes: number
  snapshot_write_failures: number
  snapshot_write_failure_ids?: string[]
  snapshot_retrieval_status: 'available' | 'unavailable' | 'error'
  snapshot_retrieval_reason?: string
  snapshot_retrieval_error?: string
  persisted_snapshot_count: number
  completeness: 'complete' | 'partial' | 'unavailable'
}

export interface IncidentWirelessContext {
  classification:
    | 'lora_mesh_pressure'
    | 'bluetooth_onboarding_issue'
    | 'wifi_backhaul_instability'
    | 'mixed_path_degradation'
    | 'sparse_evidence_incident'
    | 'recurring_unknown_pattern'
    | 'unsupported_wireless_domain_observed'
  primary_domain: 'lora' | 'bluetooth' | 'wifi' | 'mixed' | 'unknown'
  observed_domains?: string[]
  evidence_posture: 'live' | 'historical' | 'partial' | 'sparse' | 'imported' | 'unsupported'
  confidence_posture: 'evidence_backed' | 'mixed' | 'sparse' | 'inconclusive'
  summary: string
  reasons?: IncidentWirelessReason[]
  evidence_gaps?: string[]
  inspect_next?: string[]
  unsupported?: IncidentWirelessUnsupported[]
}

export interface IncidentWirelessReason {
  code: string
  statement: string
  evidence_refs?: string[]
}

export interface IncidentWirelessUnsupported {
  domain: string
  scope: string
  note: string
}

export interface IncidentEvidenceItem {
  kind: string
  reference_id?: string
  summary: string
  observed_at?: string
  supports_only?: string
}

export interface IncidentDomainHint {
  domain: string
  evidence_refs?: string[]
  note?: string
}

export interface IncidentGuidanceItem {
  id: string
  title: string
  rationale: string
  evidence_refs?: string[]
  confidence: 'low' | 'medium'
}

export interface IncidentSimilarityRecord {
  incident_id: string
  title?: string
  state?: string
  occurred_at?: string
  similarity_reason?: string[]
  match_category?: string
  weighted_score?: number
  matched_dimensions?: string[]
  unmatched_dimensions?: string[]
  weak_sparse_dimensions?: string[]
  insufficient_evidence?: boolean
  match_explanation?: string[]
}

export interface IncidentActionPattern {
  action_type: string
  count: number
}

export interface IncidentActionOutcomeMemory {
  action_type: string
  action_label?: string
  occurrence_count: number
  sample_size: number
  outcome_framing:
    | 'improvement_observed'
    | 'deterioration_observed'
    | 'mixed_historical_evidence'
    | 'insufficient_evidence'
    | 'no_clear_post_action_signal'
  evidence_strength: 'sparse' | 'moderate' | 'strong'
  observed_post_action_status: string
  improvement_observed_count: number
  deterioration_observed_count: number
  inconclusive_count: number
  caveats?: string[]
  inspect_before_reuse?: string[]
  evidence_refs?: string[]
  snapshot_refs?: string[]
}

export interface IncidentActionOutcomeEvidenceSummary {
  transport_name?: string
  dead_letters_count: number
  transport_alerts_count: number
  incident_state?: string
  action_result?: string
  action_lifecycle?: string
}

export interface IncidentActionOutcomeSnapshot {
  snapshot_id: string
  signature_key: string
  incident_id: string
  action_id: string
  action_type: string
  action_label?: string
  derived_classification: string
  evidence_sufficiency: 'sufficient' | 'partial' | 'insufficient'
  window_start: string
  window_end: string
  pre_action_evidence: IncidentActionOutcomeEvidenceSummary
  post_action_evidence: IncidentActionOutcomeEvidenceSummary
  observed_signal_count: number
  caveats?: string[]
  inspect_before_reuse?: string[]
  evidence_refs?: string[]
  association_only: boolean
  derivation_version?: string
  schema_version?: string
  derived_at: string
}

/** Control-plane action row (matches backend ActionRecord) */
export interface ControlActionRecord {
  id: string
  transport_name?: string
  action_type: string
  lifecycle_state?: string
  result?: string
  reason?: string
  outcome_detail?: string
  created_at?: string
  executed_at?: string
  completed_at?: string
  expires_at?: string
  execution_mode?: string
  proposed_by?: string
  submitted_by?: string
  approved_by?: string
  approved_at?: string
  rejected_by?: string
  rejected_at?: string
  approval_expires_at?: string
  blast_radius_class?: string
  requires_separate_approver?: boolean
  incident_id?: string
  execution_started_at?: string
  sod_bypass?: boolean
  sod_bypass_actor?: string
  sod_bypass_reason?: string
  target_segment?: string
  target_node?: string
  approval_mode?: string
  required_approvals?: number
  collected_approvals?: number
  approval_basis?: string[]
  approval_policy_source?: string
  high_blast_radius?: boolean
  approval_escalated_due_to_blast_radius?: boolean
  execution_source?: string
}

/** Labels derived server-side for operator clarity (queue / approval / execution). */
export interface ControlActionOperatorView {
  target_summary?: string
  approval_status?: string
  execution_status?: string
  queue_status?: string
  second_operator_note?: string
  sod_blocks_self?: boolean
  break_glass_in_history?: boolean
  linked_incident_id?: string
}

/**
 * Row shape for mesh/node control history and pending approvals
 * (GET /api/v1/control/history, enriched pending_approvals on operational-state).
 */
export interface MeshNodeControlAction {
  id: string
  command?: string
  action_type?: string
  target_node?: string
  target_segment?: string
  target_node_id?: string
  target_transport?: string
  transport_name?: string
  result: string
  denial_reason?: string
  created_at?: string
  executed_at?: string
  outcome_detail?: string
  advisory_only?: boolean
  lifecycle_state?: string
  execution_mode?: string
  proposed_by?: string
  approved_by?: string
  evidence_bundle_id?: string
  approval_expires_at?: string
  blast_radius_class?: string
  requires_separate_approver?: boolean
  incident_id?: string
  execution_started_at?: string
  sod_bypass?: boolean
  sod_bypass_actor?: string
  sod_bypass_reason?: string
  approval_mode?: string
  required_approvals?: number
  collected_approvals?: number
  approval_basis?: string[]
  approval_policy_source?: string
  high_blast_radius?: boolean
  approval_escalated_due_to_blast_radius?: boolean
  execution_source?: string
  operator_view?: ControlActionOperatorView
  details?: Record<string, unknown>
}

/** Single row from control reality matrix (GET /api/v1/control/status). */
export interface ControlRealityMatrixItem {
  action_type: string
  actuator_exists: boolean
  reversible: boolean
  blast_radius_class: string
  safe_for_guarded_auto: boolean
  advisory_only: boolean
  notes: string
}

/** GET /api/v1/control/status — automation mode and executor snapshot. */
export interface ControlStatusResponse {
  mode?: string
  reality_matrix?: ControlRealityMatrixItem[]
  queue_depth?: number
  queue_capacity?: number
  active_actions?: number
  policy_summary?: string
  emergency_disable?: boolean
}

/** GET /api/v1/control/history */
export interface ControlHistoryResponse {
  actions?: MeshNodeControlAction[]
}

/** GET /api/v1/control/operational-state — fields used by the operator console. */
export interface ControlQueueMetrics {
  queued_lifecycle_pending_count: number
  awaiting_approval_count: number
  approved_waiting_executor_count: number
  oldest_queued_pending_created_at: string
  oldest_approved_waiting_executor_created_at: string
}

export interface ControlExecutorPresence {
  executor_activity: 'active' | 'inactive' | 'unknown' | string
  executor_last_heartbeat_at: string
  executor_last_reported_kind: string
  executor_heartbeat_basis: string
  executor_presence_note: string
  backlog_requires_active_executor: boolean
}

export interface ActiveFreezeWindow {
  id: string
  reason: string
  actor: string
  scope: string
  created_at: string
}

export interface ActiveMaintenanceWindow {
  id: string
  reason: string
  actor: string
  starts_at: string
  ends_at: string
}

export interface ControlOperationalStateResponse {
  automation_mode?: 'normal' | 'frozen' | 'maintenance' | string
  freeze_count?: number
  approval_backlog?: number
  snapshot_at?: string
  queue_metrics?: ControlQueueMetrics
  executor?: ControlExecutorPresence
  active_freezes?: ActiveFreezeWindow[]
  active_maintenance?: ActiveMaintenanceWindow[]
  pending_approvals?: MeshNodeControlAction[]
}

export interface TrustUIHints {
  approve_control: boolean
  reject_control: boolean
  execute_control: boolean
  read_actions: boolean
  incident_handoff_write: boolean
  incident_mutate: boolean
}

export interface PanelResponse {
  generated_at?: string
  operator_state?: string
  summary?: string
  capabilities?: string[]
  trust_ui?: TrustUIHints
}

export interface Finding {
  component: string;
  severity: string;
  message: string;
  guidance: string;
}

export interface PrivacyFinding {
  severity: 'critical' | 'high' | 'medium' | 'low'
  message: string
  remediation: string
}

export interface Recommendation {
  category: string
  priority: string
  message: string
  actionable: boolean
  action?: string
}

export interface AuditLog {
  category: string
  level: string
  message: string
  created_at: string
}

export interface PanelData {
  operator_state: string
  summary: string
  short_commands: string[]
  device_menu: string
}

export interface MeshState {
  nodes: NodeInfo[]
  messages: Message[]
  connected: boolean
}

// Self-Observability Types (Phase 10)

export interface InternalComponentHealth {
  name: string
  health: 'healthy' | 'degraded' | 'failing' | 'unknown'
  last_success: string
  last_failure: string
  error_count: number
  success_count: number
  error_rate: number
}

export interface InternalHealthResponse {
  overall_health: 'healthy' | 'degraded' | 'failing' | 'unknown'
  components: InternalComponentHealth[]
}

export interface FreshnessMarker {
  component: string
  last_update: string
  age_seconds: number
  status: string
  expected_interval_seconds: number
  stale_threshold_seconds: number
}

export interface FreshnessResponse {
  markers: FreshnessMarker[]
  stale_components: string[]
}

export interface SLOStatus {
  name: string
  description: string
  current_value: number
  target: number
  status: 'healthy' | 'at_risk' | 'breached' | 'unknown'
  budget_used: number
  unit: string
  window: string
  window_start: string
  window_end: string
  evaluated_at: string
}

export interface SLOResponse {
  slos: SLOStatus[]
}

export interface InternalMetricsResponse {
  timestamp: string
  pipeline_latency: Record<string, number>
  worker_heartbeats: Record<string, string>
  queue_depths: Record<string, number>
  error_rates: Record<string, number>
  resource_usage: {
    memory_used_bytes: number
    goroutines: number
    num_gc: number
  }
  operation_counts: Record<string, number>
}

// Utility types
export type HealthState = 'healthy' | 'degraded' | 'unhealthy' | 'unknown'

export function getHealthState(health: HealthScore | undefined): HealthState {
  if (!health) return 'unknown'
  const state = health.state?.toLowerCase()
  if (state === 'healthy' || state === 'ok') return 'healthy'
  if (state === 'degraded' || state === 'warning') return 'degraded'
  if (state === 'unhealthy' || state === 'critical' || state === 'error') return 'unhealthy'
  return 'unknown'
}

export function formatTimestamp(timestamp: string | undefined): string {
  if (!timestamp) return '—'
  try {
    const date = new Date(timestamp)
    return date.toLocaleString()
  } catch {
    return timestamp
  }
}

export function formatRelativeTime(timestamp: string | undefined): string {
  if (!timestamp) return 'never'
  try {
    const date = new Date(timestamp)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    const diffHours = Math.floor(diffMins / 60)
    const diffDays = Math.floor(diffHours / 24)

    if (diffMins < 1) return 'just now'
    if (diffMins < 60) return `${diffMins}m ago`
    if (diffHours < 24) return `${diffHours}h ago`
    if (diffDays < 7) return `${diffDays}d ago`
    return date.toLocaleDateString()
  } catch {
    return timestamp
  }
}
