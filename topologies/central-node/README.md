# Central node scaffold

The central node is the hub-side runtime boundary for MEL deployments.

Expected placement:

- `config/` for small, operator-edited configuration artifacts such as transport settings, topic naming, and service-level defaults.
- `memory-management/` for larger state, persistence, retention, aggregation, or other data-heavy modules as they become real and verified.

Current truth:

- MEL's production runtime still lives under `cmd/`, `internal/`, `configs/`, and `migrations/`.
- This scaffold does not claim that separate central-node executables or storage modules already exist.
