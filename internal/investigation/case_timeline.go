package investigation

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/fleet"
)

// CaseTimingPosture is the normalized ordering posture for case reconstruction.
// It is intentionally broader than raw event posture so operators can see when
// a case sequence is exact, source-local, or only best-effort.
type CaseTimingPosture string

const (
	CaseTimingLocallyOrdered                CaseTimingPosture = "locally_ordered"
	CaseTimingImportedPreservedOrder        CaseTimingPosture = "imported_preserved_order"
	CaseTimingMergedBestEffortOrder         CaseTimingPosture = "merged_best_effort_order"
	CaseTimingOrderingUncertainClockSkew    CaseTimingPosture = "ordering_uncertain_clock_skew"
	CaseTimingOrderingUncertainMissingTS    CaseTimingPosture = "ordering_uncertain_missing_timestamps"
	CaseTimingHistoricalImportNotLive       CaseTimingPosture = "historical_import_not_live"
	CaseTimingMixedFreshnessWindow          CaseTimingPosture = "mixed_freshness_window"
	CaseTimingSourceOrderGlobalOrderUnknown CaseTimingPosture = "source_order_known_global_order_unknown"
)

// CaseTimingSummary explains how to read the case's chronology.
type CaseTimingSummary struct {
	PrimaryPosture     CaseTimingPosture   `json:"primary_posture"`
	Postures           []CaseTimingPosture `json:"postures,omitempty"`
	ExactSequence      bool                `json:"exact_sequence"`
	TimestampAxes      []string            `json:"timestamp_axes,omitempty"`
	LocalEventCount    int                 `json:"local_event_count,omitempty"`
	ImportedEventCount int                 `json:"imported_event_count,omitempty"`
	BestEffortCount    int                 `json:"best_effort_count,omitempty"`
	Note               string              `json:"note"`
}

// CaseEventRelationType classifies how a raw canonical event relates to a case.
type CaseEventRelationType string

const (
	CaseEventRelationTransportContext    CaseEventRelationType = "transport_context"
	CaseEventRelationSupportingEvidence  CaseEventRelationType = "supporting_evidence"
	CaseEventRelationConflictingEvidence CaseEventRelationType = "conflicting_evidence"
	CaseEventRelationImportValidation    CaseEventRelationType = "import_validation"
	CaseEventRelationMergeDisposition    CaseEventRelationType = "merge_disposition"
	CaseEventRelationIncidentContext     CaseEventRelationType = "incident_context"
	CaseEventRelationControlContext      CaseEventRelationType = "control_context"
	CaseEventRelationOperationalContext  CaseEventRelationType = "operational_context"
	CaseEventRelationFreshnessContext    CaseEventRelationType = "freshness_context"
	CaseEventRelationHistoricalContext   CaseEventRelationType = "historical_context"
)

// CaseEventContribution keeps relatedness distinct from causality.
type CaseEventContribution string

const (
	CaseEventContributionSupporting  CaseEventContribution = "supporting"
	CaseEventContributionConflicting CaseEventContribution = "conflicting"
	CaseEventContributionContextual  CaseEventContribution = "contextual"
	CaseEventContributionUncertain   CaseEventContribution = "uncertainty"
)

