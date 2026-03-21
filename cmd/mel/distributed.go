package main

// distributed.go — CLI commands for MEL's distributed kernel layer.
//
// These commands interact with the running mel daemon via its HTTP API,
// exposing federation, kernel replay, snapshot, region, and peer management
// to operators. They require a running daemon and the config pointing to it.
//
// Commands:
//   kernel-status    — kernel state, event log stats, backpressure, durability
//   kernel-replay    — deterministic event replay (full/windowed/scenario/dry-run/verify)
//   kernel-snapshot  — snapshot management (create/list)
//   kernel-backup    — kernel backup management (create/list)
//   kernel-eventlog  — event log inspection and query
//   federation       — federation status and sync health
//   peers            — peer list and management
//   region           — region health summary
//   topology         — global topology view

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
)

// ─── HTTP Client Helpers ──────────────────────────────────────────────────────

// apiGet performs a GET request against the daemon API and returns parsed JSON.
func apiGet(baseURL, path string) (any, error) {
	url := strings.TrimRight(baseURL, "/") + path
	resp, err := http.Get(url) //nolint:gosec // base URL is operator-provided config
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response from %s: %w", url, err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GET %s: status %d: %s", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var result any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response from %s: %w", url, err)
	}
	return result, nil
}

// apiPost performs a POST request with a JSON body and returns parsed JSON.
func apiPost(baseURL, path string, payload any) (any, error) {
	url := strings.TrimRight(baseURL, "/") + path
	var reqBody []byte
	var err error
	if payload != nil {
		reqBody, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(reqBody)) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response from %s: %w", url, err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("POST %s: status %d: %s", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var result any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response from %s: %w", url, err)
	}
	return result, nil
}

// daemonURL returns the base URL for the running daemon from config.
func daemonURL(cfg config.Config) string {
	addr := cfg.Bind.API
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	if !strings.HasPrefix(addr, "http") {
		return "http://" + addr
	}
	return addr
}

func loadCfgForDist(args []string) (config.Config, string, string) {
	f := flag.NewFlagSet("dist", flag.ExitOnError)
	cfgPath := f.String("config", configFlagDefault(), "config file path")
	urlFlag := f.String("url", "", "daemon URL (overrides config bind.api)")
	_ = f.Parse(args)
	cfg, _, err := loadConfigFile(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config %s: %v\n", *cfgPath, err)
		os.Exit(1)
	}
	base := daemonURL(cfg)
	if *urlFlag != "" {
		base = *urlFlag
	}
	return cfg, *cfgPath, base
}

// ─── kernel-status ────────────────────────────────────────────────────────────

func kernelStatusCmd(args []string) {
	_, _, base := loadCfgForDist(args)

	type section struct {
		path  string
		label string
	}

	sections := []section{
		{"/api/v1/kernel/eventlog/stats", "event_log"},
		{"/api/v1/kernel/backpressure", "backpressure"},
		{"/api/v1/kernel/durability", "durability"},
		{"/api/v1/federation/status", "federation"},
	}

	result := map[string]any{}
	var errs []string

	for _, s := range sections {
		data, err := apiGet(base, s.path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", s.label, err))
			result[s.label] = map[string]any{"error": err.Error()}
		} else {
			result[s.label] = data
		}
	}

	if len(errs) > 0 {
		result["warnings"] = errs
	}
	mustPrint(result)
}

// ─── kernel-replay ────────────────────────────────────────────────────────────

