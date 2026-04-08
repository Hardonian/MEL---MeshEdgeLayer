---
name: Install/setup issue
about: First-run, build, or environment problems
title: '[SETUP] '
labels: kind:setup,area:environment,needs:triage
assignees: ''
---

## Goal

What were you trying to run?

## Environment

- Host OS / arch:
- Go version:
- Node version (`node -v`, should be 24.x for frontend/site):
- MEL commit/version:

## Exact command sequence

Paste the exact commands and first error.

```text
paste command output
```

## Doctor output

```text
./bin/mel doctor --config <path>
```

(paste output, redacted)

## Quick checks

- [ ] Read `docs/getting-started/QUICKSTART.md`
- [ ] Ran `make build` before `make smoke`
- [ ] Confirmed Node 24.x for frontend/site commands
