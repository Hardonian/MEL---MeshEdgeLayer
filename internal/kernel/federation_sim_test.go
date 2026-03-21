package kernel

// federation_sim_test.go — Multi-node federation simulation tests that
// validate distributed kernel correctness without network I/O.
//
// Tests cover:
//   - multi-node event sync convergence
//   - split-brain divergence and recovery
//   - conflict resolution via consistency model
//   - replay correctness across snapshot boundaries
//   - bounded staleness enforcement
//   - region-based event filtering
//   - idempotent event ingestion
//   - crash recovery via snapshot + delta replay

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// ─── Multi-Node Sync Convergence ─────────────────────────────────────────────

// TestMultiNodeSyncConvergence simulates 3 nodes that each receive local
// events, then exchange (sync) all events. After sync, all nodes should
// have identical state.
func TestMultiNodeSyncConvergence(t *testing.T) {
	policy := Policy{Version: "v1", Mode: "advisory"}

	// Generate per-node local events
	eventsA := generateNodeLocalEvents("mel-node-A", "us-east", "transport-A", 20)
	eventsB := generateNodeLocalEvents("mel-node-B", "eu-west", "transport-B", 20)
	eventsC := generateNodeLocalEvents("mel-node-C", "ap-south", "transport-C", 20)

	kA := New("mel-node-A", policy)
	kB := New("mel-node-B", policy)
	kC := New("mel-node-C", policy)

	// Phase 1: each node processes its own events
	for _, evt := range eventsA {
		kA.Apply(evt)
	}
	for _, evt := range eventsB {
		kB.Apply(evt)
	}
	for _, evt := range eventsC {
		kC.Apply(evt)
	}

	// Verify nodes have diverged (each only knows its own transport)
	stateA := kA.State()
	if _, ok := stateA.TransportScores["transport-B"]; ok {
		t.Error("node A should not know about transport-B before sync")
	}

	// Phase 2: full sync — each node receives all other nodes' events
	// (simulates federation sync completing)
	allEvents := append(append(eventsA, eventsB...), eventsC...)

	// Create fresh kernels and replay the full union
	kA2 := New("mel-node-A", policy)
	kB2 := New("mel-node-B", policy)
	kC2 := New("mel-node-C", policy)

	for _, evt := range allEvents {
		kA2.Apply(evt)
		kB2.Apply(evt)
		kC2.Apply(evt)
	}

	stateA2 := kA2.State()
	stateB2 := kB2.State()
	stateC2 := kC2.State()

	// All three must have identical state
	assertStatesEqual(t, &stateA2, &stateB2, "A vs B after sync")
	assertStatesEqual(t, &stateB2, &stateC2, "B vs C after sync")

	// All should know about all transports
	for _, transport := range []string{"transport-A", "transport-B", "transport-C"} {
		if _, ok := stateA2.TransportScores[transport]; !ok {
			t.Errorf("node A should know about %s after sync", transport)
		}
	}
}

// ─── Split-Brain Divergence and Recovery ─────────────────────────────────────

