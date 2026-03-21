package simulation

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/intelligence"
	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/status"
	"github.com/mel-project/mel/internal/transport"
)

// ImpactScope defines the scope/level at which impact is assessed
type ImpactScope string

const (
	ScopeTransport ImpactScope = "transport" // Single transport impact
	ScopeSegment   ImpactScope = "segment"   // Segment-level impact
	ScopeMesh      ImpactScope = "mesh"      // Full mesh impact
)

// ImpactType categorizes the type of predicted impact
type ImpactType string

const (
	ImpactTypeConnectivity  ImpactType = "connectivity"  // Network/connectivity impact
	ImpactTypeDataFlow      ImpactType = "data_flow"     // Data ingestion/flow impact
	ImpactTypeObservability ImpactType = "observability" // Monitoring/observability impact
	ImpactTypeControlPath   ImpactType = "control_path"  // Control signal path impact
	ImpactTypeCascading     ImpactType = "cascading"     // Secondary cascading effects
)

// ImpactStatus distinguishes between observed and predicted impacts
type ImpactStatus string

const (
	ImpactStatusObserved     ImpactStatus = "observed"     // Impact is already occurring
	ImpactStatusPredicted    ImpactStatus = "predicted"    // Impact is predicted if action taken
	ImpactStatusHypothetical ImpactStatus = "hypothetical" // Theoretical impact under certain conditions
)

// PredictedImpact represents a single predicted impact entry
type PredictedImpact struct {
	// Scope indicates the level at which this impact occurs
	Scope ImpactScope `json:"scope"`

	// Type categorizes the impact
	Type ImpactType `json:"type"`

	// Status distinguishes observed vs predicted
	Status ImpactStatus `json:"status"`

	// Component identifies the affected component (transport name, segment ID, etc.)
	Component string `json:"component"`

	// Description explains the expected impact
	Description string `json:"description"`

	// Severity of the impact
	Severity RiskLevel `json:"severity"`

	// Likelihood of this impact occurring (0.0-1.0)
	Likelihood float64 `json:"likelihood"`

	// AffectedNodes lists nodes expected to be affected
	AffectedNodes []string `json:"affected_nodes,omitempty"`

	// AffectedTransports lists transports expected to be affected
	AffectedTransports []string `json:"affected_transports,omitempty"`

	// DurationEstimate predicts how long the impact will last
	DurationEstimate time.Duration `json:"duration_estimate,omitempty"`

	// Mitigation suggests how to address this impact
	Mitigation string `json:"mitigation,omitempty"`
}

// CascadingEffect represents a secondary/tertiary effect that may follow the primary impact
type CascadingEffect struct {
	// Trigger describes what condition triggers this cascading effect
	Trigger string `json:"trigger"`

	// Effect describes the expected cascading outcome
	Effect string `json:"effect"`

	// ChainDepth indicates how many steps removed from primary action (1=direct, 2=secondary, etc.)
	ChainDepth int `json:"chain_depth"`

	// Probability of this cascading effect occurring
	Probability float64 `json:"probability"`

	// Severity if the cascade occurs
	Severity RiskLevel `json:"severity"`

	// Preventable indicates if this cascade can be prevented/mitigated
	Preventable bool `json:"preventable"`

	// PreventionAction describes how to prevent this cascade
	PreventionAction string `json:"prevention_action,omitempty"`
}

// ActionImpactProfile defines the inherent impact characteristics of each action type
type ActionImpactProfile struct {
	// BaseRadius is the inherent blast radius for this action type (0.0-1.0)
	BaseRadius float64

	// Scope is the default scope level for this action
	Scope ImpactScope

	// DisruptsConnectivity indicates if this action disrupts network connectivity
	DisruptsConnectivity bool

	// DisruptsDataFlow indicates if this action disrupts data ingestion/flow
	DisruptsDataFlow bool

	// RecoveryTime is the typical time to recover from this action
	RecoveryTime time.Duration

	// CanCascade indicates if this action can trigger cascading effects
	CanCascade bool

	// Description provides human-readable explanation
	Description string
}

