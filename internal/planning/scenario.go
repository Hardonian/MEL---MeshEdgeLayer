package planning

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/topology"
)

// RunScenario produces a bounded what-if assessment.
func RunScenario(kind ScenarioKind, targetNode int64, ar topology.AnalysisResult, mi meshintel.Assessment, now time.Time) ScenarioAssessment {
	return RunScenarioWithClass(kind, targetNode, "", ar, mi, now)
}

// RunScenarioWithClass allows optional candidate_class for add-style scenarios (handheld|fixed_relay|infrastructure_anchor|event_ephemeral).
func RunScenarioWithClass(kind ScenarioKind, targetNode int64, candidateClass string, ar topology.AnalysisResult, mi meshintel.Assessment, now time.Time) ScenarioAssessment {
	h := sha256.New()
	fmt.Fprintf(h, "sc:%s:%d:%s:%s", kind, targetNode, ar.Snapshot.GraphHash, mi.AssessmentID)
	sid := fmt.Sprintf("sc-%s-%x", now.UTC().Format("20060102-150405"), h.Sum(nil)[:4])

	class := parseCandidateClass(candidateClass)
	impactKind := ImpactAdd
	switch kind {
	case ScenarioRemoveNode:
		impactKind = ImpactRemove
	case ScenarioMoveNode, ScenarioElevateNode:
		impactKind = ImpactMove
	case ScenarioChangeRole, ScenarioReduceInfraRole:
		impactKind = ImpactRole
	case ScenarioUptimeAlwaysOn:
		impactKind = ImpactUptime
	case ScenarioAddNode:
		if class == "" {
			class = NodeClassHandheld
		}
		impactKind = ImpactAdd
	case ScenarioAddInfrastructure:
		class = NodeClassInfraAnchor
		impactKind = ImpactAdd
	case ScenarioBridgeClusters, ScenarioAddRedundancy:
		impactKind = ImpactAdd
		class = NodeClassFixedRelay
	}

	var ni *meshintel.NodeTopologyIntel
	for i := range mi.NodeIntel {
		if mi.NodeIntel[i].NodeNum == targetNode {
			ni = &mi.NodeIntel[i]
			break
		}
	}

	impact := EstimateImpact(impactKind, targetNode, class, ar, mi)
	if kind == ScenarioBridgeClusters || kind == ScenarioAddRedundancy {
		impact.Lines = []string{
			"Bridging / redundancy aims to add a second path between components — benefit depends on whether the new node can hear both sides.",
			"MEL cannot confirm RF reachability without post-change observations.",
		}
		impact.Outcome.BenefitBand = OutcomeUncertainButPromising
		if mi.Topology.FragmentationScore > 0.3 {
			impact.Outcome.BenefitBand = OutcomeLikelyHighLeverage
		}
	}
	if kind == ScenarioElevateNode && targetNode != 0 {
		impact.Lines = append([]string{"Elevation changes RF but MEL only sees results via new packet paths — treat as a move with upside if line-of-sight improves."}, impact.Lines...)
	}

	effects := ScenarioEffectsSummary{
		BootstrapViability:   deltaBootstrap(mi, impact.Outcome.BenefitBand),
		Fragmentation:        deltaFragmentation(kind, impact.Outcome.BenefitBand),
		DependencyConcentration: deltaDependency(kind, ni, mi),
		Resilience:           deltaResilience(kind, impact.Outcome.BenefitBand),
		RoutingStress:        deltaRouting(kind, mi),
		CoverageContribution: "Coverage is not modeled; only topology connectivity and relay hints.",
		RelayValue:           relayNote(ni),
		OperatorValue:        "Operator value follows bootstrap/resilience and observation burden — not SKU economics.",
	}

	drivers := []string{
		fmt.Sprintf("mesh_viability=%s", mi.Bootstrap.Viability),
		fmt.Sprintf("cluster_shape=%s", mi.Topology.ClusterShape),
	}
	if ni != nil {
		drivers = append(drivers, fmt.Sprintf("node_relay_value_score=%.2f", ni.RelayValueScore))
	}

	conf := ConfidenceAssessment{
		Level:             mi.Bootstrap.Confidence,
		Score:             coarseConfidenceScore(mi.Bootstrap.Confidence),
		MissingInputs:     []string{"terrain", "antenna_height", "obstructions", "rf_spectrum_noise"},
		WouldValidateWith: []string{"new_edges_after_change", "stable_component_merge_in_observed_graph"},
		TopologyOnlyLimits: []string{
			"Scenario is topology- and traffic-signal-based; not RF simulation.",
		},
	}

	return ScenarioAssessment{
		ScenarioID:           sid,
		Kind:                 kind,
		TargetNodeNum:        targetNode,
		Outcome:              impact.Outcome,
		Verdict:              impact.Verdict,
		Confidence:           conf,
		Drivers:              drivers,
		AssumptionImpact:     []string{"If operator assumptions about placement differ from reality, outcomes may differ widely."},
		Risks:                risksFromImpact(impact),
		EffectsSummary:       effects,
		EvidenceModel:        PlanningEvidenceModel,
		ComputedAt:           now.UTC().Format(time.RFC3339),
		ReferencedGraph:      ar.Snapshot.GraphHash,
		ReferencedAssessment: mi.AssessmentID,
	}
}

