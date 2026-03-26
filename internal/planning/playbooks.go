package planning

import (
	"fmt"

	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/topology"
)

// SuggestPlaybooks returns playbooks grounded in current viability and shape.
func SuggestPlaybooks(ar topology.AnalysisResult, mi meshintel.Assessment) []Playbook {
	var out []Playbook
	v := mi.Bootstrap.Viability
	shape := mi.Topology.ClusterShape

	anchors := []string{
		fmt.Sprintf("viability=%s", v),
		fmt.Sprintf("cluster_shape=%s", shape),
		fmt.Sprintf("largest_component=%d", mi.Bootstrap.LargestComponentSize),
	}

	if v == meshintel.ViabilityIsolated || len(ar.ScoredNodes) <= 1 {
		out = append(out, loneWolfPlaybook(anchors))
	}
	if v == meshintel.ViabilityWeakBootstrap || shape == meshintel.ShapeFragileBridge {
		out = append(out, fragilePlaybook(anchors))
	}
	if shape == meshintel.ShapeCorridor || shape == meshintel.ShapeSparseCluster {
		out = append(out, corridorPlaybook(anchors))
	}
	if v == meshintel.ViabilityEmergingCluster || v == meshintel.ViabilityViableLocalMesh {
		out = append(out, neighborhoodSeedPlaybook(anchors))
	}
	if mi.Topology.DependencyConcentrationScore > 0.45 || len(ar.BridgeNodes) > 0 {
		out = append(out, dependencyReducePlaybook(anchors, ar.BridgeNodes))
	}
	if mi.RoutingPressure.OverRouterizationRiskScore.Score > 0.5 {
		out = append(out, resilienceUpgradePlaybook(anchors))
	}
	out = append(out, eventMeshPlaybook(anchors))

	return out
}

func loneWolfPlaybook(anchors []string) Playbook {
	return Playbook{
		Class:            PlaybookLoneWolf,
		Title:            "Cold-start: from lone node to first useful link",
		Summary:          "Establish one mutual RF path before scaling endpoints. MEL only sees packets — verify on-air, not just broker visibility.",
		MinimumMilestone: "Two nodes appear in the same connected component in topology with fresh edges.",
		Steps: []PlaybookStep{
			{Order: 1, Title: "Confirm transport observation", Rationale: "If MEL is not ingesting, you are blind to progress.", ObserveHours: 4, SuccessIndicators: []string{"topology_links rows increase", "last_seen_at fresh"}, FailureIndicators: []string{"no new edges after power cycle"}, DoNotProceedIf: []string{"transport disconnected for whole window"}},
			{Order: 2, Title: "Add or elevate one stationary node", Rationale: "Handheld-only cold starts often fail to form stable RF paths.", ObserveHours: 24, SuccessIndicators: []string{"largest_component_size >= 2", "bootstrap viability improves"}, FailureIndicators: []string{"still isolated in graph"}, RollbackOrPauseNote: "Revisit placement before buying more hardware."},
			{Order: 3, Title: "Observe before adding more nodes", Rationale: "Density without backbone increases forwarding stress.", ObserveHours: 72, SuccessIndicators: []string{"stable edges across stale window"}, FailureIndicators: []string{"flapping edges or all stale"}},
		},
		Limits:          []string{"Not RF planning — graph only.", "No guarantee of neighborhood-scale coverage."},
		EvidenceAnchors: anchors,
	}
}

func fragilePlaybook(anchors []string) Playbook {
	return Playbook{
		Class:            PlaybookFragileStabilize,
		Title:            "Stabilize a fragile bridge or weak bootstrap",
		Summary:          "Prefer redundancy on the narrow path before expanding footprint.",
		MinimumMilestone: "A second path appears OR bridge node gains degree ≥ 2 with fresh edges.",
		Steps: []PlaybookStep{
			{Order: 1, Title: "Identify bridge-class nodes", Rationale: "Single-path structures fail when one site blinks.", ObserveHours: 12, SuccessIndicators: []string{"bridge list matches field intuition"}, FailureIndicators: []string{"stale-only edges"}},
			{Order: 2, Title: "Add redundancy near the bridge", Rationale: "Parallel corridor beats more endpoints behind the same choke point.", ObserveHours: 48, SuccessIndicators: []string{"fragmentation score trends down"}, FailureIndicators: []string{"partition count increases"}},
			{Order: 3, Title: "Defer corridor growth until stable", Rationale: "Growing a thin corridor without backbone repeats fragility.", ObserveHours: 72, SuccessIndicators: []string{"stable observations 72h"}, DoNotProceedIf: []string{"transport down", "mostly stale nodes"}},
		},
		Limits:          []string{"Bridge detection is heuristic on packet graph."},
		EvidenceAnchors: anchors,
	}
}

func corridorPlaybook(anchors []string) Playbook {
	return Playbook{
		Class:            PlaybookCorridor,
		Title:            "Corridor / route seeding",
		Summary:          "Place relays along the corridor with overlapping RF — MEL validates via edges, not maps.",
		MinimumMilestone: "Sequential pairs in the corridor show edges in topology.",
		Steps: []PlaybookStep{
			{Order: 1, Title: "Anchor ends first", Rationale: "Ends define the path; middle fills once anchors hear each other indirectly.", ObserveHours: 24, SuccessIndicators: []string{"path forms toward middle"}, FailureIndicators: []string{"no progress toward connectivity"}},
			{Order: 2, Title: "Add middle relay only where gap remains", Rationale: "Avoid redundant routers in same RF cell.", ObserveHours: 48, SuccessIndicators: []string{"continuous component along corridor"}, FailureIndicators: []string{"over-routerization signals rise"}},
		},
		Limits:          []string{"No terrain — operator must validate line-of-sight."},
		EvidenceAnchors: anchors,
	}
}

