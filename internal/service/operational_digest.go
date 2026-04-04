package service

import (
	"strings"
	"time"

	"github.com/mel-project/mel/internal/models"
)

const operationalDigestSchemaVersion = "mel.operator_operational_digest/v1"

// BuildOperationalDigest returns deterministic row-count snapshots from this instance's database.
func (a *App) BuildOperationalDigest() models.OperatorOperationalDigestDTO {
	out := models.OperatorOperationalDigestDTO{
		SchemaVersion: operationalDigestSchemaVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		WindowHours:   24,
	}
	out.References.Timeline = "/api/v1/timeline"
	out.References.Incidents = "/api/v1/incidents"
	out.References.ControlHistory = "/api/v1/control/history"
	out.TruthNotes = []string{
		"Counts reflect rows in this MEL instance database only, not fleet-wide posture.",
		"Resolved-in-7-days uses incident resolved_at when present; absence of rows is not proof of calm runtime.",
	}
	if a != nil && a.Cfg.Integration.Enabled {
		out.TruthNotes = append(out.TruthNotes, "Outbound integration webhooks are enabled in config; delivery is best-effort and audited separately from this digest.")
	}

	if a == nil || a.DB == nil {
		out.TruthNotes = append(out.TruthNotes, "Database unavailable; digest counts are empty.")
		return out
	}
	id, err := a.DB.EnsureInstanceID()
	if err == nil && strings.TrimSpace(id) != "" {
		out.InstanceID = strings.TrimSpace(id)
	}

	snap, err := a.DB.OperationalDigestSnapshot()
	if err != nil {
		out.TruthNotes = append(out.TruthNotes, "Snapshot query failed; partial or empty counts.")
		return out
	}
	out.Counts.OpenIncidents = snap.OpenIncidents
	out.Counts.CriticalOpenIncidents = snap.CriticalOpenIncidents
	out.Counts.HighOpenIncidents = snap.HighOpenIncidents
	out.Counts.ResolvedLast7Days = snap.ResolvedLast7Days
	out.Counts.ControlActionsTotal = snap.ControlActionsTotal
	out.Counts.PendingApprovalActions = snap.PendingApprovalActions
	out.Counts.AwaitingExecutorActions = snap.AwaitingExecutorActions
	out.Counts.OperatorNotesTotal = snap.OperatorNotesTotal

	cutoff := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	win, werr := a.DB.OperationalDigestWindow(cutoff)
	if werr != nil {
		out.TruthNotes = append(out.TruthNotes, "24h window counts unavailable due to query error.")
	} else {
		out.Window.IncidentsOpened = win.IncidentsOpened
		out.Window.ControlActionsCreated = win.ControlActionsCreated
		out.Window.OperatorNotesCreated = win.OperatorNotesCreated
	}
	return out
}
