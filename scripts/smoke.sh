#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
mkdir -p .tmp
CFG=.tmp/smoke.json
./bin/mel init --config "$CFG" --force >/dev/null
if [ ! -f "$CFG" ]; then
  if [ -f configs/mel.generated.json ]; then
    cp configs/mel.generated.json "$CFG"
  fi
fi
if [ ! -f "$CFG" ]; then
  echo "smoke: expected config fixture at $CFG after init" >&2
  exit 1
fi
python3 - <<'PY'
from pathlib import Path
p = Path('.tmp/smoke.json')
text = p.read_text()
text = text.replace('./data/mel.db', '.tmp/data/mel.db').replace('./data', '.tmp/data')
p.write_text(text)
PY
chmod 600 "$CFG"
mkdir -p .tmp/data
./bin/mel config validate --config "$CFG"
./bin/mel doctor --config "$CFG" || true
./bin/mel status --config "$CFG" >/dev/null
./bin/mel replay --config "$CFG" --limit 5 >/dev/null
./bin/mel export --config "$CFG" --out .tmp/export.json >/dev/null
./bin/mel import validate --bundle .tmp/export.json
./bin/mel backup create --config "$CFG" --out .tmp/backup.tgz >/dev/null
./bin/mel backup restore --bundle .tmp/backup.tgz --dry-run --destination .tmp/restore-preview
(./bin/mel serve --debug --config "$CFG" >.tmp/agent.log 2>&1 & echo $! >.tmp/agent.pid)
sleep 2
curl -fsS http://127.0.0.1:8080/healthz >/dev/null
curl -fsS http://127.0.0.1:8080/readyz >/dev/null || true
curl -fsS http://127.0.0.1:8080/api/v1/readyz >/dev/null || true
curl -fsS http://127.0.0.1:8080/api/v1/status >/dev/null
curl -fsS http://127.0.0.1:8080/api/v1/diagnostics >/dev/null
curl -fsS http://127.0.0.1:8080/api/v1/messages >/dev/null
curl -fsS http://127.0.0.1:8080/metrics >/dev/null
kill "$(cat .tmp/agent.pid)"
wait "$(cat .tmp/agent.pid)" 2>/dev/null || true
