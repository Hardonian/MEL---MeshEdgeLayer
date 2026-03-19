# MEL canonical execution roadmap

This file is the execution contract for the work completed in this repository on 2026-03-19. It is not a marketing plan.

## MEL scope boundaries

### What MEL does

- Ingests real Meshtastic packets from configured serial, TCP, and MQTT transports.
- Persists packet, node, telemetry, and audit evidence locally in SQLite.
- Exposes truthful read-only operator diagnostics through CLI, UI, and JSON endpoints.
- Surfaces degraded and blocked states explicitly instead of inventing healthy traffic.

### What MEL does not do

- No routing, transmit, publish, or radio control-plane behavior.
- No unsupported transport claims for BLE or HTTP.
- No hardware-verification claim unless exercised with real hardware.
- No marking ingest as successful without a confirmed SQLite write.

### Definitions

- **Supported transport**: a transport path with real ingest code, doctor coverage, documentation coverage, and verification in this repo.
- **Verified**: demonstrated by code plus automated tests and/or smoke commands stored in the evidence pack. Hardware verification remains bounded to the hardware actually exercised.
- **Historical only**: MEL has persisted evidence for a transport, but the current command cannot prove a live connection.

## Phase 0 — canonical roadmap artifact

### Deliverables
- This file in `docs/roadmap/ROADMAP_EXECUTION.md`.

### Acceptance criteria
- Defines phases 0–8, deliverables, acceptance criteria, and verification steps.
- Defines scope boundaries, supported transport, and verified.

### Verification
- File exists in-repo and is committed.

## Phase 1 — operator trust + reality hardening

### Deliverables
- Doctor v2 with config validation, SQLite write/read checks, transport truth, and actionable findings.
- Shared status model for CLI and API.
- Explicit transport states: `disabled`, `configured_not_attempted`, `attempting`, `configured_offline`, `connected_no_ingest`, `ingesting`, `historical_only`, `error`.
- Ingest truth enforcement: runtime ingest only counts after SQLite writes.

### Acceptance criteria
- `mel doctor` and `/api/v1/status` use the same transport truth vocabulary.
- No transport reports ingest without persisted evidence.
- Failures are visible in doctor and logs.

### Verification
- `./bin/mel doctor --config <path>`
- `./bin/mel status --config <path>`
- `curl -fsS http://127.0.0.1:8080/api/v1/status`

## Phase 2 — observability + metrics

### Deliverables
- JSON `/metrics` endpoint.
- Structured JSON logging with `event_type`.
- `--debug` runtime flag.

### Acceptance criteria
- Metrics expose total messages, last ingest, per-transport counters, and ingest rate.
- Logs emit `transport_connected`, `transport_failed`, `ingest_received`, `ingest_dropped`, and `db_error`.

### Verification
- `curl -fsS http://127.0.0.1:8080/metrics`
- `./bin/mel serve --debug --config <path>`

## Phase 3 — transport hardening

### Deliverables
- Reconnect loops for direct and MQTT transports.
- Idle timeout handling and malformed-frame recovery for direct transports.
- MQTT topic mismatch detection.
- Duplicate detection and source labeling across transports.

### Acceptance criteria
- Disconnects do not silently hang the ingest loop.
- Duplicate packets are dropped truthfully.

### Verification
- `go test ./internal/transport/...`
- smoke test with MQTT reconnect and replay evidence.

## Phase 4 — data model + protobuf expansion

### Deliverables
- Message typing for `text`, `position`, `node_info`, `telemetry`, and `unknown`.
- Raw payload fallback for unsupported payloads.
- Deterministic migration support for multiple migration files.

### Acceptance criteria
- Database and APIs retain typed payload evidence plus raw fallback bytes.
- Backward-compatible migrations still open fresh databases cleanly.

### Verification
- `go test ./internal/db ./internal/meshtastic ./cmd/mel`
- `/api/v1/messages` and `mel replay` output typed payload metadata.

## Phase 5 — edge intelligence (strictly bounded)

### Deliverables
- Node tracking with last seen and message counts.
- Read-only filtering by node and message type.
- Read-only replay command.

### Acceptance criteria
- No send, routing, or control-plane claims were introduced.

### Verification
- `./bin/mel nodes --config <path>`
- `./bin/mel node inspect <id> --config <path>`
- `./bin/mel replay --config <path> --node <id> --type text`

## Phase 6 — security + safety

### Deliverables
- 0600 config enforcement for operator config files.
- Input validation for malformed transport data.
- Explicit non-claims documentation.

### Acceptance criteria
- Launch fails on overly-broad operator config permissions.
- Invalid input is surfaced as `error`, not silently ignored.

### Verification
- `./bin/mel config validate --config <path>`
- `./bin/mel doctor --config <path>` with bad permissions and bad transport input.

## Phase 7 — contributor + extensibility

### Deliverables
- Transport interface contract document.
- Protobuf extension guide.
- Definition of supported transport in contributor docs.

### Acceptance criteria
- New contributors can add a transport or message type without guessing acceptance rules.

### Verification
- Docs exist and align with code paths.

## Phase 8 — release maturity

### Deliverables
- Release checklist.
- Evidence pack.
- Changelog entry for the execution pass.

### Acceptance criteria
- Build, test, CLI, API, and smoke verification are captured.
- Remaining caveats are explicit.

### Verification
- `make build`
- `go test ./...`
- `./scripts/smoke.sh`
- evidence pack files under `docs/release/evidence/`
