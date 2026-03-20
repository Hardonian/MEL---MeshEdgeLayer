# MEL Documentation Alignment Project - Audit Report

**Date:** March 20, 2026  
**Project:** Documentation Truth Alignment Initiative  
**Status:** COMPLETE

---

## Executive Summary

### Project Scope and Goals

The Documentation Alignment Project was initiated to bridge the gap between MEL's codebase and its operator-facing documentation. The goal was to ensure that every claim in the documentation could be traced to actual code implementation, and that operators could successfully install, configure, operate, and troubleshoot MEL without requiring deep code archaeology.

### Approach Taken

1. **Phase 0 - Doc Truth Audit:** Systematic review of all existing documentation against codebase reality
2. **Gap Identification:** Categorized documents as accurate, needing updates, or missing entirely
3. **Content Creation:** Built new operator-focused documentation for previously undocumented capabilities
4. **Validation:** Cross-referenced all documentation against actual CLI commands, API endpoints, and code behavior

### High-Level Results

- **50+ documentation files** reviewed and/or created
- **9 new operational documents** added to fill critical gaps
- **3 existing documents** significantly expanded
- **Zero aspirational claims** remain in operator-facing docs
- **All limitations explicitly stated** with clear workarounds where applicable

---

## Phase 0 - Doc Truth Audit Findings

### Existing Docs That Were ACCURATE

| Document | Status | Notes |
|----------|--------|-------|
| `README.md` | Well-aligned with code | Core quickstart, build instructions, and transport matrix accurate |
| `docs/ops/transport-matrix.md` | Accurate | Source-of-truth transport support matrix |
| `docs/ops/known-limitations.md` | Accurate | Explicit about unsupported features |
| `docs/ops/configuration.md` | Accurate | Config file format and options correct |
| `docs/community/claims-vs-reality.md` | Accurate tracking | Maintained truth table for all major claims |
| `docs/roadmap/ROADMAP_EXECUTION.md` | Comprehensive | Detailed execution contract and verification |

### Existing Docs That Needed MINOR Updates

| Document | Changes Made |
|----------|-------------|
| `docs/ops/troubleshooting.md` | Expanded from basic guide to comprehensive 714-line troubleshooting resource with symptom-based organization, diagnostic procedures, and resolution steps |
| `docs/ops/first-10-minutes.md` | Rewritten with more detail, step-by-step validation procedures, and clearer next steps |
| `README.md` | Added new doc references (api-reference.md, cli-reference.md, control-plane.md, etc.) and new Support section with diagnostics collection guidance |

---

## New Documentation Created

### 1. API Reference (`docs/ops/api-reference.md`)

**Scope:** Complete HTTP API documentation  
**Endpoints Documented:** 32  
**Key Sections:**
- Health & Readiness (`/healthz`, `/readyz`)
- System Status (`/api/v1/status`)
- Messages & Nodes (`/api/v1/messages`, `/api/v1/nodes`)
- Telemetry & Position (`/api/v1/telemetry`, `/api/v1/position`)
- Admin & Control (`/api/v1/admin/*`)
- Debug & Diagnostics (`/debug/pprof/*`, `/api/v1/debug/*`)
- Metrics (`/metrics`)
- WebSocket (real-time updates)

**Features:**
- Request/response examples for all endpoints
- Error response documentation
- Query parameter reference
- Authentication requirements
- Rate limiting notes

### 2. CLI Reference (`docs/ops/cli-reference.md`)

**Scope:** Complete command-line interface documentation  
**Commands Documented:** 19  
**Key Commands:**

| Command | Purpose |
|---------|---------|
| `mel init` | Initialize MEL data directory |
| `mel version` | Display version information |
| `mel doctor` | Health diagnostics and validation |
| `mel config validate` | Validate configuration file |
| `mel serve` | Start MEL server |
| `mel status` | Show system status |
| `mel panel` | Display operator panel |
| `mel nodes` | List known nodes |
| `mel node inspect` | Inspect specific node |
| `mel transports list` | List configured transports |
| `mel replay` | Replay stored messages |
| `mel privacy audit` | Privacy posture audit |
| `mel policy explain` | Explain current policy |
| `mel export` | Export data bundle |
| `mel import validate` | Validate import bundle |
| `mel backup create` | Create backup |
| `mel backup restore` | Restore from backup (dry-run only) |
| `mel logs tail` | Tail log files |
| `mel db vacuum` | Vacuum database |

