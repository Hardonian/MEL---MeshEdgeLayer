# ADR-0001 Repo Layout

## Decision

Use a disciplined single-repo Go layout with `cmd/`, `internal/`, `migrations/`, `proto/`, `configs/`, `scripts/`, `docs/`, and a documentation-first `topologies/` scaffold for central-node and extension-node assets.

## Why

- Keeps daemon and CLI tightly aligned.
- Makes offline verification simple.
- Avoids generation drift by pinning the protobuf subset used for v0.1 verification.
- Gives future central-node and extension-node assets deterministic locations without overstating shipped runtime support.
