#!/usr/bin/env bash
# MEL development environment helper.
# From repo root:  . ./scripts/dev-env.sh
# Or:             bash ./scripts/dev-env.sh
#
# Ensures Node 24.x (frontend contract) via nvm when available, and python3/python
# for make product-verify. Safe to re-run; idempotent for already-correct env.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

_fail() {
  echo "dev-env.sh: $*" >&2
  if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    exit 1
  fi
  return 1
}

export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
if [[ -s "${NVM_DIR}/nvm.sh" ]]; then
  # shellcheck source=/dev/null
  . "${NVM_DIR}/nvm.sh"
  nvm install >/dev/null
  nvm use >/dev/null
elif [[ -s "/usr/local/opt/nvm/nvm.sh" ]]; then
  # shellcheck source=/dev/null
  . "/usr/local/opt/nvm/nvm.sh"
  nvm install >/dev/null
  nvm use >/dev/null
fi

NODE_VER="$(node -v 2>/dev/null || echo "")"
if [[ ! "${NODE_VER}" =~ ^v24\. ]]; then
  _fail "need Node.js 24.x for MEL frontend (see .nvmrc / frontend/.nvmrc). Current: ${NODE_VER:-not found}. Install nvm: https://github.com/nvm-sh/nvm then run: . ./scripts/dev-env.sh"
fi

if command -v python3 >/dev/null 2>&1; then
  PY="$(command -v python3)"
elif command -v python >/dev/null 2>&1; then
  PY="$(command -v python)"
else
  _fail "need python3 (or python) on PATH for make product-verify"
fi

GO_VER="$(go version 2>/dev/null || echo "go not found")"
echo "MEL dev-env: Node ${NODE_VER}, Python ${PY}, ${GO_VER}"
