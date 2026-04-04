# Launch screenshot checklist (implementation-grounded)

Use these frames to show **honest operator truth**, not generic monitoring. Capture after `make build` and `./bin/mel serve` with real or demo data (`./bin/mel demo scenarios`, `./scripts/demo-evidence.sh`).

| # | Surface | What must be visible | Honesty bar |
|---|---------|----------------------|-------------|
| 1 | Command surface (`/`) | Shift baseline, attention rows, stale/poll context | No implication of live RF proof |
| 2 | Incidents list (`/incidents`) | Open queue, evidence/sparse/degraded cues if present | Priorities from stored intelligence, not invented health |
| 3 | Incident detail (`/incidents/:id`) | Decision pack / replay / proof export if applicable | Separates observed vs inferred vs workflow state |
| 4 | Topology (`/topology`) | Graph + observed vs inferred callout | No map-as-coverage claim |
| 5 | Control actions (`/control-actions`) | Lifecycle filters, approval vs execution wording | Governance boundaries explicit |
| 6 | Settings (`/settings#effective-config`) | Effective config / platform posture rows | Claims bounded to runtime API |

**Avoid:** empty states presented as “all green,” cropped banners that hide degraded semantics, or captions that promise mesh routing or coverage MEL does not evidence.

See also: [`docs/community/claims-vs-reality.md`](../community/claims-vs-reality.md), [`docs/ops/transport-matrix.md`](transport-matrix.md).
