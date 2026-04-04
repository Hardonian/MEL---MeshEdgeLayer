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
  if ! grep -q -- "$pattern" "$file" 2>/dev/null; then
    fail "$label not found in $file"
  fi
  pass "$label"
}

require_file "AGENTS.md"
require_file "docs/repo-os/README.md"
require_file "docs/repo-os/verification-matrix.md"
require_file "docs/repo-os/release-readiness.md"
require_file "docs/ops/support-matrix.md"

require_pattern "AGENTS.md" "BLE ingest" "AGENTS transport matrix marks BLE ingest"
require_pattern "AGENTS.md" "Unsupported" "AGENTS marks unsupported features"
require_pattern "AGENTS.md" "Not a mesh routing stack" "AGENTS preserves non-routing claim"

require_pattern "docs/ops/support-matrix.md" "BLE ingest" "Support matrix mentions BLE ingest"
require_pattern "docs/ops/support-matrix.md" "Unsupported" "Support matrix marks unsupported"
require_pattern "docs/ops/support-matrix.md" "not a mesh routing" "Support matrix preserves non-routing claim"

require_pattern "docs/repo-os/verification-matrix.md" "support" "Verification matrix includes support"
require_pattern "docs/repo-os/release-readiness.md" "Unsupported" "Release gate includes unsupported"

pass "MEL reality-check baseline complete"