// BlastRadiusPredictor provides enhanced blast radius prediction that extends
// intelligence.EstimateBlastRadius with detailed pre-action impact analysis.
type BlastRadiusPredictor struct {
	// mesh provides the current mesh topology and health state
	mesh status.MeshDrilldown

	// actionProfiles contains impact profiles for each action type
	actionProfiles map[string]ActionImpactProfile
}

// NewBlastRadiusPredictor creates a new predictor with the current mesh state
func NewBlastRadiusPredictor(mesh status.MeshDrilldown) *BlastRadiusPredictor {
	return &BlastRadiusPredictor{
		mesh:           mesh,
		actionProfiles: buildActionImpactProfiles(),
	}
}

// PredictBlastRadius performs comprehensive blast radius prediction for a proposed action.
// It extends intelligence.EstimateBlastRadius with detailed pre-action impact analysis
// including transport-level, segment-level, and mesh-level impacts.
//
// The prediction:
//   - Estimates nodes affected if the action is taken
//   - Identifies segments/groups impacted
//   - Models potential cascading effects
//   - Provides confidence level based on data quality
//   - Clearly distinguishes observed vs predicted impacts
//   - Considers action type (restart vs backoff vs suppress have different radii)
func (p *BlastRadiusPredictor) PredictBlastRadius(
	action control.ControlAction,
	priorities []models.PriorityItem,
	nodes []models.Node,
) BlastRadiusPrediction {
	startTime := time.Now()

	// Get the base blast radius from the existing intelligence function
	baseScore, baseMessage := intelligence.EstimateBlastRadius(priorities, nodes)

	// Get action profile for action-specific adjustments
	profile := p.getActionProfile(action.ActionType)

	// Build the detailed prediction
	prediction := p.buildPrediction(action, priorities, nodes, baseScore, baseMessage, profile)

	// Add detailed impact analysis
	prediction.ServiceImpact = p.assessServiceImpacts(action, profile)

	// Store metadata about prediction time
	_ = startTime // Used for potential future latency tracking

	return prediction
}

// PredictBlastRadiusDetailed returns both the prediction and detailed impact breakdown
func (p *BlastRadiusPredictor) PredictBlastRadiusDetailed(
	action control.ControlAction,
	priorities []models.PriorityItem,
	nodes []models.Node,
) (BlastRadiusPrediction, []PredictedImpact, []CascadingEffect) {
	prediction := p.PredictBlastRadius(action, priorities, nodes)

	// Build detailed impact list
	impacts := p.predictDetailedImpacts(action, prediction)

	// Build cascading effects
	cascades := p.predictCascadingEffects(action, prediction)

	return prediction, impacts, cascades
}

// buildPrediction constructs the blast radius prediction with all fields populated
func (p *BlastRadiusPredictor) buildPrediction(
	action control.ControlAction,
	priorities []models.PriorityItem,
	nodes []models.Node,
	baseScore float64,
	baseMessage string,
	profile ActionImpactProfile,
) BlastRadiusPrediction {
	// Determine affected components based on mesh topology
	affectedTransports := p.identifyAffectedTransports(action)
	affectedNodes := p.identifyAffectedNodes(action, affectedTransports)
	affectedSegments := p.identifyAffectedSegments(action, affectedTransports)

	// Calculate adjusted score based on action profile and mesh state
	adjustedScore := p.calculateAdjustedScore(baseScore, action, profile, affectedTransports, affectedNodes)

	// Determine classification based on scope and score
	classification := p.classifyBlastRadius(adjustedScore, profile.Scope, len(affectedTransports), len(affectedSegments))

	// Calculate confidence based on data quality
	confidence := p.calculateConfidence(action, affectedTransports)

	// Build comprehensive description
	description := p.buildDescription(action, baseMessage, affectedTransports, affectedNodes, affectedSegments, profile)

	return BlastRadiusPrediction{
		Score:              adjustedScore,
		Classification:     classification,
		Description:        description,
		AffectedTransports: affectedTransports,
		AffectedNodes:      affectedNodes,
		AffectedSegments:   affectedSegments,
		Confidence:         confidence,
	}
}

