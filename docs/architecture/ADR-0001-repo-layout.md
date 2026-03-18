# ADR-0001 Repo Layout

## Decision

Use a disciplined single-repo Go layout with `cmd/`, `internal/`, `migrations/`, `proto/`, `configs/`, `scripts/`, and `docs/`.

## Why

- Keeps daemon and CLI tightly aligned.
- Makes offline verification simple.
- Avoids generation drift by pinning the protobuf subset used for v0.1 verification.
