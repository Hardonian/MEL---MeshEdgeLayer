package db

import (
	"path/filepath"
	"testing"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/models"
)

func TestRecommendationOutcomes_InsertAndList(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	inc := models.Incident{
		ID:           "inc-a",
		Category:     "transport",
		Severity:     "high",
		Title:        "t",
		Summary:      "s",
		ResourceType: "transport",
		ResourceID:   "mqtt-a",
		State:        "open",
		OccurredAt:   "2026-03-29T10:00:00Z",
	}
	if err := d.UpsertIncident(inc); err != nil {
		t.Fatal(err)
	}
	rec := IncidentRecommendationOutcomeRecord{
		ID:               "iro-test-1",
		IncidentID:       "inc-a",
		RecommendationID: "guide-incident-record",
		Outcome:          "accepted",
		ActorID:          "op-1",
		Note:             "ok",
		CreatedAt:        "2026-03-29T12:00:00Z",
	}
	if err := d.InsertIncidentRecommendationOutcome(rec); err != nil {
		t.Fatal(err)
	}
	rows, err := d.RecommendationOutcomesForIncident("inc-a", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Outcome != "accepted" {
		t.Fatalf("got %+v", rows)
	}
}

func TestIntelSignalOutcomes_LatestByCode(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	inc := models.Incident{
		ID:           "inc-b",
		Category:     "transport",
		Severity:     "high",
		Title:        "t",
		Summary:      "s",
		ResourceType: "transport",
		ResourceID:   "mqtt-a",
		State:        "open",
		OccurredAt:   "2026-03-29T10:00:00Z",
	}
	if err := d.UpsertIncident(inc); err != nil {
		t.Fatal(err)
	}
	if err := d.InsertIncidentIntelSignalOutcome(IncidentIntelSignalOutcomeRecord{
		ID:         "iso-1",
		IncidentID: "inc-b",
		SignalCode: "evidence_thin_review_needed",
		Outcome:    "dismissed",
		ActorID:    "op-1",
		CreatedAt:  "2026-03-29T12:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	if err := d.InsertIncidentIntelSignalOutcome(IncidentIntelSignalOutcomeRecord{
		ID:         "iso-2",
		IncidentID: "inc-b",
		SignalCode: "evidence_thin_review_needed",
		Outcome:    "reviewed",
		ActorID:    "op-1",
		CreatedAt:  "2026-03-29T13:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	m, err := d.LatestIntelSignalOutcomesByIncident("inc-b")
	if err != nil {
		t.Fatal(err)
	}
	if m["evidence_thin_review_needed"].Outcome != "reviewed" {
		t.Fatalf("want latest reviewed, got %+v", m)
	}
}

func TestCorrelationGroup_SignatureLink(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	base := models.Incident{
		Category:     "transport",
		Severity:     "high",
		Title:        "t",
		Summary:      "s",
		ResourceType: "transport",
		ResourceID:   "mqtt-a",
		State:        "open",
		OccurredAt:   "2026-03-29T10:00:00Z",
	}
	for _, id := range []string{"inc-1", "inc-2"} {
		inc := base
		inc.ID = id
		if err := d.UpsertIncident(inc); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.LinkIncidentToSignature("sig-test", "inc-1"); err != nil {
		t.Fatal(err)
	}
	if err := d.LinkIncidentToSignature("sig-test", "inc-2"); err != nil {
		t.Fatal(err)
	}
	groups, err := d.CorrelationGroupsForIncident("inc-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	ids, err := d.CorrelatedIncidentIDsForGroup(groups[0].GroupID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("members %+v", ids)
	}
}
