package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/auth"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/incidentdecisionpack"
	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/operatorreadiness"
)

func (a *App) attachDecisionPack(inc *models.Incident) {
	if inc == nil || a == nil || a.DB == nil {
		return
	}
	readiness := operatorreadiness.FromConfig(a.Cfg)
	adj := a.loadPackAdjudication(inc.ID)
	pack := incidentdecisionpack.Build(*inc, adj, readiness, time.Now().UTC())
	inc.DecisionPack = &pack
}

func (a *App) loadPackAdjudication(incidentID string) *models.IncidentDecisionPackAdjudication {
	rec, ok, err := a.DB.GetIncidentDecisionPackAdjudication(incidentID)
	if err != nil || !ok {
		return nil
	}
	cues, err := db.DecodeCueOutcomesJSON(rec.CueOutcomesJSON)
	if err != nil {
		cues = nil
	}
	return &models.IncidentDecisionPackAdjudication{
		Reviewed:          rec.Reviewed,
		ReviewedAt:        rec.ReviewedAt,
		ReviewedByActorID: rec.ReviewedByActorID,
		Useful:            rec.Useful,
		OperatorNote:      rec.OperatorNote,
		CueOutcomes:       cues,
		UpdatedAt:         rec.UpdatedAt,
	}
}

// PatchIncidentDecisionPackAdjudication persists minimal operator feedback on the decision pack.
func (a *App) PatchIncidentDecisionPackAdjudication(incidentID, actorID string, patch models.IncidentDecisionPackAdjudicationPatch) error {
	if a == nil || a.DB == nil {
		return fmt.Errorf("service not available")
	}
	incidentID = strings.TrimSpace(incidentID)
	if incidentID == "" {
		return fmt.Errorf("incident id is required")
	}
	_, ok, err := a.DB.IncidentByID(incidentID)
	if err != nil {
		return fmt.Errorf("could not load incident: %w", err)
	}
	if !ok {
		return fmt.Errorf("incident not found: %s", incidentID)
	}
	if strings.TrimSpace(actorID) == "" {
		actorID = "system"
	}

	prev, hadPrev, _ := a.DB.GetIncidentDecisionPackAdjudication(incidentID)
	rec := db.IncidentDecisionPackAdjudicationRecord{
		IncidentID:        incidentID,
		Reviewed:          prev.Reviewed,
		ReviewedAt:        prev.ReviewedAt,
		ReviewedByActorID: prev.ReviewedByActorID,
		Useful:            prev.Useful,
		OperatorNote:      prev.OperatorNote,
		CueOutcomesJSON:   prev.CueOutcomesJSON,
	}
	if !hadPrev {
		rec.CueOutcomesJSON = "[]"
	}

	if patch.Reviewed != nil {
		rec.Reviewed = *patch.Reviewed
		if rec.Reviewed && (!prev.Reviewed || !hadPrev) {
			rec.ReviewedAt = time.Now().UTC().Format(time.RFC3339)
			rec.ReviewedByActorID = actorID
		}
	}
	if patch.Useful != nil {
		u := strings.TrimSpace(*patch.Useful)
		if u != "" && u != "useful" && u != "not_useful" {
			return fmt.Errorf("invalid useful value")
		}
		rec.Useful = u
	}
	if patch.OperatorNote != nil {
		rec.OperatorNote = strings.TrimSpace(*patch.OperatorNote)
	}

	existingCues, _ := db.DecodeCueOutcomesJSON(rec.CueOutcomesJSON)
	if patch.ReplaceCueOutcomes {
		existingCues = nil
	}
	byID := map[string]models.IncidentDecisionPackCueOutcome{}
	for _, c := range existingCues {
		id := strings.TrimSpace(c.CueID)
		if id == "" {
			continue
		}
		byID[id] = c
	}
	for _, c := range patch.CueOutcomes {
		id := strings.TrimSpace(c.CueID)
		if id == "" {
			continue
		}
		o := strings.TrimSpace(c.Outcome)
		if o != "" && o != "accepted" && o != "dismissed" {
			return fmt.Errorf("invalid cue outcome for %q", id)
		}
		byID[id] = models.IncidentDecisionPackCueOutcome{CueID: id, Outcome: o, Note: strings.TrimSpace(c.Note)}
	}
	var merged []models.IncidentDecisionPackCueOutcome
	for _, c := range byID {
		merged = append(merged, c)
	}
	b, err := json.Marshal(merged)
	if err != nil {
		return err
	}
	rec.CueOutcomesJSON = string(b)
	rec.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := a.DB.UpsertIncidentDecisionPackAdjudication(rec); err != nil {
		return fmt.Errorf("could not persist adjudication: %w", err)
	}
	_ = a.DB.InsertRBACAuditLog(auth.AuditEntry{
		ID:           newTrustID("aud"),
		ActorID:      auth.OperatorID(actorID),
		ActionClass:  auth.ActionControl,
		ActionDetail: "incident_decision_pack_adjudication",
		ResourceType: "incident",
		ResourceID:   incidentID,
		Reason:       "decision pack operator adjudication updated",
		Result:       auth.AuditResultSuccess,
		Timestamp:    time.Now().UTC(),
	})
	_ = a.DB.InsertTimelineEvent(db.TimelineEvent{
		EventID:    newTrustID("tl"),
		EventType:  "operator_adjudication",
		Summary:    "incident decision pack adjudication updated: " + incidentID,
		Severity:   "info",
		ActorID:    actorID,
		ResourceID: incidentID,
		Details: map[string]any{
			"incident_id": incidentID,
			"reviewed":    rec.Reviewed,
		},
	})
	return nil
}
