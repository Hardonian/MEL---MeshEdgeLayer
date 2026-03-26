package retention

import (
	"fmt"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/planning"
)

func Run(database *db.DB, cfg config.Config) error {
	if database == nil {
		return nil
	}
	now := time.Now().UTC()
	if err := database.PruneMessages(now.AddDate(0, 0, -cfg.Retention.MessagesDays)); err != nil {
		return err
	}
	if err := database.PruneTelemetry(now.AddDate(0, 0, -cfg.Retention.TelemetryDays)); err != nil {
		return err
	}
	if err := database.PruneAuditLogs(now.AddDate(0, 0, -cfg.Retention.AuditDays)); err != nil {
		return err
	}
	if err := database.PruneDeadLetters(now.AddDate(0, 0, -cfg.Retention.AuditDays)); err != nil {
		return err
	}
	if err := database.PruneTransportIntelligence(time.Now().UTC().AddDate(0, 0, -cfg.Intelligence.Retention.HealthSnapshotDays), cfg.Intelligence.Retention.HealthSnapshotMaxRows); err != nil {
		return err
	}
	controlCutoff := time.Now().UTC().AddDate(0, 0, -cfg.Control.RetentionDays)
	if err := database.PruneControlHistory(controlCutoff, cfg.Intelligence.Retention.HealthSnapshotMaxRows); err != nil {
		return err
	}
	// Retention for runtime status and evidence: 90 days aligns with operational telemetry window
	runtimeCutoff := time.Now().UTC().AddDate(0, 0, -90).Format(time.RFC3339)
	if err := database.Exec(fmt.Sprintf("DELETE FROM transport_runtime_status WHERE updated_at < '%s';", runtimeCutoff)); err != nil {
		return err
	}
	if err := database.Exec(fmt.Sprintf("DELETE FROM transport_runtime_evidence WHERE updated_at < '%s';", runtimeCutoff)); err != nil {
		return err
	}
	// Retention for config apply history: 90 days allows rollback context for recent changes
	configCutoff := time.Now().UTC().AddDate(0, 0, -90).Format(time.RFC3339)
	if err := database.Exec(fmt.Sprintf("DELETE FROM config_apply_history WHERE applied_at < '%s';", configCutoff)); err != nil {
		return err
	}
	// Retention for retention_jobs itself: 30 days prevents unbounded growth of job metadata
	retentionJobsCutoff := time.Now().UTC().AddDate(0, 0, -30).Format(time.RFC3339)
	if err := database.Exec(fmt.Sprintf("DELETE FROM retention_jobs WHERE last_run < '%s';", retentionJobsCutoff)); err != nil {
		return err
	}
	if err := planning.PrunePlans(database, 200); err != nil {
		return err
	}
	if err := planning.PruneArtifacts(database, 300); err != nil {
		return err
	}
	if err := planning.PruneRecommendationOutcomes(database, 500); err != nil {
		return err
	}
	if err := planning.PrunePlanExecutions(database, 300); err != nil {
		return err
	}
	return database.Exec("INSERT INTO retention_jobs(job_name,last_run,last_status,details) VALUES('default', datetime('now'), 'ok', 'retention sweep complete incl. transport intelligence pruning');")
}
