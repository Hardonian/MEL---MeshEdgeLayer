# Contributing to MEL

Thanks for considering a contribution. MEL values **truthful claims, deterministic behavior, and reviewable changes** over flashy demos.

If you only read one section, read **"How to land a clean first PR"** below.

## Start here

1. Read [`README.md`](README.md) and [`docs/README.md`](docs/README.md).
2. Read [`AGENTS.md`](AGENTS.md) for non-negotiable truth boundaries.
3. Pick a scoped task from [`docs/contributor/FIRST_PR_PATHS.md`](docs/contributor/FIRST_PR_PATHS.md).

## What we welcome

- Clarity improvements (UI copy, docs, onboarding) that reduce operator confusion.
- Tests that improve confidence in ingest, truth semantics, and control-path behavior.
- Diagnostics and error-surface improvements (`mel doctor`, API/status visibility).
- Deterministic fixture/scenario improvements for demos and regression.

## What will get rejected quickly

- Claims of support without implementation + verification evidence.
- UI/API wording that collapses degraded/partial/unknown into “healthy”.
- Big dependency additions with weak justification.
- Large drive-by refactors without operator value.

## Local dev baseline

```bash
make build
make lint
make test
make smoke
```

Node `24.x` is required for frontend/site tasks.

```bash
. ./scripts/dev-env.sh
```

## How to land a clean first PR

1. **Choose a small scope** (docs, one UI state, one test surface, one diagnostic path).
2. **State the claim boundary** in your PR: what changed, what did not.
3. **Run verification** appropriate to the touched surface.
4. **Include residual risk** (if any) explicitly.

Recommended first-PR lanes:
- Docs truth-tightening: broken links, overclaims, onboarding friction.
- Frontend clarity: explicit degraded/historical wording in existing components.
- Go tests: table-driven tests around existing behavior.

## Verification expectations

Minimum expectation for most PRs:

- `make lint`
- `make test`
- `make build`
- `make smoke` (when runtime behavior changed)

For frontend-heavy changes also run:
- `make frontend-typecheck`
- `make frontend-test`

For release-shaped confidence:
- `make premerge-verify`

## Issue and PR workflow

- Use [Issue templates](.github/ISSUE_TEMPLATE/).
- Use the [PR template](.github/pull_request_template.md).
- Classify major work as Maintenance / Leverage / Moat per [`docs/repo-os/change-classification.md`](docs/repo-os/change-classification.md).

## Security and privacy

- Never commit secrets, precise operator locations, or sensitive credentials.
- Use private advisories for vulnerabilities (see [`SECURITY.md`](SECURITY.md)).
- Keep privacy and trust boundaries explicit; no silent fail-open behavior.

## Community standards

- [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md)
- [`SUPPORT.md`](SUPPORT.md)
- [`SECURITY.md`](SECURITY.md)
