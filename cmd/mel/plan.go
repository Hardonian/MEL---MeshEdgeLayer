package main

import (
	"encoding/json"
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
	case "edit":
		f := fs("plan-edit")
		path := f.String("config", configFlagDefault(), "config")
		planID := f.String("plan", "", "plan id")
		jsonPath := f.String("file", "", "path to JSON deployment plan body (full plan object)")
		_ = f.Parse(args[1:])
		if strings.TrimSpace(*planID) == "" || strings.TrimSpace(*jsonPath) == "" {
			panic("usage: mel plan edit --plan <id> --file <path.json> --config <path>")
		}
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		d := openDB(cfg)
		b, err := os.ReadFile(*jsonPath)
		if err != nil {
			panic(err)
		}
		var p planning.DeploymentPlan
		if err := json.Unmarshal(b, &p); err != nil {
			panic(err)
		}
		p.PlanID = *planID
		if err := planning.SavePlan(d, &p); err != nil {
			panic(err)
		}
		saved, _, _ := planning.GetPlan(d, p.PlanID)
		mustPrint(saved)
	case "inputs":
		f := fs("plan-inputs")
		path := f.String("config", configFlagDefault(), "config")
		setID := f.String("set", "", "input set id (optional, generated if empty)")
		title := f.String("title", "operator inputs", "input set title")
		kv := f.String("kv", "", "comma-separated key=value assumptions (operator source)")
		_ = f.Parse(args[1:])
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		d := openDB(cfg)
		id, err := planning.SaveInputSet(d, *setID, *title)
		if err != nil {
			panic(err)
		}
		store := topology.NewStore(d)
		transportOK := meshintel.TransportLikelyConnectedFromRuntime(d)
		now := time.Now().UTC()
		th := topology.StaleThresholdsFromConfig(cfg.Topology.NodeStaleMinutes, cfg.Topology.LinkStaleMinutes)
		nodes, _ := store.ListNodes(5000)
		links, _ := store.ListLinks(10000)
		ar := topology.Analyze(nodes, links, th, now)
		mi := meshintel.ComputeLive(cfg, d, store, transportOK, now)
		var items []planning.AssumptionItem
		for _, part := range strings.Split(*kv, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			k, v, ok := strings.Cut(part, "=")
			if !ok {
				continue
			}
			items = append(items, planning.AssumptionItem{
				Key: strings.TrimSpace(k), Value: strings.TrimSpace(v),
				Source: planning.AssumptionSourceOperator, Confidence: planning.AssumptionConfMedium,
				Sensitivity: planning.SensitivityMedium,
			})
		}
		payload := planning.BuildInputVersionPayload(id, 0, items, ar.Snapshot.GraphHash, mi.AssessmentID)
		vid, err := planning.SaveInputVersion(d, id, payload)
		if err != nil {
			panic(err)
		}
		p, _, _ := planning.GetInputVersion(d, vid)
		mustPrint(map[string]any{"input_set_id": id, "version_id": vid, "payload": p})
	case "mark-step":
		f := fs("plan-mark-step")
		path := f.String("config", configFlagDefault(), "config")
		planID := f.String("plan", "", "plan id")
		stepID := f.String("step", "", "step id")
		note := f.String("note", "", "operator note")
		obs := f.Int("observe-hours", 24, "observation horizon hours")
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
		nodes, _ := store.ListNodes(5000)
		links, _ := store.ListLinks(10000)
		ar := topology.Analyze(nodes, links, th, now)
		mi := meshintel.ComputeLive(cfg, d, store, transportOK, now)
		sum, _ := planning.ComputeResilience(ar, mi)
		base := planning.PostChangeMetricsSnapshot{
			Captured:            true,
			FragmentationBefore: mi.Topology.FragmentationScore,
			ResilienceBefore:    sum.ResilienceScore,
		}
		eid, err := planning.StartPlanExecution(d, *planID, ar.Snapshot.GraphHash, mi.AssessmentID, base, *obs, "")
		if err != nil {
			panic(err)
		}
		sid, err := planning.MarkStepExecuted(d, eid, *stepID, *note)
		if err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"execution_id": eid, "step_execution_id": sid})
	case "outcome":
		f := fs("plan-outcome")
		path := f.String("config", configFlagDefault(), "config")
		execID := f.String("execution", "", "execution id from mark-step")
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
		nodes, _ := store.ListNodes(5000)
		links, _ := store.ListLinks(10000)
		ar := topology.Analyze(nodes, links, th, now)
		miAfter := meshintel.ComputeLive(cfg, d, store, transportOK, now)
		exec, ok, err := planning.GetPlanExecution(d, *execID)
		if err != nil || !ok {
			panic("execution not found")
		}
		miBefore := miAfter
		if strings.TrimSpace(exec.MeshAssessmentID) != "" {
			if a, ok2, err2 := meshintel.GetAssessmentByID(d, exec.MeshAssessmentID); err2 == nil && ok2 {
				miBefore = a
			}
		}
		vr := planning.ValidateExecution(exec, miBefore, ar, miAfter, now)
		vid, err := planning.SaveValidation(d, *execID, vr)
		if err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"validation_id": vid, "validation": vr})
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
  mel plan edit --plan <id> --file <plan.json> --config <path>
  mel plan inputs [--set id] [--title "..."] [--kv key=value,...] --config <path>
  mel plan mark-step --plan <id> --step <step_id> [--note "..."] [--observe-hours N] --config <path>
  mel plan outcome --execution <execution_id> --config <path>
  mel plan compare --ids id1,id2 --config <path>`)
}
