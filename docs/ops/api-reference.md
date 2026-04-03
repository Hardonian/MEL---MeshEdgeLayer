# MEL API Reference

This document provides a comprehensive reference for the MEL (MeshEdgeLayer) HTTP API. All endpoints return JSON responses unless otherwise noted.

**Base URL:** Determined by `bind.api` configuration (default: `localhost:8080`)

## Authentication

When `auth.enabled` is `true` in the configuration, all endpoints require HTTP Basic Authentication:
- Username: `auth.ui_user`
- Password: `auth.ui_password`

Unauthenticated requests receive:
```json
{
  "error": {
    "code": "auth_required",
    "message": "authentication is required for this MEL endpoint"
  }
}
```

---

## Health & Readiness Endpoints

### Semantics (read this first)

| Surface | What it proves |
|--------|----------------|
| `GET /healthz` | **Liveness only** — the HTTP process responds. |
| `GET /readyz` and `GET /api/v1/readyz` | **Readiness** — status snapshot built; HTTP **200** when at least one **enabled** transport is in `ingesting`, or when **no transports are enabled** (explicit idle). HTTP **503** when snapshot assembly fails or enabled transports exist but none are ingesting. |
| `GET /api/v1/status` | **Authoritative transport/system truth** — full `Snapshot` plus operator `panel`. |
| `mel doctor` / `mel preflight` | **Host-level checks** — config file mode, DB/schema, audit chain, serial/TCP reachability, etc. Preflight adds optional `GET /healthz` against `bind.api`. |

Support bundles and exports may still contain topology samples, message excerpts, and broker **endpoints** (passwords are redacted in `config`). Review before sharing.

### GET /healthz
Basic **liveness** check — does not prove ingest, MQTT, or serial.

**Response:**
```json
{
  "ok": true
}
```

---

### GET /readyz
Readiness with the same JSON contract as `GET /api/v1/readyz` (see below).

---

### GET /api/v1/readyz
Versioned readiness for probes and automation. **Semantically identical** to `GET /readyz` (same handler).

**HTTP status:** `200` when `ready` is true; `503` when `ready` is false or the status snapshot cannot be built.

**Example (ready, ingest proven):**
```json
{
  "api_version": "v1",
  "ready": true,
  "status": "ready",
  "reason_codes": [],
  "summary": "Live ingest is confirmed by SQLite writes.",
  "checked_at": "2026-03-23T12:00:00Z",
  "process_ready": true,
  "ingest_ready": true,
  "stale_ingest_evidence": false,
  "snapshot_generated_at": "2026-03-23T12:00:00Z",
  "schema_version": "…",
  "operator_state": "ready",
  "mesh_state": "healthy",
  "components": [
    {"name": "process", "state": "ok", "detail": "HTTP handler responding"},
    {"name": "snapshot", "state": "ok", "detail": "status snapshot assembled"},
    {"name": "ingest", "state": "ok", "detail": "at least one transport in ingesting state"}
  ],
  "transports": [],
  "operator_next_steps": []
}
```

**Stable fields:**
- `api_version` — contract version (`v1`).
- `ready` / `status` — `ready` boolean and `status` of `ready` or `not_ready`.
- `reason_codes` — machine codes (e.g. `INGEST_NOT_PROVEN`, `TRANSPORT_IDLE`, `STALE_INGEST_EVIDENCE`, `SNAPSHOT_UNAVAILABLE`, `DEGRADED_TRANSPORT`, `MESH_*`).
- `summary` — short operator-facing explanation.
- `checked_at` — RFC3339 when the handler evaluated readiness.
- `ingest_ready` — at least one enabled transport in `ingesting` effective state.
- `stale_ingest_evidence` — last persisted ingest older than the server threshold while still ingesting.
- `components` — bounded sub-checks (`process`, `snapshot`, `ingest`, `mesh`).
- `transports` — same transport reports as the status snapshot (for evidence; use `/api/v1/status` for the full document).
- `operator_next_steps` — actionable strings including CLI hints.

**Error Responses:**
- `503` — `error_class`: `not_ready` (ingest not proven) or `snapshot_unavailable` (DB/migration/evidence assembly failure).

---

### GET /api/v1/support-bundle
**Requires** capability `export_support_bundle` (same as manifest export). Returns `application/zip` containing:
- `bundle.json` — redacted support payload (see `internal/support`).
- `doctor.json` — structured output aligned with `mel doctor`, with bundle-specific redaction (config path omitted; `config_inspect` fingerprint only).
- `imported_evidence.json` and `remote_evidence_export.json` when offline remote evidence exists.

**Important posture notes:**
- Support bundles preserve remote evidence as **offline imported data**, not live federation state.
- `remote_evidence_export.json` is re-importable as `mel_remote_evidence_batch`.
- Authenticity of imported origin is not cryptographically verified by core MEL.

---

## Metrics Endpoints

### GET /metrics
JSON metrics endpoint (not Prometheus text exposition format).

**Response:**
```json
{
  "generated_at": "2026-03-20T12:00:00Z",
  "window_seconds": 300,
  "total_messages": 15432,
  "last_ingest_time": "2026-03-20T11:59:55Z",
  "transport_metrics": [...],
  "ingest_rate_per_sec": {
    "mqtt-primary": 2.5,
    "serial-local": 0.1
  },
  "dead_letters_total": 3,
  "control_metrics": {
    "decisions_total": 150,
    "executions_total": 45,
    "denials_total": 105,
    "cooldown_denials": 20,
    "override_denials": 5,
    "missing_actuator_denials": 10,
    "active_actions": 2,
    "queue_depth": 1,
    "execution_latency_seconds": 1.25,
    "denials_by_reason": {
      "policy": 30,
      "mode": 15,
      "override": 5,
      "low_confidence": 10,
      "transient": 8,
      "cooldown": 20,
      "budget": 5,
      "missing_actuator": 10,
      "unknown_blast_radius": 2,
      "no_alternate_path": 3,
      "irreversible": 7,
      "conflict": 8,
      "attribution_weak": 4
    }
  }
}
```

