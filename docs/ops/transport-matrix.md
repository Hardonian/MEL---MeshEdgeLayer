# Transport matrix

This matrix documents what MEL actually implements today.

| Transport | Implementation status | Config method | Ingest | Send | Metadata fetch | Node fetch | Health reporting | Verification | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `serial` | Supported | `serial_device`, `serial_baud` | Yes | No | No | No | Yes | `./bin/mel doctor`, `./bin/mel transports list`, UI/API transport counters | Uses Meshtastic stream framing over a local serial device. MEL runs `stty`, opens the device read/write, and consumes `FromRadio` packet frames. |
| `tcp` | Supported | `tcp_host` + `tcp_port` or `endpoint` | Yes | No | No | No | Yes | `./bin/mel doctor`, `./bin/mel transports list`, UI/API transport counters | Uses the same Meshtastic stream framing over TCP. Endpoint must be a Meshtastic-compatible framed stream. |
| `serialtcp` | Implemented but partial | `endpoint` | Yes | No | No | No | Yes | `./bin/mel config validate`, `./bin/mel transports list` | Runtime code treats this as a direct stream alias. MEL does not yet ship a separate hardened operator workflow or example config for it. |
| `mqtt` | Supported | `endpoint`, `topic`, `client_id` | Yes | No | No | No | Yes | `./bin/mel transports list`, UI/API packet counters, persisted observations | Existing MQTT subscribe path remains supported. `mel doctor` intentionally does not perform broker reachability probes. |
| `http` | Unsupported | Feature-gated only | No | No | No | No | Yes | `./bin/mel transports list` | Explicitly unsupported in this release. |
| `ble` | Unsupported | Feature-gated only | No | No | No | No | Yes | `./bin/mel transports list` | Explicitly unsupported in this release. |

## Selection rules

1. Prefer exactly one direct transport (`serial` or `tcp`) for the clearest operator story.
2. Hybrid direct + MQTT ingest is allowed, but MEL warns about contention and duplicate-observation risk.
3. MEL never reports a direct transport as healthy unless the underlying connection succeeded.
4. `connected but idle` is distinct from `live data flowing`.
5. Node inventory comes from observed packets, not from a transport-side control query.

## Evidence in code

- Transport construction: `internal/transport/transport.go`
- Direct serial/TCP stream handling: `internal/transport/direct.go`
- MQTT subscribe handling: `internal/transport/mqtt.go`
- Direct reachability checks in doctor: `cmd/mel/main.go`
- Shared ingest and persistence path: `internal/service/app.go`
