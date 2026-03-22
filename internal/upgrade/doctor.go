package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/version"
)

func schemaVersionNumeric(database *db.DB) (int, string, error) {
	if database == nil {
		return 0, "", fmt.Errorf("database not available")
	}
	schemaVer, err := database.Scalar(`SELECT version FROM schema_migrations ORDER BY CAST(substr(version,1,4) AS INTEGER) DESC, version DESC LIMIT 1;`)
	if err != nil {
		return 0, "", err
	}
	return version.MigrationNumeric(schemaVer), schemaVer, nil
}

// CheckStatus represents the status of a single check
type CheckStatus string

const (
	CheckPass CheckStatus = "pass"
	CheckFail CheckStatus = "fail"
	CheckWarn CheckStatus = "warn"
	CheckSkip CheckStatus = "skip"
)

// UpgradeCheck represents a single pre-upgrade check
type UpgradeCheck struct {
	Name        string         `json:"name"`
	Status      CheckStatus    `json:"status"`
	Message     string         `json:"message"`
	Severity    string         `json:"severity"` // "critical", "warning", "info"
	Remediation string         `json:"remediation,omitempty"`
	Details     map[string]any `json:"details,omitempty"`
}

// UpgradeReadinessReport contains the result of all pre-upgrade checks
type UpgradeReadinessReport struct {
	Ready          bool                `json:"ready"`
	OverallStatus  string              `json:"overall_status"` // "ready", "not_ready", "warnings"
	Timestamp      string              `json:"timestamp"`
	VersionInfo    version.VersionInfo `json:"version_info"`
	CurrentSchema  int                 `json:"current_schema"`
	RequiredSchema int                 `json:"required_schema"`
	Checks         []UpgradeCheck      `json:"checks"`
	Summary        UpgradeSummary      `json:"summary"`
}

// UpgradeSummary provides a count of checks by status
type UpgradeSummary struct {
	Total    int `json:"total"`
	Passed   int `json:"passed"`
	Failed   int `json:"failed"`
	Warnings int `json:"warnings"`
	Skipped  int `json:"skipped"`
}

// RunUpgradeChecks runs all pre-upgrade validation checks
func RunUpgradeChecks(cfg config.Config, database *db.DB) *UpgradeReadinessReport {
	report := &UpgradeReadinessReport{
		Ready:          true,
		OverallStatus:  "ready",
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		VersionInfo:    version.GetVersion(),
		CurrentSchema:  0,
		RequiredSchema: version.CurrentSchemaVersion,
		Checks:         []UpgradeCheck{},
		Summary:        UpgradeSummary{},
	}

	// Run all checks
	report.Checks = append(report.Checks, checkDiskSpace(cfg)...)
	report.Checks = append(report.Checks, checkDBIntegrity(cfg, database)...)
	report.Checks = append(report.Checks, checkBackupExists(cfg, database)...)
	report.Checks = append(report.Checks, checkVersionCompatibility()...)
	report.Checks = append(report.Checks, checkSchemaVersion(database)...)

	if database != nil {
		cur, id, err := schemaVersionNumeric(database)
		if err == nil {
			report.CurrentSchema = cur
		}
		_ = id
	}

	// Calculate summary
	for _, check := range report.Checks {
		report.Summary.Total++
		switch check.Status {
		case CheckPass:
			report.Summary.Passed++
		case CheckFail:
			report.Summary.Failed++
			report.Ready = false
			report.OverallStatus = "not_ready"
		case CheckWarn:
			report.Summary.Warnings++
			if report.OverallStatus == "ready" {
				report.OverallStatus = "warnings"
			}
		case CheckSkip:
			report.Summary.Skipped++
		}
	}

	return report
}

