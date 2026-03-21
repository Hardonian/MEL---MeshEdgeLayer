package kernel

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewKernel(t *testing.T) {
	k := New("mel-test-001", Policy{
		Version:              "v1",
		Mode:                 "advisory",
		AllowedActions:       []string{"restart_transport", "trigger_health_recheck"},
		RequireMinConfidence: 0.75,
		MaxActionsPerWindow:  8,
	})

	if k.NodeID() != "mel-test-001" {
		t.Errorf("expected node ID mel-test-001, got %s", k.NodeID())
	}

	state := k.State()
	if len(state.NodeScores) != 0 {
		t.Errorf("expected empty node scores, got %d", len(state.NodeScores))
	}
}

func TestApplyObservation(t *testing.T) {
	k := New("mel-test-001", Policy{
		Version: "v1",
		Mode:    "advisory",
	})

	obs := ObservationData{
		Transport:   "mqtt-local",
		NodeNum:     12345,
		NodeID:      "!abcdef",
		MessageType: "text",
	}
	obsJSON, _ := json.Marshal(obs)

	evt := Event{
		ID:           NewEventID(),
		Type:         EventObservation,
		Timestamp:    time.Now().UTC(),
		SourceNodeID: "mel-test-001",
		Subject:      "mqtt-local",
		Data:         obsJSON,
	}

	effects := k.Apply(evt)

	state := k.State()

	// Node should be registered
	if _, ok := state.NodeRegistry[12345]; !ok {
		t.Error("expected node 12345 in registry")
	}

	// Transport score should exist
	ts, ok := state.TransportScores["mqtt-local"]
	if !ok {
		t.Error("expected transport score for mqtt-local")
	}
	if ts.HealthScore <= 0 {
		t.Error("expected positive health score")
	}

	// Node score should exist with effects
	ns, ok := state.NodeScores["!abcdef"]
	if !ok {
		t.Error("expected node score for !abcdef")
	}
	if ns.ActivityScore <= 0 {
		t.Error("expected positive activity score")
	}

	// Should have produced score update effect
	found := false
	for _, eff := range effects {
		if eff.Type == EffectUpdateScore {
			found = true
		}
	}
	if !found {
		t.Error("expected EffectUpdateScore effect")
	}

	// Logical clock should have advanced
	if state.LogicalClock == 0 {
		t.Error("expected logical clock > 0")
	}
}

func TestApplyAnomaly(t *testing.T) {
	k := New("mel-test-001", Policy{
		Version:              "v1",
		Mode:                 "guarded_auto",
		AllowedActions:       []string{"trigger_health_recheck"},
		RequireMinConfidence: 0.5,
	})

	// First establish a transport
	obs := ObservationData{Transport: "mqtt-local", NodeNum: 100, NodeID: "!node1"}
	obsJSON, _ := json.Marshal(obs)
	k.Apply(Event{
		ID: NewEventID(), Type: EventObservation, Timestamp: time.Now().UTC(),
		SourceNodeID: "mel-test-001", Data: obsJSON,
	})

	// Now send anomaly to degrade health severely
	for i := 0; i < 20; i++ {
		anomaly := AnomalyData{
			Transport: "mqtt-local",
			NodeID:    "!node1",
			Category:  "dead_letter",
			Severity:  "high",
			Score:     0.9,
		}
		anomalyJSON, _ := json.Marshal(anomaly)
		k.Apply(Event{
			ID: NewEventID(), Type: EventAnomaly, Timestamp: time.Now().UTC(),
			SourceNodeID: "mel-test-001", Data: anomalyJSON,
		})
	}

	state := k.State()
	ts := state.TransportScores["mqtt-local"]
	if ts.HealthScore >= 0.5 {
		t.Errorf("expected degraded health, got %.2f", ts.HealthScore)
	}
}

