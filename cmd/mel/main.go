package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/mel-project/mel/internal/backup"
	"github.com/mel-project/mel/internal/cliout"
	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/diagnostics"
	"github.com/mel-project/mel/internal/doctor"
	"github.com/mel-project/mel/internal/fleet"
	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/operatorlang"
	"github.com/mel-project/mel/internal/policy"
	"github.com/mel-project/mel/internal/privacy"
	"github.com/mel-project/mel/internal/security"
	"github.com/mel-project/mel/internal/service"
	statuspkg "github.com/mel-project/mel/internal/status"
	"github.com/mel-project/mel/internal/ui"
	"github.com/mel-project/mel/internal/version"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "mel error: %v\n", r)
			os.Exit(1)
		}
	}()
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	rest, g := parseGlobalFlags(os.Args[2:])
	cliGlobal = g
	cmd := os.Args[1]
	switch cmd {
	case "init":
		initCmd(rest)
	case "version":
		fmt.Println(version.GetFullVersionString())
	case "config":
		configCmd(rest)
	case "serve":
		serveCmd(rest)
	case "doctor":
		doctorCmd(rest)
	case "preflight":
		preflightCmd(rest)
	case "bootstrap":
		bootstrapCmd(rest)
	case "upgrade":
		upgradeCmd(rest)
	case "audit":
		auditCmd(rest)
	case "status":
		statusCmd(rest)
	case "fleet":
		fleetCmd(rest)
	case "panel":
		panelCmd(rest)
	case "nodes":
		nodesCmd(rest)
	case "node":
		nodeCmd(rest)
	case "transports":
		transportsCmd(rest)
	case "inspect":
		inspectCmd(rest)
	case "db":
		dbCmd(rest)
	case "export":
		exportCmd(rest)
	case "import":
		importCmd(rest)
	case "logs":
		logsCmd(rest)
	case "policy":
		policyCmd(rest)
	case "control":
		controlCmd(rest)
	case "action":
		actionCmd(rest)
	case "incident":
		incidentCmd(rest)
	case "timeline":
		timelineCmd(rest)
	case "freeze":
		freezeCmd(rest)
	case "maintenance":
		maintenanceCmd(rest)
	case "notes":
		notesCmd(rest)
	case "privacy":
		privacyCmd(rest)
	case "backup":
		backupCmd(rest)
	case "replay":
		replayCmd(rest)
	case "diagnostics":
		diagnosticsCmd(rest)
	case "health":
		healthCmd(rest)
	case "dev-simulate-mqtt":
		simulateCmd(rest)
	case "simulate":
		actionSimulateCmd(rest)
	case "ui":
		uiCmd(rest)
	case "gui":
		guiCmd(rest)
	case "alerts":
		alertsCmd(rest)
	case "actions":
		actionsCmd(rest)
	case "explain":
		explainCmd(rest)
	case "support":
		supportCmd(rest)
	case "demo":
		demoCmd(rest)
	case "dev":
		devCmd(rest)
	case "mode":
		modeCmd(rest)
	case "integration":
		integrationCmd(rest)
	case "trace":
		traceCmd(rest)
	// Distributed kernel commands
	case "kernel-status":
		kernelStatusCmd(rest)
	case "kernel-replay":
		kernelReplayCmd(rest)
	case "kernel-snapshot":
		kernelSnapshotCmd(rest)
	case "kernel-backup":
		kernelBackupCmd(rest)
	case "kernel-eventlog":
		kernelEventlogCmd(rest)
	case "federation":
		federationCmd(rest)
	case "peers":
		peersCmd(rest)
	case "region":
		regionCmd(rest)
	case "topology":
		topologyCmd(rest)
	case "mesh":
		meshCmd(rest)
	case "plan":
		planCmd(rest)
	case "playbook":
		playbookCmd(rest)
	case "recommend":
		recommendCmd(rest)
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`mel commands:
Global flags (before subcommand): --config <path> --profile <name> --json|--text --wide --color|--no-color

  init
  version
  bootstrap run|validate --config <path> [--dry-run]
  upgrade preflight --config <path>
  audit verify --config <path>
  doctor --config <path>
  preflight --config <path> [--skip-serve-check]
  config validate|show|inspect|diff|risk|keys --config <path>
  serve [--debug] --config <path>
  status --config <path>
  fleet truth --config <path>
  fleet evidence import --file <path.json> --config <path> [--strict-origin] [--actor id]
  fleet evidence list --config <path> [--limit n]
  fleet evidence show <import-id> --config <path>
  panel [--format text|json] --config <path>
  nodes --config <path>
  node inspect <node-id> --config <path>  (see also: node whatif)
  transports list --config <path>
  inspect transport <name> --config <path>
  inspect mesh --config <path>
  inspect topology [--refresh] --config <path>
  mesh bootstrap|topology|diagnose|recommend|inspect|history|simulate|resilience|critical --config <path>
  plan create|list|show|edit|compare|inputs|mark-step|outcome --config <path>
  playbook suggest --config <path>
  recommend next --config <path>
  alerts --config <path> [--active=false] [--since RFC3339] [--filter s] [--limit n]
  actions list|pending|history --config <path>
  explain policy --config <path>
  mode --config <path>
  replay run --config <path> [--node <id>] [--type <message-type>] [--since RFC3339] [--filter s] [--limit <n>]
  replay diff --config <a> --against <b> [--url <daemon>] [--mode dry_run|...] [--since|--until RFC3339]
  replay kernel ... (alias for mel kernel-replay)
  privacy audit [--format json|text] --config <path>
  policy explain --config <path>
  control status --config <path>
  control history --config <path> [--transport <name>] [--limit <n>]
  control approve <action-id> --config <path> [--note "..."] [--i-understand-break-glass-sod]
  control reject <action-id> --config <path> [--note "..."] [--i-understand-break-glass-sod]
  control pending --config <path>
  control inspect <action-id> --config <path>
  control operational-state --config <path>
  action list|pending|inspect|queue|approve|reject --config <path>  (trust-aligned control actions; approve/reject use full service path)
  incident inspect <id> --config <path>
  incident handoff <id> --to <operator> --summary "..." --config <path> [--pending-actions id1,id2] [--recent-actions ...] [--risks ...]
  freeze create --config <path> --reason "..." [--scope-type global|transport|action_type] [--scope-value <name>] [--expires-at <RFC3339>]
  freeze list --config <path>
  freeze clear <freeze-id> --config <path>
  maintenance create --config <path> --starts-at <RFC3339> --ends-at <RFC3339> [--title "..."] [--reason "..."]
  maintenance list --config <path>
  maintenance cancel <window-id> --config <path>
  timeline --config <path> [--start <RFC3339>] [--end <RFC3339>] [--limit <n>]
  notes add --config <path> --ref-type <type> --ref-id <id> --content "..."
  notes list --config <path> --ref-type <type> --ref-id <id>
  export --config <path> [--out path]
  import validate --bundle <path>
  backup create --config <path> [--out path]
  backup restore --bundle <path> --dry-run (required) [--destination dir]
  logs tail --config <path> [--limit n] [--category s] [--since RFC3339] [--filter s]
  db vacuum --config <path>
  health internal|freshness|slo|metrics|trust --config <path>
  support bundle --config <path> [--out path.zip]
  integration test --url <https://hook> [--event-type mel.test]
  trace <action-id> --config <path>
  demo run [mqtt-local] [--endpoint host:port] [--topic msh/...]
  demo scenarios [--json]
  demo replay <scenario-id>
  demo seed --scenario <id> --config <path> [--force] [--capture-dir <dir>]
  demo init-sandbox --out <path>
  demo evidence-run --scenario <id> --config <path> [--capture-dir <dir>]
  dev run (prints recommended serve invocation)
  ui --config <path>
  gui --config <path>
  simulate action <type> --transport <name> --config <path>
  dev-simulate-mqtt

Distributed kernel commands (require running daemon):
  kernel-status --config <path> [--url <url>]
  kernel-replay --config <path> [--url <url>] [--mode full|windowed|scenario|dry_run|verification]
                [--from-seq <n>] [--to-seq <n>] [--since <RFC3339>] [--until <RFC3339>]
                [--max-events <n>] [--policy-mode <mode>] [--compact]
  kernel-snapshot list --config <path> [--url <url>] [--limit <n>]
  kernel-snapshot create --config <path> [--url <url>]
  kernel-backup list --config <path> [--url <url>]
  kernel-backup create --config <path> [--url <url>]
  kernel-eventlog stats --config <path> [--url <url>]
  kernel-eventlog query --config <path> [--url <url>] [--type <event-type>] [--node <node-id>]
                        [--subject <s>] [--since <RFC3339>] [--until <RFC3339>] [--limit <n>]
  federation status --config <path> [--url <url>]
  federation sync-health --config <path> [--url <url>]
  peers list --config <path> [--url <url>]
  region health [--region <id>] --config <path> [--url <url>]
  topology global --config <path> [--url <url>]

Topology model commands (require running daemon):
  topology nodes --config <path> [--url <url>]
  topology node <node_num> --config <path> [--url <url>]
  topology links --config <path> [--url <url>]
  topology analysis --config <path> [--url <url>]
  topology snapshots --config <path> [--url <url>]
  topology sources --config <path> [--url <url>]
  topology bookmarks --config <path> [--url <url>]
  topology export --config <path> [--url <url>]
  topology recovery --config <path> [--url <url>]`)
}

