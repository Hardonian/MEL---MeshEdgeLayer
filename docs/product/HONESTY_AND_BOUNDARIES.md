# Honesty and Boundaries

## Product claim rules

- If evidence is missing, MEL must show unknown/degraded/partial state.
- If transport is unsupported, MEL must label unsupported (not “coming soon” as current behavior).
- If an action is advisory-only, copy must not imply execution.

## Current hard boundaries

- BLE ingest: unsupported.
- HTTP ingest: unsupported.
- MEL as RF routing/propagation executor: not implemented.
- Restore operations: validation-only in current release posture.

## Evidence hierarchy

1. Persisted typed runtime evidence.
2. Deterministic calculators/heuristics.
3. Assistive inference (non-canonical).
4. Narrative/explanatory text.

When these conflict, higher layers win.

## Operator-facing wording standard

Use: **observed / inferred / estimated / unknown**.

Avoid: **guaranteed / fixed / delivered / healthy** without direct proof conditions.
