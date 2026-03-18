# Direct-node quickstart

## Preferred deployment mode

Run MEL on a Raspberry Pi or Linux host that is physically attached to a stock Meshtastic node over USB serial, or pointed at a Meshtastic-compatible TCP stream endpoint.

## Serial quickstart

1. Copy `configs/mel.serial.example.json` to your target path.
2. Set `storage.data_dir` and `storage.database_path` to a writable persistent location.
3. Set `transports[0].serial_device` to the real node path, for example `/dev/ttyUSB0` or `/dev/serial/by-id/...`.
4. Add the MEL service user to `dialout` or `uucp` as appropriate for the distro.
5. Run `mel doctor --config /etc/mel/mel.json`.
6. Start `mel serve --config /etc/mel/mel.json`.
7. Visit the UI or `mel transports list` to confirm the transport is reachable.

## TCP quickstart

1. Copy `configs/mel.tcp.example.json`.
2. Set `tcp_host` and `tcp_port` to the direct Meshtastic-compatible endpoint.
3. Run `mel doctor --config /etc/mel/mel.json`.
4. Start `mel serve --config /etc/mel/mel.json`.

## What success looks like

- `mel doctor` reports no direct-transport findings.
- The UI/API transport state is `connected_idle` or `connected_ingesting`.
- `mel status` shows a `last_successful_ingest` timestamp once packets arrive.
- `mel nodes` and `mel node inspect` return real observed nodes.

## What MEL does when no radio packets arrive

MEL keeps the DB and UI empty except for configuration and health evidence. It does not fabricate nodes, topology, or messages.

## Direct transport state meanings

- `configured_not_attempted`: config is valid, but the MEL service is not running yet.
- `connecting`: MEL is actively opening the direct connection.
- `connect_failed`: the latest connect attempt failed.
- `connected_idle`: MEL opened the stream but has not decoded a live packet yet.
- `connected_ingesting`: live packet decode and ingest succeeded.
- `degraded`: MEL is connected but has recently seen malformed frames or ingest handler failures.
- `retrying`: the stream dropped and MEL is sleeping before the next connect attempt.
