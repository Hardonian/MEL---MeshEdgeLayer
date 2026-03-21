package consistency

import (
	"testing"
	"time"

	"github.com/mel-project/mel/internal/kernel"
)

func TestCheckStalenessWithinBounds(t *testing.T) {
	bounds := DefaultBoundedStaleness()
	check := CheckStaleness(100, 95, 200, 195, time.Now().UTC().Add(-30*time.Second), bounds)
	if check.IsStale {
		t.Errorf("expected not stale, got stale: %v", check.StaleReasons)
	}
}

func TestCheckStalenessClockDrift(t *testing.T) {
	bounds := DefaultBoundedStaleness()
	check := CheckStaleness(5000, 100, 200, 195, time.Now().UTC(), bounds)
	if !check.IsStale {
		t.Error("expected stale due to clock drift")
	}
	found := false
	for _, r := range check.StaleReasons {
		if len(r) > 0 {
			found = true
		}
	}
	if !found {
		t.Error("expected stale reason")
	}
}

func TestCheckStalenessSequenceLag(t *testing.T) {
	bounds := DefaultBoundedStaleness()
	check := CheckStaleness(100, 100, 10000, 0, time.Now().UTC(), bounds)
	if !check.IsStale {
		t.Error("expected stale due to sequence lag")
	}
}

func TestCheckStalenessTimeDrift(t *testing.T) {
	bounds := DefaultBoundedStaleness()
	check := CheckStaleness(100, 100, 200, 200, time.Now().UTC().Add(-10*time.Minute), bounds)
	if !check.IsStale {
		t.Error("expected stale due to time drift")
	}
}

func TestCompareAndResolveIdenticalStates(t *testing.T) {
	state := buildTestState()
	divergences, resolved := CompareAndResolve(state, state)
	if len(divergences) != 0 {
		t.Errorf("expected 0 divergences for identical states, got %d", len(divergences))
	}
	if len(resolved.NodeScores) != len(state.NodeScores) {
		t.Errorf("resolved state should have same node count")
	}
}

func TestCompareAndResolveNodeScoreDominance(t *testing.T) {
	local := buildTestState()
	remote := buildTestState()

	// Remote has worse score for node-1
	ns := remote.NodeScores["node-1"]
	ns.CompositeScore = 0.2
	ns.HealthScore = 0.1
	ns.Classification = "failing"
	remote.NodeScores["node-1"] = ns

	divergences, resolved := CompareAndResolve(local, remote)

	// Should detect divergence
	found := false
	for _, d := range divergences {
		if d.Key == "node-1.classification" {
			found = true
			if d.Strategy != StrategyScoreDominance {
				t.Errorf("expected score_dominance strategy, got %s", d.Strategy)
			}
			if d.Level != DivergenceMajor {
				t.Errorf("expected major divergence, got %s", d.Level)
			}
		}
	}
	if !found {
		t.Error("expected divergence for node-1 classification")
	}

	// Resolved state should have the worse score
	resolvedScore := resolved.NodeScores["node-1"]
	if resolvedScore.Classification != "failing" {
		t.Errorf("expected resolved classification 'failing', got '%s'", resolvedScore.Classification)
	}
	if resolvedScore.CompositeScore != 0.2 {
		t.Errorf("expected resolved composite 0.2, got %.2f", resolvedScore.CompositeScore)
	}
}

func TestCompareAndResolveTransportScoreDominance(t *testing.T) {
	local := buildTestState()
	remote := buildTestState()

	ts := remote.TransportScores["mqtt"]
	ts.HealthScore = 0.1
	ts.Classification = "dead"
	remote.TransportScores["mqtt"] = ts

	divergences, resolved := CompareAndResolve(local, remote)

	found := false
	for _, d := range divergences {
		if d.Key == "mqtt.classification" {
			found = true
		}
	}
	if !found {
		t.Error("expected transport divergence")
	}

	if resolved.TransportScores["mqtt"].Classification != "dead" {
		t.Error("expected resolved transport to take worse classification")
	}
}

func TestCompareAndResolveActionLifecycleAdvancement(t *testing.T) {
	local := buildTestState()
	remote := buildTestState()

	// Remote has more advanced lifecycle
	as := remote.ActionStates["action-1"]
	as.Lifecycle = "completed"
	remote.ActionStates["action-1"] = as

	divergences, resolved := CompareAndResolve(local, remote)

	found := false
	for _, d := range divergences {
		if d.Key == "action-1.lifecycle" {
			found = true
			if d.Level != DivergenceCritical {
				t.Errorf("expected critical divergence for action lifecycle, got %s", d.Level)
			}
		}
	}
	if !found {
		t.Error("expected action lifecycle divergence")
	}

	if resolved.ActionStates["action-1"].Lifecycle != "completed" {
		t.Errorf("expected completed lifecycle, got %s", resolved.ActionStates["action-1"].Lifecycle)
	}
}

func TestCompareAndResolveFreezeUnionMerge(t *testing.T) {
	local := buildTestState()
	remote := buildTestState()

	// Remote has an additional freeze
	remote.ActiveFreezes["frz-remote-1"] = kernel.FreezeState{
		FreezeID:   "frz-remote-1",
		ScopeType:  "transport",
		ScopeValue: "serial",
		Reason:     "remote maintenance",
		CreatedAt:  time.Now().UTC(),
	}

	divergences, resolved := CompareAndResolve(local, remote)

	// Must find the freeze divergence
	found := false
	for _, d := range divergences {
		if d.Key == "frz-remote-1" {
			found = true
			if d.Strategy != StrategyUnionMerge {
				t.Errorf("expected union_merge strategy, got %s", d.Strategy)
			}
			if d.Level != DivergenceCritical {
				t.Errorf("expected critical level for freeze, got %s", d.Level)
			}
		}
	}
	if !found {
		t.Error("expected freeze divergence")
	}

	// Resolved must have both freezes
	if _, ok := resolved.ActiveFreezes["frz-local-1"]; !ok {
		t.Error("resolved should keep local freeze")
	}
	if _, ok := resolved.ActiveFreezes["frz-remote-1"]; !ok {
		t.Error("resolved should add remote freeze (union)")
	}
}

