# Issue routing label taxonomy

Use these labels to keep post-launch triage operational and doctrine-aligned.

## Required baseline

- `needs:triage` — new issue has not been classified.
- `kind:bug` / `kind:feature` / `kind:docs` / `kind:setup` / `kind:truth-semantics` / `kind:contributor-experience` — one primary class.

## Doctrine-sensitive classes

- `area:degraded-state` — stale/partial/degraded/unknown signaling risk.
- `area:control-safety` — approval/dispatch/execution/audit boundary risk.
- `area:topology-or-status` — topology/path/delivery overclaim or inference confusion.
- `area:environment` — install/toolchain/verification environment friction.

## Resolution posture

- `status:needs-repro` — cannot act until deterministic repro exists.
- `status:ready-for-fix` — scoped and reproducible.
- `status:docs-or-claim-narrowing` — implementation unchanged; claim must be bounded.
- `status:blocked-external` — waiting on environment/tooling outside repo control.

## Triage loop

1. Apply one `kind:*` and keep `needs:triage` until first maintainer response.
2. Add one `area:*` label for trust/safety sensitive issues.
3. Remove `needs:triage` only after assigning next action (`status:*` or assignee).
4. For truth/semantics reports, link exact API fields + UI copy before closing.
