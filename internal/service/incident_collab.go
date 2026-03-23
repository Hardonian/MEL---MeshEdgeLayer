package service

// incident_collab.go — Durable incident ownership and operator handoff.

import (
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/auth"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/models"
)

// IncidentHandoff assigns ownership and records a concise handoff for the next operator.
func (a *App) IncidentHandoff(incidentID, fromActorID string, req models.IncidentHandoffRequest) error {
	if a == nil || a.DB == nil {
		return fmt.Errorf("service not available")
	}
	incidentID = strings.TrimSpace(incidentID)
	if incidentID == "" {
		return fmt.Errorf("incident id is required")
	}
	to := strings.TrimSpace(req.ToOperatorID)
	if to == "" {
		return fmt.Errorf("to_operator_id is required")
	}
	summary := strings.TrimSpace(req.HandoffSummary)
	if summary == "" {
		return fmt.Errorf("handoff_summary is required")
	}
	if strings.TrimSpace(fromActorID) == "" {
		fromActorID = "system"
	}

	inc, ok, err := a.DB.IncidentByID(incidentID)
	if err != nil {
		return fmt.Errorf("could not load incident: %w", err)
	}
	if !ok {
		return fmt.Errorf("incident not found: %s", incidentID)
	}

	if len(req.PendingActions) > 0 {
		inc.PendingActions = append([]string(nil), req.PendingActions...)
	}
	if len(req.RecentActions) > 0 {
		inc.RecentActions = append([]string(nil), req.RecentActions...)
	}
	if len(req.LinkedEvidence) > 0 {
		inc.LinkedEvidence = append([]map[string]any(nil), req.LinkedEvidence...)
	}
	if len(req.Risks) > 0 {
		inc.Risks = append([]string(nil), req.Risks...)
	}
	inc.OwnerActorID = to
	inc.HandoffSummary = summary
	if inc.Metadata == nil {
		inc.Metadata = map[string]any{}
	}
	inc.Metadata["last_handoff_from"] = fromActorID
	inc.Metadata["last_handoff_at"] = time.Now().UTC().Format(time.RFC3339)

	if err := a.DB.UpsertIncident(inc); err != nil {
		return fmt.Errorf("could not persist handoff: %w", err)
	}

	_ = a.DB.InsertRBACAuditLog(auth.AuditEntry{
		ID:           newTrustID("aud"),
		ActorID:      auth.OperatorID(fromActorID),
		ActionClass:  auth.ActionControl,
		ActionDetail: "incident_handoff",
		ResourceType: "incident",
		ResourceID:   incidentID,
		Reason:       summary,
		Result:       auth.AuditResultSuccess,
		Timestamp:    time.Now().UTC(),
	})
	_ = a.DB.InsertTimelineEvent(db.TimelineEvent{
		EventID:    newTrustID("tl"),
		EventType:  "incident_handoff",
		Summary:    "incident handed off: " + incidentID + " → " + to,
		Severity:   "info",
		ActorID:    fromActorID,
		ResourceID: incidentID,
		Details: map[string]any{
			"incident_id": incidentID,
			"to":          to,
			"summary":     summary,
		},
	})
	return nil
}

// IncidentByID returns a full incident row for API/CLI, including canonical linked control actions.
func (a *App) IncidentByID(id string) (models.Incident, bool, error) {
	if a == nil || a.DB == nil {
		return models.Incident{}, false, fmt.Errorf("service not available")
	}
	inc, ok, err := a.DB.IncidentByID(strings.TrimSpace(id))
	if err != nil || !ok {
		return inc, ok, err
	}
	linked, err := a.DB.ControlActionsByIncidentID(inc.ID, 100)
	if err != nil {
		return models.Incident{}, false, fmt.Errorf("could not load linked actions: %w", err)
	}
	if len(linked) > 0 {
		inc.LinkedControlActions = make([]models.ActionRecord, 0, len(linked))
		for _, r := range linked {
			inc.LinkedControlActions = append(inc.LinkedControlActions, ActionRecordFromDB(r))
		}
	}
	return inc, true, nil
}

// RecentIncidentsWithLinkedActions returns recent incidents with linked_control_actions populated from the canonical FK.
func (a *App) RecentIncidentsWithLinkedActions(limit int) ([]models.Incident, error) {
	if a == nil || a.DB == nil {
		return nil, fmt.Errorf("service not available")
	}
	incs, err := a.DB.RecentIncidents(limit)
	if err != nil {
		return nil, err
	}
	if len(incs) == 0 {
		return incs, nil
	}
	ids := make([]string, 0, len(incs))
	for _, inc := range incs {
		ids = append(ids, inc.ID)
	}
	actions, err := a.DB.ControlActionsForIncidentIDs(ids, 500)
	if err != nil {
		return nil, err
	}
	byInc := make(map[string][]models.ActionRecord)
	for _, r := range actions {
		if strings.TrimSpace(r.IncidentID) == "" {
			continue
		}
		byInc[r.IncidentID] = append(byInc[r.IncidentID], ActionRecordFromDB(r))
	}
	for i := range incs {
		if linked, ok := byInc[incs[i].ID]; ok && len(linked) > 0 {
			incs[i].LinkedControlActions = linked
		}
	}
	return incs, nil
}
