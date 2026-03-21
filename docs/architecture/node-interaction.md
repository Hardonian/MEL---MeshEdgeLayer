# Node Interaction & Status Lifecycle

MEL tracks the lifecycle of every node observed on the mesh. Unlike a simple "last seen" timer, MEL maintains a status matrix based on transport-specific evidence.

## Node Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Discovered: Packet Received
    Discovered --> Active: Valid Ingest (>1 pk/min)
    Active --> Stale: No ingest > 15m
    Stale --> Active: Packet Received
    Stale --> Lost: No ingest > 2h
    Lost --> Active: Packet Received
```

## Telemetry Evidence

MEL doesn't just display telemetry; it reconstructs the history of a node's performance.

![Meshtastic Node Telemetry](/c:/Users/scott/.gemini/antigravity/brain/3857245b-4abd-4d41-9b8b-41da0a674b43/meshtastic_node_telemetry_isometric_1774057734657.png)

### Key Metrics Tracked

- **Signal Quality (SNR/RSSI)**: Correlated across multiple transports if available.
- **Battery/Voltage**: Extracted from telemetry packets and bucketed for trend analysis.
- **Hop Count**: Used to estimate position in the mesh topology.
- **Message Velocity**: Frequency of user and system messages.

*MEL — Truthful Local-First Mesh Observability.*
