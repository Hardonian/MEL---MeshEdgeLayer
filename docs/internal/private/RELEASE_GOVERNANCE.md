# Internal Release Governance

## Required sign-offs

- Product truth review (capability and maturity language).
- Engineering verification review (`make lint/test/build/smoke`).
- Ops/support readiness review.
- Privacy/security boundary review for changed surfaces.

## Blockers

- Unsupported feature implied as shipped.
- Missing degraded-state signaling for new paths.
- No verification artifacts for high-risk claims.

## Evidence discipline

Each release candidate should include a dated evidence note under `docs/release/evidence/`.
