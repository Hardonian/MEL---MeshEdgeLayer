// Package doctor implements host-level diagnostic checks shared by the CLI and support tooling.
package doctor

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/privacy"
	"github.com/mel-project/mel/internal/security"
	statuspkg "github.com/mel-project/mel/internal/status"
	"github.com/mel-project/mel/internal/transport"
	"github.com/mel-project/mel/internal/upgrade"
	"github.com/mel-project/mel/internal/version"
)

// Run performs the same checks as `mel doctor` and returns structured output plus findings.
// Callers may augment the returned map (e.g. preflight adds operator_next_steps).
func Run(cfg config.Config, path string) (map[string]any, []map[string]string) {
	findings := ValidateConfigFile(path, cfg)
	database, err := db.Open(cfg)
	if err != nil {
		findings = append(findings, map[string]string{"component": "db", "severity": "critical", "message": err.Error(), "guidance": "Fix storage.database_path or parent directory permissions before launch."})
	}
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
		if n, err := database.HighestMigrationNumeric(); err == nil {
			dbChecks["migration_numeric"] = n
			dbChecks["binary_expects_migration_numeric"] = version.CurrentSchemaVersion
			if n != version.CurrentSchemaVersion {
				findings = append(findings, map[string]string{
					"component": "schema_compat",
					"severity":  "critical",
					"message":   version.DescribeSchemaGap(n, version.CurrentSchemaVersion),
					"guidance":  "Run mel bootstrap run --config <path> or mel serve to apply migrations, or align binary with database backup.",
				})
			}
		}
		chainRep, cerr := database.VerifyAuditLogChain()
		if cerr != nil {
			findings = append(findings, map[string]string{"component": "audit_chain", "severity": "high", "message": cerr.Error(), "guidance": "Run mel audit verify --config <path> after fixing database access."})
		} else if !chainRep.OK {
			findings = append(findings, map[string]string{"component": "audit_chain", "severity": "high", "message": chainRep.Error, "guidance": "Treat as potential tampering; restore from backup or investigate audit_logs chain_hash continuity."})
		} else {
			dbChecks["audit_chain_ok"] = true
			dbChecks["audit_chain_verified_rows"] = chainRep.VerifiedRows
			dbChecks["audit_chain_legacy_rows"] = chainRep.LegacyRows
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
	statusSnap, statusErr := statuspkg.Collect(cfg, database, nil, nil, path)
	if statusErr != nil {
		findings = append(findings, map[string]string{"component": "status", "severity": "high", "message": statusErr.Error(), "guidance": "Fix transport or database reporting before relying on doctor output."})
	}
	findings = append(findings, TransportChecks(cfg, database)...)
	findings = append(findings, BindConflict(cfg)...)
	loadedBytes, _ := os.ReadFile(path)
	eff := config.Inspect(cfg, loadedBytes)
	upgradeReport := upgrade.RunUpgradeChecks(cfg, database)
	out := map[string]any{
		"doctor_version": "v2",
		"config":         path,
		"config_inspect": eff,
		"upgrade":        upgradeReport,
		"findings":       findings,
		"db":             dbChecks,
		"summary": map[string]any{
			"privacy_findings":       privacy.Summary(privacy.Audit(cfg)),
			"enabled_transports":     EnabledTransportNames(cfg),
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
	return out, findings
}

// ValidateConfigFile returns doctor-style findings for config and filesystem prerequisites.
func ValidateConfigFile(path string, cfg config.Config) []map[string]string {
	findings := make([]map[string]string, 0)
	if err := config.Validate(cfg); err != nil {
		findings = append(findings, map[string]string{"component": "config", "severity": "critical", "message": err.Error(), "guidance": "Fix the listed config validation errors before launching MEL."})
	}
	if err := security.CheckFileMode(path); err != nil {
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

// BindConflict checks whether bind.api's port is likely free on loopback when bound to all interfaces.
func BindConflict(cfg config.Config) []map[string]string {
	findings := make([]map[string]string, 0)
	host, portStr, err := net.SplitHostPort(cfg.Bind.API)
	if err != nil {
		return findings
	}
	if host != "" && host != "0.0.0.0" && host != "::" && host != "[::]" {
		return findings
	}
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", portStr))
	if err != nil {
		findings = append(findings, map[string]string{
			"component": "bind",
			"severity":  "high",
			"message":   fmt.Sprintf("port %s appears in use or not bindable on loopback: %v", portStr, err),
			"guidance":  "Stop the conflicting process or change bind.api before starting MEL.",
		})
		return findings
	}
	_ = ln.Close()
	return findings
}

// TransportChecks performs static and persisted-runtime transport checks (serial path, TCP dial, MQTT topic lint, runtime errors).
func TransportChecks(cfg config.Config, database *db.DB) []map[string]string {
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

// EnabledTransportNames returns names of enabled transports in config order.
func EnabledTransportNames(cfg config.Config) []string {
	names := make([]string, 0)
	for _, t := range cfg.Transports {
		if t.Enabled {
			names = append(names, t.Name)
		}
	}
	return names
}

func firstError(err error, fallback string) string {
	if err != nil {
		return err.Error()
	}
	return fallback
}
