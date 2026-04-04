#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

DRY_RUN_OUTPUT="$(make -n release-verify-chain 2>&1)"
CI_COUNT="$(printf '%s\n' "$DRY_RUN_OUTPUT" | grep -c "npm ci" || true)"

if [[ "$CI_COUNT" -ne 1 ]]; then
  echo "[check-frontend-install-churn] FAIL: expected exactly 1 'npm ci' in release-verify-chain dry-run, got $CI_COUNT" >&2
  echo "[check-frontend-install-churn] Hint: keep frontend install centralized through frontend-install during chained verification." >&2
  exit 1
fi

echo "[check-frontend-install-churn] PASS: release-verify-chain dry-run includes exactly one npm ci invocation"
