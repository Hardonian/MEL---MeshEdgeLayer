package meshintel

import (
	"crypto/sha256"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/topology"
)

const evidenceModel = "Derived from topology_links and nodes (packet relay/to_node fields), plus optional recent messages rollup. Not RF coverage proof. MEL does not modify Meshtastic routing."

// Compute builds a full assessment from topology analysis and message signals.
func Compute(cfg config.Config, ar topology.AnalysisResult, sig MessageSignals, transportConnected bool, now time.Time) Assessment {
	th := topology.StaleThresholdsFromConfig(cfg.Topology.NodeStaleMinutes, cfg.Topology.LinkStaleMinutes)
	if !cfg.Topology.Enabled {
		return emptyAssessment(now, ar.Snapshot.GraphHash, false, th)
	}

	nodes := ar.ScoredNodes
	links := ar.ScoredLinks
	adj := buildUndirectedAdj(nodes, links)
	comp := components(nodes, adj)
	largest := 0
	for _, c := range comp {
		if len(c) > largest {
			largest = len(c)
		}
	}
	peerCount := len(nodes)
	if peerCount == 0 {
		peerCount = 0
	}

	bootstrap := computeBootstrap(nodes, links, ar, comp, largest, len(comp), sig, transportConnected, th, now)
	topoMetrics, nodeIntel := computeTopologyIntel(nodes, links, adj, comp, largest, ar)
	routing := computeRoutingPressure(nodes, links, sig, transportConnected)
	proto := computeProtocolFit(bootstrap, topoMetrics, routing, sig, transportConnected)
	recs := rankRecommendations(bootstrap, topoMetrics, routing, proto, transportConnected, sig)

	h := sha256.New()
	fmt.Fprintf(h, "meshintel:%s:%d:%d", ar.Snapshot.GraphHash, len(nodes), len(links))
	sum := fmt.Sprintf("%x", h.Sum(nil))[:16]
	aid := fmt.Sprintf("mi-%s-%s", now.UTC().Format("20060102-150405"), sum[:8])

	return Assessment{
		AssessmentID:    aid,
		ComputedAt:      now.UTC().Format(time.RFC3339),
		GraphHash:       ar.Snapshot.GraphHash,
		TopologyEnabled: true,
		MessageSignals:  sig,
		Bootstrap:       bootstrap,
		Topology:        topoMetrics,
		NodeIntel:       nodeIntel,
		RoutingPressure: routing,
		ProtocolFit:     proto,
		Recommendations: recs,
		EvidenceModel:   evidenceModel,
	}
}

func emptyAssessment(now time.Time, graphHash string, enabled bool, th topology.StaleThresholds) Assessment {
	return Assessment{
		AssessmentID:    fmt.Sprintf("mi-empty-%s", now.UTC().Format("20060102-150405")),
		ComputedAt:      now.UTC().Format(time.RFC3339),
		GraphHash:       graphHash,
		TopologyEnabled: enabled,
		MessageSignals:  MessageSignals{},
		Bootstrap: BootstrapAssessment{
			LoneWolfScore:           1,
			BootstrapReadinessScore: 0,
			Viability:               ViabilityIsolated,
			Confidence:              ConfidenceLow,
			Explanation: BootstrapExplanation{
				EvidenceUsed:         []string{"topology_disabled_or_no_nodes"},
				WeakensViability:     []string{"no_observed_mesh_evidence"},
				StrengthensViability: nil,
				Missing:              []string{"ingested_nodes", "topology_links"},
				TopNextAction:        "Enable topology in config and connect a transport so MEL can observe packets.",
			},
			ObservationWindowHint: formatWindowHint(th),
		},
		Topology: MeshTopologyMetrics{
			ClusterShape: ShapeIsolatedSingle,
			Explanation: TopologyExplanation{
				EvidenceStrength: "none",
				Limits:           []string{"No topology data to analyze."},
			},
		},
		RoutingPressure: routingPressureInsufficient("no topology graph"),
		ProtocolFit: ProtocolFitAssessment{
			FitClass:          ProtocolFitInsufficientEvidence,
			ArchitectureClass: ArchObserveOnly,
			Confidence:        ConfidenceLow,
			Explanation: ProtocolFitExplanation{
				ObservedFacts:        []string{"No nodes or topology disabled."},
				CheaperStepsFirst:    []string{"Connect transport and confirm packet ingest."},
				PlacementBeforeProto: "Insufficient evidence to compare placement vs protocol.",
				EvidenceAdequacy:     "none",
				Limits:               []string{"Cannot assess protocol fit without observed mesh."},
			},
		},
		Recommendations: []MeshRecommendation{
			{
				Rank: 1, Class: RecInvestigateTransport, Title: "Establish transport and packet visibility",
				Severity: "high", Urgency: "soon",
				EvidenceSummary: []string{"No topology evidence in MEL yet."},
				ExpectedBenefit: "Enables all downstream bootstrap and topology assessments.",
				DownsideRisk:    "None; observation-only.",
				Confidence:      0.95,
			},
		},
		EvidenceModel: evidenceModel,
	}
}

func formatWindowHint(th topology.StaleThresholds) string {
	return fmt.Sprintf("node_stale=%s link_stale=%s", th.NodeStaleDuration, th.LinkStaleDuration)
}

