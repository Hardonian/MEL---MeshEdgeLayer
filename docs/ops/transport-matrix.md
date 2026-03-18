# Transport matrix

| Transport | Implementation status | Ingest | Send | Metadata fetch | Node fetch | Health reporting | Config apply | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `serial` | Supported | Yes | No | No | No | Yes | No | Uses Meshtastic stream framing over a local serial device. MEL configures the port with `stty` and then reads real `FromRadio` packet frames. |
| `tcp` | Supported | Yes | No | No | No | Yes | No | Uses the same Meshtastic stream framing over TCP. |
| `mqtt` | Supported | Yes | No | No | No | Yes | No | Existing MQTT ingest path remains supported. |
| `http` | Unsupported | No | No | No | No | Yes | No | Explicitly feature-gated. |
| `ble` | Unsupported | No | No | No | No | Yes | No | Explicitly feature-gated. |

## Selection rules

1. Prefer exactly one direct transport (`serial` or `tcp`).
2. If both direct and MQTT are enabled, MEL ingests from both but surfaces contention warnings.
3. MEL never reports a direct transport as healthy unless the underlying connection succeeded.
4. `connected but idle` is distinct from `live data flowing`.
