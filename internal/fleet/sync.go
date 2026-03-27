package fleet

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
)

// SyncScopeMetadata persists operator-declared scope from config into instance_metadata for export/support parity.
func SyncScopeMetadata(cfg config.Config, database *db.DB) error {
	if database == nil {
		return nil
	}
	if _, err := database.EnsureInstanceID(); err != nil {
		return err
	}
	s := strings.TrimSpace(cfg.Scope.SiteID)
	if s != "" {
		if err := database.SetInstanceMetadata(db.MetaSiteID, s); err != nil {
			return fmt.Errorf("persist site_id: %w", err)
		}
	}
	f := strings.TrimSpace(cfg.Scope.FleetID)
	if f != "" {
		if err := database.SetInstanceMetadata(db.MetaFleetID, f); err != nil {
			return fmt.Errorf("persist fleet_id: %w", err)
		}
	}
	l := strings.TrimSpace(cfg.Scope.FleetLabel)
	if l != "" {
		if err := database.SetInstanceMetadata(db.MetaFleetLabel, l); err != nil {
			return fmt.Errorf("persist fleet_label: %w", err)
		}
	}
	if cfg.Scope.ExpectedFleetReporterCount > 0 {
		v := strconv.Itoa(cfg.Scope.ExpectedFleetReporterCount)
		if err := database.SetInstanceMetadata(db.MetaExpectedFleetReporters, v); err != nil {
			return fmt.Errorf("persist expected_fleet_reporters: %w", err)
		}
	}
	return nil
}