func buildUndirectedAdj(nodes []topology.Node, links []topology.Link) map[int64]map[int64]bool {
	adj := make(map[int64]map[int64]bool)
	for _, n := range nodes {
		adj[n.NodeNum] = make(map[int64]bool)
	}
	for _, l := range links {
		if adj[l.SrcNodeNum] == nil {
			adj[l.SrcNodeNum] = make(map[int64]bool)
		}
		if adj[l.DstNodeNum] == nil {
			adj[l.DstNodeNum] = make(map[int64]bool)
		}
		adj[l.SrcNodeNum][l.DstNodeNum] = true
		if !l.Directional {
			adj[l.DstNodeNum][l.SrcNodeNum] = true
		} else {
			// Packet-derived directed edges still imply visibility both ends for "who heard whom" style graph
			adj[l.DstNodeNum][l.SrcNodeNum] = true
		}
	}
	return adj
}

func components(nodes []topology.Node, adj map[int64]map[int64]bool) [][]int64 {
	seen := make(map[int64]bool)
	var out [][]int64
	for _, n := range nodes {
		if seen[n.NodeNum] {
			continue
		}
		var comp []int64
		q := []int64{n.NodeNum}
		seen[n.NodeNum] = true
		for len(q) > 0 {
			cur := q[0]
			q = q[1:]
			comp = append(comp, cur)
			for nb := range adj[cur] {
				if !seen[nb] {
					seen[nb] = true
					q = append(q, nb)
				}
			}
		}
		out = append(out, comp)
	}
	return out
}

func computeBootstrap(
	nodes []topology.Node,
	links []topology.Link,
	ar topology.AnalysisResult,
	comp [][]int64,
	largestComp, compCount int,
	sig MessageSignals,
	transportConnected bool,
	th topology.StaleThresholds,
	now time.Time,
) BootstrapAssessment {
	ex := BootstrapExplanation{EvidenceUsed: []string{}}
	windowHint := formatWindowHint(th)
	ex.EvidenceUsed = append(ex.EvidenceUsed,
		fmt.Sprintf("nodes=%d topology_links=%d", len(nodes), len(links)),
		fmt.Sprintf("largest_connected_component=%d component_count=%d", largestComp, compCount),
		sig.WindowDescription,
	)

	isolated := len(ar.IsolatedNodes)
	nonIsolated := len(nodes) - isolated
	edgePerNode := 0.0
	if len(nodes) > 0 {
		edgePerNode = float64(len(links)) / float64(len(nodes))
	}

	// Lone wolf: 1.0 = maximally alone / unobserved mesh
	lone := 0.0
	if len(nodes) <= 1 {
		lone = 1.0
	} else if largestComp <= 1 {
		lone = 0.95
	} else if largestComp == 2 && len(nodes) == 2 {
		lone = 0.55
	} else if float64(isolated)/float64(len(nodes)) > 0.5 {
		lone = 0.75
	} else if largestComp < len(nodes) && compCount > 1 {
		lone = 0.45
	} else if edgePerNode < 0.8 && largestComp < 4 {
		lone = 0.5
	} else {
		lone = math.Max(0, 0.35-(edgePerNode*0.08))
	}

	// Readiness combines connectivity, recency, transport
	readiness := 0.0
	if nonIsolated > 0 {
		readiness += 0.25
	}
	if largestComp >= 3 {
		readiness += 0.2
	} else if largestComp == 2 {
		readiness += 0.12
	}
	if edgePerNode >= 1.2 {
		readiness += 0.2
	} else if edgePerNode >= 0.5 {
		readiness += 0.1
	}
	freshRatio := freshNodeRatio(nodes, th, now)
	readiness += 0.2 * freshRatio
	if transportConnected {
		readiness += 0.15
	}
	readiness = clamp01(readiness)

	viability := ViabilityEmergingCluster
	switch {
	case len(nodes) == 0:
		viability = ViabilityIsolated
		lone, readiness = 1, 0
	case largestComp <= 1:
		viability = ViabilityIsolated
	case isInfrastructureAnchored(nodes, links):
		viability = ViabilityInfrastructureDependent
	case largestComp == 2 && edgePerNode < 0.6:
		viability = ViabilityWeakBootstrap
	case !transportConnected && len(nodes) > 0:
		viability = ViabilityUnstableIntermittent
		ex.WeakensViability = append(ex.WeakensViability, "no_transport_connected_observation_may_stale")
	case freshRatio < 0.35 && len(nodes) >= 2:
		viability = ViabilityUnstableIntermittent
		ex.WeakensViability = append(ex.WeakensViability, "many_nodes_stale_or_stale_window_exceeded")
	case largestComp >= 4 && edgePerNode >= 1.0 && freshRatio >= 0.5:
		viability = ViabilityViableLocalMesh
	case largestComp >= 3:
		viability = ViabilityEmergingCluster
	default:
		viability = ViabilityWeakBootstrap
	}

	if isolated > 0 {
		ex.WeakensViability = append(ex.WeakensViability, fmt.Sprintf("isolated_nodes=%d_in_observed_graph", isolated))
	}
	if largestComp < len(nodes) {
		ex.WeakensViability = append(ex.WeakensViability, "partitioned_components_observed")
	}
	if nonIsolated > 1 && edgePerNode >= 1 {
		ex.StrengthensViability = append(ex.StrengthensViability, "multiple_edges_per_node_suggest_redundancy_or_dense_traffic_paths")
	}
	if sig.TotalMessages > 20 {
		ex.StrengthensViability = append(ex.StrengthensViability, "recent_message_traffic_observed")
	}
	ex.Missing = []string{
		"terrain_model",
		"true_rf_neighbors_outside_packet_path",
		"battery_duty_cycle_per_node_unless_telemetry_ingested",
	}

	conf := ConfidenceMedium
	if len(nodes) < 2 || sig.TotalMessages < 5 {
		conf = ConfidenceLow
	}
	if len(nodes) >= 5 && sig.TotalMessages >= 50 && freshRatio > 0.6 {
		conf = ConfidenceHigh
	}

	ex.TopNextAction = topBootstrapAction(viability, lone, transportConnected, largestComp)

	return BootstrapAssessment{
		LoneWolfScore:           round2(lone),
		BootstrapReadinessScore: round2(readiness),
		Viability:               viability,
		Explanation:               ex,
		Confidence:                conf,
		PeerCountObserved:         len(nodes),
		UniquePeersOverWindow:     len(nodes),
		LargestComponentSize:      largestComp,
		ComponentCount:            compCount,
		ObservationWindowHint:     windowHint,
	}
}

