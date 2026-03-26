package main

import (
	"time"

	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/planning"
	"github.com/mel-project/mel/internal/topology"
)

func recommendCmd(args []string) {
	if len(args) == 0 || args[0] != "next" {
		panic(`usage: mel recommend next --config <path>`)
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
	var retro planning.RecommendationRetrospective
	if len(mi.Recommendations) > 0 {
		r := mi.Recommendations[0]
		key := planning.RecordRecommendationOutcomeKey(r.Rank, r.Class)
		if r2, err := planning.RecommendationRetrospectiveForKey(d, key); err == nil {
			retro = r2
		}
	}
	bm := planning.ComputeBestNextMove(ar, mi, retro)
	out := map[string]any{
		"best_next_move":          bm,
		"wait_versus_expand_hint": planning.WaitVersusExpandHeuristic(mi),
		"evidence_model":          planning.PlanningEvidenceModel,
	}
	mustPrint(out)
}
