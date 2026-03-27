package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ControlActionRecord struct {
	ID              string         `json:"id"`
	DecisionID      string         `json:"decision_id,omitempty"`
	ActionType      string         `json:"action_type"`
	TargetTransport string         `json:"target_transport,omitempty"`
	TargetSegment   string         `json:"target_segment,omitempty"`
	TargetNode      string         `json:"target_node,omitempty"`
	Reason          string         `json:"reason"`
	Confidence      float64        `json:"confidence"`
	TriggerEvidence []string       `json:"trigger_evidence,omitempty"`
	EpisodeID       string         `json:"episode_id,omitempty"`
	CreatedAt       string         `json:"created_at"`
	ExecutedAt      string         `json:"executed_at,omitempty"`
	CompletedAt     string         `json:"completed_at,omitempty"`
	Result          string         `json:"result,omitempty"`
	Reversible      bool           `json:"reversible"`
	ExpiresAt       string         `json:"expires_at,omitempty"`
	OutcomeDetail   string         `json:"outcome_detail,omitempty"`
	Mode            string         `json:"mode"`
	PolicyRule      string         `json:"policy_rule,omitempty"`
	LifecycleState  string         `json:"lifecycle_state,omitempty"`
	AdvisoryOnly    bool           `json:"advisory_only,omitempty"`
	DenialCode      string         `json:"denial_code,omitempty"`
	ClosureState    string         `json:"closure_state,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`

	// Trust / approval fields (migration 0017)
	ExecutionMode     string `json:"execution_mode,omitempty"`
	ProposedBy        string `json:"proposed_by,omitempty"`
	ApprovedBy        string `json:"approved_by,omitempty"`
	ApprovedAt        string `json:"approved_at,omitempty"`
	RejectedBy        string `json:"rejected_by,omitempty"`
	RejectedAt        string `json:"rejected_at,omitempty"`
	ApprovalNote      string `json:"approval_note,omitempty"`
	ApprovalExpiresAt string `json:"approval_expires_at,omitempty"`
	BlastRadiusClass  string `json:"blast_radius_class,omitempty"`
	EvidenceBundleID  string `json:"evidence_bundle_id,omitempty"`

	// Trust / SoD / linkage (migration 0021)
	SubmittedBy              string `json:"submitted_by,omitempty"`
	RequiresSeparateApprover bool   `json:"requires_separate_approver,omitempty"`
	IncidentID               string `json:"incident_id,omitempty"`
	ExecutionStartedAt       string `json:"execution_started_at,omitempty"`
	SodBypass                bool   `json:"sod_bypass,omitempty"`
	SodBypassActor           string `json:"sod_bypass_actor,omitempty"`
	SodBypassReason          string `json:"sod_bypass_reason,omitempty"`

	// Policy / approval truth (migration 0022)
	ApprovalMode                     string   `json:"approval_mode,omitempty"`
	RequiredApprovals                int      `json:"required_approvals,omitempty"`
	CollectedApprovals               int      `json:"collected_approvals,omitempty"`
	ApprovalBasis                    []string `json:"approval_basis,omitempty"`
	ApprovalPolicySource             string   `json:"approval_policy_source,omitempty"`
	HighBlastRadius                  bool     `json:"high_blast_radius,omitempty"`
	ApprovalEscalatedDueToBlastRadius bool    `json:"approval_escalated_due_to_blast_radius,omitempty"`
	ExecutionSource                  string   `json:"execution_source,omitempty"`
}

