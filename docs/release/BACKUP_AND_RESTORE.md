# Backup and Restore

## Current support state

- Backup creation: supported.
- Restore: validation-only (`--dry-run` required).

## Operator procedure

```bash
./bin/mel backup create --config /etc/mel/mel.json --out ./mel-backup.tgz
./bin/mel backup restore --bundle ./mel-backup.tgz --dry-run --destination ./restore-preview
```

## Required evidence

For release/go-live readiness, capture:

- backup command output
- restore dry-run validation output
- caveats discovered during preview

## Boundary statement

Do not claim production write-restore behavior until non-dry-run restore is implemented and verified.
