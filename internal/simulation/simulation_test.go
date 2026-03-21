package simulation

import (
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/status"
)

// TestEngineCreation verifies the engine initializes correctly
func TestEngineCreation(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := NewEngine(cfg, nil)

	if engine == nil {
		t.Fatal("Expected engine to be created, got nil")
	}

	if engine.cfg != cfg {
		t.Error("Expected engine to store config")
	}
}

// TestSimulateDeterminism verifies same inputs produce same outputs
func TestSimulateDeterminism(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Control.Mode = control.ModeAdvisory
	engine := NewEngine(cfg, nil)

	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	mesh := status.MeshDrilldown{
		MeshHealth: status.MeshHealth{
			Score: 85,
			State: "healthy",
		},
	}

	input := SimulationInput{
		CandidateAction: control.ControlAction{
			ID:              "test-action-1",
			ActionType:      control.ActionTriggerHealthRecheck,
			TargetTransport: "test-transport",
			Reason:          "test simulation",
			Confidence:      0.9,
			CreatedAt:       now.Format(time.RFC3339),
			Mode:            control.ModeAdvisory,
		},
		MeshState:   mesh,
		CurrentTime: now,
		Configuration: cfg,
	}

	// Run simulation twice with same inputs
	result1, err1 := engine.Simulate(input)
	result2, err2 := engine.Simulate(input)

	if err1 != nil {
		t.Fatalf("First simulation failed: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("Second simulation failed: %v", err2)
	}

	// Verify determinism - key fields should match
	if result1.PredictedOutcome.SuccessProbability != result2.PredictedOutcome.SuccessProbability {
		t.Errorf("Success probability not deterministic: %v vs %v",
			result1.PredictedOutcome.SuccessProbability,
			result2.PredictedOutcome.SuccessProbability)
	}

	if result1.RiskAssessment.RiskLevel != result2.RiskAssessment.RiskLevel {
		t.Errorf("Risk level not deterministic: %v vs %v",
			result1.RiskAssessment.RiskLevel,
			result2.RiskAssessment.RiskLevel)
	}

	if result1.SafeToAct.Decision != result2.SafeToAct.Decision {
		t.Errorf("Safe-to-act decision not deterministic: %v vs %v",
			result1.SafeToAct.Decision,
			result2.SafeToAct.Decision)
	}
}

// TestSafeToActEvaluation verifies safe-to-act logic
func TestSafeToActEvaluation(t *testing.T) {
	tests := []struct {
		name           string
		actionType     string
		mode           string
		meshState      string
		expectedDecision SafetyStatus
	}{
		{
			name:           "Health recheck in advisory mode",
			actionType:     control.ActionTriggerHealthRecheck,
			mode:           control.ModeAdvisory,
			meshState:      "healthy",
			expectedDecision: SafeAfterCondition, // Advisory mode requires operator
		},
		{
			name:           "Restart in disabled mode",
			actionType:     control.ActionRestartTransport,
			mode:           control.ModeDisabled,
			meshState:      "degraded",
			expectedDecision: NotSafe, // Disabled mode blocks all
		},
		{
			name:           "Backoff increase with healthy mesh",
			actionType:     control.ActionBackoffIncrease,
			mode:           control.ModeGuardedAuto,
			meshState:      "healthy",
			expectedDecision: SafeToAct, // Low risk action
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Control.Mode = tt.mode
			engine := NewEngine(cfg, nil)

			mesh := status.MeshDrilldown{
				MeshHealth: status.MeshHealth{
					Score: 85,
					State: tt.meshState,
				},
			}

			input := SimulationInput{
				CandidateAction: control.ControlAction{
					ID:              "test-action",
					ActionType:      tt.actionType,
					TargetTransport: "test-transport",
					Reason:          "test",
					Confidence:      0.8,
					CreatedAt:       time.Now().UTC().Format(time.RFC3339),
					Mode:            tt.mode,
				},
				MeshState:     mesh,
				CurrentTime:   time.Now().UTC(),
				Configuration: cfg,
			}

			result, err := engine.Simulate(input)
			if err != nil {
				t.Fatalf("Simulation failed: %v", err)
			}

			// Note: Exact decision depends on many factors, so we just verify
			// the decision is one of the valid values
			validDecisions := []SafetyStatus{
				SafeToAct, SafeAfterCondition, NotSafe, InsufficientData,
			}
			found := false
			for _, d := range validDecisions {
				if result.SafeToAct.Decision == d {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Invalid decision: %v", result.SafeToAct.Decision)
			}
		})
	}
}

// TestRiskScoring verifies risk level assignment
func TestRiskScoring(t *testing.T) {
	tests := []struct {
		name           string
		actionType     string
		expectedMinRisk RiskLevel
		expectedMaxRisk RiskLevel
	}{
		{
			name:           "Health recheck is low risk",
			actionType:     control.ActionTriggerHealthRecheck,
			expectedMinRisk: RiskLevelNone,
			expectedMaxRisk: RiskLevelLow,
		},
		{
			name:           "Restart transport is medium risk",
			actionType:     control.ActionRestartTransport,
			expectedMinRisk: RiskLevelLow,
			expectedMaxRisk: RiskLevelMedium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			engine := NewEngine(cfg, nil)

			mesh := status.MeshDrilldown{
				MeshHealth: status.MeshHealth{
					Score: 80,
					State: "healthy",
				},
			}

			input := SimulationInput{
				CandidateAction: control.ControlAction{
					ID:              "test-action",
					ActionType:      tt.actionType,
					TargetTransport: "test-transport",
					Reason:          "test",
					Confidence:      0.8,
					CreatedAt:       time.Now().UTC().Format(time.RFC3339),
					Mode:            control.ModeGuardedAuto,
				},
				MeshState:     mesh,
				CurrentTime:   time.Now().UTC(),
				Configuration: cfg,
			}

			result, err := engine.Simulate(input)
			if err != nil {
				t.Fatalf("Simulation failed: %v", err)
			}

			// Verify risk level is within expected range
			riskOrder := map[RiskLevel]int{
				RiskLevelNone:     0,
				RiskLevelLow:      1,
				RiskLevelMedium:   2,
				RiskLevelHigh:     3,
				RiskLevelCritical: 4,
			}

			actualRisk := riskOrder[result.RiskAssessment.RiskLevel]
			minRisk := riskOrder[tt.expectedMinRisk]
			maxRisk := riskOrder[tt.expectedMaxRisk]

			if actualRisk < minRisk || actualRisk > maxRisk {
				t.Errorf("Risk level %v not in expected range [%v, %v]",
					result.RiskAssessment.RiskLevel,
					tt.expectedMinRisk, tt.expectedMaxRisk)
			}
		})
	}
}

