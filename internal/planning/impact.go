package planning

import (
	"fmt"
	"strings"

	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/topology"
)

// ParseImpactCandidateClass maps API query values to CandidateNodeClass.
func ParseImpactCandidateClass(s string) CandidateNodeClass {
	return parseCandidateClass(s)
}

// EstimateImpact returns a bounded impact assessment for add/move/remove/role/uptime interventions.
func EstimateImpact(kind ImpactKind, nodeNum int64, candidateClass CandidateNodeClass, ar topology.AnalysisResult, mi meshintel.Assessment) NodeImpactAssessment {
	bridgeSet := mapFromInts(ar.BridgeNodes)
	n, ok := NodeByNum(ar, nodeNum)
	ni := nodeIntel(mi, nodeNum)

	var lines []string
	var evidence []string
	outcome := ExpectedOutcome{BenefitBand: OutcomePlausibleModerate}
	verdict := VerdictProceedWithCaution
	var blast []string

	switch kind {
	case ImpactAdd:
		lines, evidence, outcome, verdict, blast = impactAdd(candidateClass, ar, mi)
	case ImpactMove:
		if !ok {
			return missingNodeImpact(kind, nodeNum)
		}
		lines, evidence, outcome, verdict, blast = impactMove(n, ni, bridgeSet, ar, mi)
	case ImpactRemove:
		if !ok {
			return missingNodeImpact(kind, nodeNum)
		}
		lines, evidence, outcome, verdict, blast = impactRemove(n, ni, bridgeSet, ar)
	case ImpactRole:
		if !ok {
			return missingNodeImpact(kind, nodeNum)
		}
		lines, evidence, outcome, verdict, blast = impactRole(n, ni, mi)
	case ImpactUptime:
		if !ok {
			return missingNodeImpact(kind, nodeNum)
		}
		lines, evidence, outcome, verdict, blast = impactUptime(n, ni, bridgeSet, mi)
	default:
		lines = []string{"Unsupported impact kind."}
		outcome.BenefitBand = OutcomeHighRiskLowConfidence
		verdict = VerdictInsufficientData
	}

	conf := ConfidenceAssessment{
		Level:             mi.Bootstrap.Confidence,
		Score:             coarseConfidenceScore(mi.Bootstrap.Confidence),
		MissingInputs:     []string{"rf_neighbors_not_in_graph", "terrain", "antenna_patterns"},
		WouldValidateWith: []string{"sustained_packet_paths_after_change", "stable_edges_in_topology_links"},
		TopologyOnlyLimits: []string{
			"Impact is structural (observed graph), not a coverage map.",
		},
	}
	outcome.TopologyOnlyNote = "Topology-only estimate; no RF simulation."

	return NodeImpactAssessment{
		Kind:           kind,
		NodeNum:        nodeNum,
		CandidateClass: candidateClass,
		Outcome:        outcome,
		Verdict:        verdict,
		BlastRadius:    blast,
		Confidence:     conf,
		Lines:          lines,
		Evidence:       evidence,
	}
}

func missingNodeImpact(kind ImpactKind, nodeNum int64) NodeImpactAssessment {
	return NodeImpactAssessment{
		Kind:    kind,
		NodeNum: nodeNum,
		Outcome: ExpectedOutcome{BenefitBand: OutcomeHighRiskLowConfidence, Rationale: []string{"Node not present in observed topology."}},
		Verdict: VerdictInsufficientData,
		Lines:   []string{fmt.Sprintf("Node %d not found in current topology store — cannot estimate impact.", nodeNum)},
		Evidence: []string{"no_matching_topology_node"},
		Confidence: ConfidenceAssessment{
			Level: meshintel.ConfidenceLow,
			Score: 0.2,
			MissingInputs: []string{"observed_node_record"},
		},
	}
}

func mapFromInts(nums []int64) map[int64]bool {
	m := make(map[int64]bool)
	for _, x := range nums {
		m[x] = true
	}
	return m
}

