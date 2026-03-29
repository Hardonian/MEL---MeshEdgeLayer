package db

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/models"
)

// --- Incident fingerprints (migration 0032) ---

type IncidentFingerprintRecord struct {
	IncidentID               string
	FingerprintSchemaVersion string
	ProfileVersion           string
	LegacySignatureKey       string
	CanonicalHash            string
	ComponentJSON            string
	SparsityJSON             string
	ComputedAt               string
}

func (d *DB) UpsertIncidentFingerprint(rec IncidentFingerprintRecord) error {
	if strings.TrimSpace(rec.IncidentID) == "" {
		return fmt.Errorf("incident id is required")
	}
	if rec.FingerprintSchemaVersion == "" {
		rec.FingerprintSchemaVersion = "mel.incident_fingerprint/v1"
	}
	if rec.ProfileVersion == "" {
		rec.ProfileVersion = "weights/v1"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if rec.ComputedAt == "" {
		rec.ComputedAt = now
	}
	if rec.ComponentJSON == "" {
		rec.ComponentJSON = "{}"
	}
	if rec.SparsityJSON == "" {
		rec.SparsityJSON = "[]"
	}
	sql := fmt.Sprintf(`INSERT INTO incident_fingerprints(incident_id,fingerprint_schema_version,profile_version,legacy_signature_key,canonical_hash,component_json,sparsity_json,computed_at)
VALUES('%s','%s','%s','%s','%s','%s','%s','%s')
ON CONFLICT(incident_id) DO UPDATE SET
fingerprint_schema_version=excluded.fingerprint_schema_version,
profile_version=excluded.profile_version,
legacy_signature_key=excluded.legacy_signature_key,
canonical_hash=excluded.canonical_hash,
component_json=excluded.component_json,
sparsity_json=excluded.sparsity_json,
computed_at=excluded.computed_at;`,
		esc(rec.IncidentID), esc(rec.FingerprintSchemaVersion), esc(rec.ProfileVersion),
		esc(rec.LegacySignatureKey), esc(rec.CanonicalHash), esc(rec.ComponentJSON), esc(rec.SparsityJSON), esc(rec.ComputedAt))
	err := d.Exec(sql)
	if err != nil && strings.Contains(err.Error(), "no such table") {
		return nil
	}
	return err
}

func (d *DB) IncidentFingerprintByID(incidentID string) (IncidentFingerprintRecord, bool, error) {
	incidentID = strings.TrimSpace(incidentID)
	if incidentID == "" {
		return IncidentFingerprintRecord{}, false, nil
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT incident_id,fingerprint_schema_version,profile_version,legacy_signature_key,canonical_hash,COALESCE(component_json,'{}'),COALESCE(sparsity_json,'[]'),computed_at
FROM incident_fingerprints WHERE incident_id='%s' LIMIT 1;`, esc(incidentID)))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return IncidentFingerprintRecord{}, false, nil
		}
		return IncidentFingerprintRecord{}, false, err
	}
	if len(rows) == 0 {
		return IncidentFingerprintRecord{}, false, nil
	}
	row := rows[0]
	return IncidentFingerprintRecord{
		IncidentID:               asString(row["incident_id"]),
		FingerprintSchemaVersion: asString(row["fingerprint_schema_version"]),
		ProfileVersion:           asString(row["profile_version"]),
		LegacySignatureKey:       asString(row["legacy_signature_key"]),
		CanonicalHash:            asString(row["canonical_hash"]),
		ComponentJSON:            asString(row["component_json"]),
		SparsityJSON:             asString(row["sparsity_json"]),
		ComputedAt:               asString(row["computed_at"]),
	}, true, nil
}

func (d *DB) IncidentFingerprintsByLegacySignature(signatureKey, excludeIncidentID string, limit int) ([]IncidentFingerprintRecord, error) {
	signatureKey = strings.TrimSpace(signatureKey)
	if signatureKey == "" {
		return nil, nil
	}
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT incident_id,fingerprint_schema_version,profile_version,legacy_signature_key,canonical_hash,COALESCE(component_json,'{}'),COALESCE(sparsity_json,'[]'),computed_at
FROM incident_fingerprints WHERE legacy_signature_key='%s' AND incident_id!='%s' ORDER BY computed_at DESC LIMIT %d;`,
		esc(signatureKey), esc(excludeIncidentID), limit))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil, nil
		}
		return nil, err
	}
	return fingerprintRowsToRecords(rows), nil
}

