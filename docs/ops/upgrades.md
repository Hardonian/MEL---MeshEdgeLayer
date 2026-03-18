# Upgrades

1. Create a backup.
2. Replace the binary.
3. Restart MEL.
4. Check schema version and doctor output.

Helper:

```bash
sudo ./scripts/upgrade-linux.sh
```

If schema compatibility changes in a future release, MEL should warn before operators attempt a downgrade.
