package db

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RemoteImportBatchRecord is the durable audit row for one offline import attempt.
// The raw payload and validation JSON are preserved verbatim for later investigation.
type RemoteImportBatchRecord struct {
	ID                       string          `json:"id"`
	ImportedAt               string          `json:"imported_at"`
	LocalInstanceID          string          `json:"local_instance_id"`
	LocalSiteID              string          `json:"local_site_id,omitempty"`
	LocalFleetID             string          `json:"local_fleet_id,omitempty"`
	SourceType               string          `json:"source_type,omitempty"`
	SourceName               string          `json:"source_name,omitempty"`
	SourcePath               string          `json:"source_path,omitempty"`
	SupportBundleID          string          `json:"support_bundle_id,omitempty"`
	FormatKind               string          `json:"format_kind"`
	SchemaVersion            string          `json:"schema_version"`
	ClaimedOriginInstanceID  string          `json:"claimed_origin_instance_id,omitempty"`
	ClaimedOriginSiteID      string          `json:"claimed_origin_site_id,omitempty"`
	ClaimedFleetID           string          `json:"claimed_fleet_id,omitempty"`
	ExportedAt               string          `json:"exported_at,omitempty"`
	CapabilityPosture        json.RawMessage `json:"capability_posture,omitempty"`
	Validation               json.RawMessage `json:"validation"`
	RawPayload               json.RawMessage `json:"raw_payload"`
	ItemCount                int             `json:"item_count"`
	AcceptedCount            int             `json:"accepted_count"`
	AcceptedWithCaveatsCount int             `json:"accepted_with_caveats_count"`
	RejectedCount            int             `json:"rejected_count"`
	PartialSuccess           bool            `json:"partial_success"`
	Note                     string          `json:"note,omitempty"`
}

// ImportedRemoteEvidenceRecord is a persisted imported item row (accepted or rejected).
// JSON fields preserve raw imported content plus the normalized canonical representation used for investigation.
type ImportedRemoteEvidenceRecord struct {
	ID                      string          `json:"id"`
	BatchID                 string          `json:"batch_id"`
	ItemID                  string          `json:"item_id,omitempty"`
	SequenceNo              int             `json:"sequence_no"`
	ImportedAt              string          `json:"imported_at"`
	LocalInstanceID         string          `json:"local_instance_id"`
	LocalSiteID             string          `json:"local_site_id,omitempty"`
	LocalFleetID            string          `json:"local_fleet_id,omitempty"`
	SourceType              string          `json:"source_type,omitempty"`
	SourceName              string          `json:"source_name,omitempty"`
	SourcePath              string          `json:"source_path,omitempty"`
	ValidationStatus        string          `json:"validation_status"`
	Validation              json.RawMessage `json:"validation"`
	Bundle                  json.RawMessage `json:"bundle"`
	Evidence                json.RawMessage `json:"evidence"`
	Event                   json.RawMessage `json:"event,omitempty"`
	Normalized              json.RawMessage `json:"normalized,omitempty"`
	ClaimedOriginInstanceID string          `json:"claimed_origin_instance_id,omitempty"`
	ClaimedOriginSiteID     string          `json:"claimed_origin_site_id,omitempty"`
	ClaimedFleetID          string          `json:"claimed_fleet_id,omitempty"`
	OriginInstanceID        string          `json:"origin_instance_id"`
	OriginSiteID            string          `json:"origin_site_id"`
	EvidenceClass           string          `json:"evidence_class"`
	ObservationOriginClass  string          `json:"observation_origin_class"`
	CorrelationID           string          `json:"correlation_id,omitempty"`
	ObservedAt              string          `json:"observed_at,omitempty"`
	ReceivedAt              string          `json:"received_at,omitempty"`
	RecordedAt              string          `json:"recorded_at,omitempty"`
	TimingPosture           string          `json:"timing_posture,omitempty"`
	MergeDisposition        string          `json:"merge_disposition,omitempty"`
	MergeCorrelationID      string          `json:"merge_correlation_id,omitempty"`
	Rejected                bool            `json:"rejected"`
}

