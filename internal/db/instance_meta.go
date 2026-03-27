package db

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

const (
	MetaBootConfigFingerprint = "boot_config_fingerprint"
	MetaBootConfigPath        = "boot_config_path"
	MetaBootAt                = "boot_at"
	// MetaInstanceID is a durable random id for this SQLite database; stable across process restarts.
	MetaInstanceID = "instance_id"
)

func (d *DB) SetInstanceMetadata(key, value string) error {
	if !IsSafeIdentifier(key) {
		return fmt.Errorf("invalid metadata key")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	sql := fmt.Sprintf(`INSERT INTO instance_metadata(key,value,updated_at) VALUES('%s','%s','%s')
		ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at;`,
		esc(key), esc(value), esc(now))
	return d.Exec(sql)
}

// EnsureInstanceID returns the persisted instance id, creating one on first use.
func (d *DB) EnsureInstanceID() (string, error) {
	if d == nil {
		return "", fmt.Errorf("database is nil")
	}
	if v, ok, _ := d.GetInstanceMetadata(MetaInstanceID); ok && v != "" {
		return v, nil
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate instance id: %w", err)
	}
	id := hex.EncodeToString(b[:])
	if err := d.SetInstanceMetadata(MetaInstanceID, id); err != nil {
		return "", err
	}
	return id, nil
}

func (d *DB) GetInstanceMetadata(key string) (string, bool, error) {
	if !IsSafeIdentifier(key) {
		return "", false, fmt.Errorf("invalid metadata key")
	}
	v, err := d.Scalar(fmt.Sprintf(`SELECT value FROM instance_metadata WHERE key='%s' LIMIT 1;`, esc(key)))
	if err != nil {
		return "", false, err
	}
	if v == "" {
		return "", false, nil
	}
	return v, true, nil
}

// RecordBackupCompletion inserts a row into backup_metadata after a successful backup bundle write.
func (d *DB) RecordBackupCompletion(bundlePath string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	sql := fmt.Sprintf(`INSERT INTO backup_metadata(backup_time,bundle_path) VALUES('%s','%s');`,
		esc(now), esc(bundlePath))
	return d.Exec(sql)
}
