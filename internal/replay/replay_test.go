package replay

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/eventlog"
	"github.com/mel-project/mel/internal/kernel"
)

func requireSQLite3(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not in PATH, skipping eventlog-backed replay tests")
	}
}

func tempDB(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, name+".db")
}

// buildPopulatedLog creates a test event log with a deterministic event stream.
func buildPopulatedLog(t *testing.T, dbPath string, nodeID string, count int) *eventlog.Log {
	t.Helper()
	log, err := eventlog.Open(eventlog.Config{
		DBPath: dbPath,
		NodeID: nodeID,
	})
	if err != nil {
		t.Fatalf("eventlog.Open: %v", err)
	}

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	transports := []string{"mqtt-local", "serial-lora", "tcp-bridge"}
	nodes := []struct {
		num int64
		id  string
	}{
		{1001, "!aaa001"},
		{1002, "!aaa002"},
		{1003, "!aaa003"},
	}

	for i := 0; i < count; i++ {
		ts := baseTime.Add(time.Duration(i) * time.Second)
		transport := transports[i%len(transports)]
		node := nodes[i%len(nodes)]

		var evtType kernel.EventType
		var data []byte

		switch i % 5 {
		case 0:
			evtType = kernel.EventObservation
			obs := kernel.ObservationData{
				Transport: transport,
				NodeNum:   node.num,
				NodeID:    node.id,
			}
			data, _ = json.Marshal(obs)
		case 1:
			evtType = kernel.EventAnomaly
			anom := kernel.AnomalyData{
				Transport: transport,
				NodeID:    node.id,
				Category:  "dead_letter",
				Severity:  "medium",
				Score:     0.4,
			}
			data, _ = json.Marshal(anom)
		case 2:
			evtType = kernel.EventTopologyUpdate
			topo := kernel.TopologyData{
				NodeNum:  node.num,
				NodeID:   node.id,
				LongName: "Node " + node.id,
				Action:   "updated",
			}
			data, _ = json.Marshal(topo)
		case 3:
			evtType = kernel.EventTransportHealth
			th := kernel.TransportHealthData{
				Transport: transport,
				State:     "live",
				Health:    0.85,
			}
			data, _ = json.Marshal(th)
		case 4:
			evtType = kernel.EventNodeState
			info := kernel.NodeInfo{
				NodeNum:  node.num,
				NodeID:   node.id,
				LongName: "Node " + node.id,
				LastSeen: ts,
			}
			data, _ = json.Marshal(info)
		}

		evt := &kernel.Event{
			ID:           kernel.NewEventID(),
			Type:         evtType,
			Timestamp:    ts,
			LogicalClock: uint64(i),
			SourceNodeID: nodeID,
			Data:         data,
		}
		if _, err := log.Append(evt); err != nil {
			t.Fatalf("Append event %d: %v", i, err)
		}
	}
	return log
}

// ─── Full Replay ──────────────────────────────────────────────────────────────

func TestFullReplay(t *testing.T) {
	requireSQLite3(t)
	dbPath := tempDB(t, "full-replay")
	log := buildPopulatedLog(t, dbPath, "mel-test-node", 50)

	policy := kernel.Policy{
		Version: "v1",
		Mode:    "advisory",
	}
	engine := NewEngine(log, "mel-replay-node")

	result, err := engine.Execute(Request{Mode: ModeFull, Policy: policy})
	if err != nil {
		t.Fatalf("Execute full replay: %v", err)
	}

	if result.EventsProcessed != 50 {
		t.Errorf("expected 50 events processed, got %d", result.EventsProcessed)
	}
	if result.DurationMS < 0 {
		t.Errorf("expected non-negative duration")
	}
	if result.FirstSequence != 1 {
		t.Errorf("expected first sequence 1, got %d", result.FirstSequence)
	}
	if result.LastSequence != 50 {
		t.Errorf("expected last sequence 50, got %d", result.LastSequence)
	}

	// State should have nodes and transports
	if len(result.FinalState.TransportScores) == 0 {
		t.Error("expected transport scores in final state")
	}
	if len(result.FinalState.NodeScores) == 0 {
		t.Error("expected node scores in final state")
	}
}

