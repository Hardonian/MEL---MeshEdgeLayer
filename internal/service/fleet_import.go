package service

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/fleet"
)

// ImportRemoteEvidenceBundle validates and persists an offline remote evidence payload.
// This is not live federation: it is file/import scoped, instance-local storage, read-only with respect to remote execution.
func (a *App) ImportRemoteEvidenceBundle(raw []byte, strictOrigin bool, actor string) (map[string]any, error) {
	return a.importRemoteEvidencePayload(raw, strictOrigin, actor, fleet.RemoteEvidenceImportSource{SourceType: "api_body"})
}

// ImportRemoteEvidenceFile validates and persists an offline remote evidence payload that came from a local file path.
func (a *App) ImportRemoteEvidenceFile(raw []byte, strictOrigin bool, actor, sourcePath string) (map[string]any, error) {
	sourcePath = strings.TrimSpace(sourcePath)
	source := fleet.RemoteEvidenceImportSource{
		SourceType: "file",
		SourcePath: sourcePath,
		SourceName: filepath.Base(sourcePath),
	}
	return a.importRemoteEvidencePayload(raw, strictOrigin, actor, source)
}

func (a *App) importRemoteEvidencePayload(raw []byte, strictOrigin bool, actor string, source fleet.RemoteEvidenceImportSource) (map[string]any, error) {
	if a.DB == nil {
		return nil, fmt.Errorf("database not available")
	}
	localID, err := a.DB.EnsureInstanceID()
	if err != nil {
		return nil, err
	}
	truth, err := fleet.BuildTruthSummary(a.Cfg, a.DB)
	if err != nil {
		return nil, err
	}

	actorID := strings.TrimSpace(actor)
	if actorID == "" {
		actorID = "system"
	}
	importedAt := fleet.ImportNowRFC3339()

	payload, parseErr := fleet.ValidateRemoteEvidenceImportPayload(raw, truth.SiteID, truth.FleetID, fleet.IngestValidateOptions{
		StrictOriginMatch: strictOrigin,
	})
	batchID := "impb-" + uuid.NewString()
	batchRecord, err := buildRemoteImportBatchRecord(batchID, importedAt, localID, truth, source, raw, payload)
	if err != nil {
		return nil, err
	}

	existingRows, _ := a.DB.ListImportedRemoteEvidence(1000)
	itemRows, inspections, summaries, timelineEvents, err := a.buildImportedEvidenceRows(batchID, importedAt, localID, truth, actorID, source, payload, existingRows)
	if err != nil {
		return nil, err
	}
	timelineEvents = append([]db.TimelineEvent{buildRemoteImportBatchTimelineEvent(batchRecord, payload, actorID)}, timelineEvents...)

	if err := a.DB.PersistRemoteImportBatch(batchRecord, itemRows, timelineEvents); err != nil {
		return nil, err
	}

	status := importStatusFromValidation(payload.Validation, parseErr)
	out := map[string]any{
		"status":            status,
		"batch_id":          batchID,
		"validation":        payload.Validation,
		"input_kind":        payload.InputKind,
		"truth_posture":     truth,
		"local_instance_id": localID,
		"source":            source,
		"items":             summaries,
		"item_inspections":  inspections,
		"accepted_count":    payload.Validation.AcceptedCount + payload.Validation.AcceptedWithCaveatsCount,
		"rejected_count":    payload.Validation.RejectedCount,
		"item_count":        payload.Validation.ItemCount,
		"note":              "Offline import only; imported evidence remains distinct from local observations and does not create live federation or remote control.",
	}
	if status == "error" {
		out["error"] = parseErr.Error()
	}
	return out, nil
}

