package planning

import (
	"fmt"
	"sort"

	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/topology"
)

// ComputeResilience derives criticality and redundancy profiles from topology + mesh intel.
func ComputeResilience(ar topology.AnalysisResult, mi meshintel.Assessment) (MeshResilienceSummary, []NodeResilienceProfile) {
	nodes := ar.ScoredNodes
	bridgeSet := make(map[int64]bool)
	for _, b := range ar.BridgeNodes {
		bridgeSet[b] = true
	}
	intelByNum := make(map[int64]meshintel.NodeTopologyIntel)
	for _, ni := range mi.NodeIntel {
		intelByNum[ni.NodeNum] = ni
	}

	adj := buildAdjCounts(nodes, ar.ScoredLinks)
	var profiles []NodeResilienceProfile
	for _, n := range nodes {
		deg := adj[n.NodeNum]
		ni := intelByNum[n.NodeNum]
		crit := 0.35
		if bridgeSet[n.NodeNum] {
			crit += 0.45
		}
		if deg <= 1 && len(nodes) > 1 {
			crit += 0.15
		}
		if ni.IsBridgeCritical {
			crit += 0.1
		}
		if ni.RelayValueScore > 0.65 {
			crit += 0.05
		}
		crit = clamp01(crit)

		redund := 0.2 + 0.15*float64(minInt(deg, 6))
		if bridgeSet[n.NodeNum] {
			redund -= 0.35
		}
		if deg >= 3 {
			redund += 0.15
		}
		redund = clamp01(redund)

		partRisk := 0.25
		if bridgeSet[n.NodeNum] || deg <= 1 {
			partRisk += 0.45
		}
		if ni.IsBridgeCritical {
			partRisk += 0.15
		}
		partRisk = clamp01(partRisk)

		resilience := clamp01(0.55*redund + 0.45*(1-partRisk))

		redundOpp := clamp01(partRisk*0.7 + (1-redund)*0.3)

		spof := SPOFPossible
		if bridgeSet[n.NodeNum] && deg <= 2 {
			spof = SPOFProbable
		}
		if deg >= 3 && !bridgeSet[n.NodeNum] {
			spof = SPOFNone
		}
		if len(nodes) < 2 {
			spof = SPOFUnknown
		}

		ex := []string{
			fmt.Sprintf("undirected_degree=%d_in_observed_graph", deg),
		}
		if bridgeSet[n.NodeNum] {
			ex = append(ex, "marked_bridge_critical_by_topology_heuristic")
		}
		if ni.RelayValueScore > 0 {
			ex = append(ex, fmt.Sprintf("relay_value_score=%.2f_from_packet_graph", ni.RelayValueScore))
		}

		profiles = append(profiles, NodeResilienceProfile{
			NodeNum:                    n.NodeNum,
			ShortName:                  n.ShortName,
			CriticalNodeScore:          round2(crit),
			RedundancyScore:            round2(redund),
			PartitionRiskScore:       round2(partRisk),
			ResilienceScore:            round2(resilience),
			RedundancyOpportunityScore: round2(redundOpp),
			SPOFClass:                  spof,
			RecoveryPriority:           0,
			Explanation:                ex,
		})
	}

	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i].CriticalNodeScore == profiles[j].CriticalNodeScore {
			return profiles[i].PartitionRiskScore > profiles[j].PartitionRiskScore
		}
		return profiles[i].CriticalNodeScore > profiles[j].CriticalNodeScore
	})
	for i := range profiles {
		profiles[i].RecoveryPriority = i + 1
	}

	frag := mi.Topology.FragmentationScore
	dep := mi.Topology.DependencyConcentrationScore
	meshRes := clamp01(0.5*(1-frag) + 0.5*(1-dep))
	partMesh := clamp01(frag*0.6 + dep*0.4)

	conf := ConfidenceAssessment{
		Level:    mi.Bootstrap.Confidence,
		Score:    coarseConfidenceScore(mi.Bootstrap.Confidence),
		MissingInputs: []string{
			"rf_propagation_model",
			"terrain_and_obstruction",
			"true_battery_uptime_per_node",
		},
		TopologyOnlyLimits: []string{
			"Resilience is inferred from packet-derived topology, not physical site survey.",
		},
	}

	var fragLines []string
	if frag > 0.35 {
		fragLines = append(fragLines, "Observed graph shows meaningful fragmentation between components.")
	}
	if dep > 0.45 {
		fragLines = append(fragLines, "Traffic or degree concentration suggests dependency on a small set of nodes.")
	}
	if len(ar.BridgeNodes) > 0 {
		fragLines = append(fragLines, fmt.Sprintf("Bridge-like nodes detected: %v — loss may partition the graph.", ar.BridgeNodes))
	}
	if len(fragLines) == 0 {
		fragLines = append(fragLines, "No extreme structural fragility flags in the current observed graph — still not proof of RF redundancy.")
	}

	nextMove := "Observe and gather history before large changes."
	if frag > 0.4 || len(ar.BridgeNodes) > 0 {
		nextMove = "Highest leverage is often adding an alternate path or improving uptime on a bridge-class node before expanding endpoints."
	} else if mi.Bootstrap.Viability == meshintel.ViabilityWeakBootstrap || mi.Bootstrap.Viability == meshintel.ViabilityIsolated {
		nextMove = "Establish mutual RF visibility (often one elevated stationary node) before adding more handhelds."
	} else if mi.RoutingPressure.OverRouterizationRiskScore.Score > 0.55 {
		nextMove = "Reduce forwarding pressure (roles/placement) before adding more routers."
	}

	summary := MeshResilienceSummary{
		ResilienceScore:      round2(meshRes),
		RedundancyScore:      round2(1 - dep),
		PartitionRiskScore:   round2(partMesh),
		FragilityExplanation: fragLines,
		NextBestMoveSummary:  nextMove,
		Confidence:           conf,
	}
	return summary, profiles
}

