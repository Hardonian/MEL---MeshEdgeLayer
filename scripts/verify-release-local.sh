#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

fail() {
  echo "[verify-release-local] FAIL: $1" >&2
  exit 1
}

pass() {
  echo "[verify-release-local] PASS: $1"
}

if [[ -x /usr/local/go/bin/go ]]; then
  GO_BIN="/usr/local/go/bin/go"
elif command -v go >/dev/null 2>&1; then
  GO_BIN="$(command -v go)"
else
  fail "go not found. Install Go 1.24+ or provide /usr/local/go/bin/go"
fi

GO_VER="$($GO_BIN version 2>/dev/null || true)"
if [[ ! "$GO_VER" =~ go1\.(24|25|26|27) ]]; then
  fail "unsupported Go toolchain: ${GO_VER:-unknown}. Need Go 1.24+"
fi
pass "Go toolchain: $GO_VER ($GO_BIN)"

if [[ -s "${NVM_DIR:-$HOME/.nvm}/nvm.sh" ]]; then
  # shellcheck source=/dev/null
  . "${NVM_DIR:-$HOME/.nvm}/nvm.sh"
  nvm use 24 >/dev/null 2>&1 || true
fi

NODE_VER="$(node -v 2>/dev/null || true)"
if [[ ! "$NODE_VER" =~ ^v24\. ]]; then
  fail "Node 24.x required for frontend verification. Current: ${NODE_VER:-not found}"
fi
pass "Node runtime: $NODE_VER"

if command -v python3 >/dev/null 2>&1; then
  PY="$(command -v python3)"
elif command -v python >/dev/null 2>&1; then
  PY="$(command -v python)"
else
  fail "python3/python not found; required by make product-verify"
fi
pass "Python runtime: $PY"

echo "[verify-release-local] Running release-reality verification sequence"
./scripts/repo-os-reality-check.sh
make product-verify frontend-verify test build-cli smoke

pass "Local release verification sequence completed"
