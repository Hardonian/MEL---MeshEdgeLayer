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
