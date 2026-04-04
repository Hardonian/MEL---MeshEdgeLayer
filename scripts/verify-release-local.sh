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

./scripts/require-node24.sh --context "verify-release-local frontend verification"
pass "Node runtime: $(node -v)"

if command -v python3 >/dev/null 2>&1; then
  PY="$(command -v python3)"
elif command -v python >/dev/null 2>&1; then
  PY="$(command -v python)"
else
  fail "python3/python not found; required by make product-verify"
fi
pass "Python runtime: $PY"

make check-frontend-install-churn

if [[ "${VERIFY_SKIP_CLEAN_INSTALL:-0}" == "1" ]]; then
  echo "[verify-release-local] Running FAST local-only verification sequence (VERIFY_SKIP_CLEAN_INSTALL=1)"
  ./scripts/repo-os-reality-check.sh
  make product-verify frontend-verify-fast test build-cli smoke
  pass "FAST local-only verification completed (not release-grade; skipped clean frontend install)"
  exit 0
fi

echo "[verify-release-local] Running release-reality verification sequence"
./scripts/repo-os-reality-check.sh
make release-verify-chain

pass "Local release verification sequence completed"
