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

## Label bootstrap / sync (fresh repo or fork)

MEL issue templates reference labels, but GitHub does **not** auto-provision those labels from template frontmatter. In a fresh repo/fork:

1. Create baseline labels first (`needs:triage` + one `kind:*` + doctrine-sensitive `area:*` + `status:*`).
2. Keep optional/legacy labels (`enhancement`) only if your maintainer workflow still uses them.
3. If templates reference a missing label, either create it immediately or edit the template to an existing label set before opening intake.

CLI example with GitHub CLI (`gh`) from repo root:

```bash
gh label create 'needs:triage' --color d73a4a --description 'new issue has not been classified' --force
gh label create 'kind:bug' --color b60205 --description 'deterministic defect in implementation or verification' --force
gh label create 'kind:docs' --color 0e8a16 --description 'documentation drift or claim narrowing' --force
gh label create 'kind:setup' --color 5319e7 --description 'environment/setup friction' --force
gh label create 'kind:truth-semantics' --color fbca04 --description 'truth boundary, degraded, or overclaim semantics' --force
gh label create 'kind:contributor-experience' --color 1d76db --description 'contributor workflow/tooling friction' --force
gh label create 'area:degraded-state' --color d4c5f9 --description 'stale/partial/degraded/unknown signaling risk' --force
gh label create 'area:control-safety' --color c2e0c6 --description 'control lifecycle/approval/audit boundary risk' --force
gh label create 'area:topology-or-status' --color f9d0c4 --description 'topology/path/status overclaim risk' --force
gh label create 'area:environment' --color fef2c0 --description 'toolchain/runtime environment mismatch' --force
gh label create 'status:needs-repro' --color e99695 --description 'waiting for deterministic reproduction evidence' --force
gh label create 'status:ready-for-fix' --color 0e8a16 --description 'scoped and reproducible; ready to implement' --force
gh label create 'status:docs-or-claim-narrowing' --color c2e0c6 --description 'implementation unchanged; wording/claim must tighten' --force
gh label create 'status:blocked-external' --color bfdadc --description 'blocked on dependency/tooling outside repo control' --force
```
