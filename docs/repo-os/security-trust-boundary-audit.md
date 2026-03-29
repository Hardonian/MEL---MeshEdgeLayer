# Security & Trust-Boundary Audit

Use for auth, RBAC/capability checks, internal APIs, exports, runtime config, tenant/actor scoping.

## Boundary definition
- [ ] Actor/system boundaries are explicit at endpoint and service layers.
- [ ] Control execution boundary is separate from control request submission.
- [ ] Historical/imported evidence boundary is separate from live telemetry boundary.

## Access control
- [ ] Least privilege maintained; no broad permission shortcuts.
- [ ] Cross-tenant/cross-scope access paths are denied by default.
- [ ] Capability checks are centralized/typed where possible.

## Failure semantics
- [ ] No silent fail-open on authn/authz checks.
- [ ] HTTP/CLI errors use truthful semantics (401/403/404/409/422 where relevant).
- [ ] Denied and conflict paths are auditable.

## Data protection
- [ ] No secret/token leakage in logs, responses, or exported artifacts.
- [ ] Evidence exports include redaction posture and provenance metadata.
- [ ] Sensitive config expectations (permissions/ownership) are validated.

## Anti-theatre
- [ ] Security claims map to implemented checks/tests.
- [ ] No “hardened/secure” language without verifiable controls.
