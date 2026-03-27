# MEL Kernel Architecture

## Overview

The MEL Kernel is the deterministic core of the MEL distributed control-plane system. It processes events and produces effects without executing side effects directly.

## Architecture Layers

```
┌─────────────────────────────────────────────────┐
│              Control Surfaces                    │
│         (CLI / API / Web UI)                     │
├─────────────────────────────────────────────────┤
│              Federation Layer                    │
│   (Peer Sync, Heartbeats, Split-Brain Safety)   │
├─────────────────────────────────────────────────┤
│              Adapter Layer                       │
│      (Transport Inputs, Action Execution)        │
├─────────────────────────────────────────────────┤
│          ┌─────────────────────────┐             │
│          │     KERNEL (deterministic)            │
│          │  ┌─────────────────┐    │             │
│          │  │ Event Handlers  │    │             │
│          │  │ Scoring Engine  │    │             │
│          │  │ Classification  │    │             │
│          │  │ Policy Eval     │    │             │
│          │  │ Action Lifecycle│    │             │
│          │  │ Evidence Bundle │    │             │
│          │  └─────────────────┘    │             │
│          │                         │             │
│          │  State (deterministic)  │             │
│          └─────────────────────────┘             │
├─────────────────────────────────────────────────┤
│              Event Log (append-only)             │
│         Source of Truth / Durable Stream          │
├─────────────────────────────────────────────────┤
│              Storage (SQLite + WAL)              │
│     Snapshots / Backups / Integrity Checks       │
└─────────────────────────────────────────────────┘
```

## Kernel Boundaries

### Kernel Owns
- Event ingestion and normalization
- Observation scoring and classification
- Node and transport health computation
- Action lifecycle management (propose → execute → complete)
- Evidence bundle construction
- Policy evaluation
- Decision generation
- Logical clock management

### Kernel Does NOT Own
- Transport I/O (adapter layer)
- HTTP API serving (control surface)
- Direct action execution (adapter dispatches effects)
- Database writes (event log handles persistence)
- Peer communication (federation layer)

## Determinism Guarantee

Given:
1. An identical event stream (same events in same order)
2. An identical policy version

The kernel MUST produce:
- Identical node scores
- Identical transport scores
- Identical classifications
- Identical proposed actions
- Identical state

This guarantee enables:
- Full replay from event log
- Snapshot + delta replay
- Verification replay (compare actual vs replayed state)
- Scenario replay (what-if with modified policy)

## Event Model

All inputs to the kernel are normalized into `Event` values:

| Field | Description |
|-------|-------------|
| `ID` | Globally unique event identifier |
| `SequenceNum` | Locally monotonic (assigned by event log) |
| `Type` | Event classification (observation, anomaly, etc.) |
| `Timestamp` | Wall-clock time (UTC) |
| `LogicalClock` | Lamport clock for causal ordering |
| `SourceNodeID` | Originating MEL instance |
| `SourceRegion` | Region of originating node |
| `Subject` | Primary entity (transport, node, action) |
| `Data` | JSON payload |
| `Checksum` | SHA-256 integrity hash |

### Event Types
- `observation` - mesh data received
- `anomaly` - anomaly detected
- `topology_update` - node joined/left/updated
- `transport_health` - transport state change
- `action_proposed` / `action_executed` / `action_completed`
- `freeze_created` / `freeze_cleared`
- `policy_change` - policy update
- `peer_joined` / `peer_left` - federation events
- `sync_received` - cross-node sync event
- `region_health` - region health update

## Effect Model

The kernel emits `Effect` values that are dispatched by the adapter layer:

| Effect Type | Description |
|-------------|-------------|
| `propose_action` | Propose a control action for execution |
| `update_score` | Node/transport score changed |
| `classify_node` | Node classification changed |
| `emit_alert` | Alert condition detected |
| `record_evidence` | Evidence bundle captured |
| `update_state` | State change notification |

## Scoring

### Node Scores
- `health_score` (0-1): based on observation frequency and anomaly history
- `trust_score` (0-1): based on behavior consistency
- `activity_score` (0-1): based on recent observation frequency
- `anomaly_score` (0-1): based on anomaly frequency and severity
- `composite_score`: weighted aggregate (health 0.4, trust 0.2, activity 0.2, inverse-anomaly 0.2)

### Classifications
- `healthy` (composite >= 0.8)
- `degraded` (0.5 <= composite < 0.8)
- `failing` (0.2 <= composite < 0.5)
- `dead` (composite < 0.2)

## Packages

| Package | Purpose |
|---------|---------|
| `internal/kernel` | Deterministic core, types, handlers, coordination, backpressure, durability |
| `internal/eventlog` | Append-only durable event stream |
| `internal/replay` | Deterministic replay engine |
| `internal/snapshot` | State checkpointing |
| `internal/federation` | Multi-instance coordination |
| `internal/region` | Region-aware operation |
| `internal/consistency` | Consistency model, conflict resolution, divergence detection, bounded staleness |
| `internal/service` | Kernel bridge, wiring, background workers |
| `cmd/mel` | CLI commands for distributed operations |
