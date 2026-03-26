package meshintel

import (
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/topology"
)

func now() time.Time {
	return time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
}

func TestComputeIsolatedSingleNode(t *testing.T) {
	cfg := config.Default()
	nodes := []topology.Node{{NodeNum: 1, LastSeenAt: now().Format(time.RFC3339), TrustClass: topology.TrustTrusted}}
	links := []topology.Link{}
	ar := topology.Analyze(nodes, links, topology.DefaultStaleThresholds(), now())
	sig := MessageSignals{TotalMessages: 0, WindowDescription: "test", TransportConnected: false}
	a := Compute(cfg, ar, sig, false, now())
	if a.Bootstrap.Viability != ViabilityIsolated {
		t.Fatalf("viability want isolated got %s", a.Bootstrap.Viability)
	}
	if a.Bootstrap.LoneWolfScore < 0.9 {
		t.Fatalf("lone wolf want high got %v", a.Bootstrap.LoneWolfScore)
	}
	if len(a.Recommendations) == 0 {
		t.Fatal("expected recommendations")
	}
}

func TestComputeViableCluster(t *testing.T) {
	cfg := config.Default()
	ts := now().Format(time.RFC3339)
	nodes := []topology.Node{
		{NodeNum: 1, LastSeenAt: ts, TrustClass: topology.TrustTrusted},
		{NodeNum: 2, LastSeenAt: ts, TrustClass: topology.TrustTrusted},
		{NodeNum: 3, LastSeenAt: ts, TrustClass: topology.TrustTrusted},
		{NodeNum: 4, LastSeenAt: ts, TrustClass: topology.TrustTrusted},
	}
	links := []topology.Link{
		{EdgeID: "a", SrcNodeNum: 1, DstNodeNum: 2, Observed: true, LastObservedAt: ts, QualityScore: 0.8, Reliability: 0.8, SourceTrustLevel: 0.9, ObservationCount: 5},
		{EdgeID: "b", SrcNodeNum: 2, DstNodeNum: 3, Observed: true, LastObservedAt: ts, QualityScore: 0.8, Reliability: 0.8, SourceTrustLevel: 0.9, ObservationCount: 5},
		{EdgeID: "c", SrcNodeNum: 3, DstNodeNum: 4, Observed: true, LastObservedAt: ts, QualityScore: 0.8, Reliability: 0.8, SourceTrustLevel: 0.9, ObservationCount: 5},
		{EdgeID: "d", SrcNodeNum: 4, DstNodeNum: 1, Observed: true, LastObservedAt: ts, QualityScore: 0.8, Reliability: 0.8, SourceTrustLevel: 0.9, ObservationCount: 5},
	}
	ar := topology.Analyze(nodes, links, topology.DefaultStaleThresholds(), now())
	sig := MessageSignals{TotalMessages: 80, WindowDescription: "test", TransportConnected: true, MessagesWithHop: 40, AvgHopLimit: 2.5}
	a := Compute(cfg, ar, sig, true, now())
	if a.Bootstrap.Viability != ViabilityViableLocalMesh {
		t.Fatalf("viability want viable_local_mesh got %s", a.Bootstrap.Viability)
	}
	if a.Topology.FragmentationScore > 0.1 {
		t.Fatalf("fragmentation should be low, got %v", a.Topology.FragmentationScore)
	}
}

func TestComputeInfrastructureDependent(t *testing.T) {
	cfg := config.Default()
	ts := now().Format(time.RFC3339)
	nodes := []topology.Node{
		{NodeNum: 1, LastSeenAt: ts, LastBrokerSeenAt: ts, TrustClass: topology.TrustTrusted},
		{NodeNum: 2, LastSeenAt: ts, LastBrokerSeenAt: ts, TrustClass: topology.TrustTrusted},
	}
	links := []topology.Link{
		{EdgeID: "x", SrcNodeNum: 1, DstNodeNum: 2, Observed: true, RelayDependent: true, LastObservedAt: ts, QualityScore: 0.6, Reliability: 0.6, SourceTrustLevel: 0.85, ObservationCount: 3},
	}
	ar := topology.Analyze(nodes, links, topology.DefaultStaleThresholds(), now())
	sig := MessageSignals{TotalMessages: 10, TransportConnected: true}
	a := Compute(cfg, ar, sig, true, now())
	if a.Bootstrap.Viability != ViabilityInfrastructureDependent {
		t.Fatalf("viability want infrastructure_dependent got %s", a.Bootstrap.Viability)
	}
}

func TestProtocolFitInsufficientWithOneNode(t *testing.T) {
	cfg := config.Default()
	ts := now().Format(time.RFC3339)
	nodes := []topology.Node{{NodeNum: 1, LastSeenAt: ts}}
	links := []topology.Link{}
	ar := topology.Analyze(nodes, links, topology.DefaultStaleThresholds(), now())
	sig := MessageSignals{TotalMessages: 100, TransportConnected: true}
	a := Compute(cfg, ar, sig, true, now())
	if a.ProtocolFit.FitClass != ProtocolFitInsufficientEvidence {
		t.Fatalf("protocol fit want insufficient_evidence got %s", a.ProtocolFit.FitClass)
	}
}

func TestRankRecommendationsTransportDown(t *testing.T) {
	cfg := config.Default()
	nodes := []topology.Node{
		{NodeNum: 1, LastSeenAt: now().Format(time.RFC3339)},
		{NodeNum: 2, LastSeenAt: now().Format(time.RFC3339)},
	}
	links := []topology.Link{{EdgeID: "e", SrcNodeNum: 1, DstNodeNum: 2, Observed: true, LastObservedAt: now().Format(time.RFC3339), QualityScore: 0.7, Reliability: 0.7, SourceTrustLevel: 0.8, ObservationCount: 2}}
	ar := topology.Analyze(nodes, links, topology.DefaultStaleThresholds(), now())
	sig := MessageSignals{}
	a := Compute(cfg, ar, sig, false, now())
	if a.Recommendations[0].Class != RecInvestigateTransport {
		t.Fatalf("first rec should be transport investigate, got %s", a.Recommendations[0].Class)
	}
}