// InsertImportedRemoteEvidence persists a new remote evidence import row.
func (d *DB) InsertImportedRemoteEvidence(rec ImportedRemoteEvidenceRecord) error {
	if strings.TrimSpace(rec.ID) == "" {
		return fmt.Errorf("id required")
	}
	if strings.TrimSpace(rec.BatchID) == "" {
		// Use ID as BatchID if not provided (for older or simple callers)
		rec.BatchID = rec.ID
	}
	sql, err := importedRemoteEvidenceInsertSQL(rec)
	if err != nil {
		return err
	}
	return d.Exec(sql)
}

// PersistRemoteImportBatch stores one batch, its imported items, and any materialized timeline events in one SQLite transaction.
func (d *DB) PersistRemoteImportBatch(batch RemoteImportBatchRecord, items []ImportedRemoteEvidenceRecord, timelineEvents []TimelineEvent) error {
	if strings.TrimSpace(batch.ID) == "" {
		return fmt.Errorf("batch id required")
	}
	if strings.TrimSpace(batch.ImportedAt) == "" {
		return fmt.Errorf("batch imported_at required")
	}
	if len(batch.Validation) == 0 {
		return fmt.Errorf("batch validation json required")
	}
	if len(batch.RawPayload) == 0 {
		return fmt.Errorf("batch raw payload json required")
	}
	statements := []string{"BEGIN IMMEDIATE;"}
	statements = append(statements, remoteImportBatchInsertSQL(batch))
	for _, item := range items {
		sql, err := importedRemoteEvidenceInsertSQL(item)
		if err != nil {
			return err
		}
		statements = append(statements, sql)
	}
	for _, ev := range timelineEvents {
		sql, err := timelineEventInsertSQL(ev)
		if err != nil {
			return err
		}
		statements = append(statements, sql)
	}
	statements = append(statements, "COMMIT;")
	return d.ExecScript(strings.Join(statements, "\n"))
}

func remoteImportBatchInsertSQL(rec RemoteImportBatchRecord) string {
	return fmt.Sprintf(`INSERT INTO remote_import_batches(id, imported_at, local_instance_id, local_site_id, local_fleet_id, source_type, source_name, source_path, support_bundle_id, format_kind, schema_version, claimed_origin_instance_id, claimed_origin_site_id, claimed_fleet_id, exported_at, capability_posture_json, validation_json, raw_payload_json, item_count, accepted_count, accepted_with_caveats_count, rejected_count, partial_success, note) VALUES('%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s',%d,%d,%d,%d,%d,'%s');`,
		esc(rec.ID), esc(rec.ImportedAt), esc(rec.LocalInstanceID), esc(rec.LocalSiteID), esc(rec.LocalFleetID),
		esc(rec.SourceType), esc(rec.SourceName), esc(rec.SourcePath), esc(rec.SupportBundleID),
		esc(rec.FormatKind), esc(rec.SchemaVersion),
		esc(rec.ClaimedOriginInstanceID), esc(rec.ClaimedOriginSiteID), esc(rec.ClaimedFleetID),
		esc(rec.ExportedAt), esc(string(rec.CapabilityPosture)), esc(string(rec.Validation)), esc(string(rec.RawPayload)),
		rec.ItemCount, rec.AcceptedCount, rec.AcceptedWithCaveatsCount, rec.RejectedCount, boolInt(rec.PartialSuccess), esc(rec.Note))
}

