# Remote evidence import (offline only)

This runbook describes the shipped MEL behavior for remote evidence ingest: **offline JSON import into the local SQLite database** with preserved provenance, validation posture, timing posture, and audit trail. It is **not** live federation, remote execution, remote authority, or fleet-wide truth.

## Accepted import contracts

MEL accepts two canonical JSON payloads for `mel fleet evidence import` and `POST /api/v1/fleet/remote-evidence`:

1. `mel_remote_evidence_bundle`
Single imported item with one required `evidence` envelope and one optional `event` envelope.

2. `mel_remote_evidence_batch`
Multi-item offline container with batch-level claimed origin, source context, capability posture snapshot, and an `items[]` array of canonical `mel_remote_evidence_bundle` entries.

Generic MEL exports and support bundles are **not** directly importable as remote evidence. They must be wrapped in one of the canonical formats above.

## What validation proves

Validation is structural and scope-aware. It proves that MEL could parse the payload, evaluate the required fields, and decide whether the payload is acceptable for local historical investigation.

Validation does **not** prove:

- the claimed origin is authentic,
- the remote instance is currently live,
- the imported evidence is representative of the full fleet,
- the remote observation order is globally comparable to local order,
- repeated observations imply flooding, congestion, routing certainty, or coverage certainty.

Every accepted import remains explicitly marked as historical, offline, and authenticity-unverified unless you verify it outside MEL.

## Required fields and rejection rules

For each imported item MEL requires:

- `schema_version = "1.0"`
- `kind = "mel_remote_evidence_bundle"`
- `evidence.evidence_class` with a known MEL evidence enum
- `evidence.origin_instance_id`
- `evidence.observation_origin_class` with a known enum
- at least one timing field in `evidence` or `event`: `observed_at`, `received_at`, or `recorded_at`
- a stable merge/correlation basis: `evidence.correlation_id`, `event.event_id`, a timestamp, or non-empty evidence details

If an `event` envelope is present, MEL also requires:

- `event.event_id`
- `event.event_type`
- `event.summary`
- `event.origin_instance_id`
- consistency with `evidence` for origin instance and correlation id when both are set

Common rejection paths:

- `malformed_json`
- `unsupported_schema_version`
- `unsupported_bundle_kind`
- `missing_scope`
- `missing_timestamps`
- `unsupported_evidence_type`
- `invalid_merge_basis`
- `conflicting_origin_site`
- `event_origin_instance_mismatch`

## Batch behavior

Batch import is **per-item** with a batch audit record.

- If every item is rejected, the batch is rejected and retained only as audit evidence.
- If some items are accepted and some are rejected, MEL records `accepted_partial_bundle`.
- Accepted items stay separate from rejected items; MEL does not silently drop rejected rows or silently upgrade them to trusted evidence.

For every batch MEL persists:

- local importing instance id
- local site/fleet scope snapshot
- source type, source name, and source path when known
- claimed origin instance/site/fleet scope
- exported_at and imported_at
- batch validation JSON
- raw batch payload JSON
- accepted/rejected counts

For every imported item MEL persists:

- raw imported bundle JSON
- extracted evidence JSON
- optional remote event JSON
- typed validation JSON
- local import batch/source linkage
- claimed origin vs evidence origin
- observed/received/recorded/imported timing fields
- timing posture
- merge disposition and merge correlation id

## Timing and ordering posture

MEL preserves the difference between:

- `observed_at`: when the remote observer says the evidence was observed
- `received_at`: when the remote observer says it received it
- `recorded_at`: when the remote observer says it persisted or materialized it
- `imported_at`: when this MEL instance imported the payload

Timeline materialization stays honest:

- `remote_import_batch` is ordered locally by this instance's import time
- `remote_evidence_import_item` is the local audit event for one imported row
- `remote_event_materialized` preserves the remote event/evidence timing basis when present

Possible timing postures include:

- `local_ordered`
- `imported_preserved_order`
- `receive_time_differs_from_observed_time`
- `import_time_not_equal_event_time`
- `ordering_uncertain_missing_timestamps`
- `historical_import_not_live`

These are investigation aids, not proof of fleet-wide total order.

## Merge and provenance inspection

MEL does not silently collapse imported evidence into local observations.

Operators can inspect:

- local vs imported provenance
- claimed origin vs local import context
- validation outcome and reason codes
- batch/source lineage
- related imported rows
- merge classification and merge key
- timing posture and remaining unknowns

This is intentionally inspectable because repeated or similar remote observations do **not** prove RF flooding, congestion, route stability, or coverage certainty.

## Operator workflows

- CLI import: `mel fleet evidence import --file <payload.json> --config <cfg> [--strict-origin]`
- CLI list items: `mel fleet evidence list --config <cfg> [--batch <batch-id>]`
- CLI show item: `mel fleet evidence show <import-id> --config <cfg>`
- CLI list batches: `mel fleet evidence batches --config <cfg>`
- CLI show batch: `mel fleet evidence batch-show <batch-id> --config <cfg>`
- CLI validate only: `mel import validate --bundle <path>`
- API import/list/detail: `/api/v1/fleet/remote-evidence`
- API batch list/detail: `/api/v1/fleet/imports` and `/api/v1/fleet/imports/{id}`
- API timeline list/detail: `/api/v1/timeline` and `/api/v1/timeline/{id}`
- API merge explanation: `/api/v1/fleet/merge-explain`

## Support bundle behavior

Support bundles now include:

- `imported_evidence.json`
- `remote_evidence_export.json`
- remote import timeline rows inside `timeline.json`

`remote_evidence_export.json` is a canonical `mel_remote_evidence_batch` export of imported rows for offline handoff or re-import. It is still offline-only and authenticity-unverified by default.

## Scope boundary

Core MEL keeps `federation_read_only_evidence_ingest = offline_bundle_import_local_sqlite`.

That means:

- imported evidence is read-only,
- remote evidence import does not create a remote control channel,
- live multi-site sync is still out of scope here,
- the local instance remains the authority for what it imported and how it interpreted it.
