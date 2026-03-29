# Trusted Control Governance Checklist

Use for actions, approvals, queueing, execution, break-glass, incidents linked to controls.

## Lifecycle truth
- [ ] Action lifecycle states are explicit (e.g., submitted/queued/approved/rejected/executing/succeeded/failed/cancelled).
- [ ] UI/API never collapse submission into execution.
- [ ] Final outcome state includes proof of success or failure.

## Approval and separation
- [ ] Approval requirements are enforced by code, not docs alone.
- [ ] Approval actor and execution actor attribution are auditable.
- [ ] Any bypass/break-glass path is explicit, logged, and narrowly scoped.

## Incident linkage
- [ ] Incident-linked actions preserve referential integrity (incident ID, action ID, timestamps).
- [ ] Causal linkage is marked as asserted evidence, not assumed certainty.

## Blast-radius honesty
- [ ] Scope/targeting details are visible before approval.
- [ ] Risk and rollback constraints are surfaced to operator.

## Security and auditability
- [ ] No hidden escalation paths.
- [ ] Unauthorized actions fail closed with clear semantics.
- [ ] Audit events are persisted for submission, approval, execution attempt, and outcome.
