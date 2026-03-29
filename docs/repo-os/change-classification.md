# PR / Change Classification Model (MEL)

Classify every PR as **Maintenance**, **Leverage**, or **Moat**.

## 1) Maintenance
Definition: correctness, stability, drift reduction, bounded refactors with no new durable advantage.

Required in PR:
- Problem statement and root cause.
- Verification evidence.
- Residual risk/caveat note.

## 2) Leverage
Definition: reusable workflow acceleration, clearer operator truth, safer deterministic execution, or lower operational toil with measurable trust gain.

Required in PR:
- Maintenance requirements plus:
- Which operator workflow got faster/safer and how.
- Which checklist/runbook/audit module was updated.
- Why this improves mixed-network truth or control reliability.

## 3) Moat
Definition: compounding advantage in operational memory, decision quality, evidence trust, action learning, or workflow lock-in that cannot be reproduced by UI cloning alone.

Required in PR:
- Leverage requirements plus:
- Explicit compounding mechanism (what accumulates over time).
- Why a UI clone without MEL history/workflows cannot match outcome quality.
- KPI/evidence signal to validate moat effect.
- Effect on local-first/private/self-hosted resilience and affordability.

## Classification guardrails
- Do not label work “Moat” if it is mostly UI polish.
- Do not label work “Leverage” without measurable workflow or trust effect.
- Default to Maintenance when uncertain.
