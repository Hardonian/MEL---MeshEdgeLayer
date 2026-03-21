package db

// trust.go — DB operations for the trust and operability layer:
//   - Approval lifecycle (approve/reject/pending-approval)
//   - Evidence bundles
//   - Control freezes
//   - Maintenance windows
//   - Operator notes
//   - Unified timeline events
//   - Control plane state

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ─── Evidence Bundles ─────────────────────────────────────────────────────────

// EvidenceBundleRecord is the durable provenance record tying every action /
// decision to the observations, health data, and policy context that led to it.
type EvidenceBundleRecord struct {
	ID                   string         `json:"id"`
	ActionID             string         `json:"action_id,omitempty"`
	AlertID              string         `json:"alert_id,omitempty"`
	DecisionID           string         `json:"decision_id,omitempty"`
	Observations         []any          `json:"observations,omitempty"`
	Anomalies            []any          `json:"anomalies,omitempty"`
	HealthSnapshots      []any          `json:"health_snapshots,omitempty"`
	PolicyVersion        string         `json:"policy_version,omitempty"`
	Explanation          map[string]any `json:"explanation,omitempty"`
	TransportHealth      map[string]any `json:"transport_health,omitempty"`
	PriorDecisions       []any          `json:"prior_decisions,omitempty"`
	OperatorAnnotations  []any          `json:"operator_annotations,omitempty"`
	ExecutionResult      map[string]any `json:"execution_result,omitempty"`
	IntegrityHash        string         `json:"integrity_hash,omitempty"`
	SourceType           string         `json:"source_type,omitempty"`
	CapturedAt           string         `json:"captured_at,omitempty"`
	UpdatedAt            string         `json:"updated_at,omitempty"`
}

