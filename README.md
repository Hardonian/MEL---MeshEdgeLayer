# MEL — MeshEdgeLayer

MEL is a local-first ingest, persistence, and operator-observability layer for **stock Meshtastic deployments**. Today it can ingest real Meshtastic traffic from:

- a **direct serial-attached node**,
- a **Meshtastic-compatible TCP stream**, or
- an **MQTT broker carrying Meshtastic protobuf envelopes**.

MEL is **not** a firmware fork, not a routing replacement, not a remote control plane for radios, and not a claim of universal Meshtastic transport support. MEL only documents support that exists in code and has repo-local verification coverage.

## Why MEL exists

Stock Meshtastic clients are good at interacting with radios. MEL exists to give operators a small local edge service that:

- stores what it actually observed,
- keeps transport health and degraded states explicit,
- exposes a local UI and JSON API,
- runs privacy and policy checks against the active config, and
- lets operators inspect, export, back up, and retain local observations.

## Current implementation status

### Implemented today

- Go daemon and CLI.
- SQLite persistence with deterministic `sqlite3` CLI migrations.
- Local UI plus versioned `/api/v1/*` JSON endpoints.
- `mel doctor`, `mel config validate`, `mel privacy audit`, `mel policy explain`.
- Export, backup creation, and restore **dry-run validation**.
- Real ingest via serial direct-node, TCP direct-node, and MQTT.
- Meshtastic protobuf subset parsing for observed message, user, and position fields that MEL stores today.

### Explicitly not claimed today

- BLE ingest.
- HTTP transport ingest.
- Radio transmit / publish.
- Node admin or configuration control.
- Metadata fetch or node fetch from the transport layer.
- Multi-radio arbitration.
- Full protobuf coverage.
- At-rest SQLite encryption implemented by MEL itself.
- A metrics listener.

## Transport support matrix

| Transport / path | Status | Config method | How to verify | Caveats |
| --- | --- | --- | --- | --- |
| Serial direct-node | Implemented and verified | `type: "serial"`, `serial_device`, optional `serial_baud` | `mel doctor`, UI/API transport health, message/node persistence | Ingest only. Requires local device access and `stty`. |
| TCP direct-node | Implemented and verified | `type: "tcp"`, `tcp_host`/`tcp_port` or `endpoint` | `mel doctor`, UI/API transport health, message/node persistence | Ingest only. Endpoint must speak Meshtastic stream framing, not HTTP. |
| MQTT ingest | Implemented and verified | `type: "mqtt"`, `endpoint`, `topic`, `client_id` | start `mel serve`, observe `/api/v1/status`, `/api/v1/messages`, CLI status/export | Ingest only. RC1 does not publish back to the broker. |
| Hybrid direct + MQTT | Implemented but partial | enable two transports | `mel transports list`, doctor/config lints, packet persistence | Supported for ingest only. Operators must handle duplicate-observation risk and radio ownership realities. |
| `serialtcp` alias | Experimental / not hardened | `type: "serialtcp"`, `endpoint` | direct transport health only | Uses the same direct TCP reader path but is not documented as a primary operator workflow. |
| BLE | Explicitly unsupported | `type: "ble"` | `mel transports list` / UI shows unsupported | Feature flag does not make it work. |
| HTTP transport | Explicitly unsupported | `type: "http"` | `mel transports list` / UI shows unsupported | No live device path is wired. |
| Send / publish / admin control | Planned / not implemented | none | n/a | MEL intentionally refuses to claim radio control until code and tests exist. |

See [docs/ops/transport-matrix.md](docs/ops/transport-matrix.md) for the evidence-oriented version.

## Architecture summary

1. `mel serve` loads defaults, JSON config, and supported `MEL_*` environment overrides.
2. Config validation and linting check transport requirements, privacy posture, remote bind safety, and known RC1 no-op knobs.
3. MEL opens SQLite, applies deterministic migrations through the `sqlite3` CLI, and runs retention before ingest starts.
4. Each enabled transport runs in its own reconnect loop.
5. Serial/TCP direct frames and MQTT envelopes normalize into the same Meshtastic envelope shape.
6. Ingested observations are persisted to `messages`, `nodes`, `telemetry_samples`, and `audit_logs`.
7. The UI, CLI, and `/api/v1/*` read from the same local state and database truth.

