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

- No fake transport support or fake live-state certainty.
- No collapsing degraded/partial/unknown into “healthy.”
- No silent auth/trust-boundary broadening.
- No submission=approval=execution shortcuts on control paths.
- No docs claims stronger than implementation + verification.

## Local development baseline

```bash
make build
make lint
make test
make smoke
```

Release-shaped confidence:

```bash
make premerge-verify
```

Environment notes:
- Go 1.24+
- Node 24.x for `frontend/` and `site/`
- From repo root: `. ./scripts/dev-env.sh`

## Contribution lanes

- **Docs / runbooks:** clarify operator decisions, remove drift, reduce ambiguity.
- **Frontend:** improve evidence semantics and degraded-state clarity.
- **Go/backend:** transport reliability, diagnostics, deterministic behavior.
- **Demo/scenarios:** strengthen fixture realism and reproducibility.

Deep links:
- [Docs + runbooks contribution](docs/contributor/DOCS_AND_RUNBOOK_CONTRIBUTION.md)
- [Frontend contribution](docs/contributor/FRONTEND_CONTRIBUTION.md)
- [Transport contribution](docs/contributor/TRANSPORT_CONTRIBUTION.md)
- [Architecture map](docs/contributor/ARCHITECTURE_MAP.md)

## PR requirements

Every PR should include:

1. **Design intent** (what changed + why)
2. **Operator impact** (what users notice)
3. **Verification evidence** (exact commands + outcomes)
4. **Residual risk** (partial/unknown/out-of-scope)
5. **Change class**: Maintenance / Leverage / Moat ([guide](docs/repo-os/change-classification.md))

Template: `.github/pull_request_template.md`

## Verification minimums by change type

- Docs-only: human proofread + link sanity + scoped checks.
- Frontend: `make frontend-typecheck`, `make frontend-test`, `make lint`.
- Backend/runtime: `make test`; add `make smoke` when runtime behavior changes.
- Capability/trust-boundary changes: run relevant repo-os audits and document results.

References:
- [Verification matrix](docs/repo-os/verification-matrix.md)
- [Release readiness](docs/repo-os/release-readiness.md)

## Community standards

- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- [SECURITY.md](SECURITY.md)
- [SUPPORT.md](SUPPORT.md)

MEL is GPL-3.0 licensed; see [LICENSE](LICENSE).