func fs(name string) *flag.FlagSet { return flag.NewFlagSet(name, flag.ExitOnError) }

func loadCfg(args []string) (config.Config, string) {
	f := fs("cfg")
	path := f.String("config", configFlagDefault(), "config")
	_ = f.Parse(args)
	cfg, _, err := config.LoadWithOptions(*path, config.LoadOptions{Profile: cliGlobal.Profile})
	if err != nil {
		panic(err)
	}
	return cfg, *path
}

func initCmd(args []string) {
	f := fs("init")
	path := f.String("config", "configs/mel.generated.json", "config output path")
	force := f.Bool("force", false, "overwrite existing file")
	_ = f.Parse(args)
	if _, err := os.Stat(*path); err == nil && !*force {
		panic(fmt.Errorf("config already exists at %s; use --force to overwrite", *path))
	}
	cfg, err := config.WriteInit(*path)
	if err != nil {
		panic(err)
	}
	mustPrint(map[string]any{"status": "initialized", "config": *path, "bind": cfg.Bind.API, "database": cfg.Storage.DatabasePath})
	fmt.Printf("\nSUCCESS: Configuration initialized at %s\n", *path)
	fmt.Println("NEXT STEP: Run 'mel doctor' to verify your environment, database permissions, and device connectivity.")
}

func configCmd(args []string) {
	if len(args) == 0 {
		panic("usage: mel config validate|show|inspect|diff|risk|keys [--key prefix] [--format text|json] --config <path>")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "validate":
		f := fs("config-validate")
		path := f.String("config", configFlagDefault(), "config")
		_ = f.Parse(rest)
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		findings := doctor.ValidateConfigFile(*path, cfg)
		mustPrint(map[string]any{"status": map[bool]string{true: "valid", false: "invalid"}[len(findings) == 0], "findings": findings, "lints": config.LintConfig(cfg)})
		if len(findings) > 0 {
			os.Exit(1)
		}
	case "inspect":
		f := fs("config-inspect")
		path := f.String("config", configFlagDefault(), "config")
		_ = f.Parse(rest)
		cfg, loadedBytes, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		mustPrint(config.Inspect(cfg, loadedBytes))
	case "show":
		f := fs("config-show")
		path := f.String("config", configFlagDefault(), "config")
		format := f.String("format", "json", "json|text")
		_ = f.Parse(rest)
		cfg, loadedBytes, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		eff := config.Inspect(cfg, loadedBytes)
		if *format == "text" {
			fmt.Printf("fingerprint: %s\n", eff.Fingerprint)
			fmt.Printf("strict_mode: %v\n", cfg.StrictMode)
			fmt.Printf("control.mode: %s\n", cfg.Control.Mode)
			fmt.Printf("bind.api: %s\n", cfg.Bind.API)
			fmt.Printf("integration.enabled: %v\n", cfg.Integration.Enabled)
			if len(eff.Violations) > 0 {
				fmt.Println("risk_violations:")
				for _, v := range eff.Violations {
					fmt.Printf("  - %s: %s (current=%s safe=%s)\n", v.Field, v.Issue, v.Current, v.Safe)
				}
			}
			return
		}
		mustPrint(eff)
	case "diff":
		f := fs("config-diff")
		pathA := f.String("config", configFlagDefault(), "left config file")
		pathB := f.String("against", "", "right config file (required)")
		format := f.String("format", "json", "json|text")
		_ = f.Parse(rest)
		if strings.TrimSpace(*pathB) == "" {
			panic("usage: mel config diff --config <a> --against <b>")
		}
		cfgA, _, err := loadConfigFile(*pathA)
		if err != nil {
			panic(err)
		}
		cfgB, _, err := config.LoadWithOptions(*pathB, config.LoadOptions{})
		if err != nil {
			panic(err)
		}
		if err := config.Validate(cfgA); err != nil {
			panic(fmt.Errorf("left config invalid: %w", err))
		}
		if err := config.Validate(cfgB); err != nil {
			panic(fmt.Errorf("right config invalid: %w", err))
		}
		entries := config.Diff(cfgA, cfgB)
		if *format == "text" {
			fmt.Print(config.FormatDiffText(entries))
			return
		}
		mustPrint(map[string]any{"diff": entries, "left": *pathA, "right": *pathB})
	case "risk":
		f := fs("config-risk")
		path := f.String("config", configFlagDefault(), "config")
		_ = f.Parse(rest)
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		v := config.ValidateSafeDefaults(cfg)
		l := config.LintConfig(cfg)
		mustPrint(map[string]any{"violations": v, "lints": l, "strict_mode": cfg.StrictMode})
	case "keys":
		f := fs("config-keys")
		prefix := f.String("key", "", "filter keys by prefix")
		_ = f.Parse(rest)
		keys := make([]string, 0, len(config.ConfigKeyHelp))
		for k := range config.ConfigKeyHelp {
			keys = append(keys, k)
		}
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				if keys[j] < keys[i] {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}
		p := strings.ToLower(strings.TrimSpace(*prefix))
		var rows [][]string
		for _, k := range keys {
			if p != "" && !strings.HasPrefix(strings.ToLower(k), p) {
				continue
			}
			rows = append(rows, []string{k, config.ConfigKeyHelp[k]})
		}
		if cliGlobal.JSON {
			out := make([]map[string]string, 0, len(rows))
			for _, r := range rows {
				out = append(out, map[string]string{"key": r[0], "help": r[1]})
			}
			mustPrint(map[string]any{"keys": out})
			return
		}
		cliout.Table(os.Stdout, []string{"key", "help"}, rows, cliGlobal.Wide, 72)
	default:
		panic("usage: mel config validate|show|inspect|diff|risk|keys [--key prefix] [--format text|json] --config <path>")
	}
}

func serveCmd(args []string) {
	f := fs("serve")
	path := f.String("config", configFlagDefault(), "config")
	debug := f.Bool("debug", false, "enable debug logging")
	_ = f.Parse(args)
	cfg, _, err := loadConfigFile(*path)
	if err != nil {
		panic(err)
	}
	if err := security.CheckFileMode(*path); err != nil {
		panic(err)
	}
	app, err := service.New(cfg, *debug)
	if err != nil {
		panic(err)
	}
	app.ConfigPath = *path
	app.Web.SetConfigPath(*path)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := app.Start(ctx); err != nil {
		panic(err)
	}
}

// runDoctor performs the same checks as `mel doctor` and returns structured output plus findings.
// Callers may augment the returned map (e.g. preflight adds operator_next_steps).
func runDoctor(cfg config.Config, path string) (map[string]any, []map[string]string) {
	return doctor.Run(cfg, path)
}

func doctorCmd(args []string) {
	cfg, path := loadCfg(args)
	out, findings := runDoctor(cfg, path)
	mustPrint(out)

	fmt.Println()
	fmt.Println("=== Self-Observability ===")
	printLocalHealth()
	fmt.Println()
	printLocalFreshness()
	fmt.Println()
	printLocalSLO()

	if len(findings) > 0 {
		os.Exit(1)
	}
}

