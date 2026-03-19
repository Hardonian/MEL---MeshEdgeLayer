package service

import (
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/transport"
)

func TestEvaluateTransportIntelligencePersistsAndDeduplicatesAlerts(t *testing.T) {
	tc := config.TransportConfig{Name: "mqtt-primary", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test"}
	app := newTestApp(t, tc)
	app.Transports = []transport.Transport{&stubTransport{name: tc.Name, typ: tc.Type, health: transport.Health{Name: tc.Name, Type: tc.Type, State: transport.StateRetrying, EpisodeID: "ep-1", FailureCount: 3, ObservationDrops: 4, LastHeartbeatAt: time.Now().UTC().Add(-4 * time.Minute).Format(time.RFC3339)}}}
	mustInsertTransportAuditAt(t, app.DB, time.Now().UTC().Add(-2*time.Minute), tc.Name, tc.Type, transport.ReasonRetryThresholdExceeded, map[string]any{"transport": tc.Name, "type": tc.Type, "episode_id": "ep-1"})
	mustInsertTransportAuditAt(t, app.DB, time.Now().UTC().Add(-90*time.Second), tc.Name, tc.Type, transport.ReasonObservationDropped, map[string]any{"transport": tc.Name, "type": tc.Type, "drop_count": 4, "drop_cause": "observation_queue_saturation"})

	now := time.Now().UTC()
	app.evaluateTransportIntelligence(now)
	app.evaluateTransportIntelligence(now.Add(10 * time.Second))

	alerts, err := app.DB.TransportAlerts(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) == 0 {
		t.Fatal("expected persisted active alerts")
	}
	if len(db.SortedAlertIDs(alerts)) != len(alerts) {
		t.Fatalf("expected sorted alert ids helper to preserve cardinality, got %+v", alerts)
	}
	ids := map[string]struct{}{}
	for _, alert := range alerts {
		if _, ok := ids[alert.ID]; ok {
			t.Fatalf("expected deduplicated alert ids, got %+v", alerts)
		}
		ids[alert.ID] = struct{}{}
	}
}

func TestEvaluateTransportIntelligenceResolvesRecoveredAlerts(t *testing.T) {
	tc := config.TransportConfig{Name: "direct-primary", Type: "tcp", Enabled: true, Endpoint: "127.0.0.1:4403"}
	app := newTestApp(t, tc)
	app.Transports = []transport.Transport{&stubTransport{name: tc.Name, typ: tc.Type, health: transport.Health{Name: tc.Name, Type: tc.Type, State: transport.StateRetrying, EpisodeID: "ep-1", FailureCount: 2, LastHeartbeatAt: time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339)}}}
	mustInsertTransportAuditAt(t, app.DB, time.Now().UTC().Add(-2*time.Minute), tc.Name, tc.Type, transport.ReasonTimeoutFailure, map[string]any{"transport": tc.Name, "type": tc.Type, "episode_id": "ep-1"})
	app.evaluateTransportIntelligence(time.Now().UTC())

	app.Transports = []transport.Transport{&stubTransport{name: tc.Name, typ: tc.Type, health: transport.Health{Name: tc.Name, Type: tc.Type, State: transport.StateLive, LastHeartbeatAt: time.Now().UTC().Add(-10 * time.Second).Format(time.RFC3339), LastIngestAt: time.Now().UTC().Add(-10 * time.Second).Format(time.RFC3339)}}}
	if err := app.DB.Exec("DELETE FROM audit_logs WHERE category='transport'; DELETE FROM dead_letters;"); err != nil {
		t.Fatal(err)
	}
	app.evaluateTransportIntelligence(time.Now().UTC().Add(20 * time.Second))

	alerts, err := app.DB.TransportAlerts(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 0 {
		t.Fatalf("expected recovered transport alerts to resolve, got %+v", alerts)
	}
}