// sqlControlActionSelectList is the canonical column projection for control_actions rows.
const sqlControlActionSelectList = `id,COALESCE(decision_id,'') AS decision_id,action_type,COALESCE(target_transport,'') AS target_transport,COALESCE(target_segment,'') AS target_segment,COALESCE(target_node,'') AS target_node,reason,confidence,COALESCE(trigger_evidence_json,'[]') AS trigger_evidence_json,COALESCE(episode_id,'') AS episode_id,created_at,COALESCE(executed_at,'') AS executed_at,COALESCE(completed_at,'') AS completed_at,COALESCE(result,'') AS result,reversible,COALESCE(expires_at,'') AS expires_at,COALESCE(outcome_detail,'') AS outcome_detail,mode,COALESCE(policy_rule,'') AS policy_rule,COALESCE(lifecycle_state,'') AS lifecycle_state,COALESCE(advisory_only,0) AS advisory_only,COALESCE(denial_code,'') AS denial_code,COALESCE(closure_state,'') AS closure_state,COALESCE(metadata_json,'{}') AS metadata_json,COALESCE(execution_mode,'auto') AS execution_mode,COALESCE(proposed_by,'system') AS proposed_by,COALESCE(approved_by,'') AS approved_by,COALESCE(approved_at,'') AS approved_at,COALESCE(rejected_by,'') AS rejected_by,COALESCE(rejected_at,'') AS rejected_at,COALESCE(approval_note,'') AS approval_note,COALESCE(approval_expires_at,'') AS approval_expires_at,COALESCE(blast_radius_class,'unknown') AS blast_radius_class,COALESCE(evidence_bundle_id,'') AS evidence_bundle_id,COALESCE(submitted_by,'system') AS submitted_by,COALESCE(requires_separate_approver,0) AS requires_separate_approver,COALESCE(incident_id,'') AS incident_id,COALESCE(execution_started_at,'') AS execution_started_at,COALESCE(sod_bypass,0) AS sod_bypass,COALESCE(sod_bypass_actor,'') AS sod_bypass_actor,COALESCE(sod_bypass_reason,'') AS sod_bypass_reason,COALESCE(approval_mode,'single_approver') AS approval_mode,COALESCE(required_approvals,1) AS required_approvals,COALESCE(collected_approvals,0) AS collected_approvals,COALESCE(approval_basis_json,'[]') AS approval_basis_json,COALESCE(approval_policy_source,'mel_config.control') AS approval_policy_source,COALESCE(high_blast_radius,0) AS high_blast_radius,COALESCE(approval_escalated_due_to_blast_radius,0) AS approval_escalated_due_to_blast_radius,COALESCE(execution_source,'') AS execution_source`

type ControlDecisionRecord struct {
	ID                string         `json:"id"`
	CandidateActionID string         `json:"candidate_action_id"`
	ActionType        string         `json:"action_type"`
	TargetTransport   string         `json:"target_transport,omitempty"`
	TargetSegment     string         `json:"target_segment,omitempty"`
	Reason            string         `json:"reason"`
	Confidence        float64        `json:"confidence"`
	Allowed           bool           `json:"allowed"`
	DenialReason      string         `json:"denial_reason,omitempty"`
	DenialCode        string         `json:"denial_code,omitempty"`
	SafetyChecks      map[string]any `json:"safety_checks,omitempty"`
	DecisionInputs    map[string]any `json:"decision_inputs,omitempty"`
	PolicySummary     map[string]any `json:"policy_summary,omitempty"`
	CreatedAt         string         `json:"created_at"`
	Mode              string         `json:"mode"`
	OperatorOverride  bool           `json:"operator_override"`
}

type ControlActionRealityRecord struct {
	ActionType         string `json:"action_type"`
	ActuatorExists     bool   `json:"actuator_exists"`
	Reversible         bool   `json:"reversible"`
	BlastRadiusKnown   bool   `json:"blast_radius_known"`
	BlastRadiusClass   string `json:"blast_radius_class"`
	SafeForGuardedAuto bool   `json:"safe_for_guarded_auto"`
	AdvisoryOnly       bool   `json:"advisory_only"`
	DenialCode         string `json:"denial_code,omitempty"`
	Notes              string `json:"notes,omitempty"`
	UpdatedAt          string `json:"updated_at,omitempty"`
}