func topBootstrapAction(v LocalMeshViabilityClassification, lone float64, transportOK bool, largest int) string {
	if !transportOK {
		return "Restore or connect a transport so MEL continues to observe mesh traffic before changing deployment."
	}
	switch v {
	case ViabilityIsolated, ViabilityWeakBootstrap:
		if lone > 0.7 {
			return "Add one well-placed always-on node (elevated if possible) or move an existing node to improve mutual visibility — handheld density alone rarely fixes lone-wolf starts."
		}
		return "Confirm at least two nodes can hear each other on RF (not only via MQTT); add a stationary relay path if packets only appear via broker."
	case ViabilityInfrastructureDependent:
		return "Reduce dependence on a single broker-visible path: add a local RF relay or second ingress so the mesh is not a single spoke into infrastructure."
	case ViabilityUnstableIntermittent:
		return "Stabilize power and placement first; wait for sustained observations across the stale window before adding hardware."
	case ViabilityViableLocalMesh:
		return "Keep observing; tune roles and placement before adding more nodes."
	default:
		return "Continue observing while improving one stationary elevated link — emerging clusters usually need backbone before more endpoints."
	}
}

func isInfrastructureAnchored(nodes []topology.Node, links []topology.Link) bool {
	if len(nodes) < 2 {
		return false
	}
	brokerish := 0
	for _, n := range nodes {
		if strings.TrimSpace(n.LastBrokerSeenAt) != "" && strings.TrimSpace(n.LastDirectSeenAt) == "" {
			brokerish++
		}
	}
	relayHeavy := 0
	for _, l := range links {
		if l.RelayDependent {
			relayHeavy++
		}
	}
	allRelay := len(links) > 0 && relayHeavy == len(links)
	return brokerish*2 >= len(nodes) || (len(links) > 0 && float64(relayHeavy)/float64(len(links)) > 0.55) || allRelay
}

func freshNodeRatio(nodes []topology.Node, th topology.StaleThresholds, now time.Time) float64 {
	if len(nodes) == 0 {
		return 0
	}
	fresh := 0
	for _, n := range nodes {
		t := parseTime(n.LastSeenAt)
		if !t.IsZero() && now.Sub(t) <= th.NodeStaleDuration && !n.Stale {
			fresh++
		}
	}
	return float64(fresh) / float64(len(nodes))
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(s))
	if err != nil {
		return time.Time{}
	}
	return t
}