func TestDeterministicReplay(t *testing.T) {
	policy := Policy{
		Version:              "v1",
		Mode:                 "advisory",
		AllowedActions:       []string{"restart_transport"},
		RequireMinConfidence: 0.75,
	}

	// Generate a fixed event stream
	events := generateFixedEventStream(50)

	// Run through kernel A
	ka := New("mel-test-a", policy)
	var effectsA []Effect
	for _, evt := range events {
		effs := ka.Apply(evt)
		effectsA = append(effectsA, effs...)
	}
	stateA := ka.State()

	// Run through kernel B (same events, same policy)
	kb := New("mel-test-b", policy)
	var effectsB []Effect
	for _, evt := range events {
		effs := kb.Apply(evt)
		effectsB = append(effectsB, effs...)
	}
	stateB := kb.State()

	// States must match (except nodeID references)
	if len(stateA.NodeScores) != len(stateB.NodeScores) {
		t.Errorf("node scores count mismatch: %d vs %d", len(stateA.NodeScores), len(stateB.NodeScores))
	}

	for id, scoreA := range stateA.NodeScores {
		scoreB, ok := stateB.NodeScores[id]
		if !ok {
			t.Errorf("node %s missing in state B", id)
			continue
		}
		if scoreA.Classification != scoreB.Classification {
			t.Errorf("classification mismatch for %s: %s vs %s", id, scoreA.Classification, scoreB.Classification)
		}
		if absDiff(scoreA.CompositeScore, scoreB.CompositeScore) > 0.0001 {
			t.Errorf("composite score mismatch for %s: %.4f vs %.4f", id, scoreA.CompositeScore, scoreB.CompositeScore)
		}
	}

	if len(stateA.TransportScores) != len(stateB.TransportScores) {
		t.Errorf("transport scores count mismatch: %d vs %d", len(stateA.TransportScores), len(stateB.TransportScores))
	}

	// Effect counts should match
	if len(effectsA) != len(effectsB) {
		t.Errorf("effect count mismatch: %d vs %d", len(effectsA), len(effectsB))
	}
}

func absDiff(a, b float64) float64 {
	d := a - b
	if d < 0 {
		return -d
	}
	return d
}

func TestFreezeLifecycle(t *testing.T) {
	k := New("mel-test-001", Policy{Version: "v1", Mode: "advisory"})

	// Create freeze
	freezeData := FreezeData{
		FreezeID:   "frz-001",
		ScopeType:  "transport",
		ScopeValue: "mqtt-local",
		Reason:     "maintenance",
	}
	data, _ := json.Marshal(freezeData)
	k.Apply(Event{
		ID: NewEventID(), Type: EventFreezeCreated, Timestamp: time.Now().UTC(),
		SourceNodeID: "mel-test-001", Data: data,
	})

	state := k.State()
	if _, ok := state.ActiveFreezes["frz-001"]; !ok {
		t.Error("expected freeze frz-001 in state")
	}

	// Clear freeze
	k.Apply(Event{
		ID: NewEventID(), Type: EventFreezeCleared, Timestamp: time.Now().UTC(),
		SourceNodeID: "mel-test-001", Data: data,
	})

	state = k.State()
	if _, ok := state.ActiveFreezes["frz-001"]; ok {
		t.Error("freeze frz-001 should have been cleared")
	}
}

