# MEL Repo Operating System

This directory is MEL’s durable execution spine for truthful implementation, operator trust, and release discipline.

## Canonical priority order
1. `AGENTS.md` (root) — primary contributor/agent contract.
2. `docs/repo-os/model-spec.md` — repo-local execution model spec for imperfect/short prompts.
3. Verification + release gates (`verification-matrix.md`, `release-readiness.md`).
4. Supporting audits/checklists/modules in this directory.

If guidance conflicts, follow the higher-priority artifact and tighten weaker wording downstream.

## Scope
Apply these artifacts for every change touching ingest, health, incidents, controls, privacy, local inference, evidence/proofpacks, API semantics, UI truth rendering, security boundaries, or release claims.

## Canonical entry points
- `AGENTS.md` (root): MEL identity, truth hierarchy, precision layering, privacy/build-vs-borrow policy.
- `model-spec.md`: execution behavior contract for future agents/model prompts.
- `verification-matrix.md`: required checks by change type.
- `release-readiness.md`: merge/release reality gate.
- `terminology.md`: operator-first language policy.

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
- `skills/README.md`
- `../architecture/communications-hub-blueprint.md`
- `../privacy/platform-policy.md`

## Minimal workflow
1. Classify work (`change-classification.md`).
2. Apply relevant skills/checklists (`skills/README.md`).
3. Run truth/control/security/moat audits.
4. Execute verification obligations (`verification-matrix.md`).
5. Run release reality gate (`release-readiness.md`).
6. Include evidence + residual risk + caveats in PR.
