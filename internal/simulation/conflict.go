package simulation

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/status"
	"github.com/mel-project/mel/internal/transport"
)

// ConflictType categorizes the type of conflict detected.
type ConflictType string

const (
	// ConflictTypeAction indicates conflicting actions on the same target
	ConflictTypeAction ConflictType = "action_conflict"
	// ConflictTypeDuplicate indicates a duplicate or recently executed action
	ConflictTypeDuplicate ConflictType = "duplicate_action"
	// ConflictTypeSequence indicates a violation of recovery sequence
	ConflictTypeSequence ConflictType = "sequence_violation"
	// ConflictTypeStaleData indicates action based on stale data
	ConflictTypeStaleData ConflictType = "stale_data"
	// ConflictTypeSelfDegradation indicates action during MEL self-degradation
	ConflictTypeSelfDegradation ConflictType = "self_degradation"
	// ConflictTypeCooldown indicates timing/cooldown violation
	ConflictTypeCooldown ConflictType = "cooldown_violation"
	// ConflictTypeDependency indicates missing dependency for action
	ConflictTypeDependency ConflictType = "dependency_conflict"
	// ConflictTypeSafety indicates safety conflict with current state
	ConflictTypeSafety ConflictType = "safety_conflict"
)

// ConflictDetector provides comprehensive conflict detection for control actions.
// It analyzes proposed actions against historical actions, current mesh state,
// data freshness, and recovery sequences to identify potential conflicts.
type ConflictDetector struct {
	// cooldownWindow defines the minimum time between actions on the same target
	cooldownWindow time.Duration
	// duplicateWindow defines how far back to look for duplicate actions
	duplicateWindow time.Duration
	// staleDataThreshold defines when data is considered too old for decision-making
	staleDataThreshold time.Duration
	// maxActionsPerTarget limits actions on a single target within the window
	maxActionsPerTarget int
}

// ConflictResult represents a single detected conflict with full context.
type ConflictResult struct {
	// ConflictID uniquely identifies this conflict instance
	ConflictID string `json:"conflict_id"`
	// Severity indicates the severity level of the conflict
	Severity ConflictSeverity `json:"severity"`
	// Type categorizes the conflict
	Type ConflictType `json:"type"`
	// Description provides human-readable explanation
	Description string `json:"description"`
	// Resource identifies the affected resource (transport, node, segment)
	Resource string `json:"resource,omitempty"`
	// ConflictingActionID references the action causing conflict (if applicable)
	ConflictingActionID string `json:"conflicting_action_id,omitempty"`
	// Resolution provides recommendation for resolving the conflict
	Resolution string `json:"resolution"`
	// Timestamp when the conflict was detected
	Timestamp time.Time `json:"timestamp"`
	// Metadata holds additional conflict-specific data
	Metadata map[string]any `json:"metadata,omitempty"`
}

// NewConflictDetector creates a new conflict detector with default settings.
func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{
		cooldownWindow:      30 * time.Second,
		duplicateWindow:     5 * time.Minute,
		staleDataThreshold:  2 * time.Minute,
		maxActionsPerTarget: 3,
	}
}

// NewConflictDetectorWithOptions creates a conflict detector with custom settings.
func NewConflictDetectorWithOptions(
	cooldownWindow,
	duplicateWindow,
	staleDataThreshold time.Duration,
	maxActionsPerTarget int,
) *ConflictDetector {
	return &ConflictDetector{
		cooldownWindow:      cooldownWindow,
		duplicateWindow:     duplicateWindow,
		staleDataThreshold:  staleDataThreshold,
		maxActionsPerTarget: maxActionsPerTarget,
	}
}

