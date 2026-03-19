# Protobuf extension guide

When extending Meshtastic payload support in MEL:

1. Prefer vendored schema already present in-repo.
2. Add parsing logic in `internal/meshtastic/protobuf.go`.
3. Add message typing and persistence updates in `internal/service/app.go`.
4. Preserve raw payload fallback for any field set not fully parsed.
5. Add tests that prove parsing and storage behavior.
6. Update operator docs to narrow or expand claims truthfully.

## Current typed payload support

- `text`
- `position`
- `node_info`
- `telemetry` as raw payload evidence until a full vendored schema is available
- `unknown`
