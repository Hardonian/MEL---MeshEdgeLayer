package investigation

import "time"

// CaseKind classifies a bounded operator investigation problem.
type CaseKind string

const (
	CaseTransportDegradation   CaseKind = "transport_degradation"
	CaseEvidenceFreshnessGap   CaseKind = "evidence_freshness_gap"
	CasePartialFleetVisibility CaseKind = "partial_fleet_visibility"
	CaseImportedHistorical     CaseKind = "imported_historical_attention"
	CaseIncidentCandidate      CaseKind = "incident_candidate"
)

// CaseStatus describes how MEL is presenting the case to the operator.
type CaseStatus string

const (
	CaseStatusActiveAttention CaseStatus = "active_attention"
	CaseStatusMonitoring      CaseStatus = "monitoring"
	CaseStatusHistoricalOnly  CaseStatus = "historical_context"
)

// RelatedRecordKind classifies inspectable objects linked from a case.
type RelatedRecordKind string

const (
	RecordTransportRuntime RelatedRecordKind = "transport_runtime"
	RecordTransportAlert   RelatedRecordKind = "transport_alert"
	RecordTimelineEvent    RelatedRecordKind = "timeline_event"
	RecordIncident         RelatedRecordKind = "incident"
	RecordControlAction    RelatedRecordKind = "control_action"
	RecordImportedEvidence RelatedRecordKind = "imported_remote_evidence"
	RecordImportBatch      RelatedRecordKind = "remote_import_batch"
	RecordFleetTruth       RelatedRecordKind = "fleet_truth"
	RecordStatusSnapshot   RelatedRecordKind = "status_snapshot"
)

// RelatedRecord is a typed edge from an investigation case to an inspectable
// underlying record. These are relationship references, not a second source of
// truth.
type RelatedRecord struct {
	Kind       RelatedRecordKind `json:"kind"`
	ID         string            `json:"id"`
	Relation   string            `json:"relation"`
	Summary    string            `json:"summary"`
	Scope      ScopePosture      `json:"scope"`
	InspectCLI string            `json:"inspect_cli,omitempty"`
	InspectAPI string            `json:"inspect_api,omitempty"`
}

// Case is the canonical operator attention object. Cases are derived from
// findings, evidence gaps, and recommendations, then linked back to raw
// timeline/import/incident/transport records.
type Case struct {
	ID                string          `json:"id"`
	Kind              CaseKind        `json:"kind"`
	Status            CaseStatus      `json:"status"`
	Attention         AttentionLevel  `json:"attention"`
	Certainty         float64         `json:"certainty"`
	Title             string          `json:"title"`
	Summary           string          `json:"summary"`
	AttentionReason   string          `json:"attention_reason"`
	WhyItMatters      string          `json:"why_it_matters"`
	Scope             ScopePosture    `json:"scope"`
	HistoricalOnly    bool            `json:"historical_only"`
	CurrentEvidence   bool            `json:"current_evidence"`
	FindingIDs        []string        `json:"finding_ids,omitempty"`
	EvidenceGapIDs    []string        `json:"evidence_gap_ids,omitempty"`
	RecommendationIDs []string        `json:"recommendation_ids,omitempty"`
	RelatedRecords    []RelatedRecord `json:"related_records,omitempty"`
	SafeToConsider    string          `json:"safe_to_consider"`
	OutOfScope        string          `json:"out_of_scope"`
	MissingEvidence   string          `json:"missing_evidence,omitempty"`
	ObservedAt        string          `json:"observed_at"`
	UpdatedAt         string          `json:"updated_at"`
	Source            string          `json:"source"`
}

// CaseCounts provides aggregate counts for machine-readable consumers.
type CaseCounts struct {
	TotalCases                int `json:"total_cases"`
	ActiveAttentionCases      int `json:"active_attention_cases"`
	MonitoringCases           int `json:"monitoring_cases"`
	HistoricalOnlyCases       int `json:"historical_only_cases"`
	CasesConstrainedByGaps    int `json:"cases_constrained_by_gaps"`
	HighAttentionLowCertainty int `json:"high_attention_low_certainty"`
}

