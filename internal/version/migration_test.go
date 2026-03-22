package version

import "testing"

func TestMigrationNumeric(t *testing.T) {
	if got := MigrationNumeric("0019_operational_trust"); got != 19 {
		t.Fatalf("got %d", got)
	}
	if got := MigrationNumeric(""); got != 0 {
		t.Fatalf("empty: got %d", got)
	}
}
