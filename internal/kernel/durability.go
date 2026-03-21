package kernel

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// StorageDurability provides crash-safe writes, corruption detection,
// recovery from partial writes, and backup/restore for the kernel's
// SQLite databases.
type StorageDurability struct {
	dbPath    string
	backupDir string
}

// NewStorageDurability creates a durability manager.
func NewStorageDurability(dbPath, backupDir string) *StorageDurability {
	return &StorageDurability{
		dbPath:    dbPath,
		backupDir: backupDir,
	}
}

// IntegrityCheck runs SQLite integrity check on the database.
func (sd *StorageDurability) IntegrityCheck() (bool, string, error) {
	cmd := exec.Command("sqlite3", sd.dbPath, "PRAGMA integrity_check;")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, string(out), fmt.Errorf("integrity check failed: %w", err)
	}
	result := strings.TrimSpace(string(out))
	return result == "ok", result, nil
}

// SetJournalMode sets the SQLite journal mode (WAL recommended for durability).
func (sd *StorageDurability) SetJournalMode(mode string) error {
	sql := fmt.Sprintf("PRAGMA journal_mode=%s;", mode)
	cmd := exec.Command("sqlite3", sd.dbPath, sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("set journal mode: %w: %s", err, out)
	}
	return nil
}

// EnableWAL enables Write-Ahead Logging for crash safety.
func (sd *StorageDurability) EnableWAL() error {
	return sd.SetJournalMode("wal")
}

// Checkpoint forces a WAL checkpoint.
func (sd *StorageDurability) Checkpoint() error {
	cmd := exec.Command("sqlite3", sd.dbPath, "PRAGMA wal_checkpoint(TRUNCATE);")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("checkpoint: %w: %s", err, out)
	}
	return nil
}

// Backup creates a consistent backup of the database using SQLite's backup API.
func (sd *StorageDurability) Backup() (string, error) {
	if err := os.MkdirAll(sd.backupDir, 0o755); err != nil {
		return "", fmt.Errorf("create backup dir: %w", err)
	}

	timestamp := time.Now().UTC().Format("20060102-150405")
	backupPath := filepath.Join(sd.backupDir, fmt.Sprintf("mel-backup-%s.db", timestamp))

	sql := fmt.Sprintf(".backup '%s'", backupPath)
	cmd := exec.Command("sqlite3", sd.dbPath, sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("backup: %w: %s", err, out)
	}

	return backupPath, nil
}

// Restore restores a database from a backup file.
func (sd *StorageDurability) Restore(backupPath string) error {
	// Verify backup integrity first
	cmd := exec.Command("sqlite3", backupPath, "PRAGMA integrity_check;")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("backup integrity check failed: %w: %s", err, out)
	}
	if strings.TrimSpace(string(out)) != "ok" {
		return fmt.Errorf("backup file is corrupt: %s", out)
	}

	// Copy backup over current database
	src, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("open backup: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(sd.dbPath)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy backup: %w", err)
	}

	return nil
}

// FileChecksum returns the SHA-256 checksum of the database file.
func (sd *StorageDurability) FileChecksum() (string, error) {
	f, err := os.Open(sd.dbPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ListBackups returns available backup files sorted by name (newest first).
func (sd *StorageDurability) ListBackups() ([]BackupInfo, error) {
	entries, err := os.ReadDir(sd.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var backups []BackupInfo
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".db") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		backups = append(backups, BackupInfo{
			Path:      filepath.Join(sd.backupDir, e.Name()),
			Name:      e.Name(),
			SizeBytes: info.Size(),
			CreatedAt: info.ModTime(),
		})
	}
	return backups, nil
}

// PruneBackups removes old backups, keeping only the most recent `keep` files.
func (sd *StorageDurability) PruneBackups(keep int) error {
	backups, err := sd.ListBackups()
	if err != nil {
		return err
	}

	if len(backups) <= keep {
		return nil
	}

	for _, b := range backups[keep:] {
		_ = os.Remove(b.Path)
	}
	return nil
}

// BackupInfo describes a backup file.
type BackupInfo struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}
