# ADR-0002: Runtime layers and truthful current scope

## Status
Accepted

## Decision
MEL keeps one daemon and one CLI, with a shared service core.

Current transport claims are intentionally limited to:

- supported ingest from `serial`, `tcp`, and `mqtt`,
- explicit unsupported status for `ble` and `http`,
- no send, publish, metadata fetch, or node-fetch control path claims.

## Consequences

- Operators get a small runtime with transport health that can be verified locally.
- CLI, API, and UI reuse the same privacy, policy, and transport capability primitives.
- Future transport work must earn support status through code, tests, and operator docs before the README or release docs can promote it.