func neighborhoodSeedPlaybook(anchors []string) Playbook {
	return Playbook{
		Class:            PlaybookNeighborhoodSeed,
		Title:            "Neighborhood seed (3-node phased)",
		Summary:          "One backbone plus two edge nodes often beats three handhelds with no anchor.",
		MinimumMilestone: "Three nodes in one component with at least one stationary candidate.",
		Steps: []PlaybookStep{
			{Order: 1, Title: "Establish backbone", Rationale: "Always-on anchor stabilizes observations.", ObserveHours: 24, SuccessIndicators: []string{"fresh edges to anchor"}, FailureIndicators: []string{"anchor stale"}},
			{Order: 2, Title: "Add second hop", Rationale: "Grow radius from backbone.", ObserveHours: 48, SuccessIndicators: []string{"degree growth at edge"}, FailureIndicators: []string{"only star topology with no lateral paths"}},
			{Order: 3, Title: "Add third for redundancy or coverage", Rationale: "Third node should fix a gap, not duplicate role.", ObserveHours: 72, SuccessIndicators: []string{"redundancy score improves"}, DoNotProceedIf: []string{"routing pressure already high"}},
		},
		Limits:          []string{"Sequencing is advisory — local constraints dominate."},
		EvidenceAnchors: anchors,
	}
}

func dependencyReducePlaybook(anchors []string, bridges []int64) Playbook {
	step1 := "Map broker-heavy or relay-dependent edges."
	if len(bridges) > 0 {
		step1 = fmt.Sprintf("Bridge nodes in observed graph include %v — validate in field.", bridges)
	}
	return Playbook{
		Class:            PlaybookDependencyReduce,
		Title:            "Reduce dependency on one infrastructure choke point",
		Summary:          "Add a local RF path that does not depend on the same ingress or single bridge.",
		MinimumMilestone: "Second ingress or lateral RF path appears in topology.",
		Steps: []PlaybookStep{
			{Order: 1, Title: "Document current dependency", Rationale: step1, ObserveHours: 8, SuccessIndicators: []string{"dependency concentration understood"}, FailureIndicators: []string{"no edges to reason about"}},
			{Order: 2, Title: "Introduce alternate path", Rationale: "Parallel reduces blast radius of one site.", ObserveHours: 48, SuccessIndicators: []string{"dependency concentration falls"}, FailureIndicators: []string{"same single bridge still"}},
		},
		Limits:          []string{"Cannot see off-mesh RF."},
		EvidenceAnchors: anchors,
	}
}

func resilienceUpgradePlaybook(anchors []string) Playbook {
	return Playbook{
		Class:            PlaybookResilienceUpgrade,
		Title:            "Routing pressure / resilience upgrade",
		Summary:          "When forwarding stress is high, roles and placement beat more routers.",
		MinimumMilestone: "Over-routerization risk stops rising across observation window.",
		Steps: []PlaybookStep{
			{Order: 1, Title: "Measure before adding hardware", Rationale: "Confirm stress is structural in MEL signals.", ObserveHours: 24, SuccessIndicators: []string{"stable metrics window"}, FailureIndicators: []string{"transport gaps"}},
			{Order: 2, Title: "Reduce router roles or segment", Rationale: "Lower duplicate forwarding before expanding.", ObserveHours: 48, SuccessIndicators: []string{"routing pressure metrics ease"}, FailureIndicators: []string{"reachability loss"}},
		},
		Limits:          []string{"Routing metrics are proxies from ingested traffic."},
		EvidenceAnchors: anchors,
	}
}

func eventMeshPlaybook(anchors []string) Playbook {
	return Playbook{
		Class:            PlaybookEventMesh,
		Title:            "Temporary event mesh",
		Summary:          "Use a short observation window; tear-down should not strand dependent home nodes.",
		MinimumMilestone: "Event nodes isolated in graph from production mesh OR explicit operator note.",
		Steps: []PlaybookStep{
			{Order: 1, Title: "Pre-position anchors", Rationale: "Event density without backbone raises collision risk.", ObserveHours: 4, SuccessIndicators: []string{"edges among event nodes"}, FailureIndicators: []string{"no local mesh"}},
			{Order: 2, Title: "Observe during event", Rationale: "Validate forwarding stress live.", ObserveHours: 8, SuccessIndicators: []string{"metrics stable"}, FailureIndicators: []string{"storm of stale nodes"}},
			{Order: 3, Title: "Post-event rollback plan", Rationale: "Return roles/placement to sustainable.", ObserveHours: 24, SuccessIndicators: []string{"home mesh unchanged or recovered"}, FailureIndicators: []string{"unexpected dependency on event relay"}},
		},
		Limits:          []string{"Event topology may not generalize to permanent installs."},
		EvidenceAnchors: anchors,
	}
}
