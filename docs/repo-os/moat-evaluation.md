# Moat Evaluation Checklist (MEL-Specific)

Use for roadmap items, major features, and architecture refactors.

## Compounding data advantage
- [ ] Captures new operational history that accumulates (incident signatures, topology drift, dead-letter patterns, remediation outcomes).
- [ ] Improves evidence fidelity/provenance, not just display polish.

## Workflow lock-in
- [ ] Strengthens MEL as system of record (incident-linked controls, approvals, evidence packs, review surfaces).
- [ ] Reduces operator dependence on external ad-hoc tooling.

## Decision-engine depth
- [ ] Improves explainable anomaly/remediation guidance using real evidence.
- [ ] Adds reusable taxonomies/rules, not one-off heuristics.

## Audit trust
- [ ] Improves replayability, causality chains, and action attribution.
- [ ] Preserves truthful degraded semantics under failure.

## Copyability pressure test
- [ ] Advantage cannot be replicated from UI alone without underlying historical corpus/workflow integration.
- [ ] Explicitly states why this is Maintenance vs Leverage vs Moat.

## Reject if true
- [ ] Mostly cosmetic without trust/evidence/workflow gain.
- [ ] Adds claims without verification.
- [ ] Introduces ambiguity in control outcomes or state truth.
