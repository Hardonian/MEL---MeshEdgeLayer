package intelligence

import (
	"sort"
	"strings"

	"github.com/mel-project/mel/internal/models"
)

// SequenceRecoveryActions given a set of recommendations
func SequenceRecoveryActions(recommendations []Recommendation) []models.RecoveryStep {
	var stages []models.RecoveryStep

	// Temporary buckets
	verification := []Recommendation{}
	infrastructure := []Recommendation{}
	connectivity := []Recommendation{}
	operational := []Recommendation{}
	cleanup := []Recommendation{}

	for _, rec := range recommendations {
		if strings.Contains(rec.Code, "verify") || strings.Contains(rec.Code, "doctor") || strings.Contains(rec.Code, "inspect") {
			verification = append(verification, rec)
		} else if strings.Contains(rec.Code, "database") || strings.Contains(rec.Code, "config") {
			infrastructure = append(infrastructure, rec)
		} else if strings.Contains(rec.Code, "transport") || strings.Contains(rec.Code, "reconnect") {
			connectivity = append(connectivity, rec)
		} else if strings.Contains(rec.Code, "control") || strings.Contains(rec.Code, "suppress") {
			operational = append(operational, rec)
		} else {
			cleanup = append(cleanup, rec)
		}
	}

	currentStage := 1
	// Stage 1: Verification first
	for _, rec := range verification {
		stages = append(stages, models.RecoveryStep{
			Stage:         currentStage,
			Action:        rec.Action,
			Justification: rec.Rationale + " before attempting recovery actions",
			Status:        "pending",
		})
	}
	if len(verification) > 0 { currentStage++ }

	// Stage 2: Infrastructure
	for _, rec := range infrastructure {
		stages = append(stages, models.RecoveryStep{
			Stage:         currentStage,
			Action:        rec.Action,
			Justification: "Critical system component must be healthy first",
			Status:        "blocked",
			UnsafeEarly:   true,
		})
	}
	if len(infrastructure) > 0 { currentStage++ }

	// Stage 3: Connectivity
	for _, rec := range connectivity {
		stages = append(stages, models.RecoveryStep{
			Stage:         currentStage,
			Action:        rec.Action,
			Justification: "Restore transport connectivity",
			Status:        "blocked",
			UnsafeEarly:   true,
		})
	}
	if len(connectivity) > 0 { currentStage++ }

	// Stage 4: Operations
	for _, rec := range operational {
		stages = append(stages, models.RecoveryStep{
			Stage:         currentStage,
			Action:        rec.Action,
			Justification: "Clear operational noise and stabilize control plane",
			Status:        "optional",
		})
	}

	sort.Slice(stages, func(i, j int) bool {
		return stages[i].Stage < stages[j].Stage
	})

	return stages
}
