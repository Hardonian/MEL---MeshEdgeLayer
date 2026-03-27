# Canonical Timeline & Evidence Provenance

MEL (MeshEdgeLayer) is an operator system. Its primary responsibility is not to invent a synthetic global truth, but to maintain a **durable, inspectable, and honest record** of what happened, where it happened, who observed it, and how it was merged.

The **Unified Operator Timeline** is the canonical substrate for this record.

## 1. The Matrix of Event Posture

When an operator queries the timeline (`mel timeline list`), every control action, incident, and imported evidence bundle is surfaced with explicit metadata describing its origin and ordering posture. There is no silent collapse.

The canonical representation (`db.TimelineEvent`) guarantees the following fields:

- `scope_posture`: Whether the event occurred locally (`local`), was imported offline (`remote_imported`), or is a best-effort correlation (`best_effort_fleet`).
- `origin_instance_id`: The exact MEL instance ID that recorded or originated the event.
- `timing_posture`: How the `event_time` should be trusted (e.g., `local_ordered` vs `receive_time_differs_from_observed_time`).
- `merge_disposition`: How the system handled potential duplication or ambiguity (`raw_only`, `summary_with_contributor_lineage`, `merged_canonical_summary`).

## 2. No Silent "Live Mesh" Fiction

When evidence is imported from a remote site or fleet device, MEL does not pretend the two nodes are talking live over MQTT. 

1. **Import is Offline**: Remote bundles are ingested into `imported_remote_evidence` via `mel fleet evidence import`.
2. **Timeline Provenance**: A `remote_evidence_import` event is recorded. Its `scope_posture` is immediately tagged `remote_imported`.
3. **Inspector Drilldown**: The command `mel timeline inspect <event_id>` will unpack the timing skew and evidence bounds, explaining to the operator *why* the order might be best-effort.

## 3. Investigating Merge Constraints

MEL implements *Merge Classifications*, deterministic logic that categorizes evidence duplication. Instead of silently discarding data (which destroys forensic paths if nodes are spoofed), MEL classifies overlap into explicit states:

- `exact_duplicate`: Within the same observer scope (safe to dedupe).
- `near_duplicate_candidate`: Different observers reporting the identical packet fingerprint. MEL preserves both so operator tooling can calculate density, rather than hallucinating "fleet-wide reachability."
- `related_distinct` / `conflicting`: Handled distinctly in the trace output.

## 4. Operator Drilldown Flow

**1. Finding anomalies:**
```bash
mel timeline list --start "2026-03-01T00:00:00Z"
```

**2. Inspecting the precise truth constraints of an event:**
```bash
mel timeline inspect <event-id>
```

The inspect output explicitly warns if the event is from another truth domain (offline imported) and forces the operator to recognize potential clock skew.

## 5. Support & Export Parity

Because `support bundle` runs standard DB export procedures that snapshot `timeline_events`, the entirety of the merged timeline—including origin constraints and dedupe classifications—is preserved for offline core audits. 

**There are no side-channel tables for "UI only" metrics.** If it isn't in `timeline_events` or the core telemetry tables, it isn't operator truth.
