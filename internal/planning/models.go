// Package planning implements truthful deployment planning, bounded what-if reasoning,
// resilience analysis, and plan comparison. Estimates are topology- and evidence-based;
// they are not RF propagation simulation.
package planning

import "github.com/mel-project/mel/internal/meshintel"

// PlanningEvidenceModel matches meshintel evidence wording so API consumers see one consistent model.
const PlanningEvidenceModel = "Derived from topology_links and nodes (packet relay/to_node fields), plus optional recent messages rollup. Not RF coverage proof. MEL does not modify Meshtastic routing."

// ProvenanceKind marks how a fact or input was obtained.
type ProvenanceKind string

const (
	ProvenanceObserved         ProvenanceKind = "observed"
	ProvenanceOperatorSupplied ProvenanceKind = "operator_supplied"
	ProvenanceInferred         ProvenanceKind = "inferred"
	ProvenanceUnknown          ProvenanceKind = "unknown"
)

// PlanningInput is an optional operator hint with explicit provenance.
type PlanningInput struct {
	Key         string         `json:"key"`
	Value       string         `json:"value,omitempty"`
	Provenance  ProvenanceKind `json:"provenance"`
	Description string         `json:"description,omitempty"`
}

// AssumptionSet groups labeled assumptions for a plan or scenario.
type AssumptionSet struct {
	Items []PlanningInput `json:"items"`
	Notes []string        `json:"notes,omitempty"`
}

// QualitativeOutcome is a bounded benefit / risk label (not a fake percentage).
type QualitativeOutcome string

const (
	OutcomeLikelyLowBenefit      QualitativeOutcome = "likely_low_benefit"
	OutcomePlausibleModerate     QualitativeOutcome = "plausible_moderate_benefit"
	OutcomeLikelyHighLeverage    QualitativeOutcome = "likely_high_leverage"
	OutcomeUncertainButPromising QualitativeOutcome = "uncertain_but_promising"
	OutcomeHighRiskLowConfidence QualitativeOutcome = "high_risk_low_confidence"
	OutcomeLikelyHarmful         QualitativeOutcome = "likely_harmful"
)

// ExpectedOutcome summarizes expected effects in qualitative terms.
type ExpectedOutcome struct {
	BenefitBand      QualitativeOutcome `json:"benefit_band"`
	Rationale        []string           `json:"rationale"`
	TopologyOnlyNote string             `json:"topology_only_note,omitempty"`
}

// RiskFactor is one identified risk with basis.
type RiskFactor struct {
	Code        string   `json:"code"`
	Severity    string   `json:"severity"` // low, medium, high
	Description string   `json:"description"`
	Evidence    []string `json:"evidence,omitempty"`
}

// ConfidenceAssessment explains uncertainty for an assessment block.
type ConfidenceAssessment struct {
	Level              meshintel.ConfidenceLevel `json:"level"`
	Score              float64                   `json:"score"` // 0–1 coarse
	MissingInputs      []string                  `json:"missing_inputs,omitempty"`
	WouldValidateWith  []string                  `json:"would_validate_with,omitempty"`
	WouldFalsifyWith   []string                  `json:"would_falsify_with,omitempty"`
	TopologyOnlyLimits []string                  `json:"topology_only_limits,omitempty"`
}

// PlanVerdict is a coarse overall judgment on a plan or scenario.
type PlanVerdict string

const (
	VerdictProceedWithCaution PlanVerdict = "proceed_with_caution"
	VerdictDeferObserve       PlanVerdict = "defer_observe"
	VerdictLikelyWorthwhile   PlanVerdict = "likely_worthwhile"
	VerdictInsufficientData   PlanVerdict = "insufficient_data"
	VerdictLikelyHarmful      PlanVerdict = "likely_harmful"
)

// DeploymentStepKind classifies a single proposed change.
type DeploymentStepKind string

const (
	StepAddNode              DeploymentStepKind = "add_node"
	StepRemoveNode           DeploymentStepKind = "remove_node"
	StepMoveNode             DeploymentStepKind = "move_node"
	StepElevateNode          DeploymentStepKind = "elevate_node"
	StepChangeRole           DeploymentStepKind = "change_role"
	StepImproveUptime        DeploymentStepKind = "improve_uptime"
	StepAddInfrastructure    DeploymentStepKind = "add_infrastructure"
	StepReduceInfraIntensity DeploymentStepKind = "reduce_infrastructure_intensity"
	StepBridgeClusters       DeploymentStepKind = "bridge_clusters"
	StepObserveOnly          DeploymentStepKind = "observe_only"
)

