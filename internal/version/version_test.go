package version

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
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

func TestCurrentSchemaVersionMatchesHighestMigrationFile(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	entries, err := os.ReadDir(filepath.Join(root, "migrations"))
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	highest := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".sql")
		parts := strings.SplitN(name, "_", 2)
		if len(parts) == 0 {
			continue
		}
		n, err := strconv.Atoi(parts[0])
		if err != nil {
			t.Fatalf("migration prefix parse failed for %q: %v", e.Name(), err)
		}
		if n > highest {
			highest = n
		}
	}
	if highest != CurrentSchemaVersion {
		t.Fatalf("CurrentSchemaVersion=%d, highest migration file=%d", CurrentSchemaVersion, highest)
	}
}
