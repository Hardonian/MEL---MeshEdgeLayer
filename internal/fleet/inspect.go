package fleet

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mel-project/mel/internal/db"
)

const relatedEvidenceScanLimit = 500

// ImportedEvidenceClaimedOrigin preserves the bundle's claimed sender context.
type ImportedEvidenceClaimedOrigin struct {
	ClaimedOriginInstanceID string `json:"claimed_origin_instance_id,omitempty"`
	ClaimedOriginSiteID     string `json:"claimed_origin_site_id,omitempty"`
	ClaimedFleetID          string `json:"claimed_fleet_id,omitempty"`
	SourceLabel             string `json:"source_label,omitempty"`
	ImportReason            string `json:"import_reason,omitempty"`
}

// ImportedEvidenceSource preserves the local batch/source context for one imported item.
type ImportedEvidenceSource struct {
	BatchID    string `json:"batch_id"`
	SequenceNo int    `json:"sequence_no"`
	SourceType string `json:"source_type,omitempty"`
	SourceName string `json:"source_name,omitempty"`
	SourcePath string `json:"source_path,omitempty"`
}

// ImportedEvidenceProvenance preserves local import context separately from remote claimed origin.
type ImportedEvidenceProvenance struct {
	LocalInstanceID        string                 `json:"local_instance_id"`
	LocalSiteID            string                 `json:"local_site_id,omitempty"`
	LocalFleetID           string                 `json:"local_fleet_id,omitempty"`
	OriginInstanceID       string                 `json:"origin_instance_id"`
	OriginSiteID           string                 `json:"origin_site_id,omitempty"`
	ObservationOriginClass ObservationOriginClass `json:"observation_origin_class"`
	TruthBoundary          string                 `json:"truth_boundary"`
	ExecutionBoundary      string                 `json:"execution_boundary"`
}

// ImportedEvidenceTiming describes the honest timing basis for imported evidence.
type ImportedEvidenceTiming struct {
	ObservedAt       string               `json:"observed_at,omitempty"`
	ReceivedAt       string               `json:"received_at,omitempty"`
	RecordedAt       string               `json:"recorded_at,omitempty"`
	ImportedAt       string               `json:"imported_at"`
	EventTimeSource  string               `json:"event_time_source,omitempty"`
	PrimaryPosture   TimingOrderPosture   `json:"primary_posture"`
	TimingPostures   []TimingOrderPosture `json:"timing_postures"`
	OrderDisplayNote string               `json:"order_display_note"`
	Notes            []string             `json:"notes,omitempty"`
}

// ImportedEvidenceFieldDifference captures one concrete difference between related evidence rows.
type ImportedEvidenceFieldDifference struct {
	Field        string `json:"field"`
	CurrentValue string `json:"current_value,omitempty"`
	OtherValue   string `json:"other_value,omitempty"`
	Significance string `json:"significance"`
}

// RelatedImportedEvidence explains why a prior import was treated as duplicate/near/conflicting.
type RelatedImportedEvidence struct {
	ImportID           string                            `json:"import_id"`
	ImportedAt         string                            `json:"imported_at"`
	OriginInstanceID   string                            `json:"origin_instance_id"`
	EvidenceClass      EvidenceClass                     `json:"evidence_class"`
	Classification     MergeClassification               `json:"classification"`
	ExplainOperator    string                            `json:"explain_operator"`
	SharedFields       []string                          `json:"shared_fields,omitempty"`
	Differences        []ImportedEvidenceFieldDifference `json:"differences,omitempty"`
	RetainedSeparately bool                              `json:"retained_separately"`
	MergeApplied       bool                              `json:"merge_applied"`
	OrderingPosture    TimingOrderPosture                `json:"ordering_posture"`
	Unknowns           []string                          `json:"unknowns,omitempty"`
}

// ImportedEvidenceSummary is the concise operator-facing summary for list views.
type ImportedEvidenceSummary struct {
	ImportID                 string                   `json:"import_id"`
	Source                   ImportedEvidenceSource   `json:"source"`
	ImportedAt               string                   `json:"imported_at"`
	Rejected                 bool                     `json:"rejected"`
	SchemaVersion            string                   `json:"schema_version"`
	Kind                     string                   `json:"kind"`
	EvidenceClass            EvidenceClass            `json:"evidence_class"`
	OriginInstanceID         string                   `json:"origin_instance_id"`
	OriginSiteID             string                   `json:"origin_site_id,omitempty"`
	ClaimedOriginInstanceID  string                   `json:"claimed_origin_instance_id,omitempty"`
	ClaimedOriginSiteID      string                   `json:"claimed_origin_site_id,omitempty"`
	ClaimedFleetID           string                   `json:"claimed_fleet_id,omitempty"`
	ObservationOriginClass   ObservationOriginClass   `json:"observation_origin_class"`
	CorrelationID            string                   `json:"correlation_id,omitempty"`
	ObservedAt               string                   `json:"observed_at,omitempty"`
	ReceivedAt               string                   `json:"received_at,omitempty"`
	RecordedAt               string                   `json:"recorded_at,omitempty"`
	Validation               RemoteEvidenceValidation `json:"validation"`
	PrimaryTimingPosture     TimingOrderPosture       `json:"primary_timing_posture"`
	TimingPostures           []TimingOrderPosture     `json:"timing_postures"`
	RemoteEventPresent       bool                     `json:"remote_event_present"`
	RelatedEvidenceCount     int                      `json:"related_evidence_count"`
	RelatedDispositionCounts map[string]int           `json:"related_disposition_counts,omitempty"`
	Summary                  string                   `json:"summary"`
	Unknowns                 []string                 `json:"unknowns,omitempty"`
}

