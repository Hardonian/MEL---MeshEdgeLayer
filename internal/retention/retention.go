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
	return database.Exec("INSERT INTO retention_jobs(job_name,last_run,last_status,details) VALUES('default', datetime('now'), 'ok', 'retention sweep complete incl. transport intelligence pruning');")
}
