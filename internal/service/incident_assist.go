package service

import (
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
