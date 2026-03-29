package proofpack

import (
	"encoding/json"
	"fmt"
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