func buildRemoteImportBatchRecord(batchID, importedAt, localID string, truth fleet.FleetTruthSummary, source fleet.RemoteEvidenceImportSource, raw []byte, payload fleet.RemoteEvidenceImportPayload) (db.RemoteImportBatchRecord, error) {
	validationJSON, err := json.Marshal(payload.Validation)
	if err != nil {
		return db.RemoteImportBatchRecord{}, err
	}
	var capabilityJSON []byte
	if payload.Batch.CapabilityPosture != (fleet.CapabilityPosture{}) {
		capabilityJSON, err = json.Marshal(payload.Batch.CapabilityPosture)
		if err != nil {
			return db.RemoteImportBatchRecord{}, err
		}
	}
	schemaVersion := strings.TrimSpace(payload.Batch.SchemaVersion)
	if schemaVersion == "" && len(payload.Items) > 0 {
		schemaVersion = strings.TrimSpace(payload.Items[0].Bundle.SchemaVersion)
	}
	if schemaVersion == "" {
		schemaVersion = fleet.RemoteEvidenceBatchSchemaVersion
	}
	formatKind := strings.TrimSpace(payload.InputKind)
	if formatKind == "" {
		formatKind = "unknown"
	}
	return db.RemoteImportBatchRecord{
		ID:                       batchID,
		ImportedAt:               importedAt,
		LocalInstanceID:          localID,
		LocalSiteID:              truth.SiteID,
		LocalFleetID:             truth.FleetID,
		SourceType:               strings.TrimSpace(source.SourceType),
		SourceName:               strings.TrimSpace(source.SourceName),
		SourcePath:               strings.TrimSpace(source.SourcePath),
		SupportBundleID:          strings.TrimSpace(source.SupportBundleID),
		FormatKind:               formatKind,
		SchemaVersion:            schemaVersion,
		ClaimedOriginInstanceID:  strings.TrimSpace(payload.Batch.ClaimedOrigin.InstanceID),
		ClaimedOriginSiteID:      strings.TrimSpace(payload.Batch.ClaimedOrigin.SiteID),
		ClaimedFleetID:           strings.TrimSpace(payload.Batch.ClaimedOrigin.FleetID),
		ExportedAt:               strings.TrimSpace(payload.Batch.ExportedAt),
		CapabilityPosture:        capabilityJSON,
		Validation:               validationJSON,
		RawPayload:               append([]byte(nil), raw...),
		ItemCount:                payload.Validation.ItemCount,
		AcceptedCount:            payload.Validation.AcceptedCount,
		AcceptedWithCaveatsCount: payload.Validation.AcceptedWithCaveatsCount,
		RejectedCount:            payload.Validation.RejectedCount,
		PartialSuccess:           payload.Validation.Outcome == fleet.ValidationAcceptedPartial,
		Note:                     payload.Validation.OfflineOnlyNote,
	}, nil
}