func importedRemoteEvidenceInsertSQL(rec ImportedRemoteEvidenceRecord) (string, error) {
	if strings.TrimSpace(rec.ID) == "" {
		return "", fmt.Errorf("import item id required")
	}
	if strings.TrimSpace(rec.BatchID) == "" {
		return "", fmt.Errorf("batch id required for imported item")
	}
	if len(rec.Validation) == 0 {
		return "", fmt.Errorf("validation json required")
	}
	if len(rec.Bundle) == 0 || len(rec.Evidence) == 0 {
		return "", fmt.Errorf("bundle and evidence json required")
	}
	rej := 0
	if rec.Rejected {
		rej = 1
	}
	return fmt.Sprintf(`INSERT INTO imported_remote_evidence(id, batch_id, item_id, sequence_no, imported_at, local_instance_id, local_site_id, local_fleet_id, source_type, source_name, source_path, validation_status, validation_json, bundle_json, evidence_json, event_json, normalized_json, claimed_origin_instance_id, claimed_origin_site_id, claimed_fleet_id, origin_instance_id, origin_site_id, evidence_class, observation_origin_class, correlation_id, observed_at, received_at, recorded_at, timing_posture, merge_disposition, merge_correlation_id, rejected) VALUES('%s','%s','%s',%d,'%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s',%d);`,
		esc(rec.ID), esc(rec.BatchID), esc(rec.ItemID), rec.SequenceNo,
		esc(rec.ImportedAt), esc(rec.LocalInstanceID), esc(rec.LocalSiteID), esc(rec.LocalFleetID),
		esc(rec.SourceType), esc(rec.SourceName), esc(rec.SourcePath),
		esc(rec.ValidationStatus), esc(string(rec.Validation)), esc(string(rec.Bundle)), esc(string(rec.Evidence)), esc(string(rec.Event)), esc(string(rec.Normalized)),
		esc(rec.ClaimedOriginInstanceID), esc(rec.ClaimedOriginSiteID), esc(rec.ClaimedFleetID),
		esc(rec.OriginInstanceID), esc(rec.OriginSiteID), esc(rec.EvidenceClass), esc(rec.ObservationOriginClass), esc(rec.CorrelationID),
		esc(rec.ObservedAt), esc(rec.ReceivedAt), esc(rec.RecordedAt),
		esc(rec.TimingPosture), esc(rec.MergeDisposition), esc(rec.MergeCorrelationID), rej), nil
}

func timelineEventInsertSQL(ev TimelineEvent) (string, error) {
	if strings.TrimSpace(ev.EventID) == "" {
		return "", fmt.Errorf("timeline event id is required")
	}
	if strings.TrimSpace(ev.EventType) == "" {
		return "", fmt.Errorf("timeline event type is required")
	}
	if strings.TrimSpace(ev.EventTime) == "" {
		return "", fmt.Errorf("timeline event time is required")
	}
	if strings.TrimSpace(ev.ActorID) == "" {
		ev.ActorID = "system"
	}
	if strings.TrimSpace(ev.ScopePosture) == "" {
		ev.ScopePosture = "local"
	}
	if strings.TrimSpace(ev.TimingPosture) == "" {
		ev.TimingPosture = "local_ordered"
	}
	if strings.TrimSpace(ev.MergeDisposition) == "" {
		ev.MergeDisposition = "raw_only"
	}
	detailsJSON, _ := json.Marshal(ev.Details)
	return fmt.Sprintf(`INSERT OR IGNORE INTO timeline_events(id,event_time,event_type,summary,severity,actor_id,resource_id,details_json,scope_posture,origin_instance_id,timing_posture,merge_disposition,merge_correlation_id,import_id) VALUES('%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s');`,
		esc(ev.EventID), esc(ev.EventTime), esc(ev.EventType), esc(ev.Summary),
		esc(ev.Severity), esc(ev.ActorID), esc(ev.ResourceID), esc(string(detailsJSON)),
		esc(ev.ScopePosture), esc(ev.OriginInstanceID), esc(ev.TimingPosture),
		esc(ev.MergeDisposition), esc(ev.MergeCorrelationID), esc(ev.ImportID)), nil
}

