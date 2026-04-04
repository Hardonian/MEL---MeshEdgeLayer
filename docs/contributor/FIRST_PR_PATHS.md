# First PR paths

Small, reviewable changes that respect MEL’s truth boundaries and verification chain.

## Docs-only (good first contribution)

- Fix a broken link or outdated command in `docs/` or `README.md`.
- Add cross-links between [community](../community/START_HERE.md) and an existing runbook.
- Narrow wording where docs overclaim transport or live state.

**Verify**: `make reality-check` if product paths change; otherwise human review.

## Frontend (UI copy, empty states, a11y)

- Adjust copy in `frontend/src/pages/` or `frontend/src/components/` to clarify degraded / historical semantics.
- Add tests alongside pages that already have `*.test.tsx` neighbors.

**Verify**: `make frontend-typecheck`, `make frontend-test`, `make lint`.

## Go (tests, logging, error surfaces)

- Add table-driven tests next to the package you touch.
- Improve error messages surfaced in `mel doctor` or API responses **without** changing external contracts unless coordinated.

**Verify**: `go test ./path/...`, `make test`, `make smoke` if runtime behavior changes.

## Demo / fixtures

- Add a scenario in `internal/demo/catalog.go` with tests in `internal/demo/demo_test.go`.

**Verify**: `make demo-verify`.

## What to avoid in a first PR

- Declaring new supported transports without code + tests + README matrix update.
- Collapsing degraded/partial/unknown into “healthy” in UI or API.
- Large dependency additions (Go stdlib-first policy per [CONTRIBUTING.md](../../CONTRIBUTING.md)).

## PR hygiene

Follow [CONTRIBUTING.md](../../CONTRIBUTING.md) and classify work (Maintenance / Leverage / Moat) per [change classification](../repo-os/change-classification.md).
