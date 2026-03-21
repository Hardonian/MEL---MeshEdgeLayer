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

	return models.OperatorBriefingDTO{
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
