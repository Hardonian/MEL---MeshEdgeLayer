package topology

import (
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
)

func TestBuildIntelligenceViewMapModes(t *testing.T) {
	now := time.Now().UTC()
	nodes := []Node{
		{NodeNum: 1, LastSeenAt: now.Format(time.RFC3339), TrustClass: TrustTrusted, LocationState: LocExact, LatRedacted: 1.2, LonRedacted: 3.4},
	}
	links := []Link{}
	ar := Analyze(nodes, links, DefaultStaleThresholds(), now)

	cfg := config.Default()
	cfg.Privacy.MapReportingAllowed = false
	v := BuildIntelligenceView(cfg, ar, false, now)
	if v["view_mode"] != "graph" {
		t.Fatalf("expected graph when map reporting disallowed, got %v", v["view_mode"])
	}
	if ga, _ := v["google_maps_basemap_available"].(bool); ga {
		t.Fatalf("expected google_maps_basemap_available false by default, got %v", v["google_maps_basemap_available"])
	}

	cfg.Privacy.MapReportingAllowed = true
	v2 := BuildIntelligenceView(cfg, ar, true, now)
	if v2["view_mode"] != "map" {
		t.Fatalf("expected map when all nodes map-eligible, got %v", v2["view_mode"])
	}

	nodes2 := []Node{
		{NodeNum: 1, LastSeenAt: now.Format(time.RFC3339), TrustClass: TrustTrusted, LocationState: LocExact, LatRedacted: 1.0, LonRedacted: 2.0},
		{NodeNum: 2, LastSeenAt: now.Format(time.RFC3339), TrustClass: TrustTrusted, LocationState: LocUnknown},
	}
	ar2 := Analyze(nodes2, links, DefaultStaleThresholds(), now)
	v3 := BuildIntelligenceView(cfg, ar2, true, now)
	if v3["view_mode"] != "map_partial" {
		t.Fatalf("expected map_partial, got %v", v3["view_mode"])
	}
}

func TestSnapshotIncludesLinkEvidenceLine(t *testing.T) {
	now := time.Now().UTC()
	s := buildSnapshot([]Node{{NodeNum: 1, LastSeenAt: now.Format(time.RFC3339)}}, nil, nil, now)
	found := false
	for _, line := range s.Explanation {
		if line == "link_evidence=packet_relay_or_unicast_destination_not_rf_proof" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected link evidence line in explanation: %#v", s.Explanation)
	}
}
