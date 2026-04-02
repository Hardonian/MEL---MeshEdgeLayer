# Compatibility and Support Matrix

## Runtime platforms

| Platform | Status | Notes |
|---|---|---|
| Linux amd64 | Supported | Primary deployment target. |
| Linux arm64 | Supported | Includes Raspberry Pi-class edge systems. |
| Termux (Android) | Supported with caveats | Service lifecycle differs from systemd hosts. |

## Transport compatibility

| Transport | Ingest | Notes |
|---|---|---|
| Serial | Yes | Direct node path. |
| TCP | Yes | Meshtastic framing required. |
| MQTT | Yes | Broker/runtime disconnect semantics must be explicit. |
| BLE | No | Unsupported. |
| HTTP | No | Unsupported. |

## Control compatibility

| Mode | Status | Notes |
|---|---|---|
| disabled | Supported | Observation only. |
| advisory | Supported | Recommendations without execution. |
| guarded_auto | Supported with policy bounds | Requires safety checks and compatible actuator path. |
