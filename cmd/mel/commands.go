package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/cliout"
	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
	integ "github.com/mel-project/mel/internal/integration"
	"github.com/mel-project/mel/internal/policy"
	"github.com/mel-project/mel/internal/semantics"
	"github.com/mel-project/mel/internal/support"
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
	b, err := support.Create(cfg, d, version.GetFullVersionString())
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
	if len(args) == 0 || args[0] != "run" {
		panic("usage: mel demo run mqtt-local [--endpoint host:port] [--topic msh/...]")
	}
	f := fs("demo-run")
	endpoint := f.String("endpoint", "127.0.0.1:18830", "TCP listen address for stub broker")
	topic := f.String("topic", "msh/US/2/e/test", "topic embedded in stub publish")
	_ = f.Parse(args[1:])
	fmt.Fprintf(os.Stderr, "demo: starting stub MQTT publisher on %s (Ctrl+C to stop)\n", *endpoint)
	fmt.Fprintf(os.Stderr, "demo: this is a local test harness only; it does not connect to a real broker.\n")
	simulateCmd([]string{"--endpoint", *endpoint, "--topic", *topic})
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
