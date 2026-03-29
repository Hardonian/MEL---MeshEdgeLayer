# MEL Repo Operating System

This directory is MEL’s execution system for truthful implementation and release discipline.

## Scope
Use these artifacts for every change that touches ingest, health, incidents, controls, UI truth rendering, API semantics, security boundaries, or release claims.

## Canonical entry points
- `AGENTS.md` (root): default operating contract for contributors/agents.
- `verification-matrix.md`: required checks by change type.
- `release-readiness.md`: merge/release reality gate.
- `terminology.md`: canonical terminology + language rules for operator truth.

## Audit modules
- `operator-truth-audit.md`
- `transport-truth-audit.md`
- `trusted-control-governance.md`
- `security-trust-boundary-audit.md`
- `moat-evaluation.md`

## Execution modules
- `change-classification.md`
- `incident-learning-loop.md`
- `prompt-headers.md`
- `architecture-trust-boundaries.md`
- `../architecture/communications-hub-blueprint.md`
- `../privacy/platform-policy.md`

## How to use (minimal workflow)
1. Classify the work (`change-classification.md`).
2. Run applicable audits (truth/transport/control/security/moat).
3. Execute verification obligations (`verification-matrix.md`).
4. Run reality gate (`release-readiness.md`).
5. Include evidence + residual risk in PR.