func fingerprintRowsToRecords(rows []map[string]any) []IncidentFingerprintRecord {
	out := make([]IncidentFingerprintRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, IncidentFingerprintRecord{
			IncidentID:               asString(row["incident_id"]),
			FingerprintSchemaVersion: asString(row["fingerprint_schema_version"]),
			ProfileVersion:           asString(row["profile_version"]),
			LegacySignatureKey:       asString(row["legacy_signature_key"]),
			CanonicalHash:            asString(row["canonical_hash"]),
			ComponentJSON:            asString(row["component_json"]),
			SparsityJSON:             asString(row["sparsity_json"]),
			ComputedAt:               asString(row["computed_at"]),
		})
	}
	return out
}

// --- Recommendation effectiveness aggregates ---

type RecEffectivenessRecord struct {
	ScopeKey          string
	RecommendationID  string
	TotalCount        int
	AcceptedCount     int
	RejectedCount     int
	IneffectiveCount  int
	WorsenedCount     int
	ModifiedCount     int
	NotAttemptedCount int
	UnknownCount      int
	LastOutcomeAt     string
	UpdatedAt         string
}

func (d *DB) AccumulateRecommendationEffectiveness(scopeKey, recommendationID, outcome string) error {
	scopeKey = strings.TrimSpace(scopeKey)
	recommendationID = strings.TrimSpace(recommendationID)
	outcome = strings.ToLower(strings.TrimSpace(outcome))
	if scopeKey == "" || recommendationID == "" || outcome == "" {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT total_count,accepted_count,rejected_count,ineffective_count,worsened_count,modified_count,not_attempted_count,unknown_count
FROM incident_rec_effectiveness WHERE scope_key='%s' AND recommendation_id='%s' LIMIT 1;`, esc(scopeKey), esc(recommendationID)))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil
		}
		return err
	}
	var rec RecEffectivenessRecord
	rec.ScopeKey = scopeKey
	rec.RecommendationID = recommendationID
	rec.LastOutcomeAt = now
	rec.UpdatedAt = now
	if len(rows) > 0 {
		row := rows[0]
		rec.TotalCount = int(asInt(row["total_count"]))
		rec.AcceptedCount = int(asInt(row["accepted_count"]))
		rec.RejectedCount = int(asInt(row["rejected_count"]))
		rec.IneffectiveCount = int(asInt(row["ineffective_count"]))
		rec.WorsenedCount = int(asInt(row["worsened_count"]))
		rec.ModifiedCount = int(asInt(row["modified_count"]))
		rec.NotAttemptedCount = int(asInt(row["not_attempted_count"]))
		rec.UnknownCount = int(asInt(row["unknown_count"]))
	}
	rec.TotalCount++
	switch outcome {
	case "accepted", "resolved_incident":
		rec.AcceptedCount++
	case "rejected":
		rec.RejectedCount++
	case "ineffective":
		rec.IneffectiveCount++
	case "worsened":
		rec.WorsenedCount++
	case "modified":
		rec.ModifiedCount++
	case "not_attempted":
		rec.NotAttemptedCount++
	default:
		rec.UnknownCount++
	}
	sql := fmt.Sprintf(`INSERT INTO incident_rec_effectiveness(scope_key,recommendation_id,total_count,accepted_count,rejected_count,ineffective_count,worsened_count,modified_count,not_attempted_count,unknown_count,last_outcome_at,updated_at)
VALUES('%s','%s',%d,%d,%d,%d,%d,%d,%d,%d,'%s','%s')
ON CONFLICT(scope_key,recommendation_id) DO UPDATE SET
total_count=excluded.total_count,
accepted_count=excluded.accepted_count,
rejected_count=excluded.rejected_count,
ineffective_count=excluded.ineffective_count,
worsened_count=excluded.worsened_count,
modified_count=excluded.modified_count,
not_attempted_count=excluded.not_attempted_count,
unknown_count=excluded.unknown_count,
last_outcome_at=excluded.last_outcome_at,
updated_at=excluded.updated_at;`,
		esc(rec.ScopeKey), esc(rec.RecommendationID), rec.TotalCount, rec.AcceptedCount, rec.RejectedCount, rec.IneffectiveCount,
		rec.WorsenedCount, rec.ModifiedCount, rec.NotAttemptedCount, rec.UnknownCount, esc(rec.LastOutcomeAt), esc(rec.UpdatedAt))
	err = d.Exec(sql)
	if err != nil && strings.Contains(err.Error(), "no such table") {
		return nil
	}
	return err
}

func (d *DB) RecEffectivenessByScope(scopeKey string) ([]RecEffectivenessRecord, error) {
	scopeKey = strings.TrimSpace(scopeKey)
	if scopeKey == "" {
		return nil, nil
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT scope_key,recommendation_id,total_count,accepted_count,rejected_count,ineffective_count,worsened_count,modified_count,not_attempted_count,unknown_count,last_outcome_at,updated_at
FROM incident_rec_effectiveness WHERE scope_key='%s';`, esc(scopeKey)))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil, nil
		}
		return nil, err
	}
	out := make([]RecEffectivenessRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, RecEffectivenessRecord{
			ScopeKey:          asString(row["scope_key"]),
			RecommendationID:  asString(row["recommendation_id"]),
			TotalCount:        int(asInt(row["total_count"])),
			AcceptedCount:     int(asInt(row["accepted_count"])),
			RejectedCount:     int(asInt(row["rejected_count"])),
			IneffectiveCount:  int(asInt(row["ineffective_count"])),
			WorsenedCount:     int(asInt(row["worsened_count"])),
			ModifiedCount:     int(asInt(row["modified_count"])),
			NotAttemptedCount: int(asInt(row["not_attempted_count"])),
			UnknownCount:      int(asInt(row["unknown_count"])),
			LastOutcomeAt:     asString(row["last_outcome_at"]),
			UpdatedAt:         asString(row["updated_at"]),
		})
	}
	return out, nil
}