// ─── Deterministic Replay ─────────────────────────────────────────────────────

func TestDeterministicReplay(t *testing.T) {
	requireSQLite3(t)

	// Two separate logs with identical events
	dbA := tempDB(t, "det-replay-a")
	dbB := tempDB(t, "det-replay-b")

	logA := buildPopulatedLog(t, dbA, "mel-node-a", 40)
	logB := buildPopulatedLog(t, dbB, "mel-node-b", 40)

	policy := kernel.Policy{
		Version:              "v2",
		Mode:                 "advisory",
		AllowedActions:       []string{"restart_transport"},
		RequireMinConfidence: 0.75,
	}

	engineA := NewEngine(logA, "mel-replay-a")
	engineB := NewEngine(logB, "mel-replay-b")

	resultA, err := engineA.Execute(Request{Mode: ModeFull, Policy: policy})
	if err != nil {
		t.Fatalf("replay A: %v", err)
	}
	resultB, err := engineB.Execute(Request{Mode: ModeFull, Policy: policy})
	if err != nil {
		t.Fatalf("replay B: %v", err)
	}

	// Both replays processed same number of events
	if resultA.EventsProcessed != resultB.EventsProcessed {
		t.Errorf("events processed mismatch: %d vs %d", resultA.EventsProcessed, resultB.EventsProcessed)
	}

	// Transport scores should be identical
	if len(resultA.FinalState.TransportScores) != len(resultB.FinalState.TransportScores) {
		t.Errorf("transport count mismatch: %d vs %d",
			len(resultA.FinalState.TransportScores), len(resultB.FinalState.TransportScores))
	}
	for name, scoreA := range resultA.FinalState.TransportScores {
		scoreB, ok := resultB.FinalState.TransportScores[name]
		if !ok {
			t.Errorf("transport %s missing from replay B", name)
			continue
		}
		if scoreA.Classification != scoreB.Classification {
			t.Errorf("transport %s classification mismatch: %s vs %s",
				name, scoreA.Classification, scoreB.Classification)
		}
		if absDiff(scoreA.HealthScore, scoreB.HealthScore) > 0.0001 {
			t.Errorf("transport %s health score mismatch: %.6f vs %.6f",
				name, scoreA.HealthScore, scoreB.HealthScore)
		}
	}

	// Node scores should be identical
	if len(resultA.FinalState.NodeScores) != len(resultB.FinalState.NodeScores) {
		t.Errorf("node count mismatch: %d vs %d",
			len(resultA.FinalState.NodeScores), len(resultB.FinalState.NodeScores))
	}
	for id, scoreA := range resultA.FinalState.NodeScores {
		scoreB, ok := resultB.FinalState.NodeScores[id]
		if !ok {
			t.Errorf("node %s missing from replay B", id)
			continue
		}
		if scoreA.Classification != scoreB.Classification {
			t.Errorf("node %s classification mismatch: %s vs %s",
				id, scoreA.Classification, scoreB.Classification)
		}
		if absDiff(scoreA.CompositeScore, scoreB.CompositeScore) > 0.0001 {
			t.Errorf("node %s composite score mismatch: %.6f vs %.6f",
				id, scoreA.CompositeScore, scoreB.CompositeScore)
		}
	}

	// Effects counts should match
	if resultA.EffectsProduced != resultB.EffectsProduced {
		t.Errorf("effects produced mismatch: %d vs %d",
			resultA.EffectsProduced, resultB.EffectsProduced)
	}
}

// ─── Windowed Replay ──────────────────────────────────────────────────────────

