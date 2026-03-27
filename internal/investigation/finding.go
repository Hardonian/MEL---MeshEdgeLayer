// Package investigation provides the canonical operator investigation and
// decision-support substrate for MEL. It unifies findings, evidence gaps,
// and recommendations across doctor/diagnostics/health/readiness surfaces
// into a single inspectable model.
//
// Design constraints:
//   - Findings are not diagnoses. They describe what MEL observes.
//   - Evidence gaps are first-class. Missing data constrains conclusions.
//   - Recommendations are symptom-level unless root cause is established.
//   - Scope and provenance are preserved (local vs imported vs merged).
//   - Repeated observations do not prove flooding/congestion.
//   - High attention may coexist with low certainty.
package investigation

import "time"

// FindingCategory classifies what surface produced the finding.
type FindingCategory string

const (
	CategoryTransport FindingCategory = "transport"
	CategoryDatabase  FindingCategory = "database"
	CategoryConfig    FindingCategory = "config"
	CategoryMesh      FindingCategory = "mesh"
	CategoryControl   FindingCategory = "control"
	CategoryStorage   FindingCategory = "storage"
	CategoryRetention FindingCategory = "retention"
	CategorySecurity  FindingCategory = "security"
	CategoryFleet     FindingCategory = "fleet"
	CategoryImport    FindingCategory = "import"
)

// AttentionLevel indicates how urgently the operator should look at this.
// It is deliberately separate from certainty.
type AttentionLevel string

const (
	AttentionCritical AttentionLevel = "critical"
	AttentionHigh     AttentionLevel = "high"
	AttentionMedium   AttentionLevel = "medium"
	AttentionLow      AttentionLevel = "low"
	AttentionInfo     AttentionLevel = "info"
)

// ScopePosture describes the provenance boundary of evidence.
type ScopePosture string

const (
	ScopeLocal          ScopePosture = "local"
	ScopeImported       ScopePosture = "imported"
	ScopeMerged         ScopePosture = "merged"
	ScopeHistoricalOnly ScopePosture = "historical_only"
	ScopePartialFleet   ScopePosture = "partial_fleet"
)

// Finding is the canonical observation unit for operator investigation.
// A Finding describes what MEL observes, not what it concludes.
type Finding struct {
	// ID is a stable, deterministic identifier for this finding instance.
	// Format: <code>:<scope>:<resource> (e.g. "transport_failed:local:serial-radio")
	ID string `json:"id"`

	// Code is the machine-readable finding type (e.g. "transport_failed",
	// "stale_reporters", "import_mismatch").
	Code string `json:"code"`

	// Category classifies the system surface that produced this finding.
	Category FindingCategory `json:"category"`

	// Attention indicates how urgently the operator should look at this.
	// Attention != certainty. A finding can be high-attention but low-certainty.
	Attention AttentionLevel `json:"attention"`

	// Certainty is a 0.0–1.0 score indicating how confident MEL is in this
	// finding based on available evidence. Low certainty means evidence is
	// partial, stale, or conflicting.
	Certainty float64 `json:"certainty"`

	// Title is the operator-readable one-line summary.
	Title string `json:"title"`

	// Explanation is the operator-readable multi-line explanation of what
	// MEL observes and why it matters.
	Explanation string `json:"explanation"`

	// WhyItMatters explains the operational significance in operator terms.
	WhyItMatters string `json:"why_it_matters"`

	// Scope describes the provenance boundary of the evidence backing this
	// finding. Findings from imported evidence are not equivalent to local
	// confirmation.
	Scope ScopePosture `json:"scope"`

	// ResourceID identifies the affected resource (transport name, node id, etc.).
	ResourceID string `json:"resource_id,omitempty"`

	// EvidenceIDs lists the specific evidence artifacts backing this finding.
	// These can be timeline event IDs, incident IDs, alert IDs, etc.
	EvidenceIDs []string `json:"evidence_ids,omitempty"`

	// EvidenceSnapshot captures key evidence values at observation time.
	EvidenceSnapshot map[string]any `json:"evidence_snapshot,omitempty"`

	// EvidenceGapIDs references any evidence gaps that limit this finding's
	// certainty. If non-empty, the finding's conclusions are constrained.
	EvidenceGapIDs []string `json:"evidence_gap_ids,omitempty"`

	// RecommendationIDs references typed recommendations for this finding.
	RecommendationIDs []string `json:"recommendation_ids,omitempty"`

	// ObservedAt is when MEL first observed the condition.
	ObservedAt string `json:"observed_at"`

	// GeneratedAt is when this finding was assembled.
	GeneratedAt string `json:"generated_at"`

	// Source identifies which MEL subsystem produced this finding.
	Source string `json:"source"`

	// OperatorActionRequired is true when operator intervention is needed.
	OperatorActionRequired bool `json:"operator_action_required"`

	// CanAutoRecover is true when MEL or the system can potentially recover
	// without operator intervention.
	CanAutoRecover bool `json:"can_auto_recover"`
}

// NewFinding creates a Finding with required fields and a deterministic ID.
func NewFinding(code string, category FindingCategory, attention AttentionLevel, certainty float64, title, explanation string, now time.Time) Finding {
	return Finding{
		ID:          code + ":" + string(category),
		Code:        code,
		Category:    category,
		Attention:   attention,
		Certainty:   certainty,
		Title:       title,
		Explanation: explanation,
		Scope:       ScopeLocal,
		GeneratedAt: now.UTC().Format(time.RFC3339),
		Source:      "investigation",
	}
}