func preflightCmd(args []string) {
	f := fs("preflight")
	path := f.String("config", configFlagDefault(), "config")
	skipServe := f.Bool("skip-serve-check", false, "do not probe bind.api for HTTP /healthz (use when mel serve is not expected to be running)")
	_ = f.Parse(args)
	cfg, _, err := loadConfigFile(*path)
	if err != nil {
		panic(err)
	}
	out, findings := runDoctor(cfg, *path)
	serveProbe := probeServeHealth(cfg, *skipServe)
	out["serve_probe"] = serveProbe
	out["preflight_version"] = "v1"
	preflightOK := len(findings) == 0
	if !*skipServe && strings.TrimSpace(cfg.Bind.API) != "" {
		if r, ok := serveProbe["reachable"].(bool); !ok || !r {
			preflightOK = false
		}
	}
	out["preflight_ok"] = preflightOK
	next := preflightNextSteps(cfg, *path, findings, serveProbe)
	out["operator_next_steps"] = next
	mustPrint(out)
	if !cliGlobal.JSON {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Operator next steps:")
		for i, s := range next {
			fmt.Fprintf(os.Stderr, "  %d. %s\n", i+1, s)
		}
	}
	if len(findings) > 0 {
		os.Exit(1)
	}
	if v, ok := serveProbe["reachable"].(bool); ok && !v && !*skipServe {
		os.Exit(1)
	}
}

func probeServeHealth(cfg config.Config, skip bool) map[string]any {
	bind := strings.TrimSpace(cfg.Bind.API)
	out := map[string]any{
		"skipped":     skip,
		"bind_api":    bind,
		"reachable":   nil,
		"http_status": 0,
		"error":       "",
	}
	if skip || bind == "" {
		if bind == "" {
			out["error"] = "bind.api not configured"
		}
		return out
	}
	host, port, err := net.SplitHostPort(bind)
	if err != nil {
		out["reachable"] = false
		out["error"] = fmt.Sprintf("bind.api must be host:port: %v", err)
		return out
	}
	probeHost := host
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		probeHost = "127.0.0.1"
	}
	url := "http://" + net.JoinHostPort(probeHost, port) + "/healthz"
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		out["reachable"] = false
		out["error"] = err.Error()
		return out
	}
	_ = resp.Body.Close()
	out["http_status"] = resp.StatusCode
	out["reachable"] = resp.StatusCode == http.StatusOK
	if resp.StatusCode != http.StatusOK {
		out["error"] = fmt.Sprintf("HTTP %d from %s", resp.StatusCode, url)
	}
	return out
}

func preflightNextSteps(cfg config.Config, path string, findings []map[string]string, serveProbe map[string]any) []string {
	var steps []string
	if len(findings) > 0 {
		steps = append(steps, "Resolve preflight findings (see findings[] in JSON output), starting with critical severity.")
	}
	if skipped, _ := serveProbe["skipped"].(bool); !skipped {
		if reachable, ok := serveProbe["reachable"].(bool); ok && !reachable {
			steps = append(steps, "If you expect the API to be up: start `mel serve --config "+path+"` or free the bind port; if this is a cold host check, pass --skip-serve-check.")
		}
	}
	if len(findings) == 0 {
		if skipped, _ := serveProbe["skipped"].(bool); skipped || serveProbe["reachable"] == true {
			steps = append(steps, "Start or continue with `mel serve --config "+path+"`, then use GET /readyz and `mel status --config "+path+"` for live subsystem truth.")
		}
	}
	enabled := 0
	for _, t := range cfg.Transports {
		if t.Enabled {
			enabled++
		}
	}
	if enabled == 0 {
		steps = append(steps, "No transports are enabled; MEL will idle until you enable serial, TCP, or MQTT in config.")
	}
	steps = append(steps, "Collect triage evidence with `mel support bundle --config "+path+"` before sharing diagnostics externally.")
	return steps
}

func statusCmd(args []string) {
	cfg, path := loadCfg(args)
	d := openDB(cfg)
	snap, err := statuspkg.Collect(cfg, d, nil, nil, path)
	if err != nil {
		panic(err)
	}
	mustPrint(snap)
}

func fleetCmd(args []string) {
	if len(args) == 0 {
		panic("usage: mel fleet truth|evidence ... --config <path>")
	}
	switch args[0] {
	case "truth":
		cfg, _ := loadCfg(args[1:])
		d := openDB(cfg)
		_ = fleet.SyncScopeMetadata(cfg, d)
		summary, err := fleet.BuildTruthSummary(cfg, d)
		if err != nil {
			panic(err)
		}
		mustPrint(summary)
	case "evidence":
		fleetEvidenceCmd(args[1:])
	default:
		panic("usage: mel fleet truth|evidence import|list|show --config <path>")
	}
}

func fleetEvidenceCmd(args []string) {
	if len(args) == 0 {
		panic("usage: mel fleet evidence import|list|show --config <path>")
	}
	switch args[0] {
	case "import":
		f := fs("fleet-evidence-import")
		path := f.String("config", configFlagDefault(), "config")
		file := f.String("file", "", "path to mel_remote_evidence_bundle JSON (required)")
		strict := f.Bool("strict-origin", false, "reject when claimed_origin_instance_id mismatches evidence.origin_instance_id")
		actor := f.String("actor", "cli-operator", "actor id for audit")
		_ = f.Parse(args[1:])
		if strings.TrimSpace(*file) == "" {
			panic("--file is required")
		}
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		raw, err := os.ReadFile(*file)
		if err != nil {
			panic(err)
		}
		app := openServiceApp(cfg)
		out, err := app.ImportRemoteEvidenceBundle(raw, *strict, *actor)
		if err != nil {
			panic(err)
		}
		mustPrint(out)
	case "list":
		f := fs("fleet-evidence-list")
		path := f.String("config", configFlagDefault(), "config")
		limit := f.Int("limit", 50, "max rows")
		_ = f.Parse(args[1:])
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		app := openServiceApp(cfg)
		rows, err := app.ListImportedRemoteEvidence(*limit)
		if err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"imports": rows, "count": len(rows)})
	case "show":
		if len(args) < 2 {
			panic("usage: mel fleet evidence show <import-id> --config <path>")
		}
		id := args[1]
		f := fs("fleet-evidence-show")
		path := f.String("config", configFlagDefault(), "config")
		_ = f.Parse(args[2:])
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		app := openServiceApp(cfg)
		rec, ok, err := app.GetImportedRemoteEvidence(id)
		if err != nil {
			panic(err)
		}
		if !ok {
			panic("import not found: " + id)
		}
		mustPrint(rec)
	default:
		panic("usage: mel fleet evidence import|list|show --config <path>")
	}
}

func panelCmd(args []string) {
	f := fs("panel")
	path := f.String("config", configFlagDefault(), "config")
	format := f.String("format", "text", "text|json")
	_ = f.Parse(args)
	cfg, _, err := loadConfigFile(*path)
	if err != nil {
		panic(err)
	}
	snap, err := statuspkg.Collect(cfg, openDB(cfg), nil, nil, *path)
	if err != nil {
		panic(err)
	}
	panel := statuspkg.BuildPanel(snap)
	if *format == "json" {
		mustPrint(panel)
		return
	}
	printPanelText(panel)
}

func printPanelText(panel statuspkg.Panel) {
	fmt.Printf("MEL PANEL %s [%s]\n", panel.GeneratedAt, strings.ToUpper(panel.OperatorState))
	fmt.Println(panel.Summary)
	fmt.Println()
	for _, metric := range panel.Transports {
		scoreStr := "n/a"
		if metric.Score != nil {
			scoreStr = formatHealthScore(*metric.Score)
		}
		fmt.Printf("[%s] %-16s %s score=%s msgs=%d", metric.Label, metric.Name, metric.State, scoreStr, metric.Messages)
		if metric.LastIngest != "" {
			fmt.Printf(" last=%s", metric.LastIngest)
		}
		fmt.Println()
		if metric.Detail != "" {
			fmt.Printf("    %s\n", metric.Detail)
		}
	}
	if len(panel.Transports) == 0 {
		fmt.Println("[ ] no transports")
	}
	fmt.Println()
	fmt.Printf("Short commands: %s\n", strings.Join(panel.ShortCommands, " | "))
	fmt.Println("8-bit operator menu:")
	for _, item := range panel.OperatorMenu {
		fmt.Printf("  %s %-5s %s\n", item.Key, item.Label, item.Action)
	}
}

func formatHealthScore(score int) string {
	switch {
	case score >= 90:
		return fmt.Sprintf("%d*", score)
	case score >= 70:
		return fmt.Sprintf("%d!", score)
	default:
		return fmt.Sprintf("%d**", score)
	}
}

func nodesCmd(args []string) {
	cfg, _ := loadCfg(args)
	d := openDB(cfg)
	rows, err := d.QueryRows("SELECT n.node_num,n.node_id,n.long_name,n.short_name,n.last_seen,n.last_gateway_id,n.lat_redacted,n.lon_redacted,n.altitude,n.last_snr,n.last_rssi,(SELECT COUNT(*) FROM messages m WHERE m.from_node=n.node_num) AS message_count FROM nodes n ORDER BY updated_at DESC;")
	if err != nil {
		panic(err)
	}
	mustPrint(map[string]any{"nodes": rows})
}

