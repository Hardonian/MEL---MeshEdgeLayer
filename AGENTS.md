# MEL agent notes

- Keep MEL honest: no mock mesh data, no dead routes, no placeholder production code.
- Prefer Go stdlib-only implementations unless a dependency is already vendored in-repo.
- Use `sqlite3` CLI for database work in this environment; keep schema and migrations deterministic.
- All operator-facing features must degrade explicitly when no transport is connected.

## Central and extension node layout

- Keep hub-side assets under `topologies/central-node/` and constrained device assets under `topologies/extension-node/`.
- Small static settings belong in the relevant `config/` directory.
- Larger stateful or data-heavy assets belong in the relevant `memory-management/` directory.
- Do not let scaffolding imply shipped support: if a node runtime, transport, OTA path, or persistent storage path is not implemented and verified, say so explicitly in docs and code comments.

## Cursor Cloud specific instructions

### Services overview

| Service | Command | Port | Notes |
|---|---|---|---|
| MEL backend | `./bin/mel serve --config configs/mel.json` | 8080 | Serves JSON API, web dashboard, and healthz. Init first with `./bin/mel init --config configs/mel.json --force`. |
| Frontend dev | `cd frontend && npx vite --port 3000` | 3000 | Vite dev server; proxies `/api` to backend on 8080. |

### Build, lint, test

- Standard commands are in the `Makefile` and `CONTRIBUTING.md`: `make build`, `make lint`, `make test`, `make smoke`.
- Frontend: `npm run dev`, `npm run lint`, `npm run typecheck`, `npx vitest run` (all from `frontend/`).
- Go 1.24+ is required (see `go.mod`). The VM update script installs it to `/usr/local/go`.

### Caveats discovered during setup

- The codebase has pre-existing compile errors on `main` as of commit `58621a9`. These are field-name mismatches between struct definitions and usages (e.g. `ev.ID` vs `ev.EventID` in `TimelineEvent`, `s.writeJSON` vs package-level `writeJSON`). A fix commit is included in the environment setup branch.
- Several test packages (`internal/db`, `internal/service`, `internal/web`, `internal/upgrade`, `internal/version`) have pre-existing failures unrelated to environment setup. These are SQL schema drift and type-mismatch issues in test files.
- `make smoke` (which runs `scripts/smoke.sh`) requires `python3` for config file patching and the built binaries in `bin/`.
- No Docker, no external databases, and no network services are required — SQLite is embedded via `modernc.org/sqlite`.
- Transports (Serial, TCP, MQTT) are optional and degrade explicitly to idle mode when unconfigured.
