package main

import (
	"time"

	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/planning"
	"github.com/mel-project/mel/internal/topology"
)

func nodeWhatifCmd(args []string) {
	f := fs("node-whatif")
	path := f.String("config", configFlagDefault(), "config")
	kind := f.String("kind", "move", "add|move|remove|role|uptime")
	node := f.Int64("node", 0, "node_num (required for non-add)")
	class := f.String("class", "", "for add: handheld|fixed_relay|infrastructure_anchor|event_ephemeral")
	_ = f.Parse(args)
	cfg, _, err := loadConfigFile(*path)
	if err != nil {
		panic(err)
	}
	d := openDB(cfg)
	store := topology.NewStore(d)
	transportOK := meshintel.TransportLikelyConnectedFromRuntime(d)
	now := time.Now().UTC()
	th := topology.StaleThresholdsFromConfig(cfg.Topology.NodeStaleMinutes, cfg.Topology.LinkStaleMinutes)
	nodes, _ := store.ListNodes(5000)
	links, _ := store.ListLinks(10000)
	ar := topology.Analyze(nodes, links, th, now)
	mi := meshintel.ComputeLive(cfg, d, store, transportOK, now)

	var ik planning.ImpactKind
	switch *kind {
	case "add":
		ik = planning.ImpactAdd
	case "move":
		ik = planning.ImpactMove
	case "remove":
		ik = planning.ImpactRemove
	case "role":
		ik = planning.ImpactRole
	case "uptime":
		ik = planning.ImpactUptime
	default:
		panic("kind must be add|move|remove|role|uptime")
	}
	cc := planning.ParseImpactCandidateClass(*class)
	impact := planning.EstimateImpact(ik, *node, cc, ar, mi)
	mustPrint(impact)
}
