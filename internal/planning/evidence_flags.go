package planning

// PlanningEvidenceFlags exposes machine-readable uncertainty posture for operator-facing clients.
// Flags capture evidence quality constraints; they do not prove causality.
type PlanningEvidenceFlags struct {
	BaselineMissing                            bool `json:"baseline_missing"`
	ConfoundedSameAssessmentContext            bool `json:"confounded_same_assessment_context"`
	DirectionalOnly                            bool `json:"directional_only"`
	Inconclusive                               bool `json:"inconclusive"`
	TopologyOrGraphDriftDetected               bool `json:"topology_or_graph_drift_detected"`
	LimitedConfidence                          bool `json:"limited_confidence"`
	NoAdvisories                               bool `json:"no_advisories"`
	RecommendationPresentWithUncertainEvidence bool `json:"recommendation_present_with_uncertain_evidence"`
}
