package service

import (
	"strings"

	"github.com/mel-project/mel/internal/incidentassist"
	"github.com/mel-project/mel/internal/models"
)

func (a *App) attachAssistSignals(inc *models.Incident) {
	if inc == nil || inc.Intelligence == nil {
		return
	}
	inc.AssistSignals = incidentassist.Compute(*inc, inc.Intelligence)
	if inc.AssistSignals == nil || len(inc.AssistSignals.Signals) == 0 {
		return
	}
	if a == nil || a.DB == nil {
		return
	}
	latest, err := a.DB.LatestIntelSignalOutcomesByIncident(inc.ID)
	if err != nil || len(latest) == 0 {
		return
	}
	sigs := inc.AssistSignals.Signals
	for i := range sigs {
		if row, ok := latest[sigs[i].Code]; ok {
			sigs[i].OperatorState = &models.IncidentAssistSignalOperatorState{
				LatestOutcome: row.Outcome,
				LatestAt:      row.CreatedAt,
				ActorID:       row.ActorID,
			}
		}
	}
}

// overlayAssistOperatorStateFromPack applies durable pack cue outcomes over intel-table snapshots when a pack adjudication row exists.
func (a *App) overlayAssistOperatorStateFromPack(inc *models.Incident) {
	if inc == nil || inc.AssistSignals == nil || len(inc.AssistSignals.Signals) == 0 {
		return
	}
	pack := inc.DecisionPack
	if pack == nil || pack.OperatorAdjudication == nil {
		return
	}
	adj := pack.OperatorAdjudication
	if strings.TrimSpace(adj.UpdatedAt) == "" {
		return
	}
	byCode := map[string]models.IncidentDecisionPackCueOutcome{}
	for _, c := range adj.CueOutcomes {
		id := strings.TrimSpace(c.CueID)
		if id == "" {
			continue
		}
		byCode[id] = c
	}
	actor := strings.TrimSpace(adj.ReviewedByActorID)
	for i := range inc.AssistSignals.Signals {
		code := strings.TrimSpace(inc.AssistSignals.Signals[i].Code)
		cue, ok := byCode[code]
		if !ok || strings.TrimSpace(cue.Outcome) == "" {
			continue
		}
		inc.AssistSignals.Signals[i].OperatorState = &models.IncidentAssistSignalOperatorState{
			LatestOutcome: cue.Outcome,
			LatestAt:      adj.UpdatedAt,
			ActorID:       actor,
		}
	}
}