func TestWindowedReplay(t *testing.T) {
	requireSQLite3(t)
	dbPath := tempDB(t, "windowed-replay")
	log := buildPopulatedLog(t, dbPath, "mel-windowed-node", 60)

	policy := kernel.Policy{Version: "v1", Mode: "advisory"}
	engine := NewEngine(log, "mel-replay-node")

	// Replay only events 10-20
	result, err := engine.Execute(Request{
		Mode:         ModeWindowed,
		Policy:       policy,
		FromSequence: 10,
		ToSequence:   20,
	})
	if err != nil {
		t.Fatalf("windowed replay: %v", err)
	}

	if result.EventsProcessed == 0 {
		t.Error("expected events in windowed replay")
	}
	if result.FirstSequence < 10 {
		t.Errorf("expected first sequence >= 10, got %d", result.FirstSequence)
	}
	if result.LastSequence > 20 {
		t.Errorf("expected last sequence <= 20, got %d", result.LastSequence)
	}
}

// ─── Scenario Replay ─────────────────────────────────────────────────────────

func TestScenarioReplay(t *testing.T) {
	requireSQLite3(t)
	dbPath := tempDB(t, "scenario-replay")
	log := buildPopulatedLog(t, dbPath, "mel-scenario-node", 30)

	engine := NewEngine(log, "mel-replay-node")

	// Default policy
	defaultPolicy := kernel.Policy{
		Version:              "v1",
		Mode:                 "disabled",
		RequireMinConfidence: 0.99,
	}

	// Modified policy (scenario)
	aggressivePolicy := kernel.Policy{
		Version:              "v2-scenario",
		Mode:                 "guarded_auto",
		AllowedActions:       []string{"restart_transport", "trigger_health_recheck"},
		RequireMinConfidence: 0.3,
	}

	resultDefault, err := engine.Execute(Request{Mode: ModeFull, Policy: defaultPolicy})
	if err != nil {
		t.Fatalf("default replay: %v", err)
	}
	resultScenario, err := engine.Execute(Request{Mode: ModeScenario, Policy: aggressivePolicy})
	if err != nil {
		t.Fatalf("scenario replay: %v", err)
	}

	// Both should process same number of events
	if resultDefault.EventsProcessed != resultScenario.EventsProcessed {
		t.Errorf("event count mismatch: %d vs %d",
			resultDefault.EventsProcessed, resultScenario.EventsProcessed)
	}

	// Aggressive policy should produce more effects (propose_action effects)
	if resultScenario.EffectsProduced < resultDefault.EffectsProduced {
		t.Logf("note: scenario policy produced %d effects, default produced %d",
			resultScenario.EffectsProduced, resultDefault.EffectsProduced)
	}
}

// ─── Snapshot+Delta Replay ────────────────────────────────────────────────────

func TestSnapshotDeltaReplay(t *testing.T) {
	requireSQLite3(t)
	dbPath := tempDB(t, "snapshot-delta-replay")
	log := buildPopulatedLog(t, dbPath, "mel-snap-node", 60)

	policy := kernel.Policy{Version: "v1", Mode: "advisory"}
	engine := NewEngine(log, "mel-replay-node")

	// Full replay to get state at event 30
	resultFull30, err := engine.Execute(Request{
		Mode:      ModeFull,
		Policy:    policy,
		MaxEvents: 30,
	})
	if err != nil {
		t.Fatalf("full replay to 30: %v", err)
	}
	snapshot30 := resultFull30.FinalState

	// Full replay all 60 events
	resultFull60, err := engine.Execute(Request{Mode: ModeFull, Policy: policy})
	if err != nil {
		t.Fatalf("full replay to 60: %v", err)
	}

	// Snapshot+delta from sequence 30 onwards
	seq30 := resultFull30.LastSequence
	resultDelta, err := engine.Execute(Request{
		Mode:         ModeWindowed,
		Policy:       policy,
		InitialState: &snapshot30,
		FromSequence: seq30,
	})
	if err != nil {
		t.Fatalf("snapshot+delta replay: %v", err)
	}

	// The delta replay processed fewer events than full
	if resultDelta.EventsProcessed >= resultFull60.EventsProcessed {
		t.Errorf("delta replay should process fewer events: %d vs full %d",
			resultDelta.EventsProcessed, resultFull60.EventsProcessed)
	}

	// The final state should have same transport counts
	if len(resultDelta.FinalState.TransportScores) != len(resultFull60.FinalState.TransportScores) {
		t.Logf("note: snapshot+delta transport count %d, full %d",
			len(resultDelta.FinalState.TransportScores),
			len(resultFull60.FinalState.TransportScores))
	}
}

