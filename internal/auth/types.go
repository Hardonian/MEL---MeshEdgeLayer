package auth

import (
	"time"
)

// OperatorID is a unique identifier for an operator.
// Currently implemented as a string for flexibility; can be UUID in future.
type OperatorID string

// OperatorRole defines the permission level of an operator.
// Role hierarchy (ascending permissions): viewer < responder < operator < admin
type OperatorRole string

const (
	// RoleViewer can only read/status-view - no modifications allowed
	RoleViewer OperatorRole = "viewer"
	// RoleResponder can acknowledge alerts and perform basic responses
	RoleResponder OperatorRole = "responder"
	// RoleOperator can perform control actions and configuration changes
	RoleOperator OperatorRole = "operator"
	// RoleAdmin has full access including user management and system configuration
	RoleAdmin OperatorRole = "admin"
)

// ActionClass categorizes types of actions that can be performed.
// Each action class maps to specific RBAC permissions.
type ActionClass string

const (
	// ActionRead - read-only access to status, metrics, logs
	ActionRead ActionClass = "read"
	// ActionAck - acknowledge alerts and incidents
	ActionAck ActionClass = "ack"
	// ActionSuppress - suppress notifications or noisy sources
	ActionSuppress ActionClass = "suppress"
	// ActionControl - execute control actions (restart, backoff, etc.)
	ActionControl ActionClass = "control"
	// ActionConfig - modify configuration
	ActionConfig ActionClass = "config"
	// ActionExport - export data, logs, diagnostics
	ActionExport ActionClass = "export"
)

// AuthorizationContext carries operator identity and session information
// through the request processing pipeline.
type AuthorizationContext struct {
	// OperatorID is the unique identifier of the acting operator
	OperatorID OperatorID
	// Role is the operator's current role
	Role OperatorRole
	// SessionID identifies the session (for future multi-session support)
	SessionID string
	// Timestamp when this context was created
	Timestamp time.Time
	// AuthMethod describes how the operator was authenticated
	AuthMethod string
}

// DefaultAdminContext returns an AuthorizationContext representing
// the single-operator mode (system/single-user).
// This is used when MEL operates without multi-operator authentication.
func DefaultAdminContext() AuthorizationContext {
	return AuthorizationContext{
		OperatorID:  "system",
		Role:        RoleAdmin,
		SessionID:   "single-operator-session",
		Timestamp:   time.Now().UTC(),
		AuthMethod:  "shared-credentials",
	}
}

// IsSingleOperatorMode returns true if this context represents
// the default single-operator mode (shared credentials).
func (ac AuthorizationContext) IsSingleOperatorMode() bool {
	return ac.OperatorID == "system" || ac.AuthMethod == "shared-credentials"
}

// String returns a string representation of the role
func (or OperatorRole) String() string {
	return string(or)
}

// String returns a string representation of the action class
func (ac ActionClass) String() string {
	return string(ac)
}
