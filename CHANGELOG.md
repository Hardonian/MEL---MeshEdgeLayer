# Changelog

## Unreleased

### Topology intelligence (ingest-derived graph, API, UI, CLI)

- **Ingest:** On each new mesh packet, MEL records a source-attributed `node_observations` row and upserts `topology_links` from `relay_node` and unicast `to_node` (labeled in snapshot explanations as packet evidence, not RF proof).
- **Service:** Topology store is wired into the web server; a background worker refreshes stale flags, recomputes scores, persists snapshots, and prunes observation/snapshot history per `topology` config.
- **API:** `GET /api/v1/topology` intelligence bundle; `GET /api/v1/topology/links/{edge_id}` and `GET /api/v1/topology/segments/{cluster_id}` drilldowns; node detail responses now include scored factors and next actions.
- **Web:** `/topology` page with SVG graph (dashed = non-observed), optional redacted coordinate scatter when `privacy.map_reporting_allowed` and positions exist.
- **CLI:** `mel inspect topology [--refresh]` reads the local DB (optional on-demand refresh).

### Readiness API parity, diagnostics UI, support bundle doctor.json

- **API:** `GET /api/v1/readyz` mirrors `GET /readyz` with a stable JSON contract (`api_version`, `status`, `reason_codes`, `components`, etc.). HTTP **503** when enabled transports are not ingesting or the status snapshot cannot be built; explicit **idle** (no enabled transports) returns **200**.
- **Web:** Diagnostics page includes an operator readiness panel wired to `/api/v1/readyz` and `/api/v1/status`; `/api/v1/diagnostics` returns `{ generated_at, summary, findings }` for the UI.
- **Support bundle:** ZIP from `mel support bundle` and `GET /api/v1/support-bundle` includes `doctor.json` (same logic as `mel doctor`, with bundle redaction: no raw config path; `config_inspect` fingerprint only). `bundle.json` carries `doctor_json` metadata when generated with a known config path.
- **CLI:** Doctor implementation lives in `internal/doctor` for reuse by CLI and bundles.

### Operator preflight, readiness clarity, support bundles, and control evidence

- **CLI:** `mel preflight` runs the same checks as `mel doctor`, optionally probes `GET /healthz` on `bind.api` (loopback-safe for `0.0.0.0`), and emits `operator_next_steps` plus `preflight_ok` in JSON.
- **API:** `/readyz` returns `snapshot_generated_at`, `schema_version`, `operator_next_steps`, and uses HTTP 503 with `error_class` when the status snapshot cannot be built (process up, evidence unavailable).
- **Support bundle:** `mel support bundle` / `internal/support` now includes status snapshot, operator panel, upgrade readiness, control-plane trust snapshot, recent control actions/decisions, incidents, active transport alerts, and privacy summary (config remains redacted via `privacy.RedactConfig`).
- **Control plane:** `persistentEvidence` for `backoff_increase` now treats `observation_drops` with `count=0` and `evidence_loss` anomaly rows as first-class evidence (regression test added).
- **Topology scoring:** Contradicted links are capped low so scores cannot read as healthy when observations conflict.
- **Web:** Topology handlers use package-level `writeJSON` (fixes `go vet` / build break).

### Phase 8 - Release Maturity
**Status: COMPLETED** - 2026-03-20

Phase 8 (Release Maturity) has been completed. All verification items from the release checklist have been verified and documented:
- Documentation alignment verified
- Build verification passed (`make build`)
- Test verification mostly passed (11/14 packages)
- Smoke test passed
- CLI and API verification completed
- Failure scenarios tested

See `docs/release/RELEASE_CHECKLIST.md` for full details.

### Added
- Canonical execution roadmap and release evidence documentation.
- Doctor v2, shared status snapshot logic, replay filtering, and JSON metrics endpoints.
- Structured event logging with debug mode and transport truth counters.
- Contributor transport-contract and protobuf-extension guides.

### Changed
- Transport state reporting now uses one explicit vocabulary across CLI, UI, and API.
- Runtime ingest is counted only after SQLite writes succeed.
- Message persistence now labels typed payload evidence and preserves raw payload fallback.
- Config validation now enforces `0600` operator config permissions for production use.

### Fixed
- Direct and MQTT ingest loops now surface disconnects, malformed input, duplicates, and handler failures explicitly.
- Smoke and verification flows now exercise metrics, status, and replay instead of relying on partial status output.