**Note:** `control_metrics` is only present when database is available.

---

## v0 Compatibility API

These endpoints are maintained for backward compatibility. New code should use v1 endpoints.

### GET /api/status
Legacy status endpoint (identical to `/api/v1/status`).

### GET /api/nodes
Legacy nodes endpoint (identical to `/api/v1/nodes`).

### GET /api/transports
Legacy transports endpoint (identical to `/api/v1/transports`).

### GET /api/privacy/audit
Legacy privacy audit endpoint (identical to `/api/v1/privacy/audit`).

### GET /api/recommendations
Legacy recommendations endpoint (identical to `/api/v1/policy/explain`).

### GET /api/logs
Legacy audit logs endpoint (identical to `/api/v1/events`).

### GET /api/dead-letters
Legacy dead letters endpoint (identical to `/api/v1/dead-letters`).

---

## v1 API

### GET /api/v1/fleet/truth
Returns the instance-scoped fleet truth posture used to interpret local vs imported evidence.

This endpoint is intentionally bounded:
- it describes this instance's scope and truth boundary,
- it does not claim global fleet completeness,
- it does not imply live federation authority.

---

### POST /api/v1/fleet/remote-evidence
Offline remote evidence ingest. The request body must be raw JSON matching one of:

- `mel_remote_evidence_bundle`
- `mel_remote_evidence_batch`

**Behavior:**
- validates the payload structurally,
- persists an import batch audit row,
- persists per-item imported evidence rows,
- materializes local timeline audit events,
- never turns imported evidence into a remote control channel.

**Successful response fields:**
- `status`
- `batch_id`
- `validation`
- `input_kind`
- `items`
- `item_inspections`
- `accepted_count`
- `rejected_count`

**Notes:**
- accepted imports are still historical/offline and authenticity-unverified by default,
- partial success is explicit through `accepted_partial_bundle`,
- generic MEL export bundles are not importable here unless wrapped in the canonical remote evidence contract.

---

### GET /api/v1/fleet/remote-evidence
Lists imported remote evidence item rows plus normalized summaries.

**Query params:**
- `limit`
- `batch_id` — limit list to one persisted import batch

The response preserves:
- local/imported distinction,
- validation posture,
- timing posture,
- related-evidence/merge summary counts.

---

### GET /api/v1/fleet/remote-evidence/{id}
Returns one imported evidence row plus full inspection detail.

The inspection includes:
- source batch/path context,
- claimed origin vs local provenance,
- timing posture,
- merge inspection,
- related evidence analysis,
- remaining unknowns.

---

### GET /api/v1/fleet/imports
Lists persisted remote import batches.

These are **offline audit containers**, not live peer state. The response includes:
- raw batch rows,
- normalized summaries,
- truth posture note explaining the read-only/offline boundary.

---

### GET /api/v1/fleet/imports/{id}
Returns one import batch with:
- the batch audit row,
- imported item rows in that batch,
- batch inspection,
- per-item inspections.

Use this when an operator needs to answer:
- what was imported,
- from where it claimed to come,
- what MEL accepted or rejected,
- what remains unverified or partial.

---

### GET /api/v1/fleet/merge-explain
Explains MEL's merge/dedupe classification logic for two candidate keys.

This endpoint is intentionally narrow:
- it explains structural dedupe posture,
- it does not prove flooding, congestion, route certainty, or RF coverage.

---

### GET /api/v1/timeline
Unified instance-local timeline. Imported remote evidence appears as explicit remote-prefixed event types such as:

- `remote_import_batch`
- `remote_evidence_import_item`
- `remote_event_materialized`

**Query params:**
- `start`
- `end`
- `limit`
- `event_type`
- `scope_posture`

**Ordering note:** the response is instance-local or import-local only. No global total order is implied.

---

### GET /api/v1/timeline/{id}
Returns one timeline event plus its ordering posture note.

For imported remote evidence events, inspect `details_json` for:
- validation posture,
- provenance,
- timing basis,
- merge inspection,
- canonical evidence and remote event envelopes.

---

### GET /api/v1/investigations/cases
Returns bounded investigation cases. Each case now includes a normalized timing posture so operators can see whether case chronology is exact or best-effort before opening detail.

---

### GET /api/v1/investigations/cases/{id}
Returns full investigation case detail:

- current case posture,
- linked findings, evidence gaps, and recommendations,
- `linked_events` for exact raw canonical event linkage,
- `evolution` entries explaining how the current case posture was shaped,
- normalized case timing posture.

Related events contribute context to the case. They do not automatically prove causality or root cause.

---

### GET /api/v1/investigations/cases/{id}/timeline
Returns the case's temporal reconstruction only:

- `timing`
- `linked_events`
- `evolution`

Use this when the operator question is:

- what changed for this case,
- which raw events are exact versus case-level reconstruction,
- which timing caveats bound the sequence,
- how imports, validation, merge, or freshness posture shaped the current case.

---

### GET /api/v1/status
Full system status with persisted summary.