func computeTopologyIntel(
	nodes []topology.Node,
	links []topology.Link,
	adj map[int64]map[int64]bool,
	comp [][]int64,
	largest int,
	ar topology.AnalysisResult,
) (MeshTopologyMetrics, []NodeTopologyIntel) {
	ex := TopologyExplanation{
		Limits: []string{
			"Graph is packet-derived; high degree can mean forwarding visibility, not geographic coverage.",
		},
	}
	if len(nodes) < 3 {
		ex.EvidenceStrength = "weak"
		ex.Limits = append(ex.Limits, "Few nodes — shape classification is tentative.")
	} else if len(links) < len(nodes) {
		ex.EvidenceStrength = "moderate"
	} else {
		ex.EvidenceStrength = "moderate_strong"
	}

	// Fragmentation
	frag := 0.0
	if len(nodes) > 0 {
		frag = 1.0 - float64(largest)/float64(len(nodes))
	}

	bridgeSet := make(map[int64]bool)
	for _, b := range ar.BridgeNodes {
		bridgeSet[b] = true
	}

	// Degrees
	inDeg := make(map[int64]int)
	outDeg := make(map[int64]int)
	undirectedDeg := make(map[int64]int)
	for _, n := range nodes {
		undirectedDeg[n.NodeNum] = len(adj[n.NodeNum])
	}
	for _, l := range links {
		outDeg[l.SrcNodeNum]++
		inDeg[l.DstNodeNum]++
	}

	// Hub concentration (dependency): Herfindahl on undirected degrees
	var degList []int
	sumDeg := 0
	for _, n := range nodes {
		d := undirectedDeg[n.NodeNum]
		degList = append(degList, d)
		sumDeg += d
	}
	hhi := 0.0
	if sumDeg > 0 {
		for _, d := range degList {
			p := float64(d) / float64(sumDeg)
			hhi += p * p
		}
	}
	depConc := clamp01((hhi - 1/float64(max(1, len(nodes)))) / 0.75)

	// Shape classification
	shape := classifyShape(len(nodes), len(links), largest, len(comp), frag, hhi, bridgeSet, undirectedDeg)

	// Infrastructure leverage score: higher => stationary/elevated infra likely high leverage
	infra := 0.35
	if shape == ShapeSparseCluster || shape == ShapeIsolatedSingle {
		infra += 0.35
	}
	if frag > 0.25 {
		infra += 0.15
	}
	if isInfrastructureAnchored(nodes, links) {
		infra += 0.1
	}
	infra = clamp01(infra)

	ex.TopContributingFactors = append(ex.TopContributingFactors,
		fmt.Sprintf("largest_component=%d_of_%d_nodes", largest, len(nodes)),
		fmt.Sprintf("fragmentation_score=%.2f", frag),
	)
	if len(bridgeSet) > 0 {
		ex.RiskyStructure = append(ex.RiskyStructure, "bridge_critical_nodes_present_in_observed_graph")
	}
	if depConc > 0.45 {
		ex.RiskyStructure = append(ex.RiskyStructure, "traffic_visibility_concentrated_on_few_nodes")
	}
	if shape == ShapeCorridor {
		ex.WeakPoints = append(ex.WeakPoints, "corridor_like_paths_break_if_endpoints_drop")
	}
	ex.LeverageOpportunities = append(ex.LeverageOpportunities,
		"If scores show sparse or fragmented graph, one elevated always-on relay often beats more handhelds.",
	)

	nodeIntel := make([]NodeTopologyIntel, 0, len(nodes))
	for _, n := range nodes {
		ud := undirectedDeg[n.NodeNum]
		cov := clamp01(float64(ud) / 8.0)
		relay := clamp01(float64(inDeg[n.NodeNum]+outDeg[n.NodeNum]) / 10.0)
		if n.HealthState == topology.HealthStale {
			cov *= 0.5
			relay *= 0.5
		}
		place := n.HealthScore * (0.5 + 0.5*cov)
		notes := []string{}
		if bridgeSet[n.NodeNum] {
			notes = append(notes, "appears_bridge_critical_in_observed_graph")
		}
		if ud == 0 {
			notes = append(notes, "no_links_in_observed_graph")
		}
		nodeIntel = append(nodeIntel, NodeTopologyIntel{
			NodeNum:                   n.NodeNum,
			CoverageContributionScore: round2(cov),
			RelayValueScore:           round2(relay),
			PlacementQualityScore:     round2(clamp01(place)),
			InDegree:                  inDeg[n.NodeNum],
			OutDegree:                 outDeg[n.NodeNum],
			UndirectedDegree:          ud,
			IsBridgeCritical:          bridgeSet[n.NodeNum],
			Role:                      n.Role,
			Notes:                     notes,
		})
	}
	sort.Slice(nodeIntel, func(i, j int) bool { return nodeIntel[i].NodeNum < nodeIntel[j].NodeNum })

	metrics := MeshTopologyMetrics{
		FragmentationScore:             round2(frag),
		DependencyConcentrationScore:   round2(depConc),
		InfrastructureLeverageScore:    round2(infra),
		ClusterShape:                   shape,
		Explanation:                    ex,
	}
	return metrics, nodeIntel
}

func classifyShape(nodeCount, edgeCount, largest, compCount int, frag, hhi float64, bridges map[int64]bool, deg map[int64]int) ClusterShapeClassification {
	if nodeCount <= 1 {
		return ShapeIsolatedSingle
	}
	if compCount > 1 && frag > 0.4 {
		return ShapePartitioned
	}
	if len(bridges) > 0 && largest >= 3 {
		return ShapeFragileBridge
	}
	maxDeg := 0
	for _, d := range deg {
		if d > maxDeg {
			maxDeg = d
		}
	}
	avgDeg := 0.0
	if nodeCount > 0 {
		s := 0
		for _, d := range deg {
			s += d
		}
		avgDeg = float64(s) / float64(nodeCount)
	}
	if maxDeg >= 3 && hhi > 0.35 && avgDeg < 2.5 {
		return ShapeHubAndSpoke
	}
	density := 0.0
	if nodeCount > 1 {
		maxE := nodeCount * (nodeCount - 1) / 2
		density = float64(edgeCount) / float64(maxE)
	}
	if density > 0.35 && avgDeg >= 2.5 {
		return ShapeDenseLocal
	}
	if avgDeg < 1.4 && edgeCount <= nodeCount+2 {
		return ShapeCorridor
	}
	if isInfrastructureAnchoredShape(nodeCount, edgeCount) {
		return ShapeInfrastructureAnchored
	}
	if nodeCount >= 2 && edgeCount <= nodeCount {
		return ShapeSparseCluster
	}
	return ShapeSparseCluster
}