func (d *DB) UpsertEvidenceBundle(bundle EvidenceBundleRecord) error {
	if strings.TrimSpace(bundle.ID) == "" {
		return fmt.Errorf("evidence bundle id is required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if bundle.CapturedAt == "" {
		bundle.CapturedAt = now
	}
	bundle.UpdatedAt = now
	if bundle.SourceType == "" {
		bundle.SourceType = "system"
	}
	obsJSON, _ := json.Marshal(bundle.Observations)
	anomJSON, _ := json.Marshal(bundle.Anomalies)
	snapJSON, _ := json.Marshal(bundle.HealthSnapshots)
	explJSON, _ := json.Marshal(bundle.Explanation)
	thJSON, _ := json.Marshal(bundle.TransportHealth)
	priorJSON, _ := json.Marshal(bundle.PriorDecisions)
	annotJSON, _ := json.Marshal(bundle.OperatorAnnotations)
	resultJSON, _ := json.Marshal(bundle.ExecutionResult)
	sql := fmt.Sprintf(`INSERT INTO evidence_bundles(id,action_id,alert_id,decision_id,observations_json,anomalies_json,health_snapshots_json,policy_version,explanation_json,transport_health_json,prior_decisions_json,operator_annotations,execution_result_json,integrity_hash,source_type,captured_at,updated_at)
VALUES('%s',%s,%s,%s,'%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s')
ON CONFLICT(id) DO UPDATE SET action_id=excluded.action_id,alert_id=excluded.alert_id,decision_id=excluded.decision_id,observations_json=excluded.observations_json,anomalies_json=excluded.anomalies_json,health_snapshots_json=excluded.health_snapshots_json,policy_version=excluded.policy_version,explanation_json=excluded.explanation_json,transport_health_json=excluded.transport_health_json,prior_decisions_json=excluded.prior_decisions_json,operator_annotations=excluded.operator_annotations,execution_result_json=excluded.execution_result_json,integrity_hash=excluded.integrity_hash,source_type=excluded.source_type,updated_at=excluded.updated_at;`,
		esc(bundle.ID), sqlString(bundle.ActionID), sqlString(bundle.AlertID), sqlString(bundle.DecisionID),
		esc(string(obsJSON)), esc(string(anomJSON)), esc(string(snapJSON)),
		esc(bundle.PolicyVersion), esc(string(explJSON)), esc(string(thJSON)),
		esc(string(priorJSON)), esc(string(annotJSON)), esc(string(resultJSON)),
		esc(bundle.IntegrityHash), esc(bundle.SourceType), esc(bundle.CapturedAt), esc(bundle.UpdatedAt))
	return d.Exec(sql)
}

func (d *DB) EvidenceBundleByID(id string) (EvidenceBundleRecord, bool, error) {
	safeID, err := ValidateSQLInput(id)
	if err != nil {
		logSuspiciousSQL(id, err.Error())
		return EvidenceBundleRecord{}, false, fmt.Errorf("invalid id: %w", err)
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id,COALESCE(action_id,'') AS action_id,COALESCE(alert_id,'') AS alert_id,COALESCE(decision_id,'') AS decision_id,COALESCE(observations_json,'[]') AS observations_json,COALESCE(anomalies_json,'[]') AS anomalies_json,COALESCE(health_snapshots_json,'[]') AS health_snapshots_json,COALESCE(policy_version,'') AS policy_version,COALESCE(explanation_json,'{}') AS explanation_json,COALESCE(transport_health_json,'{}') AS transport_health_json,COALESCE(prior_decisions_json,'[]') AS prior_decisions_json,COALESCE(operator_annotations,'[]') AS operator_annotations,COALESCE(execution_result_json,'{}') AS execution_result_json,COALESCE(integrity_hash,'') AS integrity_hash,COALESCE(source_type,'system') AS source_type,captured_at,COALESCE(updated_at,'') AS updated_at
FROM evidence_bundles WHERE id='%s' LIMIT 1;`, safeID))
	if err != nil {
		return EvidenceBundleRecord{}, false, err
	}
	if len(rows) == 0 {
		return EvidenceBundleRecord{}, false, nil
	}
	return evidenceBundleFromRow(rows[0]), true, nil
}

func (d *DB) EvidenceBundleByActionID(actionID string) (EvidenceBundleRecord, bool, error) {
	safeID, err := ValidateSQLInput(actionID)
	if err != nil {
		logSuspiciousSQL(actionID, err.Error())
		return EvidenceBundleRecord{}, false, fmt.Errorf("invalid action id: %w", err)
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id,COALESCE(action_id,'') AS action_id,COALESCE(alert_id,'') AS alert_id,COALESCE(decision_id,'') AS decision_id,COALESCE(observations_json,'[]') AS observations_json,COALESCE(anomalies_json,'[]') AS anomalies_json,COALESCE(health_snapshots_json,'[]') AS health_snapshots_json,COALESCE(policy_version,'') AS policy_version,COALESCE(explanation_json,'{}') AS explanation_json,COALESCE(transport_health_json,'{}') AS transport_health_json,COALESCE(prior_decisions_json,'[]') AS prior_decisions_json,COALESCE(operator_annotations,'[]') AS operator_annotations,COALESCE(execution_result_json,'{}') AS execution_result_json,COALESCE(integrity_hash,'') AS integrity_hash,COALESCE(source_type,'system') AS source_type,captured_at,COALESCE(updated_at,'') AS updated_at
FROM evidence_bundles WHERE action_id='%s' ORDER BY captured_at DESC LIMIT 1;`, safeID))
	if err != nil {
		return EvidenceBundleRecord{}, false, err
	}
	if len(rows) == 0 {
		return EvidenceBundleRecord{}, false, nil
	}
	return evidenceBundleFromRow(rows[0]), true, nil
}

func evidenceBundleFromRow(row map[string]any) EvidenceBundleRecord {
	b := EvidenceBundleRecord{
		ID:            asString(row["id"]),
		ActionID:      asString(row["action_id"]),
		AlertID:       asString(row["alert_id"]),
		DecisionID:    asString(row["decision_id"]),
		PolicyVersion: asString(row["policy_version"]),
		IntegrityHash: asString(row["integrity_hash"]),
		SourceType:    asString(row["source_type"]),
		CapturedAt:    asString(row["captured_at"]),
		UpdatedAt:     asString(row["updated_at"]),
	}
	_ = json.Unmarshal([]byte(asString(row["observations_json"])), &b.Observations)
	_ = json.Unmarshal([]byte(asString(row["anomalies_json"])), &b.Anomalies)
	_ = json.Unmarshal([]byte(asString(row["health_snapshots_json"])), &b.HealthSnapshots)
	_ = json.Unmarshal([]byte(asString(row["explanation_json"])), &b.Explanation)
	_ = json.Unmarshal([]byte(asString(row["transport_health_json"])), &b.TransportHealth)
	_ = json.Unmarshal([]byte(asString(row["prior_decisions_json"])), &b.PriorDecisions)
	_ = json.Unmarshal([]byte(asString(row["operator_annotations"])), &b.OperatorAnnotations)
	_ = json.Unmarshal([]byte(asString(row["execution_result_json"])), &b.ExecutionResult)
	return b
}

