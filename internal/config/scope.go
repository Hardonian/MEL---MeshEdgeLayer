package config

import (
	"fmt"
	"strings"
)

func normalizeScope(cfg *Config) {
	if cfg.Scope.ExpectedFleetReporterCount < 0 {
		cfg.Scope.ExpectedFleetReporterCount = 0
	}
}

func validateScope(cfg Config) []string {
	var errs []string
	if cfg.Scope.ExpectedFleetReporterCount < 0 {
		errs = append(errs, "scope.expected_fleet_reporter_count must be zero or positive")
	}
	if cfg.Scope.ExpectedFleetReporterCount > 1 && strings.TrimSpace(cfg.Scope.SiteID) == "" {
		errs = append(errs, "when scope.expected_fleet_reporter_count > 1, scope.site_id should be set so partial-fleet visibility is explicit")
	}
	if cfg.Scope.ExpectedFleetReporterCount > 10000 {
		errs = append(errs, fmt.Sprintf("scope.expected_fleet_reporter_count unrealistically large: %d", cfg.Scope.ExpectedFleetReporterCount))
	}
	return errs
}
