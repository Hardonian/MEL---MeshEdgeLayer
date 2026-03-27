# MEL Replay System

## Overview

The replay system enables deterministic reconstruction of MEL kernel state from the event log. Given the same event stream and policy, replay produces identical results.

## Replay Modes

### Full Replay
Replays the entire event log from genesis (sequence 0). Produces complete state reconstruction.

### Windowed Replay
Replays events within a time or sequence range. Requires an initial state (snapshot) for accuracy.

### Scenario Replay (What-If)
Replays events against a modified policy to evaluate "what would have happened if we used this policy?"

### Dry-Run Replay
Replays with modified policy and compares against actual outcomes. Useful for policy evaluation before deployment.

### Verification Replay
Replays and compares the resulting state against a known-good expected state. Reports divergences with severity levels.

## Usage

### API

```
POST /api/v1/kernel/replay
{
  "mode": "full",
  "policy": {
    "version": "v1",
    "mode": "advisory",
    "allowed_actions": ["restart_transport"],
    "require_min_confidence": 0.75
  },
  "from_sequence": 0,
  "to_sequence": 0,
  "max_events": 10000
}
```

### Response

```json
{
  "mode": "full",
  "events_processed": 5432,
  "effects_produced": 128,
  "final_state": { ... },
  "started_at": "2025-01-15T10:00:00Z",
  "completed_at": "2025-01-15T10:00:02Z",
  "duration_ms": 2100,
  "first_sequence": 1,
  "last_sequence": 5432
}
```

### Verification Response (additional fields)

```json
{
  "verified": false,
  "divergences": [
    {
      "category": "node_score",
      "key": "!nodeA.classification",
      "expected": "healthy",
      "actual": "degraded",
      "severity": "warning"
    }
  ]
}
```

## Snapshot + Delta Replay

To avoid replaying from zero every time:

1. Take periodic state snapshots (configurable interval)
2. On replay, load the nearest snapshot before the target range
3. Replay only the delta events after the snapshot

```
POST /api/v1/kernel/replay
{
  "mode": "windowed",
  "initial_state": { ... },  // loaded from snapshot
  "from_sequence": 5000,
  "to_sequence": 6000,
  "policy": { ... }
}
```

## Snapshots

### Create
```
POST /api/v1/kernel/snapshots
```

### List
```
GET /api/v1/kernel/snapshots?limit=10
```

### Snapshot Contents
- Node registry state
- Node scores
- Transport scores
- Action lifecycle states
- Active freezes
- Policy version
- Last event sequence number
- Integrity hash (SHA-256 of serialized state)

### Verification
Each snapshot includes an integrity hash. On load, the hash is recomputed and compared to detect corruption.

## Divergence Tolerances

| Category | Tolerance | Severity |
|----------|-----------|----------|
| Node classification | Exact match required | warning |
| Composite score | ±0.001 | info |
| Transport classification | Exact match required | warning |
| Action lifecycle | Exact match required | error |
| Policy version | Exact match required | warning |
| Missing node | - | error |
| Extra node | - | warning |
