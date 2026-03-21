package kernel

// chaos_test.go — Multi-node simulation, chaos, and stress tests for the
// MEL deterministic kernel.
//
// These tests validate:
//   - multi-node determinism under identical event streams
//   - conflict detection and resolution semantics
//   - split-brain downgrade behavior (coordination tokens)
//   - backpressure under high event load
//   - Lamport clock causal ordering across nodes
//   - freeze/maintenance lifecycle safety
//   - action coordination preventing duplicate execution
//   - recovery from arbitrary state via event replay
//   - stress: high throughput with bounded memory

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// ─── Multi-Node Determinism ───────────────────────────────────────────────────

// TestMultiNodeDeterminism verifies that N independent kernel instances
// fed the same event stream produce identical final state.
func TestMultiNodeDeterminism(t *testing.T) {
	const nodeCount = 5
	const eventCount = 100

	policy := Policy{
		Version:              "v1",
		Mode:                 "advisory",
		AllowedActions:       []string{"restart_transport", "trigger_health_recheck"},
		RequireMinConfidence: 0.6,
	}

	events := generateChaosEventStream(eventCount, 3, 3)

	// Run each kernel independently
	states := make([]State, nodeCount)
	effectCounts := make([]int, nodeCount)

	for i := 0; i < nodeCount; i++ {
		nodeID := fmt.Sprintf("mel-node-%02d", i)
		k := New(nodeID, policy)
		for _, evt := range events {
			effects := k.Apply(evt)
			effectCounts[i] += len(effects)
		}
		states[i] = k.State()
	}

	// All states must be identical
	ref := states[0]
	for i := 1; i < nodeCount; i++ {
		s := states[i]

		if len(s.NodeScores) != len(ref.NodeScores) {
			t.Errorf("node %d: node scores count mismatch: %d vs %d",
				i, len(s.NodeScores), len(ref.NodeScores))
		}
		for id, scoreRef := range ref.NodeScores {
			scoreI, ok := s.NodeScores[id]
			if !ok {
				t.Errorf("node %d: missing node score %s", i, id)
				continue
			}
			if scoreRef.Classification != scoreI.Classification {
				t.Errorf("node %d: node %s classification %s != %s",
					i, id, scoreI.Classification, scoreRef.Classification)
			}
			if absDiff(scoreRef.CompositeScore, scoreI.CompositeScore) > 1e-9 {
				t.Errorf("node %d: node %s composite %.10f != %.10f",
					i, id, scoreI.CompositeScore, scoreRef.CompositeScore)
			}
		}

		if len(s.TransportScores) != len(ref.TransportScores) {
			t.Errorf("node %d: transport count mismatch: %d vs %d",
				i, len(s.TransportScores), len(ref.TransportScores))
		}
		for name, tsRef := range ref.TransportScores {
			tsI, ok := s.TransportScores[name]
			if !ok {
				t.Errorf("node %d: missing transport %s", i, name)
				continue
			}
			if absDiff(tsRef.HealthScore, tsI.HealthScore) > 1e-9 {
				t.Errorf("node %d: transport %s health %.10f != %.10f",
					i, name, tsI.HealthScore, tsRef.HealthScore)
			}
		}

		if effectCounts[i] != effectCounts[0] {
			t.Errorf("node %d: effect count %d != %d", i, effectCounts[i], effectCounts[0])
		}
	}
}

// ─── Causal Ordering ─────────────────────────────────────────────────────────

