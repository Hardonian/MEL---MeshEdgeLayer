package investigation

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/transport"
)

func TestDeriveBuildsCanonicalCasesWithExplicitBoundaries(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Scope.SiteID = "site-a"
	cfg.Scope.ExpectedFleetReporterCount = 3
	cfg.Transports = []config.TransportConfig{{
		Name:     "mqtt",
		Type:     "mqtt",
		Enabled:  true,
		Endpoint: "127.0.0.1:1883",
		Topic:    "msh/test",
		ClientID: "mel-test",
	}}

	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.UpsertTransportRuntime(db.TransportRuntime{
		Name:          "mqtt",
		Type:          "mqtt",
		Source:        "127.0.0.1:1883",
		Enabled:       true,
		State:         transport.StateFailed,
		Detail:        "connect failed",
		LastError:     "broker unreachable",
		FailureCount:  3,
		UpdatedAt:     "2026-03-27T00:00:00Z",
		LastFailureAt: "2026-03-27T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	runtimeStates, err := database.TransportRuntimeStatuses()
	if err != nil {
		t.Fatal(err)
	}
	summary := Derive(cfg, database, []transport.Health{{
		Name:         "mqtt",
		Type:         "mqtt",
		Source:       "127.0.0.1:1883",
		State:        transport.StateFailed,
		LastError:    "broker unreachable",
		FailureCount: 3,
	}}, runtimeStates, time.Date(2026, 3, 27, 1, 0, 0, 0, time.UTC))

	transportCase := findCaseByKind(summary.Cases, CaseTransportDegradation)
	if transportCase == nil {
		t.Fatal("expected transport degradation case")
	}
	if !strings.Contains(transportCase.OutOfScope, "Do not infer fleet-wide outage") {
		t.Fatalf("expected bounded out-of-scope guidance, got %q", transportCase.OutOfScope)
	}
	if len(transportCase.RelatedRecords) == 0 {
		t.Fatal("expected related records on transport case")
	}

	fleetCase := findCaseByKind(summary.Cases, CasePartialFleetVisibility)
	if fleetCase == nil {
		t.Fatal("expected partial fleet visibility case")
	}
	if !strings.Contains(fleetCase.OutOfScope, "Do not claim fleet-wide health") {
		t.Fatalf("expected no-fake-fleet-certainty guidance, got %q", fleetCase.OutOfScope)
	}
	if summary.CaseCounts.ActiveAttentionCases == 0 {
		t.Fatalf("expected active attention case counts, got %+v", summary.CaseCounts)
	}
}

func findCaseByKind(cases []Case, kind CaseKind) *Case {
	for i := range cases {
		if cases[i].Kind == kind {
			return &cases[i]
		}
	}
	return nil
}
