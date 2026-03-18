# Runtime flow

1. `mel serve` or `mel-agent` loads defaults, JSON config, and the supported `MEL_*` environment overrides.
2. Config validation rejects malformed transport setup and unsafe remote bind unless the explicit insecure override is set.
3. Config linting surfaces risky but still-runnable posture such as remote exposure, long retention, unsupported BLE flags, placeholder metrics config, and multi-transport contention.
4. MEL opens SQLite, applies deterministic migrations through the `sqlite3` CLI, and runs retention before transport ingest begins.
5. Each enabled transport enters its own reconnect loop.
6. Supported transport payloads normalize into Meshtastic envelope observations.
7. Ingest persists messages, nodes, telemetry samples, and audit evidence.
8. Privacy findings and policy recommendations are evaluated from active config state.
9. The local UI and versioned JSON API present transport health, node inventory, messages, privacy findings, recommendations, and events.

## Scope truth

This runtime currently claims real ingest support for:

- serial direct-node,
- TCP direct-node,
- MQTT.

It does **not** claim live support for:

- BLE,
- HTTP transport ingest,
- send/control/admin operations,
- a metrics listener.
