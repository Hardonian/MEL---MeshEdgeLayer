#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
mkdir -p .tmp
CFG=.tmp/smoke.json
cp configs/mel.example.json "$CFG"
sed -i 's#./data#.tmp/data#g' "$CFG"
sed -i 's#./data/mel.db#.tmp/data/mel.db#g' "$CFG"
mkdir -p .tmp/data
./bin/mel config validate --config "$CFG"
./bin/mel doctor --config "$CFG"
(./bin/mel serve --config "$CFG" >.tmp/agent.log 2>&1 & echo $! >.tmp/agent.pid)
sleep 2
curl -fsS http://127.0.0.1:8080/healthz >/dev/null
kill "$(cat .tmp/agent.pid)"
wait "$(cat .tmp/agent.pid)" 2>/dev/null || true
