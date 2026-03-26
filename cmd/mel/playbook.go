package main

import (
	"time"

	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/planning"
	"github.com/mel-project/mel/internal/topology"
)

func playbookCmd(args []string) {
	if len(args) == 0 || args[0] != "suggest" {
		panic(`usage: mel playbook suggest --config <path>`)
	}
	cfg, _ := loadCfg(args[1:])
	d := openDB(cfg)
	store := topology.NewStore(d)
	transportOK := meshintel.TransportLikelyConnectedFromRuntime(d)
	now := time.Now().UTC()
	th := topology.StaleThresholdsFromConfig(cfg.Topology.NodeStaleMinutes, cfg.Topology.LinkStaleMinutes)
	nodes, _ := store.ListNodes(5000)
	links, _ := store.ListLinks(10000)
	ar := topology.Analyze(nodes, links, th, now)
	mi := meshintel.ComputeLive(cfg, d, store, transportOK, now)
	pb := planning.SuggestPlaybooks(ar, mi)
	mustPrint(map[string]any{
		"playbooks":           pb,
		"count":               len(pb),
		"evidence_model":      planning.PlanningEvidenceModel,
		"transport_connected": transportOK,
	})
}
