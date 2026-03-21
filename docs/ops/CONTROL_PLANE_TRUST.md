# MEL Control Plane Trust Model

MEL's control plane is designed so every autonomous action is attributable,
auditable, and blockable by operators. This document describes the trust model
that governs how MEL makes and records decisions.

## Core Invariants

1. **No action without a decision record.** Every control action is linked to a
   `ControlDecision` record in the database. There is no hidden execution path.

2. **No fake approvals.** An action moves from `pending_approval` to `pending`
   (runnable) only when an operator explicitly approves it via the API or CLI.
   The DB `UPDATE` is guarded by a `WHERE lifecycle_state='pending_approval'`
   clause — approving a completed or running action is a no-op.

3. **Freeze is enforced in two places.** The freeze check runs at proposal time
   (in `evaluateControl`) *and* at execution time (in `executeControlAction`).
   A freeze installed between proposal and execution still stops the action.

4. **Evidence bundles are immutable after capture.** The integrity hash in
   `evidence_bundles.integrity_hash` covers the payload JSON at the time of
   capture. An updated bundle carries a new hash.

5. **All mutations are audited.** Every approve, reject, freeze, and note
   operation writes a row to `audit_logs` via `InsertRBACAuditLog`.

## Execution Modes

| Mode | Behaviour |
|------|-----------|
| `auto` | Action executes immediately if the control decision allows it and no freeze is active. |
| `approval_required` | Action is persisted in `pending_approval` state; execution waits for operator approval. |
| `manual_only` | Action is proposed but never executed by MEL; operator must run it out-of-band. |
| `dry_run` | Execution logic runs but actuator side-effects are suppressed; result is logged. |

Execution mode is resolved per-action by `resolveExecutionMode` in
`internal/service/trust.go`. The config fields
`control.require_approval_for_action_types` and
`control.require_approval_for_high_blast_radius` influence this resolution.

## Freeze and Maintenance Windows

**Freezes** are runtime-writable records that stop MEL from executing
autonomous actions. Scope:

- `global` — blocks every action on every transport.
- `transport` — blocks actions targeting a specific transport.
- `action_type` — blocks a specific action type globally.

Freezes can carry an optional `expires_at` timestamp; MEL's cleanup worker
expires them automatically.

**Maintenance windows** are time-bounded. An active window (current time is
between `starts_at` and `ends_at`) suppresses autonomous action in the same
scopes as freezes.

## Action Lifecycle

```
proposed
  └─► pending_approval  (if execution_mode = approval_required)
        ├─► pending      (approved — re-queued for execution)
        └─► completed/rejected  (rejected or expired)
  └─► pending           (auto mode)
        └─► running
              ├─► completed
              └─► recovered (after retry)
```

## Evidence Bundles

Every action that reaches the execution gate gets an evidence bundle. The bundle
captures:

- The control decision and policy summary in effect at proposal time.
- Transport health snapshot at proposal time.
- Prior related decisions.
- Integrity hash (SHA-256 of the JSON payload).

Evidence bundles are stored in the `evidence_bundles` table and are accessible
via `mel control inspect <action-id>` or `GET /api/v1/control/actions/<id>/inspect`.

## Blast Radius Classes

| Class | Scope |
|-------|-------|
| `local` | Affects only local MEL state, no mesh impact. |
| `transport` | Affects one transport connection. |
| `mesh` | May affect mesh topology. |
| `global` | Affects all transports and mesh state. |
| `unknown` | Not yet classified; treated as high-blast for approval purposes. |

When `control.require_approval_for_high_blast_radius = true`, actions with
blast radius `mesh`, `global`, or `unknown` require operator approval.

## Actor Attribution

All mutations carry an actor identity:

- API operations: the `X-Operator-ID` header or HTTP Basic Auth username.
- CLI operations: the `--actor` flag (default `cli-operator`).
- Automated system operations: `system`.

Actor IDs are stored in `approved_by`, `rejected_by`, `created_by`,
`cleared_by`, and in the `audit_logs` table.