// ─── Control Approval ─────────────────────────────────────────────────────────

// PendingApprovalActions returns all control_actions in pending_approval state.
func (d *DB) PendingApprovalActions(limit int) ([]ControlActionRecord, error) {
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id,COALESCE(decision_id,'') AS decision_id,action_type,COALESCE(target_transport,'') AS target_transport,COALESCE(target_segment,'') AS target_segment,COALESCE(target_node,'') AS target_node,reason,confidence,COALESCE(trigger_evidence_json,'[]') AS trigger_evidence_json,COALESCE(episode_id,'') AS episode_id,created_at,COALESCE(executed_at,'') AS executed_at,COALESCE(completed_at,'') AS completed_at,COALESCE(result,'') AS result,reversible,COALESCE(expires_at,'') AS expires_at,COALESCE(outcome_detail,'') AS outcome_detail,mode,COALESCE(policy_rule,'') AS policy_rule,COALESCE(lifecycle_state,'') AS lifecycle_state,COALESCE(advisory_only,0) AS advisory_only,COALESCE(denial_code,'') AS denial_code,COALESCE(closure_state,'') AS closure_state,COALESCE(metadata_json,'{}') AS metadata_json,COALESCE(execution_mode,'auto') AS execution_mode,COALESCE(proposed_by,'system') AS proposed_by,COALESCE(approved_by,'') AS approved_by,COALESCE(approved_at,'') AS approved_at,COALESCE(rejected_by,'') AS rejected_by,COALESCE(rejected_at,'') AS rejected_at,COALESCE(approval_note,'') AS approval_note,COALESCE(approval_expires_at,'') AS approval_expires_at,COALESCE(blast_radius_class,'unknown') AS blast_radius_class,COALESCE(evidence_bundle_id,'') AS evidence_bundle_id
FROM control_actions WHERE lifecycle_state='pending_approval' ORDER BY created_at ASC LIMIT %d;`, limit))
	if err != nil {
		return nil, err
	}
	out := make([]ControlActionRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, controlActionFromRowExtended(row))
	}
	return out, nil
}

// ApproveControlAction moves a pending_approval action to pending (ready for execution).
// Returns ErrNotFound if the action does not exist or is not in pending_approval state.
func (d *DB) ApproveControlAction(id, approvedBy, note string) error {
	safeID, err := ValidateSQLInput(id)
	if err != nil {
		logSuspiciousSQL(id, err.Error())
		return fmt.Errorf("invalid id: %w", err)
	}
	safeBy := esc(approvedBy)
	safeNote := esc(note)
	now := esc(time.Now().UTC().Format(time.RFC3339))
	sql := fmt.Sprintf(`UPDATE control_actions SET lifecycle_state='pending', result='approved', approved_by='%s', approved_at='%s', approval_note='%s' WHERE id='%s' AND lifecycle_state='pending_approval';`,
		safeBy, now, safeNote, safeID)
	return d.Exec(sql)
}

// RejectControlAction moves a pending_approval action to a rejected terminal state.
func (d *DB) RejectControlAction(id, rejectedBy, note string) error {
	safeID, err := ValidateSQLInput(id)
	if err != nil {
		logSuspiciousSQL(id, err.Error())
		return fmt.Errorf("invalid id: %w", err)
	}
	safeBy := esc(rejectedBy)
	safeNote := esc(note)
	now := esc(time.Now().UTC().Format(time.RFC3339))
	sql := fmt.Sprintf(`UPDATE control_actions SET lifecycle_state='completed', result='rejected', closure_state='rejected_by_operator', rejected_by='%s', rejected_at='%s', approval_note='%s', completed_at='%s' WHERE id='%s' AND lifecycle_state='pending_approval';`,
		safeBy, now, safeNote, now, safeID)
	return d.Exec(sql)
}

// ExpireStaleApprovalActions marks all pending_approval actions whose
// approval_expires_at is in the past as expired.
func (d *DB) ExpireStaleApprovalActions(now time.Time) error {
	ts := esc(now.UTC().Format(time.RFC3339))
	sql := fmt.Sprintf(`UPDATE control_actions SET lifecycle_state='completed', result='approval_expired', closure_state='approval_expired', completed_at='%s'
WHERE lifecycle_state='pending_approval' AND approval_expires_at != '' AND approval_expires_at <= '%s';`, ts, ts)
	return d.Exec(sql)
}

// controlActionFromRowExtended extends controlActionFromRow with the new trust columns.
func controlActionFromRowExtended(row map[string]any) ControlActionRecord {
	base := controlActionFromRow(row)
	base.ExecutionMode = asString(row["execution_mode"])
	base.ProposedBy = asString(row["proposed_by"])
	base.ApprovedBy = asString(row["approved_by"])
	base.ApprovedAt = asString(row["approved_at"])
	base.RejectedBy = asString(row["rejected_by"])
	base.RejectedAt = asString(row["rejected_at"])
	base.ApprovalNote = asString(row["approval_note"])
	base.ApprovalExpiresAt = asString(row["approval_expires_at"])
	base.BlastRadiusClass = asString(row["blast_radius_class"])
	base.EvidenceBundleID = asString(row["evidence_bundle_id"])
	return base
}

// ─── Control Freezes ─────────────────────────────────────────────────────────

// FreezeRecord represents a runtime-writable freeze on autonomous control actions.
type FreezeRecord struct {
	ID         string `json:"id"`
	ScopeType  string `json:"scope_type"`  // global | transport | action_type
	ScopeValue string `json:"scope_value"` // '' for global
	Reason     string `json:"reason"`
	CreatedBy  string `json:"created_by"`
	CreatedAt  string `json:"created_at"`
	ExpiresAt  string `json:"expires_at,omitempty"`
	ClearedBy  string `json:"cleared_by,omitempty"`
	ClearedAt  string `json:"cleared_at,omitempty"`
	Active     bool   `json:"active"`
}

func (d *DB) CreateFreeze(f FreezeRecord) error {
	if strings.TrimSpace(f.ID) == "" {
		return fmt.Errorf("freeze id is required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if f.CreatedAt == "" {
		f.CreatedAt = now
	}
	if f.ScopeType == "" {
		f.ScopeType = "global"
	}
	if f.CreatedBy == "" {
		f.CreatedBy = "system"
	}
	sql := fmt.Sprintf(`INSERT INTO control_freezes(id,scope_type,scope_value,reason,created_by,created_at,expires_at,active)
VALUES('%s','%s','%s','%s','%s','%s',%s,1);`,
		esc(f.ID), esc(f.ScopeType), esc(f.ScopeValue), esc(f.Reason),
		esc(f.CreatedBy), esc(f.CreatedAt), sqlString(f.ExpiresAt))
	return d.Exec(sql)
}

func (d *DB) ClearFreeze(id, clearedBy string) error {
	safeID, err := ValidateSQLInput(id)
	if err != nil {
		logSuspiciousSQL(id, err.Error())
		return fmt.Errorf("invalid freeze id: %w", err)
	}
	now := esc(time.Now().UTC().Format(time.RFC3339))
	safeBy := esc(clearedBy)
	sql := fmt.Sprintf(`UPDATE control_freezes SET active=0, cleared_by='%s', cleared_at='%s' WHERE id='%s' AND active=1;`,
		safeBy, now, safeID)
	return d.Exec(sql)
}

func (d *DB) ActiveFreezes() ([]FreezeRecord, error) {
	rows, err := d.QueryRows(`SELECT id,COALESCE(scope_type,'global') AS scope_type,COALESCE(scope_value,'') AS scope_value,COALESCE(reason,'') AS reason,COALESCE(created_by,'system') AS created_by,created_at,COALESCE(expires_at,'') AS expires_at,COALESCE(cleared_by,'') AS cleared_by,COALESCE(cleared_at,'') AS cleared_at,active
FROM control_freezes WHERE active=1 ORDER BY created_at ASC;`)
	if err != nil {
		return nil, err
	}
	out := make([]FreezeRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, FreezeRecord{
			ID:         asString(row["id"]),
			ScopeType:  asString(row["scope_type"]),
			ScopeValue: asString(row["scope_value"]),
			Reason:     asString(row["reason"]),
			CreatedBy:  asString(row["created_by"]),
			CreatedAt:  asString(row["created_at"]),
			ExpiresAt:  asString(row["expires_at"]),
			ClearedBy:  asString(row["cleared_by"]),
			ClearedAt:  asString(row["cleared_at"]),
			Active:     asInt(row["active"]) == 1,
		})
	}
	return out, nil
}

// IsFrozen returns true if any active freeze applies to the given transport and
// action type. Checks global, transport-scoped, and action-type-scoped freezes.
func (d *DB) IsFrozen(transportName, actionType string) (bool, string, error) {
	freezes, err := d.ActiveFreezes()
	if err != nil {
		return false, "", err
	}
	now := time.Now().UTC()
	for _, f := range freezes {
		// Check expiry
		if f.ExpiresAt != "" {
			exp, err2 := time.Parse(time.RFC3339, f.ExpiresAt)
			if err2 == nil && now.After(exp) {
				// expired but not yet cleaned up; skip
				continue
			}
		}
		switch f.ScopeType {
		case "global":
			return true, f.Reason, nil
		case "transport":
			if strings.TrimSpace(transportName) != "" && f.ScopeValue == transportName {
				return true, f.Reason, nil
			}
		case "action_type":
			if strings.TrimSpace(actionType) != "" && f.ScopeValue == actionType {
				return true, f.Reason, nil
			}
		}
	}
	return false, "", nil
}

// ExpireOldFreezes clears active freezes whose expires_at is past.
func (d *DB) ExpireOldFreezes(now time.Time) error {
	ts := esc(now.UTC().Format(time.RFC3339))
	sql := fmt.Sprintf(`UPDATE control_freezes SET active=0, cleared_by='system_expiry', cleared_at='%s'
WHERE active=1 AND expires_at != '' AND expires_at <= '%s';`, ts, ts)
	return d.Exec(sql)
}

// ─── Maintenance Windows ──────────────────────────────────────────────────────

// MaintenanceWindowRecord represents a time-bounded suppression of autonomous control.
type MaintenanceWindowRecord struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Reason      string `json:"reason"`
	ScopeType   string `json:"scope_type"`  // global | transport
	ScopeValue  string `json:"scope_value"`
	StartsAt    string `json:"starts_at"`
	EndsAt      string `json:"ends_at"`
	CreatedBy   string `json:"created_by"`
	CreatedAt   string `json:"created_at"`
	CancelledBy string `json:"cancelled_by,omitempty"`
	CancelledAt string `json:"cancelled_at,omitempty"`
	Active      bool   `json:"active"`
}

func (d *DB) CreateMaintenanceWindow(mw MaintenanceWindowRecord) error {
	if strings.TrimSpace(mw.ID) == "" {
		return fmt.Errorf("maintenance window id is required")
	}
	if mw.CreatedAt == "" {
		mw.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if mw.CreatedBy == "" {
		mw.CreatedBy = "system"
	}
	if mw.ScopeType == "" {
		mw.ScopeType = "global"
	}
	sql := fmt.Sprintf(`INSERT INTO maintenance_windows(id,title,reason,scope_type,scope_value,starts_at,ends_at,created_by,created_at,active)
VALUES('%s','%s','%s','%s','%s','%s','%s','%s','%s',1);`,
		esc(mw.ID), esc(mw.Title), esc(mw.Reason), esc(mw.ScopeType),
		esc(mw.ScopeValue), esc(mw.StartsAt), esc(mw.EndsAt),
		esc(mw.CreatedBy), esc(mw.CreatedAt))
	return d.Exec(sql)
}

func (d *DB) CancelMaintenanceWindow(id, cancelledBy string) error {
	safeID, err := ValidateSQLInput(id)
	if err != nil {
		logSuspiciousSQL(id, err.Error())
		return fmt.Errorf("invalid maintenance window id: %w", err)
	}
	now := esc(time.Now().UTC().Format(time.RFC3339))
	safeBy := esc(cancelledBy)
	sql := fmt.Sprintf(`UPDATE maintenance_windows SET active=0, cancelled_by='%s', cancelled_at='%s' WHERE id='%s' AND active=1;`,
		safeBy, now, safeID)
	return d.Exec(sql)
}

func (d *DB) ActiveMaintenanceWindows(now time.Time) ([]MaintenanceWindowRecord, error) {
	ts := esc(now.UTC().Format(time.RFC3339))
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id,COALESCE(title,'') AS title,COALESCE(reason,'') AS reason,COALESCE(scope_type,'global') AS scope_type,COALESCE(scope_value,'') AS scope_value,starts_at,ends_at,COALESCE(created_by,'system') AS created_by,created_at,COALESCE(cancelled_by,'') AS cancelled_by,COALESCE(cancelled_at,'') AS cancelled_at,active
FROM maintenance_windows WHERE active=1 AND starts_at <= '%s' AND ends_at >= '%s' ORDER BY starts_at ASC;`, ts, ts))
	if err != nil {
		return nil, err
	}
	out := make([]MaintenanceWindowRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, maintenanceWindowFromRow(row))
	}
	return out, nil
}

