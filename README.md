# MEL — MeshEdgeLayer

MEL is a privacy-first, local-first edge control-plane and observability layer for stock Meshtastic networks. It adds operator visibility, safer retention, policy checks, and a minimal local UI around supported Meshtastic interfaces without requiring firmware forks or protocol-breaking changes.

## What MEL is

- A local daemon and CLI for Meshtastic-adjacent operations.
- A truthful observability layer for packets, nodes, retention, transport health, and privacy posture.
- A compatibility-preserving overlay around stock Meshtastic nodes.
- A local web UI served by the daemon.
- A policy and privacy audit tool that treats MQTT, map reporting, and metadata leakage as first-class risks.

## What MEL is not

- Not a Meshtastic firmware fork.
- Not a replacement routing protocol.
- Not guaranteed anonymity on RF.
- Not a magical anti-jam solution.
- Not a fake AI mesh optimizer.
- Not a cloud requirement.

## Current supported interfaces

- **MQTT ingest**: implemented end-to-end in v0.1 with a real MQTT client and Meshtastic `ServiceEnvelope` parsing.
- **HTTP / serial-TCP / BLE**: represented as real transport types with explicit feature-gated capability reporting, but not enabled in this milestone.

## Privacy model

- Localhost bind by default.
- No remote bind without explicit config.
- Privacy audit flags unsafe MQTT posture, map reporting, remote bind without auth, long retention, and precise position storage.
- Precise positions are redacted in normal node views; sensitive fields are designed for encrypted-at-rest handling when a 32-byte storage key is supplied.

## Supported platforms

- Linux
- Raspberry Pi / other arm64 Linux targets
- Termux / Android: documented as manual foreground mode only in this milestone

## Quickstart

```bash
make verify
cp configs/mel.example.json .tmp/mel.json
mkdir -p .tmp/data
sed -i 's#./data#.tmp/data#g' .tmp/mel.json
sed -i 's#./data/mel.db#.tmp/data/mel.db#g' .tmp/mel.json
./bin/mel serve --config .tmp/mel.json
```

Then open <http://127.0.0.1:8080/>.

## Development setup

Requirements:
- Go 1.25+
- `sqlite3`
- `protoc`
- `curl`

Useful commands:

```bash
./scripts/verify-proto.sh
gofmt -w $(find . -name '*.go' -not -path './vendor/*')
go vet ./...
go test ./...
make build
./scripts/smoke.sh
```

## Verification commands

```bash
./scripts/verify-proto.sh
gofmt -w $(find . -name '*.go' -not -path './vendor/*')
go vet ./...
go test ./...
make build
./scripts/smoke.sh
```

## CLI commands

- `mel version`
- `mel doctor --config <path>`
- `mel config validate --config <path>`
- `mel serve --config <path>`
- `mel status`
- `mel nodes`
- `mel transports list --config <path>`
- `mel db vacuum --config <path>`
- `mel export --config <path>`
- `mel logs tail --config <path>`
- `mel policy explain --config <path>`
- `mel privacy audit --config <path>`

## Real Meshtastic path in v0.1

MEL v0.1 ships one complete ingest path: Meshtastic-style MQTT `ServiceEnvelope` payloads over a real MQTT TCP session. The repository also includes a local simulator command (`mel dev-simulate-mqtt`) used by tests and operators for deterministic validation without claiming live radio access.

## Current limitations

- MQTT only supports plain TCP in this milestone; TLS-capable brokers should be fronted by a local secure tunnel if needed.
- BLE is not implemented and is explicitly feature-gated.
- HTTP and serial/TCP transport modules are modeled but not wired to Meshtastic devices in this milestone.
- SQLite access uses the system `sqlite3` CLI to preserve an offline, stdlib-only Go build in this environment.

## Compatibility guarantees

- MEL does not require Meshtastic firmware changes.
- If MEL stops, stock mesh behavior continues unchanged.
- MEL does not claim to change on-air routing inside node firmware.