func isInfrastructureAnchoredShape(nodeCount, edgeCount int) bool {
	// heuristic: few edges relative to nodes often correlates with broker-only vis
	return nodeCount >= 3 && edgeCount > 0 && float64(edgeCount)/float64(nodeCount) < 0.7
}

func computeRoutingPressure(nodes []topology.Node, links []topology.Link, sig MessageSignals, transportOK bool) RoutingPressureBundle {
	insuf := "insufficient_message_history_for_airtime_collision_inference"
	collision := ScoredMetric{
		Name: "collision_risk", Score: 0.25, Scale: "0_1", Basis: "suspected_proxy",
		Evidence: []string{insuf, "no_snr_histogram_in_this_assessment"},
		Confidence: ConfidenceLow, IsSuspected: true, Uncertainty: "Use spectrum tools for real collision data.",
	}
	if sig.TotalMessages > 30 {
		collision.Score = clamp01(0.15 + sig.DuplicateRelayHotspot*0.35 + sig.RebroadcastPathProxy*0.25 + sig.RelayMaxShare*0.15)
		collision.Evidence = []string{
			fmt.Sprintf("messages_in_window=%d", sig.TotalMessages),
			fmt.Sprintf("duplicate_relay_hotspot_ratio=%.3f", sig.DuplicateRelayHotspot),
			fmt.Sprintf("rebroadcast_path_proxy=%.3f", sig.RebroadcastPathProxy),
			fmt.Sprintf("relay_max_share=%.3f distinct_relays=%d", sig.RelayMaxShare, sig.DistinctRelayNodes),
		}
		if len(sig.HopBuckets) > 0 {
			collision.Evidence = append(collision.Evidence, "hop_buckets="+topHistogramSummary(sig.HopBuckets, 4))
		}
		collision.Confidence = ConfidenceMedium
	}

	dup := ScoredMetric{
		Name: "duplicate_forward_pressure", Score: clamp01(sig.DuplicateRelayHotspot*0.5 + sig.RelayMaxShare*0.5),
		Scale: "0_1", Basis: "observed_relay_field_repeat_rate",
		Evidence: []string{
			fmt.Sprintf("relay_repeat_proxy=%.3f", sig.DuplicateRelayHotspot),
			fmt.Sprintf("relay_max_share=%.3f distinct_relays=%d", sig.RelayMaxShare, sig.DistinctRelayNodes),
		},
		Confidence: ConfidenceMedium, IsSuspected: sig.TotalMessages < 15,
	}
	if sig.TotalMessages < 10 {
		dup.Score = 0.2
		dup.Confidence = ConfidenceLow
		dup.Uncertainty = "Need more messages to estimate rebroadcast pressure."
	}

	hopStress := ScoredMetric{
		Name: "hop_budget_stress", Score: 0.2, Scale: "0_1", Basis: "hop_limit_distribution",
		Confidence: ConfidenceLow, IsSuspected: true,
	}
	if sig.MessagesWithHop > 0 {
		hopStress.Score = clamp01(sig.AvgHopLimit / 7.0)
		hopStress.Evidence = []string{
			fmt.Sprintf("avg_hop_limit=%.2f max=%d", sig.AvgHopLimit, sig.MaxHopLimit),
			fmt.Sprintf("messages_with_hop=%d", sig.MessagesWithHop),
		}
		if len(sig.HopBuckets) > 0 {
			hopStress.Evidence = append(hopStress.Evidence, "hop_distribution="+topHistogramSummary(sig.HopBuckets, 6))
		}
		hopStress.Confidence = ConfidenceMedium
		if sig.MaxHopLimit >= 6 {
			hopStress.Evidence = append(hopStress.Evidence, "high_hop_limit_observed_packets_may_have_traveled_many_hops")
		}
	}

	bcast := ScoredMetric{
		Name: "broadcast_domain_pressure", Scale: "0_1", Basis: "fanout_proxy",
		Score: clamp01(float64(len(links)) / float64(max(1, len(nodes))*3)),
		Evidence: []string{fmt.Sprintf("observed_edges=%d nodes=%d", len(links), len(nodes))},
		Confidence: ConfidenceLow, IsSuspected: true,
		Uncertainty: "Cannot see channel utilization; edge count is a weak proxy.",
	}

	routerRoles := countRouterLike(nodes)
	overRouter := ScoredMetric{
		Name: "over_routerization_risk", Scale: "0_1",
		Basis: "role_label_heuristic",
		Score: clamp01(float64(routerRoles) / float64(max(1, len(nodes))) * 1.4),
		Evidence: []string{fmt.Sprintf("router_like_roles=%d_of_%d_nodes", routerRoles, len(nodes))},
		Confidence: ConfidenceLow, IsSuspected: true,
		Uncertainty: "Role names are operator-supplied; verify on devices.",
	}

	weakOnward := ScoredMetric{
		Name: "weak_onward_propagation", Scale: "0_1", Basis: "relay_to_edge_ratio",
		Score: 0.35, Confidence: ConfidenceMedium, IsSuspected: true,
	}
	relayEdges := 0
	for _, l := range links {
		if l.RelayDependent {
			relayEdges++
		}
	}
	if len(links) > 0 {
		weakOnward.Score = clamp01(1.0 - float64(relayEdges)/float64(len(links)))
		weakOnward.Evidence = []string{fmt.Sprintf("relay_dependent_edges=%d_of_%d", relayEdges, len(links))}
	}

	sink := ScoredMetric{
		Name: "suspected_packet_sink", Scale: "0_1", Basis: "in_degree_vs_out_degree",
		Confidence: ConfidenceLow, IsSuspected: true,
	}
	sinkScore := 0.0
	inSum, outSum := 0, 0
	inDeg := map[int64]int{}
	outDeg := map[int64]int{}
	for _, l := range links {
		outDeg[l.SrcNodeNum]++
		inDeg[l.DstNodeNum]++
	}
	for _, n := range nodes {
		inSum += inDeg[n.NodeNum]
		outSum += outDeg[n.NodeNum]
	}
	if inSum+outSum > 0 {
		sinkScore = clamp01(float64(inSum) / float64(inSum+outSum+1) - 0.35)
	}
	sink.Score = sinkScore
	sink.Evidence = []string{fmt.Sprintf("aggregate_in_edges=%d out_edges=%d", inSum, outSum)}

	routeConc := ScoredMetric{
		Name: "route_concentration_risk", Scale: "0_1", Basis: "graph_concentration",
		Score: 0.3, Confidence: ConfidenceMedium,
	}
	// Reuse simple HHI on undirected degree
	adj := buildUndirectedAdj(nodes, links)
	und := make(map[int64]int)
	for _, n := range nodes {
		und[n.NodeNum] = len(adj[n.NodeNum])
	}
	sum := 0
	for _, d := range und {
		sum += d
	}
	if sum > 0 {
		hhi := 0.0
		for _, d := range und {
			p := float64(d) / float64(sum)
			hhi += p * p
		}
		routeConc.Score = clamp01(hhi)
		routeConc.Evidence = []string{fmt.Sprintf("degree_concentration_hhi=%.3f", hhi)}
	}

	summary := []string{
		"MEL does not fix Meshtastic routing; these are observability proxies only.",
	}
	if !transportOK {
		summary = append(summary, "Transport disconnected — routing diagnostics may be stale.")
	}
	if weakOnward.Score > 0.55 {
		summary = append(summary, "Weak onward propagation in observed graph — may be sparse density or placement, not only routing.")
	}
	if dup.Score > 0.5 {
		summary = append(summary, "Repeated relay fields suggest possible duplicate-forwarding hotspots (suspected).")
	}
	if sig.RebroadcastPathProxy > 0.55 && sig.TotalMessages > 40 {
		summary = append(summary, "High share of packets carry relay_node — rebroadcast path activity is visible in ingest (not spectrum proof).")
	}
	if len(sig.PortnumBuckets) > 0 && sig.TotalMessages > 20 {
		summary = append(summary, "Top portnums by volume: "+topHistogramSummary(sig.PortnumBuckets, 5)+".")
	}

	return RoutingPressureBundle{
		CollisionRiskScore:            collision,
		DuplicateForwardPressureScore: dup,
		HopBudgetStressScore:          hopStress,
		BroadcastDomainPressureScore:  bcast,
		OverRouterizationRiskScore:    overRouter,
		WeakOnwardPropagationScore:    weakOnward,
		SuspectedPacketSinkScore:      sink,
		RouteConcentrationRiskScore:   routeConc,
		SummaryLines:                  summary,
	}
}

