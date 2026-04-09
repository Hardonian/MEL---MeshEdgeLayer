// Package shared provides standardized patterns for MEL
package shared

import (
	"fmt"
	"time"
)

// PostureLevel represents the overall health level
type PostureLevel string

const (
	PostureLevelHealthy   PostureLevel = "healthy"
	PostureLevelDegraded  PostureLevel = "degraded"
	PostureLevelUnhealthy PostureLevel = "unhealthy"
	PostureLevelUnknown   PostureLevel = "unknown"
)

// ComponentState represents individual component states
type ComponentState string

const (
	ComponentStateOK           ComponentState = "ok"
	ComponentStateHealthy      ComponentState = "healthy"
	ComponentStateDegraded     ComponentState = "degraded"
	ComponentStateUnhealthy    ComponentState = "unhealthy"
	ComponentStateFailed       ComponentState = "failed"
	ComponentStateNotReady     ComponentState = "not_ready"
	ComponentStateIdle         ComponentState = "idle"
	ComponentStateUnknown      ComponentState = "unknown"
	ComponentStateDisabled     ComponentState = "disabled"
	ComponentStateMisconfigured ComponentState = "misconfigured"
)

// ComponentCriticality defines how critical a component is
type ComponentCriticality string

const (
	ComponentCriticalityCritical ComponentCriticality = "critical"
	ComponentCriticalityHigh     ComponentCriticality = "high"
	ComponentCriticalityMedium   ComponentCriticality = "medium"
	ComponentCriticalityLow      ComponentCriticality = "low"
	ComponentCriticalityOptional ComponentCriticality = "optional"
)

// ComponentPosture represents a single component's health
type ComponentPosture struct {
	ID                string               `json:"id"`
	Name              string               `json:"name"`
	Level             PostureLevel         `json:"level"`
	State             ComponentState       `json:"state"`
	ReasonCodes       []string             `json:"reasonCodes"`
	Detail            string               `json:"detail"`
	CheckedAt         time.Time            `json:"checkedAt"`
	LastActivityAt    *time.Time           `json:"lastActivityAt,omitempty"`
	Stale             bool                 `json:"stale"`
	StaleThresholdMs  int64                `json:"staleThresholdMs,omitempty"`
}

// SystemPosture represents the overall system health
type SystemPosture struct {
	Overall         PostureLevel       `json:"overall"`
	Summary         string             `json:"summary"`
	ReasonCodes     []string           `json:"reasonCodes"`
	Components      []ComponentPosture `json:"components"`
	Degraded        bool               `json:"degraded"`
	DegradedReasons []string           `json:"degradedReasons"`
	FailClosed      bool               `json:"failClosed"`
	CheckedAt       time.Time          `json:"checkedAt"`
}

// DegradationThresholds defines when to mark components/system degraded
type DegradationThresholds struct {
	ErrorRatePercent         float64 `json:"errorRatePercent"`
	StaleAgeMs               int64   `json:"staleAgeMs"`
	FailureCount             int     `json:"failureCount"`
	CriticalComponentFailed  bool    `json:"criticalComponentFailed"`
	DegradedComponentRatio   float64 `json:"degradedComponentRatio"`
	AnyCriticalDegraded      bool    `json:"anyCriticalDegraded"`
}

// DefaultDegradationThresholds provides sensible defaults
var DefaultDegradationThresholds = DegradationThresholds{
	ErrorRatePercent:        5.0,
	StaleAgeMs:              120000, // 2 minutes
	FailureCount:            10,
	CriticalComponentFailed: true,
	DegradedComponentRatio:  0.5,
	AnyCriticalDegraded:     true,
}

// ComponentDefinition provides metadata about a component
type ComponentDefinition struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	Criticality  ComponentCriticality `json:"criticality"`
	Category     string               `json:"category"`
	Dependencies []string             `json:"dependencies,omitempty"`
}

