# MEL Release Checklist

Use this checklist for every release. Complete every item; do not skip.

---

## Pre-Release

### Code Readiness

- [ ] All feature branches merged to `main` (or release branch).
- [ ] `go build ./...` passes with no errors.
- [ ] `go vet ./...` passes with no warnings.
- [ ] `go test ./...` passes; all tests green.
- [ ] No `TODO: pre-release` or `FIXME` comments left in code.
- [ ] `go.mod` and `go.sum` are up to date (`go mod tidy`).

### Database Migrations

- [ ] New migration file follows naming convention `NNNN_<description>.sql`.
- [ ] Migration is idempotent (uses `CREATE TABLE IF NOT EXISTS`, `ADD COLUMN` with `DEFAULT`).
- [ ] Migration tested against an existing production-schema database.
- [ ] Migration tested against a fresh empty database.
- [ ] `mel doctor` passes after migration.

### Security Review

- [ ] No new secret material hardcoded in source.
- [ ] New config fields validated in `config.Validate`.
- [ ] New API endpoints protected by auth middleware (capability checks).
- [ ] SQL queries use `esc()` / `ValidateSQLInput()`; no raw string interpolation from user input.
- [ ] File permission checks present for any new files written to disk.

### Documentation

- [ ] `CONTROL_PLANE_TRUST.md` updated if trust model changed.
- [ ] `CONFIG_REFERENCE.md` updated for new config fields.
- [ ] `API_CLI_REFERENCE.md` updated for new endpoints and commands.
- [ ] `OPERATIONS_RUNBOOK.md` updated for new operator workflows.
- [ ] `CHANGELOG.md` entry written.

---

## Release Build

- [ ] Version constant updated in `internal/version/version.go`.
- [ ] Release tag created: `git tag v<semver>`.
- [ ] Cross-compiled binaries built:
  - `GOOS=linux GOARCH=amd64`
  - `GOOS=linux GOARCH=arm64`
  - `GOOS=linux GOARCH=arm GOARM=7` (Raspberry Pi)
- [ ] Binaries stripped and compressed.
- [ ] SHA-256 checksums file generated.
- [ ] Binaries and checksums uploaded to release page.

---

## Deployment Validation

Run the following after deploying to a staging environment:

- [ ] `mel doctor --config <path>` returns no `critical` findings.
- [ ] `mel config validate --config <path>` exits 0.
- [ ] `mel serve` starts without panics in log.
- [ ] `GET /api/v1/health` returns `{"status":"ok"}`.
- [ ] `GET /api/v1/status` returns transport state.
- [ ] `mel control operational-state --config <path>` returns valid JSON.
- [ ] `mel timeline --config <path>` returns valid JSON.
- [ ] Migration applied: `mel db vacuum` exits 0.
- [ ] Backup created and validated: `mel backup create && mel backup restore --dry-run`.

### Control Plane Smoke Tests (if control plane enabled)

- [ ] `mel control status` shows expected mode.
- [ ] Install a global freeze, verify `mel control operational-state` reflects it.
- [ ] Clear the freeze, verify operational state is clean.
- [ ] Create and cancel a maintenance window.
- [ ] (If approval required): trigger a pending action, approve it, verify it executes.

---

## Post-Release

- [ ] Release announcement posted.
- [ ] `main` branch bumped to next development version.
- [ ] Monitoring alert thresholds reviewed for new metrics.
- [ ] Staging environment restored to clean state.

---

## Rollback Plan

If a defect is found after release:

1. Install a global freeze on all production instances immediately:
   ```bash
   mel freeze create --reason "Release rollback in progress" --actor "ops"
   ```

2. Stop the MEL service:
   ```bash
   sudo systemctl stop mel
   ```

3. Restore the previous binary from backup.

4. If a migration was applied and the schema is incompatible, restore from
   the pre-upgrade backup:
   ```bash
   mel backup restore --bundle <pre-upgrade-bundle> --dry-run
   # After validation:
   cp <db-from-bundle> <database_path>
   ```

5. Start MEL with the previous binary and validate.

6. Clear the freeze once the rollback is confirmed healthy.