// ImportedEvidenceInspection is the full operator drilldown for one imported row.
type ImportedEvidenceInspection struct {
	ImportID         string                        `json:"import_id"`
	Source           ImportedEvidenceSource        `json:"source"`
	ImportedAt       string                        `json:"imported_at"`
	Rejected         bool                          `json:"rejected"`
	SchemaVersion    string                        `json:"schema_version"`
	Kind             string                        `json:"kind"`
	Validation       RemoteEvidenceValidation      `json:"validation"`
	ClaimedOrigin    ImportedEvidenceClaimedOrigin `json:"claimed_origin"`
	Provenance       ImportedEvidenceProvenance    `json:"provenance"`
	Timing           ImportedEvidenceTiming        `json:"timing"`
	EvidenceEnvelope EvidenceEnvelope              `json:"evidence_envelope"`
	RemoteEvent      *EventEnvelope                `json:"remote_event_envelope,omitempty"`
	LocalImportEvent EventEnvelope                 `json:"local_import_event_envelope"`
	RelatedEvidence  []RelatedImportedEvidence     `json:"related_evidence,omitempty"`
	MergeInspection  MergeInspection               `json:"merge_inspection"`
	Summary          string                        `json:"summary"`
	Unknowns         []string                      `json:"unknowns,omitempty"`
}

// RemoteImportBatchSummary is the concise operator-facing summary for one import batch.
type RemoteImportBatchSummary struct {
	BatchID                  string                           `json:"batch_id"`
	ImportedAt               string                           `json:"imported_at"`
	FormatKind               string                           `json:"format_kind"`
	SchemaVersion            string                           `json:"schema_version"`
	Source                   RemoteEvidenceImportSource       `json:"source"`
	ClaimedOrigin            RemoteEvidenceBatchClaimedOrigin `json:"claimed_origin"`
	Validation               RemoteEvidenceBatchValidation    `json:"validation"`
	ItemCount                int                              `json:"item_count"`
	AcceptedCount            int                              `json:"accepted_count"`
	AcceptedWithCaveatsCount int                              `json:"accepted_with_caveats_count"`
	RejectedCount            int                              `json:"rejected_count"`
	PartialSuccess           bool                             `json:"partial_success"`
	Note                     string                           `json:"note,omitempty"`
}

// RemoteImportBatchInspection is the full operator drilldown for one import batch.
type RemoteImportBatchInspection struct {
	Batch           RemoteImportBatchSummary     `json:"batch"`
	ItemSummaries   []ImportedEvidenceSummary    `json:"item_summaries,omitempty"`
	ItemInspections []ImportedEvidenceInspection `json:"item_inspections,omitempty"`
}

// InspectImportedRemoteEvidenceRecords builds full inspections for a set of imported rows.
func InspectImportedRemoteEvidenceRecords(truth FleetTruthSummary, rows []db.ImportedRemoteEvidenceRecord) ([]ImportedEvidenceInspection, error) {
	out := make([]ImportedEvidenceInspection, 0, len(rows))
	for _, row := range rows {
		inspection, err := InspectImportedRemoteEvidenceRecord(truth, row, rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inspection)
	}
	return out, nil
}

// SummarizeImportedRemoteEvidenceRecords builds stable list summaries for imported rows.
func SummarizeImportedRemoteEvidenceRecords(truth FleetTruthSummary, rows []db.ImportedRemoteEvidenceRecord) ([]ImportedEvidenceSummary, error) {
	inspections, err := InspectImportedRemoteEvidenceRecords(truth, rows)
	if err != nil {
		return nil, err
	}
	out := make([]ImportedEvidenceSummary, 0, len(inspections))
	for _, inspection := range inspections {
		out = append(out, inspection.SummaryView())
	}
	return out, nil
}

