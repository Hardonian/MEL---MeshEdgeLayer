# PR / Change Classification Model

Classify every PR as **Maintenance**, **Leverage**, or **Moat**.

## 1) Maintenance
Definition: correctness, stability, drift reduction, refactors with no new durable advantage.

Required in PR:
- Problem statement and root cause.
- Verification evidence.
- Residual risk/caveat note.

## 2) Leverage
Definition: reusable workflow acceleration, operator clarity, or safer execution patterns.

Required in PR:
- Maintenance requirements plus:
- Which operator workflow got faster/safer and how.
- Which checklist/runbook/audit module was updated.

## 3) Moat
Definition: compounding advantage in operational memory, decision quality, evidence trust, or workflow lock-in.

Required in PR:
- Leverage requirements plus:
- Explicit compounding mechanism (what accumulates over time).
- Why a UI clone without MEL history/workflows cannot match outcome quality.
- KPI/evidence signal to validate moat effect.

## Classification guardrails
- Do not label work “Moat” if it is mostly UI polish.
- Do not label work “Leverage” without measurable workflow effect.
- Default to Maintenance when uncertain.
