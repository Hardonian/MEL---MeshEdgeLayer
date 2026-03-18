# First 10 minutes: prove MEL is real

This is the skeptical-operator flow.

## 1. Build the binaries

```bash
make build
```

## 2. Pick one real supported transport

### Serial

```bash
mkdir -p .tmp/eval
cp configs/mel.serial.example.json .tmp/eval/mel.json
python3 - <<'PY'
from pathlib import Path
p = Path('.tmp/eval/mel.json')
text = p.read_text()
text = text.replace('./data', '.tmp/eval/data').replace('/dev/ttyUSB0', '/dev/serial/by-id/REPLACE_ME')
p.write_text(text)
PY
```

### MQTT

```bash
mkdir -p .tmp/eval
cp configs/mel.mqtt-only.example.json .tmp/eval/mel.json
python3 - <<'PY'
from pathlib import Path
p = Path('.tmp/eval/mel.json')
text = p.read_text().replace('./data', '.tmp/eval/data')
p.write_text(text)
PY
```

## 3. Validate and inspect the config

```bash
./bin/mel config validate --config .tmp/eval/mel.json
./bin/mel doctor --config .tmp/eval/mel.json
./bin/mel transports list --config .tmp/eval/mel.json
```

What to expect:

- a real serial/TCP problem is surfaced as a real finding,
- zero enabled transports is surfaced explicitly,
- unsupported transport types stay marked unsupported.

## 4. Start MEL

```bash
./bin/mel serve --config .tmp/eval/mel.json
```

In another shell:

```bash
curl -fsS http://127.0.0.1:8080/healthz
curl -fsS http://127.0.0.1:8080/api/v1/status | jq .
curl -fsS http://127.0.0.1:8080/api/v1/transports | jq .
```

## 5. Wait for one real packet

Once the configured source emits a packet, verify:

```bash
./bin/mel status --config .tmp/eval/mel.json
./bin/mel nodes --config .tmp/eval/mel.json
curl -fsS http://127.0.0.1:8080/api/v1/messages | jq '.messages[0]'
sqlite3 .tmp/eval/data/mel.db 'select transport_name, from_node, portnum, rx_time from messages order by id desc limit 5;'
```

## 6. Check privacy/export/backup claims

```bash
./bin/mel privacy audit --format text --config .tmp/eval/mel.json
./bin/mel export --config .tmp/eval/mel.json --out .tmp/eval/export.json
./bin/mel import validate --bundle .tmp/eval/export.json
./bin/mel backup create --config .tmp/eval/mel.json --out .tmp/eval/backup.tgz
./bin/mel backup restore --bundle .tmp/eval/backup.tgz --dry-run --destination .tmp/eval/restore-preview
```

## 7. Confirm the boundaries

Before deciding MEL is ready for your environment, confirm that you are **not** expecting:

- BLE ingest,
- HTTP direct-node ingest,
- packet transmit or publish,
- admin/config control of the node,
- authoritative node inventory fetch,
- a write-path restore flow.
