package db

import (
	"fmt"
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
	return d.Exec(sql)
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
