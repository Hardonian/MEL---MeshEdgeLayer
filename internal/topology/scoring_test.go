package topology

import (
	"testing"
	"time"
)

func TestScoreNodeHealthy(t *testing.T) {
	now := time.Now().UTC()
	n := Node{
		NodeNum:          1234,
		NodeID:           "!abc1",
		ShortName:        "Test1",
		LastSeenAt:       now.Add(-2 * time.Minute).Format(time.RFC3339),
		LastDirectSeenAt: now.Add(-2 * time.Minute).Format(time.RFC3339),
		TrustClass:       TrustDirectLocal,
		LastSNR:          12.0,
	}
	links := []Link{
		{EdgeID: "e1", SrcNodeNum: 1234, DstNodeNum: 5678, Observed: true, QualityScore: 0.9, Reliability: 0.9},
		{EdgeID: "e2", SrcNodeNum: 1234, DstNodeNum: 9012, Observed: true, QualityScore: 0.8, Reliability: 0.8},
		{EdgeID: "e3", SrcNodeNum: 1234, DstNodeNum: 3456, Observed: true, QualityScore: 0.7, Reliability: 0.7},
	}
	thresholds := DefaultStaleThresholds()
	score, state, factors := ScoreNode(n, links, thresholds, now)

	if score < 0.7 {
		t.Errorf("expected healthy score >= 0.7, got %f", score)
	}
	if state != HealthHealthy {
		t.Errorf("expected healthy state, got %s", state)
	}
	if len(factors) != 7 {
		t.Errorf("expected 7 factors, got %d", len(factors))
	}
	// Verify all factors are named and have basis
	for _, f := range factors {
		if f.Name == "" {
			t.Error("factor missing name")
		}
		if f.Basis == "" {
			t.Error("factor missing basis")
		}
	}
}

func TestScoreNodeStale(t *testing.T) {
	now := time.Now().UTC()
	n := Node{
		NodeNum:    1234,
		LastSeenAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
		TrustClass: TrustUnknown,
		Stale:      true,
	}
	thresholds := DefaultStaleThresholds()
	score, state, _ := ScoreNode(n, nil, thresholds, now)

	if score > 0.3 {
		t.Errorf("expected low score for stale node, got %f", score)
	}
	if state != HealthStale {
		t.Errorf("expected stale state, got %s", state)
	}
}

func TestScoreNodeIsolated(t *testing.T) {
	now := time.Now().UTC()
	n := Node{
		NodeNum:    1234,
		LastSeenAt: now.Add(-5 * time.Minute).Format(time.RFC3339),
		TrustClass: TrustTrusted,
	}
	_, state, _ := ScoreNode(n, nil, DefaultStaleThresholds(), now)
	if state != HealthIsolated {
		t.Errorf("expected isolated state for node with no links, got %s", state)
	}
}

func TestScoreNodeQuarantined(t *testing.T) {
	now := time.Now().UTC()
	n := Node{
		NodeNum:          1234,
		LastSeenAt:       now.Format(time.RFC3339),
		TrustClass:       TrustTrusted,
		Quarantined:      true,
		QuarantineReason: "contradictory reports",
	}
	links := []Link{{Observed: true, QualityScore: 0.9}}
	_, state, _ := ScoreNode(n, links, DefaultStaleThresholds(), now)
	if state != HealthQuarantined {
		t.Errorf("expected quarantined state, got %s", state)
	}
}

func TestScoreLinkBasic(t *testing.T) {
	now := time.Now().UTC()
	l := Link{
		LastObservedAt:   now.Add(-5 * time.Minute).Format(time.RFC3339),
		Observed:         true,
		SourceTrustLevel: 0.9,
		Reliability:      0.85,
		ObservationCount: 30,
	}
	score, factors := ScoreLink(l, DefaultStaleThresholds(), now)
	if score < 0.6 {
		t.Errorf("expected decent link score, got %f", score)
	}
	if len(factors) != 7 {
		t.Errorf("expected 7 factors, got %d", len(factors))
	}
}

func TestScoreLinkContradiction(t *testing.T) {
	now := time.Now().UTC()
	l := Link{
		LastObservedAt:     now.Format(time.RFC3339),
		Observed:           true,
		SourceTrustLevel:   0.5,
		Reliability:        0.3,
		Contradiction:      true,
		IntermittenceCount: 15,
		ObservationCount:   5,
	}
	score, _ := ScoreLink(l, DefaultStaleThresholds(), now)
	if score > 0.5 {
		t.Errorf("expected low score for contradicted link, got %f", score)
	}
}

func TestTrustClassToValue(t *testing.T) {
	tests := []struct {
		tc  TrustClass
		min float64
		max float64
	}{
		{TrustDirectLocal, 0.9, 1.0},
		{TrustTrusted, 0.8, 0.9},
		{TrustPartial, 0.4, 0.6},
		{TrustUntrusted, 0.1, 0.2},
		{TrustUnknown, 0.2, 0.4},
	}
	for _, tt := range tests {
		v := trustClassToValue(tt.tc)
		if v < tt.min || v > tt.max {
			t.Errorf("trustClassToValue(%s) = %f, expected [%f, %f]", tt.tc, v, tt.min, tt.max)
		}
	}
}

func TestSNRToQuality(t *testing.T) {
	if snrToQuality(15) != 1.0 {
		t.Error("high SNR should be 1.0")
	}
	if snrToQuality(-15) != 0.0 {
		t.Error("very low SNR should be 0.0")
	}
	mid := snrToQuality(0)
	if mid < 0.4 || mid > 0.6 {
		t.Errorf("SNR 0 should be ~0.5, got %f", mid)
	}
}
