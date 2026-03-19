# Central and extension node layout

## Purpose

This document defines where future node-role-specific assets belong without overstating current implementation status.

MEL today is a central-side ingest and observability service. The repo does **not** currently ship extension-node firmware, OTA services, or separate node-specific runtimes. The layout below exists so future additions land in deterministic locations and keep operator-facing truth intact.

## Repository placement

### Central node

Use `topologies/central-node/` for hub-side assets that support MEL deployments.

- `topologies/central-node/config/`
  - small static configuration files
  - transport endpoints, topic maps, service defaults, and other operator-auditable settings
- `topologies/central-node/memory-management/`
  - larger stateful modules or assets
  - persistence orchestration, aggregation state, retention helpers, and other data-heavy concerns

### Extension node

Use `topologies/extension-node/` for constrained device assets once they exist.

- `topologies/extension-node/config/`
  - small device settings such as credentials, thresholds, and timing knobs
- `topologies/extension-node/memory-management/`
  - only bounded local buffers or compact retry/error state when implementation evidence exists

## Placement rules

1. Small static configuration belongs under the relevant `config/` directory.
2. Larger stateful or data-heavy logic belongs under the relevant `memory-management/` directory.
3. Operator-facing claims must stay explicit when no live transport, firmware, or persistent device storage exists.
4. New assets must align with MEL's existing truth model: no fake transport success, no placeholder production code, and no unsupported runtime claims.

## Integration guidance

### Central node expectations

- Standardize topic naming and data formats with the existing MEL transport and persistence model.
- Keep durable state and aggregation concerns separate from small operator config.
- Preserve explicit degraded reporting when upstream extension nodes or transports are absent.

### Extension node expectations

- Prefer minimal, OTA-friendly config files.
- Keep any local buffering bounded and explicit.
- Surface retry, disconnection, and low-power degraded states in machine-visible ways so the central node does not infer health it cannot prove.

## Current implementation boundary

The current shipped runtime remains:

- executables in `cmd/`
- runtime logic in `internal/`
- operator config examples in `configs/`
- SQLite schema in `migrations/`

The `topologies/` tree is a scaffold and documentation boundary, not proof of implemented new runtimes.
