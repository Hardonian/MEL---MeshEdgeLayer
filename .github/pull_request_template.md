# MEL Pull Request

## Design Intent

- What change is being made and why?
- How does this reduce entropy or increase structural coherence?

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
