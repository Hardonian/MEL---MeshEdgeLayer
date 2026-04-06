# MEL Operator Vibe System (Canonical)

Status: canonical implementation guide for frontend identity, copy, and review.
Applies to: product UI, docs, onboarding text, screenshots, and release messaging.

## 1) Product identity and intent

MEL is a **trusted-control and incident-intelligence workbench**. The visual and language system must reinforce:

- deterministic operator truth;
- explicit degraded / partial / unknown state handling;
- evidence-first decision support;
- privacy-first local-first operations.

This vibe is **field-ready technical clarity**, not performance art.

## 2) Anti-goals (hard bans)

Do not ship:

- fake certainty, fake live status, or implied capabilities not implemented;
- “hacker theater” motifs (scanline overlays, glitch effects, random neon glow noise);
- language that implies trespass, intrusion, evasion, or unsafe behavior;
- decorative labels that hide operational meaning.

## 3) Visual system contract

### 3.1 Surface and color behavior

- Foundation: near-black / charcoal surfaces by default in dark mode.
- Accent colors are **signal semantics**, never paragraph color.
- Status colors are stable across pages:
  - live/healthy (`signal-live`),
  - observed (`signal-observed`),
  - degraded (`signal-degraded`),
  - critical (`signal-critical`),
  - inferred (`signal-inferred`, explicitly non-canonical),
  - stale / unsupported / frozen where relevant.
- Focus rings are always visible and high-contrast.

### 3.2 Typography

- Sans for prose and high-volume reading surfaces.
- Mono for IDs, timestamps, queue metadata, route or packet details.
- Never use monospace for long-form copy.

### 3.3 Panel and chrome

- Panel edges indicate containment and state, not glow effects.
- Borders/separators must carry hierarchy before shadows do.
- Radius remains minimal and instrument-like.

### 3.4 Motion

- Motion communicates causality (load, expand, enter, dismiss), not ambience.
- Respect `prefers-reduced-motion` globally.
- No core state can be conveyed only by animation.

## 4) Voice and copy contract

Voice: calm, competent, concise, slightly clandestine, never theatrical.

Rules:

- verbs first for actions where useful;
- avoid hype and blame;
- prefer explicit operational nouns (transport, ingest, evidence, approval, execution, audit);
- use “workbench” for the main operating surface;
- always separate observed vs inferred vs unknown.

## 5) Approved and banned metaphors

Approved: workbench, field kit, signal lane, route view, evidence trail, shift handoff.

Banned: breach, exploit, bypass, stealth mode, ghost mode, blackhat, zero-day theater language.

## 6) Accessibility and legal safety gates

Any UI/copy change must keep:

- WCAG-compliant text/non-text contrast;
- keyboard focus discoverability and logical order;
- explicit reduced-motion fallback;
- plain language for destructive actions and privacy impact;
- no illegal/unsafe implication in text or imagery.

## 7) Verification gates for vibe-related changes

For changes to style/copy/navigation surfaces, include:

1. lint + tests + build evidence;
2. screenshots or manual visual notes for changed surfaces;
3. contrast/focus/reduced-motion checks;
4. explicit statement of claims narrowed (if capability unchanged).

## 8) Channel adaptation rules

- In-app copy: short, imperative, state-rich.
- Docs: field-manual style, scannable sections, grounded in implementation.
- Marketing/readme: trust, resilience, and evidence boundaries first; no claim inflation.

## 9) Good vs bad examples

Good:

- “Topology shows observed relationships, not RF propagation proof.”
- “Dispatch submitted. Approval still required before execution.”
- “No recent ingest evidence. State is unknown until new records arrive.”

Bad:

- “Network is healthy.” (without bounded evidence context)
- “AI confirmed root cause.”
- “Stealth mode for covert operations.”
