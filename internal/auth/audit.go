package auth

import (
	"encoding/json"
	"time"
)

// AuditResult represents the outcome of an auditable action.
type AuditResult string

const (
	// AuditResultSuccess indicates the action was completed successfully
	AuditResultSuccess AuditResult = "success"
	// AuditResultDenied indicates the action was denied by policy/permissions
	AuditResultDenied AuditResult = "denied"
	// AuditResultFailed indicates the action failed due to an error
	AuditResultFailed AuditResult = "failed"
	// AuditResultPartial indicates the action partially succeeded
	AuditResultPartial AuditResult = "partial"
)

// AuditEntry represents a single audit log entry for action attribution.
// All actions that modify system state should be logged with this structure.
type AuditEntry struct {
	// ID is the unique identifier for this audit entry
	ID string `json:"id"`
	// ActorID is the operator who performed the action
	ActorID OperatorID `json:"actor_id"`
	// ActionClass is the category of action performed
	ActionClass ActionClass `json:"action_class"`
	// ActionDetail is the specific action (e.g., "restart_transport")
	ActionDetail string `json:"action_detail"`
	// ResourceType is the type of resource being acted upon
	ResourceType string `json:"resource_type"`
	// ResourceID is the identifier of the specific resource
	ResourceID string `json:"resource_id,omitempty"`
	// Reason is the operator-provided or system-generated reason
	Reason string `json:"reason"`
	// Result indicates the outcome of the action
	Result AuditResult `json:"result"`
	// Details contains additional JSON-encoded details
	Details map[string]any `json:"details,omitempty"`
	// Timestamp when the action occurred
	Timestamp time.Time `json:"timestamp"`
	// SessionID identifies the session (for future multi-session support)
	SessionID string `json:"session_id,omitempty"`
	// RemoteAddr is the client address (for network actions)
	RemoteAddr string `json:"remote_addr,omitempty"`
}

// AuditEntryParams is a convenience struct for creating audit entries.
type AuditEntryParams struct {
	ActorID      OperatorID
	ActionClass  ActionClass
	ActionDetail string
	ResourceType string
	ResourceID   string
	Reason       string
	Result       AuditResult
	Details      map[string]any
	SessionID    string
	RemoteAddr   string
}

// NewAuditEntry creates a new AuditEntry with the current timestamp.
func NewAuditEntry(params AuditEntryParams) AuditEntry {
	return AuditEntry{
		ID:           generateAuditID(),
		ActorID:      params.ActorID,
		ActionClass:  params.ActionClass,
		ActionDetail: params.ActionDetail,
		ResourceType: params.ResourceType,
		ResourceID:   params.ResourceID,
		Reason:       params.Reason,
		Result:       params.Result,
		Details:      params.Details,
		Timestamp:    time.Now().UTC(),
		SessionID:    params.SessionID,
		RemoteAddr:   params.RemoteAddr,
	}
}

// NewAuditEntryFromContext creates a new AuditEntry using an AuthorizationContext.
func NewAuditEntryFromContext(ctx AuthorizationContext, params AuditEntryParams) AuditEntry {
	entry := NewAuditEntry(params)
	entry.ActorID = ctx.OperatorID
	entry.SessionID = ctx.SessionID
	return entry
}

// ToJSON serializes the audit entry to JSON.
func (e AuditEntry) ToJSON() (string, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ToMap converts the audit entry to a map for database insertion.
func (e AuditEntry) ToMap() map[string]any {
	return map[string]any{
		"id":            e.ID,
		"actor_id":      string(e.ActorID),
		"action_class": string(e.ActionClass),
		"action_detail": e.ActionDetail,
		"resource_type": e.ResourceType,
		"resource_id":   e.ResourceID,
		"reason":        e.Reason,
		"result":        string(e.Result),
		"details":       e.Details,
		"timestamp":     e.Timestamp.Format(time.RFC3339),
		"session_id":    e.SessionID,
		"remote_addr":   e.RemoteAddr,
	}
}

// LogControlAction creates an audit entry for a control action execution.
func LogControlAction(ctx AuthorizationContext, actionType, resourceType, resourceID, reason string, result AuditResult, details map[string]any) AuditEntry {
	return NewAuditEntryFromContext(ctx, AuditEntryParams{
		ActionClass:  ActionControl,
		ActionDetail: actionType,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Reason:       reason,
		Result:       result,
		Details:      details,
	})
}

// LogConfigAction creates an audit entry for a configuration change.
func LogConfigAction(ctx AuthorizationContext, configKey, reason string, result AuditResult, details map[string]any) AuditEntry {
	return NewAuditEntryFromContext(ctx, AuditEntryParams{
		ActionClass:  ActionConfig,
		ActionDetail: "config_change",
		ResourceType: "configuration",
		ResourceID:   configKey,
		Reason:       reason,
		Result:       result,
		Details:      details,
	})
}

// LogDeniedAction creates an audit entry for a denied action attempt.
func LogDeniedAction(ctx AuthorizationContext, actionClass ActionClass, actionDetail, resourceType, resourceID, reason string) AuditEntry {
	return NewAuditEntryFromContext(ctx, AuditEntryParams{
		ActionClass:  actionClass,
		ActionDetail: actionDetail,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Reason:       reason,
		Result:       AuditResultDenied,
		Details:      nil,
	})
}

// LogExportAction creates an audit entry for a data export.
func LogExportAction(ctx AuthorizationContext, exportType, reason string, result AuditResult, details map[string]any) AuditEntry {
	return NewAuditEntryFromContext(ctx, AuditEntryParams{
		ActionClass:  ActionExport,
		ActionDetail: exportType,
		ResourceType: "export",
		Reason:       reason,
		Result:       result,
		Details:      details,
	})
}

// generateAuditID generates a unique ID for an audit entry.
func generateAuditID() string {
	// Simple ID generation - in production, use UUID
	return time.Now().UTC().Format("20060102150405.000000")
}