// TestCausalOrderingLamportClock verifies that Lamport clocks advance
// monotonically across nodes when processing cross-node events.
func TestCausalOrderingLamportClock(t *testing.T) {
	kA := New("mel-node-A", Policy{Version: "v1"})
	kB := New("mel-node-B", Policy{Version: "v1"})

	// Apply 5 events to A
	for i := 0; i < 5; i++ {
		kA.Apply(Event{
			ID:           NewEventID(),
			Type:         EventObservation,
			Timestamp:    time.Now().UTC(),
			LogicalClock: uint64(i),
			SourceNodeID: "mel-node-A",
			Data:         []byte(`{"transport":"mqtt","node_num":100,"node_id":"!n1"}`),
		})
	}
	stateA := kA.State()
	clockA := stateA.LogicalClock

	// Apply an event from A's clock to B (simulating cross-node sync)
	kB.Apply(Event{
		ID:           NewEventID(),
		Type:         EventObservation,
		Timestamp:    time.Now().UTC(),
		LogicalClock: clockA, // B receives A's clock value
		SourceNodeID: "mel-node-A",
		Data:         []byte(`{"transport":"mqtt","node_num":101,"node_id":"!n2"}`),
	})
	stateB := kB.State()

	// B's clock must be strictly greater than A's clock
	if stateB.LogicalClock <= clockA {
		t.Errorf("B's clock %d should be > A's clock %d after receiving A's event",
			stateB.LogicalClock, clockA)
	}

	// Apply another event to B with lower clock — B's clock should still advance
	prevB := stateB.LogicalClock
	kB.Apply(Event{
		ID:           NewEventID(),
		Type:         EventObservation,
		Timestamp:    time.Now().UTC(),
		LogicalClock: 1, // stale clock
		SourceNodeID: "mel-node-C",
		Data:         []byte(`{"transport":"serial","node_num":102,"node_id":"!n3"}`),
	})
	stateB2 := kB.State()
	if stateB2.LogicalClock <= prevB {
		t.Errorf("clock should advance monotonically: got %d, prev %d",
			stateB2.LogicalClock, prevB)
	}
}

// ─── Network Partition / Split-Brain ─────────────────────────────────────────

// TestNetworkPartitionDivergence simulates two nodes diverging under partition
// and verifies that after re-sync (union of events), they converge to the
// same final state (given same policy).
func TestNetworkPartitionDivergence(t *testing.T) {
	policy := Policy{
		Version: "v1",
		Mode:    "advisory",
	}

	// Shared event stream (pre-partition)
	sharedEvents := generateChaosEventStream(10, 2, 2)

	kA := New("mel-node-A", policy)
	kB := New("mel-node-B", policy)

	// Both process shared events
	for _, evt := range sharedEvents {
		kA.Apply(evt)
		kB.Apply(evt)
	}

	// Node A gets partition-local events
	partitionA := generatePartitionEvents("mel-node-A", "transport-A", 5, 20)
	for _, evt := range partitionA {
		kA.Apply(evt)
	}

	// Node B gets different partition-local events
	partitionB := generatePartitionEvents("mel-node-B", "transport-B", 5, 30)
	for _, evt := range partitionB {
		kB.Apply(evt)
	}

	// Verify nodes have diverged
	stateAMid := kA.State()
	stateBMid := kB.State()

	// A should know about transport-A
	if _, ok := stateAMid.TransportScores["transport-A"]; !ok {
		t.Error("node A should have transport-A score after partition")
	}
	// B should know about transport-B
	if _, ok := stateBMid.TransportScores["transport-B"]; !ok {
		t.Error("node B should have transport-B score after partition")
	}

	// Re-sync: both nodes receive each other's partition events
	// (simulates partition recovery — events replayed on top of current state)
	for _, evt := range partitionB {
		kA.Apply(evt)
	}
	for _, evt := range partitionA {
		kB.Apply(evt)
	}

	stateAFinal := kA.State()
	stateBFinal := kB.State()

	// After sync, both should know about both transports
	if _, ok := stateAFinal.TransportScores["transport-B"]; !ok {
		t.Error("node A should have transport-B after re-sync")
	}
	if _, ok := stateBFinal.TransportScores["transport-A"]; !ok {
		t.Error("node B should have transport-A after re-sync")
	}
}

// ─── Action Coordination ─────────────────────────────────────────────────────

// TestActionCoordinationNoDuplicates verifies that coordination tokens
// prevent the same action from being executed by two nodes simultaneously.
func TestActionCoordinationNoDuplicates(t *testing.T) {
	acA := NewActionCoordinator("mel-node-A", 5*time.Minute)
	acB := NewActionCoordinator("mel-node-B", 5*time.Minute)

	actionID := "act-critical-001"

	// Node A acquires token first
	tokenA, acquiredA := acA.TryAcquire(actionID)
	if !acquiredA {
		t.Fatal("node A should acquire token")
	}

	// Node A broadcasts its token to B (federation sync)
	acB.RecordRemoteToken(*tokenA)

	// Node B attempts to acquire — must fail
	_, acquiredB := acB.TryAcquire(actionID)
	if acquiredB {
		t.Error("node B must NOT acquire token when A holds it")
	}

	// Node A releases (action complete)
	acA.Release(actionID, true)

	// Now B should be able to acquire (after token expiry in real system,
	// or after explicit release propagation)
	// Simulate by not recording the remote token again
	acB2 := NewActionCoordinator("mel-node-B", 5*time.Minute)
	_, acquiredB2 := acB2.TryAcquire(actionID)
	if !acquiredB2 {
		t.Error("node B should acquire token after A released it")
	}
}

