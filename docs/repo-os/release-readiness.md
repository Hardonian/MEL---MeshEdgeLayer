# Release Readiness / Go-Live Reality Pass

Use this before merging changes that alter capability claims or operator-critical surfaces.

## 1) Claim-to-implementation alignment
- [ ] Every capability claim is implemented and verified.
- [ ] Unsupported/partial features are labeled with explicit boundaries.
- [ ] No docs/UI text implies unimplemented transport/protocol/control behavior.

## 2) Operator-truth integrity
- [ ] Live/stale/historical/imported/partial/degraded/unknown semantics are preserved end-to-end.
- [ ] Degraded and unknown states are explicit in API and operator surfaces.
- [ ] No certainty language outruns evidence sufficiency.

## 3) Precision-layering integrity
- [ ] Diagnostics/health/recommendation changes preserve observed vs inferred vs estimated vs unknown distinctions.
- [ ] Physical/environment context is not presented as proven root cause without evidence.
- [ ] Mixed-channel truth posture (channel/path/support/privacy/assist flags) is explicit when relevant.

## 4) Control and governance integrity
- [ ] Action lifecycle semantics are explicit and tested.
- [ ] Approval requirements and audit trails are enforced.
- [ ] Execution outcomes include clear proof/failure evidence.

## 5) Privacy and trust boundaries
- [ ] Access controls were validated for changed paths.
- [ ] No new fail-open behavior or silent boundary broadening.
- [ ] Secret handling and redaction checks are satisfied.
- [ ] Telemetry/privacy defaults remain explicit and privacy-preserving.

## 6) Local inference integrity (if applicable)
- [ ] Assistive inference output is labeled non-canonical.
- [ ] Base MEL truth/control viability does not depend on inference runtime success.
- [ ] Runtime fallback behavior is truthful and explicit.

## 7) Operational readiness
- [ ] Runbooks updated for new operator behaviors.
- [ ] Config/runtime prerequisites documented.
- [ ] Known caveats and residual risks documented with concrete scope.
- [ ] `GET /api/v1/platform/posture` reflects true telemetry/export/delete/inference policy for this release.

## 8) Verification evidence package
- [ ] Verification commands/results attached.
- [ ] Tests updated for new truth/control/privacy semantics.
- [ ] Evidence artifacts linked (logs, screenshots, payload examples) where applicable.

## Merge gate
Do not mark “ready” unless all required boxes are checked or explicitly waived with owner + rationale.
