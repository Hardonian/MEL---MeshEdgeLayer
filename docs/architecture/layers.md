# Layers

## Layer 1 — Transport edge

- `internal/transport`
- Owns connectivity, serial/TCP direct readers, MQTT subscribe handling, health state, packet counters, and explicit unsupported transport reporting.

## Layer 2 — Mesh normalization + state core

- `internal/meshtastic`, `internal/db`, `internal/meshstate`, `migrations/`
- Parses the supported protobuf subset, stores truthful observations, and keeps a lightweight in-memory summary for the UI.

## Layer 3 — Policy + privacy engine

- `internal/policy`, `internal/privacy`, `internal/retention`
- Produces machine-readable findings and recommendations from current config and local policy rules.

## Layer 4 — Service / API layer

- `internal/service`, `internal/web`
- Starts transport loops, records audit logs, and exposes versioned local endpoints.

## Layer 5 — Operator UX layer

- `cmd/mel`, web UI in `internal/web`
- Implements init, doctor, status, nodes, node inspection, transport inspection, privacy, policy, export, import validation, backup, and local troubleshooting flows.

## Layer 6 — Packaging + runtime ops layer

- `scripts/`, `docs/ops/`, `docs/ops/systemd/mel.service`
- Documents install, upgrade, rollback, service hardening, and evaluation workflows.

## Layer 7 — Extension layer

- `internal/plugins`
- Minimal alert plugin boundary only. No marketplace or fake plugin runtime is claimed.
