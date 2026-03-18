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
- The UI transport table shows `connected but idle` or `live data flowing`.
- `mel status` shows a `last_successful_ingest` timestamp once packets arrive.
- `mel nodes` and `mel node inspect` return real observed nodes.

## What MEL does when no radio packets arrive

MEL keeps the DB and UI empty except for configuration and health evidence. It does not fabricate nodes, topology, or messages.