func (d *DB) UpsertControlAction(action ControlActionRecord) error {
	if strings.TrimSpace(action.ID) == "" {
		return fmt.Errorf("control action id is required")
	}
	if action.CreatedAt == "" {
		action.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	triggerJSON, _ := json.Marshal(action.TriggerEvidence)
	metadataJSON, _ := json.Marshal(action.Metadata)
	basisJSON, _ := json.Marshal(action.ApprovalBasis)
	if action.ExecutionMode == "" {
		action.ExecutionMode = "auto"
	}
	if action.ProposedBy == "" {
		action.ProposedBy = "system"
	}
	if action.SubmittedBy == "" {
		action.SubmittedBy = action.ProposedBy
	}
	if action.BlastRadiusClass == "" {
		action.BlastRadiusClass = "unknown"
	}
	if action.ApprovalMode == "" {
		action.ApprovalMode = "single_approver"
	}
	if action.RequiredApprovals <= 0 {
		action.RequiredApprovals = 1
	}
	if action.ApprovalPolicySource == "" {
		action.ApprovalPolicySource = "mel_config.control"
	}
	if strings.TrimSpace(action.ExecutionSource) == "" {
		// NOT NULL column; sqlString("") becomes NULL — use explicit sentinel.
		action.ExecutionSource = "unspecified"
	}
	sql := fmt.Sprintf(`INSERT INTO control_actions(id,decision_id,action_type,target_transport,target_segment,target_node,reason,confidence,trigger_evidence_json,episode_id,created_at,executed_at,completed_at,result,reversible,expires_at,outcome_detail,mode,policy_rule,lifecycle_state,advisory_only,denial_code,closure_state,metadata_json,execution_mode,proposed_by,approved_by,approved_at,rejected_by,rejected_at,approval_note,approval_expires_at,blast_radius_class,evidence_bundle_id,submitted_by,requires_separate_approver,incident_id,execution_started_at,sod_bypass,sod_bypass_actor,sod_bypass_reason,approval_mode,required_approvals,collected_approvals,approval_basis_json,approval_policy_source,high_blast_radius,approval_escalated_due_to_blast_radius,execution_source)
VALUES('%s',%s,'%s',%s,%s,%s,'%s',%f,'%s',%s,'%s',%s,%s,%s,%d,%s,%s,'%s','%s','%s',%d,%s,%s,'%s','%s','%s',%s,%s,%s,%s,%s,%s,'%s',%s,'%s',%d,%s,%s,%d,%s,%s,'%s',%d,%d,'%s','%s',%d,%d,%s)
ON CONFLICT(id) DO UPDATE SET decision_id=excluded.decision_id,action_type=excluded.action_type,target_transport=excluded.target_transport,target_segment=excluded.target_segment,target_node=excluded.target_node,reason=excluded.reason,confidence=excluded.confidence,trigger_evidence_json=excluded.trigger_evidence_json,episode_id=excluded.episode_id,created_at=excluded.created_at,executed_at=excluded.executed_at,completed_at=excluded.completed_at,result=excluded.result,reversible=excluded.reversible,expires_at=excluded.expires_at,outcome_detail=excluded.outcome_detail,mode=excluded.mode,policy_rule=excluded.policy_rule,lifecycle_state=excluded.lifecycle_state,advisory_only=excluded.advisory_only,denial_code=excluded.denial_code,closure_state=excluded.closure_state,metadata_json=excluded.metadata_json,execution_mode=excluded.execution_mode,proposed_by=excluded.proposed_by,approved_by=excluded.approved_by,approved_at=excluded.approved_at,rejected_by=excluded.rejected_by,rejected_at=excluded.rejected_at,approval_note=excluded.approval_note,approval_expires_at=excluded.approval_expires_at,blast_radius_class=excluded.blast_radius_class,evidence_bundle_id=excluded.evidence_bundle_id,submitted_by=excluded.submitted_by,requires_separate_approver=excluded.requires_separate_approver,incident_id=excluded.incident_id,execution_started_at=excluded.execution_started_at,sod_bypass=excluded.sod_bypass,sod_bypass_actor=excluded.sod_bypass_actor,sod_bypass_reason=excluded.sod_bypass_reason,approval_mode=excluded.approval_mode,required_approvals=excluded.required_approvals,collected_approvals=excluded.collected_approvals,approval_basis_json=excluded.approval_basis_json,approval_policy_source=excluded.approval_policy_source,high_blast_radius=excluded.high_blast_radius,approval_escalated_due_to_blast_radius=excluded.approval_escalated_due_to_blast_radius,execution_source=excluded.execution_source;`,
		esc(action.ID), sqlString(action.DecisionID), esc(action.ActionType), sqlString(action.TargetTransport), sqlString(action.TargetSegment), sqlString(action.TargetNode), esc(action.Reason), action.Confidence, esc(string(triggerJSON)), sqlString(action.EpisodeID), esc(action.CreatedAt), sqlString(action.ExecutedAt), sqlString(action.CompletedAt), sqlString(action.Result), boolInt(action.Reversible), sqlString(action.ExpiresAt), sqlString(action.OutcomeDetail), esc(action.Mode), esc(action.PolicyRule), esc(action.LifecycleState), boolInt(action.AdvisoryOnly), sqlString(action.DenialCode), sqlString(action.ClosureState), esc(string(metadataJSON)),
		esc(action.ExecutionMode), esc(action.ProposedBy), sqlString(action.ApprovedBy), sqlString(action.ApprovedAt), sqlString(action.RejectedBy), sqlString(action.RejectedAt), sqlString(action.ApprovalNote), sqlString(action.ApprovalExpiresAt), esc(action.BlastRadiusClass), sqlString(action.EvidenceBundleID),
		esc(action.SubmittedBy), boolInt(action.RequiresSeparateApprover), sqlString(action.IncidentID), sqlString(action.ExecutionStartedAt), boolInt(action.SodBypass), sqlString(action.SodBypassActor), sqlString(action.SodBypassReason),
		esc(action.ApprovalMode), action.RequiredApprovals, action.CollectedApprovals, esc(string(basisJSON)), esc(action.ApprovalPolicySource), boolInt(action.HighBlastRadius), boolInt(action.ApprovalEscalatedDueToBlastRadius), sqlString(action.ExecutionSource))
	return d.Exec(sql)
}

func (d *DB) UpsertControlDecision(decision ControlDecisionRecord) error {
	if strings.TrimSpace(decision.ID) == "" {
		return fmt.Errorf("control decision id is required")
	}
	if decision.CreatedAt == "" {
		decision.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	safetyJSON, _ := json.Marshal(decision.SafetyChecks)
	inputJSON, _ := json.Marshal(decision.DecisionInputs)
	policyJSON, _ := json.Marshal(decision.PolicySummary)
	sql := fmt.Sprintf(`INSERT INTO control_decisions(id,candidate_action_id,action_type,target_transport,target_segment,reason,confidence,allowed,denial_reason,denial_code,safety_checks_json,decision_inputs_json,policy_summary_json,created_at,mode,operator_override)
VALUES('%s','%s','%s',%s,%s,'%s',%f,%d,%s,%s,'%s','%s','%s','%s','%s',%d)
ON CONFLICT(id) DO UPDATE SET candidate_action_id=excluded.candidate_action_id,action_type=excluded.action_type,target_transport=excluded.target_transport,target_segment=excluded.target_segment,reason=excluded.reason,confidence=excluded.confidence,allowed=excluded.allowed,denial_reason=excluded.denial_reason,denial_code=excluded.denial_code,safety_checks_json=excluded.safety_checks_json,decision_inputs_json=excluded.decision_inputs_json,policy_summary_json=excluded.policy_summary_json,created_at=excluded.created_at,mode=excluded.mode,operator_override=excluded.operator_override;`,
		esc(decision.ID), esc(decision.CandidateActionID), esc(decision.ActionType), sqlString(decision.TargetTransport), sqlString(decision.TargetSegment), esc(decision.Reason), decision.Confidence, boolInt(decision.Allowed), sqlString(decision.DenialReason), sqlString(decision.DenialCode), esc(string(safetyJSON)), esc(string(inputJSON)), esc(string(policyJSON)), esc(decision.CreatedAt), esc(decision.Mode), boolInt(decision.OperatorOverride))
	return d.Exec(sql)
}

func (d *DB) UpsertControlActionReality(record ControlActionRealityRecord) error {
	if strings.TrimSpace(record.ActionType) == "" {
		return fmt.Errorf("control action type is required")
	}
	if record.UpdatedAt == "" {
		record.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	sql := fmt.Sprintf(`INSERT INTO control_action_reality(action_type,actuator_exists,reversible,blast_radius_known,blast_radius_class,safe_for_guarded_auto,advisory_only,denial_code,notes,updated_at)
VALUES('%s',%d,%d,%d,'%s',%d,%d,%s,%s,'%s')
ON CONFLICT(action_type) DO UPDATE SET actuator_exists=excluded.actuator_exists,reversible=excluded.reversible,blast_radius_known=excluded.blast_radius_known,blast_radius_class=excluded.blast_radius_class,safe_for_guarded_auto=excluded.safe_for_guarded_auto,advisory_only=excluded.advisory_only,denial_code=excluded.denial_code,notes=excluded.notes,updated_at=excluded.updated_at;`,
		esc(record.ActionType), boolInt(record.ActuatorExists), boolInt(record.Reversible), boolInt(record.BlastRadiusKnown), esc(record.BlastRadiusClass), boolInt(record.SafeForGuardedAuto), boolInt(record.AdvisoryOnly), sqlString(record.DenialCode), sqlString(record.Notes), esc(record.UpdatedAt))
	return d.Exec(sql)
}

// ControlActionRealityByType returns the reality matrix row for an action type, if present.
func (d *DB) ControlActionRealityByType(actionType string) (ControlActionRealityRecord, bool, error) {
	if strings.TrimSpace(actionType) == "" {
		return ControlActionRealityRecord{}, false, nil
	}
	safe := esc(actionType)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT action_type, actuator_exists, reversible, blast_radius_known, COALESCE(blast_radius_class,'') AS blast_radius_class,
COALESCE(safe_for_guarded_auto,0) AS safe_for_guarded_auto, COALESCE(advisory_only,0) AS advisory_only, COALESCE(denial_code,'') AS denial_code,
COALESCE(notes,'') AS notes, COALESCE(updated_at,'') AS updated_at FROM control_action_reality WHERE action_type='%s' LIMIT 1;`, safe))
	if err != nil {
		return ControlActionRealityRecord{}, false, err
	}
	if len(rows) == 0 {
		return ControlActionRealityRecord{}, false, nil
	}
	row := rows[0]
	return ControlActionRealityRecord{
		ActionType:         asString(row["action_type"]),
		ActuatorExists:     asInt(row["actuator_exists"]) == 1,
		Reversible:         asInt(row["reversible"]) == 1,
		BlastRadiusKnown:   asInt(row["blast_radius_known"]) == 1,
		BlastRadiusClass:   asString(row["blast_radius_class"]),
		SafeForGuardedAuto: asInt(row["safe_for_guarded_auto"]) == 1,
		AdvisoryOnly:       asInt(row["advisory_only"]) == 1,
		DenialCode:         asString(row["denial_code"]),
		Notes:              asString(row["notes"]),
		UpdatedAt:          asString(row["updated_at"]),
	}, true, nil
}

func (d *DB) ControlActionRealities() ([]ControlActionRealityRecord, error) {
	rows, err := d.QueryRows(`SELECT action_type, actuator_exists, reversible, blast_radius_known, COALESCE(blast_radius_class,'') AS blast_radius_class,
COALESCE(safe_for_guarded_auto,0) AS safe_for_guarded_auto, COALESCE(advisory_only,0) AS advisory_only, COALESCE(denial_code,'') AS denial_code,
COALESCE(notes,'') AS notes, COALESCE(updated_at,'') AS updated_at FROM control_action_reality ORDER BY action_type;`)
	if err != nil {
		return nil, err
	}
	out := make([]ControlActionRealityRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, ControlActionRealityRecord{
			ActionType:         asString(row["action_type"]),
			ActuatorExists:     asInt(row["actuator_exists"]) == 1,
			Reversible:         asInt(row["reversible"]) == 1,
			BlastRadiusKnown:   asInt(row["blast_radius_known"]) == 1,
			BlastRadiusClass:   asString(row["blast_radius_class"]),
			SafeForGuardedAuto: asInt(row["safe_for_guarded_auto"]) == 1,
			AdvisoryOnly:       asInt(row["advisory_only"]) == 1,
			DenialCode:         asString(row["denial_code"]),
			Notes:              asString(row["notes"]),
			UpdatedAt:          asString(row["updated_at"]),
		})
	}
	return out, nil
}

func (d *DB) ControlActions(transportName, actionType, start, end, lifecycleState string, limit, offset int) ([]ControlActionRecord, error) {
	limit = clampLimit(limit)
	if offset < 0 {
		offset = 0
	}
	clauses := []string{"1=1"}
	if strings.TrimSpace(transportName) != "" {
		clauses = append(clauses, fmt.Sprintf("target_transport='%s'", esc(transportName)))
	}
	if strings.TrimSpace(actionType) != "" {
		clauses = append(clauses, fmt.Sprintf("action_type='%s'", esc(actionType)))
	}
	if strings.TrimSpace(start) != "" {
		clauses = append(clauses, fmt.Sprintf("created_at >= '%s'", esc(start)))
	}
	if strings.TrimSpace(end) != "" {
		clauses = append(clauses, fmt.Sprintf("created_at <= '%s'", esc(end)))
	}
	if ls := strings.TrimSpace(lifecycleState); ls != "" {
		clauses = append(clauses, fmt.Sprintf("lifecycle_state='%s'", esc(ls)))
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT %s
FROM control_actions WHERE %s ORDER BY created_at DESC LIMIT %d OFFSET %d;`, sqlControlActionSelectList, strings.Join(clauses, " AND "), limit, offset))
	if err != nil {
		return nil, err
	}
	out := make([]ControlActionRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, controlActionFromRow(row))
	}
	return out, nil
}

