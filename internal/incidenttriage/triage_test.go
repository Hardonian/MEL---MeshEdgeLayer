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
	if sig.QueueOrderingContract != "open_incident_workbench_v2" {
		t.Fatalf("contract %q", sig.QueueOrderingContract)
	}
	if sig.QueueOrderingContractVersion != "2" {
		t.Fatalf("version %q", sig.QueueOrderingContractVersion)
	}
	if sig.QueueSortPrimary != sig.Tier {
		t.Fatalf("sort primary %d tier %d", sig.QueueSortPrimary, sig.Tier)
	}
	if sig.QueueSortSecondaryValidity != "valid_rfc3339" {
		t.Fatalf("validity %q", sig.QueueSortSecondaryValidity)
	}
	if len(sig.QueueSortTuple) != 3 {
		t.Fatalf("tuple len %d", len(sig.QueueSortTuple))
	}
	if sig.OrderingUncertainty == "" {
		t.Fatal("expected ordering uncertainty")
	}
	if sig.QueueOrderingPosture != "canonical_v2" {
		t.Fatalf("posture %q want canonical_v2", sig.QueueOrderingPosture)
	}
}

func TestQueueOrderingDeterministicAcrossEqualTier(t *testing.T) {
	a := models.Incident{ID: "zzz", State: "open", ReviewState: "investigating", UpdatedAt: "2020-01-01T00:00:00Z"}
	b := models.Incident{ID: "aaa", State: "open", ReviewState: "investigating", UpdatedAt: "2020-01-01T00:00:00Z"}
	sa := ComputeForIncident(a)
	sb := ComputeForIncident(b)
	if sa.QueueSortTuple[0] != sb.QueueSortTuple[0] || sa.QueueSortTuple[1] != sb.QueueSortTuple[1] {
		t.Fatalf("expected same tier and recency, got %v vs %v", sa.QueueSortTuple, sb.QueueSortTuple)
	}
	if sa.QueueSortTuple[2] == sb.QueueSortTuple[2] {
		t.Fatalf("expected different tie-break for different ids")
	}
	same := ComputeForIncident(a)
	if sa.QueueSortTuple[2] != same.QueueSortTuple[2] {
		t.Fatalf("tie-break not stable for same incident id")
	}
}

func TestQueueOrderingMissingUpdatedAt(t *testing.T) {
	inc := models.Incident{ID: "x", State: "open", ReviewState: "investigating"}
	sig := ComputeForIncident(inc)
	if sig.QueueSortSecondaryValidity != "missing" {
		t.Fatalf("want missing, got %q", sig.QueueSortSecondaryValidity)
	}
	if sig.QueueSortSecondaryNumeric != 0 {
		t.Fatalf("want 0 ns, got %d", sig.QueueSortSecondaryNumeric)
	}
	if sig.QueueSortTuple[1] != 9223372036854775807 { // math.MaxInt64
		t.Fatalf("missing recency should use MaxInt64 inverted rank, got %d", sig.QueueSortTuple[1])
	}
	if sig.QueueOrderingPosture != "degraded_partial_recency" {
		t.Fatalf("posture %q", sig.QueueOrderingPosture)
	}
	if len(sig.QueueOrderingDegradedReasons) == 0 {
		t.Fatal("expected degraded reasons")
	}
}

func TestQueueSortKeyLexMoreRecentFirst(t *testing.T) {
	old := models.Incident{ID: "a", State: "open", ReviewState: "investigating", UpdatedAt: "2020-01-01T00:00:00Z"}
	newer := models.Incident{ID: "b", State: "open", ReviewState: "investigating", UpdatedAt: "2021-06-15T12:00:00Z"}
	so := ComputeForIncident(old)
	sn := ComputeForIncident(newer)
	if sn.QueueSortKeyLex >= so.QueueSortKeyLex {
		t.Fatalf("newer incident should sort before older: %q vs %q", sn.QueueSortKeyLex, so.QueueSortKeyLex)
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

func TestComputeForIncident_ReplayWorseningRaisesTier(t *testing.T) {
	inc := models.Incident{
		ID:    "replay-worsening",
		State: "open",
		ReplaySummary: &models.IncidentReplaySummary{
			Semantic:        "active_changing",
			HistoryPattern:  "worsening",
			Comparability:   "comparable",
			NeedsAttention:  true,
			AttentionReason: "history_worsening",
			Summary:         "Replay posture active changing in bounded window.",
		},
	}
	sig := ComputeForIncident(inc)
	if sig.Tier > 2 {
		t.Fatalf("expected tier <=2 for worsening replay, got %d", sig.Tier)
	}
	found := false
	for _, c := range sig.Codes {
		if c == "replay_history_worsening" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected replay_history_worsening in codes: %#v", sig.Codes)
	}
}

func TestComputeForIncident_ReplayNotComparableAddsUncertainty(t *testing.T) {
	inc := models.Incident{
		ID:    "replay-thin",
		State: "open",
		ReplaySummary: &models.IncidentReplaySummary{
			Semantic:        "sparse",
			HistoryPattern:  "thin_history",
			Comparability:   "not_comparable",
			NeedsAttention:  true,
			AttentionReason: "history_thin",
			NotComparable:   []string{"insufficient_replay_rows"},
		},
	}
	sig := ComputeForIncident(inc)
	if sig.Tier > 2 {
		t.Fatalf("expected tier <=2 for not_comparable replay, got %d", sig.Tier)
	}
	hasUncertainty := false
	for _, u := range sig.UncertaintyNotes {
		if u != "" {
			hasUncertainty = true
			break
		}
	}
	if !hasUncertainty {
		t.Fatalf("expected uncertainty notes for replay not comparable: %#v", sig.UncertaintyNotes)
	}
}
