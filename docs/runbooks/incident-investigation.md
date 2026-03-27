# Hostile-first incident investigation

This runbook is for operators investigating an active or recent incident on a MEL gateway. It assumes hostile conditions: incomplete data, transport failures, clock skew on remote evidence, and operators under time pressure.

## Rule zero

**Observation is not coverage. Missing data is not proof of absence.**

If a transport is disconnected, its evidence is unavailable — not negative. Do not conclude the mesh is healthy because the database is quiet. Verify transport liveness first.

## Step 1: Establish system posture

```bash
mel status --config <path>
mel health trust --config <path>
mel diagnostics --config <path>
```

Check:

- Are all expected transports connected? If not, findings are scoped to connected transports only.
- Are there active freezes? (`mel freeze list`)
- Are there active maintenance windows? (`mel maintenance list`)
- What is the fleet visibility posture? (partial fleet: only this instance)

## Step 2: Timeline investigation

```bash
# Full timeline: what happened, in order
mel timeline list --start "2026-03-01T00:00:00Z" --config <path>

# Filter to specific event types (control_action, remote_evidence_import, incident, etc.)
mel timeline list --type control_action --config <path>

# Filter to scope posture (local, remote_imported, best_effort_fleet)
mel timeline list --scope remote_imported --config <path>

# Drill into a specific event
mel timeline inspect <event-id> --config <path>
```

**API parity:**

```http
GET /api/v1/timeline?start=2026-03-01T00:00:00Z
GET /api/v1/timeline?event_type=control_action
GET /api/v1/timeline?scope_posture=remote_imported
```

When inspecting a timeline event, read the investigation guidance in the output:

- `scope_posture=remote_imported`: event came from another truth domain (offline). Treat timing as best-effort.
- `timing_posture=local_ordered`: strict deterministic order within this instance.
- Otherwise: potential clock skew; do not assume causal ordering across instances.

## Step 3: Incident drilldown

```bash
# List all incidents
mel incident list --config <path>

# Inspect a specific incident
mel incident inspect <incident-id> --config <path>

# Handoff to another operator
mel incident handoff <id> --from operator-a --to operator-b --summary "..." --config <path>
```

**API parity:**

```http
GET /api/v1/incidents
GET /api/v1/incidents/<id>
POST /api/v1/incidents/<id>/handoff
```

## Step 4: Control action audit

```bash
# What was done? What was the evidence basis?
mel action inspect <action-id> --config <path>

# Full trace with evidence chain
mel trace <action-id> --config <path>

# Who approved/rejected?
mel action history --config <path>
```

Check:

- Was the action operator-initiated or automation-proposed?
- What was the lifecycle state? (pending_approval, approved, executed, rejected, expired)
- Was there a break-glass override?

## Step 5: Remote evidence (if applicable)

```bash
# List imported remote evidence
mel fleet evidence list --config <path>

# Inspect a specific import
mel fleet evidence show <import-id> --config <path>

# Check merge disposition between two evidence keys
curl -s "http://localhost:8080/api/v1/fleet/merge-explain?key_a=...&key_b=...&same_observer=false"
```

Remember:

- **Validation ≠ truth.** Structural acceptance does not mean the remote site is healthy.
- **No authenticity by default.** Origin fields are claimed, not cryptographically verified.
- **No global order.** Imported events preserve import/observation time distinctions, not a total fleet order.

## Step 6: Operator notes

Attach investigation context for handoff or audit trail:

```bash
mel notes add --ref-type incident --ref-id <incident-id> --content "Root cause: ..." --config <path>
mel notes list --ref-type incident --ref-id <incident-id> --config <path>
```

## Step 7: Support bundle export

When escalating to second-line support:

```bash
mel support --config <path> --out support-bundle.zip
```

The support bundle includes:

- `MANIFEST.md` — read this first; it explains every file
- `bundle.json` — full monolith for machine ingestion
- `timeline.json` — full event history
- `control_actions.json` — action audit trail
- `incidents.json` — incident context
- `imported_evidence.json` — remote evidence imports and validation
- `diagnostics.json` — active findings

## Common anti-patterns to avoid

1. **"The dashboard shows green"** — The dashboard reflects this instance's database. If transports are disconnected, it shows nothing, not health.
2. **"We received 5 packets from 3 gateways, so the mesh is working"** — Repeated observations are symptoms, not coverage proof. You cannot conclude RF reach from packet counts.
3. **"The import was accepted"** — Acceptance means structural validation passed. It does not mean the remote site's data is truthful or that the two sites share a common timeline.
4. **"No incidents, so the system is healthy"** — No incidents means no incidents were *created*. Check diagnostics and transport alerts separately.
