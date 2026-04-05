package intelligence

import (
	"time"

	"github.com/mel-project/mel/internal/models"
)

// GenerateBriefing for current system status
func GenerateBriefing(priorities []models.PriorityItem, recommendations []Recommendation, sequence []models.RecoveryStep, blastRadiusMessage string, now time.Time) models.OperatorBriefingDTO {
	var status = "Healthy"
	var causes = []string{}
	var notes = []string{}

	if len(priorities) > 0 {
		status = "Degraded"
		for _, p := range priorities {
			if p.Severity == "critical" {
				status = "Critical"
			}
			if val, ok := p.Metadata["likely_causes"].([]string); ok {
				causes = append(causes, val...)
			}
		}
	}

	if len(priorities) > 0 && len(recommendations) == 0 {
		notes = append(notes, "No automated recommendations available for these issues; manual triage required")
	}

	if status == "Critical" {
		notes = append(notes, "Focus on recovery sequencing to clear blockers first")
	}

	truthBasis := []string{
		"Open incidents and diagnostic findings from this instance (SQLite-backed where available).",
		"Rank and recovery sequence are deterministic heuristics over those inputs — not RF/path proof or ML inference.",
	}
	if len(priorities) == 0 {
		truthBasis = append(truthBasis, "No open incident or high-signal diagnostic rows in the briefing window — overall posture reflects absence of ranked issues, not proof of mesh health.")
	}

	return models.OperatorBriefingDTO{
		APIVersion:          "v1",
		TruthBasis:          truthBasis,
		OverallStatus:       status,
		TopPriorities:       priorities,
		LikelyCauses:        dedupe(causes),
		RecommendedSequence: sequence,
		BlastRadiusEstimate: blastRadiusMessage,
		UncertaintyNotes:    notes,
		GeneratedAt:         now.Format(time.RFC3339),
	}
}

func dedupe(in []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range in {
		if !seen[s] {
			out = append(out, s)
			seen[s] = true
		}
	}
	return out
}
