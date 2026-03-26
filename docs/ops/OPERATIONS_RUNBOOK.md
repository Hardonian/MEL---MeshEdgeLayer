# MEL Operations Runbook

This runbook covers day-to-day and incident-response procedures for operators
running MEL in production.

## Quick Reference — Critical Commands

```bash
# Immediate emergency stop (global freeze)
mel freeze create --reason "Emergency stop" --config mel.json

# Check what is frozen or in maintenance
mel freeze list --config mel.json
mel maintenance list --config mel.json

# Review pending approvals
mel control pending --config mel.json

# Approve / reject a pending action
mel control approve <action-id> --note "OK" --config mel.json
mel control reject  <action-id> --note "Too risky" --config mel.json

# View operational posture
mel control operational-state --config mel.json

# Inspect an action with its evidence bundle and notes
mel control inspect <action-id> --config mel.json

# View unified timeline of all events
mel timeline --start 2026-03-01T00:00:00Z --end 2026-03-31T23:59:59Z --config mel.json

# Add an operator note to any resource
mel notes add --ref-type action --ref-id <id> --content "See ticket #123" --config mel.json
```

---

## Runbook: Emergency Freeze

Use when MEL is taking or proposing actions you do not want executed.

1. **Install a global freeze immediately:**
   ```bash
   mel freeze create \
     --reason "Unplanned outage — halting autonomous control" \
     --actor "ops-alice" \
     --config mel.json
   ```
   Note the `freeze_id` in the output.

2. **Verify the freeze is active:**
   ```bash
   mel freeze list --config mel.json
   mel control operational-state --config mel.json
   ```

3. **Investigate.** Review the timeline:
   ```bash
   mel timeline --limit 50 --config mel.json
   ```

4. **When ready to resume, clear the freeze:**
   ```bash
   mel freeze clear <freeze-id> --actor "ops-alice" --config mel.json
   ```

5. **Review any pending-approval actions** that queued during the freeze:
   ```bash
   mel control pending --config mel.json
   ```

---

## Runbook: Scheduled Maintenance

1. **Create a maintenance window before the change:**
   ```bash
   mel maintenance create \
     --title "Router firmware upgrade" \
     --reason "Scheduled upgrade window" \
     --starts-at 2026-04-01T02:00:00Z \
     --ends-at   2026-04-01T04:00:00Z \
     --actor "ops-bob" \
     --config mel.json
   ```

2. **Verify the window:**
   ```bash
   mel maintenance list --config mel.json
   ```

3. **Perform your maintenance.**

4. **Cancel early if maintenance completes ahead of schedule:**
   ```bash
   mel maintenance cancel <window-id> --actor "ops-bob" --config mel.json
   ```

---

## Runbook: Approval Queue Management

When `control.require_approval_for_action_types` or
`control.require_approval_for_high_blast_radius` is enabled, actions queue in
`pending_approval` state.

1. **Check the queue:**
   ```bash
   mel control pending --config mel.json
   ```

2. **Inspect each action before deciding:**
   ```bash
   mel control inspect <action-id> --config mel.json
   ```
   Review `evidence_bundle.explanation`, `blast_radius_class`, and
   `transport_health`.

3. **Approve if safe (canonical path):**
   ```bash
   mel action approve <action-id> \
     --note "Reviewed evidence bundle; transport health OK" \
     --actor "ops-alice" \
     --config mel.json
   ```
   This runs the full service approve path and may perform **at most one** dequeue of
   the in-memory control queue in this CLI process (not a continuous worker). For
   sustained backlog draining, run **`mel serve`** so the control executor loop stays up.

   `mel control approve` remains as **break-glass** only (requires
   `--i-understand-break-glass-sod`) and records durable metadata marking the legacy entrypoint.

4. **Reject if risky:**
   ```bash
   mel action reject <action-id> \
     --note "Blast radius too wide during peak traffic" \
     --actor "ops-alice" \
     --config mel.json
   ```

