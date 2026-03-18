package privacy

import (
	"testing"

	"github.com/mel-project/mel/internal/config"
)

func TestAuditFindings(t *testing.T) {
	cfg := config.Default()
	cfg.Bind.AllowRemote = true
	cfg.Privacy.MapReportingAllowed = true
	cfg.Privacy.RedactExports = false
	findings := Audit(cfg)
	if len(findings) < 3 {
		t.Fatalf("expected findings, got %d", len(findings))
	}
	if findings[0].Severity == "" {
		t.Fatal("expected severity ordering")
	}
}

func TestSummary(t *testing.T) {
	s := Summary([]Finding{{Severity: "critical"}, {Severity: "high"}, {Severity: "high"}})
	if s["high"] != 2 || s["critical"] != 1 {
		t.Fatalf("unexpected summary: %#v", s)
	}
}
