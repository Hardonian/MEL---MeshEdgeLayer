# MEL Consistency Model

## Overview

MEL uses eventual consistency with bounded staleness for its distributed state. This document defines exactly what can diverge, what must not diverge, and how divergence is detected and resolved.

## Consistency Guarantees

### Eventual Consistency with Bounded Staleness

- Nodes will converge to the same state given the same events
- Staleness is bounded by sync interval (default: 60 seconds)
- During partitions, staleness is unbounded but safe (restricted operations)

### Monotonic State Improvements

- Scores can only improve relative to their inputs (no spurious degradation)
- Node classifications move in predictable directions
- Policy changes are versioned and totally ordered

## What Can Diverge

| State | Can Diverge? | Notes |
|-------|-------------|-------|
| Node scores | Yes (temporarily) | Converge on sync |
| Transport scores | Yes (temporarily) | Converge on sync |
| Node classifications | Yes (temporarily) | Follows score convergence |
| Event log order | Yes (across nodes) | Each node has local ordering |
| Logical clock | Yes | Converges via Lamport protocol |
| Action proposals | Yes | Different nodes may propose different actions |

## What Must NOT Diverge

| State | Why |
|-------|-----|
| Policy version (long-term) | Divergent policy is a conflict |
| Action execution outcome | Actions are idempotent and coordinated |
| Evidence integrity | Checksums prevent tampering |
| Event content | Checksum verification prevents corruption |
| Operator approvals/rejections | These are authoritative operations |

## Conflict Resolution Strategies

Implemented in `internal/consistency/consistency.go` with `CompareAndResolve()`.

### Last-Write-Wins (LWW) — `StrategyLastWriteWins`

Used for:
- Node registry entries (newer `last_seen` wins)
- Region health aggregates (newer `last_update_at` wins)
- Peer state updates

Safe because these are soft state that gets recomputed.

### Score-Based Dominance — `StrategyScoreDominance`

Used for:
- Node scores (take the **worse** composite score — conservative/safe)
- Transport scores (take the **worse** health score)

When two nodes disagree on classification:
- The lower (more degraded) score wins
- Higher anomaly scores are preserved
- Lower health scores are preserved
- This ensures safety: if any node sees degradation, all nodes respect it

### Policy Precedence — `StrategyPolicyPrecedence`

When policy versions diverge:
- Higher version string takes precedence
- Alerts are generated for operator review
- During transition, the more restrictive policy applies

### Operator Override Priority — `StrategyOperatorOverride`

Operator actions always take precedence:
- Manual approvals/rejections are authoritative
- Operator freezes propagate to all peers
- Operator notes are append-only (no conflict possible)

### Union Merge — `StrategyUnionMerge`

Used for safety-critical state:
- **Active freezes**: both local and remote freezes are kept (safety: never drop a freeze)
- **New nodes**: nodes known only to remote are added locally
- **New transports**: transports known only to remote are added locally

### Action Lifecycle Advancement

Action state conflicts use lifecycle ordering:
- `proposed(1) → approved(2) → running(3) → completed/rejected(4)`
- The more advanced lifecycle state always wins
- This prevents actions from regressing to earlier states

## Bounded Staleness — `CheckStaleness()`

Three dimensions are checked:
- **Clock drift**: max Lamport clock difference (default: 1000)
- **Sequence lag**: max event sequence lag (default: 500)
- **Time drift**: max wall-clock time since last peer contact (default: 5 minutes)

A peer is marked stale if **any** dimension exceeds its bound.

## Divergence Detection

### Automatic
- Heartbeats carry policy version → divergence detected immediately
- Event checksums verified on sync → corruption detected
- Duplicate event IDs detected on ingestion → dedup applied

### Periodic
- Partition checker runs every 2× heartbeat interval
- Score comparison on sync responses
- Snapshot hash comparison (optional)

## Split-Brain Behavior

During detected split-brain:

1. Each partition operates independently
2. Autopilot actions are restricted (configurable)
3. Autonomous action count is tracked per-node
4. Operator alerts are generated
5. On reconnection:
   - Events are synced bidirectionally
   - Conflicting actions are detected
   - Operator may need to resolve conflicts manually

## Bounded Growth

To prevent unbounded state growth:
- Event log retention (configurable, default 14 days)
- Snapshot rotation (keep last N, default 10)
- Backup pruning (keep last N backups)
- Dedup map bounded to 100K entries with LRU eviction
- Action states pruned on completion
- Freeze states pruned on expiry
