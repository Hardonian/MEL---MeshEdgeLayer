package intelligence

import (
	"time"
)

)

// Recommendation represents a safe next step for an operator
type Recommendation struct {
	Code              string         `json:"code"`
	Action            string         `json:"action"`
	Rationale         string         `json:"rationale"`
	EvidenceReference []string       `json:"evidence_reference"`
	Confidence        float64        `json:"confidence"` // 0.0 to 1.0
	RiskLevel         string         `json:"risk_level"` // 'low', 'medium', 'high', 'unknown'
	Reversibility     string         `json:"reversibility"` // 'high', 'medium', 'low', 'none'
	Prerequisites     []string       `json:"prerequisites,omitempty"`
	CanAutomate       bool           `json:"can_automate"`
	WhyNotAct         string         `json:"why_not_act,omitempty"`
}

// StateComparison represents a diff between current and last-known-good state
type StateComparison struct {
	CurrentTime       time.Time      `json:"current_time"`
	BaselineTime      time.Time      `json:"baseline_time"`
	BaselineName      string         `json:"baseline_name"` // e.g. 'last-known-good'
	ChangedComponents []string       `json:"changed_components"`
	Regressions       []string       `json:"regressions"`
	Diff              map[string]any `json:"diff"`
}
