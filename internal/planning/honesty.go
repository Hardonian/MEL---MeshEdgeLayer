package planning

import "strings"

// Bounded language helpers — use across API/CLI/UI copy generation.
const (
	PhraseAdvisoryPrefix           = "Advisory estimate"
	PhraseTopologyOnly             = "Topology-only estimate"
	PhraseInsufficientEvidence     = "Insufficient evidence for a strong preference"
	PhraseNoGuaranteePartition     = "Does not prove a partition will occur"
	PhraseNotRFCoverage            = "Not RF coverage prediction"
)

// QualifyRecommendation prefixes titles to avoid "best/optimal" overclaiming.
func QualifyRecommendation(title string) string {
	t := strings.TrimSpace(title)
	if t == "" {
		return t
	}
	lower := strings.ToLower(t)
	if strings.HasPrefix(lower, "best ") || strings.HasPrefix(lower, "optimal ") {
		return "Plausible next step: " + t
	}
	return t
}
