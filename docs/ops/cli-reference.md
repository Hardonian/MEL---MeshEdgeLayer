# MEL CLI Reference

## Overview

MEL (Mesh Event Logger) is a CLI tool for managing mesh network observations, transports, and data. This document provides comprehensive reference for all available commands.

---

## Global Conventions

- All commands that require configuration accept `--config` flag (default: `configs/mel.example.json`)
- Output is JSON by default for programmatic consumption
- Text output available for select commands via `--format text`
- Exit code 0 indicates success, non-zero indicates error

---

## Configuration Commands

### `mel init`

Initialize a new MEL configuration file.

**Usage:**
```
mel init [--config <path>] [--force]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.generated.json` | Output path for generated config |
| `--force` | `false` | Overwrite existing config file |

**Exit Codes:**
- `0` - Configuration initialized successfully
- `1` - Config already exists (without `--force`) or write error

**Example Output:**
```json
{
  "status": "initialized",
  "config": "configs/mel.generated.json",
  "bind": "127.0.0.1:8080",
  "database": "data/mel.db"
}
```

**Common Use Cases:**
```bash
# Initialize with defaults
mel init

# Create config at specific path
mel init --config /etc/mel/config.json

# Regenerate existing config
mel init --config /etc/mel/config.json --force
```

---

### `mel config validate`

Validate configuration file for errors and security issues.

**Usage:**
```
mel config validate --config <path>
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.example.json` | Path to config file to validate |

**Exit Codes:**
- `0` - Configuration is valid
- `1` - Validation errors found (see `findings` array)

**Example Output:**
```json
{
  "status": "valid",
  "findings": [],
  "lints": []
}
```

**Example Output (with issues):**
```json
{
  "status": "invalid",
  "findings": [
    {
      "component": "config_file",
      "severity": "high",
      "message": "config file permissions too open",
      "guidance": "Operator config files must be chmod 600 before MEL will trust them in production."
    }
  ],
  "lints": [
    {
      "id": "storage.data_dir",
      "severity": "warning",
      "message": "data_dir does not exist",
      "remediation": "Create the data directory before starting MEL"
    }
  ]
}
```

---

## Runtime Commands

### `mel serve`

Start the MEL service with configured transports.

**Usage:**
```
mel serve [--debug] --config <path>
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.example.json` | Path to configuration file |
| `--debug` | `false` | Enable debug logging output |

**Exit Codes:**
- `0` - Graceful shutdown (SIGINT/SIGTERM received)
- `1` - Configuration error or service startup failure

**Common Use Cases:**
```bash
# Start with production config
mel serve --config /etc/mel/config.json

# Start with debug logging for troubleshooting
mel serve --config /etc/mel/config.json --debug

# Run in background with logging
mel serve --config /etc/mel/config.json 2>&1 | tee /var/log/mel.log
```

**Notes:**
- Blocks until SIGINT or SIGTERM received
- Requires config file to have permissions 0600 in production

---

### `mel doctor`

Run comprehensive system diagnostics.

**Usage:**
```
mel doctor --config <path>
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.example.json` | Path to configuration file |

**Exit Codes:**
- `0` - No critical findings
- `1` - One or more findings require attention

**Checks Performed:**
- Config file validation and permissions
- Database accessibility (read/write)
- Schema version compatibility
- Transport connectivity (serial, TCP, MQTT)
- Transport state and errors

**Example Output:**
```json
{
  "doctor_version": "v2",
  "config": "/etc/mel/config.json",
  "findings": [],
  "db": {
    "path": "data/mel.db",
    "write_ok": true,
    "read_ok": true,
    "schema_version": 3
  },
  "summary": {
    "privacy_findings": {"total": 0, "critical": 0},
    "enabled_transports": ["local-node"],
    "last_successful_ingest": "2026-03-20T14:32:11Z",
    "transport_status": {
      "local-node": "live"
    },
    "what_mel_does": [
      "observes configured transports and persists received packets to SQLite",
      "reports live vs historical transport truth without inventing traffic",
      "exposes read-only HTTP status, nodes, messages, and metrics endpoints"
    ],
    "what_mel_does_not_do": [
      "does not claim unsupported Meshtastic transports or send capability",
      "does not prove hardware validation that was not exercised in this environment",
      "does not mark ingest successful unless the message was written to SQLite"
    ]
  }
}
```