// SummarizeRemoteImportBatches builds stable list summaries for import batches.
func SummarizeRemoteImportBatches(rows []db.RemoteImportBatchRecord) ([]RemoteImportBatchSummary, error) {
	out := make([]RemoteImportBatchSummary, 0, len(rows))
	for _, row := range rows {
		validation, err := decodeRemoteImportBatchValidation(row)
		if err != nil {
			return nil, err
		}
		out = append(out, RemoteImportBatchSummary{
			BatchID:       row.ID,
			ImportedAt:    row.ImportedAt,
			FormatKind:    strings.TrimSpace(row.FormatKind),
			SchemaVersion: strings.TrimSpace(row.SchemaVersion),
			Source: RemoteEvidenceImportSource{
				SourceType:      strings.TrimSpace(row.SourceType),
				SourceName:      strings.TrimSpace(row.SourceName),
				SourcePath:      strings.TrimSpace(row.SourcePath),
				SupportBundleID: strings.TrimSpace(row.SupportBundleID),
			},
			ClaimedOrigin: RemoteEvidenceBatchClaimedOrigin{
				InstanceID: strings.TrimSpace(row.ClaimedOriginInstanceID),
				SiteID:     strings.TrimSpace(row.ClaimedOriginSiteID),
				FleetID:    strings.TrimSpace(row.ClaimedFleetID),
			},
			Validation:               validation,
			ItemCount:                row.ItemCount,
			AcceptedCount:            row.AcceptedCount,
			AcceptedWithCaveatsCount: row.AcceptedWithCaveatsCount,
			RejectedCount:            row.RejectedCount,
			PartialSuccess:           row.PartialSuccess,
			Note:                     strings.TrimSpace(row.Note),
		})
	}
	return out, nil
}

// InspectRemoteImportBatchRecord builds a full operator drilldown for one persisted import batch.
func InspectRemoteImportBatchRecord(truth FleetTruthSummary, batch db.RemoteImportBatchRecord, batchItems, allItems []db.ImportedRemoteEvidenceRecord) (RemoteImportBatchInspection, error) {
	summaries, err := SummarizeRemoteImportBatches([]db.RemoteImportBatchRecord{batch})
	if err != nil {
		return RemoteImportBatchInspection{}, err
	}
	compareRows := allItems
	if len(compareRows) == 0 {
		compareRows = batchItems
	}
	itemInspections := make([]ImportedEvidenceInspection, 0, len(batchItems))
	for _, item := range batchItems {
		inspection, err := InspectImportedRemoteEvidenceRecord(truth, item, compareRows)
		if err != nil {
			return RemoteImportBatchInspection{}, err
		}
		itemInspections = append(itemInspections, inspection)
	}
	itemSummaries := make([]ImportedEvidenceSummary, 0, len(itemInspections))
	for _, inspection := range itemInspections {
		itemSummaries = append(itemSummaries, inspection.SummaryView())
	}
	return RemoteImportBatchInspection{
		Batch:           summaries[0],
		ItemSummaries:   itemSummaries,
		ItemInspections: itemInspections,
	}, nil
}

// SummaryView collapses an inspection into the operator-facing list DTO.
func (i ImportedEvidenceInspection) SummaryView() ImportedEvidenceSummary {
	counts := map[string]int{}
	for _, rel := range i.RelatedEvidence {
		counts[string(rel.Classification.Disposition)]++
	}
	return ImportedEvidenceSummary{
		ImportID:                 i.ImportID,
		Source:                   i.Source,
		ImportedAt:               i.ImportedAt,
		Rejected:                 i.Rejected,
		SchemaVersion:            i.SchemaVersion,
		Kind:                     i.Kind,
		EvidenceClass:            i.EvidenceEnvelope.EvidenceClass,
		OriginInstanceID:         i.Provenance.OriginInstanceID,
		OriginSiteID:             i.Provenance.OriginSiteID,
		ClaimedOriginInstanceID:  i.ClaimedOrigin.ClaimedOriginInstanceID,
		ClaimedOriginSiteID:      i.ClaimedOrigin.ClaimedOriginSiteID,
		ClaimedFleetID:           i.ClaimedOrigin.ClaimedFleetID,
		ObservationOriginClass:   i.Provenance.ObservationOriginClass,
		CorrelationID:            strings.TrimSpace(i.EvidenceEnvelope.CorrelationID),
		ObservedAt:               strings.TrimSpace(i.Timing.ObservedAt),
		ReceivedAt:               strings.TrimSpace(i.Timing.ReceivedAt),
		RecordedAt:               strings.TrimSpace(i.Timing.RecordedAt),
		Validation:               i.Validation,
		PrimaryTimingPosture:     i.Timing.PrimaryPosture,
		TimingPostures:           append([]TimingOrderPosture(nil), i.Timing.TimingPostures...),
		RemoteEventPresent:       i.RemoteEvent != nil,
		RelatedEvidenceCount:     len(i.RelatedEvidence),
		Summary:                  i.Summary,
		Unknowns:                 append([]string(nil), i.Unknowns...),
		RelatedDispositionCounts: countsOrNil(counts),
	}
}

