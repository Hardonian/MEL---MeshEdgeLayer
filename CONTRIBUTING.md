# Contributing to MEL

## Ground rules

- Keep MEL honest: no fake transports, no mock mesh dashboards, no dead routes.
- Preserve truthful stock Meshtastic positioning.
- Prefer stdlib-only Go unless a dependency is already vendored.
- Make degraded states explicit in CLI, API, UI, docs, and scripts.
- Add tests with any behavior change that affects operator truth, privacy, config, transport behavior, or persistence.

## Repo structure at a glance

- `cmd/mel`: operator CLI and the primary `serve` entrypoint.
- `cmd/mel-agent`: minimal daemon wrapper.
- `internal/config`: config schema, normalization, validation, and env overrides.
- `internal/transport`: supported/unsupported transport implementations and capability reporting.
- `internal/meshtastic`: protobuf subset parsing and direct-frame normalization.
- `internal/service`: shared ingest and runtime orchestration.
- `internal/web`: local UI and HTTP API.
- `internal/db`, `migrations/`: SQLite persistence and schema.
- `docs/ops`: install, evaluation, transport, and troubleshooting docs.

## Safe extension points

- policy/privacy logic,
- exporter plugins,
- alert plugins,
- docs and operator workflows,
- protobuf subset extensions that are backed by tests and truthful docs.

Core transport ownership and storage semantics should be changed only when the implementation and tests stay truthful.

## Transport acceptance criteria

Do not mark a transport as supported until all of the following are true:

1. the config schema accepts the required fields,
2. `transport.Build` wires it into runtime construction,
3. `Connect`, `Subscribe`, and `Health` expose real behavior,
4. degraded and unsupported states are machine-visible,
5. `mel doctor` and operator docs explain how to verify it,
6. tests cover the implemented success/failure path,
7. README and `docs/ops/transport-matrix.md` are updated in the same change.

## Protobuf / message support criteria

When adding packet parsing support:

- document exactly which payload or port number is now interpreted,
- persist only fields MEL can explain and verify,
- avoid claiming whole-protocol support when only a subset is implemented,
- add or update tests in `internal/meshtastic` and any impacted ingest/storage paths.

## Typical workflow

```bash
make verify
./scripts/smoke.sh
```

If docs or examples change, also validate the commands you documented.

## Pull requests

Include:

- design intent,
- operator impact,
- verification evidence,
- residual risk or limitations,
- doc updates whenever transport, config, CLI, or API truth changed.
