# Release checklist (truth-first template)

Use this checklist per release candidate. Do **not** commit it as globally passed unless the commands were re-run for the current commit and environment.

## Truth and support posture

- [ ] README, transport matrix, and known limitations match current code and tests.
- [ ] Unsupported transports remain explicitly labeled unsupported (BLE ingest, HTTP ingest, MEL RF routing/propagation).
- [ ] No new control-plane or runtime claims exceed deterministic evidence.

## Runtime/tooling contract

- [ ] Go toolchain required by `go.mod` is installed.
- [ ] Frontend Node runtime is `24.x` (`frontend/.nvmrc`, `frontend/package.json`, `frontend/scripts/require-node24.mjs`).
- [ ] `./bin/mel` artifact exists before running smoke (`make build-cli` or `make build`).

## Verification gates (run on current commit)

- [ ] Prefer `make premerge-verify` for deterministic chained local verification (single frontend `npm ci` per run, enforced by `make check-frontend-install-churn`).
- [ ] Optional local iteration only: `make premerge-verify-fast` (never substitute for release-grade verification evidence).
- [ ] `make lint`
- [ ] `make frontend-typecheck`
- [ ] `make frontend-test`
- [ ] `make test`
- [ ] `make build`
- [ ] `make smoke`
- [ ] `make product-verify` (for release candidates)

Record exact pass/fail output and any environment limitations in the release notes.

## Evidence pack

- [ ] Command output evidence stored under `docs/release/evidence/` with date + commit reference.
- [ ] Caveats/residual risk documented with explicit boundaries.
- [ ] UI screenshot evidence attached for operator-facing UI changes when browser tooling is available.

## Residual risk disclosure

- [ ] Known pre-existing failures (if any) explicitly listed.
- [ ] Environment-blocked checks (if any) explicitly listed.
- [ ] Anything still not claimable is documented in release notes.

## Companion release docs

- [Release criteria](./RELEASE_CRITERIA.md)
- [Upgrade and migration](./UPGRADE_AND_MIGRATION.md)
- [Backup and restore](./BACKUP_AND_RESTORE.md)
- [Compatibility and support matrix](./COMPATIBILITY_AND_SUPPORT_MATRIX.md)
- [Support runbook](./SUPPORT_RUNBOOK.md)
- [Security model](./SECURITY_MODEL.md)
- [Privacy and data posture](./PRIVACY_AND_DATA_POSTURE.md)
