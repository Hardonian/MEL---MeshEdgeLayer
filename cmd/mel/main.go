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
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/policy"
	"github.com/mel-project/mel/internal/privacy"
	"github.com/mel-project/mel/internal/security"
	"github.com/mel-project/mel/internal/service"
	"github.com/mel-project/mel/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "init":
		initCmd(os.Args[2:])
	case "version":
		fmt.Println(version.Version)
	case "config":
		configCmd(os.Args[2:])
	case "serve":
		serveCmd(os.Args[2:])
	case "doctor":
		doctorCmd(os.Args[2:])
	case "status":
		statusCmd(os.Args[2:])
	case "nodes":
		nodesCmd(os.Args[2:])
	case "node":
		nodeCmd(os.Args[2:])
	case "transports":
		transportsCmd(os.Args[2:])
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
	case "privacy":
		privacyCmd(os.Args[2:])
	case "backup":
		backupCmd(os.Args[2:])
	case "dev-simulate-mqtt":
		simulateCmd(os.Args[2:])
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
  serve --config <path>
  status --config <path>
  nodes --config <path>
  node inspect <node-id> --config <path>
  transports list --config <path>
  privacy audit [--format json|text] --config <path>
  policy explain --config <path>
  export --config <path> [--out path]
  import validate --bundle <path>
  backup create --config <path> [--out path]
  backup restore --bundle <path> --dry-run [--destination dir]
  logs tail --config <path>
  db vacuum --config <path>
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
}

func configCmd(args []string) {
	if len(args) == 0 || args[0] != "validate" {
		panic("usage: mel config validate --config <path>")
	}
	cfg, _ := loadCfg(args[1:])
	if err := config.Validate(cfg); err != nil {
		panic(err)
	}
	mustPrint(map[string]any{"status": "valid", "lints": config.LintConfig(cfg)})
}

func serveCmd(args []string) {
	cfg, path := loadCfg(args)
	if err := security.CheckFileMode(path); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}
	app, err := service.New(cfg)
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
	findings := []map[string]string{}
	if err := config.Validate(cfg); err != nil {
		findings = append(findings, map[string]string{"component": "config", "severity": "critical", "message": err.Error()})
	}
	if _, err := db.Open(cfg); err != nil {
		findings = append(findings, map[string]string{"component": "db", "severity": "critical", "message": err.Error()})
	}
	if info, err := os.Stat(cfg.Storage.DataDir); err != nil {
		findings = append(findings, map[string]string{"component": "data_dir", "severity": "high", "message": err.Error()})
	} else if !info.IsDir() {
		findings = append(findings, map[string]string{"component": "data_dir", "severity": "high", "message": "data_dir is not a directory"})
	}
	if err := security.CheckFileMode(path); err != nil {
		findings = append(findings, map[string]string{"component": "config_file", "severity": "medium", "message": err.Error()})
	}
	for _, lint := range config.LintConfig(cfg) {
		findings = append(findings, map[string]string{"component": lint.ID, "severity": lint.Severity, "message": lint.Message})
	}
	out := map[string]any{"config": path, "findings": findings, "summary": map[string]any{"privacy_findings": privacy.Summary(privacy.Audit(cfg)), "enabled_transports": enabledTransportNames(cfg)}}
	mustPrint(out)
	if len(findings) > 0 {
		os.Exit(1)
	}
}

func statusCmd(args []string) {
	cfg, _ := loadCfg(args)
	d := openDB(cfg)
	nodes, _ := d.Scalar("SELECT COUNT(*) FROM nodes;")
	messages, _ := d.Scalar("SELECT COUNT(*) FROM messages;")
	schemaVersion, _ := d.SchemaVersion()
	mustPrint(map[string]any{
		"bind":            cfg.Bind.API,
		"bind_local_only": !cfg.Bind.AllowRemote,
		"schema_version":  schemaVersion,
		"nodes":           nodes,
		"messages":        messages,
		"transports":      cfg.Transports,
	})
}

func nodesCmd(args []string) {
	cfg, _ := loadCfg(args)
	d := openDB(cfg)
	rows, err := d.QueryRows("SELECT node_num,node_id,long_name,short_name,last_seen,last_gateway_id,lat_redacted,lon_redacted,altitude,last_snr,last_rssi FROM nodes ORDER BY updated_at DESC;")
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
	rows, err := d.QueryRows(fmt.Sprintf("SELECT node_num,node_id,long_name,short_name,last_seen,last_gateway_id,lat_redacted,lon_redacted,altitude,last_snr,last_rssi FROM nodes WHERE CAST(node_num AS TEXT)='%s' OR node_id='%s' LIMIT 1;", escape(target), escape(target)))
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
	mustPrint(map[string]any{"transports": cfg.Transports, "contention_warning": len(enabledTransportNames(cfg)) > 1})
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
	messages, err := d.QueryRows("SELECT transport_name,packet_id,channel_id,gateway_id,from_node,to_node,portnum,payload_text,rx_time FROM messages ORDER BY id DESC LIMIT 250;")
	if err != nil {
		panic(err)
	}
	bundle := map[string]any{"exported_at": time.Now().UTC().Format(time.RFC3339), "redacted": cfg.Privacy.RedactExports, "nodes": nodes, "messages": messages}
	if cfg.Privacy.RedactExports {
		bundle["messages"] = redactMessages(messages)
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
		bundlePath := f.String("bundle", "", "bundle path")
		dryRun := f.Bool("dry-run", false, "validate only")
		destination := f.String("destination", ".", "restore directory")
		_ = f.Parse(args[1:])
		if *bundlePath == "" {
			panic("--bundle is required")
		}
		if !*dryRun {
			panic("only --dry-run restore is implemented in this release candidate")
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
	var names []string
	for _, t := range cfg.Transports {
		if t.Enabled {
			names = append(names, t.Name)
		}
	}
	return names
}

func redactMessages(rows []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		cloned := map[string]any{}
		for k, v := range row {
			cloned[k] = v
		}
		cloned["payload_text"] = "[redacted]"
		out = append(out, cloned)
	}
	return out
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

func escape(v string) string { return strings.ReplaceAll(v, "'", "''") }

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