Detailed docs:

- [docs/architecture/overview.md](docs/architecture/overview.md)
- [docs/architecture/runtime-flow.md](docs/architecture/runtime-flow.md)
- [docs/architecture/transport-flow.md](docs/architecture/transport-flow.md)

## Quickstart that works

### 1. Build MEL

```bash
make build
```

### 2. Pick one config example

- Direct serial: `configs/mel.serial.example.json`
- Direct TCP: `configs/mel.tcp.example.json`
- MQTT only: `configs/mel.mqtt-only.example.json`
- Hybrid ingest: `configs/mel.hybrid.example.json`

### 3. Copy it and set real paths

```bash
mkdir -p .tmp
cp configs/mel.serial.example.json .tmp/mel.json
python3 - <<'PY'
from pathlib import Path
p = Path('.tmp/mel.json')
text = p.read_text()
text = text.replace('./data/mel.db', '.tmp/data/mel.db').replace('./data', '.tmp/data')
p.write_text(text)
PY
chmod 600 .tmp/mel.json
```

Then edit the transport block:

- serial: set a real `serial_device` such as `/dev/serial/by-id/...`
- tcp: set `tcp_host` and `tcp_port`
- mqtt: set broker `endpoint` and `topic`

### 4. Validate config and local prerequisites

```bash
./bin/mel config validate --config .tmp/mel.json
./bin/mel doctor --config .tmp/mel.json
```

Interpret `mel doctor` honestly:

- **no transports enabled** = MEL will stay idle by design.
- **serial device not found / permission denied** = direct-node setup is incomplete.
- **TCP endpoint unreachable** = wrong host/port or wrong protocol.
- **MQTT is enabled** = doctor validates config posture but intentionally does not require broker reachability.
- **`historical_ingest_seen` in `summary.transport_observations`** = doctor found prior packets for that transport in SQLite, but it is still not claiming live connectivity in the current run.

### 5. Start MEL

```bash
./bin/mel serve --config .tmp/mel.json
```

Then open:

- UI: <http://127.0.0.1:8080/>
- status API: <http://127.0.0.1:8080/api/v1/status>
- messages API: <http://127.0.0.1:8080/api/v1/messages>

### 6. Confirm success with real evidence

A healthy first run shows one of these explicit states:

```bash
make build
mkdir -p .tmp/mqtt
cp configs/mel.mqtt-only.example.json .tmp/mqtt/mel.json
python3 - <<'PY'
from pathlib import Path
p = Path('.tmp/mqtt/mel.json')
text = p.read_text().replace('./data', '.tmp/mqtt/data')
p.write_text(text)
PY
chmod 600 .tmp/mqtt/mel.json
./bin/mel config validate --config .tmp/mqtt/mel.json
./bin/mel serve --config .tmp/mqtt/mel.json
```

Open <http://127.0.0.1:8080/> and confirm the transport state is one of:

- `configured_not_attempted` before the service starts
- `connect_failed` or `retrying` if the node/path is unavailable
- `connected but idle`
- `live data flowing`
- `unsupported`

Once packets arrive, verify with:

```bash
./bin/mel status --config .tmp/mel.json
./bin/mel nodes --config .tmp/mel.json
./bin/mel export --config .tmp/mel.json --out .tmp/export.json
```

## Local node attachment

### Serial direct-node

Use MEL on a Linux or Raspberry Pi host that owns the radio-attached serial device.

Requirements:

- a real node path in `serial_device`,
- `stty` available on the host,
- write access to the MEL data directory,
- user access to the serial device, usually via `dialout` or `uucp`.

MEL reads real Meshtastic stream frames. It does **not** send packets back to the radio in RC1.

### TCP direct-node

Use `type: "tcp"` only when the target endpoint really exposes Meshtastic stream framing. A web UI, JSON API, or generic TCP tunnel is not enough.

## MQTT ingest

MQTT remains supported for ingest. MEL's MQTT path:

- connects directly to the configured broker endpoint,
- subscribes to the configured topic,
- parses protobuf envelopes it receives,
- stores messages and nodes locally,
- does **not** publish, administer radios, or claim broker-side control behavior.

