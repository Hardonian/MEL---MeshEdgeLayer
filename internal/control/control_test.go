package control

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/transport"
)

func TestEvaluateGuardedAutoAllowsRestartWithPersistentEvidence(t *testing.T) {
	cfg := config.Default()
	cfg.Control.Mode = ModeGuardedAuto
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	cfg.Transports = []config.TransportConfig{{Name: "mqtt", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test", ClientID: "mel-test"}}
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	if err := database.UpsertTransportAlert(db.TransportAlertRecord{
		ID:               "mqtt|retry_threshold_exceeded|retry-threshold",
		TransportName:    "mqtt",
		TransportType:    "mqtt",
		Severity:         "critical",
		Reason:           transport.ReasonRetryThresholdExceeded,
		Summary:          "retry threshold exceeded",
		FirstTriggeredAt: now.Add(-4 * time.Minute).Format(time.RFC3339),
		LastUpdatedAt:    now.Add(-30 * time.Second).Format(time.RFC3339),
		Active:           true,
		EpisodeID:        "ep-1",
		ClusterKey:       "retry-threshold",
		TriggerCondition: "retry_threshold_exceeded_count=2",
	}); err != nil {
		t.Fatal(err)
	}
	for _, ts := range []time.Time{now.Add(-2 * time.Minute), now.Add(-1 * time.Minute)} {
		if err := database.UpsertTransportAnomalySnapshot(db.TransportAnomalySnapshot{
			BucketStart:   ts.Format(time.RFC3339),
			TransportName: "mqtt",
			TransportType: "mqtt",
			Reason:        transport.ReasonRetryThresholdExceeded,
			Count:         1,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := database.UpsertNode(map[string]any{
		"node_num":        int64(42),
		"node_id":         "!0042",
		"long_name":       "Node 42",
		"short_name":      "N42",
		"last_seen":       now.Format(time.RFC3339),
		"last_gateway_id": "gw",
		"last_snr":        1.5,
		"last_rssi":       -70,
		"lat_redacted":    0.0,
		"lon_redacted":    0.0,
		"altitude":        0,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := database.InsertMessage(map[string]any{
		"transport_name": "mqtt",
		"packet_id":      int64(7),
		"dedupe_hash":    "ctrl-msg-1",
		"channel_id":     "",
		"gateway_id":     "gw",
		"from_node":      int64(42),
		"to_node":        int64(0),
		"portnum":        int64(1),
		"payload_text":   "hi",
		"payload_json":   map[string]any{"message_type": "text"},
		"raw_hex":        "01",
		"rx_time":        now.Add(-20 * time.Second).Format(time.RFC3339),
		"hop_limit":      int64(0),
		"relay_node":     int64(0),
	}); err != nil {
		t.Fatal(err)
	}

	eval, err := Evaluate(cfg, database, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateFailed, FailureCount: 3, EpisodeID: "ep-1"}}, now)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, decision := range eval.Decisions {
		if decision.CandidateAction.ActionType == ActionRestartTransport {
			found = true
			if !decision.Allowed {
				t.Fatalf("expected restart decision to be allowed, got denial=%q checks=%v", decision.DenialReason, decision.SafetyChecks)
			}
		}
	}
	if !found {
		t.Fatalf("expected restart decision, got %+v", eval.Decisions)
	}
}

func TestEvaluateAdvisoryDeniesAutomationOnManualOnlyTransport(t *testing.T) {
	cfg := config.Default()
	cfg.Control.Mode = ModeAdvisory
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	cfg.Transports = []config.TransportConfig{{Name: "mqtt", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test", ClientID: "mel-test", ManualOnly: true}}
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	if err := database.UpsertTransportAlert(db.TransportAlertRecord{
		ID:               "mqtt|retry_threshold_exceeded|retry-threshold",
		TransportName:    "mqtt",
		TransportType:    "mqtt",
		Severity:         "critical",
		Reason:           transport.ReasonRetryThresholdExceeded,
		Summary:          "retry threshold exceeded",
		FirstTriggeredAt: now.Add(-4 * time.Minute).Format(time.RFC3339),
		LastUpdatedAt:    now.Add(-30 * time.Second).Format(time.RFC3339),
		Active:           true,
		ClusterKey:       "retry-threshold",
		TriggerCondition: "retry_threshold_exceeded_count=2",
	}); err != nil {
		t.Fatal(err)
	}
	for _, ts := range []time.Time{now.Add(-2 * time.Minute), now.Add(-1 * time.Minute)} {
		if err := database.UpsertTransportAnomalySnapshot(db.TransportAnomalySnapshot{
			BucketStart:   ts.Format(time.RFC3339),
			TransportName: "mqtt",
			TransportType: "mqtt",
			Reason:        transport.ReasonRetryThresholdExceeded,
			Count:         1,
		}); err != nil {
			t.Fatal(err)
		}
	}
	eval, err := Evaluate(cfg, database, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateFailed, FailureCount: 3}}, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, decision := range eval.Decisions {
		if decision.CandidateAction.ActionType == ActionRestartTransport {
			if decision.Allowed {
				t.Fatal("expected advisory/manual_only transport to deny automation")
			}
			if !decision.OperatorOverride {
				t.Fatalf("expected operator override to be recorded, got %+v", decision)
			}
			return
		}
	}
	t.Fatalf("expected denied restart decision, got %+v", eval.Decisions)
}
