# MEL Recovery Runbook

## Scenario: Database Corruption

### Detection
```
GET /api/v1/kernel/durability
→ {"integrity_check": false, "detail": "..."}
```

### Recovery Steps
1. Stop MEL
2. Check backup availability: `GET /api/v1/kernel/backups`
3. Restore from most recent clean backup
4. Replay events from the backup's snapshot point
5. Restart MEL
6. Verify: `GET /api/v1/kernel/durability`

## Scenario: Split-Brain Detected

### Detection
```
GET /api/v1/federation/status
→ {"split_brain": true, "conflict_count": 2}
```

### Response Steps
1. Check which peers are partitioned: `GET /api/v1/federation/peers`
2. Investigate network connectivity between nodes
3. If network issue: fix connectivity, peers auto-reconnect
4. If permanent: remove unreachable peers from config
5. Review conflicts after resolution
6. Check for duplicate action execution

## Scenario: Peer Permanently Lost

### Steps
1. Remove peer from `federation.peers` config
2. Restart MEL (hot-reload if supported)
3. The lost peer's synced events remain in local log
4. No data loss on the surviving node

## Scenario: Event Log Growing Too Large

### Detection
```
GET /api/v1/kernel/eventlog/stats
→ {"total_events": 500000, ...}
```

### Steps
1. Verify retention config: `federation.event_log_retention_days`
2. Create a snapshot before compaction: `POST /api/v1/kernel/snapshots`
3. Reduce retention if appropriate
4. Compaction runs automatically based on retention policy

## Scenario: High Backpressure / Event Drops

### Detection
```
GET /api/v1/kernel/backpressure
→ {"rejected": 1500, "throttled": 300, "pending_count": 48000}
```

### Steps
1. Check if event throughput exceeds configured limits
2. Increase `max_events_per_second` if hardware can handle it
3. Increase `max_pending_events` if memory allows
4. Investigate if a transport is flooding events
5. Consider adding sync scope filters to reduce federation traffic

## Scenario: Node Recovery After Crash

### Steps
1. MEL restarts automatically (systemd/supervisor)
2. SQLite WAL mode ensures crash-safe writes
3. Event log recovers last sequence number from database
4. Kernel state can be restored from latest snapshot
5. Delta events replayed from snapshot point to current
6. Federation sync catches up missed events from peers

## Scenario: Full Cluster Restart

### Steps
1. Start each MEL node independently
2. Each node loads its own event log and latest snapshot
3. Federation auto-discovers peers via configured endpoints
4. Heartbeats resume, peer states update
5. Sync catches up any events missed during downtime
6. Verify: `GET /api/v1/federation/status` on each node

## Scenario: State Divergence After Partition Recovery

### Detection
After a split-brain resolves, peer states may have diverged. The consistency model (`internal/consistency`) detects this automatically during sync.

### Verification Steps
1. Check convergence between local and remote state using `CheckConvergence()`
2. Review divergence report: critical (action lifecycle, freezes, policy), major (classification mismatch), minor (score drift)
3. `CompareAndResolve()` automatically resolves using strategy-specific rules:
   - Score dominance: takes the worse score (conservative safety)
   - Lifecycle advancement: takes the more advanced action state
   - Union merge: keeps all active freezes from both sides
   - Policy precedence: higher version wins
4. If critical divergences remain unresolved, operator intervention required
5. Verify bounded staleness: `CheckStaleness()` confirms peers are within acceptable drift bounds (clock drift < 1000, sequence lag < 500, time drift < 5 minutes)

## Scenario: Region Degraded

### Detection
```
GET /api/v1/topology/global
→ {"degraded_regions": ["us-east"], ...}
```

### Steps
1. Check region health: `GET /api/v1/topology/region/us-east`
2. Identify failing nodes and transports
3. Cross-region fallback is automatic for scoring
4. Consider isolating region if issues spread: set `isolated: true`
5. Investigate and fix underlying transport/node issues
6. Un-isolate region when healthy
