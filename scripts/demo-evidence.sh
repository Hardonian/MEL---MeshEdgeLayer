#!/usr/bin/env bash
# One-shot: init sandbox config, seed scenario, capture CLI JSON evidence.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
SCENARIO="${1:-healthy-private-mesh}"
CFG="${2:-demo_sandbox/mel.demo.json}"
mkdir -p "$(dirname "$CFG")"
./bin/mel demo init-sandbox --out "$CFG" >/dev/null
chmod 600 "$CFG"
./bin/mel demo evidence-run --scenario "$SCENARIO" --config "$CFG"
echo "See demo_evidence/ next to demo_sandbox.db and demo_evidence_bundle.json in the configured data directory."