// checkDiskSpace verifies adequate disk space is available
func checkDiskSpace(cfg config.Config) []UpgradeCheck {
	dbPath := cfg.Storage.DatabasePath
	if dbPath == "" {
		dbPath = "./data/mel.db"
	}

	// Get the directory containing the database
	dir := filepath.Dir(dbPath)

	// Check if we can stat the directory
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return []UpgradeCheck{{
			Name:        "disk_space",
			Status:      CheckWarn,
			Message:     fmt.Sprintf("Data directory does not exist: %s", dir),
			Severity:    "warning",
			Remediation: "Data directory will be created on startup",
		}}
	}

	// For now, we just check if the directory is writable
	// In production, you'd want to check actual disk space
	testFile := filepath.Join(dir, ".space_check")
	err = os.WriteFile(testFile, []byte(""), 0644)
	if err != nil {
		return []UpgradeCheck{{
			Name:        "disk_space",
			Status:      CheckFail,
			Message:     fmt.Sprintf("Cannot write to data directory: %s", dir),
			Severity:    "critical",
			Remediation: "Ensure the data directory exists and is writable",
		}}
	}
	os.Remove(testFile)

	return []UpgradeCheck{{
		Name:     "disk_space",
		Status:   CheckPass,
		Message:  "Data directory is writable",
		Severity: "info",
		Details:  map[string]any{"directory": dir},
	}}
}

// checkDBIntegrity verifies the database is healthy
func checkDBIntegrity(cfg config.Config, database *db.DB) []UpgradeCheck {
	if database == nil {
		return []UpgradeCheck{{
			Name:        "db_integrity",
			Status:      CheckSkip,
			Message:     "Database not available",
			Severity:    "warning",
			Remediation: "Database connection required for full validation",
		}}
	}

	// Check if we can query the database
	_, err := database.Scalar("SELECT COUNT(*) FROM schema_migrations;")
	if err != nil {
		return []UpgradeCheck{{
			Name:        "db_integrity",
			Status:      CheckFail,
			Message:     fmt.Sprintf("Cannot query schema_migrations: %v", err),
			Severity:    "critical",
			Remediation: "Ensure database is accessible and migrations have been applied",
		}}
	}

	// Check for common tables
	requiredTables := []string{"nodes", "messages", "schema_migrations"}
	for _, table := range requiredTables {
		_, err := database.Scalar(fmt.Sprintf("SELECT COUNT(*) FROM %s;", table))
		if err != nil {
			return []UpgradeCheck{{
				Name:        "db_integrity",
				Status:      CheckFail,
				Message:     fmt.Sprintf("Required table '%s' not found or not accessible", table),
				Severity:    "critical",
				Remediation: "Ensure all migrations have been applied",
				Details:     map[string]any{"table": table, "error": err.Error()},
			}}
		}
	}

	return []UpgradeCheck{{
		Name:     "db_integrity",
		Status:   CheckPass,
		Message:  "Database is accessible and tables exist",
		Severity: "info",
		Details:  map[string]any{"tables_checked": len(requiredTables)},
	}}
}

// checkBackupExists verifies a recent backup is available
func checkBackupExists(cfg config.Config, database *db.DB) []UpgradeCheck {
	if database == nil {
		return []UpgradeCheck{{
			Name:     "backup_exists",
			Status:   CheckSkip,
			Message:  "Database not available",
			Severity: "warning",
		}}
	}

	// Check for recent backup timestamp in database
	// This is a placeholder - in production, you'd check for actual backup files
	backupTime, err := database.Scalar("SELECT COALESCE(MAX(backup_time), '') FROM backup_metadata;")
	if err != nil || backupTime == "" {
		return []UpgradeCheck{{
			Name:        "backup_exists",
			Status:      CheckWarn,
			Message:     "No recent backup found in database",
			Severity:    "warning",
			Remediation: "Create a backup before upgrading: mel backup create --config <path>",
		}}
	}

	// Parse backup time and check if it's recent (within 7 days)
	backupTimeParsed, err := time.Parse(time.RFC3339, backupTime)
	if err != nil {
		return []UpgradeCheck{{
			Name:        "backup_exists",
			Status:      CheckWarn,
			Message:     "Could not parse backup timestamp",
			Severity:    "warning",
			Remediation: "Create a fresh backup before upgrading",
		}}
	}

	daysSinceBackup := time.Since(backupTimeParsed).Hours() / 24
	if daysSinceBackup > 7 {
		return []UpgradeCheck{{
			Name:        "backup_exists",
			Status:      CheckWarn,
			Message:     fmt.Sprintf("Last backup was %.0f days ago", daysSinceBackup),
			Severity:    "warning",
			Remediation: "Create a new backup before upgrading: mel backup create --config <path>",
			Details:     map[string]any{"days_since_backup": daysSinceBackup},
		}}
	}

	return []UpgradeCheck{{
		Name:     "backup_exists",
		Status:   CheckPass,
		Message:  fmt.Sprintf("Recent backup exists (%.0f days ago)", daysSinceBackup),
		Severity: "info",
		Details:  map[string]any{"days_since_backup": daysSinceBackup},
	}}
}

