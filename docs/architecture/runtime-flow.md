# Runtime flow

1. `mel serve` loads defaults, JSON config, and the env overrides implemented in `internal/config/config.go`.
2. Config validation rejects incomplete transport configuration and unsafe remote bind posture unless the explicit insecure override is set.
3. The daemon opens SQLite, applies deterministic migrations through the `sqlite3` CLI, and runs retention before transport ingest begins.
4. Each enabled transport enters its own reconnect loop.
5. Supported transport payloads are normalized into a shared Meshtastic envelope path.
6. Messages, nodes, telemetry samples, and audit evidence are persisted into local state.
7. Privacy findings and policy recommendations are evaluated from actual config state.
8. The local API and UI present truthful health, node inventory, recent messages, privacy findings, and event logs.

Current supported ingest paths are:

- direct serial,
- direct TCP,
- MQTT subscribe.

Current explicitly unsupported ingest paths are:

- BLE,
- HTTP.
