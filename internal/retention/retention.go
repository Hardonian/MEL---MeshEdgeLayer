package retention

import (
	"fmt"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
)

func Run(database *db.DB, cfg config.Config) error {
	if database == nil {
		return nil
	}
	msgCutoff := time.Now().UTC().AddDate(0, 0, -cfg.Retention.MessagesDays).Format(time.RFC3339)
	teleCutoff := time.Now().UTC().AddDate(0, 0, -cfg.Retention.TelemetryDays).Format(time.RFC3339)
	auditCutoff := time.Now().UTC().AddDate(0, 0, -cfg.Retention.AuditDays).Format(time.RFC3339)
	if err := database.Exec(fmt.Sprintf("DELETE FROM messages WHERE rx_time != '' AND rx_time < '%s';", msgCutoff)); err != nil {
		return err
	}
	if err := database.Exec(fmt.Sprintf("DELETE FROM telemetry_samples WHERE observed_at < '%s';", teleCutoff)); err != nil {
		return err
	}
	if err := database.Exec(fmt.Sprintf("DELETE FROM audit_logs WHERE created_at < '%s';", auditCutoff)); err != nil {
		return err
	}
	if err := database.Exec(fmt.Sprintf("DELETE FROM dead_letters WHERE created_at < '%s';", auditCutoff)); err != nil {
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
	return database.Exec("INSERT INTO retention_jobs(job_name,last_run,last_status,details) VALUES('default', datetime('now'), 'ok', 'retention sweep complete incl. transport intelligence pruning');")
}