func (a *App) buildImportedEvidenceRows(batchID, importedAt, localID string, truth fleet.FleetTruthSummary, actorID string, source fleet.RemoteEvidenceImportSource, payload fleet.RemoteEvidenceImportPayload, existingRows []db.ImportedRemoteEvidenceRecord) ([]db.ImportedRemoteEvidenceRecord, []fleet.ImportedEvidenceInspection, []fleet.ImportedEvidenceSummary, []db.TimelineEvent, error) {
	rows := make([]db.ImportedRemoteEvidenceRecord, 0, len(payload.Items))
	inspections := make([]fleet.ImportedEvidenceInspection, 0, len(payload.Items))
	summaries := make([]fleet.ImportedEvidenceSummary, 0, len(payload.Items))
	timelineEvents := make([]db.TimelineEvent, 0, 1+len(payload.Items)*2)
	comparisonRows := append([]db.ImportedRemoteEvidenceRecord(nil), existingRows...)

	for _, item := range payload.Items {
		itemID := "imp-" + uuid.NewString()
		bundleJSON, err := json.Marshal(item.Bundle)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		evidenceJSON, err := json.Marshal(item.Bundle.Evidence)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		validationJSON, err := json.Marshal(item.Validation)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		eventJSON, err := json.Marshal(item.Bundle.Event)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		rec := db.ImportedRemoteEvidenceRecord{
			ID:                      itemID,
			BatchID:                 batchID,
			ItemID:                  fmt.Sprintf("%s:%03d", batchID, item.Sequence),
			SequenceNo:              item.Sequence,
			ImportedAt:              importedAt,
			LocalInstanceID:         localID,
			LocalSiteID:             truth.SiteID,
			LocalFleetID:            truth.FleetID,
			SourceType:              strings.TrimSpace(source.SourceType),
			SourceName:              strings.TrimSpace(source.SourceName),
			SourcePath:              strings.TrimSpace(source.SourcePath),
			ValidationStatus:        string(item.Validation.Outcome),
			Validation:              validationJSON,
			Bundle:                  bundleJSON,
			Evidence:                evidenceJSON,
			Event:                   eventJSON,
			ClaimedOriginInstanceID: strings.TrimSpace(item.Bundle.ClaimedOriginInstanceID),
			ClaimedOriginSiteID:     strings.TrimSpace(item.Bundle.ClaimedOriginSiteID),
			ClaimedFleetID:          strings.TrimSpace(item.Bundle.ClaimedFleetID),
			OriginInstanceID:        strings.TrimSpace(item.Bundle.Evidence.OriginInstanceID),
			OriginSiteID:            strings.TrimSpace(item.Bundle.Evidence.OriginSiteID),
			EvidenceClass:           string(item.Bundle.Evidence.EvidenceClass),
			ObservationOriginClass:  string(item.Bundle.Evidence.OriginClass),
			CorrelationID:           strings.TrimSpace(item.Bundle.Evidence.CorrelationID),
			ObservedAt:              strings.TrimSpace(item.Bundle.Evidence.ObservedAt),
			ReceivedAt:              strings.TrimSpace(item.Bundle.Evidence.ReceivedAt),
			RecordedAt:              strings.TrimSpace(item.Bundle.Evidence.RecordedAt),
			Rejected:                item.Validation.Outcome == fleet.ValidationRejected,
		}
		inspection, err := fleet.InspectImportedRemoteEvidenceRecord(truth, rec, append(comparisonRows, rec))
		if err != nil {
			return nil, nil, nil, nil, err
		}
		normalizedJSON, err := json.Marshal(inspection)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		rec.Normalized = normalizedJSON
		rec.TimingPosture = string(inspection.Timing.PrimaryPosture)
		rec.MergeDisposition = string(inspection.MergeInspection.Classification.Disposition)
		rec.MergeCorrelationID = inspection.MergeInspection.Classification.MergeKey

		rows = append(rows, rec)
		comparisonRows = append(comparisonRows, rec)
		inspections = append(inspections, inspection)
		summaries = append(summaries, inspection.SummaryView())
		timelineEvents = append(timelineEvents, buildRemoteImportItemTimelineEvent(rec, inspection, actorID))
		if !rec.Rejected {
			timelineEvents = append(timelineEvents, buildRemoteMaterializedEventTimelineEvent(rec, inspection))
		}
	}

	return rows, inspections, summaries, timelineEvents, nil
}

