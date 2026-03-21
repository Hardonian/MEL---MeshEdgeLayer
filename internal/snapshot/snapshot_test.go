package snapshot

import (
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/kernel"
)

func requireSQLite3(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not in PATH, skipping snapshot tests")
	}
}

func tempDB(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "snapshot-test.db")
}

func buildTestState() *kernel.State {
	state := kernel.NewState()
	state.PolicyVersion = "v1"
	state.LogicalClock = 42
	state.LastEventID = "evt-test-001"
	state.LastSequenceNum = 100

	state.NodeScores["!node001"] = kernel.NodeScore{
		NodeID:         "!node001",
		Transport:      "mqtt-local",
		HealthScore:    0.85,
		TrustScore:     0.7,
		ActivityScore:  0.9,
		AnomalyScore:   0.1,
		CompositeScore: 0.82,
		Classification: "healthy",
		UpdatedAt:      time.Now().UTC(),
	}
	state.TransportScores["mqtt-local"] = kernel.TransportScore{
		Transport:      "mqtt-local",
		HealthScore:    0.90,
		ReliabilityPct: 0.95,
		AnomalyRate:    0.05,
		Classification: "healthy",
		UpdatedAt:      time.Now().UTC(),
	}
	state.NodeRegistry[1001] = kernel.NodeInfo{
		NodeNum:   1001,
		NodeID:    "!node001",
		LongName:  "Test Node 1",
		ShortName: "TN1",
		LastSeen:  time.Now().UTC(),
		Region:    "us-east",
	}
	return state
}

// TestCreateAndList verifies basic snapshot creation and listing.
func TestCreateAndList(t *testing.T) {
	requireSQLite3(t)

	store, err := NewStore(tempDB(t), "mel-test-node")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	state := buildTestState()
	snap, err := store.Create(state, 100)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if snap.ID == "" {
		t.Error("expected non-empty snapshot ID")
	}
	if snap.NodeID != "mel-test-node" {
		t.Errorf("expected node ID mel-test-node, got %s", snap.NodeID)
	}
	if snap.SequenceNum != 100 {
		t.Errorf("expected sequence num 100, got %d", snap.SequenceNum)
	}
	if snap.IntegrityHash == "" {
		t.Error("expected non-empty integrity hash")
	}
	if snap.PolicyVersion != "v1" {
		t.Errorf("expected policy version v1, got %s", snap.PolicyVersion)
	}

	// List snapshots
	snaps, err := store.List(10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(snaps) != 1 {
		t.Errorf("expected 1 snapshot, got %d", len(snaps))
	}
}

// TestLatest verifies that Latest() returns the most recent snapshot.
func TestLatest(t *testing.T) {
	requireSQLite3(t)

	store, err := NewStore(tempDB(t), "mel-latest-node")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	state := buildTestState()

	// Create 3 snapshots
	for i := 1; i <= 3; i++ {
		state.LastSequenceNum = uint64(i * 100)
		if _, err := store.Create(state, uint64(i*100)); err != nil {
			t.Fatalf("Create snap %d: %v", i, err)
		}
		time.Sleep(time.Millisecond) // ensure distinct timestamps
	}

	latest, err := store.Latest()
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if latest == nil {
		t.Fatal("expected non-nil latest snapshot")
	}
	if latest.SequenceNum != 300 {
		t.Errorf("expected latest seq 300, got %d", latest.SequenceNum)
	}
}

// TestVerify verifies that snapshot integrity checking works.
func TestVerify(t *testing.T) {
	requireSQLite3(t)

	store, err := NewStore(tempDB(t), "mel-verify-node")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	state := buildTestState()
	snap, err := store.Create(state, 50)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Verify the snapshot as-stored — should pass
	ok, err := store.Verify(snap)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !ok {
		t.Error("expected verification to pass for intact snapshot")
	}

	// Tamper with the hash — verification should fail
	tampered := *snap
	tampered.IntegrityHash = "000000000000000000000000000000000000000000000000000000000000dead"
	ok, err = store.Verify(&tampered)
	if err != nil {
		t.Fatalf("Verify tampered: %v", err)
	}
	if ok {
		t.Error("expected verification to fail for tampered snapshot")
	}
}

// TestByID verifies fetching a snapshot by its ID.
func TestByID(t *testing.T) {
	requireSQLite3(t)

	store, err := NewStore(tempDB(t), "mel-byid-node")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	state := buildTestState()
	snap, err := store.Create(state, 42)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fetched, err := store.ByID(snap.ID)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected non-nil snapshot")
	}
	if fetched.ID != snap.ID {
		t.Errorf("expected ID %s, got %s", snap.ID, fetched.ID)
	}
	if fetched.IntegrityHash != snap.IntegrityHash {
		t.Errorf("integrity hash mismatch: %s vs %s", snap.IntegrityHash, fetched.IntegrityHash)
	}

	// Non-existent ID should return nil, no error
	notFound, err := store.ByID("non-existent-id")
	if err != nil {
		t.Errorf("expected nil error for missing ID, got %v", err)
	}
	if notFound != nil {
		t.Error("expected nil for non-existent snapshot ID")
	}
}

