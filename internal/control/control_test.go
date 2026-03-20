package control

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	statuspkg "github.com/mel-project/mel/internal/status"
	"github.com/mel-project/mel/internal/transport"
)

func TestEvaluateGuardedAutoAllowsRestartWithPersistentEvidence(t *testing.T) {
	cfg := config.Default()
	cfg.Control.Mode = ModeGuardedAuto
cfg.Control.AllowTransportRestart = true
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

func TestDefaultActionRealityMatrixKeepsOnlySafeActuatorsExecutable(t *testing.T) {
	matrix := ActionRealityByType()
	for _, actionType := range []string{ActionRestartTransport, ActionResubscribeTransport, ActionBackoffIncrease, ActionBackoffReset, ActionTriggerHealthRecheck} {
		item := matrix[actionType]
		if !item.ActuatorExists || !item.Reversible || !item.BlastRadiusKnown || !item.SafeForGuardedAuto || item.AdvisoryOnly {
			t.Fatalf("expected %s to remain executable, got %+v", actionType, item)
		}
	}
	for _, actionType := range []string{ActionTemporarilyDeprioritize, ActionTemporarilySuppressNoisySource, ActionClearSuppression} {
		item := matrix[actionType]
		if !item.AdvisoryOnly || item.SafeForGuardedAuto {
			t.Fatalf("expected %s to remain advisory-only, got %+v", actionType, item)
		}
	}
}

func TestEvaluateGuardedAutoDeniesRoutingWithoutAlternatePath(t *testing.T) {
	cfg := config.Default()
	cfg.Control.Mode = ModeGuardedAuto
cfg.Control.AllowTransportRestart = true
	cfg.Control.AllowMeshLevelActions = true
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	cfg.Transports = []config.TransportConfig{
		{Name: "mqtt-a", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test/a", ClientID: "a"},
	}
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	for _, ts := range []time.Time{now.Add(-3 * time.Minute), now.Add(-1 * time.Minute)} {
		if err := database.InsertTransportHealthSnapshot(db.TransportHealthSnapshot{
			SnapshotTime:  ts.Format(time.RFC3339),
			TransportName: "mqtt-a",
			TransportType: "mqtt",
			State:         "failed",
			Score:         10,
		}); err != nil {
			t.Fatal(err)
		}
	}
	eval, err := Evaluate(cfg, database, []transport.Health{{Name: "mqtt-a", Type: "mqtt", State: transport.StateFailed, FailureCount: 2}}, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, decision := range eval.Decisions {
		if decision.CandidateAction.ActionType == ActionTemporarilyDeprioritize {
			if decision.DenialCode != DenialNoAlternatePath {
				t.Fatalf("expected no_alternate_path denial, got %+v", decision)
			}
			return
		}
	}
	t.Fatalf("expected deprioritize decision, got %+v", eval.Decisions)
}

func TestEvaluateGuardedAutoDeniesSuppressionWithWeakAttribution(t *testing.T) {
	cfg := config.Default()
	cfg.Control.Mode = ModeGuardedAuto
cfg.Control.AllowTransportRestart = true
	cfg.Control.AllowMeshLevelActions = true
	cfg.Control.AllowSourceSuppression = true
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	cfg.Transports = []config.TransportConfig{{Name: "mqtt", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test", ClientID: "mel-test"}}
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	for _, ts := range []time.Time{now.Add(-2 * time.Minute), now.Add(-1 * time.Minute)} {
		if err := database.InsertTransportHealthSnapshot(db.TransportHealthSnapshot{
			SnapshotTime:  ts.Format(time.RFC3339),
			TransportName: "mqtt",
			TransportType: "mqtt",
			State:         "unstable",
			Score:         40,
		}); err != nil {
			t.Fatal(err)
		}
	}
	for _, ts := range []time.Time{now.Add(-2 * time.Minute), now.Add(-1 * time.Minute)} {
		if err := database.UpsertTransportAnomalySnapshot(db.TransportAnomalySnapshot{
			BucketStart:   ts.Format(time.RFC3339),
			TransportName: "mqtt",
			TransportType: "mqtt",
			Reason:        transport.ReasonMalformedFrame,
			Count:         3,
		}); err != nil {
			t.Fatal(err)
		}
	}
	for _, nodeNum := range []int64{41, 42} {
		if err := database.UpsertNode(map[string]any{
			"node_num":        nodeNum,
			"node_id":         fmt.Sprintf("!%04d", nodeNum),
			"long_name":       fmt.Sprintf("Node %d", nodeNum),
			"short_name":      fmt.Sprintf("N%d", nodeNum),
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
			"packet_id":      nodeNum,
			"dedupe_hash":    fmt.Sprintf("ctrl-msg-%d", nodeNum),
			"channel_id":     "",
			"gateway_id":     "gw",
			"from_node":      nodeNum,
			"to_node":        int64(0),
			"portnum":        int64(1),
			"payload_text":   "noise",
			"payload_json":   map[string]any{"message_type": "text"},
			"raw_hex":        "01",
			"rx_time":        now.Add(-20 * time.Second).Format(time.RFC3339),
			"hop_limit":      int64(0),
			"relay_node":     int64(0),
		}); err != nil {
			t.Fatal(err)
		}
	}
	eval, err := Evaluate(cfg, database, []transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateRetrying, FailureCount: 1}}, now)
	if err != nil {
		t.Fatal(err)
	}
	_ = eval
	decision := evaluateCandidate(cfg, database, PolicyFromConfig(cfg), ControlAction{
		ID:              "candidate-suppress",
		ActionType:      ActionTemporarilySuppressNoisySource,
		TargetTransport: "mqtt",
		Reason:          "malformed flood",
		Confidence:      0.92,
		TriggerEvidence: []string{"malformed frames persist"},
		CreatedAt:       now.Format(time.RFC3339),
		Reversible:      true,
		ExpiresAt:       now.Add(10 * time.Minute).Format(time.RFC3339),
		Mode:            ModeGuardedAuto,
		PolicyRule:      "test",
	}, statuspkg.MeshDrilldown{}, buildRuntimeSignals([]transport.Health{{Name: "mqtt", Type: "mqtt", State: transport.StateRetrying, FailureCount: 1}}), nil, map[string][]db.ControlActionRecord{}, ActionRealityByType(), now)
	if decision.DenialCode != DenialAttributionWeak {
		t.Fatalf("expected attribution_weak denial, got %+v", decision)
	}
}
