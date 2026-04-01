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

func TestComputeForIncident_QueueOrderingFilled(t *testing.T) {
	inc := models.Incident{
		ID:          "c",
		State:       "open",
		ReviewState: "investigating",
		UpdatedAt:   "2020-01-01T00:00:00Z",
	}
	sig := ComputeForIncident(inc)
	if sig.QueueOrderingContract != "open_incident_workbench_v1" {
		t.Fatalf("contract %q", sig.QueueOrderingContract)
	}
	if sig.QueueSortPrimary != sig.Tier {
		t.Fatalf("sort primary %d tier %d", sig.QueueSortPrimary, sig.Tier)
	}
	if sig.OrderingUncertainty == "" {
		t.Fatal("expected ordering uncertainty")
	}
}

func TestComputeForIncident_MeshRoutingCompanionStress(t *testing.T) {
	inc := models.Incident{
		ID:          "d",
		State:       "open",
		ReviewState: "investigating",
		Intelligence: &models.IncidentIntelligence{
			EvidenceStrength: "moderate",
			MeshRoutingCompanion: &models.MeshRoutingIntelCompanion{
				Applicable:                     true,
				TopologyEnabled:                true,
				TransportConnected:             true,
				EvidenceModel:                  "ingest_graph_v1",
				AssessmentComputedAt:           "2020-01-01T00:00:00Z",
				SuspectedRelayHotspot:          true,
				WeakOnwardPropagationSuspected: false,
				HopBudgetStressSuspected:       false,
			},
		},
	}
	sig := ComputeForIncident(inc)
	found := false
	for _, c := range sig.Codes {
		if c == "mesh_routing_pressure_companion" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected mesh_routing_pressure_companion in %#v", sig.Codes)
	}
	if sig.Tier > 2 {
		t.Fatalf("expected tier <=2 for mesh stress, got %d", sig.Tier)
	}
}