// DeploymentClassHint mirrors meshintel hints for plan steps.
type DeploymentClassHint string

const (
	ClassHandheld      DeploymentClassHint = "handheld_companion"
	ClassBase          DeploymentClassHint = "base_always_on"
	ClassElevatedFixed DeploymentClassHint = "elevated_stationary"
	ClassCorridorRelay DeploymentClassHint = "corridor_relay"
	ClassEventTemp     DeploymentClassHint = "event_temporary"
	ClassUnknown       DeploymentClassHint = "unspecified"
)

// CandidatePlacement describes a hypothetical new or moved node placement (no map precision).
type CandidatePlacement struct {
	Label           string              `json:"label,omitempty"`
	DeploymentClass DeploymentClassHint `json:"deployment_class"`
	PlacementNotes  []string            `json:"placement_notes,omitempty"`
	OperatorHints   []PlanningInput     `json:"operator_hints,omitempty"`
}

// CandidateRoleChange is a proposed Meshtastic-style role change (advisory).
type CandidateRoleChange struct {
	NodeNum   int64  `json:"node_num,omitempty"`
	FromRole  string `json:"from_role,omitempty"`
	ToRole    string `json:"to_role,omitempty"`
	Rationale string `json:"rationale,omitempty"`
}

// CandidateNodeClass classifies a hypothetical node for add scenarios.
type CandidateNodeClass string

const (
	NodeClassHandheld       CandidateNodeClass = "handheld"
	NodeClassFixedRelay     CandidateNodeClass = "fixed_relay"
	NodeClassInfraAnchor    CandidateNodeClass = "infrastructure_anchor"
	NodeClassEventEphemeral CandidateNodeClass = "event_ephemeral"
)

// DeploymentStep is one unit of work in a plan.
type DeploymentStep struct {
	ID                 string               `json:"id"`
	Intent             string               `json:"intent"`
	Kind               DeploymentStepKind   `json:"kind"`
	TargetNodeNums     []int64              `json:"target_node_nums,omitempty"`
	TargetClusterHint  string               `json:"target_cluster_hint,omitempty"`
	CandidatePlacement *CandidatePlacement  `json:"candidate_placement,omitempty"`
	RoleChange         *CandidateRoleChange `json:"role_change,omitempty"`
	CandidateClass     CandidateNodeClass   `json:"candidate_class,omitempty"`
	Assumptions        AssumptionSet        `json:"assumptions"`
	ExpectedBenefits   []string             `json:"expected_benefits,omitempty"`
	ExpectedRisks      []RiskFactor         `json:"expected_risks,omitempty"`
	EvidenceBasis      []string             `json:"evidence_basis,omitempty"`
	Uncertainty        ConfidenceAssessment `json:"uncertainty"`
	ObserveAfterHours  int                  `json:"observe_after_hours,omitempty"`
	DoNotProceedIf     []string             `json:"do_not_proceed_if,omitempty"`
}

// DeploymentPlan is a persisted or ephemeral operator deployment proposal.
type DeploymentPlan struct {
	PlanID            string           `json:"plan_id"`
	Title             string           `json:"title"`
	Status            string           `json:"status"` // draft, active, archived
	Intent            string           `json:"intent"`
	Steps             []DeploymentStep `json:"steps"`
	CreatedAt         string           `json:"created_at,omitempty"`
	UpdatedAt         string           `json:"updated_at,omitempty"`
	GraphHashAt       string           `json:"graph_hash_at,omitempty"`
	Notes             []string         `json:"notes,omitempty"`
	InputSetVersionID string           `json:"input_set_version_id,omitempty"` // optional FK to planning_input_versions.version_id
}

// EvidenceModelClassification states how much operator assumption augmented the analysis.
type EvidenceModelClassification string

const (
	EvidenceTopologyOnly                EvidenceModelClassification = "topology_only"
	EvidenceTopologyAssumptionAugmented EvidenceModelClassification = "topology_operator_assumptions"
)

// AssumptionSource tags where an assumption came from.
type AssumptionSource string

const (
	AssumptionSourceObserved AssumptionSource = "observed"
	AssumptionSourceOperator AssumptionSource = "operator"
	AssumptionSourceInferred AssumptionSource = "inferred"
	AssumptionSourceUnknown  AssumptionSource = "unknown"
)

// AssumptionConfidence is a coarse band (not statistical certainty).
type AssumptionConfidence string