func nodeCmd(args []string) {
	if len(args) < 1 {
		panic("usage: mel node inspect <node-id> | mel node whatif --kind <kind> ... --config <path>")
	}
	if args[0] == "whatif" {
		nodeWhatifCmd(args[1:])
		return
	}
	if len(args) < 2 || args[0] != "inspect" {
		panic("usage: mel node inspect <node-id> --config <path> | mel node whatif --kind <kind> [--node n] --config <path>")
	}
	target := args[1]
	cfg, _ := loadCfg(args[2:])
	d := openDB(cfg)
	rows, err := d.QueryRows(fmt.Sprintf("SELECT n.node_num,n.node_id,n.long_name,n.short_name,n.last_seen,n.last_gateway_id,n.lat_redacted,n.lon_redacted,n.altitude,n.last_snr,n.last_rssi,(SELECT COUNT(*) FROM messages m WHERE m.from_node=n.node_num) AS message_count FROM nodes n WHERE CAST(n.node_num AS TEXT)='%s' OR n.node_id='%s' LIMIT 1;", escape(target), escape(target)))
	if err != nil {
		panic(err)
	}
	if len(rows) == 0 {
		panic(fmt.Errorf("node %s not found in local state", target))
	}
	mustPrint(rows[0])
}

func transportsCmd(args []string) {
	if len(args) == 0 || args[0] != "list" {
		panic("usage: mel transports list --config <path>")
	}
	cfg, path := loadCfg(args[1:])
	snap, err := statuspkg.Collect(cfg, openDB(cfg), nil, nil, path)
	if err != nil {
		panic(err)
	}
	mustPrint(map[string]any{"transports": snap.Transports, "contention_warning": len(doctor.EnabledTransportNames(cfg)) > 1, "selection_rule": "prefer one direct-node transport; hybrid direct+MQTT dedupes only when both paths expose byte-identical mesh packet payloads, so operators must still verify duplicate behavior in their own deployment"})
}

func inspectCmd(args []string) {
	if len(args) == 0 {
		panic("usage: mel inspect transport <name> --config <path> | mel inspect mesh --config <path> | mel inspect topology [--refresh] --config <path>")
	}
	switch args[0] {
	case "transport":
		if len(args) < 2 {
			panic("usage: mel inspect transport <name> --config <path>")
		}
		name := args[1]
		cfg, _ := loadCfg(args[2:])
		d := openDB(cfg)
		drilldown, err := statuspkg.InspectTransport(cfg, d, nil, name, time.Now().UTC())
		if err != nil {
			panic(err)
		}
		mustPrint(drilldown)
	case "mesh":
		cfg, _ := loadCfg(args[1:])
		d := openDB(cfg)
		drilldown, err := statuspkg.InspectMesh(cfg, d, nil, time.Now().UTC())
		if err != nil {
			panic(err)
		}
		mustPrint(drilldown)
	case "topology":
		inspectTopologyCmd(args[1:])
	default:
		panic("usage: mel inspect transport <name> --config <path> | mel inspect mesh --config <path> | mel inspect topology [--refresh] --config <path>")
	}
}

func diagnosticsCmd(args []string) {
	f := fs("diagnostics")
	configPath := f.String("config", configFlagDefault(), "path to config file")
	jsonOutput := f.Bool("json", false, "output as JSON")
	f.Parse(args)

	cfg, _, err := loadConfigFile(*configPath)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	database := openDB(cfg)

	report := diagnostics.RunAllChecks(cfg, database, nil, nil, time.Now().UTC())

	if *jsonOutput {
		mustPrint(report)
	} else {
		fmt.Printf("=== MEL Diagnostics Report ===\n")
		fmt.Printf("Generated: %s\n\n", report.GeneratedAt.Format(time.RFC3339))
		fmt.Printf("Summary:\n")
		fmt.Printf("  Total:   %d\n", report.Summary.TotalCount)
		fmt.Printf("  Critical: %d\n", report.Summary.CriticalCount)
		fmt.Printf("  Warning: %d\n", report.Summary.WarningCount)
		fmt.Printf("  Info:    %d\n\n", report.Summary.InfoCount)

		if len(report.Diagnostics) == 0 {
			fmt.Println("No issues detected.")
			return
		}

		fmt.Println("Diagnostics:")
		for _, d := range report.Diagnostics {
			fmt.Printf("\n[%s] %s\n", strings.ToUpper(d.Severity), d.Title)
			fmt.Printf("  Code: %s\n", d.Code)
			fmt.Printf("  Component: %s\n", d.Component)
			fmt.Printf("  %s\n", d.Explanation)
			if len(d.LikelyCauses) > 0 {
				fmt.Printf("  Likely causes:\n")
				for _, cause := range d.LikelyCauses {
					fmt.Printf("    - %s\n", cause)
				}
			}
			if len(d.RecommendedSteps) > 0 {
				fmt.Printf("  Recommended steps:\n")
				for _, step := range d.RecommendedSteps {
					fmt.Printf("    - %s\n", step)
				}
			}
			fmt.Printf("  Auto-recover: %v | Operator action: %v\n", d.CanAutoRecover, d.OperatorActionRequired)
		}
	}
}

func replayCmd(args []string) {
	if len(args) == 0 {
		replayMessageQueryCmd(nil)
		return
	}
	switch args[0] {
	case "run":
		replayMessageQueryCmd(args[1:])
	case "diff":
		replayDiffCmd(args[1:])
	case "kernel":
		kernelReplayCmd(args[1:])
	default:
		replayMessageQueryCmd(args)
	}
}

func replayDiffCmd(args []string) {
	f := fs("replay-diff")
	cfgPath := f.String("config", configFlagDefault(), "left config file")
	againstPath := f.String("against", "", "right config file (required)")
	urlFlag := f.String("url", "", "daemon URL (overrides config bind.api)")
	mode := f.String("mode", "dry_run", "replay mode passed to kernel API")
	since := f.String("since", "", "RFC3339")
	until := f.String("until", "", "RFC3339")
	maxEvents := f.Int("max-events", 0, "max events (0 = default)")
	policyMode := f.String("policy-mode", "", "override policy mode")
	policyVersion := f.String("policy-version", "v1", "policy version")
	compact := f.Bool("compact", true, "omit effects in comparison payload")
	_ = f.Parse(args)
	if strings.TrimSpace(*againstPath) == "" {
		panic("usage: mel replay diff --config <a> --against <b> [kernel replay flags]")
	}
	cfgL, err := loadConfigSide(*cfgPath, cliGlobal.Profile)
	if err != nil {
		panic(err)
	}
	cfgR, err := loadConfigSide(*againstPath, "")
	if err != nil {
		panic(err)
	}
	baseL := daemonURL(cfgL)
	if *urlFlag != "" {
		baseL = *urlFlag
	}
	baseR := daemonURL(cfgR)
	if *urlFlag != "" {
		baseR = *urlFlag
	}
	reqL := buildKernelReplayRequest(cfgL, *mode, *since, *until, *maxEvents, *policyMode, *policyVersion)
	reqR := buildKernelReplayRequest(cfgR, *mode, *since, *until, *maxEvents, *policyMode, *policyVersion)
	left, err := apiPost(baseL, "/api/v1/kernel/replay", reqL)
	if err != nil {
		panic(fmt.Errorf("left replay: %w", err))
	}
	right, err := apiPost(baseR, "/api/v1/kernel/replay", reqR)
	if err != nil {
		panic(fmt.Errorf("right replay: %w", err))
	}
	if *compact {
		stripReplayEffects(left)
		stripReplayEffects(right)
	}
	ml, _ := json.Marshal(left)
	mr, _ := json.Marshal(right)
	mustPrint(map[string]any{
		"mode":              "kernel_replay_diff",
		"left_config":       *cfgPath,
		"right_config":      *againstPath,
		"left_url":          baseL,
		"right_url":         baseR,
		"identical":         string(ml) == string(mr),
		"left_result":       left,
		"right_result":      right,
		"replay_parameters": reqL,
	})
}

func buildKernelReplayRequest(cfg config.Config, mode, since, until string, maxEvents int, policyMode, policyVersion string) map[string]any {
	req := map[string]any{
		"mode": mode,
		"policy": map[string]any{
			"version":                policyVersion,
			"mode":                   policyMode,
			"require_min_confidence": cfg.Control.RequireMinConfidence,
			"allowed_actions":        cfg.Control.AllowedActions,
		},
	}
	if since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			req["since"] = t.Format(time.RFC3339)
		}
	}
	if until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil {
			req["until"] = t.Format(time.RFC3339)
		}
	}
	if maxEvents > 0 {
		req["max_events"] = maxEvents
	}
	return req
}

