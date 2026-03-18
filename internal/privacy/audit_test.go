package privacy

import (
	"github.com/mel-project/mel/internal/config"
	"testing"
)

func TestAuditFindings(t *testing.T) {
	cfg := config.Default()
	cfg.Bind.AllowRemote = true
	cfg.Privacy.MapReportingAllowed = true
	findings := Audit(cfg)
	if len(findings) < 2 {
		t.Fatalf("expected findings, got %d", len(findings))
	}
}