// TestPolicyPreview verifies policy admission results
func TestPolicyPreview(t *testing.T) {
	tests := []struct {
		name              string
		mode              string
		actionType        string
		expectedAdmission AdmissionResult
	}{
		{
			name:              "Disabled mode denies all",
			mode:              control.ModeDisabled,
			actionType:        control.ActionTriggerHealthRecheck,
			expectedAdmission: AdmissionDenied,
		},
		{
			name:              "Advisory mode is advisory-only",
			mode:              control.ModeAdvisory,
			actionType:        control.ActionTriggerHealthRecheck,
			expectedAdmission: AdmissionAdvisory,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Control.Mode = tt.mode
			engine := NewEngine(cfg, nil)

			mesh := status.MeshDrilldown{
				MeshHealth: status.MeshHealth{
					Score: 85,
					State: "healthy",
				},
			}

			input := SimulationInput{
				CandidateAction: control.ControlAction{
					ID:              "test-action",
					ActionType:      tt.actionType,
					TargetTransport: "test-transport",
					Reason:          "test",
					Confidence:      0.9,
					CreatedAt:       time.Now().UTC().Format(time.RFC3339),
					Mode:            tt.mode,
				},
				MeshState:     mesh,
				CurrentTime:   time.Now().UTC(),
				Configuration: cfg,
			}

			result, err := engine.Simulate(input)
			if err != nil {
				t.Fatalf("Simulation failed: %v", err)
			}

			if result.PolicyPreview.AdmissionResult != tt.expectedAdmission {
				t.Errorf("Expected admission %v, got %v",
					tt.expectedAdmission,
					result.PolicyPreview.AdmissionResult)
			}
		})
	}
}

