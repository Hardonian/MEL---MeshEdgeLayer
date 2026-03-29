# Operator Truth Audit Checklist

Use for PRs touching UI, API, docs, health, incident wording, controls, telemetry semantics.

## Claim integrity
- [ ] Every operator-facing claim is backed by code-path evidence.
- [ ] No certainty language (“healthy”, “fixed”, “delivered”, “connected”) without proof conditions.
- [ ] Unknown is rendered as unknown (not healthy by omission).

## State semantics
- [ ] Live vs stale vs historical vs imported/offline are explicitly differentiated.
- [ ] Partial ingest/degraded state is machine-visible (status field/state enum) and human-visible.
- [ ] Dead letters or parse failures are not hidden behind success summaries.

## UI/API parity
- [ ] UI wording does not outrun API contract semantics.
- [ ] API docs and examples match current response fields.
- [ ] No frontend-only heuristics acting as canonical truth when typed backend contract exists/should exist.

## Causality integrity
- [ ] Incident cause statements are evidence-linked, not speculative.
- [ ] Recommendation/remediation language communicates confidence bounds.

## Release honesty
- [ ] Unsupported capabilities are labeled unsupported.
- [ ] Known limitations are updated where user perception changed.
