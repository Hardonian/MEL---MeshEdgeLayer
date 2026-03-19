package status

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/transport"
)

func TestEvaluateTransportIntelligenceScoresFailuresDeterministically(t *testing.T) {
	cfg, database := newHealthTestDB(t)
	cfg.Transports = []config.TransportConfig{{Name: "mqtt-primary", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test"}}
	now := time.Date(2026, 3, 19, 0, 5, 0, 0, time.UTC)
	mustInsertTransportAudit(t, database, now.Add(-2*time.Minute), transport.ReasonTimeoutFailure, map[string]any{"transport": "mqtt-primary", "type": "mqtt", "episode_id": "ep-1"})
	mustInsertTransportAudit(t, database, now.Add(-90*time.Second), transport.ReasonTimeoutStall, map[string]any{"transport": "mqtt-primary", "type": "mqtt", "episode_id": "ep-1"})
	mustInsertTransportAudit(t, database, now.Add(-30*time.Second), transport.ReasonObservationDropped, map[string]any{"transport": "mqtt-primary", "type": "mqtt", "drop_count": 3, "drop_cause": "ingest_queue_saturation"})
	mustInsertDeadLetterAt(t, database, now.Add(-45*time.Second), db.DeadLetter{TransportName: "mqtt-primary", TransportType: "mqtt", Topic: "msh/test", Reason: transport.ReasonRetryThresholdExceeded, PayloadHex: "aa", Details: map[string]any{"episode_id": "ep-1"}})
	if err := database.UpsertTransportRuntime(db.TransportRuntime{Name: "mqtt-primary", Type: "mqtt", Source: "127.0.0.1:1883", Enabled: true, State: transport.StateRetrying, Detail: "retrying", LastHeartbeatAt: now.Add(-4 * time.Minute).Format(time.RFC3339), EpisodeID: "ep-1", FailureCount: 3, ObservationDrops: 3, Reconnects: 2}); err != nil {
		t.Fatal(err)
	}

	intel, err := EvaluateTransportIntelligence(cfg, database, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	health := intel.HealthByTransport["mqtt-primary"]
	if health.Score != 0 {
		t.Fatalf("expected fully degraded score clamp to 0, got %+v", health)
	}
	if health.State != "failed" {
		t.Fatalf("expected failed health state, got %+v", health)
	}
	if health.Signals.ObservationDrops != 3 || !health.Signals.ActiveEpisode {
		t.Fatalf("expected observation-drop and active-episode signals, got %+v", health.Signals)
	}
	if len(intel.ClustersByTransport["mqtt-primary"]) == 0 {
		t.Fatalf("expected failure clusters, got %+v", intel)
	}
}

func TestEvaluateTransportIntelligenceRecoversGraduallyFromHistoricalEvidence(t *testing.T) {
	cfg, database := newHealthTestDB(t)
	cfg.Transports = []config.TransportConfig{{Name: "direct-primary", Type: "tcp", Enabled: true, Endpoint: "127.0.0.1:4403"}}
	now := time.Date(2026, 3, 19, 0, 15, 0, 0, time.UTC)
	mustInsertTransportAudit(t, database, now.Add(-10*time.Minute), transport.ReasonTimeoutFailure, map[string]any{"transport": "direct-primary", "type": "tcp", "episode_id": "ep-older"})
	mustInsertDeadLetterAt(t, database, now.Add(-10*time.Minute), db.DeadLetter{TransportName: "direct-primary", TransportType: "tcp", Topic: "", Reason: transport.ReasonRetryThresholdExceeded, PayloadHex: "aa", Details: map[string]any{"episode_id": "ep-older"}})
	if err := database.UpsertTransportRuntime(db.TransportRuntime{Name: "direct-primary", Type: "tcp", Source: "127.0.0.1:4403", Enabled: true, State: transport.StateLive, Detail: "live ingest confirmed by SQLite writes", LastHeartbeatAt: now.Add(-20 * time.Second).Format(time.RFC3339), LastMessageAt: now.Add(-20 * time.Second).Format(time.RFC3339)}); err != nil {
		t.Fatal(err)
	}

	intel, err := EvaluateTransportIntelligence(cfg, database, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	health := intel.HealthByTransport["direct-primary"]
	if health.State != "degraded" {
		t.Fatalf("expected gradual degraded recovery, got %+v", health)
	}
	if health.Score >= 100 || health.Score < 70 {
		t.Fatalf("expected residual historical penalty without failed state, got %+v", health)
	}
}

func newHealthTestDB(t *testing.T) (config.Config, *db.DB) {
	t.Helper()
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return cfg, database
}

func mustInsertTransportAudit(t *testing.T, database *db.DB, createdAt time.Time, message string, details map[string]any) {
	t.Helper()
	if err := database.InsertAuditLog("transport", "warning", message, details); err != nil {
		t.Fatal(err)
	}
	if err := database.Exec("UPDATE audit_logs SET created_at='" + createdAt.UTC().Format(time.RFC3339) + "' WHERE id=(SELECT MAX(id) FROM audit_logs);"); err != nil {
		t.Fatal(err)
	}
}

func mustInsertDeadLetterAt(t *testing.T, database *db.DB, createdAt time.Time, dl db.DeadLetter) {
	t.Helper()
	if err := database.InsertDeadLetter(dl); err != nil {
		t.Fatal(err)
	}
	if err := database.Exec("UPDATE dead_letters SET created_at='" + createdAt.UTC().Format(time.RFC3339) + "' WHERE id=(SELECT MAX(id) FROM dead_letters);"); err != nil {
		t.Fatal(err)
	}
}

func TestEvaluateTransportIntelligenceProducesHealthExplanation(t *testing.T) {
	cfg, database := newHealthTestDB(t)
	cfg.Transports = []config.TransportConfig{{Name: "mqtt-primary", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test"}}
	now := time.Date(2026, 3, 19, 0, 10, 0, 0, time.UTC)
	mustInsertTransportAudit(t, database, now.Add(-90*time.Second), transport.ReasonObservationDropped, map[string]any{"transport": "mqtt-primary", "type": "mqtt", "drop_count": 4, "drop_cause": "ingest_queue_saturation"})
	mustInsertTransportAudit(t, database, now.Add(-60*time.Second), transport.ReasonHandlerRejection, map[string]any{"transport": "mqtt-primary", "type": "mqtt"})
	if err := database.UpsertTransportRuntime(db.TransportRuntime{Name: "mqtt-primary", Type: "mqtt", Source: "127.0.0.1:1883", Enabled: true, State: transport.StateRetrying, EpisodeID: "ep-9", FailureCount: 2, ObservationDrops: 4, LastHeartbeatAt: now.Add(-3 * time.Minute).Format(time.RFC3339)}); err != nil {
		t.Fatal(err)
	}
	intel, err := EvaluateTransportIntelligence(cfg, database, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	explanation := intel.HealthByTransport["mqtt-primary"].Explanation
	if explanation.TransportName != "mqtt-primary" || explanation.ActiveEpisodeID != "ep-9" {
		t.Fatalf("unexpected explanation identity: %+v", explanation)
	}
	if len(explanation.TopPenalties) == 0 {
		t.Fatalf("expected penalty explanation, got %+v", explanation)
	}
	if explanation.TopPenalties[0].Penalty <= 0 {
		t.Fatalf("expected positive penalty, got %+v", explanation.TopPenalties)
	}
	if explanation.ObservationDrops != 4 {
		t.Fatalf("expected observation drops in explanation, got %+v", explanation)
	}
	if len(explanation.RecoveryBlockers) == 0 {
		t.Fatalf("expected recovery blockers, got %+v", explanation)
	}
}

func TestEvaluateTransportIntelligenceExplainsBurstCause(t *testing.T) {
	cfg, database := newHealthTestDB(t)
	cfg.Transports = []config.TransportConfig{{Name: "mqtt-primary", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test"}}
	now := time.Date(2026, 3, 19, 0, 12, 0, 0, time.UTC)
	mustInsertTransportAudit(t, database, now.Add(-30*time.Second), transport.ReasonObservationDropped, map[string]any{"transport": "mqtt-primary", "type": "mqtt", "drop_count": 6, "drop_cause": "event_bus_drops"})
	intel, err := EvaluateTransportIntelligence(cfg, database, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	explanation := intel.HealthByTransport["mqtt-primary"].Explanation
	found := false
	for _, blocker := range explanation.RecoveryBlockers {
		if blocker == "drop_cause:event_bus_drops x6" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected burst blocker, got %+v", explanation.RecoveryBlockers)
	}
}
