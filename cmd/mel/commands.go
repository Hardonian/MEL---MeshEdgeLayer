package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/cliout"
	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/demo"
	integ "github.com/mel-project/mel/internal/integration"
	"github.com/mel-project/mel/internal/investigation"
	"github.com/mel-project/mel/internal/policy"
	"github.com/mel-project/mel/internal/semantics"
	"github.com/mel-project/mel/internal/support"
	"github.com/mel-project/mel/internal/transport"
	"github.com/mel-project/mel/internal/version"
)

func alertsCmd(args []string) {
	f := fs("alerts")
	path := f.String("config", configFlagDefault(), "config")
	activeOnly := f.Bool("active", true, "only active alerts")
	since := f.String("since", "", "filter last_updated_at >= RFC3339")
	filter := f.String("filter", "", "substring match on id|reason|summary|transport")
	limit := f.Int("limit", 100, "max rows")
	_ = f.Parse(args)
	cfg, _, err := loadConfigFile(*path)
	if err != nil {
		panic(err)
	}
	d := openDB(cfg)
	rows, err := d.TransportAlerts(!*activeOnly)
	if err != nil {
		panic(err)
	}
	var sinceT time.Time
	if *since != "" {
		var err error
		sinceT, err = time.Parse(time.RFC3339, *since)
		if err != nil {
			panic(err)
		}
	}
	fsub := strings.ToLower(strings.TrimSpace(*filter))
	var filtered []db.TransportAlertRecord
	for _, r := range rows {
		if !sinceT.IsZero() {
			if lu, err := time.Parse(time.RFC3339, r.LastUpdatedAt); err == nil && lu.Before(sinceT) {
				continue
			}
		}
		if fsub != "" {
			hay := strings.ToLower(r.ID + " " + r.Reason + " " + r.Summary + " " + r.TransportName)
			if !strings.Contains(hay, fsub) {
				continue
			}
		}
		filtered = append(filtered, r)
		if len(filtered) >= *limit {
			break
		}
	}
	if cliGlobal.JSON {
		mustPrint(map[string]any{"alerts": filtered, "count": len(filtered)})
		return
	}
	headers := []string{"sev", "transport", "reason", "summary", "active", "updated"}
	var table [][]string
	for _, r := range filtered {
		sev := r.Severity
		if cliGlobal.Color {
			sev = semantics.FormatSeverityForTTY(r.Severity, true)
		} else {
			sev = strings.ToUpper(r.Severity)
		}
		table = append(table, []string{
			sev,
			r.TransportName,
			r.Reason,
			r.Summary,
			fmt.Sprintf("%v", r.Active),
			r.LastUpdatedAt,
		})
	}
	cliout.Table(os.Stdout, headers, table, cliGlobal.Wide, 56)
}

func actionsCmd(args []string) {
	if len(args) == 0 {
		panic("usage: mel actions list|pending|history --config <path>")
	}
	switch args[0] {
	case "list", "pending":
		controlCmd(append([]string{"pending"}, args[1:]...))
	case "history":
		controlCmd(append([]string{"history"}, args[1:]...))
	default:
		panic("usage: mel actions list|pending|history --config <path>")
	}
}

func explainCmd(args []string) {
	if len(args) == 0 || args[0] != "policy" {
		panic("usage: mel explain policy --config <path>")
	}
	policyCmd(append([]string{"explain"}, args[1:]...))
}

func supportCmd(args []string) {
	if len(args) == 0 || args[0] != "bundle" {
		panic("usage: mel support bundle --config <path> [--out path.zip]")
	}
	f := fs("support-bundle")
	path := f.String("config", configFlagDefault(), "config")
	out := f.String("out", "", "write zip to path (default: mel-support-<unix>.zip in cwd)")
	_ = f.Parse(args[1:])
	cfg, _, err := loadConfigFile(*path)
	if err != nil {
		panic(err)
	}
	d := openDB(cfg)
	b, err := support.Create(cfg, d, version.GetFullVersionString(), *path, time.Time{})
	if err != nil {
		panic(err)
	}
	z, err := b.ToZip()
	if err != nil {
		panic(err)
	}
	outPath := strings.TrimSpace(*out)
	if outPath == "" {
		outPath = fmt.Sprintf("mel-support-%d.zip", time.Now().Unix())
	}
	if err := os.WriteFile(outPath, z, 0o600); err != nil {
		panic(err)
	}
	mustPrint(map[string]any{"status": "written", "path": outPath, "bytes": len(z)})
}

