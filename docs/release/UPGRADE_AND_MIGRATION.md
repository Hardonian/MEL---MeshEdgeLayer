# Upgrade and Migration

## Safe upgrade sequence

1. Create a backup bundle before changing binaries.
2. Deploy new binary.
3. Start MEL and run `mel doctor`.
4. Validate ingest and action lifecycle behavior.
5. Confirm no unsupported feature assumptions were introduced.

## Migration posture

- Database schema evolution follows repo migrations.
- Downgrades are not guaranteed unless explicitly documented per release.
- If compatibility risk exists, block rollout and preserve backup evidence.

## Current boundary

- Restore remains validation-only (`--dry-run`) in current release posture.

## References

- [Ops upgrades](../ops/upgrades.md)
- [Backup/restore](../ops/backup-restore.md)
- [Release checklist](./RELEASE_CHECKLIST.md)