// identifyAffectedTransports determines which transports will be affected by the action
func (p *BlastRadiusPredictor) identifyAffectedTransports(action control.ControlAction) []string {
	affected := make(map[string]struct{})

	// Primary target is always affected
	if action.TargetTransport != "" {
		affected[action.TargetTransport] = struct{}{}
	}

	// Check for correlated failures that may indicate broader impact
	for _, failure := range p.mesh.CorrelatedFailures {
		if containsString(failure.Transports, action.TargetTransport) {
			// If target is part of a correlated failure, all transports in that
			// correlation group may be affected
			for _, t := range failure.Transports {
				affected[t] = struct{}{}
			}
		}
	}

	// Check for shared segments
	for _, segment := range p.mesh.DegradedSegments {
		if containsString(segment.Transports, action.TargetTransport) {
			// If target is in a degraded segment, related transports may be affected
			for _, t := range segment.Transports {
				affected[t] = struct{}{}
			}
		}
	}

	// For mesh-level actions, all transports are potentially affected
	if action.TargetSegment != "" || isMeshWideAction(action.ActionType) {
		for _, alert := range p.mesh.ActiveAlerts {
			affected[alert.TransportName] = struct{}{}
		}
		// Also add transports from health explanation
		for _, t := range p.mesh.MeshHealthExplanation.AffectedTransports {
			affected[t] = struct{}{}
		}
	}

	return mapKeysSorted(affected)
}

// identifyAffectedNodes determines which nodes will be affected
func (p *BlastRadiusPredictor) identifyAffectedNodes(action control.ControlAction, affectedTransports []string) []string {
	affected := make(map[string]struct{})

	// Add nodes from affected transports
	transportSet := make(map[string]struct{})
	for _, t := range affectedTransports {
		transportSet[t] = struct{}{}
	}

	// From correlated failures
	for _, failure := range p.mesh.CorrelatedFailures {
		for _, t := range failure.Transports {
			if _, ok := transportSet[t]; ok {
				for _, nodeID := range failure.NodeIDs {
					affected[nodeID] = struct{}{}
				}
			}
		}
	}

	// From degraded segments
	for _, segment := range p.mesh.DegradedSegments {
		for _, t := range segment.Transports {
			if _, ok := transportSet[t]; ok {
				for _, nodeID := range segment.Nodes {
					affected[nodeID] = struct{}{}
				}
			}
		}
	}

	// From mesh health explanation
	for _, nodeID := range p.mesh.MeshHealthExplanation.AffectedNodes {
		affected[nodeID] = struct{}{}
	}

	// If action targets a specific node
	if action.TargetNode != "" {
		affected[action.TargetNode] = struct{}{}
	}

	return mapKeysSorted(affected)
}

// identifyAffectedSegments determines which segments will be impacted
func (p *BlastRadiusPredictor) identifyAffectedSegments(action control.ControlAction, affectedTransports []string) []string {
	affected := make(map[string]struct{})

	// Target segment is always affected
	if action.TargetSegment != "" {
		affected[action.TargetSegment] = struct{}{}
	}

	transportSet := make(map[string]struct{})
	for _, t := range affectedTransports {
		transportSet[t] = struct{}{}
	}

	// Check all degraded segments
	for _, segment := range p.mesh.DegradedSegments {
		for _, t := range segment.Transports {
			if _, ok := transportSet[t]; ok {
				affected[segment.SegmentID] = struct{}{}
				break
			}
		}
	}

	// Check for segments related to active alerts on affected transports
	for _, alert := range p.mesh.ActiveAlerts {
		if _, ok := transportSet[alert.TransportName]; ok {
			// Find or create segment reference for this alert
			segmentID := fmt.Sprintf("segment:%s:%s", alert.Reason, alert.TransportName)
			affected[segmentID] = struct{}{}
		}
	}

	return mapKeysSorted(affected)
}