func routingPressureInsufficient(reason string) RoutingPressureBundle {
	b := routingPressureEmpty()
	b.SummaryLines = append(b.SummaryLines, reason)
	return b
}

func routingPressureEmpty() RoutingPressureBundle {
	mk := func(name string) ScoredMetric {
		return ScoredMetric{
			Name: name, Score: 0, Scale: "0_1", Basis: "insufficient_evidence",
			Confidence: ConfidenceLow, IsSuspected: true,
			Uncertainty: "insufficient_evidence_for_congestion_claim",
		}
	}
	return RoutingPressureBundle{
		CollisionRiskScore:            mk("collision_risk"),
		DuplicateForwardPressureScore: mk("duplicate_forward_pressure"),
		HopBudgetStressScore:          mk("hop_budget_stress"),
		BroadcastDomainPressureScore:  mk("broadcast_domain_pressure"),
		OverRouterizationRiskScore:    mk("over_routerization_risk"),
		WeakOnwardPropagationScore:    mk("weak_onward_propagation"),
		SuspectedPacketSinkScore:      mk("suspected_packet_sink"),
		RouteConcentrationRiskScore:   mk("route_concentration_risk"),
		SummaryLines:                  []string{"Insufficient data for routing pressure conclusions."},
	}
}

func countRouterLike(nodes []topology.Node) int {
	n := 0
	for _, node := range nodes {
		r := strings.ToLower(strings.TrimSpace(node.Role))
		if strings.Contains(r, "router") || strings.Contains(r, "repeater") {
			n++
		}
	}
	return n
}