**Common Use Cases:**
```bash
# Pre-flight check before starting service
mel doctor --config /etc/mel/config.json && mel serve --config /etc/mel/config.json

# Troubleshoot connectivity issues
mel doctor --config /etc/mel/config.json | jq '.findings[]'
```

---

## Status Commands

### `mel status`

Get current system status snapshot.

**Usage:**
```
mel status --config <path>
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.example.json` | Path to configuration file |

**Exit Codes:**
- `0` - Status retrieved successfully
- `1` - Database or configuration error

**Example Output:**
```json
{
  "uptime": "72h15m30s",
  "database": {
    "path": "data/mel.db",
    "size_bytes": 16777216,
    "message_count": 15420,
    "node_count": 47
  },
  "transports": {
    "local-node": {
      "state": "live",
      "last_ingest": "2026-03-20T14:32:11Z",
      "messages_total": 15420
    }
  },
  "last_successful_ingest": "2026-03-20T14:32:11Z"
}
```

---

### `mel panel`

Display operator-facing status panel.

**Usage:**
```
mel panel [--format text|json] --config <path>
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.example.json` | Path to configuration file |
| `--format` | `text` | Output format: `text` or `json` |

**Exit Codes:**
- `0` - Panel displayed successfully
- `1` - Status collection error

**Example Output (text):**
```
MEL PANEL 2026-03-20T14:32:11Z [HEALTHY]
System operational. 1 transport active.

[live] local-node       live score=95* msgs=15420 last=2026-03-20T14:32:11Z

Short commands: status | nodes | panel | logs | doctor
8-bit device menu:
  1 status  Show system status
  2 nodes   List mesh nodes
  3 panel   Display this panel
```

**Example Output (json):**
```json
{
  "generated_at": "2026-03-20T14:32:11Z",
  "operator_state": "healthy",
  "summary": "System operational. 1 transport active.",
  "transports": [
    {
      "label": "[live]",
      "name": "local-node",
      "state": "live",
      "score": 95,
      "messages": 15420,
      "last_ingest": "2026-03-20T14:32:11Z"
    }
  ],
  "short_commands": ["status", "nodes", "panel", "logs", "doctor"],
  "device_menu": [
    {"key": "1", "label": "status", "action": "Show system status"},
    {"key": "2", "label": "nodes", "action": "List mesh nodes"}
  ]
}
```

**Common Use Cases:**
```bash
# Quick operator view
mel panel --config /etc/mel/config.json

# Integration with monitoring
mel panel --format json --config /etc/mel/config.json | jq '.operator_state'
```

---

## Node Commands

### `mel nodes`

List all known mesh nodes.

**Usage:**
```
mel nodes --config <path>
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.example.json` | Path to configuration file |

**Exit Codes:**
- `0` - Nodes listed successfully
- `1` - Database error

**Example Output:**
```json
{
  "nodes": [
    {
      "node_num": 12345,
      "node_id": "!abcd1234",
      "long_name": "Relay Node Alpha",
      "short_name": "RNA",
      "last_seen": "2026-03-20T14:30:00Z",
      "last_gateway_id": "!gateway01",
      "lat_redacted": null,
      "lon_redacted": null,
      "altitude": 150,
      "last_snr": 8.5,
      "last_rssi": -75,
      "message_count": 342
    }
  ]
}
```

**Common Use Cases:**
```bash
# List all nodes
mel nodes --config /etc/mel/config.json

# Count total nodes
mel nodes --config /etc/mel/config.json | jq '.nodes | length'

# Find nodes by name pattern
mel nodes --config /etc/mel/config.json | jq '.nodes[] | select(.long_name | contains("Relay"))'
```

---

### `mel node inspect`

Get detailed information about a specific node.

**Usage:**
```
mel node inspect <node-id> --config <path>
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `node-id` | Node number (e.g., `12345`) or node ID (e.g., `!abcd1234`) |

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.example.json` | Path to configuration file |

**Exit Codes:**
- `0` - Node found and displayed
- `1` - Node not found or database error

**Example Output:**
```json
{
  "node_num": 12345,
  "node_id": "!abcd1234",
  "long_name": "Relay Node Alpha",
  "short_name": "RNA",
  "last_seen": "2026-03-20T14:30:00Z",
  "last_gateway_id": "!gateway01",
  "lat_redacted": null,
  "lon_redacted": null,
  "altitude": 150,
  "last_snr": 8.5,
  "last_rssi": -75,
  "message_count": 342
}
```

**Common Use Cases:**
```bash
# Inspect by node number
mel node inspect 12345 --config /etc/mel/config.json

# Inspect by node ID
mel node inspect '!abcd1234' --config /etc/mel/config.json
```