// calculateAdjustedScore adjusts the base score based on action type and context
func (p *BlastRadiusPredictor) calculateAdjustedScore(
	baseScore float64,
	action control.ControlAction,
	profile ActionImpactProfile,
	affectedTransports []string,
	affectedNodes []string,
) float64 {
	// Start with action's inherent radius
	score := profile.BaseRadius

	// Adjust based on base intelligence score (blend the two)
	score = (score + baseScore) / 2

	// Adjust based on number of affected transports
	transportMultiplier := 1.0 + (float64(len(affectedTransports)) * 0.1)
	if transportMultiplier > 1.5 {
		transportMultiplier = 1.5
	}
	score *= transportMultiplier

	// Adjust based on mesh health state
	switch p.mesh.MeshHealth.State {
	case "failed":
		score *= 1.3 // Higher impact in failed state
	case "unstable":
		score *= 1.2 // Elevated impact in unstable state
	case "degraded":
		score *= 1.1 // Slightly elevated in degraded state
	}

	// Adjust for critical segments
	if len(p.mesh.MeshHealth.CriticalSegments) > 0 {
		score *= 1.15
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// classifyBlastRadius determines the classification label
func (p *BlastRadiusPredictor) classifyBlastRadius(score float64, scope ImpactScope, numTransports, numSegments int) string {
	// Primary classification by score
	if score >= 0.75 {
		return "systemic"
	}
	if score >= 0.5 {
		return "mesh_wide"
	}
	if score >= 0.25 {
		return "segmented"
	}

	// Secondary classification by scope
	if scope == ScopeMesh || numSegments > 2 {
		return "segmented"
	}
	if scope == ScopeSegment || numTransports > 1 {
		return "multi_transport"
	}

	return "local_transport"
}

// calculateConfidence determines prediction confidence based on data quality
func (p *BlastRadiusPredictor) calculateConfidence(action control.ControlAction, affectedTransports []string) float64 {
	confidence := 0.85 // Base confidence

	// Reduce confidence if we have limited topology data
	if len(p.mesh.MeshHealthExplanation.AffectedTransports) == 0 {
		confidence -= 0.15
	}

	// Reduce confidence if mesh state is unknown
	if p.mesh.MeshHealth.State == "" {
		confidence -= 0.20
	}

	// Reduce confidence for actions with unknown profiles
	if _, ok := p.actionProfiles[action.ActionType]; !ok {
		confidence -= 0.15
	}

	// Increase confidence for actions with strong trigger evidence
	if len(action.TriggerEvidence) >= 3 {
		confidence += 0.05
	}

	// Cap confidence
	if confidence > 0.95 {
		confidence = 0.95
	}
	if confidence < 0.40 {
		confidence = 0.40
	}

	return confidence
}

// buildDescription creates a comprehensive human-readable description
func (p *BlastRadiusPredictor) buildDescription(
	action control.ControlAction,
	baseMessage string,
	affectedTransports, affectedNodes, affectedSegments []string,
	profile ActionImpactProfile,
) string {
	parts := []string{}

	// Base message
	if baseMessage != "" {
		parts = append(parts, baseMessage)
	}

	// Action-specific context
	parts = append(parts, profile.Description)

	// Transport impact summary
	if len(affectedTransports) > 0 {
		if len(affectedTransports) == 1 {
			parts = append(parts, fmt.Sprintf("Primary impact on transport: %s", affectedTransports[0]))
		} else {
			parts = append(parts, fmt.Sprintf("Impact spans %d transports: %s",
				len(affectedTransports), strings.Join(affectedTransports, ", ")))
		}
	}

	// Node impact summary
	if len(affectedNodes) > 0 {
		if len(affectedNodes) <= 5 {
			parts = append(parts, fmt.Sprintf("Affected nodes: %s", strings.Join(affectedNodes, ", ")))
		} else {
			parts = append(parts, fmt.Sprintf("Affected nodes: %d total (%s and %d others)",
				len(affectedNodes), strings.Join(affectedNodes[:3], ", "), len(affectedNodes)-3))
		}
	}

	// Segment impact summary
	if len(affectedSegments) > 0 {
		parts = append(parts, fmt.Sprintf("Spans %d mesh segment(s)", len(affectedSegments)))
	}

	// Recovery estimate
	if profile.RecoveryTime > 0 {
		parts = append(parts, fmt.Sprintf("Estimated recovery time: %v", profile.RecoveryTime))
	}

	return strings.Join(parts, "; ")
}

// predictDetailedImpacts generates a detailed list of predicted impacts
func (p *BlastRadiusPredictor) predictDetailedImpacts(action control.ControlAction, prediction BlastRadiusPrediction) []PredictedImpact {
	impacts := make([]PredictedImpact, 0)
	profile := p.getActionProfile(action.ActionType)

	// Transport-level impacts
	for _, transportName := range prediction.AffectedTransports {
		status := ImpactStatusPredicted
		if p.isTransportCurrentlyAffected(transportName) {
			status = ImpactStatusObserved
		}

		// Connectivity impact
		if profile.DisruptsConnectivity {
			impacts = append(impacts, PredictedImpact{
				Scope:            ScopeTransport,
				Type:             ImpactTypeConnectivity,
				Status:           status,
				Component:        transportName,
				Description:      fmt.Sprintf("Transport %s will experience connectivity disruption", transportName),
				Severity:         p.impactSeverityForScore(prediction.Score),
				Likelihood:       prediction.Confidence,
				DurationEstimate: profile.RecoveryTime,
				Mitigation:       "Monitor transport health; action is reversible via reconnect loop",
			})
		}

		// Data flow impact
		if profile.DisruptsDataFlow {
			impacts = append(impacts, PredictedImpact{
				Scope:            ScopeTransport,
				Type:             ImpactTypeDataFlow,
				Status:           status,
				Component:        transportName,
				Description:      fmt.Sprintf("Data ingestion from %s will be interrupted", transportName),
				Severity:         p.impactSeverityForScore(prediction.Score),
				Likelihood:       prediction.Confidence,
				DurationEstimate: profile.RecoveryTime,
				Mitigation:       "Ensure alternate transports are healthy; consider temporary buffering",
			})
		}
	}

	// Segment-level impacts
	for _, segmentID := range prediction.AffectedSegments {
		impacts = append(impacts, PredictedImpact{
			Scope:       ScopeSegment,
			Type:        ImpactTypeControlPath,
			Status:      ImpactStatusPredicted,
			Component:   segmentID,
			Description: fmt.Sprintf("Control path through segment %s may be affected", segmentID),
			Severity:    p.impactSeverityForScore(prediction.Score),
			Likelihood:  prediction.Confidence * 0.8,
			Mitigation:  "Verify segment redundancy before proceeding",
		})
	}

	// Mesh-level observability impact
	if prediction.Score > 0.5 {
		impacts = append(impacts, PredictedImpact{
			Scope:       ScopeMesh,
			Type:        ImpactTypeObservability,
			Status:      ImpactStatusPredicted,
			Component:   "mesh_observability",
			Description: "Mesh-wide observability may be reduced during action execution",
			Severity:    RiskLevelMedium,
			Likelihood:  prediction.Confidence * 0.6,
			Mitigation:  "Ensure monitoring systems have alternative data paths",
		})
	}

	return impacts
}

// predictCascadingEffects models potential secondary and tertiary effects
func (p *BlastRadiusPredictor) predictCascadingEffects(action control.ControlAction, prediction BlastRadiusPrediction) []CascadingEffect {
	effects := make([]CascadingEffect, 0)
	profile := p.getActionProfile(action.ActionType)

	if !profile.CanCascade {
		return effects
	}

	// Check for correlated failure patterns that could cascade
	for _, failure := range p.mesh.CorrelatedFailures {
		if len(failure.Transports) >= 2 {
			effects = append(effects, CascadingEffect{
				Trigger:          fmt.Sprintf("Action affects transport involved in correlation: %s", failure.Reason),
				Effect:           fmt.Sprintf("Other transports in correlation group may experience increased load: %v", failure.Transports),
				ChainDepth:       2,
				Probability:      prediction.Confidence * 0.4,
				Severity:         RiskLevelMedium,
				Preventable:      true,
				PreventionAction: "Stagger actions across correlated transports; monitor load distribution",
			})
		}
	}

	// Check for degraded segment cascades
	if len(p.mesh.DegradedSegments) > 0 {
		effects = append(effects, CascadingEffect{
			Trigger:          "Action targets transport in degraded segment",
			Effect:           "Segment degradation may worsen, affecting remaining healthy transports in segment",
			ChainDepth:       2,
			Probability:      prediction.Confidence * 0.35,
			Severity:         RiskLevelHigh,
			Preventable:      true,
			PreventionAction: "Verify segment has redundancy; consider segment-level action instead",
		})
	}

	// Check for mesh health cascades
	if p.mesh.MeshHealth.State == "degraded" || p.mesh.MeshHealth.State == "unstable" {
		effects = append(effects, CascadingEffect{
			Trigger:          fmt.Sprintf("Action executed while mesh is %s", p.mesh.MeshHealth.State),
			Effect:           "Mesh health may further degrade; recovery time may be extended",
			ChainDepth:       1,
			Probability:      prediction.Confidence * 0.5,
			Severity:         RiskLevelHigh,
			Preventable:      false,
			PreventionAction: "Consider waiting for mesh stabilization or reduce action scope",
		})
	}

	// Evidence loss cascade (for data-disrupting actions)
	if profile.DisruptsDataFlow && p.mesh.MeshHealthExplanation.EvidenceLossSummary.ObservationDrops > 0 {
		effects = append(effects, CascadingEffect{
			Trigger:          "Data flow disruption during existing evidence loss conditions",
			Effect:           "Additional observability data may be lost during action execution",
			ChainDepth:       1,
			Probability:      prediction.Confidence * 0.6,
			Severity:         RiskLevelMedium,
			Preventable:      true,
			PreventionAction: "Ensure evidence buffer capacity; schedule during low-traffic period",
		})
	}

	return effects
}

// assessServiceImpacts evaluates impact on specific services
func (p *BlastRadiusPredictor) assessServiceImpacts(action control.ControlAction, profile ActionImpactProfile) []ServiceImpact {
	impacts := make([]ServiceImpact, 0)

	// Ingest service impact
	if profile.DisruptsDataFlow {
		impacts = append(impacts, ServiceImpact{
			ServiceName:      "ingest",
			ImpactLevel:      RiskLevelMedium,
			Description:      fmt.Sprintf("Data ingestion will be interrupted for target transport during action"),
			DurationEstimate: profile.RecoveryTime,
		})
	}

	// Monitoring service impact
	if len(p.mesh.ActiveAlerts) > 0 {
		impacts = append(impacts, ServiceImpact{
			ServiceName:      "monitoring",
			ImpactLevel:      RiskLevelLow,
			Description:      "Alert state may temporarily fluctuate during transport restart",
			DurationEstimate: profile.RecoveryTime / 2,
		})
	}

	// Control plane impact
	if action.TargetSegment != "" || isMeshWideAction(action.ActionType) {
		impacts = append(impacts, ServiceImpact{
			ServiceName:      "control_plane",
			ImpactLevel:      RiskLevelMedium,
			Description:      "Mesh-level control actions may experience delayed propagation",
			DurationEstimate: profile.RecoveryTime,
		})
	}

	return impacts
}

// getActionProfile returns the impact profile for an action type
func (p *BlastRadiusPredictor) getActionProfile(actionType string) ActionImpactProfile {
	if profile, ok := p.actionProfiles[actionType]; ok {
		return profile
	}
	// Default profile for unknown actions
	return ActionImpactProfile{
		BaseRadius:           0.5,
		Scope:                ScopeTransport,
		DisruptsConnectivity: true,
		DisruptsDataFlow:     true,
		RecoveryTime:         30 * time.Second,
		CanCascade:           true,
		Description:          "Unknown action type - assuming moderate impact",
	}
}

// isTransportCurrentlyAffected checks if a transport is already in a degraded state
func (p *BlastRadiusPredictor) isTransportCurrentlyAffected(transportName string) bool {
	for _, t := range p.mesh.MeshHealthExplanation.AffectedTransports {
		if t == transportName {
			return true
		}
	}
	return false
}

// impactSeverityForScore converts a blast radius score to risk level
func (p *BlastRadiusPredictor) impactSeverityForScore(score float64) RiskLevel {
	switch {
	case score >= 0.75:
		return RiskLevelCritical
	case score >= 0.5:
		return RiskLevelHigh
	case score >= 0.25:
		return RiskLevelMedium
	default:
		return RiskLevelLow
	}
}

// buildActionImpactProfiles creates the impact profiles for each action type
func buildActionImpactProfiles() map[string]ActionImpactProfile {
	return map[string]ActionImpactProfile{
		control.ActionRestartTransport: {
			BaseRadius:           0.3,
			Scope:                ScopeTransport,
			DisruptsConnectivity: true,
			DisruptsDataFlow:     true,
			RecoveryTime:         15 * time.Second,
			CanCascade:           true,
			Description:          "Transport restart interrupts single transport; bounded reconnect loop provides automatic recovery",
		},
		control.ActionResubscribeTransport: {
			BaseRadius:           0.25,
			Scope:                ScopeTransport,
			DisruptsConnectivity: true,
			DisruptsDataFlow:     false,
			RecoveryTime:         10 * time.Second,
			CanCascade:           false,
			Description:          "Resubscription affects only subscription state; data flow typically maintained",
		},
		control.ActionBackoffIncrease: {
			BaseRadius:           0.15,
			Scope:                ScopeTransport,
			DisruptsConnectivity: false,
			DisruptsDataFlow:     false,
			RecoveryTime:         5 * time.Second,
			CanCascade:           false,
			Description:          "Backoff increase is gradual; no immediate disruption, affects reconnection timing",
		},
		control.ActionBackoffReset: {
			BaseRadius:           0.1,
			Scope:                ScopeTransport,
			DisruptsConnectivity: false,
			DisruptsDataFlow:     false,
			RecoveryTime:         0,
			CanCascade:           false,
			Description:          "Backoff reset is immediate with no disruption; may increase reconnection attempts",
		},
		control.ActionTemporarilyDeprioritize: {
			BaseRadius:           0.4,
			Scope:                ScopeMesh,
			DisruptsConnectivity: false,
			DisruptsDataFlow:     false,
			RecoveryTime:         0,
			CanCascade:           true,
			Description:          "Deprioritization affects routing decisions across mesh; impact depends on alternate path health",
		},
		control.ActionTemporarilySuppressNoisySource: {
			BaseRadius:           0.2,
			Scope:                ScopeSegment,
			DisruptsConnectivity: false,
			DisruptsDataFlow:     true,
			RecoveryTime:         0,
			CanCascade:           false,
			Description:          "Source suppression reduces data flow from specific source; reduces noise but may miss valid data",
		},
		control.ActionClearSuppression: {
			BaseRadius:           0.1,
			Scope:                ScopeSegment,
			DisruptsConnectivity: false,
			DisruptsDataFlow:     false,
			RecoveryTime:         0,
			CanCascade:           false,
			Description:          "Clearing suppression restores normal data flow; minimal risk",
		},
		control.ActionTriggerHealthRecheck: {
			BaseRadius:           0.05,
			Scope:                ScopeTransport,
			DisruptsConnectivity: false,
			DisruptsDataFlow:     false,
			RecoveryTime:         0,
			CanCascade:           false,
			Description:          "Health recheck is read-only with no state changes; no disruption expected",
		},
	}
}

// isMeshWideAction determines if an action type affects the entire mesh
func isMeshWideAction(actionType string) bool {
	switch actionType {
	case control.ActionTemporarilyDeprioritize:
		return true
	default:
		return false
	}
}

// Helper functions

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func mapKeysSorted(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// PredictBlastRadiusForSimulation provides a convenience method for the simulation engine
// that uses the mesh topology directly without requiring separate priority/node lists
func (p *BlastRadiusPredictor) PredictBlastRadiusForSimulation(
	action control.ControlAction,
	runtimeTransports []transport.Health,
) BlastRadiusPrediction {
	// Build priorities from mesh state
	priorities := p.buildPrioritiesFromMesh(action)

	// Build node list from mesh state
	nodes := p.buildNodesFromMesh()

	return p.PredictBlastRadius(action, priorities, nodes)
}

// buildPrioritiesFromMesh creates priority items from mesh drilldown state
func (p *BlastRadiusPredictor) buildPrioritiesFromMesh(action control.ControlAction) []models.PriorityItem {
	priorities := make([]models.PriorityItem, 0)

	// Add active alerts as priorities
	for _, alert := range p.mesh.ActiveAlerts {
		if alert.TransportName == action.TargetTransport || action.TargetTransport == "" {
			priorities = append(priorities, models.PriorityItem{
				ID:       alert.ID,
				Category: "transport",
				Severity: alert.Severity,
				Title:    alert.Reason,
				Metadata: map[string]any{
					"resource_id":        alert.TransportName,
					"affected_transport": alert.TransportName,
					"alert_summary":      alert.Summary,
				},
			})
		}
	}

	// Add degraded segments as priorities
	for _, segment := range p.mesh.DegradedSegments {
		if containsString(segment.Transports, action.TargetTransport) || action.TargetTransport == "" {
			category := "segment"
			if segment.Severity == "critical" {
				category = "system"
			}
			priorities = append(priorities, models.PriorityItem{
				ID:       segment.SegmentID,
				Category: category,
				Severity: segment.Severity,
				Title:    segment.Reason,
				Metadata: map[string]any{
					"segment_id":          segment.SegmentID,
					"affected_transports": segment.Transports,
					"affected_nodes":      segment.Nodes,
				},
			})
		}
	}

	return priorities
}

// buildNodesFromMesh creates node list from mesh drilldown state
func (p *BlastRadiusPredictor) buildNodesFromMesh() []models.Node {
	nodes := make([]models.Node, 0)
	seen := make(map[string]bool)

	// Add nodes from mesh health explanation
	for _, nodeID := range p.mesh.MeshHealthExplanation.AffectedNodes {
		if !seen[nodeID] {
			nodes = append(nodes, models.Node{NodeID: nodeID})
			seen[nodeID] = true
		}
	}

	// Add nodes from correlated failures
	for _, failure := range p.mesh.CorrelatedFailures {
		for _, nodeID := range failure.NodeIDs {
			if !seen[nodeID] {
				nodes = append(nodes, models.Node{NodeID: nodeID})
				seen[nodeID] = true
			}
		}
	}

	// Add nodes from degraded segments
	for _, segment := range p.mesh.DegradedSegments {
		for _, nodeID := range segment.Nodes {
			if !seen[nodeID] {
				nodes = append(nodes, models.Node{NodeID: nodeID})
				seen[nodeID] = true
			}
		}
	}

	return nodes
}

// BlastRadiusSummary provides a simplified summary of predicted impact
type BlastRadiusSummary struct {
	// TotalImpacts is the count of predicted impacts
	TotalImpacts int `json:"total_impacts"`

	// ObservedImpacts already occurring
	ObservedImpacts int `json:"observed_impacts"`

	// PredictedImpacts that may occur if action is taken
	PredictedImpacts int `json:"predicted_impacts"`

	// CascadingRisk indicates if cascading effects are possible
	CascadingRisk bool `json:"cascading_risk"`

	// PrimaryImpact is the main expected impact
	PrimaryImpact string `json:"primary_impact"`

	// RecoveryEstimate is the expected time to full recovery
	RecoveryEstimate time.Duration `json:"recovery_estimate"`
}

// Summarize returns a simplified summary of the blast radius prediction
func (p *BlastRadiusPredictor) Summarize(prediction BlastRadiusPrediction, impacts []PredictedImpact) BlastRadiusSummary {
	observed := 0
	predicted := 0
	cascadingRisk := false

	for _, impact := range impacts {
		switch impact.Status {
		case ImpactStatusObserved:
			observed++
		case ImpactStatusPredicted:
			predicted++
		}
	}

	// Check for cascading risk based on score and mesh state
	if prediction.Score > 0.5 || p.mesh.MeshHealth.State == "unstable" || p.mesh.MeshHealth.State == "degraded" {
		cascadingRisk = true
	}

	profile := p.getActionProfile(prediction.AffectedTransports[0]) // Use first transport as proxy
	if len(prediction.AffectedTransports) == 0 {
		profile = p.actionProfiles[control.ActionTriggerHealthRecheck]
	}

	return BlastRadiusSummary{
		TotalImpacts:     len(impacts),
		ObservedImpacts:  observed,
		PredictedImpacts: predicted,
		CascadingRisk:    cascadingRisk,
		PrimaryImpact:    prediction.Description,
		RecoveryEstimate: profile.RecoveryTime,
	}
}
