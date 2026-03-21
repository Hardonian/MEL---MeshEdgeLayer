package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/simulation"
)

func actionSimulateCmd(args []string) {
	if len(args) == 0 {
		actionSimulateUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "action":
		simulateActionSubCmd(args[1:])
	default:
		actionSimulateUsage()
		os.Exit(1)
	}
}

func actionSimulateUsage() {
	fmt.Println(`mel simulate commands:
  simulate action <action-type> --transport <name> --config <path> [--format text|json] [--output <file>]

Available action types:
  restart_transport, resubscribe_transport, backoff_increase, backoff_reset,
  temporarily_deprioritize_transport, temporarily_suppress_noisy_source,
  clear_suppression, trigger_health_recheck

Examples:
  mel simulate action restart_transport --transport mqtt-primary --config mel.json
  mel simulate action backoff_increase --transport serial --format json`)
}

func simulateActionSubCmd(args []string) {
	f := flag.NewFlagSet("simulate-action", flag.ExitOnError)
	configPath := f.String("config", configFlagDefault(), "path to config file")
	transportName := f.String("transport", "", "target transport name")
	format := f.String("format", "text", "output format: text or json")
	outputPath := f.String("output", "", "write output to file (optional)")
	f.Parse(args)

	if len(f.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "Error: action type required")
		actionSimulateUsage()
		os.Exit(1)
	}

	actionType := f.Args()[0]
	validTypes := map[string]bool{
		control.ActionRestartTransport:               true,
		control.ActionResubscribeTransport:           true,
		control.ActionBackoffIncrease:                true,
		control.ActionBackoffReset:                   true,
		control.ActionTemporarilyDeprioritize:        true,
		control.ActionTemporarilySuppressNoisySource: true,
		control.ActionClearSuppression:               true,
		control.ActionTriggerHealthRecheck:           true,
	}

	if !validTypes[actionType] {
		fmt.Fprintf(os.Stderr, "Error: unknown action type: %s\n", actionType)
		actionSimulateUsage()
		os.Exit(1)
	}

	cfg, _, err := loadConfigFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	database, err := db.Open(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	now := time.Now().UTC()

	input := simulation.SimulationInput{
		SimulationID: fmt.Sprintf("sim-%d", now.UnixNano()),
		Timestamp:    now,
		ProposedAction: control.ControlAction{
			ID:              fmt.Sprintf("sim-%d", now.UnixNano()),
			ActionType:      actionType,
			TargetTransport: *transportName,
			Reason:          "operator-requested simulation",
			Confidence:      0.8,
			CreatedAt:       now.Format(time.RFC3339),
			Mode:            cfg.Control.Mode,
		},
	}

	// Run simulation
	engine := simulation.NewEngine(cfg, database)
	result, err := engine.Simulate(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Simulation error: %v\n", err)
		os.Exit(1)
	}

	// Output results
	if *format == "json" {
		outputSimulationJSON(result, *outputPath)
	} else {
		outputSimulationText(result, actionType, *transportName)
	}
}

func outputSimulationJSON(result simulation.SimulationResult, outputPath string) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	if outputPath != "" {
		if err := os.WriteFile(outputPath, data, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Simulation results written to: %s\n", outputPath)
	} else {
		fmt.Println(string(data))
	}
}

