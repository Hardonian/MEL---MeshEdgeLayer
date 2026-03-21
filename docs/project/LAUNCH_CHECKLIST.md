# MEL Launch Checklist (RC1)

## Last updated: 2026-03-21

This checklist defines the "Launch Quality Bar" for MEL. No release leaves without passing this pass.

## 🏁 Minimum Launch Quality

- [ ] **Build Integrity**: `make build` and `make cross-build` must complete without errors.
- [ ] **Lint & Style**: `gofmt` and `go vet` must pass across all packages.
- [ ] **Unit Tests**: `go test ./...` must pass with 0 failures.
- [ ] **Binary Versioning**: `./bin/mel version` must report the correct Git commit and build time.
- [ ] **Smoke Test**: `./scripts/smoke.sh` must verify real ingest-to-persistence flow.

## 🩺 Diagnostic Readiness

- [ ] **Doctor v2**: `mel doctor` must correctly identify missing configs, invalid permissions, and unreachable transports.
- [ ] **Reality Check**: Doctor summary must accurately reflect "What MEL Does" and "What MEL Does Not Do".
- [ ] **Persistence Proof**: Doctor must prove a real SQLite write/read cycle.

## 🛡️ Security & Privacy

- [ ] **Config Hardening**: Launch must fail if config file permissions are broader than 0600.
- [ ] **Privacy Audit**: `mel privacy audit` must correctly identify position storage status.
- [ ] **Redaction Ready**: `mel export` must default to redacted mode unless explicitly overridden.

## 🤝 Contributor Readiness

- [ ] **README Skimmability**: Is the value prop and quickstart clear in 30 seconds?
- [ ] **CONTRIBUTING.md**: Are the repo truths and safe extension areas documented?
- [ ] **Issue Templates**: Do bug and feature templates exist in `.github/ISSUE_TEMPLATE`?
- [ ] **PR Template**: Does the pull request template require design intent and verification?

## 🖥️ UI & CLI Polish

- [ ] **Help-Text Consistency**: Do `mel --help` commands match the actual implemention?
- [ ] **Empty States**: Do the Web Dashboard and TUI handle "no data" gracefully with guidance?
- [ ] **Zero Theatre**: Is all copy technically honest and free of marketing fluff?
