# Contributor paths by role

Every path below points at **real directories and commands** in this repository. If something drifts, fix the doc in the same PR as the code.

## Everyone

| Step | Where |
| --- | --- |
| Build CLI + embed frontend | `make build` or `make build-cli` |
| Lint / test | `make lint`, `make test` |
| Frontend checks | `make frontend-typecheck`, `make frontend-test` (Node **24.x** — `. ./scripts/dev-env.sh`) |
| Release-style chain | `make premerge-verify` |
| Repo doctrine | [AGENTS.md](../../AGENTS.md), [docs/repo-os/README.md](../repo-os/README.md) |

## Frontend (React / Vite / TypeScript)

| Topic | Doc / location |
| --- | --- |
| Local dev | [frontend/README.md](../../frontend/README.md) |
| Contribution guide | [FRONTEND_CONTRIBUTION.md](../contributor/FRONTEND_CONTRIBUTION.md) |
| Pages | `frontend/src/pages/` |
| Shared UI | `frontend/src/components/` |
| API hooks | `frontend/src/hooks/` |

Verification: `make frontend-lint` (via `make lint`), `make frontend-typecheck`, `make frontend-test`, then `make build` so assets land in `internal/web/assets/`.

## Transport, ingest, decoders

| Topic | Doc / location |
| --- | --- |
| Transport contract | [transport-interface-contract.md](../contributor/transport-interface-contract.md) |
| Protobuf extension | [protobuf-extension-guide.md](../contributor/protobuf-extension-guide.md) |
| Go ingest / workers | `internal/` (search for transport implementations; MQTT/serial/TCP are the supported paths per README matrix) |
| CLI surface | `cmd/mel/` |

Verification: `make test`, targeted `go test ./internal/...` for touched packages, `make smoke` when runtime behavior changes, plus [transport truth audit](../repo-os/transport-truth-audit.md) if semantics move.

## Docs and runbooks

| Topic | Location |
| --- | --- |
| Getting started | `docs/getting-started/` |
| Operations | `docs/ops/`, `docs/runbooks/` |
| Product | `docs/product/` |
| Contribution guide | [DOCS_AND_RUNBOOK_CONTRIBUTION.md](../contributor/DOCS_AND_RUNBOOK_CONTRIBUTION.md) |

Verification: link check by eye in PR; run `make reality-check` or `make product-verify` when product claims or paths change.

## Demos and scenarios

| Topic | Location |
| --- | --- |
| Scenario catalog (Go) | `internal/demo/catalog.go` |
| CLI | `./bin/mel demo --help` and `cmd/mel/commands.go` |
| Evidence script | `scripts/demo-evidence.sh` |
| Docs | [Scenario library](SCENARIO_LIBRARY.md), [launch and demo runbook](../runbooks/launch-and-demo.md) |

Verification: `make demo-verify` after `make build-cli`.

## Architecture orientation

See [ARCHITECTURE_MAP.md](../contributor/ARCHITECTURE_MAP.md) for a folder-level map and boundaries.