func kernelReplayCmd(args []string) {
	f := flag.NewFlagSet("kernel-replay", flag.ExitOnError)
	cfgPath := f.String("config", configFlagDefault(), "config file path")
	urlFlag := f.String("url", "", "daemon URL (overrides config bind.api)")
	mode := f.String("mode", "full", "replay mode: full|windowed|scenario|dry_run|verification")
	fromSeq := f.Uint64("from-seq", 0, "replay from this sequence number")
	toSeq := f.Uint64("to-seq", 0, "replay up to this sequence number (0 = latest)")
	since := f.String("since", "", "replay events since this RFC3339 timestamp")
	until := f.String("until", "", "replay events until this RFC3339 timestamp")
	maxEvents := f.Int("max-events", 0, "max events to replay (0 = engine default)")
	policyMode := f.String("policy-mode", "", "override policy mode (disabled|advisory|guarded_auto)")
	policyVersion := f.String("policy-version", "v1", "policy version for replay")
	compact := f.Bool("compact", false, "compact output (no effects list)")
	_ = f.Parse(args)

	cfg, _, err := loadConfigFile(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}
	base := daemonURL(cfg)
	if *urlFlag != "" {
		base = *urlFlag
	}

	// Build replay request
	req := map[string]any{
		"mode": *mode,
		"policy": map[string]any{
			"version":                *policyVersion,
			"mode":                   *policyMode,
			"require_min_confidence": cfg.Control.RequireMinConfidence,
			"allowed_actions":        cfg.Control.AllowedActions,
		},
	}

	if *fromSeq > 0 {
		req["from_sequence"] = *fromSeq
	}
	if *toSeq > 0 {
		req["to_sequence"] = *toSeq
	}
	if *since != "" {
		t, err := time.Parse(time.RFC3339, *since)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --since: %v\n", err)
			os.Exit(1)
		}
		req["since"] = t.Format(time.RFC3339)
	}
	if *until != "" {
		t, err := time.Parse(time.RFC3339, *until)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --until: %v\n", err)
			os.Exit(1)
		}
		req["until"] = t.Format(time.RFC3339)
	}
	if *maxEvents > 0 {
		req["max_events"] = *maxEvents
	}

	result, err := apiPost(base, "/api/v1/kernel/replay", req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "replay failed: %v\n", err)
		os.Exit(1)
	}

	// In compact mode, strip the effects list (can be large)
	if *compact {
		if m, ok := result.(map[string]any); ok {
			delete(m, "effects")
		}
	}

	mustPrint(result)
}

// ─── kernel-snapshot ─────────────────────────────────────────────────────────

func kernelSnapshotCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mel kernel-snapshot <list|create> [flags]")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list":
		kernelSnapshotListCmd(rest)
	case "create":
		kernelSnapshotCreateCmd(rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", sub)
		fmt.Fprintln(os.Stderr, "usage: mel kernel-snapshot <list|create>")
		os.Exit(1)
	}
}

func kernelSnapshotListCmd(args []string) {
	f := flag.NewFlagSet("kernel-snapshot list", flag.ExitOnError)
	cfgPath := f.String("config", configFlagDefault(), "config file")
	urlFlag := f.String("url", "", "daemon URL")
	limit := f.Int("limit", 10, "max snapshots to list")
	_ = f.Parse(args)

	cfg, _, err := loadConfigFile(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}
	base := daemonURL(cfg)
	if *urlFlag != "" {
		base = *urlFlag
	}

	path := fmt.Sprintf("/api/v1/kernel/snapshots?limit=%d", *limit)
	result, err := apiGet(base, path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "snapshot list failed: %v\n", err)
		os.Exit(1)
	}
	mustPrint(result)
}

func kernelSnapshotCreateCmd(args []string) {
	_, _, base := loadCfgForDist(args)
	result, err := apiPost(base, "/api/v1/kernel/snapshots", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "snapshot create failed: %v\n", err)
		os.Exit(1)
	}
	mustPrint(result)
}

// ─── kernel-backup ───────────────────────────────────────────────────────────

func kernelBackupCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mel kernel-backup <list|create> [flags]")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		_, _, base := loadCfgForDist(rest)
		result, err := apiGet(base, "/api/v1/kernel/backups")
		if err != nil {
			fmt.Fprintf(os.Stderr, "backup list failed: %v\n", err)
			os.Exit(1)
		}
		mustPrint(result)
	case "create":
		_, _, base := loadCfgForDist(rest)
		result, err := apiPost(base, "/api/v1/kernel/backup", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "backup create failed: %v\n", err)
			os.Exit(1)
		}
		mustPrint(result)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", sub)
		os.Exit(1)
	}
}

// ─── kernel-eventlog ─────────────────────────────────────────────────────────

func kernelEventlogCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mel kernel-eventlog <stats|query> [flags]")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "stats":
		_, _, base := loadCfgForDist(rest)
		result, err := apiGet(base, "/api/v1/kernel/eventlog/stats")
		if err != nil {
			fmt.Fprintf(os.Stderr, "eventlog stats failed: %v\n", err)
			os.Exit(1)
		}
		mustPrint(result)
	case "query":
		kernelEventlogQueryCmd(rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", sub)
		os.Exit(1)
	}
}