func computeProtocolFit(
	bootstrap BootstrapAssessment,
	topo MeshTopologyMetrics,
	routing RoutingPressureBundle,
	sig MessageSignals,
	transportOK bool,
) ProtocolFitAssessment {
	ex := ProtocolFitExplanation{
		CheaperStepsFirst: []string{
			"Improve placement and stationary backbone before evaluating alternate protocols.",
			"Reduce unnecessary router/repeater roles if over-forwarding is suspected.",
		},
		PlacementBeforeProto: "Placement and role correction should precede protocol replacement decisions.",
		Limits: []string{
			"MEL does not measure spectrum occupancy or firmware queue drops.",
		},
	}
	ex.ObservedFacts = append(ex.ObservedFacts,
		fmt.Sprintf("viability=%s", bootstrap.Viability),
		fmt.Sprintf("cluster_shape=%s", topo.ClusterShape),
		fmt.Sprintf("messages_in_window=%d", sig.TotalMessages),
	)

	fit := ProtocolFitGood
	arch := ArchPlacementRolesFirst
	conf := ConfidenceMedium
	primary := ""

	if bootstrap.PeerCountObserved < 2 {
		fit = ProtocolFitInsufficientEvidence
		arch = ArchObserveOnly
		conf = ConfidenceLow
		ex.EvidenceAdequacy = "too_few_nodes"
		return ProtocolFitAssessment{FitClass: fit, ArchitectureClass: arch, Explanation: ex, Confidence: conf}
	}

	if !transportOK {
		fit = ProtocolFitInsufficientEvidence
		arch = ArchFixTransport
		primary = "transport"
		ex.EvidenceAdequacy = "transport_disconnected"
		conf = ConfidenceLow
		return ProtocolFitAssessment{FitClass: fit, ArchitectureClass: arch, Explanation: ex, Confidence: conf, PrimaryLimitingFactor: primary}
	}

	if bootstrap.Viability == ViabilityIsolated || bootstrap.Viability == ViabilityWeakBootstrap {
		fit = ProtocolFitGood
		arch = ArchPlacementRolesFirst
		primary = "density_or_placement"
		ex.ObservedFacts = append(ex.ObservedFacts, "sparse_or_isolated_observed_graph")
	}

	if topo.FragmentationScore > 0.45 {
		fit = ProtocolFitAcceptableCaveats
		arch = ArchAddStationaryInfra
		primary = "fragmentation"
	}

	if routing.DuplicateForwardPressureScore.Score > 0.55 && routing.DuplicateForwardPressureScore.Confidence != ConfidenceLow {
		fit = ProtocolFitStressedDensity
		arch = ArchReduceForwardingPressure
		primary = "forwarding_pressure_suspected"
	}

	if routing.OverRouterizationRiskScore.Score > 0.55 {
		fit = ProtocolFitRolePlacementFirst
		primary = "role_misuse_suspected"
	}

	if topo.ClusterShape == ShapePartitioned && topo.FragmentationScore > 0.35 {
		fit = ProtocolFitSegmentBridge
		arch = ArchSegmentOrBridge
		primary = "partitioning"
	}

	if bootstrap.Viability == ViabilityInfrastructureDependent && topo.InfrastructureLeverageScore > 0.55 {
		fit = ProtocolFitStructuredInfra
		arch = ArchAddStationaryInfra
		primary = "infrastructure_dependency"
	}

	if routing.HopBudgetStressScore.Score > 0.65 && sig.MessagesWithHop > 20 {
		fit = ProtocolFitStressedDensity
		primary = "hop_budget_pressure_suspected"
	}

	if sig.TotalMessages < 15 {
		fit = ProtocolFitInsufficientEvidence
		arch = ArchObserveOnly
		conf = ConfidenceLow
		ex.EvidenceAdequacy = "low_message_count"
	} else {
		ex.EvidenceAdequacy = "moderate_packet_history"
	}

	if fit == ProtocolFitGood && primary == "" {
		primary = "none_identified"
	}

	return ProtocolFitAssessment{
		FitClass:              fit,
		ArchitectureClass:     arch,
		Explanation:           ex,
		Confidence:            conf,
		PrimaryLimitingFactor: primary,
	}
}

