package web

import (
	"net/http"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/fleet"
	"github.com/mel-project/mel/internal/logging"
	"github.com/mel-project/mel/internal/operatorreadiness"
	"github.com/mel-project/mel/internal/platform"
	"github.com/mel-project/mel/internal/runtime"
	"github.com/mel-project/mel/internal/upgrade"
	"github.com/mel-project/mel/internal/version"
)

// versionHandler returns version information about the MEL instance
func (s *Server) versionHandler(w http.ResponseWriter, r *http.Request) {
	v := version.GetVersion()

	// Get schema version from database if available
	var dbSchemaVersion string
	var schemaNumeric int
	var schemaOK bool
	if s.db != nil {
		dbSchemaVersion, _ = s.db.SchemaVersion()
		if n, err := s.db.HighestMigrationNumeric(); err == nil {
			schemaNumeric = n
			schemaOK = n == version.CurrentSchemaVersion
		}
	}

	eff := config.Inspect(s.cfg, nil)
	bootMeta := map[string]any{}
	if s.db != nil {
		if fp, ok, _ := s.db.GetInstanceMetadata(db.MetaBootConfigFingerprint); ok {
			bootMeta["last_boot_canonical_fingerprint"] = fp
		}
		if p, ok, _ := s.db.GetInstanceMetadata(db.MetaBootConfigPath); ok {
			bootMeta["last_boot_config_path"] = p
		}
		if t, ok, _ := s.db.GetInstanceMetadata(db.MetaBootAt); ok {
			bootMeta["last_boot_at"] = t
		}
	}

	out := map[string]any{
		"version":                      v.Version,
		"topology_model_enabled":       s.cfg.Topology.Enabled,
		"git_commit":                   v.GitCommit,
		"build_time":                   v.BuildTime,
		"go_version":                   v.GoVersion,
		"db_schema_version":            v.DBSchemaVersion,
		"db_actual_version":            dbSchemaVersion,
		"db_migration_numeric":         schemaNumeric,
		"schema_matches_binary":        schemaOK,
		"compatibility_level":          v.CompatibilityLevel,
		"config_canonical_fingerprint": eff.CanonicalFingerprint,
		"boot_metadata":                bootMeta,
		"product":                      runtime.BuildProductEnvelope(s.cfg),
		"platform_posture":             platform.BuildPosture(s.cfg),
		"operator_readiness":           operatorreadiness.FromConfig(s.cfg),
	}
	if s.db != nil {
		if id, err := s.db.EnsureInstanceID(); err == nil {
			out["instance_id"] = id
		}
		if ft, err := fleet.BuildTruthSummary(s.cfg, s.db); err == nil {
			out["fleet_truth"] = ft
		}
	}
	if !s.processStartedAt.IsZero() {
		p := runtime.NewProcessIdentity(s.processStartedAt)
		out["process"] = p
		out["uptime_seconds"] = int64(time.Since(s.processStartedAt).Seconds())
	}
	writeJSON(w, http.StatusOK, out)
}

// upgradeHealthHandler returns upgrade readiness status
func (s *Server) upgradeHealthHandler(w http.ResponseWriter, r *http.Request) {
	// Run upgrade checks
	report := upgrade.RunUpgradeChecks(s.cfg, s.db)

	writeJSON(w, http.StatusOK, report)
}

func (s *Server) auditVerifyHandler(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": db.ErrDatabaseUnavailable})
		return
	}
	rep, err := s.db.VerifyAuditLogChain()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(logging.ClassifyError(err)))
		return
	}
	code := http.StatusOK
	if !rep.OK {
		code = http.StatusConflict
	}
	writeJSON(w, code, rep)
}
