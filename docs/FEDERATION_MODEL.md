# MEL Federation Model

## Overview

MEL federation enables multiple MEL instances to operate as a coordinated distributed system without naive clustering. Each instance operates independently with its own event log and state, selectively sharing data with peers.

## Design Principles

1. **Independence first**: Each node is fully functional alone
2. **Selective sharing**: Only share what's needed (sync scopes)
3. **Trust boundaries**: Peers have explicit trust levels
4. **Safe degradation**: Partitions restrict, never break
5. **No single point of failure**: No coordinator node

## Federation vs Clustering

| Property | Clustering | Federation (MEL) |
|----------|-----------|-------------------|
| Single state | Yes | No - each node has own state |
| Leader election | Required | Not needed |
| Split-brain risk | High | Managed with safety policies |
| Partial connectivity | Problematic | Expected and handled |
| Independent operation | No | Yes |
| Selective sync | No | Yes (sync scopes) |

## Peer Model

Each federation peer has:

- **NodeID**: Unique identifier
- **Region**: Geographic/logical region
- **Endpoint**: Base URL for API communication
- **TrustLevel**: 0 (untrusted), 1 (read-only sync), 2 (full sync), 3 (authority)
- **SyncScope**: What data types/regions/transports to sync
- **State**: active, suspected, partitioned, decommission

## Trust Levels

| Level | Name | Permissions |
|-------|------|-------------|
| 0 | Untrusted | No sync, heartbeat only |
| 1 | Read-only | Can receive events, cannot write |
| 2 | Full sync | Bidirectional event sync |
| 3 | Authority | Can override local decisions |

## Sync Scopes

Sync scopes control what data flows between peers:

```json
{
  "event_types": ["observation", "anomaly", "transport_health"],
  "regions": ["us-east", "eu-west"],
  "transports": ["mqtt-backbone"],
  "exclude_types": ["operator_action"]
}
```

Empty fields mean "no restriction" (match all).

## Communication

### Heartbeats
- Periodic (default: 30s)
- Carry: node ID, region, sequence number, logical clock, policy version, state
- Enable partition detection

### Event Sync
- Pull-based (requester asks for events after a sequence number)
- Filtered by sync scope
- Idempotent ingestion (duplicate detection by event ID)
- Batched (configurable batch size)

### Conflict Detection
- Duplicate event detection
- Policy divergence detection (via heartbeat)
- Divergent score detection
- Split-brain detection

## Split-Brain Safety

When a split-brain condition is detected (>50% of peers partitioned):

1. **Alert operator** (always, by default)
2. **Restrict autopilot** (configurable): require approval for auto actions
3. **Limit autonomous actions** (configurable): max N actions during partition
4. **Full approval required** (optional): all actions need operator approval

### Detection
- Based on heartbeat timeouts
- `suspected`: missed > N heartbeats (default: 3)
- `partitioned`: missed > M heartbeats (default: 10)
- Split-brain: >50% of peers in `partitioned` state

### Resolution
- Automatic when peers reconnect and heartbeats resume
- Autonomous action counter resets on resolution

## Configuration

```json
{
  "federation": {
    "enabled": true,
    "node_id": "mel-prod-east-1",
    "node_name": "Production East 1",
    "region": "us-east",
    "peers": [
      {
        "node_id": "mel-prod-west-1",
        "name": "Production West 1",
        "endpoint": "https://mel-west.example.com:8081",
        "region": "us-west",
        "trust_level": 2,
        "sync_types": ["observation", "anomaly", "transport_health"]
      }
    ],
    "heartbeat_interval_seconds": 30,
    "suspect_after_missed": 3,
    "partition_after_missed": 10,
    "sync_batch_size": 100,
    "sync_interval_seconds": 60,
    "split_brain_policy": {
      "restrict_autopilot": true,
      "require_approval": false,
      "alert_operator": true,
      "max_autonomous_actions": 5
    }
  }
}
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/federation/status` | GET | Federation status overview |
| `/api/v1/federation/peers` | GET | List federation peers |
| `/api/v1/federation/heartbeat` | POST | Receive heartbeat from peer |
| `/api/v1/federation/sync` | POST | Handle sync request from peer |
| `/api/v1/federation/sync/health` | GET | Sync health metrics |

## Consistency Model

See [CONSISTENCY_MODEL.md](CONSISTENCY_MODEL.md) for details on:
- Eventual consistency with bounded staleness
- Conflict resolution strategies
- What can and cannot diverge between nodes
