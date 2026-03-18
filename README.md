# MEL — MeshEdgeLayer

MEL is a lightweight, local-first, privacy-first edge layer for stock Meshtastic deployments. It runs as a single local daemon plus CLI, keeps observability truthful, and refuses to pretend it changes on-air routing or stock firmware behavior.

## MEL RC1 design truth

- **Stock firmware only.** MEL does not require a Meshtastic firmware fork.
- **No fake mesh state.** MEL boots empty when no real transport is connected.
- **Local bind by default.** Remote exposure requires deliberate config, and MEL warns when that posture is unsafe.
- **Single-host first.** Linux and Raspberry Pi are first-class. Termux is supported for foreground/manual operation only in this RC.
- **Real operator workflows.** `mel init`, `mel doctor`, `mel serve`, `mel status`, `mel nodes`, `mel node inspect`, `mel transports list`, `mel privacy audit`, `mel policy explain`, `mel export`, `mel import validate`, `mel backup create`, and `mel backup restore --dry-run` are implemented.
- **Truthful transport support.** MQTT ingest is real. HTTP, serial-TCP, and BLE remain explicitly unsupported in this RC and are reported that way.

## What MEL is

- A local daemon and CLI for Meshtastic-adjacent observability and policy checks.
- A durable SQLite-backed store for real node, message, telemetry, and audit observations.
- A privacy and policy layer that flags risky MQTT, retention, map reporting, and remote bind posture.
- A small local web UI that shows truthful empty states instead of injected demo content.

## What MEL is not

- Not a firmware fork.
- Not a routing replacement.
- Not a cloud control plane.
- Not a mesh simulator.
- Not a claim that BLE or serial transports are production-ready today.

## Quickstart

```bash
make verify
./bin/mel init --config .tmp/mel.json
mkdir -p .tmp/data
python3 - <<'PY'
from pathlib import Path
p = Path('.tmp/mel.json')
text = p.read_text()
text = text.replace('./data/mel.db', '.tmp/data/mel.db').replace('./data', '.tmp/data')
p.write_text(text)
PY
./bin/mel doctor --config .tmp/mel.json || true
./bin/mel serve --config .tmp/mel.json
```

Open <http://127.0.0.1:8080/>.

## Supported transports in RC1

| Transport | Status | Notes |
| --- | --- | --- |
| MQTT ingest | Supported | Real TCP MQTT subscribe path with packet accounting and reconnect attempts. |
| HTTP API | Unsupported | Feature-gated, not wired to real Meshtastic devices in RC1. |
| Serial-TCP | Unsupported | Explicitly surfaced as unsupported. |
| BLE | Unsupported | No production claim in this RC. |

## Config precedence

1. Built-in defaults.
2. JSON config file.
3. `MEL_*` environment overrides.

`mel config validate` prints active lints so operators can see when a config is technically valid but still risky.

## Common commands

```bash
./bin/mel init --config /etc/mel/mel.json
./bin/mel doctor --config /etc/mel/mel.json
./bin/mel config validate --config /etc/mel/mel.json
./bin/mel serve --config /etc/mel/mel.json
./bin/mel status --config /etc/mel/mel.json
./bin/mel nodes --config /etc/mel/mel.json
./bin/mel node inspect 12345 --config /etc/mel/mel.json
./bin/mel transports list --config /etc/mel/mel.json
./bin/mel privacy audit --format text --config /etc/mel/mel.json
./bin/mel policy explain --config /etc/mel/mel.json
./bin/mel export --config /etc/mel/mel.json --out ./mel-export.json
./bin/mel import validate --bundle ./mel-export.json
./bin/mel backup create --config /etc/mel/mel.json --out ./mel-backup.tgz
./bin/mel backup restore --bundle ./mel-backup.tgz --dry-run --destination ./restore-preview
```

## Packaging and operations

- One-shot Linux install script: `scripts/install-linux.sh`
- Upgrade helper: `scripts/upgrade-linux.sh`
- Uninstall helper: `scripts/uninstall-linux.sh`
- Hardened systemd unit: `docs/ops/systemd/mel.service`
- Termux launcher: `scripts/termux-run.sh`

## Verification

```bash
./scripts/verify-proto.sh
make verify
./scripts/smoke.sh
```

## Documentation map

- Architecture: `docs/architecture/`
- Privacy: `docs/privacy/`
- Operations: `docs/ops/`
- Product boundaries: `docs/product/`
- Community / OSS posture: `docs/community/`

## License

MEL is released under the MIT License. See `LICENSE`.
