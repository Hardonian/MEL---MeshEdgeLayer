package version

import (
	"testing"
)

func TestGetVersion(t *testing.T) {
	v := GetVersion()

	if v.Version == "" {
		t.Error("Version should not be empty")
	}

	if v.DBSchemaVersion == 0 {
		t.Error("DBSchemaVersion should be set")
	}
}

func TestGetVersionString(t *testing.T) {
	// Save original values
	origVersion := Version
	origCommit := GitCommit
	origLevel := CompatibilityLevel

	// Test with dev build
	Version = "0.1.0-dev"
	GitCommit = "abc1234"
	CompatibilityLevel = "dev"

	vStr := GetVersionString()
	if vStr == "" {
		t.Error("Version string should not be empty")
	}

	// Restore
	Version = origVersion
	GitCommit = origCommit
	CompatibilityLevel = origLevel
}

func TestGetFullVersionString(t *testing.T) {
	vStr := GetFullVersionString()
	if vStr == "" {
		t.Error("Full version string should not be empty")
	}
}

func TestIsDevBuild(t *testing.T) {
	// Save original
	origLevel := CompatibilityLevel
	origVer := Version

	CompatibilityLevel = "dev"
	Version = "1.0.0"
	if !IsDevBuild() {
		t.Error("Should be dev build with dev compatibility level")
	}

	CompatibilityLevel = "stable"
	Version = "1.0.0"
	if IsDevBuild() {
		t.Error("Should not be dev build with stable compatibility level and stable version string")
	}

	// Restore
	CompatibilityLevel = origLevel
	Version = origVer
}

func TestIsStable(t *testing.T) {
	// Save original
	orig := CompatibilityLevel

	CompatibilityLevel = "stable"
	if !IsStable() {
		t.Error("Should be stable with stable compatibility level")
	}

	CompatibilityLevel = "dev"
	if IsStable() {
		t.Error("Should not be stable with dev compatibility level")
	}

	// Restore
	CompatibilityLevel = orig
}
