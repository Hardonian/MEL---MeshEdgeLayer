# Evaluate MEL in 10 minutes

This guide is for skeptical first users. It focuses on proving what MEL actually does without overstating Meshtastic compatibility.

## Path A: prove MEL with a real supported transport

Choose one:

- serial direct-node
- TCP direct-node
- MQTT ingest

Then:

```bash
make build
cp configs/mel.serial.example.json .tmp/mel.json
python3 - <<'PY'
from pathlib import Path
p = Path('.tmp/mel.json')
text = p.read_text()
text = text.replace('./data/mel.db', '.tmp/data/mel.db').replace('./data', '.tmp/data')
p.write_text(text)
PY
./bin/mel config validate --config .tmp/mel.json
./bin/mel doctor --config .tmp/mel.json
./bin/mel serve --config .tmp/mel.json
```

Now verify:

1. open <http://127.0.0.1:8080/api/v1/status>
2. watch transport health move to `connected but idle` or `live data flowing`
3. run `./bin/mel status --config .tmp/mel.json`
4. run `./bin/mel nodes --config .tmp/mel.json`
5. run `./bin/mel export --config .tmp/mel.json --out .tmp/export.json`
6. run `./bin/mel doctor --config .tmp/mel.json` again after ingest

What this proves:

- MEL can ingest from a supported source.
- MEL persists observations.
- MEL exposes local truth through CLI, API, and UI.

What this does **not** prove:

- BLE support
- radio transmit support
- admin/control operations
- full protobuf coverage

## Path B: repo-local self-test of MEL's ingest/storage/UI stack

Use this only when you want to prove the MEL codepath itself without requiring hardware. This is a **MEL self-test**, not proof of direct-node interoperability.

```bash
make build
cp configs/mel.mqtt-only.example.json .tmp/mel.json
python3 - <<'PY'
from pathlib import Path
p = Path('.tmp/mel.json')
text = p.read_text()
text = text.replace('127.0.0.1:1883', '127.0.0.1:18830')
text = text.replace('msh/US/2/e/#', 'msh/US/2/e/test')
text = text.replace('./data/mel.db', '.tmp/data/mel.db').replace('./data', '.tmp/data')
text = text.replace('"client_id": "mel-local"', '"client_id": "mel-local", "reconnect_seconds": 1')
p.write_text(text)
PY
./bin/mel config validate --config .tmp/mel.json
./bin/mel dev-simulate-mqtt --endpoint 127.0.0.1:18830 --topic msh/US/2/e/test
```

In another shell:

```bash
./bin/mel serve --config .tmp/mel.json
curl -fsS http://127.0.0.1:8080/api/v1/status
curl -fsS http://127.0.0.1:8080/api/v1/messages
./bin/mel nodes --config .tmp/mel.json
```

What this proves:

- the MQTT subscribe path works,
- packet normalization works,
- SQLite persistence works,
- UI/API/CLI surfaces show real stored observations.

What this does **not** prove:

- radio hardware access,
- serial permissions,
- TCP stream interoperability,
- real broker posture or production deployment hardening.
