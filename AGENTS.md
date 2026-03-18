# MEL agent notes

- Keep MEL honest: no mock mesh data, no dead routes, no placeholder production code.
- Prefer Go stdlib-only implementations unless a dependency is already vendored in-repo.
- Use `sqlite3` CLI for database work in this environment; keep schema and migrations deterministic.
- All operator-facing features must degrade explicitly when no transport is connected.
