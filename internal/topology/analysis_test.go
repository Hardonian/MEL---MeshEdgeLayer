package topology

import (
	"testing"
	"time"
)

func TestAnalyzeEmptyTopology(t *testing.T) {
	result := Analyze(nil, nil, DefaultStaleThresholds(), time.Now().UTC())
	if result.Snapshot.NodeCount != 0 {
		t.Errorf("expected 0 nodes, got %d", result.Snapshot.NodeCount)
	}
	if result.Snapshot.EdgeCount != 0 {
		t.Errorf("expected 0 edges, got %d", result.Snapshot.EdgeCount)
	}
}

func TestAnalyzeSimpleTopology(t *testing.T) {
	now := time.Now().UTC()
	nodes := []Node{
		{NodeNum: 1, ShortName: "A", LastSeenAt: now.Format(time.RFC3339), TrustClass: TrustTrusted, LastDirectSeenAt: now.Format(time.RFC3339)},
		{NodeNum: 2, ShortName: "B", LastSeenAt: now.Format(time.RFC3339), TrustClass: TrustTrusted, LastDirectSeenAt: now.Format(time.RFC3339)},
		{NodeNum: 3, ShortName: "C", LastSeenAt: now.Format(time.RFC3339), TrustClass: TrustTrusted, LastDirectSeenAt: now.Format(time.RFC3339)},
	}
	links := []Link{
		{EdgeID: "e1", SrcNodeNum: 1, DstNodeNum: 2, Observed: true, LastObservedAt: now.Format(time.RFC3339), Reliability: 0.9, SourceTrustLevel: 0.9, ObservationCount: 10},
		{EdgeID: "e2", SrcNodeNum: 2, DstNodeNum: 3, Observed: true, LastObservedAt: now.Format(time.RFC3339), Reliability: 0.8, SourceTrustLevel: 0.8, ObservationCount: 8},
	}

	result := Analyze(nodes, links, DefaultStaleThresholds(), now)

	if result.Snapshot.NodeCount != 3 {
		t.Errorf("expected 3 nodes, got %d", result.Snapshot.NodeCount)
	}
	if result.Snapshot.EdgeCount != 2 {
		t.Errorf("expected 2 edges, got %d", result.Snapshot.EdgeCount)
	}
	if result.Snapshot.DirectEdgeCount != 2 {
		t.Errorf("expected 2 direct edges, got %d", result.Snapshot.DirectEdgeCount)
	}
	if len(result.IsolatedNodes) != 0 {
		t.Errorf("expected no isolated nodes, got %d", len(result.IsolatedNodes))
	}
	if result.Snapshot.GraphHash == "" {
		t.Error("graph hash should not be empty")
	}
	if result.AnalyzedAt == "" {
		t.Error("analyzed_at should be set")
	}
}

func TestAnalyzeIsolatedNode(t *testing.T) {
	now := time.Now().UTC()
	nodes := []Node{
		{NodeNum: 1, ShortName: "Connected", LastSeenAt: now.Format(time.RFC3339), TrustClass: TrustTrusted},
		{NodeNum: 2, ShortName: "Connected2", LastSeenAt: now.Format(time.RFC3339), TrustClass: TrustTrusted},
		{NodeNum: 99, ShortName: "Isolated", LastSeenAt: now.Format(time.RFC3339), TrustClass: TrustTrusted},
	}
	links := []Link{
		{EdgeID: "e1", SrcNodeNum: 1, DstNodeNum: 2, Observed: true, LastObservedAt: now.Format(time.RFC3339)},
	}

	result := Analyze(nodes, links, DefaultStaleThresholds(), now)

	if len(result.IsolatedNodes) != 1 || result.IsolatedNodes[0] != 99 {
		t.Errorf("expected isolated node 99, got %v", result.IsolatedNodes)
	}
	// Should generate recommendation to add relay near isolated node
	hasRelayRec := false
	for _, r := range result.Recommendations {
		if r.Type == "add_relay" {
			for _, n := range r.AffectedNodes {
				if n == 99 {
					hasRelayRec = true
				}
			}
		}
	}
	if !hasRelayRec {
		t.Error("expected recommendation to add relay near isolated node 99")
	}
}