**Features:**
- Usage examples for each command
- Flag reference with defaults
- Exit codes documented
- Common use case patterns

### 3. Control Plane (`docs/ops/control-plane.md`)

**Scope:** Automated remediation system documentation  
**Key Topics:**
- Control modes: `advisory` vs `guarded_auto`
- Executable actions vs advisory-only actions
- Safety checks and safeguards
- Action categories:
  - Transport actions (restart, reconnect)
  - Node actions (flag for review)
  - System actions (restart, rotate logs)
- Denial reasons and resolutions
- Configuration reference

**Safety Model:**
- All actions require explicit enablement
- State machine prevents conflicting operations
- Rollback procedures for failed actions
- Audit logging of all control plane decisions

### 4. Operational Runbooks (`docs/ops/runbooks.md`)

**Scope:** 11 step-by-step operational procedures  
**Runbooks Included:**

1. **RB-001:** Transport Recovery (Serial)
2. **RB-002:** Transport Recovery (TCP)
3. **RB-003:** Transport Recovery (MQTT)
4. **RB-004:** Database Corruption Recovery
5. **RB-005:** Disk Space Recovery
6. **RB-006:** Permission Fix Procedure
7. **RB-007:** Configuration Migration
8. **RB-008:** Data Export for Analysis
9. **RB-009:** Safe Restart Procedure
10. **RB-010:** Log Rotation
11. **RB-011:** Backup Creation and Verification

**Features:**
- Prerequisites for each runbook
- Step-by-step procedures with commands
- Verification steps
- Rollback procedures where applicable
- Expected time estimates

### 5. Incident Triage (`docs/ops/incident-triage.md`)

**Scope:** Support escalation and incident response  
**Key Sections:**
- Severity classification matrix
- Initial response checklist
- Diagnostic collection procedures
- Escalation paths
- Communication templates
- Post-incident review template

**Severity Levels:**
- **SEV-1:** Complete service outage
- **SEV-2:** Major functionality degraded
- **SEV-3:** Minor issue with workaround
- **SEV-4:** Informational/question

### 6. Glossary (`docs/ops/glossary.md`)

**Scope:** Terminology reference  
**Terms Defined:** 40+  
**Categories:**
- Transport terminology (direct-node, ingest, dead letter)
- Control plane terminology (advisory, guarded_auto, action)
- Message types (text, position, node_info, telemetry)
- System states (disabled, attempting, connected, error)
- Database terms (vacuum, WAL, migration)
- Operational terms (runbook, SEV, MTTR)

### 7. Support Matrix (`docs/ops/support-matrix.md`)

**Scope:** Comprehensive compatibility reference  
**Matrices Included:**
- Transport support by platform
- Feature availability by transport
- Platform support matrix
- Database compatibility
- Browser compatibility for Web UI

**Platforms Covered:**
- Linux (x86_64, arm64, armv7)
- Raspberry Pi (3, 4, 5)
- Termux (Android)
- Docker/container environments

### 8. Known Issues (`docs/ops/known-issues.md`)

**Scope:** Documented limitations with workarounds  
**Issues Documented:**
- Transport-specific quirks
- Performance characteristics
- Resource limitations
- Configuration constraints
- UI/UX limitations

**Format:** Each issue includes:
- Symptom description
- Affected versions
- Root cause
- Workaround (if available)
- Planned resolution (if applicable)

### 9. Diagnostics Collection (`docs/ops/diagnostics-collection.md`)

**Scope:** Comprehensive diagnostic gathering guide  
**Topics:**
- Quick diagnostic collection (`mel doctor`)
- Comprehensive diagnostics bundle
- Log collection procedures
- Configuration sanitization (for sharing)
- Database inspection queries
- Transport-specific diagnostics

**Includes:**
- Shell script for automated collection
- Checklist for manual collection
- Redaction guidance for sensitive data
- Storage requirements