// ListRemoteImportBatches returns recent import batches newest first.
func (d *DB) ListRemoteImportBatches(limit int) ([]RemoteImportBatchRecord, error) {
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(
		`SELECT id, imported_at, local_instance_id, COALESCE(local_site_id,'') AS local_site_id, COALESCE(local_fleet_id,'') AS local_fleet_id, COALESCE(source_type,'') AS source_type, COALESCE(source_name,'') AS source_name, COALESCE(source_path,'') AS source_path, COALESCE(support_bundle_id,'') AS support_bundle_id, format_kind, schema_version, COALESCE(claimed_origin_instance_id,'') AS claimed_origin_instance_id, COALESCE(claimed_origin_site_id,'') AS claimed_origin_site_id, COALESCE(claimed_fleet_id,'') AS claimed_fleet_id, COALESCE(exported_at,'') AS exported_at, COALESCE(capability_posture_json,'{}') AS capability_posture_json, validation_json, item_count, accepted_count, accepted_with_caveats_count, rejected_count, partial_success, COALESCE(note,'') AS note FROM remote_import_batches ORDER BY imported_at DESC LIMIT %d;`, limit))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil, nil
		}
		return nil, err
	}
	out := make([]RemoteImportBatchRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, remoteImportBatchFromRow(row, false))
	}
	return out, nil
}

// GetRemoteImportBatch returns one batch by id.
func (d *DB) GetRemoteImportBatch(id string) (RemoteImportBatchRecord, bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return RemoteImportBatchRecord{}, false, fmt.Errorf("id required")
	}
	rows, err := d.QueryRows(fmt.Sprintf(
		`SELECT id, imported_at, local_instance_id, COALESCE(local_site_id,'') AS local_site_id, COALESCE(local_fleet_id,'') AS local_fleet_id, COALESCE(source_type,'') AS source_type, COALESCE(source_name,'') AS source_name, COALESCE(source_path,'') AS source_path, COALESCE(support_bundle_id,'') AS support_bundle_id, format_kind, schema_version, COALESCE(claimed_origin_instance_id,'') AS claimed_origin_instance_id, COALESCE(claimed_origin_site_id,'') AS claimed_origin_site_id, COALESCE(claimed_fleet_id,'') AS claimed_fleet_id, COALESCE(exported_at,'') AS exported_at, COALESCE(capability_posture_json,'{}') AS capability_posture_json, validation_json, raw_payload_json, item_count, accepted_count, accepted_with_caveats_count, rejected_count, partial_success, COALESCE(note,'') AS note FROM remote_import_batches WHERE id='%s' LIMIT 1;`, esc(id)))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return RemoteImportBatchRecord{}, false, nil
		}
		return RemoteImportBatchRecord{}, false, err
	}
	if len(rows) == 0 {
		return RemoteImportBatchRecord{}, false, nil
	}
	return remoteImportBatchFromRow(rows[0], true), true, nil
}

func remoteImportBatchFromRow(row map[string]any, includeRaw bool) RemoteImportBatchRecord {
	rec := RemoteImportBatchRecord{
		ID:                       asString(row["id"]),
		ImportedAt:               asString(row["imported_at"]),
		LocalInstanceID:          asString(row["local_instance_id"]),
		LocalSiteID:              asString(row["local_site_id"]),
		LocalFleetID:             asString(row["local_fleet_id"]),
		SourceType:               asString(row["source_type"]),
		SourceName:               asString(row["source_name"]),
		SourcePath:               asString(row["source_path"]),
		SupportBundleID:          asString(row["support_bundle_id"]),
		FormatKind:               asString(row["format_kind"]),
		SchemaVersion:            asString(row["schema_version"]),
		ClaimedOriginInstanceID:  asString(row["claimed_origin_instance_id"]),
		ClaimedOriginSiteID:      asString(row["claimed_origin_site_id"]),
		ClaimedFleetID:           asString(row["claimed_fleet_id"]),
		ExportedAt:               asString(row["exported_at"]),
		CapabilityPosture:        json.RawMessage(asString(row["capability_posture_json"])),
		Validation:               json.RawMessage(asString(row["validation_json"])),
		ItemCount:                int(asInt(row["item_count"])),
		AcceptedCount:            int(asInt(row["accepted_count"])),
		AcceptedWithCaveatsCount: int(asInt(row["accepted_with_caveats_count"])),
		RejectedCount:            int(asInt(row["rejected_count"])),
		PartialSuccess:           asString(row["partial_success"]) == "1" || asInt(row["partial_success"]) == 1,
		Note:                     asString(row["note"]),
	}
	if includeRaw {
		rec.RawPayload = json.RawMessage(asString(row["raw_payload_json"]))
	}
	return rec
}