// ComponentStateToLevel converts a component state to a posture level
func ComponentStateToLevel(state ComponentState) PostureLevel {
	switch state {
	case ComponentStateOK, ComponentStateHealthy:
		return PostureLevelHealthy
	case ComponentStateDegraded, ComponentStateIdle, ComponentStateMisconfigured:
		return PostureLevelDegraded
	case ComponentStateFailed, ComponentStateUnhealthy, ComponentStateNotReady:
		return PostureLevelUnhealthy
	case ComponentStateUnknown, ComponentStateDisabled:
		return PostureLevelUnknown
	default:
		return PostureLevelUnknown
	}
}

// BuildComponentPosture creates a component posture from state
func BuildComponentPosture(
	definition ComponentDefinition,
	state ComponentState,
	detail string,
	opts ...ComponentPostureOption,
) ComponentPosture {
	now := time.Now().UTC()
	
	posture := ComponentPosture{
		ID:               definition.ID,
		Name:             definition.Name,
		Level:            ComponentStateToLevel(state),
		State:            state,
		ReasonCodes:      []string{},
		Detail:           detail,
		CheckedAt:        now,
		LastActivityAt:   &now,
		Stale:            false,
		StaleThresholdMs: DefaultDegradationThresholds.StaleAgeMs,
	}
	
	// Apply options
	for _, opt := range opts {
		opt(&posture)
	}
	
	// Check staleness
	if posture.LastActivityAt != nil {
		age := now.Sub(*posture.LastActivityAt).Milliseconds()
		posture.Stale = age > posture.StaleThresholdMs
	}
	
	// Add reason codes based on state
	if posture.Stale {
		posture.ReasonCodes = append(posture.ReasonCodes, "STALE_EVIDENCE")
	}
	if state == ComponentStateDegraded {
		posture.ReasonCodes = append(posture.ReasonCodes, "DEGRADED_STATE")
	}
	if state == ComponentStateFailed {
		posture.ReasonCodes = append(posture.ReasonCodes, "COMPONENT_FAILED")
	}
	
	return posture
}

// ComponentPostureOption configures a component posture
type ComponentPostureOption func(*ComponentPosture)

// WithLastActivity sets the last activity time
func WithLastActivity(t time.Time) ComponentPostureOption {
	return func(p *ComponentPosture) {
		p.LastActivityAt = &t
	}
}

// WithStaleThreshold sets the stale threshold
func WithStaleThreshold(ms int64) ComponentPostureOption {
	return func(p *ComponentPosture) {
		p.StaleThresholdMs = ms
	}
}

// WithReasonCodes sets reason codes
func WithReasonCodes(codes ...string) ComponentPostureOption {
	return func(p *ComponentPosture) {
		p.ReasonCodes = codes
	}
}

// AggregatePosture combines component postures into system posture
func AggregatePosture(
	components []ComponentPosture,
	definitions []ComponentDefinition,
	thresholds *DegradationThresholds,
) SystemPosture {
	if thresholds == nil {
		thresholds = &DefaultDegradationThresholds
	}
	
	now := time.Now().UTC()
	
	// Build criticality map
	criticalityMap := make(map[string]ComponentCriticality)
	for _, def := range definitions {
		criticalityMap[def.ID] = def.Criticality
	}
	
	// Count states
	unhealthyCount := 0
	degradedCount := 0
	unknownCount := 0
	criticalFailed := false
	criticalDegraded := false
	
	for _, comp := range components {
		switch comp.Level {
		case PostureLevelUnhealthy:
			unhealthyCount++
		case PostureLevelDegraded:
			degradedCount++
		case PostureLevelUnknown:
			unknownCount++
		}
		
		if criticalityMap[comp.ID] == ComponentCriticalityCritical {
			if comp.Level == PostureLevelUnhealthy {
				criticalFailed = true
			}
			if comp.Level == PostureLevelDegraded {
				criticalDegraded = true
			}
		}
	}
	
	// Determine overall posture
	overall :=