# Contributing to MEL

## Ground rules

- Keep MEL honest: no fake transports, no fake mesh data, no dead routes.
- Narrow public claims when implementation proof is missing.
- Preserve stock Meshtastic compatibility boundaries.
- Prefer stdlib-only Go unless a dependency is already vendored.
- Make degraded states explicit in CLI, API, and UI.
- Add tests with any behavior change that affects operator truth, privacy, config, transport health, or persistence.

## Repo truths contributors should know

- SQLite access intentionally uses the `sqlite3` CLI in this environment.
- MEL currently supports ingest from serial direct-node, TCP direct-node, and MQTT.
- BLE, HTTP ingest, send/publish, metadata fetch, and admin/control are not current features.
- Metrics config exists, but RC1 does not ship a metrics server.
- `storage.encryption_required` is not at-rest encryption; do not document it that way.

## Build and verify

```bash
make build
make test
./scripts/smoke.sh
```

Use `make verify` when you want the full repo pass, including cross-builds.

## Safe extension areas

- policy and privacy logic
- docs and operator workflows
- export and backup workflows
- transport hardening backed by code and tests
- protobuf decode expansion backed by fixtures/tests

## Transport changes

A transport can only be documented as supported when all of the following are true:

1. there is a real implementation path in `internal/transport`,
2. `mel doctor`, CLI/API/UI, or both expose truthful health for it,
3. there is repo-local verification coverage,
4. docs state its caveats precisely.

## Pull requests

Include:

- design intent,
- operator impact,
- verification evidence,
- residual risk or remaining limitations.