func TestActionLifecycle(t *testing.T) {
	k := New("mel-test-001", Policy{Version: "v1", Mode: "advisory"})

	// Propose action
	proposed := ActionProposedData{
		ActionID:   "act-001",
		ActionType: "restart_transport",
		Target:     "mqtt-local",
		Reason:     "health degraded",
		Confidence: 0.9,
	}
	data, _ := json.Marshal(proposed)
	k.Apply(Event{
		ID: NewEventID(), Type: EventActionProposed, Timestamp: time.Now().UTC(),
		SourceNodeID: "mel-test-001", Data: data,
	})

	state := k.State()
	if as, ok := state.ActionStates["act-001"]; !ok {
		t.Error("expected action act-001 in state")
	} else if as.Lifecycle != "proposed" {
		t.Errorf("expected lifecycle proposed, got %s", as.Lifecycle)
	}

	// Execute action
	executed := ActionExecutedData{ActionID: "act-001", Result: "started"}
	data, _ = json.Marshal(executed)
	k.Apply(Event{
		ID: NewEventID(), Type: EventActionExecuted, Timestamp: time.Now().UTC(),
		SourceNodeID: "mel-test-001", Data: data,
	})

	state = k.State()
	if as := state.ActionStates["act-001"]; as.Lifecycle != "running" {
		t.Errorf("expected lifecycle running, got %s", as.Lifecycle)
	}

	// Complete action
	completed := ActionCompletedData{ActionID: "act-001", Result: "success"}
	data, _ = json.Marshal(completed)
	k.Apply(Event{
		ID: NewEventID(), Type: EventActionCompleted, Timestamp: time.Now().UTC(),
		SourceNodeID: "mel-test-001", Data: data,
	})

	state = k.State()
	if as := state.ActionStates["act-001"]; as.Lifecycle != "completed" {
		t.Errorf("expected lifecycle completed, got %s", as.Lifecycle)
	}
}

func TestTopologyUpdate(t *testing.T) {
	k := New("mel-test-001", Policy{Version: "v1"})

	// Add node
	topo := TopologyData{
		NodeNum:   42,
		NodeID:    "!node42",
		LongName:  "Test Node 42",
		ShortName: "TN42",
		Action:    "joined",
	}
	data, _ := json.Marshal(topo)
	k.Apply(Event{
		ID: NewEventID(), Type: EventTopologyUpdate, Timestamp: time.Now().UTC(),
		SourceNodeID: "mel-test-001", Data: data,
	})

	state := k.State()
	node, ok := state.NodeRegistry[42]
	if !ok {
		t.Fatal("expected node 42 in registry")
	}
	if node.LongName != "Test Node 42" {
		t.Errorf("expected long name 'Test Node 42', got %s", node.LongName)
	}

	// Remove node
	topo.Action = "left"
	data, _ = json.Marshal(topo)
	k.Apply(Event{
		ID: NewEventID(), Type: EventTopologyUpdate, Timestamp: time.Now().UTC(),
		SourceNodeID: "mel-test-001", Data: data,
	})

	state = k.State()
	if _, ok := state.NodeRegistry[42]; ok {
		t.Error("node 42 should have been removed")
	}
}

func TestLogicalClockAdvancement(t *testing.T) {
	k := New("mel-test-001", Policy{Version: "v1"})

	// Event with higher clock should advance local clock
	evt := Event{
		ID:           NewEventID(),
		Type:         EventNodeState,
		Timestamp:    time.Now().UTC(),
		LogicalClock: 100,
		SourceNodeID: "mel-remote-001",
		Data:         []byte("{}"),
	}
	k.Apply(evt)

	state := k.State()
	if state.LogicalClock <= 100 {
		t.Errorf("expected logical clock > 100, got %d", state.LogicalClock)
	}
}

