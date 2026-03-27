# Support bundle interpretation guide

This document is for second-line support engineers, incident responders, and anyone who receives a MEL support bundle zip and needs to understand what is inside and how to use it.

## Quick start

1. Unzip the support bundle.
2. Read `MANIFEST.md` — it is the index and provides context for all other files.
3. Start with `investigation.json` for the canonical investigation summary.
4. Use `investigation_cases.json` to inspect bounded case drilldown.
5. Use `timeline.json` for chronological investigation.
6. Use the specific section files (`control_actions.json`, `incidents.json`, `imported_evidence.json`) for targeted drilldown.

## File reference

|File|Contents|When to use|
|---|---|---|
|`MANIFEST.md`|Index, interpretation notes, quick commands|Always read first|
|`bundle.json`|Full monolith (all sections, machine-readable)|Tooling ingestion; grep for fields|
|`doctor.json`|`mel doctor` output (secrets redacted)|Host and config health|
|`timeline.json`|Full unified event timeline|Chronological investigation|
|`control_actions.json`|Control actions and decisions|Audit trail: who did what and why|
|`incidents.json`|Incident records|Correlation, handoff context|
|`imported_evidence.json`|Remote evidence batches, imported rows, and inspections|Provenance, validation, timing, merge posture|
|`remote_evidence_export.json`|Canonical offline re-export of imported evidence|Hand off the same imported evidence as a truthful offline batch|
|`diagnostics.json`|Active diagnostic findings|Issues and recommended next steps|
|`investigation.json`|Canonical investigation summary|Operator attention posture, cases, findings, evidence gaps, recommendations, and physics boundaries|
|`investigation_cases.json`|Expanded investigation cases|Case list plus expanded case detail for second-line reconstruction|

## Reading investigation output

Use these meanings consistently:

- `case` = a bounded operator attention object. It is not proof of root cause.
- `finding` = what MEL is observing from available evidence.
- `evidence_gap` = what MEL still does not know and why that matters.
- `recommendation` = the next safe inspection step MEL can justify.

Every case should reduce back to inspectable raw records such as transport runtime, alerts, incidents, timeline events, or imported evidence rows.

## Reading the timeline

The timeline is a UNION of disparate event types into a single chronological stream. Each event has explicit posture fields:

### `scope_posture`

|Value|Meaning|
|---|---|
|`local`|Event was recorded by this instance|
|`remote_imported`|Event was imported from another truth domain (offline bundle)|
|`best_effort_fleet`|Correlated across instances; ordering is not strict|

### `timing_posture`

|Value|Meaning|
|---|---|
|`local_ordered`|Strict deterministic order within this instance|
|`receive_time_differs_from_observed_time`|Observed and received timestamps differ; clock skew possible|

### `merge_disposition`

|Value|Meaning|
|---|---|
|`raw_only`|No merge processing; single-origin event|
|`summary_with_contributor_lineage`|Summary with preserved contributor trail|
|`merged_canonical_summary`|Merged from multiple sources into canonical summary|

## Reading imported evidence

`imported_evidence.json` contains batch-level and item-level sections:

- `batches` — persisted import batch audit rows
- `batch_inspections` — decoded batch validation/source drilldown
- `imports` — persisted imported item rows
- `inspections` — normalized per-item provenance/timing/merge drilldown

Each imported item includes:

- **`validation`** — Structured validation outcome (accepted, accepted_with_caveats, rejected) with machine-readable reason codes.
- **`bundle`** — Raw import bundle JSON (the exact file that was imported).
- **`evidence`** — Normalized evidence envelope.
- **`origin_instance_id`** — The MEL instance that *claims* to have originated this evidence.
- **`origin_site_id`** — The site that *claims* to own this evidence.
- **`batch_id` / `source_*`** — The local import batch and local file/source context.
- **`timing_posture`** — How MEL believes the row should be ordered/interpreted.
- **`merge_disposition` / `merge_correlation_id`** — Why MEL kept the row raw, related, or comparable.

**Critical**: "claimed" means the origin fields are self-reported. MEL does not cryptographically verify origin unless external verification is added by the operator.

### Validation outcomes

|Outcome|Meaning|
|---|---|
|`accepted`|Structurally valid, no caveats|
|`accepted_with_caveats`|Structurally valid but with warnings (e.g., partial observation)|
|`rejected`|Structurally invalid or scope conflict|

### Batch outcomes

Batch validation is separate from per-item validation.

|Outcome|Meaning|
|---|---|
|`accepted_with_caveats`|Every item was imported, but the whole batch remains historical/offline/unverified|
|`accepted_partial_bundle`|Some items were imported, some were rejected|
|`rejected`|No item was accepted; the raw batch survives only as audit evidence|

### Reading `remote_evidence_export.json`

This file is a canonical `mel_remote_evidence_batch` payload generated from the imported rows already stored locally.

Use it when you need to:

- hand the exact imported evidence to another MEL instance,
- attach a truthful offline batch to an escalation,
- preserve the distinction between local evidence and imported evidence.

Do not read it as proof of live sync, cross-instance authority, or fleet-wide completeness.

## Reading control actions

Each action includes lifecycle state and evidence chain:

|Field|What to check|
|---|---|
|`lifecycle_state`|pending_approval, approved, executed, rejected, expired|
|`execution_mode`|manual, automated, automation_gated|
|`proposed_by`|Who or what proposed the action|
|`approved_by`|Who approved (empty if auto-executed)|
|`trigger_evidence`|JSON evidence that triggered this action|
|`reason`|Human-readable justification|
|`outcome_detail`|Result of execution|

## What the bundle does NOT tell you

1. **Global fleet health.** The bundle reflects one instance's database. Other instances may have evidence this one does not.
2. **RF coverage.** Packet observations do not prove coverage area or mesh connectivity.
3. **Transport liveness at export time.** The bundle captures database state, not real-time status.
4. **Causal ordering across instances.** Timeline order is instance-local.
5. **Data completeness.** Missing data may indicate transport failure, not health.

## Escalation checklist

Before escalating beyond second-line:

- [ ] Read `MANIFEST.md`
- [ ] Check `diagnostics.json` for active findings
- [ ] Check `timeline.json` for the event sequence leading to the issue
- [ ] Verify transport connectivity in `bundle.json` → `status_snapshot.transports`
- [ ] Check for active freezes in `bundle.json` → `control_plane_state`
- [ ] Note the `fleet_truth` posture — is this a partial-fleet view?
- [ ] Attach operator notes to the relevant incident/action for audit trail
