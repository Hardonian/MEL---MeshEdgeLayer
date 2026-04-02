# Release Criteria

A MEL release is acceptable only if all gates below pass.

## Truth and capability gates

- Supported/unsupported transport claims match implementation.
- No UI/docs wording implies MEL performs RF routing/transmit execution.
- Degraded/unknown states remain explicit in operator surfaces.
- Assistive inference remains non-canonical and failure-tolerant.

## Operational safety gates

- Upgrade path documented with rollback caveats.
- Backup create works; restore posture is accurately labeled.
- Support bundle and proofpack export paths validated.
- Secrets/configuration handling guidance reviewed.

## Verification gates

- `make lint`, `make test`, `make build`, `make smoke` executed.
- Repo-OS verification and release-readiness checks reviewed.
- New docs links validated.

## Evidence gate

Attach command outputs and caveats under `docs/release/evidence/` for each release candidate.