func TestAnalyzeBridgeNode(t *testing.T) {
	now := time.Now().UTC()
	// Node 2 is bridge-critical: removing it disconnects 1 from 3
	nodes := []Node{
		{NodeNum: 1, ShortName: "A", LastSeenAt: now.Format(time.RFC3339), TrustClass: TrustTrusted},
		{NodeNum: 2, ShortName: "Bridge", LastSeenAt: now.Format(time.RFC3339), TrustClass: TrustTrusted},
		{NodeNum: 3, ShortName: "C", LastSeenAt: now.Format(time.RFC3339), TrustClass: TrustTrusted},
	}
	links := []Link{
		{EdgeID: "e1", SrcNodeNum: 1, DstNodeNum: 2, Observed: true, LastObservedAt: now.Format(time.RFC3339)},
		{EdgeID: "e2", SrcNodeNum: 2, DstNodeNum: 3, Observed: true, LastObservedAt: now.Format(time.RFC3339)},
	}

	result := Analyze(nodes, links, DefaultStaleThresholds(), now)

	if len(result.BridgeNodes) != 1 || result.BridgeNodes[0] != 2 {
		t.Errorf("expected bridge node 2, got %v", result.BridgeNodes)
	}
	// Should have bottleneck for bridge
	hasBridgeBottleneck := false
	for _, b := range result.Bottlenecks {
		if b.Type == "single_point_of_failure" {
			hasBridgeBottleneck = true
		}
	}
	if !hasBridgeBottleneck {
		t.Error("expected single_point_of_failure bottleneck for bridge node")
	}
}

func TestAnalyzeStaleRegion(t *testing.T) {
	now := time.Now().UTC()
	nodes := []Node{
		{NodeNum: 1, ShortName: "Fresh", LastSeenAt: now.Format(time.RFC3339), TrustClass: TrustTrusted},
		{NodeNum: 2, ShortName: "Stale1", LastSeenAt: now.Add(-2 * time.Hour).Format(time.RFC3339), TrustClass: TrustTrusted, Stale: true},
		{NodeNum: 3, ShortName: "Stale2", LastSeenAt: now.Add(-2 * time.Hour).Format(time.RFC3339), TrustClass: TrustTrusted, Stale: true},
	}
	links := []Link{
		{EdgeID: "e1", SrcNodeNum: 1, DstNodeNum: 2, Observed: true, LastObservedAt: now.Format(time.RFC3339)},
		{EdgeID: "e2", SrcNodeNum: 2, DstNodeNum: 3, Observed: true, LastObservedAt: now.Add(-2 * time.Hour).Format(time.RFC3339)},
	}

	result := Analyze(nodes, links, DefaultStaleThresholds(), now)

	if len(result.StaleRegions) != 1 {
		t.Errorf("expected 1 stale region, got %d", len(result.StaleRegions))
	}
}

func TestAnalyzeRecommendationEvidence(t *testing.T) {
	now := time.Now().UTC()
	nodes := []Node{
		{NodeNum: 1, ShortName: "Q", Quarantined: true, QuarantineReason: "contradictory source", LastSeenAt: now.Format(time.RFC3339), TrustClass: TrustUntrusted},
	}

	result := Analyze(nodes, nil, DefaultStaleThresholds(), now)

	for _, r := range result.Recommendations {
		if r.Summary == "" {
			t.Error("recommendation missing summary")
		}
		if r.Confidence <= 0 {
			t.Error("recommendation missing confidence")
		}
		if r.Basis == "" {
			t.Error("recommendation missing basis")
		}
	}
}

func TestBuildSnapshotHash(t *testing.T) {
	now := time.Now().UTC()
	nodes1 := []Node{{NodeNum: 1}, {NodeNum: 2}}
	nodes2 := []Node{{NodeNum: 1}, {NodeNum: 2}, {NodeNum: 3}}
	links := []Link{{SrcNodeNum: 1, DstNodeNum: 2, Observed: true}}

	snap1 := buildSnapshot(nodes1, links, nil, now)
	snap2 := buildSnapshot(nodes2, links, nil, now)

	if snap1.GraphHash == snap2.GraphHash {
		t.Error("different topologies should have different graph hashes")
	}
}
