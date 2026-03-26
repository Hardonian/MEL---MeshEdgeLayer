package planning

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/topology"
)

type planScoreEntry struct {
	id     string
	label  string
	upside float64
	safety float64
	entry  ComparisonRankEntry
}

// ComparePlans compares named deployment plans (minimal fields: id, title, steps summary).
func ComparePlans(plans []DeploymentPlan, ar topology.AnalysisResult, mi meshintel.Assessment, now time.Time) PlanComparison {
	if len(plans) == 0 {
		return PlanComparison{
			SummaryLines: []string{"No plans to compare."},
			Confidence: ConfidenceAssessment{
				Level: meshintel.ConfidenceLow,
				Score: 0.2,
			},
		}
	}

	var entries []planScoreEntry
	for _, p := range plans {
		if strings.TrimSpace(p.PlanID) == "" {
			continue
		}
		up, safe, ce := scorePlan(p, ar, mi)
		entries = append(entries, planScoreEntry{id: p.PlanID, label: p.Title, upside: up, safety: safe, entry: ce})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].upside > entries[j].upside })
	var byUpside []ComparisonRankEntry
	bestUpside := ""
	if len(entries) > 0 {
		bestUpside = entries[0].id
	}
	for _, e := range entries {
		byUpside = append(byUpside, e.entry)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].safety > entries[j].safety })
	var bySafety []ComparisonRankEntry
	bestRes := ""
	if len(entries) > 0 {
		bestRes = entries[0].id
	}
	for _, e := range entries {
		bySafety = append(bySafety, e.entry)
	}

	lowRegret := ""
	cheapest := ""
	waitID := ""

	if len(entries) > 0 {
		lowRegret = pickLowRegret(entries)
		cheapest = pickCheapest(entries)
		waitID = pickWait(entries)
	}

	summary := []string{
		fmt.Sprintf("Compared %d plan(s) using topology-only signals at %s.", len(plans), now.UTC().Format(time.RFC3339)),
		"Upside ranking favors structural leverage; safety ranking favors reversibility and fewer bridge moves.",
	}

	return PlanComparison{
		ComparedIDs:        idsOf(plans),
		RankedByUpside:     byUpside,
		RankedBySafety:     bySafety,
		LowRegretPick:      lowRegret,
		BestUpsidePick:     bestUpside,
		BestResiliencePick: bestRes,
		CheapestPlausible:  cheapest,
		WaitObserveOption:    waitID,
		SummaryLines:         summary,
		Confidence: ConfidenceAssessment{
			Level:             mi.Bootstrap.Confidence,
			Score:             coarseConfidenceScore(mi.Bootstrap.Confidence),
			MissingInputs:     []string{"field_costs", "site_access", "hardware_inventory"},
			TopologyOnlyLimits: []string{"No dollar costs modeled — complexity proxy from step kinds only."},
		},
	}
}

func idsOf(plans []DeploymentPlan) []string {
	var out []string
	for _, p := range plans {
		if p.PlanID != "" {
			out = append(out, p.PlanID)
		}
	}
	return out
}

func scorePlan(p DeploymentPlan, ar topology.AnalysisResult, mi meshintel.Assessment) (upside float64, safety float64, entry ComparisonRankEntry) {
	upside = 0.4
	safety = 0.5
	reversibility := "medium"
	obsBurden := "moderate"
	complexity := 0

	for _, st := range p.Steps {
		complexity++
		switch st.Kind {
		case StepObserveOnly:
			safety += 0.15
			upside += 0.05
		case StepAddNode, StepAddInfrastructure:
			upside += 0.12
			safety -= 0.02
			reversibility = "medium"
		case StepMoveNode, StepElevateNode:
			upside += 0.1
			safety -= 0.08
			reversibility = "low"
		case StepRemoveNode:
			upside -= 0.05
			safety -= 0.15
			reversibility = "low"
		case StepChangeRole, StepReduceInfraIntensity:
			upside += 0.08
			safety += 0.05
			reversibility = "high"
		case StepBridgeClusters:
			upside += 0.15
			safety -= 0.05
		}
	}
	if mi.Topology.FragmentationScore > 0.35 {
		upside += 0.1
	}
	if len(ar.BridgeNodes) > 0 {
		safety -= 0.05
	}
	upside = clamp01(upside)
	safety = clamp01(safety)

	if complexity > 4 {
		obsBurden = "heavy"
	} else if complexity <= 1 {
		obsBurden = "light"
	}

	entry = ComparisonRankEntry{
		ID:                  p.PlanID,
		Label:               p.Title,
		Upside:              fmt.Sprintf("structural score %.2f (higher = more leverage in graph terms)", upside),
		DownsideIfWrong:   "depends on unobserved RF; may add forwarding load without new paths",
		ConfidenceNote:    fmt.Sprintf("mesh confidence=%s", mi.Bootstrap.Confidence),
		Reversibility:       reversibility,
		ObservationBurden: obsBurden,
	}
	return upside, safety, entry
}

func pickLowRegret(entries []planScoreEntry) string {
	for _, e := range entries {
		if strings.Contains(e.entry.Reversibility, "high") {
			return e.id
		}
	}
	if len(entries) > 0 {
		return entries[len(entries)-1].id
	}
	return ""
}

func pickCheapest(entries []planScoreEntry) string {
	best := ""
	bestScore := -1.0
	for _, e := range entries {
		c := e.safety
		if strings.Contains(e.entry.ObservationBurden, "light") {
			c += 0.1
		}
		if e.entry.Reversibility == "high" {
			c += 0.05
		}
		if c > bestScore {
			bestScore = c
			best = e.id
		}
	}
	return best
}

func pickWait(entries []planScoreEntry) string {
	for _, e := range entries {
		low := strings.ToLower(e.label)
		if strings.Contains(low, "observe") || strings.Contains(low, "wait") {
			return e.id
		}
	}
	return ""
}
