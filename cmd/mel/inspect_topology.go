package main

import (
	"time"

	"github.com/mel-project/mel/internal/topology"
)

func inspectTopologyCmd(args []string) {
	refresh := false
	var rest []string
	for _, a := range args {
		if a == "--refresh" {
			refresh = true
			continue
		}
		rest = append(rest, a)
	}
	cfg, _ := loadCfg(rest)
	d := openDB(cfg)
	store := topology.NewStore(d)
	th := topology.StaleThresholdsFromConfig(cfg.Topology.NodeStaleMinutes, cfg.Topology.LinkStaleMinutes)
	now := time.Now().UTC()
	if refresh && cfg.Topology.Enabled {
		_ = store.RefreshStale(th)
		if ar, err := store.RefreshScores(th, now); err == nil {
			_ = store.SaveSnapshot(ar.Snapshot)
			maxHist := cfg.Topology.MaxSnapshotHistory
			if maxHist <= 0 {
				maxHist = 200
			}
			_ = store.PruneSnapshots(maxHist)
			_ = store.PruneObservations(cfg.Topology.MaxObservationsPerNode)
		}
	}
	nodes, err := store.ListNodes(5000)
	if err != nil {
		panic(err)
	}
	links, err := store.ListLinks(10000)
	if err != nil {
		panic(err)
	}
	ar := topology.Analyze(nodes, links, th, now)
	out := map[string]any{
		"topology_enabled": cfg.Topology.Enabled,
		"refreshed":        refresh,
		"analysis":         ar,
		"staleness": map[string]any{
			"node_stale_minutes": cfg.Topology.NodeStaleMinutes,
			"link_stale_minutes": cfg.Topology.LinkStaleMinutes,
		},
		"evidence_model": "Links from topology_links are derived from ingested mesh packets (relay_node / to_node); not RF adjacency proof.",
	}
	mustPrint(out)
}
