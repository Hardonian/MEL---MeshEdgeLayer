package auth

// RoleHierarchy defines the role hierarchy for permission escalation.
// Higher roles inherit all permissions from lower roles.
var RoleHierarchy = map[OperatorRole]int{
	RoleViewer:    0,
	RoleResponder: 1,
	RoleOperator:  2,
	RoleAdmin:     3,
}

// RolePermissions defines which action classes are allowed for each role.
// This matrix is currently ADVISORY ONLY in single-operator mode.
// Full RBAC enforcement requires future multi-operator support.
var RolePermissions = map[OperatorRole][]ActionClass{
	RoleViewer: {
		ActionRead,
	},
	RoleResponder: {
		ActionRead,
		ActionAck,
	},
	RoleOperator: {
		ActionRead,
		ActionAck,
		ActionSuppress,
		ActionControl,
		ActionExport,
	},
	RoleAdmin: {
		ActionRead,
		ActionAck,
		ActionSuppress,
		ActionControl,
		ActionConfig,
		ActionExport,
	},
}

// CanPerform checks if a given role has permission to perform an action class.
// Returns true if the role is allowed to perform the action.
//
// IMPORTANT: This function is currently advisory only in single-operator mode.
// It will always return true when operating in single-operator mode (shared credentials).
// Full RBAC enforcement requires future implementation of multi-operator authentication.
func CanPerform(role OperatorRole, actionClass ActionClass) bool {
	// In single-operator/admin mode, allow all actions
	// This is the current behavior - RBAC is advisory only
	if role == RoleAdmin {
		return true
	}

	allowedActions, exists := RolePermissions[role]
	if !exists {
		return false
	}

	for _, allowed := range allowedActions {
		if allowed == actionClass {
			return true
		}
	}

	return false
}

// CanPerformWithContext checks if the authorization context has permission
// to perform the given action class.
// This is the main entry point for RBAC checks.
func CanPerformWithContext(ctx AuthorizationContext, actionClass ActionClass) bool {
	// Single-operator mode: allow all actions (advisory RBAC)
	if ctx.IsSingleOperatorMode() {
		return true
	}

	return CanPerform(ctx.Role, actionClass)
}

// RoleAllowsControl checks if a role allows control actions.
func RoleAllowsControl(role OperatorRole) bool {
	return CanPerform(role, ActionControl)
}

// RoleAllowsConfig checks if a role allows configuration changes.
func RoleAllowsConfig(role OperatorRole) bool {
	return CanPerform(role, ActionConfig)
}

// RoleAllowsExport checks if a role allows data export.
func RoleAllowsExport(role OperatorRole) bool {
	return CanPerform(role, ActionExport)
}

// RoleCanAcknowledge checks if a role can acknowledge alerts.
func RoleCanAcknowledge(role OperatorRole) bool {
	return CanPerform(role, ActionAck)
}

// RoleCanSuppress checks if a role can suppress notifications/sources.
func RoleCanSuppress(role OperatorRole) bool {
	return CanPerform(role, ActionSuppress)
}

// GetAllowedActions returns all action classes allowed for a given role.
func GetAllowedActions(role OperatorRole) []ActionClass {
	actions, exists := RolePermissions[role]
	if !exists {
		return []ActionClass{}
	}
	result := make([]ActionClass, len(actions))
	copy(result, actions)
	return result
}

// DeniedActionReason provides a structured reason when an action is denied.
type DeniedActionReason struct {
	Reason   string
	Role     OperatorRole
	Action   ActionClass
	Details  string
}

// NewDeniedActionReason creates a new denied action reason.
func NewDeniedActionReason(role OperatorRole, action ActionClass, details string) DeniedActionReason {
	return DeniedActionReason{
		Reason:  "action_not_permitted_by_role",
		Role:    role,
		Action:  action,
		Details: details,
	}
}