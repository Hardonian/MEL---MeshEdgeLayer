package db

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ImportedRemoteEvidenceRecord is a persisted imported bundle row (local DB is not global authority).
// JSON fields are stored verbatim for audit; consumers may unmarshal into domain types in service/API layers.
type ImportedRemoteEvidenceRecord struct {
	ID                     string          `json:"id"`
	ImportedAt             string          `json:"imported_at"`
	LocalInstanceID        string          `json:"local_instance_id"`
	Validation             json.RawMessage `json:"validation"`
	Bundle                 json.RawMessage `json:"bundle"`
	Evidence               json.RawMessage `json:"evidence"`
	OriginInstanceID       string          `json:"origin_instance_id"`
	OriginSiteID           string          `json:"origin_site_id"`
	EvidenceClass          string          `json:"evidence_class"`
	ObservationOriginClass string          `json:"observation_origin_class"`
	Rejected               bool            `json:"rejected"`
}

// InsertImportedRemoteEvidence persists an import attempt (accepted or rejected audit).
func (d *DB) InsertImportedRemoteEvidence(rec ImportedRemoteEvidenceRecord) error {
	if strings.TrimSpace(rec.ID) == "" {
		return fmt.Errorf("import id required")
	}
	if len(rec.Validation) == 0 {
		return fmt.Errorf("validation json required")
	}
	if len(rec.Bundle) == 0 || len(rec.Evidence) == 0 {
		return fmt.Errorf("bundle and evidence json required")
	}
	rej := 0
	if rec.Rejected {
		rej = 1
	}
	sql := fmt.Sprintf(`INSERT INTO imported_remote_evidence(
id, imported_at, local_instance_id, validation_json, bundle_json, evidence_json,
origin_instance_id, origin_site_id, evidence_class, observation_origin_class, rejected
) VALUES('%s','%s','%s','%s','%s','%s','%s','%s','%s','%s',%d);`,
		esc(rec.ID), esc(rec.ImportedAt), esc(rec.LocalInstanceID),
		esc(string(rec.Validation)), esc(string(rec.Bundle)), esc(string(rec.Evidence)),
		esc(rec.OriginInstanceID), esc(rec.OriginSiteID), esc(rec.EvidenceClass),
		esc(rec.ObservationOriginClass), rej)
	return d.Exec(sql)
}

// ListImportedRemoteEvidence returns recent imports newest first.
func (d *DB) ListImportedRemoteEvidence(limit int) ([]ImportedRemoteEvidenceRecord, error) {
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(
		`SELECT id, imported_at, local_instance_id, validation_json, bundle_json, evidence_json,
origin_instance_id, origin_site_id, evidence_class, observation_origin_class, rejected
FROM imported_remote_evidence ORDER BY imported_at DESC LIMIT %d;`, limit))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil, nil
		}
		return nil, err
	}
	out := make([]ImportedRemoteEvidenceRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, ImportedRemoteEvidenceRecord{
			ID:                     asString(row["id"]),
			ImportedAt:             asString(row["imported_at"]),
			LocalInstanceID:        asString(row["local_instance_id"]),
			Validation:             json.RawMessage(asString(row["validation_json"])),
			Bundle:                 json.RawMessage(asString(row["bundle_json"])),
			Evidence:               json.RawMessage(asString(row["evidence_json"])),
			OriginInstanceID:       asString(row["origin_instance_id"]),
			OriginSiteID:           asString(row["origin_site_id"]),
			EvidenceClass:          asString(row["evidence_class"]),
			ObservationOriginClass: asString(row["observation_origin_class"]),
			Rejected:               asString(row["rejected"]) == "1",
		})
	}
	return out, nil
}

// GetImportedRemoteEvidence returns one record by id.
func (d *DB) GetImportedRemoteEvidence(id string) (ImportedRemoteEvidenceRecord, bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return ImportedRemoteEvidenceRecord{}, false, fmt.Errorf("id required")
	}
	rows, err := d.QueryRows(fmt.Sprintf(
		`SELECT id, imported_at, local_instance_id, validation_json, bundle_json, evidence_json,
origin_instance_id, origin_site_id, evidence_class, observation_origin_class, rejected
FROM imported_remote_evidence WHERE id='%s' LIMIT 1;`, esc(id)))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return ImportedRemoteEvidenceRecord{}, false, nil
		}
		return ImportedRemoteEvidenceRecord{}, false, err
	}
	if len(rows) == 0 {
		return ImportedRemoteEvidenceRecord{}, false, nil
	}
	row := rows[0]
	rec := ImportedRemoteEvidenceRecord{
		ID:                     asString(row["id"]),
		ImportedAt:             asString(row["imported_at"]),
		LocalInstanceID:        asString(row["local_instance_id"]),
		Validation:             json.RawMessage(asString(row["validation_json"])),
		Bundle:                 json.RawMessage(asString(row["bundle_json"])),
		Evidence:               json.RawMessage(asString(row["evidence_json"])),
		OriginInstanceID:       asString(row["origin_instance_id"]),
		OriginSiteID:           asString(row["origin_site_id"]),
		EvidenceClass:          asString(row["evidence_class"]),
		ObservationOriginClass: asString(row["observation_origin_class"]),
		Rejected:               asString(row["rejected"]) == "1",
	}
	return rec, true, nil
}
