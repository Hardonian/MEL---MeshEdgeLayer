package planning

import (
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/topology"
)

// ValidateExecution compares baseline assessment embedded in execution record to current graph (honest, may be inconclusive).
func ValidateExecution(exec PlanExecutionRecord, beforeMI meshintel.Assessment, ar topology.AnalysisResult, afterMI meshintel.Assessment, now time.Time) ValidationResult {
	v := ValidationResult{
		ExecutionID:           exec.ExecutionID,
		ValidatedAt:           now.UTC().Format(time.RFC3339),
		GraphHashAfter:        strings.TrimSpace(afterMI.GraphHash),
		MeshAssessmentIDAfter: strings.TrimSpace(afterMI.AssessmentID),
		Verdict:               OutcomeVerdictInconclusive,
		Caveat:                "",
		Lines:                 []string{},
		Metrics: PostChangeMetricsSnapshot{
			FragmentationBefore: beforeMI.Topology.FragmentationScore,
			FragmentationAfter:  afterMI.Topology.FragmentationScore,
			ResilienceBefore:    0,
			ResilienceAfter:     0,
		},
	}
	sumB, _ := ComputeResilience(ar, beforeMI)
	sumA, _ := ComputeResilience(ar, afterMI)
	v.Metrics.ResilienceBefore = sumB.ResilienceScore
	v.Metrics.ResilienceAfter = sumA.ResilienceScore
	if exec.BaselineMetrics.Captured {
		v.Metrics.FragmentationBefore = exec.BaselineMetrics.FragmentationBefore
		v.Metrics.ResilienceBefore = exec.BaselineMetrics.ResilienceBefore
	}
	v.Metrics.FragmentationAfter = afterMI.Topology.FragmentationScore

	if exec.ObservationHorizonHours > 0 {
		started, err := time.Parse(time.RFC3339, exec.StartedAt)
		if err == nil && now.Sub(started) < time.Duration(exec.ObservationHorizonHours)*time.Hour {
			v.Verdict = OutcomeVerdictInsufficientObservation
			v.Caveat = "Observation window not elapsed — treat as preliminary."
			v.Lines = append(v.Lines, v.Caveat)
			return v
		}
	}

	// Confounding: graph hash changed but we cannot prove causality
	if exec.PlanGraphHash != "" && afterMI.GraphHash != "" && exec.PlanGraphHash != afterMI.GraphHash {
		v.Lines = append(v.Lines, "Graph hash changed since plan baseline — compare trends cautiously if other changes may have occurred.")
	}

	fragDelta := v.Metrics.FragmentationAfter - v.Metrics.FragmentationBefore
	resDelta := v.Metrics.ResilienceAfter - v.Metrics.ResilienceBefore

	if fragDelta < -0.03 && resDelta > 0.02 {
		v.Verdict = OutcomeVerdictSupported
		v.Lines = append(v.Lines, fmt.Sprintf("Fragmentation decreased (delta %.3f) and aggregate resilience increased (delta %.3f) — directionally consistent with improvement.", fragDelta, resDelta))
	} else if fragDelta > 0.03 && resDelta < -0.02 {
		v.Verdict = OutcomeVerdictContradicted
		v.Lines = append(v.Lines, fmt.Sprintf("Fragmentation increased (delta %.3f) and resilience decreased (delta %.3f) — outcome not matching typical improvement expectation.", fragDelta, resDelta))
	} else {
		v.Verdict = OutcomeVerdictInconclusive
		v.Lines = append(v.Lines, fmt.Sprintf("No strong directional signal: frag delta %.3f, resilience delta %.3f — may need longer observation or clearer baseline.", fragDelta, resDelta))
	}
	return v
}

