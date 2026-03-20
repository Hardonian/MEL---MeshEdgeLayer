# MEL Production Hardening: Phase 0 Audit and Implementation Plan

**Project**: MEL (MeshEdgeLayer)  
**Objective**: Transform from "technically impressive backend" to "field-deployable, operator-trustworthy product"  
**Date**: 2026-03-20  
**Auditor**: Architect Mode

---

## EXECUTIVE SUMMARY

MEL has a solid technical foundation with:
- Multi-transport ingest (serial, TCP, MQTT)
- Structured persistence (SQLite with migrations)
- Control plane with policy guardrails
- Existing React frontend with TypeScript
- CLI tooling for operators
- Privacy audit capabilities
- Retention and backup systems

**Critical Gap**: The system lacks production-credible operator visibility, troubleshooting workflows, and field diagnostics. An operator cannot currently:
- See a true fleet health overview with stale data detection
- Troubleshoot transport issues without CLI expertise
- Export a redacted support bundle from the UI
- View node-level diagnostics in the console
- Understand why actions were denied

---

## A. EXISTING SURFACES (Current Reality)

### A.1 Backend API Routes

| Endpoint | Purpose | Status |
|----------|---------|--------|
| `GET /healthz` | Basic liveness | Implemented |
| `GET /readyz` | Readiness with transport health | Implemented |
| `GET /metrics` | Prometheus-compatible metrics | Implemented |
| `GET /api/v1/status` | Full system status | Implemented |
| `GET /api/v1/nodes` | Node list | Implemented |
| `GET /api/v1/messages` | Recent messages | Implemented |
| `GET /api/v1/transports` | Transport status | Implemented |
| `GET /api/v1/dead-letters` | Dead letter queue | Implemented |
| `GET /api/v1/events` | Audit logs | Implemented |
| `GET /api/v1/policy/explain` | Policy recommendations | Implemented |
| `GET /api/v1/control/status` | Control plane status | Implemented |
| `GET /api/v1/control/history` | Action history | Implemented |

**Gap**: No `/api/v1/diagnostics` endpoint for structured troubleshooting data.

### A.2 CLI Commands

| Command | Purpose |
|---------|---------|
| `mel doctor` | Comprehensive diagnostics |
| `mel status` | System status |
| `mel panel` | Operator-facing status panel |
| `mel config validate` | Config validation |
| `mel nodes` | Node list |
| `mel node inspect <id>` | Node detail |
| `mel transports list` | Transport status |
| `mel inspect transport/mesh` | Deep inspection |
| `mel control status/history` | Control plane |
| `mel export` | Data export |
| `mel privacy audit` | Privacy findings |
| `mel backup create/restore` | Backup operations |

### A.3 Frontend Pages

| Route | Page | Current State |
|-------|------|---------------|
| `/` | Dashboard | Basic stats, privacy alerts, transport status |
| `/status` | Status | Transport health, system metrics |
| `/nodes` | Nodes | Node list, click for detail |
| `/nodes/:nodeId` | Node Detail | Basic node info (same component) |
| `/messages` | Messages | Message list |
| `/privacy` | Privacy | Privacy findings |
| `/dead-letters` | Dead Letters | Dead letter queue |
| `/recommendations` | Recommendations | Policy recommendations |
| `/events` | Events | Audit log |
| `/settings` | Settings | Config display |

**Gaps**:
- No dedicated Diagnostics/Troubleshooting page
- No Topology/Mesh visualization
- No Support Bundle export UI
- Node detail lacks deep diagnostics
- No Control Action review page

### A.4 Database Schema

| Table | Purpose |
|-------|---------|
| `messages` | Ingested mesh messages |
| `nodes` | Node registry |
| `telemetry_samples` | Telemetry data |
| `audit_logs` | System events |
| `dead_letters` | Failed ingest |
| `config_apply_history` | Config changes |
| `transport_runtime_status` | Transport state |
| `transport_runtime_evidence` | Transport metrics |
| `control_actions` | Control actions |
| `control_decisions` | Policy decisions |

