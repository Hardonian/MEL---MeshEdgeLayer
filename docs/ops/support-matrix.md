# MEL Support Matrix

This matrix defines what MEL supports today and what it does not claim.

## Ingest transport support

| Transport surface | Status | Truth boundary |
|---|---|---|
| Serial (direct node) | Supported | Live claims require persisted ingest evidence. |
| TCP (direct node) | Supported | Endpoint must provide Meshtastic framing; claims remain evidence-bounded. |
| MQTT ingest | Supported | Subscribe ingest path only; disconnects and partial ingest must remain explicit. |
| BLE ingest | Unsupported | No ingest implementation; label unsupported everywhere. |
| HTTP ingest | Unsupported | No ingest implementation; no optimistic wording. |

## Platform support

| Platform | Status | Notes |
|---|---|---|
| Linux amd64 | Supported | Primary target. |
| Linux arm64 (including Raspberry Pi) | Supported | Common edge deployment path. |
| Termux (Android) | Supported with caveats | Useful for field/dev workflows; service model differs from systemd. |

## Control mode support

| Mode | Status | Use |
|---|---|---|
| `disabled` | Supported | Observe only; no action execution. |
| `advisory` | Supported (default) | Guidance and recommendations without execution. |
| `guarded_auto` | Supported | Executes only when policy and actuator reality allow it. |

## Action reality

The following remain recommendation-only until an actuator path exists:
- `temporarily_deprioritize`
- `temporarily_suppress_noisy_source`
- `clear_suppression`

## Explicit non-claims

- MEL is not a mesh routing/transmit stack.
- MEL does not provide BLE or HTTP ingest.
- MEL does not prove RF coverage/propagation by itself.