// CaseDetail expands a case into the linked canonical findings, gaps, and
// recommendations.
type CaseDetail struct {
	Case            Case             `json:"case"`
	Findings        []Finding        `json:"findings,omitempty"`
	EvidenceGaps    []EvidenceGap    `json:"evidence_gaps,omitempty"`
	Recommendations []Recommendation `json:"recommendations,omitempty"`
}

// NewCase creates a Case with the required timestamps.
func NewCase(id string, kind CaseKind, status CaseStatus, attention AttentionLevel, certainty float64, title, summary, updatedAt string) Case {
	return Case{
		ID:         id,
		Kind:       kind,
		Status:     status,
		Attention:  attention,
		Certainty:  certainty,
		Title:      title,
		Summary:    summary,
		ObservedAt: updatedAt,
		UpdatedAt:  updatedAt,
		Source:     "investigation",
	}
}

// CaseDetail returns the expanded detail view for a single case ID.
func (s Summary) CaseDetail(id string) (CaseDetail, bool) {
	findingByID := make(map[string]Finding, len(s.Findings))
	for _, finding := range s.Findings {
		findingByID[finding.ID] = finding
	}
	gapByID := make(map[string]EvidenceGap, len(s.EvidenceGaps))
	for _, gap := range s.EvidenceGaps {
		gapByID[gap.ID] = gap
	}
	recByID := make(map[string]Recommendation, len(s.Recommendations))
	for _, rec := range s.Recommendations {
		recByID[rec.ID] = rec
	}
	for _, c := range s.Cases {
		if c.ID != id {
			continue
		}
		detail := CaseDetail{Case: c}
		for _, findingID := range c.FindingIDs {
			if finding, ok := findingByID[findingID]; ok {
				detail.Findings = append(detail.Findings, finding)
			}
		}
		for _, gapID := range c.EvidenceGapIDs {
			if gap, ok := gapByID[gapID]; ok {
				detail.EvidenceGaps = append(detail.EvidenceGaps, gap)
			}
		}
		for _, recID := range c.RecommendationIDs {
			if rec, ok := recByID[recID]; ok {
				detail.Recommendations = append(detail.Recommendations, rec)
			}
		}
		return detail, true
	}
	return CaseDetail{}, false
}

// CaseDetails returns expanded detail views for every case in summary order.
func (s Summary) CaseDetails() []CaseDetail {
	out := make([]CaseDetail, 0, len(s.Cases))
	for _, c := range s.Cases {
		detail, ok := s.CaseDetail(c.ID)
		if ok {
			out = append(out, detail)
		}
	}
	return out
}

func caseCounts(cases []Case) CaseCounts {
	counts := CaseCounts{TotalCases: len(cases)}
	for _, c := range cases {
		switch c.Status {
		case CaseStatusActiveAttention:
			counts.ActiveAttentionCases++
		case CaseStatusMonitoring:
			counts.MonitoringCases++
		case CaseStatusHistoricalOnly:
			counts.HistoricalOnlyCases++
		}
		if len(c.EvidenceGapIDs) > 0 {
			counts.CasesConstrainedByGaps++
		}
		if (c.Attention == AttentionCritical || c.Attention == AttentionHigh) && c.Certainty < 0.8 {
			counts.HighAttentionLowCertainty++
		}
	}
	return counts
}

func newestTimestamp(current, candidate string) string {
	if candidate == "" {
		return current
	}
	if current == "" {
		return candidate
	}
	currentTime, currentErr := time.Parse(time.RFC3339, current)
	candidateTime, candidateErr := time.Parse(time.RFC3339, candidate)
	if currentErr != nil || candidateErr != nil {
		if candidate > current {
			return candidate
		}
		return current
	}
	if candidateTime.After(currentTime) {
		return candidate
	}
	return current
}