---

## Files Changed

### Modified Files

| File | Changes |
|------|---------|
| `README.md` | Added new doc references, Support section, diagnostics collection guidance |
| `docs/ops/troubleshooting.md` | Expanded from basic guide to comprehensive 714-line resource |
| `docs/ops/first-10-minutes.md` | Rewritten with more detail and validation procedures |

### New Files Created

| File | Lines | Purpose |
|------|-------|---------|
| `docs/ops/DOCUMENTATION_AUDIT_REPORT.md` | This file | Project summary and audit results |
| `docs/ops/api-reference.md` | 1239 | Complete API endpoint documentation |
| `docs/ops/cli-reference.md` | ~800 | Complete CLI command reference |
| `docs/ops/control-plane.md` | ~500 | Control plane modes and actions |
| `docs/ops/runbooks.md` | ~900 | 11 operational runbooks |
| `docs/ops/incident-triage.md` | ~400 | Support escalation procedures |
| `docs/ops/glossary.md` | ~600 | 40+ term definitions |
| `docs/ops/support-matrix.md` | ~500 | Compatibility matrices |
| `docs/ops/known-issues.md` | ~400 | Documented issues with workarounds |
| `docs/ops/diagnostics-collection.md` | ~600 | Diagnostic gathering guide |

---

## Key Operator Guidance Now Covered

### What MEL Is/Is Not

**MEL IS:**
- A local-first ingest layer for Meshtastic traffic
- A persistence layer using SQLite with deterministic migrations
- An observability layer with CLI tools and HTTP API
- A tool that only documents support existing in code

**MEL IS NOT:**
- A BLE ingest solution (explicitly unsupported)
- An HTTP transport ingest solution (explicitly unsupported)
- A transmit/publish/routing layer
- A source of fake/placeholder data

### Supported Transports with Caveats

| Transport | Support Level | Caveats |
|-----------|--------------|---------|
| Serial direct-node | Supported for ingest | Requires local device access + `stty` |
| TCP direct-node | Supported for ingest | Must speak Meshtastic framing, not HTTP |
| MQTT | Supported for ingest | Subscribe only; does not publish |
| BLE | Explicitly unsupported | No production claim |
| HTTP | Explicitly unsupported | No live ingest path |

### Control Plane Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `advisory` | Observes but doesn't act | Default; safe for all deployments |
| `guarded_auto` | Executes safe actions only | Requires operator validation |
| `auto` | Executes all enabled actions | Not recommended without testing |

### Executable vs Advisory-Only Actions

**Executable Actions:**
- Transport restart/reconnect
- System log rotation
- Database vacuum

**Advisory-Only Actions:**
- Node flagging for review
- Configuration recommendations
- Manual intervention prompts

### Transport States

| State | Meaning |
|-------|---------|
| `disabled` | Transport not configured |
| `configured_not_attempted` | Configured but not started |
| `attempting` | Connection in progress |
| `configured_offline` | Connection failed |
| `connected_no_ingest` | Connected but no data flowing |
| `ingesting` | Normal operation |
| `historical_only` | No live connection, using stored data |
| `error` | Error state with details |

### Safety Checks for Automation

Before any automated action:
1. Mode check (advisory vs auto)
2. Action enablement check
3. Resource availability check
4. Conflict detection
5. Rollback capability verification

### Denial Reasons and Resolutions

| Reason | Cause | Resolution |
|--------|-------|------------|
| `mode_advisory` | Control plane in advisory mode | Change mode or execute manually |
| `action_disabled` | Action not enabled | Enable in configuration |
| `resource_unavailable` | Required resource busy | Wait and retry |
| `conflict_detected` | Conflicting operation in progress | Complete or cancel conflicting operation |

### Incident Triage Procedures

1. **Classify severity** using defined matrix
2. **Collect diagnostics** using `mel doctor`
3. **Follow runbook** for issue type
4. **Escalate** if unresolved within SLA
5. **Document** resolution for future reference

### Diagnostics Collection

Standard diagnostic bundle includes:
- `mel doctor` output
- `mel status` output
- Configuration (sanitized)
- Recent log excerpts
- Database integrity check
- Transport state summary

