package investigation

// EvidenceGap represents a specific piece of missing, stale, or ambiguous
// evidence that limits MEL's ability to draw conclusions. Evidence gaps are
// first-class — they are not footnotes. They constrain findings and
// recommendations.
type EvidenceGap struct {
	// ID is a stable, deterministic identifier.
	// Format: <reason>:<scope>:<resource> (e.g. "missing_expected_reporters:local:serial-radio")
	ID string `json:"id"`

	// Reason is the typed reason code for this evidence gap.
	Reason EvidenceGapReason `json:"reason"`

	// Title is the operator-readable one-line summary.
	Title string `json:"title"`

	// Explanation describes what evidence is missing and why it matters.
	Explanation string `json:"explanation"`

	// Impact describes how this gap constrains conclusions.
	Impact string `json:"impact"`

	// Scope describes the provenance boundary of this gap.
	Scope ScopePosture `json:"scope"`

	// ResourceID identifies the affected resource if applicable.
	ResourceID string `json:"resource_id,omitempty"`

	// ConstrainedFindingIDs lists findings whose certainty is reduced by
	// this gap.
	ConstrainedFindingIDs []string `json:"constrained_finding_ids,omitempty"`

	// ConstrainedRecommendationIDs lists recommendations limited by this gap.
	ConstrainedRecommendationIDs []string `json:"constrained_recommendation_ids,omitempty"`

	// GeneratedAt is when this gap was identified.
	GeneratedAt string `json:"generated_at"`
}

// EvidenceGapReason is a typed reason code for evidence gaps.
type EvidenceGapReason string

const (
	// GapMissingExpectedReporters means configured transports or expected
	// nodes have not reported recently. Absence of evidence is not evidence
	// of absence.
	GapMissingExpectedReporters EvidenceGapReason = "missing_expected_reporters"

	// GapStaleContributors means some reporters have not sent fresh data
	// within their expected interval. Conclusions based on their last-known
	// state may be outdated.
	GapStaleContributors EvidenceGapReason = "stale_contributors"

	// GapImportedHistoricalOnly means the only evidence for a condition
	// comes from an imported historical bundle, not current local observation.
	GapImportedHistoricalOnly EvidenceGapReason = "imported_historical_only"

	// GapOrderingUncertain means event ordering across sources is best-effort.
	// No global total order is established.
	GapOrderingUncertain EvidenceGapReason = "ordering_uncertain"

	// GapObserverConflict means different observers report contradictory
	// information about the same resource or condition.
	GapObserverConflict EvidenceGapReason = "observer_conflict"

	// GapScopeIncomplete means the system does not have visibility into all
	// relevant nodes, transports, or segments.
	GapScopeIncomplete EvidenceGapReason = "scope_incomplete"

	// GapAuthenticityUnverified means evidence was accepted without
	// cryptographic verification of its origin.
	GapAuthenticityUnverified EvidenceGapReason = "authenticity_unverified"

	// GapNoLocalConfirmation means a condition was reported by a remote or
	// imported source but has not been independently confirmed by this
	// instance's own observations.
	GapNoLocalConfirmation EvidenceGapReason = "no_local_confirmation"

	// GapNoRouteProof means path, topology, or route information is partial
	// or absent. Coverage claims cannot be validated.
	GapNoRouteProof EvidenceGapReason = "no_route_proof"

	// GapDatabaseDegraded means the local database is unreachable or in
	// degraded mode, limiting evidence availability.
	GapDatabaseDegraded EvidenceGapReason = "database_degraded"
)

// NewEvidenceGap creates an EvidenceGap with required fields.
func NewEvidenceGap(reason EvidenceGapReason, title, explanation, impact string, scope ScopePosture, generatedAt string) EvidenceGap {
	return EvidenceGap{
		ID:          string(reason) + ":" + string(scope),
		Reason:      reason,
		Title:       title,
		Explanation: explanation,
		Impact:      impact,
		Scope:       scope,
		GeneratedAt: generatedAt,
	}
}
