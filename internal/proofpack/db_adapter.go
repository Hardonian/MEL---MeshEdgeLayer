package proofpack

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/models"
)

// DBAdapter implements DataSource using the MEL database layer.
type DBAdapter struct {
	DB *db.DB
}

// NewDBAdapter creates a DataSource backed by the MEL database.
func NewDBAdapter(database *db.DB) *DBAdapter {
	return &DBAdapter{DB: database}
}

func (d *DBAdapter) IncidentByID(id string) (models.Incident, bool, error) {
	return d.DB.IncidentByID(id)
}

func (d *DBAdapter) SignatureKeyForIncident(incidentID string) (string, error) {
	rows, err := d.DB.QueryRows(fmt.Sprintf(
		`SELECT signature_key FROM incident_signature_incidents WHERE incident_id='%s' ORDER BY linked_at DESC LIMIT 1;`,
		db.EscString(incidentID)))
	if err != nil || len(rows) == 0 {
		return "", err
	}
	return strings.TrimSpace(asString(rows[0]["signature_key"])), nil
}

func (d *DBAdapter) ControlActionsByIncidentID(incidentID string, limit int) ([]ActionEvidence, error) {
	rows, err := d.DB.ControlActionsByIncidentID(incidentID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]ActionEvidence, 0, len(rows))
	for _, r := range rows {
		var basis []string
		if len(r.ApprovalBasis) > 0 {
			basis = append([]string(nil), r.ApprovalBasis...)
		}
		out = append(out, ActionEvidence{
			ID:               r.ID,
			ActionType:       r.ActionType,
			TransportName:    r.TargetTransport,
			TargetNode:       r.TargetNode,
			TargetSegment:    r.TargetSegment,
			LifecycleState:   r.LifecycleState,
			Result:           r.Result,
			Reason:           r.Reason,
			OutcomeDetail:    r.OutcomeDetail,
			ExecutionMode:    r.ExecutionMode,
			BlastRadiusClass: r.BlastRadiusClass,
			HighBlastRadius:  r.HighBlastRadius,
			ProposedBy:       r.ProposedBy,
			SubmittedBy:      r.SubmittedBy,
			ApprovedBy:       r.ApprovedBy,
			ApprovedAt:       r.ApprovedAt,
			RejectedBy:       r.RejectedBy,
			RejectedAt:       r.RejectedAt,
			CreatedAt:        r.CreatedAt,
			ExecutedAt:       r.ExecutedAt,
			CompletedAt:      r.CompletedAt,
			IncidentID:       r.IncidentID,
			SodBypass:        r.SodBypass,
			SodBypassReason:  r.SodBypassReason,
			ApprovalBasis:    basis,
			ExecutionSource:  r.ExecutionSource,
		})
	}
	return out, nil
}

func (d *DBAdapter) ActionOutcomeSnapshotsBySignature(signatureKey, excludeIncidentID string, limit int) ([]ActionOutcomeSnapshot, error) {
	rows, err := d.DB.ActionOutcomeSnapshotsBySignature(signatureKey, excludeIncidentID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]ActionOutcomeSnapshot, 0, len(rows))
	for _, row := range rows {
		out = append(out, ActionOutcomeSnapshot{
			SnapshotID:            row.SnapshotID,
			SignatureKey:          row.SignatureKey,
			IncidentID:            row.IncidentID,
			ActionID:              row.ActionID,
			ActionType:            row.ActionType,
			ActionLabel:           row.ActionLabel,
			DerivedClassification: row.DerivedClassification,
			EvidenceSufficiency:   row.EvidenceSufficiency,
			WindowStart:           row.WindowStart,
			WindowEnd:             row.WindowEnd,
			PreActionEvidence: ActionOutcomeEvidenceSummary{
				TransportName:        asMapString(row.PreActionSummary, "transport_name"),
				DeadLettersCount:     asMapInt(row.PreActionSummary, "dead_letters_count"),
				TransportAlertsCount: asMapInt(row.PreActionSummary, "transport_alerts_count"),
				IncidentState:        asMapString(row.PreActionSummary, "incident_state"),
				ActionResult:         asMapString(row.PreActionSummary, "action_result"),
				ActionLifecycle:      asMapString(row.PreActionSummary, "action_lifecycle"),
			},
			PostActionEvidence: ActionOutcomeEvidenceSummary{
				TransportName:        asMapString(row.PostActionSummary, "transport_name"),
				DeadLettersCount:     asMapInt(row.PostActionSummary, "dead_letters_count"),
				TransportAlertsCount: asMapInt(row.PostActionSummary, "transport_alerts_count"),
				IncidentState:        asMapString(row.PostActionSummary, "incident_state"),
				ActionResult:         asMapString(row.PostActionSummary, "action_result"),
				ActionLifecycle:      asMapString(row.PostActionSummary, "action_lifecycle"),
			},
			ObservedSignalCount: row.ObservedSignalCount,
			Caveats:             append([]string(nil), row.Caveats...),
			InspectBeforeReuse:  append([]string(nil), row.InspectBeforeReuse...),
			EvidenceRefs:        append([]string(nil), row.EvidenceRefs...),
			AssociationOnly:     row.AssociationOnly,
			DerivationVersion:   row.DerivationVersion,
			SchemaVersion:       row.SchemaVersion,
			DerivedAt:           row.DerivedAt,
		})
	}
	return out, nil
}

