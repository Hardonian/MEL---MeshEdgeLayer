# MEL Writing Style Guide (UI + Docs)

Use this guide for operator-facing text in UI, docs, onboarding, and release notes.

## 1) Product voice

Baseline voice:
- calm;
- competent;
- direct;
- evidence-bounded.

Never write with theatrical, juvenile, or hype-heavy tone.

## 2) Canonical terminology

Use:
- **operator console** for `/` and generic primary web surface references;
- **incident workbench** for `/incidents` queue and triage language;
- transport, ingest, evidence, incident, approval, dispatch, execution, audit;
- degraded, partial, stale, unknown, unsupported.

Avoid:
- dashboard (except legacy code/API identifiers);
- command surface as a user-facing generic label;
- vague claims like “fully healthy” without bounded evidence.

## 3) Tone and wording rules

- Prefer verb-first action labels: `Refresh evidence`, `Review incident`, `Export proofpack`.
- Keep sentence shape simple: what happened, impact, next action.
- Avoid blame language: “you forgot”, “bad config”, “broken setup”.
- Avoid filler interjections: “Oops!”, “Yikes!”, “Uh oh!”.

## 4) State wording templates

Use these phrases or close equivalents:
- Live: “Recent persisted ingest evidence is present.”
- Stale: “Evidence is stale; verify current transport state.”
- Degraded/Partial: “Connected with gaps; review failure markers.”
- Unknown: “No current evidence; state is unknown.”
- Unsupported: “Not implemented in this MEL runtime.”

## 5) Error copy pattern

Every error line should include:
1. failure;
2. operator impact;
3. next action.

Example:
- “Diagnostics fetch failed. Findings may be stale. Retry or verify transport connectivity.”

## 6) Safety, trust, and legal boundaries

Never imply:
- unauthorized/covert behavior;
- unsupported transport/runtime capabilities;
- submission equals approval or execution;
- inferred output equals canonical truth.

## 7) Page and component naming rules

- Page titles should be short operator nouns: `Diagnostics`, `Recommendations`, `Settings`.
- Avoid decorative naming (“cockpit mode”, “mission center”, “cyber board”).
- Component names should encode function/truth intent (`OperatorTruthRibbon`, `SupportBundleExport`).

## 8) Strong vs weak examples

Strong:
- “No active recommendations from current evidence.”
- “Approval pending. Dispatch has not started.”

Weak:
- “Everything is great.”
- “Stealth remediation complete.”

## 9) Review checklist for copy changes

Before merge, confirm:
- terminology matches canon;
- wording preserves degraded/partial/unknown semantics;
- claims do not exceed implementation evidence;
- destructive actions are explicit about consequences.
