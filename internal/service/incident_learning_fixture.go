package service

import (
	"fmt"
	"strings"
)

// IncidentLearningFixture returns a deterministic JSON-serializable bundle for regression tests
// and golden fixtures (fingerprints, similarity slice, ranking head). Does not include live secrets.
func (a *App) IncidentLearningFixture(incidentID string) (map[string]any, error) {
	if a == nil || a.DB == nil {
		return nil, fmt.Errorf("service not available")
	}
	incidentID = strings.TrimSpace(incidentID)
	if incidentID == "" {
		return nil, fmt.Errorf("incident id is required")
	}
	inc, ok, err := a.IncidentByID(incidentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("incident not found: %s", incidentID)
	}
	intel := inc.Intelligence
	out := map[string]any{
		"fixture_kind": "mel.incident_learning_fixture/v1",
		"incident_id":  inc.ID,
	}
	if intel == nil {
		out["intelligence"] = nil
		return out, nil
	}
	fp := map[string]any{}
	if intel.Fingerprint != nil {
		fp["canonical_hash"] = intel.Fingerprint.CanonicalHash
		fp["schema_version"] = intel.Fingerprint.SchemaVersion
		fp["profile_version"] = intel.Fingerprint.ProfileVersion
	}
	out["fingerprint"] = fp
	sim := make([]map[string]any, 0, len(intel.SimilarIncidents))
	for _, s := range intel.SimilarIncidents {
		sim = append(sim, map[string]any{
			"incident_id":           s.IncidentID,
			"match_category":        s.MatchCategory,
			"weighted_score":        s.WeightedScore,
			"insufficient_evidence": s.InsufficientEvidence,
		})
	}
	out["similarity_head"] = sim
	rank := make([]map[string]any, 0, len(intel.RunbookRecommendations))
	for _, r := range intel.RunbookRecommendations {
		rank = append(rank, map[string]any{
			"id":         r.ID,
			"strength":   r.Strength,
			"rank_score": r.RankScore,
			"suppressed": r.Suppressed,
		})
	}
	out["recommendation_rank_head"] = rank
	return out, nil
}