**Response:**
```json
{
  "snapshot": {
    "messages": 15432,
    "nodes": [
      {
        "num": 12345,
        "id": "!abcd1234",
        "long_name": "Base Station",
        "short_name": "BS",
        "last_seen": "2026-03-20T11:59:00Z",
        "gateway_id": "!gateway01",
        "lat_redacted": false,
        "lon_redacted": false,
        "altitude": 150,
        "last_snr": 12.5,
        "last_rssi": -85
      }
    ]
  },
  "persisted_summary": {
    "messages": "15432",
    "nodes": "45",
    "last_ingest": "2026-03-20T11:59:55Z"
  },
  "status": {
    "generated_at": "2026-03-20T12:00:00Z",
    "bind": "localhost:8080",
    "bind_local_only": true,
    "schema_version": "1.2.3",
    "configured_transport_modes": ["mqtt", "serial"],
    "messages": 15432,
    "nodes": 45,
    "last_successful_ingest": "2026-03-20T11:59:55Z",
    "transports": [...],
    "recent_transport_incidents": [...],
    "active_transport_alerts": [...],
    "mesh": {...}
  },
  "panel": {
    "generated_at": "2026-03-20T12:00:00Z",
    "operator_state": "ready",
    "summary": "Live ingest is confirmed by SQLite writes.",
    "short_commands": ["S=Status", "T=Transports", "N=Nodes", "R=Replay", "D=Doctor"],
    "web_hints": [...],
    "device_menu": [...],
    "transports": [...]
  },
  "privacy_summary": {
    "critical": 0,
    "high": 1,
    "medium": 2,
    "low": 0,
    "info": 1
  },
  "bind_local_default": true
}
```

**Error Responses:**
- `500 Internal Server Error` - Status collection failed

---

### GET /api/v1/nodes
List all observed nodes with message counts.

**Response:**
```json
{
  "nodes": [
    {
      "node_num": 12345,
      "node_id": "!abcd1234",
      "long_name": "Base Station",
      "short_name": "BS",
      "last_seen": "2026-03-20T11:59:00Z",
      "last_gateway_id": "!gateway01",
      "lat_redacted": 0,
      "lon_redacted": 0,
      "altitude": 150,
      "last_snr": 12.5,
      "last_rssi": -85,
      "message_count": 342
    }
  ]
}
```

**Error Responses:**
- `500 Internal Server Error` - Database query failed

---

### GET /api/v1/node/{id}
Get detailed information for a specific node.

**Path Parameters:**
- `id` - Node number (integer) or node ID (string like `!abcd1234`)

**Response:**
```json
{
  "node": {
    "node_num": 12345,
    "node_id": "!abcd1234",
    "long_name": "Base Station",
    "short_name": "BS",
    "last_seen": "2026-03-20T11:59:00Z",
    "last_gateway_id": "!gateway01",
    "lat_redacted": 0,
    "lon_redacted": 0,
    "altitude": 150,
    "last_snr": 12.5,
    "last_rssi": -85,
    "message_count": 342
  }
}
```

**Error Responses:**
- `400 Bad Request` - Missing node identifier
- `404 Not Found` - Node not found in local observations
- `500 Internal Server Error` - Database query failed

---

### GET /api/v1/transports
List all configured transports with health status.

**Response:**
```json
{
  "transports": [
    {
      "name": "mqtt-primary",
      "type": "mqtt",
      "source": "broker.example.com:1883",
      "enabled": true,
      "effective_state": "live",
      "runtime_state": "live",
      "persisted_state": "live",
      "status_scope": "runtime+persisted",
      "detail": "Live ingest is confirmed by successful database writes.",
      "guidance": "Live ingest is confirmed by successful database writes.",
      "last_attempt_at": "2026-03-20T11:55:00Z",
      "last_ingest_at": "2026-03-20T11:59:55Z",
      "last_heartbeat_at": "2026-03-20T11:59:55Z",
      "last_error": "",
      "last_failure_at": "",
      "episode_id": "",
      "failure_count": 0,
      "observation_drops": 0,
      "total_messages": 15432,
      "persisted_messages": 15432,
      "error_count": 0,
      "dropped_count": 0,
      "reconnect_attempts": 0,
      "consecutive_timeouts": 0,
      "dead_letters": 0,
      "retry_status": "live evidence present; no retry pending",
      "capabilities": {
        "ingest_supported": true,
        "send_supported": true,
        "metadata_fetch_supported": false,
        "node_fetch_supported": false,
        "health_supported": true,
        "config_apply_supported": false,
        "implementation_status": "stable",
        "notes": ""
      },
      "health": {
        "transport_name": "mqtt-primary",
        "transport_type": "mqtt",
        "score": 100,
        "state": "healthy",
        "last_evaluated_at": "2026-03-20T12:00:00Z",
        "primary_reason": "",
        "signals": {
          "recent_failures": 0,
          "dead_letter_count": 0,
          "retry_count": 0,
          "last_heartbeat_delta_seconds": 5,
          "anomaly_rate": 0.0,
          "observation_drops": 0,
          "active_episode": false
        },
        "explanation": {...}
      },
      "active_alerts": [],
      "recent_anomalies": [],
      "failure_clusters": []
    }
  ],
  "configured_modes": ["mqtt", "serial"],
  "recent_transport_incidents": [...],
  "active_transport_alerts": [...]
}
```

**Transport States:**
- `disabled` - Transport is disabled in configuration
- `configured` / `configured_not_attempted` - Configured but never attempted
- `connecting` / `attempting` - Actively establishing connection
- `live` / `ingesting` - Active data flow confirmed
- `idle` / `connected_no_ingest` - Connected but no recent ingest
- `retrying` / `configured_offline` - Connection failed, backing off
- `failed` / `error` - Terminal failure state
- `historical_only` - Past messages exist but current live connectivity unproven

