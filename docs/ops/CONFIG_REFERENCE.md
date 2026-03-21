# MEL Configuration Reference

This document describes every configuration field in `mel.json`.
All fields are optional unless marked **(required)**.

Generate a starter config with: `mel init --config mel.json`

---

## Top-level structure

```json
{
  "bind":          { ... },
  "storage":       { ... },
  "auth":          { ... },
  "transports":    [ ... ],
  "privacy":       { ... },
  "control":       { ... },
  "intelligence":  { ... },
  "log_level":     "info"
}
```

---

## `bind`

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `api` | string | `"127.0.0.1:18080"` | Listen address for the HTTP API. |
| `allow_remote` | bool | `false` | Allow non-loopback connections. Requires `auth.token` or `auth.basic_auth`. |

---

## `storage`

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `database_path` | string | `"data/mel.db"` | Path to the SQLite database file. |
| `data_dir` | string | `"data"` | Directory for auxiliary data files. Must exist and be writable. |

---

## `auth`

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `session_secret` | string | *(generated)* | HMAC secret for session tokens. Rotate with care. |
| `token` | string | `""` | Static bearer token. Required when `bind.allow_remote = true`. |
| `basic_auth` | object | `null` | `{"username": "...", "password": "..."}` for Basic Auth. |

---

## `transports`

Each element is a transport configuration object.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | string | **(required)** | Unique transport identifier. Used in scoped freezes and maintenance windows. |
| `type` | string | **(required)** | One of `serial`, `tcp`, `serialtcp`, `mqtt`. |
| `enabled` | bool | `false` | Set to `true` to activate this transport. |
| `endpoint` | string | `""` | Combined `host:port` for TCP/MQTT transports. |
| `serial_device` | string | `""` | Device path for serial transports (e.g. `/dev/ttyUSB0`). |
| `serial_baud` | int | `115200` | Baud rate for serial transports. |
| `tcp_host` | string | `""` | TCP host (alternative to `endpoint`). |
| `tcp_port` | int | `0` | TCP port (alternative to `endpoint`). |
| `mqtt_qos` | int | `1` | MQTT QoS level (0, 1, or 2). |
| `mqtt_keep_alive_sec` | int | `30` | MQTT keepalive interval in seconds. |
| `client_id` | string | `""` | MQTT client ID. **(required for MQTT transports)** |
| `topic` | string | `""` | MQTT topic filter. **(required for MQTT transports)** |
| `read_timeout_sec` | int | `15` | Read timeout in seconds. |
| `write_timeout_sec` | int | `5` | Write timeout in seconds. |
| `max_timeouts` | int | `3` | Consecutive timeouts before transport error state. |

---

## `privacy`

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `redact_exports` | bool | `true` | Redact precise positions from export bundles. |
| `store_precise_positions` | bool | `false` | Store GPS coordinates without rounding. Not recommended. |

---

## `control`

Controls MEL's autonomous control plane behaviour.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable the control plane. |
| `mode` | string | `"guarded_auto"` | `observe_only` \| `guarded_auto` \| `approval_required` \| `disabled`. |
| `require_approval_for_action_types` | []string | `[]` | Action types that must wait for operator approval before executing. Example: `["restart_transport", "reconfigure_transport"]`. |
| `require_approval_for_high_blast_radius` | bool | `false` | When `true`, actions with blast radius `mesh`, `global`, or `unknown` require operator approval. |
| `approval_timeout_seconds` | int | `0` | Seconds before a `pending_approval` action automatically expires. `0` = no timeout. Max: 86400 (24 h). |

### `control.mode` values

| Value | Behaviour |
|-------|-----------|
| `observe_only` | No actions executed; decisions are recorded only. |
| `guarded_auto` | Actions execute automatically within configured blast-radius limits. |
| `approval_required` | All actions require operator approval. Equivalent to setting `require_approval_for_high_blast_radius = true` and listing all action types. |
| `disabled` | Control plane worker is not started. |

---

## `intelligence`

Controls MEL's intelligence layer (health scoring, alerts, anomaly detection).

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable the intelligence layer. |
| `retention.health_snapshot_days` | int | `7` | Days to retain health snapshots. Min: 7. |
| `alerts.minimum_state_duration_seconds` | int | `30` | Minimum alert duration before escalation. Min: 30. |
| `alerts.cooldown_seconds` | int | `60` | Cooldown between repeated alerts. Min: 60. |
| `alerts.recovery_score_healthy` | int | `70` | Health score threshold for recovery confirmation. |

---

## Environment Variable Overrides

Most scalar config fields can be overridden with environment variables:

| Variable | Config field |
|----------|-------------|
| `MEL_BIND_API` | `bind.api` |
| `MEL_DATABASE_PATH` | `storage.database_path` |
| `MEL_AUTH_TOKEN` | `auth.token` |
| `MEL_LOG_LEVEL` | `log_level` |

---

## Validation

Run `mel config validate --config mel.json` to check your configuration.
Run `mel doctor --config mel.json` for a full environment check.

Common validation errors:

| Error | Fix |
|-------|-----|
| `allow_remote requires auth` | Set `auth.token` or `auth.basic_auth` before enabling remote access. |
| `mqtt transport missing client_id` | Set `transports[n].client_id` for all MQTT transports. |
| `approval_timeout_seconds must be 0-86400` | Clamp value to range `[0, 86400]`. |
| `health_snapshot_days must be >= 7` | Increase retention to at least 7 days. |
