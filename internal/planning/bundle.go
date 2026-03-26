package planning

import (
	"fmt"
	"sort"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/topology"
)

// BuildBundle is the main entry: resilience + playbooks + ranked hints from mesh intel.
func BuildBundle(cfg config.Config, ar topology.AnalysisResult, mi meshintel.Assessment, transportConnected bool, now time.Time, retro RecommendationRetrospective) PlanningBundle {
	resSummary, profiles := ComputeResilience(ar, mi)
	playbooks := SuggestPlaybooks(ar, mi)
	ranked := rankNextPlans(ar, mi)
	best := ComputeBestNextMove(ar, mi, retro)

	limits := ExplainLimits()
	if !cfg.Topology.Enabled {
		limits = append(limits, "Topology model disabled — planning uses empty or stale graph.")
	}
	if !transportConnected {
		limits = append(limits, "No ingest-capable transport — observations may be stale; defer major deployment changes.")
	}

	return PlanningBundle{
		EvidenceModel:      PlanningEvidenceModel,
		GraphHash:          ar.Snapshot.GraphHash,
		MeshAssessmentID:   mi.AssessmentID,
		TransportConnected: transportConnected,
		TopologyEnabled:    cfg.Topology.Enabled,
		Resilience:         resSummary,
		NodeProfiles:       profiles,
		RankedNextPlans:    ranked,
		BestNextMove:       best,
		WaitVersusExpand:   WaitVersusExpandHeuristic(mi),
		Playbooks:          playbooks,
		Limits:             limits,
		ComputedAt:         now.UTC().Format(time.RFC3339),
	}
}

func rankNextPlans(ar topology.AnalysisResult, mi meshintel.Assessment) []RankedPlanHint {
	var hints []RankedPlanHint
	for i, r := range mi.Recommendations {
		band := OutcomePlausibleModerate
		switch r.Severity {
		case "high":
			band = OutcomeLikelyHighLeverage
		case "low", "info":
			band = OutcomeLikelyLowBenefit
		}
		v := VerdictProceedWithCaution
		if r.Class == meshintel.RecKeepObserve || r.Class == meshintel.RecGatherMoreHistory {
			v = VerdictDeferObserve
			band = OutcomeUncertainButPromising
		}
		if r.Confidence < 0.35 {
			v = VerdictInsufficientData
		}
		id := fmt.Sprintf("mesh-rec-%d", i+1)
		if r.Rank > 0 {
			id = fmt.Sprintf("mesh-rec-%d", r.Rank)
		}
		hints = append(hints, RankedPlanHint{
			Rank:        i + 1,
			ID:          id,
			Title:       QualifyRecommendation(r.Title),
			Verdict:     v,
			BenefitBand: band,
			Lines: append([]string{
				r.ExpectedBenefit,
				"Downside if wrong: " + r.DownsideRisk,
			}, r.EvidenceSummary...),
		})
	}
	sort.Slice(hints, func(i, j int) bool {
		return hints[i].Rank < hints[j].Rank
	})
	return hints
}
