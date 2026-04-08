# Contributing to MEL

Thanks for helping improve MEL.

This project rewards clear thinking, bounded claims, and reproducible verification.
If the code cannot prove a claim, narrow the claim.

## Fast path

1. Pick a lane: [Community START_HERE](docs/community/START_HERE.md) and [Contributor paths](docs/community/CONTRIBUTOR_PATHS.md).
2. Choose a scoped first change: [First PR paths](docs/contributor/FIRST_PR_PATHS.md).
3. Read the contract: [AGENTS.md](AGENTS.md) and [repo-os](docs/repo-os/README.md).
4. Run verification for your surface.
5. Submit PR with evidence and residual risk.

## Non-negotiables

- No fake transport support or fake live-state certainty.
- No collapsing degraded/partial/unknown into “healthy.”
- No silent auth/trust-boundary broadening.
- No submission=approval=execution shortcuts on control paths.
- No docs claims stronger than implementation + verification.

## Local development

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

### Environment notes

- Go 1.24+.
- Node 24.x for `frontend/` and `site/` commands.
- From repo root: `. ./scripts/dev-env.sh` to activate/check Node 24.
- `make smoke` requires `./bin/mel` (from `make build` or `make build-cli`).

## Contribution lanes

- **Docs / runbooks**: tighten wording, remove confusion, improve actionability.
- **Frontend**: clarify operator state semantics, improve evidence readability.
- **Go/backend**: tests, diagnostics, transport hardening, deterministic error handling.
- **Demo/scenarios**: improve fixture realism and repeatability.

Details:
- [Docs + runbooks contribution](docs/contributor/DOCS_AND_RUNBOOK_CONTRIBUTION.md)
- [Frontend contribution](docs/contributor/FRONTEND_CONTRIBUTION.md)
- [Transport contribution](docs/contributor/TRANSPORT_CONTRIBUTION.md)
- [Architecture map](docs/contributor/ARCHITECTURE_MAP.md)

## PR requirements

Every PR should include:

1. **Design intent**: what changed and why.
2. **Operator impact**: what users will notice.
3. **Verification evidence**: exact commands and outcomes.
4. **Residual risk**: what remains partial, unknown, or intentionally out of scope.
5. **Change class**: Maintenance / Leverage / Moat ([classification guide](docs/repo-os/change-classification.md)).

Use the PR template at `.github/pull_request_template.md`.

## Verification minimums by change type

- Docs-only: human proofread + link sanity + scoped checks as needed.
- Frontend: `make frontend-typecheck`, `make frontend-test`, `make lint`.
- Backend/runtime: `make test`, plus `make smoke` if runtime behavior changed.
- Capability claims or trust boundaries: run relevant repo-os audits and document results.

Canonical verification references:
- [Verification matrix](docs/repo-os/verification-matrix.md)
- [Release readiness](docs/repo-os/release-readiness.md)

## Community standards

- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- [SECURITY.md](SECURITY.md)
- [SUPPORT.md](SUPPORT.md)

MEL is GPL-3.0 licensed; see [LICENSE](LICENSE).
