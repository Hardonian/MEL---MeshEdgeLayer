# MEL Operator Vibe System (Canonical)

Status: canonical implementation contract for UI identity, language, and review.
Applies to: frontend, docs, onboarding copy, screenshots, release notes.

## 1) Product identity

MEL is an **operator console for incident intelligence and trusted control**.

The product must read as:
- dark-first, field-ready, dense but readable;
- restrained, infrastructural, and evidence-led;
- explicit about degraded, partial, stale, and unknown states;
- technically calm (never theatrical).

## 2) Anti-goals (hard bans)

Do not ship:
- visual theatre (glow noise, decorative blur, fake terminal cosplay, cyberpunk imitation);
- copy that implies unsupported capability, fake live truth, or hidden certainty;
- “hacker” framing that suggests illegal/intrusive behavior;
- naming drift that fractures orientation (dashboard/console/workbench roulette).

## 3) Legal and ethical boundaries

- Never imply MEL performs RF routing/propagation execution unless implemented and verified.
- Never represent assistive inference as canonical truth.
- Never use language that implies covert, unauthorized, or unsafe operations.
- Privacy/security wording must describe implemented controls only.

## 4) Accessibility requirements

Required on every UI change:
- WCAG-compliant text and non-text contrast;
- unmistakable keyboard focus indication;
- complete `prefers-reduced-motion` fallback;
- readable density (no body monospace, no low-contrast decorative labels);
- destructive actions must state impact and scope.

## 5) Terminology canon

Use these terms consistently:
- **operator console**: primary top-level surface (`/`).
- **incident workbench**: incident queue + triage surface (`/incidents`).
- **console**: acceptable shorthand for operator console.

Avoid as generic synonyms:
- dashboard
- command surface
- workspace (unless it means temporary browser-local focus context)

## 6) Approved and banned metaphors

Approved: operator console, incident workbench, signal lane, evidence trail, shift handoff, route view.

Banned: breach mode, stealth mode, exploit path, blackhat, zero-day vibe, “war room roleplay”.

## 7) Tone rules

Voice must be:
- calm, competent, concise;
- minimally expressive, never juvenile;
- explicit about uncertainty and boundaries.

Writing rules:
- prefer verb-first action labels;
- state observed vs inferred vs unknown directly;
- never use blamey or hype language;
- no “Oops!”.

## 8) State wording contract

Preferred state phrases:
- Live: “Recent persisted ingest evidence is present.”
- Stale: “Evidence is older than freshness target.”
- Partial/Degraded: “Connected with gaps; review failure markers.”
- Unknown: “No current evidence; state is unknown.”
- Unsupported: “Not implemented in this MEL runtime.”

## 9) Naming rules (pages/components)

Page naming:
- use operator nouns (`Diagnostics`, `Recommendations`, `Topology`, `Incidents`);
- avoid decorative metaphors.

Component naming:
- include truth/intent where useful (`OperatorTruthRibbon`, `StaleDataBanner`);
- avoid generic cosmetic names for semantic primitives.

## 10) Strong vs weak copy examples

Strong:
- “No recent ingest evidence. Runtime state is unknown until new records arrive.”
- “Submission recorded. Approval is still required before dispatch.”

Weak:
- “Everything looks healthy.”
- “AI confirmed the cause.”
- “Stealth mode active.”

## 11) Measurement and review guardrails

Identity work is complete only when:
- terminology is coherent across nav, headings, and docs;
- token and typography choices are implemented in code, not prose-only;
- core pages adopt the same shell/panel/copy conventions;
- verification evidence is attached (lint/tests/build + manual a11y checks);
- residual gaps are explicitly listed.

Review checklist for UI PRs:
1. Claim boundaries still match implementation.
2. Degraded/partial/unknown wording remains explicit.
3. No decorative glow/blur/theatre regressions.
4. Keyboard focus and reduced-motion behavior verified.
5. Terminology follows canon (operator console vs incident workbench).
