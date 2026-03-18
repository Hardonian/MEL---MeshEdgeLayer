# MEL — MeshEdgeLayer

MEL is a local-first Meshtastic edge collector for operators who want truthful transport health, durable local storage, a small local API/UI, and explicit privacy/retention controls without pretending to replace stock node routing or firmware behavior.

## What MEL is

MEL sits beside stock Meshtastic nodes.

- It **ingests** mesh observations from a real transport that MEL can actually open today.
- It **normalizes and stores** those observations in local SQLite tables.
- It **surfaces operator evidence** through CLI commands, a local API, a local UI, audit logs, privacy findings, and retention/export workflows.
- It **does not** claim to be a routing replacement, a radio control plane, or a complete node-management stack.

## Current implementation status

MEL is currently credible as a cautious OSS preview for:

- direct serial ingest from a stock node on Linux / Raspberry Pi,
- direct TCP ingest from a Meshtastic-compatible framed stream endpoint,
- MQTT subscribe ingest,
- local persistence, retention, export, privacy audit, backup creation, and restore dry-run validation,
- local-only operator workflows through the CLI, HTTP API, and built-in UI.

MEL is **not** yet a full transport multiplexer, publish bridge, admin/config tool, BLE client, or firmware-side control plane.

## Transport support matrix

| Transport / path | Status | Config method | Verification method | Caveats |
| --- | --- | --- | --- | --- |
| Serial direct-node (`type: serial`) | Supported | `serial_device`, `serial_baud` | `./bin/mel doctor`, `./bin/mel transports list`, UI transport status, packet counters | Ingest only. Requires host access to the device and `stty`. |
| TCP direct-node (`type: tcp`) | Supported | `tcp_host` + `tcp_port` or `endpoint` | `./bin/mel doctor`, `./bin/mel transports list`, UI transport status, packet counters | Ingest only. Endpoint must speak Meshtastic framing, not HTTP. |
| Direct stream alias (`type: serialtcp`) | Implemented but partial | `endpoint` | `./bin/mel config validate`, `./bin/mel transports list` | Uses the same direct reader as TCP, but MEL ships no separate operator workflow or example for it. |
| MQTT ingest (`type: mqtt`) | Supported | `endpoint`, `topic`, `client_id` | `./bin/mel transports list`, packet counters, persisted messages | Subscribe ingest only. `mel doctor` intentionally does not probe broker reachability. |
| BLE (`type: ble`) | Unsupported | Feature-gated only | `./bin/mel transports list` shows unsupported | No live BLE implementation is claimed. |
| HTTP (`type: http`) | Unsupported | Feature-gated only | `./bin/mel transports list` shows unsupported | No live HTTP node attachment is claimed. |
| Publish / transmit | Unsupported | None | N/A | `SendPacket` is disabled for all current transports. |
| Metadata fetch / radio info fetch | Unsupported | None | N/A | No transport currently exposes metadata fetch. |
| Node inventory fetch from transport control path | Unsupported | None | N/A | Node inventory is derived from observed packets only. |

The detailed matrix, with code-level grounding and operator rules, lives in `docs/ops/transport-matrix.md`.

## Positioning in the Meshtastic ecosystem

MEL is best understood as a **truthful edge collector and observability layer**.

- **Relative to stock Meshtastic clients:** MEL does not replace the mobile app or official clients.
- **Relative to radios/nodes:** MEL attaches to a real node transport when configured, but does not claim radio administration or routing control.
- **Relative to MQTT backhaul:** MEL can ingest MQTT traffic, but it does not claim broker management, publish support, or topology authority.
- **Relative to local direct-node attachment:** direct serial/TCP ingest is the preferred operator path for this milestone.
- **Relative to persistence and observability:** MEL's differentiator is local evidence, privacy posture, export/backup support, and explicit degraded states.

## Architecture summary

At runtime MEL:

1. loads default config, JSON config, and a limited set of `MEL_*` env overrides,
2. validates config and lints risky posture,
3. opens SQLite through the `sqlite3` CLI and applies deterministic migrations,
4. starts one reconnect loop per enabled transport,
5. normalizes supported packets into a shared Meshtastic envelope path,
6. stores messages, nodes, telemetry samples, and audit evidence locally,
7. exposes status via CLI, `/api/v1/*`, and the built-in HTML UI.

