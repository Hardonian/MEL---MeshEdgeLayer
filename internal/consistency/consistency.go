// Package consistency implements the MEL consistency model for federated
// distributed operation.
//
// The model provides:
//   - eventual consistency with bounded staleness
//   - monotonic state improvements (scores only improve or remain)
//   - explicit conflict resolution strategies
//   - divergence detection between nodes
//   - convergence verification
//
// Consistency guarantees:
//   - Within a single node: total ordering via sequence numbers
//   - Across nodes: causal ordering via Lamport clocks
//   - During partitions: bounded divergence with explicit detection
//   - After partition recovery: convergence via event replay union
//
// What MUST NOT diverge:
//   - Action lifecycle state (proposed → approved → executing → completed/failed)
//   - Active freeze scopes (safety-critical)
//   - Policy version (operator intent)
//
// What MAY diverge temporarily:
//   - Node scores (converge via observations)
//   - Transport health (converge via health events)
//   - Region health aggregates (recomputed periodically)
package consistency

import (
	"fmt"
	"time"

	"github.com/mel-project/mel/internal/kernel"
)

// ─── Resolution Strategies ───────────────────────────────────────────────────

// Strategy identifies a conflict resolution strategy.
type Strategy string

const (
	// StrategyLastWriteWins resolves by wall-clock timestamp (newest wins).
	// Safe for: scores, health metrics, non-critical metadata.
	StrategyLastWriteWins Strategy = "last_write_wins"

	// StrategyScoreDominance resolves by taking the worse (lower) score.
	// This is the conservative/safe approach: assume the degraded observation
	// is more likely correct than the healthy one.
	StrategyScoreDominance Strategy = "score_dominance"

	// StrategyPolicyPrecedence resolves by policy version ordering.
	// Higher/newer policy version takes precedence.
	StrategyPolicyPrecedence Strategy = "policy_precedence"

	// StrategyOperatorOverride resolves in favor of operator-initiated changes.
	// Any change from an operator event takes priority over automated ones.
	StrategyOperatorOverride Strategy = "operator_override"

	// StrategyUnionMerge combines both sides (e.g., both freezes kept).
	StrategyUnionMerge Strategy = "union_merge"
)

// ─── Divergence Detection ────────────────────────────────────────────────────

// DivergenceLevel classifies how severe a divergence is.
type DivergenceLevel string

const (
	DivergenceNone     DivergenceLevel = "none"
	DivergenceMinor    DivergenceLevel = "minor"    // score drift within tolerance
	DivergenceMajor    DivergenceLevel = "major"    // classification disagreement
	DivergenceCritical DivergenceLevel = "critical" // safety-affecting (freeze, policy, action lifecycle)
)

// Divergence records a specific inconsistency between two nodes' states.
type Divergence struct {
	Category    string          `json:"category"`
	Key         string          `json:"key"`
	LocalValue  string          `json:"local_value"`
	RemoteValue string          `json:"remote_value"`
	Level       DivergenceLevel `json:"level"`
	Strategy    Strategy        `json:"strategy"`
	Resolution  string          `json:"resolution"` // what was done
	Timestamp   time.Time       `json:"timestamp"`
}

// BoundedStaleness defines the acceptable staleness bounds.
type BoundedStaleness struct {
	// MaxClockDrift is the maximum allowed Lamport clock difference
	// between two nodes before staleness is flagged.
	MaxClockDrift uint64 `json:"max_clock_drift"`

	// MaxSequenceLag is the maximum allowed sequence number difference
	// between a peer's last synced seq and our current seq.
	MaxSequenceLag uint64 `json:"max_sequence_lag"`

	// MaxTimeDrift is the maximum wall-clock difference before staleness.
	MaxTimeDrift time.Duration `json:"max_time_drift"`
}

// DefaultBoundedStaleness returns production-safe defaults.
func DefaultBoundedStaleness() BoundedStaleness {
	return BoundedStaleness{
		MaxClockDrift:  1000,
		MaxSequenceLag: 500,
		MaxTimeDrift:   5 * time.Minute,
	}
}

