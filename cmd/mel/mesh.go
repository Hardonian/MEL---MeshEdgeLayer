package main

import (
	"time"

	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/planning"
	"github.com/mel-project/mel/internal/topology"
)

func meshCmd(args []string) {
	if len(args) == 0 {
		meshUsage()
	}
	cfg, _ := loadCfg(args[1:])
	d := openDB(cfg)
	store := topology.NewStore(d)
	transportOK := meshintel.TransportLikelyConnectedFromRuntime(d)
	now := time.Now().UTC()

	switch args[0] {
	case "bootstrap":
		a := meshintel.ComputeLive(cfg, d, store, transportOK, now)
		mustPrint(map[string]any{
			"bootstrap":            a.Bootstrap,
			"evidence_model":       a.EvidenceModel,
			"topology_enabled":     a.TopologyEnabled,
			"transport_connected":  transportOK,
			"assessment_id":        a.AssessmentID,
			"computed_at":          a.ComputedAt,
		})
	case "topology":
		a := meshintel.ComputeLive(cfg, d, store, transportOK, now)
		mustPrint(map[string]any{
			"topology_metrics":    a.Topology,
			"node_intel":          a.NodeIntel,
			"evidence_model":      a.EvidenceModel,
			"topology_enabled":    a.TopologyEnabled,
			"transport_connected": transportOK,
		})
	case "diagnose":
		a := meshintel.ComputeLive(cfg, d, store, transportOK, now)
		mustPrint(map[string]any{
			"routing_pressure":    a.RoutingPressure,
			"protocol_fit":        a.ProtocolFit,
			"bootstrap_summary": map[string]any{"viability": a.Bootstrap.Viability, "lone_wolf_score": a.Bootstrap.LoneWolfScore},
			"evidence_model":      a.EvidenceModel,
		})
	case "recommend":
		a := meshintel.ComputeLive(cfg, d, store, transportOK, now)
		mustPrint(map[string]any{
			"recommendations":     a.Recommendations,
			"protocol_fit":        a.ProtocolFit,
			"bootstrap_viability": a.Bootstrap.Viability,
			"evidence_model":      a.EvidenceModel,
		})
	case "inspect":
		a := meshintel.ComputeLive(cfg, d, store, transportOK, now)
		mustPrint(a)
	case "history":
		f := fs("mesh-history")
		path := f.String("config", configFlagDefault(), "config")
		limit := f.Int("limit", 20, "max rows")
		_ = f.Parse(args[1:])
		cfg2, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		d2 := openDB(cfg2)
		list, err := meshintel.RecentSnapshots(d2, *limit)
		if err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"assessments": list, "count": len(list)})
	case "simulate":
		f := fs("mesh-simulate")
		path := f.String("config", configFlagDefault(), "config")
		kind := f.String("kind", "add_node", "scenario kind (add_node, remove_node, move_node, ...)")
		node := f.Int64("node", 0, "target node_num")
		class := f.String("class", "", "candidate_class for add scenarios")
		_ = f.Parse(args[1:])
		cfg2, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		d2 := openDB(cfg2)
		store2 := topology.NewStore(d2)
		transportOK := meshintel.TransportLikelyConnectedFromRuntime(d2)
		now := time.Now().UTC()
		th := topology.StaleThresholdsFromConfig(cfg2.Topology.NodeStaleMinutes, cfg2.Topology.LinkStaleMinutes)
		nodes, _ := store2.ListNodes(5000)
		links, _ := store2.ListLinks(10000)
		ar := topology.Analyze(nodes, links, th, now)
		mi := meshintel.ComputeLive(cfg2, d2, store2, transportOK, now)
		sk, ok := planning.NormalizeScenarioKind(*kind)
		if !ok {
			panic("unknown scenario kind")
		}
		sa := planning.RunScenarioWithClass(sk, *node, *class, ar, mi, now)
		mustPrint(sa)
	case "resilience":
		cfg2, _ := loadCfg(args[1:])
		d2 := openDB(cfg2)
		store2 := topology.NewStore(d2)
		transportOK := meshintel.TransportLikelyConnectedFromRuntime(d2)
		now := time.Now().UTC()
		th := topology.StaleThresholdsFromConfig(cfg2.Topology.NodeStaleMinutes, cfg2.Topology.LinkStaleMinutes)
		nodes, _ := store2.ListNodes(5000)
		links, _ := store2.ListLinks(10000)
		ar := topology.Analyze(nodes, links, th, now)
		mi := meshintel.ComputeLive(cfg2, d2, store2, transportOK, now)
		summary, _ := planning.ComputeResilience(ar, mi)
		mustPrint(map[string]any{
			"resilience":            summary,
			"evidence_model":      planning.PlanningEvidenceModel,
			"transport_connected": transportOK,
		})
	case "critical":
		cfg2, _ := loadCfg(args[1:])
		d2 := openDB(cfg2)
		store2 := topology.NewStore(d2)
		transportOK := meshintel.TransportLikelyConnectedFromRuntime(d2)
		now := time.Now().UTC()
		th := topology.StaleThresholdsFromConfig(cfg2.Topology.NodeStaleMinutes, cfg2.Topology.LinkStaleMinutes)
		nodes, _ := store2.ListNodes(5000)
		links, _ := store2.ListLinks(10000)
		ar := topology.Analyze(nodes, links, th, now)
		mi := meshintel.ComputeLive(cfg2, d2, store2, transportOK, now)
		_, profiles := planning.ComputeResilience(ar, mi)
		mustPrint(map[string]any{
			"node_profiles":       profiles,
			"evidence_model":      planning.PlanningEvidenceModel,
			"transport_connected": transportOK,
		})
	default:
		meshUsage()
	}
}

func meshUsage() {
	panic(`usage:
  mel mesh bootstrap --config <path>
  mel mesh topology --config <path>
  mel mesh diagnose --config <path>
  mel mesh recommend --config <path>
  mel mesh inspect --config <path>   (full assessment JSON)
  mel mesh history --config <path> [--limit n]
  mel mesh simulate --kind <kind> [--node n] [--class ...] --config <path>  (bounded what-if)
  mel mesh resilience --config <path>
  mel mesh critical --config <path>`)
}
