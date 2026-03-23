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

// IncidentByID returns a full incident row for API/CLI.
func (a *App) IncidentByID(id string) (models.Incident, bool, error) {
	if a == nil || a.DB == nil {
		return models.Incident{}, false, fmt.Errorf("service not available")
	}
	return a.DB.IncidentByID(strings.TrimSpace(id))
}