func buildRemoteImportBatchTimelineEvent(batch db.RemoteImportBatchRecord, payload fleet.RemoteEvidenceImportPayload, actorID string) db.TimelineEvent {
	status := string(payload.Validation.Outcome)
	severity := "info"
	if payload.Validation.Outcome == fleet.ValidationRejected {
		severity = "warning"
	}
	if payload.Validation.Outcome == fleet.ValidationAcceptedPartial {
		severity = "warning"
	}
	summary := fmt.Sprintf("remote import batch %s %s (%d accepted, %d rejected)", batch.ID, status, batch.AcceptedCount+batch.AcceptedWithCaveatsCount, batch.RejectedCount)
	return db.TimelineEvent{
		EventID:          batch.ID,
		EventTime:        batch.ImportedAt,
		EventType:        "remote_import_batch",
		Summary:          summary,
		Severity:         severity,
		ActorID:          actorID,
		ResourceID:       batch.ID,
		ScopePosture:     "remote_import_batch",
		TimingPosture:    string(fleet.TimingOrderLocalOrdered),
		MergeDisposition: "raw_only",
		ImportID:         batch.ID,
		Details: map[string]any{
			"batch_id":                batch.ID,
			"source":                  map[string]any{"type": batch.SourceType, "name": batch.SourceName, "path": batch.SourcePath},
			"validation":              payload.Validation,
			"claimed_origin":          payload.Batch.ClaimedOrigin,
			"input_kind":              payload.InputKind,
			"item_count":              payload.Validation.ItemCount,
			"accepted_count":          payload.Validation.AcceptedCount + payload.Validation.AcceptedWithCaveatsCount,
			"rejected_count":          payload.Validation.RejectedCount,
			"offline_only_federation": true,
			"note":                    payload.Validation.OfflineOnlyNote,
		},
	}
}

func buildRemoteImportItemTimelineEvent(rec db.ImportedRemoteEvidenceRecord, inspection fleet.ImportedEvidenceInspection, actorID string) db.TimelineEvent {
	severity := ternary(rec.Rejected, "warning", "info")
	status := strings.TrimSpace(rec.ValidationStatus)
	if status == "" {
		status = ternary(rec.Rejected, "rejected", "accepted_with_caveats")
	}
	summary := fmt.Sprintf("remote evidence item %s %s from %s", rec.ID, status, rec.OriginInstanceID)
	if rec.OriginSiteID != "" {
		summary += "@" + rec.OriginSiteID
	}
	return db.TimelineEvent{
		EventID:            rec.ID + ":audit",
		EventTime:          rec.ImportedAt,
		EventType:          "remote_evidence_import_item",
		Summary:            summary,
		Severity:           severity,
		ActorID:            actorID,
		ResourceID:         rec.ID,
		ScopePosture:       "remote_imported",
		OriginInstanceID:   rec.OriginInstanceID,
		TimingPosture:      string(inspection.Timing.PrimaryPosture),
		MergeDisposition:   string(inspection.MergeInspection.Classification.Disposition),
		MergeCorrelationID: inspection.MergeInspection.Classification.MergeKey,
		ImportID:           rec.ID,
		Details:            fleet.BuildImportTimelineDetails(inspection, actorID),
	}
}

func buildRemoteMaterializedEventTimelineEvent(rec db.ImportedRemoteEvidenceRecord, inspection fleet.ImportedEvidenceInspection) db.TimelineEvent {
	eventTime := bestRemoteEventTime(inspection)
	summary := remoteMaterializedSummary(inspection)
	details := map[string]any{
		"batch_id":                    rec.BatchID,
		"item_id":                     rec.ID,
		"validation":                  inspection.Validation,
		"claimed_origin":              inspection.ClaimedOrigin,
		"provenance":                  inspection.Provenance,
		"timing":                      inspection.Timing,
		"canonical_evidence_envelope": inspection.EvidenceEnvelope,
		"remote_event_envelope":       inspection.RemoteEvent,
		"merge_inspection":            inspection.MergeInspection,
		"unknowns":                    inspection.Unknowns,
		"historical_only":             true,
		"note":                        "Materialized from offline import. This row preserves claimed remote event timing/provenance for investigation and does not imply live federation or global order.",
	}
	if eventTime == "" {
		eventTime = rec.ImportedAt
	}
	mergeDisposition := string(inspection.MergeInspection.Classification.Disposition)
	if mergeDisposition == "" {
		mergeDisposition = "raw_only"
	}
	return db.TimelineEvent{
		EventID:            rec.ID + ":remote_event",
		EventTime:          eventTime,
		EventType:          "remote_event_materialized",
		Summary:            summary,
		Severity:           "info",
		ActorID:            "remote_import",
		ResourceID:         rec.ID,
		ScopePosture:       "remote_reported",
		OriginInstanceID:   rec.OriginInstanceID,
		TimingPosture:      string(inspection.Timing.PrimaryPosture),
		MergeDisposition:   mergeDisposition,
		MergeCorrelationID: inspection.MergeInspection.Classification.MergeKey,
		ImportID:           rec.ID,
		Details:            details,
	}
}

