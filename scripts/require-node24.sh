#!/usr/bin/env bash
set -euo pipefail

CONTEXT="${1:-frontend verification}"
if [[ "$CONTEXT" == "--context" ]]; then
  CONTEXT="${2:-frontend verification}"
fi

NODE_VER="$(node -v 2>/dev/null || true)"
if [[ -z "$NODE_VER" ]]; then
  echo "[runtime-contract] Node.js not found for ${CONTEXT}. MEL frontend requires Node 24.x." >&2
  echo "[runtime-contract] Run '. ./scripts/dev-env.sh' or install Node 24 via nvm, then retry." >&2
  exit 1
fi

if [[ ! "$NODE_VER" =~ ^v24\. ]]; then
  echo "[runtime-contract] Node 24.x required for ${CONTEXT}. Detected ${NODE_VER}." >&2
  echo "[runtime-contract] Run '. ./scripts/dev-env.sh' (repo root) to activate Node 24, then retry." >&2
  exit 1
fi
