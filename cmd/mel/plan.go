package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/planning"
	"github.com/mel-project/mel/internal/topology"
)

func planCmd(args []string) {
	if len(args) == 0 {
		planUsage()
	}
	switch args[0] {
	case "create":
		f := fs("plan-create")
		path := f.String("config", configFlagDefault(), "config")
		title := f.String("title", "", "plan title")
		intent := f.String("intent", "", "intent")
		_ = f.Parse(args[1:])
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		d := openDB(cfg)
		p := planning.DeploymentPlan{
			Title:  *title,
			Intent: *intent,
			Status: "draft",
			Steps:  []planning.DeploymentStep{},
		}
		if *title == "" {
			p.Title = "untitled plan"
		}
		if err := planning.SavePlan(d, &p); err != nil {
			panic(err)
		}
		saved, _, _ := planning.GetPlan(d, p.PlanID)
		mustPrint(saved)
	case "list":
		cfg, _ := loadCfg(args[1:])
		d := openDB(cfg)
		list, err := planning.ListPlans(d, 200)
		if err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"plans": list, "count": len(list)})
	case "show":
		if len(args) < 2 {
			panic("usage: mel plan show <plan_id> --config <path>")
		}
		planID := args[1]
		rest := args[2:]
		cfg, _ := loadCfg(rest)
		d := openDB(cfg)
		p, ok, err := planning.GetPlan(d, planID)
		if err != nil {
			panic(err)
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "plan not found")
			os.Exit(1)
		}
		mustPrint(p)
	case "compare":
		f := fs("plan-compare")
		path := f.String("config", configFlagDefault(), "config")
		ids := f.String("ids", "", "comma-separated plan ids")
		_ = f.Parse(args[1:])
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		d := openDB(cfg)
		store := topology.NewStore(d)
		transportOK := meshintel.TransportLikelyConnectedFromRuntime(d)
		now := time.Now().UTC()
		th := topology.StaleThresholdsFromConfig(cfg.Topology.NodeStaleMinutes, cfg.Topology.LinkStaleMinutes)
		var plans []planning.DeploymentPlan
		for _, id := range strings.Split(*ids, ",") {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if p, ok, err := planning.GetPlan(d, id); err == nil && ok {
				plans = append(plans, p)
			}
		}
		nodes, _ := store.ListNodes(5000)
		links, _ := store.ListLinks(10000)
		ar := topology.Analyze(nodes, links, th, now)
		mi := meshintel.ComputeLive(cfg, d, store, transportOK, now)
		pc := planning.ComparePlans(plans, ar, mi, now)
		mustPrint(pc)
	default:
		planUsage()
	}
}

func planUsage() {
	panic(`usage:
  mel plan create --title "..." [--intent "..."] --config <path>
  mel plan list --config <path>
  mel plan show <plan_id> --config <path>
  mel plan compare --ids id1,id2 --config <path>`)
}
