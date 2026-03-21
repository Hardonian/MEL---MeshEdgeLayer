# MEL Event Model

## Canonical Event Log

All core inputs to MEL become events in the canonical event log. The event log is:

- **Append-only**: Events are never modified or deleted (except by retention compaction)
- **Ordered**: Locally monotonic sequence numbers
- **Uniquely identifiable**: Every event has a globally unique ID
- **Timestamped**: Both wall-clock (UTC) and logical clock (Lamport)
- **Source-attributed**: Every event identifies its originating MEL instance
- **Durable**: SQLite-backed with WAL mode for crash safety

## Event Structure

```json
{
  "id": "evt-a1b2c3d4e5f6a1b2c3d4e5f6",
  "sequence_num": 42,
  "type": "observation",
  "timestamp": "2025-01-15T10:30:00.000Z",
  "logical_clock": 156,
  "source_node_id": "mel-abc123",
  "source_region": "us-east",
  "subject": "mqtt-local",
  "data": "{\"transport\":\"mqtt-local\",\"node_num\":12345}",
  "metadata": {"priority": "normal"},
  "policy_version": "v1.2",
  "causal_parent": "evt-previous-id",
  "checksum": "sha256hex..."
}
```

## Event Types

| Type | Source | Description |
|------|--------|-------------|
| `observation` | Transport adapter | Mesh data received from a transport |
| `anomaly` | Intelligence layer | Anomaly detected in transport/node behavior |
| `topology_update` | Mesh state | Node joined, left, or updated |
| `policy_change` | Operator/config | Policy configuration changed |
| `operator_action` | Operator | Manual operator action |
| `approval` | Operator | Action approved |
| `rejection` | Operator | Action rejected |
| `adapter_state` | Transport | Transport adapter state change |
| `transport_health` | Health checker | Transport health update |
| `node_state` | Mesh state | Node state change |
| `action_proposed` | Kernel | Control action proposed |
| `action_executed` | Executor | Control action execution started |
| `action_completed` | Executor | Control action completed |
| `freeze_created` | Operator | Automation freeze created |
| `freeze_cleared` | Operator | Automation freeze cleared |
| `maintenance_start` | Operator | Maintenance window started |
| `maintenance_end` | Operator | Maintenance window ended |
| `snapshot_created` | System | State snapshot taken |
| `peer_joined` | Federation | New federation peer discovered |
| `peer_left` | Federation | Federation peer disconnected |
| `sync_received` | Federation | Events received from peer sync |
| `region_health` | Region manager | Region health update |

## Storage Schema

```sql
CREATE TABLE kernel_event_log (
    event_id        TEXT PRIMARY KEY,
    sequence_num    INTEGER NOT NULL UNIQUE,
    event_type      TEXT NOT NULL,
    timestamp       TEXT NOT NULL,
    logical_clock   INTEGER NOT NULL DEFAULT 0,
    source_node_id  TEXT NOT NULL DEFAULT '',
    source_region   TEXT NOT NULL DEFAULT '',
    subject         TEXT NOT NULL DEFAULT '',
    data            TEXT NOT NULL DEFAULT '{}',
    metadata        TEXT NOT NULL DEFAULT '{}',
    policy_version  TEXT NOT NULL DEFAULT '',
    causal_parent   TEXT NOT NULL DEFAULT '',
    checksum        TEXT NOT NULL DEFAULT ''
);
```

## Logical Clock

The event log uses Lamport logical clocks for causal ordering across federated nodes:

1. Each event increments the local clock: `clock = max(local, remote) + 1`
2. When receiving events from peers, the clock advances to maintain causal ordering
3. The logical clock does NOT replace the wall-clock timestamp — both are stored

## Integrity

Each event has a SHA-256 checksum computed over: `ID | SequenceNum | Type | Timestamp | Data`

This enables:
- Corruption detection during replay
- Integrity verification of synced events
- Tamper detection for evidence bundles

## Retention and Compaction

The event log supports bounded growth through:
- Time-based retention (configurable days)
- Compaction that preserves the most recent N events
- Sequence-based minimum retention (never compact below a threshold)

Compaction only removes events that are:
1. Older than the retention period AND
2. Have a sequence number below the minimum keep threshold
