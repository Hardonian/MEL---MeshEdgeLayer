# Operator-truth principles

MEL is an **incident-intelligence and trusted-control** system, not a decorative dashboard. These principles are the product contract; they align with [AGENTS.md](../AGENTS.md) and [repo-os/terminology.md](repo-os/terminology.md).

1. **Evidence before narrative** — Persisted ingest records, state transitions, and audit events outrank commentary and UI copy.
2. **Explicit degraded states** — Stale, partial, historical, imported/offline, and unknown must stay visible; they must not be collapsed into “healthy.”
3. **No transport theatre** — Supported vs unsupported paths are labeled honestly (see root README matrix). No implied BLE/HTTP ingest or MEL-as-RF-router fiction.
4. **Trusted control is separable** — Submission, approval, dispatch, execution result, and audit are distinct; blur is a defect.
5. **Inference is assistive** — Any LLM or heuristic layer is non-canonical when it disagrees with deterministic evidence.
6. **Local-first viability** — Base usefulness does not require a mandatory cloud control plane or hidden telemetry defaults.

**Deeper reads:** [product/HONESTY_AND_BOUNDARIES.md](product/HONESTY_AND_BOUNDARIES.md), [ops/CONTROL_PLANE_TRUST.md](ops/CONTROL_PLANE_TRUST.md), [architecture/CONTROL_PLANE_TRUST_MODEL.md](architecture/CONTROL_PLANE_TRUST_MODEL.md).