func demoCmd(args []string) {
	if len(args) == 0 {
		demoUsage()
	}
	switch args[0] {
	case "run":
		rest := args[1:]
		if len(rest) > 0 && rest[0] == "mqtt-local" {
			rest = rest[1:]
		}
		f := fs("demo-run")
		endpoint := f.String("endpoint", "127.0.0.1:18830", "TCP listen address for stub broker")
		topic := f.String("topic", "msh/US/2/e/test", "topic embedded in stub publish")
		_ = f.Parse(rest)
		fmt.Fprintf(os.Stderr, "demo: starting stub MQTT publisher on %s (Ctrl+C to stop)\n", *endpoint)
		fmt.Fprintf(os.Stderr, "demo: this is a local test harness only; it does not connect to a real broker.\n")
		simulateCmd([]string{"--endpoint", *endpoint, "--topic", *topic})
	case "scenarios":
		list := demo.Scenarios()
		if cliGlobal.JSON {
			mustPrint(map[string]any{"scenarios": list, "count": len(list)})
			return
		}
		for _, s := range list {
			fmt.Printf("%s\t[%s]\t%s\n", s.ID, s.Class, s.Title)
		}
	case "replay":
		if len(args) < 2 {
			panic("usage: mel demo replay <scenario-id> [--json]")
		}
		id := args[1]
		rep, ok := demo.ReplayFor(id)
		if !ok {
			panic(fmt.Errorf("unknown scenario %q", id))
		}
		mustPrint(rep)
	case "seed":
		demoSeedCmd(args[1:])
	case "init-sandbox":
		demoInitSandboxCmd(args[1:])
	case "evidence-run":
		demoEvidenceRunCmd(args[1:])
	default:
		demoUsage()
	}
}

func demoUsage() {
	panic(`usage:
  mel demo run [--endpoint host:port] [--topic msh/...]   (stub MQTT publisher; local harness only)
  mel demo scenarios [--json]
  mel demo replay <scenario-id> [--json]
  mel demo seed --scenario <id> --config <path> [--force] [--capture-dir <dir>]
  mel demo init-sandbox --out <path/to/demo-sandbox.json>
  mel demo evidence-run --scenario <id> --config <path> [--capture-dir <dir>] [--skip-capture]`)
}

func demoSeedCmd(args []string) {
	f := fs("demo-seed")
	path := f.String("config", configFlagDefault(), "config (use demo sandbox paths)")
	scenario := f.String("scenario", "", "scenario id (required)")
	force := f.Bool("force", false, "override sandbox path checks")
	captureDir := f.String("capture-dir", "", "optional directory for CLI evidence JSON")
	skipCapture := f.Bool("skip-capture", false, "do not run mel subprocess captures")
	_ = f.Parse(args)
	if strings.TrimSpace(*scenario) == "" {
		panic("--scenario is required")
	}
	cfg, _, err := loadConfigFile(*path)
	if err != nil {
		panic(err)
	}
	bin, _ := os.Executable()
	opt := demo.SeedOptions{Force: *force, CaptureDir: *captureDir, MELBinary: bin, SkipCapture: *skipCapture}
	bundle, err := demo.Execute(cfg, *scenario, opt)
	if err != nil {
		panic(err)
	}
	bundle.ConfigPath = *path
	if *captureDir != "" {
		bundle.EvidenceDir = *captureDir
	}
	if *captureDir != "" && !*skipCapture {
		bundle.CLIOutputs = captureDemoCLI(bin, *path, *captureDir)
	}
	_ = writeDemoEvidenceBundle(cfg, bundle)
	mustPrint(map[string]any{"status": "seeded", "scenario": *scenario, "bundle": bundle})
}