---

## Transport Commands

### `mel transports list`

List configured transports and their status.

**Usage:**
```
mel transports list --config <path>
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.example.json` | Path to configuration file |

**Exit Codes:**
- `0` - Transports listed successfully
- `1` - Status collection error

**Example Output:**
```json
{
  "transports": {
    "local-node": {
      "type": "serial",
      "enabled": true,
      "state": "live",
      "last_ingest": "2026-03-20T14:32:11Z"
    },
    "mqtt-broker": {
      "type": "mqtt",
      "enabled": true,
      "state": "idle",
      "last_ingest": null
    }
  },
  "contention_warning": false,
  "selection_rule": "prefer one direct-node transport; hybrid direct+MQTT dedupes only when both paths expose byte-identical mesh packet payloads, so operators must still verify duplicate behavior in their own deployment"
}
```

**Notes:**
- `contention_warning` is `true` when multiple transports are enabled
- Hybrid direct+MQTT setups require manual verification of deduplication behavior

---

### `mel inspect transport`

Get detailed inspection data for a specific transport.

**Usage:**
```
mel inspect transport <name> --config <path>
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `name` | Transport name as configured |

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.example.json` | Path to configuration file |

**Exit Codes:**
- `0` - Inspection data retrieved
- `1` - Transport not found or error

**Example Output:**
```json
{
  "name": "local-node",
  "type": "serial",
  "state": "live",
  "metrics": {
    "messages_total": 15420,
    "messages_last_hour": 142,
    "error_count": 0
  },
  "recent_errors": [],
  "configuration": {
    "device": "/dev/ttyUSB0",
    "baud": 115200
  }
}
```

---

### `mel inspect mesh`

Get mesh-wide inspection data.

**Usage:**
```
mel inspect mesh --config <path>
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.example.json` | Path to configuration file |

**Exit Codes:**
- `0` - Mesh data retrieved
- `1` - Database error

**Example Output:**
```json
{
  "node_count": 47,
  "active_nodes_last_hour": 12,
  "total_messages": 15420,
  "messages_last_hour": 342,
  "transport_breakdown": {
    "local-node": 8200,
    "mqtt-broker": 7220
  },
  "top_talkers": [
    {"node_id": "!abcd1234", "message_count": 520},
    {"node_id": "!efgh5678", "message_count": 413}
  ]
}
```

---

## Data Commands

### `mel replay`

Replay recent messages from the database.

**Usage:**
```
mel replay --config <path> [--node <id>] [--type <message-type>] [--limit <n>]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.example.json` | Path to configuration file |
| `--node` | `""` | Filter by node number |
| `--type` | `""` | Filter by message type |
| `--limit` | `50` | Maximum number of messages to return |

**Exit Codes:**
- `0` - Messages retrieved
- `1` - Database error

**Example Output:**
```json
{
  "messages": [
    {
      "transport_name": "local-node",
      "packet_id": 12345,
      "from_node": 12345,
      "to_node": 4294967295,
      "portnum": 4,
      "payload_text": "Hello mesh!",
      "payload_json": "{\"message_type\":\"text\",...}",
      "rx_time": "2026-03-20T14:32:11Z",
      "created_at": "2026-03-20T14:32:12Z"
    }
  ],
  "filters": {
    "node": "",
    "type": "",
    "limit": 50
  }
}
```

**Common Use Cases:**
```bash
# Recent messages
mel replay --config /etc/mel/config.json

# Messages from specific node
mel replay --config /etc/mel/config.json --node 12345

# Last 100 text messages
mel replay --config /etc/mel/config.json --type text --limit 100

# Tail-like functionality
watch -n 5 'mel replay --config /etc/mel/config.json --limit 5'
```

---

### `mel logs tail`

Show recent audit log entries.

**Usage:**
```
mel logs tail --config <path>
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.example.json` | Path to configuration file |

**Exit Codes:**
- `0` - Logs retrieved
- `1` - Database error

**Example Output:**
```json
[
  {
    "category": "transport",
    "level": "info",
    "message": "Transport local-node connected",
    "created_at": "2026-03-20T14:30:00Z"
  },
  {
    "category": "ingest",
    "level": "info",
    "message": "Stored packet from node !abcd1234",
    "created_at": "2026-03-20T14:30:01Z"
  }
]
```

