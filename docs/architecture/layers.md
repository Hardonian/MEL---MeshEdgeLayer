# Layers

## Layer 1 — Transport edge

- `internal/transport`
- Owns connectivity, MQTT subscribe, health state, packet counters, and unsupported capability reporting.

## Layer 2 — Mesh state core

- `internal/db`, `internal/meshstate`, `migrations/`
- Persists truthful observations and keeps a lightweight in-memory summary for the UI.

## Layer 3 — Policy + privacy engine

- `internal/policy`, `internal/privacy`, `internal/retention`
- Produces machine-readable findings and recommendations.

## Layer 4 — Service / API layer

- `internal/service`, `internal/web`
- Starts transport loops, records audit logs, and exposes versioned local endpoints.

## Layer 5 — Operator UX layer

- `cmd/mel`, web UI in `internal/web`
- Implements init, doctor, status, nodes, privacy, policy, export, import validation, and backup flows.

## Layer 6 — Packaging + runtime ops layer

- `scripts/`, `docs/ops/`, `docs/ops/systemd/mel.service`
- Documents install, upgrade, rollback, and service hardening.

## Layer 7 — Extension layer

- `internal/plugins`
- Minimal alert plugin boundary only. No marketplace or fake plugin runtime is claimed.
