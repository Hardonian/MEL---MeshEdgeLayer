# Contributing to MEL

Thanks for improving MEL.

MEL values bounded claims, deterministic behavior, and verification evidence.
If code cannot prove a claim, narrow the claim.

## Fast contribution path

1. Read constraints: [AGENTS.md](AGENTS.md) and [repo-os](docs/repo-os/README.md).
2. Pick a safe lane: [First PR paths](docs/contributor/FIRST_PR_PATHS.md).
3. Build + verify the touched surface.
4. Submit PR with exact command output and residual risk.

## Non-negotiable invariants

- Clarity improvements (UI copy, docs, onboarding) that reduce operator confusion.
- Tests that improve confidence in ingest, truth semantics, and control-path behavior.
- Diagnostics and error-surface improvements (`mel doctor`, API/status visibility).
- Deterministic fixture/scenario improvements for demos and regression.

## Local development baseline

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

Environment notes:
- Go 1.24+
- Node 24.x for `frontend/` and `site/`
- From repo root: `. ./scripts/dev-env.sh`

Recommended first-PR lanes:
- Docs truth-tightening: broken links, overclaims, onboarding friction.
- Frontend clarity: explicit degraded/historical wording in existing components.
- Go tests: table-driven tests around existing behavior.

- **Docs / runbooks:** clarify operator decisions, remove drift, reduce ambiguity.
- **Frontend:** improve evidence semantics and degraded-state clarity.
- **Go/backend:** transport reliability, diagnostics, deterministic behavior.
- **Demo/scenarios:** strengthen fixture realism and reproducibility.

Deep links:
- [Docs + runbooks contribution](docs/contributor/DOCS_AND_RUNBOOK_CONTRIBUTION.md)
- [Frontend contribution](docs/contributor/FRONTEND_CONTRIBUTION.md)
- [Transport contribution](docs/contributor/TRANSPORT_CONTRIBUTION.md)
- [Architecture map](docs/contributor/ARCHITECTURE_MAP.md)

- `make lint`
- `make test`
- `make build`
- `make smoke` (when runtime behavior changed)

For frontend-heavy changes also run:
- `make frontend-typecheck`
- `make frontend-test`

1. **Design intent** (what changed + why)
2. **Operator impact** (what users notice)
3. **Verification evidence** (exact commands + outcomes)
4. **Residual risk** (partial/unknown/out-of-scope)
5. **Change class**: Maintenance / Leverage / Moat ([guide](docs/repo-os/change-classification.md))

Template: `.github/pull_request_template.md`

- Use [Issue templates](.github/ISSUE_TEMPLATE/).
- Use the [PR template](.github/pull_request_template.md).
- Classify major work as Maintenance / Leverage / Moat per [`docs/repo-os/change-classification.md`](docs/repo-os/change-classification.md).

- Docs-only: human proofread + link sanity + scoped checks.
- Frontend: `make frontend-typecheck`, `make frontend-test`, `make lint`.
- Backend/runtime: `make test`; add `make smoke` when runtime behavior changes.
- Capability/trust-boundary changes: run relevant repo-os audits and document results.

References:
- [Verification matrix](docs/repo-os/verification-matrix.md)
- [Release readiness](docs/repo-os/release-readiness.md)

## Community standards

- [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md)
- [`SUPPORT.md`](SUPPORT.md)
- [`SECURITY.md`](SECURITY.md)
