package readiness

import (
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/status"
	"github.com/mel-project/mel/internal/transport"
)

func TestEvaluateNotReadyWithoutIngest(t *testing.T) {
	cfg := config.Default()
	cfg.Transports = []config.TransportConfig{{Name: "m", Type: "mqtt", Enabled: true}}
	snap := status.Snapshot{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Transports: []status.TransportReport{
			{Name: "m", Enabled: true, EffectiveState: transport.StateHistoricalOnly},
		},
		Mesh: status.MeshDrilldown{MeshHealth: status.MeshHealth{State: "degraded"}},
	}
	r := Evaluate(cfg, snap, true, time.Now())
	if r.Ready {
		t.Fatal("expected not ready")
	}
	if r.Status != "not_ready" {
		t.Fatalf("status: %q", r.Status)
	}
}

func TestEvaluateIdleNoTransports(t *testing.T) {
	cfg := config.Default()
	snap := status.Snapshot{GeneratedAt: time.Now().UTC().Format(time.RFC3339)}
	r := Evaluate(cfg, snap, true, time.Now())
	if !r.Ready {
		t.Fatal("expected ready when idle")
	}
}