// ListImportedRemoteEvidence returns recent imported item rows newest first.
func (d *DB) ListImportedRemoteEvidence(limit int) ([]ImportedRemoteEvidenceRecord, error) {
	return d.listImportedRemoteEvidenceWhere("", limit)
}

// ImportedRemoteEvidenceByBatch returns imported item rows for one batch ordered by sequence.
func (d *DB) ImportedRemoteEvidenceByBatch(batchID string) ([]ImportedRemoteEvidenceRecord, error) {
	batchID = strings.TrimSpace(batchID)
	if batchID == "" {
		return nil, fmt.Errorf("batch id required")
	}
	return d.listImportedRemoteEvidenceWhere(fmt.Sprintf("WHERE batch_id='%s'", esc(batchID)), MaxRows)
}

func (d *DB) listImportedRemoteEvidenceWhere(where string, limit int) ([]ImportedRemoteEvidenceRecord, error) {
	limit = clampLimit(limit)
	order := "ORDER BY imported_at DESC, sequence_no ASC"
	if strings.Contains(where, "batch_id=") {
		order = "ORDER BY sequence_no ASC, imported_at ASC"
	}
	rows, err := d.QueryRows(fmt.Sprintf(
		`SELECT id, batch_id, COALESCE(item_id,'') AS item_id, COALESCE(sequence_no,0) AS sequence_no, imported_at, local_instance_id, COALESCE(local_site_id,'') AS local_site_id, COALESCE(local_fleet_id,'') AS local_fleet_id, COALESCE(source_type,'') AS source_type, COALESCE(source_name,'') AS source_name, COALESCE(source_path,'') AS source_path, COALESCE(validation_status,'') AS validation_status, validation_json, bundle_json, evidence_json, COALESCE(event_json,'') AS event_json, COALESCE(normalized_json,'') AS normalized_json, COALESCE(claimed_origin_instance_id,'') AS claimed_origin_instance_id, COALESCE(claimed_origin_site_id,'') AS claimed_origin_site_id, COALESCE(claimed_fleet_id,'') AS claimed_fleet_id, origin_instance_id, origin_site_id, evidence_class, observation_origin_class, COALESCE(correlation_id,'') AS correlation_id, COALESCE(observed_at,'') AS observed_at, COALESCE(received_at,'') AS received_at, COALESCE(recorded_at,'') AS recorded_at, COALESCE(timing_posture,'') AS timing_posture, COALESCE(merge_disposition,'') AS merge_disposition, COALESCE(merge_correlation_id,'') AS merge_correlation_id, rejected FROM imported_remote_evidence %s %s LIMIT %d;`, where, order, limit))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil, nil
		}
		return nil, err
	}
	out := make([]ImportedRemoteEvidenceRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, importedRemoteEvidenceFromRow(row))
	}
	return out, nil
}