func stripReplayEffects(v any) {
	m, ok := v.(map[string]any)
	if !ok {
		return
	}
	delete(m, "effects")
}

func replayMessageQueryCmd(args []string) {
	f := fs("replay-run")
	path := f.String("config", configFlagDefault(), "config")
	node := f.String("node", "", "filter by node number")
	messageType := f.String("type", "", "filter by message type")
	limit := f.Int("limit", 50, "maximum rows")
	since := f.String("since", "", "filter rx_time >= RFC3339")
	filter := f.String("filter", "", "substring match on payload_text|transport")
	_ = f.Parse(args)
	cfg, _, err := loadConfigFile(*path)
	if err != nil {
		panic(err)
	}
	d := openDB(cfg)
	clauses := []string{"1=1"}
	if *node != "" {
		clauses = append(clauses, fmt.Sprintf("(CAST(from_node AS TEXT)='%s' OR CAST(to_node AS TEXT)='%s')", escape(*node), escape(*node)))
	}
	if *messageType != "" {
		clauses = append(clauses, fmt.Sprintf("payload_json LIKE '%%%s%%'", escape(fmt.Sprintf(`\"message_type\":\"%s\"`, *messageType))))
	}
	if strings.TrimSpace(*since) != "" {
		clauses = append(clauses, fmt.Sprintf("rx_time >= '%s'", escape(*since)))
	}
	if strings.TrimSpace(*filter) != "" {
		sub := escape(*filter)
		clauses = append(clauses, fmt.Sprintf("(payload_text LIKE '%%%s%%' OR transport_name LIKE '%%%s%%')", sub, sub))
	}
	rows, err := d.QueryRows(fmt.Sprintf("SELECT transport_name,packet_id,from_node,to_node,portnum,payload_text,payload_json,rx_time,created_at FROM messages WHERE %s ORDER BY id DESC LIMIT %d;", strings.Join(clauses, " AND "), *limit))
	if err != nil {
		panic(err)
	}
	mustPrint(map[string]any{
		"mode":     "local_message_replay_source",
		"messages": rows,
		"filters":  map[string]any{"node": *node, "type": *messageType, "limit": *limit, "since": *since, "filter": *filter},
	})
}

func dbCmd(args []string) {
	if len(args) == 0 || args[0] != "vacuum" {
		panic("usage: mel db vacuum --config <path>")
	}
	cfg, _ := loadCfg(args[1:])
	d := openDB(cfg)
	if err := d.Vacuum(); err != nil {
		panic(err)
	}
	mustPrint(map[string]any{"status": "vacuum complete"})
}

func exportCmd(args []string) {
	f := fs("export")
	path := f.String("config", configFlagDefault(), "config")
	outPath := f.String("out", "", "write export bundle to file instead of stdout")
	_ = f.Parse(args)
	cfg, _, err := loadConfigFile(*path)
	if err != nil {
		panic(err)
	}
	d := openDB(cfg)
	nodes, err := d.QueryRows("SELECT node_num,node_id,long_name,short_name,last_seen,lat_redacted,lon_redacted,altitude FROM nodes ORDER BY node_num;")
	if err != nil {
		panic(err)
	}
	messages, err := d.QueryRows("SELECT transport_name,packet_id,channel_id,gateway_id,from_node,to_node,portnum,payload_text,payload_json,rx_time FROM messages ORDER BY id DESC LIMIT 250;")
	if err != nil {
		panic(err)
	}
	deadLetters, err := d.QueryRows("SELECT transport_name,transport_type,topic,reason,payload_hex,details_json,created_at FROM dead_letters ORDER BY id DESC LIMIT 250;")
	if err != nil {
		panic(err)
	}
	auditLogs, err := d.QueryRows("SELECT category,level,message,details_json,created_at FROM audit_logs ORDER BY id DESC LIMIT 250;")
	if err != nil {
		panic(err)
	}
	bundle := map[string]any{"exported_at": time.Now().UTC().Format(time.RFC3339), "redacted": cfg.Privacy.RedactExports, "nodes": nodes, "messages": messages, "dead_letters": deadLetters, "audit_logs": auditLogs}
	if cfg.Privacy.RedactExports {
		bundle["messages"] = privacy.RedactMessages(messages)
	}
	writeOutput(bundle, *outPath)
}

func importCmd(args []string) {
	if len(args) == 0 || args[0] != "validate" {
		panic("usage: mel import validate --bundle <path>")
	}
	f := fs("import-validate")
	bundlePath := f.String("bundle", "", "bundle path")
	_ = f.Parse(args[1:])
	if *bundlePath == "" {
		panic("--bundle is required")
	}
	b, err := os.ReadFile(*bundlePath)
	if err != nil {
		panic(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		panic(err)
	}
	_, hasNodes := payload["nodes"]
	mustPrint(map[string]any{"valid": hasNodes, "keys": sortedKeys(payload)})
	if !hasNodes {
		os.Exit(1)
	}
}

func logsCmd(args []string) {
	if len(args) == 0 || args[0] != "tail" {
		panic("usage: mel logs tail --config <path> [--limit n] [--category s] [--since RFC3339] [--filter substring]")
	}
	f := fs("logs-tail")
	path := f.String("config", configFlagDefault(), "config")
	limit := f.Int("limit", 50, "max rows")
	category := f.String("category", "", "filter category prefix")
	since := f.String("since", "", "created_at >= RFC3339")
	filter := f.String("filter", "", "substring match on category|level|message")
	_ = f.Parse(args[1:])
	cfg, _, err := loadConfigFile(*path)
	if err != nil {
		panic(err)
	}
	d := openDB(cfg)
	clauses := []string{"1=1"}
	if strings.TrimSpace(*category) != "" {
		clauses = append(clauses, fmt.Sprintf("category LIKE '%s%%'", escape(*category)))
	}
	if strings.TrimSpace(*since) != "" {
		clauses = append(clauses, fmt.Sprintf("created_at >= '%s'", escape(*since)))
	}
	if strings.TrimSpace(*filter) != "" {
		sub := escape(*filter)
		clauses = append(clauses, fmt.Sprintf("(category LIKE '%%%s%%' OR level LIKE '%%%s%%' OR message LIKE '%%%s%%')", sub, sub, sub))
	}
	q := fmt.Sprintf("SELECT category,level,message,created_at FROM audit_logs WHERE %s ORDER BY id DESC LIMIT %d;", strings.Join(clauses, " AND "), *limit)
	rows, err := d.QueryRows(q)
	if err != nil {
		panic(err)
	}
	mustPrint(map[string]any{"logs": rows, "count": len(rows), "filters": map[string]any{"category": *category, "since": *since, "filter": *filter, "limit": *limit}})
}

func controlCmd(args []string) {
	if len(args) == 0 {
		panic("usage: mel control status|history --config <path>")
	}
	switch args[0] {
	case "status":
		cfg, _ := loadCfg(args[1:])
		d := openDB(cfg)
		eval, err := control.Evaluate(cfg, d, nil, time.Now().UTC())
		if err != nil {
			panic(err)
		}
		mustPrint(eval.Explanation)
	case "history":
		f := fs("control-history")
		path := f.String("config", configFlagDefault(), "config")
		transportName := f.String("transport", "", "filter by transport")
		start := f.String("start", "", "start time RFC3339")
		end := f.String("end", "", "end time RFC3339")
		limit := f.Int("limit", 50, "max rows")
		offset := f.Int("offset", 0, "offset")
		_ = f.Parse(args[1:])
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		d := openDB(cfg)
		actions, err := d.ControlActions(*transportName, "", *start, *end, "", *limit, *offset)
		if err != nil {
			panic(err)
		}
		decisions, err := d.ControlDecisions(*transportName, "", *start, *end, *limit, *offset)
		if err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"actions": actions, "decisions": decisions, "transport": *transportName, "start": *start, "end": *end, "pagination": map[string]any{"limit": *limit, "offset": *offset}})
	case "pending":
		cfg, _ := loadCfg(args[1:])
		d := openDB(cfg)
		pending, err := d.PendingApprovalActions(100)
		if err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"pending_approval": pending, "count": len(pending)})
	case "approve":
		if len(args) < 2 {
			panic("usage: mel control approve <action-id> --config <path> [--note '...'] --i-understand-break-glass-sod")
		}
		actionID := args[1]
		f := fs("control-approve")
		path := f.String("config", configFlagDefault(), "config")
		note := f.String("note", "", "approval note")
		actor := f.String("actor", "cli-operator", "operator identity")
		breakGlassAck := f.Bool("i-understand-break-glass-sod", false, "required: acknowledge emergency use of legacy entrypoint (canonical: mel action approve)")
		_ = f.Parse(args[2:])
		if !*breakGlassAck {
			fmt.Fprintln(os.Stderr, "mel control approve: BLOCKED — legacy entrypoint.")
			fmt.Fprintln(os.Stderr, "Canonical path runs the full service approve path (audit, timeline, executor queue): mel action approve <action-id> --config <path>")
			fmt.Fprintln(os.Stderr, "mel control approve exists only as break-glass and still records durable metadata marking this entrypoint.")
			fmt.Fprintln(os.Stderr, "To proceed: pass --i-understand-break-glass-sod")
			os.Exit(2)
		}
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		app := openServiceApp(cfg)
		if _, err := app.ApproveActionWithOpts(actionID, *actor, *note, service.ApprovalOpts{BreakGlassLegacyCLI: true}); err != nil {
			panic(err)
		}
		did := app.ProcessNextQueuedControlAction(context.Background())
		fmt.Fprintln(os.Stderr, "WARNING: break-glass legacy entrypoint used (mel control approve). Canonical operator path: mel action approve. Executor queue processed one slot if pending.")
		mustPrint(map[string]any{
			"status":            "approved",
			"action_id":         actionID,
			"actor":             *actor,
			"break_glass":       true,
			"legacy_entrypoint": "mel_control_approve",
			"preferred_path":    "mel action approve",
			"processed_queue":   did,
		})
	case "reject":
		if len(args) < 2 {
			panic("usage: mel control reject <action-id> --config <path> [--note '...'] --i-understand-break-glass-sod")
		}
		actionID := args[1]
		f := fs("control-reject")
		path := f.String("config", configFlagDefault(), "config")
		note := f.String("note", "", "rejection reason")
		actor := f.String("actor", "cli-operator", "operator identity")
		breakGlassAck := f.Bool("i-understand-break-glass-sod", false, "required: acknowledge emergency use of legacy entrypoint (canonical: mel action reject)")
		_ = f.Parse(args[2:])
		if !*breakGlassAck {
			fmt.Fprintln(os.Stderr, "mel control reject: BLOCKED — legacy entrypoint.")
			fmt.Fprintln(os.Stderr, "Canonical path: mel action reject <action-id> --config <path>")
			fmt.Fprintln(os.Stderr, "To proceed: pass --i-understand-break-glass-sod")
			os.Exit(2)
		}
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		app := openServiceApp(cfg)
		if _, err := app.RejectActionWithOpts(actionID, *actor, *note, service.ApprovalOpts{BreakGlassLegacyCLI: true}); err != nil {
			panic(err)
		}
		fmt.Fprintln(os.Stderr, "WARNING: break-glass legacy entrypoint used (mel control reject). Canonical operator path: mel action reject.")
		mustPrint(map[string]any{
			"status":            "rejected",
			"action_id":         actionID,
			"actor":             *actor,
			"break_glass":       true,
			"legacy_entrypoint": "mel_control_reject",
			"preferred_path":    "mel action reject",
		})
	case "inspect":
		if len(args) < 2 {
			panic("usage: mel control inspect <action-id> --config <path>")
		}
		actionID := args[1]
		cfg, _ := loadCfg(args[2:])
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
		out := map[string]any{
			"action":          action,
			"evidence_bundle": nil,
			"notes":           notes,
		}
		if bundleOK {
			out["evidence_bundle"] = bundle
		}
		mustPrint(out)
	case "operational-state":
		cfg, _ := loadCfg(args[1:])
		d := openDB(cfg)
		state, err := d.ControlPlaneStateSnapshot(time.Now().UTC())
		if err != nil {
			panic(err)
		}
		mustPrint(state)
	default:
		panic("usage: mel control status|history|pending|approve|reject|inspect|operational-state --config <path>")
	}
}

