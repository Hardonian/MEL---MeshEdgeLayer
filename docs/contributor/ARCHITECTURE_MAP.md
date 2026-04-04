# Architecture map (orientation)

High-level map of where concerns live in this repository. For doctrine and verification, use [docs/repo-os/README.md](../repo-os/README.md).

## Runtime and CLI

| Area | Path | Notes |
| --- | --- | --- |
| CLI entry | `cmd/mel/` | Operator commands including `serve`, `doctor`, `demo` |
| Agent binary | `cmd/mel-agent/` | Separate build target |
| Embedded web assets | `internal/web/assets/` | Populated by `make build` from `frontend/dist/` |

## Backend (Go)

| Area | Path | Notes |
| --- | --- | --- |
| Core domain logic | `internal/` | Large tree: search by package name when diving deep |
| Migrations | `migrations/` | Schema evolution |
| Demo scenarios | `internal/demo/` | Catalog, seed, tests |

## Frontend

| Area | Path | Notes |
| --- | --- | --- |
| SPA | `frontend/` | Vite + React + TypeScript |
| Pages | `frontend/src/pages/` | Route-level screens |
| Components | `frontend/src/components/` | Shared UI and feature panels |
| Hooks | `frontend/src/hooks/` | API polling and context |

## Configuration and examples

| Area | Path |
| --- | --- |
| Example configs | `configs/`, `examples/` |
| Topology cookbooks | `topologies/` |

## Automation

| Area | Path |
| --- | --- |
| Makefile targets | `Makefile` |
| Smoke / verify scripts | `scripts/` |
| CI | `.github/workflows/` |

## Extension points (today)

- **Transports / ingest**: see [transport-interface-contract.md](transport-interface-contract.md); stay within the [README](../../README.md) support matrix when changing behavior docs.
- **Protobuf / payloads**: [protobuf-extension-guide.md](protobuf-extension-guide.md).
- **Frontend**: typed API clients and hooks — prefer extending existing patterns in `frontend/src/hooks/useApi.tsx` and related modules.