**Common Use Cases:**
```bash
# View recent logs
mel logs tail --config /etc/mel/config.json

# Monitor for errors
mel logs tail --config /etc/mel/config.json | jq '.[] | select(.level == "error")'
```

---

### `mel db vacuum`

Optimize database by reclaiming free space.

**Usage:**
```
mel db vacuum --config <path>
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.example.json` | Path to configuration file |

**Exit Codes:**
- `0` - Vacuum completed
- `1` - Database error

**Example Output:**
```json
{
  "status": "vacuum complete"
}
```

**Common Use Cases:**
```bash
# Weekly maintenance
mel db vacuum --config /etc/mel/config.json

# After bulk deletion
mel db vacuum --config /etc/mel/config.json
```

---

## Control Commands

### `mel control status`

Get control plane status and explanation.

**Usage:**
```
mel control status --config <path>
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.example.json` | Path to configuration file |

**Exit Codes:**
- `0` - Status retrieved
- `1` - Evaluation error

**Example Output:**
```json
{
  "current_state": "active",
  "explanation": "Control plane is active and processing messages",
  "policies_applied": 3,
  "last_action": "2026-03-20T14:30:00Z"
}
```

---

### `mel control history`

Query control plane action history.

**Usage:**
```
mel control history --config <path> [--transport <name>] [--start <time>] [--end <time>] [--limit <n>] [--offset <n>]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.example.json` | Path to configuration file |
| `--transport` | `""` | Filter by transport name |
| `--start` | `""` | Start time (RFC3339 format) |
| `--end` | `""` | End time (RFC3339 format) |
| `--limit` | `50` | Maximum rows to return |
| `--offset` | `0` | Pagination offset |

**Exit Codes:**
- `0` - History retrieved
- `1` - Database error

**Example Output:**
```json
{
  "actions": [
    {
      "transport_name": "local-node",
      "action": "allow",
      "reason": "default_policy",
      "created_at": "2026-03-20T14:30:00Z"
    }
  ],
  "decisions": [
    {
      "transport_name": "local-node",
      "decision": "forward",
      "created_at": "2026-03-20T14:30:00Z"
    }
  ],
  "transport": "",
  "start": "",
  "end": "",
  "pagination": {
    "limit": 50,
    "offset": 0
  }
}
```

**Common Use Cases:**
```bash
# Recent control actions
mel control history --config /etc/mel/config.json

# Actions for specific transport
mel control history --config /etc/mel/config.json --transport local-node

# Actions in time range
mel control history --config /etc/mel/config.json --start 2026-03-20T00:00:00Z --end 2026-03-21T00:00:00Z

# Paginate through results
mel control history --config /etc/mel/config.json --limit 100 --offset 100
```

---

## Policy Commands

### `mel policy explain`

Display active policy configuration and effects.

**Usage:**
```
mel policy explain --config <path>
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.example.json` | Path to configuration file |

**Exit Codes:**
- `0` - Policy explanation generated
- `1` - Configuration error

**Example Output:**
```json
{
  "policies": [
    {
      "name": "default_allow",
      "effect": "allow",
      "applies_to": ["all_transports"],
      "conditions": []
    }
  ],
  "effective_rules": 1,
  "default_action": "allow"
}
```

---

## Privacy Commands

### `mel privacy audit`

Audit privacy configuration and data handling.

**Usage:**
```
mel privacy audit [--format json|text] --config <path>
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.example.json` | Path to configuration file |
| `--format` | `json` | Output format: `json` or `text` |

**Exit Codes:**
- `0` - Audit completed
- `1` - Configuration error

**Example Output (json):**
```json
{
  "summary": {
    "total": 2,
    "critical": 0,
    "high": 1,
    "medium": 1,
    "low": 0
  },
  "findings": [
    {
      "severity": "high",
      "message": "Location data not redacted in exports",
      "remediation": "Enable privacy.redact_exports in configuration"
    }
  ]
}
```

**Example Output (text):**
```
Privacy audit summary: {total:2 critical:0 high:1 medium:1 low:0}
- [HIGH] Location data not redacted in exports
  remediation: Enable privacy.redact_exports in configuration
- [MEDIUM] Database permissions may be too permissive
  remediation: Ensure database file is chmod 600
```

---

## Backup/Export Commands

### `mel fleet evidence import`

Import canonical offline remote evidence into the local SQLite database.

**Usage:**
```
mel fleet evidence import --file <path.json> --config <path> [--strict-origin] [--actor <id>]
```