// DetectConflicts performs comprehensive conflict detection for a proposed action.
// It checks for:
//   - Conflicting actions (same target, incompatible types)
//   - Duplicate/recent actions
//   - Recovery sequence violations
//   - Stale data
//   - MEL self-degradation
//   - Cooldown violations
//   - Dependency conflicts
//   - Safety conflicts
func (cd *ConflictDetector) DetectConflicts(
	proposed control.ControlAction,
	actionHistory []db.ControlActionRecord,
	mesh status.MeshDrilldown,
	freshnessReports []models.FreshnessReport,
) []ConflictResult {
	conflicts := []ConflictResult{}
	timestamp := time.Now().UTC()

	// Check for conflicting active actions
	conflicts = append(conflicts, cd.detectConflictingActions(proposed, actionHistory, timestamp)...)

	// Check for duplicate/recent actions
	conflicts = append(conflicts, cd.detectDuplicateActions(proposed, actionHistory, timestamp)...)

	// Check for recovery sequence violations
	conflicts = append(conflicts, cd.detectSequenceViolations(proposed, actionHistory, timestamp)...)

	// Check for stale data
	conflicts = append(conflicts, cd.detectStaleData(proposed, freshnessReports, timestamp)...)

	// Check for MEL self-degradation
	conflicts = append(conflicts, cd.detectSelfDegradation(proposed, mesh, timestamp)...)

	// Check for cooldown violations
	conflicts = append(conflicts, cd.detectCooldownViolations(proposed, actionHistory, timestamp)...)

	// Check for dependency conflicts
	conflicts = append(conflicts, cd.detectDependencyConflicts(proposed, actionHistory, mesh, timestamp)...)

	// Check for safety conflicts
	conflicts = append(conflicts, cd.detectSafetyConflicts(proposed, mesh, timestamp)...)

	// Sort conflicts by severity (most severe first)
	sort.Slice(conflicts, func(i, j int) bool {
		return severityRank(conflicts[i].Severity) > severityRank(conflicts[j].Severity)
	})

	return conflicts
}

// detectConflictingActions finds actions that conflict with the proposed action
// due to incompatible types on the same target.
func (cd *ConflictDetector) detectConflictingActions(
	proposed control.ControlAction,
	history []db.ControlActionRecord,
	timestamp time.Time,
) []ConflictResult {
	conflicts := []ConflictResult{}

	// Define incompatible action pairs
	incompatiblePairs := map[string][]string{
		control.ActionRestartTransport: {
			control.ActionResubscribeTransport,
			control.ActionBackoffReset,
		},
		control.ActionResubscribeTransport: {
			control.ActionRestartTransport,
		},
		control.ActionBackoffIncrease: {
			control.ActionBackoffReset,
		},
		control.ActionBackoffReset: {
			control.ActionBackoffIncrease,
			control.ActionRestartTransport,
		},
	}

	incompatibleTypes, hasIncompatibles := incompatiblePairs[proposed.ActionType]
	if !hasIncompatibles {
		return conflicts
	}

	for _, action := range history {
		// Skip if different target
		if !cd.sameTarget(proposed, action) {
			continue
		}

		// Skip completed/inactive actions
		if !cd.isActionActive(action, timestamp) {
			continue
		}

		// Check if action type is incompatible
		for _, incompatibleType := range incompatibleTypes {
			if action.ActionType == incompatibleType {
				conflicts = append(conflicts, ConflictResult{
					ConflictID:          fmt.Sprintf("conflict-action-%s-%s", proposed.ID, action.ID),
					Severity:            ConflictSeverityMajor,
					Type:                ConflictTypeAction,
					Description:         fmt.Sprintf("Incompatible action types: %s conflicts with active %s on %s", proposed.ActionType, action.ActionType, cd.targetString(proposed)),
					Resource:            cd.targetString(proposed),
					ConflictingActionID: action.ID,
					Resolution:          fmt.Sprintf("Wait for %s to complete or cancel it before proceeding", action.ActionType),
					Timestamp:           timestamp,
					Metadata: map[string]any{
						"proposed_type":    proposed.ActionType,
						"conflicting_type": action.ActionType,
						"target":           cd.targetString(proposed),
					},
				})
			}
		}
	}

	return conflicts
}