// ControlActionsByIncidentID returns control actions linked to the given incident (canonical incident_id).
// ControlActionsForIncidentIDs returns control actions whose incident_id is in ids (batch fetch for API list enrichment).
func (d *DB) ControlActionsForIncidentIDs(ids []string, maxRows int) ([]ControlActionRecord, error) {
	if maxRows <= 0 {
		maxRows = 500
	}
	if maxRows > 2000 {
		maxRows = 2000
	}
	var filtered []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, err := ValidateSQLInput(id); err != nil {
			continue
		}
		filtered = append(filtered, id)
	}
	if len(filtered) == 0 {
		return nil, nil
	}
	quoted := make([]string, 0, len(filtered))
	for _, id := range filtered {
		quoted = append(quoted, "'"+esc(id)+"'")
	}
	inList := strings.Join(quoted, ",")
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT %s
FROM control_actions WHERE incident_id IN (%s) ORDER BY created_at DESC LIMIT %d;`,
		sqlControlActionSelectList, inList, maxRows))
	if err != nil {
		return nil, err
	}
	out := make([]ControlActionRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, controlActionFromRow(row))
	}
	return out, nil
}

func (d *DB) ControlActionsByIncidentID(incidentID string, limit int) ([]ControlActionRecord, error) {
	limit = clampLimit(limit)
	if strings.TrimSpace(incidentID) == "" {
		return nil, nil
	}
	safeID, err := ValidateSQLInput(incidentID)
	if err != nil {
		logSuspiciousSQL(incidentID, err.Error())
		return nil, fmt.Errorf("invalid incident id: %w", err)
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT %s
FROM control_actions WHERE incident_id='%s' ORDER BY created_at DESC LIMIT %d;`, sqlControlActionSelectList, safeID, limit))
	if err != nil {
		return nil, err
	}
	out := make([]ControlActionRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, controlActionFromRow(row))
	}
	return out, nil
}

