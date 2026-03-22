#!/usr/bin/env bash
# Captures a real PNG from the live MEL web UI using headless Chrome.
# Requires: ./bin/mel, google-chrome, curl.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
SCENARIO="${1:-healthy-private-mesh}"
OUT="${2:-demo_sandbox/evidence/ui-dashboard.png}"
CFG=".tmp/demo-ui-screenshot.json"
mkdir -p "$(dirname "$OUT")" .tmp demo_sandbox/data
./bin/mel demo init-sandbox --out "$CFG" >/dev/null
chmod 600 "$CFG"
./bin/mel demo seed --scenario "$SCENARIO" --config "$CFG" --skip-capture >/dev/null
./bin/mel serve --debug --config "$CFG" >.tmp/mel-screenshot.log 2>&1 &
PID=$!
cleanup() { kill "$PID" 2>/dev/null || true; wait "$PID" 2>/dev/null || true; }
trap cleanup EXIT
for i in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1:8080/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 0.2
done
CHROME=""
for c in google-chrome-stable google-chrome chromium-browser chromium; do
  if command -v "$c" >/dev/null 2>&1; then CHROME="$c"; break; fi
done
if [[ -z "$CHROME" ]]; then
  echo "no headless chrome found (install google-chrome-stable)" >&2
  exit 1
fi
USER_DATA="$(mktemp -d)"
trap 'rm -rf "$USER_DATA"' EXIT
# timeout + isolated user-data-dir avoids hanging on stale profiles in CI
timeout 45s "$CHROME" --headless=new --disable-gpu --no-sandbox --disable-dev-shm-usage \
  --user-data-dir="$USER_DATA" --window-size=1400,900 \
  --screenshot="$OUT" "http://127.0.0.1:8080/" >/dev/null 2>/dev/null
echo "wrote $OUT"