func deltaBootstrap(mi meshintel.Assessment, band QualitativeOutcome) string {
	if band == OutcomeLikelyLowBenefit || band == OutcomeLikelyHarmful {
		return "likely_flat_or_worse"
	}
	if mi.Bootstrap.Viability == meshintel.ViabilityIsolated {
		return "uncertain_until_mutual_visibility"
	}
	return "plausible_improvement_if_paths_appear"
}

func deltaFragmentation(kind ScenarioKind, band QualitativeOutcome) string {
	switch kind {
	case ScenarioRemoveNode:
		return "likely_worse_if_bridge"
	case ScenarioBridgeClusters, ScenarioAddRedundancy:
		return "likely_improve_if_second_path"
	default:
		if band == OutcomeLikelyHarmful {
			return "likely_worse"
		}
		return "uncertain"
	}
}

func deltaDependency(kind ScenarioKind, ni *meshintel.NodeTopologyIntel, mi meshintel.Assessment) string {
	if kind == ScenarioAddInfrastructure || kind == ScenarioBridgeClusters {
		return "likely_lower_if_new_path_reduces_hub_reliance"
	}
	if kind == ScenarioReduceInfraRole && ni != nil && ni.RelayValueScore > 0.6 {
		return "may_increase_if_relay_shifts_elsewhere"
	}
	if mi.Topology.DependencyConcentrationScore > 0.5 {
		return "high_concentration_observed"
	}
	return "uncertain"
}

func deltaResilience(kind ScenarioKind, band QualitativeOutcome) string {
	switch kind {
	case ScenarioRemoveNode:
		return "likely_worse"
	case ScenarioBridgeClusters, ScenarioAddRedundancy, ScenarioAddInfrastructure:
		return "likely_improve"
	case ScenarioUptimeAlwaysOn:
		return "likely_improve_if_node_is_structural"
	default:
		if band == OutcomeLikelyHighLeverage {
			return "plausible_improve"
		}
		return "uncertain"
	}
}

func deltaRouting(kind ScenarioKind, mi meshintel.Assessment) string {
	if kind == ScenarioReduceInfraRole || kind == ScenarioChangeRole {
		if mi.RoutingPressure.OverRouterizationRiskScore.Score > 0.5 {
			return "likely_relief_if_roles_match_density"
		}
	}
	if kind == ScenarioAddNode && mi.RoutingPressure.OverRouterizationRiskScore.Score > 0.55 {
		return "may_increase_if_new_node_forwards_heavily"
	}
	return "uncertain"
}

func relayNote(ni *meshintel.NodeTopologyIntel) string {
	if ni == nil {
		return "No per-node intel row — relay value unknown."
	}
	return fmt.Sprintf("Relay value score %.2f is packet-graph derived, not RF.", ni.RelayValueScore)
}

func parseCandidateClass(s string) CandidateNodeClass {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "handheld", "companion":
		return NodeClassHandheld
	case "fixed_relay", "relay", "corridor":
		return NodeClassFixedRelay
	case "infrastructure_anchor", "infra", "anchor", "base":
		return NodeClassInfraAnchor
	case "event_ephemeral", "event", "temporary":
		return NodeClassEventEphemeral
	default:
		return ""
	}
}

func risksFromImpact(impact NodeImpactAssessment) []RiskFactor {
	var out []RiskFactor
	for _, b := range impact.BlastRadius {
		out = append(out, RiskFactor{
			Code:        "blast_radius",
			Severity:    "medium",
			Description: b,
			Evidence:    impact.Evidence,
		})
	}
	if len(out) == 0 {
		out = append(out, RiskFactor{
			Code:        "observation_gap",
			Severity:    "low",
			Description: "Pre-change graph may miss hidden RF paths.",
			Evidence:    []string{"packet_derived_topology"},
		})
	}
	return out
}

// NormalizeScenarioKind parses a string into ScenarioKind.
func NormalizeScenarioKind(s string) (ScenarioKind, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "add_node", "add":
		return ScenarioAddNode, true
	case "remove_node", "remove":
		return ScenarioRemoveNode, true
	case "move_node", "move":
		return ScenarioMoveNode, true
	case "elevate_node", "elevate":
		return ScenarioElevateNode, true
	case "change_role", "role":
		return ScenarioChangeRole, true
	case "uptime", "always_on", "convert_intermittent_to_always_on":
		return ScenarioUptimeAlwaysOn, true
	case "add_infrastructure", "infra":
		return ScenarioAddInfrastructure, true
	case "reduce_infra", "reduce_infrastructure_role":
		return ScenarioReduceInfraRole, true
	case "bridge", "bridge_clusters":
		return ScenarioBridgeClusters, true
	case "redundancy", "add_redundancy":
		return ScenarioAddRedundancy, true
	default:
		return "", false
	}
}
