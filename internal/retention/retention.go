package retention

import (
	"fmt"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
)

func Run(database *db.DB, cfg config.Config) error {
	msgCutoff := time.Now().UTC().AddDate(0, 0, -cfg.Retention.MessagesDays).Format(time.RFC3339)
	teleCutoff := time.Now().UTC().AddDate(0, 0, -cfg.Retention.TelemetryDays).Format(time.RFC3339)
	if err := database.Exec(fmt.Sprintf("DELETE FROM messages WHERE rx_time != '' AND rx_time < '%s';", msgCutoff)); err != nil {
		return err
	}
	if err := database.Exec(fmt.Sprintf("DELETE FROM telemetry_samples WHERE observed_at < '%s';", teleCutoff)); err != nil {
		return err
	}
	return database.Exec("INSERT INTO retention_jobs(job_name,last_run,last_status,details) VALUES('default', datetime('now'), 'ok', 'retention sweep complete');")
}
