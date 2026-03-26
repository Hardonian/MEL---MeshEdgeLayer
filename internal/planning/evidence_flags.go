package planning

// PlanningEvidenceFlags exposes machine-readable uncertainty posture for operator-facing clients.
// Flags capture evidence quality constraints; they do not prove causality.
type PlanningEvidenceFlags struct {
	BaselineMissing                            bool `json:"baseline_missing,omitempty"`
	ConfoundedSameAssessmentContext            bool `json:"confounded_same_assessment_context,omitempty"`
	DirectionalOnly                            bool `json:"directional_only,omitempty"`
	Inconclusive                               bool `json:"inconclusive,omitempty"`
	TopologyOrGraphDriftDetected               bool `json:"topology_or_graph_drift_detected,omitempty"`
	LimitedConfidence                          bool `json:"limited_confidence,omitempty"`
	NoAdvisories                               bool `json:"no_advisories,omitempty"`
	RecommendationPresentWithUncertainEvidence bool `json:"recommendation_present_with_uncertain_evidence,omitempty"`
}