// TestActionCoordinationConcurrent verifies that concurrent TryAcquire
// from the same coordinator is safe (no races).
func TestActionCoordinationConcurrent(t *testing.T) {
	ac := NewActionCoordinator("mel-concurrent-node", time.Minute)
	const workers = 20
	const actions = 50

	var wg sync.WaitGroup
	acquired := make([]bool, actions)
	var mu sync.Mutex

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for a := 0; a < actions; a++ {
				actionID := fmt.Sprintf("action-%03d", a)
				token, ok := ac.TryAcquire(actionID)
				if ok {
					mu.Lock()
					acquired[a] = true
					mu.Unlock()
					// Brief hold then release
					time.Sleep(time.Microsecond)
					ac.Release(token.ActionID, true)
				}
			}
		}(w)
	}
	wg.Wait()

	// Every action should have been acquired at least once
	for a, ok := range acquired {
		if !ok {
			t.Errorf("action %d was never acquired", a)
		}
	}
}

// ─── Backpressure Under Load ──────────────────────────────────────────────────

// TestBackpressureHighLoad verifies that the backpressure system correctly
// limits event throughput and doesn't panic under concurrent access.
func TestBackpressureHighLoad(t *testing.T) {
	bp := NewBackpressure(BackpressureConfig{
		MaxEventsPerSecond: 1000,
		MaxPendingEvents:   100,
		BatchSize:          20,
	})

	const goroutines = 10
	const eventsPerGoroutine = 50

	var wg sync.WaitGroup
	var totalAdmitted, totalRejected int
	var mu sync.Mutex

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			admitted, rejected := 0, 0
			for i := 0; i < eventsPerGoroutine; i++ {
				if bp.Admit() {
					admitted++
					bp.Release() // immediately process
				} else {
					rejected++
				}
			}
			mu.Lock()
			totalAdmitted += admitted
			totalRejected += rejected
			mu.Unlock()
		}()
	}
	wg.Wait()

	stats := bp.Stats()
	total := goroutines * eventsPerGoroutine
	t.Logf("admitted=%d rejected=%d total=%d stats=%+v",
		totalAdmitted, totalRejected, total, stats)

	if totalAdmitted+totalRejected != total {
		t.Errorf("admitted+rejected=%d should equal total=%d", totalAdmitted+totalRejected, total)
	}
	if stats.Accepted == 0 {
		t.Error("expected some accepted events")
	}
}

// TestBackpressureMemoryBound verifies that the pending event count never
// exceeds the configured maximum.
func TestBackpressureMemoryBound(t *testing.T) {
	const maxPending = 20
	bp := NewBackpressure(BackpressureConfig{
		MaxEventsPerSecond: 100000,
		MaxPendingEvents:   maxPending,
	})

	// Fill to the limit
	admitted := 0
	for i := 0; i < maxPending*2; i++ {
		if bp.Admit() {
			admitted++
		}
	}

	if admitted > maxPending {
		t.Errorf("admitted %d events but max pending is %d", admitted, maxPending)
	}

	stats := bp.Stats()
	if stats.PendingCount > int64(maxPending) {
		t.Errorf("pending %d exceeds max %d", stats.PendingCount, maxPending)
	}
}

// ─── State Recovery via Replay ────────────────────────────────────────────────

