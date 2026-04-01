package db

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/models"
)

type IncidentSignatureRecord struct {
	SignatureKey      string
	SignatureLabel    string
	Category          string
	ResourceType      string
	ReasonKey         string
	FirstSeenAt       string
	LastSeenAt        string
	MatchCount        int
	ExampleIncidentID string
	LastSummary       string
}

type IncidentActionOutcomeSnapshotRecord struct {
	SnapshotID            string
	SignatureKey          string
	IncidentID            string
	ActionID              string
	ActionType            string
	ActionLabel           string
	DerivedClassification string
	EvidenceSufficiency   string
	PreActionSummary      map[string]any
	PostActionSummary     map[string]any
	ObservedSignalCount   int
	Caveats               []string
	InspectBeforeReuse    []string
	EvidenceRefs          []string
	AssociationOnly       bool
	WindowStart           string
	WindowEnd             string
	DerivationVersion     string
	SchemaVersion         string
	DerivedAt             string
	CreatedAt             string
	UpdatedAt             string
}

func (d *DB) UpsertIncidentSignature(sig IncidentSignatureRecord) error {
	if sig.SignatureKey == "" {
		return fmt.Errorf("signature key is required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if sig.FirstSeenAt == "" {
		sig.FirstSeenAt = now
	}
	if sig.LastSeenAt == "" {
		sig.LastSeenAt = now
	}
	sql := fmt.Sprintf(`INSERT INTO incident_signatures(signature_key,signature_label,category,resource_type,reason_key,first_seen_at,last_seen_at,match_count,example_incident_id,last_summary)
		VALUES('%s','%s','%s','%s','%s','%s','%s',1,'%s','%s')
		ON CONFLICT(signature_key) DO UPDATE SET
			signature_label=excluded.signature_label,
			last_seen_at=excluded.last_seen_at,
			match_count=incident_signatures.match_count + 1,
			example_incident_id=excluded.example_incident_id,
			last_summary=excluded.last_summary;`,
		esc(sig.SignatureKey), esc(sig.SignatureLabel), esc(sig.Category), esc(sig.ResourceType), esc(sig.ReasonKey),
		esc(sig.FirstSeenAt), esc(sig.LastSeenAt), esc(sig.ExampleIncidentID), esc(sig.LastSummary))
	return d.Exec(sql)
}

func (d *DB) LinkIncidentToSignature(signatureKey, incidentID string) error {
	if signatureKey == "" || incidentID == "" {
		return fmt.Errorf("signature key and incident id are required")
	}
	sql := fmt.Sprintf(`INSERT OR IGNORE INTO incident_signature_incidents(signature_key,incident_id,linked_at)
		VALUES('%s','%s','%s');`,
		esc(signatureKey), esc(incidentID), esc(time.Now().UTC().Format(time.RFC3339)))
	if err := d.Exec(sql); err != nil {
		return err
	}
	_ = d.EnsureSignatureCorrelationGroup(signatureKey)
	return nil
}

func (d *DB) SignatureByKey(signatureKey string) (IncidentSignatureRecord, bool, error) {
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT signature_key,signature_label,category,resource_type,reason_key,first_seen_at,last_seen_at,match_count,example_incident_id,last_summary
		FROM incident_signatures WHERE signature_key='%s' LIMIT 1;`, esc(signatureKey)))
	if err != nil {
		return IncidentSignatureRecord{}, false, err
	}
	if len(rows) == 0 {
		return IncidentSignatureRecord{}, false, nil
	}
	row := rows[0]
	return IncidentSignatureRecord{
		SignatureKey:      asString(row["signature_key"]),
		SignatureLabel:    asString(row["signature_label"]),
		Category:          asString(row["category"]),
		ResourceType:      asString(row["resource_type"]),
		ReasonKey:         asString(row["reason_key"]),
		FirstSeenAt:       asString(row["first_seen_at"]),
		LastSeenAt:        asString(row["last_seen_at"]),
		MatchCount:        int(asInt(row["match_count"])),
		ExampleIncidentID: asString(row["example_incident_id"]),
		LastSummary:       asString(row["last_summary"]),
	}, true, nil
}

// MaxSignatureFamilyPeerScan caps per-signature peer rows considered for resolved/reopened tallies (latency bound).
// Exact family size uses a separate COUNT; when truncated, resolved/reopened apply only to the scan window.
const MaxSignatureFamilyPeerScan = 5000

// SignatureFamilyResolvedStats scans other incidents linked to the same signature (excludes current id).
// peerTotal is an exact COUNT(*); resolved/reopened may be window-limited when peerTotal > maxScan (see truncated return).
func (d *DB) SignatureFamilyResolvedStats(signatureKey, excludeIncidentID string, maxScan int) (peerTotal, resolvedPeers, reopenedPeers int, samplePeerID string, truncated bool, err error) {
	signatureKey = strings.TrimSpace(signatureKey)
	excludeIncidentID = strings.TrimSpace(excludeIncidentID)
	if signatureKey == "" || d == nil {
		return 0, 0, 0, "", false, nil
	}
	if maxScan <= 0 {
		maxScan = MaxSignatureFamilyPeerScan
	}
	cntRows, err := d.QueryRows(fmt.Sprintf(`SELECT COUNT(*) AS c FROM incident_signature_incidents s
		WHERE s.signature_key='%s' AND s.incident_id!='%s';`, esc(signatureKey), esc(excludeIncidentID)))
	if err != nil {
		return 0, 0, 0, "", false, err
	}
	if len(cntRows) > 0 {
		peerTotal = int(asInt(cntRows[0]["c"]))
	}
	if peerTotal == 0 {
		return 0, 0, 0, "", false, nil
	}
	truncated = peerTotal > maxScan
	limit := peerTotal
	if truncated {
		limit = maxScan
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT i.id, i.state, COALESCE(i.reopened_from_incident_id,'') AS reopened_from
		FROM incident_signature_incidents s
		JOIN incidents i ON i.id = s.incident_id
		WHERE s.signature_key='%s' AND i.id!='%s'
		ORDER BY s.linked_at DESC
		LIMIT %d;`, esc(signatureKey), esc(excludeIncidentID), limit))
	if err != nil {
		return peerTotal, 0, 0, "", truncated, err
	}
	for _, row := range rows {
		st := strings.ToLower(strings.TrimSpace(asString(row["state"])))
		if st == "resolved" || st == "closed" {
			resolvedPeers++
		}
		if strings.TrimSpace(asString(row["reopened_from"])) != "" {
			reopenedPeers++
		}
		if samplePeerID == "" {
			samplePeerID = asString(row["id"])
		}
	}
	return peerTotal, resolvedPeers, reopenedPeers, samplePeerID, truncated, nil
}