### A.5 Configuration Model

```json
{
  "bind": { "api", "metrics", "allow_remote" },
  "auth": { "enabled", "session_secret", "ui_user", "ui_password" },
  "storage": { "data_dir", "database_path", "encryption_key_env" },
  "logging": { "level", "format" },
  "retention": { "messages_days", "telemetry_days", "audit_days" },
  "privacy": { "store_precise_positions", "redact_exports", "trust_list" },
  "transports": [ { "name", "type", "enabled", "source", ... } ],
  "features": { "web_ui", "metrics", "ble_experimental" },
  "control": { "mode", "policy_rules", ... },
  "rate_limits": { "http_rps", "transport_reconnect_seconds" }
}
```

### A.6 Control System

- **Modes**: `disabled`, `advisory`, `guarded_auto`
- **Actions**: Transport restart, resubscribe, backoff adjust, suppression, health recheck
- **Lifecycle**: `pending` → `running` → `completed`/`recovered`
- **Denial Codes**: Policy, mode, cooldown, missing actuator, irreversible, etc.

---

## B. ROOT OPERATIONAL GAPS

### B.1 Critical Gaps (Blocking Production)

| Gap | Impact | Severity |
|-----|--------|----------|
| No structured diagnostics API/UI | Operators cannot troubleshoot without CLI | Critical |
| No support bundle export UI | Cannot collect field evidence | Critical |
| Dashboard lacks stale data detection | Operators see false "healthy" states | Critical |
| No node-level diagnostics view | Cannot drill into node issues | Critical |
| No action denial explanation UI | Operators don't know why actions blocked | High |
| No transport reconnect pattern visibility | Hard to diagnose connectivity issues | High |
| No topology/mesh visualization | Cannot see mesh structure | Medium |

### B.2 Security/Privacy Gaps

| Gap | Impact |
|-----|--------|
| No explicit secret redaction in UI exports | Secrets may leak in screenshots/shares |
| No config validation warnings in UI | Operators unaware of unsafe configs |
| Control endpoints may lack input validation | Potential injection risk |

### B.3 Deployment/Runtime Gaps

| Gap | Impact |
|-----|--------|
| No startup validation of dangerous config combos | May start in unsafe state |
| No degraded mode for partial failures | Hard failure on non-critical issues |
| No disk/storage health visibility | Silent data loss risk |

---

## C. IMPLEMENTATION PLAN

### Phase 1: Backend Diagnostics System

**Goal**: Create structured diagnostics that surface real operational issues.

**Deliverables**:
1. `internal/diagnostics/` package with structured diagnostic checks
2. `GET /api/v1/diagnostics` endpoint
3. CLI `mel diagnostics` command
4. Diagnostic checks for:
   - Transport connectivity issues
   - MQTT broker session stability
   - Stale mesh snapshots (nodes silent > threshold)
   - Dead letter accumulation
   - DB write/read issues
   - Config validation errors
   - Storage pressure
   - Control mode without safeguards
   - Unsupported transport configurations

**Diagnostic Structure**:
```go
type Diagnostic struct {
    Code            string   // machine-readable: "transport_mqtt_stale_session"
    Severity        string   // "critical", "warning", "info"
    Component       string   // "transport", "database", "config", "mesh"
    Title           string   // human-readable title
    Explanation     string   // what is wrong
    LikelyCauses    []string // possible root causes
    RecommendedSteps []string // what to do
    Evidence        map[string]any // supporting data
    CanAutoRecover  bool
    OperatorActionRequired bool
}
```

### Phase 2: Support Bundle System

**Goal**: Enable operators to export redacted diagnostic bundles.

**Deliverables**:
1. `internal/supportbundle/` package
2. `GET /api/v1/support-bundle` endpoint (generates on-demand)
3. CLI `mel export support-bundle` command
4. Bundle contents:
   - Version/build metadata
   - Redacted config summary (secrets removed)
   - Transport state snapshot
   - Recent alerts/incidents
   - Diagnostics report
   - Node health summary
   - Recent control actions
   - Retention/DB summary
