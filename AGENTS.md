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