5. **Set `control.approval_timeout_seconds`** to avoid stale approvals
   accumulating. Timed-out actions are moved to `approval_expired` state.

---

## Runbook: Incident Timeline Review

After an incident, reconstruct the sequence of events:

```bash
mel timeline \
  --start 2026-03-20T00:00:00Z \
  --end   2026-03-21T00:00:00Z \
  --limit 200 \
  --config mel.json
```

Event types in the timeline:
- `action` — control action proposed or updated (from `control_actions` table).
- `action_approved` — operator approved a pending-approval action.
- `action_rejected` — operator rejected a pending-approval action.
- `action_expired` — pending-approval action expired without approval.
- `freeze_created` — automation freeze installed.
- `freeze_cleared` — automation freeze removed.
- `freeze_expired` — freeze removed by expiry timer.
- `maintenance_created` — maintenance window scheduled.
- `maintenance_cancelled` — maintenance window cancelled early.
- `approval_backlog_warn` — system emitted a warning about high approval queue depth.
- `incident` — incident opened or updated.
- `note` — operator added a note to any resource.

Add post-incident notes to relevant actions:
```bash
mel notes add \
  --ref-type action \
  --ref-id   <action-id> \
  --content  "RCA: transport restart was triggered by stale retry counter" \
  --actor    "ops-alice" \
  --config   mel.json
```

---

## Runbook: Checking Self-Observability

MEL reports on its own health via the `health` command:

```bash
mel health internal   --config mel.json   # component health registry
mel health freshness  --config mel.json   # data freshness per component
mel health slo        --config mel.json   # SLO compliance
mel health metrics    --config mel.json   # pipeline latency, queue depths
mel health trust      --config mel.json   # control-plane trust: mode, freezes, backlog
```

The `mel health trust` command shows:
- **Automation mode** — `normal`, `frozen`, or `maintenance`
- **Active freezes** — count of active freeze records
- **Approval backlog** — number of actions awaiting operator approval

For live systems with the API running, trust health is also available at:
```
GET /api/v1/health/trust
```

The trust health degrades (`degraded` status) when:
- Global automation is frozen
- Approval backlog exceeds 5 pending actions

Also available via the `doctor` command which runs all checks and includes
transport-level diagnostics:
```bash
mel doctor --config mel.json
```

---

## Runbook: Database Maintenance

```bash
# Compact the database
mel db vacuum --config mel.json

# Create a backup before upgrades
mel backup create --config mel.json --out /backup/mel-$(date +%Y%m%d).tar.gz

# Validate a backup bundle (dry-run only)
mel backup restore --bundle /backup/mel-20260301.tar.gz --dry-run
```

---

## API Equivalents

All CLI operations have HTTP API counterparts:

| CLI | API |
|-----|-----|
| `mel control operational-state` | `GET /api/v1/control/operational-state` |
| `mel control pending` | `GET /api/v1/control/actions/pending` |
| `mel control approve <id>` | `POST /api/v1/control/actions/<id>/approve` |
| `mel control reject <id>` | `POST /api/v1/control/actions/<id>/reject` |
| `mel control inspect <id>` | `GET /api/v1/control/actions/<id>/inspect` |
| `mel freeze list` | `GET /api/v1/control/freeze` |
| `mel freeze create` | `POST /api/v1/control/freeze` |
| `mel freeze clear <id>` | `DELETE /api/v1/control/freeze/<id>` |
| `mel maintenance list` | `GET /api/v1/control/maintenance` |
| `mel maintenance create` | `POST /api/v1/control/maintenance` |
| `mel maintenance cancel <id>` | `DELETE /api/v1/control/maintenance/<id>` |
| `mel timeline` | `GET /api/v1/timeline` |
| `mel notes add` | `POST /api/v1/operator/notes` |
| `mel health trust` | `GET /api/v1/health/trust` |