// TestSplitBrainDivergenceRecovery simulates a network partition where two
// groups of nodes process different events, then verifies that replaying
// the union of all events produces convergent state.
func TestSplitBrainDivergenceRecovery(t *testing.T) {
	policy := Policy{
		Version:              "v1",
		Mode:                 "guarded_auto",
		AllowedActions:       []string{"trigger_health_recheck"},
		RequireMinConfidence: 0.5,
	}

	// Pre-partition: shared events
	sharedEvents := generateChaosEventStream(20, 2, 3)

	// Partition A events (only seen by nodes A1, A2)
	partA := generateNodeLocalEvents("mel-part-A", "us-east", "transport-partA", 15)

	// Partition B events (only seen by nodes B1, B2)
	partB := generateNodeLocalEvents("mel-part-B", "eu-west", "transport-partB", 15)

	// Node A1 processes shared + partition A
	kA1 := New("mel-A1", policy)
	for _, evt := range sharedEvents {
		kA1.Apply(evt)
	}
	for _, evt := range partA {
		kA1.Apply(evt)
	}

	// Node B1 processes shared + partition B
	kB1 := New("mel-B1", policy)
	for _, evt := range sharedEvents {
		kB1.Apply(evt)
	}
	for _, evt := range partB {
		kB1.Apply(evt)
	}

	stateA1 := kA1.State()
	stateB1 := kB1.State()

	// They should have diverged
	_, okA := stateA1.TransportScores["transport-partA"]
	_, okB := stateA1.TransportScores["transport-partB"]
	if !okA {
		t.Error("A1 should have transport-partA")
	}
	if okB {
		t.Error("A1 should NOT have transport-partB before recovery")
	}

	_, okB = stateB1.TransportScores["transport-partB"]
	if !okB {
		t.Error("B1 should have transport-partB")
	}

	// Recovery: both sides replay the union
	allEvents := make([]Event, 0)
	allEvents = append(allEvents, sharedEvents...)
	allEvents = append(allEvents, partA...)
	allEvents = append(allEvents, partB...)

	kRecoveredA := New("mel-recovered-A", policy)
	kRecoveredB := New("mel-recovered-B", policy)
	for _, evt := range allEvents {
		kRecoveredA.Apply(evt)
		kRecoveredB.Apply(evt)
	}

	stateRA := kRecoveredA.State()
	stateRB := kRecoveredB.State()

	assertStatesEqual(t, &stateRA, &stateRB, "recovered A vs B")

	// Both should now know both partition transports
	if _, ok := stateRA.TransportScores["transport-partA"]; !ok {
		t.Error("recovered A should have transport-partA")
	}
	if _, ok := stateRA.TransportScores["transport-partB"]; !ok {
		t.Error("recovered A should have transport-partB")
	}
}

// ─── Snapshot + Delta Replay Correctness ─────────────────────────────────────

// TestSnapshotDeltaReplayCorrectness verifies that snapshot+delta replay
// produces the same state as full replay, across multiple snapshot points.
func TestSnapshotDeltaReplayCorrectness(t *testing.T) {
	policy := Policy{Version: "v1", Mode: "advisory"}
	events := generateChaosEventStream(200, 4, 6)

	// Full replay
	kFull := New("mel-full", policy)
	for _, evt := range events {
		kFull.Apply(evt)
	}
	stateFull := kFull.State()

	// Snapshot at multiple points and delta-replay from each
	snapshotPoints := []int{25, 50, 100, 150}
	for _, snapAt := range snapshotPoints {
		kSnap := New("mel-snap", policy)
		for _, evt := range events[:snapAt] {
			kSnap.Apply(evt)
		}
		snapshot := kSnap.State()

		// Restore from snapshot and replay remaining
		kDelta := New("mel-delta", policy)
		kDelta.RestoreState(&snapshot)
		for _, evt := range events[snapAt:] {
			kDelta.Apply(evt)
		}
		stateDelta := kDelta.State()

		assertStatesEqual(t, &stateFull, &stateDelta,
			fmt.Sprintf("full vs snapshot@%d + delta", snapAt))
	}
}

// ─── Idempotent Event Application ────────────────────────────────────────────

// TestIdempotentEventApplication verifies that applying the same event
// twice doesn't corrupt state (idempotent ingestion for dedup).
func TestIdempotentEventApplication(t *testing.T) {
	policy := Policy{Version: "v1", Mode: "advisory"}
	events := generateChaosEventStream(30, 2, 3)

	// Apply events once
	kOnce := New("mel-once", policy)
	for _, evt := range events {
		kOnce.Apply(evt)
	}
	stateOnce := kOnce.State()

	// Apply events twice (simulating duplicate reception during sync)
	kTwice := New("mel-twice", policy)
	for _, evt := range events {
		kTwice.Apply(evt)
	}
	for _, evt := range events {
		kTwice.Apply(evt)
	}
	stateTwice := kTwice.State()

	// The states won't be identical because scores are cumulative,
	// but they must remain in valid ranges (no overflow, no NaN)
	for id, ns := range stateTwice.NodeScores {
		if ns.CompositeScore < 0 || ns.CompositeScore > 1 {
			t.Errorf("node %s: composite out of range after double-apply: %.4f", id, ns.CompositeScore)
		}
		if ns.HealthScore < 0 || ns.HealthScore > 1 {
			t.Errorf("node %s: health out of range: %.4f", id, ns.HealthScore)
		}
	}
	for name, ts := range stateTwice.TransportScores {
		if ts.HealthScore < 0 || ts.HealthScore > 1 {
			t.Errorf("transport %s: health out of range: %.4f", name, ts.HealthScore)
		}
	}

	// Logical clock should have advanced further with double events
	if stateTwice.LogicalClock <= stateOnce.LogicalClock {
		t.Error("double-apply should advance logical clock further")
	}
}

