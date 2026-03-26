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
	dims   DecisionDimensionScores
}

// ComparePlans compares named deployment plans with explicit decision dimensions.
func ComparePlans(plans []DeploymentPlan, ar topology.AnalysisResult, mi meshintel.Assessment, now time.Time) PlanComparison {
	evClass := EvidenceTopologyOnly
	for _, p := range plans {
		if strings.TrimSpace(p.InputSetVersionID) != "" {
			evClass = EvidenceTopologyAssumptionAugmented
			break
		}
	}

	if len(plans) == 0 {
		return PlanComparison{
			SummaryLines: []string{"No plans to compare."},
			Confidence: ConfidenceAssessment{
				Level: meshintel.ConfidenceLow,
				Score: 0.2,
			},
			EvidenceClassification: evClass,
		}
	}

	var entries []planScoreEntry
	for _, p := range plans {
		if strings.TrimSpace(p.PlanID) == "" {
			continue
		}
		up, safe, ce, dims := scorePlanFull(p, ar, mi)
		entries = append(entries, planScoreEntry{id: p.PlanID, label: p.Title, upside: up, safety: safe, entry: ce, dims: dims})
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

	sort.Slice(entries, func(i, j int) bool { return entries[i].dims.LowRegretScore > entries[j].dims.LowRegretScore })
	var byLowRegret []ComparisonRankEntry
	lowRegret := ""
	bestDiag := ""
	if len(entries) > 0 {
		lowRegret = entries[0].id
		for _, e := range entries {
			byLowRegret = append(byLowRegret, e.entry)
		}
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].dims.DiagnosticValueScore > entries[j].dims.DiagnosticValueScore })
	if len(entries) > 0 {
		bestDiag = entries[0].id
	}

	cheapest := pickCheapest(entries)
	waitID := pickWait(entries)

	rankingNotes := []string{
		"If uptime is unstable, ranking could change toward observe-first moves.",
		"If placement assumptions are wrong, upside picks may not materialize.",
	}

	summary := []string{
		fmt.Sprintf("Compared %d plan(s) using topology-bounded signals at %s.", len(plans), now.UTC().Format(time.RFC3339)),
		"Upside ranking uses structural leverage proxies; low-regret blends reversibility, diagnostic value, and observation burden.",
		PhraseNoGuaranteePartition + ".",
	}

	return PlanComparison{
		ComparedIDs:          idsOf(plans),
		RankedByUpside:       byUpside,
		RankedBySafety:       bySafety,
		RankedByLowRegret:    byLowRegret,
		LowRegretPick:        lowRegret,
		BestUpsidePick:       bestUpside,
		BestResiliencePick:   bestRes,
		BestDiagnosticPick:   bestDiag,
		CheapestPlausible:  cheapest,
		WaitObserveOption:    waitID,
		RankingCouldChangeIf: rankingNotes,
		SummaryLines:         summary,
		EvidenceClassification: evClass,
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

func scorePlanFull(p DeploymentPlan, ar topology.AnalysisResult, mi meshintel.Assessment) (upside float64, safety float64, entry ComparisonRankEntry, dims DecisionDimensionScores) {
	upside = 0.4
	safety = 0.5
	reversibility := "medium"
	obsBurden := "moderate"
	complexity := 0
	frag := AssumptionFragilityScore(extractAssumptionsFromPlan(p), mi)

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
		case StepImproveUptime:
			upside += 0.07
			safety += 0.1
			reversibility = "high"
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

	revScore := reversibilityToScore(reversibility)
	obsScore := observationBurdenToScore(obsBurden)
	diagnostic := clamp01(0.4*revScore + 0.35*(1-obsScore) + 0.25*(1-frag))
	learning := diagnostic
	costProxy := clamp01(float64(complexity) * 0.12)
	disruption := clamp01(costProxy*0.7 + (1-revScore)*0.3)
	lowRegret := clamp01(0.35*revScore + 0.25*diagnostic + 0.2*(1-frag) + 0.2*(1-obsScore) - 0.1*disruption)
	expansionReady := clamp01(upside*0.5 + (1-mi.Topology.FragmentationScore)*0.5)

	dims = DecisionDimensionScores{
		ReversibilityScore:         revScore,
		ObservationBurdenScore:     obsScore,
		DiagnosticValueScore:       diagnostic,
		LearningValueScore:         learning,
		CostComplexityProxy:        costProxy,
		LowRegretScore:             lowRegret,
		ExpansionReadinessScore:    expansionReady,
		AssumptionFragilityScore:   frag,
		OperationalDisruptionScore: disruption,
		UpsideScore:                upside,
		UncertaintyPenalty:         1 - coarseConfidenceScore(mi.Bootstrap.Confidence),
	}

	narrative := []string{
		fmt.Sprintf("Reversibility=%s; observation burden=%s.", reversibility, obsBurden),
		fmt.Sprintf("Assumption fragility (0–1)=%.2f — higher means rankings are more sensitive to unknowns.", frag),
	}
	if mi.Bootstrap.Viability == meshintel.ViabilityUnstableIntermittent {
		narrative = append(narrative, "Mesh shows unstable/intermittent viability — improving uptime often outranks adding hardware.")
	}

	entry = ComparisonRankEntry{
		ID:                  p.PlanID,
		Label:               p.Title,
		Upside:              fmt.Sprintf("structural upside proxy %.2f (not RF proof)", upside),
		DownsideIfWrong:   "depends on unobserved RF; may add forwarding load without new paths",
		ConfidenceNote:    fmt.Sprintf("mesh confidence=%s", mi.Bootstrap.Confidence),
		Reversibility:       reversibility,
		ObservationBurden: obsBurden,
		Dimensions:          dims,
		NarrativeLines:      narrative,
	}
	return upside, safety, entry, dims
}

func extractAssumptionsFromPlan(p DeploymentPlan) []AssumptionItem {
	var out []AssumptionItem
	for _, st := range p.Steps {
		for _, it := range st.Assumptions.Items {
			ai := AssumptionItem{
				Key:         it.Key,
				Value:       it.Value,
				Source:      AssumptionSource(it.Provenance),
				Description: it.Description,
			}
			if ai.Source == "" {
				ai.Source = AssumptionSourceUnknown
			}
			out = append(out, ai)
		}
	}
	return out
}

func reversibilityToScore(r string) float64 {
	switch strings.ToLower(r) {
	case "high":
		return 0.85
	case "low":
		return 0.25
	default:
		return 0.55
	}
}

func observationBurdenToScore(b string) float64 {
	switch strings.ToLower(b) {
	case "light":
		return 0.2
	case "heavy":
		return 0.85
	default:
		return 0.5
	}
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
		c += e.dims.DiagnosticValueScore * 0.05
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
