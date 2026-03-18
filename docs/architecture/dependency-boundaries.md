# Dependency boundaries

- `internal/transport` does not depend on `internal/web` or CLI packages.
- `internal/privacy` and `internal/policy` depend only on `internal/config`.
- `internal/service` composes transport, storage, policy, privacy, and web layers.
- `cmd/mel` depends on service and stable internal primitives for direct operator workflows.
- Plugins consume events only and return alert objects; they do not mutate transport or storage internals directly.

The goal is one daemon with clear edges, not a future-enterprise service mesh.