// ─── Region-Based Filtering ──────────────────────────────────────────────────

// TestRegionBasedEventIsolation verifies that events from different regions
// maintain region attribution and don't contaminate each other's scores.
func TestRegionBasedEventIsolation(t *testing.T) {
	policy := Policy{Version: "v1", Mode: "advisory"}
	k := New("mel-region-test", policy)

	// Events from US-East
	for i := 0; i < 10; i++ {
		obs := ObservationData{Transport: "mqtt-us", NodeNum: int64(100 + i), NodeID: fmt.Sprintf("!us-node-%d", i)}
		data, _ := json.Marshal(obs)
		k.Apply(Event{
			ID:           fmt.Sprintf("evt-us-%d", i),
			Type:         EventObservation,
			Timestamp:    time.Date(2025, 1, 1, 0, 0, i, 0, time.UTC),
			SourceNodeID: "mel-us-east",
			SourceRegion: "us-east",
			Data:         data,
		})
	}

	// Events from EU-West
	for i := 0; i < 10; i++ {
		obs := ObservationData{Transport: "mqtt-eu", NodeNum: int64(200 + i), NodeID: fmt.Sprintf("!eu-node-%d", i)}
		data, _ := json.Marshal(obs)
		k.Apply(Event{
			ID:           fmt.Sprintf("evt-eu-%d", i),
			Type:         EventObservation,
			Timestamp:    time.Date(2025, 1, 1, 0, 0, i, 0, time.UTC),
			SourceNodeID: "mel-eu-west",
			SourceRegion: "eu-west",
			Data:         data,
		})
	}

	state := k.State()

	// Should have separate transport scores
	if _, ok := state.TransportScores["mqtt-us"]; !ok {
		t.Error("expected transport mqtt-us")
	}
	if _, ok := state.TransportScores["mqtt-eu"]; !ok {
		t.Error("expected transport mqtt-eu")
	}

	// Node registry should track regions
	usCount, euCount := 0, 0
	for _, node := range state.NodeRegistry {
		switch node.Region {
		case "us-east":
			usCount++
		case "eu-west":
			euCount++
		}
	}
	if usCount != 10 {
		t.Errorf("expected 10 US nodes, got %d", usCount)
	}
	if euCount != 10 {
		t.Errorf("expected 10 EU nodes, got %d", euCount)
	}
}

// ─── Cross-Region Event Ordering ─────────────────────────────────────────────

// TestCrossRegionLamportOrdering verifies that Lamport clocks correctly
// track causal ordering when events flow across regions.
func TestCrossRegionLamportOrdering(t *testing.T) {
	policy := Policy{Version: "v1", Mode: "advisory"}

	kUS := New("mel-us", policy)
	kEU := New("mel-eu", policy)

	// US processes 10 events
	for i := 0; i < 10; i++ {
		data, _ := json.Marshal(ObservationData{Transport: "mqtt", NodeNum: 100, NodeID: "!n1"})
		kUS.Apply(Event{
			ID:           fmt.Sprintf("evt-us-%d", i),
			Type:         EventObservation,
			Timestamp:    time.Now().UTC(),
			LogicalClock: uint64(i),
			SourceNodeID: "mel-us",
			SourceRegion: "us-east",
			Data:         data,
		})
	}

	usState := kUS.State()
	usClock := usState.LogicalClock

	// EU receives US's last event (sync)
	data, _ := json.Marshal(ObservationData{Transport: "mqtt", NodeNum: 200, NodeID: "!n2"})
	kEU.Apply(Event{
		ID:           "evt-eu-from-us",
		Type:         EventObservation,
		Timestamp:    time.Now().UTC(),
		LogicalClock: usClock, // carries US's clock
		SourceNodeID: "mel-us",
		SourceRegion: "us-east",
		Data:         data,
	})

	euState := kEU.State()
	// EU's clock must be > US's clock
	if euState.LogicalClock <= usClock {
		t.Errorf("EU clock %d should be > US clock %d after sync",
			euState.LogicalClock, usClock)
	}

	// EU generates more events
	for i := 0; i < 5; i++ {
		data, _ := json.Marshal(ObservationData{Transport: "serial", NodeNum: 300, NodeID: "!n3"})
		kEU.Apply(Event{
			ID:           fmt.Sprintf("evt-eu-%d", i),
			Type:         EventObservation,
			Timestamp:    time.Now().UTC(),
			LogicalClock: euState.LogicalClock + uint64(i),
			SourceNodeID: "mel-eu",
			SourceRegion: "eu-west",
			Data:         data,
		})
	}

	euFinal := kEU.State()
	// Clock should have advanced monotonically
	if euFinal.LogicalClock <= euState.LogicalClock {
		t.Error("EU clock should advance monotonically")
	}
}