// TestStateRoundTrip verifies that state serialized into a snapshot can be
// deserialized and used to restore kernel state correctly.
func TestStateRoundTrip(t *testing.T) {
	requireSQLite3(t)

	store, err := NewStore(tempDB(t), "mel-roundtrip-node")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	original := buildTestState()
	snap, err := store.Create(original, 99)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Fetch and verify state is intact
	fetched, err := store.ByID(snap.ID)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}

	// Node scores should match
	if len(fetched.State.NodeScores) != len(original.NodeScores) {
		t.Errorf("node scores count: %d vs %d",
			len(fetched.State.NodeScores), len(original.NodeScores))
	}
	for id, orig := range original.NodeScores {
		fetched_, ok := fetched.State.NodeScores[id]
		if !ok {
			t.Errorf("node %s missing from snapshot", id)
			continue
		}
		if fetched_.Classification != orig.Classification {
			t.Errorf("node %s classification mismatch: %s vs %s",
				id, fetched_.Classification, orig.Classification)
		}
	}

	// Transport scores should match
	for name, orig := range original.TransportScores {
		fetched_, ok := fetched.State.TransportScores[name]
		if !ok {
			t.Errorf("transport %s missing from snapshot", name)
			continue
		}
		if fetched_.Classification != orig.Classification {
			t.Errorf("transport %s classification mismatch: %s vs %s",
				name, fetched_.Classification, orig.Classification)
		}
	}

	// Policy version preserved
	if fetched.State.PolicyVersion != original.PolicyVersion {
		t.Errorf("policy version: %s vs %s", fetched.State.PolicyVersion, original.PolicyVersion)
	}
}

// TestPrune verifies that old snapshots are pruned while keeping a minimum.
func TestPrune(t *testing.T) {
	requireSQLite3(t)

	store, err := NewStore(tempDB(t), "mel-prune-node")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	state := buildTestState()

	// Create 5 snapshots
	for i := 0; i < 5; i++ {
		if _, err := store.Create(state, uint64(i+1)); err != nil {
			t.Fatalf("Create snap %d: %v", i, err)
		}
		time.Sleep(2 * time.Millisecond)
	}

	snaps, _ := store.List(10)
	if len(snaps) != 5 {
		t.Fatalf("expected 5 snapshots before prune, got %d", len(snaps))
	}

	// Prune snapshots older than now, keep minimum 2
	err = store.Prune(time.Now().Add(time.Hour), 2)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}

	remaining, _ := store.List(10)
	if len(remaining) < 2 {
		t.Errorf("expected at least 2 remaining after prune, got %d", len(remaining))
	}
}

// TestSnapshotIntegrityHash verifies determinism: same state produces same hash.
func TestSnapshotIntegrityHashDeterminism(t *testing.T) {
	requireSQLite3(t)

	storeA, _ := NewStore(tempDB(t), "mel-hash-node")
	storeB, _ := NewStore(tempDB(t), "mel-hash-node")

	state := buildTestState()
	snapA, err := storeA.Create(state, 10)
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	snapB, err := storeB.Create(state, 10)
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}

	// Snapshots of identical state should have identical integrity hashes
	if snapA.IntegrityHash != snapB.IntegrityHash {
		t.Errorf("integrity hash should be deterministic for same state: %s vs %s",
			snapA.IntegrityHash, snapB.IntegrityHash)
	}
}
