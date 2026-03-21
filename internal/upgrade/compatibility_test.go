package upgrade

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		version   string
		wantMajor int
		wantMinor int
		wantPatch int
	}{
		{"v1.0.0", 1, 0, 0},
		{"v0.1.0", 0, 1, 0},
		{"1.2.3", 1, 2, 3},
		{"v2.0.0-rc1", 2, 0, 0},
		{"0.1.0-beta", 0, 1, 0},
	}

	for _, tt := range tests {
		major, minor, patch := ParseVersion(tt.version)
		if major != tt.wantMajor || minor != tt.wantMinor || patch != tt.wantPatch {
			t.Errorf("ParseVersion(%q) = (%d, %d, %d), want (%d, %d, %d)",
				tt.version, major, minor, patch, tt.wantMajor, tt.wantMinor, tt.wantPatch)
		}
	}
}

func TestGetCompatibilityLevel(t *testing.T) {
	tests := []struct {
		version string
		want    CompatibilityLevel
	}{
		{"v1.0.0", LevelStable},
		{"1.0.0", LevelStable},
		{"v1.0.0-rc1", LevelPreview},
		{"v1.0.0-beta", LevelPreview},
		{"v0.9.0", LevelPreview},
		{"v0.1.0-dev", LevelDev},
	}

	for _, tt := range tests {
		got := GetCompatibilityLevel(tt.version)
		if got != tt.want {
			t.Errorf("GetCompatibilityLevel(%q) = %v, want %v", tt.version, got, tt.want)
		}
	}
}

func TestCheckCompatibility(t *testing.T) {
	tests := []struct {
		current   string
		target    string
		wantAllow bool
		wantSev   string
	}{
		{"v0.1.0", "v0.2.0", true, "warning"}, // pre-1.0 warning
		{"v1.0.0", "v1.1.0", true, "info"},    // minor upgrade
		{"v1.0.0", "v2.0.0", true, "warning"}, // major upgrade
		{"v1.0.0", "v0.1.0", false, "error"},  // downgrade
		{"v1.0.0", "v3.0.0", false, "error"},  // skip major
	}

	for _, tt := range tests {
		result := CheckCompatibility(tt.current, tt.target)
		if result.Allowed != tt.wantAllow {
			t.Errorf("CheckCompatibility(%q, %q).Allowed = %v, want %v",
				tt.current, tt.target, result.Allowed, tt.wantAllow)
		}
		if result.Severity != tt.wantSev {
			t.Errorf("CheckCompatibility(%q, %q).Severity = %v, want %v",
				tt.current, tt.target, result.Severity, tt.wantSev)
		}
	}
}

func TestIsUpgradeAllowed(t *testing.T) {
	allowed, reason := IsUpgradeAllowed("v0.1.0", "v0.2.0")
	if !allowed {
		t.Errorf("Expected upgrade to be allowed, reason: %s", reason)
	}

	allowed, _ = IsUpgradeAllowed("v1.0.0", "v3.0.0")
	if allowed {
		t.Error("Expected upgrade to be disallowed for skipping major versions")
	}
}

func TestGetRequiredMigrations(t *testing.T) {
	migrations := GetRequiredMigrations(10, 15)
	if len(migrations) != 5 {
		t.Errorf("Expected 5 migrations, got %d", len(migrations))
	}

	// Check first migration
	if migrations[0].Version != "0011" {
		t.Errorf("Expected first migration to be 0011, got %s", migrations[0].Version)
	}
}

func TestIsMigrationPathSafe(t *testing.T) {
	safe, issues := IsMigrationPathSafe(10, 12)
	// Migration 11 is in this range and is destructive/irreversible
	if safe {
		t.Error("Expected migration path to not be safe for migrations 10->12")
	}
	if len(issues) == 0 {
		t.Error("Expected some safety issues to be reported")
	}

	// Path 0->5 crosses migration 0002, which is marked destructive in metadata.
	safe, _ = IsMigrationPathSafe(0, 5)
	if safe {
		t.Error("Expected migration path 0->5 to be unsafe (includes destructive migration 0002)")
	}
}

func TestIsRollbackSupported(t *testing.T) {
	supported, _ := IsRollbackSupported("v1.0.0", "v1.0.0")
	if !supported {
		t.Error("Expected rollback to same version to work")
	}

	supported, _ = IsRollbackSupported("v1.1.0", "v1.0.0")
	if supported {
		t.Error("Expected rollback to not be officially supported")
	}

	supported, _ = IsRollbackSupported("v2.0.0", "v1.0.0")
	if supported {
		t.Error("Expected major version rollback to not be supported")
	}
}

func TestParseFullVersion(t *testing.T) {
	major, minor, patch, preRelease := ParseFullVersion("v1.2.3-rc1")
	if major != 1 || minor != 2 || patch != 3 || preRelease != "rc1" {
		t.Errorf("ParseFullVersion failed: got (%d, %d, %d, %s), want (1, 2, 3, rc1)",
			major, minor, patch, preRelease)
	}

	major, minor, patch, preRelease = ParseFullVersion("v2.0.0")
	if major != 2 || minor != 0 || patch != 0 || preRelease != "" {
		t.Errorf("ParseFullVersion failed: got (%d, %d, %d, %s), want (2, 0, 0, \"\")",
			major, minor, patch, preRelease)
	}
}