// --- Runbook entries ---

type RunbookEntryRecord struct {
	ID                       string
	Status                   string
	SourceKind               string
	LegacySignatureKey       string
	FingerprintCanonicalHash string
	Title                    string
	Body                     string
	EvidenceRefJSON          string
	SourceIncidentIDsJSON    string
	PromotionBasis           string
	CreatedAt                string
	UpdatedAt                string
	ReviewedAt               string
	ReviewerActorID          string
}

func (d *DB) InsertRunbookEntry(rec RunbookEntryRecord) error {
	if strings.TrimSpace(rec.ID) == "" {
		return fmt.Errorf("runbook id required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if rec.CreatedAt == "" {
		rec.CreatedAt = now
	}
	if rec.UpdatedAt == "" {
		rec.UpdatedAt = now
	}
	if rec.EvidenceRefJSON == "" {
		rec.EvidenceRefJSON = "[]"
	}
	if rec.SourceIncidentIDsJSON == "" {
		rec.SourceIncidentIDsJSON = "[]"
	}
	sql := fmt.Sprintf(`INSERT OR IGNORE INTO incident_runbook_entries(id,status,source_kind,legacy_signature_key,fingerprint_canonical_hash,title,body,evidence_ref_json,source_incident_ids_json,promotion_basis,created_at,updated_at,reviewed_at,reviewer_actor_id)
VALUES('%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s');`,
		esc(rec.ID), esc(rec.Status), esc(rec.SourceKind), esc(rec.LegacySignatureKey), esc(rec.FingerprintCanonicalHash),
		esc(rec.Title), esc(rec.Body), esc(rec.EvidenceRefJSON), esc(rec.SourceIncidentIDsJSON), esc(rec.PromotionBasis),
		esc(rec.CreatedAt), esc(rec.UpdatedAt), esc(rec.ReviewedAt), esc(rec.ReviewerActorID))
	err := d.Exec(sql)
	if err != nil && strings.Contains(err.Error(), "no such table") {
		return nil
	}
	return err
}

func (d *DB) RunbookEntriesForSignature(signatureKey string, limit int) ([]RunbookEntryRecord, error) {
	signatureKey = strings.TrimSpace(signatureKey)
	if signatureKey == "" {
		return nil, nil
	}
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id,status,source_kind,legacy_signature_key,fingerprint_canonical_hash,title,body,evidence_ref_json,source_incident_ids_json,promotion_basis,created_at,updated_at,reviewed_at,reviewer_actor_id
FROM incident_runbook_entries WHERE legacy_signature_key='%s' ORDER BY updated_at DESC LIMIT %d;`, esc(signatureKey), limit))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil, nil
		}
		return nil, err
	}
	return runbookRows(rows), nil
}

func (d *DB) RunbookEntriesForFingerprintHash(h string, limit int) ([]RunbookEntryRecord, error) {
	h = strings.TrimSpace(h)
	if h == "" {
		return nil, nil
	}
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id,status,source_kind,legacy_signature_key,fingerprint_canonical_hash,title,body,evidence_ref_json,source_incident_ids_json,promotion_basis,created_at,updated_at,reviewed_at,reviewer_actor_id
FROM incident_runbook_entries WHERE fingerprint_canonical_hash='%s' ORDER BY updated_at DESC LIMIT %d;`, esc(h), limit))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil, nil
		}
		return nil, err
	}
	return runbookRows(rows), nil
}