func nodeIntel(mi meshintel.Assessment, nodeNum int64) *meshintel.NodeTopologyIntel {
	for i := range mi.NodeIntel {
		if mi.NodeIntel[i].NodeNum == nodeNum {
			return &mi.NodeIntel[i]
		}
	}
	return nil
}

func impactAdd(class CandidateNodeClass, ar topology.AnalysisResult, mi meshintel.Assessment) ([]string, []string, ExpectedOutcome, PlanVerdict, []string) {
	var lines []string
	evidence := []string{
		fmt.Sprintf("largest_component=%d", mi.Bootstrap.LargestComponentSize),
		fmt.Sprintf("viability=%s", mi.Bootstrap.Viability),
	}
	benefit := OutcomePlausibleModerate
	verdict := VerdictProceedWithCaution
	var blast []string

	switch class {
	case NodeClassInfraAnchor:
		lines = append(lines, "Adding a stationary, always-on anchor often raises bootstrap viability more than another handheld when the graph is sparse or fragmented.")
		if mi.Topology.FragmentationScore > 0.25 {
			benefit = OutcomeLikelyHighLeverage
		}
	case NodeClassHandheld:
		lines = append(lines, "Another handheld often has lower leverage until at least one stable backbone path exists.")
		if mi.Bootstrap.Viability == meshintel.ViabilityWeakBootstrap || mi.Bootstrap.LargestComponentSize <= 2 {
			benefit = OutcomeLikelyLowBenefit
			verdict = VerdictDeferObserve
		}
	case NodeClassFixedRelay, NodeClassEventEphemeral:
		lines = append(lines, "A fixed relay can bridge gaps; event/temporary nodes help only if observations confirm paths during the window.")
	default:
		lines = append(lines, "Unspecified node class — assume modest structural effect until role and placement are known.")
		benefit = OutcomeUncertainButPromising
	}

	if mi.Bootstrap.Viability == meshintel.ViabilityIsolated {
		verdict = VerdictDeferObserve
		lines = append(lines, "Observed graph still isolated — fix mutual visibility or transport observation before counting on new hardware.")
	}

	blast = append(blast, "If placement is wrong, you may add noise/forwarding load without improving paths MEL can see.")
	return lines, evidence, ExpectedOutcome{BenefitBand: benefit, Rationale: lines}, verdict, blast
}

func impactMove(n topology.Node, ni *meshintel.NodeTopologyIntel, bridges map[int64]bool, ar topology.AnalysisResult, mi meshintel.Assessment) ([]string, []string, ExpectedOutcome, PlanVerdict, []string) {
	var lines []string
	evidence := []string{}
	if ni != nil {
		evidence = append(evidence, fmt.Sprintf("placement_quality_score=%.2f", ni.PlacementQualityScore))
	}
	if bridges[n.NodeNum] {
		evidence = append(evidence, "node_flagged_bridge_critical")
	}
	benefit := OutcomePlausibleModerate
	verdict := VerdictProceedWithCaution
	var blast []string

	lines = append(lines, "Moving a node only helps if RF visibility improves; MEL will infer that later via new packet paths — not via maps here.")
	if ni != nil && ni.PlacementQualityScore < 0.45 {
		benefit = OutcomeLikelyHighLeverage
		lines = append(lines, "Observed placement quality is weak — elevation or clearer line-of-sight is a higher-odds fix than buying another device.")
	}
	if bridges[n.NodeNum] {
		lines = append(lines, "This node looks bridge-like — moving it is higher risk: you may break the only observed path.")
		verdict = VerdictProceedWithCaution
		blast = append(blast, "Temporary partition if the only corridor moved away from neighbors.")
		benefit = OutcomeUncertainButPromising
	}
	if mi.RoutingPressure.OverRouterizationRiskScore.Score > 0.55 && strings.Contains(strings.ToLower(n.Role), "router") {
		lines = append(lines, "Routing pressure is elevated — consider role/placement before relocating a router-class node.")
	}
	return lines, evidence, ExpectedOutcome{BenefitBand: benefit, Rationale: lines}, verdict, blast
}

