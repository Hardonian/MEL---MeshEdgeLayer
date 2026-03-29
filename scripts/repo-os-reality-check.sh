#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

fail() {
  echo "[reality-check] FAIL: $1" >&2
  exit 1
}

pass() {
  echo "[reality-check] PASS: $1"
}

require_file() {
  local file="$1"
  [[ -f "$file" ]] || fail "missing required file: $file"
  pass "file present: $file"
}

require_pattern() {
  local file="$1"
  local pattern="$2"
  local label="$3"
  if ! rg -q "$pattern" "$file"; then
    fail "$label not found in $file"
  fi
  pass "$label"
}

require_file "AGENTS.md"
require_file "docs/repo-os/README.md"
require_file "docs/repo-os/verification-matrix.md"
require_file "docs/repo-os/release-readiness.md"
require_file "docs/ops/support-matrix.md"

require_pattern "AGENTS.md" "BLE ingest \| Unsupported" "AGENTS transport matrix marks BLE ingest unsupported"
require_pattern "AGENTS.md" "HTTP ingest \| Unsupported" "AGENTS transport matrix marks HTTP ingest unsupported"
require_pattern "AGENTS.md" "Not a mesh routing stack" "AGENTS preserves non-routing claim"

require_pattern "docs/ops/support-matrix.md" "\| BLE ingest \| Unsupported" "Support matrix marks BLE ingest unsupported"
require_pattern "docs/ops/support-matrix.md" "\| HTTP ingest \| Unsupported" "Support matrix marks HTTP ingest unsupported"
require_pattern "docs/ops/support-matrix.md" "MEL is not a mesh routing/transmit stack" "Support matrix preserves non-routing claim"

require_pattern "docs/repo-os/verification-matrix.md" "Docs \+ support-matrix alignment check" "Verification matrix includes docs/support alignment"
require_pattern "docs/repo-os/release-readiness.md" "Unsupported/partial features are labeled" "Release gate includes unsupported/partial labeling"

pass "MEL reality-check baseline complete"