// detectDuplicateActions finds duplicate or recently executed actions.
func (cd *ConflictDetector) detectDuplicateActions(
	proposed control.ControlAction,
	history []db.ControlActionRecord,
	timestamp time.Time,
) []ConflictResult {
	conflicts := []ConflictResult{}

	for _, action := range history {
		// Skip if different target
		if !cd.sameTarget(proposed, action) {
			continue
		}

		// Skip if different action type
		if action.ActionType != proposed.ActionType {
			continue
		}

		// Parse action timestamp
		actionTime, ok := cd.parseTime(action.CreatedAt)
		if !ok {
			continue
		}

		// Check if within duplicate window
		if timestamp.Sub(actionTime) > cd.duplicateWindow {
			continue
		}

		// Determine severity based on time proximity
		severity := ConflictSeverityMinor
		timeSince := timestamp.Sub(actionTime)
		if timeSince < cd.cooldownWindow {
			severity = ConflictSeverityModerate
		}
		if timeSince < 30*time.Second {
			severity = ConflictSeverityMajor
		}

		conflicts = append(conflicts, ConflictResult{
			ConflictID:          fmt.Sprintf("conflict-duplicate-%s-%s", proposed.ID, action.ID),
			Severity:            severity,
			Type:                ConflictTypeDuplicate,
			Description:         fmt.Sprintf("Duplicate %s action on %s was executed %v ago", proposed.ActionType, cd.targetString(proposed), timeSince.Round(time.Second)),
			Resource:            cd.targetString(proposed),
			ConflictingActionID: action.ID,
			Resolution:          "Review if the previous action achieved the desired outcome before repeating",
			Timestamp:           timestamp,
			Metadata: map[string]any{
				"action_type":      proposed.ActionType,
				"previous_result":  action.Result,
				"time_since_last":  timeSince.String(),
				"duplicate_window": cd.duplicateWindow.String(),
			},
		})
	}

	return conflicts
}

// detectSequenceViolations checks if the action violates required recovery sequences.
func (cd *ConflictDetector) detectSequenceViolations(
	proposed control.ControlAction,
	history []db.ControlActionRecord,
	timestamp time.Time,
) []ConflictResult {
	conflicts := []ConflictResult{}

	// Define required sequences: action -> required prerequisite
	requiredSequences := map[string][]string{
		control.ActionBackoffReset: {
			control.ActionBackoffIncrease,
		},
		control.ActionClearSuppression: {
			control.ActionTemporarilySuppressNoisySource,
		},
	}

	requiredPrereqs, hasPrereqs := requiredSequences[proposed.ActionType]
	if !hasPrereqs {
		return conflicts
	}

	// Check if any prerequisite was executed recently and successfully
	prereqFound := false
	for _, action := range history {
		if !cd.sameTarget(proposed, action) {
			continue
		}

		for _, prereq := range requiredPrereqs {
			if action.ActionType != prereq {
				continue
			}

			actionTime, ok := cd.parseTime(action.CreatedAt)
			if !ok {
				continue
			}

			// Prerequisite must be within reasonable window and successful
			if timestamp.Sub(actionTime) < 30*time.Minute &&
				(action.Result == control.ResultExecutedSuccessfully || action.Result == control.ResultExecutedNoop) {
				prereqFound = true
				break
			}
		}
		if prereqFound {
			break
		}
	}

	if !prereqFound {
		conflicts = append(conflicts, ConflictResult{
			ConflictID:  fmt.Sprintf("conflict-sequence-%s", proposed.ID),
			Severity:    ConflictSeverityModerate,
			Type:        ConflictTypeSequence,
			Description: fmt.Sprintf("%s requires %s to be executed first on %s", proposed.ActionType, strings.Join(requiredPrereqs, " or "), cd.targetString(proposed)),
			Resource:    cd.targetString(proposed),
			Resolution:  fmt.Sprintf("Execute %s before attempting %s", strings.Join(requiredPrereqs, " or "), proposed.ActionType),
			Timestamp:   timestamp,
			Metadata: map[string]any{
				"action_type":            proposed.ActionType,
				"required_prerequisites": requiredPrereqs,
			},
		})
	}

	return conflicts
}