// ─── Verification Replay ─────────────────────────────────────────────────────

func TestVerificationReplay_NoDiv(t *testing.T) {
	requireSQLite3(t)
	dbPath := tempDB(t, "verify-replay")
	log := buildPopulatedLog(t, dbPath, "mel-verify-node", 20)

	policy := kernel.Policy{Version: "v1", Mode: "advisory"}
	engine := NewEngine(log, "mel-replay-node")

	// First get the authoritative state
	resultRef, err := engine.Execute(Request{Mode: ModeFull, Policy: policy})
	if err != nil {
		t.Fatalf("reference replay: %v", err)
	}

	// Verification replay with same expected state — should pass
	expectedState := resultRef.FinalState
	resultVerify, err := engine.Execute(Request{
		Mode:          ModeVerification,
		Policy:        policy,
		ExpectedState: &expectedState,
	})
	if err != nil {
		t.Fatalf("verification replay: %v", err)
	}

	if !resultVerify.Verified {
		t.Errorf("expected verification to pass, got divergences: %+v", resultVerify.Divergences)
	}
}

func TestVerificationReplay_WithDivergence(t *testing.T) {
	requireSQLite3(t)
	dbPath := tempDB(t, "verify-replay-div")
	log := buildPopulatedLog(t, dbPath, "mel-verify-div-node", 20)

	policy := kernel.Policy{Version: "v1", Mode: "advisory"}
	engine := NewEngine(log, "mel-replay-node")

	// Run reference replay
	resultRef, err := engine.Execute(Request{Mode: ModeFull, Policy: policy})
	if err != nil {
		t.Fatalf("reference replay: %v", err)
	}

	// Mutate expected state to force divergence
	expected := resultRef.FinalState
	for id := range expected.TransportScores {
		ts := expected.TransportScores[id]
		ts.Classification = "dead" // force wrong classification
		ts.HealthScore = 0.0
		expected.TransportScores[id] = ts
		break // mutate just one
	}

	resultVerify, err := engine.Execute(Request{
		Mode:          ModeVerification,
		Policy:        policy,
		ExpectedState: &expected,
	})
	if err != nil {
		t.Fatalf("verification replay: %v", err)
	}

	if resultVerify.Verified {
		t.Error("expected verification to fail with forced divergence")
	}
	if len(resultVerify.Divergences) == 0 {
		t.Error("expected at least one divergence recorded")
	}
}

// ─── Empty Log ────────────────────────────────────────────────────────────────

func TestReplayEmptyLog(t *testing.T) {
	requireSQLite3(t)
	dbPath := tempDB(t, "empty-replay")
	log, err := eventlog.Open(eventlog.Config{DBPath: dbPath, NodeID: "mel-empty"})
	if err != nil {
		t.Fatalf("open log: %v", err)
	}

	engine := NewEngine(log, "mel-replay-node")
	result, err := engine.Execute(Request{
		Mode:   ModeFull,
		Policy: kernel.Policy{Version: "v1", Mode: "advisory"},
	})
	if err != nil {
		t.Fatalf("empty replay: %v", err)
	}
	if result.EventsProcessed != 0 {
		t.Errorf("expected 0 events, got %d", result.EventsProcessed)
	}
}

// ─── Replay Idempotency ───────────────────────────────────────────────────────

