# Direct-node quickstart

Use this flow when MEL is attached to a real Meshtastic node on Linux or Raspberry Pi.

## Serial direct-node

1. Build MEL with `make build`.
2. Copy `configs/mel.serial.example.json` to your target path.
3. Change `storage.data_dir` and `storage.database_path` to a persistent writable location.
4. Set `transports[0].serial_device` to a real device path such as `/dev/serial/by-id/...`.
5. Ensure the MEL user can open the device, usually via `dialout` or `uucp`.
6. Run `./bin/mel config validate --config <path>`.
7. Run `./bin/mel doctor --config <path>`.
8. Start `./bin/mel serve --config <path>`.
9. Visit the UI or `/api/v1/status` and confirm the transport becomes `connected_no_ingest_evidence` or `ingesting`.

## TCP direct-node

1. Copy `configs/mel.tcp.example.json`.
2. Set `tcp_host` and `tcp_port` or `endpoint` to a real Meshtastic-compatible stream endpoint.
3. Run `./bin/mel config validate --config <path>`.
4. Run `./bin/mel doctor --config <path>`.
5. Start `./bin/mel serve --config <path>`.

## What success looks like

- `mel doctor` reports no direct-transport reachability or permission findings.
- `/api/v1/status` shows the direct transport as `ok: true`.
- `mel status` shows `last_successful_ingest` after packets arrive.
- `mel nodes` and `mel node inspect` return real locally observed nodes.
- This repository pass did not include live radio hardware, so serial/TCP claims remain code/test verified plus operator-checkable through doctor/runtime state.

## What MEL does not do here

- It does not invent nodes or traffic when the radio is quiet.
- It does not send packets back to the node in RC1.
- It does not claim BLE fallback if serial or TCP fail.
