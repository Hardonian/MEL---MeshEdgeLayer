# Dependency and license inventory

Canonical answers for “what ships” and “under what terms” live here and in the root [`LICENSE`](../../LICENSE). If this file disagrees with `LICENSE`, **`LICENSE` wins**.

## Project license

- **MEL repository default:** [GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.html) — see root `LICENSE`.

## Major runtime / build components

| Area | Stack | Notes |
| --- | --- | --- |
| Daemon / CLI | Go 1.24+ | See `go.mod` for module dependencies (not stdlib-only). |
| Operator console (embedded UI) | React + Vite + TypeScript (`frontend/`) | Built into the `mel` binary via `internal/web/assets/`. |
| Public orientation site | Next.js (`site/`) | Separate from the operator console; optional deploy surface. |
| Persistence | `modernc.org/sqlite` (pure Go SQLite) | Linked as a Go module; see `go.mod`. |
| Meshtastic compatibility | In-repo protobuf schemas | Used for decode/interop boundaries; schema licensing follows upstream Meshtastic where applicable. |

## Upstream credit

Credit upstreams in product copy and release notes when user-visible behavior or wire formats come from them (e.g. Meshtastic ecosystem). Do not imply endorsement.

## Node workspaces

- **`frontend/`** and **`site/`** each carry their own `package-lock.json`. Run `npm ci` per directory for deterministic installs; repository verification runs both via `make frontend-verify` and `make site-verify` (see `Makefile`).
