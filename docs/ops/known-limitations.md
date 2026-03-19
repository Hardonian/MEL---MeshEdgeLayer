# Known limitations

MEL RC1 is intentionally smaller than a full Meshtastic management stack. These limitations are current product truth, not roadmap placeholders.

## Unsupported transports

- BLE transport is explicitly unsupported.
- HTTP transport is explicitly unsupported.
- `serialtcp` exists in code but is not promoted as a hardened operator path.

## Partial protocol coverage

- MEL parses and stores only the protobuf fields it currently uses for packet, user, and position observations.
- Full Meshtastic protobuf coverage is not claimed.
- Unsupported payload types may still be stored as generic packet observations without rich semantic decoding.

## Control-path gaps

- No transmit / publish path.
- No node admin or radio configuration operations.
- No transport metadata fetch.
- No transport-driven node inventory fetch.

## Multi-transport and multi-node caveats

- MEL warns about more than one direct transport.
- Hybrid direct + MQTT ingest can duplicate observations.
- MEL does not arbitrate radio ownership across multiple clients or multiple direct attachments.

## Config and platform caveats

- `storage.encryption_required` does not encrypt SQLite at rest.
- `bind.metrics` and `features.metrics` do not create a metrics endpoint.
- `features.ble_experimental` does not create BLE support.
- `rate_limits.http_rps` is not enforced by the current HTTP server.
- Serial direct-node requires host `stty` plus device permissions.
- SQLite operations depend on the `sqlite3` CLI being available.
- Direct serial and TCP ingest are code/test verified here, but not live-hardware verified in this environment.

## UI / API truth boundaries

- The UI and API show only local observations and health state.
- `/api/v1/status` explicitly separates runtime in-memory state from persisted SQLite history.
- They do not invent topology, backlog, or historical mesh state when nothing has been ingested.
- If no transport is connected, empty state is expected behavior.

## Backup / restore caveats

- Backup creation is implemented.
- Restore is validation-only (`--dry-run`) in RC1.
