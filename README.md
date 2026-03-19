# MEL — MeshEdgeLayer

MEL is a local-first ingest, persistence, and operator-observability layer for **stock Meshtastic deployments**. MEL only documents support that exists in code and has repo-local verification coverage.

## What MEL does

- Ingests real Meshtastic traffic from **serial direct-node**, **TCP direct-node**, and **MQTT** transports.
- Persists observed packets, nodes, telemetry samples, and audit events into SQLite using deterministic `sqlite3` CLI migrations.
- Exposes truthful operator views through `mel doctor`, `mel status`, `mel replay`, the HTML UI, and `/api/v1/*` JSON endpoints.
- Reports explicit transport truth with the same state vocabulary across CLI and API: `disabled`, `configured_not_attempted`, `attempting`, `configured_offline`, `connected_no_ingest`, `ingesting`, `historical_only`, and `error`.
- Ships a JSON `/metrics` endpoint that reflects the same counters shown by doctor and status.

## What MEL does not do

- No BLE ingest.
- No HTTP transport ingest.
- No transmit, publish, routing, or control-plane behavior.
- No fake packets, fake nodes, or placeholder transport success.
- No hardware-verification claims that were not exercised in this repo environment.
- No full Meshtastic protobuf coverage; unsupported payloads are stored truthfully as raw payload bytes.

See the canonical execution contract in [docs/roadmap/ROADMAP_EXECUTION.md](docs/roadmap/ROADMAP_EXECUTION.md).

## Supported transport matrix

| Transport / path | Status | Verification surface | Caveats |
| --- | --- | --- | --- |
| Serial direct-node | Supported for ingest | `mel doctor`, `mel status`, `/api/v1/status`, `/metrics`, DB evidence | Requires local device access plus `stty`. |
| TCP direct-node | Supported for ingest | `mel doctor`, `mel status`, `/api/v1/status`, `/metrics`, DB evidence | Endpoint must speak Meshtastic framing, not HTTP. |
| MQTT ingest | Supported for ingest | `mel status`, `/api/v1/status`, `/api/v1/messages`, `/metrics`, smoke test | MEL subscribes only; it does not publish. |
| Hybrid direct + MQTT | Supported with caveats | dedupe evidence, transport truth, doctor warnings | Operators must verify duplicate behavior and radio ownership in deployment. |
| `serialtcp` alias | Implemented but not primary | same as TCP path | Alias of the direct TCP reader path. |
| BLE / HTTP | Explicitly unsupported | `mel transports list`, UI, docs | Unsupported means no production claim. |

The detailed source-of-truth version lives in [docs/ops/transport-matrix.md](docs/ops/transport-matrix.md).

## Quickstart

### 1. Build

```bash
make build
```

### 2. Copy a config example and lock its mode

```bash
mkdir -p .tmp
cp configs/mel.mqtt-only.example.json .tmp/mel.json
python3 - <<'PY'
from pathlib import Path
p = Path('.tmp/mel.json')
text = p.read_text()
text = text.replace('./data/mel.db', '.tmp/data/mel.db').replace('./data', '.tmp/data')
p.write_text(text)
PY
chmod 600 .tmp/mel.json
```

### 3. Validate and inspect truth before launch

```bash
./bin/mel config validate --config .tmp/mel.json
./bin/mel doctor --config .tmp/mel.json
```

`mel doctor` is authoritative:

- it validates config and file permissions,
- verifies SQLite write/read behavior,
- reports actionable transport findings,
- distinguishes persisted historical evidence from live runtime proof,
- never reports ingest unless SQLite writes exist.

### 4. Serve MEL

```bash
./bin/mel serve --debug --config .tmp/mel.json
```

### 5. Inspect local evidence

- UI: <http://127.0.0.1:8080/>
- Status: <http://127.0.0.1:8080/api/v1/status>
- Messages: <http://127.0.0.1:8080/api/v1/messages>
- Metrics: <http://127.0.0.1:8080/metrics>

### 6. Replay what MEL actually stored

```bash
./bin/mel replay --config .tmp/mel.json --limit 20
./bin/mel replay --config .tmp/mel.json --node 12345 --type text --limit 20
```

## CLI overview

```text
mel init
mel version
mel doctor --config <path>
mel config validate --config <path>
mel serve [--debug] --config <path>
mel status --config <path>
mel panel [--format text|json] --config <path>
mel nodes --config <path>
mel node inspect <node-id> --config <path>
mel transports list --config <path>
mel replay --config <path> [--node <id>] [--type <message-type>] [--limit <n>]
mel privacy audit [--format json|text] --config <path>
mel policy explain --config <path>
mel export --config <path> [--out path]
mel import validate --bundle <path>
mel backup create --config <path> [--out path]
mel backup restore --bundle <path> --dry-run [--destination dir]
mel logs tail --config <path>
mel db vacuum --config <path>
```

## Message typing truth

MEL currently stores and labels these message classes:

- `text`
- `position`
- `node_info`
- `telemetry` as raw payload evidence when the full telemetry protobuf schema is not vendored in-repo
- `unknown` with raw payload retention

## Operator docs

- [docs/ops/configuration.md](docs/ops/configuration.md)
- [docs/ops/diagnostics.md](docs/ops/diagnostics.md)
- [docs/ops/transport-matrix.md](docs/ops/transport-matrix.md)
- [docs/ops/known-limitations.md](docs/ops/known-limitations.md)
- [docs/release/RELEASE_CHECKLIST.md](docs/release/RELEASE_CHECKLIST.md)

## Contributor docs

- [docs/contributor/transport-interface-contract.md](docs/contributor/transport-interface-contract.md)
- [docs/contributor/protobuf-extension-guide.md](docs/contributor/protobuf-extension-guide.md)
- [docs/architecture/central-extension-node-layout.md](docs/architecture/central-extension-node-layout.md)
- [docs/architecture/operator-panels.md](docs/architecture/operator-panels.md)

## Topology scaffolds

- `topologies/central-node/` reserves hub-side space for small config files in `config/` and larger stateful assets in `memory-management/`.
- `topologies/extension-node/` reserves constrained-device space for small config files in `config/` and only bounded local state in `memory-management/`.
- These scaffolds are documentation-first and do not claim that MEL already ships separate node-specific runtimes.
