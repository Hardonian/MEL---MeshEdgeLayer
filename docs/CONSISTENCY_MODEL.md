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

### Last-Write-Wins (LWW)

Used for:
- Node scores (newer timestamp wins)
- Transport scores (newer timestamp wins)
- Peer state updates

Safe because scores are soft state that gets recomputed.

### Score-Based Dominance

When two nodes disagree on classification:
- The node with more recent observations wins
- If tied, the lower (more conservative) score wins

### Policy Precedence

When policy versions diverge:
- Higher version number takes precedence
- Alerts are generated for operator review
- During transition, the more restrictive policy applies

### Operator Override Priority

Operator actions always take precedence:
- Manual approvals/rejections are authoritative
- Operator freezes propagate to all peers
- Operator notes are append-only (no conflict possible)

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
