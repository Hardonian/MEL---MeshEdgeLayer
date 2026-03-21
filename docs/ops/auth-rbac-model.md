# Auth/RBAC Model Documentation

> **⚠️ IMPORTANT: MEL currently operates in single-operator mode with shared credentials.**
> Full RBAC enforcement requires future work on multi-operator authentication.

## Overview

This document describes the authentication and authorization framework for MEL (MeshEdgeLayer). The current implementation provides foundational scaffolding for multi-operator support while operating in single-operator mode.

## Current Limitations

MEL currently operates in **single-operator mode** with the following characteristics:

- **Shared Credentials**: All operators use the same API key or Basic Auth credentials
- **Single Identity**: All actions are attributed to "system" operator
- **Admin-Only Access**: No role-based restrictions are enforced
- **No Session Management**: operator_sessions table exists but is not actively used

### Explicit Limitations

| Feature | Current Status | Future Work |
|---------|---------------|-------------|
| Multi-operator auth | Not implemented | Required |
| Role-based access control | Advisory only | Required |
| Session tracking | Table exists, unused | Required |
| JWT/OAuth | Not implemented | Future consideration |
| Per-operator audit attribution | Not implemented | Required |

## Role Hierarchy

MEL defines four operator roles with escalating permissions:

```
RoleViewer    → RoleResponder → RoleOperator → RoleAdmin
    (0)            (1)            (2)            (3)
```

### Role Permissions Matrix

| Action Class | Viewer | Responder | Operator | Admin |
|--------------|--------|-----------|----------|-------|
| read         | ✓      | ✓         | ✓        | ✓     |
| ack          |        | ✓         | ✓        | ✓     |
| suppress     |        |           | ✓        | ✓     |
| control      |        |           | ✓        | ✓     |
| config       |        |           |          | ✓     |
| export       |        |           | ✓        | ✓     |

## Action Classification

Actions are classified into the following categories:

### ActionRead
- View status and metrics
- Access dashboard and reports
- Query node information
- View transport health

### ActionAck
- Acknowledge alerts
- Mark incidents as reviewed
- Clear notifications

### ActionSuppress
- Suppress noisy message sources
- Temporarily disable alerts

### ActionControl
- Restart transports
- Adjust backoff settings
- Trigger health rechecks
- Execute control actions

### ActionConfig
- Modify configuration
- Change system settings
- Update policies

### ActionExport
- Export logs
- Download diagnostics
- Generate reports

## Audit Trail Structure

The `audit_log` table provides action attribution with the following fields:

| Field | Type | Description |
|-------|------|-------------|
| id | TEXT | Unique audit entry identifier |
| timestamp | TEXT | When the action occurred (RFC3339) |
| actor_id | TEXT | Operator who performed the action |
| action_class | TEXT | Category of action (read, ack, control, etc.) |
| action_detail | TEXT | Specific action type |
| resource_type | TEXT | Type of resource acted upon |
| resource_id | TEXT | Identifier of specific resource |
| reason | TEXT | Operator-provided or system reason |
| result | TEXT | Outcome (success, denied, failed, partial) |
| details | TEXT | JSON-encoded additional details |
| session_id | TEXT | Session identifier (for future use) |
| remote_addr | TEXT | Client address |

## Database Migrations

### 0013_audit_log.sql
Creates the `audit_log` table for action attribution.

### 0014_operator_sessions.sql
Creates the `operator_sessions` table for future session-based authentication.

## Middleware Integration

The auth middleware (`internal/auth/middleware.go`) provides:

- `WithAuthContext()`: Extracts identity from X-API-Key header or Basic Auth
- `GetAuthContextFromRequest()`: Retrieves auth context in handlers
- `RequirePermission()`: Permission check middleware (currently advisory)
- `RequireRole()`: Role check middleware (currently advisory)

## Migration Path to Multi-Operator

To enable full multi-operator RBAC support:

1. **Implement operator identity store**
   - Add operator table with credentials
   - Support multiple API keys
   - Integrate with external identity providers

2. **Enable session management**
   - Use operator_sessions table
   - Implement session lifecycle (create, refresh, expire)
   - Add session validation to middleware

3. **Enforce RBAC**
   - Update CanPerformWithContext() to check actual permissions
   - Return 403 for unauthorized actions
   - Add role management UI

4. **Add audit attribution**
   - Map actions to actual operator IDs
   - Include session information in audit logs
   - Maintain chain of custody for all actions

## Security Considerations

### Current State (Single-Operator Mode)

- Authentication uses Basic Auth or API key
- No granular authorization checks
- All actions logged with "system" actor
- No operator isolation

### Target State (Multi-Operator)

- Per-operator authentication
- Role-based permission enforcement
- Complete audit trail with operator attribution
- Session-based access control

## References

- [CONTROL_PLANE_TRUST_MODEL.md](../architecture/CONTROL_PLANE_TRUST_MODEL.md)
- [PRODUCTION_MATURITY_MATRIX.md](../architecture/PRODUCTION_MATURITY_MATRIX.md)
- [API Reference](./api-reference.md)

---

*Last updated: 2026-03-21*
