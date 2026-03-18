# Transport matrix

| Transport | Implementation status | Ingest | Send | Metadata fetch | Node fetch | Health reporting | Config apply | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `serial` | Supported | Yes | No | No | No | Yes | No | Uses Meshtastic stream framing over a local serial device. MEL configures the port with `stty ... min 0 time 10` and reads real `FromRadio` packet frames without mock traffic. |
| `tcp` | Supported | Yes | No | No | No | Yes | No | Uses the same Meshtastic stream framing over TCP. |
| `mqtt` | Supported | Yes | No | No | No | Yes | No | Existing MQTT ingest path remains supported. |
| `http` | Unsupported | No | No | No | No | Yes | No | Explicitly feature-gated. |
| `ble` | Unsupported | No | No | No | No | Yes | No | Explicitly feature-gated. |

## Selection rules

1. Prefer exactly one direct transport (`serial` or `tcp`).
2. If both direct and MQTT are enabled, MEL ingests from both but surfaces contention warnings.
3. MEL never reports a direct transport as healthy unless the underlying connection succeeded.
4. `connected but idle` is distinct from `live data flowing`.

## Hybrid dedupe guarantee

MEL dedupes cross-transport observations by hashing the canonical mesh packet bytes. That works when direct and MQTT expose the same packet payload. It does **not** guarantee collapse of duplicates when a transport rewrites or omits packet bytes outside that canonical packet body, so production operators should still verify hybrid behavior on their own node path.