See `docs/architecture/overview.md`, `docs/architecture/runtime-flow.md`, and `docs/architecture/transport-flow.md`.

## Quickstart that actually works

### Option A — direct serial on Linux or Raspberry Pi

```bash
make build
mkdir -p .tmp/demo
cp configs/mel.serial.example.json .tmp/demo/mel.json
python3 - <<'PY'
from pathlib import Path
p = Path('.tmp/demo/mel.json')
text = p.read_text()
text = text.replace('./data', '.tmp/demo/data').replace('/dev/ttyUSB0', '/dev/serial/by-id/REPLACE_ME')
p.write_text(text)
PY
./bin/mel config validate --config .tmp/demo/mel.json
./bin/mel doctor --config .tmp/demo/mel.json
./bin/mel serve --config .tmp/demo/mel.json
```

### Option B — MQTT-only evaluation

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
./bin/mel config validate --config .tmp/mqtt/mel.json
./bin/mel serve --config .tmp/mqtt/mel.json
```

Open <http://127.0.0.1:8080/> and confirm the transport state is one of:

- `configured but unreachable`
- `connected but idle`
- `live data flowing`
- `unsupported`

If no packets arrive, MEL intentionally leaves node/message views empty.

## Configuration overview

MEL config is a single JSON file.

Important sections:

- `bind`: local API/UI and metrics bind addresses.
- `auth`: HTTP basic auth for remote exposure.
- `storage`: data directory and SQLite path.
- `retention`: retention windows for messages, telemetry, audits, and precise positions.
- `privacy`: export redaction, precise position handling, MQTT posture, and trust list.
- `transports`: the real ingress source list.
- `rate_limits`: reconnect timing and HTTP rate shaping.

Actual env overrides are intentionally small and currently include:

- `MEL_BIND_API`, `MEL_BIND_METRICS`, `MEL_BIND_ALLOW_REMOTE`
- `MEL_DB_PATH`, `MEL_DATA_DIR`
- `MEL_AUTH_ENABLED`, `MEL_SESSION_SECRET`, `MEL_UI_USER`, `MEL_UI_PASSWORD`, `MEL_AUTH_ALLOW_INSECURE_REMOTE`
- `MEL_PRIVACY_STORE_PRECISE_POSITIONS`, `MEL_PRIVACY_MAP_REPORTING_ALLOWED`, `MEL_PRIVACY_MQTT_ENCRYPTION_REQUIRED`
- `MEL_RETENTION_MESSAGES_DAYS`

See `docs/ops/configuration.md` and `internal/config/config.go` for the current schema.

## Local node attachment today

### Supported today

- one serial-attached stock node on Linux / Raspberry Pi,
- one Meshtastic-compatible TCP stream endpoint,
- hybrid direct + MQTT ingest if you deliberately accept duplicate-observation risk.

### Not supported today

- BLE direct-node attachment,
- HTTP direct-node attachment,
- radio transmit / admin / config apply workflows,
- transport failover semantics beyond independent reconnect loops,
- authoritative node inventory fetch outside observed packets.

## MQTT support today

MQTT is still real and supported, but MEL now documents it as **one ingest path**, not as the entire product identity.

What MEL claims:

- subscribe ingest,
- packet counters and health state,
- shared normalization into the same local message/node path,
- privacy linting around encryption expectations, anonymous access, and JSON-oriented topics.

What MEL does not claim:

- publish support,
- broker management,
- broker reachability proof from `mel doctor`.

## Persistence, retention, export, and backup

MEL stores local state in SQLite tables created by `migrations/0001_init.sql`, including:

- `messages`
- `nodes`
- `telemetry_samples`
- `audit_logs`
- `retention_jobs`
- supporting tables such as `channels`, `trust_records`, `topology_edges`, and `config_apply_history`

Current operator flows:

- `./bin/mel export --config <path> [--out path]`
- `./bin/mel import validate --bundle <path>`
- `./bin/mel backup create --config <path> --out <bundle>`
- `./bin/mel backup restore --bundle <bundle> --dry-run --destination <dir>`

Restore is intentionally **dry-run only** in this release so operators can inspect bundle validity before MEL claims a write-back restore path.

## CLI commands

```bash
./bin/mel init --config ./mel.json
./bin/mel version
./bin/mel config validate --config ./mel.json
./bin/mel doctor --config ./mel.json
./bin/mel serve --config ./mel.json
./bin/mel status --config ./mel.json
./bin/mel nodes --config ./mel.json
./bin/mel node inspect 12345 --config ./mel.json
./bin/mel transports list --config ./mel.json
./bin/mel logs tail --config ./mel.json
./bin/mel db vacuum --config ./mel.json
./bin/mel privacy audit --format text --config ./mel.json
./bin/mel policy explain --config ./mel.json
./bin/mel export --config ./mel.json --out ./mel-export.json
./bin/mel import validate --bundle ./mel-export.json
./bin/mel backup create --config ./mel.json --out ./mel-backup.tgz
./bin/mel backup restore --bundle ./mel-backup.tgz --dry-run --destination ./restore-preview
```

## UI and API scope

The built-in UI and API are local operator surfaces, not a separate product tier.

Current API surfaces include:

- `/healthz`, `/readyz`
- `/api/status`, `/api/nodes`, `/api/transports`, `/api/privacy/audit`, `/api/recommendations`, `/api/logs`
- `/api/v1/status`, `/api/v1/nodes`, `/api/v1/node/{id}`, `/api/v1/transports`, `/api/v1/messages`, `/api/v1/privacy/audit`, `/api/v1/policy/explain`, `/api/v1/events`

The UI reports:

- onboarding steps,
- transport health,
- observed nodes,
- recent messages,
- privacy findings,
- policy recommendations,
- audit/event history.

If the transport is idle or disconnected, those views stay empty except for truthful health/audit evidence.

## How to verify MEL in 10 minutes

Use `docs/ops/first-10-minutes.md` for a skeptical-operator evaluation flow.

Short version:

1. Build MEL.
2. Start it with one supported transport.
3. Confirm `/healthz` and `/api/v1/status` answer.
4. Watch transport counters move from zero after a real packet arrives.
5. Inspect `messages` and `nodes` through CLI or `sqlite3`.
6. Run `privacy audit`, `export`, and `backup restore --dry-run`.
7. Confirm unsupported features remain explicitly unsupported.

## Known limitations

- protobuf decoding is intentionally partial and currently focused on packet envelope basics plus user/position payload handling,
- direct transport support is ingest-only,
- no transmit / publish / config apply path is claimed,
- no BLE or HTTP node attachment is claimed,
- multi-transport deployments need operator judgment around contention and duplicates,
- `mel doctor` verifies direct serial/TCP reachability, but not MQTT broker reachability,
- restore is dry-run only,
- local UI/API auth is basic auth when enabled, not a larger identity system.

See `docs/ops/known-limitations.md`.

## Safety and privacy notes

- Keep MEL bound to localhost unless remote access is deliberate and defended.
- Keep exports redacted unless you have a specific reason not to.
- Keep precise position storage disabled unless your operating posture requires it.
- Treat public/default MQTT topics as a deliberate choice, not a hidden default.
- Run `./bin/mel privacy audit --config <path>` and `./bin/mel doctor --config <path>` before go-live changes.

## Roadmap: planned vs implemented

Implemented now:

- serial direct ingest,
- TCP direct ingest,
- MQTT subscribe ingest,
- local API/UI,
- persistence, retention, export, backup creation, restore dry-run,
- privacy audit and policy explanation.

Planned or explicitly not yet implemented:

- BLE direct-node support,
- HTTP direct-node support,
- send/publish paths,
- metadata and node fetch control paths,
- stronger transport arbitration for shared-radio scenarios,
- broader protobuf decode coverage.

## Contributing

See `CONTRIBUTING.md` for repo structure, transport acceptance criteria, and verification expectations.

## License

MEL is released under the MIT License. See `LICENSE`.