// InspectImportedRemoteEvidenceRecord decodes one persisted import row into canonical operator drilldown.
func InspectImportedRemoteEvidenceRecord(truth FleetTruthSummary, row db.ImportedRemoteEvidenceRecord, rows []db.ImportedRemoteEvidenceRecord) (ImportedEvidenceInspection, error) {
	bundle, validation, err := decodeImportedRemoteEvidence(row)
	if err != nil {
		return ImportedEvidenceInspection{}, err
	}
	timing := buildImportedEvidenceTiming(bundle, row.ImportedAt)
	related, relatedMerge := buildRelatedImportedEvidence(row, bundle, rows)
	unknowns := collectImportedEvidenceUnknowns(row, bundle, related)
	inspection := ImportedEvidenceInspection{
		ImportID: row.ID,
		Source: ImportedEvidenceSource{
			BatchID:    strings.TrimSpace(row.BatchID),
			SequenceNo: row.SequenceNo,
			SourceType: strings.TrimSpace(row.SourceType),
			SourceName: strings.TrimSpace(row.SourceName),
			SourcePath: strings.TrimSpace(row.SourcePath),
		},
		ImportedAt:    row.ImportedAt,
		Rejected:      row.Rejected,
		SchemaVersion: strings.TrimSpace(bundle.SchemaVersion),
		Kind:          strings.TrimSpace(bundle.Kind),
		Validation:    validation,
		ClaimedOrigin: ImportedEvidenceClaimedOrigin{
			ClaimedOriginInstanceID: strings.TrimSpace(bundle.ClaimedOriginInstanceID),
			ClaimedOriginSiteID:     strings.TrimSpace(bundle.ClaimedOriginSiteID),
			ClaimedFleetID:          strings.TrimSpace(bundle.ClaimedFleetID),
			SourceLabel:             strings.TrimSpace(bundle.ImportContext.SourceLabel),
			ImportReason:            strings.TrimSpace(bundle.ImportContext.ImportReason),
		},
		Provenance: ImportedEvidenceProvenance{
			LocalInstanceID:        truth.InstanceID,
			LocalSiteID:            truth.SiteID,
			LocalFleetID:           truth.FleetID,
			OriginInstanceID:       strings.TrimSpace(bundle.Evidence.OriginInstanceID),
			OriginSiteID:           strings.TrimSpace(bundle.Evidence.OriginSiteID),
			ObservationOriginClass: bundle.Evidence.OriginClass,
			TruthBoundary:          "Imported remote evidence remains distinct from locally observed evidence; this SQLite database is not fleet-wide authority.",
			ExecutionBoundary:      "Read-only federation posture only: imported evidence does not create a remote execution or control path.",
		},
		Timing:           timing,
		EvidenceEnvelope: bundle.Evidence,
		RemoteEvent:      cloneEventEnvelope(bundle.Event),
		LocalImportEvent: buildLocalImportEventEnvelope(truth, row, bundle, validation),
		RelatedEvidence:  related,
		MergeInspection:  MergeInspectionFromClassification(relatedMerge),
		Unknowns:         unknowns,
	}
	if strings.TrimSpace(row.TimingPosture) != "" {
		inspection.Timing.PrimaryPosture = TimingOrderPosture(strings.TrimSpace(row.TimingPosture))
		inspection.Timing.TimingPostures = uniqueTimingPostures(append(inspection.Timing.TimingPostures, inspection.Timing.PrimaryPosture))
	}
	if strings.TrimSpace(row.MergeDisposition) != "" {
		inspection.MergeInspection.Classification.Disposition = DedupeDisposition(strings.TrimSpace(row.MergeDisposition))
		inspection.MergeInspection.Explain = ExplainMergeForOperator(inspection.MergeInspection.Classification)
	}
	if strings.TrimSpace(row.MergeCorrelationID) != "" {
		inspection.MergeInspection.Classification.MergeKey = strings.TrimSpace(row.MergeCorrelationID)
	}
	inspection.Summary = buildImportedEvidenceSummaryText(inspection)
	return inspection, nil
}

// BuildImportTimelineDetails returns JSON-friendly provenance for timeline event details.
func BuildImportTimelineDetails(inspection ImportedEvidenceInspection, actor string) map[string]any {
	return map[string]any{
		"import_id":                   inspection.ImportID,
		"source":                      inspection.Source,
		"actor":                       strings.TrimSpace(actor),
		"validation":                  inspection.Validation,
		"claimed_origin":              inspection.ClaimedOrigin,
		"provenance":                  inspection.Provenance,
		"timing":                      inspection.Timing,
		"canonical_evidence_envelope": inspection.EvidenceEnvelope,
		"remote_event_envelope":       inspection.RemoteEvent,
		"local_import_event_envelope": inspection.LocalImportEvent,
		"merge_inspection":            inspection.MergeInspection,
		"related_evidence":            inspection.RelatedEvidence,
		"unknowns":                    inspection.Unknowns,
		"federation_note":             "Offline import only; not live synchronization, remote liveness, or cross-instance control.",
	}
}

func decodeImportedRemoteEvidence(row db.ImportedRemoteEvidenceRecord) (RemoteEvidenceBundle, RemoteEvidenceValidation, error) {
	var bundle RemoteEvidenceBundle
	if err := json.Unmarshal(row.Bundle, &bundle); err != nil {
		return bundle, RemoteEvidenceValidation{}, fmt.Errorf("decode bundle for %s: %w", row.ID, err)
	}
	var validation RemoteEvidenceValidation
	if err := json.Unmarshal(row.Validation, &validation); err != nil {
		return bundle, validation, fmt.Errorf("decode validation for %s: %w", row.ID, err)
	}
	return bundle, validation, nil
}