func (d *DB) ControlDecisions(transportName, actionType, start, end string, limit, offset int) ([]ControlDecisionRecord, error) {
	limit = clampLimit(limit)
	if offset < 0 {
		offset = 0
	}
	clauses := []string{"1=1"}
	if strings.TrimSpace(transportName) != "" {
		clauses = append(clauses, fmt.Sprintf("target_transport='%s'", esc(transportName)))
	}
	if strings.TrimSpace(actionType) != "" {
		clauses = append(clauses, fmt.Sprintf("action_type='%s'", esc(actionType)))
	}
	if strings.TrimSpace(start) != "" {
		clauses = append(clauses, fmt.Sprintf("created_at >= '%s'", esc(start)))
	}
	if strings.TrimSpace(end) != "" {
		clauses = append(clauses, fmt.Sprintf("created_at <= '%s'", esc(end)))
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id, candidate_action_id, action_type, COALESCE(target_transport,'') AS target_transport, COALESCE(target_segment,'') AS target_segment, reason, confidence, allowed, COALESCE(denial_reason,'') AS denial_reason, COALESCE(denial_code,'') AS denial_code, COALESCE(safety_checks_json,'{}') AS safety_checks_json, COALESCE(decision_inputs_json,'{}') AS decision_inputs_json, COALESCE(policy_summary_json,'{}') AS policy_summary_json, created_at, mode, operator_override
FROM control_decisions WHERE %s ORDER BY created_at DESC LIMIT %d OFFSET %d;`, strings.Join(clauses, " AND "), limit, offset))
	if err != nil {
		return nil, err
	}
	out := make([]ControlDecisionRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, controlDecisionFromRow(row))
	}
	return out, nil
}

func (d *DB) ControlActionByID(id string) (ControlActionRecord, bool, error) {
	if strings.TrimSpace(id) == "" {
		return ControlActionRecord{}, false, nil
	}
	safeID, err := ValidateSQLInput(id)
	if err != nil {
		logSuspiciousSQL(id, err.Error())
		return ControlActionRecord{}, false, fmt.Errorf("invalid id: %w", err)
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT %s
FROM control_actions WHERE id='%s' LIMIT 1;`, sqlControlActionSelectList, safeID))
	if err != nil {
		return ControlActionRecord{}, false, err
	}
	if len(rows) == 0 {
		return ControlActionRecord{}, false, nil
	}
	return controlActionFromRow(rows[0]), true, nil
}