func impactRemove(n topology.Node, ni *meshintel.NodeTopologyIntel, bridges map[int64]bool, ar topology.AnalysisResult) ([]string, []string, ExpectedOutcome, PlanVerdict, []string) {
	var lines []string
	evidence := []string{fmt.Sprintf("health_state=%s", n.HealthState)}
	if ni != nil {
		evidence = append(evidence, fmt.Sprintf("relay_value=%.2f", ni.RelayValueScore))
	}
	benefit := OutcomeLikelyHarmful
	verdict := VerdictLikelyHarmful
	var blast []string

	if bridges[n.NodeNum] {
		lines = append(lines, "Removing this node would likely increase fragmentation — it appears bridge-like in the observed graph.")
		blast = append(blast, "Neighbors may become isolated in MEL's observed topology.")
	} else if ni != nil && ni.UndirectedDegree >= 3 {
		benefit = OutcomePlausibleModerate
		verdict = VerdictProceedWithCaution
		lines = append(lines, "Degree is high enough that removal might be survivable — still verify on-air behavior.")
	} else {
		lines = append(lines, "Removal risk is dominated by whether this node is the only path between parts of the mesh.")
		blast = append(blast, "Possible partition if degree is misleading or edges are stale.")
	}
	return lines, evidence, ExpectedOutcome{BenefitBand: benefit, Rationale: lines}, verdict, blast
}

func impactRole(n topology.Node, ni *meshintel.NodeTopologyIntel, mi meshintel.Assessment) ([]string, []string, ExpectedOutcome, PlanVerdict, []string) {
	lines := []string{
		"Reducing router/relay aggressiveness can lower forwarding pressure when the graph is dense.",
		"Changing roles does not change physics — placement and power dominate when sparse.",
	}
	evidence := []string{fmt.Sprintf("role=%s", n.Role)}
	if ni != nil {
		evidence = append(evidence, fmt.Sprintf("relay_value=%.2f", ni.RelayValueScore))
	}
	benefit := OutcomePlausibleModerate
	verdict := VerdictProceedWithCaution
	var blast []string
	if mi.RoutingPressure.OverRouterizationRiskScore.Score > 0.5 {
		benefit = OutcomeLikelyHighLeverage
	}
	if ni != nil && ni.IsBridgeCritical {
		blast = append(blast, "If this node is the only viable forwarder between regions, de-routing may harm reachability.")
		verdict = VerdictProceedWithCaution
	}
	return lines, evidence, ExpectedOutcome{BenefitBand: benefit, Rationale: lines}, verdict, blast
}

func impactUptime(n topology.Node, ni *meshintel.NodeTopologyIntel, bridges map[int64]bool, mi meshintel.Assessment) ([]string, []string, ExpectedOutcome, PlanVerdict, []string) {
	lines := []string{
		"Making a high-value intermittent node always-on often beats adding another endpoint — if this node is structurally important.",
	}
	evidence := []string{fmt.Sprintf("stale=%v", n.Stale)}
	if ni != nil {
		evidence = append(evidence, fmt.Sprintf("relay_value=%.2f", ni.RelayValueScore))
	}
	benefit := OutcomePlausibleModerate
	verdict := VerdictLikelyWorthwhile
	if ni != nil && (ni.RelayValueScore > 0.55 || bridges[n.NodeNum]) {
		benefit = OutcomeLikelyHighLeverage
	}
	if n.Stale {
		lines = append(lines, "Node looks stale — uptime improvement may be power/placement, not firmware.")
	}
	var blast []string
	blast = append(blast, "If the node is poorly placed, more uptime only amplifies forwarding stress.")
	return lines, evidence, ExpectedOutcome{BenefitBand: benefit, Rationale: lines}, verdict, blast
}