func (d *DB) SimilarIncidentsBySignature(signatureKey, excludeIncidentID string, limit int) ([]models.Incident, error) {
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT i.id, i.category, i.severity, i.title, i.summary, i.resource_type, i.resource_id, i.state,
		COALESCE(i.actor_id,'') AS actor_id, i.occurred_at, i.updated_at, COALESCE(i.resolved_at,'') AS resolved_at, COALESCE(i.metadata_json,'{}') AS metadata_json,
		COALESCE(i.owner_actor_id,'') AS owner_actor_id, COALESCE(i.handoff_summary,'') AS handoff_summary,
		COALESCE(i.pending_actions_json,'[]') AS pending_actions_json, COALESCE(i.recent_actions_json,'[]') AS recent_actions_json,
		COALESCE(i.linked_evidence_json,'[]') AS linked_evidence_json, COALESCE(i.risks_json,'[]') AS risks_json
		FROM incident_signature_incidents s
		JOIN incidents i ON i.id = s.incident_id
		WHERE s.signature_key='%s' AND i.id!='%s'
		ORDER BY i.occurred_at DESC
		LIMIT %d;`, esc(signatureKey), esc(excludeIncidentID), limit))
	if err != nil {
		return nil, err
	}
	out := make([]models.Incident, 0, len(rows))
	for _, row := range rows {
		out = append(out, incidentFromRow(row))
	}
	return out, nil
}

func (d *DB) DeadLettersForTransportBetween(transportName, fromInclusive, toInclusive string, limit int) ([]map[string]any, error) {
	if transportName == "" {
		return nil, nil
	}
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id, reason, created_at FROM dead_letters
		WHERE transport_name='%s' AND created_at>='%s' AND created_at<='%s'
		ORDER BY created_at DESC LIMIT %d;`, esc(transportName), esc(fromInclusive), esc(toInclusive), limit))
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (d *DB) TransportAlertsForWindow(transportName, fromInclusive, toInclusive string, limit int) ([]TransportAlertRecord, error) {
	if transportName == "" {
		return nil, nil
	}
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id, transport_name, transport_type, severity, reason, summary, first_triggered_at, last_updated_at,
		COALESCE(resolved_at,'') AS resolved_at, active, COALESCE(episode_id,'') AS episode_id, COALESCE(cluster_key,'') AS cluster_key,
		COALESCE(contributing_reasons_json,'[]') AS contributing_reasons_json, COALESCE(penalty_snapshot_json,'[]') AS penalty_snapshot_json,
		COALESCE(trigger_condition,'') AS trigger_condition
		FROM transport_alerts
		WHERE transport_name='%s' AND last_updated_at>='%s' AND last_updated_at<='%s'
		ORDER BY last_updated_at DESC LIMIT %d;`, esc(transportName), esc(fromInclusive), esc(toInclusive), limit))
	if err != nil {
		return nil, err
	}
	out := make([]TransportAlertRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, alertRecordFromRow(row))
	}
	return out, nil
}

func (d *DB) UpsertIncidentActionOutcomeSnapshot(snapshot IncidentActionOutcomeSnapshotRecord) error {
	if strings.TrimSpace(snapshot.SignatureKey) == "" {
		return fmt.Errorf("signature key is required")
	}
	if strings.TrimSpace(snapshot.IncidentID) == "" {
		return fmt.Errorf("incident id is required")
	}
	if strings.TrimSpace(snapshot.ActionID) == "" {
		return fmt.Errorf("action id is required")
	}
	if strings.TrimSpace(snapshot.ActionType) == "" {
		return fmt.Errorf("action type is required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if strings.TrimSpace(snapshot.SnapshotID) == "" {
		sum := sha1.Sum([]byte(snapshot.SignatureKey + "|" + snapshot.IncidentID + "|" + snapshot.ActionID))
		snapshot.SnapshotID = fmt.Sprintf("aos-%x", sum[:8])
	}
	if strings.TrimSpace(snapshot.DerivedClassification) == "" {
		snapshot.DerivedClassification = "inconclusive"
	}
	if strings.TrimSpace(snapshot.EvidenceSufficiency) == "" {
		snapshot.EvidenceSufficiency = "insufficient"
	}
	if strings.TrimSpace(snapshot.DerivationVersion) == "" {
		snapshot.DerivationVersion = "incident_action_outcome_eval/v1"
	}
	if strings.TrimSpace(snapshot.SchemaVersion) == "" {
		snapshot.SchemaVersion = "1.0.0"
	}
	if strings.TrimSpace(snapshot.DerivedAt) == "" {
		snapshot.DerivedAt = now
	}
	if strings.TrimSpace(snapshot.CreatedAt) == "" {
		snapshot.CreatedAt = now
	}
	snapshot.UpdatedAt = now
	if strings.TrimSpace(snapshot.WindowStart) == "" {
		snapshot.WindowStart = snapshot.DerivedAt
	}
	if strings.TrimSpace(snapshot.WindowEnd) == "" {
		snapshot.WindowEnd = snapshot.DerivedAt
	}
	preJSON, _ := json.Marshal(firstNonNilMap(snapshot.PreActionSummary))
	postJSON, _ := json.Marshal(firstNonNilMap(snapshot.PostActionSummary))
	caveatsJSON, _ := json.Marshal(snapshot.Caveats)
	inspectJSON, _ := json.Marshal(snapshot.InspectBeforeReuse)
	evidenceRefsJSON, _ := json.Marshal(snapshot.EvidenceRefs)

	sql := fmt.Sprintf(`INSERT INTO incident_action_outcome_snapshots(
		snapshot_id,signature_key,incident_id,action_id,action_type,action_label,derived_classification,evidence_sufficiency,
		pre_action_summary_json,post_action_summary_json,observed_signal_count,caveats_json,inspect_before_reuse_json,evidence_refs_json,association_only,
		derivation_window_start,derivation_window_end,derivation_version,schema_version,derived_at,created_at,updated_at
	) VALUES (
		'%s','%s','%s','%s','%s','%s','%s','%s',
		'%s','%s',%d,'%s','%s','%s',%d,
		'%s','%s','%s','%s','%s','%s','%s'
	)
	ON CONFLICT(signature_key,incident_id,action_id) DO UPDATE SET
		snapshot_id=excluded.snapshot_id,
		action_type=excluded.action_type,
		action_label=excluded.action_label,
		derived_classification=excluded.derived_classification,
		evidence_sufficiency=excluded.evidence_sufficiency,
		pre_action_summary_json=excluded.pre_action_summary_json,
		post_action_summary_json=excluded.post_action_summary_json,
		observed_signal_count=excluded.observed_signal_count,
		caveats_json=excluded.caveats_json,
		inspect_before_reuse_json=excluded.inspect_before_reuse_json,
		evidence_refs_json=excluded.evidence_refs_json,
		association_only=excluded.association_only,
		derivation_window_start=excluded.derivation_window_start,
		derivation_window_end=excluded.derivation_window_end,
		derivation_version=excluded.derivation_version,
		schema_version=excluded.schema_version,
		derived_at=excluded.derived_at,
		updated_at=excluded.updated_at;`,
		esc(snapshot.SnapshotID), esc(snapshot.SignatureKey), esc(snapshot.IncidentID), esc(snapshot.ActionID), esc(snapshot.ActionType), esc(snapshot.ActionLabel),
		esc(snapshot.DerivedClassification), esc(snapshot.EvidenceSufficiency), esc(string(preJSON)), esc(string(postJSON)), snapshot.ObservedSignalCount,
		esc(string(caveatsJSON)), esc(string(inspectJSON)), esc(string(evidenceRefsJSON)), boolInt(snapshot.AssociationOnly),
		esc(snapshot.WindowStart), esc(snapshot.WindowEnd), esc(snapshot.DerivationVersion), esc(snapshot.SchemaVersion), esc(snapshot.DerivedAt), esc(snapshot.CreatedAt), esc(snapshot.UpdatedAt))
	return d.Exec(sql)
}

func (d *DB) ActionOutcomeSnapshotsBySignature(signatureKey, excludeIncidentID string, limit int) ([]IncidentActionOutcomeSnapshotRecord, error) {
	if strings.TrimSpace(signatureKey) == "" {
		return nil, nil
	}
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT snapshot_id, signature_key, incident_id, action_id, action_type, COALESCE(action_label,'') AS action_label,
		derived_classification, evidence_sufficiency, COALESCE(pre_action_summary_json,'{}') AS pre_action_summary_json,
		COALESCE(post_action_summary_json,'{}') AS post_action_summary_json, COALESCE(observed_signal_count,0) AS observed_signal_count,
		COALESCE(caveats_json,'[]') AS caveats_json, COALESCE(inspect_before_reuse_json,'[]') AS inspect_before_reuse_json,
		COALESCE(evidence_refs_json,'[]') AS evidence_refs_json, COALESCE(association_only,1) AS association_only,
		derivation_window_start, derivation_window_end, COALESCE(derivation_version,'incident_action_outcome_eval/v1') AS derivation_version,
		COALESCE(schema_version,'1.0.0') AS schema_version, derived_at, created_at, updated_at
		FROM incident_action_outcome_snapshots
		WHERE signature_key='%s' AND incident_id!='%s'
		ORDER BY derived_at DESC
		LIMIT %d;`, esc(signatureKey), esc(excludeIncidentID), limit))
	if err != nil {
		return nil, err
	}
	out := make([]IncidentActionOutcomeSnapshotRecord, 0, len(rows))
	for _, row := range rows {
		record := IncidentActionOutcomeSnapshotRecord{
			SnapshotID:            asString(row["snapshot_id"]),
			SignatureKey:          asString(row["signature_key"]),
			IncidentID:            asString(row["incident_id"]),
			ActionID:              asString(row["action_id"]),
			ActionType:            asString(row["action_type"]),
			ActionLabel:           asString(row["action_label"]),
			DerivedClassification: asString(row["derived_classification"]),
			EvidenceSufficiency:   asString(row["evidence_sufficiency"]),
			ObservedSignalCount:   int(asInt(row["observed_signal_count"])),
			AssociationOnly:       asInt(row["association_only"]) != 0,
			WindowStart:           asString(row["derivation_window_start"]),
			WindowEnd:             asString(row["derivation_window_end"]),
			DerivationVersion:     asString(row["derivation_version"]),
			SchemaVersion:         asString(row["schema_version"]),
			DerivedAt:             asString(row["derived_at"]),
			CreatedAt:             asString(row["created_at"]),
			UpdatedAt:             asString(row["updated_at"]),
		}
		_ = json.Unmarshal([]byte(asString(row["pre_action_summary_json"])), &record.PreActionSummary)
		_ = json.Unmarshal([]byte(asString(row["post_action_summary_json"])), &record.PostActionSummary)
		_ = json.Unmarshal([]byte(asString(row["caveats_json"])), &record.Caveats)
		_ = json.Unmarshal([]byte(asString(row["inspect_before_reuse_json"])), &record.InspectBeforeReuse)
		_ = json.Unmarshal([]byte(asString(row["evidence_refs_json"])), &record.EvidenceRefs)
		out = append(out, record)
	}
	return out, nil
}

func firstNonNilMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	return in
}