// GetImportedRemoteEvidence returns one item row by id.
func (d *DB) GetImportedRemoteEvidence(id string) (ImportedRemoteEvidenceRecord, bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return ImportedRemoteEvidenceRecord{}, false, fmt.Errorf("id required")
	}
	rows, err := d.QueryRows(fmt.Sprintf(
		`SELECT id, batch_id, COALESCE(item_id,'') AS item_id, COALESCE(sequence_no,0) AS sequence_no, imported_at, local_instance_id, COALESCE(local_site_id,'') AS local_site_id, COALESCE(local_fleet_id,'') AS local_fleet_id, COALESCE(source_type,'') AS source_type, COALESCE(source_name,'') AS source_name, COALESCE(source_path,'') AS source_path, COALESCE(validation_status,'') AS validation_status, validation_json, bundle_json, evidence_json, COALESCE(event_json,'') AS event_json, COALESCE(normalized_json,'') AS normalized_json, COALESCE(claimed_origin_instance_id,'') AS claimed_origin_instance_id, COALESCE(claimed_origin_site_id,'') AS claimed_origin_site_id, COALESCE(claimed_fleet_id,'') AS claimed_fleet_id, origin_instance_id, origin_site_id, evidence_class, observation_origin_class, COALESCE(correlation_id,'') AS correlation_id, COALESCE(observed_at,'') AS observed_at, COALESCE(received_at,'') AS received_at, COALESCE(recorded_at,'') AS recorded_at, COALESCE(timing_posture,'') AS timing_posture, COALESCE(merge_disposition,'') AS merge_disposition, COALESCE(merge_correlation_id,'') AS merge_correlation_id, rejected FROM imported_remote_evidence WHERE id='%s' LIMIT 1;`, esc(id)))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return ImportedRemoteEvidenceRecord{}, false, nil
		}
		return ImportedRemoteEvidenceRecord{}, false, err
	}
	if len(rows) == 0 {
		return ImportedRemoteEvidenceRecord{}, false, nil
	}
	return importedRemoteEvidenceFromRow(rows[0]), true, nil
}

func importedRemoteEvidenceFromRow(row map[string]any) ImportedRemoteEvidenceRecord {
	return ImportedRemoteEvidenceRecord{
		ID:                      asString(row["id"]),
		BatchID:                 asString(row["batch_id"]),
		ItemID:                  asString(row["item_id"]),
		SequenceNo:              int(asInt(row["sequence_no"])),
		ImportedAt:              asString(row["imported_at"]),
		LocalInstanceID:         asString(row["local_instance_id"]),
		LocalSiteID:             asString(row["local_site_id"]),
		LocalFleetID:            asString(row["local_fleet_id"]),
		SourceType:              asString(row["source_type"]),
		SourceName:              asString(row["source_name"]),
		SourcePath:              asString(row["source_path"]),
		ValidationStatus:        asString(row["validation_status"]),
		Validation:              json.RawMessage(asString(row["validation_json"])),
		Bundle:                  json.RawMessage(asString(row["bundle_json"])),
		Evidence:                json.RawMessage(asString(row["evidence_json"])),
		Event:                   json.RawMessage(asString(row["event_json"])),
		Normalized:              json.RawMessage(asString(row["normalized_json"])),
		ClaimedOriginInstanceID: asString(row["claimed_origin_instance_id"]),
		ClaimedOriginSiteID:     asString(row["claimed_origin_site_id"]),
		ClaimedFleetID:          asString(row["claimed_fleet_id"]),
		OriginInstanceID:        asString(row["origin_instance_id"]),
		OriginSiteID:            asString(row["origin_site_id"]),
		EvidenceClass:           asString(row["evidence_class"]),
		ObservationOriginClass:  asString(row["observation_origin_class"]),
		CorrelationID:           asString(row["correlation_id"]),
		ObservedAt:              asString(row["observed_at"]),
		ReceivedAt:              asString(row["received_at"]),
		RecordedAt:              asString(row["recorded_at"]),
		TimingPosture:           asString(row["timing_posture"]),
		MergeDisposition:        asString(row["merge_disposition"]),
		MergeCorrelationID:      asString(row["merge_correlation_id"]),
		Rejected:                asString(row["rejected"]) == "1" || asInt(row["rejected"]) == 1,
	}
}
