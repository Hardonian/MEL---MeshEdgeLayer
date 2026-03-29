# MEL Pull Request

## Design Intent

- What change is being made and why?
- How does this reduce entropy or increase structural coherence?
- Work classification: **Maintenance / Leverage / Moat** (see `docs/repo-os/change-classification.md`).

## Operator Impact

- How does this change affect the operator experience (CLI, UI, Logs)?
- Are there any changes to `mel doctor` or `mel status` reporting?

## Verification Evidence

- [ ] **Unit Tests**: `go test ./...`
- [ ] **Lint/Vet**: `make lint`
- [ ] **Manual Proof**: List the exact commands and output you used to verify the change (e.g., `mel doctor`).

## Privacy & Security

- [ ] No new PII or precise location data is stored by default.
- [ ] No changes to config file permission requirements.
- [ ] Any new CLI/API exports have redaction support where applicable.

## Risks & Remaining Limitations

- Describe any residual risk or explicitly say "none beyond documented scope".

## Repo-OS Audits

- [ ] Operator Truth Audit (`docs/repo-os/operator-truth-audit.md`)
- [ ] Transport Truth & Degraded-State Audit (`docs/repo-os/transport-truth-audit.md`)
- [ ] Trusted Control Governance Checklist (if control paths touched) (`docs/repo-os/trusted-control-governance.md`)
- [ ] Security / Trust-Boundary Audit (`docs/repo-os/security-trust-boundary-audit.md`)
- [ ] Moat Evaluation completed for major feature/refactor (`docs/repo-os/moat-evaluation.md`)
