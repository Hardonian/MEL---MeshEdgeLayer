# Skill: Operator Truth and Transport Boundaries

Use when touching ingest, status/health semantics, API/UI truth wording, support matrix docs, or transport diagnostics.

## Checklist
- [ ] Claims are bounded by persisted + timestamped evidence.
- [ ] Live/stale/historical/imported/partial/degraded/unknown are uncollapsed.
- [ ] Unsupported surfaces are explicitly labeled unsupported.
- [ ] No MEL RF routing/propagation execution implication.
- [ ] Degraded/partial gaps are machine-visible in API/CLI/UI.
- [ ] Dead-letter/disconnect evidence remains visible.
- [ ] Docs/README/runbooks match implementation.

## Evidence to attach
- Tests for state transitions + degraded paths.
- Example payload or CLI output demonstrating honest state labels.
- Updated support matrix references where wording changed.