func (d *DB) IncompleteControlActions(limit int) ([]ControlActionRecord, error) {
	limit = clampLimit(limit)
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT %s
FROM control_actions WHERE lifecycle_state IN ('pending','running') ORDER BY created_at ASC LIMIT %d;`, sqlControlActionSelectList, limit))
	if err != nil {
		return nil, err
	}
	out := make([]ControlActionRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, controlActionFromRow(row))
	}
	return out, nil
}

func controlActionFromRow(row map[string]any) ControlActionRecord {
	record := ControlActionRecord{
		ID:              asString(row["id"]),
		DecisionID:      asString(row["decision_id"]),
		ActionType:      asString(row["action_type"]),
		TargetTransport: asString(row["target_transport"]),
		TargetSegment:   asString(row["target_segment"]),
		TargetNode:      asString(row["target_node"]),
		Reason:          asString(row["reason"]),
		Confidence:      asFloat(row["confidence"]),
		EpisodeID:       asString(row["episode_id"]),
		CreatedAt:       asString(row["created_at"]),
		ExecutedAt:      asString(row["executed_at"]),
		CompletedAt:     asString(row["completed_at"]),
		Result:          asString(row["result"]),
		Reversible:      asInt(row["reversible"]) == 1,
		ExpiresAt:       asString(row["expires_at"]),
		OutcomeDetail:   asString(row["outcome_detail"]),
		Mode:            asString(row["mode"]),
		PolicyRule:      asString(row["policy_rule"]),
		LifecycleState:  asString(row["lifecycle_state"]),
		AdvisoryOnly:    asInt(row["advisory_only"]) == 1,
		DenialCode:      asString(row["denial_code"]),
		ClosureState:    asString(row["closure_state"]),
		Metadata:        map[string]any{},
	}
	_ = json.Unmarshal([]byte(asString(row["trigger_evidence_json"])), &record.TriggerEvidence)
	_ = json.Unmarshal([]byte(asString(row["metadata_json"])), &record.Metadata)
	// Trust fields (may be absent for rows predating migration 0017)
	record.ExecutionMode = asString(row["execution_mode"])
	record.ProposedBy = asString(row["proposed_by"])
	record.ApprovedBy = asString(row["approved_by"])
	record.ApprovedAt = asString(row["approved_at"])
	record.RejectedBy = asString(row["rejected_by"])
	record.RejectedAt = asString(row["rejected_at"])
	record.ApprovalNote = asString(row["approval_note"])
	record.ApprovalExpiresAt = asString(row["approval_expires_at"])
	record.BlastRadiusClass = asString(row["blast_radius_class"])
	record.EvidenceBundleID = asString(row["evidence_bundle_id"])
	record.SubmittedBy = asString(row["submitted_by"])
	record.RequiresSeparateApprover = asInt(row["requires_separate_approver"]) == 1
	record.IncidentID = asString(row["incident_id"])
	record.ExecutionStartedAt = asString(row["execution_started_at"])
	record.SodBypass = asInt(row["sod_bypass"]) == 1
	record.SodBypassActor = asString(row["sod_bypass_actor"])
	record.SodBypassReason = asString(row["sod_bypass_reason"])
	record.ApprovalMode = asString(row["approval_mode"])
	record.RequiredApprovals = int(asInt(row["required_approvals"]))
	record.CollectedApprovals = int(asInt(row["collected_approvals"]))
	_ = json.Unmarshal([]byte(asString(row["approval_basis_json"])), &record.ApprovalBasis)
	record.ApprovalPolicySource = asString(row["approval_policy_source"])
	record.HighBlastRadius = asInt(row["high_blast_radius"]) == 1
	record.ApprovalEscalatedDueToBlastRadius = asInt(row["approval_escalated_due_to_blast_radius"]) == 1
	record.ExecutionSource = asString(row["execution_source"])
	if record.ExecutionMode == "" {
		record.ExecutionMode = "auto"
	}
	if record.BlastRadiusClass == "" {
		record.BlastRadiusClass = "unknown"
	}
	if record.SubmittedBy == "" {
		record.SubmittedBy = "system"
	}
	if record.ApprovalMode == "" {
		record.ApprovalMode = "single_approver"
	}
	if record.RequiredApprovals <= 0 {
		record.RequiredApprovals = 1
	}
	if record.ApprovalPolicySource == "" {
		record.ApprovalPolicySource = "mel_config.control"
	}
	return record
}

func controlDecisionFromRow(row map[string]any) ControlDecisionRecord {
	record := ControlDecisionRecord{
		ID:                asString(row["id"]),
		CandidateActionID: asString(row["candidate_action_id"]),
		ActionType:        asString(row["action_type"]),
		TargetTransport:   asString(row["target_transport"]),
		TargetSegment:     asString(row["target_segment"]),
		Reason:            asString(row["reason"]),
		Confidence:        asFloat(row["confidence"]),
		Allowed:           asInt(row["allowed"]) == 1,
		DenialReason:      asString(row["denial_reason"]),
		DenialCode:        asString(row["denial_code"]),
		SafetyChecks:      map[string]any{},
		DecisionInputs:    map[string]any{},
		PolicySummary:     map[string]any{},
		CreatedAt:         asString(row["created_at"]),
		Mode:              asString(row["mode"]),
		OperatorOverride:  asInt(row["operator_override"]) == 1,
	}
	_ = json.Unmarshal([]byte(asString(row["safety_checks_json"])), &record.SafetyChecks)
	_ = json.Unmarshal([]byte(asString(row["decision_inputs_json"])), &record.DecisionInputs)
	_ = json.Unmarshal([]byte(asString(row["policy_summary_json"])), &record.PolicySummary)
	return record
}

func copyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// ControlDecisionByID returns the control decision with the given ID, or
// (zero, false, nil) if not found.
func (d *DB) ControlDecisionByID(id string) (ControlDecisionRecord, bool, error) {
	if strings.TrimSpace(id) == "" {
		return ControlDecisionRecord{}, false, nil
	}
	safeID, err := ValidateSQLInput(id)
	if err != nil {
		logSuspiciousSQL(id, err.Error())
		return ControlDecisionRecord{}, false, fmt.Errorf("invalid id: %w", err)
	}
	rows, err := d.QueryRows(fmt.Sprintf(`SELECT id, candidate_action_id, action_type,
COALESCE(target_transport,'') AS target_transport, COALESCE(target_segment,'') AS target_segment,
reason, confidence, allowed,
COALESCE(denial_reason,'') AS denial_reason, COALESCE(denial_code,'') AS denial_code,
COALESCE(safety_checks_json,'{}') AS safety_checks_json,
COALESCE(decision_inputs_json,'{}') AS decision_inputs_json,
COALESCE(policy_summary_json,'{}') AS policy_summary_json,
created_at, mode, operator_override
FROM control_decisions WHERE id='%s' LIMIT 1;`, safeID))
	if err != nil {
		return ControlDecisionRecord{}, false, err
	}
	if len(rows) == 0 {
		return ControlDecisionRecord{}, false, nil
	}
	return controlDecisionFromRow(rows[0]), true, nil
}