func demoInitSandboxCmd(args []string) {
	f := fs("demo-init-sandbox")
	out := f.String("out", "demo-sandbox/mel.demo.json", "output config path")
	_ = f.Parse(args)
	dir := filepath.Dir(*out)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(*out, demo.DefaultSandboxConfigBytes(), 0o600); err != nil {
		panic(err)
	}
	mustPrint(map[string]any{"status": "written", "path": *out, "hint": "Run: mel demo seed --scenario <id> --config " + *out})
}

func demoEvidenceRunCmd(args []string) {
	f := fs("demo-evidence-run")
	path := f.String("config", configFlagDefault(), "config")
	scenario := f.String("scenario", "", "scenario id (required)")
	captureDir := f.String("capture-dir", "", "directory for evidence JSON (default: <data_dir>/demo_evidence)")
	skipCapture := f.Bool("skip-capture", false, "seed only; no CLI captures")
	force := f.Bool("force", false, "override sandbox path checks")
	_ = f.Parse(args)
	if strings.TrimSpace(*scenario) == "" {
		panic("--scenario is required")
	}
	cfg, _, err := loadConfigFile(*path)
	if err != nil {
		panic(err)
	}
	outDir := strings.TrimSpace(*captureDir)
	if outDir == "" {
		outDir = filepath.Join(cfg.Storage.DataDir, "demo_evidence")
	}
	bin, _ := os.Executable()
	bundle, err := demo.Execute(cfg, *scenario, demo.SeedOptions{Force: *force, CaptureDir: outDir, MELBinary: bin, SkipCapture: *skipCapture})
	if err != nil {
		panic(err)
	}
	bundle.ConfigPath = *path
	bundle.EvidenceDir = outDir
	if !*skipCapture {
		bundle.CLIOutputs = captureDemoCLI(bin, *path, outDir)
	}
	_ = writeDemoEvidenceBundle(cfg, bundle)
	mustPrint(map[string]any{"status": "evidence_run_complete", "scenario": *scenario, "capture_dir": outDir, "bundle": bundle})
}

func captureDemoCLI(melBin, cfgPath, dir string) []string {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	outputs := []string{}
	run := func(name string, args ...string) {
		outPath := filepath.Join(dir, name+".json")
		cmd := exec.Command(melBin, args...)
		b, err := cmd.Output()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				b = append(b, ee.Stderr...)
			}
		}
		if name == "doctor" {
			if idx := bytes.IndexByte(b, '\n'); idx >= 0 {
				b = b[:idx]
			}
		}
		if err := os.WriteFile(outPath, b, 0o644); err != nil {
			panic(err)
		}
		outputs = append(outputs, outPath)
	}
	run("doctor", "doctor", "--config", cfgPath, "--json")
	run("inspect_mesh", "inspect", "mesh", "--config", cfgPath)
	run("privacy_audit", "privacy", "audit", "--config", cfgPath, "--format", "json")
	run("status", "status", "--config", cfgPath)
	return outputs
}

func writeDemoEvidenceBundle(cfg config.Config, bundle demo.DemoEvidenceBundle) error {
	dir := filepath.Dir(cfg.Storage.DatabasePath)
	b, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "demo_evidence_bundle.json"), b, 0o644)
}

func devCmd(args []string) {
	if len(args) == 0 {
		panic("usage: mel dev run [-- mel serve args...]")
	}
	switch args[0] {
	case "run":
		rest := args[1:]
		bin, err := os.Executable()
		if err != nil {
			bin = "mel"
		}
		cmdline := append([]string{bin, "serve"}, rest...)
		mustPrint(map[string]any{
			"hint":    "Run the daemon from your shell; this command only prints the recommended invocation.",
			"example": strings.Join(cmdline, " "),
		})
	default:
		panic("usage: mel dev run [-- mel serve args...]")
	}
}

func modeCmd(args []string) {
	f := fs("mode")
	path := f.String("config", configFlagDefault(), "config")
	_ = f.Parse(args)
	cfg, _ := loadCfg([]string{"--config", *path})
	d := openDB(cfg)
	eval, err := control.Evaluate(cfg, d, nil, time.Now().UTC())
	if err != nil {
		panic(err)
	}
	posture := config.SecurityBanner(cfg)
	mustPrint(map[string]any{
		"control_mode":           cfg.Control.Mode,
		"emergency_disable":      cfg.Control.EmergencyDisable,
		"execution_summary":      eval.Explanation,
		"security_posture":       posture,
		"policy_recommendations": policy.Explain(cfg),
	})
}