// detectStaleData checks if the action is based on stale data.
func (cd *ConflictDetector) detectStaleData(
	proposed control.ControlAction,
	freshnessReports []models.FreshnessReport,
	timestamp time.Time,
) []ConflictResult {
	conflicts := []ConflictResult{}

	// Map action types to components they depend on
	componentDependencies := map[string][]string{
		control.ActionRestartTransport:               {"transport", "health"},
		control.ActionResubscribeTransport:           {"transport", "subscribe"},
		control.ActionBackoffIncrease:                {"transport", "metrics"},
		control.ActionBackoffReset:                   {"transport", "metrics"},
		control.ActionTemporarilyDeprioritize:        {"routing", "health"},
		control.ActionTemporarilySuppressNoisySource: {"ingest", "classify"},
		control.ActionTriggerHealthRecheck:           {"health", "control"},
	}

	dependencies, hasDeps := componentDependencies[proposed.ActionType]
	if !hasDeps {
		return conflicts
	}

	for _, report := range freshnessReports {
		// Check if this component is relevant to our action
		isRelevant := false
		for _, dep := range dependencies {
			if strings.Contains(strings.ToLower(report.Component), dep) {
				isRelevant = true
				break
			}
		}
		if !isRelevant {
			continue
		}

		// Check if data is stale
		if report.Status != "fresh" && report.Status != "unknown" {
			age := time.Duration(report.AgeSeconds) * time.Second
			severity := ConflictSeverityMinor
			if age > cd.staleDataThreshold*2 {
				severity = ConflictSeverityModerate
			}
			if age > cd.staleDataThreshold*4 {
				severity = ConflictSeverityMajor
			}

			conflicts = append(conflicts, ConflictResult{
				ConflictID:  fmt.Sprintf("conflict-stale-%s-%s", proposed.ID, report.Component),
				Severity:    severity,
				Type:        ConflictTypeStaleData,
				Description: fmt.Sprintf("%s data is %s (age: %v), which may affect %s action reliability", report.Component, report.Status, age.Round(time.Second), proposed.ActionType),
				Resource:    report.Component,
				Resolution:  fmt.Sprintf("Wait for fresh %s data or verify current state through alternative means", report.Component),
				Timestamp:   timestamp,
				Metadata: map[string]any{
					"component":      report.Component,
					"data_status":    report.Status,
					"data_age":       report.AgeSeconds,
					"action_type":    proposed.ActionType,
					"last_update":    report.LastUpdate,
				},
			})
		}
	}

	return conflicts
}

// detectSelfDegradation checks if MEL itself is degraded.
func (cd *ConflictDetector) detectSelfDegradation(
	proposed control.ControlAction,
	mesh status.MeshDrilldown,
	timestamp time.Time,
) []ConflictResult {
	conflicts := []ConflictResult{}

	// Check mesh health score
	if mesh.MeshHealth.Score < 30 {
		conflicts = append(conflicts, ConflictResult{
			ConflictID:  fmt.Sprintf("conflict-self-degradation-%s", proposed.ID),
			Severity:    ConflictSeverityCritical,
			Type:        ConflictTypeSelfDegradation,
			Description: fmt.Sprintf("MEL is in severe degradation state (health score: %d). Executing %s may be unsafe.", mesh.MeshHealth.Score, proposed.ActionType),
			Resource:    "mel-system",
			Resolution:  "Address MEL health issues first: check for evidence loss, connectivity problems, or internal saturation",
			Timestamp:   timestamp,
			Metadata: map[string]any{
				"mesh_health_score": mesh.MeshHealth.Score,
				"mesh_state":        mesh.MeshHealth.State,
				"action_type":       proposed.ActionType,
				"recovery_blockers": mesh.MeshHealthExplanation.RecoveryBlockers,
			},
		})
	} else if mesh.MeshHealth.Score < 50 {
		conflicts = append(conflicts, ConflictResult{
			ConflictID:  fmt.Sprintf("conflict-self-degradation-%s", proposed.ID),
			Severity:    ConflictSeverityMajor,
			Type:        ConflictTypeSelfDegradation,
			Description: fmt.Sprintf("MEL is degraded (health score: %d). Exercise caution with %s.", mesh.MeshHealth.Score, proposed.ActionType),
			Resource:    "mel-system",
			Resolution:  "Consider waiting for mesh recovery or use advisory-only mode",
			Timestamp:   timestamp,
			Metadata: map[string]any{
				"mesh_health_score": mesh.MeshHealth.Score,
				"mesh_state":        mesh.MeshHealth.State,
				"action_type":       proposed.ActionType,
			},
		})
	}

	// Check for critical degraded segments
	for _, segment := range mesh.MeshHealth.CriticalSegments {
		conflicts = append(conflicts, ConflictResult{
			ConflictID:  fmt.Sprintf("conflict-critical-segment-%s-%s", proposed.ID, segment.SegmentID),
			Severity:    ConflictSeverityMajor,
			Type:        ConflictTypeSelfDegradation,
			Description: fmt.Sprintf("Critical segment %s affects %s. Action %s may be impacted.", segment.SegmentID, strings.Join(segment.Transports, ", "), proposed.ActionType),
			Resource:    segment.SegmentID,
			Resolution:  fmt.Sprintf("Address critical segment issue: %s", segment.Explanation),
			Timestamp:   timestamp,
			Metadata: map[string]any{
				"segment_id":    segment.SegmentID,
				"severity":      segment.Severity,
				"reason":        segment.Reason,
				"affected_transports": segment.Transports,
			},
		})
	}

	return conflicts
}

