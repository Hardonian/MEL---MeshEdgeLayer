package service

import (
	"testing"
	"time"

	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/models"
)

func TestMeshRoutingCompanion_MeshTopologyIncident(t *testing.T) {
	a := newSoDTestApp(t)
	a.Cfg.Topology.Enabled = true
	a.meshIntelMu.Lock()
	a.meshIntelLatest = meshintel.Assessment{
		TopologyEnabled: true,
		ComputedAt:      time.Now().UTC().Format(time.RFC3339),
		GraphHash:       "gh-test",
		MessageSignals: meshintel.MessageSignals{
			WindowDescription:      "test_window",
			TotalMessages:          50,
			MessagesWithHop:        10,
			DuplicateRelayHotspot:  0.6,
			TransportConnected:     true,
			DistinctRelayNodes:     3,
			RelayMaxShare:          0.4,
		},
		RoutingPressure: meshintel.RoutingPressureBundle{
			SummaryLines: []string{"proxy line"},
			DuplicateForwardPressureScore: meshintel.ScoredMetric{
				Name: "duplicate_forward_pressure", Score: 0.55, Scale: "0_1",
			},
			WeakOnwardPropagationScore: meshintel.ScoredMetric{
				Name: "weak_onward_propagation", Score: 0.6, Scale: "0_1",
			},
			HopBudgetStressScore: meshintel.ScoredMetric{
				Name: "hop_budget_stress", Score: 0.6, Scale: "0_1",
			},
		},
		EvidenceModel: "test",
	}
	a.meshIntelHas = true
	a.meshIntelMu.Unlock()

	inc := models.Incident{
		ID:           "inc-mesh-1",
		Category:     "mesh_topology",
		Severity:     "warning",
		Title:        "Topo churn",
		ResourceType: "mesh",
		ResourceID:   "n1",
		State:        "open",
		OccurredAt:   time.Now().UTC().Format(time.RFC3339),
	}
	intel := &models.IncidentIntelligence{EvidenceStrength: "sparse"}
	c := a.meshRoutingCompanionForIncident(inc, intel)
	if c == nil || !c.Applicable {
		t.Fatalf("expected applicable companion, got %+v", c)
	}
	if !c.SuspectedRelayHotspot {
		t.Error("expected suspected relay hotspot")
	}
	if c.SuggestedTopologySearch == "" {
		t.Fatal("expected topology search string")
	}
}

func TestOperatorSuggestedActions_IncludesReplayAndTopology(t *testing.T) {
	a := newSoDTestApp(t)
	a.Cfg.Topology.Enabled = true
	inc := models.Incident{
		ID:           "inc-sug-1",
		Category:     "mesh_topology",
		ResourceType: "mesh",
		State:        "open",
		OccurredAt:   time.Now().UTC().Format(time.RFC3339),
	}
	intel := &models.IncidentIntelligence{
		EvidenceStrength: "sparse",
		MeshRoutingCompanion: &models.MeshRoutingIntelCompanion{
			Applicable:              true,
			SuggestedTopologySearch: "incident=inc-sug-1&filter=incident_focus",
		},
	}
	acts := operatorSuggestedActions(a.Cfg, inc, intel)
	var hasReplay, hasTopo bool
	for _, x := range acts {
		if x.ID == "inspect-replay" {
			hasReplay = true
		}
		if x.ID == "inspect-topology" {
			hasTopo = true
			if x.Href == "" || x.Href == "/topology" {
				t.Errorf("topology href should include query, got %q", x.Href)
			}
		}
	}
	if !hasReplay || !hasTopo {
		t.Fatalf("expected replay and topology actions, got %#v", acts)
	}
}
