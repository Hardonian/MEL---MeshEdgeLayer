# Feature Maturity Classification

This page is the public maturity contract for MEL surfaces.

## Labels

- **GA**: reliable for production use within stated boundaries.
- **Beta**: functional but still stabilizing; expect API/UX changes.
- **Experimental**: for evaluation; no compatibility guarantees.
- **Unsupported**: intentionally not implemented.
- **Roadmap**: planned/considered; not currently shipped.

## Current classification

### GA
- Serial/TCP/MQTT ingest.
- Incident timeline and triage surfaces.
- Action approval and audit flow.
- Support bundles and proofpack export.
- Diagnostics (`mel doctor`) and status APIs.

### Beta
- Guarded automation policy tuning in heterogeneous deployments.
- Topology/planning advisory surfaces where evidence can be sparse.

### Experimental
- Local assistive inference routing/runtime combinations.
- Extended recommendation strategies requiring more field evidence.

### Unsupported
- BLE ingest.
- HTTP ingest.
- MEL-executed RF routing/transmit operations.

### Roadmap (not current claims)
- Restore paths beyond dry-run validation.
- Expanded actuator-backed remediation coverage.
- Richer cross-incident recommendation memory with explicit uncertainty scoring.