func bestRemoteEventTime(inspection fleet.ImportedEvidenceInspection) string {
	if inspection.RemoteEvent != nil {
		if ts := strings.TrimSpace(inspection.RemoteEvent.ObservedAt); ts != "" {
			return ts
		}
		if ts := strings.TrimSpace(inspection.RemoteEvent.RecordedAt); ts != "" {
			return ts
		}
		if ts := strings.TrimSpace(inspection.RemoteEvent.ReceivedAt); ts != "" {
			return ts
		}
	}
	if ts := strings.TrimSpace(inspection.EvidenceEnvelope.ObservedAt); ts != "" {
		return ts
	}
	if ts := strings.TrimSpace(inspection.EvidenceEnvelope.RecordedAt); ts != "" {
		return ts
	}
	if ts := strings.TrimSpace(inspection.EvidenceEnvelope.ReceivedAt); ts != "" {
		return ts
	}
	return strings.TrimSpace(inspection.ImportedAt)
}

func remoteMaterializedSummary(inspection fleet.ImportedEvidenceInspection) string {
	if inspection.RemoteEvent != nil && strings.TrimSpace(inspection.RemoteEvent.Summary) != "" {
		return fmt.Sprintf("remote event from %s: %s", inspection.Provenance.OriginInstanceID, strings.TrimSpace(inspection.RemoteEvent.Summary))
	}
	return fmt.Sprintf("remote %s from %s", inspection.EvidenceEnvelope.EvidenceClass, inspection.Provenance.OriginInstanceID)
}

func importStatusFromValidation(validation fleet.RemoteEvidenceBatchValidation, parseErr error) string {
	if parseErr != nil {
		return "error"
	}
	if validation.Outcome == fleet.ValidationRejected {
		return "rejected"
	}
	return "imported"
}



func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}

// ListImportedRemoteEvidence returns recent import audit item rows.
func (a *App) ListImportedRemoteEvidence(limit int) ([]db.ImportedRemoteEvidenceRecord, error) {
	if a.DB == nil {
		return nil, fmt.Errorf("database not available")
	}
	return a.DB.ListImportedRemoteEvidence(limit)
}

// ImportedRemoteEvidenceByBatch returns item rows for one batch.
func (a *App) ImportedRemoteEvidenceByBatch(batchID string) ([]db.ImportedRemoteEvidenceRecord, error) {
	if a.DB == nil {
		return nil, fmt.Errorf("database not available")
	}
	return a.DB.ImportedRemoteEvidenceByBatch(batchID)
}

// GetImportedRemoteEvidence returns one imported item row.
func (a *App) GetImportedRemoteEvidence(id string) (db.ImportedRemoteEvidenceRecord, bool, error) {
	if a.DB == nil {
		return db.ImportedRemoteEvidenceRecord{}, false, fmt.Errorf("database not available")
	}
	return a.DB.GetImportedRemoteEvidence(id)
}

// ListRemoteImportBatches returns recent batch audit rows.
func (a *App) ListRemoteImportBatches(limit int) ([]db.RemoteImportBatchRecord, error) {
	if a.DB == nil {
		return nil, fmt.Errorf("database not available")
	}
	return a.DB.ListRemoteImportBatches(limit)
}

// GetRemoteImportBatch returns one batch audit row.
func (a *App) GetRemoteImportBatch(id string) (db.RemoteImportBatchRecord, bool, error) {
	if a.DB == nil {
		return db.RemoteImportBatchRecord{}, false, fmt.Errorf("database not available")
	}
	return a.DB.GetRemoteImportBatch(id)
}
