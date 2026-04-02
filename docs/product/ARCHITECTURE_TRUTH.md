# Architecture Truth (Operator-Facing)

This page maps product promises to implementation truth surfaces.

## Truth-producing layers

- Ingest workers and persisted packet records.
- Status/doctor/readiness diagnostics.
- Control action lifecycle records.
- Evidence exports (proofpack/support bundle).

## Truth-consuming layers

- Web UI pages and CLI output.
- Runbooks and operational workflows.
- Recommendation/planning surfaces.

## Required alignment rules

- Presentation must never exceed backend evidence certainty.
- Degraded states are machine-visible first, then human-readable.
- Advisory intelligence cannot override deterministic safety gates.

For deeper technical architecture see:
- [Architecture overview](../architecture/overview.md)
- [Operational boundaries](../architecture/OPERATIONAL_BOUNDARIES.md)
- [Control plane trust model](../architecture/CONTROL_PLANE_TRUST_MODEL.md)
