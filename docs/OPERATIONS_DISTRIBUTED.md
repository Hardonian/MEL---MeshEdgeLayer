# MEL Distributed Operations Guide

## Deployment Models

### Single Node (Default)
Standard MEL deployment. Federation disabled. All features work as before.

### Multi-Node, Single Region
Multiple MEL instances in the same region, observing different transports or the same mesh from different vantage points.

### Multi-Region
MEL instances across geographic regions, each with region-local transports and nodes.

## Configuration

### Enabling Federation

```json
{
  "federation": {
    "enabled": true,
    "node_id": "mel-east-1",
    "region": "us-east",
    "peers": [
      {
        "node_id": "mel-west-1",
        "endpoint": "https://mel-west.example.com:8080",
        "region": "us-west",
        "trust_level": 2
      }
    ]
  }
}
```

### Region Setup

Regions are automatically created when:
- Nodes with `region` metadata are observed
- Peers in different regions send heartbeats
- Operator manually adds regions

## Operational Procedures

### Adding a Peer

1. Configure the peer in `federation.peers`
2. Restart MEL (or hot-reload config)
3. Verify connectivity: `GET /api/v1/federation/status`

### Removing a Peer

1. Remove from `federation.peers`
2. Restart MEL
3. The peer will be marked as `decommission` and eventually pruned

### Monitoring Federation Health

```
GET /api/v1/federation/status
GET /api/v1/federation/peers
GET /api/v1/federation/sync/health
```

### Creating Snapshots

```
POST /api/v1/kernel/snapshots
GET /api/v1/kernel/snapshots?limit=10
```

### Running Replay

```
POST /api/v1/kernel/replay
{
  "mode": "full",
  "policy": { ... }
}
```

### Backups

```
POST /api/v1/kernel/backup       # Create backup
GET /api/v1/kernel/backups        # List backups
GET /api/v1/kernel/durability     # Storage health
```

### Global Topology

```
GET /api/v1/topology/global
GET /api/v1/topology/region/{region_id}
```

## Failure Scenarios

### Peer Unreachable
- Heartbeats fail → peer moves to `suspected` → `partitioned`
- Sync continues to attempt at configured interval
- On reconnection, missed events are synced

### Split-Brain
- Detected when >50% peers are partitioned
- Autopilot restricted (configurable)
- Operator alerted
- System continues to operate safely in degraded mode

### Database Corruption
- `GET /api/v1/kernel/durability` checks integrity
- Restore from backup if needed
- WAL mode prevents most corruption scenarios

### Region Isolation
- Region continues to operate independently
- Cross-region fallback to healthiest region
- No data loss (events queued for sync)

## Consistency Verification

### Check Convergence Between Peers

After sync events or partition recovery, verify that peer states have converged using the consistency model (`internal/consistency`):

- **Bounded staleness**: `CheckStaleness()` evaluates clock drift, sequence lag, and time drift against configurable bounds
- **Divergence detection**: `CompareAndResolve()` identifies and resolves conflicts using strategy-specific resolution (score dominance, lifecycle advancement, union merge, policy precedence)
- **Convergence report**: `CheckConvergence()` returns whether two states have converged with critical/major/minor divergence counts

Resolution strategies:
| Category | Strategy | Rule |
|----------|----------|------|
| Node/transport scores | Score dominance | Take worse (lower) score — conservative |
| Action lifecycle | Lifecycle advancement | Take more advanced state — no regression |
| Active freezes | Union merge | Keep all freezes — safety-critical |
| Policy version | Policy precedence | Higher version wins |
| Node registry | Last-write-wins | Newer `last_seen` wins |

## Monitoring Checklist

| Check | Endpoint | Alert Threshold |
|-------|----------|-----------------|
| Federation enabled | `/api/v1/federation/status` | `enabled: false` when expected |
| Peer connectivity | `/api/v1/federation/peers` | Any peer `state: partitioned` |
| Split-brain | `/api/v1/federation/status` | `split_brain: true` |
| Sync health | `/api/v1/federation/sync/health` | High lag or errors |
| Event log size | `/api/v1/kernel/eventlog/stats` | Unbounded growth |
| Backpressure | `/api/v1/kernel/backpressure` | High reject/throttle rates |
| Storage integrity | `/api/v1/kernel/durability` | Integrity check fails |
| Region health | `/api/v1/topology/global` | Any degraded region |
