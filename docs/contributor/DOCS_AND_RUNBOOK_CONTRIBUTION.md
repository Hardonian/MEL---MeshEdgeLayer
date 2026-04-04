# Docs and runbook contribution

## Principles

- Every command in docs must match the **current** `Makefile` or CLI help text.
- Cross-link to canonical doctrine ([AGENTS.md](../../AGENTS.md), [docs/repo-os/terminology.md](../repo-os/terminology.md)) instead of duplicating long truth tables.
- Prefer “observed vs inferred vs unknown” language for mesh and RF topics.

## High-traffic entry points

| Doc | Purpose |
| --- | --- |
| [README.md](../../README.md) | First conversion surface |
| [docs/getting-started/README.md](../getting-started/README.md) | Onboarding sequence |
| [docs/runbooks/README.md](../runbooks/README.md) | Operational procedures |
| [docs/community/START_HERE.md](../community/START_HERE.md) | Community / role routing |

## Runbooks

Add or edit under `docs/runbooks/`. Link from `docs/runbooks/README.md` and from community docs when operators need them.

## Verification

- `make reality-check` — repo structure and scripted expectations.
- `make product-verify` — when product claims or scripted paths change.

## Contribution classification

Docs that narrow overbroad claims are usually **Maintenance** or **Leverage**; see [change classification](../repo-os/change-classification.md).
