package planning

import (
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/topology"
)

// ValidateExecution compares baseline assessment embedded in execution record to current graph (honest, may be inconclusive).
// If execution and "after" share the same mesh_assessment_id, validation cannot represent independent snapshots (confounded).
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

	baselineID := strings.TrimSpace(exec.MeshAssessmentID)
	afterID := strings.TrimSpace(afterMI.AssessmentID)
	sameAssessment := baselineID != "" && afterID != "" && baselineID == afterID

	if !exec.BaselineMetrics.Captured && baselineID == "" {
		v.Lines = append(v.Lines, "No baseline mesh assessment id was recorded for this execution — before/after metrics may both reflect the same live compute; treat deltas as non-causal.")
	}
	if sameAssessment {
		v.Verdict = OutcomeVerdictConfounded
		v.Caveat = "Before and after reference the same mesh assessment id — not an independent before/after snapshot."
		v.Lines = append(v.Lines, v.Caveat)
		v.Lines = append(v.Lines, "If the baseline snapshot was pruned or never stored, validation is directional at best.")
		return v
	}

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
		v.Lines = append(v.Lines, "Concurrent operational changes can confound directional validation — this verdict is not proof of causality.")
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
		v.Lines = append(v.Lines, "Validation here is directional (topology-derived metrics); it is not RF coverage or propagation proof.")
	}
	return v
}
