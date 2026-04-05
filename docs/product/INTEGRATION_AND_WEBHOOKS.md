# Integration surfaces and outbound webhooks

This document describes **implemented** integration paths in MEL. It does not imply live federation, cross-instance authority, or delivery guarantees beyond what the code enforces.

## Outbound webhooks (`integration` config)

When `integration.enabled` is true and destinations are configured, MEL posts versioned JSON envelopes (`schema_version: mel.integration.v1`) for selected event kinds. See `internal/integration/outbound.go` for the `Event` shape and `internal/service/integration_dispatch.go` for dispatch triggers (alerts, anomalies, transport state, control-action summaries).

- Delivery is **best-effort** with rate limiting per destination (`min_interval_seconds`).
- Failures are logged; they are **not** silently treated as success.
- Treat webhook payloads as **operator notifications**, not canonical proof of mesh runtime.

## Operator API: operational digest

`GET /api/v1/operator/digest` returns `mel.operator_operational_digest/v1`: deterministic **SQLite row counts** and a fixed **24-hour activity window** (incidents opened by `occurred_at`, control actions and operator notes by `created_at`) for **this instance only**.

- Requires `read_status` or `read_incidents` (same family as incidents list).
- Use for shift handoff, ticketing bridges, and cron pull — not as fleet-wide posture.

## Intelligence briefing

`GET /api/v1/intelligence/briefing` returns ranked operational issues derived from diagnostics and recent incidents. It uses **bounded heuristics**; pair with the digest and raw `/api/v1/incidents` / `/api/v1/timeline` for review.

- Requires `read_status` or `read_incidents`.

## UI

The console exposes **Operational review** (`/operational-review`) with digest, briefing, and JSON export of the digest for external tools.
