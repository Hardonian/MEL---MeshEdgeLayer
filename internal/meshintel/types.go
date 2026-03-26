// Package meshintel computes explainable mesh bootstrap, topology placement,
// routing-pressure diagnostics, protocol-fit advisory, and ranked next-step
// recommendations from observed MEL evidence only (nodes, links, messages).
package meshintel

// ConfidenceLevel is a coarse uncertainty band for derived assessments.
type ConfidenceLevel string

const (
	ConfidenceHigh   ConfidenceLevel = "high"
	ConfidenceMedium ConfidenceLevel = "medium"
	ConfidenceLow    ConfidenceLevel = "low"
)

// LocalMeshViabilityClassification normalizes bootstrap / cluster viability.
type LocalMeshViabilityClassification string

const (
	ViabilityIsolated                LocalMeshViabilityClassification = "isolated"
	ViabilityWeakBootstrap           LocalMeshViabilityClassification = "weak_bootstrap"
	ViabilityEmergingCluster         LocalMeshViabilityClassification = "emerging_cluster"
	ViabilityViableLocalMesh         LocalMeshViabilityClassification = "viable_local_mesh"
	ViabilityInfrastructureDependent LocalMeshViabilityClassification = "infrastructure_dependent"
	ViabilityUnstableIntermittent    LocalMeshViabilityClassification = "unstable_intermittent"
)

// ClusterShapeClassification describes inferred graph shape from packet-derived edges.
type ClusterShapeClassification string

const (
	ShapeIsolatedSingle      ClusterShapeClassification = "isolated_single"
	ShapeSparseCluster       ClusterShapeClassification = "sparse_cluster"
	ShapeCorridor            ClusterShapeClassification = "corridor"
	ShapeDenseLocal          ClusterShapeClassification = "dense_local"
	ShapeHubAndSpoke         ClusterShapeClassification = "hub_and_spoke"
	ShapeFragileBridge       ClusterShapeClassification = "fragile_bridge"
	ShapeInfrastructureAnchored ClusterShapeClassification = "infrastructure_anchored"
	ShapePartitioned         ClusterShapeClassification = "partitioned"
)

// ProtocolFitClass classifies fit of managed-flood style use (Meshtastic-like) to observed behavior.
type ProtocolFitClass string

const (
	ProtocolFitGood                  ProtocolFitClass = "good_fit_managed_flood"
	ProtocolFitAcceptableCaveats     ProtocolFitClass = "acceptable_fit_with_caveats"
	ProtocolFitRolePlacementFirst    ProtocolFitClass = "role_or_placement_correction_recommended"
	ProtocolFitStressedDensity       ProtocolFitClass = "likely_stressed_by_density_or_forwarding"
	ProtocolFitStructuredInfra       ProtocolFitClass = "candidate_structured_infrastructure"
	ProtocolFitSegmentBridge         ProtocolFitClass = "candidate_segmented_or_bridged_architecture"
	ProtocolFitAlternateEvaluation   ProtocolFitClass = "candidate_alternate_protocol_evaluation"
	ProtocolFitInsufficientEvidence  ProtocolFitClass = "insufficient_evidence"
)

// ArchitectureRecommendationClass is a coarse architecture posture hint (never prescriptive protocol switching).
type ArchitectureRecommendationClass string

const (
	ArchObserveOnly              ArchitectureRecommendationClass = "observe_and_gather_history"
	ArchPlacementRolesFirst      ArchitectureRecommendationClass = "placement_and_roles_first"
	ArchAddStationaryInfra       ArchitectureRecommendationClass = "add_stationary_infrastructure"
	ArchReduceForwardingPressure ArchitectureRecommendationClass = "reduce_forwarding_pressure"
	ArchSegmentOrBridge          ArchitectureRecommendationClass = "consider_segmentation_or_bridging"
	ArchFixTransport             ArchitectureRecommendationClass = "stabilize_transport_before_topology_changes"
)

// RecommendationClass is a bounded operator-safe next action family.
type RecommendationClass string

