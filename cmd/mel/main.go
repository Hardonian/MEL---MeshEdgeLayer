package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/mel-project/mel/internal/backup"
	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/diagnostics"
	"github.com/mel-project/mel/internal/policy"
	"github.com/mel-project/mel/internal/privacy"
	"github.com/mel-project/mel/internal/security"
	"github.com/mel-project/mel/internal/service"
	statuspkg "github.com/mel-project/mel/internal/status"
	"github.com/mel-project/mel/internal/transport"
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
	switch os.Args[1] {
	case "init":
		initCmd(os.Args[2:])
	case "version":
		fmt.Println(version.GetFullVersionString())
	case "config":
		configCmd(os.Args[2:])
	case "serve":
		serveCmd(os.Args[2:])
	case "doctor":
		doctorCmd(os.Args[2:])
	case "status":
		statusCmd(os.Args[2:])
	case "panel":
		panelCmd(os.Args[2:])
	case "nodes":
		nodesCmd(os.Args[2:])
	case "node":
		nodeCmd(os.Args[2:])
	case "transports":
		transportsCmd(os.Args[2:])
	case "inspect":
		inspectCmd(os.Args[2:])
	case "db":
		dbCmd(os.Args[2:])
	case "export":
		exportCmd(os.Args[2:])
	case "import":
		importCmd(os.Args[2:])
	case "logs":
		logsCmd(os.Args[2:])
	case "policy":
		policyCmd(os.Args[2:])
	case "control":
		controlCmd(os.Args[2:])
	case "privacy":
		privacyCmd(os.Args[2:])
	case "backup":
		backupCmd(os.Args[2:])
	case "replay":
		replayCmd(os.Args[2:])
	case "diagnostics":
		diagnosticsCmd(os.Args[2:])
	case "health":
		healthCmd(os.Args[2:])
	case "dev-simulate-mqtt":
		simulateCmd(os.Args[2:])
	case "ui":
		uiCmd(os.Args[2:])
	case "gui":
		guiCmd(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`mel commands:
  init
  version
  doctor --config <path>
  config validate --config <path>
  serve [--debug] --config <path>
  status --config <path>
  panel [--format text|json] --config <path>
  nodes --config <path>
  node inspect <node-id> --config <path>
  transports list --config <path>
  inspect transport <name> --config <path>
  inspect mesh --config <path>
  replay --config <path> [--node <id>] [--type <message-type>] [--limit <n>]
  privacy audit [--format json|text] --config <path>
  policy explain --config <path>
  control status --config <path>
  control history --config <path> [--transport <name>] [--limit <n>]
  export --config <path> [--out path]
  import validate --bundle <path>
  backup create --config <path> [--out path]
  backup restore --bundle <path> --dry-run (required) [--destination dir]
  logs tail --config <path>
  db vacuum --config <path>
  health internal|freshness|slo|metrics --config <path>
  ui --config <path>
  gui --config <path>
  dev-simulate-mqtt`)
}

func fs(name string) *flag.FlagSet { return flag.NewFlagSet(name, flag.ExitOnError) }

func loadCfg(args []string) (config.Config, string) {
	f := fs("cfg")
	path := f.String("config", "configs/mel.example.json", "config")
	_ = f.Parse(args)
	cfg, _, err := config.Load(*path)
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
	if len(args) == 0 || (args[0] != "validate" && args[0] != "inspect") {
		panic("usage: mel config validate|inspect --config <path>")
	}
	f := fs("config-" + args[0])
	path := f.String("config", "configs/mel.example.json", "config")
	_ = f.Parse(args[1:])
	cfg, loadedBytes, err := config.Load(*path)
	if err != nil {
		panic(err)
	}

	if args[0] == "validate" {
		findings := validateConfigFile(*path, cfg)
		mustPrint(map[string]any{"status": map[bool]string{true: "valid", false: "invalid"}[len(findings) == 0], "findings": findings, "lints": config.LintConfig(cfg)})
		if len(findings) > 0 {
			os.Exit(1)
		}
	} else if args[0] == "inspect" {
		mustPrint(config.Inspect(cfg, loadedBytes))
	}
}