func buildAdjCounts(nodes []topology.Node, links []topology.Link) map[int64]int {
	adj := make(map[int64]int)
	for _, n := range nodes {
		adj[n.NodeNum] = 0
	}
	seen := make(map[string]bool)
	for _, l := range links {
		if l.SrcNodeNum == l.DstNodeNum {
			continue
		}
		a, b := l.SrcNodeNum, l.DstNodeNum
		if a > b {
			a, b = b, a
		}
		key := fmt.Sprintf("%d-%d", a, b)
		if seen[key] {
			continue
		}
		seen[key] = true
		if _, ok := adj[a]; ok {
			adj[a]++
		}
		if _, ok := adj[b]; ok {
			adj[b]++
		}
	}
	return adj
}

func coarseConfidenceScore(c meshintel.ConfidenceLevel) float64 {
	switch c {
	case meshintel.ConfidenceHigh:
		return 0.78
	case meshintel.ConfidenceMedium:
		return 0.55
	default:
		return 0.32
	}
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func round2(x float64) float64 {
	return float64(int(x*100+0.5)) / 100
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// NodeByNum finds a node in analysis result.
func NodeByNum(ar topology.AnalysisResult, nodeNum int64) (topology.Node, bool) {
	for _, n := range ar.ScoredNodes {
		if n.NodeNum == nodeNum {
			return n, true
		}
	}
	return topology.Node{}, false
}

// ExplainLimits returns standard honesty strings for responses.
func ExplainLimits() []string {
	return []string{
		"No RF coverage simulation — only packet-derived graph and message rollups.",
		"No terrain or obstruction model unless operator supplies notes (still advisory).",
		"Hidden RF paths not yet observed as packets are invisible to MEL.",
	}
}

// EvidenceModelString is the canonical evidence string for planning outputs.
func EvidenceModelString() string { return PlanningEvidenceModel }