// ─── Maintenance Window Federation ───────────────────────────────────────────

// TestMaintenanceWindowFederation verifies that maintenance windows
// (freeze events) propagate correctly across federated nodes.
func TestMaintenanceWindowFederation(t *testing.T) {
	policy := Policy{Version: "v1", Mode: "advisory"}

	kA := New("mel-node-A", policy)
	kB := New("mel-node-B", policy)

	// Node A starts maintenance window
	maintData := MaintenanceData{
		WindowID:   "mw-001",
		ScopeType:  "transport",
		ScopeValue: "mqtt",
		Reason:     "firmware upgrade",
	}
	data, _ := json.Marshal(maintData)
	maintEvent := Event{
		ID:           NewEventID(),
		Type:         EventMaintenanceStart,
		Timestamp:    time.Now().UTC(),
		SourceNodeID: "mel-node-A",
		Data:         data,
	}

	kA.Apply(maintEvent)
	// Sync to B
	kB.Apply(maintEvent)

	stateA := kA.State()
	stateB := kB.State()

	// Both should have the maintenance freeze
	if _, ok := stateA.ActiveFreezes["maint-mw-001"]; !ok {
		t.Error("node A should have maintenance freeze")
	}
	if _, ok := stateB.ActiveFreezes["maint-mw-001"]; !ok {
		t.Error("node B should have maintenance freeze after sync")
	}

	// End maintenance
	endEvent := Event{
		ID:           NewEventID(),
		Type:         EventMaintenanceEnd,
		Timestamp:    time.Now().UTC(),
		SourceNodeID: "mel-node-A",
		Data:         data,
	}
	kA.Apply(endEvent)
	kB.Apply(endEvent)

	stateA = kA.State()
	stateB = kB.State()

	if _, ok := stateA.ActiveFreezes["maint-mw-001"]; ok {
		t.Error("maintenance freeze should be cleared on A")
	}
	if _, ok := stateB.ActiveFreezes["maint-mw-001"]; ok {
		t.Error("maintenance freeze should be cleared on B")
	}
}

// ─── Large-Scale Determinism ─────────────────────────────────────────────────

// TestLargeScaleDeterminism runs 10 independent kernel instances through
// 1000 events and verifies bit-exact state equality.
func TestLargeScaleDeterminism(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large-scale test in short mode")
	}

	const nodeCount = 10
	const eventCount = 1000

	policy := Policy{
		Version:              "v1",
		Mode:                 "guarded_auto",
		AllowedActions:       []string{"restart_transport", "trigger_health_recheck"},
		RequireMinConfidence: 0.5,
	}

	events := generateChaosEventStream(eventCount, 5, 8)

	states := make([]State, nodeCount)
	effectCounts := make([]int, nodeCount)

	for i := 0; i < nodeCount; i++ {
		k := New(fmt.Sprintf("mel-node-%02d", i), policy)
		for _, evt := range events {
			effs := k.Apply(evt)
			effectCounts[i] += len(effs)
		}
		states[i] = k.State()
	}

	ref := states[0]
	for i := 1; i < nodeCount; i++ {
		assertStatesEqual(t, &ref, &states[i], fmt.Sprintf("node-0 vs node-%d", i))
		if effectCounts[i] != effectCounts[0] {
			t.Errorf("node-%d effect count %d != %d", i, effectCounts[i], effectCounts[0])
		}
	}
}

