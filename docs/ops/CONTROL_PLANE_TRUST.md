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
| `unknown` | Not yet classified; informative only unless another policy rule applies. |

When `control.require_approval_for_high_blast_radius = true`, actions with
blast radius `mesh` or `global` require operator approval. This is **config-driven**:
dynamic classification does not silently add new approval rules beyond these flags.

## Approval vs execution (single approver)

MEL enforces **one explicit approval step** per action that is in `approval_required`
mode. There is **no quorum** and no multi-stage approval in this version.

- **Approved** means the row left `pending_approval` and is authorized to run; it does
  **not** mean the actuator has finished.
- **`pending` + `result=approved`** means approved and **waiting for the in-process
  control executor** (`mel serve`) or a **CLI one-shot dequeue** after `mel action approve`.
- **HTTP `POST .../approve`** records approval and may enqueue **this** action only; it
  does **not** drain unrelated backlog. Use `mel serve` for continuous draining.

Separation of duties: the **submitter** (`submitted_by`) cannot approve or reject the
same row unless **break-glass** is used (`mel control ...` legacy path with explicit ack,
or HTTP `break_glass_sod_ack` with `break_glass_sod_reason`).

## Actor Attribution

All mutations carry an actor identity:

- API operations: the `X-Operator-ID` header or HTTP Basic Auth username.
- CLI operations: the `--actor` flag (default `cli-operator`).
- Automated system operations: `system`.

Actor IDs are stored in `approved_by`, `rejected_by`, `created_by`,
`cleared_by`, and in the `audit_logs` table.

## Timeline Events

Every trust-layer mutation emits an explicit event into the `timeline_events`
table (migration 0018). This supplements the UNION-query view of the timeline
with durable first-class rows that survive source-table retention pruning.

Events emitted automatically:
- `action_approved` — on `ApproveAction`
- `action_rejected` — on `RejectAction`
- `freeze_created` — on `CreateFreeze`
- `freeze_cleared` — on `ClearFreeze`
- `maintenance_created` — on `CreateMaintenanceWindow`
- `maintenance_cancelled` — on `CancelMaintenanceWindow`
- `approval_backlog_warn` — when cleanup loop detects ≥ 5 pending-approval actions

All events carry `event_time`, `event_type`, `summary`, `severity`,
`actor_id`, `resource_id`, and a `details_json` blob.

## Self-Observability

The `trust` component is tracked in the selfobs health registry. It records:
- **success** on each successful cleanup-loop run (expiry + freeze expiry)
- **failure** when cleanup fails due to DB errors

The `trust` component freshness is updated via `selfobs.MarkFresh("trust")`
each cycle, allowing the freshness tracker to detect if the cleanup loop stops.

Trust health is available at:
- `GET /api/v1/health/trust` — mode, freeze count, backlog, component health
- `mel health trust --config <path>` — human-readable CLI summary

Trust health degrades when:
- `automation_mode` is `frozen`
- `approval_backlog` ≥ 5 pending actions

## InspectAction

`InspectAction` (service layer) / `GET /api/v1/control/actions/<id>/inspect` /
`mel control inspect <id>` returns:

```json
{
  "action":          { ...ControlActionRecord },
  "decision":        { ...ControlDecisionRecord | null },
  "evidence_bundle": { ...EvidenceBundleRecord | null },
  "notes":           [ ...OperatorNoteRecord ],
  "inspected_at":    "2026-03-21T12:00:00Z"
}
```

The decision lookup uses the direct-by-ID query `ControlDecisionByID`
(added to `internal/db/control.go`) for O(1) retrieval.
