package db

import (
	"fmt"
	"time"
)

const (
	MetaBootConfigFingerprint = "boot_config_fingerprint"
	MetaBootConfigPath        = "boot_config_path"
	MetaBootAt                = "boot_at"
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