func decodeRemoteImportBatchValidation(row db.RemoteImportBatchRecord) (RemoteEvidenceBatchValidation, error) {
	var validation RemoteEvidenceBatchValidation
	if err := json.Unmarshal(row.Validation, &validation); err != nil {
		return validation, fmt.Errorf("decode batch validation for %s: %w", row.ID, err)
	}
	return validation, nil
}

func buildImportedEvidenceTiming(bundle RemoteEvidenceBundle, importedAt string) ImportedEvidenceTiming {
	postures := []TimingOrderPosture{TimingOrderImportedPreserved, TimingOrderUncertainClockSkew}
	notes := []string{
		"Imported timestamps remain observer-bounded and may reflect remote clock skew.",
	}
	observedAt := strings.TrimSpace(bundle.Evidence.ObservedAt)
	receivedAt := strings.TrimSpace(bundle.Evidence.ReceivedAt)
	recordedAt := strings.TrimSpace(bundle.Evidence.RecordedAt)
	importedAt = strings.TrimSpace(importedAt)

	primary := TimingOrderImportedPreserved
	if observedAt != "" && receivedAt != "" && observedAt != receivedAt {
		postures = append(postures, TimingOrderReceiveDiffersFromObserved)
		primary = TimingOrderReceiveDiffersFromObserved
		notes = append(notes, "Receive time differs from observed time; import order must not be read as event order.")
	}
	if observedAt != "" && importedAt != "" && observedAt != importedAt {
		postures = append(postures, TimingOrderImportTimeNotEqualEventTime)
		if primary == TimingOrderImportedPreserved {
			primary = TimingOrderImportTimeNotEqualEventTime
		}
		notes = append(notes, "Import time records when this MEL instance received the bundle, not when the remote event occurred.")
	}
	return ImportedEvidenceTiming{
		ObservedAt:       observedAt,
		ReceivedAt:       receivedAt,
		RecordedAt:       recordedAt,
		ImportedAt:       importedAt,
		EventTimeSource:  strings.TrimSpace(bundle.Evidence.EventTimeSrc),
		PrimaryPosture:   primary,
		TimingPostures:   uniqueTimingPostures(postures),
		OrderDisplayNote: "Use local import order for audit history only. Cross-instance total ordering is not implied.",
		Notes:            notes,
	}
}

func buildLocalImportEventEnvelope(truth FleetTruthSummary, row db.ImportedRemoteEvidenceRecord, bundle RemoteEvidenceBundle, validation RemoteEvidenceValidation) EventEnvelope {
	summary := fmt.Sprintf("remote evidence import %s (%s)", row.ID, validation.Outcome)
	if row.Rejected {
		summary = fmt.Sprintf("remote evidence import rejected %s", row.ID)
	}
	return EventEnvelope{
		EventID:          row.ID,
		CorrelationID:    strings.TrimSpace(bundle.Evidence.CorrelationID),
		OriginInstanceID: truth.InstanceID,
		OriginSiteID:     truth.SiteID,
		ObservedAt:       row.ImportedAt,
		RecordedAt:       row.ImportedAt,
		ReceivedAt:       row.ImportedAt,
		OrderingBasis:    string(TimingOrderLocalOrdered),
		EventType:        "remote_evidence_import",
		Summary:          summary,
		Details: map[string]any{
			"outcome":         validation.Outcome,
			"rejected":        row.Rejected,
			"remote_evidence": true,
		},
	}
}

func buildRelatedImportedEvidence(current db.ImportedRemoteEvidenceRecord, bundle RemoteEvidenceBundle, rows []db.ImportedRemoteEvidenceRecord) ([]RelatedImportedEvidence, MergeClassification) {
	limit := relatedEvidenceScanLimit
	if len(rows) < limit {
		limit = len(rows)
	}
	related := []RelatedImportedEvidence{}
	worst := MergeClassification{
		Disposition:  DedupeRelatedDistinct,
		MergePosture: MergePostureRawOnly,
		MergeKey:     "",
		Notes:        "no related imported evidence found",
	}
	for _, candidate := range rows[:limit] {
		if candidate.ID == current.ID {
			continue
		}
		candidateBundle, _, err := decodeImportedRemoteEvidence(candidate)
		if err != nil {
			continue
		}
		if !shouldCompareImportedEvidence(bundle, candidateBundle) {
			continue
		}
		rel := compareImportedEvidence(current, bundle, candidate, candidateBundle)
		related = append(related, rel)
		if relationSeverity(rel.Classification.Disposition) > relationSeverity(worst.Disposition) {
			worst = rel.Classification
		}
	}
	sort.SliceStable(related, func(i, j int) bool {
		return related[i].ImportedAt > related[j].ImportedAt
	})
	if len(related) == 0 {
		key := structuralEvidenceFingerprint(bundle.Evidence, bundle.Event)
		if key == "" {
			key = exactEvidenceFingerprint(bundle.Evidence, bundle.Event)
		}
		worst = MergeClassification{
			Disposition:  DedupeRelatedDistinct,
			MergePosture: MergePostureRawOnly,
			MergeKey:     hashString(key + "|" + current.ID),
			Notes:        "no related imported evidence found",
		}
	}
	return related, worst
}

