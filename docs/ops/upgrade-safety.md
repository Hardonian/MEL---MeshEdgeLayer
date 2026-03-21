# MEL Upgrade Safety Guide

**Version:** 1.0  
**Date:** March 2026  
**Status:** Implemented

This document provides comprehensive guidance on MEL upgrade procedures, version compatibility, and rollback safety.

---

## Version Compatibility Matrix

### MEL Versions

| Version | Type | Schema Version | Support Status |
|---------|------|----------------|----------------|
| 0.1.x | Development | 1-15 | Not for production |
| 0.9.x | Preview/RC | 15 | Testing only |
| 1.0.x | Stable | 15 | Production ready |

### Upgrade Path Support

| From | To | Supported | Notes |
|------|-----|-----------|-------|
| 0.1.x | 0.9.x | Yes | Direct upgrade |
| 0.1.x | 1.0.x | Yes | Via 0.9.x recommended |
| 0.9.x | 1.0.x | Yes | Direct upgrade |
| 0.9.x | 0.1.x | No | Rollback via backup |
| 1.0.x | 0.9.x | No | Rollback via backup |

---

## Upgrade Procedures

### Pre-Upgrade Checklist

Before upgrading MEL, complete the following:

- [ ] Review release notes for breaking changes
- [ ] Run `mel config validate --config configs/mel.json`
- [ ] Create a backup: `mel backup create --config configs/mel.json --out backup.zip`
- [ ] Verify backup integrity
- [ ] Test in staging environment (if available)
- [ ] Schedule maintenance window
- [ ] Notify users of downtime

### Standard Upgrade Steps

```bash
# 1. Create backup
mel backup create --config configs/mel.json --out mel-backup-$(date +%Y%m%d).zip

# 2. Stop MEL
sudo systemctl stop mel

# 3. Replace binary
cp mel-v1.0.0 /usr/local/bin/mel

# 4. Validate config
mel config validate --config configs/mel.json

# 5. Start MEL
sudo systemctl start mel

# 6. Verify health
mel doctor --config configs/mel.json

# 7. Check version
mel version
```

### Post-Upgrade Verification

- [ ] Process starts without errors
- [ ] Database migrations complete successfully
- [ ] API responds correctly: `curl http://localhost:8080/api/v1/version`
- [ ] UI loads correctly
- [ ] Transports connect successfully
- [ ] Data ingestion working
- [ ] No errors in logs: `mel logs tail --config configs/mel.json`

---

## Rollback Constraints and Limitations

### When Rollback is NOT Supported

- **Schema downgrade:** MEL does not support automatic schema downgrades
- **Major version rollback:** Cannot rollback across major versions
- **More than one minor version:** Cannot rollback more than one minor version

### Rollback Procedure (via backup)

```bash
# 1. Stop MEL
sudo systemctl stop mel

# 2. Restore from backup
mel backup restore --bundle mel-backup-20260321.zip --destination ./data

# 3. Restore previous binary
cp mel-v0.9.0 /usr/local/bin/mel

# 4. Start MEL
sudo systemctl start mel
```

### Emergency Rollback (Data Loss)

```bash
# 1. Stop MEL
# 2. Restore database file directly
cp backups/mel.db.pre-upgrade ./data/mel.db
# 3. Restart
```

**Warning:** This method may result in data loss if changes were made after the backup was created.

---

## Required Backups Before Upgrade

### Critical Operations Requiring Backup

The following operations **require** a recent backup:

1. **Schema migrations** - Any migration that modifies table structure
2. **Column drops** - Removing data columns
3. **Table drops** - Removing entire tables
4. **Data deletions** - Bulk data removal operations

### Backup Age Requirements

- **Maximum age:** 7 days
- **Recommended:** Within 24 hours of upgrade

### Creating a Backup

```bash
# Create backup
mel backup create --config configs/mel.json --out mel-backup-$(date +%Y%m%d).zip

# Verify backup
mel backup restore --bundle mel-backup.zip --dry-run
```

---

## Breaking Changes Policy

### What Constitutes a Breaking Change

**API:**
- Removal of endpoints
- Change in request/response format
- Change in authentication method
- Removal of query parameters

**CLI:**
- Removal of commands
- Change in required flags
- Change in output format

**Config:**
- Removal of configuration keys
- Change in key semantics
- New required keys without defaults