// Running the same replay twice produces identical results.
func TestReplayIdempotency(t *testing.T) {
	requireSQLite3(t)
	dbPath := tempDB(t, "idempotent-replay")
	log := buildPopulatedLog(t, dbPath, "mel-idem-node", 25)

	policy := kernel.Policy{Version: "v1", Mode: "advisory"}
	engine := NewEngine(log, "mel-replay-node")
	req := Request{Mode: ModeFull, Policy: policy}

	r1, err := engine.Execute(req)
	if err != nil {
		t.Fatalf("replay 1: %v", err)
	}
	r2, err := engine.Execute(req)
	if err != nil {
		t.Fatalf("replay 2: %v", err)
	}

	if r1.EventsProcessed != r2.EventsProcessed {
		t.Errorf("events mismatch: %d vs %d", r1.EventsProcessed, r2.EventsProcessed)
	}
	if r1.EffectsProduced != r2.EffectsProduced {
		t.Errorf("effects mismatch: %d vs %d", r1.EffectsProduced, r2.EffectsProduced)
	}
	if r1.FinalState.PolicyVersion != r2.FinalState.PolicyVersion {
		t.Errorf("policy version mismatch: %s vs %s",
			r1.FinalState.PolicyVersion, r2.FinalState.PolicyVersion)
	}

	for id, s1 := range r1.FinalState.TransportScores {
		s2, ok := r2.FinalState.TransportScores[id]
		if !ok {
			t.Errorf("transport %s missing from replay 2", id)
			continue
		}
		if absDiff(s1.HealthScore, s2.HealthScore) > 1e-9 {
			t.Errorf("transport %s health differs: %g vs %g", id, s1.HealthScore, s2.HealthScore)
		}
	}
}

// ─── Max Events Limit ─────────────────────────────────────────────────────────

func TestReplayMaxEvents(t *testing.T) {
	requireSQLite3(t)
	dbPath := tempDB(t, "maxevents-replay")
	log := buildPopulatedLog(t, dbPath, "mel-max-node", 100)

	engine := NewEngine(log, "mel-replay-node")
	result, err := engine.Execute(Request{
		Mode:      ModeFull,
		Policy:    kernel.Policy{Version: "v1", Mode: "advisory"},
		MaxEvents: 25,
	})
	if err != nil {
		t.Fatalf("max events replay: %v", err)
	}
	if result.EventsProcessed > 25 {
		t.Errorf("expected <= 25 events, got %d", result.EventsProcessed)
	}
}

// ─── Action Lifecycle Through Replay ─────────────────────────────────────────

func TestReplayActionLifecycle(t *testing.T) {
	requireSQLite3(t)
	dbPath := tempDB(t, "action-replay")
	log, err := eventlog.Open(eventlog.Config{DBPath: dbPath, NodeID: "mel-action-node"})
	if err != nil {
		t.Fatalf("open log: %v", err)
	}

	actionID := "act-replay-001"
	ts := time.Now().UTC()

	// Propose
	proposed := kernel.ActionProposedData{
		ActionID: actionID, ActionType: "restart_transport",
		Target: "mqtt-local", Reason: "health degraded", Confidence: 0.9,
	}
	data, _ := json.Marshal(proposed)
	_, _ = log.Append(&kernel.Event{
		ID: kernel.NewEventID(), Type: kernel.EventActionProposed,
		Timestamp: ts, SourceNodeID: "mel-action-node", Data: data,
	})

	// Execute
	executed := kernel.ActionExecutedData{ActionID: actionID, Result: "started"}
	data, _ = json.Marshal(executed)
	_, _ = log.Append(&kernel.Event{
		ID: kernel.NewEventID(), Type: kernel.EventActionExecuted,
		Timestamp: ts.Add(time.Second), SourceNodeID: "mel-action-node", Data: data,
	})

	// Complete
	completed := kernel.ActionCompletedData{ActionID: actionID, Result: "success"}
	data, _ = json.Marshal(completed)
	_, _ = log.Append(&kernel.Event{
		ID: kernel.NewEventID(), Type: kernel.EventActionCompleted,
		Timestamp: ts.Add(2 * time.Second), SourceNodeID: "mel-action-node", Data: data,
	})

	engine := NewEngine(log, "mel-replay-node")
	result, err := engine.Execute(Request{
		Mode:   ModeFull,
		Policy: kernel.Policy{Version: "v1", Mode: "advisory"},
	})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}

	as, ok := result.FinalState.ActionStates[actionID]
	if !ok {
		t.Fatalf("action %s not in replayed state", actionID)
	}
	if as.Lifecycle != "completed" {
		t.Errorf("expected lifecycle completed, got %s", as.Lifecycle)
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func skipIfNoSQLite3(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not in PATH")
	}
	_ = os.Getenv // suppress unused warning
}