func kernelEventlogQueryCmd(args []string) {
	f := flag.NewFlagSet("kernel-eventlog query", flag.ExitOnError)
	cfgPath := f.String("config", configFlagDefault(), "config file")
	urlFlag := f.String("url", "", "daemon URL")
	eventType := f.String("type", "", "filter by event type")
	nodeID := f.String("node", "", "filter by source node ID")
	subject := f.String("subject", "", "filter by subject")
	since := f.String("since", "", "events since RFC3339 timestamp")
	until := f.String("until", "", "events until RFC3339 timestamp")
	fromSeq := f.Uint64("from-seq", 0, "events after sequence number")
	limit := f.Int("limit", 50, "max events to return")
	_ = f.Parse(args)

	cfg, _, err := loadConfigFile(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}
	base := daemonURL(cfg)
	if *urlFlag != "" {
		base = *urlFlag
	}

	query := map[string]any{
		"limit": *limit,
	}
	if *eventType != "" {
		query["event_type"] = *eventType
	}
	if *nodeID != "" {
		query["source_node_id"] = *nodeID
	}
	if *subject != "" {
		query["subject"] = *subject
	}
	if *since != "" {
		t, err := time.Parse(time.RFC3339, *since)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --since: %v\n", err)
			os.Exit(1)
		}
		query["since"] = t.Format(time.RFC3339)
	}
	if *until != "" {
		t, err := time.Parse(time.RFC3339, *until)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --until: %v\n", err)
			os.Exit(1)
		}
		query["until"] = t.Format(time.RFC3339)
	}
	if *fromSeq > 0 {
		query["after_sequence"] = *fromSeq
	}

	result, err := apiPost(base, "/api/v1/kernel/eventlog/query", query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "eventlog query failed: %v\n", err)
		os.Exit(1)
	}
	mustPrint(result)
}

// ─── federation ──────────────────────────────────────────────────────────────

func federationCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mel federation <status|sync-health> [flags]")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "status":
		_, _, base := loadCfgForDist(rest)
		result, err := apiGet(base, "/api/v1/federation/status")
		if err != nil {
			fmt.Fprintf(os.Stderr, "federation status failed: %v\n", err)
			os.Exit(1)
		}
		mustPrint(result)
	case "sync-health":
		_, _, base := loadCfgForDist(rest)
		result, err := apiGet(base, "/api/v1/federation/sync/health")
		if err != nil {
			fmt.Fprintf(os.Stderr, "federation sync-health failed: %v\n", err)
			os.Exit(1)
		}
		mustPrint(result)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", sub)
		fmt.Fprintln(os.Stderr, "usage: mel federation <status|sync-health>")
		os.Exit(1)
	}
}

// ─── peers ───────────────────────────────────────────────────────────────────

func peersCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mel peers <list> [flags]")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		_, _, base := loadCfgForDist(rest)
		result, err := apiGet(base, "/api/v1/federation/peers")
		if err != nil {
			fmt.Fprintf(os.Stderr, "peers list failed: %v\n", err)
			os.Exit(1)
		}
		mustPrint(result)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", sub)
		fmt.Fprintln(os.Stderr, "usage: mel peers list")
		os.Exit(1)
	}
}

// ─── region ──────────────────────────────────────────────────────────────────

func regionCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mel region <health> [--region <id>] [flags]")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "health":
		f := flag.NewFlagSet("region health", flag.ExitOnError)
		cfgPath := f.String("config", configFlagDefault(), "config file")
		urlFlag := f.String("url", "", "daemon URL")
		regionID := f.String("region", "", "region ID (omit for local region)")
		_ = f.Parse(rest)

		cfg, _, err := loadConfigFile(*cfgPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
			os.Exit(1)
		}
		base := daemonURL(cfg)
		if *urlFlag != "" {
			base = *urlFlag
		}

		rid := *regionID
		if rid == "" {
			rid = cfg.Federation.Region
			if rid == "" {
				rid = "default"
			}
		}
		result, err := apiGet(base, "/api/v1/topology/region/"+rid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "region health failed: %v\n", err)
			os.Exit(1)
		}
		mustPrint(result)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", sub)
		fmt.Fprintln(os.Stderr, "usage: mel region health [--region <id>]")
		os.Exit(1)
	}
}

// ─── topology ────────────────────────────────────────────────────────────────

func topologyCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mel topology <global> [flags]")
		os.Exit(1)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "global":
		_, _, base := loadCfgForDist(rest)
		result, err := apiGet(base, "/api/v1/topology/global")
		if err != nil {
			fmt.Fprintf(os.Stderr, "global topology failed: %v\n", err)
			os.Exit(1)
		}
		mustPrint(result)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", sub)
		fmt.Fprintln(os.Stderr, "usage: mel topology global")
		os.Exit(1)
	}
}