---

### GET /api/v1/transports/health
Transport health summary.

**Response:**
```json
{
  "transport_health": [
    {
      "transport_name": "mqtt-primary",
      "transport_type": "mqtt",
      "runtime_state": "live",
      "effective_state": "live",
      "health": {
        "score": 100,
        "state": "healthy",
        "primary_reason": "no dominant reason",
        "explanation": {...}
      },
      "active_alerts": [],
      "recent_anomalies": [],
      "failure_clusters": [],
      "last_failure_at": "",
      "episode_id": "",
      "failure_count": 0,
      "observation_drops": 0
    }
  ]
}
```

---

### GET /api/v1/transports/alerts
Active transport alerts.

**Response:**
```json
{
  "transport_alerts": [
    {
      "id": "alert-001",
      "transport_name": "mqtt-primary",
      "transport_type": "mqtt",
      "severity": "warn",
      "reason": "timeout_failure",
      "summary": "Connection timeout detected",
      "first_triggered_at": "2026-03-20T11:30:00Z",
      "last_updated_at": "2026-03-20T11:35:00Z",
      "active": true,
      "episode_id": "ep-001",
      "cluster_key": "cluster-001",
      "contributing_reasons": ["timeout_failure", "retry_threshold_exceeded"],
      "cluster_reference": "ref-001",
      "penalty_snapshot": [
        {
          "reason": "timeout_failure",
          "penalty": 15,
          "count": 3,
          "window": "5m"
        }
      ],
      "trigger_condition": "consecutive_timeouts > 3"
    }
  ]
}
```

---

### GET /api/v1/transports/anomalies
Recent transport anomalies.

**Response:**
```json
{
  "transport_anomalies": [
    {
      "transport_name": "mqtt-primary",
      "transport_type": "mqtt",
      "recent_anomalies": [
        {
          "transport_name": "mqtt-primary",
          "window": "5m",
          "counts_by_reason": {
            "timeout_failure": 3,
            "retry_threshold_exceeded": 1
          },
          "dead_letters": 0,
          "retry_events": 1,
          "anomaly_rate": 0.8,
          "observation_drops": 0,
          "active_episode_ids": ["ep-001"],
          "drop_causes": {}
        }
      ],
      "failure_clusters": [
        {
          "transport_name": "mqtt-primary",
          "transport_type": "mqtt",
          "reason": "timeout_failure",
          "count": 5,
          "first_seen": "2026-03-20T11:25:00Z",
          "last_seen": "2026-03-20T11:35:00Z",
          "severity": "warn",
          "episode_id": "ep-001",
          "includes_dead_letter": false,
          "includes_observation_drops": false,
          "cluster_key": "mqtt-primary|timeout_failure|ep-001"
        }
      ]
    }
  ]
}
```

---

### GET /api/v1/transports/health/history
Historical health snapshots for transports.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `transport` | string | "" (all) | Filter by transport name |
| `start` | string (RFC3339) | "" | Start time |
| `end` | string (RFC3339) | "" | End time |
| `limit` | integer | `cfg.intelligence.queries.default_limit` | Max results (capped by `max_limit`) |
| `offset` | integer | 0 | Pagination offset |

**Response:**
```json
{
  "history": [
    {
      "transport_name": "mqtt-primary",
      "transport_type": "mqtt",
      "health_score": 85,
      "health_state": "degraded",
      "primary_reason": "timeout_failure",
      "recorded_at": "2026-03-20T11:30:00Z",
      "episode_id": "ep-001",
      "failure_count": 3,
      "observation_drops": 0
    }
  ],
  "pagination": {
    "limit": 100,
    "offset": 0
  },
  "transport": "mqtt-primary",
  "start": "2026-03-20T10:00:00Z",
  "end": "2026-03-20T12:00:00Z"
}
```

---

### GET /api/v1/transports/alerts/history
Historical alerts for transports.

**Query Parameters:** Same as `/api/v1/transports/health/history`

**Response:**
```json
{
  "history": [
    {
      "id": "alert-001",
      "transport_name": "mqtt-primary",
      "transport_type": "mqtt",
      "severity": "warn",
      "reason": "timeout_failure",
      "summary": "Connection timeout detected",
      "first_triggered_at": "2026-03-20T11:30:00Z",
      "last_updated_at": "2026-03-20T11:35:00Z",
      "active": false,
      "episode_id": "ep-001",
      "cluster_key": "cluster-001",
      "contributing_reasons": ["timeout_failure"],
      "cluster_reference": "ref-001",
      "trigger_condition": "consecutive_timeouts > 3"
    }
  ],
  "pagination": {...},
  "transport": "mqtt-primary",
  "start": "2026-03-20T10:00:00Z",
  "end": "2026-03-20T12:00:00Z"
}
```

---

### GET /api/v1/transports/anomalies/history
Historical anomaly data for transports.

**Query Parameters:** Same as `/api/v1/transports/health/history`

**Response:**
```json
{
  "history": [
    {
      "transport_name": "mqtt-primary",
      "window_label": "5m",
      "recorded_at": "2026-03-20T11:30:00Z",
      "counts_by_reason": {
        "timeout_failure": 3
      },
      "dead_letter_count": 0,
      "retry_event_count": 1,
      "observation_drops": 0,
      "drop_causes": {}
    }
  ],
  "pagination": {...},
  "transport": "mqtt-primary",
  "start": "2026-03-20T10:00:00Z",
  "end": "2026-03-20T12:00:00Z"
}
```

