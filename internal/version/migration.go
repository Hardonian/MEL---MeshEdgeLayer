package version

import (
	"fmt"
	"strconv"
	"strings"
)

// MigrationNumeric extracts the leading migration number from a schema_migrations.version
// value such as "0019_operational_trust" → 19.
func MigrationNumeric(versionID string) int {
	versionID = strings.TrimSpace(versionID)
	if versionID == "" {
		return 0
	}
	var prefix strings.Builder
	for _, r := range versionID {
		if r >= '0' && r <= '9' {
			prefix.WriteRune(r)
			continue
		}
		break
	}
	if prefix.Len() == 0 {
		return 0
	}
	n, err := strconv.Atoi(prefix.String())
	if err != nil {
		return 0
	}
	return n
}

// DescribeSchemaGap returns a human-readable comparison of DB vs binary schema expectations.
func DescribeSchemaGap(dbNumeric, required int) string {
	if dbNumeric < required {
		return fmt.Sprintf("database schema is behind this binary: db migration number %d, binary expects %d", dbNumeric, required)
	}
	if dbNumeric > required {
		return fmt.Sprintf("database schema is newer than this binary: db migration number %d, binary expects %d", dbNumeric, required)
	}
	return "schema migration level matches binary"
}
