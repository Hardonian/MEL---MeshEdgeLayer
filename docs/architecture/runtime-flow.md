# Runtime flow

1. `mel serve` loads defaults, JSON config, and `MEL_*` environment overrides.
2. Config validation rejects unsafe remote bind unless operators deliberately set the unsafe dev override.
3. The daemon opens SQLite, applies deterministic migrations through the `sqlite3` CLI, and runs retention before transport ingest begins.
4. Each enabled transport enters its own reconnect loop.
5. Supported transport payloads are normalized into Meshtastic envelope observations.
6. Messages and node observations are persisted into local state.
7. Privacy findings and policy recommendations are evaluated from actual config state.
8. The local API and UI present truthful health, node inventory, privacy findings, and event logs.

This is the only claimed runtime path in RC1. MEL does not claim live BLE or serial transport support because the code does not implement those paths yet.
