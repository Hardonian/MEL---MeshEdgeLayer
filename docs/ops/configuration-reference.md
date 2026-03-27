# Configuration Guide & Reference

This document describes the configuration model for MEL and provides a detailed field reference for `mel.json`.

## Core Security Rules

MEL enforces strict security posture in production environments:

- **Config Permissions**: Production operator config files must be `chmod 0600`.
- **Enforcement**: `mel serve` will refuse to start if the config file mode is broader than `0600`.
- **Validation**: `mel config validate` and `mel doctor` report overly-broad permissions as high-severity findings.

---

## Configuration Reference

Generate a starter config with: `mel init --config mel.json`

### Top-Level Structure

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

### `bind` (Network Settings)

| Field          | Type   | Default            | Description                                 |
| :------------- | :----- | :----------------- | :------------------------------------------ |
| `api`          | string | `"127.0.0.1:8080"` | Listen address for the HTTP API and Web UI. |
| `allow_remote` | bool   | `false`            | Allow non-loopback connections.             |

### `storage` (Persistence)

| Field           | Type   | Default         | Description                       |
| :-------------- | :----- | :-------------- | :-------------------------------- |
| `database_path` | string | `"data/mel.db"` | Path to the SQLite database file. |
| `data_dir`      | string | `"data"`        | Directory for auxiliary data files. |

### `auth` (Access Control)

| Field        | Type   | Default | Description                               |
| :----------- | :----- | :------ | :---------------------------------------- |
| `token`      | string | `""`    | Static bearer token for API access.       |
| `basic_auth` | object | `null`  | Basic Auth credentials for UI/API access. |

### `transports` (Ingest Sources)

Each entry in the `transports` array defines an ingest source.

| Field           | Type   | Default      | Description                               |
| :-------------- | :----- | :----------- | :---------------------------------------- |
| `name`          | string | **Required** | Unique identifier for this transport.      |
| `type`          | string | **Required** | One of `serial`, `tcp`, `mqtt`.           |
| `enabled`       | bool   | `false`      | Set to `true` to activate.                |
| `endpoint`      | string | `""`         | `host:port` for TCP/MQTT transports.      |
| `serial_device` | string | `""`         | Path for serial devices (e.g. `ttyUSB0`). |
| `serial_baud`   | int    | `115200`     | Baud rate for serial connections.         |
| `mqtt_qos`      | int    | `1`          | MQTT QoS level (0, 1, or 2).              |
| `client_id`     | string | `""`         | Unique MQTT client identifier.            |
| `topic`         | string | `""`         | MQTT topic filter (e.g. `msh/#`).         |

#### Reliability Knobs (Defaults)

- `read_timeout_sec`: `15`
- `write_timeout_sec`: `5`
- `max_timeouts`: `3` (consecutive failures before transport error)

---

### `control` (Autonomous Control Plane)

| Field                                    | Type     | Default        | Description                            |
| :--------------------------------------- | :------- | :------------- | :------------------------------------- |
| `enabled`                                | bool     | `false`        | Enable the control plane worker.       |
| `mode`                                   | string   | `"guarded_auto"` | `observe_only`, `guarded_auto`, etc.   |
| `require_approval_for_action_types`      | []string | `[]`           | Action types requiring manual approval. |
| `require_approval_for_high_blast_radius` | bool     | `false`        | Gate actions with large impact.        |

---

### `intelligence` (Health & Alerts)

| Field                          | Type | Default | Description                      |
| :----------------------------- | :--- | :------ | :------------------------------- |
| `enabled`                      | bool | `false` | Enable health scoring.           |
| `retention.health_snapshot_days` | int  | `7`     | Retention for history snapshots. |
| `alerts.minimum_state_duration` | int  | `30`    | Seconds before triggering alert. |

---

## Runtime State Definitions

MEL uses a standardized state engine for all transports:

- `disabled`: Explicitly disabled.
- `configured_not_attempted`: Idle, waiting for initial connection.
- `attempting`: Actively establishing connection.
- `configured_offline`: Reachability check failed.
- `connected_no_ingest`: Link established, no packets yet.
- `ingesting`: Active data flow confirmed.
- `historical_only`: No live connection, history available.
- `error`: Terminal failure; intervention required.

## Validation

Always validate your configuration before deployment:

```bash
mel config validate --config mel.json
```

For a full environment check (including DB and serial access), use:

```bash
mel doctor --config mel.json
```
