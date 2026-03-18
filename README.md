# MEL — MeshEdgeLayer

MEL is a local-first edge layer for stock Meshtastic deployments. The preferred deployment mode is now a Raspberry Pi or Linux host attached directly to a real node over a serial or Meshtastic-compatible TCP stream. MEL persists what it truly observes, exposes transport health and limitations explicitly, and does not claim to alter stock firmware routing behavior.

## Design truth summary

- **Direct-node first.** Serial and TCP direct-node ingest are the primary operator story for this milestone.
- **Truthful empty states.** If no transport is enabled, or a configured transport cannot ingest, MEL stays empty and says so.
- **Real ingest only.** MQTT ingest remains supported. HTTP and BLE are still not production-supported and are surfaced as unsupported.
- **Local durability.** SQLite stores messages, nodes, telemetry samples, and audit evidence across restarts.
- **Explicit capabilities.** CLI, API, UI, and docs expose what each transport can and cannot do.

## Supported transport matrix

| Transport | Status | Ingest | Send | Metadata fetch | Node fetch | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Serial direct-node | Supported | Yes | No | No | No | Preferred Pi/Linux deployment mode. Requires host access to the serial device. |
| TCP direct-node | Supported | Yes | No | No | No | For Meshtastic-compatible TCP stream endpoints. |
| MQTT ingest | Supported | Yes | No | No | No | Preserved and still supported. |
| HTTP | Unsupported | No | No | No | No | Not wired to real devices in this milestone. |
| BLE | Unsupported | No | No | No | No | Still feature-gated and not claimed as working. |

See `docs/ops/transport-matrix.md` for the detailed capability table.

## Quickstart: direct serial on Linux or Raspberry Pi

```bash
make build
cp configs/mel.serial.example.json /etc/mel/mel.json
sudo mkdir -p /var/lib/mel
sudo chown "$USER":"$USER" /var/lib/mel
python3 - <<'PY'
from pathlib import Path
p = Path('/etc/mel/mel.json')
text = p.read_text()
text = text.replace('./data', '/var/lib/mel')
p.write_text(text)
PY
sudo usermod -aG dialout "$USER"
mel doctor --config /etc/mel/mel.json
mel serve --config /etc/mel/mel.json
```

Open <http://127.0.0.1:8080/> and confirm the transport status is one of:

- `configured but unreachable`
- `connected but idle`
- `live data flowing`

## Common commands

```bash
mel init --config /etc/mel/mel.json
mel doctor --config /etc/mel/mel.json
mel config validate --config /etc/mel/mel.json
mel serve --config /etc/mel/mel.json
mel status --config /etc/mel/mel.json
mel nodes --config /etc/mel/mel.json
mel node inspect 12345 --config /etc/mel/mel.json
mel transports list --config /etc/mel/mel.json
mel privacy audit --format text --config /etc/mel/mel.json
mel policy explain --config /etc/mel/mel.json
mel export --config /etc/mel/mel.json --out ./mel-export.json
mel backup create --config /etc/mel/mel.json --out ./mel-backup.tgz
mel backup restore --bundle ./mel-backup.tgz --dry-run --destination ./restore-preview
```

## Operator behavior notes

- MEL prefers **one direct-node transport** at a time.
- Hybrid direct + MQTT ingest is allowed, but MEL warns about contention and duplicate-observation risk.
- If a serial device disappears or a TCP endpoint drops, MEL marks the transport unhealthy, records the last error, and retries with backoff.
- MEL only claims ingest support for direct-node transports in this milestone. Send/config-control paths remain disabled until proven safe.

## Documentation map

- Direct quickstart: `docs/ops/direct-node-quickstart.md`
- Transport capability matrix: `docs/ops/transport-matrix.md`
- Transport troubleshooting: `docs/ops/troubleshooting-transports.md`
- Direct transport architecture flow: `docs/architecture/transport-flow.md`
- Linux / Pi install notes: `docs/ops/install-linux.md`, `docs/ops/install-pi.md`, `docs/ops/raspberry-pi.md`
- Hardened systemd unit: `docs/ops/systemd/mel.service`

## Verification

```bash
./scripts/verify-proto.sh
make verify
./scripts/smoke.sh
go test ./... -count=1
```

## License

MEL is released under the MIT License. See `LICENSE`.