func runbookRows(rows []map[string]any) []RunbookEntryRecord {
	out := make([]RunbookEntryRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, RunbookEntryRecord{
			ID:                       asString(row["id"]),
			Status:                   asString(row["status"]),
			SourceKind:               asString(row["source_kind"]),
			LegacySignatureKey:       asString(row["legacy_signature_key"]),
			FingerprintCanonicalHash: asString(row["fingerprint_canonical_hash"]),
			Title:                    asString(row["title"]),
			Body:                     asString(row["body"]),
			EvidenceRefJSON:          asString(row["evidence_ref_json"]),
			SourceIncidentIDsJSON:    asString(row["source_incident_ids_json"]),
			PromotionBasis:           asString(row["promotion_basis"]),
			CreatedAt:                asString(row["created_at"]),
			UpdatedAt:                asString(row["updated_at"]),
			ReviewedAt:               asString(row["reviewed_at"]),
			ReviewerActorID:          asString(row["reviewer_actor_id"]),
		})
	}
	return out
}

// --- Fault domains ---

type FaultDomainRecord struct {
	ID                 string
	DomainKey          string
	Basis              string
	Uncertainty        string
	RationaleJSON      string
	EvidenceBundleJSON string
	CreatedAt          string
	UpdatedAt          string
}

func (d *DB) UpsertFaultDomain(rec FaultDomainRecord) error {
	if strings.TrimSpace(rec.ID) == "" || strings.TrimSpace(rec.DomainKey) == "" {
		return fmt.Errorf("fault domain id and domain_key required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if rec.CreatedAt == "" {
		rec.CreatedAt = now
	}
	if rec.UpdatedAt == "" {
		rec.UpdatedAt = now
	}
	if rec.RationaleJSON == "" {
		rec.RationaleJSON = "[]"
	}
	if rec.EvidenceBundleJSON == "" {
		rec.EvidenceBundleJSON = "{}"
	}
	sql := fmt.Sprintf(`INSERT INTO incident_fault_domains(id,domain_key,basis,uncertainty,rationale_json,evidence_bundle_json,created_at,updated_at)
VALUES('%s','%s','%s','%s','%s','%s','%s','%s')
ON CONFLICT(domain_key) DO UPDATE SET
basis=excluded.basis,
uncertainty=excluded.uncertainty,
rationale_json=excluded.rationale_json,
evidence_bundle_json=excluded.evidence_bundle_json,
updated_at=excluded.updated_at;`,
		esc(rec.ID), esc(rec.DomainKey), esc(rec.Basis), esc(rec.Uncertainty),
		esc(rec.RationaleJSON), esc(rec.EvidenceBundleJSON), esc(rec.CreatedAt), esc(rec.UpdatedAt))
	err := d.Exec(sql)
	if err != nil && strings.Contains(err.Error(), "no such table") {
		return nil
	}
	return err
}

// FaultDomainMember is a single row for incident_fault_domain_members.
type FaultDomainMember struct {
	Kind     string
	ID       string
	Reason   string
	JoinedAt string
}

func (d *DB) ReplaceFaultDomainMembers(domainID string, members []FaultDomainMember) error {
	domainID = strings.TrimSpace(domainID)
	if domainID == "" {
		return nil
	}
	del := fmt.Sprintf(`DELETE FROM incident_fault_domain_members WHERE domain_id='%s';`, esc(domainID))
	if err := d.Exec(del); err != nil && !strings.Contains(err.Error(), "no such table") {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, m := range members {
		if strings.TrimSpace(m.ID) == "" || strings.TrimSpace(m.Kind) == "" {
			continue
		}
		ja := m.JoinedAt
		if ja == "" {
			ja = now
		}
		q := fmt.Sprintf(`INSERT OR REPLACE INTO incident_fault_domain_members(domain_id,member_kind,member_id,joined_reason,joined_at) VALUES('%s','%s','%s','%s','%s');`,
			esc(domainID), esc(m.Kind), esc(m.ID), esc(m.Reason), esc(ja))
		if err := d.Exec(q); err != nil && !strings.Contains(err.Error(), "no such table") {
			return err
		}
	}
	return nil
}

func (d *DB) FaultDomainsForIncident(incidentID string) ([]models.IncidentFaultDomain, error) {
	incidentID = strings.TrimSpace(incidentID)
	if incidentID == "" {
		return nil, nil
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT d.id,d.domain_key,d.basis,d.uncertainty,d.rationale_json,d.evidence_bundle_json
FROM incident_fault_domains d
JOIN incident_fault_domain_members m ON m.domain_id=d.id
WHERE m.member_kind='incident' AND m.member_id='%s'
ORDER BY d.updated_at DESC;`, esc(incidentID)))
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil, nil
		}
		return nil, err
	}
	out := make([]models.IncidentFaultDomain, 0, len(rows))
	for _, row := range rows {
		var rationale []string
		_ = json.Unmarshal([]byte(asString(row["rationale_json"])), &rationale)
		bundle := map[string]string{}
		_ = json.Unmarshal([]byte(asString(row["evidence_bundle_json"])), &bundle)
		kinds, _ := d.faultDomainMemberKindList(asString(row["id"]))
		out = append(out, models.IncidentFaultDomain{
			DomainID:       asString(row["id"]),
			DomainKey:      asString(row["domain_key"]),
			Basis:          asString(row["basis"]),
			Uncertainty:    asString(row["uncertainty"]),
			Rationale:      rationale,
			EvidenceBundle: bundle,
			MemberKinds:    kinds,
		})
	}
	return out, nil
}

func (d *DB) faultDomainMemberKindList(domainID string) ([]string, error) {
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT DISTINCT member_kind FROM incident_fault_domain_members WHERE domain_id='%s';`, esc(domainID)))
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	for _, row := range rows {
		k := asString(row["member_kind"])
		if k != "" {
			seen[k] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}

// SignatureKeyForIncident returns the most recently linked signature key for an incident.
func (d *DB) SignatureKeyForIncident(incidentID string) (string, error) {
	incidentID = strings.TrimSpace(incidentID)
	if incidentID == "" {
		return "", nil
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT signature_key FROM incident_signature_incidents WHERE incident_id='%s' ORDER BY linked_at DESC LIMIT 1;`, esc(incidentID)))
	if err != nil || len(rows) == 0 {
		return "", err
	}
	return strings.TrimSpace(asString(rows[0]["signature_key"])), nil
}

// RecordToIncidentFingerprint converts DB row to API model.
func RecordToIncidentFingerprint(rec IncidentFingerprintRecord) (models.IncidentFingerprint, error) {
	var comp map[string][]string
	_ = json.Unmarshal([]byte(rec.ComponentJSON), &comp)
	if comp == nil {
		comp = map[string][]string{}
	}
	var sparse []string
	_ = json.Unmarshal([]byte(rec.SparsityJSON), &sparse)
	return models.IncidentFingerprint{
		SchemaVersion:      rec.FingerprintSchemaVersion,
		ProfileVersion:     rec.ProfileVersion,
		LegacySignatureKey: rec.LegacySignatureKey,
		CanonicalHash:      rec.CanonicalHash,
		Components:         comp,
		SparsityMarkers:    sparse,
		ComputedAt:         rec.ComputedAt,
	}, nil
}
