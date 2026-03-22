package db

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mel-project/mel/internal/version"
)

// HighestMigrationNumeric returns the maximum leading numeric prefix among applied migrations.
func (d *DB) HighestMigrationNumeric() (int, error) {
	s, err := d.Scalar(`SELECT COALESCE(MAX(CAST(substr(version,1,4) AS INTEGER)), 0) FROM schema_migrations;`)
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, fmt.Errorf("parse migration level: %w", err)
	}
	return n, nil
}

// RequireSchemaCompatibleWithBinary refuses startup when SQLite migrations are behind or ahead of this binary.
func (d *DB) RequireSchemaCompatibleWithBinary() error {
	n, err := d.HighestMigrationNumeric()
	if err != nil {
		return err
	}
	req := version.CurrentSchemaVersion
	if n < req {
		return fmt.Errorf("database migrations incomplete: applied level %d, binary requires %d (%s)", n, req, version.DescribeSchemaGap(n, req))
	}
	if n > req {
		return fmt.Errorf("database is newer than this binary: applied level %d, binary expects %d (%s)", n, req, version.DescribeSchemaGap(n, req))
	}
	return nil
}
