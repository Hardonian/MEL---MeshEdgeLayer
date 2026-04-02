package service

import (
	"encoding/json"
	"fmt"
	"sort"
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
	if a == nil || a.DB == nil {
		return nil
	}
	id := strings.TrimSpace(incidentID)
	if id == "" {
		return nil
	}
	rec, hadRow, err := a.DB.GetIncidentDecisionPackAdjudication(id)
	if err != nil {
		return nil
	}
	var cues []models.IncidentDecisionPackCueOutcome
	if hadRow {
		cues, err = db.DecodeCueOutcomesJSON(rec.CueOutcomesJSON)
		if err != nil {
			cues = nil
		}
	}
	latest, _ := a.DB.LatestIntelSignalOutcomesByIncident(id)
	var latestIntelAt string
	for _, row := range latest {
		if strings.TrimSpace(row.CreatedAt) > latestIntelAt {
			latestIntelAt = strings.TrimSpace(row.CreatedAt)
		}
	}
	byID := map[string]models.IncidentDecisionPackCueOutcome{}
	for _, c := range cues {
		cid := strings.TrimSpace(c.CueID)
		if cid == "" {
			continue
		}
		byID[cid] = c
	}
	for code, row := range latest {
		if strings.TrimSpace(row.Outcome) == "" {
			continue
		}
		cur := byID[code]
		if strings.TrimSpace(cur.Outcome) == "" {
			byID[code] = models.IncidentDecisionPackCueOutcome{
				CueID:   code,
				Outcome: strings.TrimSpace(row.Outcome),
				Note:    strings.TrimSpace(row.Note),
			}
		}
	}
	merged := make([]models.IncidentDecisionPackCueOutcome, 0, len(byID))
	for _, c := range byID {
		merged = append(merged, c)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].CueID < merged[j].CueID })

	out := &models.IncidentDecisionPackAdjudication{CueOutcomes: merged}
	if hadRow {
		out.Reviewed = rec.Reviewed
		out.ReviewedAt = rec.ReviewedAt
		out.ReviewedByActorID = rec.ReviewedByActorID
		out.Useful = rec.Useful
		out.OperatorNote = rec.OperatorNote
		out.UpdatedAt = rec.UpdatedAt
	} else if len(merged) > 0 && latestIntelAt != "" {
		out.UpdatedAt = latestIntelAt
	}
	if len(merged) == 0 && !hadRow {
		return nil
	}
	return out
}

func validPackCueOutcome(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "accepted", "dismissed", "reviewed", "snoozed":
		return true
	default:
		return false
	}
}

func validCueReasonCode(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	switch strings.ToLower(s) {
	case "false_positive", "needs_more_evidence", "defer", "other":
		return true
	default:
		return false
	}
}

// SyncPackCueOutcomeFromIntelSignal merges one assist-signal adjudication into incident_decision_pack_adjudication (dual-write path from POST .../intel-signal-outcome).
func (a *App) SyncPackCueOutcomeFromIntelSignal(incidentID, actorID, signalCode, outcome, note string) error {
	if a == nil || a.DB == nil {
		return fmt.Errorf("service not available")
	}
	incidentID = strings.TrimSpace(incidentID)
	signalCode = strings.TrimSpace(signalCode)
	outcome = strings.TrimSpace(outcome)
	if incidentID == "" || signalCode == "" || outcome == "" {
		return nil
	}
	if strings.TrimSpace(actorID) == "" {
		actorID = "system"
	}
	prev, hadPrev, err := a.DB.GetIncidentDecisionPackAdjudication(incidentID)
	if err != nil {
		return err
	}
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
	existing, _ := db.DecodeCueOutcomesJSON(rec.CueOutcomesJSON)
	byID := map[string]models.IncidentDecisionPackCueOutcome{}
	for _, c := range existing {
		id := strings.TrimSpace(c.CueID)
		if id == "" {
			continue
		}
		byID[id] = c
	}
	cur := byID[signalCode]
	cur.CueID = signalCode
	cur.Outcome = outcome
	cur.Note = strings.TrimSpace(note)
	byID[signalCode] = cur
	var merged []models.IncidentDecisionPackCueOutcome
	for _, c := range byID {
		merged = append(merged, c)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].CueID < merged[j].CueID })
	b, err := json.Marshal(merged)
	if err != nil {
		return err
	}
	rec.CueOutcomesJSON = string(b)
	rec.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return a.DB.UpsertIncidentDecisionPackAdjudication(rec)
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
		if !validPackCueOutcome(o) {
			return fmt.Errorf("invalid cue outcome for %q", id)
		}
		rc := strings.TrimSpace(c.ReasonCode)
		if !validCueReasonCode(rc) {
			return fmt.Errorf("invalid cue reason_code for %q", id)
		}
		byID[id] = models.IncidentDecisionPackCueOutcome{CueID: id, Outcome: o, Note: strings.TrimSpace(c.Note), ReasonCode: rc}
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
	for _, c := range patch.CueOutcomes {
		id := strings.TrimSpace(c.CueID)
		o := strings.TrimSpace(c.Outcome)
		if id == "" || o == "" {
			continue
		}
		_ = a.DB.InsertIncidentIntelSignalOutcome(db.IncidentIntelSignalOutcomeRecord{
			ID:         newTrustID("iso"),
			IncidentID: incidentID,
			SignalCode: id,
			Outcome:    o,
			ActorID:    actorID,
			Note:       strings.TrimSpace(c.Note),
		})
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