const (
	AssumptionConfLow    AssumptionConfidence = "low"
	AssumptionConfMedium AssumptionConfidence = "medium"
	AssumptionConfHigh   AssumptionConfidence = "high"
)

// AssumptionSensitivity estimates how much outcomes move if this input is wrong.
type AssumptionSensitivity string

const (
	SensitivityLow    AssumptionSensitivity = "low"
	SensitivityMedium AssumptionSensitivity = "medium"
	SensitivityHigh   AssumptionSensitivity = "high"
)

// AssumptionItem is one structured planning input with provenance.
type AssumptionItem struct {
	Key         string                `json:"key"`
	Value       string                `json:"value,omitempty"`
	Source      AssumptionSource      `json:"source"`
	Confidence  AssumptionConfidence  `json:"confidence,omitempty"`
	Sensitivity AssumptionSensitivity `json:"sensitivity,omitempty"`
	Description string                `json:"description,omitempty"`
	UsedByModel bool                  `json:"used_by_model"` // false if captured but not yet consumed by estimators
}

// EvidenceReference points at observed artifacts (graph hash, assessment id, etc.).
type EvidenceReference struct {
	Kind string `json:"kind"` // graph_hash, mesh_assessment, topology_snapshot
	Ref  string `json:"ref"`
}

// ValidationTarget describes what signal would confirm or refute a prediction.
type ValidationTarget struct {
	Label        string `json:"label"`
	MetricHint   string `json:"metric_hint,omitempty"`
	ObserveHours int    `json:"observe_hours,omitempty"`
}

// MissingInputNotice lists inputs that weaken confidence when absent.
type MissingInputNotice struct {
	Key         string `json:"key"`
	Impact      string `json:"impact"` // weakens_confidence, blocks_specific_claim
	Description string `json:"description,omitempty"`
}

// InputConflictNotice surfaces contradictory operator inputs.
type InputConflictNotice struct {
	Keys        []string `json:"keys"`
	Description string   `json:"description"`
}

// PlanningInputVersionPayload is the versioned document stored for an input set.
type PlanningInputVersionPayload struct {
	VersionNum        int                         `json:"version_num"`
	InputSetID        string                      `json:"input_set_id"`
	EvidenceModel     EvidenceModelClassification `json:"evidence_classification"`
	Assumptions       []AssumptionItem            `json:"assumptions"`
	ObservedAnchors   []EvidenceReference         `json:"observed_anchors,omitempty"`
	MissingInputs     []MissingInputNotice        `json:"missing_inputs,omitempty"`
	Conflicts         []InputConflictNotice       `json:"conflicts,omitempty"`
	ValidationTargets []ValidationTarget          `json:"validation_targets,omitempty"`
	Notes             []string                    `json:"notes,omitempty"`
	CreatedAt         string                      `json:"created_at,omitempty"`
}