---

### GET /api/v1/transports/inspect/{name}
Detailed drilldown for a specific transport.

**Path Parameters:**
- `name` - Transport name (exact match, case-sensitive)

**Response:**
```json
{
  "transport_name": "mqtt-primary",
  "transport_type": "mqtt",
  "health": {
    "transport_name": "mqtt-primary",
    "transport_type": "mqtt",
    "score": 85,
    "state": "degraded",
    "last_evaluated_at": "2026-03-20T12:00:00Z",
    "primary_reason": "timeout_failure",
    "signals": {...},
    "explanation": {...}
  },
  "health_explanation": {
    "transport_name": "mqtt-primary",
    "score": 85,
    "state": "degraded",
    "top_penalties": [...],
    "active_cluster_reason": "timeout_failure",
    "active_cluster_count": 5,
    "active_episode_id": "ep-001",
    "failure_count": 3,
    "observation_drops": 0,
    "dead_letter_count": 0,
    "recovery_blockers": ["runtime_state:retrying", "active_failure_episode:ep-001 (3 failures)"]
  },
  "recent_clusters": [...],
  "recent_alerts": [...],
  "anomaly_summary": [...],
  "last_incidents": [...],
  "episode_history": [...],
  "health_history": [...],
  "alert_history": [...],
  "anomaly_history": [...],
  "transport_connected": true
}
```

**Error Responses:**
- `400 Bad Request` - Missing transport name
- `404 Not Found` - Transport not found

---

### GET /api/v1/mesh
Current mesh state.

**Response:**
```json
{
  "mesh_health": {
    "score": 85,
    "state": "degraded",
    "degraded_segments": [...],
    "critical_segments": [...],
    "dominant_failure_reason": "timeout_failure"
  },
  "mesh_health_explanation": {
    "mesh_score": 85,
    "mesh_state": "degraded",
    "dominant_failure_reason": "timeout_failure",
    "affected_transports": ["mqtt-primary"],
    "affected_nodes": [],
    "top_penalties": [...],
    "active_clusters": [...],
    "active_alerts": [...],
    "degraded_segments": [...],
    "evidence_loss_summary": {
      "ingest_drops": 0,
      "observation_drops": 0,
      "bus_drops": 0
    },
    "recovery_blockers": [...]
  },
  "correlated_failures": [
    {
      "reason": "timeout_failure",
      "transports": ["mqtt-primary", "mqtt-backup"],
      "node_ids": [],
      "count": 10,
      "window": "5m",
      "severity": "warn",
      "explanation": "timeout_failure observed across 2 transports within 5m"
    }
  ],
  "degraded_segments": [...],
  "root_cause_analysis": {
    "primary_cause": "connectivity_issue",
    "confidence": "high",
    "supporting_evidence": [...],
    "explanation": "Timeout, retry-threshold, or heartbeat-loss evidence dominates across transports, which points to a connectivity issue."
  },
  "operator_recommendations": [
    {
      "action": "Check network connectivity",
      "reason": "Timeout, retry-threshold, or heartbeat-loss evidence is correlated across transports.",
      "confidence": "high",
      "related_transports": ["mqtt-primary", "mqtt-backup"],
      "related_segments": ["segment:timeout_failure:mesh/messages"]
    }
  ],
  "routing_recommendations": [
    {
      "action": "deprioritize_degraded_transport",
      "target_transport": "mqtt-primary",
      "reason": "mqtt-primary is degraded with score=85; keep routing advisory-only and visible to operators.",
      "confidence": "medium"
    }
  ],
  "active_alerts": [...],
  "recent_clusters": [...],
  "history_summary": {
    "health_points": 144,
    "alert_points": 12,
    "anomaly_points": 48,
    "retained_since": "2026-03-13T12:00:00Z",
    "retention_boundary": "bounded by intelligence.retention.health_snapshot_days=7 and intelligence.retention.health_snapshot_max_rows=10000"
  }
}
```

---

### GET /api/v1/mesh/inspect
Detailed mesh inspection with recommendations.

**Response:** Same structure as `/api/v1/mesh` (full MeshDrilldown)

---

### GET /api/v1/messages
List recent messages with optional filtering.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `node` | string | "" | Filter by node (number or ID) |
| `type` | string | "" | Filter by message type (searches payload_json) |
| `limit` | integer | 100 | Max results (max 500) |

**Response:**
```json
{
  "messages": [
    {
      "transport_name": "mqtt-primary",
      "packet_id": 12345,
      "from_node": 98765,
      "to_node": 12345,
      "portnum": 1,
      "payload_text": "Hello mesh!",
      "payload_json": "{\"message_type\":\"text\",\"text\":\"Hello mesh!\"}",
      "rx_time": "2026-03-20T11:59:55Z",
      "created_at": "2026-03-20T11:59:56Z"
    }
  ],
  "filters": {
    "limit": ["100"]
  }
}
```

---

### GET /api/v1/panel
Compact operator panel summary.