const (
	RecAddElevatedStationary       RecommendationClass = "add_elevated_stationary_node"
	RecImprovePlacement          RecommendationClass = "improve_current_node_placement"
	RecReduceRouterRoleUsage     RecommendationClass = "reduce_router_role_usage"
	RecKeepObserve               RecommendationClass = "keep_current_topology_and_observe"
	RecAddAlwaysOnBase           RecommendationClass = "add_always_on_base_node"
	RecAddMobileCompanion        RecommendationClass = "add_mobile_companion_node"
	RecInvestigatePowerDutyCycle RecommendationClass = "investigate_power_duty_cycle_mismatch"
	RecInvestigateTransport      RecommendationClass = "investigate_transport_health_before_topology_changes"
	RecSegmentOrBridge           RecommendationClass = "segment_or_bridge_network"
	RecGatherMoreHistory         RecommendationClass = "gather_more_observation_history_before_major_change"
)

// DeploymentClassHint suggests deployment category without hardware SKUs.
type DeploymentClassHint string

const (
	DeployHandheldCompanion DeploymentClassHint = "handheld_companion"
	DeployBaseWindowsill    DeploymentClassHint = "base_windowsill"
	DeployAtticRooftop      DeploymentClassHint = "attic_rooftop"
	DeployVehicle           DeploymentClassHint = "vehicle"
	DeploySolarRelayOutdoor DeploymentClassHint = "solar_relay_fixed_outdoor"
	DeploySensorEndpoint    DeploymentClassHint = "sensor_endpoint"
	DeployEventTemporary    DeploymentClassHint = "event_temporary_node"
)

// ScoredMetric is a single bounded score with explainability.
type ScoredMetric struct {
	Name         string          `json:"name"`
	Score        float64         `json:"score"`
	Scale        string          `json:"scale"` // "0_1" or "0_100"
	Basis        string          `json:"basis"`
	Evidence     []string        `json:"evidence,omitempty"`
	Confidence   ConfidenceLevel `json:"confidence"`
	Uncertainty  string          `json:"uncertainty,omitempty"`
	IsSuspected  bool            `json:"is_suspected,omitempty"`
}

// BootstrapExplanation answers what evidence was used and what to do next.
type BootstrapExplanation struct {
	EvidenceUsed        []string `json:"evidence_used"`
	WeakensViability    []string `json:"weakens_viability"`
	StrengthensViability []string `json:"strengthens_viability"`
	Missing             []string `json:"missing"`
	TopNextAction       string   `json:"top_next_action"`
}

// BootstrapAssessment is mesh-global bootstrap / lone-wolf intelligence.
type BootstrapAssessment struct {
	LoneWolfScore            float64                          `json:"lone_wolf_score"`
	BootstrapReadinessScore  float64                          `json:"bootstrap_readiness_score"`
	Viability                LocalMeshViabilityClassification `json:"viability"`
	Explanation              BootstrapExplanation             `json:"explanation"`
	Confidence               ConfidenceLevel                  `json:"confidence"`
	PeerCountObserved        int                              `json:"peer_count_observed"`
	UniquePeersOverWindow    int                              `json:"unique_peers_over_window"`
	LargestComponentSize     int                              `json:"largest_component_size"`
	ComponentCount           int                              `json:"component_count"`
	ObservationWindowHint    string                           `json:"observation_window_hint"`
}

// TopologyExplanation summarizes topology-level derived conclusions.
type TopologyExplanation struct {
	TopContributingFactors []string `json:"top_contributing_factors"`
	RiskyStructure         []string `json:"risky_structure"`
	WeakPoints             []string `json:"weak_points"`
	LeverageOpportunities  []string `json:"leverage_opportunities"`
	EvidenceStrength       string   `json:"evidence_strength"`
	Limits                 []string `json:"limits"`
}

// MeshTopologyMetrics are aggregate placement / shape metrics (0–1 unless noted).
type MeshTopologyMetrics struct {
	FragmentationScore            float64 `json:"fragmentation_score"`
	DependencyConcentrationScore float64 `json:"dependency_concentration_score"`
	InfrastructureLeverageScore   float64 `json:"infrastructure_leverage_score"`
	ClusterShape                  ClusterShapeClassification `json:"cluster_shape"`
	Explanation                   TopologyExplanation      `json:"explanation"`
}

// NodeTopologyIntel is per-node contribution / placement hints from the observed graph.
type NodeTopologyIntel struct {
	NodeNum                   int64   `json:"node_num"`
	CoverageContributionScore float64 `json:"coverage_contribution_score"`
	RelayValueScore           float64 `json:"relay_value_score"`
	PlacementQualityScore     float64 `json:"placement_quality_score"`
	InDegree                  int     `json:"in_degree"`
	OutDegree                 int     `json:"out_degree"`
	UndirectedDegree          int     `json:"undirected_degree"`
	IsBridgeCritical          bool    `json:"is_bridge_critical"`
	Role                      string  `json:"role,omitempty"`
	Notes                     []string `json:"notes,omitempty"`
}

