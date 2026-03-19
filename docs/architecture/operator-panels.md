# Operator panels: CLI, Web UI, and simple device menu

## Purpose

MEL now exposes a compact operator panel so teams can inspect transport truth quickly without losing the underlying evidence model.

This panel is intentionally read-only. It does not imply control-plane support, OTA management, or on-device firmware support beyond the planning guidance documented here.

## Surfaces

### CLI

- `mel panel --config <path>` prints a compact text panel.
- `mel panel --format json --config <path>` emits the same operator state as structured JSON for scripting.

### Web UI / API

- `/api/v1/panel` returns the compact panel model.
- The Web UI adds an **Instrument panel** section that shows operator state, a compact summary, and short commands.

### 8-bit / constrained-device concept

For future extension-node work, use the panel vocabulary below for very small displays:

- `A State` — overall operator state and ingest truth
- `B Link` — transport state plus last error
- `C Msgs` — persisted/runtime message counters
- `D Retry` — reconnect attempts and offline guidance

This is a planning contract only. The repository does not currently ship extension-node firmware.

## Design rules

1. The compact panel must be derived from the same status truth as `mel status` and `/api/v1/status`.
2. `ready` is reserved for confirmed live ingest backed by SQLite writes.
3. `degraded` is used when MEL has partial evidence only: attempting, offline, connected-no-ingest, historical-only, or error.
4. `idle` is explicit when no transport is configured or no runtime proof exists yet.
5. Short commands must stay mnemonic and operator-safe: inspect only, never imply write/control actions that MEL does not ship.
