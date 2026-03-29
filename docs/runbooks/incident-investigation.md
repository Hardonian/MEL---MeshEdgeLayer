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
mel investigate --config <path>
mel investigate cases --config <path>
```

Check:

- Are all expected transports connected? If not, findings are scoped to connected transports only.
- Are there active freezes? (`mel freeze list`)
- Are there active maintenance windows? (`mel maintenance list`)
- What is the fleet visibility posture? (partial fleet: only this instance)
- Which investigation cases exist? A case is a bounded operator problem, not a confirmed diagnosis.

Interpret investigation output this way:

- `case` = a bounded operator attention object tying together findings, gaps, recommendations, and raw-record links.
- `finding` = what MEL observes.
- `evidence gap` = what MEL still does not know.
- `recommendation` = the next safe inspection step MEL can justify from the evidence.
- `linked event` = a raw canonical event that contributed context to the case. It is related evidence, not automatic proof of cause.
- `case evolution` = MEL's typed reconstruction of how the current case posture was shaped by retained evidence, gaps, validation outcomes, merges, and recommendations.

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
# List bounded investigation cases first
mel investigate cases --config <path>

# Show one investigation case with linked findings, gaps, recommendations, and raw records
mel investigate show <case-id> --config <path>

# Show the case's temporal spine: linked raw events, timing posture, and evolution entries
mel investigate timeline <case-id> --config <path>

# List all incidents
mel incident list --config <path>

# Inspect a specific incident
mel incident inspect <incident-id> --config <path>

# Handoff to another operator
mel incident handoff <id> --from operator-a --to operator-b --summary "..." --config <path>
```

**API parity:**

```http
GET /api/v1/investigations
GET /api/v1/investigations/cases
GET /api/v1/investigations/cases/<id>
GET /api/v1/investigations/cases/<id>/timeline
GET /api/v1/incidents
GET /api/v1/incidents/<id>
POST /api/v1/incidents/<id>/handoff
```

### Reading case timing posture

Case detail and case timeline output now surface a normalized timing posture:

- `locally_ordered` = linked raw events were recorded locally in this MEL instance; sequence is exact only within this database.
- `historical_import_not_live` = imported evidence is shaping the case as historical context; it is not live local proof.
- `source_order_known_global_order_unknown` = some source-local ordering may be known, but the case spans more than one evidence domain and no global total order is implied.
- `ordering_uncertain_clock_skew` / `ordering_uncertain_missing_timestamps` = sequence is still useful, but exact ordering remains bounded by missing or skewed timestamps.
- `merged_best_effort_order` = merge/dedupe classification affected the visible order or grouping.
- `mixed_freshness_window` = the case spans stale and fresher evidence windows; sequence is not the same as freshness certainty.

Read `linked events` and `case evolution` together:

- linked events are the exact raw timeline rows MEL could connect to the case,
- case evolution explains how the current case posture was shaped,
- neither section grants MEL permission to invent a clean incident narrative.

### Reading action-outcome memory (incident intelligence)

When `action_outcome_memory[]` is present on an incident, treat it as historical association only:

- `outcome_framing` summarizes how prior similar incidents trended after that action type; it is not a recommendation or a causal claim.
- `sample_size` and `evidence_strength` bound confidence; sparse sample counts should be treated as weak context.
- `mixed_historical_evidence`, `insufficient_evidence`, and `no_clear_post_action_signal` are distinct caution states and should not be collapsed into "healthy" or "safe to execute".
- `inspect_before_reuse[]` is an operator review requirement before reusing the action pattern in a live incident.

### Reading mixed wireless context (incident intelligence)

When `wireless_context` is present on an incident, treat it as a deterministic context summary — not RF root-cause proof.

- `classification` is bounded to operator-facing categories (`lora_mesh_pressure`, `wifi_backhaul_instability`, `mixed_path_degradation`, `sparse_evidence_incident`, `unsupported_wireless_domain_observed`, `recurring_unknown_pattern`).
- `observed_domains[]` is term/evidence linked context, not a claim that MEL has full telemetry for every domain.
- `confidence_posture` and `evidence_posture` must be read together; `sparse`/`unsupported` means MEL is preserving evidence without strong diagnosis.
- `unsupported[]` explicitly marks domains MEL cannot currently diagnose directly (for example BLE ingest).
- `evidence_gaps[]` carries machine-visible uncertainty markers; do not collapse these to "healthy" or "known cause."

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
- `investigation_cases.json` — case list plus expanded case detail, linked raw events, timing posture, and case evolution
- `control_actions.json` — action audit trail
- `incidents.json` — incident context
- `imported_evidence.json` — remote evidence imports and validation
- `diagnostics.json` — active findings

## Common anti-patterns to avoid

1. **"The dashboard shows green"** — The dashboard reflects this instance's database. If transports are disconnected, it shows nothing, not health.
2. **"We received 5 packets from 3 gateways, so the mesh is working"** — Repeated observations are symptoms, not coverage proof. You cannot conclude RF reach from packet counts.
3. **"The import was accepted"** — Acceptance means structural validation passed. It does not mean the remote site's data is truthful or that the two sites share a common timeline.
4. **"No incidents, so the system is healthy"** — No incidents means no incidents were *created*. Check diagnostics and transport alerts separately.
5. **"High attention means MEL knows the cause"** — Attention is about operator urgency; certainty is still bounded by evidence gaps, stale reporters, and partial scope.