func openServiceApp(cfg config.Config) *service.App {
	app, err := service.New(cfg, false)
	if err != nil {
		panic(err)
	}
	return app
}

func enrichControlActionsCLI(rows []db.ControlActionRecord) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		m, err := json.Marshal(row)
		if err != nil {
			continue
		}
		var base map[string]any
		if err := json.Unmarshal(m, &base); err != nil {
			continue
		}
		ov := operatorlang.ActionOperatorLabels(row)
		base["operator_view"] = ov
		out = append(out, base)
	}
	return out
}

func actionCmd(args []string) {
	if len(args) == 0 {
		panic("usage: mel action list|pending|inspect|queue|approve|reject --config <path>")
	}
	switch args[0] {
	case "list":
		f := fs("action-list")
		path := f.String("config", configFlagDefault(), "config")
		transportName := f.String("transport", "", "filter by transport")
		lifecycle := f.String("lifecycle-state", "", "filter by lifecycle_state (e.g. pending_approval, pending, running, completed)")
		start := f.String("start", "", "start time RFC3339")
		end := f.String("end", "", "end time RFC3339")
		limit := f.Int("limit", 50, "max rows")
		offset := f.Int("offset", 0, "offset")
		_ = f.Parse(args[1:])
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		d := openDB(cfg)
		actions, err := d.ControlActions(*transportName, "", *start, *end, *lifecycle, *limit, *offset)
		if err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"actions": enrichControlActionsCLI(actions), "transport": *transportName, "start": *start, "end": *end, "pagination": map[string]any{"limit": *limit, "offset": *offset}})
	case "pending":
		cfg, _ := loadCfg(args[1:])
		d := openDB(cfg)
		pending, err := d.PendingApprovalActions(100)
		if err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"pending_approval": pending, "count": len(pending)})
	case "inspect":
		if len(args) < 2 {
			panic("usage: mel action inspect <action-id> --config <path>")
		}
		actionID := args[1]
		cfg, _ := loadCfg(args[2:])
		app := openServiceApp(cfg)
		out, err := app.InspectAction(actionID)
		if err != nil {
			panic(err)
		}
		mustPrint(out)
	case "queue":
		f := fs("action-queue")
		path := f.String("config", configFlagDefault(), "config")
		actionType := f.String("type", "", "control action type (required)")
		transport := f.String("transport", "", "target transport name")
		segment := f.String("segment", "", "target segment")
		node := f.String("node", "", "target node")
		reason := f.String("reason", "", "rationale (required)")
		incident := f.String("incident", "", "incident id to link (optional)")
		conf := f.Float64("confidence", 0.9, "confidence 0..1")
		actor := f.String("actor", "cli-operator", "operator identity (recorded as submitter)")
		_ = f.Parse(args[1:])
		if strings.TrimSpace(*actionType) == "" || strings.TrimSpace(*reason) == "" {
			panic("usage: mel action queue --type <action_type> --reason \"...\" --config <path> [--transport name] [--incident id] [--actor id]")
		}
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		app := openServiceApp(cfg)
		id, err := app.QueueOperatorControlAction(*actor, *actionType, *transport, *segment, *node, *reason, *conf, *incident)
		if err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"status": "queued", "action_id": id})
	case "approve":
		if len(args) < 2 {
			panic("usage: mel action approve <action-id> --config <path> [--note '...'] [--actor id]")
		}
		actionID := args[1]
		f := fs("action-approve")
		path := f.String("config", configFlagDefault(), "config")
		note := f.String("note", "", "approval note")
		actor := f.String("actor", "cli-operator", "operator identity (recorded in audit trail)")
		breakGlassSod := f.Bool("break-glass-sod-ack", false, "acknowledge same-actor approval when separation-of-duties would otherwise block (requires --break-glass-sod-reason)")
		breakGlassReason := f.String("break-glass-sod-reason", "", "required with --break-glass-sod-ack: auditable reason for SoD bypass")
		_ = f.Parse(args[2:])
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		app := openServiceApp(cfg)
		resp, err := app.ApproveAction(actionID, *actor, *note, *breakGlassSod, *breakGlassReason)
		if err != nil {
			panic(err)
		}
		didExec := app.ProcessNextQueuedControlAction(context.Background())
		line := "APPROVED"
		if didExec {
			line = "APPROVED; ONE QUEUED ACTION EXECUTED (OR ATTEMPTED); BACKLOG MAY REMAIN — use mel serve for continuous draining"
		} else if resp != nil && resp.QueuedForExecution {
			line = "APPROVED; QUEUED FOR EXECUTOR (mel serve or CLI one-shot dequeue)"
		} else {
			line = "APPROVED; QUEUE SATURATED — APPROVAL RECORDED; EXECUTION NOT QUEUED (FREE A SLOT OR RESTART)"
		}
		fmt.Fprintln(os.Stderr, line)
		fmt.Fprintln(os.Stderr, "Note: MEL uses single-approver approval (required_approvals=1). HTTP approve does not drain the full backlog.")
		out := map[string]any{
			"status":                                "approved",
			"action_id":                             actionID,
			"actor":                                 *actor,
			"one_shot_executor_dequeue_ran":         didExec,
			"approval_does_not_imply_execution":     true,
			"continuous_backlog_requires_mel_serve": true,
		}
		if resp != nil {
			out["queued_for_execution"] = resp.QueuedForExecution
			out["policy"] = resp.Policy
		}
		mustPrint(out)
	case "reject":
		if len(args) < 2 {
			panic("usage: mel action reject <action-id> --config <path> [--note '...'] [--actor id]")
		}
		actionID := args[1]
		f := fs("action-reject")
		path := f.String("config", configFlagDefault(), "config")
		note := f.String("note", "", "rejection reason")
		actor := f.String("actor", "cli-operator", "operator identity (recorded in audit trail)")
		_ = f.Parse(args[2:])
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		app := openServiceApp(cfg)
		if err := app.RejectAction(actionID, *actor, *note); err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"status": "rejected", "action_id": actionID, "actor": *actor})
	default:
		panic("usage: mel action list|pending|inspect|queue|approve|reject --config <path>")
	}
}

