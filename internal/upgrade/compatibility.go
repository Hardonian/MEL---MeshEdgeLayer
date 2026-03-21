package upgrade

import (
	"fmt"
	"strconv"
	"strings"
)

// CompatibilityLevel represents the stability of a version
type CompatibilityLevel string

const (
	// LevelStable is a stable production release
	LevelStable CompatibilityLevel = "stable"
	// LevelPreview is a preview/RC release
	LevelPreview CompatibilityLevel = "preview"
	// LevelDev is a development build
	LevelDev CompatibilityLevel = "dev"
)

// UpgradePath represents a planned upgrade
type UpgradePath struct {
	FromVersion string
	ToVersion   string
	FromSchema  int
	ToSchema    int
}

// UpgradeResult contains the result of an upgrade compatibility check
type UpgradeResult struct {
	Allowed          bool
	Reason          string
	Severity        string // "error", "warning", "info"
	RequiredMigrations []string
	BreakingChanges []string
}

// MigrationInfo represents information about a schema migration
type MigrationInfo struct {
	Version      string
	Name         string
	Reversible   bool
	Destructive  bool
	Description  string
}

// CheckCompatibility checks if upgrading from one version to another is compatible
func CheckCompatibility(current, target string) *UpgradeResult {
	result := &UpgradeResult{
		Allowed:          true,
		Severity:         "info",
		Reason:           "Upgrade path is compatible",
		RequiredMigrations: []string{},
		BreakingChanges:  []string{},
	}

	currentLevel := GetCompatibilityLevel(current)
	targetLevel := GetCompatibilityLevel(target)

	// Check if upgrading from stable to dev is allowed
	if currentLevel == LevelStable && targetLevel == LevelDev {
		result.Allowed = false
		result.Severity = "error"
		result.Reason = "Cannot downgrade from stable to development build"
		return result
	}

	// Check major version differences
	currentMajor, _, _ := ParseVersion(current)
	targetMajor, _, _ := ParseVersion(target)

	if targetMajor > currentMajor+1 {
		result.Allowed = false
		result.Severity = "error"
		result.Reason = fmt.Sprintf("Cannot skip major versions: %d -> %d", currentMajor, targetMajor)
		return result
	}

	if targetMajor > currentMajor {
		// Major version upgrade - warn about breaking changes
		result.Severity = "warning"
		result.Reason = fmt.Sprintf("Major version upgrade: v%d.x.x to v%d.x.x", currentMajor, targetMajor)
		result.BreakingChanges = []string{
			"Database schema changes may be required",
			"Configuration format changes possible",
			"API breaking changes may apply",
		}
	}

	// Check for pre-1.0 instability
	if currentMajor == 0 || targetMajor == 0 {
		result.Severity = "warning"
		if result.Reason == "Upgrade path is compatible" {
			result.Reason = "Pre-1.0 versions may include breaking changes"
		}
	}

	return result
}

// IsUpgradeAllowed returns whether an upgrade is allowed with a reason
func IsUpgradeAllowed(fromVersion, toVersion string) (bool, string) {
	result := CheckCompatibility(fromVersion, toVersion)
	return result.Allowed, result.Reason
}

// GetRequiredMigrations returns the list of migrations needed for a schema upgrade
func GetRequiredMigrations(currentSchema, targetSchema int) []MigrationInfo {
	migrations := []MigrationInfo{}

	// Generate migration info based on schema version difference
	for i := currentSchema + 1; i <= targetSchema; i++ {
		migrations = append(migrations, MigrationInfo{
			Version:     fmt.Sprintf("%04d", i),
			Name:        getMigrationName(i),
			Reversible:  isMigrationReversible(i),
			Destructive: isMigrationDestructive(i),
			Description: getMigrationDescription(i),
		})
	}

	return migrations
}

// IsRollbackSupported checks if rollback from one version to another is feasible
func IsRollbackSupported(fromVersion, toVersion string) (bool, string) {
	fromMajor, fromMinor, _ := ParseVersion(fromVersion)
	toMajor, toMinor, _ := ParseVersion(toVersion)

	// Rollback is not officially supported
	if fromMajor > toMajor {
		return false, "Rollback across major versions is not supported. Restore from backup instead."
	}

	if fromMajor == toMajor && fromMinor-toMinor > 1 {
		return false, fmt.Sprintf("Rollback of more than one minor version (%d -> %d) is not supported", fromMinor, toMinor)
	}

	return true, "Rollback may work but is not guaranteed. Ensure you have a backup."
}

