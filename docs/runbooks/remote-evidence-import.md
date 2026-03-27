# Remote evidence import (offline bundles)

This runbook matches the behavior shipped in core MEL: **file-based import into the local SQLite database**. It is **not** live multi-site sync, mesh-wide authority, or cryptographic proof of origin.

## What you can do

- Import a JSON bundle that wraps one canonical `evidence` object (`mel_remote_evidence_bundle`).
- Receive a **typed validation outcome** (accepted, accepted-with-caveats, rejected) with machine-readable reason codes.
- Inspect **audit rows** via API, CLI, and support bundles (`imported_remote_evidence` in `bundle.json`).
- See a **timeline** row `remote_evidence_import` with drilldown details (validation, merge inspection stub, ordering posture).

## What MEL does not conclude

- **Validation ≠ truth**: structural acceptance does not mean the remote site is healthy, reachable, or authoritative.
- **No authenticity by default**: origin fields are **claimed**; core does not verify signatures end-to-end unless you add that outside MEL.
- **No global order**: timeline order is **instance-local**; imported events preserve import/observation time distinctions in the stored envelope, not a total fleet order.
- **Repeated observers ≠ flooding proof**: merge/dedupe helpers classify keys only; they do not infer congestion or RF conclusions.

## Supported bundle shape

- `schema_version`: must be `"1.0"`.
- `kind`: must be `"mel_remote_evidence_bundle"`.
- `evidence`: required fields include `evidence_class`, `origin_instance_id`, `observation_origin_class` (known enum), and `physical_uncertainty_posture`.

Optional: `claimed_origin_instance_id`, `claimed_origin_site_id`, `claimed_fleet_id`, `import_context`.

## Scope conflicts

If this instance has `scope.site_id` configured, evidence `origin_site_id` must match or the import is **rejected**. Same for `claimed_*` when present. This prevents silent cross-site mis-association, not “fleet approval.”

## Operator workflows

- CLI: `mel fleet evidence import --file <bundle.json> --config <cfg>` (optional `--strict-origin`).
- CLI: `mel fleet evidence list|show <id>`.
- API: `POST /api/v1/fleet/remote-evidence` (body = raw JSON), `GET /api/v1/fleet/remote-evidence`, `GET /api/v1/fleet/remote-evidence/{id}`.
- Merge explain (dedupe semantics only): `GET /api/v1/fleet/merge-explain?key_a=...&key_b=...&same_observer=true|false`.

## Federation posture

Capability field `federation_read_only_evidence_ingest` is `offline_bundle_import_local_sqlite` in core: **read-only** with respect to remote execution; imports do not enable remote control paths.