func serveCmd(args []string) {
	f := fs("serve")
	path := f.String("config", "configs/mel.example.json", "config")
	debug := f.Bool("debug", false, "enable debug logging")
	_ = f.Parse(args)
	cfg, _, err := config.Load(*path)
	if err != nil {
		panic(err)
	}
	if err := requireConfigMode(*path); err != nil {
		panic(err)
	}
	app, err := service.New(cfg, *debug)
	if err != nil {
		panic(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := app.Start(ctx); err != nil {
		panic(err)
	}
}

func doctorCmd(args []string) {
	cfg, path := loadCfg(args)
	findings := validateConfigFile(path, cfg)
	database, err := db.Open(cfg)
	if err != nil {
		findings = append(findings, map[string]string{"component": "db", "severity": "critical", "message": err.Error(), "guidance": "Fix storage.database_path or parent directory permissions before launch."})
	}
	// Check database file permissions
	if _, err := os.Stat(cfg.Storage.DatabasePath); err == nil {
		if err := security.CheckFileMode(cfg.Storage.DatabasePath); err != nil {
			findings = append(findings, map[string]string{"component": "db_perms", "severity": "high", "message": err.Error(), "guidance": "Run 'chmod 600 " + cfg.Storage.DatabasePath + "' to restrict access to the database file."})
		}
	}
	dbChecks := map[string]any{"path": cfg.Storage.DatabasePath, "write_ok": false, "read_ok": false}
	if database != nil {
		if schemaVersion, err := database.SchemaVersion(); err != nil {
			findings = append(findings, map[string]string{"component": "schema", "severity": "critical", "message": err.Error(), "guidance": "Migrations must complete before launch."})
		} else {
			dbChecks["schema_version"] = schemaVersion
		}
		if err := database.Exec("CREATE TABLE IF NOT EXISTS doctor_write_check(v INTEGER); DELETE FROM doctor_write_check; INSERT INTO doctor_write_check(v) VALUES (1);"); err != nil {
			findings = append(findings, map[string]string{"component": "db_write", "severity": "critical", "message": err.Error(), "guidance": "Ensure sqlite3 can write to the configured database path."})
		} else {
			dbChecks["write_ok"] = true
			if value, err := database.Scalar("SELECT v FROM doctor_write_check LIMIT 1;"); err != nil || value != "1" {
				findings = append(findings, map[string]string{"component": "db_read", "severity": "critical", "message": firstError(err, fmt.Sprintf("unexpected readback value %q", value)), "guidance": "Doctor must be able to read back its temporary validation row."})
			} else {
				dbChecks["read_ok"] = true
			}
		}
	}
	statusSnap, statusErr := statuspkg.Collect(cfg, database, nil)
	if statusErr != nil {
		findings = append(findings, map[string]string{"component": "status", "severity": "high", "message": statusErr.Error(), "guidance": "Fix transport or database reporting before relying on doctor output."})
	}
	findings = append(findings, doctorTransportChecks(cfg, database)...)
	out := map[string]any{
		"doctor_version": "v2",
		"config":         path,
		"findings":       findings,
		"db":             dbChecks,
		"summary": map[string]any{
			"privacy_findings":       privacy.Summary(privacy.Audit(cfg)),
			"enabled_transports":     enabledTransportNames(cfg),
			"last_successful_ingest": statusSnap.LastSuccessfulIngest,
			"transport_status":       statusSnap.Transports,
			"what_mel_does": []string{
				"observes configured transports and persists received packets to SQLite",
				"reports live vs historical transport truth without inventing traffic",
				"exposes read-only HTTP status, nodes, messages, and metrics endpoints",
			},
			"what_mel_does_not_do": []string{
				"does not claim unsupported Meshtastic transports or send capability",
				"does not prove hardware validation that was not exercised in this environment",
				"does not mark ingest successful unless the message was written to SQLite",
			},
		},
	}
	mustPrint(out)

	// Add self-observability output
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

func statusCmd(args []string) {
	cfg, _ := loadCfg(args)
	d := openDB(cfg)
	snap, err := statuspkg.Collect(cfg, d, nil)
	if err != nil {
		panic(err)
	}
	mustPrint(snap)
}

func panelCmd(args []string) {
	f := fs("panel")
	path := f.String("config", "configs/mel.example.json", "config")
	format := f.String("format", "text", "text|json")
	_ = f.Parse(args)
	cfg, _, err := config.Load(*path)
	if err != nil {
		panic(err)
	}
	snap, err := statuspkg.Collect(cfg, openDB(cfg), nil)
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
	if len(args) < 2 || args[0] != "inspect" {
		panic("usage: mel node inspect <node-id> --config <path>")
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
	cfg, _ := loadCfg(args[1:])
	snap, err := statuspkg.Collect(cfg, openDB(cfg), nil)
	if err != nil {
		panic(err)
	}
	mustPrint(map[string]any{"transports": snap.Transports, "contention_warning": len(enabledTransportNames(cfg)) > 1, "selection_rule": "prefer one direct-node transport; hybrid direct+MQTT dedupes only when both paths expose byte-identical mesh packet payloads, so operators must still verify duplicate behavior in their own deployment"})
}

func inspectCmd(args []string) {
	if len(args) == 0 {
		panic("usage: mel inspect transport <name> --config <path> | mel inspect mesh --config <path>")
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
	default:
		panic("usage: mel inspect transport <name> --config <path> | mel inspect mesh --config <path>")
	}
}

func diagnosticsCmd(args []string) {
	f := fs("diagnostics")
	configPath := f.String("config", "configs/mel.example.json", "path to config file")
	jsonOutput := f.Bool("json", false, "output as JSON")
	f.Parse(args)

	cfg, _, err := config.Load(*configPath)
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
	f := fs("replay")
	path := f.String("config", "configs/mel.example.json", "config")
	node := f.String("node", "", "filter by node number")
	messageType := f.String("type", "", "filter by message type")
	limit := f.Int("limit", 50, "maximum rows")
	_ = f.Parse(args)
	cfg, _, err := config.Load(*path)
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
	rows, err := d.QueryRows(fmt.Sprintf("SELECT transport_name,packet_id,from_node,to_node,portnum,payload_text,payload_json,rx_time,created_at FROM messages WHERE %s ORDER BY id DESC LIMIT %d;", strings.Join(clauses, " AND "), *limit))
	if err != nil {
		panic(err)
	}
	mustPrint(map[string]any{"messages": rows, "filters": map[string]any{"node": *node, "type": *messageType, "limit": *limit}})
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
	path := f.String("config", "configs/mel.example.json", "config")
	outPath := f.String("out", "", "write export bundle to file instead of stdout")
	_ = f.Parse(args)
	cfg, _, err := config.Load(*path)
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
		panic("usage: mel logs tail --config <path>")
	}
	cfg, _ := loadCfg(args[1:])
	d := openDB(cfg)
	rows, err := d.QueryRows("SELECT category,level,message,created_at FROM audit_logs ORDER BY id DESC LIMIT 20;")
	if err != nil {
		panic(err)
	}
	mustPrint(rows)
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
		path := f.String("config", "configs/mel.example.json", "config")
		transportName := f.String("transport", "", "filter by transport")
		start := f.String("start", "", "start time RFC3339")
		end := f.String("end", "", "end time RFC3339")
		limit := f.Int("limit", 50, "max rows")
		offset := f.Int("offset", 0, "offset")
		_ = f.Parse(args[1:])
		cfg, _, err := config.Load(*path)
		if err != nil {
			panic(err)
		}
		d := openDB(cfg)
		actions, err := d.ControlActions(*transportName, "", *start, *end, *limit, *offset)
		if err != nil {
			panic(err)
		}
		decisions, err := d.ControlDecisions(*transportName, "", *start, *end, *limit, *offset)
		if err != nil {
			panic(err)
		}
		mustPrint(map[string]any{"actions": actions, "decisions": decisions, "transport": *transportName, "start": *start, "end": *end, "pagination": map[string]any{"limit": *limit, "offset": *offset}})
	default:
		panic("usage: mel control status|history --config <path>")
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
	path := f.String("config", "configs/mel.example.json", "config")
	format := f.String("format", "json", "json|text")
	_ = f.Parse(args[1:])
	cfg, _, err := config.Load(*path)
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
		path := f.String("config", "configs/mel.example.json", "config")
		outPath := f.String("out", "", "bundle output path")
		_ = f.Parse(args[1:])
		cfg, _, err := config.Load(*path)
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

func validateConfigFile(path string, cfg config.Config) []map[string]string {
	findings := make([]map[string]string, 0)
	if err := config.Validate(cfg); err != nil {
		findings = append(findings, map[string]string{"component": "config", "severity": "critical", "message": err.Error(), "guidance": "Fix the listed config validation errors before launching MEL."})
	}
	if err := requireConfigMode(path); err != nil {
		findings = append(findings, map[string]string{"component": "config_file", "severity": "high", "message": err.Error(), "guidance": "Operator config files must be chmod 600 before MEL will trust them in production."})
	}
	if info, err := os.Stat(cfg.Storage.DataDir); err != nil {
		findings = append(findings, map[string]string{"component": "data_dir", "severity": "high", "message": err.Error(), "guidance": "Create the data directory and grant MEL access before launch."})
	} else if !info.IsDir() {
		findings = append(findings, map[string]string{"component": "data_dir", "severity": "high", "message": "data_dir is not a directory", "guidance": "Point storage.data_dir at a writable directory."})
	}
	for _, lint := range config.LintConfig(cfg) {
		findings = append(findings, map[string]string{"component": lint.ID, "severity": lint.Severity, "message": lint.Message, "guidance": lint.Remediation})
	}
	return findings
}

func requireConfigMode(path string) error { return security.CheckFileMode(path) }

func doctorTransportChecks(cfg config.Config, database *db.DB) []map[string]string {
	findings := make([]map[string]string, 0)
	enabled := 0
	directEnabled := 0
	for _, t := range cfg.Transports {
		if !t.Enabled {
			continue
		}
		enabled++
		switch t.Type {
		case "serial":
			directEnabled++
			device := t.SerialDevice
			if device == "" {
				device = t.Endpoint
			}
			info, err := os.Stat(device)
			if err != nil {
				if os.IsNotExist(err) {
					findings = append(findings, map[string]string{"component": t.Name, "severity": "high", "message": "serial device not found: " + device, "guidance": "Reconnect the node, confirm the configured path, and prefer /dev/serial/by-id/... for stable naming."})
				} else if os.IsPermission(err) {
					findings = append(findings, map[string]string{"component": t.Name, "severity": "high", "message": "permission denied reading serial device: " + device, "guidance": "Add the MEL service user to dialout/uucp or update udev rules, then retry."})
				} else {
					findings = append(findings, map[string]string{"component": t.Name, "severity": "high", "message": err.Error(), "guidance": "Inspect the serial path and host dmesg output for device errors."})
				}
				continue
			}
			if info.Mode()&os.ModeDevice == 0 {
				findings = append(findings, map[string]string{"component": t.Name, "severity": "medium", "message": "configured serial path exists but is not a device: " + device, "guidance": "Point MEL at the real tty device exposed by the Meshtastic node."})
			}
			f, err := os.OpenFile(device, os.O_RDWR, 0)
			if err != nil {
				msg := "serial device exists but could not be opened: " + err.Error()
				guidance := "Ensure no other client owns the node and that MEL has read/write access."
				if os.IsPermission(err) {
					msg = "serial device permission denied; add the MEL service user to dialout/uucp or adjust udev rules"
					guidance = "Refresh group membership or service credentials, then rerun doctor."
				} else if strings.Contains(strings.ToLower(err.Error()), "resource busy") || strings.Contains(strings.ToLower(err.Error()), "device or resource busy") {
					msg = "serial device is busy; another process appears to own the port"
					guidance = "Stop other Meshtastic clients, serial consoles, or lingering services before starting MEL."
				}
				findings = append(findings, map[string]string{"component": t.Name, "severity": "high", "message": msg, "guidance": guidance})
			} else {
				_ = f.Close()
			}
		case "tcp", "serialtcp":
			directEnabled++
			endpoint := t.Endpoint
			if endpoint == "" {
				endpoint = net.JoinHostPort(t.TCPHost, fmt.Sprint(t.TCPPort))
			}
			conn, err := net.DialTimeout("tcp", endpoint, 2*time.Second)
			if err != nil {
				findings = append(findings, map[string]string{"component": t.Name, "severity": "high", "message": "TCP endpoint unreachable: " + endpoint + ": " + err.Error(), "guidance": "Confirm host/port, listener protocol, and firewall reachability from this machine."})
			} else {
				_ = conn.Close()
			}
		case "mqtt":
			if !strings.Contains(t.Topic, "msh/") {
				findings = append(findings, map[string]string{"component": t.Name, "severity": "medium", "message": "MQTT topic does not look like a Meshtastic topic filter: " + t.Topic, "guidance": "Confirm the broker topic pattern matches the packet feed you expect MEL to ingest."})
			}
		}
	}
	if enabled == 0 {
		findings = append(findings, map[string]string{"component": "transports", "severity": "medium", "message": "no transports are enabled; MEL will start but remain explicitly idle", "guidance": "Enable exactly one primary transport before expecting live mesh data."})
	}
	if directEnabled > 1 {
		findings = append(findings, map[string]string{"component": "transports", "severity": "high", "message": "multiple direct-node transports are enabled; choose one to avoid serial/TCP ownership contention", "guidance": "Run one direct serial/TCP attachment path at a time unless you have proven shared radio ownership outside MEL."})
	}
	if database != nil {
		runtimeRows, err := database.TransportRuntimeStatuses()
		if err == nil {
			for _, row := range runtimeRows {
				if row.State == transport.StateError && row.LastError != "" {
					findings = append(findings, map[string]string{"component": row.Name, "severity": "high", "message": "last runtime error: " + row.LastError, "guidance": "Fix the surfaced transport error, then rerun doctor and confirm the state advances beyond error."})
				}
			}
		}
	}
	return findings
}

func openDB(cfg config.Config) *db.DB {
	d, err := db.Open(cfg)
	if err != nil {
		panic(err)
	}
	return d
}

func mustPrint(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
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

func enabledTransportNames(cfg config.Config) []string {
	names := make([]string, 0)
	for _, t := range cfg.Transports {
		if t.Enabled {
			names = append(names, t.Name)
		}
	}
	return names
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

func firstError(err error, fallback string) string {
	if err != nil {
		return err.Error()
	}
	return fallback
}

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