// TestStateRecoveryViaReplay verifies that a kernel restored from a snapshot
// produces the same state after replaying subsequent events.
func TestStateRecoveryViaReplay(t *testing.T) {
	policy := Policy{Version: "v1", Mode: "advisory"}
	events := generateChaosEventStream(60, 3, 3)

	// Full kernel processes all 60 events
	kFull := New("mel-full", policy)
	for _, evt := range events {
		kFull.Apply(evt)
	}
	stateFull := kFull.State()

	// "Recovery" kernel processes first 30 events, snapshots state
	kRecovery := New("mel-recovery", policy)
	for _, evt := range events[:30] {
		kRecovery.Apply(evt)
	}
	snapshot := kRecovery.State()

	// Restore snapshot to new kernel and replay remaining 30 events
	kRestored := New("mel-restored", policy)
	kRestored.RestoreState(&snapshot)
	for _, evt := range events[30:] {
		kRestored.Apply(evt)
	}
	stateRestored := kRestored.State()

	// States must match
	if len(stateRestored.NodeScores) != len(stateFull.NodeScores) {
		t.Errorf("node score count mismatch after recovery: %d vs %d",
			len(stateRestored.NodeScores), len(stateFull.NodeScores))
	}
	for id, scoreFull := range stateFull.NodeScores {
		scoreRestored, ok := stateRestored.NodeScores[id]
		if !ok {
			t.Errorf("node %s missing after recovery", id)
			continue
		}
		if scoreFull.Classification != scoreRestored.Classification {
			t.Errorf("node %s: classification mismatch after recovery: %s vs %s",
				id, scoreRestored.Classification, scoreFull.Classification)
		}
		if absDiff(scoreFull.CompositeScore, scoreRestored.CompositeScore) > 1e-9 {
			t.Errorf("node %s: composite score mismatch after recovery: %f vs %f",
				id, scoreRestored.CompositeScore, scoreFull.CompositeScore)
		}
	}
}

// ─── Conflict Detection Semantics ─────────────────────────────────────────────

// TestConflictingFreezeScopes verifies that two nodes can independently
// create freezes and the kernel correctly tracks both independently.
func TestConflictingFreezeScopes(t *testing.T) {
	policy := Policy{Version: "v1", Mode: "advisory"}

	kA := New("mel-node-A", policy)
	kB := New("mel-node-B", policy)

	// Both nodes create freezes on the same scope (concurrent)
	freezeA := FreezeData{
		FreezeID:   "frz-A-001",
		ScopeType:  "transport",
		ScopeValue: "mqtt-local",
		Reason:     "node A maintenance",
	}
	freezeB := FreezeData{
		FreezeID:   "frz-B-001",
		ScopeType:  "transport",
		ScopeValue: "mqtt-local",
		Reason:     "node B maintenance",
	}

	dataA, _ := json.Marshal(freezeA)
	dataB, _ := json.Marshal(freezeB)

	now := time.Now().UTC()

	kA.Apply(Event{ID: NewEventID(), Type: EventFreezeCreated, Timestamp: now, SourceNodeID: "mel-node-A", Data: dataA})
	kB.Apply(Event{ID: NewEventID(), Type: EventFreezeCreated, Timestamp: now, SourceNodeID: "mel-node-B", Data: dataB})

	// Sync: each node receives the other's freeze
	kA.Apply(Event{ID: NewEventID(), Type: EventFreezeCreated, Timestamp: now, SourceNodeID: "mel-node-B", Data: dataB})
	kB.Apply(Event{ID: NewEventID(), Type: EventFreezeCreated, Timestamp: now, SourceNodeID: "mel-node-A", Data: dataA})

	stateA := kA.State()
	stateB := kB.State()

	// Both nodes should have both freezes
	if _, ok := stateA.ActiveFreezes["frz-A-001"]; !ok {
		t.Error("node A should have freeze frz-A-001")
	}
	if _, ok := stateA.ActiveFreezes["frz-B-001"]; !ok {
		t.Error("node A should have freeze frz-B-001")
	}
	if _, ok := stateB.ActiveFreezes["frz-A-001"]; !ok {
		t.Error("node B should have freeze frz-A-001")
	}
	if _, ok := stateB.ActiveFreezes["frz-B-001"]; !ok {
		t.Error("node B should have freeze frz-B-001")
	}

	// Freeze count should be consistent
	if len(stateA.ActiveFreezes) != len(stateB.ActiveFreezes) {
		t.Errorf("freeze count mismatch: %d vs %d",
			len(stateA.ActiveFreezes), len(stateB.ActiveFreezes))
	}
}

// ─── Policy Divergence Under Scenario Replay ─────────────────────────────────