func outputSimulationText(result simulation.SimulationResult, actionType, transportName string) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     MEL ACTION SIMULATION RESULTS                            ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Action Summary
	fmt.Printf("Action:        %s\n", actionType)
	fmt.Printf("Target:        %s\n", transportName)
	fmt.Printf("Timestamp:     %s\n", result.CompletedAt.Format(time.RFC3339))
	fmt.Printf("Engine:        %s\n", result.Metadata.ModelVersion)
	fmt.Println()

	// Safe to Act Decision
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ SAFE-TO-ACT DECISION                                                        │")
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
	fmt.Printf("Decision:      %s\n", result.SafeToAct.Decision)
	fmt.Printf("Confidence:    %.0f%%\n", result.SafeToAct.Confidence*100)
	if result.SafeToAct.PrimaryReason != "" {
		fmt.Printf("Reason:        %s\n", result.SafeToAct.PrimaryReason)
	}
	fmt.Println()

	// Predicted Outcome
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ PREDICTED OUTCOME                                                           │")
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
	fmt.Printf("Success Probability: %.0f%%\n", result.PredictedOutcome.SuccessProbability*100)
	fmt.Printf("Expected State:      %s\n", result.PredictedOutcome.ExpectedState)
	if len(result.PredictedOutcome.SideEffects) > 0 {
		fmt.Println("Side Effects:")
		for _, effect := range result.PredictedOutcome.SideEffects {
			fmt.Printf("  • [%s] %s\n", effect.Component, effect.Effect)
		}
	}
	fmt.Println()

	// Risk Assessment
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ RISK ASSESSMENT                                                             │")
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
	fmt.Printf("Risk Level:    %s\n", result.RiskAssessment.OverallRisk)
	fmt.Printf("Safety Level:  %s\n", result.RiskAssessment.SafetyLevel)
	fmt.Printf("Confidence:    %.2f/1.0\n", result.RiskAssessment.Confidence)
	if len(result.RiskAssessment.RiskFactors) > 0 {
		fmt.Println("Risk Factors:")
		for _, factor := range result.RiskAssessment.RiskFactors {
			fmt.Printf("  • [%s] %s\n", factor.Category, factor.Description)
		}
	}
	fmt.Println()

	// Policy Preview
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ POLICY PREVIEW                                                              │")
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
	fmt.Printf("Admission:     %s\n", result.PolicyPreview.Result)
	if !result.PolicyPreview.Allowed {
		fmt.Printf("Denial Reason: %s\n", result.PolicyPreview.DenialReason)
	}
	fmt.Println()

	// Blast Radius
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ BLAST RADIUS PREDICTION                                                     │")
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
	fmt.Printf("Impact Score:  %.0f%%\n", result.BlastRadius.Score*100)
	fmt.Printf("Classification: %s\n", result.BlastRadius.Classification)
	fmt.Printf("Description:   %s\n", result.BlastRadius.Description)
	if len(result.BlastRadius.AffectedTransports) > 0 {
		fmt.Printf("Transports:    %s\n", strings.Join(result.BlastRadius.AffectedTransports, ", "))
	}
	if len(result.BlastRadius.AffectedNodes) > 0 {
		fmt.Printf("Nodes:         %s\n", strings.Join(result.BlastRadius.AffectedNodes, ", "))
	}
	fmt.Println()

	// Conflicts
	if len(result.Conflicts) > 0 {
		fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
		fmt.Println("│ DETECTED CONFLICTS                                                          │")
		fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
		for _, conflict := range result.Conflicts {
			fmt.Printf("[%s] %s: %s\n", conflict.Severity, conflict.Type, conflict.Description)
			if conflict.Resolution != "" {
				fmt.Printf("  Resolution: %s\n", conflict.Resolution)
			}
		}
		fmt.Println()
	}

	// Outcome Branches
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ OUTCOME BRANCHES                                                            │")
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
	for _, branch := range result.OutcomeBranches {
		fmt.Printf("\n[%s] Probability: %.0f%%\n", branch.Scenario, branch.Probability*100)
		fmt.Printf("  Description: %s\n", branch.Description)
		if len(branch.TriggeringConditions) > 0 {
			fmt.Printf("  Conditions: %s\n", strings.Join(branch.TriggeringConditions, "; "))
		}
	}
	fmt.Println()

	// Guidance and Next Steps
	if result.SafeToAct.OperatorGuidance != "" {
		fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
		fmt.Println("│ OPERATOR GUIDANCE                                                           │")
		fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
		fmt.Printf("  %s\n", result.SafeToAct.OperatorGuidance)
		fmt.Println()
	}

	if len(result.SafeToAct.AlternativeActions) > 0 {
		fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
		fmt.Println("│ ALTERNATIVE ACTIONS                                                         │")
		fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
		for _, alt := range result.SafeToAct.AlternativeActions {
			fmt.Printf("  • %s\n", alt)
		}
		fmt.Println()
	}

	// Footer
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("This is a predictive simulation based on current system state.")
	fmt.Println("Actual outcomes may vary depending on external factors.")
	fmt.Println()
}