func integrationCmd(args []string) {
	if len(args) == 0 || args[0] != "test" {
		panic("usage: mel integration test --url <https://example.com/hook> [--event-type mel.test]")
	}
	f := flag.NewFlagSet("integration-test", flag.ExitOnError)
	url := f.String("url", "", "webhook URL to POST a test event (required)")
	eventType := f.String("event-type", "mel.integration.test", "event_type field")
	_ = f.Parse(args[1:])
	if strings.TrimSpace(*url) == "" {
		panic("--url is required")
	}
	client := &integ.Client{UserAgent: "mel-cli/1.0"}
	ev := integ.Event{
		SchemaVersion: "mel.integration.v1",
		EventType:     *eventType,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Source:        "mel-cli",
		Summary:       "MEL integration connectivity test",
		Details:       map[string]any{"cli": true},
	}
	res := client.DeliverWebhook(context.Background(), *url, ev)
	mustPrint(res)
	if !res.Success {
		os.Exit(1)
	}
}

func traceCmd(args []string) {
	if len(args) < 1 {
		panic("usage: mel trace <action-id> --config <path>")
	}
	actionID := args[0]
	cfg, _ := loadCfg(args[1:])
	d := openDB(cfg)
	action, ok, err := d.ControlActionByID(actionID)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("action not found: " + actionID)
	}
	bundle, bundleOK, _ := d.EvidenceBundleByActionID(actionID)
	notes, _ := d.OperatorNotesByRef("action", actionID, 50)
	decisions, _ := d.ControlDecisions("", actionID, "", "", 20, 0)
	out := map[string]any{
		"action":            action,
		"evidence_bundle":   nil,
		"operator_notes":    notes,
		"related_decisions": decisions,
	}
	if bundleOK {
		out["evidence_bundle"] = bundle
	}
	mustPrint(out)
}
func investigateCmd(args []string) {
	mode := "summary"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		mode = args[0]
		args = args[1:]
	}

	switch mode {
	case "summary", "list":
		f := fs("investigate")
		path := f.String("config", configFlagDefault(), "config")
		_ = f.Parse(args)
		summary := deriveInvestigationSummary(*path)
		if cliGlobal.JSON {
			mustPrint(summary)
			return
		}
		printInvestigationSummaryText(summary)
	case "cases":
		f := fs("investigate-cases")
		path := f.String("config", configFlagDefault(), "config")
		_ = f.Parse(args)
		summary := deriveInvestigationSummary(*path)
		if cliGlobal.JSON {
			mustPrint(map[string]any{
				"generated_at": summary.GeneratedAt,
				"case_counts":  summary.CaseCounts,
				"count":        len(summary.Cases),
				"cases":        summary.Cases,
			})
			return
		}
		printInvestigationCasesText(summary)
	case "show":
		if len(args) == 0 {
			panic("usage: mel investigate show <case-id> --config <path>")
		}
		caseID := strings.TrimSpace(args[0])
		f := fs("investigate-show")
		path := f.String("config", configFlagDefault(), "config")
		_ = f.Parse(args[1:])
		summary := deriveInvestigationSummary(*path)
		detail, ok := summary.CaseDetail(caseID)
		if !ok {
			panic("investigation case not found: " + caseID)
		}
		if cliGlobal.JSON {
			mustPrint(detail)
			return
		}
		printInvestigationCaseDetailText(detail)
	default:
		panic("usage: mel investigate [--config <path>] | mel investigate cases --config <path> | mel investigate show <case-id> --config <path>")
	}
}

