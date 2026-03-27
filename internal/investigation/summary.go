package investigation

// Summary is the canonical operator investigation object. It ties together
// findings, evidence gaps, and recommendations into a bounded, inspectable
// unit that an operator can navigate to understand what MEL sees, what
// evidence supports that, what is uncertain, and what to do next.
//
// Summary is derived, not persisted. It is assembled from current system
// state each time it is requested. This avoids stale investigation state
// and keeps the investigation substrate truthful.
type Summary struct {
	// GeneratedAt is when this summary was assembled.
	GeneratedAt string `json:"generated_at"`

	// OverallAttention is the highest attention level across all findings.
	OverallAttention AttentionLevel `json:"overall_attention"`

	// OverallCertainty is the lowest certainty across critical/high findings.
	// When certainty is low, the operator should investigate further before
	// acting.
	OverallCertainty float64 `json:"overall_certainty"`

	// Headline is a one-line operator-readable status summary.
	Headline string `json:"headline"`

	// AttentionSummary explains why MEL is asking the operator to look at
	// something, and explicitly distinguishes attention from certainty.
	AttentionSummary string `json:"attention_summary"`

	// Findings is the ordered list of canonical findings. Ordered by
	// attention (critical first), then by certainty (higher first).
	Findings []Finding `json:"findings"`

	// EvidenceGaps lists all identified evidence gaps that constrain findings
	// and recommendations. These are first-class — not hidden footnotes.
	EvidenceGaps []EvidenceGap `json:"evidence_gaps"`

	// Recommendations lists typed next-steps for the operator, ordered by
	// the findings/gaps that motivated them.
	Recommendations []Recommendation `json:"recommendations"`

	// Cases groups findings, gaps, recommendations, and raw-record links into
	// bounded operator attention objects.
	Cases []Case `json:"cases,omitempty"`

	// Counts provides aggregate counts for machine-readable consumers.
	Counts SummaryCounts `json:"counts"`

	// CaseCounts provides aggregate counts for case-oriented operator flows.
	CaseCounts CaseCounts `json:"case_counts"`

	// ScopePosture describes the overall evidence provenance.
	ScopePosture string `json:"scope_posture"`

	// PhysicsBoundary is a visible, never-hidden reminder of what MEL
	// cannot conclude from the available evidence.
	PhysicsBoundary PhysicsBoundary `json:"physics_boundary"`

	caseDetails map[string]CaseDetail
}

// SummaryCounts provides aggregate counts for machine-readable consumers.
type SummaryCounts struct {
	TotalFindings              int `json:"total_findings"`
	CriticalFindings           int `json:"critical_findings"`
	HighFindings               int `json:"high_findings"`
	MediumFindings             int `json:"medium_findings"`
	LowFindings                int `json:"low_findings"`
	InfoFindings               int `json:"info_findings"`
	EvidenceGaps               int `json:"evidence_gaps"`
	Recommendations            int `json:"recommendations"`
	OperatorActionRequired     int `json:"operator_action_required"`
	FindingsConstrainedByGaps  int `json:"findings_constrained_by_gaps"`
	RecommendationsWithCaveats int `json:"recommendations_with_caveats"`
}

// PhysicsBoundary reminds operators what MEL cannot determine from
// available evidence. This is always present and never collapsed.
type PhysicsBoundary struct {
	// Statements is the list of physics/network reality constraints that
	// bound all conclusions in this investigation.
	Statements []string `json:"statements"`
}

// DefaultPhysicsBoundary returns the standard physics/uncertainty boundary
// that applies to every investigation.
func DefaultPhysicsBoundary() PhysicsBoundary {
	return PhysicsBoundary{
		Statements: []string{
			"Repeated multi-observer reports do not prove flooding or congestion.",
			"Symptom patterns do not prove root cause.",
			"Missing observations do not prove absence of the observed condition.",
			"Imported historical evidence is not current local proof.",
			"Merged patterns are bounded by observer coverage and timing posture.",
			"Stale reporters reduce certainty of all findings that depend on them.",
			"Path, topology, and coverage remain evidence-bounded and often partial.",
			"Event ordering across sources is best-effort; no global total order is implied.",
		},
	}
}
