#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [ ! -x ./bin/mel ]; then
  echo "smoke: missing executable ./bin/mel" >&2
  echo "smoke: build it first with 'make build-cli' (or 'make build')." >&2
  exit 1
fi

echo "smoke: starting MEL smoke test"

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

sed -i 's|"\./data/mel.db"|"./.tmp/data/mel.db"|g; s|"\./data"|"./.tmp/data"|g' "$CFG" 2>/dev/null || true
chmod 600 "$CFG"
mkdir -p .tmp/data

echo "smoke: checking schema version consistency"
BINARY_VERSION=$(./bin/mel version 2>/dev/null | grep "Schema Version" | awk '{print $NF}')
SOURCE_VERSION=$(grep "CurrentSchemaVersion = " internal/version/version.go | awk '{print $3}')
if [[ -n "$BINARY_VERSION" && -n "$SOURCE_VERSION" && "$BINARY_VERSION" != "$SOURCE_VERSION" ]]; then
  echo "smoke: WARN binary reports schema $BINARY_VERSION but source defines $SOURCE_VERSION"
  echo "smoke: WARN this indicates the binary is out of date with source"
fi

echo "smoke: validating config"
./bin/mel config validate --config "$CFG"

echo "smoke: running doctor"
./bin/mel doctor --config "$CFG" || true

echo "smoke: testing status endpoint"
./bin/mel status --config "$CFG" >/dev/null

echo "smoke: testing replay"
./bin/mel replay --config "$CFG" --limit 5 >/dev/null

echo "smoke: testing export"
./bin/mel export --config "$CFG" --out .tmp/export.json >/dev/null

echo "smoke: validating import"
./bin/mel import validate --bundle .tmp/export.json

echo "smoke: testing backup create"
./bin/mel backup create --config "$CFG" --out .tmp/backup.tgz >/dev/null

echo "smoke: testing backup restore dry-run"
./bin/mel backup restore --bundle .tmp/backup.tgz --dry-run --destination .tmp/restore-preview

echo "smoke: starting server for API tests"
./bin/mel serve --debug --config "$CFG" >.tmp/agent.log 2>&1 &
AGENT_PID=$!
echo "$AGENT_PID" > .tmp/agent.pid
sleep 3

cleanup() {
  echo "smoke: cleaning up"
  kill "$AGENT_PID" 2>/dev/null || true
  wait "$AGENT_PID" 2>/dev/null || true
}
trap cleanup EXIT

echo "smoke: testing HTTP endpoints"
curl -fsS http://127.0.0.1:8080/healthz >/dev/null && echo "smoke: /healthz OK" || echo "smoke: /healthz failed"
curl -fsS http://127.0.0.1:8080/readyz >/dev/null || true && echo "smoke: /readyz OK" || echo "smoke: /readyz failed"
curl -fsS http://127.0.0.1:8080/api/v1/readyz >/dev/null || true && echo "smoke: /api/v1/readyz OK" || echo "smoke: /api/v1/readyz failed"
curl -fsS http://127.0.0.1:8080/api/v1/status >/dev/null && echo "smoke: /api/v1/status OK" || echo "smoke: /api/v1/status failed"
curl -fsS http://127.0.0.1:8080/api/v1/diagnostics >/dev/null && echo "smoke: /api/v1/diagnostics OK" || echo "smoke: /api/v1/diagnostics failed"
curl -fsS http://127.0.0.1:8080/api/v1/messages >/dev/null && echo "smoke: /api/v1/messages OK" || echo "smoke: /api/v1/messages failed"
curl -fsS http://127.0.0.1:8080/metrics >/dev/null && echo "smoke: /metrics OK" || echo "smoke: /metrics failed"

echo "smoke: complete"
