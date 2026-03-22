#!/usr/bin/env bash
# Acceptance path for operational trust features (requires sqlite3, built ./bin/mel).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MEL="${MEL:-$ROOT/bin/mel}"
CFG="$ROOT/.tmp/acceptance-mel.json"
DATA="$ROOT/.tmp/acceptance-data"
rm -rf "$DATA"
mkdir -p "$DATA"
cat >"$CFG" <<'JSON'
{
  "bind": {"api": "127.0.0.1:18099"},
  "storage": {"data_dir": "DATA_DIR_PLACEHOLDER", "database_path": "DATA_DIR_PLACEHOLDER/mel.db"},
  "features": {"web_ui": false},
  "transports": []
}
JSON
# shellcheck disable=SC2016
sed -i "s|DATA_DIR_PLACEHOLDER|$DATA|g" "$CFG"
chmod 600 "$CFG"

echo "== bootstrap validate"
"$MEL" bootstrap validate --config "$CFG" --json
echo "== bootstrap run"
"$MEL" bootstrap run --config "$CFG" --json
echo "== audit verify"
"$MEL" audit verify --config "$CFG" --json
echo "== upgrade preflight"
"$MEL" upgrade preflight --config "$CFG" --json | head -c 400
echo ""
echo "GREEN: acceptance-operational-trust.sh completed"
