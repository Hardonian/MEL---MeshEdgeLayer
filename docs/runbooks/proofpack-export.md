# Proofpack Export Runbook

## What It Is

A proofpack is an incident-scoped evidence bundle assembled by MEL for audit,
export, and operator review. It captures everything MEL knows about a single
incident at assembly time:

- The incident record (state, severity, owner, handoff context)
- Linked control actions (full lifecycle, approval chain, outcomes)
- Timeline events (chronological view of actions, freezes, notes)
- Transport health context (health snapshots around the incident window)
- Dead letters (ingest failures in the incident time window)
- Operator notes attached to the incident
- RBAC audit log entries for the incident
- Explicit evidence gap markers

## What It Is Not

- Not proof of current state. It is a historical snapshot.
- Not an attestation of correctness. It records what was available.
- Not a replacement for the support bundle (which exports system-wide diagnostics).
- Not a live feed. Each proofpack is assembled once at request time.

## How To Use

### Via the UI

1. Navigate to **Incidents** in the MEL console.
2. Find the incident you want to export.
3. Click the **Export proofpack** button on the incident card.
4. The browser downloads a JSON file: `mel-proofpack-{incident-id}.json`.

### Via the API

```
GET /api/v1/incidents/{incident-id}/proofpack
GET /api/v1/incidents/{incident-id}/proofpack?download=true
```

**Required capabilities**: `export_support_bundle` or `read_incidents`.

The `?download=true` query parameter sets the `Content-Disposition` header for
browser-friendly file download.

### Via CLI (future)

```
mel incident proofpack <incident-id> --output proofpack.json
```

CLI proofpack export is planned but not yet implemented.

## Evidence Window

The proofpack assembler computes a time window around the incident:

- **Start**: `occurred_at - 30 minutes`
- **End**: `resolved_at + 30 minutes` (or now + 30 minutes if unresolved)

Timeline events, transport snapshots, and dead letters within this window are
included. Control actions are included based on the canonical incident FK
linkage, not the time window.

## Evidence Gaps

Every proofpack includes an `evidence_gaps` array. Each entry describes a known
limitation in the assembled evidence:

- **info**: Expected gaps (e.g., no actions linked, no dead letters in window)
- **warning**: Data source failures (e.g., could not query timeline, action
  count reached limit)
- **critical**: Would indicate fundamental assembly failures (rare)

If no gaps are detected, the array contains a single entry:
```json
{"category": "assessment", "severity": "info", "description": "no evidence gaps detected during assembly"}
```

Consumers must check `evidence_gaps` before treating the proofpack as complete.

## Degraded Behavior

| Condition | Behavior |
|---|---|
| Incident not found | HTTP 404, no proofpack generated |
| Database unavailable | HTTP 503, no proofpack generated |
| Timeline query fails | Proofpack generated with empty timeline + warning gap |
| Transport snapshots unavailable | Proofpack generated with empty transport_context + warning gap |
| Dead letter query fails | Proofpack generated with empty dead_letter_evidence + warning gap |
| Action query fails | Proofpack generated with empty linked_actions + warning gap |
| Audit log query fails | Proofpack generated with empty audit_entries + warning gap |
| Result count exceeds limit | Proofpack generated with truncated data + warning gap noting limit |

The assembler is designed to produce the best available proofpack even when some
data sources are degraded. The evidence_gaps array always reflects the true
assembly conditions.

## Schema

The proofpack JSON follows format version `1.0.0`. Key fields:

```
format_version: "1.0.0"
assembly:
  assembled_at: RFC3339 timestamp
  assembled_by: actor ID
  instance_id: MEL instance
  incident_id: scoping incident
  time_window_from / time_window_to: evidence window
  action_count, timeline_count, etc.: item counts
  evidence_gap_count: number of gaps
  assembly_duration_ms: wall-clock assembly time
incident: full incident record at assembly time
linked_actions[]: control actions with approval chain
timeline[]: chronological events with provenance markers
transport_context[]: health snapshots in window
dead_letter_evidence[]: ingest failures in window
operator_notes[]: notes on the incident
audit_entries[]: RBAC audit log entries
evidence_gaps[]: explicit gap markers
```

## Audit Trail

Every proofpack export is:
1. Logged to the RBAC audit log (action_class: `export`, action_detail: `proofpack_export`)
2. Recorded as a timeline event (event_type: `proofpack_export`)

This ensures proofpack exports are themselves auditable.

## Performance

- Assembly queries are bounded by configurable limits (default: 200 actions, 500 timeline events, 200 dead letters, 50 transport snapshots)
- Typical assembly takes <500ms for incidents with moderate evidence
- Large incidents with many linked actions may take longer
- The assembler does not cache; each request assembles fresh

## Remaining Limitations

- No diff/comparison between proofpacks of the same incident at different times
- No cryptographic signing or tamper-evidence on the proofpack itself
- No streaming assembly for very large incidents
- CLI export not yet implemented
- No proofpack viewer page in the UI (download-only for now)