**Response:**
```json
{
  "generated_at": "2026-03-20T12:00:00Z",
  "operator_state": "ready",
  "summary": "Live ingest is confirmed by SQLite writes.",
  "short_commands": ["S=Status", "T=Transports", "N=Nodes", "R=Replay", "D=Doctor"],
  "web_hints": [
    "Open /api/v1/status for full transport truth.",
    "Open /api/v1/panel for compact operator state.",
    "Use the Web UI to verify live ingest versus historical-only evidence."
  ],
  "device_menu": [
    {
      "key": "A",
      "label": "State",
      "action": "Show operator state and overall ingest truth"
    },
    {
      "key": "B",
      "label": "Link",
      "action": "Cycle transport states and last errors"
    },
    {
      "key": "C",
      "label": "Msgs",
      "action": "Show persisted and runtime message counters"
    },
    {
      "key": "D",
      "label": "Retry",
      "action": "Show reconnect attempts and offline guidance"
    }
  ],
  "transports": [
    {
      "name": "mqtt-primary",
      "label": "M",
      "state": "live",
      "messages": 15432,
      "last_ingest": "2026-03-20T11:59:55Z",
      "detail": "Live ingest is confirmed by successful database writes.",
      "score": 100
    }
  ]
}
```

---

### GET /api/v1/privacy/audit
Privacy audit findings.

**Response:**
```json
{
  "findings": [
    {
      "id": "export-redaction-disabled",
      "severity": "medium",
      "message": "Exports are configured without redaction.",
      "remediation": "Enable privacy.redact_exports for operator-safe exports.",
      "evidence": ["privacy.redact_exports=false"]
    },
    {
      "id": "empty-trust-list",
      "severity": "info",
      "message": "No trust list is configured.",
      "remediation": "Leave it empty if intentional, or add known node IDs for stricter export and policy workflows.",
      "evidence": ["privacy.trust_list=[]"]
    }
  ],
  "summary": {
    "critical": 0,
    "high": 0,
    "medium": 1,
    "low": 0,
    "info": 1
  }
}
```

**Finding Severities:** `critical`, `high`, `medium`, `low`, `info`

---

### GET /api/v1/policy/explain
Policy recommendations.

**Response:**
```json
{
  "recommendations": [
    {
      "id": "disable-precise-position-storage",
      "summary": "Disable precise position storage unless your workflow truly requires it.",
      "severity": "high",
      "reason": "Position history can be sensitive personal data and is rarely required for local observability.",
      "evidence": ["privacy.store_precise_positions=true"],
      "remediation": "Turn off privacy.store_precise_positions or require operator approval plus storage key management."
    },
    {
      "id": "require-mqtt-encryption",
      "summary": "Require MQTT transport encryption or disable MQTT for privacy-sensitive deployments.",
      "severity": "high",
      "reason": "Broker hops can expose message content and metadata when transport encryption is not enforced.",
      "evidence": ["privacy.mqtt_encryption_required=false"],
      "remediation": "Keep MQTT local, add TLS via a local tunnel, or disable the transport."
    }
  ]
}
```

---

### GET /api/v1/events
Audit logs/events.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `transport` | string | "" | Filter by transport name |

**Response:**
```json
{
  "events": [
    {
      "category": "transport",
      "level": "warn",
      "message": "timeout_failure",
      "details_json": "{\"transport\":\"mqtt-primary\",\"type\":\"mqtt\",\"episode_id\":\"ep-001\"}",
      "created_at": "2026-03-20T11:30:00Z"
    }
  ]
}
```

---

### GET /api/v1/audit-logs
Alias for `/api/v1/events`.

---

### GET /api/v1/dead-letters
Dead letter queue entries.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `transport` | string | "" | Filter by transport name |

**Response:**
```json
{
  "dead_letters": [
    {
      "transport_name": "mqtt-primary",
      "transport_type": "mqtt",
      "topic": "mesh/messages",
      "reason": "retry_threshold_exceeded",
      "payload_hex": "080112...",
      "details_json": "{\"episode_id\":\"ep-001\",\"final\":true}",
      "created_at": "2026-03-20T11:25:00Z"
    }
  ]
}
```

---

### GET /api/v1/incidents
Recent transport incidents.

**Response:**
```json
{
  "recent_transport_incidents": [
    {
      "id": "inc-001",
      "transport_name": "mqtt-primary",
      "severity": "warn",
      "reason": "timeout_failure",
      "episode_id": "ep-001",
      "started_at": "2026-03-20T11:25:00Z",
      "ended_at": "2026-03-20T11:35:00Z",
      "duration_seconds": 600
    }
  ]
}
```

---

### GET /api/v1/incidents/{id}/proofpack

Assembles and returns an incident-scoped evidence proofpack for audit and export.

**Required capabilities:** `export_support_bundle` or `read_incidents`

**Query parameters:**
- `download=true` — sets `Content-Disposition: attachment` for browser download
  with filename `mel-proofpack-{incident-id}-{assembled-at}.json` (falls back to
  `...-snapshot.json` when assembly timestamp is unavailable)

**Response:** JSON proofpack (format_version `1.0.0`) containing:
- `assembly` — assembly metadata (who, when, instance, time window, item counts), including:
  - `action_outcome_snapshot_status`: `complete | partial | unavailable`
  - `action_outcome_snapshot_trace`: retrieval posture (`retrieval_status`, `status_reason`, `retrieval_error`, `retrieval_limited`, `signature_key_present`, `max_snapshots`)
- `incident` — full incident record at assembly time
- `linked_actions[]` — control actions linked via incident FK
- `timeline[]` — chronological events in the evidence window
- `transport_context[]` — transport health snapshots in the window
- `dead_letter_evidence[]` — dead letters in the window
- `operator_notes[]` — notes attached to the incident
- `audit_entries[]` — RBAC audit log entries for the incident
- `evidence_gaps[]` — explicit markers for missing or degraded evidence

**Error responses:**
- `404` — incident not found
- `503` — proofpack assembly not available (service not wired)
- `403` — export disabled by policy (`platform.retention.allow_export=false`)