func TestCoordinationToken(t *testing.T) {
	ac := NewActionCoordinator("mel-node-1", 5*time.Minute)

	// Acquire token
	token, acquired := ac.TryAcquire("action-001")
	if !acquired {
		t.Fatal("expected to acquire token")
	}
	if token.OwnerNodeID != "mel-node-1" {
		t.Errorf("expected owner mel-node-1, got %s", token.OwnerNodeID)
	}

	// Re-acquire same action should succeed (same owner)
	_, acquired = ac.TryAcquire("action-001")
	if !acquired {
		t.Error("expected re-acquire to succeed for same owner")
	}

	// Record remote token
	ac.RecordRemoteToken(CoordinationToken{
		TokenID:     "remote-tok-1",
		ActionID:    "action-002",
		OwnerNodeID: "mel-node-2",
		AcquiredAt:  time.Now().UTC(),
		ExpiresAt:   time.Now().UTC().Add(5 * time.Minute),
	})

	// Try to acquire remote-held action should fail
	_, acquired = ac.TryAcquire("action-002")
	if acquired {
		t.Error("expected acquire to fail for remote-held action")
	}

	// Release and verify
	ac.Release("action-001", true)
	if ac.IsOwned("action-001") {
		t.Error("expected action-001 to not be owned after release")
	}

	// Active tokens
	if ac.ActiveTokens() != 1 { // remote token still active
		t.Errorf("expected 1 active token, got %d", ac.ActiveTokens())
	}

	// Cleanup
	ac.Cleanup()
}

func TestBackpressure(t *testing.T) {
	bp := NewBackpressure(BackpressureConfig{
		MaxEventsPerSecond: 100,
		MaxPendingEvents:   10,
	})

	// Should admit events up to pending limit
	for i := 0; i < 10; i++ {
		if !bp.Admit() {
			t.Errorf("expected event %d to be admitted", i)
		}
	}

	// Next should be rejected (pending full)
	if bp.Admit() {
		t.Error("expected event to be rejected due to pending limit")
	}

	// Release some
	for i := 0; i < 5; i++ {
		bp.Release()
	}

	// Should admit again
	if !bp.Admit() {
		t.Error("expected event to be admitted after release")
	}

	stats := bp.Stats()
	if stats.Accepted < 10 {
		t.Errorf("expected at least 10 accepted, got %d", stats.Accepted)
	}
	if stats.Rejected == 0 {
		t.Error("expected some rejections")
	}
}

func TestComputeChecksum(t *testing.T) {
	evt := Event{
		ID:          "evt-test-001",
		SequenceNum: 42,
		Type:        EventObservation,
		Timestamp:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Data:        []byte(`{"transport":"mqtt"}`),
	}

	cs1 := ComputeChecksum(evt)
	cs2 := ComputeChecksum(evt)

	if cs1 != cs2 {
		t.Error("checksum should be deterministic")
	}

	evt.Data = []byte(`{"transport":"serial"}`)
	cs3 := ComputeChecksum(evt)
	if cs1 == cs3 {
		t.Error("different data should produce different checksum")
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func generateFixedEventStream(count int) []Event {
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	events := make([]Event, 0, count)

	for i := 0; i < count; i++ {
		ts := baseTime.Add(time.Duration(i) * time.Second)
		nodeNum := int64(100 + (i % 5))
		nodeID := "!node" + string(rune('A'+i%5))
		transport := "transport-" + string(rune('1'+i%3))

		var evtType EventType
		var data []byte

		switch i % 4 {
		case 0:
			evtType = EventObservation
			obs := ObservationData{Transport: transport, NodeNum: nodeNum, NodeID: nodeID}
			data, _ = json.Marshal(obs)
		case 1:
			evtType = EventAnomaly
			anom := AnomalyData{Transport: transport, NodeID: nodeID, Category: "dead_letter", Severity: "medium", Score: 0.3}
			data, _ = json.Marshal(anom)
		case 2:
			evtType = EventTopologyUpdate
			topo := TopologyData{NodeNum: nodeNum, NodeID: nodeID, LongName: "Node " + nodeID, Action: "updated"}
			data, _ = json.Marshal(topo)
		case 3:
			evtType = EventTransportHealth
			th := TransportHealthData{Transport: transport, State: "live", Health: 0.8}
			data, _ = json.Marshal(th)
		}

		events = append(events, Event{
			ID:           "evt-fixed-" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
			SequenceNum:  uint64(i + 1),
			Type:         evtType,
			Timestamp:    ts,
			LogicalClock: uint64(i),
			SourceNodeID: "mel-fixed-source",
			Data:         data,
		})
	}

	return events
}