## Config overview

Main config groups:

- `bind`: API/UI bind address. Remote bind requires deliberate auth posture.
- `auth`: HTTP basic auth for UI/API.
- `storage`: data directory and SQLite path.
- `retention`: message, telemetry, audit, and precise-position retention windows.
- `privacy`: export redaction, precise positions, MQTT encryption policy posture, map reporting, trust list.
- `transports`: actual ingest configuration.
- `features.web_ui`: if `false`, MEL still serves JSON endpoints but does not register the HTML UI route.

### Config truths and caveats

- `storage.encryption_required` is a **validation guard only** in RC1; MEL does not encrypt the SQLite file itself.
- `bind.metrics` and `features.metrics` are reserved knobs; RC1 does **not** start a metrics listener.
- `features.ble_experimental` does not enable a working BLE transport.
- MEL supports `MEL_*` environment overrides only for the fields currently read in `internal/config/config.go`.

See [docs/ops/configuration.md](docs/ops/configuration.md).

## Persistence, retention, export, and backup

MEL persists to SQLite and currently uses these operator-facing commands:

```bash
./bin/mel export --config .tmp/mel.json --out .tmp/export.json
./bin/mel backup create --config .tmp/mel.json --out .tmp/mel-backup.tgz
./bin/mel backup restore --bundle .tmp/mel-backup.tgz --dry-run --destination .tmp/restore-preview
./bin/mel db vacuum --config .tmp/mel.json
```

Restore is intentionally **dry-run only** in RC1.

## CLI overview

```text
mel init
mel version
mel doctor --config <path>
mel config validate --config <path>
mel serve --config <path>
mel status --config <path>
mel nodes --config <path>
mel node inspect <node-id> --config <path>
mel transports list --config <path>
mel privacy audit [--format json|text] --config <path>
mel policy explain --config <path>
mel export --config <path> [--out path]
mel import validate --bundle <path>
mel backup create --config <path> [--out path]
mel backup restore --bundle <path> --dry-run [--destination dir]
mel logs tail --config <path>
mel db vacuum --config <path>
```

## UI and API scope

The UI and `/api/v1/*` expose local truth only:

- transport health,
- observed nodes,
- recent messages,
- privacy findings,
- policy recommendations,
- audit/event records.

They do **not** expose a fabricated mesh topology, node admin flows, or a full Meshtastic management plane.

## First 10 minutes evaluation flow

Use [docs/ops/evaluate-in-10-minutes.md](docs/ops/evaluate-in-10-minutes.md) for a skeptical first-user path. It includes:

- a real transport evaluation path,
- a repo-local MQTT self-test path to prove MEL's ingest/storage/UI stack,
- clear statements about what that self-test does and does not prove.

## Known limitations

See [docs/ops/known-limitations.md](docs/ops/known-limitations.md). Important RC1 limits:

- no BLE or HTTP ingest,
- no send/control path,
- partial protobuf coverage,
- no multi-radio arbitration,
- no MEL-provided at-rest encryption,
- no metrics endpoint,
- restore is dry-run only.

## Security and privacy notes

- Keep MEL bound to localhost unless you have a reviewed remote-access design.
- Enable auth before any remote exposure.
- Keep `privacy.redact_exports=true` unless you deliberately need raw exports.
- Treat MQTT and long retention as privacy-sensitive choices.
- Tighten config file permissions to `0600` where possible.

See [SECURITY.md](SECURITY.md) and [docs/privacy/privacy-posture.md](docs/privacy/privacy-posture.md).

## Contributing

Start with [CONTRIBUTING.md](CONTRIBUTING.md). Transport or docs work must narrow claims when implementation proof is missing.

## Roadmap: implemented vs planned

### Implemented now

- serial direct-node ingest
- TCP direct-node ingest
- MQTT ingest
- local UI/API
- privacy/policy/doctor flows
- SQLite persistence, export, backup, retention

### Planned / not implemented

- transport send paths
- node admin/control operations
- BLE ingest
- HTTP transport ingest
- metrics endpoint
- broader protobuf coverage
- restore write path

## License

MIT. See [LICENSE](LICENSE).
