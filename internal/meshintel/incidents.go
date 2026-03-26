package meshintel

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/models"
)

const (
	minMessagesForViabilityIncident = int64(40)
	consecutiveBadThreshold         = 3
	readinessDropThreshold          = 0.12
)

func viabilityIsBad(v LocalMeshViabilityClassification) bool {
	switch v {
	case ViabilityIsolated, ViabilityWeakBootstrap, ViabilityUnstableIntermittent:
		return true
	default:
		return false
	}
}

func viabilityWasHealthyBaseline(v string) bool {
	s := strings.TrimSpace(v)
	return s == string(ViabilityViableLocalMesh) || s == string(ViabilityEmergingCluster)
}

// EvaluateViabilityRegression updates streak state and may open a single conservative incident
// when observed mesh viability regresses from a previously healthier baseline (not for first-time lone wolves).
func EvaluateViabilityRegression(d *db.DB, a Assessment, sig MessageSignals, transportOK bool) error {
	if d == nil || !a.TopologyEnabled {
		return nil
	}
	st, err := LoadMeshIntelState(d)
	if err != nil {
		return err
	}
	v := a.Bootstrap.Viability
	readiness := a.Bootstrap.BootstrapReadinessScore
	vStr := string(v)

	if !viabilityIsBad(v) {
		st.ConsecutiveBad = 0
		st.LastGoodViability = vStr
		st.LastGoodReadiness = readiness
		st.LastViability = vStr
		st.LastIncidentFingerprint = ""
		return SaveMeshIntelState(d, st)
	}

	st.ConsecutiveBad++
	st.LastViability = vStr

	if st.ConsecutiveBad < consecutiveBadThreshold {
		return SaveMeshIntelState(d, st)
	}
	if sig.TotalMessages < minMessagesForViabilityIncident {
		return SaveMeshIntelState(d, st)
	}
	if !transportOK {
		return SaveMeshIntelState(d, st)
	}
	if !viabilityWasHealthyBaseline(st.LastGoodViability) {
		return SaveMeshIntelState(d, st)
	}
	if readiness > st.LastGoodReadiness-readinessDropThreshold {
		// Not a material drop vs stored baseline
		return SaveMeshIntelState(d, st)
	}

	fp := regressionFingerprint(vStr, a.GraphHash, sig.TotalMessages, readiness, st.LastGoodReadiness, st.LastGoodViability)
	if fp == st.LastIncidentFingerprint {
		return SaveMeshIntelState(d, st)
	}

	incID := "meshintel-viability-regression"
	title := "Mesh viability regression (observed)"
	summary := fmt.Sprintf(
		"Bootstrap viability=%s for %d consecutive topology refreshes; readiness %.2f vs prior healthy baseline %.2f (%s). Evidence: %d messages in rollup window. Packet-derived graph — not RF proof.",
		vStr, st.ConsecutiveBad, readiness, st.LastGoodReadiness, st.LastGoodViability, sig.TotalMessages,
	)
	inc := models.Incident{
		ID:           incID,
		Category:     "mesh_topology",
		Severity:     "warning",
		Title:        title,
		Summary:      summary,
		ResourceType: "mesh",
		ResourceID:   "bootstrap",
		State:        "open",
		ActorID:      "mel",
		OccurredAt:   time.Now().UTC().Format(time.RFC3339),
		Metadata: map[string]any{
			"source":                "mesh_intel",
			"viability":             vStr,
			"consecutive_bad":       st.ConsecutiveBad,
			"readiness":             readiness,
			"last_good_viability":   st.LastGoodViability,
			"last_good_readiness":   st.LastGoodReadiness,
			"assessment_id":         a.AssessmentID,
			"graph_hash":            a.GraphHash,
			"messages_in_window":    sig.TotalMessages,
			"evidence_note":         "Advisory only; verify RF path and transport before acting.",
		},
	}
	if err := d.UpsertIncident(inc); err != nil {
		return err
	}
	st.LastIncidentFingerprint = fp
	st.ConsecutiveBad = 0
	return SaveMeshIntelState(d, st)
}

func regressionFingerprint(v, graphHash string, total int64, readiness, lastGood float64, lastGoodV string) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%d|%.4f|%.4f|%s", v, graphHash, total, readiness, lastGood, lastGoodV)
	sum := h.Sum(nil)
	return fmt.Sprintf("%x", sum)
}
