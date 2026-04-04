#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

fail() {
  echo "[product-verify] FAIL: $1" >&2
  exit 1
}

pass() {
  echo "[product-verify] PASS: $1"
}

require_file() {
  local file="$1"
  [[ -f "$file" ]] || fail "missing required file: $file"
  pass "file present: $file"
}

echo "=== MEL Product System Verification ==="

echo "--- Checking required product documentation ---"
require_file "docs/product/PRODUCT_OVERVIEW.md"
require_file "docs/product/CAPABILITY_MATRIX.md"
require_file "docs/product/FEATURE_MATURITY.md"
require_file "docs/product/EDITION_PACKAGING.md"
require_file "docs/product/DIFFERENTIATION_AND_MOAT.md"

echo "--- Checking required release documentation ---"
require_file "docs/release/RELEASE_CRITERIA.md"
require_file "docs/release/UPGRADE_AND_MIGRATION.md"
require_file "docs/release/BACKUP_AND_RESTORE.md"
require_file "docs/release/SUPPORT_RUNBOOK.md"
require_file "docs/release/SECURITY_MODEL.md"
require_file "docs/release/KNOWN_LIMITATIONS.md"

echo "--- Checking required getting-started documentation ---"
require_file "docs/getting-started/QUICKSTART.md"
require_file "docs/getting-started/FIRST_INCIDENT_GUIDE.md"

echo "--- Checking required internal documentation ---"
require_file "docs/internal/private/PRICING_STRATEGY.md"
require_file "docs/internal/private/RISK_REGISTER.md"

echo "--- Verifying feature maturity labels ---"
FEATURE_FILE="docs/product/FEATURE_MATURITY.md"
for label in "GA" "Beta" "Experimental" "Unsupported" "Roadmap"; do
  if ! grep -q "### ${label}" "$FEATURE_FILE" 2>/dev/null; then
    fail "feature maturity missing section: $label"
  fi
done
pass "All feature maturity labels present (GA, Beta, Experimental, Unsupported, Roadmap)"

echo "--- Verifying transport unsupported claims ---"
for file in "docs/product/CAPABILITY_MATRIX.md" "docs/product/FEATURE_MATURITY.md" "docs/release/KNOWN_LIMITATIONS.md"; do
  if [[ -f "$file" ]]; then
    if ! grep -q "BLE" "$file" 2>/dev/null || ! grep -qi "unsupported" "$file" 2>/dev/null; then
      fail "$file must keep explicit unsupported posture for BLE"
    fi
    if ! grep -q "HTTP" "$file" 2>/dev/null || ! grep -qi "unsupported" "$file" 2>/dev/null; then
      fail "$file must keep explicit unsupported posture for HTTP"
    fi
    pass "$file has BLE and HTTP unsupported claims"
  fi
done

echo "--- Checking for broken markdown links ---"
ERRORS=0
for dir in "docs/product" "docs/release" "docs/getting-started" "docs/internal/private"; do
  if [[ -d "$dir" ]]; then
    find "$dir" -name "*.md" 2>/dev/null | while read -r md; do
      grep -oE '\]\([^)]+\)' "$md" 2>/dev/null | sed 's/](\(.*\))/\1/' | sed 's/#.*//' | sed 's/?.*//' | sort -u | while read -r link; do
        skip=false
        if [[ "$link" == http://* ]] || [[ "$link" == https://* ]] || [[ "$link" == "#"* ]] || [[ "$link" == mailto:* ]]; then
          skip=true
        fi
        if [[ "$skip" == false && -n "$link" && ! -f "$link" ]]; then
          echo "[product-verify] WARN: broken link in $md -> $link"
          ERRORS=$((ERRORS + 1))
        fi
      done
    done
  fi
done

if [[ $ERRORS -gt 0 ]]; then
  echo "[product-verify] WARN: $ERRORS broken links found (see above)"
else
  pass "No broken markdown links detected"
fi

echo "=== Product system verification complete ==="
