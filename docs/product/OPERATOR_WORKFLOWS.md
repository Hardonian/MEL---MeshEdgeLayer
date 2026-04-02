# Operator Workflows

## Shift-start workflow

1. Run readiness checks (`mel doctor`, status snapshot).
2. Confirm transport ingest posture (live/stale/degraded).
3. Review pending approvals and active freezes/maintenance windows.
4. Open incident queue and prioritize by evidence freshness/severity.

## First-incident workflow

1. Open incident detail.
2. Separate observed evidence from inferred context.
3. Evaluate action candidates and approval requirements.
4. Record notes and export proofpack for traceability.

## Support-escalation workflow

1. Export support bundle.
2. Attach timeline and relevant action IDs.
3. Include known limitations references to prevent false bug reports.
4. Track outcome and update runbook memory.
