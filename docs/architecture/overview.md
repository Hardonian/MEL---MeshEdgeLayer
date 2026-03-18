# Architecture overview

MEL RC1 is a single-process local edge service with one CLI and one daemon runtime.

## Major runtime pieces

- `cmd/mel`: operator CLI.
- `cmd/mel-agent`: thin daemon entrypoint that starts the same service runtime.
- `internal/config`: defaults, JSON load, env overrides, validation, lints.
- `internal/transport`: transport implementations and capability reporting.
- `internal/service`: transport loops, shared ingest, persistence, audit logging.
- `internal/db`: SQLite access through the `sqlite3` CLI.
- `internal/web`: local UI and JSON API.
- `internal/privacy` / `internal/policy` / `internal/retention`: operator posture and hygiene logic.

## Current dataflow

1. Load config.
2. Validate config and compute lints.
3. Open SQLite and apply migrations.
4. Run retention.
5. Start enabled transport loops.
6. Normalize supported packets into MEL's envelope shape.
7. Persist messages, nodes, telemetry, and audit events.
8. Serve local UI/API from the same state.

## Current scope boundary

MEL is currently an ingest-and-observability layer. It is not a radio control plane, not a firmware extension, and not a replacement for stock Meshtastic clients.

See also:

- `docs/architecture/runtime-flow.md`
- `docs/architecture/transport-flow.md`
- `docs/product/what-mel-is-not.md`