// RoutingPressureBundle holds suspected routing / flood pressure diagnostics.
type RoutingPressureBundle struct {
	CollisionRiskScore             ScoredMetric `json:"collision_risk"`
	DuplicateForwardPressureScore  ScoredMetric `json:"duplicate_forward_pressure"`
	HopBudgetStressScore           ScoredMetric `json:"hop_budget_stress"`
	BroadcastDomainPressureScore   ScoredMetric `json:"broadcast_domain_pressure"`
	OverRouterizationRiskScore     ScoredMetric `json:"over_routerization_risk"`
	WeakOnwardPropagationScore     ScoredMetric `json:"weak_onward_propagation"`
	SuspectedPacketSinkScore       ScoredMetric `json:"suspected_packet_sink"`
	RouteConcentrationRiskScore    ScoredMetric `json:"route_concentration_risk"`
	SummaryLines                   []string     `json:"summary_lines"`
}

// ProtocolFitExplanation documents protocol-neutral advisory reasoning.
type ProtocolFitExplanation struct {
	ObservedFacts        []string `json:"observed_facts"`
	CheaperStepsFirst    []string `json:"cheaper_steps_first"`
	PlacementBeforeProto string   `json:"placement_before_protocol_note"`
	EvidenceAdequacy     string   `json:"evidence_adequacy"`
	Limits               []string `json:"limits"`
}

// ProtocolFitAssessment is the protocol-fit / architecture advisory layer.
type ProtocolFitAssessment struct {
	FitClass            ProtocolFitClass                `json:"fit_class"`
	ArchitectureClass   ArchitectureRecommendationClass `json:"architecture_class"`
	Explanation         ProtocolFitExplanation          `json:"explanation"`
	Confidence          ConfidenceLevel                 `json:"confidence"`
	PrimaryLimitingFactor string                        `json:"primary_limiting_factor,omitempty"`
}

// MeshRecommendation is a ranked, explainable next step.
type MeshRecommendation struct {
	Rank             int                 `json:"rank"`
	Class            RecommendationClass `json:"class"`
	Title            string              `json:"title"`
	Severity         string              `json:"severity"` // info, low, medium, high
	Urgency          string              `json:"urgency"`  // soon, when_convenient, deferred
	EvidenceSummary  []string            `json:"evidence_summary"`
	ExpectedBenefit  string              `json:"expected_benefit"`
	DownsideRisk     string              `json:"downside_risk"`
	Confidence       float64             `json:"confidence"`
	Prerequisites    []string            `json:"prerequisites,omitempty"`
	WhyNotOthers     string              `json:"why_not_others,omitempty"`
	DeploymentHints  []DeploymentClassHint `json:"deployment_hints,omitempty"`
}

// MessageSignals is rollup statistics from recent messages table rows.
type MessageSignals struct {
	WindowDescription     string  `json:"window_description"`
	TotalMessages         int64   `json:"total_messages"`
	MessagesWithRelay     int64   `json:"messages_with_relay"`
	MessagesWithHop       int64   `json:"messages_with_hop"`
	AvgHopLimit           float64 `json:"avg_hop_limit"`
	MaxHopLimit           int     `json:"max_hop_limit"`
	DistinctFromNodes     int     `json:"distinct_from_nodes"`
	DuplicateRelayHotspot float64 `json:"duplicate_relay_hotspot_ratio"`
	TransportConnected    bool    `json:"transport_connected"`
}

// Assessment is the full mesh intelligence payload for API / persistence / UI.
type Assessment struct {
	AssessmentID       string               `json:"assessment_id"`
	ComputedAt         string               `json:"computed_at"`
	GraphHash          string               `json:"graph_hash"`
	TopologyEnabled    bool                 `json:"topology_enabled"`
	Bootstrap          BootstrapAssessment  `json:"bootstrap"`
	Topology           MeshTopologyMetrics  `json:"topology"`
	NodeIntel          []NodeTopologyIntel  `json:"node_intel"`
	RoutingPressure    RoutingPressureBundle `json:"routing_pressure"`
	ProtocolFit        ProtocolFitAssessment `json:"protocol_fit"`
	Recommendations    []MeshRecommendation `json:"recommendations"`
	EvidenceModel      string               `json:"evidence_model"`
}
