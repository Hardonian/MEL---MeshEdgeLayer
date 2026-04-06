# MEL Writing Style Guide (UI + Docs)

This guide standardizes operator-facing copy across UI, docs, and release notes.

## Voice baseline

- Calm.
- Competent.
- Direct.
- Evidence-bounded.

Avoid slang, jokes, or theatrical “hacker” tone.

## Preferred terminology

Use these terms consistently:

- workbench (primary operations surface)
- transport, ingest, packet evidence
- incident, evidence strength, degraded, partial, unknown
- approval, dispatch, execution, audit
- topology (observed context, not RF proof)

## Terms to avoid

- dashboard (unless it references a legacy API identifier)
- stealth, exploit, breach, bypass, blackhat
- “all good”, “fully healthy”, “guaranteed” without proof context
- “Oops!” and blamey phrasing

## State wording patterns

- Healthy: “Connected and ingesting recent records.”
- Degraded: “Connected with gaps; review failure markers.”
- Unknown: “No current evidence; state is unknown.”
- Unsupported: “Not implemented in this MEL runtime.”

## Error message pattern

Use:

1. what failed;
2. impact;
3. actionable next step.

Example:

- “Diagnostics fetch failed. Current findings may be stale. Retry now or check transport connectivity.”

## Action copy pattern

- Primary actions: verb-first (`Refresh evidence`, `Review incident`, `Export proofpack`).
- Destructive actions: explicit consequence (`Delete local records`, `Revoke pending approval`).

## Severity wording

- info: informational context
- warning: degraded/partial condition
- critical: stop-the-line operational risk

Do not use dramatic synonyms (catastrophic, meltdown, apocalypse) in product UI.

## Trust boundary wording reminders

- Submission is not approval.
- Approval is not execution.
- History informs context; it does not prove live runtime state.