func shouldCompareImportedEvidence(a, b RemoteEvidenceBundle) bool {
	if strings.TrimSpace(a.Evidence.OriginInstanceID) == "" || strings.TrimSpace(b.Evidence.OriginInstanceID) == "" {
		return false
	}
	if strings.TrimSpace(a.Evidence.CorrelationID) != "" && strings.TrimSpace(a.Evidence.CorrelationID) == strings.TrimSpace(b.Evidence.CorrelationID) {
		return true
	}
	if a.Event != nil && b.Event != nil && strings.TrimSpace(a.Event.EventID) != "" && strings.TrimSpace(a.Event.EventID) == strings.TrimSpace(b.Event.EventID) {
		return true
	}
	return strings.TrimSpace(a.Evidence.OriginInstanceID) == strings.TrimSpace(b.Evidence.OriginInstanceID) &&
		a.Evidence.EvidenceClass == b.Evidence.EvidenceClass
}

func compareImportedEvidence(current db.ImportedRemoteEvidenceRecord, bundle RemoteEvidenceBundle, other db.ImportedRemoteEvidenceRecord, otherBundle RemoteEvidenceBundle) RelatedImportedEvidence {
	sameObserver := observerIdentity(bundle.Evidence) == observerIdentity(otherBundle.Evidence)
	exactA := exactEvidenceFingerprint(bundle.Evidence, bundle.Event)
	exactB := exactEvidenceFingerprint(otherBundle.Evidence, otherBundle.Event)
	structuralA := structuralEvidenceFingerprint(bundle.Evidence, bundle.Event)
	structuralB := structuralEvidenceFingerprint(otherBundle.Evidence, otherBundle.Event)
	classification := MergeClassification{
		Disposition:  DedupeRelatedDistinct,
		MergePosture: MergePostureRawOnly,
		MergeKey:     hashString(structuralA + "|" + structuralB),
		Notes:        "distinct imported evidence retained separately",
	}
	switch {
	case exactA != "" && exactA == exactB:
		classification = ClassifyMerge(exactA, exactB, sameObserver)
	case structuralA != "" && structuralA == structuralB:
		if materiallyConflicting(bundle, otherBundle) {
			classification = MergeClassification{
				Disposition:  DedupeConflicting,
				MergePosture: MergePostureNoSilentCollapse,
				MergeKey:     hashString(structuralA),
				Notes:        "same structural identity but materially different timing/details; rows retained separately",
			}
		} else {
			classification = ClassifyMerge(structuralA, structuralB, sameObserver)
		}
	default:
		classification = MergeClassification{
			Disposition:  DedupeRelatedDistinct,
			MergePosture: MergePostureRawOnly,
			MergeKey:     hashString(structuralA + "|" + structuralB),
			Notes:        "related origin or evidence class only; no silent merge",
		}
	}
	sharedFields, differences := importedEvidenceFieldComparison(bundle, otherBundle)
	unknowns := []string{}
	if strings.TrimSpace(bundle.Evidence.CorrelationID) == "" || strings.TrimSpace(otherBundle.Evidence.CorrelationID) == "" {
		unknowns = append(unknowns, "correlation_id_missing_for_at_least_one_row")
	}
	return RelatedImportedEvidence{
		ImportID:           other.ID,
		ImportedAt:         other.ImportedAt,
		OriginInstanceID:   strings.TrimSpace(otherBundle.Evidence.OriginInstanceID),
		EvidenceClass:      otherBundle.Evidence.EvidenceClass,
		Classification:     classification,
		ExplainOperator:    ExplainMergeForOperator(classification),
		SharedFields:       sharedFields,
		Differences:        differences,
		RetainedSeparately: true,
		MergeApplied:       false,
		OrderingPosture:    relatedOrderingPosture(bundle, otherBundle),
		Unknowns:           unknownsOrNil(unknowns),
	}
}