func deriveInvestigationSummary(path string) investigation.Summary {
	cfg, _, err := loadConfigFile(path)
	if err != nil {
		panic(err)
	}
	d := openDB(cfg)
	runtimeStates, _ := d.TransportRuntimeStatuses()
	var mockHealth []transport.Health
	for _, rs := range runtimeStates {
		mockHealth = append(mockHealth, transport.Health{
			Name:            rs.Name,
			Type:            rs.Type,
			Source:          rs.Source,
			State:           rs.State,
			LastAttemptAt:   rs.LastAttemptAt,
			LastError:       rs.LastError,
			LastConnectedAt: rs.LastConnectedAt,
			LastSuccessAt:   rs.LastSuccessAt,
			LastIngestAt:    rs.LastMessageAt,
			LastHeartbeatAt: rs.LastHeartbeatAt,
			LastFailureAt:   rs.LastFailureAt,
			TotalMessages:   rs.TotalMessages,
			PacketsDropped:  rs.PacketsDropped,
			FailureCount:    rs.FailureCount,
		})
	}
	return investigation.Derive(cfg, d, mockHealth, runtimeStates, time.Now().UTC())
}

func printInvestigationSummaryText(summary investigation.Summary) {
	fmt.Printf("\nMEL CANONICAL INVESTIGATION [truthful; bounded; evidence-backed]\n")
	fmt.Printf("Generated: %s | Attention: %s | Certainty: %.2f\n\n",
		summary.GeneratedAt,
		semantics.FormatSeverityForTTY(string(summary.OverallAttention), cliGlobal.Color),
		summary.OverallCertainty)

	fmt.Printf("HEADLINE: %s\n", summary.Headline)
	fmt.Printf("CONTEXT:  %s\n\n", summary.AttentionSummary)

	if len(summary.Cases) > 0 {
		fmt.Printf("--- CASES (%d) [bounded operator attention objects] ---\n", len(summary.Cases))
		for i, c := range summary.Cases {
			sev := semantics.FormatSeverityForTTY(string(c.Attention), cliGlobal.Color)
			fmt.Printf("[%d] %s | %s | %.2f cert | %s\n", i+1, sev, c.Status, c.Certainty, c.Title)
			fmt.Printf("    ID:             %s\n", c.ID)
			fmt.Printf("    SUMMARY:        %s\n", c.Summary)
			fmt.Printf("    WHY IT MATTERS: %s\n", c.WhyItMatters)
			if c.MissingEvidence != "" {
				fmt.Printf("    MISSING:        %s\n", c.MissingEvidence)
			}
			fmt.Printf("    SAFE TO CONSIDER: %s\n", c.SafeToConsider)
			fmt.Printf("    OUT OF SCOPE:   %s\n", c.OutOfScope)
			fmt.Println()
		}
	}

	if len(summary.Findings) > 0 {
		fmt.Printf("--- FINDINGS (%d) ---\n", len(summary.Findings))
		for i, finding := range summary.Findings {
			sev := semantics.FormatSeverityForTTY(string(finding.Attention), cliGlobal.Color)
			cert := fmt.Sprintf("%.2f cert", finding.Certainty)
			fmt.Printf("[%d] %s | %s | %s\n", i+1, sev, cert, finding.Title)
			fmt.Printf("    ID:             %s\n", finding.ID)
			fmt.Printf("    WHY IT MATTERS: %s\n", finding.WhyItMatters)
			fmt.Printf("    EXPLANATION:    %s\n", finding.Explanation)
			if finding.ResourceID != "" {
				fmt.Printf("    RESOURCE:       %s\n", finding.ResourceID)
			}
			if len(finding.EvidenceGapIDs) > 0 {
				fmt.Printf("    CAVEATS:        Limited by %d evidence gap(s)\n", len(finding.EvidenceGapIDs))
			}
			fmt.Println()
		}
	}

	if len(summary.EvidenceGaps) > 0 {
		fmt.Printf("--- EVIDENCE GAPS (%d) [missing truth; absence of evidence] ---\n", len(summary.EvidenceGaps))
		for i, gap := range summary.EvidenceGaps {
			fmt.Printf("[%d] %s: %s\n", i+1, gap.Reason, gap.Title)
			fmt.Printf("    EXPLANATION: %s\n", gap.Explanation)
			fmt.Printf("    IMPACT:      %s\n", gap.Impact)
			fmt.Println()
		}
	}

	if len(summary.Recommendations) > 0 {
		fmt.Printf("--- RECOMMENDATIONS (%d) [evidence-grounded next steps] ---\n", len(summary.Recommendations))
		for i, rec := range summary.Recommendations {
			fmt.Printf("[%d] %s\n", i+1, rec.Action)
			fmt.Printf("    RATIONALE: %s\n", rec.Rationale)
			if len(rec.UncertaintyLimits) > 0 {
				fmt.Printf("    CAVEATS:   %s\n", strings.Join(rec.UncertaintyLimits, " | "))
			}
			fmt.Printf("    AUTHORITY: %s\n", rec.ActionAuthority)
			fmt.Println()
		}
	}

	fmt.Printf("--- PHYSICS BOUNDARY ---\n")
	for _, statement := range summary.PhysicsBoundary.Statements {
		fmt.Printf("• %s\n", statement)
	}
	fmt.Println()
}

