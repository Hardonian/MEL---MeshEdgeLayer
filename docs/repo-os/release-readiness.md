# Release Readiness / Go-Live Reality Pass

Use this before merging changes that alter capability claims or operator-critical surfaces.

## 1) Claim-to-implementation alignment
- [ ] Every capability claim is implemented and verified.
- [ ] Unsupported/partial features are labeled with explicit boundaries.
- [ ] No docs/UI text implies unimplemented transport/protocol/control behavior.

## 2) Operator-truth integrity
- [ ] Live/stale/historical/imported/partial semantics are preserved end-to-end.
- [ ] Degraded states are explicit in API and operator surfaces.
- [ ] Unknown states are not rendered as healthy by default.

## 3) Control and governance integrity
- [ ] Action lifecycle semantics are explicit and tested.
- [ ] Approval requirements and audit trails are enforced.
- [ ] Execution outcomes include clear proof/failure evidence.

## 4) Security and trust boundaries
- [ ] Access controls were validated for changed paths.
- [ ] No new fail-open behavior or silent boundary broadening.
- [ ] Secret handling and redaction checks are satisfied.

## 5) Operational readiness
- [ ] Runbooks updated for new operator behaviors.
- [ ] Config/runtime prerequisites documented.
- [ ] Known caveats and residual risks documented with concrete scope.

## 6) Verification evidence package
- [ ] Verification commands/results attached.
- [ ] Tests updated for new truth/control semantics.
- [ ] Evidence artifacts linked (logs, screenshots, payload examples) where applicable.

## Merge gate
Do not mark “ready” unless all required boxes are checked or explicitly waived with owner + rationale.
