# Runtime flow

1. `mel serve` loads config and enforces operator config file permissions.
2. Config validation and linting surface risky posture such as remote exposure, long retention, unsupported BLE flags, and multi-transport contention.
3. MEL opens SQLite, applies deterministic migrations, and runs retention before ingest starts.
4. Each enabled transport enters a reconnect loop.
5. Transports emit explicit runtime states instead of silently assuming success.
6. Shared ingest persists packets, updates nodes, stores telemetry samples when available, and only then marks ingest as successful.
7. UI, CLI, `/api/v1/status`, and `/metrics` all read from the same persisted and runtime truth model.
