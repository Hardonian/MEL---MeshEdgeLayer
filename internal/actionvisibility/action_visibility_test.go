package actionvisibility

import (
	"testing"

	"github.com/mel-project/mel/internal/models"
)

func TestFromIncident_linkedObserved(t *testing.T) {
	inc := models.Incident{
		ID: "i1",
		LinkedControlActions: []models.ActionRecord{
			{ID: "a1", LifecycleState: "pending_approval"},
		},
	}
	p := FromIncident(inc, true)
	if p.Kind != "linked_observed" {
		t.Fatalf("kind=%q", p.Kind)
	}
	if !p.SuggestControlQueue {
		t.Fatal("expected suggest queue for approval wait")
	}
	if p.LinkedRowCount != 1 {
		t.Fatalf("count=%d", p.LinkedRowCount)
	}
}

func TestFromIncident_visibilityLimited(t *testing.T) {
	inc := models.Incident{
		ID:             "i1",
		PendingActions: []string{"x"},
	}
	p := FromIncident(inc, false)
	if p.Kind != "visibility_limited" {
		t.Fatalf("kind=%q", p.Kind)
	}
	if p.Reason != "capability_limited" {
		t.Fatalf("reason=%q", p.Reason)
	}
	if p.PendingRefCount != 1 {
		t.Fatalf("pending refs=%d", p.PendingRefCount)
	}
}

func TestFromIncident_referencesOnly(t *testing.T) {
	inc := models.Incident{
		ID:             "i1",
		PendingActions: []string{"act-1"},
	}
	p := FromIncident(inc, true)
	if p.Kind != "references_only" {
		t.Fatalf("kind=%q", p.Kind)
	}
	if p.Reason != "partial_payload_only" {
		t.Fatalf("reason=%q", p.Reason)
	}
}

func TestFromIncident_actionContextDegraded(t *testing.T) {
	inc := models.Incident{
		ID: "i1",
		Intelligence: &models.IncidentIntelligence{
			EvidenceStrength: "sparse",
			ActionOutcomeTrace: &models.IncidentActionOutcomeTrace{
				SnapshotRetrievalStatus: "error",
				Completeness:            "partial",
			},
		},
	}
	p := FromIncident(inc, true)
	if p.Kind != "action_context_degraded" {
		t.Fatalf("kind=%q", p.Kind)
	}
}
