package main

import (
	"time"

	"github.com/mel-project/mel/internal/meshintel"
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
  mel mesh history --config <path> [--limit n]`)
}