5. Redaction rules for secrets, tokens, passwords, precise positions

### Phase 3: Frontend Diagnostics Page

**Goal**: Real troubleshooting UI for operators.

**Deliverables**:
1. New route `/diagnostics`
2. `Diagnostics.tsx` page component
3. Grouped by severity/component
4. Expandable diagnostic cards with evidence
5. Direct links to affected resources (transport, node)
6. "Export Support Bundle" button

### Phase 4: Enhanced Dashboard

**Goal**: True fleet overview with stale data detection.

**Deliverables**:
1. Dashboard enhancements:
   - Snapshot age indicator with freshness warning
   - Stale node count (last seen > threshold)
   - Active alerts count with severity breakdown
   - Risk posture summary
   - Recent control actions (last 5)
   - Dead letter accumulation indicator
2. Backend: `GET /api/v1/overview` endpoint aggregating fleet state

### Phase 5: Node Detail Enhancement

**Goal**: Per-node diagnostics and action visibility.

**Deliverables**:
1. Enhanced `/nodes/:nodeId` route:
   - Node identity and metadata
   - Last seen with freshness indicator
   - Recent anomalies (if any)
   - Health explanation
   - Action history for this node
   - Command eligibility status
   - Evidence snippets
   - Drift/stale flags
2. Backend: `GET /api/v1/nodes/:id/detail` endpoint

### Phase 6: Transport Operations Page

**Goal**: Deep transport visibility for troubleshooting.

**Deliverables**:
1. New route `/transports/:name`
2. Transport detail view:
   - Runtime state with explanation
   - Reconnect pattern graph/text
   - Timeout streaks
   - Dead letter events
   - Recent errors with context
   - Session/broker health
3. Backend: Enhanced transport evidence in existing endpoints

### Phase 7: Control Actions Review Page

**Goal**: Visibility into autonomous control decisions.

**Deliverables**:
1. New route `/control-actions`
2. Control actions view:
   - Recent actions with status
   - Denied actions with reasons
   - Safety guard that applied
   - Rollback/compensating actions
3. Backend: `GET /api/v1/control/actions` endpoint

### Phase 8: Startup Validation & Degraded Mode

**Goal**: Safe startup with clear degradation.

**Deliverables**:
1. Startup validation:
   - Required config presence
   - Conflicting feature flags
   - Incompatible transport modes
   - Missing storage paths
   - Invalid retention settings
   - Unsafe control-mode settings
2. Degraded mode behavior:
   - Mark system unhealthy when critical subsystems fail
   - Continue operating with partial capability
   - Emit clear diagnostics
3. `GET /healthz` and `GET /readyz` hardening

### Phase 9: Security & Redaction Hardening

**Goal**: Production-credible security boundaries.

**Deliverables**:
1. Secret redaction in all exports (bundle, API responses)
2. Config validation warnings for unsafe settings
3. Input validation on all write endpoints
4. Audit logging for sensitive operations
5. Privacy findings for security issues

### Phase 10: Documentation & Runbooks

**Goal**: First-operator readiness.

**Deliverables**:
1. `docs/ops/deployment-guide.md`
2. `docs/ops/operator-guide.md`
3. `docs/ops/troubleshooting-guide.md`
4. `docs/ops/security-privacy-notes.md`
5. Updated known limitations

---

## D. VERIFICATION CHECKLIST

### Backend Verification
- [ ] `mel diagnostics` returns structured findings
- [ ] `mel export support-bundle` creates redacted bundle
- [ ] `GET /api/v1/diagnostics` returns JSON diagnostics
- [ ] `GET /api/v1/support-bundle` triggers download
- [ ] Startup validation prevents dangerous configs
- [ ] Degraded mode works when DB is unavailable
- [ ] All endpoints handle empty/partial data gracefully

