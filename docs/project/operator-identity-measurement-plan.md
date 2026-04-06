# Operator Identity Measurement Plan

## Goal

Measure whether MEL's operator-console identity improves clarity and trust while preserving accessibility, verification discipline, and claim integrity.

## Scope

Applies to UI/copy/terminology/system-shell changes that impact operator orientation across:
- operator console (`/`),
- incident workbench (`/incidents`),
- diagnostics/recommendations/settings/topology/help surfaces,
- contributor-facing docs that guide future UI text.

## Evidence model

Use existing local artifacts (tests, builds, support bundles, route behavior, issue reports). Do not add hidden telemetry.

## Metrics and guardrails

### 1) Terminology coherence

Target:
- no user-facing drift between dashboard/command-surface/workbench/console labels.

Checks:
- repo grep for banned generic terms in user-facing docs/UI;
- nav label and page-title consistency review.

### 2) Operator orientation

Target:
- faster shift-start orientation and fewer navigation misroutes.

Proxies:
- command palette route usage;
- incident return-path continuity;
- reduced support reports with “where do I do X?” ambiguity.

### 3) Truth comprehension

Target:
- fewer misreads of stale/partial/unknown as healthy/live.

Proxies:
- support bundle narratives;
- incident notes referencing stale/degraded misunderstandings;
- reduced confusion in triage comments.

### 4) Accessibility safety

Required gates per change:
- focus ring visibility on modified controls;
- reduced-motion behavior verified on changed surfaces;
- text/non-text contrast review;
- keyboard flow sanity for palette/help/navigation.

### 5) Verification integrity

Target:
- no identity-only PR merges without implementation and checks.

Required evidence:
- lint/test/build outcomes;
- targeted frontend checks for touched surfaces;
- explicit residual gap register.

## PR review checklist additions

For identity-related PRs include:
1. terminology decisions (and why);
2. files updated across code + docs;
3. verification commands and outcomes;
4. accessibility notes (focus, contrast, reduced motion);
5. residual risks/follow-up seams.

## Failure triggers

Treat as regression if any occurs:
- user-facing copy reintroduces “dashboard”/“command surface” as generic labels;
- decorative motion/effects return without functional purpose;
- wording hides degraded/partial/unknown states;
- claims exceed persisted evidence.

## Follow-up instrumentation seam (optional)

If MEL adds opt-in analytics later, add explicit consented measures for:
- first-session navigation success,
- command palette find-success rate,
- time-to-incident-workbench from home console,
with privacy-preserving defaults and clear policy controls.