// PlanningInputSetMeta is list metadata for an input set.
type PlanningInputSetMeta struct {
	InputSetID string `json:"input_set_id"`
	Title      string `json:"title"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}

// ScenarioKind is the what-if scenario type.
type ScenarioKind string

const (
	ScenarioAddNode           ScenarioKind = "add_node"
	ScenarioRemoveNode        ScenarioKind = "remove_node"
	ScenarioMoveNode          ScenarioKind = "move_node"
	ScenarioElevateNode       ScenarioKind = "elevate_node"
	ScenarioChangeRole        ScenarioKind = "change_role"
	ScenarioUptimeAlwaysOn    ScenarioKind = "convert_intermittent_to_always_on"
	ScenarioAddInfrastructure ScenarioKind = "add_infrastructure_class"
	ScenarioReduceInfraRole   ScenarioKind = "reduce_infrastructure_role"
	ScenarioBridgeClusters    ScenarioKind = "bridge_weak_clusters"
	ScenarioAddRedundancy     ScenarioKind = "add_redundancy_to_bridge"
)

// ScenarioAssessment is the bounded output of a what-if run.
type ScenarioAssessment struct {
	ScenarioID           string                 `json:"scenario_id"`
	Kind                 ScenarioKind           `json:"kind"`
	TargetNodeNum        int64                  `json:"target_node_num,omitempty"`
	Outcome              ExpectedOutcome        `json:"expected_outcome"`
	Verdict              PlanVerdict            `json:"verdict"`
	Confidence           ConfidenceAssessment   `json:"confidence"`
	Drivers              []string               `json:"drivers"` // observed facts
	AssumptionImpact     []string               `json:"assumption_impact,omitempty"`
	Risks                []RiskFactor           `json:"risks,omitempty"`
	EffectsSummary       ScenarioEffectsSummary `json:"effects_summary"`
	EvidenceModel        string                 `json:"evidence_model"`
	ComputedAt           string                 `json:"computed_at"`
	ReferencedGraph      string                 `json:"referenced_graph_hash,omitempty"`
	ReferencedAssessment string                 `json:"referenced_mesh_assessment_id,omitempty"`
}

// ScenarioEffectsSummary lists qualitative deltas (not numeric RF).
type ScenarioEffectsSummary struct {
	BootstrapViability      string `json:"bootstrap_viability_delta"` // e.g. "likely_improve", "likely_worsen", "uncertain"
	Fragmentation           string `json:"fragmentation_delta"`
	DependencyConcentration string `json:"dependency_concentration_delta"`
	Resilience              string `json:"resilience_delta"`
	RoutingStress           string `json:"routing_stress_delta"`
	CoverageContribution    string `json:"coverage_contribution_note"`
	RelayValue              string `json:"relay_value_note"`
	OperatorValue           string `json:"operator_value_note"`
}

// ImpactKind identifies an impact-estimation request.
type ImpactKind string

const (
	ImpactAdd    ImpactKind = "add"
	ImpactMove   ImpactKind = "move"
	ImpactRemove ImpactKind = "remove"
	ImpactRole   ImpactKind = "role_change"
	ImpactUptime ImpactKind = "uptime_improvement"
)

// NodeImpactAssessment is impact estimation for a concrete intervention.
type NodeImpactAssessment struct {
	Kind           ImpactKind           `json:"kind"`
	NodeNum        int64                `json:"node_num,omitempty"`
	CandidateClass CandidateNodeClass   `json:"candidate_class,omitempty"`
	Outcome        ExpectedOutcome      `json:"expected_outcome"`
	Verdict        PlanVerdict          `json:"verdict"`
	BlastRadius    []string             `json:"blast_radius_if_wrong,omitempty"`
	Confidence     ConfidenceAssessment `json:"confidence"`
	Lines          []string             `json:"summary_lines"`
	Evidence       []string             `json:"evidence"`
}

// PlaybookClass identifies a seeding / growth playbook template.
type PlaybookClass string

const (
	PlaybookLoneWolf          PlaybookClass = "lone_wolf_bootstrap"
	PlaybookNeighborhoodSeed  PlaybookClass = "neighborhood_seed"
	PlaybookFragileStabilize  PlaybookClass = "fragile_cluster_stabilization"
	PlaybookEventMesh         PlaybookClass = "event_mesh"
	PlaybookCorridor          PlaybookClass = "corridor_expansion"
	PlaybookResilienceUpgrade PlaybookClass = "resilience_upgrade"
	PlaybookDependencyReduce  PlaybookClass = "dependency_reduction"
)

// PlaybookStep is one ordered step in a community seeding playbook.
type PlaybookStep struct {
	Order               int      `json:"order"`
	Title               string   `json:"title"`
	Rationale           string   `json:"rationale"`
	ObserveHours        int      `json:"observe_hours"`
	SuccessIndicators   []string `json:"success_indicators"`
	FailureIndicators   []string `json:"failure_indicators"`
	DoNotProceedIf      []string `json:"do_not_proceed_if,omitempty"`
	RollbackOrPauseNote string   `json:"rollback_or_pause_note,omitempty"`
}

// Playbook is a field-guide style sequence derived from observed state.
type Playbook struct {
	Class            PlaybookClass  `json:"class"`
	Title            string         `json:"title"`
	Summary          string         `json:"summary"`
	MinimumMilestone string         `json:"minimum_viable_milestone"`
	Steps            []PlaybookStep `json:"steps"`
	Limits           []string       `json:"limits"`
	EvidenceAnchors  []string       `json:"evidence_anchors"`
}

// SinglePointOfFailureClass coarse SPOF labeling.
type SinglePointOfFailureClass string

const (
	SPOFNone     SinglePointOfFailureClass = "none_indicated"
	SPOFProbable SinglePointOfFailureClass = "probable_bridge_spof"
	SPOFPossible SinglePointOfFailureClass = "possible_dependency"
	SPOFUnknown  SinglePointOfFailureClass = "unknown_insufficient_graph"
)

// NodeResilienceProfile is per-node criticality and redundancy signals.
type NodeResilienceProfile struct {
	NodeNum                    int64                     `json:"node_num"`
	ShortName                  string                    `json:"short_name,omitempty"`
	CriticalNodeScore          float64                   `json:"critical_node_score"`          // 0–1 higher = more critical
	RedundancyScore            float64                   `json:"redundancy_score"`             // 0–1 higher = more alternate paths
	PartitionRiskScore         float64                   `json:"partition_risk_score"`         // loss would fragment
	ResilienceScore            float64                   `json:"resilience_score"`             // composite "good to keep up"
	RedundancyOpportunityScore float64                   `json:"redundancy_opportunity_score"` // benefit of adding alt path here
	SPOFClass                  SinglePointOfFailureClass `json:"spof_class"`
	RecoveryPriority           int                       `json:"recovery_priority"` // 1 = highest
	Explanation                []string                  `json:"explanation"`
}

// MeshResilienceSummary aggregates cluster-level resilience.
type MeshResilienceSummary struct {
	ResilienceScore      float64              `json:"resilience_score"` // 0–1
	RedundancyScore      float64              `json:"redundancy_score"`
	PartitionRiskScore   float64              `json:"partition_risk_score"`
	FragilityExplanation []string             `json:"fragility_explanation"`
	NextBestMoveSummary  string               `json:"next_best_move_summary"`
	Confidence           ConfidenceAssessment `json:"confidence"`
}

// PlanComparison compares two or more plans or scenarios.
type PlanComparison struct {
	ComparedIDs            []string                    `json:"compared_ids"`
	RankedByUpside         []ComparisonRankEntry       `json:"ranked_by_upside"`
	RankedBySafety         []ComparisonRankEntry       `json:"ranked_by_safety"`
	RankedByLowRegret      []ComparisonRankEntry       `json:"ranked_by_low_regret,omitempty"`
	LowRegretPick          string                      `json:"low_regret_pick_id,omitempty"`
	BestUpsidePick         string                      `json:"best_upside_pick_id,omitempty"`
	BestResiliencePick     string                      `json:"best_resilience_pick_id,omitempty"`
	BestDiagnosticPick     string                      `json:"best_diagnostic_pick_id,omitempty"`
	CheapestPlausible      string                      `json:"cheapest_plausible_pick_id,omitempty"`
	WaitObserveOption      string                      `json:"wait_observe_option_id,omitempty"`
	RankingCouldChangeIf   []string                    `json:"ranking_could_change_if,omitempty"`
	SummaryLines           []string                    `json:"summary_lines"`
	Confidence             ConfidenceAssessment        `json:"confidence"`
	EvidenceClassification EvidenceModelClassification `json:"evidence_classification"`
}

// DecisionDimensionScores are explicit 0–1 scores for ranking narratives (not dollar costs).
type DecisionDimensionScores struct {
	ReversibilityScore         float64 `json:"reversibility_score"`
	ObservationBurdenScore     float64 `json:"observation_burden_score"`
	DiagnosticValueScore       float64 `json:"diagnostic_value_score"`
	LearningValueScore         float64 `json:"learning_value_score"`
	CostComplexityProxy        float64 `json:"cost_complexity_proxy"`
	LowRegretScore             float64 `json:"low_regret_score"`
	ExpansionReadinessScore    float64 `json:"expansion_readiness_score"`
	AssumptionFragilityScore   float64 `json:"assumption_fragility_score"`
	OperationalDisruptionScore float64 `json:"operational_disruption_score"`
	UpsideScore                float64 `json:"upside_score"`
	UncertaintyPenalty         float64 `json:"uncertainty_penalty"`
}

// ComparisonRankEntry ranks one candidate with tradeoff notes.
type ComparisonRankEntry struct {
	ID                string                  `json:"id"`
	Label             string                  `json:"label"`
	Upside            string                  `json:"upside"`
	DownsideIfWrong   string                  `json:"downside_if_wrong"`
	ConfidenceNote    string                  `json:"confidence_note"`
	Reversibility     string                  `json:"reversibility"` // high, medium, low
	ObservationBurden string                  `json:"observation_burden"`
	Dimensions        DecisionDimensionScores `json:"dimensions"`
	NarrativeLines    []string                `json:"narrative_lines,omitempty"`
}

// OutcomeVerdict closes the loop on plan predictions.
type OutcomeVerdict string

const (
	OutcomeVerdictSupported               OutcomeVerdict = "prediction_direction_supported"
	OutcomeVerdictContradicted            OutcomeVerdict = "prediction_contradicted"
	OutcomeVerdictInconclusive            OutcomeVerdict = "inconclusive"
	OutcomeVerdictInsufficientObservation OutcomeVerdict = "insufficient_observation_period"
	OutcomeVerdictConfounded              OutcomeVerdict = "confounded_concurrent_changes"
)

// PlanExecutionRecord captures operator attempt and observation window.
type PlanExecutionRecord struct {
	ExecutionID             string                    `json:"execution_id"`
	PlanID                  string                    `json:"plan_id"`
	PlanGraphHash           string                    `json:"plan_graph_hash"`
	MeshAssessmentID        string                    `json:"mesh_assessment_id"`
	BaselineMetrics         PostChangeMetricsSnapshot `json:"baseline_metrics,omitempty"`
	Status                  string                    `json:"status"` // attempted, in_observation, completed
	StartedAt               string                    `json:"started_at"`
	UpdatedAt               string                    `json:"updated_at"`
	ObservationHorizonHours int                       `json:"observation_horizon_hours"`
	Notes                   string                    `json:"notes,omitempty"`
}

// StepExecutionRecord is one step marked executed.
type StepExecutionRecord struct {
	StepExecutionID string `json:"step_execution_id"`
	ExecutionID     string `json:"execution_id"`
	StepID          string `json:"step_id"`
	Status          string `json:"status"`
	AttemptedAt     string `json:"attempted_at"`
	OperatorNote    string `json:"operator_note,omitempty"`
}

// ValidationResult compares post-change mesh to expectations.
type ValidationResult struct {
	ValidationID          string                    `json:"validation_id"`
	ExecutionID           string                    `json:"execution_id"`
	ValidatedAt           string                    `json:"validated_at"`
	GraphHashAfter        string                    `json:"graph_hash_after"`
	MeshAssessmentIDAfter string                    `json:"mesh_assessment_id_after"`
	Verdict               OutcomeVerdict            `json:"verdict"`
	EvidenceFlags         PlanningEvidenceFlags     `json:"evidence_flags"`
	Caveat                string                    `json:"caveat,omitempty"`
	Lines                 []string                  `json:"lines"`
	Metrics               PostChangeMetricsSnapshot `json:"metrics"`
}

// PostChangeMetricsSnapshot is compact before/after style signals.
type PostChangeMetricsSnapshot struct {
	Captured            bool    `json:"captured,omitempty"` // true when execution recorded baseline at start
	FragmentationBefore float64 `json:"fragmentation_before,omitempty"`
	FragmentationAfter  float64 `json:"fragmentation_after,omitempty"`
	ResilienceBefore    float64 `json:"resilience_before,omitempty"`
	ResilienceAfter     float64 `json:"resilience_after,omitempty"`
}

// RecommendationRetrospective is stored history for a recommendation key.
type RecommendationRetrospective struct {
	RecommendationKey string `json:"recommendation_key"`
	SuccessCount      int    `json:"success_count"`
	InconclusiveCount int    `json:"inconclusive_count"`
	ContradictedCount int    `json:"contradicted_count"`
	TotalRecorded     int    `json:"total_recorded"`
}

// PlanningBundle is the full advisory snapshot for API responses.
type PlanningBundle struct {
	EvidenceModel      string                  `json:"evidence_model"`
	GraphHash          string                  `json:"graph_hash"`
	MeshAssessmentID   string                  `json:"mesh_assessment_id,omitempty"`
	TransportConnected bool                    `json:"transport_connected"`
	TopologyEnabled    bool                    `json:"topology_enabled"`
	EvidenceFlags      PlanningEvidenceFlags   `json:"evidence_flags"`
	Resilience         MeshResilienceSummary   `json:"resilience"`
	NodeProfiles       []NodeResilienceProfile `json:"node_profiles"`
	RankedNextPlans    []RankedPlanHint        `json:"ranked_next_plans"`
	BestNextMove       BestNextMove            `json:"best_next_move"`
	WaitVersusExpand   string                  `json:"wait_versus_expand_hint"`
	Playbooks          []Playbook              `json:"playbooks"`
	Limits             []string                `json:"limits"`
	ComputedAt         string                  `json:"computed_at"`
}

// RankedPlanHint is a concise next-step option with tradeoffs.
type RankedPlanHint struct {
	Rank        int                `json:"rank"`
	ID          string             `json:"id"`
	Title       string             `json:"title"`
	Verdict     PlanVerdict        `json:"verdict"`
	BenefitBand QualitativeOutcome `json:"benefit_band"`
	Lines       []string           `json:"lines"`
}
