package investigation

// Recommendation is a typed operator next-step grounded in evidence.
//
// Recommendations tell the operator what to inspect, verify, or monitor.
// They do NOT imply MEL can execute the action unless ActionAuthority is
// "mel_executable". They do NOT imply root cause unless the evidence
// contract supports it.
type Recommendation struct {
	// ID is a stable, deterministic identifier.
	ID string `json:"id"`

	// Code is the machine-readable recommendation type.
	Code RecommendationCode `json:"code"`

	// Action is the operator-readable description of what to do.
	Action string `json:"action"`

	// Rationale explains why this recommendation is being made,
	// grounded in specific evidence.
	Rationale string `json:"rationale"`

	// EvidenceBasis lists the specific evidence backing this recommendation.
	EvidenceBasis []string `json:"evidence_basis,omitempty"`

	// FindingIDs links this recommendation to the findings that triggered it.
	FindingIDs []string `json:"finding_ids,omitempty"`

	// UncertaintyLimits describes what this recommendation cannot determine
	// due to evidence gaps. Empty means no known limitations.
	UncertaintyLimits []string `json:"uncertainty_limits,omitempty"`

	// EvidenceGapIDs lists evidence gaps that constrain this recommendation.
	EvidenceGapIDs []string `json:"evidence_gap_ids,omitempty"`

	// ActionAuthority describes who or what can perform this action.
	// Values: "operator_only", "operator_verify", "mel_executable"
	// Most recommendations are operator_only or operator_verify.
	ActionAuthority string `json:"action_authority"`

	// Scope describes the provenance boundary relevant to this recommendation.
	Scope ScopePosture `json:"scope"`

	// GeneratedAt is when this recommendation was assembled.
	GeneratedAt string `json:"generated_at"`
}

// RecommendationCode is a typed category for recommendations.
type RecommendationCode string

const (
	RecInspectTransport       RecommendationCode = "inspect_transport"
	RecVerifyReporter         RecommendationCode = "verify_reporter"
	RecCompareLocalVsImported RecommendationCode = "compare_local_vs_imported"
	RecInspectMergeConflict   RecommendationCode = "inspect_merge_conflict"
	RecVerifySiteBackhaul     RecommendationCode = "verify_site_backhaul"
	RecCollectMoreEvidence    RecommendationCode = "collect_more_evidence"
	RecTreatAsHistoricalOnly  RecommendationCode = "treat_as_historical_only"
	RecNoSafeConclusionYet    RecommendationCode = "no_safe_conclusion_yet"
	RecVerifyDatabaseHealth   RecommendationCode = "verify_database_health"
	RecCheckConfig            RecommendationCode = "check_config"
	RecCheckMeshDevices       RecommendationCode = "check_mesh_devices"
	RecWaitForFreshEvidence   RecommendationCode = "wait_for_fresh_evidence"
	RecRunDiagnostics         RecommendationCode = "run_diagnostics"
	RecReviewControlActions   RecommendationCode = "review_control_actions"
	RecInspectDeadLetters     RecommendationCode = "inspect_dead_letters"
	RecInspectAuditLog        RecommendationCode = "inspect_audit_log"
	RecMonitorForChanges      RecommendationCode = "monitor_for_changes"
	RecSecurityReview         RecommendationCode = "security_review"
)

// NewRecommendation creates a Recommendation with required fields.
func NewRecommendation(code RecommendationCode, action, rationale, authority string, scope ScopePosture, generatedAt string) Recommendation {
	return Recommendation{
		ID:              string(code),
		Code:            code,
		Action:          action,
		Rationale:       rationale,
		ActionAuthority: authority,
		Scope:           scope,
		GeneratedAt:     generatedAt,
	}
}