// StalenessCheck evaluates whether a remote peer is within acceptable staleness bounds.
type StalenessCheck struct {
	PeerNodeID   string        `json:"peer_node_id"`
	ClockDrift   uint64        `json:"clock_drift"`
	SequenceLag  uint64        `json:"sequence_lag"`
	TimeDrift    time.Duration `json:"time_drift"`
	IsStale      bool          `json:"is_stale"`
	StaleReasons []string      `json:"stale_reasons,omitempty"`
	CheckedAt    time.Time     `json:"checked_at"`
}

// CheckStaleness evaluates whether a peer is within bounded staleness.
func CheckStaleness(
	localClock, remoteClock uint64,
	localSeq, remoteLastSyncSeq uint64,
	remoteLastSeen time.Time,
	bounds BoundedStaleness,
) StalenessCheck {
	now := time.Now().UTC()
	check := StalenessCheck{
		CheckedAt: now,
	}

	// Clock drift
	if localClock > remoteClock {
		check.ClockDrift = localClock - remoteClock
	} else {
		check.ClockDrift = remoteClock - localClock
	}
	if check.ClockDrift > bounds.MaxClockDrift {
		check.IsStale = true
		check.StaleReasons = append(check.StaleReasons,
			fmt.Sprintf("clock drift %d exceeds max %d", check.ClockDrift, bounds.MaxClockDrift))
	}

	// Sequence lag
	if localSeq > remoteLastSyncSeq {
		check.SequenceLag = localSeq - remoteLastSyncSeq
	}
	if check.SequenceLag > bounds.MaxSequenceLag {
		check.IsStale = true
		check.StaleReasons = append(check.StaleReasons,
			fmt.Sprintf("sequence lag %d exceeds max %d", check.SequenceLag, bounds.MaxSequenceLag))
	}

	// Time drift
	check.TimeDrift = now.Sub(remoteLastSeen)
	if check.TimeDrift > bounds.MaxTimeDrift {
		check.IsStale = true
		check.StaleReasons = append(check.StaleReasons,
			fmt.Sprintf("time drift %v exceeds max %v", check.TimeDrift, bounds.MaxTimeDrift))
	}

	return check
}

// ─── State Comparison and Conflict Resolution ────────────────────────────────