func (d *DBAdapter) TimelineEventsForIncident(incidentID, from, to string, limit int) ([]TimelineEntry, error) {
	// Get timeline events in the window that reference this incident.
	events, err := d.DB.TimelineEvents(from, to, limit)
	if err != nil {
		return nil, err
	}
	out := make([]TimelineEntry, 0)
	for _, ev := range events {
		// Include events that reference this incident directly, or that
		// reference a linked action, or that are general system events
		// in the time window.
		if ev.ResourceID == incidentID || ev.EventType == "incident" || ev.EventType == "incident_handoff" ||
			isSystemEvent(ev.EventType) {
			out = append(out, timelineEntryFromDB(ev))
		}
	}
	return out, nil
}

func (d *DBAdapter) TransportHealthSnapshotsInWindow(from, to string, limit int) ([]TransportSnapshot, error) {
	snapshots, err := d.DB.TransportHealthSnapshots("", from, to, limit, 0)
	if err != nil {
		return nil, err
	}
	out := make([]TransportSnapshot, 0, len(snapshots))
	for _, s := range snapshots {
		out = append(out, TransportSnapshot{
			TransportName:              s.TransportName,
			TransportType:              s.TransportType,
			Score:                      s.Score,
			State:                      s.State,
			SnapshotTime:               s.SnapshotTime,
			ActiveAlertCount:           s.ActiveAlertCount,
			DeadLetterCountWindow:      s.DeadLetterCountWindow,
			ObservationDropCountWindow: s.ObservationDropCountWindow,
		})
	}
	return out, nil
}

func (d *DBAdapter) DeadLettersInWindow(from, to string, limit int) ([]DeadLetterEntry, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := d.DB.QueryRows(fmt.Sprintf(
		`SELECT transport_name, COALESCE(transport_type,'') AS transport_type, COALESCE(topic,'') AS topic, reason, COALESCE(details_json,'{}') AS details_json, created_at FROM dead_letters WHERE created_at >= '%s' AND created_at <= '%s' ORDER BY created_at DESC LIMIT %d;`,
		db.EscString(from), db.EscString(to), limit))
	if err != nil {
		return nil, err
	}
	out := make([]DeadLetterEntry, 0, len(rows))
	for _, row := range rows {
		details := map[string]interface{}{}
		if dj := strings.TrimSpace(asString(row["details_json"])); dj != "" && dj != "{}" {
			_ = json.Unmarshal([]byte(dj), &details)
		}
		out = append(out, DeadLetterEntry{
			TransportName: asString(row["transport_name"]),
			TransportType: asString(row["transport_type"]),
			Topic:         asString(row["topic"]),
			Reason:        asString(row["reason"]),
			CreatedAt:     asString(row["created_at"]),
			Details:       details,
		})
	}
	return out, nil
}

