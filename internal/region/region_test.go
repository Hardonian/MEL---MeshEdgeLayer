package region

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
		t.Skip("sqlite3 not found in PATH")
	}
}

func tempDB(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "test-region.db")
}

func TestNewManager(t *testing.T) {
	requireSQLite3(t)
	m, err := NewManager("us-east", tempDB(t))
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	if m.LocalRegion() != "us-east" {
		t.Errorf("expected local region us-east, got %s", m.LocalRegion())
	}
}

func TestRegionHealth(t *testing.T) {
	requireSQLite3(t)
	m, err := NewManager("us-east", tempDB(t))
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create kernel state with nodes in different regions
	state := kernel.NewState()
	state.NodeRegistry[1] = kernel.NodeInfo{NodeNum: 1, NodeID: "!n1", Region: "us-east", LastSeen: time.Now().UTC()}
	state.NodeRegistry[2] = kernel.NodeInfo{NodeNum: 2, NodeID: "!n2", Region: "us-east", LastSeen: time.Now().UTC()}
	state.NodeRegistry[3] = kernel.NodeInfo{NodeNum: 3, NodeID: "!n3", Region: "eu-west", LastSeen: time.Now().UTC()}
	state.NodeScores["!n1"] = kernel.NodeScore{NodeID: "!n1", Classification: "healthy"}
	state.NodeScores["!n2"] = kernel.NodeScore{NodeID: "!n2", Classification: "degraded"}
	state.NodeScores["!n3"] = kernel.NodeScore{NodeID: "!n3", Classification: "healthy"}

	m.UpdateHealthFromKernel(state)

	// Check us-east health
	h, ok := m.RegionHealth("us-east")
	if !ok {
		t.Fatal("expected us-east health")
	}
	if h.NodeCount != 2 {
		t.Errorf("expected 2 nodes in us-east, got %d", h.NodeCount)
	}
	if h.HealthyNodes != 1 {
		t.Errorf("expected 1 healthy node in us-east, got %d", h.HealthyNodes)
	}
	if h.DegradedNodes != 1 {
		t.Errorf("expected 1 degraded node in us-east, got %d", h.DegradedNodes)
	}

	// Check eu-west health
	h2, ok := m.RegionHealth("eu-west")
	if !ok {
		t.Fatal("expected eu-west health")
	}
	if h2.NodeCount != 1 {
		t.Errorf("expected 1 node in eu-west, got %d", h2.NodeCount)
	}
}

func TestGlobalTopology(t *testing.T) {
	requireSQLite3(t)
	m, err := NewManager("us-east", tempDB(t))
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	state := kernel.NewState()
	state.NodeRegistry[1] = kernel.NodeInfo{NodeNum: 1, NodeID: "!n1", Region: "us-east", LastSeen: time.Now().UTC()}
	state.NodeRegistry[2] = kernel.NodeInfo{NodeNum: 2, NodeID: "!n2", Region: "eu-west", LastSeen: time.Now().UTC()}
	state.NodeScores["!n1"] = kernel.NodeScore{NodeID: "!n1", Classification: "healthy"}
	state.NodeScores["!n2"] = kernel.NodeScore{NodeID: "!n2", Classification: "healthy"}

	m.UpdateHealthFromKernel(state)

	topo := m.ComputeGlobalTopology()
	if topo.TotalNodes != 2 {
		t.Errorf("expected 2 total nodes, got %d", topo.TotalNodes)
	}
	if topo.GlobalHealth != 1.0 {
		t.Errorf("expected global health 1.0, got %.2f", topo.GlobalHealth)
	}
	if len(topo.Regions) < 2 {
		t.Errorf("expected at least 2 regions, got %d", len(topo.Regions))
	}
}

func TestDegradedRegion(t *testing.T) {
	requireSQLite3(t)
	m, err := NewManager("us-east", tempDB(t))
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	state := kernel.NewState()
	// All nodes failing in us-east
	for i := int64(1); i <= 4; i++ {
		state.NodeRegistry[i] = kernel.NodeInfo{NodeNum: i, NodeID: "!n" + string(rune('0'+i)), Region: "us-east"}
		state.NodeScores["!n"+string(rune('0'+i))] = kernel.NodeScore{Classification: "failing"}
	}
	m.UpdateHealthFromKernel(state)

	if !m.IsRegionDegraded("us-east") {
		t.Error("expected us-east to be degraded")
	}
}

func TestCrossRegionLink(t *testing.T) {
	requireSQLite3(t)
	m, err := NewManager("us-east", tempDB(t))
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	m.UpdateCrossRegionLink(CrossRegionLink{
		RegionA:     "us-east",
		RegionB:     "eu-west",
		Latency:     85.5,
		Healthy:     true,
		LastChecked: time.Now().UTC(),
	})

	topo := m.ComputeGlobalTopology()
	if len(topo.CrossRegionLinks) != 1 {
		t.Errorf("expected 1 cross-region link, got %d", len(topo.CrossRegionLinks))
	}

	// Update same link
	m.UpdateCrossRegionLink(CrossRegionLink{
		RegionA: "us-east",
		RegionB: "eu-west",
		Latency: 120.0,
		Healthy: false,
	})

	topo = m.ComputeGlobalTopology()
	if len(topo.CrossRegionLinks) != 1 {
		t.Errorf("expected still 1 link after update, got %d", len(topo.CrossRegionLinks))
	}
}

func TestPreferredRegion(t *testing.T) {
	requireSQLite3(t)
	m, err := NewManager("us-east", tempDB(t))
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	m.AddRegion(Region{ID: "eu-west", Name: "EU West"})

	state := kernel.NewState()
	// us-east: all failing
	state.NodeRegistry[1] = kernel.NodeInfo{NodeNum: 1, NodeID: "!n1", Region: "us-east"}
	state.NodeScores["!n1"] = kernel.NodeScore{Classification: "failing"}
	// eu-west: all healthy
	state.NodeRegistry[2] = kernel.NodeInfo{NodeNum: 2, NodeID: "!n2", Region: "eu-west"}
	state.NodeScores["!n2"] = kernel.NodeScore{Classification: "healthy"}

	m.UpdateHealthFromKernel(state)

	preferred := m.PreferredRegion()
	if preferred != "eu-west" {
		t.Errorf("expected preferred region eu-west, got %s", preferred)
	}
}