// CompareAndResolve compares local and remote states and produces a
// merged state using the appropriate resolution strategy for each category.
// Returns the list of divergences found and the resolved state.
func CompareAndResolve(local, remote *kernel.State) ([]Divergence, *kernel.State) {
	var divergences []Divergence
	now := time.Now().UTC()

	// Start with a copy of local state as the base
	resolved := copyState(local)

	// ─── Node Scores: score_dominance (take worse score) ─────────────
	for id, remoteScore := range remote.NodeScores {
		localScore, exists := resolved.NodeScores[id]
		if !exists {
			// Remote has a node we don't — union merge
			resolved.NodeScores[id] = remoteScore
			divergences = append(divergences, Divergence{
				Category:    "node_score",
				Key:         id,
				LocalValue:  "missing",
				RemoteValue: remoteScore.Classification,
				Level:       DivergenceMinor,
				Strategy:    StrategyUnionMerge,
				Resolution:  "added from remote",
				Timestamp:   now,
			})
			continue
		}

		// If classifications differ, this is major
		if localScore.Classification != remoteScore.Classification {
			divergences = append(divergences, Divergence{
				Category:    "node_score",
				Key:         id + ".classification",
				LocalValue:  localScore.Classification,
				RemoteValue: remoteScore.Classification,
				Level:       DivergenceMajor,
				Strategy:    StrategyScoreDominance,
				Timestamp:   now,
			})
		}

		// Score dominance: take the worse (lower) composite score
		if remoteScore.CompositeScore < localScore.CompositeScore {
			localScore.CompositeScore = remoteScore.CompositeScore
			localScore.Classification = remoteScore.Classification
			localScore.AnomalyScore = maxFloat(localScore.AnomalyScore, remoteScore.AnomalyScore)
			localScore.HealthScore = minFloat(localScore.HealthScore, remoteScore.HealthScore)
			resolved.NodeScores[id] = localScore
			divergences[len(divergences)-1].Resolution = "took worse score from remote"
		} else {
			if len(divergences) > 0 && divergences[len(divergences)-1].Key == id+".classification" {
				divergences[len(divergences)-1].Resolution = "kept local (worse or equal)"
			}
		}
	}

	// ─── Transport Scores: score_dominance ────────────────────────────
	for name, remoteTS := range remote.TransportScores {
		localTS, exists := resolved.TransportScores[name]
		if !exists {
			resolved.TransportScores[name] = remoteTS
			divergences = append(divergences, Divergence{
				Category:    "transport_score",
				Key:         name,
				LocalValue:  "missing",
				RemoteValue: fmt.Sprintf("%.3f", remoteTS.HealthScore),
				Level:       DivergenceMinor,
				Strategy:    StrategyUnionMerge,
				Resolution:  "added from remote",
				Timestamp:   now,
			})
			continue
		}
		if localTS.Classification != remoteTS.Classification {
			divergences = append(divergences, Divergence{
				Category:    "transport_score",
				Key:         name + ".classification",
				LocalValue:  localTS.Classification,
				RemoteValue: remoteTS.Classification,
				Level:       DivergenceMajor,
				Strategy:    StrategyScoreDominance,
				Timestamp:   now,
			})
			if remoteTS.HealthScore < localTS.HealthScore {
				localTS.HealthScore = remoteTS.HealthScore
				localTS.Classification = remoteTS.Classification
				localTS.AnomalyRate = maxFloat(localTS.AnomalyRate, remoteTS.AnomalyRate)
				resolved.TransportScores[name] = localTS
				divergences[len(divergences)-1].Resolution = "took worse score from remote"
			} else {
				divergences[len(divergences)-1].Resolution = "kept local (worse or equal)"
			}
		}
	}

	// ─── Action States: safety-critical, last-write-wins with operator override ─
	for id, remoteAction := range remote.ActionStates {
		localAction, exists := resolved.ActionStates[id]
		if !exists {
			resolved.ActionStates[id] = remoteAction
			divergences = append(divergences, Divergence{
				Category:    "action_state",
				Key:         id,
				LocalValue:  "missing",
				RemoteValue: remoteAction.Lifecycle,
				Level:       DivergenceCritical,
				Strategy:    StrategyUnionMerge,
				Resolution:  "added from remote",
				Timestamp:   now,
			})
			continue
		}
		if localAction.Lifecycle != remoteAction.Lifecycle {
			divergences = append(divergences, Divergence{
				Category:    "action_state",
				Key:         id + ".lifecycle",
				LocalValue:  localAction.Lifecycle,
				RemoteValue: remoteAction.Lifecycle,
				Level:       DivergenceCritical,
				Strategy:    StrategyLastWriteWins,
				Timestamp:   now,
			})
			// Take the more advanced lifecycle state
			if lifecycleOrder(remoteAction.Lifecycle) > lifecycleOrder(localAction.Lifecycle) {
				resolved.ActionStates[id] = remoteAction
				divergences[len(divergences)-1].Resolution = "took more advanced lifecycle from remote"
			} else {
				divergences[len(divergences)-1].Resolution = "kept local (more advanced lifecycle)"
			}
		}
	}

	// ─── Active Freezes: union_merge (safety-critical) ────────────────
	for id, remoteFreeze := range remote.ActiveFreezes {
		if _, exists := resolved.ActiveFreezes[id]; !exists {
			resolved.ActiveFreezes[id] = remoteFreeze
			divergences = append(divergences, Divergence{
				Category:    "active_freeze",
				Key:         id,
				LocalValue:  "missing",
				RemoteValue: remoteFreeze.Reason,
				Level:       DivergenceCritical,
				Strategy:    StrategyUnionMerge,
				Resolution:  "added freeze from remote (safety: keep all freezes)",
				Timestamp:   now,
			})
		}
	}

	// ─── Policy Version: policy_precedence ─────────────────────────────
	if local.PolicyVersion != remote.PolicyVersion && remote.PolicyVersion != "" {
		divergences = append(divergences, Divergence{
			Category:    "policy_version",
			Key:         "policy_version",
			LocalValue:  local.PolicyVersion,
			RemoteValue: remote.PolicyVersion,
			Level:       DivergenceCritical,
			Strategy:    StrategyPolicyPrecedence,
			Timestamp:   now,
		})
		// Take higher version
		if remote.PolicyVersion > local.PolicyVersion {
			resolved.PolicyVersion = remote.PolicyVersion
			divergences[len(divergences)-1].Resolution = "adopted remote policy version"
		} else {
			divergences[len(divergences)-1].Resolution = "kept local policy version"
		}
	}

	// ─── Node Registry: union_merge ───────────────────────────────────
	for nodeNum, remoteInfo := range remote.NodeRegistry {
		localInfo, exists := resolved.NodeRegistry[nodeNum]
		if !exists {
			resolved.NodeRegistry[nodeNum] = remoteInfo
			continue
		}
		// LWW for registry: take most recent last_seen
		if remoteInfo.LastSeen.After(localInfo.LastSeen) {
			resolved.NodeRegistry[nodeNum] = remoteInfo
		}
	}

	// ─── Region Health: last_write_wins ───────────────────────────────
	for regionID, remoteHealth := range remote.RegionHealth {
		localHealth, exists := resolved.RegionHealth[regionID]
		if !exists {
			resolved.RegionHealth[regionID] = remoteHealth
			continue
		}
		if remoteHealth.LastUpdateAt.After(localHealth.LastUpdateAt) {
			resolved.RegionHealth[regionID] = remoteHealth
		}
	}

	// Advance logical clock to max of both
	if remote.LogicalClock > resolved.LogicalClock {
		resolved.LogicalClock = remote.LogicalClock
	}

	return divergences, resolved
}