See `docs/runbooks/proofpack-export.md` for full operational guide.

---

## Control Plane Endpoints

### GET /api/v1/control/status
Control plane status and configuration.

**Response:**
```json
{
  "mode": "advisory",
  "active_actions": [],
  "recent_actions": [],
  "pending_actions": [],
  "denied_actions": [],
  "policy_summary": {
    "mode": "advisory",
    "allowed_actions": ["transport_restart", "source_suppression"],
    "max_actions_per_window": 5,
    "cooldown_per_target": 300,
    "require_min_confidence": 0.7,
    "allow_mesh_level": true,
    "allow_transport_restart": true,
    "allow_source_suppression": true,
    "action_window_seconds": 3600,
    "restart_cap_per_window": 3
  },
  "reality_matrix": [],
  "reasons_for_denial": [],
  "emergency_disable": false,
  "status": "control unavailable without service control hooks"
}
```

**Note:** When service control hooks are not configured, the status indicates this explicitly. The control plane operates in "advisory" mode by default, meaning it will log recommendations but not execute actions automatically.

**Control Modes:**
- `disabled` - Control plane is completely disabled
- `advisory` - Recommendations logged, no automatic actions
- `assisted` - Some automatic actions with operator confirmation
- `automatic` - Full automatic remediation (requires confidence thresholds)

---

### GET /api/v1/control/actions
Control actions and decisions history.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `transport` | string | "" | Filter by transport name |
| `start` | string (RFC3339) | "" | Start time |
| `end` | string (RFC3339) | "" | End time |
| `limit` | integer | default | Max results |
| `offset` | integer | 0 | Pagination offset |

**Response:**
```json
{
  "actions": [
    {
      "id": "action-001",
      "transport_name": "mqtt-primary",
      "action_type": "transport_restart",
      "trigger_reason": "failure_threshold_exceeded",
      "lifecycle_state": "completed",
      "result": "executed_successfully",
      "executed_at": "2026-03-20T11:30:00Z",
      "completed_at": "2026-03-20T11:30:05Z",
      "reversible": true,
      "expires_at": ""
    }
  ],
  "decisions": [
    {
      "id": "decision-001",
      "transport_name": "mqtt-primary",
      "proposed_action": "transport_restart",
      "allowed": false,
      "denial_code": "cooldown",
      "denial_reason": "Action is within cooldown period for target",
      "confidence": 0.8,
      "created_at": "2026-03-20T11:35:00Z"
    }
  ],
  "in_flight": [],
  "reality_matrix": [],
  "transport": "mqtt-primary",
  "start": "2026-03-20T10:00:00Z",
  "end": "2026-03-20T12:00:00Z",
  "pagination": {
    "limit": 100,
    "offset": 0
  }
}
```

**Denial Codes:**
- `policy` - Action violates policy
- `mode` - Current mode doesn't allow this action
- `override` - Operator override denied
- `low_confidence` - Confidence below threshold
- `transient` - Transient condition prevented execution
- `cooldown` - Action in cooldown period
- `budget` - Action budget exceeded
- `missing_actuator` - No actuator available
- `unknown_blast_radius` - Cannot determine impact
- `no_alternate_path` - No alternate routing path
- `irreversible` - Action is irreversible
- `conflict` - Conflicting action in progress
- `attribution_weak` - Cannot attribute issue to target

---

### GET /api/v1/control/history
Control history (same data as `/api/v1/control/actions` but without the structured wrapper).

**Query Parameters:** Same as `/api/v1/control/actions`

**Response:** Raw payload from control history function (structure varies by implementation)

---

### GET /api/v1/config/inspect

Returns the effective runtime-loaded config (redacted), plus fingerprints and safe-default violations.

**Response:**
```json
{
  "fingerprint": "sha256-of-loaded-config-or-effective-json",
  "canonical_fingerprint": "sha256-of-canonical-effective-config",
  "values": {
    "bind": { "api": "127.0.0.1:8080", "metrics": "" },
    "auth": { "enabled": false, "ui_user": "admin" },
    "storage": { "database_path": "./data/mel.db" },
    "privacy": { "redact_exports": true, "map_reporting_allowed": false },
    "features": {
      "google_maps_in_topology_ui": false,
      "google_maps_api_key_env": "",
      "metrics": false
    }
  },
  "violations": [
    {
      "field": "control.mode",
      "issue": "non-advisory control mode enabled",
      "current": "enabled",
      "safe": "advisory or disabled"
    }
  ]
}
```

`values` is the redacted runtime-loaded config object (not a health assertion).

---

### GET /api/v1/control/operational-state

Snapshot of freezes, maintenance windows, pending approvals, **queue metrics**, and
**executor presence** (heartbeat from `mel serve` when available). Executor activity may
be `active`, `inactive`, or `unknown` if no heartbeat has been recorded.

**Typed posture fields currently emitted:**

```json
{
  "automation_mode": "normal|frozen|maintenance",
  "freeze_count": 0,
  "approval_backlog": 0,
  "snapshot_at": "2026-04-02T12:00:00Z",
  "queue_metrics": {
    "queued_lifecycle_pending_count": 0,
    "awaiting_approval_count": 0,
    "approved_waiting_executor_count": 0,
    "oldest_queued_pending_created_at": "",
    "oldest_approved_waiting_executor_created_at": ""
  },
  "executor": {
    "executor_activity": "active|inactive|unknown",
    "executor_last_heartbeat_at": "",
    "executor_last_reported_kind": "",
    "executor_heartbeat_basis": "control_plane_state",
    "executor_presence_note": "",
    "backlog_requires_active_executor": true
  },
  "active_freezes": [
    {
      "id": "frz-1",
      "scope_type": "global|transport|action_type",
      "scope_value": "",
      "reason": "",
      "created_by": "",
      "created_at": ""
    }
  ],
  "active_maintenance": [
    {
      "id": "mw-1",
      "reason": "",
      "created_by": "",
      "starts_at": "",
      "ends_at": ""
    }
  ]
}
```