---

## Validation Results

### Docs Reference Exact API Endpoints

- All 32 API endpoints documented with actual paths
- Request/response examples validated against code
- Query parameters match implementation
- Error codes documented from actual error handling

### Docs Reference Exact CLI Commands

- All 19 CLI commands documented
- Flag names and defaults match code
- Exit codes documented from main.go
- Examples tested against actual binary

### Examples Derived from Actual Code

- Configuration examples match example configs
- Output examples match actual tool output
- Error messages match error handling code
- State transitions match state machine logic

### No Aspirational Features Documented

- BLE support explicitly marked as unsupported
- HTTP transport explicitly marked as unsupported
- Metrics endpoint noted as reserved/no-op until implemented
- Restore documented as dry-run only

### All Limitations Explicitly Stated

Every limitation from `docs/ops/known-limitations.md`:
- BLE transport explicitly unsupported
- HTTP transport explicitly unsupported
- MEL does not send packets or administer radios
- No hardware validation claims for container environment
- No "100% reliability" claim
- Telemetry stored as raw bytes until full schema vendored
- Hybrid deployments require operator validation

---

## Remaining Gaps

### No Critical Gaps

All operational capabilities are now documented:
- Installation covered
- Configuration covered
- Operation covered
- Troubleshooting covered
- API reference complete
- CLI reference complete

### Future Documentation Work

The following areas may benefit from future documentation as features evolve:

1. **Multi-node federation** - When multi-node coordination features are implemented
2. **Advanced telemetry parsing** - When full telemetry protobuf schema is vendored
3. **Cloud deployment guides** - When specific cloud platform support is verified
4. **Hardware-specific tuning** - When validated on specific radio hardware

These are intentionally **not documented** as they do not yet exist in the codebase.

---

## Conclusion

### Documentation Now Accurately Reflects Code Truth

The Documentation Alignment Project has successfully:

1. **Validated existing documentation** against actual code
2. **Created comprehensive new documentation** for all operational needs
3. **Eliminated aspirational claims** from operator-facing docs
4. **Documented all limitations** with clear caveats
5. **Provided practical guidance** for installation, operation, and recovery

### Operators Can Now:

- **Install MEL** using clear, validated procedures
- **Configure MEL** with accurate configuration reference
- **Operate MEL** with confidence in documented behavior
- **Debug MEL** using comprehensive troubleshooting guides
- **Recover MEL** using step-by-step runbooks
- **Integrate MEL** using complete API and CLI references

### Without Repo Archaeology

Every operator question now has a documented answer:
- "What transports are supported?" → `docs/ops/transport-matrix.md`
- "How do I troubleshoot X?" → `docs/ops/troubleshooting.md`
- "What's the API for Y?" → `docs/ops/api-reference.md`
- "How do I recover from Z?" → `docs/ops/runbooks.md`

### Project Status: COMPLETE

The documentation now serves as a **source of truth** that aligns with code reality, enabling operators to deploy and operate MEL with confidence.

---

## Appendix: Document Inventory

### Quick Reference Card

| Need | Document |
|------|----------|
| First time setup | `docs/ops/first-10-minutes.md` |
| Configuration help | `docs/ops/configuration.md` |
| Something broken | `docs/ops/troubleshooting.md` |
| API integration | `docs/ops/api-reference.md` |
| CLI usage | `docs/ops/cli-reference.md` |
| Control plane | `docs/ops/control-plane.md` |
| Operational procedures | `docs/ops/runbooks.md` |
| Incident response | `docs/ops/incident-triage.md` |
| Terminology | `docs/ops/glossary.md` |
| Platform support | `docs/ops/support-matrix.md` |
| Known issues | `docs/ops/known-issues.md` |
| Diagnostics | `docs/ops/diagnostics-collection.md` |
| Transport details | `docs/ops/transport-matrix.md` |
| Limitations | `docs/ops/known-limitations.md` |

---

*Report generated: March 20, 2026*  
*Project lead: Documentation Truth Initiative*  
*Validation method: Code-to-doc trace verification*
