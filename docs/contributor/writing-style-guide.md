# MEL writing style guide (docs + UI)

Use this guide for README, docs, UI copy, issue templates, and release notes.

Goal: MEL should sound like one coherent project — precise, useful, and allergic to theatre.

## Voice baseline

Write like a calm operator who has seen real outages:
- clear
- compact
- technically literate
- slightly dry
- never hype-heavy

Snark is allowed only when it improves clarity.

## House rules

1. **State evidence, not vibes.**
2. **Name uncertainty explicitly** (`unknown`, `partial`, `degraded`, `stale`).
3. **Do not imply unsupported capability.**
4. **Prefer short words over abstract jargon.**
5. **Lead with what the operator should do next.**

## Canonical terms

Prefer these terms consistently:
- operator console
- ingest evidence
- incident
- proofpack
- action lifecycle: submission → approval → dispatch → execution → audit
- support posture: supported / unsupported / degraded / unknown

Avoid fuzzy alternatives:
- “fully healthy” (unless backed by precise criteria)
- “real-time certainty”
- “AI-powered insight” without boundaries
- decorative military/cyber naming

## Tone examples

Good:
- “Recent persisted ingest evidence is present.”
- “State is degraded: transport connected with gaps.”
- “Unsupported in current MEL runtime.”

Bad:
- “Everything looks great.”
- “Autonomous mesh intelligence complete.”
- “Enterprise-grade battlefield dashboard.”

## Error-message pattern

For UI/API/CLI errors, include:
1. what failed,
2. operator impact,
3. next action.

Example:
`Diagnostics fetch failed. Findings may be stale. Retry or verify transport connectivity.`

## README/docs structure defaults

- Start with what MEL is and is not.
- Add support/limitation boundaries early.
- Keep quickstart short and runnable.
- Put deeper theory after actionable guidance.
- Link to canonical pages instead of duplicating large matrices.

## Copy review checklist

Before merge, confirm:
- terminology matches canon;
- claims are bounded by implementation;
- degraded/unknown semantics are preserved;
- wording is shorter and clearer than before;
- personality does not undermine trust.