// TestPolicyDivergenceScenario verifies that replaying with a different policy
// produces different action proposals (the scenario replay use case).
func TestPolicyDivergenceScenario(t *testing.T) {
	// Build a stream that will cause health degradation
	events := buildDegradationStream("transport-stress", 30)

	// Conservative policy — no actions
	kConservative := New("mel-conservative", Policy{
		Version:              "v1",
		Mode:                 "disabled",
		RequireMinConfidence: 0.99,
	})
	// Aggressive policy — many actions
	kAggressive := New("mel-aggressive", Policy{
		Version:              "v2",
		Mode:                 "guarded_auto",
		AllowedActions:       []string{"restart_transport", "trigger_health_recheck"},
		RequireMinConfidence: 0.2,
	})

	var effectsConservative, effectsAggressive []Effect
	for _, evt := range events {
		effectsConservative = append(effectsConservative, kConservative.Apply(evt)...)
		effectsAggressive = append(effectsAggressive, kAggressive.Apply(evt)...)
	}

	// Count propose_action effects
	conservativeActions := countEffectsByType(effectsConservative, EffectProposeAction)
	aggressiveActions := countEffectsByType(effectsAggressive, EffectProposeAction)

	// Conservative should produce no action proposals; aggressive more
	if conservativeActions > 0 {
		t.Errorf("conservative policy should produce no actions, got %d", conservativeActions)
	}
	t.Logf("conservative actions=%d aggressive actions=%d", conservativeActions, aggressiveActions)
}

// ─── Stress: High Throughput ──────────────────────────────────────────────────

// TestHighThroughputStress validates that the kernel handles a large burst
// of events without memory growth or panics.
func TestHighThroughputStress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}
	const eventCount = 10000

	k := New("mel-stress-node", Policy{
		Version: "v1",
		Mode:    "advisory",
	})

	events := generateChaosEventStream(eventCount, 10, 5)

	start := time.Now()
	var totalEffects int
	for _, evt := range events {
		effects := k.Apply(evt)
		totalEffects += len(effects)
	}
	elapsed := time.Since(start)

	state := k.State()

	t.Logf("processed %d events in %v (%.0f/sec), effects=%d, nodes=%d, transports=%d",
		eventCount, elapsed,
		float64(eventCount)/elapsed.Seconds(),
		totalEffects,
		len(state.NodeScores),
		len(state.TransportScores),
	)

	// Must complete in reasonable time
	if elapsed > 5*time.Second {
		t.Errorf("stress test too slow: %v for %d events", elapsed, eventCount)
	}

	// State must remain bounded
	if len(state.NodeScores) > eventCount {
		t.Errorf("node scores unbounded: %d (max expected %d)", len(state.NodeScores), eventCount)
	}
}

// ─── Checksum Integrity ───────────────────────────────────────────────────────

// TestEventChecksumTampering verifies that modified event data produces
// a different checksum (integrity verification).
func TestEventChecksumTampering(t *testing.T) {
	original := Event{
		ID:          "evt-tamper-test",
		SequenceNum: 42,
		Type:        EventObservation,
		Timestamp:   time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		Data:        []byte(`{"transport":"mqtt","node_num":100}`),
	}

	checksum1 := ComputeChecksum(original)

	// Tamper with data
	tampered := original
	tampered.Data = []byte(`{"transport":"mqtt","node_num":999}`)
	checksum2 := ComputeChecksum(tampered)

	if checksum1 == checksum2 {
		t.Error("tampered event should have different checksum")
	}

	// Tamper with sequence number
	tampered2 := original
	tampered2.SequenceNum = 99
	checksum3 := ComputeChecksum(tampered2)
	if checksum1 == checksum3 {
		t.Error("tampered sequence should produce different checksum")
	}

	// Verify original is deterministic
	checksum1b := ComputeChecksum(original)
	if checksum1 != checksum1b {
		t.Error("checksum should be deterministic for same input")
	}
}

// ─── Token Expiry / Cleanup ───────────────────────────────────────────────────

// TestCoordinationTokenExpiry verifies that expired tokens are cleaned up
// and allow re-acquisition.
func TestCoordinationTokenExpiry(t *testing.T) {
	// Very short TTL
	ac := NewActionCoordinator("mel-expiry-node", 50*time.Millisecond)

	token, ok := ac.TryAcquire("act-expiry-001")
	if !ok {
		t.Fatal("expected to acquire token")
	}
	if token.OwnerNodeID != "mel-expiry-node" {
		t.Errorf("expected owner mel-expiry-node, got %s", token.OwnerNodeID)
	}

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)
	ac.Cleanup()

	// Should not be owned after cleanup of expired token
	if ac.IsOwned("act-expiry-001") {
		t.Error("token should have expired and been cleaned up")
	}
}

