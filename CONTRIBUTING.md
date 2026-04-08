# Contributing to MEL

Thanks for improving MEL. Keep claims bounded to implementation and evidence.

## 1) Start safely

1. Read [AGENTS.md](AGENTS.md) and [docs/repo-os/README.md](docs/repo-os/README.md).
2. Pick a scoped lane from [docs/contributor/FIRST_PR_PATHS.md](docs/contributor/FIRST_PR_PATHS.md).
3. Run the relevant verification commands.
4. Submit exact command output plus residual risk in your PR.

## 2) Non-negotiables

- Do not imply unsupported capability (BLE ingest, HTTP ingest, MEL RF routing).
- Do not collapse stale/historical/partial/degraded/unknown into “healthy/live”.
- Do not treat local inference as canonical truth.
- Do not bypass control lifecycle semantics (submission ≠ approval ≠ execution ≠ audit).

## 3) Local dev baseline

```bash
. ./scripts/dev-env.sh
make lint
make test
make build
make smoke
```

For frontend-heavy changes also run:

```bash
make frontend-typecheck
make frontend-test
```

## 4) Good first contribution lanes

- Docs truth-tightening (broken links, overclaims, funnel clarity).
- Operator-facing degraded/unknown wording clarity in existing UI.
- Deterministic tests for ingest, control lifecycle, and evidence semantics.
- Diagnostics/runbook hardening without broadening capability claims.

## 5) What a good PR includes

1. **Why:** problem and scope.
2. **Truth boundary:** what this change does and does *not* claim.
3. **Operator impact:** UI/API/CLI/docs changes.
4. **Verification evidence:** exact commands + outcomes.
5. **Residual risk:** caveats and known unknowns.
6. **Change class:** Maintenance / Leverage / Moat ([guide](docs/repo-os/change-classification.md)).

Template: [.github/pull_request_template.md](.github/pull_request_template.md)

## 6) Community and intake

- Bugs/setup/docs/features: use [.github/ISSUE_TEMPLATE](.github/ISSUE_TEMPLATE)
- Conduct: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- Security: [SECURITY.md](SECURITY.md)
- Support: [SUPPORT.md](SUPPORT.md)
