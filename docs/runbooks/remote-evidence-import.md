# Remote evidence import (offline bundles)

This runbook matches the behavior shipped in core MEL: **file-based import into the local SQLite database**. It is **not** live multi-site sync, mesh-wide authority, or cryptographic proof of origin.

## What you can do

- Import a canonical offline JSON bundle (`mel_remote_evidence_bundle`) with one `evidence` envelope and an optional matching `event` envelope.
- Receive a **typed validation outcome** (accepted, accepted-with-caveats, rejected) with machine-readable reason codes.
- Inspect **audit rows and normalized drilldown** via API, CLI, and support bundles (`imported_remote_evidence` plus `imported_remote_evidence_inspections` in `bundle.json`).
- See a **timeline** row `remote_evidence_import` with canonical evidence/import-event drilldown, timing posture, and related-evidence analysis.

## What MEL does not conclude

- **Validation ≠ truth**: structural acceptance does not mean the remote site is healthy, reachable, or authoritative.
- **No authenticity by default**: origin fields are **claimed**; core does not verify signatures end-to-end unless you add that outside MEL.
- **No global order**: timeline order is **instance-local**; imported events preserve import/observation time distinctions in the stored envelope, not a total fleet order.
- **Repeated observers ≠ flooding proof**: merge/dedupe helpers classify keys only; they do not infer congestion or RF conclusions.

## Supported bundle shape

- `schema_version`: must be `"1.0"`.
- `kind`: must be `"mel_remote_evidence_bundle"`.
- `evidence`: required fields include `evidence_class`, `origin_instance_id`, `observation_origin_class` (known enum), and `physical_uncertainty_posture`.
- `event` is optional, but if present it must be internally consistent with `evidence`:
  - `event_id`, `event_type`, `summary`, and `origin_instance_id` are required.
  - `event.origin_instance_id` must match `evidence.origin_instance_id`.
  - `event.correlation_id` must match `evidence.correlation_id` when both are set.

Optional top-level fields: `claimed_origin_instance_id`, `claimed_origin_site_id`, `claimed_fleet_id`, `import_context`.

## Accepted formats

- **Supported for `mel fleet evidence import`**: canonical `mel_remote_evidence_bundle` JSON only.
- **Not supported for remote-evidence import**: generic MEL exports/support bundles. `mel import validate` will identify them, but MEL will not treat them as canonical remote evidence without an explicit bundle wrapper.

## Scope conflicts

If this instance has `scope.site_id` configured, evidence `origin_site_id` must match or the import is **rejected**. Same for `claimed_*` when present. This prevents silent cross-site mis-association, not “fleet approval.”

## Operator workflows

- CLI: `mel fleet evidence import --file <bundle.json> --config <cfg>` (optional `--strict-origin`).
- CLI: `mel fleet evidence list` for summary posture and `mel fleet evidence show <id>` for raw row + normalized inspection.
- CLI: `mel import validate --bundle <path>` distinguishes canonical remote-evidence bundles from general MEL export bundles.
- API: `POST /api/v1/fleet/remote-evidence` (body = raw JSON), `GET /api/v1/fleet/remote-evidence`, `GET /api/v1/fleet/remote-evidence/{id}`.
- Merge explain (dedupe semantics only): `GET /api/v1/fleet/merge-explain?key_a=...&key_b=...&same_observer=true|false`.

For each imported row, MEL now preserves:

- claimed origin vs locally recorded import context,
- observed vs received vs recorded vs imported time,
- optional remote event envelope vs local import audit event,
- related imported rows and why MEL treated them as exact/near/conflicting/related without silently merging them.

## Federation posture

Capability field `federation_read_only_evidence_ingest` is `offline_bundle_import_local_sqlite` in core: **read-only** with respect to remote execution; imports do not enable remote control paths.
