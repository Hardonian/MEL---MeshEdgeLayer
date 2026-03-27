# Runbook: Database Maintenance & Recovery

MEL uses SQLite for its primary persistence. While it's generally low-maintenance, growth and high-concurrency environments require periodic tuning and cleanup.

---

## 🚦 Growth Symptoms & Diagnostic Steps

### Symptom
- Database file (`mel.db`) growing beyond expected size (e.g. > 10GB).
- Application-level storage alerts firing.
- Query performance degradation (e.g., slow dashboard load).

### Diagnostic Steps

```bash
# Check current system status and DB size
mel status

# Check SQLite internal health
sqlite3 /path/to/mel.db "PRAGMA integrity_check;"
```

Review table sizes to find the growth source:
- `messages`: Historical packets.
- `dead_letters`: Failed packets.
- `control_actions`: Historical control plane actions.
- `audit_logs`: Application logs and audit trail.

---

## 🛠️ Resolution Steps

### 1. Retention Tuning
Adjust your `mel.json` to keep only what's necessary:
```json
"storage": {
  "retention_days": 30,
  "dead_letter_retention_days": 7
}
```

### 2. Manual Cleanup
Run a cleanup pass to prune records older than your retention policy:
```bash
mel db cleanup --all
```

### 3. Database Vacuuming
Reclaim empty space from the database file:
```bash
# WARNING: This will lock the database for a duration proportional to its size.
# Stop the mel service first if on a low-perf host.
sqlite3 /path/to/mel.db "VACUUM;"
```

---

## 🏥 System Recovery

### Symptom
The MEL service refuses to start with persistent database errors (e.g., `disk I/O error`, `malformed file`).

### Resolution Steps

1. **Verify Mount & Permissions**
   - Ensure the disk is not full: `df -h`
   - Ensure the `mel` user has `RW` access to the DB path.

2. **Recover from Malformed File** (Last Resort)
   If the file is corrupt:
   ```bash
   # Try to recover as much as possible to a new file
   sqlite3 mel.db ".recover" | sqlite3 mel_recovered.db
   mv mel_recovered.db mel.db
   ```

3. **Restoring from Backup**
   If you have a `mel.backup` file:
   ```bash
   mel backup restore --file /path/to/backup.mel --config /etc/mel/mel.json
   ```

---

## 🚀 Prevention & Best Practices

- **Monitor Disk Space**: Alert at **80%** disk utilization.
- **Scheduled Backups**: Use `mel backup create` daily.
- **Use WAL Mode**: MEL defaults to Write-Ahead Logging (WAL) for better concurrency; do not disable it.
- **SQLite Suitability**: If you consistently exceed **5000 writes/sec**, consider offloading to a central aggregation instance.