// checkVersionCompatibility checks if the current version is compatible
func checkVersionCompatibility() []UpgradeCheck {
	v := version.GetVersion()
	checks := []UpgradeCheck{}

	// Check compatibility level
	if v.CompatibilityLevel == version.CompatibilityDev {
		checks = append(checks, UpgradeCheck{
			Name:        "version_compatibility",
			Status:      CheckWarn,
			Message:     "Running development build",
			Severity:    "warning",
			Remediation: "Use stable release for production environments",
		})
	} else if v.CompatibilityLevel == version.CompatibilityPreview {
		checks = append(checks, UpgradeCheck{
			Name:        "version_compatibility",
			Status:      CheckWarn,
			Message:     "Running preview/RC build",
			Severity:    "warning",
			Remediation: "Preview releases may have unknown issues",
		})
	} else {
		checks = append(checks, UpgradeCheck{
			Name:     "version_compatibility",
			Status:   CheckPass,
			Message:  "Running stable release",
			Severity: "info",
		})
	}

	// Check for pre-1.0 versions
	major, _, _ := ParseVersion(v.Version)
	if major == 0 {
		checks = append(checks, UpgradeCheck{
			Name:        "pre_1.0",
			Status:      CheckWarn,
			Message:     "Pre-1.0 versions may include breaking changes",
			Severity:    "warning",
			Remediation: "Review release notes for breaking changes",
		})
	}

	return checks
}

// checkSchemaVersion checks if schema version is compatible
func checkSchemaVersion(database *db.DB) []UpgradeCheck {
	if database == nil {
		return []UpgradeCheck{{
			Name:     "schema_version",
			Status:   CheckSkip,
			Message:  "Database not available",
			Severity: "warning",
		}}
	}

	currentSchema, schemaVer, err := schemaVersionNumeric(database)
	if err != nil {
		return []UpgradeCheck{{
			Name:        "schema_version",
			Status:      CheckFail,
			Message:     fmt.Sprintf("Cannot read schema version: %v", err),
			Severity:    "critical",
			Remediation: "Ensure migrations have been applied correctly",
		}}
	}

	requiredSchema := version.CurrentSchemaVersion

	if currentSchema < requiredSchema {
		migrationsNeeded := requiredSchema - currentSchema
		return []UpgradeCheck{{
			Name:        "schema_version",
			Status:      CheckFail,
			Message:     fmt.Sprintf("Database schema is outdated: migration %q (numeric %d), binary expects %d (%d behind)", schemaVer, currentSchema, requiredSchema, migrationsNeeded),
			Severity:    "critical",
			Remediation: "Run `mel serve --config <path>` or `mel bootstrap run --config <path>` to apply pending SQLite migrations",
			Details: map[string]any{
				"current_migration_id": schemaVer,
				"current_schema":       currentSchema,
				"required_schema":      requiredSchema,
				"migrations_needed":    migrationsNeeded,
			},
		}}
	}

	if currentSchema > requiredSchema {
		return []UpgradeCheck{{
			Name:        "schema_version",
			Status:      CheckFail,
			Message:     fmt.Sprintf("Database schema is newer than app: migration %q (numeric %d), binary expects %d", schemaVer, currentSchema, requiredSchema),
			Severity:    "critical",
			Remediation: "Downgrade MEL or restore from a compatible backup",
			Details: map[string]any{
				"current_migration_id": schemaVer,
				"current_schema":       currentSchema,
				"required_schema":      requiredSchema,
			},
		}}
	}

	return []UpgradeCheck{{
		Name:     "schema_version",
		Status:   CheckPass,
		Message:  fmt.Sprintf("Schema migration level matches binary (%d, %s)", currentSchema, schemaVer),
		Severity: "info",
		Details: map[string]any{
			"current_migration_id": schemaVer,
			"schema_numeric":       currentSchema,
		},
	}}
}