func importedEvidenceFieldComparison(a, b RemoteEvidenceBundle) ([]string, []ImportedEvidenceFieldDifference) {
	type pair struct {
		field        string
		current      string
		other        string
		significance string
	}
	pairs := []pair{
		{"origin_instance_id", strings.TrimSpace(a.Evidence.OriginInstanceID), strings.TrimSpace(b.Evidence.OriginInstanceID), "Different origin_instance_id means the rows do not describe the same reporter."},
		{"origin_site_id", strings.TrimSpace(a.Evidence.OriginSiteID), strings.TrimSpace(b.Evidence.OriginSiteID), "Site mismatch blocks silent cross-site collapse."},
		{"observer_instance_id", observerIdentity(a.Evidence), observerIdentity(b.Evidence), "Observer changes require preserving per-observer provenance."},
		{"evidence_class", strings.TrimSpace(string(a.Evidence.EvidenceClass)), strings.TrimSpace(string(b.Evidence.EvidenceClass)), "Evidence classes differ; no canonical merge applied."},
		{"correlation_id", strings.TrimSpace(a.Evidence.CorrelationID), strings.TrimSpace(b.Evidence.CorrelationID), "Correlation mismatch weakens any claim that the rows describe one event."},
		{"observed_at", strings.TrimSpace(a.Evidence.ObservedAt), strings.TrimSpace(b.Evidence.ObservedAt), "Observed time drift may reflect remote clock differences or genuinely distinct events."},
		{"received_at", strings.TrimSpace(a.Evidence.ReceivedAt), strings.TrimSpace(b.Evidence.ReceivedAt), "Receive time drift is not event-order proof."},
		{"recorded_at", strings.TrimSpace(a.Evidence.RecordedAt), strings.TrimSpace(b.Evidence.RecordedAt), "Recorded time drift reflects per-instance persistence timing, not global ordering."},
		{"event_time_source", strings.TrimSpace(a.Evidence.EventTimeSrc), strings.TrimSpace(b.Evidence.EventTimeSrc), "Different event-time sources weaken direct ordering comparisons."},
		{"details_hash", detailsHash(a.Evidence.Details), detailsHash(b.Evidence.Details), "Detail payload differences prevent a silent merge into one canonical story."},
		{"remote_event_id", eventID(a.Event), eventID(b.Event), "Remote event IDs differ or are absent."},
		{"remote_event_type", eventType(a.Event), eventType(b.Event), "Different remote event types mean the rows should stay separate."},
	}
	shared := []string{}
	differences := []ImportedEvidenceFieldDifference{}
	for _, p := range pairs {
		switch {
		case p.current == "" && p.other == "":
		case p.current == p.other:
			shared = append(shared, p.field)
		default:
			differences = append(differences, ImportedEvidenceFieldDifference{
				Field:        p.field,
				CurrentValue: p.current,
				OtherValue:   p.other,
				Significance: p.significance,
			})
		}
	}
	sort.Strings(shared)
	return shared, differences
}

func materiallyConflicting(a, b RemoteEvidenceBundle) bool {
	if strings.TrimSpace(a.Evidence.OriginSiteID) != "" && strings.TrimSpace(b.Evidence.OriginSiteID) != "" &&
		strings.TrimSpace(a.Evidence.OriginSiteID) != strings.TrimSpace(b.Evidence.OriginSiteID) {
		return true
	}
	if strings.TrimSpace(a.Evidence.ObservedAt) != "" && strings.TrimSpace(b.Evidence.ObservedAt) != "" &&
		strings.TrimSpace(a.Evidence.ObservedAt) != strings.TrimSpace(b.Evidence.ObservedAt) {
		return true
	}
	if detailsHash(a.Evidence.Details) != detailsHash(b.Evidence.Details) {
		return true
	}
	if a.Event != nil && b.Event != nil && strings.TrimSpace(a.Event.EventType) != "" &&
		strings.TrimSpace(b.Event.EventType) != "" && strings.TrimSpace(a.Event.EventType) != strings.TrimSpace(b.Event.EventType) {
		return true
	}
	return false
}

func relatedOrderingPosture(a, b RemoteEvidenceBundle) TimingOrderPosture {
	if strings.TrimSpace(a.Evidence.ObservedAt) != "" && strings.TrimSpace(b.Evidence.ObservedAt) != "" &&
		strings.TrimSpace(a.Evidence.ObservedAt) != strings.TrimSpace(b.Evidence.ObservedAt) {
		return TimingOrderMergedBestEffort
	}
	if strings.TrimSpace(a.Evidence.ReceivedAt) != "" && strings.TrimSpace(b.Evidence.ReceivedAt) != "" &&
		strings.TrimSpace(a.Evidence.ReceivedAt) != strings.TrimSpace(b.Evidence.ReceivedAt) {
		return TimingOrderReceiveDiffersFromObserved
	}
	return TimingOrderUncertainClockSkew
}

func collectImportedEvidenceUnknowns(row db.ImportedRemoteEvidenceRecord, bundle RemoteEvidenceBundle, related []RelatedImportedEvidence) []string {
	unknowns := []string{
		"authenticity_not_cryptographically_verified",
		"import_does_not_imply_remote_site_is_currently_live",
	}
	if strings.TrimSpace(row.BatchID) == "" {
		unknowns = append(unknowns, "batch_id_missing")
	}
	if strings.TrimSpace(bundle.Evidence.OriginSiteID) == "" {
		unknowns = append(unknowns, "origin_site_id_missing")
	}
	if strings.TrimSpace(bundle.Evidence.CorrelationID) == "" {
		unknowns = append(unknowns, "correlation_id_missing")
	}
	if bundle.Event == nil {
		unknowns = append(unknowns, "remote_event_envelope_absent")
	}
	if row.Rejected {
		unknowns = append(unknowns, "rejected_import_retained_for_audit_only")
	}
	if len(related) == 0 {
		unknowns = append(unknowns, "no_related_imported_evidence_found")
	}
	return unknownsOrNil(uniqueStrings(unknowns))
}

