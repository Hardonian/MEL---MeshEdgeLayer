# Known limitations

This page is intentionally blunt. It lists the current edges MEL does **not** claim to close.

## Transport limitations

- MEL supports ingest from `serial`, `tcp`, and `mqtt` today.
- `serialtcp` exists as a direct-stream alias in config/runtime code, but MEL does not yet ship a separate operator guide or hardening story for it.
- MEL does **not** claim BLE transport support.
- MEL does **not** claim HTTP transport support.
- MEL does **not** claim send, publish, radio control, or config-apply transport behavior.
- MEL does **not** claim transport failover beyond retry loops per enabled transport.

## Meshtastic protocol limitations

- Protobuf handling is partial.
- MEL currently parses the envelope, base packet fields, user payloads, and position payloads that map into its local storage path.
- MEL does not claim complete support for every Meshtastic protobuf message or admin/control primitive.

## State-model limitations

- Local node inventory is derived from observed packets, not from an authoritative fetch from the node.
- UI and API state reflect what MEL has observed and stored locally, not a full mesh truth service.
- Topology is not inferred beyond stored observed facts.

## Operator workflow limitations

- `mel doctor` checks direct serial/TCP reachability, but intentionally does not prove MQTT broker reachability.
- Backup restore is dry-run only.
- The built-in auth model is basic auth when enabled; MEL does not claim a broader identity or tenancy layer.
- The built-in metrics bind config exists, but this release does not expose a separate metrics server.

## Deployment limitations

- Linux and Raspberry Pi are the primary supported direct-node environments.
- Termux is documented as a foreground/manual path, not a durable background-service claim.
- Multi-node and multi-transport deployments require operator review for contention, duplication, and radio ownership semantics.