func rankRecommendations(
	bootstrap BootstrapAssessment,
	topo MeshTopologyMetrics,
	routing RoutingPressureBundle,
	proto ProtocolFitAssessment,
	transportOK bool,
	sig MessageSignals,
) []MeshRecommendation {
	var recs []MeshRecommendation
	add := func(class RecommendationClass, title, sev, urg string, conf float64, ev []string, ben, risk string, why string, hints []DeploymentClassHint) {
		recs = append(recs, MeshRecommendation{
			Rank:            len(recs) + 1,
			Class:           class,
			Title:           title,
			Severity:        sev,
			Urgency:         urg,
			EvidenceSummary: ev,
			ExpectedBenefit: ben,
			DownsideRisk:    risk,
			Confidence:      conf,
			WhyNotOthers:    why,
			DeploymentHints: hints,
		})
	}

	if !transportOK {
		add(RecInvestigateTransport, "Stabilize transport before changing mesh layout", "high", "soon", 0.92,
			[]string{"No live transport connected — observations will go stale."},
			"Accurate bootstrap and topology assessments.", "Low — connectivity check only.", "",
			nil)
	}

	switch bootstrap.Viability {
	case ViabilityIsolated, ViabilityWeakBootstrap:
		add(RecAddElevatedStationary, "Add one elevated, always-on node before more handhelds", "high", "soon", 0.78,
			[]string{fmt.Sprintf("viability=%s lone_wolf_score=%.2f", bootstrap.Viability, bootstrap.LoneWolfScore)},
			"Improves mutual visibility and breaks lone-wolf starts.", "Placement and power planning effort.",
			"Extra handhelds rarely fix zero-backbone situations.",
			[]DeploymentClassHint{DeploySolarRelayOutdoor, DeployAtticRooftop, DeployBaseWindowsill})
		add(RecImprovePlacement, "Improve placement of existing nodes", "medium", "when_convenient", 0.65,
			[]string{"Sparse edges in packet-derived graph."},
			"May yield more links without new hardware.", "Physical access required.", "", nil)
	case ViabilityInfrastructureDependent:
		add(RecAddAlwaysOnBase, "Add local RF backbone to reduce broker-only dependency", "medium", "soon", 0.7,
			[]string{"Many nodes visible primarily via broker path or relay-heavy edges."},
			"Reduces single-path dependence on infrastructure.", "More hardware or site access.", "", nil)
	}

	if topo.FragmentationScore > 0.35 {
		add(RecAddElevatedStationary, "Address fragmentation with a bridging stationary node", "medium", "when_convenient", 0.62,
			[]string{fmt.Sprintf("fragmentation_score=%.2f", topo.FragmentationScore)},
			"Connects observed components.", "Wrong placement wastes a fixed node.", "", []DeploymentClassHint{DeploySolarRelayOutdoor})
	}

	if routing.OverRouterizationRiskScore.Score > 0.5 {
		add(RecReduceRouterRoleUsage, "Reduce router/repeater roles unless observably valuable", "medium", "when_convenient", 0.55,
			[]string{"Multiple nodes advertise router-like roles in MEL records."},
			"Lowers suspected duplicate forwarding pressure.", "May shorten range for some endpoints if over-corrected.",
			"Do not add more routers while this risk is elevated.", nil)
	}

	if proto.FitClass == ProtocolFitSegmentBridge {
		add(RecSegmentOrBridge, "Consider segmentation or explicit bridging between partitions", "medium", "deferred", 0.5,
			[]string{"Multiple disconnected components in observed graph."},
			"Cleaner operations per segment.", "Operational complexity.", "", nil)
	}

	if sig.TotalMessages < 20 && transportOK {
		add(RecGatherMoreHistory, "Gather more observation history before major architecture changes", "low", "deferred", 0.8,
			[]string{fmt.Sprintf("only_%d_messages_in_window", sig.TotalMessages)},
			"Avoids costly changes on thin evidence.", "Delays action.", "Premature hardware buys may not match real topology.", nil)
	}

	if len(recs) == 0 {
		add(RecKeepObserve, "Keep current topology and continue observing", "low", "deferred", 0.7,
			[]string{"No strong negative signals in current evidence."},
			"Confirms stability over time.", "None.", "", nil)
	}

	// Rank: sort by severity order then confidence
	sort.SliceStable(recs, func(i, j int) bool {
		si, sj := sevOrder(recs[i].Severity), sevOrder(recs[j].Severity)
		if si != sj {
			return si < sj
		}
		return recs[i].Confidence > recs[j].Confidence
	})
	for i := range recs {
		recs[i].Rank = i + 1
	}
	return recs
}

func sevOrder(s string) int {
	switch s {
	case "high":
		return 0
	case "medium":
		return 1
	case "low":
		return 2
	default:
		return 3
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
	return math.Round(x*100) / 100
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func topHistogramSummary(buckets []HistogramBucket, maxN int) string {
	if len(buckets) == 0 || maxN <= 0 {
		return ""
	}
	n := len(buckets)
	if n > maxN {
		n = maxN
	}
	var parts []string
	for i := 0; i < n; i++ {
		parts = append(parts, fmt.Sprintf("%s=%d", buckets[i].Key, buckets[i].Count))
	}
	return strings.Join(parts, ",")
}

// ComputeLive loads nodes and links from the topology store, rolls up recent messages,
// and returns a fresh assessment (used by API/CLI when the background worker has not run yet).
func ComputeLive(cfg config.Config, d *db.DB, store *topology.Store, transportConnected bool, now time.Time) Assessment {
	th := topology.StaleThresholdsFromConfig(cfg.Topology.NodeStaleMinutes, cfg.Topology.LinkStaleMinutes)
	if store == nil || d == nil {
		return emptyAssessment(now, "", cfg.Topology.Enabled, th)
	}
	if !cfg.Topology.Enabled {
		return emptyAssessment(now, "", false, th)
	}
	nodes, err := store.ListNodes(5000)
	if err != nil {
		nodes = nil
	}
	links, err := store.ListLinks(10000)
	if err != nil {
		links = nil
	}
	ar := topology.Analyze(nodes, links, th, now)
	sig, _ := RollupRecentMessages(d, 24*time.Hour, transportConnected)
	return Compute(cfg, ar, sig, transportConnected, now)
}
