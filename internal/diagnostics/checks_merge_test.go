package diagnostics

import (
	"testing"
	"time"

	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/transport"
)

func TestMergeTransportRuntimeEvidence_prefersLiveStateAndMaxCounters(t *testing.T) {
	live := runtimeTransportFromHealth(transport.Health{
		Name:                "mqtt",
		Type:                "mqtt",
		State:               transport.StateLive,
		LastHeartbeatAt:     time.Now().UTC().Add(-30 * time.Second).Format(time.RFC3339),
		PacketsDropped:      2,
		ConsecutiveTimeouts: 1,
		FailureCount:        1,
		ObservationDrops:    3,
	})
	persisted := db.TransportRuntime{
		Name:             "mqtt",
		State:            transport.StateRetrying,
		LastHeartbeatAt:  time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339),
		PacketsDropped:   50,
		Timeouts:         5,
		FailureCount:     10,
		ObservationDrops: 20,
		Enabled:          true,
	}
	m := mergeTransportRuntimeEvidence(live, persisted)
	if m.State != transport.StateLive {
		t.Fatalf("expected live state, got %q", m.State)
	}
	if m.PacketsDropped != 50 {
		t.Fatalf("packets_dropped: want 50, got %d", m.PacketsDropped)
	}
	if m.Timeouts != 5 {
		t.Fatalf("timeouts: want 5, got %d", m.Timeouts)
	}
	if m.FailureCount != 10 {
		t.Fatalf("failure_count: want 10, got %d", m.FailureCount)
	}
	if m.ObservationDrops != 20 {
		t.Fatalf("observation_drops: want 20, got %d", m.ObservationDrops)
	}
	lhb, _ := time.Parse(time.RFC3339, m.LastHeartbeatAt)
	if lhb.Before(time.Now().UTC().Add(-2 * time.Minute)) {
		t.Fatalf("expected newer heartbeat from live, got %s", m.LastHeartbeatAt)
	}
}

func TestMergeResolvedTransportStates_dbOnlyAndLiveOnly(t *testing.T) {
	onlyDB := []db.TransportRuntime{{Name: "serial", State: transport.StateIdle, Enabled: true}}
	out := mergeResolvedTransportStates(nil, onlyDB)
	if len(out) != 1 || out[0].Name != "serial" {
		t.Fatalf("unexpected: %+v", out)
	}

	onlyLive := []transport.Health{{Name: "tcp", Type: "tcp", State: transport.StateLive}}
	out2 := mergeResolvedTransportStates(onlyLive, nil)
	if len(out2) != 1 || out2[0].Name != "tcp" || out2[0].State != transport.StateLive {
		t.Fatalf("unexpected: %+v", out2)
	}
}