**Accepted input formats:**
- `mel_remote_evidence_bundle`
- `mel_remote_evidence_batch`

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--file` | `""` | Path to canonical remote evidence bundle or batch JSON |
| `--config` | `configs/mel.example.json` | Path to configuration file |
| `--strict-origin` | `false` | Reject claimed origin instance mismatch instead of accepting with caveat |
| `--actor` | `""` | Operator id recorded in local audit/timeline rows |

**What it does:**
- validates the payload,
- persists an import batch audit row,
- persists imported evidence item rows,
- writes timeline audit events,
- keeps imported evidence distinct from local observations.

**What it does not do:**
- create live federation,
- verify authenticity cryptographically,
- imply remote control or remote execution,
- claim fleet-wide completeness.

---

### `mel fleet evidence list`

List imported remote evidence items with summary posture.

**Usage:**
```
mel fleet evidence list --config <path> [--limit <n>] [--batch <batch-id>]
```

Use `--batch` when you want to inspect only one persisted import batch without mixing it into other imported evidence.

---

### `mel fleet evidence show`

Show one imported remote evidence item with full inspection detail.

**Usage:**
```
mel fleet evidence show <import-id> --config <path>
```

The output includes provenance, validation, timing posture, merge inspection, and related evidence analysis.

---

### `mel fleet evidence batches`

List offline remote import batches.

**Usage:**
```
mel fleet evidence batches --config <path> [--limit <n>]
```

This is the quickest way to answer "what was imported, from where, and with what validation posture?"

---

### `mel fleet evidence batch-show`

Show one persisted remote import batch and its item drilldown.

**Usage:**
```
mel fleet evidence batch-show <batch-id> --config <path>
```

Use this when you need per-batch acceptance/rejection counts, source path/name, claimed origin, and item-level inspection in one response.

---

### `mel export`

Export data bundle for analysis or migration.

**Usage:**
```
mel export --config <path> [--out <path>]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.example.json` | Path to configuration file |
| `--out` | `""` | Write to file instead of stdout |

**Exit Codes:**
- `0` - Export completed
- `1` - Database or write error

**Example Output:**
```json
{
  "exported_at": "2026-03-20T14:32:11Z",
  "redacted": false,
  "nodes": [...],
  "messages": [...],
  "dead_letters": [...],
  "audit_logs": [...]
}
```

**Notes:**
- Messages are redacted if `privacy.redact_exports` is enabled in config
- Limited to last 250 messages, dead letters, and audit logs

**Common Use Cases:**
```bash
# Export to stdout
mel export --config /etc/mel/config.json

# Export to file
mel export --config /etc/mel/config.json --out /backup/mel-export-$(date +%Y%m%d).json

# Export and compress
mel export --config /etc/mel/config.json | gzip > /backup/mel-export.json.gz
```

---

### `mel import validate`

Validate a bundle and report whether it matches MEL's canonical remote-evidence import contract.

**Usage:**
```
mel import validate --bundle <path>
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--bundle` | `""` | Path to bundle file (required) |

**Exit Codes:**
- `0` - Bundle is valid
- `1` - Bundle is invalid or unreadable

**Example Output (canonical remote evidence batch):**
```json
{
  "format": "mel_remote_evidence_batch",
  "valid": true,
  "remote_evidence_import_supported": true,
  "validation": {
    "outcome": "accepted_partial_bundle",
    "reasons": [
      "authenticity_not_cryptographically_verified",
      "historical_import_not_live",
      "accepted_partial_bundle"
    ]
  },
  "item_count": 2,
  "accepted_count": 1,
  "rejected_count": 1
}
```

**Example Output (generic export bundle):**
```json
{
  "valid": true,
  "format": "mel_export_bundle",
  "remote_evidence_import_supported": false,
  "keys": ["audit_logs", "dead_letters", "exported_at", "messages", "nodes", "redacted"],
  "note": "This is a general MEL export bundle. It is structurally valid as an export, but `mel fleet evidence import` only accepts canonical mel_remote_evidence_bundle or mel_remote_evidence_batch JSON."
}
```

**Common Use Cases:**
```bash
# Validate a canonical remote-evidence batch before import
mel import validate --bundle /evidence/remote-site.json

# Confirm that a general MEL export is not a canonical remote-evidence import bundle
mel import validate --bundle /backup/mel-export.json
```

---

### `mel backup create`

Create a complete backup bundle.

**Usage:**
```
mel backup create --config <path> [--out <path>]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `configs/mel.example.json` | Path to configuration file |
| `--out` | `""` | Bundle output path |

