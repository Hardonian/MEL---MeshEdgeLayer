# MEL API and CLI Reference

---

## CLI Overview

```
mel <command> [subcommand] [flags]
```

All commands accept `--config <path>` (default: `configs/mel.example.json`).
Commands that mutate state accept `--actor <identity>` (default: `cli-operator`).

---

## Core Commands

### `mel serve`
Start the MEL server.

```
mel serve [--debug] --config <path>
```

### `mel init`
Generate a default configuration file.

```
mel init [--config <path>] [--force]
```

### `mel doctor`
Run pre-launch environment checks.

```
mel doctor --config <path>
```

### `mel config validate`
Validate the config file without starting the server.

```
mel config validate --config <path>
```

### `mel config inspect`
Inspect the loaded and normalised configuration.

```
mel config inspect --config <path>
```

### `mel status`
Snapshot of current transport and ingest state.

```
mel status --config <path>
```

---

## Control Plane Commands

### `mel control status`
Show the current control evaluation explanation.

### `mel control history`
Show past control actions and decisions.

```
mel control history --config <path> [--transport <name>] [--start <RFC3339>] [--end <RFC3339>] [--limit <n>] [--offset <n>]
```

### `mel control pending`
List all actions awaiting operator approval.

```
mel control pending --config <path>
```

Output: `{ "pending_approval": [...], "count": N }`

### `mel control approve`
Approve a pending action and re-queue it for execution.

```
mel control approve <action-id> --config <path> [--note "..."] [--actor <id>]
```

### `mel control reject`
Reject a pending action (terminal state).

```
mel control reject <action-id> --config <path> [--note "..."] [--actor <id>]
```

### `mel control inspect`
Inspect an action along with its evidence bundle and operator notes.

```
mel control inspect <action-id> --config <path>
```

Output includes:
- `action` — full action record with execution mode, blast radius, lifecycle state.
- `evidence_bundle` — provenance snapshot (if captured).
- `notes` — operator notes attached to the action.

### `mel control operational-state`
Full operational posture snapshot: active freezes, maintenance windows,
pending approvals, config modes.

```
mel control operational-state --config <path>
```

### Mesh deployment intelligence (bootstrap, topology advisory, routing proxies)

These commands read the same SQLite evidence as the server (`nodes`, `topology_links`, recent `messages`). They do **not** prove RF coverage or change Meshtastic routing.

```
mel mesh bootstrap --config <path>     # lone-wolf / viability summary
mel mesh topology --config <path>      # cluster shape + per-node contribution proxies
mel mesh diagnose --config <path>      # routing-pressure + protocol-fit advisory
mel mesh recommend --config <path>     # ranked next steps
mel mesh inspect --config <path>      # full JSON assessment
mel mesh history --config <path>       # persisted snapshots (same DB as daemon)
```

`mel inspect topology` includes a compact `mesh_intelligence_summary` for paste-friendly field reports.

---

## Freeze Commands

### `mel freeze list`
List all active freezes.

```
mel freeze list --config <path>
```

### `mel freeze create`
Install a control freeze.

```
mel freeze create --config <path> \
  --reason "..." \
  [--scope-type global|transport|action_type] \
  [--scope-value <name>] \
  [--expires-at <RFC3339>] \
  [--actor <id>]
```

Default scope is `global`.

### `mel freeze clear`
Remove an active freeze.

```
mel freeze clear <freeze-id> --config <path> [--actor <id>]
```

---

## Maintenance Window Commands

### `mel maintenance list`
List all maintenance windows (active and historical).

```
mel maintenance list --config <path>
```

### `mel maintenance create`
Schedule a maintenance window.

```
mel maintenance create --config <path> \
  --starts-at <RFC3339> \
  --ends-at   <RFC3339> \
  [--title "..."] \
  [--reason "..."] \
  [--scope-type global|transport] \
  [--scope-value <name>] \
  [--actor <id>]
```

### `mel maintenance cancel`
Cancel a maintenance window before it ends.

```
mel maintenance cancel <window-id> --config <path> [--actor <id>]
```

---

## Timeline Command

```
mel timeline --config <path> \
  [--start <RFC3339>] \
  [--end <RFC3339>] \
  [--limit <n>]
```

Returns unified event stream across: control actions, freeze lifecycle,
maintenance windows, operator notes.

---

## Notes Commands

### `mel notes add`
Attach a freeform note to any resource.

```
mel notes add --config <path> \
  --ref-type <action|incident|transport|node> \
  --ref-id   <resource-id> \
  --content  "..." \
  [--actor <id>]
```

### `mel notes list`
List notes for a resource.

```
mel notes list --config <path> \
  --ref-type <type> \
  --ref-id   <id> \
  [--limit <n>]
```

---

## HTTP API Reference

Base URL: `http://<bind.api>/api/v1`

Authentication: Bearer token (`Authorization: Bearer <token>`) or
Basic Auth when `auth.token` or `auth.basic_auth` is configured.

For operator mutations (approve, reject, freeze, notes): include
`X-Operator-ID: <identity>` header or use Basic Auth username.

### Status

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/status` | Transport and ingest snapshot. |
| `GET` | `/health` | Liveness probe. |
| `GET` | `/metrics` | Prometheus-format metrics. |

### Nodes and Messages

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/nodes` | List known mesh nodes. |
| `GET` | `/nodes/<id>` | Single node detail. |
| `GET` | `/messages` | Recent messages. |

### Control Plane

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/control/operational-state` | Full operational posture. |
| `GET` | `/control/actions` | List control actions. |
| `GET` | `/control/actions/pending` | Pending-approval actions. |
| `GET` | `/control/actions/<id>/inspect` | Action + evidence bundle + notes. |
| `POST` | `/control/actions/<id>/approve` | Approve pending action. Body: `{"note":"..."}` |
| `POST` | `/control/actions/<id>/reject` | Reject pending action. Body: `{"note":"..."}` |

### Freezes

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/control/freeze` | List active freezes. |
| `POST` | `/control/freeze` | Create freeze. Body: `{"scope_type":"global","reason":"...","expires_at":"..."}` |
| `DELETE` | `/control/freeze/<id>` | Clear freeze. |

### Maintenance Windows

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/control/maintenance` | List maintenance windows. |
| `POST` | `/control/maintenance` | Create window. Body: `{"title":"...","starts_at":"...","ends_at":"..."}` |
| `DELETE` | `/control/maintenance/<id>` | Cancel window. |

### Timeline

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/timeline` | Unified event timeline. Query params: `start`, `end`, `limit`. |

### Operator Notes

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/operator/notes?ref_type=<t>&ref_id=<id>` | List notes for resource. |
| `POST` | `/operator/notes` | Add note. Body: `{"ref_type":"action","ref_id":"...","content":"..."}` |

### Mesh deployment intelligence

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/mesh/intelligence` | Current assessment: bootstrap, topology metrics, routing-pressure diagnostics, protocol-fit advisory, ranked recommendations. |
| `GET` | `/api/v1/mesh/intelligence/history?limit=<n>` | Recent persisted assessments (bounded retention). |
| `GET` | `/api/v1/topology` | Topology intelligence bundle; includes `mesh_intelligence` when the topology worker has computed a snapshot. |
| `GET` | `/api/v1/topology/nodes/<num>` | Node drilldown; includes `mesh_intel` for that node when available. |

---

## Common Response Fields

All API responses are JSON. Errors:

```json
{
  "error": "short description",
  "detail": "longer explanation"
}
```

Success mutations return at minimum:
```json
{
  "status": "approved"|"rejected"|"created"|"cleared"|"cancelled",
  "<resource>_id": "..."
}
```
