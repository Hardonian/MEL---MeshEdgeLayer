# Transport support matrix

This document is the operator-facing truth table for MEL RC1. A row only receives a support label that matches the current code path.

## Matrix

| Transport / mechanism | Status | Config method | Verification method | Limitations / caveats |
| --- | --- | --- | --- | --- |
| `serial` direct-node | Implemented; code/test verified only in this repo pass | `type: "serial"`, `serial_device`, optional `serial_baud`, optional `reconnect_seconds` | `mel doctor`; `/api/v1/status`; UI transport table; `mel status`; persisted `messages` / `nodes` | Ingest only. Requires host serial access plus `stty`. MEL does not send or administer the radio. This pass did not include live hardware. |
| `tcp` direct-node | Implemented; code/test verified only in this repo pass | `type: "tcp"`, `tcp_host` + `tcp_port` or `endpoint` | `mel doctor`; `/api/v1/status`; UI transport table; persisted `messages` / `nodes` | Ingest only. Endpoint must speak Meshtastic stream framing. This pass did not include live hardware. |
| `mqtt` ingest | Implemented and repo-self-tested end-to-end | `type: "mqtt"`, `endpoint`, `topic`, `client_id` | `mel serve`; `/api/v1/status`; `/api/v1/messages`; `mel export`; repo-local MQTT self-test | Ingest only. `mel doctor` validates config posture but does not attempt broker reachability. |
| direct + MQTT hybrid ingest | Implemented but partial | enable more than one transport | `mel transports list`; config lint output; doctor output; packet persistence | Duplicate observations and radio ownership contention remain operator responsibilities. |
| `serialtcp` alias | Experimental / not hardened | `type: "serialtcp"`, `endpoint` | same health surface as direct TCP | Present in code, but not promoted as a primary operator workflow. |
| `ble` | Explicitly unsupported | `type: "ble"` | transport list / UI show `unsupported` | `features.ble_experimental` does not make BLE work. |
| `http` | Explicitly unsupported | `type: "http"` | transport list / UI show `unsupported` | No live device implementation exists. |
| publish / transmit | Planned / not implemented | none | n/a | `SendPacket` returns an error for supported transports. |
| metadata fetch | Planned / not implemented | none | n/a | `FetchMetadata` returns an error for supported transports. |
| node fetch from transport | Planned / not implemented | none | n/a | MEL derives node state from observed traffic only. |
| admin / config apply | Planned / not implemented | none | n/a | No radio control plane is exposed. |
| multi-node direct ownership | Explicitly unsupported as a product guarantee | multiple direct transports | lint + doctor only | MEL warns about contention; it does not arbitrate ownership. |

## Verification evidence in repo

- Transport construction: `internal/transport/transport.go`
- Serial/TCP direct ingest: `internal/transport/direct.go`
- MQTT ingest: `internal/transport/mqtt.go`
- Shared ingest and persistence: `internal/service/app.go`
- Operator-facing health: `internal/web/web.go`, `cmd/mel/main.go`
- Direct transport tests: `internal/transport/direct_test.go`
- MQTT transport tests: `internal/transport/mqtt_test.go`

## Selection rules

1. Prefer exactly one direct transport (`serial` or `tcp`) per deployment.
2. Treat MQTT as an ingest path, not a control path.
3. Treat hybrid direct + MQTT as a deliberate duplication risk until you have verified your deployment behavior.
4. If no transport is enabled, MEL starts and remains explicitly idle.

## Canonical transport status vocabulary

- `disabled`
- `configured_not_attempted`
- `configured_offline`
- `attempting`
- `connected_no_ingest_evidence`
- `ingesting`
- `historical_ingest_seen`
- `error`
- `unsupported`
