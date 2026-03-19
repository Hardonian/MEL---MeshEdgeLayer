# Transport troubleshooting

## `configured_offline`

MEL could not reach the configured serial or TCP path.

- Check serial ownership, permissions, and cable state.
- Check TCP host, port, firewall, and whether the endpoint really exposes Meshtastic framing.

## `connected_no_ingest`

The transport is connected, but MEL has not yet persisted a packet.

- Generate real traffic on the mesh.
- Confirm the upstream MQTT topic or direct-node stream is the one carrying packets.

## `historical_only`

SQLite contains earlier packets for this transport, but the current command cannot prove a live path.

- Start `mel serve` and wait for `ingesting`.
- If it never transitions, inspect logs and doctor output.

## `error`

The transport surfaced a concrete error, malformed input, duplicate drop, or handler failure.

- Re-run `mel doctor`.
- Inspect `mel logs tail` and `/api/v1/status`.
- Fix the underlying cause before trusting the transport again.