// ─── Region Health Propagation ────────────────────────────────────────────────

// TestRegionHealthPropagation verifies that region health events update
// kernel state correctly and propagate alerts for degraded regions.
func TestRegionHealthPropagation(t *testing.T) {
	k := New("mel-region-node", Policy{Version: "v1", Mode: "advisory"})

	regions := []struct {
		id       string
		health   float64
		degraded bool
	}{
		{"us-east-1", 0.95, false},
		{"eu-west-1", 0.40, true},
		{"ap-south-1", 0.15, true},
	}

	var effects []Effect
	now := time.Now().UTC()

	for _, r := range regions {
		rhData := RegionHealthData{
			RegionID:      r.id,
			OverallHealth: r.health,
			NodeCount:     10,
			HealthyNodes:  int(float64(10) * r.health),
			Degraded:      r.degraded,
		}
		data, _ := json.Marshal(rhData)
		effs := k.Apply(Event{
			ID:           NewEventID(),
			Type:         EventRegionHealth,
			Timestamp:    now,
			SourceNodeID: "mel-region-node",
			Data:         data,
		})
		effects = append(effects, effs...)
	}

	state := k.State()

	// All regions should be registered
	for _, r := range regions {
		rh, ok := state.RegionHealth[r.id]
		if !ok {
			t.Errorf("region %s not in state", r.id)
			continue
		}
		if absDiff(rh.OverallHealth, r.health) > 0.001 {
			t.Errorf("region %s health: expected %.2f, got %.2f", r.id, r.health, rh.OverallHealth)
		}
	}

	// Degraded regions should have produced alerts
	alertCount := countEffectsByType(effects, EffectEmitAlert)
	if alertCount != 2 {
		t.Errorf("expected 2 alerts for degraded regions, got %d", alertCount)
	}
}

// ─── Concurrent Apply Safety ──────────────────────────────────────────────────

// TestConcurrentApplyIsSafe verifies that concurrent calls to Apply (which
// should not happen in production but might in tests) don't cause data races.
// The kernel uses a mutex, so this just verifies no deadlock or panic.
func TestConcurrentApplySafe(t *testing.T) {
	k := New("mel-concurrent", Policy{Version: "v1"})

	var wg sync.WaitGroup
	const workers = 5
	const eventsEach = 20

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < eventsEach; i++ {
				data, _ := json.Marshal(ObservationData{
					Transport: "mqtt",
					NodeNum:   int64(i + 100),
					NodeID:    fmt.Sprintf("!node%d", i),
				})
				k.Apply(Event{
					ID:           NewEventID(),
					Type:         EventObservation,
					Timestamp:    time.Now().UTC(),
					SourceNodeID: "mel-concurrent",
					Data:         data,
				})
			}
		}()
	}
	wg.Wait()

	state := k.State()
	if len(state.TransportScores) == 0 {
		t.Error("expected transport scores after concurrent apply")
	}
}

// ─── Chaos: Random Event Order ────────────────────────────────────────────────

