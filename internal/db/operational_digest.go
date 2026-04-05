package db

import (
	"fmt"
	"strings"
	"time"
)

// OperationalDigestCounts is a point-in-time snapshot of durable operator-relevant row counts.
type OperationalDigestCounts struct {
	OpenIncidents           int
	CriticalOpenIncidents   int
	HighOpenIncidents       int
	ResolvedLast7Days       int
	ControlActionsTotal     int
	PendingApprovalActions  int
	AwaitingExecutorActions int
	OperatorNotesTotal      int
	LastIncidentUpdatedAt   string
}

// OperationalDigestWindowCounts aggregates rows whose primary timestamp falls in [cutoff, now] (UTC).
type OperationalDigestWindowCounts struct {
	IncidentsOpened       int
	ControlActionsCreated int
	OperatorNotesCreated  int
}

// OperationalDigestSnapshot returns counts for GET /api/v1/operator/digest and similar surfaces.
func (d *DB) OperationalDigestSnapshot() (OperationalDigestCounts, error) {
	var out OperationalDigestCounts
	if d == nil {
		return out, fmt.Errorf("db nil")
	}
	openRows, err := d.QueryRows(`SELECT COUNT(*) AS c FROM incidents WHERE LOWER(state) NOT IN ('resolved','closed');`)
	if err != nil {
		return out, err
	}
	out.OpenIncidents = int(asInt(openRows[0]["c"]))

	critRows, err := d.QueryRows(`SELECT COUNT(*) AS c FROM incidents WHERE LOWER(state) NOT IN ('resolved','closed') AND LOWER(severity)='critical';`)
	if err != nil {
		return out, err
	}
	out.CriticalOpenIncidents = int(asInt(critRows[0]["c"]))

	highRows, err := d.QueryRows(`SELECT COUNT(*) AS c FROM incidents WHERE LOWER(state) NOT IN ('resolved','closed') AND LOWER(severity)='high';`)
	if err != nil {
		return out, err
	}
	out.HighOpenIncidents = int(asInt(highRows[0]["c"]))

	cutoff7d := time.Now().UTC().Add(-7 * 24 * time.Hour).Format(time.RFC3339)
	res7, err := d.QueryRows(fmt.Sprintf(`SELECT COUNT(*) AS c FROM incidents WHERE resolved_at IS NOT NULL AND resolved_at != '' AND resolved_at >= '%s';`, esc(cutoff7d)))
	if err != nil {
		return out, err
	}
	out.ResolvedLast7Days = int(asInt(res7[0]["c"]))

	caTotal, err := d.QueryRows(`SELECT COUNT(*) AS c FROM control_actions;`)
	if err != nil {
		return out, err
	}
	out.ControlActionsTotal = int(asInt(caTotal[0]["c"]))

	pend, err := d.QueryRows(`SELECT COUNT(*) AS c FROM control_actions WHERE lifecycle_state='pending_approval';`)
	if err != nil {
		return out, err
	}
	out.PendingApprovalActions = int(asInt(pend[0]["c"]))

	await, err := d.QueryRows(`SELECT COUNT(*) AS c FROM control_actions WHERE lifecycle_state='pending' AND result='approved';`)
	if err != nil {
		return out, err
	}
	out.AwaitingExecutorActions = int(asInt(await[0]["c"]))

	notes, err := d.QueryRows(`SELECT COUNT(*) AS c FROM operator_notes;`)
	if err != nil {
		return out, err
	}
	out.OperatorNotesTotal = int(asInt(notes[0]["c"]))

	lastUp, err := d.QueryRows(`SELECT COALESCE(MAX(updated_at),'') AS t FROM incidents;`)
	if err != nil {
		return out, err
	}
	if len(lastUp) > 0 {
		out.LastIncidentUpdatedAt = strings.TrimSpace(asString(lastUp[0]["t"]))
	}
	return out, nil
}

// OperationalDigestWindow returns activity counts in the inclusive window [cutoffRFC3339, now] using primary timestamps.
func (d *DB) OperationalDigestWindow(cutoffRFC3339 string) (OperationalDigestWindowCounts, error) {
	var out OperationalDigestWindowCounts
	if d == nil {
		return out, fmt.Errorf("db nil")
	}
	cutoff := strings.TrimSpace(cutoffRFC3339)
	if cutoff == "" {
		return out, fmt.Errorf("empty cutoff")
	}
	inc, err := d.QueryRows(fmt.Sprintf(`SELECT COUNT(*) AS c FROM incidents WHERE occurred_at >= '%s';`, esc(cutoff)))
	if err != nil {
		return out, err
	}
	out.IncidentsOpened = int(asInt(inc[0]["c"]))

	ca, err := d.QueryRows(fmt.Sprintf(`SELECT COUNT(*) AS c FROM control_actions WHERE created_at >= '%s';`, esc(cutoff)))
	if err != nil {
		return out, err
	}
	out.ControlActionsCreated = int(asInt(ca[0]["c"]))

	on, err := d.QueryRows(fmt.Sprintf(`SELECT COUNT(*) AS c FROM operator_notes WHERE created_at >= '%s';`, esc(cutoff)))
	if err != nil {
		return out, err
	}
	out.OperatorNotesCreated = int(asInt(on[0]["c"]))
	return out, nil
}