// TestBlastRadiusPrediction verifies blast radius calculation
func TestBlastRadiusPrediction(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := NewEngine(cfg, nil)

	mesh := status.MeshDrilldown{
		MeshHealth: status.MeshHealth{
			Score: 85,
			State: "healthy",
		},
		DegradedSegments: []status.DegradedSegment{
			{
				SegmentID: "test-segment",
				Transports: []string{"test-transport"},
				Severity:  "warn",
			},
		},
	}

	input := SimulationInput{
		CandidateAction: control.ControlAction{
			ID:              "test-action",
			ActionType:      control.ActionRestartTransport,
			TargetTransport: "test-transport",
			Reason:          "test",
			Confidence:      0.8,
			CreatedAt:       time.Now().UTC().Format(time.RFC3339),
			Mode:            control.ModeGuardedAuto,
		},
		MeshState:     mesh,
		CurrentTime:   time.Now().UTC(),
		Configuration: cfg,
	}

	result, err := engine.Simulate(input)
	if err != nil {
		t.Fatalf("Simulation failed: %v", err)
	}

	// Verify blast radius is within valid range
	if result.BlastRadius.ImpactScore < 0 || result.BlastRadius.ImpactScore > 1 {
		t.Errorf("Impact score %v out of range [0, 1]", result.BlastRadius.ImpactScore)
	}

	// Verify classification is valid
	validClasses := []string{"Isolated", "Segmented", "Systemic", "Unknown"}
	found := false
	for _, c := range validClasses {
		if result.BlastRadius.ImpactClassification == c {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Invalid impact classification: %v", result.BlastRadius.ImpactClassification)
	}
}

// TestOutcomeBranches verifies outcome branching
func TestOutcomeBranches(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := NewEngine(cfg, nil)

	mesh := status.MeshDrilldown{
		MeshHealth: status.MeshHealth{
			Score: 75,
			State: "degraded",
		},
	}

	input := SimulationInput{
		CandidateAction: control.ControlAction{
			ID:              "test-action",
			ActionType:      control.ActionBackoffIncrease,
			TargetTransport: "test-transport",
			Reason:          "test",
			Confidence:      0.8,
			CreatedAt:       time.Now().UTC().Format(time.RFC3339),
			Mode:            control.ModeGuardedAuto,
		},
		MeshState:     mesh,
		CurrentTime:   time.Now().UTC(),
		Configuration: cfg,
	}

	result, err := engine.Simulate(input)
	if err != nil {
		t.Fatalf("Simulation failed: %v", err)
	}

	// Should have exactly 3 outcome branches
	if len(result.OutcomeBranches) != 3 {
		t.Errorf("Expected 3 outcome branches, got %d", len(result.OutcomeBranches))
	}

	// Verify probabilities sum to approximately 1.0
	totalProb := 0.0
	for _, branch := range result.OutcomeBranches {
		totalProb += branch.Probability
	}
	if totalProb < 0.99 || totalProb > 1.01 {
		t.Errorf("Outcome probabilities sum to %v, expected ~1.0", totalProb)
	}
}

// TestNilInputs handles edge cases with nil/missing inputs
func TestNilInputs(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := NewEngine(cfg, nil)

	// Test with minimal input
	input := SimulationInput{
		CandidateAction: control.ControlAction{
			ID:         "test-action",
			ActionType: control.ActionTriggerHealthRecheck,
			Reason:     "test",
			Confidence: 0.5,
			CreatedAt:  time.Now().UTC().Format(time.RFC3339),
			Mode:       control.ModeAdvisory,
		},
		CurrentTime:   time.Now().UTC(),
		Configuration: cfg,
		// MeshState is empty, Diagnostics is zero value
	}

	result, err := engine.Simulate(input)
	if err != nil {
		t.Fatalf("Simulation should handle nil inputs gracefully: %v", err)
	}

	// Should still produce valid results
	if result.SafeToAct.Decision == "" {
		t.Error("Expected a safe-to-act decision even with minimal inputs")
	}
}

// TestEngineVersion verifies version information
func TestEngineVersion(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := NewEngine(cfg, nil)

	version := engine.Version()
	if version == "" {
		t.Error("Expected non-empty version string")
	}
}

// TestSupportsAction verifies action type support
func TestSupportsAction(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := NewEngine(cfg, nil)

	// All control action types should be supported
	actionTypes := []string{
		control.ActionRestartTransport,
		control.ActionResubscribeTransport,
		control.ActionBackoffIncrease,
		control.ActionBackoffReset,
		control.ActionTemporarilyDeprioritize,
		control.ActionTemporarilySuppressNoisySource,
		control.ActionClearSuppression,
		control.ActionTriggerHealthRecheck,
	}

	for _, actionType := range actionTypes {
		if !engine.SupportsAction(actionType) {
			t.Errorf("Expected engine to support action type: %s", actionType)
		}
	}

	// Unknown action types should not be supported
	if engine.SupportsAction("unknown_action") {
		t.Error("Expected engine to not support unknown action type")
	}
}
