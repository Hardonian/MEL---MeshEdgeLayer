package incidenttriage

import (
	"testing"

	"github.com/mel-project/mel/internal/models"
)

func TestComputeForIncident_FollowUpShortCircuits(t *testing.T) {
	inc := models.Incident{
		ID:          "a",
		State:       "open",
		ReviewState: "follow_up_needed",
		Intelligence: &models.IncidentIntelligence{
			SignatureMatchCount: 99,
		},
	}
	sig := ComputeForIncident(inc)
	if sig.Tier != 0 {
		t.Fatalf("tier %d want 0", sig.Tier)
	}
	if len(sig.Codes) != 1 || sig.Codes[0] != "explicit_follow_up_review" {
		t.Fatalf("codes: %#v", sig.Codes)
	}
}

func TestComputeForIncident_GovernanceFriction(t *testing.T) {
	inc := models.Incident{
		ID:    "b",
		State: "open",
		ActionVisibility: &models.IncidentActionVisibilityPosture{
			Kind: "linked_observed",
		},
		Intelligence: &models.IncidentIntelligence{
			EvidenceStrength: "moderate",
			GovernanceMemory: []models.IncidentGovernanceMemory{
				{ActionType: "restart", RejectedCount: 2, LinkedActionCount: 2},
			},
		},
	}
	sig := ComputeForIncident(inc)
	found := false
	for _, c := range sig.Codes {
		if c == "governance_friction_memory" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected governance_friction_memory in %#v", sig.Codes)
	}
}
