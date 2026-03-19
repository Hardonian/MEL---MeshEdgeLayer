package db

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
)

func TestControlActionAndDecisionPersistence(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertControlAction(ControlActionRecord{
		ID:              "action-1",
		DecisionID:      "decision-1",
		ActionType:      "restart_transport",
		TargetTransport: "mqtt",
		Reason:          "retry threshold exceeded",
		Confidence:      0.95,
		TriggerEvidence: []string{"retry threshold exceeded twice"},
		CreatedAt:       "2026-03-19T12:00:00Z",
		Result:          "executed_successfully",
		LifecycleState:  "completed",
		Mode:            "guarded_auto",
	}); err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertControlDecision(ControlDecisionRecord{
		ID:                "decision-1",
		CandidateActionID: "action-1",
		ActionType:        "restart_transport",
		TargetTransport:   "mqtt",
		Reason:            "retry threshold exceeded",
		Confidence:        0.95,
		Allowed:           true,
		SafetyChecks:      map[string]any{"policy_pass": true, "blast_radius_class": "local_transport"},
		DecisionInputs:    map[string]any{"mesh_state": "failed"},
		PolicySummary:     map[string]any{"mode": "guarded_auto"},
		CreatedAt:         "2026-03-19T12:00:00Z",
		Mode:              "guarded_auto",
	}); err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertControlActionReality(ControlActionRealityRecord{
		ActionType:         "restart_transport",
		ActuatorExists:     true,
		Reversible:         true,
		BlastRadiusKnown:   true,
		BlastRadiusClass:   "local_transport",
		SafeForGuardedAuto: true,
		UpdatedAt:          "2026-03-19T12:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	actions, err := d.ControlActions("mqtt", "", "", "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0].ActionType != "restart_transport" {
		t.Fatalf("unexpected actions: %+v", actions)
	}
	decisions, err := d.ControlDecisions("mqtt", "", "", "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 1 || !decisions[0].Allowed {
		t.Fatalf("unexpected decisions: %+v", decisions)
	}
	realities, err := d.ControlActionRealities()
	if err != nil {
		t.Fatal(err)
	}
	if len(realities) != 1 || realities[0].ActionType != "restart_transport" {
		t.Fatalf("unexpected control realities: %+v", realities)
	}
	if err := d.PruneControlHistory(time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), 100); err != nil {
		t.Fatal(err)
	}
	actions, err = d.ControlActions("mqtt", "", "", "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 0 {
		t.Fatalf("expected pruned actions, got %+v", actions)
	}
}
