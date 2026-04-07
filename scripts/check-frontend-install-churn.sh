#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

DRY_RUN_OUTPUT="$(make -n release-verify-chain 2>&1)"
CI_COUNT="$(printf '%s\n' "$DRY_RUN_OUTPUT" | grep -c "npm ci" || true)"

# release-verify-chain runs frontend-verify (npm ci in frontend/) and site-verify (npm ci in site/).
EXPECTED_CI_COUNT=2
if [[ "$CI_COUNT" -ne "$EXPECTED_CI_COUNT" ]]; then
  echo "[check-frontend-install-churn] FAIL: expected exactly ${EXPECTED_CI_COUNT} 'npm ci' in release-verify-chain dry-run, got $CI_COUNT" >&2
  echo "[check-frontend-install-churn] Hint: keep each Node workspace on a single deterministic npm ci (frontend-install, site-install)." >&2
  exit 1
fi

echo "[check-frontend-install-churn] PASS: release-verify-chain dry-run includes exactly ${EXPECTED_CI_COUNT} npm ci invocations (frontend + site)"
