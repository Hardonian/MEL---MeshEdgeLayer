package service

import (
	"testing"
	"time"

	"github.com/mel-project/mel/internal/models"
)

func TestIncidentLearningFixture_IncludesFingerprintHash(t *testing.T) {
	a := newSoDTestApp(t)
	id := "inc-fix-1"
	if err := a.DB.UpsertIncident(models.Incident{
		ID:           id,
		Category:     "transport",
		Severity:     "warning",
		Title:        "t",
		Summary:      "s",
		ResourceType: "transport",
		ResourceID:   "mqtt-sod",
		State:        "open",
		OccurredAt:   time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	fix, err := a.IncidentLearningFixture(id)
	if err != nil {
		t.Fatal(err)
	}
	fp, _ := fix["fingerprint"].(map[string]any)
	if fp == nil || fp["canonical_hash"] == "" {
		t.Fatalf("expected fingerprint canonical_hash in fixture: %#v", fix)
	}
}
