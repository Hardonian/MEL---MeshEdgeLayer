# Reusable Prompt / Execution Headers (MEL)

Use these as short headers for future agent work to avoid repeating context.

## 1) Implementation Pass
"Implement the smallest safe change that improves operator truth. No fake transport/control claims. Use typed contracts, explicit degraded states, and evidence-backed wording."

## 2) Reality Pass
"Audit UI/API/docs for overclaiming. Distinguish live vs stale vs historical vs imported vs partial. Downgrade certainty where evidence is missing."

## 3) Release Hardening Pass
"Run release-readiness gate from `docs/repo-os/release-readiness.md`; block readiness if claims exceed implementation or degraded states are implicit."

## 4) UI Truth Pass
"Ensure frontend semantics never exceed backend evidence. Unknown must remain unknown; stale/partial states must be explicit and visible."

## 5) Control Governance Pass
"For any action/approval/execution flow: enforce explicit lifecycle states, approval separation, auditable outcomes, and no hidden bypass paths."

## 6) Incident-Intelligence Pass
"Convert incident/dead-letter/action-failure learnings into tests, runbook updates, rule/schema improvements, and better drilldown evidence surfaces."

## 7) Moat Pressure-Test Header
"Classify change as Maintenance/Leverage/Moat and explain what compounding advantage (data, workflow, decisioning, evidence trust) this creates beyond UI cloneability."