// ─── Convergence Verification ────────────────────────────────────────────────

// ConvergenceReport summarizes whether two nodes have converged.
type ConvergenceReport struct {
	Converged       bool         `json:"converged"`
	DivergenceCount int          `json:"divergence_count"`
	CriticalCount   int          `json:"critical_count"`
	MajorCount      int          `json:"major_count"`
	MinorCount      int          `json:"minor_count"`
	Divergences     []Divergence `json:"divergences,omitempty"`
	CheckedAt       time.Time    `json:"checked_at"`
}

// CheckConvergence compares two states and reports whether they have converged.
func CheckConvergence(stateA, stateB *kernel.State) ConvergenceReport {
	divergences, _ := CompareAndResolve(stateA, stateB)

	report := ConvergenceReport{
		Divergences:     divergences,
		DivergenceCount: len(divergences),
		CheckedAt:       time.Now().UTC(),
	}

	for _, d := range divergences {
		switch d.Level {
		case DivergenceCritical:
			report.CriticalCount++
		case DivergenceMajor:
			report.MajorCount++
		case DivergenceMinor:
			report.MinorCount++
		}
	}

	report.Converged = report.CriticalCount == 0 && report.MajorCount == 0
	return report
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func lifecycleOrder(lifecycle string) int {
	switch lifecycle {
	case "proposed":
		return 1
	case "approved":
		return 2
	case "running":
		return 3
	case "completed":
		return 4
	case "rejected":
		return 4 // terminal states are equivalent
	default:
		return 0
	}
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func copyState(s *kernel.State) *kernel.State {
	cp := kernel.NewState()
	cp.PolicyVersion = s.PolicyVersion
	cp.LastEventID = s.LastEventID
	cp.LastSequenceNum = s.LastSequenceNum
	cp.LogicalClock = s.LogicalClock

	for k, v := range s.NodeScores {
		cp.NodeScores[k] = v
	}
	for k, v := range s.TransportScores {
		cp.TransportScores[k] = v
	}
	for k, v := range s.ActionStates {
		cp.ActionStates[k] = v
	}
	for k, v := range s.ActiveFreezes {
		cp.ActiveFreezes[k] = v
	}
	for k, v := range s.NodeRegistry {
		cp.NodeRegistry[k] = v
	}
	for k, v := range s.RegionHealth {
		cp.RegionHealth[k] = v
	}
	return cp
}
