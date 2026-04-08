#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

CFG="${1:-demo_sandbox/mel.first-proof.json}"
SCENARIO="${DEMO_SEED_SCENARIO:-healthy-private-mesh}"

mkdir -p "$(dirname "$CFG")"

echo "[mel:first-proof] Building CLI with committed embedded assets (no Node/npm required)..."
make mel-cli-go >/dev/null

echo "[mel:first-proof] Initializing sandbox config at $CFG"
./bin/mel demo init-sandbox --out "$CFG" >/dev/null
chmod 600 "$CFG"

echo "[mel:first-proof] Seeding deterministic scenario: $SCENARIO"
./bin/mel demo seed --scenario "$SCENARIO" --config "$CFG" >/dev/null

echo "[mel:first-proof] Capturing deterministic runtime evidence bundle"
./bin/mel demo evidence-run --scenario "$SCENARIO" --config "$CFG" >/dev/null

echo ""
echo "First proof is ready."
echo ""
echo "What this proves (and does not prove):"
echo "  - Proves seeded ingest evidence, incident/action timeline artifacts, and API-visible runtime state."
echo "  - Does NOT prove live RF propagation, live route execution, BLE ingest, or HTTP ingest support."
echo ""
echo "Next commands:"
echo "  ./bin/mel doctor --config $CFG"
echo "  ./bin/mel serve --config $CFG"
echo ""
echo "Then open: http://127.0.0.1:8080"
echo ""
echo "Evidence artifacts are written under the configured data dir (demo_evidence/, demo_evidence_bundle.json)."