**Exit Codes:**
- `0` - Backup created
- `1` - Backup creation error

**Example Output:**
```json
{
  "manifest_version": "1.0",
  "created_at": "2026-03-20T14:32:11Z",
  "config_path": "configs/mel.json",
  "database_path": "data/mel.db",
  "bundle_path": "/backup/mel-backup-20260320.tar.gz",
  "checksum": "sha256:abc123..."
}
```

**Common Use Cases:**
```bash
# Create backup with timestamp
mel backup create --config /etc/mel/config.json --out /backup/mel-$(date +%Y%m%d-%H%M%S).tar.gz

# Daily backup cron job
0 2 * * * mel backup create --config /etc/mel/config.json --out /backup/mel-daily.tar.gz
```

---

### `mel backup restore`

Validate a backup bundle (dry-run only).

**Usage:**
```
mel backup restore --bundle <path> --dry-run [--destination <dir>]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--bundle` | `""` | Bundle path (required) |
| `--dry-run` | N/A | Validate only (required flag) |
| `--destination` | `.` | Restore directory for validation |

**Exit Codes:**
- `0` - Backup is valid for restore
- `1` - Backup validation failed

**Example Output:**
```json
{
  "valid": true,
  "bundle": "/backup/mel-backup.tar.gz",
  "destination": "/tmp/restore",
  "manifest": {
    "version": "1.0",
    "created_at": "2026-03-20T14:32:11Z"
  },
  "issues": []
}
```

**Limitations:**
- **Restore without `--dry-run` is not implemented** in this release candidate
- Actual restore must be performed manually by extracting the bundle

**Common Use Cases:**
```bash
# Validate backup before manual restore
mel backup restore --bundle /backup/mel-backup.tar.gz --dry-run

# Validate with specific destination
mel backup restore --bundle /backup/mel-backup.tar.gz --dry-run --destination /var/lib/mel
```

---

## Development Commands

### `mel dev-simulate-mqtt`

Start a development MQTT broker that simulates mesh traffic.

**Usage:**
```
mel dev-simulate-mqtt [--endpoint <host:port>] [--topic <topic>]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--endpoint` | `127.0.0.1:18830` | MQTT broker endpoint |
| `--topic` | `msh/US/2/e/test` | Topic to publish on |

**Exit Codes:**
- `0` - Never exits (runs until interrupted)
- `1` - Bind error

**Example Output:**
```
(Listening on 127.0.0.1:18830, publishing to msh/US/2/e/test)
```

**Common Use Cases:**
```bash
# Start with defaults
mel dev-simulate-mqtt

# Custom endpoint and topic
mel dev-simulate-mqtt --endpoint 0.0.0.0:1883 --topic msh/EU/1/e/dev

# Run in background
mel dev-simulate-mqtt &
```

**Notes:**
- Generates sample Meshtastic protobuf packets
- Useful for testing MQTT transport without hardware
- Blocks indefinitely - run in background for testing

---

### `mel version`

Display MEL version information.

**Usage:**
```
mel version
```

**Exit Codes:**
- `0` - Version displayed

**Example Output:**
```
v1.2.3-rc1
```

---

## Exit Code Reference

| Exit Code | Meaning |
|-----------|---------|
| `0` | Success |
| `1` | General error (validation failed, database error, not found) |

Most errors include detailed JSON output with `severity` and `guidance` fields to aid troubleshooting.

---

## Environment Variables

MEL does not currently use environment variables for configuration. All settings are managed through the configuration file specified via `--config`.

---

## Configuration File Permissions

Production deployments require configuration files to have restrictive permissions:

```bash
chmod 600 /etc/mel/config.json
```

MEL will refuse to start with overly permissive config file permissions (world-readable).

---

## Quick Reference Card

```bash
# Setup
mel init --config /etc/mel/config.json
mel config validate --config /etc/mel/config.json
mel doctor --config /etc/mel/config.json

# Runtime
mel serve --config /etc/mel/config.json

# Monitoring
mel status --config /etc/mel/config.json
mel panel --config /etc/mel/config.json
mel logs tail --config /etc/mel/config.json

# Data inspection
mel nodes --config /etc/mel/config.json
mel node inspect <id> --config /etc/mel/config.json
mel replay --config /etc/mel/config.json --limit 10

# Maintenance
mel db vacuum --config /etc/mel/config.json
mel backup create --config /etc/mel/config.json --out /backup/mel.tar.gz
mel privacy audit --config /etc/mel/config.json
```
