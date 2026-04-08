# MEL current status

## Last updated: 2026-04-08

MEL is usable today for local-first mesh observability and incident workflows, with explicit boundaries.

## What is implemented now

- Serial/TCP direct ingest and MQTT ingest.
- Persistent evidence model in SQLite with operator-facing diagnostics.
- CLI + embedded web operator console via `mel serve`.
- Incident/control workflow surfaces with explicit lifecycle semantics.
- Deterministic fixture/demo mode for evaluation without radios.

## What is intentionally bounded

- BLE ingest: unsupported.
- HTTP ingest: unsupported.
- MEL as RF routing/transmit engine: not implemented.
- Local inference as canonical truth: not allowed.

## How to evaluate credibly

1. Run quickstart: [docs/getting-started/QUICKSTART.md](../getting-started/QUICKSTART.md).
2. Validate boundaries: [docs/ops/limitations.md](../ops/limitations.md) and [docs/community/claims-vs-reality.md](../community/claims-vs-reality.md).
3. Run verification chain from root README (`make lint`, `make test`, `make build`, `make smoke`).
4. For release-grade confidence, run `make premerge-verify`.

## Where contributions are most valuable now

- Better diagnostics and degraded-state clarity.
- Incident workflow polish (without widening claims).
- Runbooks and onboarding simplification.
- Transport reliability hardening within existing supported paths.