func (d *DBAdapter) OperatorNotesForResource(refType, refID string, limit int) ([]OperatorNote, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := d.DB.QueryRows(fmt.Sprintf(
		`SELECT id, actor_id, content, created_at FROM operator_notes WHERE ref_type='%s' AND ref_id='%s' ORDER BY created_at DESC LIMIT %d;`,
		db.EscString(refType), db.EscString(refID), limit))
	if err != nil {
		return nil, err
	}
	out := make([]OperatorNote, 0, len(rows))
	for _, row := range rows {
		out = append(out, OperatorNote{
			ID:        asString(row["id"]),
			ActorID:   asString(row["actor_id"]),
			Content:   asString(row["content"]),
			CreatedAt: asString(row["created_at"]),
		})
	}
	return out, nil
}

func (d *DBAdapter) RecommendationOutcomesForIncident(incidentID string, limit int) ([]RecommendationOutcomeEntry, error) {
	rows, err := d.DB.RecommendationOutcomesForIncident(incidentID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]RecommendationOutcomeEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, RecommendationOutcomeEntry{
			ID:               r.ID,
			RecommendationID: r.RecommendationID,
			Outcome:          r.Outcome,
			ActorID:          r.ActorID,
			Note:             r.Note,
			CreatedAt:        r.CreatedAt,
		})
	}
	return out, nil
}

func (d *DBAdapter) CorrelationGroupsForIncident(incidentID string) ([]CorrelationGroupEntry, error) {
	groups, err := d.DB.CorrelationGroupsForIncident(incidentID)
	if err != nil {
		return nil, err
	}
	out := make([]CorrelationGroupEntry, 0, len(groups))
	for _, g := range groups {
		ids, _ := d.DB.CorrelatedIncidentIDsForGroup(g.GroupID)
		others := make([]string, 0, len(ids))
		for _, id := range ids {
			if id != incidentID {
				others = append(others, id)
			}
		}
		sort.Strings(others)
		out = append(out, CorrelationGroupEntry{
			GroupID:          g.GroupID,
			CorrelationKey:   g.CorrelationKey,
			Basis:            g.Basis,
			UncertaintyNote:  g.UncertaintyNote,
			Rationale:        append([]string(nil), g.Rationale...),
			EvidenceRefs:     append([]string(nil), g.EvidenceRefs...),
			MemberCount:      len(ids),
			OtherIncidentIDs: others,
		})
	}
	return out, nil
}

func (d *DBAdapter) AuditEntriesForResource(resourceType, resourceID string, limit int) ([]AuditEntry, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := d.DB.QueryRows(fmt.Sprintf(
		`SELECT id, timestamp, actor_id, action_class, action_detail, resource_type, resource_id, result FROM audit_log WHERE resource_type='%s' AND resource_id='%s' ORDER BY timestamp DESC LIMIT %d;`,
		db.EscString(resourceType), db.EscString(resourceID), limit))
	if err != nil {
		return nil, err
	}
	out := make([]AuditEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, AuditEntry{
			ID:           asString(row["id"]),
			Timestamp:    asString(row["timestamp"]),
			ActorID:      asString(row["actor_id"]),
			ActionClass:  asString(row["action_class"]),
			ActionDetail: asString(row["action_detail"]),
			ResourceType: asString(row["resource_type"]),
			ResourceID:   asString(row["resource_id"]),
			Result:       asString(row["result"]),
		})
	}
	return out, nil
}

func timelineEntryFromDB(ev db.TimelineEvent) TimelineEntry {
	return TimelineEntry{
		EventTime:        ev.EventTime,
		EventType:        ev.EventType,
		EventID:          ev.EventID,
		Summary:          ev.Summary,
		Severity:         ev.Severity,
		ActorID:          ev.ActorID,
		ResourceID:       ev.ResourceID,
		Details:          ev.Details,
		ScopePosture:     ev.ScopePosture,
		TimingPosture:    ev.TimingPosture,
		MergeDisposition: ev.MergeDisposition,
	}
}

func isSystemEvent(eventType string) bool {
	switch eventType {
	case "freeze_created", "freeze_cleared", "maintenance", "control_action":
		return true
	}
	return false
}

func asString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func asMapString(in map[string]any, k string) string {
	if in == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(in[k]))
}

func asMapInt(in map[string]any, k string) int {
	if in == nil {
		return 0
	}
	switch v := in[k].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case string:
		var out int
		_, _ = fmt.Sscan(strings.TrimSpace(v), &out)
		return out
	default:
		var out int
		_, _ = fmt.Sscan(fmt.Sprint(v), &out)
		return out
	}
}
