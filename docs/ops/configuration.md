# Configuration

## Security rules

- Production operator config files must be `0600`.
- `mel config validate` and `mel doctor` report overly-broad config permissions as findings.
- `mel serve` refuses to start when the config file mode is broader than `0600`.

## Bind settings

- `bind.api`: primary HTTP listener for UI, API, and JSON `/metrics`.
- `bind.metrics`: reserved knob only; MEL does not start a second metrics listener.
- `allow_remote`: remote bind requires explicit authentication posture.

## Feature flags

- `features.web_ui=false` disables the HTML UI but keeps JSON endpoints.
- `features.metrics` does not add another listener; scrape `/metrics` from `bind.api`.
- `features.ble_experimental` does not make BLE supported.

## Transport config

Use one or more entries under `transports`.

Supported ingest types:

- `serial`
- `tcp`
- `mqtt`
- `serialtcp` as a partial alias of the direct TCP reader

Unsupported types remain explicitly unsupported:

- `ble`
- `http`

### Reliability knobs

These fields are normalized with deterministic defaults when omitted:

- `mqtt_qos`: defaults to `1` for MQTT ingest so the broker must at least ack delivery.
- `mqtt_keepalive_seconds`: defaults to `30`.
- `mqtt_clean_session`: defaults to `false`, so MEL requests a persistent broker session instead of claiming stateless behavior.
- `read_timeout_seconds`: defaults to `15` for direct and MQTT readers.
- `write_timeout_seconds`: defaults to `5`.
- `max_consecutive_timeouts`: defaults to `3`; after that MEL treats the transport as stalled and forces a reconnect.

These knobs harden MEL's own ingest loop. They do **not** prove broker persistence, radio health, RS485 bus wiring, or field-hardware correctness on their own.

## Runtime truth

MEL uses the same transport states everywhere:

- `disabled`
- `configured_not_attempted`
- `attempting`
- `configured_offline`
- `connected_no_ingest`
- `ingesting`
- `historical_only`
- `error`

Heartbeat activity, consecutive timeout counters, retry status, and dead-letter counts are exposed through `/api/v1/status`, `/api/v1/transports`, `/metrics`, and the HTML UI transport table.

The example config files include deterministic timeout and keepalive knobs. Runtime evidence such as `last_heartbeat_at` and `consecutive_timeouts` is not configured manually; it is produced by MEL at runtime and stored in SQLite for later inspection.