// CaseEventTimeContext preserves the timing axes that shaped one linked event.
type CaseEventTimeContext struct {
	OccurredAt string `json:"occurred_at,omitempty"`
	ObservedAt string `json:"observed_at,omitempty"`
	ReceivedAt string `json:"received_at,omitempty"`
	RecordedAt string `json:"recorded_at,omitempty"`
	ImportedAt string `json:"imported_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}

// CaseEventLink is a raw canonical event linked into case context.
type CaseEventLink struct {
	EventID          string                `json:"event_id"`
	EventType        string                `json:"event_type"`
	Summary          string                `json:"summary"`
	Severity         string                `json:"severity,omitempty"`
	RelationType     CaseEventRelationType `json:"relation_type"`
	Contribution     CaseEventContribution `json:"contribution"`
	Scope            ScopePosture          `json:"scope"`
	EventTime        string                `json:"event_time"`
	TimeBasis        string                `json:"time_basis"`
	TimeContext      CaseEventTimeContext  `json:"time_context,omitempty"`
	OrderingPosture  CaseTimingPosture     `json:"ordering_posture"`
	RawTimingPosture string                `json:"raw_timing_posture,omitempty"`
	OriginInstanceID string                `json:"origin_instance_id,omitempty"`
	ImportID         string                `json:"import_id,omitempty"`
	MergeDisposition string                `json:"merge_disposition,omitempty"`
	InspectCLI       string                `json:"inspect_cli,omitempty"`
	InspectAPI       string                `json:"inspect_api,omitempty"`
	Note             string                `json:"note,omitempty"`
}

// CaseEvolutionReason is a typed reason explaining how the current case
// posture was shaped by evidence, gaps, imports, or merge outcomes.
type CaseEvolutionReason string

const (
	CaseEvolutionCaseMaterialized                   CaseEvolutionReason = "case_materialized"
	CaseEvolutionNewSupportingEvidence              CaseEvolutionReason = "new_supporting_evidence"
	CaseEvolutionImportedSupportingEvidence         CaseEvolutionReason = "imported_supporting_evidence"
	CaseEvolutionImportedConflictingEvidence        CaseEvolutionReason = "imported_conflicting_evidence"
	CaseEvolutionReporterFreshnessDegraded          CaseEvolutionReason = "reporter_freshness_degraded"
	CaseEvolutionExpectedReporterMissing            CaseEvolutionReason = "expected_reporter_missing"
	CaseEvolutionImportValidationOutcome            CaseEvolutionReason = "import_validation_outcome"
	CaseEvolutionMergeDispositionChanged            CaseEvolutionReason = "merge_disposition_changed"
	CaseEvolutionUncertaintyIncreased               CaseEvolutionReason = "uncertainty_increased"
	CaseEvolutionUncertaintyReduced                 CaseEvolutionReason = "uncertainty_reduced"
	CaseEvolutionRecommendationChangedEvidenceDelta CaseEvolutionReason = "recommendation_changed_due_to_evidence_delta"
	CaseEvolutionTransportStateContext              CaseEvolutionReason = "transport_state_context"
)

// CaseReconstructionBasis tells operators whether a timeline entry came from an
// exact event, exact durable record, or current-state reconstruction.
type CaseReconstructionBasis string

const (
	CaseReconstructionExactEvent       CaseReconstructionBasis = "exact_event"
	CaseReconstructionExactRecord      CaseReconstructionBasis = "exact_record_snapshot"
	CaseReconstructionInferredSequence CaseReconstructionBasis = "inferred_from_ordered_evidence"
	CaseReconstructionCurrentState     CaseReconstructionBasis = "derived_from_current_case_state"
)

// CaseEvolutionEntry is a bounded, typed case-evolution moment. These entries
// summarize how the current case posture was shaped and always link back to
// inspectable evidence or explicit case-state reasoning.
type CaseEvolutionEntry struct {
	EntryID             string                  `json:"entry_id"`
	Summary             string                  `json:"summary"`
	ReasonCodes         []CaseEvolutionReason   `json:"reason_codes"`
	Scope               ScopePosture            `json:"scope"`
	OccurredAt          string                  `json:"occurred_at"`
	TimeBasis           string                  `json:"time_basis"`
	OrderingPosture     CaseTimingPosture       `json:"ordering_posture"`
	ReconstructionBasis CaseReconstructionBasis `json:"reconstruction_basis"`
	Status              CaseStatus              `json:"status"`
	Attention           AttentionLevel          `json:"attention"`
	Certainty           float64                 `json:"certainty"`
	FindingIDs          []string                `json:"finding_ids,omitempty"`
	EvidenceGapIDs      []string                `json:"evidence_gap_ids,omitempty"`
	RecommendationIDs   []string                `json:"recommendation_ids,omitempty"`
	RelatedEventIDs     []string                `json:"related_event_ids,omitempty"`
	Note                string                  `json:"note,omitempty"`
}

func buildCaseDetails(cases []Case, findings []Finding, gaps []EvidenceGap, recs []Recommendation, ctx deriveContext, generatedAt string) map[string]CaseDetail {
	findingByID := make(map[string]Finding, len(findings))
	for _, finding := range findings {
		findingByID[finding.ID] = finding
	}
	gapByID := make(map[string]EvidenceGap, len(gaps))
	for _, gap := range gaps {
		gapByID[gap.ID] = gap
	}
	recByID := make(map[string]Recommendation, len(recs))
	for _, rec := range recs {
		recByID[rec.ID] = rec
	}

	out := make(map[string]CaseDetail, len(cases))
	for _, c := range cases {
		detail := buildBaseCaseDetail(c, findingByID, gapByID, recByID)
		detail.LinkedEvents = buildCaseLinkedEvents(detail, ctx)
		detail.Case.Timing = summarizeCaseTiming(detail, generatedAt)
		detail.Evolution = buildCaseEvolution(detail, ctx, generatedAt)
		detail.Case.LinkedEventCount = len(detail.LinkedEvents)
		detail.Case.EvolutionCount = len(detail.Evolution)
		out[detail.Case.ID] = detail
	}
	return out
}

func buildBaseCaseDetail(c Case, findingByID map[string]Finding, gapByID map[string]EvidenceGap, recByID map[string]Recommendation) CaseDetail {
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
	return detail
}

func buildCaseLinkedEvents(detail CaseDetail, ctx deriveContext) []CaseEventLink {
	if len(ctx.timeline) == 0 {
		return nil
	}

	evidenceIDs := make(map[string]struct{})
	for _, finding := range detail.Findings {
		for _, evidenceID := range finding.EvidenceIDs {
			evidenceID = strings.TrimSpace(evidenceID)
			if evidenceID == "" {
				continue
			}
			evidenceIDs[evidenceID] = struct{}{}
		}
	}
	for _, record := range detail.Case.RelatedRecords {
		id := strings.TrimSpace(record.ID)
		if id == "" {
			continue
		}
		evidenceIDs[id] = struct{}{}
	}

	seen := map[string]struct{}{}
	out := make([]CaseEventLink, 0, len(ctx.timeline))
	for _, event := range ctx.timeline {
		if !eventBelongsToCase(detail, event, evidenceIDs) {
			continue
		}
		if _, ok := seen[event.EventID]; ok {
			continue
		}
		seen[event.EventID] = struct{}{}
		out = append(out, buildCaseEventLink(detail.Case, event))
	}

	sort.SliceStable(out, func(i, j int) bool {
		return caseTimeLess(out[i].EventTime, out[j].EventTime, out[i].EventID, out[j].EventID)
	})
	return out
}

func eventBelongsToCase(detail CaseDetail, event db.TimelineEvent, evidenceIDs map[string]struct{}) bool {
	if _, ok := evidenceIDs[event.EventID]; ok {
		return true
	}
	if _, ok := evidenceIDs[event.ResourceID]; ok {
		return true
	}
	if _, ok := evidenceIDs[event.ImportID]; ok {
		return true
	}

	switch detail.Case.Kind {
	case CaseTransportDegradation:
		transportName := strings.TrimPrefix(detail.Case.ID, "case:transport:")
		return timelineMatchesTransport(event, transportName)
	case CaseEvidenceFreshnessGap:
		return event.EventID == latestLinkedTimelineID(detail.Case.RelatedRecords)
	case CasePartialFleetVisibility, CaseImportedHistorical:
		return strings.HasPrefix(event.EventType, "remote_")
	case CaseIncidentCandidate:
		incidentID := strings.TrimPrefix(detail.Case.ID, "case:incident:")
		if event.EventID == incidentID || event.ResourceID == incidentID {
			return true
		}
		for _, record := range detail.Case.RelatedRecords {
			if record.Kind != RecordControlAction {
				continue
			}
			if event.EventID == record.ID || event.ResourceID == record.ID {
				return true
			}
		}
	}
	return false
}

func latestLinkedTimelineID(records []RelatedRecord) string {
	for _, record := range records {
		if record.Kind == RecordTimelineEvent {
			return record.ID
		}
	}
	return ""
}

func buildCaseEventLink(c Case, event db.TimelineEvent) CaseEventLink {
	relation, contribution, note := caseEventRelation(c, event)
	timeBasis, timeContext := timelineEventTimeContext(event)
	return CaseEventLink{
		EventID:          event.EventID,
		EventType:        event.EventType,
		Summary:          event.Summary,
		Severity:         event.Severity,
		RelationType:     relation,
		Contribution:     contribution,
		Scope:            scopeFromTimeline(event),
		EventTime:        event.EventTime,
		TimeBasis:        timeBasis,
		TimeContext:      timeContext,
		OrderingPosture:  normalizeCaseTimingPosture(event.TimingPosture),
		RawTimingPosture: strings.TrimSpace(event.TimingPosture),
		OriginInstanceID: strings.TrimSpace(event.OriginInstanceID),
		ImportID:         strings.TrimSpace(event.ImportID),
		MergeDisposition: strings.TrimSpace(event.MergeDisposition),
		InspectCLI:       fmt.Sprintf("mel timeline inspect %s --config <path>", event.EventID),
		InspectAPI:       "/api/v1/timeline/" + event.EventID,
		Note:             note,
	}
}

func caseEventRelation(c Case, event db.TimelineEvent) (CaseEventRelationType, CaseEventContribution, string) {
	switch event.EventType {
	case "remote_import_batch", "remote_evidence_import_item":
		return CaseEventRelationImportValidation, CaseEventContributionContextual,
			"Import/validation events shape case context and scope. They do not by themselves prove current state or causality."
	case "remote_event_materialized":
		switch strings.TrimSpace(event.MergeDisposition) {
		case string(fleet.DedupeConflicting), string(fleet.DedupeAmbiguous):
			return CaseEventRelationConflictingEvidence, CaseEventContributionConflicting,
				"Imported historical evidence remains related to the case but includes merge ambiguity or conflict. It does not establish a clean causal sequence."
		case string(fleet.DedupeNearDuplicate), string(fleet.DedupeMergedCanonical), string(fleet.DedupeRelatedDistinct), string(fleet.DedupeSuperseded):
			return CaseEventRelationMergeDisposition, CaseEventContributionContextual,
				"Merge classification affected how this historical event contributes to the case. Relatedness does not imply causality."
		default:
			if c.Kind == CaseImportedHistorical || c.Kind == CasePartialFleetVisibility {
				return CaseEventRelationHistoricalContext, CaseEventContributionSupporting,
					"Imported historical evidence contributes to this case as context only. It is not live local proof."
			}
			return CaseEventRelationHistoricalContext, CaseEventContributionContextual,
				"Imported historical evidence contributes context to the case and remains bounded to its origin."
		}
	case "incident":
		return CaseEventRelationIncidentContext, CaseEventContributionSupporting,
			"Incident records contribute durable operator context to the case. They are not automatic root-cause proof."
	case "control_action":
		return CaseEventRelationControlContext, CaseEventContributionContextual,
			"Control-action history is related operator context. It does not by itself prove why the case exists."
	case "freeze_created", "freeze_cleared", "maintenance":
		return CaseEventRelationOperationalContext, CaseEventContributionContextual,
			"Operational-state events provide context around the case without proving causality."
	default:
		if c.Kind == CaseTransportDegradation {
			return CaseEventRelationTransportContext, CaseEventContributionSupporting,
				"Transport-linked events contribute supporting context to this case. They do not prove wider mesh or RF impact."
		}
		if c.Kind == CaseEvidenceFreshnessGap {
			return CaseEventRelationFreshnessContext, CaseEventContributionContextual,
				"Freshness-linked events show what evidence exists, not what missing evidence proves."
		}
		return CaseEventRelationSupportingEvidence, CaseEventContributionContextual,
			"Related events contribute bounded case context and do not imply proven causality."
	}
}

func timelineEventTimeContext(event db.TimelineEvent) (string, CaseEventTimeContext) {
	ctx := CaseEventTimeContext{}
	switch event.EventType {
	case "remote_import_batch", "remote_evidence_import_item":
		ctx.ImportedAt = event.EventTime
	case "remote_event_materialized":
		ctx.ObservedAt = firstNonEmptyString(
			nestedString(event.Details, "remote_event_envelope", "observed_at"),
			nestedString(event.Details, "canonical_evidence_envelope", "observed_at"),
		)
		ctx.ReceivedAt = firstNonEmptyString(
			nestedString(event.Details, "remote_event_envelope", "received_at"),
			nestedString(event.Details, "canonical_evidence_envelope", "received_at"),
		)
		ctx.RecordedAt = firstNonEmptyString(
			nestedString(event.Details, "remote_event_envelope", "recorded_at"),
			nestedString(event.Details, "canonical_evidence_envelope", "recorded_at"),
		)
		ctx.ImportedAt = nestedString(event.Details, "timing", "imported_at")
	case "incident":
		ctx.OccurredAt = event.EventTime
	default:
		ctx.RecordedAt = event.EventTime
	}

	switch {
	case event.EventTime != "" && event.EventTime == ctx.OccurredAt:
		return "occurred_at", ctx
	case event.EventTime != "" && event.EventTime == ctx.ObservedAt:
		return "observed_at", ctx
	case event.EventTime != "" && event.EventTime == ctx.ReceivedAt:
		return "received_at", ctx
	case event.EventTime != "" && event.EventTime == ctx.ImportedAt:
		return "imported_at", ctx
	case event.EventTime != "" && event.EventTime == ctx.RecordedAt:
		return "recorded_at", ctx
	default:
		if ctx.ImportedAt != "" {
			return "imported_at", ctx
		}
		if ctx.ObservedAt != "" {
			return "observed_at", ctx
		}
		if ctx.RecordedAt != "" {
			return "recorded_at", ctx
		}
		return "event_time", ctx
	}
}

func summarizeCaseTiming(detail CaseDetail, generatedAt string) CaseTimingSummary {
	postures := make([]CaseTimingPosture, 0, len(detail.LinkedEvents)+2)
	axes := []string{}
	localCount := 0
	importedCount := 0
	bestEffort := 0
	hasLocal := false
	hasImported := false
	hasMerged := false
	hasClockSkew := false
	hasMissingTS := false
	hasHistoricalImport := detail.Case.HistoricalOnly
	hasMixedFreshness := false

	for _, event := range detail.LinkedEvents {
		postures = append(postures, event.OrderingPosture)
		if event.OrderingPosture != CaseTimingLocallyOrdered {
			bestEffort++
		}
		switch {
		case event.RelationType == CaseEventRelationImportValidation || event.RelationType == CaseEventRelationHistoricalContext || event.Scope == ScopeImported || event.Scope == ScopeHistoricalOnly:
			hasImported = true
			importedCount++
		case event.Scope == ScopeLocal || event.Scope == ScopePartialFleet:
			hasLocal = true
			localCount++
		}
		switch event.OrderingPosture {
		case CaseTimingMergedBestEffortOrder:
			hasMerged = true
		case CaseTimingOrderingUncertainClockSkew:
			hasClockSkew = true
		case CaseTimingOrderingUncertainMissingTS:
			hasMissingTS = true
		case CaseTimingHistoricalImportNotLive, CaseTimingImportedPreservedOrder:
			hasHistoricalImport = true
		case CaseTimingMixedFreshnessWindow:
			hasMixedFreshness = true
		}
		if event.TimeContext.OccurredAt != "" {
			axes = append(axes, "occurred_at")
		}
		if event.TimeContext.ObservedAt != "" {
			axes = append(axes, "observed_at")
		}
		if event.TimeContext.ReceivedAt != "" {
			axes = append(axes, "received_at")
		}
		if event.TimeContext.RecordedAt != "" {
			axes = append(axes, "recorded_at")
		}
		if event.TimeContext.ImportedAt != "" {
			axes = append(axes, "imported_at")
		}
	}

	for _, gap := range detail.EvidenceGaps {
		switch gap.Reason {
		case GapOrderingUncertain:
			hasClockSkew = true
		case GapImportedHistoricalOnly:
			hasHistoricalImport = true
		case GapStaleContributors:
			hasMixedFreshness = true
		}
	}

	primary := CaseTimingLocallyOrdered
	switch {
	case hasLocal && hasImported:
		primary = CaseTimingSourceOrderGlobalOrderUnknown
	case hasMerged:
		primary = CaseTimingMergedBestEffortOrder
	case hasMissingTS:
		primary = CaseTimingOrderingUncertainMissingTS
	case hasHistoricalImport && !hasLocal:
		primary = CaseTimingHistoricalImportNotLive
	case hasClockSkew:
		primary = CaseTimingOrderingUncertainClockSkew
	case hasMixedFreshness:
		primary = CaseTimingMixedFreshnessWindow
	case hasImported:
		primary = CaseTimingImportedPreservedOrder
	}

	postures = append(postures, primary)
	if hasHistoricalImport {
		postures = append(postures, CaseTimingHistoricalImportNotLive)
	}
	if hasMixedFreshness {
		postures = append(postures, CaseTimingMixedFreshnessWindow)
	}
	if hasLocal && hasImported {
		postures = append(postures, CaseTimingSourceOrderGlobalOrderUnknown)
	}
	postures = uniqueCaseTimingPostures(postures)

	note := buildCaseTimingNote(primary, detail.LinkedEvents, generatedAt)
	if len(detail.LinkedEvents) == 0 {
		note = "No raw canonical events are currently linked into this case detail. Case evolution below is derived from current findings, gaps, recommendations, and durable records only."
	}

	return CaseTimingSummary{
		PrimaryPosture:     primary,
		Postures:           postures,
		ExactSequence:      primary == CaseTimingLocallyOrdered && len(detail.LinkedEvents) > 0,
		TimestampAxes:      uniqueStrings(axes),
		LocalEventCount:    localCount,
		ImportedEventCount: importedCount,
		BestEffortCount:    bestEffort,
		Note:               note,
	}
}

func buildCaseTimingNote(primary CaseTimingPosture, linkedEvents []CaseEventLink, generatedAt string) string {
	switch primary {
	case CaseTimingLocallyOrdered:
		return "Linked case events were recorded by this MEL instance. Sequence is exact within this database only; it is not a fleet-wide total order."
	case CaseTimingImportedPreservedOrder:
		return "This case is shaped by imported evidence. Source-local order may be preserved, but imported chronology is not live local proof."
	case CaseTimingMergedBestEffortOrder:
		return "This case includes merge or dedupe classifications. Sequence is best-effort and should not be treated as a strict causal chain."
	case CaseTimingOrderingUncertainClockSkew:
		return "Ordering remains uncertain because reported timestamps may reflect clock skew or mismatched observed/received times."
	case CaseTimingOrderingUncertainMissingTS:
		return "Ordering remains uncertain because one or more linked records lack the timestamps needed for a clean sequence."
	case CaseTimingHistoricalImportNotLive:
		return "This case includes historical imported context. Imported chronology may explain why the operator should look deeper, but it is not live federation."
	case CaseTimingMixedFreshnessWindow:
		return "This case spans stale and fresher evidence windows. Sequence is still useful, but freshness differences bound what MEL can safely conclude."
	case CaseTimingSourceOrderGlobalOrderUnknown:
		return "This case mixes local and imported evidence. Source-local order can be known, but cross-source/global order remains unknown."
	default:
		return fmt.Sprintf("Case chronology was reconstructed at %s from linked evidence without implying a global total order.", generatedAt)
	}
}

func buildCaseEvolution(detail CaseDetail, ctx deriveContext, generatedAt string) []CaseEvolutionEntry {
	out := []CaseEvolutionEntry{}
	primary := detail.Case.Timing.PrimaryPosture
	if primary == "" {
		primary = CaseTimingLocallyOrdered
	}

	if entry, ok := buildCaseMaterializedEntry(detail, primary); ok {
		out = append(out, entry)
	}
	if entry, ok := buildTransportStateEntry(detail, ctx, primary); ok {
		out = append(out, entry)
	}
	if entry, ok := buildFreshnessEntry(detail, primary, generatedAt); ok {
		out = append(out, entry)
	}
	if entry, ok := buildImportValidationEntry(detail, primary); ok {
		out = append(out, entry)
	}
	if entry, ok := buildMergeDispositionEntry(detail, primary); ok {
		out = append(out, entry)
	}
	if entry, ok := buildUncertaintyEntry(detail, primary, generatedAt); ok {
		out = append(out, entry)
	}
	if entry, ok := buildUncertaintyReducedEntry(detail, primary); ok {
		out = append(out, entry)
	}
	if entry, ok := buildRecommendationEntry(detail, primary); ok {
		out = append(out, entry)
	}

	out = dedupeCaseEvolution(out)
	sort.SliceStable(out, func(i, j int) bool {
		return caseTimeLess(out[i].OccurredAt, out[j].OccurredAt, out[i].EntryID, out[j].EntryID)
	})
	return out
}

func buildCaseMaterializedEntry(detail CaseDetail, primary CaseTimingPosture) (CaseEvolutionEntry, bool) {
	if len(detail.Findings) == 0 && len(detail.LinkedEvents) == 0 {
		return CaseEvolutionEntry{}, false
	}
	eventIDs := []string{}
	summary := "Current case posture was materialized from retained case evidence."
	scope := detail.Case.Scope
	occurredAt := firstNonEmptyString(detail.Case.ObservedAt, detail.Case.UpdatedAt)
	basis := CaseReconstructionCurrentState
	reasons := []CaseEvolutionReason{CaseEvolutionCaseMaterialized}
	note := "Case evolution summarizes how the current case became inspectable. It is not a fabricated incident story."

	if len(detail.LinkedEvents) > 0 {
		first := detail.LinkedEvents[0]
		occurredAt = firstNonEmptyString(first.EventTime, occurredAt)
		scope = first.Scope
		basis = CaseReconstructionExactEvent
		eventIDs = []string{first.EventID}
		switch first.Contribution {
		case CaseEventContributionSupporting:
			if first.Scope == ScopeImported || first.Scope == ScopeHistoricalOnly {
				reasons = append(reasons, CaseEvolutionImportedSupportingEvidence)
				summary = "Imported evidence first made this case inspectable as bounded historical context."
			} else {
				reasons = append(reasons, CaseEvolutionNewSupportingEvidence)
				summary = "Local supporting evidence first made this case inspectable."
			}
		case CaseEventContributionConflicting:
			reasons = append(reasons, CaseEvolutionImportedConflictingEvidence)
			summary = "Conflicting or merge-ambiguous evidence first made this case inspectable."
		default:
			summary = "Case context first became inspectable through retained linked evidence."
		}
	}

	return newCaseEvolutionEntry(
		detail.Case.ID+":materialized",
		summary,
		reasons,
		scope,
		occurredAt,
		"case_observed_at",
		primary,
		basis,
		detail,
		eventIDs,
		note,
	), true
}

func buildTransportStateEntry(detail CaseDetail, ctx deriveContext, primary CaseTimingPosture) (CaseEvolutionEntry, bool) {
	if detail.Case.Kind != CaseTransportDegradation && detail.Case.Kind != CaseEvidenceFreshnessGap {
		return CaseEvolutionEntry{}, false
	}
	name := strings.TrimPrefix(detail.Case.ID, "case:transport:")
	if detail.Case.Kind == CaseEvidenceFreshnessGap {
		name = ""
	}
	var target *db.TransportRuntime
	for i := range ctx.runtimeStates {
		state := &ctx.runtimeStates[i]
		if name == "" || state.Name == name {
			target = state
			if name != "" {
				break
			}
		}
	}
	if target == nil {
		return CaseEvolutionEntry{}, false
	}

	reasons := []CaseEvolutionReason{CaseEvolutionTransportStateContext}
	summary := fmt.Sprintf("Transport runtime snapshot reports %s state=%s. This contributes to the case but does not prove wider mesh impact.", target.Name, target.State)
	if detail.Case.Kind == CaseEvidenceFreshnessGap {
		reasons = append(reasons, CaseEvolutionExpectedReporterMissing)
		summary = fmt.Sprintf("Transport runtime snapshot reports %s state=%s while fresh ingest is unproven. Missing ingest remains an evidence gap, not proof of silence.", target.Name, target.State)
	}
	return newCaseEvolutionEntry(
		detail.Case.ID+":transport-state",
		summary,
		reasons,
		ScopeLocal,
		firstNonEmptyString(target.UpdatedAt, detail.Case.UpdatedAt),
		"runtime_updated_at",
		primary,
		CaseReconstructionExactRecord,
		detail,
		nil,
		"Runtime-state snapshots are durable records, not fabricated events.",
	), true
}

func buildFreshnessEntry(detail CaseDetail, primary CaseTimingPosture, generatedAt string) (CaseEvolutionEntry, bool) {
	var reasons []CaseEvolutionReason
	scope := detail.Case.Scope
	summaryParts := []string{}

	for _, gap := range detail.EvidenceGaps {
		switch gap.Reason {
		case GapMissingExpectedReporters:
			reasons = append(reasons, CaseEvolutionExpectedReporterMissing, CaseEvolutionUncertaintyIncreased)
			summaryParts = append(summaryParts, "Expected reporters or ingest paths are missing from current evidence.")
		case GapStaleContributors:
			reasons = append(reasons, CaseEvolutionReporterFreshnessDegraded, CaseEvolutionUncertaintyIncreased)
			summaryParts = append(summaryParts, "Freshness degraded across the contributing evidence window.")
		case GapScopeIncomplete:
			reasons = append(reasons, CaseEvolutionExpectedReporterMissing, CaseEvolutionUncertaintyIncreased)
			summaryParts = append(summaryParts, "Reporter coverage remains incomplete for the current scope.")
		}
		if gap.Scope == ScopeImported || gap.Scope == ScopeHistoricalOnly {
			scope = gap.Scope
		}
	}

	reasons = uniqueCaseEvolutionReasons(reasons)
	if len(reasons) == 0 {
		return CaseEvolutionEntry{}, false
	}
	return newCaseEvolutionEntry(
		detail.Case.ID+":freshness",
		strings.Join(uniqueStrings(summaryParts), " "),
		reasons,
		scope,
		firstNonEmptyString(detail.Case.UpdatedAt, generatedAt),
		"case_updated_at",
		primary,
		CaseReconstructionCurrentState,
		detail,
		linkedEventIDs(detail.LinkedEvents),
		"Freshness and missing-reporter posture are derived from retained evidence windows and current durable records.",
	), true
}

func buildImportValidationEntry(detail CaseDetail, primary CaseTimingPosture) (CaseEvolutionEntry, bool) {
	for _, event := range detail.LinkedEvents {
		if event.RelationType != CaseEventRelationImportValidation {
			continue
		}
		summary := "Import validation outcome contributed to the case without creating live federation or proving current local state."
		if event.Severity == "warning" {
			summary = "Import validation carried caveats or rejection signals that materially shaped this case."
		}
		return newCaseEvolutionEntry(
			detail.Case.ID+":import-validation",
			summary,
			[]CaseEvolutionReason{CaseEvolutionImportValidationOutcome},
			event.Scope,
			event.EventTime,
			event.TimeBasis,
			primary,
			CaseReconstructionExactEvent,
			detail,
			[]string{event.EventID},
			"Import validation events are related to the case but do not, by themselves, prove causality or current state.",
		), true
	}
	return CaseEvolutionEntry{}, false
}

func buildMergeDispositionEntry(detail CaseDetail, primary CaseTimingPosture) (CaseEvolutionEntry, bool) {
	for _, event := range detail.LinkedEvents {
		switch event.RelationType {
		case CaseEventRelationMergeDisposition, CaseEventRelationConflictingEvidence:
			reasons := []CaseEvolutionReason{CaseEvolutionMergeDispositionChanged}
			if event.Contribution == CaseEventContributionConflicting {
				reasons = append(reasons, CaseEvolutionImportedConflictingEvidence, CaseEvolutionUncertaintyIncreased)
			}
			summary := "Merge classification affected how imported evidence contributes to the current case."
			if event.Contribution == CaseEventContributionConflicting {
				summary = "Imported evidence remains related to this case, but merge classification exposed conflict or ambiguity."
			}
			return newCaseEvolutionEntry(
				detail.Case.ID+":merge",
				summary,
				reasons,
				event.Scope,
				event.EventTime,
				event.TimeBasis,
				primary,
				CaseReconstructionExactEvent,
				detail,
				[]string{event.EventID},
				"Merge and conflict posture is evidence-bounded. Relatedness does not imply root cause.",
			), true
		}
	}
	return CaseEvolutionEntry{}, false
}

func buildUncertaintyEntry(detail CaseDetail, primary CaseTimingPosture, generatedAt string) (CaseEvolutionEntry, bool) {
	reasons := []CaseEvolutionReason{}
	scope := detail.Case.Scope
	for _, gap := range detail.EvidenceGaps {
		switch gap.Reason {
		case GapOrderingUncertain, GapImportedHistoricalOnly, GapAuthenticityUnverified, GapNoLocalConfirmation, GapObserverConflict:
			reasons = append(reasons, CaseEvolutionUncertaintyIncreased)
		}
		if gap.Scope == ScopeImported || gap.Scope == ScopeHistoricalOnly {
			scope = gap.Scope
		}
	}
	reasons = uniqueCaseEvolutionReasons(reasons)
	if len(reasons) == 0 {
		return CaseEvolutionEntry{}, false
	}
	summary := "Current case certainty is bounded by ordering, provenance, or confirmation gaps that remain unresolved."
	if detail.Case.MissingEvidence != "" {
		summary = "Current case certainty is bounded by unresolved evidence gaps: " + detail.Case.MissingEvidence
	}
	return newCaseEvolutionEntry(
		detail.Case.ID+":uncertainty",
		summary,
		reasons,
		scope,
		firstNonEmptyString(detail.Case.UpdatedAt, generatedAt),
		"case_updated_at",
		primary,
		CaseReconstructionCurrentState,
		detail,
		linkedEventIDs(detail.LinkedEvents),
		"Uncertainty entries explain why the current case posture remains bounded. They do not invent hidden history.",
	), true
}

func buildUncertaintyReducedEntry(detail CaseDetail, primary CaseTimingPosture) (CaseEvolutionEntry, bool) {
	if detail.Case.Kind != CaseImportedHistorical {
		return CaseEvolutionEntry{}, false
	}
	hasImported := false
	hasLocal := false
	var importedAt string
	var localAt string
	var related []string
	for _, event := range detail.LinkedEvents {
		if event.Scope == ScopeImported || event.Scope == ScopeHistoricalOnly {
			hasImported = true
			importedAt = firstNonEmptyString(importedAt, event.EventTime)
			related = append(related, event.EventID)
			continue
		}
		hasLocal = true
		localAt = firstNonEmptyString(localAt, event.EventTime)
		related = append(related, event.EventID)
	}
	if !hasImported || !hasLocal || caseHasGap(detail.EvidenceGaps, GapNoLocalConfirmation) {
		return CaseEvolutionEntry{}, false
	}
	occurredAt := firstNonEmptyString(localAt, detail.Case.UpdatedAt)
	return newCaseEvolutionEntry(
		detail.Case.ID+":uncertainty-reduced",
		"Local evidence also exists alongside imported history, reducing imported-only uncertainty without establishing global order or root cause.",
		[]CaseEvolutionReason{CaseEvolutionUncertaintyReduced},
		ScopeImported,
		occurredAt,
		"event_time",
		primary,
		CaseReconstructionInferredSequence,
		detail,
		uniqueStrings(related),
		"This is an inferred case-evolution step based on retained event ordering and current case composition, not a persisted prior case snapshot.",
	), true
}

func buildRecommendationEntry(detail CaseDetail, primary CaseTimingPosture) (CaseEvolutionEntry, bool) {
	for _, rec := range detail.Recommendations {
		if rec.Code != RecCompareLocalVsImported && rec.Code != RecTreatAsHistoricalOnly && rec.Code != RecNoSafeConclusionYet {
			continue
		}
		return newCaseEvolutionEntry(
			detail.Case.ID+":recommendation",
			fmt.Sprintf("Current recommendation is %q because of the case's present evidence mix and uncertainty posture.", rec.Action),
			[]CaseEvolutionReason{CaseEvolutionRecommendationChangedEvidenceDelta},
			rec.Scope,
			firstNonEmptyString(rec.GeneratedAt, detail.Case.UpdatedAt),
			"recommendation_generated_at",
			primary,
			CaseReconstructionCurrentState,
			detail,
			linkedEventIDs(detail.LinkedEvents),
			"Recommendation evolution is reconstructed from current findings, gaps, and linked evidence. It does not imply an unstored historical recommendation log.",
		), true
	}
	return CaseEvolutionEntry{}, false
}

func newCaseEvolutionEntry(id, summary string, reasons []CaseEvolutionReason, scope ScopePosture, occurredAt, timeBasis string, posture CaseTimingPosture, basis CaseReconstructionBasis, detail CaseDetail, relatedEventIDs []string, note string) CaseEvolutionEntry {
	return CaseEvolutionEntry{
		EntryID:             id,
		Summary:             summary,
		ReasonCodes:         uniqueCaseEvolutionReasons(reasons),
		Scope:               scope,
		OccurredAt:          occurredAt,
		TimeBasis:           firstNonEmptyString(timeBasis, "event_time"),
		OrderingPosture:     posture,
		ReconstructionBasis: basis,
		Status:              detail.Case.Status,
		Attention:           detail.Case.Attention,
		Certainty:           detail.Case.Certainty,
		FindingIDs:          append([]string(nil), detail.Case.FindingIDs...),
		EvidenceGapIDs:      append([]string(nil), detail.Case.EvidenceGapIDs...),
		RecommendationIDs:   append([]string(nil), detail.Case.RecommendationIDs...),
		RelatedEventIDs:     uniqueStrings(relatedEventIDs),
		Note:                note,
	}
}

func normalizeCaseTimingPosture(raw string) CaseTimingPosture {
	switch strings.TrimSpace(raw) {
	case "", string(fleet.TimingOrderLocalOrdered):
		return CaseTimingLocallyOrdered
	case string(fleet.TimingOrderImportedPreserved):
		return CaseTimingImportedPreservedOrder
	case string(fleet.TimingOrderMergedBestEffort):
		return CaseTimingMergedBestEffortOrder
	case string(fleet.TimingOrderMissingTimestamps):
		return CaseTimingOrderingUncertainMissingTS
	case string(fleet.TimingOrderUncertainClockSkew), string(fleet.TimingOrderReceiveDiffersFromObserved), string(fleet.TimingOrderImportTimeNotEqualEventTime):
		return CaseTimingOrderingUncertainClockSkew
	case string(fleet.TimingOrderHistoricalImportNotLive):
		return CaseTimingHistoricalImportNotLive
	case string(fleet.TimingOrderMixedFreshnessWindow), string(fleet.TimingOrderStaleReporterContributed):
		return CaseTimingMixedFreshnessWindow
	default:
		return CaseTimingSourceOrderGlobalOrderUnknown
	}
}

func nestedString(root map[string]any, path ...string) string {
	if root == nil {
		return ""
	}
	var current any = root
	for _, part := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current, ok = m[part]
		if !ok {
			return ""
		}
	}
	return strings.TrimSpace(fmt.Sprint(current))
}

func dedupeCaseEvolution(entries []CaseEvolutionEntry) []CaseEvolutionEntry {
	seen := map[string]struct{}{}
	out := make([]CaseEvolutionEntry, 0, len(entries))
	for _, entry := range entries {
		key := entry.EntryID + "|" + entry.OccurredAt + "|" + strings.Join(caseEvolutionReasonStrings(entry.ReasonCodes), ",")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, entry)
	}
	return out
}

func linkedEventIDs(events []CaseEventLink) []string {
	out := make([]string, 0, len(events))
	for _, event := range events {
		out = append(out, event.EventID)
	}
	return uniqueStrings(out)
}

func caseHasGap(gaps []EvidenceGap, reason EvidenceGapReason) bool {
	for _, gap := range gaps {
		if gap.Reason == reason {
			return true
		}
	}
	return false
}

func caseTimeLess(a, b, ida, idb string) bool {
	if a == b {
		return ida < idb
	}
	at, aerr := time.Parse(time.RFC3339, a)
	bt, berr := time.Parse(time.RFC3339, b)
	if aerr == nil && berr == nil {
		if at.Equal(bt) {
			return ida < idb
		}
		return at.Before(bt)
	}
	if strings.TrimSpace(a) == "" {
		return false
	}
	if strings.TrimSpace(b) == "" {
		return true
	}
	return a < b
}

func uniqueCaseTimingPostures(values []CaseTimingPosture) []CaseTimingPosture {
	seen := map[CaseTimingPosture]struct{}{}
	out := make([]CaseTimingPosture, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func uniqueCaseEvolutionReasons(values []CaseEvolutionReason) []CaseEvolutionReason {
	seen := map[CaseEvolutionReason]struct{}{}
	out := make([]CaseEvolutionReason, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func caseEvolutionReasonStrings(values []CaseEvolutionReason) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, string(value))
	}
	return out
}
