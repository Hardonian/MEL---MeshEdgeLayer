# Operator Identity Measurement Plan

## Goal

Validate whether the workbench-oriented identity improves operator clarity and trust without harming usability or accessibility.

## What to measure (current architecture compatible)

Use existing backend/frontend logs, test suites, and support-bundle evidence; do not add invasive analytics by default.

1. **Navigation clarity**
   - proxy: frequency of route jumps via command palette and return-path usage in incidents.
   - regression signal: increased back-and-forth page churn during incident triage.

2. **Task completion confidence**
   - proxy: successful progression through control lifecycle states (submission -> approval -> execution -> audit) without abandoned actions.
   - regression signal: more pending approvals aging without resolution.

3. **Truth comprehension**
   - proxy: reduced operator misreads linked to stale/partial/unknown semantics in support incidents.
   - regression signal: repeated support issues caused by over-claim interpretation.

4. **Accessibility guardrails**
   - mandatory: focus visibility, keyboard traversal, reduced-motion behavior, contrast checks on changed surfaces.

5. **Performance guardrails**
   - build-time and bundle sanity from existing frontend build/test targets.

## Experiment seams (future)

If MEL later adds opt-in product analytics, test:

- “Workbench” vs “Command surface” primary nav label comprehension;
- terse vs expanded truth-strip copy for first-time operators;
- compact vs comfortable density defaults for incident triage surfaces.

## Release checklist additions

For vibe/copy/system changes include:

- screenshot evidence of changed core surfaces;
- test/build/lint outputs;
- explicit list of wording changes that narrow claims;
- residual risk and follow-up seams.
