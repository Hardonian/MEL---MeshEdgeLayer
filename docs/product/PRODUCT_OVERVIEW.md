# MEL Product Overview

MEL (MeshEdgeLayer) is an incident-intelligence and trusted-control operating system for mixed local network operations.

It is designed for operators who need to answer three questions under degraded conditions:

1. **What happened?** (typed evidence + timeline)
2. **What is true now?** (live/stale/partial/degraded/unknown semantics)
3. **What can we safely do next?** (approval-bounded controls + audited outcomes)

## Primary user profiles

- **Single-site community operators** running local mesh deployments.
- **Incident responders** handling unstable ingest, transport drift, and signal ambiguity.
- **Systems teams** needing exportable proofpacks and deterministic audit records.

## Current capability envelope (April 2, 2026)

- Supported ingest: serial, TCP, MQTT.
- Unsupported ingest: BLE, HTTP.
- MEL is **not** a mesh routing stack and does not execute RF routing/propagation.
- Control actions are lifecycle-governed (submission/approval/dispatch/execution/audit).
- Some actions remain advisory-only when no actuator exists.

## Product wedge

MEL’s wedge is **truth-preserving local operations**:

- Evidence-first, not dashboard cosmetics.
- Explicit degraded/unknown states, not implied certainty.
- Operator-trust primitives (proofpacks, action audit, support bundles).

## Product boundaries

- No fake autonomy: assistive intelligence is non-canonical.
- No fake transport support claims.
- No collapse of historical/imported context into live truth.

See also:
- [Positioning](./POSITIONING.md)
- [Capability Matrix](./CAPABILITY_MATRIX.md)
- [Feature Maturity](./FEATURE_MATURITY.md)
- [Honesty and Boundaries](./HONESTY_AND_BOUNDARIES.md)