func incidentCmd(args []string) {
	if len(args) == 0 {
		panic("usage: mel incident inspect <id>|handoff <id> --config <path> ...")
	}
	switch args[0] {
	case "inspect":
		if len(args) < 2 {
			panic("usage: mel incident inspect <id> --config <path>")
		}
		id := args[1]
		cfg, _ := loadCfg(args[2:])
		app := openServiceApp(cfg)
		inc, ok, err := app.IncidentByID(id)
		if err != nil {
			panic(err)
		}
		if !ok {
			panic("incident not found: " + id)
		}
		mustPrint(inc)
	case "handoff":
		if len(args) < 2 {
			panic("usage: mel incident handoff <id> --to <operator> --summary \"...\" --config <path>")
		}
		id := args[1]
		f := fs("incident-handoff")
		path := f.String("config", configFlagDefault(), "config")
		to := f.String("to", "", "assignee operator id (required)")
		summary := f.String("summary", "", "handoff summary for the next operator (required)")
		from := f.String("from", "cli-operator", "handing-off operator identity")
		pending := f.String("pending-actions", "", "comma-separated control action ids")
		recent := f.String("recent-actions", "", "comma-separated control action ids")
		risks := f.String("risks", "", "comma-separated risk notes")
		_ = f.Parse(args[2:])
		if strings.TrimSpace(*to) == "" || strings.TrimSpace(*summary) == "" {
			panic("--to and --summary are required")
		}
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		app := openServiceApp(cfg)
		req := models.IncidentHandoffRequest{
			ToOperatorID:   strings.TrimSpace(*to),
			HandoffSummary: strings.TrimSpace(*summary),
		}
		if strings.TrimSpace(*pending) != "" {
			for _, p := range strings.Split(*pending, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					req.PendingActions = append(req.PendingActions, p)
				}
			}
		}
		if strings.TrimSpace(*recent) != "" {
			for _, p := range strings.Split(*recent, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					req.RecentActions = append(req.RecentActions, p)
				}
			}
		}
		if strings.TrimSpace(*risks) != "" {
			for _, p := range strings.Split(*risks, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					req.Risks = append(req.Risks, p)
				}
			}
		}
		if err := app.IncidentHandoff(id, strings.TrimSpace(*from), req); err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"status": "handed_off", "incident_id": id, "to": *to, "from": *from})
	default:
		panic("usage: mel incident inspect <id>|handoff <id> --config <path> ...")
	}
}

func timelineCmd(args []string) {
	f := fs("timeline")
	path := f.String("config", configFlagDefault(), "config")
	start := f.String("start", "", "start time RFC3339")
	end := f.String("end", "", "end time RFC3339")
	limit := f.Int("limit", 100, "max events")
	_ = f.Parse(args)
	cfg, _, err := loadConfigFile(*path)
	if err != nil {
		panic(err)
	}
	d := openDB(cfg)
	events, err := d.TimelineEvents(*start, *end, *limit)
	if err != nil {
		panic(err)
	}
	mustPrint(map[string]any{"events": events, "count": len(events), "start": *start, "end": *end})
}

func freezeCmd(args []string) {
	if len(args) == 0 {
		panic("usage: mel freeze create|list|clear --config <path>")
	}
	switch args[0] {
	case "list":
		cfg, _ := loadCfg(args[1:])
		d := openDB(cfg)
		freezes, err := d.ActiveFreezes()
		if err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"freezes": freezes, "count": len(freezes)})
	case "create":
		f := fs("freeze-create")
		path := f.String("config", configFlagDefault(), "config")
		reason := f.String("reason", "", "reason for freeze (required)")
		scopeType := f.String("scope-type", "global", "global|transport|action_type")
		scopeValue := f.String("scope-value", "", "transport name or action type (if scoped)")
		expiresAt := f.String("expires-at", "", "optional expiry RFC3339")
		actor := f.String("actor", "cli-operator", "operator identity")
		_ = f.Parse(args[1:])
		if *reason == "" {
			panic("--reason is required")
		}
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		d := openDB(cfg)
		id := fmt.Sprintf("frz-%d", time.Now().UnixNano())
		if err := d.CreateFreeze(db.FreezeRecord{
			ID: id, ScopeType: *scopeType, ScopeValue: *scopeValue,
			Reason: *reason, CreatedBy: *actor, ExpiresAt: *expiresAt,
		}); err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"status": "created", "freeze_id": id, "scope_type": *scopeType, "scope_value": *scopeValue, "reason": *reason})
	case "clear":
		if len(args) < 2 {
			panic("usage: mel freeze clear <freeze-id> --config <path>")
		}
		freezeID := args[1]
		f := fs("freeze-clear")
		path := f.String("config", configFlagDefault(), "config")
		actor := f.String("actor", "cli-operator", "operator identity")
		_ = f.Parse(args[2:])
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		d := openDB(cfg)
		if err := d.ClearFreeze(freezeID, *actor); err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"status": "cleared", "freeze_id": freezeID, "actor": *actor})
	default:
		panic("usage: mel freeze create|list|clear --config <path>")
	}
}

func maintenanceCmd(args []string) {
	if len(args) == 0 {
		panic("usage: mel maintenance create|list|cancel --config <path>")
	}
	switch args[0] {
	case "list":
		cfg, _ := loadCfg(args[1:])
		d := openDB(cfg)
		windows, err := d.AllMaintenanceWindows(50)
		if err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"maintenance_windows": windows, "count": len(windows)})
	case "create":
		f := fs("maintenance-create")
		path := f.String("config", configFlagDefault(), "config")
		title := f.String("title", "Scheduled Maintenance", "window title")
		reason := f.String("reason", "", "reason for maintenance")
		scopeType := f.String("scope-type", "global", "global|transport|action_type")
		scopeValue := f.String("scope-value", "", "transport name or action type (if scoped)")
		startsAt := f.String("starts-at", "", "start time RFC3339 (required)")
		endsAt := f.String("ends-at", "", "end time RFC3339 (required)")
		actor := f.String("actor", "cli-operator", "operator identity")
		_ = f.Parse(args[1:])
		if *startsAt == "" || *endsAt == "" {
			panic("--starts-at and --ends-at are required")
		}
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		d := openDB(cfg)
		id := fmt.Sprintf("mw-%d", time.Now().UnixNano())
		if err := d.CreateMaintenanceWindow(db.MaintenanceWindowRecord{
			ID: id, Title: *title, Reason: *reason,
			ScopeType: *scopeType, ScopeValue: *scopeValue,
			StartsAt: *startsAt, EndsAt: *endsAt, CreatedBy: *actor,
		}); err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"status": "created", "window_id": id, "title": *title, "starts_at": *startsAt, "ends_at": *endsAt})
	case "cancel":
		if len(args) < 2 {
			panic("usage: mel maintenance cancel <window-id> --config <path>")
		}
		windowID := args[1]
		f := fs("maintenance-cancel")
		path := f.String("config", configFlagDefault(), "config")
		actor := f.String("actor", "cli-operator", "operator identity")
		_ = f.Parse(args[2:])
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		d := openDB(cfg)
		if err := d.CancelMaintenanceWindow(windowID, *actor); err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"status": "cancelled", "window_id": windowID, "actor": *actor})
	default:
		panic("usage: mel maintenance create|list|cancel --config <path>")
	}
}

