package web

import (
	"net/http"

	"github.com/mel-project/mel/internal/upgrade"
	"github.com/mel-project/mel/internal/version"
)

// versionHandler returns version information about the MEL instance
func (s *Server) versionHandler(w http.ResponseWriter, r *http.Request) {
	v := version.GetVersion()

	// Get schema version from database if available
	var dbSchemaVersion string
	if s.db != nil {
		dbSchemaVersion, _ = s.db.SchemaVersion()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"version":              v.Version,
		"git_commit":          v.GitCommit,
		"build_time":          v.BuildTime,
		"go_version":          v.GoVersion,
		"db_schema_version":   v.DBSchemaVersion,
		"db_actual_version":   dbSchemaVersion,
		"compatibility_level": v.CompatibilityLevel,
	})
}

// upgradeHealthHandler returns upgrade readiness status
func (s *Server) upgradeHealthHandler(w http.ResponseWriter, r *http.Request) {
	// Run upgrade checks
	report := upgrade.RunUpgradeChecks(s.cfg, s.db)

	writeJSON(w, http.StatusOK, report)
}
