package upgrade

import (
	"fmt"
	"time"

	"github.com/mel-project/mel/internal/db"
)

// BackupRequirement defines when backups are required
type BackupRequirement int

const (
	// BackupNotRequired - no backup needed
	BackupNotRequired BackupRequirement = iota
	// BackupRecommended - backup suggested but not required
	BackupRecommended
	// BackupRequired - backup is required before proceeding
	BackupRequired
)

// BackupGuard provides backup confirmation checks for dangerous operations
type BackupGuard struct {
	db                 *db.DB
	requiredBeforeOps  []string
	lastBackupTime     time.Time
	maxBackupAge       time.Duration
}

// NewBackupGuard creates a new backup guard instance
func NewBackupGuard(database *db.DB) *BackupGuard {
	return &BackupGuard{
		db:                database,
		requiredBeforeOps: []string{"schema_migration", "data_deletion", "column_drop", "table_drop"},
		maxBackupAge:      7 * 24 * time.Hour, // 7 days
	}
}

// RequireBackupConfirmation checks if a recent backup exists before dangerous operations
func (bg *BackupGuard) RequireBackupConfirmation(operation string) (BackupRequirement, string) {
	if bg.db == nil {
		return BackupRecommended, "Database not available - cannot verify backup status"
	}

	// Check if this operation requires a backup
	requiresBackup := false
	for _, op := range bg.requiredBeforeOps {
		if operation == op {
			requiresBackup = true
			break
		}
	}

	if !requiresBackup {
		return BackupNotRequired, "Operation does not require backup"
	}

	// Check for recent backup in backup_metadata table
	backupTime, err := bg.getLastBackupTime()
	if err != nil {
		return BackupRecommended, fmt.Sprintf("Cannot verify backup status: %v", err)
	}

	if backupTime.IsZero() {
		return BackupRequired, "No backup found - a backup is required before this operation"
	}

	age := time.Since(backupTime)
	if age > bg.maxBackupAge {
		return BackupRequired, fmt.Sprintf("Last backup is too old (%.0f days) - a fresh backup is required", age.Hours()/24)
	}

	return BackupNotRequired, fmt.Sprintf("Recent backup exists (%.0f days ago)", age.Hours()/24)
}

// getLastBackupTime retrieves the timestamp of the last backup
func (bg *BackupGuard) getLastBackupTime() (time.Time, error) {
	// Try to get backup time from backup_metadata table
	result, err := bg.db.Scalar("SELECT MAX(backup_time) FROM backup_metadata;")
	if err != nil {
		return time.Time{}, err
	}

	if result == "" {
		return time.Time{}, nil
	}

	// Try parsing the time
	parsed, err := time.Parse(time.RFC3339, result)
	if err != nil {
		return time.Time{}, err
	}

	return parsed, nil
}

// RecordBackupTimestamp stores the current time as a backup timestamp
func (bg *BackupGuard) RecordBackupTimestamp() error {
	if bg.db == nil {
		return fmt.Errorf("database not available")
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Create backup_metadata table if it doesn't exist
	err := bg.db.Exec("CREATE TABLE IF NOT EXISTS backup_metadata(id INTEGER PRIMARY KEY, backup_time TEXT, created_at TEXT DEFAULT CURRENT_TIMESTAMP);")
	if err != nil {
		return fmt.Errorf("failed to create backup_metadata table: %w", err)
	}

	// Insert backup timestamp
	err = bg.db.Exec(fmt.Sprintf("INSERT INTO backup_metadata(backup_time) VALUES('%s');", now))
	if err != nil {
		return fmt.Errorf("failed to record backup timestamp: %w", err)
	}

	bg.lastBackupTime, _ = time.Parse(time.RFC3339, now)
	return nil
}

// ConfirmBackupRequirement confirms that user has acknowledged backup requirement
// This is a placeholder - in production, you'd integrate with actual user confirmation
func (bg *BackupGuard) ConfirmBackupRequirement(operation string, confirmed bool) error {
	if !confirmed {
		return fmt.Errorf("operation '%s' requires backup confirmation", operation)
	}

	// Record that backup was confirmed
	if bg.db != nil {
		now := time.Now().UTC().Format(time.RFC3339)
		_ = bg.db.Exec(fmt.Sprintf("INSERT INTO audit_logs(category, level, message, details_json, created_at) VALUES('upgrade', 'info', 'Backup confirmed for operation: %s', '{}', '%s');",
			operation, now))
	}

	return nil
}

// GetBackupAge returns the age of the last backup
func (bg *BackupGuard) GetBackupAge() (time.Duration, error) {
	backupTime, err := bg.getLastBackupTime()
	if err != nil {
		return 0, err
	}

	if backupTime.IsZero() {
		return 0, fmt.Errorf("no backup found")
	}

	return time.Since(backupTime), nil
}

// HasRecentBackup checks if there's a backup within the max age
func (bg *BackupGuard) HasRecentBackup() (bool, error) {
	age, err := bg.GetBackupAge()
	if err != nil {
		return false, nil // No backup = not recent
	}

	return age < bg.maxBackupAge, nil
}