### POST /api/v1/control/actions/{id}/approve

Approves a `pending_approval` action. Response includes explicit fields such as
`approval_does_not_imply_execution`, `http_approve_does_not_drain_queue`,
`queued_for_execution`, and a structured `policy` object (single approver, approval basis,
blast-radius flags). **Does not** execute unrelated queued work.

**Body (optional):** `note`, `break_glass_sod_ack`, `break_glass_sod_reason` (reason
required when ack is true for same-submitter SoD override).

### POST /api/v1/control/actions/{id}/reject

Rejects a `pending_approval` action. Same break-glass JSON fields as approve when the
submitter must reject their own proposal.

### GET /api/v1/control/actions/{id}/inspect

Returns `action`, `decision`, `evidence_bundle`, and `approval_policy` (structured policy
view derived from persisted rows and config).

---

## Common Data Structures

### Transport Health
```json
{
  "transport_name": "mqtt-primary",
  "transport_type": "mqtt",
  "score": 85,
  "state": "degraded",
  "last_evaluated_at": "2026-03-20T12:00:00Z",
  "primary_reason": "timeout_failure",
  "signals": {
    "recent_failures": 3,
    "dead_letter_count": 0,
    "retry_count": 2,
    "last_heartbeat_delta_seconds": 65,
    "anomaly_rate": 0.6,
    "observation_drops": 0,
    "active_episode": true
  },
  "explanation": {
    "transport_name": "mqtt-primary",
    "score": 85,
    "state": "degraded",
    "top_penalties": [
      {
        "reason": "timeout_failure",
        "penalty": 15,
        "count": 3,
        "window": "5m"
      }
    ],
    "active_cluster_reason": "timeout_failure",
    "active_cluster_count": 5,
    "active_episode_id": "ep-001",
    "failure_count": 3,
    "observation_drops": 0,
    "dead_letter_count": 0,
    "recovery_blockers": ["runtime_state:retrying", "active_failure_episode:ep-001 (3 failures)"]
  }
}
```

### Failure Cluster
```json
{
  "transport_name": "mqtt-primary",
  "transport_type": "mqtt",
  "reason": "timeout_failure",
  "count": 5,
  "first_seen": "2026-03-20T11:25:00Z",
  "last_seen": "2026-03-20T11:35:00Z",
  "severity": "warn",
  "episode_id": "ep-001",
  "includes_dead_letter": false,
  "includes_observation_drops": false,
  "cluster_key": "mqtt-primary|timeout_failure|ep-001"
}
```

### Transport Alert
```json
{
  "id": "alert-001",
  "transport_name": "mqtt-primary",
  "transport_type": "mqtt",
  "severity": "warn",
  "reason": "timeout_failure",
  "summary": "Connection timeout detected",
  "first_triggered_at": "2026-03-20T11:30:00Z",
  "last_updated_at": "2026-03-20T11:35:00Z",
  "active": true,
  "episode_id": "ep-001",
  "cluster_key": "cluster-001",
  "contributing_reasons": ["timeout_failure", "retry_threshold_exceeded"],
  "cluster_reference": "ref-001",
  "penalty_snapshot": [
    {
      "reason": "timeout_failure",
      "penalty": 15,
      "count": 3,
      "window": "5m"
    }
  ],
  "trigger_condition": "consecutive_timeouts > 3"
}
```

### Health States
- `healthy` (score >= 90)
- `degraded` (score 70-89)
- `unstable` (score 40-69)
- `failed` (score < 40)

### Alert Severities
- `critical` - Immediate action required
- `warn` - Attention needed
- `info` - Informational

## Error Response Format

All errors follow this format:
```json
{
  "error": {
    "code": "error_code",
    "message": "Human-readable error description"
  }
}
```

**Common Error Codes:**
- `auth_required` - Authentication needed
- `db_query_failed` - Database operation failed
- `status_failed` - Status collection failed
- `panel_failed` - Panel generation failed
- `missing_node` - Node identifier required
- `node_not_found` - Node not in observations
- `missing_transport` - Transport name required
- `transport_not_found` - Transport not found
- `mesh_inspect_failed` - Mesh inspection failed
- `control_status_failed` - Control status retrieval failed
- `control_actions_failed` - Control actions retrieval failed
- `control_history_failed` - Control history retrieval failed

## Web UI

When `features.web_ui` is enabled, the root path `/` serves an HTML dashboard with:
- Onboarding guide
- Status overview
- Instrument panel
- Transport health table
- Dead letters
- Node inventory
- Recent messages
- Privacy findings
- Config recommendations
- Event logs

The Web UI uses the same API endpoints and respects authentication settings.
Export respects `platform.retention.allow_export`. When disabled, API returns `403` with policy detail.

### GET /api/v1/platform/posture
Machine-visible platform policy posture (privacy/runtime/performance envelope) including:
- telemetry outbound state,
- retention/export/delete semantics,
- optional inference provider config availability,
- assist task availability/routing (`available`, `queued`, `partial`, `unavailable`).

This endpoint is authoritative for operator/runtime truth claims; assist outputs remain non-canonical by contract.