func buildImportedEvidenceSummaryText(i ImportedEvidenceInspection) string {
	status := string(i.Validation.Outcome)
	if i.Rejected {
		status = "rejected"
	}
	base := fmt.Sprintf("Import %s is %s. Origin %s", i.ImportID, status, i.Provenance.OriginInstanceID)
	if strings.TrimSpace(i.Provenance.OriginSiteID) != "" {
		base += " at site " + i.Provenance.OriginSiteID
	}
	base += fmt.Sprintf(" (%s).", i.EvidenceEnvelope.EvidenceClass)
	if strings.TrimSpace(i.Source.BatchID) != "" {
		base += " Batch " + i.Source.BatchID + "."
	}
	if len(i.RelatedEvidence) > 0 {
		base += fmt.Sprintf(" %d related imported row(s) remain separate for provenance review.", len(i.RelatedEvidence))
	}
	base += " Imported evidence is read-only and not live fleet authority."
	return base
}

func exactEvidenceFingerprint(ev EvidenceEnvelope, remoteEvent *EventEnvelope) string {
	parts := []string{
		strings.TrimSpace(ev.OriginInstanceID),
		strings.TrimSpace(ev.OriginSiteID),
		observerIdentity(ev),
		strings.TrimSpace(string(ev.EvidenceClass)),
		strings.TrimSpace(ev.CorrelationID),
		strings.TrimSpace(ev.ObservedAt),
		strings.TrimSpace(ev.ReceivedAt),
		strings.TrimSpace(ev.RecordedAt),
		strings.TrimSpace(ev.EventTimeSrc),
		detailsHash(ev.Details),
		eventID(remoteEvent),
		eventType(remoteEvent),
	}
	if allStringsEmpty(parts...) {
		return ""
	}
	return hashString(strings.Join(parts, "|"))
}

func structuralEvidenceFingerprint(ev EvidenceEnvelope, remoteEvent *EventEnvelope) string {
	if correlation := strings.TrimSpace(ev.CorrelationID); correlation != "" {
		return hashString(strings.Join([]string{
			strings.TrimSpace(ev.OriginInstanceID),
			strings.TrimSpace(string(ev.EvidenceClass)),
			correlation,
		}, "|"))
	}
	if remoteEvent != nil && strings.TrimSpace(remoteEvent.EventID) != "" {
		return hashString(strings.Join([]string{
			strings.TrimSpace(ev.OriginInstanceID),
			strings.TrimSpace(string(ev.EvidenceClass)),
			strings.TrimSpace(remoteEvent.EventID),
		}, "|"))
	}
	return hashString(strings.Join([]string{
		strings.TrimSpace(ev.OriginInstanceID),
		observerIdentity(ev),
		strings.TrimSpace(string(ev.EvidenceClass)),
		strings.TrimSpace(ev.ObservedAt),
		detailsHash(ev.Details),
	}, "|"))
}

func detailsHash(v map[string]any) string {
	if len(v) == 0 {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return hashString(string(b))
}

func hashString(v string) string {
	if strings.TrimSpace(v) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}

func observerIdentity(ev EvidenceEnvelope) string {
	if observer := strings.TrimSpace(ev.ObserverInstanceID); observer != "" {
		return observer
	}
	return strings.TrimSpace(ev.OriginInstanceID)
}

func eventID(ev *EventEnvelope) string {
	if ev == nil {
		return ""
	}
	return strings.TrimSpace(ev.EventID)
}

func eventType(ev *EventEnvelope) string {
	if ev == nil {
		return ""
	}
	return strings.TrimSpace(ev.EventType)
}

func cloneEventEnvelope(in *EventEnvelope) *EventEnvelope {
	if in == nil {
		return nil
	}
	copied := *in
	if in.Details != nil {
		copied.Details = map[string]any{}
		for k, v := range in.Details {
			copied.Details[k] = v
		}
	}
	return &copied
}

func uniqueTimingPostures(in []TimingOrderPosture) []TimingOrderPosture {
	seen := map[TimingOrderPosture]bool{}
	out := make([]TimingOrderPosture, 0, len(in))
	for _, posture := range in {
		if posture == "" || seen[posture] {
			continue
		}
		seen[posture] = true
		out = append(out, posture)
	}
	return out
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func relationSeverity(d DedupeDisposition) int {
	switch d {
	case DedupeConflicting:
		return 4
	case DedupeNearDuplicate:
		return 3
	case DedupeExactDuplicate:
		return 2
	case DedupeRelatedDistinct:
		return 1
	default:
		return 0
	}
}

func countsOrNil(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}
	return in
}

func unknownsOrNil(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	return in
}

func allStringsEmpty(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}