func notesCmd(args []string) {
	if len(args) == 0 {
		panic("usage: mel notes add|list --config <path>")
	}
	switch args[0] {
	case "add":
		f := fs("notes-add")
		path := f.String("config", configFlagDefault(), "config")
		refType := f.String("ref-type", "", "resource type (action|incident|transport) (required)")
		refID := f.String("ref-id", "", "resource ID (required)")
		content := f.String("content", "", "note content (required)")
		actor := f.String("actor", "cli-operator", "operator identity")
		_ = f.Parse(args[1:])
		if *refType == "" || *refID == "" || *content == "" {
			panic("--ref-type, --ref-id, and --content are required")
		}
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		d := openDB(cfg)
		id := fmt.Sprintf("note-%d", time.Now().UnixNano())
		if err := d.CreateOperatorNote(db.OperatorNoteRecord{
			ID: id, RefType: *refType, RefID: *refID,
			ActorID: *actor, Content: *content,
		}); err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"status": "created", "note_id": id, "ref_type": *refType, "ref_id": *refID})
	case "list":
		f := fs("notes-list")
		path := f.String("config", configFlagDefault(), "config")
		refType := f.String("ref-type", "", "resource type (required)")
		refID := f.String("ref-id", "", "resource ID (required)")
		limit := f.Int("limit", 50, "max notes")
		_ = f.Parse(args[1:])
		if *refType == "" || *refID == "" {
			panic("--ref-type and --ref-id are required")
		}
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		d := openDB(cfg)
		notes, err := d.OperatorNotesByRef(*refType, *refID, *limit)
		if err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"notes": notes, "count": len(notes), "ref_type": *refType, "ref_id": *refID})
	default:
		panic("usage: mel notes add|list --config <path>")
	}
}

func policyCmd(args []string) {
	if len(args) == 0 || args[0] != "explain" {
		panic("usage: mel policy explain --config <path>")
	}
	cfg, _ := loadCfg(args[1:])
	mustPrint(policy.Explain(cfg))
}

func privacyCmd(args []string) {
	if len(args) == 0 || args[0] != "audit" {
		panic("usage: mel privacy audit [--format json|text] --config <path>")
	}
	f := fs("privacy-audit")
	path := f.String("config", configFlagDefault(), "config")
	format := f.String("format", "json", "json|text")
	_ = f.Parse(args[1:])
	cfg, _, err := loadConfigFile(*path)
	if err != nil {
		panic(err)
	}
	findings := privacy.Audit(cfg)
	if *format == "text" {
		printPrivacyText(findings)
		return
	}
	mustPrint(map[string]any{"summary": privacy.Summary(findings), "findings": findings})
}

func backupCmd(args []string) {
	if len(args) == 0 {
		panic("usage: mel backup create|restore")
	}
	switch args[0] {
	case "create":
		f := fs("backup-create")
		path := f.String("config", configFlagDefault(), "config")
		outPath := f.String("out", "", "bundle output path")
		_ = f.Parse(args[1:])
		cfg, _, err := loadConfigFile(*path)
		if err != nil {
			panic(err)
		}
		manifest, err := backup.Create(cfg, *path, *outPath)
		if err != nil {
			panic(err)
		}
		mustPrint(manifest)
	case "restore":
		f := fs("backup-restore")
		bundlePath := f.String("bundle", "", "bundle path (required)")
		dryRun := f.Bool("dry-run", false, "validate only (required - restore without --dry-run is not implemented)")
		destination := f.String("destination", ".", "restore directory")
		_ = f.Parse(args[1:])
		if *bundlePath == "" {
			panic("--bundle is required")
		}
		if !*dryRun {
			panic("--dry-run is required; restore without --dry-run is not implemented in this release candidate")
		}
		report, err := backup.ValidateRestore(*bundlePath, *destination)
		if err != nil {
			panic(err)
		}
		mustPrint(report)
		if !report.Valid {
			os.Exit(1)
		}
	default:
		panic("usage: mel backup create|restore")
	}
}

func openDB(cfg config.Config) *db.DB {
	d, err := db.Open(cfg)
	if err != nil {
		panic(err)
	}
	return d
}

func writeOutput(v any, outPath string) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	b = append(b, '\n')
	if outPath == "" {
		fmt.Print(string(b))
		return
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(outPath, b, 0o600); err != nil {
		panic(err)
	}
	fmt.Println(outPath)
}

func printPrivacyText(findings []privacy.Finding) {
	fmt.Printf("Privacy audit summary: %+v\n", privacy.Summary(findings))
	if len(findings) == 0 {
		fmt.Println("No findings.")
		return
	}
	for _, finding := range findings {
		fmt.Printf("- [%s] %s\n  remediation: %s\n", strings.ToUpper(finding.Severity), finding.Message, finding.Remediation)
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	if len(keys) > 1 {
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				if keys[j] < keys[i] {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}
	}
	return keys
}

func escape(v string) string { return db.EscString(v) }

func simulateCmd(args []string) {
	f := fs("sim")
	endpoint := f.String("endpoint", "127.0.0.1:18830", "endpoint")
	topic := f.String("topic", "msh/US/2/e/test", "topic")
	_ = f.Parse(args)
	env := sampleEnvelope()
	runMQTTServer(*endpoint, *topic, env)
}
func sampleEnvelope() []byte {
	user := msg(
		fieldBytes(1, []byte("!abcd1234")),
		fieldBytes(2, []byte("Relay Node")),
		fieldBytes(3, []byte("RN")),
	)
	data := msg(fieldVarint(1, 4), fieldBytes(2, user))
	packet := msg(fieldFixed32(1, 12345), fieldFixed32(2, 255), fieldMsg(4, data), fieldFixed32(6, 99), fieldFixed32(7, uint32(time.Now().Unix())), fieldVarint(9, 3), fieldVarint(12, 42))
	env := msg(fieldMsg(1, packet), fieldBytes(2, []byte("mel-test")), fieldBytes(3, []byte("!gateway")))
	return env
}
func msg(parts ...[]byte) []byte { return bytes.Join(parts, nil) }
func tag(field int, wt int) []byte {
	b := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(b, uint64(field<<3|wt))
	return b[:n]
}
func fieldVarint(field int, v uint64) []byte {
	b := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(b, v)
	return append(tag(field, 0), b[:n]...)
}
func fieldFixed32(field int, v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return append(tag(field, 5), b...)
}
func fieldBytes(field int, v []byte) []byte {
	ln := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(ln, uint64(len(v)))
	out := append(tag(field, 2), ln[:n]...)
	return append(out, v...)
}
func fieldMsg(field int, v []byte) []byte { return fieldBytes(field, v) }
func runMQTTServer(endpoint, topic string, payload []byte) {
	ln, err := net.Listen("tcp", endpoint)
	if err != nil {
		panic(err)
	}
	defer ln.Close()
	conn, err := ln.Accept()
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	hdr := make([]byte, 2)
	_, _ = conn.Read(hdr)
	rem := make([]byte, 1024)
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	_, _ = conn.Read(rem)
	_, _ = conn.Write([]byte{0x20, 0x02, 0x00, 0x00})
	_, _ = conn.Read(rem)
	topicBuf := bytes.NewBuffer(nil)
	_ = binary.Write(topicBuf, binary.BigEndian, uint16(len(topic)))
	topicBuf.WriteString(topic)
	body := append(topicBuf.Bytes(), payload...)
	pkt := bytes.NewBuffer([]byte{0x30})
	writeRemaining(pkt, len(body))
	pkt.Write(body)
	_, _ = conn.Write(pkt.Bytes())
	select {}
}
func writeRemaining(buf *bytes.Buffer, n int) {
	for {
		d := byte(n % 128)
		n /= 128
		if n > 0 {
			d |= 128
		}
		buf.WriteByte(d)
		if n == 0 {
			break
		}
	}
}
func uiCmd(args []string) {
	cfg, _ := loadCfg(args)
	if err := ui.Run(cfg, openDB(cfg)); err != nil {
		panic(err)
	}
}

func guiCmd(_ []string) {
	fmt.Println("Minimal Local GUI mode is not yet implemented in this release candidate.")
	fmt.Println("To help justify its existence, provide a field use-case not satisfied by the TUI.")
	os.Exit(0)
}
