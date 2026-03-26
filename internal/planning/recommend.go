package planning

import (
	"fmt"

	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/topology"
)

// BestNextMove is a single consolidated recommendation with explicit uncertainty and wait rationale.
type BestNextMove struct {
	Title              string   `json:"title"`
	SummaryLines       []string `json:"summary_lines"`
	EvidenceAnchors    []string `json:"evidence_anchors"`
	UncertaintyNotes   []string `json:"uncertainty_notes"`
	WaitObserveRationale string `json:"wait_observe_rationale,omitempty"`
	WouldValidateWith  []string `json:"would_validate_with"`
	PrimaryVerdict     PlanVerdict `json:"primary_verdict"`
	RecommendationKey  string   `json:"recommendation_key,omitempty"`
	EvidenceClassification EvidenceModelClassification `json:"evidence_classification"`
}

// ComputeBestNextMove synthesizes mesh recommendations + resilience into one honest card.
func ComputeBestNextMove(ar topology.AnalysisResult, mi meshintel.Assessment, retro RecommendationRetrospective) BestNextMove {
	b := BestNextMove{
		EvidenceClassification: EvidenceTopologyOnly,
		WouldValidateWith: []string{
			"Stable observation window (24–72h) with transport connected",
			"Fragmentation score trend from mesh intelligence snapshots",
		},
		UncertaintyNotes: ExplainLimits(),
	}
	if len(mi.Recommendations) > 0 {
		r := mi.Recommendations[0]
		b.Title = QualifyRecommendation(r.Title)
		b.PrimaryVerdict = verdictFromRec(r)
		b.RecommendationKey = RecordRecommendationOutcomeKey(r.Rank, r.Class)
		b.SummaryLines = []string{
			r.ExpectedBenefit,
			"Downside if wrong: " + r.DownsideRisk,
		}
		b.EvidenceAnchors = append(b.EvidenceAnchors, r.EvidenceSummary...)
	} else {
		b.Title = "No strong automated preference"
		b.PrimaryVerdict = VerdictInsufficientData
		b.SummaryLines = []string{PhraseInsufficientEvidence + " — gather more observed topology history."}
	}

	sum, _ := ComputeResilience(ar, mi)
	b.SummaryLines = append(b.SummaryLines, "Resilience snapshot: "+fmt.Sprintf("score=%.2f partition-risk-proxy=%.2f", sum.ResilienceScore, sum.PartitionRiskScore))
	b.EvidenceAnchors = append(b.EvidenceAnchors, sum.FragilityExplanation...)

	if mi.Bootstrap.Viability == meshintel.ViabilityUnstableIntermittent || mi.Bootstrap.Confidence == meshintel.ConfidenceLow {
		b.WaitObserveRationale = "Evidence is weak or uptime is unstable — waiting often beats adding hardware until the graph stabilizes."
		b.PrimaryVerdict = VerdictDeferObserve
	}

	if retro.TotalRecorded > 0 {
		b.SummaryLines = append(b.SummaryLines, fmt.Sprintf("Recorded outcomes for similar keys: supported=%d inconclusive=%d contradicted=%d (historical, not predictive).",
			retro.SuccessCount, retro.InconclusiveCount, retro.ContradictedCount))
	}
	return b
}

func verdictFromRec(r meshintel.MeshRecommendation) PlanVerdict {
	if r.Class == meshintel.RecKeepObserve || r.Class == meshintel.RecGatherMoreHistory {
		return VerdictDeferObserve
	}
	if r.Confidence < 0.35 {
		return VerdictInsufficientData
	}
	return VerdictProceedWithCaution
}

// WaitVersusExpandHeuristic returns a short operator-facing line (not a command).
func WaitVersusExpandHeuristic(mi meshintel.Assessment) string {
	if mi.Bootstrap.Viability == meshintel.ViabilityUnstableIntermittent {
		return "Plausible: stabilize uptime and observation before buying more nodes."
	}
	if mi.Topology.FragmentationScore < 0.25 && mi.Bootstrap.Confidence == meshintel.ConfidenceHigh {
		return "Graph looks relatively cohesive — next bottleneck may be placement/roles rather than more endpoints."
	}
	return "If unsure, prefer a reversible diagnostic step over hardware expansion."
}

// RecordRecommendationOutcomeKey builds a stable key for mesh recommendations.
func RecordRecommendationOutcomeKey(rank int, class meshintel.RecommendationClass) string {
	return fmt.Sprintf("meshrec:%s:%d", class, rank)
}
