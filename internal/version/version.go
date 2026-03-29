package version

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// Version compatibility levels
const (
	CompatibilityStable  = "stable"
	CompatibilityDev     = "dev"
	CompatibilityPreview = "preview"
)

// CurrentSchemaVersion is the numeric prefix of the latest applied migration file
// (e.g. 0019_foo.sql → 19). Must stay in sync with migrations/.
const CurrentSchemaVersion = 32

// VersionInfo holds all version-related information about the MEL build
type VersionInfo struct {
	Version            string `json:"version"`             // Semantic version (e.g., "1.0.0-rc1")
	GitCommit          string `json:"git_commit"`          // Git commit hash
	BuildTime          string `json:"build_time"`          // Build timestamp
	GoVersion          string `json:"go_version"`          // Go runtime version
	DBSchemaVersion    int    `json:"db_schema_version"`   // Database schema version
	CompatibilityLevel string `json:"compatibility_level"` // stable|preview|dev
}

// Build-time variables - set via ldflags
var (
	Version            = "0.1.0-dev"      // -X github.com/mel-project/mel/internal/version.Version
	GitCommit          = "dev"            // -X github.com/mel-project/mel/internal/version.GitCommit
	BuildTime          = "now"            // -X github.com/mel-project/mel/internal/version.BuildTime
	CompatibilityLevel = CompatibilityDev // -X github.com/mel-project/mel/internal/version.CompatibilityLevel
)

// GetVersion returns the complete VersionInfo struct
func GetVersion() VersionInfo {
	return VersionInfo{
		Version:            Version,
		GitCommit:          GitCommit,
		BuildTime:          BuildTime,
		GoVersion:          runtime.Version(),
		DBSchemaVersion:    CurrentSchemaVersion,
		CompatibilityLevel: CompatibilityLevel,
	}
}

// GetVersionString returns a human-readable version string
func GetVersionString() string {
	v := GetVersion()
	parts := []string{v.Version}
	if v.GitCommit != "dev" && v.GitCommit != "" {
		parts = append(parts, fmt.Sprintf("(%s)", v.GitCommit[:min(7, len(v.GitCommit))]))
	}
	if v.CompatibilityLevel != CompatibilityStable {
		parts = append(parts, fmt.Sprintf("[%s]", v.CompatibilityLevel))
	}
	return strings.Join(parts, " ")
}

// GetFullVersionString returns a detailed multi-line version string
func GetFullVersionString() string {
	v := GetVersion()
	return fmt.Sprintf(`MEL Version Information:
  Version:            %s
  Git Commit:         %s
  Build Time:         %s
  Go Version:         %s
  DB Schema Version:  %d
  Compatibility:      %s`,
		v.Version,
		v.GitCommit,
		formatBuildTime(v.BuildTime),
		v.GoVersion,
		v.DBSchemaVersion,
		v.CompatibilityLevel,
	)
}

// formatBuildTime returns a readable format for the build time
func formatBuildTime(t string) string {
	if t == "" || t == "now" || t == "unknown" {
		return time.Now().Format("2006-01-02T15:04:05Z")
	}
	// Try to parse as time
	if ts, err := time.Parse(time.RFC3339, t); err == nil {
		return ts.Format("2006-01-02T15:04:05Z")
	}
	return t
}

// Min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// IsDevBuild returns true if this is a development build
func IsDevBuild() bool {
	return CompatibilityLevel == CompatibilityDev || strings.Contains(Version, "dev") || strings.Contains(Version, "alpha") || strings.Contains(Version, "beta")
}

// IsStable returns true if this is a stable release
func IsStable() bool {
	return CompatibilityLevel == CompatibilityStable
}
