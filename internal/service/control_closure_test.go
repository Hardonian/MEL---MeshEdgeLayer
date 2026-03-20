package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/transport"
)

func TestRecoverIncompleteControlActionsClosesWithoutBlindReexecution(t *testing.T) {
	app := newTestApp(t, config.TransportConfig{Name: "mqtt-primary", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test"})
	if err := app.DB.UpsertControlAction(db.ControlActionRecord{
		ID:              "pending-restart",
		DecisionID:      "decision-1",
		ActionType:      control.ActionRestartTransport,
		TargetTransport: "mqtt-primary",
		Reason:          "retry threshold exceeded",
		Confidence:      0.95,
		CreatedAt:       "2026-03-19T12:00:00Z",
		LifecycleState:  control.LifecyclePending,
		Mode:            control.ModeGuardedAuto,
	}); err != nil {
		t.Fatal(err)
	}
	app.recoverIncompleteControlActions(time.Date(2026, 3, 19, 12, 5, 0, 0, time.UTC))
	row, ok, err := app.DB.ControlActionByID("pending-restart")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected recovered action to remain persisted")
	}
	if row.LifecycleState != control.LifecycleRecovered || row.ClosureState != control.ClosureSuperseded {
		t.Fatalf("expected recovered superseded action, got %+v", row)
	}
	followup, ok, err := app.DB.ControlActionByID("pending-restart-health-recheck")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || followup.ActionType != control.ActionTriggerHealthRecheck {
		t.Fatalf("expected health recheck followup, got %+v ok=%v", followup, ok)
	}
}

func TestQueueRecoveryActionsSchedulesBackoffResetOnlyForRealRollback(t *testing.T) {
	app := newTestApp(t, config.TransportConfig{Name: "mqtt-primary", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test"})
	now := time.Date(2026, 3, 19, 12, 20, 0, 0, time.UTC)
	if err := app.DB.UpsertControlAction(db.ControlActionRecord{
		ID:              "backoff-1",
		DecisionID:      "decision-1",
		ActionType:      control.ActionBackoffIncrease,
		TargetTransport: "mqtt-primary",
		Reason:          "storm",
		Confidence:      0.9,
		CreatedAt:       now.Add(-20 * time.Minute).Format(time.RFC3339),
		ExecutedAt:      now.Add(-20 * time.Minute).Format(time.RFC3339),
		CompletedAt:     now.Add(-20 * time.Minute).Format(time.RFC3339),
		Result:          control.ResultExecutedSuccessfully,
		Reversible:      true,
		ExpiresAt:       now.Add(-1 * time.Minute).Format(time.RFC3339),
		LifecycleState:  control.LifecycleCompleted,
		Mode:            control.ModeGuardedAuto,
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.DB.UpsertTransportAnomalySnapshot(db.TransportAnomalySnapshot{
		BucketStart:      now.Add(-30 * time.Second).Format(time.RFC3339),
		TransportName:    "mqtt-primary",
		TransportType:    "mqtt",
		Reason:           transport.ReasonMalformedFrame,
		Count:            0,
		DeadLetters:      0,
		ObservationDrops: 0,
	}); err != nil {
		t.Fatal(err)
	}
	app.queueRecoveryActions(now)
	reset, ok, err := app.DB.ControlActionByID("backoff-1-reset")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || reset.ActionType != control.ActionBackoffReset {
		t.Fatalf("expected backoff reset followup, got %+v ok=%v", reset, ok)
	}
	original, ok, err := app.DB.ControlActionByID("backoff-1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || original.ClosureState != control.ClosureExpiredAndReverted {
		t.Fatalf("expected original action to close as expired_and_reverted, got %+v", original)
	}
}

func TestGuardedControlChaosLifecycle(t *testing.T) {
	// Reset startup time to bypass grace period for testing
	control.ResetStartupTimeForTests(time.Date(2026, 3, 19, 11, 0, 0, 0, time.UTC))
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Transports = []config.TransportConfig{
		{Name: "mqtt-primary", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test", ClientID: "mel-chaos"},
		{Name: "mqtt-alt", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1884", Topic: "msh/test-alt", ClientID: "mel-chaos-alt"},
	}
	app := newTestApp(t, cfg.Transports[0])
	app.Cfg = cfg
	app.transportControls["mqtt-alt"] = newTransportControlState()
	ftPrimary := &failingTransport{name: "mqtt-primary", typ: "mqtt", health: transport.Health{Name: "mqtt-primary", Type: "mqtt"}}
	ftAlt := &failingTransport{name: "mqtt-alt", typ: "mqtt", health: transport.Health{Name: "mqtt-alt", Type: "mqtt", State: transport.StateLive}}
	app.Transports = []transport.Transport{ftPrimary, ftAlt}
	app.syncControlReality()

	phaseNow := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)

	// Phase 1: clean deploy in default advisory mode.
	explanation, err := app.controlExplanation()
	if err != nil {
		t.Fatal(err)
	}
	if explanation["mode"] != control.ModeAdvisory {
		t.Fatalf("expected advisory mode on clean deploy, got %+v", explanation)
	}

	// Phase 2: normal traffic does not trigger auto-actions.
	if err := app.DB.UpsertNode(map[string]any{"node_num": int64(42), "node_id": "!0042", "long_name": "Node 42", "short_name": "N42", "last_seen": phaseNow.Format(time.RFC3339), "last_gateway_id": "gw", "last_snr": 4.2, "last_rssi": -61, "lat_redacted": 0.0, "lon_redacted": 0.0, "altitude": 0}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.DB.InsertMessage(map[string]any{"transport_name": "mqtt-primary", "packet_id": int64(1), "dedupe_hash": "clean-1", "channel_id": "", "gateway_id": "gw", "from_node": int64(42), "to_node": int64(0), "portnum": int64(1), "payload_text": "hello", "payload_json": map[string]any{"message_type": "text"}, "raw_hex": "01", "rx_time": phaseNow.Format(time.RFC3339), "hop_limit": int64(0), "relay_node": int64(0)}); err != nil {
		t.Fatal(err)
	}
	eval, err := control.Evaluate(cfg, app.DB, []transport.Health{ftPrimary.Health(), ftAlt.Health()}, phaseNow)
	if err != nil {
		t.Fatal(err)
	}
	for _, decision := range eval.Decisions {
		if decision.Allowed {
			t.Fatalf("expected passive observation only during normal traffic, got %+v", decision)
		}
	}

	// Switch to guarded_auto for execution phases.
	app.Cfg.Control.Mode = control.ModeGuardedAuto
	app.Cfg.Control.AllowTransportRestart = true

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		app.controlExecutor(ctx)
	}()

	// Phase 3: transport failure triggers bounded restart.
	if err := app.DB.UpsertTransportAlert(db.TransportAlertRecord{ID: "mqtt-primary|retry_threshold_exceeded|chaos", TransportName: "mqtt-primary", TransportType: "mqtt", Severity: "critical", Reason: transport.ReasonRetryThresholdExceeded, Summary: "retry threshold exceeded", FirstTriggeredAt: phaseNow.Add(-4 * time.Minute).Format(time.RFC3339), LastUpdatedAt: phaseNow.Add(-30 * time.Second).Format(time.RFC3339), Active: true, EpisodeID: "ep-chaos", ClusterKey: "chaos", TriggerCondition: "retry_threshold_exceeded_count=2"}); err != nil {
		t.Fatal(err)
	}
	for _, ts := range []time.Time{phaseNow.Add(-2 * time.Minute), phaseNow.Add(-1 * time.Minute)} {
		if err := app.DB.UpsertTransportAnomalySnapshot(db.TransportAnomalySnapshot{BucketStart: ts.Format(time.RFC3339), TransportName: "mqtt-primary", TransportType: "mqtt", Reason: transport.ReasonRetryThresholdExceeded, Count: 1}); err != nil {
			t.Fatal(err)
		}
	}
	ftPrimary.ForceState(transport.StateFailed, "retry threshold exceeded", "")
	ftPrimary.SetFailureCount(3)
	app.evaluateControl(phaseNow)
	waitFor(t, 2*time.Second, func() bool {
		rows, err := app.DB.ControlActions("mqtt-primary", control.ActionRestartTransport, "", "", 10, 0)
		return err == nil && len(rows) > 0 && rows[0].Result == control.ResultExecutedSuccessfully
	})

	// Phase 4: failure storm prefers backoff increase.
	stormNow := phaseNow.Add(6 * time.Minute)
	ftAlt.ForceState(transport.StateRetrying, "failure storm", "")
	ftAlt.SetFailureCount(1)
	if err := app.DB.UpsertTransportAlert(db.TransportAlertRecord{
		ID:               "mqtt-alt|evidence_loss|storm",
		TransportName:    "mqtt-alt",
		TransportType:    "mqtt",
		Severity:         "high",
		Reason:           "evidence_loss",
		Summary:          "observation drops indicate saturation",
		FirstTriggeredAt: stormNow.Add(-2 * time.Minute).Format(time.RFC3339),
		LastUpdatedAt:    stormNow.Add(-30 * time.Second).Format(time.RFC3339),
		Active:           true,
		ClusterKey:       "storm",
		TriggerCondition: "observation_drops>=3",
	}); err != nil {
		t.Fatal(err)
	}
	for _, ts := range []time.Time{stormNow.Add(-90 * time.Second), stormNow.Add(-30 * time.Second)} {
		if err := app.DB.UpsertTransportAnomalySnapshot(db.TransportAnomalySnapshot{BucketStart: ts.Format(time.RFC3339), TransportName: "mqtt-alt", TransportType: "mqtt", Reason: transport.ReasonMalformedFrame, Count: 3}); err != nil {
			t.Fatal(err)
		}
	}
	app.evaluateControl(stormNow)
	waitFor(t, 2*time.Second, func() bool {
		rows, err := app.DB.ControlActions("mqtt-alt", control.ActionBackoffIncrease, "", "", 10, 0)
		return err == nil && len(rows) > 0 && rows[0].Result == control.ResultExecutedSuccessfully
	})

	// Phase 8: recovery window closes backoff safely.
	backoffRows, err := app.DB.ControlActions("mqtt-alt", control.ActionBackoffIncrease, "", "", 10, 0)
	if err != nil || len(backoffRows) == 0 {
		t.Fatalf("expected executed backoff action, rows=%+v err=%v", backoffRows, err)
	}
	backoff := backoffRows[0]
	backoff.ExpiresAt = phaseNow.Add(30 * time.Second).Format(time.RFC3339)
	backoff.CreatedAt = phaseNow.Add(-20 * time.Minute).Format(time.RFC3339)
	_ = app.DB.UpsertControlAction(backoff)
	if err := app.DB.UpsertTransportAnomalySnapshot(db.TransportAnomalySnapshot{BucketStart: phaseNow.Add(9 * time.Minute).Format(time.RFC3339), TransportName: "mqtt-alt", TransportType: "mqtt", Reason: transport.ReasonMalformedFrame, Count: 0, DeadLetters: 0, ObservationDrops: 0}); err != nil {
		t.Fatal(err)
	}
	app.queueRecoveryActions(phaseNow.Add(6 * time.Minute))
	waitFor(t, 2*time.Second, func() bool {
		row, ok, err := app.DB.ControlActionByID(backoff.ID + "-reset")
		return err == nil && ok && row.Result == control.ResultExecutedSuccessfully
	})

	// Phase 9: operator override records the decision and blocks execution.
	app.Cfg.Control.EmergencyDisable = true
	app.evaluateControl(phaseNow.Add(7 * time.Minute))
	decisions, err := app.DB.ControlDecisions("mqtt-primary", "", "", "", 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	foundOverride := false
	for _, item := range decisions {
		if item.DenialCode == control.DenialOverride {
			foundOverride = true
			break
		}
	}
	if !foundOverride {
		t.Fatalf("expected override denial in persisted decisions, got %+v", decisions)
	}

	// Phase 10: process restart recovery closes in-flight action without duplication.
	if err := app.DB.UpsertControlAction(db.ControlActionRecord{ID: "mid-flight", DecisionID: "decision-mid", ActionType: control.ActionRestartTransport, TargetTransport: "mqtt-primary", Reason: "restart in progress", Confidence: 0.9, CreatedAt: phaseNow.Add(8 * time.Minute).Format(time.RFC3339), LifecycleState: control.LifecycleRunning, Mode: control.ModeGuardedAuto}); err != nil {
		t.Fatal(err)
	}
	app.recoverIncompleteControlActions(phaseNow.Add(9 * time.Minute))
	midFlight, ok, err := app.DB.ControlActionByID("mid-flight")
	if err != nil || !ok || midFlight.LifecycleState != control.LifecycleRecovered {
		t.Fatalf("expected recovered in-flight action, got %+v ok=%v err=%v", midFlight, ok, err)
	}

	cancel()
	<-done
}