func (d *DB) AllMaintenanceWindows(limit int) ([]MaintenanceWindowRecord, error) {
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id,COALESCE(title,'') AS title,COALESCE(reason,'') AS reason,COALESCE(scope_type,'global') AS scope_type,COALESCE(scope_value,'') AS scope_value,starts_at,ends_at,COALESCE(created_by,'system') AS created_by,created_at,COALESCE(cancelled_by,'') AS cancelled_by,COALESCE(cancelled_at,'') AS cancelled_at,active
FROM maintenance_windows ORDER BY starts_at DESC LIMIT %d;`, limit))
	if err != nil {
		return nil, err
	}
	out := make([]MaintenanceWindowRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, maintenanceWindowFromRow(row))
	}
	return out, nil
}

// IsInMaintenance returns true if any active maintenance window currently applies.
func (d *DB) IsInMaintenance(transportName string, now time.Time) (bool, string, error) {
	windows, err := d.ActiveMaintenanceWindows(now)
	if err != nil {
		return false, "", err
	}
	for _, w := range windows {
		switch w.ScopeType {
		case "global":
			return true, w.Reason, nil
		case "transport":
			if strings.TrimSpace(transportName) != "" && w.ScopeValue == transportName {
				return true, w.Reason, nil
			}
		}
	}
	return false, "", nil
}

func maintenanceWindowFromRow(row map[string]any) MaintenanceWindowRecord {
	return MaintenanceWindowRecord{
		ID:          asString(row["id"]),
		Title:       asString(row["title"]),
		Reason:      asString(row["reason"]),
		ScopeType:   asString(row["scope_type"]),
		ScopeValue:  asString(row["scope_value"]),
		StartsAt:    asString(row["starts_at"]),
		EndsAt:      asString(row["ends_at"]),
		CreatedBy:   asString(row["created_by"]),
		CreatedAt:   asString(row["created_at"]),
		CancelledBy: asString(row["cancelled_by"]),
		CancelledAt: asString(row["cancelled_at"]),
		Active:      asInt(row["active"]) == 1,
	}
}

// ─── Operator Notes ───────────────────────────────────────────────────────────

// OperatorNoteRecord is a freeform annotation an operator can attach to any
// resource in the system.
type OperatorNoteRecord struct {
	ID        string `json:"id"`
	RefType   string `json:"ref_type"` // action | incident | node | transport | segment | bundle
	RefID     string `json:"ref_id"`
	ActorID   string `json:"actor_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (d *DB) CreateOperatorNote(note OperatorNoteRecord) error {
	if strings.TrimSpace(note.ID) == "" {
		return fmt.Errorf("operator note id is required")
	}
	if strings.TrimSpace(note.RefType) == "" || strings.TrimSpace(note.RefID) == "" {
		return fmt.Errorf("operator note ref_type and ref_id are required")
	}
	if strings.TrimSpace(note.Content) == "" {
		return fmt.Errorf("operator note content is required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if note.CreatedAt == "" {
		note.CreatedAt = now
	}
	if note.UpdatedAt == "" {
		note.UpdatedAt = now
	}
	if note.ActorID == "" {
		note.ActorID = "system"
	}
	sql := fmt.Sprintf(`INSERT INTO operator_notes(id,ref_type,ref_id,actor_id,content,created_at,updated_at)
VALUES('%s','%s','%s','%s','%s','%s','%s');`,
		esc(note.ID), esc(note.RefType), esc(note.RefID),
		esc(note.ActorID), esc(note.Content), esc(note.CreatedAt), esc(note.UpdatedAt))
	return d.Exec(sql)
}

func (d *DB) OperatorNotesByRef(refType, refID string, limit int) ([]OperatorNoteRecord, error) {
	limit = clampLimit(limit)
	safeType, err := ValidateSQLInput(refType)
	if err != nil {
		logSuspiciousSQL(refType, err.Error())
		return nil, fmt.Errorf("invalid ref_type: %w", err)
	}
	safeID, err := ValidateSQLInput(refID)
	if err != nil {
		logSuspiciousSQL(refID, err.Error())
		return nil, fmt.Errorf("invalid ref_id: %w", err)
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id,ref_type,ref_id,COALESCE(actor_id,'system') AS actor_id,content,created_at,COALESCE(updated_at,'') AS updated_at
FROM operator_notes WHERE ref_type='%s' AND ref_id='%s' ORDER BY created_at DESC LIMIT %d;`,
		safeType, safeID, limit))
	if err != nil {
		return nil, err
	}
	out := make([]OperatorNoteRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, OperatorNoteRecord{
			ID:        asString(row["id"]),
			RefType:   asString(row["ref_type"]),
			RefID:     asString(row["ref_id"]),
			ActorID:   asString(row["actor_id"]),
			Content:   asString(row["content"]),
			CreatedAt: asString(row["created_at"]),
			UpdatedAt: asString(row["updated_at"]),
		})
	}
	return out, nil
}

// ─── Unified Timeline ─────────────────────────────────────────────────────────

// TimelineEvent is a single item in the unified operator event timeline.
type TimelineEvent struct {
	EventTime   string         `json:"event_time"`
	EventType   string         `json:"event_type"`  // action | decision | incident | note | freeze | maintenance
	EventID     string         `json:"event_id"`
	Summary     string         `json:"summary"`
	Severity    string         `json:"severity,omitempty"`
	ActorID     string         `json:"actor_id,omitempty"`
	ResourceID  string         `json:"resource_id,omitempty"`
	Details     map[string]any `json:"details,omitempty"`
}

// TimelineEvents returns a unified chronological view of control actions,
// incidents, freezes, maintenance windows, and operator notes.
func (d *DB) TimelineEvents(start, end string, limit int) ([]TimelineEvent, error) {
	limit = clampLimit(limit)
	clauses := []string{}
	if strings.TrimSpace(start) != "" {
		clauses = append(clauses, fmt.Sprintf("event_time >= '%s'", esc(start)))
	}
	if strings.TrimSpace(end) != "" {
		clauses = append(clauses, fmt.Sprintf("event_time <= '%s'", esc(end)))
	}
	where := ""
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}

	// Union of relevant tables projected to a common shape, sorted by time.
	// We use COALESCE and literal strings to keep this simple and safe.
	sql := fmt.Sprintf(`
SELECT event_time, event_type, event_id, summary, severity, actor_id, resource_id
FROM (
  SELECT created_at AS event_time, 'action' AS event_type, id AS event_id,
    action_type||': '||COALESCE(target_transport,'global')||' ('||COALESCE(lifecycle_state,'')||')' AS summary,
    '' AS severity, COALESCE(proposed_by,'system') AS actor_id,
    COALESCE(target_transport,'') AS resource_id
  FROM control_actions

  UNION ALL

  SELECT occurred_at AS event_time, 'incident' AS event_type, id AS event_id,
    title AS summary, severity, COALESCE(actor_id,'') AS actor_id, resource_id
  FROM incidents

  UNION ALL

  SELECT created_at AS event_time, 'freeze' AS event_type, id AS event_id,
    'freeze created: '||COALESCE(scope_type,'global')||' '||COALESCE(scope_value,'') AS summary,
    'warning' AS severity, COALESCE(created_by,'system') AS actor_id, '' AS resource_id
  FROM control_freezes

  UNION ALL

  SELECT cleared_at AS event_time, 'freeze_cleared' AS event_type, id AS event_id,
    'freeze cleared: '||COALESCE(scope_type,'global')||' '||COALESCE(scope_value,'') AS summary,
    'info' AS severity, COALESCE(cleared_by,'system') AS actor_id, '' AS resource_id
  FROM control_freezes WHERE cleared_at IS NOT NULL AND cleared_at != ''

  UNION ALL

  SELECT created_at AS event_time, 'maintenance' AS event_type, id AS event_id,
    'maintenance window: '||COALESCE(title,'') AS summary, 'info' AS severity,
    COALESCE(created_by,'system') AS actor_id, '' AS resource_id
  FROM maintenance_windows

  UNION ALL

  SELECT created_at AS event_time, 'note' AS event_type, id AS event_id,
    'note on '||ref_type||': '||SUBSTR(content,1,80) AS summary, 'info' AS severity,
    actor_id, ref_id AS resource_id
  FROM operator_notes

  UNION ALL

  SELECT event_time, event_type, id AS event_id,
    summary, severity, actor_id, resource_id
  FROM timeline_events
) AS tl
%s
ORDER BY event_time DESC LIMIT %d;`, where, limit)

	rows, err := d.QueryRows(sql)
	if err != nil {
		return nil, err
	}
	out := make([]TimelineEvent, 0, len(rows))
	for _, row := range rows {
		out = append(out, TimelineEvent{
			EventTime:  asString(row["event_time"]),
			EventType:  asString(row["event_type"]),
			EventID:    asString(row["event_id"]),
			Summary:    asString(row["summary"]),
			Severity:   asString(row["severity"]),
			ActorID:    asString(row["actor_id"]),
			ResourceID: asString(row["resource_id"]),
			Details:    map[string]any{},
		})
	}
	return out, nil
}

// ─── Explicit Timeline Event Insertion ───────────────────────────────────────

// InsertTimelineEvent inserts an explicit event into the timeline_events table.
// This is for operator-visible events that cannot be derived from other tables —
// e.g. action_approved, freeze_created, control_stale, approval_backlog_warn.
// Safe-fail: if the table does not exist yet (pre-migration), the error is
// swallowed and nil is returned so callers need not guard on migration version.
func (d *DB) InsertTimelineEvent(ev TimelineEvent) error {
	if strings.TrimSpace(ev.ID) == "" {
		return fmt.Errorf("timeline event id is required")
	}
	if ev.EventTime == "" {
		ev.EventTime = time.Now().UTC().Format(time.RFC3339)
	}
	if ev.EventType == "" {
		return fmt.Errorf("timeline event type is required")
	}
	if ev.ActorID == "" {
		ev.ActorID = "system"
	}
	detailsJSON, _ := json.Marshal(ev.Details)
	sql := fmt.Sprintf(`INSERT OR IGNORE INTO timeline_events(id,event_time,event_type,summary,severity,actor_id,resource_id,details_json)
VALUES('%s','%s','%s','%s','%s','%s','%s','%s');`,
		esc(ev.ID), esc(ev.EventTime), esc(ev.EventType), esc(ev.Summary),
		esc(ev.Severity), esc(ev.ActorID), esc(ev.ResourceID), esc(string(detailsJSON)))
	err := d.Exec(sql)
	// Treat "no such table" as safe-fail during startup before migrations run.
	if err != nil && strings.Contains(err.Error(), "no such table") {
		return nil
	}
	return err
}

// ─── Control Plane State ──────────────────────────────────────────────────────

// SetControlPlaneState updates a key-value pair in the control_plane_state table.
func (d *DB) SetControlPlaneState(key, value, updatedBy string) error {
	safeKey, err := ValidateSQLInput(key)
	if err != nil {
		logSuspiciousSQL(key, err.Error())
		return fmt.Errorf("invalid state key: %w", err)
	}
	now := esc(time.Now().UTC().Format(time.RFC3339))
	sql := fmt.Sprintf(`INSERT INTO control_plane_state(key,value_text,updated_by,updated_at)
VALUES('%s','%s','%s','%s')
ON CONFLICT(key) DO UPDATE SET value_text=excluded.value_text,updated_by=excluded.updated_by,updated_at=excluded.updated_at;`,
		safeKey, esc(value), esc(updatedBy), now)
	return d.Exec(sql)
}

// GetControlPlaneState retrieves a text value from the control_plane_state table.
func (d *DB) GetControlPlaneState(key string) (string, error) {
	safeKey, err := ValidateSQLInput(key)
	if err != nil {
		logSuspiciousSQL(key, err.Error())
		return "", fmt.Errorf("invalid state key: %w", err)
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT COALESCE(value_text,'') AS value_text FROM control_plane_state WHERE key='%s' LIMIT 1;`, safeKey))
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", nil
	}
	return asString(rows[0]["value_text"]), nil
}

// ControlPlaneStateSnapshot returns the full control plane state for the
// operator UI, combining freeze status, maintenance window status, and
// the approval backlog count.
func (d *DB) ControlPlaneStateSnapshot(now time.Time) (map[string]any, error) {
	freezes, err := d.ActiveFreezes()
	if err != nil {
		return nil, err
	}
	maintenanceWindows, err := d.ActiveMaintenanceWindows(now)
	if err != nil {
		return nil, err
	}
	pendingApproval, err := d.PendingApprovalActions(100)
	if err != nil {
		return nil, err
	}

	automationMode := "normal"
	if len(freezes) > 0 {
		automationMode = "frozen"
	} else if len(maintenanceWindows) > 0 {
		automationMode = "maintenance"
	}

	return map[string]any{
		"automation_mode":     automationMode,
		"freeze_count":        len(freezes),
		"freezes":             freezes,
		"maintenance_windows": maintenanceWindows,
		"approval_backlog":    len(pendingApproval),
		"pending_approvals":   pendingApproval,
		"snapshot_at":         now.UTC().Format(time.RFC3339),
	}, nil
}
