# MEL Install Quickstart

This guide gets MEL running on a Linux host in under 10 minutes.

## Prerequisites

- Linux host (x86_64 or arm64); see `docs/ops/support-matrix.md` for full list.
- `sqlite3` binary in `$PATH` (version 3.35+).
- Go 1.25+ to build from source, or a pre-built binary from the release page.
- Root access is **not required**; MEL runs as an unprivileged user.

## Step 1 — Install the binary

**From a release bundle:**
```bash
tar -xzf mel-linux-amd64-v*.tar.gz
sudo install -m 755 mel /usr/local/bin/mel
```

**Build from source:**
```bash
git clone https://github.com/mel-project/mel.git
cd mel
go build -o mel ./cmd/mel
sudo install -m 755 mel /usr/local/bin/mel
```

## Step 2 — Create initial configuration

```bash
mel init --config /etc/mel/mel.json
```

This writes a default configuration file with a generated session secret.
**Immediately** restrict permissions:

```bash
sudo chmod 600 /etc/mel/mel.json
sudo chown mel:mel /etc/mel/mel.json  # if running as a dedicated user
```

MEL refuses to start if the config file is readable by group or world.

## Step 3 — Configure your transport

Edit `/etc/mel/mel.json` and set up at least one transport. Example for a
USB-serial Meshtastic node:

```json
{
  "transports": [{
    "name": "serial-primary",
    "type": "serial",
    "enabled": true,
    "serial_device": "/dev/serial/by-id/usb-HELTEC...",
    "serial_baud": 115200
  }]
}
```

For MQTT:
```json
{
  "transports": [{
    "name": "mqtt-primary",
    "type": "mqtt",
    "enabled": true,
    "endpoint": "mqtt.example.com:1883",
    "topic": "msh/US/2/e/#",
    "client_id": "mel-prod-01"
  }]
}
```

## Step 4 — Run doctor

```bash
mel doctor --config /etc/mel/mel.json
```

Doctor checks:
- Config file validity and permissions.
- Database path writability.
- Transport reachability.
- SQLite version.

Fix any `critical` or `high` findings before continuing.

## Step 5 — Start MEL

**One-shot (foreground):**
```bash
mel serve --config /etc/mel/mel.json
```

**As a systemd service** (recommended for production):
```bash
sudo cp docs/ops/systemd/mel.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now mel
sudo journalctl -fu mel
```

## Step 6 — Verify

```bash
mel status --config /etc/mel/mel.json
```

Look for `"ingest_ok": true` and at least one transport in `"connected"` state.

```bash
curl http://127.0.0.1:18080/api/v1/status
```

## Step 7 — (Optional) Enable operator-approval controls

To require explicit operator approval before MEL executes high-blast-radius
actions, add to your config:

```json
{
  "control": {
    "require_approval_for_high_blast_radius": true,
    "approval_timeout_seconds": 300,
    "require_approval_for_action_types": ["reconfigure_transport"]
  }
}
```

Then re-validate:
```bash
mel config validate --config /etc/mel/mel.json
```

## Troubleshooting

See `docs/ops/troubleshooting.md` and run:
```bash
mel diagnostics --config /etc/mel/mel.json
```
