# Dependency and license inventory

- Go stdlib runtime dependencies only for the main daemon and CLI (see `go.mod` / `go.sum` for the full module graph).
- Meshtastic protobuf schema files are stored in-repo for compatibility parsing (upstream Meshtastic project; respect their licensing on schema-derived artifacts).
- SQLite is used through the system `sqlite3` CLI in contributor and CI environments for deterministic checks.
- **MEL project license:** GNU General Public License v3.0 — see [`LICENSE`](../../LICENSE) at the repository root.

Bundled frontend and optional tooling pull in additional licenses (npm lockfiles list per-package SPDX identifiers). Those are **third-party** licenses; they do not change the license of MEL’s own source unless explicitly stated.
