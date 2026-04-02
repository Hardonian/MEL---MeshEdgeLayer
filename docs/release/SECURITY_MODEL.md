# Security Model (Operational Summary)

## Core boundaries

- AuthN/AuthZ decisions must be explicit and capability-bound.
- Control actions require attributable actor identity.
- Approval, dispatch, execution, and audit remain separate states.

## Non-goals / non-claims

- No implied compliance certification.
- No claim of complete tenant isolation beyond currently implemented boundaries.
- No claim of cryptographic at-rest guarantees not implemented in product.

## Security operating checks

- Run config validation and doctor checks before go-live.
- Ensure control policy is intentional (disabled/advisory/guarded_auto).
- Keep emergency freeze paths tested and documented.

## References

- [Security policy](../../SECURITY.md)
- [Threat model](../threat-model/README.md)
- [Control-plane trust model](../architecture/CONTROL_PLANE_TRUST_MODEL.md)
