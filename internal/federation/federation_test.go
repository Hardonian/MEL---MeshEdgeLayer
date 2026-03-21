package federation

import (
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/kernel"
	"github.com/mel-project/mel/internal/logging"
)

func requireSQLite3(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not found in PATH")
	}
}

func tempDB(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "test-federation.db")
}

func TestSyncScopeMatches(t *testing.T) {
	scope := SyncScope{
		EventTypes:   []string{"observation", "anomaly"},
		Regions:      []string{"us-east", "eu-west"},
		ExcludeTypes: []string{"operator_action"},
	}

	tests := []struct {
		eventType string
		region    string
		transport string
		expected  bool
	}{
		{"observation", "us-east", "mqtt", true},
		{"anomaly", "eu-west", "", true},
		{"topology_update", "us-east", "", false}, // not in allowed types
		{"observation", "ap-south", "", false},    // not in allowed regions
		{"operator_action", "us-east", "", false}, // excluded
	}

	for _, tt := range tests {
		result := scope.Matches(tt.eventType, tt.region, tt.transport)
		if result != tt.expected {
			t.Errorf("Matches(%s, %s, %s) = %v, want %v",
				tt.eventType, tt.region, tt.transport, result, tt.expected)
		}
	}
}

func TestSyncScopeEmpty(t *testing.T) {
	scope := SyncScope{} // empty = match all
	if !scope.Matches("anything", "anywhere", "any") {
		t.Error("empty scope should match everything")
	}
}

func TestManagerPeerManagement(t *testing.T) {
	requireSQLite3(t)
	log := logging.New("error", false)
	dbPath := tempDB(t)

	m, err := NewManager(Config{
		Enabled: true,
		NodeID:  "mel-node-1",
		Region:  "us-east",
	}, log, nil, nil, dbPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if m.NodeID() != "mel-node-1" {
		t.Errorf("expected node ID mel-node-1, got %s", m.NodeID())
	}

	// Add peer
	err = m.AddPeer(Peer{
		NodeID:     "mel-node-2",
		Name:       "Node 2",
		Endpoint:   "http://node2:8080",
		Region:     "eu-west",
		TrustLevel: 2,
	})
	if err != nil {
		t.Fatalf("AddPeer failed: %v", err)
	}

	peers := m.Peers()
	if len(peers) != 1 {
		t.Errorf("expected 1 peer, got %d", len(peers))
	}

	// Can't add self
	err = m.AddPeer(Peer{NodeID: "mel-node-1"})
	if err == nil {
		t.Error("expected error when adding self as peer")
	}

	// Get by ID
	peer, ok := m.PeerByID("mel-node-2")
	if !ok {
		t.Fatal("expected to find peer mel-node-2")
	}
	if peer.Region != "eu-west" {
		t.Errorf("expected region eu-west, got %s", peer.Region)
	}

	// Remove peer
	m.RemovePeer("mel-node-2")
	peers = m.Peers()
	if len(peers) != 0 {
		t.Errorf("expected 0 peers after remove, got %d", len(peers))
	}
}

func TestHeartbeat(t *testing.T) {
	requireSQLite3(t)
	log := logging.New("error", false)
	dbPath := tempDB(t)

	m, err := NewManager(Config{
		Enabled: true,
		NodeID:  "mel-node-1",
		Region:  "us-east",
	}, log, nil, nil, dbPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	hb := m.GenerateHeartbeat()
	if hb.NodeID != "mel-node-1" {
		t.Errorf("expected node ID mel-node-1, got %s", hb.NodeID)
	}
	if hb.Region != "us-east" {
		t.Errorf("expected region us-east, got %s", hb.Region)
	}
	if hb.State != "healthy" {
		t.Errorf("expected state healthy, got %s", hb.State)
	}

	// Process heartbeat from unknown peer -> auto-register
	m.ProcessHeartbeat(Heartbeat{
		NodeID:    "mel-node-3",
		Region:    "ap-south",
		Timestamp: time.Now().UTC(),
		State:     "healthy",
	})

	peer, ok := m.PeerByID("mel-node-3")
	if !ok {
		t.Fatal("expected auto-registered peer mel-node-3")
	}
	if peer.State != PeerStateActive {
		t.Errorf("expected peer state active, got %s", string(peer.State))
	}
	if peer.TrustLevel != 1 {
		t.Errorf("expected trust level 1 for auto-registered, got %d", peer.TrustLevel)
	}
}

func TestPartitionDetection(t *testing.T) {
	requireSQLite3(t)
	log := logging.New("error", false)
	dbPath := tempDB(t)

	m, err := NewManager(Config{
		Enabled:                  true,
		NodeID:                   "mel-node-1",
		Region:                   "us-east",
		HeartbeatIntervalSeconds: 1,
		SuspectAfterMissed:       2,
		PartitionAfterMissed:     4,
		SplitBrainPolicy: SplitBrainPolicy{
			RestrictAutopilot:    true,
			MaxAutonomousActions: 3,
		},
	}, log, nil, nil, dbPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Add peers with old last-seen times
	past := time.Now().UTC().Add(-10 * time.Second)
	for _, id := range []string{"node-a", "node-b", "node-c"} {
		_ = m.AddPeer(Peer{
			NodeID:     id,
			Endpoint:   "http://" + id + ":8080",
			State:      PeerStateActive,
			LastSeen:   past,
			TrustLevel: 2,
		})
	}

	// Check partitions
	conflicts := m.CheckPartitions()

	// All peers should be partitioned -> split-brain
	if !m.IsSplitBrain() {
		t.Error("expected split-brain to be detected")
	}
	if len(conflicts) == 0 {
		t.Error("expected at least one conflict")
	}

	// Should restrict autonomous actions
	if !m.CanExecuteAutonomously() {
		// Initially should be allowed (under limit)
	}

	// Record max autonomous actions
	for i := 0; i < 3; i++ {
		m.RecordAutonomousAction()
	}

	if m.CanExecuteAutonomously() {
		t.Error("expected autonomous execution to be blocked after max actions")
	}
}

func TestFederationStatus(t *testing.T) {
	requireSQLite3(t)
	log := logging.New("error", false)
	dbPath := tempDB(t)

	m, err := NewManager(Config{
		Enabled: true,
		NodeID:  "mel-node-1",
		Region:  "us-east",
	}, log, nil, nil, dbPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	_ = m.AddPeer(Peer{
		NodeID:     "mel-node-2",
		Endpoint:   "http://node2:8080",
		Region:     "eu-west",
		TrustLevel: 2,
	})

	status := m.Status()
	if status.NodeID != "mel-node-1" {
		t.Errorf("expected node mel-node-1, got %s", status.NodeID)
	}
	if !status.Enabled {
		t.Error("expected enabled")
	}
	if status.PeerCount != 1 {
		t.Errorf("expected 1 peer, got %d", status.PeerCount)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Enabled {
		t.Error("default should be disabled")
	}
	if cfg.HeartbeatIntervalSeconds <= 0 {
		t.Error("expected positive heartbeat interval")
	}
	if !cfg.SplitBrainPolicy.RestrictAutopilot {
		t.Error("expected restrict_autopilot to be true by default")
	}
}

// Ensure IDs are non-empty
func TestIDGeneration(t *testing.T) {
	id := kernel.NewNodeID()
	if id == "" {
		t.Error("expected non-empty node ID")
	}
	eid := kernel.NewEventID()
	if eid == "" {
		t.Error("expected non-empty event ID")
	}
}
