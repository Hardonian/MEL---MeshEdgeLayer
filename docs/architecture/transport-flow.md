# Transport flow

## Direct-node flow

1. MEL loads config and validates the enabled transports.
2. The service starts one transport loop per enabled transport.
3. For `serial`, MEL configures the port with `stty`, opens the device, and reads Meshtastic stream frames.
4. For `tcp`, MEL connects to the configured endpoint and reads the same Meshtastic stream frames.
5. Direct-node frames are decoded from `FromRadio` packet payloads into MEL's canonical envelope shape.
6. The shared ingest path persists messages, updates nodes, stores telemetry samples when present, increments state counters, and emits events.
7. On disconnect or read failure, the transport health flips to unhealthy, captures the last error and disconnect time, and the service retries with backoff.

## Canonical normalization

- MQTT and direct-node ingest both normalize into the same local message/node state path.
- Dedupe hashes are derived from the packet bytes, not the transport wrapper, so identical packets from different ingest paths can converge.
- Topology is never invented; only explicitly observed message and node facts are stored.
