# Backup and restore

Create a backup bundle:

```bash
./bin/mel backup create --config /etc/mel/mel.json --out ./mel-backup.tgz
```

Validate a restore without modifying state:

```bash
./bin/mel backup restore --bundle ./mel-backup.tgz --dry-run --destination ./restore-preview
```

RC1 intentionally ships dry-run validation only for restore so operators can inspect what would happen before any write path is added.