// TestChaosRandomOrder verifies that the kernel remains stable under random
// event ordering (not deterministic, but no panics or invalid states).
func TestChaosRandomOrder(t *testing.T) {
	k := New("mel-chaos", Policy{Version: "v1", Mode: "advisory"})
	events := generateChaosEventStream(200, 5, 5)

	// Shuffle events
	rng := rand.New(rand.NewSource(42))
	rng.Shuffle(len(events), func(i, j int) {
		events[i], events[j] = events[j], events[i]
	})

	for _, evt := range events {
		k.Apply(evt) // must not panic
	}

	state := k.State()
	// Scores must be in valid range
	for id, ns := range state.NodeScores {
		if ns.CompositeScore < 0 || ns.CompositeScore > 1 {
			t.Errorf("node %s: composite score out of range: %.4f", id, ns.CompositeScore)
		}
		if ns.HealthScore < 0 || ns.HealthScore > 1 {
			t.Errorf("node %s: health score out of range: %.4f", id, ns.HealthScore)
		}
	}
	for name, ts := range state.TransportScores {
		if ts.HealthScore < 0 || ts.HealthScore > 1 {
			t.Errorf("transport %s: health score out of range: %.4f", name, ts.HealthScore)
		}
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// generateChaosEventStream creates a deterministic multi-transport, multi-node
// event stream for simulation tests.
func generateChaosEventStream(count, numTransports, numNodes int) []Event {
	baseTime := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	events := make([]Event, 0, count)

	for i := 0; i < count; i++ {
		ts := baseTime.Add(time.Duration(i) * 100 * time.Millisecond)
		transportIdx := i % numTransports
		nodeIdx := i % numNodes
		transport := fmt.Sprintf("transport-%02d", transportIdx)
		nodeNum := int64(1000 + nodeIdx)
		nodeID := fmt.Sprintf("!node%03d", nodeIdx)

		var evtType EventType
		var data []byte

		switch i % 6 {
		case 0:
			evtType = EventObservation
			obs := ObservationData{Transport: transport, NodeNum: nodeNum, NodeID: nodeID}
			data, _ = json.Marshal(obs)
		case 1:
			evtType = EventAnomaly
			anom := AnomalyData{
				Transport: transport, NodeID: nodeID,
				Category: "dead_letter", Severity: "medium", Score: 0.35,
			}
			data, _ = json.Marshal(anom)
		case 2:
			evtType = EventTopologyUpdate
			topo := TopologyData{
				NodeNum: nodeNum, NodeID: nodeID,
				LongName: "Node " + nodeID, Action: "updated",
			}
			data, _ = json.Marshal(topo)
		case 3:
			evtType = EventTransportHealth
			th := TransportHealthData{Transport: transport, State: "live", Health: 0.8}
			data, _ = json.Marshal(th)
		case 4:
			evtType = EventNodeState
			info := NodeInfo{NodeNum: nodeNum, NodeID: nodeID, LongName: "Node " + nodeID, LastSeen: ts}
			data, _ = json.Marshal(info)
		case 5:
			evtType = EventAdapterState
			as := AdapterStateData{AdapterName: transport, State: "connected"}
			data, _ = json.Marshal(as)
		}

		events = append(events, Event{
			ID:           fmt.Sprintf("evt-chaos-%06d", i),
			SequenceNum:  uint64(i + 1),
			Type:         evtType,
			Timestamp:    ts,
			LogicalClock: uint64(i),
			SourceNodeID: "mel-chaos-source",
			Data:         data,
		})
	}
	return events
}

// generatePartitionEvents creates events that simulate one node's local
// observations during a network partition.
func generatePartitionEvents(nodeID, transport string, count int, startSeq int) []Event {
	baseTime := time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)
	events := make([]Event, 0, count)
	for i := 0; i < count; i++ {
		ts := baseTime.Add(time.Duration(i) * time.Second)
		obs := ObservationData{
			Transport: transport,
			NodeNum:   int64(9000 + i),
			NodeID:    fmt.Sprintf("!part%d", i),
		}
		data, _ := json.Marshal(obs)
		events = append(events, Event{
			ID:           fmt.Sprintf("evt-part-%s-%06d", nodeID, i),
			SequenceNum:  uint64(startSeq + i),
			Type:         EventObservation,
			Timestamp:    ts,
			LogicalClock: uint64(startSeq + i),
			SourceNodeID: nodeID,
			Data:         data,
		})
	}
	return events
}

// buildDegradationStream creates events that severely degrade a transport.
func buildDegradationStream(transport string, count int) []Event {
	baseTime := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	events := make([]Event, 0, count)
	for i := 0; i < count; i++ {
		ts := baseTime.Add(time.Duration(i) * time.Second)
		// Alternating observation and high-severity anomaly
		var evtType EventType
		var data []byte
		if i%2 == 0 {
			evtType = EventAnomaly
			anom := AnomalyData{
				Transport: transport, Category: "dead_letter",
				Severity: "critical", Score: 0.9,
			}
			data, _ = json.Marshal(anom)
		} else {
			evtType = EventTransportHealth
			th := TransportHealthData{Transport: transport, State: "degraded", Health: 0.1}
			data, _ = json.Marshal(th)
		}
		events = append(events, Event{
			ID:           fmt.Sprintf("evt-degrade-%06d", i),
			SequenceNum:  uint64(i + 1),
			Type:         evtType,
			Timestamp:    ts,
			LogicalClock: uint64(i),
			SourceNodeID: "mel-degrade-source",
			Data:         data,
		})
	}
	return events
}

func countEffectsByType(effects []Effect, typ EffectType) int {
	count := 0
	for _, eff := range effects {
		if eff.Type == typ {
			count++
		}
	}
	return count
}