### Frontend Verification
- [ ] `/diagnostics` page loads with real data
- [ ] Dashboard shows snapshot age and stale indicators
- [ ] Node detail shows diagnostics and action history
- [ ] Transport detail shows reconnect patterns
- [ ] Support bundle can be exported from UI
- [ ] All pages handle backend unavailable state
- [ ] No blank screens on empty data

### Integration Verification
- [ ] End-to-end: Start → Ingest → View → Diagnose → Export
- [ ] Degraded behavior: Partial failure → Graceful UI
- [ ] Security: Secrets not in exports
- [ ] Docs match actual behavior

---

## E. ARCHITECTURE DECISIONS

1. **Extension-First**: Build on existing patterns in `internal/db/`, `internal/control/`
2. **Shared Domain Models**: Diagnostics use same types in CLI, API, and UI
3. **Redaction as Utility**: Central `internal/redaction/` package
4. **Freshness Metadata**: All data responses include `generated_at` timestamp
5. **Graceful Degradation**: UI renders partial data with clear "incomplete" indicators
6. **No Mock Data**: All UI states reflect real backend conditions

---

## F. FILES TO CREATE/MODIFY

### New Backend Files
```
internal/diagnostics/
  ├── diagnostics.go          # Core diagnostic engine
  ├── checks.go               # Individual diagnostic checks
  ├── types.go                # Diagnostic structs
  └── diagnostics_test.go

internal/supportbundle/
  ├── bundle.go               # Bundle generation
  ├── redaction.go            # Secret redaction
  └── bundle_test.go

internal/redaction/
  └── redaction.go            # Shared redaction utilities
```

### Modified Backend Files
```
cmd/mel/main.go               # Add diagnostics and bundle CLI commands
internal/db/db.go             # Add diagnostic queries
```

### New Frontend Files
```
frontend/src/pages/
  ├── Diagnostics.tsx         # Troubleshooting page
  ├── TransportDetail.tsx     # Transport deep-dive
  └── ControlActions.tsx      # Control action review

frontend/src/components/
  └── diagnostics/
      ├── DiagnosticCard.tsx
      ├── DiagnosticList.tsx
      └── SupportBundleExport.tsx
```

### Modified Frontend Files
```
frontend/src/App.tsx          # Add new routes
frontend/src/hooks/useApi.tsx # Add diagnostics hooks
frontend/src/pages/Dashboard.tsx    # Enhance with freshness
frontend/src/pages/Nodes.tsx        # Enhance with detail view
```

### New Documentation
```
docs/ops/deployment-guide.md
docs/ops/operator-guide.md
docs/ops/troubleshooting-guide.md
docs/ops/security-privacy-notes.md
```

---

## G. SUCCESS CRITERIA

This task is complete when:

1. **Operator Can Assess Health**: Dashboard clearly shows system health, stale data, and active issues
2. **Operator Can Troubleshoot**: Diagnostics page surfaces real issues with actionable guidance
3. **Operator Can Export Evidence**: Support bundle export works from UI, properly redacted
4. **Operator Can Inspect Nodes**: Node detail shows real diagnostics and action history
5. **System Starts Safely**: Validation prevents dangerous configs, degraded mode works
6. **No False Green States**: All indicators reflect real backend state
7. **Docs Enable First Use**: New operator can deploy, operate, and troubleshoot without reading source

---

## H. INTENTIONALLY NOT IN SCOPE

To avoid over-expansion, these are explicitly out of scope:

1. **Geospatial Map**: No real coordinates available; topology view only if mesh data supports it
2. **Auth System Redesign**: Harden existing auth, don't redesign
3. **New Transports**: No BLE, HTTP, or new transport types
4. **Mobile App**: Web UI only
5. **Cloud Integration**: No external telemetry sinks
6. **Historical Analytics**: No trend analysis beyond existing retention

---

**END OF AUDIT AND PLAN**