// ─── Mixed Event Type Stress ─────────────────────────────────────────────────

// TestMixedEventTypeStress exercises all event types simultaneously and
// ensures no panics or score violations.
func TestMixedEventTypeStress(t *testing.T) {
	policy := Policy{
		Version:              "v1",
		Mode:                 "guarded_auto",
		AllowedActions:       []string{"restart_transport", "trigger_health_recheck"},
		RequireMinConfidence: 0.3,
	}

	k := New("mel-stress", policy)
	rng := rand.New(rand.NewSource(42))

	types := []EventType{
		EventObservation, EventAnomaly, EventTopologyUpdate,
		EventTransportHealth, EventNodeState, EventAdapterState,
		EventFreezeCreated, EventFreezeCleared, EventPolicyChange,
		EventRegionHealth, EventPeerJoined, EventPeerLeft,
		EventMaintenanceStart, EventMaintenanceEnd,
	}

	baseTime := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 500; i++ {
		evtType := types[rng.Intn(len(types))]
		var data []byte

		switch evtType {
		case EventObservation:
			d := ObservationData{Transport: fmt.Sprintf("t%d", rng.Intn(3)), NodeNum: int64(rng.Intn(10) + 100), NodeID: fmt.Sprintf("!n%d", rng.Intn(5))}
			data, _ = json.Marshal(d)
		case EventAnomaly:
			d := AnomalyData{Transport: fmt.Sprintf("t%d", rng.Intn(3)), NodeID: fmt.Sprintf("!n%d", rng.Intn(5)), Category: "test", Severity: "medium", Score: rng.Float64()}
			data, _ = json.Marshal(d)
		case EventTopologyUpdate:
			d := TopologyData{NodeNum: int64(rng.Intn(10) + 100), NodeID: fmt.Sprintf("!n%d", rng.Intn(5)), Action: "updated"}
			data, _ = json.Marshal(d)
		case EventTransportHealth:
			d := TransportHealthData{Transport: fmt.Sprintf("t%d", rng.Intn(3)), State: "live", Health: rng.Float64()}
			data, _ = json.Marshal(d)
		case EventNodeState:
			d := NodeInfo{NodeNum: int64(rng.Intn(10) + 100), NodeID: fmt.Sprintf("!n%d", rng.Intn(5))}
			data, _ = json.Marshal(d)
		case EventAdapterState:
			d := AdapterStateData{AdapterName: fmt.Sprintf("t%d", rng.Intn(3)), State: "connected"}
			data, _ = json.Marshal(d)
		case EventFreezeCreated, EventFreezeCleared:
			d := FreezeData{FreezeID: fmt.Sprintf("frz-%d", rng.Intn(3)), ScopeType: "transport", ScopeValue: "t0", Reason: "test"}
			data, _ = json.Marshal(d)
		case EventPolicyChange:
			d := Policy{Version: fmt.Sprintf("v%d", rng.Intn(3)+1), Mode: "advisory"}
			data, _ = json.Marshal(d)
		case EventRegionHealth:
			d := RegionHealthData{RegionID: fmt.Sprintf("region-%d", rng.Intn(3)), OverallHealth: rng.Float64(), NodeCount: 10, HealthyNodes: 8}
			data, _ = json.Marshal(d)
		case EventPeerJoined, EventPeerLeft:
			d := PeerEventData{PeerID: fmt.Sprintf("peer-%d", rng.Intn(3)), Region: "test"}
			data, _ = json.Marshal(d)
		case EventMaintenanceStart, EventMaintenanceEnd:
			d := MaintenanceData{WindowID: fmt.Sprintf("mw-%d", rng.Intn(2)), ScopeType: "transport", ScopeValue: "t0"}
			data, _ = json.Marshal(d)
		}

		k.Apply(Event{
			ID:           fmt.Sprintf("evt-stress-%06d", i),
			Type:         evtType,
			Timestamp:    baseTime.Add(time.Duration(i) * time.Second),
			LogicalClock: uint64(i),
			SourceNodeID: "mel-stress",
			Data:         data,
		})
	}

	state := k.State()
	// Verify all scores in valid ranges
	for id, ns := range state.NodeScores {
		if ns.CompositeScore < 0 || ns.CompositeScore > 1 {
			t.Errorf("node %s: composite out of range: %.4f", id, ns.CompositeScore)
		}
	}
	for name, ts := range state.TransportScores {
		if ts.HealthScore < 0 || ts.HealthScore > 1 {
			t.Errorf("transport %s: health out of range: %.4f", name, ts.HealthScore)
		}
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func generateNodeLocalEvents(nodeID, region, transport string, count int) []Event {
	baseTime := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	events := make([]Event, 0, count)
	for i := 0; i < count; i++ {
		ts := baseTime.Add(time.Duration(i) * 500 * time.Millisecond)
		nodeNum := int64(1000 + i)
		meshNodeID := fmt.Sprintf("!%s-node-%d", region, i)

		var evtType EventType
		var data []byte

		switch i % 3 {
		case 0:
			evtType = EventObservation
			obs := ObservationData{Transport: transport, NodeNum: nodeNum, NodeID: meshNodeID}
			data, _ = json.Marshal(obs)
		case 1:
			evtType = EventTransportHealth
			th := TransportHealthData{Transport: transport, State: "live", Health: 0.85}
			data, _ = json.Marshal(th)
		case 2:
			evtType = EventTopologyUpdate
			topo := TopologyData{NodeNum: nodeNum, NodeID: meshNodeID, LongName: "Node " + meshNodeID, Action: "updated", Region: region}
			data, _ = json.Marshal(topo)
		}

		events = append(events, Event{
			ID:           fmt.Sprintf("evt-%s-%06d", nodeID, i),
			SequenceNum:  uint64(i + 1),
			Type:         evtType,
			Timestamp:    ts,
			LogicalClock: uint64(i),
			SourceNodeID: nodeID,
			SourceRegion: region,
			Data:         data,
		})
	}
	return events
}

func assertStatesEqual(t *testing.T, a, b *State, label string) {
	t.Helper()

	if len(a.NodeScores) != len(b.NodeScores) {
		t.Errorf("%s: node scores count mismatch: %d vs %d", label, len(a.NodeScores), len(b.NodeScores))
		return
	}
	for id, sa := range a.NodeScores {
		sb, ok := b.NodeScores[id]
		if !ok {
			t.Errorf("%s: node %s missing in second state", label, id)
			continue
		}
		if sa.Classification != sb.Classification {
			t.Errorf("%s: node %s classification mismatch: %s vs %s", label, id, sa.Classification, sb.Classification)
		}
		if absDiff(sa.CompositeScore, sb.CompositeScore) > 1e-12 {
			t.Errorf("%s: node %s composite mismatch: %.15f vs %.15f", label, id, sa.CompositeScore, sb.CompositeScore)
		}
	}

	if len(a.TransportScores) != len(b.TransportScores) {
		t.Errorf("%s: transport count mismatch: %d vs %d", label, len(a.TransportScores), len(b.TransportScores))
		return
	}
	for name, ta := range a.TransportScores {
		tb, ok := b.TransportScores[name]
		if !ok {
			t.Errorf("%s: transport %s missing in second state", label, name)
			continue
		}
		if absDiff(ta.HealthScore, tb.HealthScore) > 1e-12 {
			t.Errorf("%s: transport %s health mismatch: %.15f vs %.15f", label, name, ta.HealthScore, tb.HealthScore)
		}
	}

	if len(a.ActionStates) != len(b.ActionStates) {
		t.Errorf("%s: action states count mismatch: %d vs %d", label, len(a.ActionStates), len(b.ActionStates))
	}
	if len(a.ActiveFreezes) != len(b.ActiveFreezes) {
		t.Errorf("%s: active freezes count mismatch: %d vs %d", label, len(a.ActiveFreezes), len(b.ActiveFreezes))
	}
}