func TestCompareAndResolvePolicyPrecedence(t *testing.T) {
	local := buildTestState()
	remote := buildTestState()

	remote.PolicyVersion = "v3"

	divergences, resolved := CompareAndResolve(local, remote)

	found := false
	for _, d := range divergences {
		if d.Key == "policy_version" {
			found = true
			if d.Strategy != StrategyPolicyPrecedence {
				t.Errorf("expected policy_precedence strategy, got %s", d.Strategy)
			}
		}
	}
	if !found {
		t.Error("expected policy divergence")
	}

	if resolved.PolicyVersion != "v3" {
		t.Errorf("expected resolved policy v3, got %s", resolved.PolicyVersion)
	}
}

func TestCompareAndResolveNewNodeFromRemote(t *testing.T) {
	local := buildTestState()
	remote := buildTestState()

	// Remote has a node local doesn't know about
	remote.NodeScores["node-new"] = kernel.NodeScore{
		NodeID:         "node-new",
		HealthScore:    0.9,
		CompositeScore: 0.85,
		Classification: "healthy",
	}

	_, resolved := CompareAndResolve(local, remote)

	if _, ok := resolved.NodeScores["node-new"]; !ok {
		t.Error("resolved should include new node from remote via union merge")
	}
}

func TestCompareAndResolveLogicalClockMax(t *testing.T) {
	local := buildTestState()
	remote := buildTestState()

	local.LogicalClock = 50
	remote.LogicalClock = 200

	_, resolved := CompareAndResolve(local, remote)

	if resolved.LogicalClock != 200 {
		t.Errorf("expected resolved clock 200, got %d", resolved.LogicalClock)
	}
}

func TestCheckConvergenceConverged(t *testing.T) {
	state := buildTestState()
	report := CheckConvergence(state, state)
	if !report.Converged {
		t.Error("identical states should be converged")
	}
	if report.CriticalCount != 0 || report.MajorCount != 0 {
		t.Error("no critical or major divergences expected")
	}
}

func TestCheckConvergenceNotConverged(t *testing.T) {
	local := buildTestState()
	remote := buildTestState()

	// Make a critical divergence
	as := remote.ActionStates["action-1"]
	as.Lifecycle = "completed"
	remote.ActionStates["action-1"] = as

	report := CheckConvergence(local, remote)
	if report.Converged {
		t.Error("states with action lifecycle divergence should not be converged")
	}
	if report.CriticalCount == 0 {
		t.Error("expected critical divergence count > 0")
	}
}

func TestLifecycleOrder(t *testing.T) {
	tests := []struct {
		lifecycle string
		order     int
	}{
		{"proposed", 1},
		{"approved", 2},
		{"running", 3},
		{"completed", 4},
		{"rejected", 4},
		{"unknown", 0},
	}

	for _, tt := range tests {
		if got := lifecycleOrder(tt.lifecycle); got != tt.order {
			t.Errorf("lifecycleOrder(%q) = %d, want %d", tt.lifecycle, got, tt.order)
		}
	}
}

func TestCompareAndResolveRegistryLWW(t *testing.T) {
	local := buildTestState()
	remote := buildTestState()

	// Remote has newer last_seen for node 100
	info := remote.NodeRegistry[100]
	info.LastSeen = time.Now().UTC().Add(time.Hour)
	info.LongName = "Updated Name"
	remote.NodeRegistry[100] = info

	_, resolved := CompareAndResolve(local, remote)

	if resolved.NodeRegistry[100].LongName != "Updated Name" {
		t.Error("expected registry to take remote with newer last_seen")
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func buildTestState() *kernel.State {
	now := time.Now().UTC()
	s := kernel.NewState()
	s.PolicyVersion = "v1"
	s.LogicalClock = 100
	s.LastSequenceNum = 50

	s.NodeScores["node-1"] = kernel.NodeScore{
		NodeID:         "node-1",
		HealthScore:    0.9,
		TrustScore:     0.8,
		ActivityScore:  0.7,
		AnomalyScore:   0.1,
		CompositeScore: 0.8,
		Classification: "healthy",
		UpdatedAt:      now,
	}
	s.NodeScores["node-2"] = kernel.NodeScore{
		NodeID:         "node-2",
		HealthScore:    0.5,
		CompositeScore: 0.5,
		Classification: "degraded",
		UpdatedAt:      now,
	}

	s.TransportScores["mqtt"] = kernel.TransportScore{
		Transport:      "mqtt",
		HealthScore:    0.85,
		Classification: "healthy",
		UpdatedAt:      now,
	}

	s.ActionStates["action-1"] = kernel.ActionState{
		ActionID:   "action-1",
		ActionType: "restart_transport",
		Target:     "mqtt",
		Lifecycle:  "proposed",
		ProposedAt: now,
	}

	s.ActiveFreezes["frz-local-1"] = kernel.FreezeState{
		FreezeID:   "frz-local-1",
		ScopeType:  "transport",
		ScopeValue: "mqtt",
		Reason:     "local maintenance",
		CreatedAt:  now,
	}

	s.NodeRegistry[100] = kernel.NodeInfo{
		NodeNum:  100,
		NodeID:   "node-1",
		LongName: "Node 1",
		LastSeen: now,
	}

	return s
}
