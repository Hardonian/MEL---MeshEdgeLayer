---
name: Install / setup / bootstrap
about: First-run, build, doctor, or environment problems
title: '[SETUP] '
labels: kind:setup,area:environment,needs:triage
assignees: ''
---

## What you were trying to do

(e.g. `make build`, `mel init`, `mel serve`, first browser load)

## Environment

- Host OS / arch:
- Go version (if building from source):
- Node version (`node -v`) — MEL frontend expects **24.x** (`frontend/.nvmrc`):
- MEL version or `git rev-parse --short HEAD`:

## Exact commands

Paste the full command sequence and the **first error** (not only the last line).

## Doctor output

Run `./bin/mel doctor --config <path>` (or `mel doctor`) and paste the output (redact secrets).

## What you expected vs what happened

## Before filing (quick checks)

- [ ] Read [docs/getting-started/QUICKSTART.md](https://github.com/Hardonian/MEL-MeshEdgeLayer/blob/main/docs/getting-started/QUICKSTART.md)
- [ ] Ran `make build` or `make build-cli` before `make smoke` if testing smoke
- [ ] Confirmed Node 24.x for any `frontend/` command
- [ ] Added output of `make premerge-verify` (or first failing command from it)