// detectCooldownViolations checks for timing-based cooldown violations.
func (cd *ConflictDetector) detectCooldownViolations(
	proposed control.ControlAction,
	history []db.ControlActionRecord,
	timestamp time.Time,
) []ConflictResult {
	conflicts := []ConflictResult{}

	// Count recent actions on this target
	recentActionCount := 0
	var mostRecentAction time.Time

	for _, action := range history {
		if !cd.sameTarget(proposed, action) {
			continue
		}

		actionTime, ok := cd.parseTime(action.CreatedAt)
		if !ok {
			continue
		}

		// Check if within cooldown window
		if timestamp.Sub(actionTime) <= cd.cooldownWindow {
			recentActionCount++
			if mostRecentAction.IsZero() || actionTime.After(mostRecentAction) {
				mostRecentAction = actionTime
			}
		}
	}

	if recentActionCount >= cd.maxActionsPerTarget {
		timeUntilClear := cd.cooldownWindow - timestamp.Sub(mostRecentAction)
		conflicts = append(conflicts, ConflictResult{
			ConflictID:  fmt.Sprintf("conflict-cooldown-%s", proposed.ID),
			Severity:    ConflictSeverityModerate,
			Type:        ConflictTypeCooldown,
			Description: fmt.Sprintf("Cooldown violation: %d actions on %s within %v (max: %d)", recentActionCount, cd.targetString(proposed), cd.cooldownWindow, cd.maxActionsPerTarget),
			Resource:    cd.targetString(proposed),
			Resolution:  fmt.Sprintf("Wait %v before executing another action on this target", timeUntilClear.Round(time.Second)),
			Timestamp:   timestamp,
			Metadata: map[string]any{
				"action_count":      recentActionCount,
				"cooldown_window":   cd.cooldownWindow.String(),
				"max_per_target":    cd.maxActionsPerTarget,
				"time_until_clear":  timeUntilClear.String(),
				"most_recent_action": mostRecentAction.Format(time.RFC3339),
			},
		})
	}

	return conflicts
}