func printInvestigationCasesText(summary investigation.Summary) {
	fmt.Printf("\nMEL INVESTIGATION CASES\n")
	fmt.Printf("Generated: %s | Cases: %d\n\n", summary.GeneratedAt, len(summary.Cases))
	for i, c := range summary.Cases {
		sev := semantics.FormatSeverityForTTY(string(c.Attention), cliGlobal.Color)
		fmt.Printf("[%d] %s | %s | %.2f cert | %s\n", i+1, sev, c.Status, c.Certainty, c.Title)
		fmt.Printf("    ID:      %s\n", c.ID)
		fmt.Printf("    SUMMARY: %s\n", c.Summary)
		if c.MissingEvidence != "" {
			fmt.Printf("    MISSING: %s\n", c.MissingEvidence)
		}
		fmt.Println()
	}
}

func printInvestigationCaseDetailText(detail investigation.CaseDetail) {
	c := detail.Case
	fmt.Printf("\nMEL INVESTIGATION CASE DETAIL\n")
	fmt.Printf("ID: %s\n", c.ID)
	fmt.Printf("Title: %s\n", c.Title)
	fmt.Printf("Attention: %s | Status: %s | Certainty: %.2f\n\n",
		semantics.FormatSeverityForTTY(string(c.Attention), cliGlobal.Color), c.Status, c.Certainty)

	fmt.Printf("SUMMARY: %s\n", c.Summary)
	fmt.Printf("WHY IT MATTERS: %s\n", c.WhyItMatters)
	if c.MissingEvidence != "" {
		fmt.Printf("MISSING EVIDENCE: %s\n", c.MissingEvidence)
	}
	fmt.Printf("SAFE TO CONSIDER: %s\n", c.SafeToConsider)
	fmt.Printf("OUT OF SCOPE: %s\n\n", c.OutOfScope)

	if len(detail.Findings) > 0 {
		fmt.Printf("--- FINDINGS ---\n")
		for _, finding := range detail.Findings {
			fmt.Printf("%s | %.2f cert | %s\n", semantics.FormatSeverityForTTY(string(finding.Attention), cliGlobal.Color), finding.Certainty, finding.Title)
			fmt.Printf("  %s\n", finding.Explanation)
		}
		fmt.Println()
	}

	if len(detail.EvidenceGaps) > 0 {
		fmt.Printf("--- EVIDENCE GAPS ---\n")
		for _, gap := range detail.EvidenceGaps {
			fmt.Printf("%s: %s\n", gap.Reason, gap.Title)
			fmt.Printf("  %s\n", gap.Impact)
		}
		fmt.Println()
	}

	if len(detail.Recommendations) > 0 {
		fmt.Printf("--- RECOMMENDATIONS ---\n")
		for _, rec := range detail.Recommendations {
			fmt.Printf("%s\n", rec.Action)
			fmt.Printf("  %s\n", rec.Rationale)
			if len(rec.UncertaintyLimits) > 0 {
				fmt.Printf("  Caveats: %s\n", strings.Join(rec.UncertaintyLimits, " | "))
			}
		}
		fmt.Println()
	}

	if len(c.RelatedRecords) > 0 {
		fmt.Printf("--- RELATED RECORDS ---\n")
		for _, record := range c.RelatedRecords {
			fmt.Printf("%s | %s | %s\n", record.Kind, record.ID, record.Summary)
			if record.InspectCLI != "" {
				fmt.Printf("  CLI: %s\n", record.InspectCLI)
			}
		}
		fmt.Println()
	}
}
