package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/upgrade"
)

func bootstrapCmd(args []string) {
	if len(args) == 0 {
		panic("usage: mel bootstrap run|validate --config <path> [--dry-run]")
	}
	switch args[0] {
	case "run":
		bootstrapRun(args[1:])
	case "validate":
		bootstrapValidate(args[1:])
	default:
		panic("usage: mel bootstrap run|validate --config <path> [--dry-run]")
	}
}

func bootstrapRun(args []string) {
	f := fs("bootstrap-run")
	path := f.String("config", configFlagDefault(), "config")
	dry := f.Bool("dry-run", false, "validate only; do not create directories or open database")
	_ = f.Parse(args)
	cfg, loaded, err := loadConfigFile(*path)
	if err != nil {
		panic(err)
	}
	findings := validateConfigFile(*path, cfg)
	if *dry {
		mustPrint(map[string]any{"status": "dry_run", "would_create": []string{cfg.Storage.DataDir, filepath.Dir(cfg.Storage.DatabasePath)}, "findings": findings})
		if len(findings) > 0 {
			os.Exit(1)
		}
		return
	}
	if err := os.MkdirAll(cfg.Storage.DataDir, 0o755); err != nil {
		panic(err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.Storage.DatabasePath), 0o755); err != nil {
		panic(err)
	}
	database, err := db.Open(cfg)
	if err != nil {
		panic(err)
	}
	eff := config.Inspect(cfg, loaded)
	_ = database.InsertAuditLog("bootstrap", "info", "bootstrap run completed", map[string]any{
		"config_path":           *path,
		"canonical_fingerprint": eff.CanonicalFingerprint,
	})
	mustPrint(map[string]any{"status": "ok", "data_dir": cfg.Storage.DataDir, "database": cfg.Storage.DatabasePath, "config_inspect": eff, "findings": findings})
	if len(findings) > 0 {
		os.Exit(1)
	}
}

func bootstrapValidate(args []string) {
	f := fs("bootstrap-validate")
	path := f.String("config", configFlagDefault(), "config")
	_ = f.Parse(args)
	cfg, loaded, err := loadConfigFile(*path)
	if err != nil {
		panic(err)
	}
	findings := validateConfigFile(*path, cfg)
	database, err := db.Open(cfg)
	if err != nil {
		findings = append(findings, map[string]string{"component": "db", "severity": "critical", "message": err.Error(), "guidance": "Fix storage paths and permissions; run mel bootstrap run when ready."})
	} else {
		rep, verr := database.VerifyAuditLogChain()
		if verr != nil {
			findings = append(findings, map[string]string{"component": "audit_chain", "severity": "high", "message": verr.Error(), "guidance": "Database may be corrupt; restore from backup if verification cannot run."})
		} else if !rep.OK {
			findings = append(findings, map[string]string{"component": "audit_chain", "severity": "high", "message": rep.Error, "guidance": "Run mel audit verify for details; treat as potential tampering or partial migration."})
		}
	}
	eff := config.Inspect(cfg, loaded)
	mustPrint(map[string]any{"status": map[bool]string{true: "ok", false: "issues"}[len(findings) == 0], "config_inspect": eff, "findings": findings})
	if len(findings) > 0 {
		os.Exit(1)
	}
}

func upgradeCmd(args []string) {
	if len(args) == 0 || args[0] != "preflight" {
		panic("usage: mel upgrade preflight --config <path>")
	}
	f := fs("upgrade-preflight")
	path := f.String("config", configFlagDefault(), "config")
	_ = f.Parse(args[1:])
	cfg, _, err := loadConfigFile(*path)
	if err != nil {
		panic(err)
	}
	database, err := db.Open(cfg)
	if err != nil {
		panic(fmt.Errorf("open database for preflight: %w", err))
	}
	rep := upgrade.RunUpgradeChecks(cfg, database)
	mustPrint(rep)
	if !rep.Ready {
		os.Exit(1)
	}
}

func auditCmd(args []string) {
	if len(args) == 0 || args[0] != "verify" {
		panic("usage: mel audit verify --config <path>")
	}
	f := fs("audit-verify")
	path := f.String("config", configFlagDefault(), "config")
	_ = f.Parse(args[1:])
	cfg, _, err := loadConfigFile(*path)
	if err != nil {
		panic(err)
	}
	database, err := db.Open(cfg)
	if err != nil {
		panic(err)
	}
	rep, err := database.VerifyAuditLogChain()
	if err != nil {
		panic(err)
	}
	mustPrint(rep)
	if !rep.OK {
		os.Exit(1)
	}
}