// detectDependencyConflicts checks for missing dependencies.
func (cd *ConflictDetector) detectDependencyConflicts(
	proposed control.ControlAction,
	history []db.ControlActionRecord,
	mesh status.MeshDrilldown,
	timestamp time.Time,
) []ConflictResult {
	conflicts := []ConflictResult{}

	// Action dependencies: action type -> list of required conditions
	actionDependencies := map[string][]func(control.ControlAction, status.MeshDrilldown, []db.ControlActionRecord) (bool, string){
		control.ActionTemporarilyDeprioritize: {
			// Requires healthy alternate path
			func(a control.ControlAction, m status.MeshDrilldown, _ []db.ControlActionRecord) (bool, string) {
				for _, rec := range m.RoutingRecommendations {
					if rec.Action == "suggest_alternate_ingest_path" && rec.TargetTransport == a.TargetTransport {
						return true, ""
					}
				}
				return false, "No healthy alternate transport path available"
			},
		},
		control.ActionRestartTransport: {
			// Should not restart if already restarting
			func(a control.ControlAction, m status.MeshDrilldown, h []db.ControlActionRecord) (bool, string) {
				for _, action := range h {
					if action.TargetTransport == a.TargetTransport &&
						action.ActionType == control.ActionRestartTransport &&
						action.LifecycleState == control.LifecycleRunning {
						return false, "Restart already in progress for this transport"
					}
				}
				return true, ""
			},
		},
	}

	deps, hasDeps := actionDependencies[proposed.ActionType]
	if !hasDeps {
		return conflicts
	}

	for _, check := range deps {
		passed, reason := check(proposed, mesh, history)
		if !passed {
			conflicts = append(conflicts, ConflictResult{
				ConflictID:  fmt.Sprintf("conflict-dependency-%s", proposed.ID),
				Severity:    ConflictSeverityMajor,
				Type:        ConflictTypeDependency,
				Description: fmt.Sprintf("Dependency not satisfied for %s on %s: %s", proposed.ActionType, cd.targetString(proposed), reason),
				Resource:    cd.targetString(proposed),
				Resolution:  fmt.Sprintf("Resolve dependency: %s", reason),
				Timestamp:   timestamp,
				Metadata: map[string]any{
					"action_type":       proposed.ActionType,
					"dependency_reason": reason,
				},
			})
		}
	}

	return conflicts
}

// detectSafetyConflicts checks for safety issues given current state.
func (cd *ConflictDetector) detectSafetyConflicts(
	proposed control.ControlAction,
	mesh status.MeshDrilldown,
	timestamp time.Time,
) []ConflictResult {
	conflicts := []ConflictResult{}

	// Check for active alerts that make action unsafe
	for _, alert := range mesh.ActiveAlerts {
		// Skip if alert is not for our target
		if proposed.TargetTransport != "" && alert.TransportName != proposed.TargetTransport {
			continue
		}

		// Define unsafe combinations
		unsafeCombinations := map[string][]string{
			control.ActionRestartTransport: {
				transport.ReasonSubscribeFailure,
			},
			control.ActionResubscribeTransport: {
				transport.ReasonRetryThresholdExceeded,
			},
		}

		unsafeReasons, hasUnsafe := unsafeCombinations[proposed.ActionType]
		if !hasUnsafe {
			continue
		}

		for _, unsafeReason := range unsafeReasons {
			if alert.Reason == unsafeReason {
				conflicts = append(conflicts, ConflictResult{
					ConflictID:  fmt.Sprintf("conflict-safety-%s-%s", proposed.ID, alert.ID),
					Severity:    ConflictSeverityMajor,
					Type:        ConflictTypeSafety,
					Description: fmt.Sprintf("Safety conflict: %s may be unsafe given active %s alert on %s", proposed.ActionType, alert.Reason, alert.TransportName),
					Resource:    alert.TransportName,
					Resolution:  fmt.Sprintf("Address the %s condition before executing %s", alert.Reason, proposed.ActionType),
					Timestamp:   timestamp,
					Metadata: map[string]any{
						"action_type":  proposed.ActionType,
						"alert_reason": alert.Reason,
						"alert_id":     alert.ID,
						"severity":     alert.Severity,
					},
				})
			}
		}
	}

	// Check for correlated failures that make action risky
	for _, failure := range mesh.CorrelatedFailures {
		// If action target is part of a correlated failure, warn
		for _, transportName := range failure.Transports {
			if proposed.TargetTransport == transportName {
				severity := ConflictSeverityModerate
				if failure.Severity == "critical" {
					severity = ConflictSeverityMajor
				}

				conflicts = append(conflicts, ConflictResult{
					ConflictID:  fmt.Sprintf("conflict-safety-correlated-%s", proposed.ID),
					Severity:    severity,
					Type:        ConflictTypeSafety,
					Description: fmt.Sprintf("Safety warning: %s is part of correlated failure '%s' affecting %d transports", proposed.TargetTransport, failure.Reason, len(failure.Transports)),
					Resource:    proposed.TargetTransport,
					Resolution:  "Consider mesh-level recovery instead of individual transport action",
					Timestamp:   timestamp,
					Metadata: map[string]any{
						"failure_reason":    failure.Reason,
						"affected_count":    len(failure.Transports),
						"failure_severity":  failure.Severity,
						"affected_nodes":    failure.NodeIDs,
					},
				})
			}
		}
	}

	return conflicts
}