// IsMigrationPathSafe checks if a migration path is safe (no irreversible operations)
func IsMigrationPathSafe(fromSchema, toSchema int) (bool, []string) {
	safetyIssues := []string{}

	for i := fromSchema + 1; i <= toSchema; i++ {
		if isMigrationDestructive(i) {
			safetyIssues = append(safetyIssues, fmt.Sprintf("Migration %04d is destructive: %s", i, getMigrationName(i)))
		}
		if !isMigrationReversible(i) {
			safetyIssues = append(safetyIssues, fmt.Sprintf("Migration %04d is not reversible: %s", i, getMigrationName(i)))
		}
	}

	return len(safetyIssues) == 0, safetyIssues
}

// GetCompatibilityLevel determines the compatibility level from a version string
func GetCompatibilityLevel(v string) CompatibilityLevel {
	v = strings.ToLower(v)
	if strings.Contains(v, "-dev") || strings.Contains(v, ".dev") {
		return LevelDev
	}
	if strings.Contains(v, "-rc") || strings.Contains(v, "-alpha") || strings.Contains(v, "-beta") {
		return LevelPreview
	}
	// Check if it's a stable release (no pre-release markers and major version >= 1)
	major, _, _, preRelease := ParseFullVersion(v)
	if major >= 1 && preRelease == "" {
		return LevelStable
	}
	// Pre-1.0 releases are considered preview for compatibility purposes
	return LevelPreview
}

// ParseVersion extracts major, minor, patch from a semantic version string
func ParseVersion(v string) (major, minor, patch int) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	if len(parts) >= 1 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) >= 3 {
		// Remove any pre-release suffix
		patchPart := strings.Split(parts[2], "-")[0]
		patch, _ = strconv.Atoi(patchPart)
	}
	return
}

// ParseFullVersion extracts major, minor, patch and pre-release info
func ParseFullVersion(v string) (major, minor, patch int, preRelease string) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	if len(parts) >= 1 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) >= 3 {
		patchPart := strings.Split(parts[2], "-")[0]
		patch, _ = strconv.Atoi(patchPart)
		if idx := strings.Index(parts[2], "-"); idx != -1 {
			preRelease = parts[2][idx+1:]
		}
	}
	return
}

// Helper functions for migration information

func getMigrationName(schemaNum int) string {
	// This would be populated from actual migration files in production
	names := map[int]string{
		1:  "init",
		2:  "runtime_truth",
		3:  "dead_letters",
		4:  "transport_runtime_evidence",
		5:  "dead_letter_transport_type",
		6:  "transport_failure_episodes",
		7:  "transport_intelligence",
		8:  "transport_intelligence_operability",
		9:  "transport_mesh_anomaly_history",
		10: "guarded_control",
		11: "control_closure",
		12: "performance_indexes",
		13: "audit_log",
		14: "operator_sessions",
		15: "incidents",
	}
	if name, ok := names[schemaNum]; ok {
		return name
	}
	return fmt.Sprintf("schema_%d", schemaNum)
}

func isMigrationReversible(schemaNum int) bool {
	// Migrations that add tables or nullable columns are reversible
	// Migrations that drop columns or rename things are not
	irreversibleMigrations := []int{2, 10, 11}
	for _, m := range irreversibleMigrations {
		if m == schemaNum {
			return false
		}
	}
	return true
}

func isMigrationDestructive(schemaNum int) bool {
	// Migrations that delete data are destructive
	destructiveMigrations := []int{2, 10, 11}
	for _, m := range destructiveMigrations {
		if m == schemaNum {
			return true
		}
	}
	return false
}

func getMigrationDescription(schemaNum int) string {
	descriptions := map[int]string{
		1:  "Initial schema creation",
		2:  "Runtime truth table changes",
		3:  "Dead letter storage",
		4:  "Transport runtime evidence",
		5:  "Dead letter transport type",
		6:  "Transport failure episodes",
		7:  "Transport intelligence",
		8:  "Transport intelligence operability",
		9:  "Transport mesh anomaly history",
		10: "Guarded control actions",
		11: "Control closure",
		12: "Performance indexes",
		13: "Audit log support",
		14: "Operator sessions",
		15: "Incident tracking",
	}
	if desc, ok := descriptions[schemaNum]; ok {
		return desc
	}
	return "Schema migration"
}