**Database:**
- Non-backward-compatible schema changes
- Removal of tables/columns

### Breaking Change Process

1. **Deprecation warning:** Release with deprecation notice
2. **Migration guide:** Published with release notes
3. **Grace period:** At least one minor version before removal
4. **Breaking release:** Major version increment

---

## Schema Migration Rules

### Migration Safety Classification

| Operation | Safe | Reversible | Notes |
|-----------|------|------------|-------|
| CREATE TABLE | Yes | Yes | No existing data affected |
| CREATE INDEX | Yes | Yes | May be slow on large tables |
| ADD COLUMN (nullable) | Yes | Yes | With DEFAULT or NULL |
| ADD COLUMN (required) | Caution | No | Requires migration |
| DROP COLUMN | Dangerous | No | Data loss |
| RENAME COLUMN | Caution | No | Requires coordination |
| DROP TABLE | Dangerous | No | Data loss |
| UPDATE data | Caution | N/A | Idempotency required |

### Current Schema Version

Current schema version: **15**

### Checking Schema Version

```bash
# Via CLI
mel doctor --config configs/mel.json

# Via API
curl http://localhost:8080/api/v1/version
```

---

## Mixed-Version Cluster Warnings

### Single-Node Deployment

MEL should run as a single instance. Do not run multiple versions simultaneously.

### Configuration Across Versions

- Always use configuration compatible with the oldest version in use
- Test configuration changes in staging before production
- Keep configuration files under version control

---

## API Endpoints

### Version Information

```bash
# Get version info
curl http://localhost:8080/api/v1/version

# Response
{
  "version": "1.0.0",
  "git_commit": "abc1234",
  "build_time": "2026-03-21T00:00:00Z",
  "go_version": "go1.21",
  "db_schema_version": 15,
  "compatibility_level": "stable"
}
```

### Upgrade Health Check

```bash
# Check upgrade readiness
curl http://localhost:8080/api/v1/health/upgrade

# Response
{
  "ready": true,
  "overall_status": "ready",
  "timestamp": "2026-03-21T00:00:00Z",
  "version_info": {...},
  "current_schema": 15,
  "required_schema": 15,
  "checks": [
    {"name": "disk_space", "status": "pass"},
    {"name": "db_integrity", "status": "pass"},
    {"name": "backup_exists", "status": "pass"},
    {"name": "version_compatibility", "status": "pass"},
    {"name": "schema_version", "status": "pass"}
  ],
  "summary": {
    "total": 5,
    "passed": 5,
    "failed": 0,
    "warnings": 0,
    "skipped": 0
  }
}
```

---

## CLI Commands

### Version Command

```bash
# Display version
mel version

# Display full version details
mel version --verbose
```

### Doctor Command

```bash
# Run full health check
mel doctor --config configs/mel.json

# Run upgrade-specific checks
mel doctor --config configs/mel.json --upgrade-check
```

### Config Validation

```bash
# Validate configuration
mel config validate --config configs/mel.json

# Inspect configuration
mel config inspect --config configs/mel.json
```

---

## Support and Troubleshooting

### Common Issues

**Schema version mismatch:**
```
Error: Database schema is outdated: v14, need v15
```
Solution: Run `mel serve --config configs/mel.json` to apply migrations

**Backup required:**
```
Error: No backup found - a backup is required before this operation
```
Solution: Run `mel backup create --config configs/mel.json`

**Version incompatibility:**
```
Error: Cannot skip major versions
```
Solution: Upgrade through intermediate versions

### Getting Help

- Documentation: `/docs/ops/`
- CLI Reference: `/docs/ops/cli-reference.md`
- API Reference: `/docs/ops/api-reference.md`
- Troubleshooting: `/docs/ops/troubleshooting.md`

---

## Related Documents

- [RELEASE_COMPATIBILITY_POLICY.md](../architecture/RELEASE_COMPATIBILITY_POLICY.md) - Compatibility policy
- [PRODUCTION_MATURITY_MATRIX.md](../architecture/PRODUCTION_MATURITY_MATRIX.md) - Feature maturity
- [cli-reference.md](cli-reference.md) - CLI documentation
- [api-reference.md](api-reference.md) - API documentation
- [troubleshooting.md](troubleshooting.md) - Troubleshooting guide