// Helper methods

// sameTarget checks if two actions target the same resource.
func (cd *ConflictDetector) sameTarget(proposed control.ControlAction, existing db.ControlActionRecord) bool {
	// Check transport target
	if proposed.TargetTransport != "" && existing.TargetTransport != "" {
		return proposed.TargetTransport == existing.TargetTransport
	}

	// Check segment target
	if proposed.TargetSegment != "" && existing.TargetSegment != "" {
		return proposed.TargetSegment == existing.TargetSegment
	}

	// Check node target
	if proposed.TargetNode != "" && existing.TargetNode != "" {
		return proposed.TargetNode == existing.TargetNode
	}

	return false
}

// targetString returns a string representation of the action's target.
func (cd *ConflictDetector) targetString(action control.ControlAction) string {
	if action.TargetTransport != "" {
		return fmt.Sprintf("transport:%s", action.TargetTransport)
	}
	if action.TargetSegment != "" {
		return fmt.Sprintf("segment:%s", action.TargetSegment)
	}
	if action.TargetNode != "" {
		return fmt.Sprintf("node:%s", action.TargetNode)
	}
	return "unknown"
}

// isActionActive checks if an action is currently active (not completed/failed).
func (cd *ConflictDetector) isActionActive(action db.ControlActionRecord, now time.Time) bool {
	// Check lifecycle state
	if action.LifecycleState == control.LifecyclePending ||
		action.LifecycleState == control.LifecycleRunning {
		return true
	}

	// Check if result indicates completion
	if action.Result == control.ResultExecutedSuccessfully ||
		action.Result == control.ResultExecutedNoop {
		// Check if still within effect period (for reversible actions)
		if action.Reversible && action.ExpiresAt != "" {
			expires, ok := cd.parseTime(action.ExpiresAt)
			if ok && now.Before(expires) {
				return true
			}
		}
	}

	return false
}

// parseTime parses RFC3339 timestamps.
func (cd *ConflictDetector) parseTime(ts string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, strings.TrimSpace(ts)); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

// severityRank returns a numeric rank for severity comparison.
func severityRank(severity ConflictSeverity) int {
	switch severity {
	case ConflictSeverityCritical:
		return 5
	case ConflictSeverityMajor:
		return 4
	case ConflictSeverityModerate:
		return 3
	case ConflictSeverityMinor:
		return 2
	case ConflictSeverityNone:
		return 1
	default:
		return 0
	}
}

// HasCriticalConflicts returns true if any conflict is critical or major severity.
func HasCriticalConflicts(conflicts []ConflictResult) bool {
	for _, c := range conflicts {
		if c.Severity == ConflictSeverityCritical || c.Severity == ConflictSeverityMajor {
			return true
		}
	}
	return false
}

// CountConflictsByType returns a count of conflicts grouped by type.
func CountConflictsByType(conflicts []ConflictResult) map[ConflictType]int {
	counts := make(map[ConflictType]int)
	for _, c := range conflicts {
		counts[c.Type]++
	}
	return counts
}

// CountConflictsBySeverity returns a count of conflicts grouped by severity.
func CountConflictsBySeverity(conflicts []ConflictResult) map[ConflictSeverity]int {
	counts := make(map[ConflictSeverity]int)
	for _, c := range conflicts {
		counts[c.Severity]++
	}
	return counts
}

// FilterConflictsBySeverity returns conflicts filtered by minimum severity.
func FilterConflictsBySeverity(conflicts []ConflictResult, minSeverity ConflictSeverity) []ConflictResult {
	minRank := severityRank(minSeverity)
	var filtered []ConflictResult
	for _, c := range conflicts {
		if severityRank(c.Severity) >= minRank {
			filtered = append(filtered, c)
		}
	}
	return filtered
}
