# Configuration guide

MEL uses a single JSON config file and then applies a small set of `MEL_*` environment overrides.

## Config sections

### `bind`

- `api`: HTTP listen address for the local UI and API.
- `metrics`: reserved bind address for metrics-related expansion. MEL stores the value but does not yet expose a separate metrics server.
- `allow_remote`: when `false`, MEL normalizes empty-host binds to `127.0.0.1` and treats remote exposure as a deliberate action.

### `auth`

- `enabled`: enables HTTP basic auth for the built-in UI and API.
- `session_secret`: required to be at least 16 characters when auth is enabled.
- `ui_user` / `ui_password`: basic-auth credentials.
- `allow_insecure_remote`: explicit override for remote binds without auth. This exists for deliberate testing, not as a recommended deployment mode.

### `storage`

- `data_dir`: working directory for local MEL state.
- `database_path`: SQLite database path.
- `encryption_key_env` / `encryption_required`: config-level switches only. MEL validates the env var length requirement when encryption is required, but this release does not claim full at-rest encryption semantics beyond that validation path.

### `logging`

- `level`
- `format`

### `retention`

- `messages_days`
- `telemetry_days`
- `audit_days`
- `precise_position_days`

Retention runs at startup and records a retention job row after completion.

### `privacy`

- `store_precise_positions`
- `mqtt_encryption_required`
- `map_reporting_allowed`
- `redact_exports`
- `trust_list`

These values drive both `mel privacy audit` and the UI/API privacy summary.

### `transports`

Supported transport types in current code:

- `serial`
- `tcp`
- `serialtcp`
- `mqtt`
- `ble` as explicitly unsupported
- `http` as explicitly unsupported

Current per-type fields:

- MQTT: `endpoint`, `topic`, `client_id`, optional `username`, `password`
- Serial: `serial_device`, `serial_baud`
- TCP: `tcp_host`, `tcp_port`, or `endpoint`
- Direct reconnect behavior: `reconnect_seconds`
- `notes` is operator-facing metadata only

### `features`

- `web_ui`
- `metrics`
- `ble_experimental`

These fields are present in config and examples. Current runtime behavior does not use them as a full feature-flag system, so operators should not treat them as proof of implemented capability.

### `rate_limits`

- `http_rps`
- `transport_reconnect_seconds`

## Environment overrides actually consumed today

MEL currently consumes these env vars in code:

- `MEL_BIND_API`
- `MEL_BIND_METRICS`
- `MEL_BIND_ALLOW_REMOTE`
- `MEL_DB_PATH`
- `MEL_DATA_DIR`
- `MEL_AUTH_ENABLED`
- `MEL_SESSION_SECRET`
- `MEL_UI_USER`
- `MEL_UI_PASSWORD`
- `MEL_AUTH_ALLOW_INSECURE_REMOTE`
- `MEL_PRIVACY_STORE_PRECISE_POSITIONS`
- `MEL_PRIVACY_MAP_REPORTING_ALLOWED`
- `MEL_PRIVACY_MQTT_ENCRYPTION_REQUIRED`
- `MEL_RETENTION_MESSAGES_DAYS`

No other `MEL_*` override should be assumed to work unless it is added to `internal/config/config.go`.

## Validation and linting

Use:

```bash
./bin/mel config validate --config ./mel.json
./bin/mel doctor --config ./mel.json
./bin/mel privacy audit --config ./mel.json
./bin/mel policy explain --config ./mel.json
```

What MEL validates today:

- required bind and storage paths,
- positive retention windows,
- remote bind posture,
- enabled transport completeness,
- duplicate enabled transport names,
- transport contention warnings,
- MQTT topic and encryption posture lints,
- direct serial/TCP reachability checks in `doctor`.

## Example configs

- `configs/mel.example.json`: mixed example with serial preferred, TCP and MQTT disabled.
- `configs/mel.serial.example.json`: simplest direct-node serial deployment.
- `configs/mel.tcp.example.json`: direct TCP deployment.
- `configs/mel.hybrid.example.json`: direct + MQTT ingest with explicit caveats.
- `configs/mel.mqtt-only.example.json`: MQTT-only ingest.
