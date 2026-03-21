# MEL Release Compatibility Policy

**Version:** 1.0  
**Date:** March 2026  
**Status:** Proposed Framework

This document defines the compatibility guarantees, migration expectations, and breaking change policies for MEL releases.

---

## Versioning Scheme

MEL follows [Semantic Versioning 2.0.0](https://semver.org/):

```
MAJOR.MINOR.PATCH

MAJOR - Breaking changes requiring operator action
MINOR - New features, backward compatible
PATCH - Bug fixes, backward compatible
```

### Pre-1.0 Exception

Until MEL reaches 1.0:
- MINOR versions may include breaking changes
- Migration guides provided for MINOR changes
- No long-term support for pre-1.0 releases

---

## Schema Compatibility Expectations

### Database Schema

| Change Type | Compatibility | Migration Required | Example |
|-------------|---------------|-------------------|---------|
| New table | Backward | No | Adding `users` table |
| New column (nullable) | Backward | No | Adding `metadata` column |
| New column (non-nullable) | Breaking | Yes | Adding required `site_id` |
| Column rename | Breaking | Yes | Renaming `node_num` to `node_id` |
| Column removal | Breaking | Yes | Removing deprecated column |
| Index addition | Backward | No | Performance improvement |
| Index removal | Backward | No | Cleanup |

### Schema Migration Guarantees

**Forward migrations:**
- Always provided via [`migrations/`](migrations/) directory
- Sequential numbering ensures order
- Transaction-wrapped for atomicity
- Idempotent (safe to re-run)

**Rollback:**
- Not automatically supported
- Restore from backup recommended
- Downgrade migrations provided for critical fixes only

### Schema Version Tracking

```sql
-- Schema versions tracked in schema_migrations table
SELECT version, applied_at FROM schema_migrations ORDER BY version;
```

MEL refuses to start if:
- Database schema version > expected version (downgrade required)
- Migration fails (manual intervention required)

---

## Migration Ordering

### Migration Dependencies

Migrations must be applied in strict order:

```
0001_init.sql
0002_runtime_truth.sql
0003_dead_letters.sql
...
0012_performance_indexes.sql
```

**Rule:** Never skip a migration. Each depends on previous state.

### Migration Safety

| Operation | Safe? | Notes |
|-----------|-------|-------|
| CREATE TABLE | Yes | No existing data affected |
| CREATE INDEX | Yes | May be slow on large tables |
| ALTER TABLE ADD COLUMN | Yes | With DEFAULT or NULL |
| ALTER TABLE DROP COLUMN | Caution | Data loss |
| ALTER TABLE RENAME | Breaking | Requires coordination |
| DROP TABLE | Dangerous | Data loss |
| UPDATE data | Caution | Idempotency required |

### Migration Testing

Before release:
1. Migration tested on empty database
2. Migration tested on populated database
3. Rollback tested (if provided)
4. Performance measured for large datasets

---

## Rollback Posture

### Rollback Scenarios

| Scenario | Rollback Method | Data Loss Risk |
|----------|----------------|----------------|
| Failed upgrade | Automatic backup restore | Low (since last backup) |
| Performance regression | Version downgrade | None if no schema changes |
| Functional regression | Version downgrade | None if no schema changes |
| Schema-breaking bug | Restore backup | Since last backup |

### Backup Before Upgrade

MEL automatically creates backups before:
- Schema migrations
- Control action execution (optional)
- Explicit `mel backup create` command

**Operator responsibility:**
- Regular scheduled backups
- Backup verification
- Offsite backup storage

### Rollback Procedures

**Standard rollback:**
```bash
# 1. Stop MEL
sudo systemctl stop mel

# 2. Restore from backup
mel backup restore --bundle mel-backup-20260321.zip --destination ./data

# 3. Start previous version
./mel-v0.x.x serve --config configs/mel.json
```

**Emergency rollback (data loss):**
```bash
# 1. Stop MEL
# 2. Restore database file directly
cp backups/mel.db.pre-upgrade ./data/mel.db
# 3. Restart
```

---

## Breaking-Change Policy

### What Constitutes a Breaking Change

**API:**
- Removal of endpoints
- Change in request/response format
- Change in authentication method
- Removal of query parameters
- Change in error response format

**CLI:**
- Removal of commands
- Change in required flags
- Change in output format (JSON structure)
- Removal of environment variables

**Config:**
- Removal of configuration keys
- Change in key semantics
- New required keys without defaults
- Removal of valid values from enums

**Database:**
- Non-backward-compatible schema changes
- Removal of tables/columns used by operators
- Change in primary keys

**Behavior:**
- Change in default values that affect operation
- Change in security boundaries
- Removal of features

### Breaking Change Process

1. **Deprecation warning:** Minor release with deprecation notice
2. **Migration guide:** Published with release notes
3. **Grace period:** At least one minor version before removal
4. **Breaking release:** Major version increment (post-1.0)

### Deprecation Notice Format

```go
// In code
log.Warn("Deprecation", "config.key 'old_option' is deprecated and will be removed in v1.0. Use 'new_option' instead.")

// In documentation
## Deprecation Notice
The `old_option` configuration key is deprecated as of v0.9.0.
- **Removal version:** v1.0.0
- **Migration:** Replace `old_option: value` with `new_option: value`
- **Docs:** [Migration Guide](link)
```

---

## Config Compatibility Expectations

### Config Versioning

Configs are not versioned explicitly. Compatibility determined by:
- Key existence checks
- Type validation
- Unknown key warnings (with `strict_mode`)

### Config Validation

```bash
# Validate config before upgrade
mel config validate --config configs/mel.json
```

Validation checks:
- Required keys present
- Types correct
- Values in valid ranges
- No conflicting options

### Environment Variable Overrides

Environment variables override config file values:

```bash
MEL_AUTH_ENABLED=true mel serve --config configs/mel.json
```

**Compatibility:** Environment variables are stable interface

### Config Migration

When breaking changes occur:

```bash
# Automatic migration (where possible)
mel config migrate --from v0.8 --to v0.9 --config configs/mel.json

# Manual review required
mel config validate --config configs/mel.migrated.json
```

---

## UI/API Compatibility Assumptions

### API Stability Levels

| Endpoint Category | Stability | Guarantee |
|-------------------|-----------|-----------|
| `/api/v1/status` | Stable | No breaking changes in minor releases |
| `/api/v1/messages` | Stable | Pagination format stable |
| `/api/v1/control/*` | Evolving | May change in minor pre-1.0 |
| `/debug/*` | Unstable | No guarantee |
| `/api/internal/*` | Private | May change anytime |

### API Versioning

Current: `/api/v1/`

Future breaking changes: `/api/v2/`

**Policy:** Old API versions supported for at least 2 minor releases

### UI Compatibility

Web UI:
- Backward compatible with same-version API
- May require browser refresh on upgrade
- No explicit versioning

CLI:
- Output format stable (JSON)
- Human-readable text may change
- Exit codes stable

### WebSocket Compatibility

WebSocket protocol:
- Event format stable within major version
- Reconnection handling client responsibility
- No protocol negotiation

---

## Compatibility Matrix

### Supported Upgrade Paths

| From | To | Supported? | Notes |
|------|-----|------------|-------|
| 0.8.x | 0.9.x | Yes | Direct upgrade |
| 0.8.x | 0.10.x | Yes | Via 0.9.x recommended |
| 0.7.x | 0.9.x | No | Upgrade via intermediate |
| 0.9.x | 0.8.x | No | Rollback via backup |

### Cross-Version Compatibility

| Component | Compatibility |
|-----------|---------------|
| Database | Forward only |
| Config | Forward with validation |
| API | Backward within major version |
| CLI | Backward within major version |
| UI | Same version only |

---

## Release Support Lifecycle

### Support Periods (Post-1.0)

| Version Type | Support Period | Notes |
|--------------|----------------|-------|
| Major (x.0.0) | 12 months | Long-term support |
| Minor (x.y.0) | 3 months | Until next minor |
| Patch (x.y.z) | Until next patch | Immediate replacement |

### End-of-Life Announcement

EOL announced:
- 3 months before EOL for major versions
- 1 month before EOL for minor versions

### Security Fixes

Security patches backported to:
- Current major version
- Previous major version (6 months overlap)

---

## Upgrade Procedures

### Pre-Upgrade Checklist

- [ ] Read release notes
- [ ] Review breaking changes
- [ ] Run `mel config validate`
- [ ] Create backup
- [ ] Verify backup integrity
- [ ] Test in staging environment
- [ ] Schedule maintenance window

### Upgrade Steps

```bash
# 1. Create backup
mel backup create --config configs/mel.json --out mel-backup-$(date +%Y%m%d).zip

# 2. Stop MEL
sudo systemctl stop mel

# 3. Replace binary
cp mel-v0.9.0 /usr/local/bin/mel

# 4. Validate config (check for deprecations)
mel config validate --config configs/mel.json

# 5. Start MEL
sudo systemctl start mel

# 6. Verify health
mel doctor --config configs/mel.json
```

### Post-Upgrade Verification

- [ ] Process starts without errors
- [ ] Database migrations complete
- [ ] API responds correctly
- [ ] UI loads
- [ ] Transports connect
- [ ] Data ingestion working
- [ ] No errors in logs

---

## Testing Compatibility

### CI/CD Compatibility Testing

| Test | Frequency | Scope |
|------|-----------|-------|
| Migration tests | Every PR | All migrations |
| API compatibility | Every release | Public endpoints |
| Config validation | Every PR | All config paths |
| Rollback test | Every release | Backup/restore |

### Compatibility Test Matrix

| From Version | To Version | Test Result |
|--------------|------------|-------------|
| v0.8.0 | v0.9.0 | [Test result] |
| v0.8.0 | v0.9.1 | [Test result] |
| v0.9.0 | v0.10.0 | [Test result] |

---

## Documenting Breaking Changes

### Release Notes Format

```markdown
## Release v0.9.0

### Breaking Changes
- **Config:** `control.allow_restart` renamed to `control.allow_transport_restart`
  - Migration: Update config file
  - Deprecated in: v0.8.0
  
### New Features
- Multi-user authentication
- Config history tracking

### Deprecations
- `auth.single_user` - will be removed in v1.0.0

### Migration Guide
See [Migrating to v0.9.0](migration/v0.9.0.md)
```

### Migration Guide Template

```markdown
# Migrating from v0.8.x to v0.9.x

## Prerequisites
- Backup your database
- Export current config

## Step 1: Update Config
[Specific instructions]

## Step 2: Database Migration
[What to expect]

## Step 3: Verification
[How to verify success]

## Rollback
[If something goes wrong]
```

---

## Related Documents

- PRODUCTION_MATURITY_MATRIX.md - Current state assessment
- PRODUCTION_CLOSURE_ROADMAP.md - Improvement timeline
- docs/ops/upgrades.md - Operator upgrade guide
- CHANGELOG.md - Historical changes
- docs/release/RELEASE_CHECKLIST.md - Release process
