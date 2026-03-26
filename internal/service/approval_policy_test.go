package service

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
)

func TestEvaluateApprovalPolicy_HighBlastRadiusOnlyWhenConfigEnabled(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "pol.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	a, err := New(cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	rec := db.ControlActionRecord{
		ID:               "act-pol-1",
		ActionType:       control.ActionRestartTransport,
		ExecutionMode:    control.ExecutionModeApprovalRequired,
		LifecycleState:   control.LifecyclePendingApproval,
		BlastRadiusClass: control.BlastRadiusMesh,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
	}
	ev := a.EvaluateApprovalPolicyForRecord(rec, "")
	if ev.HighBlastRadius != true {
		t.Fatalf("expected high blast, got %+v", ev)
	}
	if ev.ApprovalEscalatedDueToBlastRadius {
		t.Fatal("blast radius alone must not imply escalation when config is off")
	}

	a.Cfg.Control.RequireApprovalForHighBlastRadius = true
	rec2 := rec
	rec2.ID = "act-pol-2"
	ev2 := a.EvaluateApprovalPolicyForRecord(rec2, "")
	if !ev2.ApprovalEscalatedDueToBlastRadius {
		t.Fatal("expected escalation when high blast policy enabled and type list did not match")
	}
}

func TestEvaluateApprovalPolicy_ActionTypeList(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "pol2.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	cfg.Control.RequireApprovalForActionTypes = []string{control.ActionRestartTransport}
	a, err := New(cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	rec := db.ControlActionRecord{
		ID:               "act-pol-3",
		ActionType:       control.ActionRestartTransport,
		ExecutionMode:    control.ExecutionModeApprovalRequired,
		LifecycleState:   control.LifecyclePendingApproval,
		BlastRadiusClass: control.BlastRadiusTransport,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
	}
	ev := a.EvaluateApprovalPolicyForRecord(rec, "")
	found := false
	for _, b := range ev.ApprovalBasis {
		if b == "require_approval_for_action_types:"+control.ActionRestartTransport {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected action type basis, got %v", ev.ApprovalBasis)
	}
	if ev.ApprovalEscalatedDueToBlastRadius {
		t.Fatal("type-matched approval should not count as blast-only escalation")
	}
}

func TestEvaluateApprovalPolicy_SoDBlocksSameSubmitter(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "pol3.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	a, err := New(cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	rec := db.ControlActionRecord{
		ID:               "act-pol-4",
		ActionType:       control.ActionRestartTransport,
		ExecutionMode:    control.ExecutionModeApprovalRequired,
		LifecycleState:   control.LifecyclePendingApproval,
		SubmittedBy:      "alice",
		ProposedBy:       "alice",
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		BlastRadiusClass: control.BlastRadiusTransport,
	}
	ev := a.EvaluateApprovalPolicyForRecord(rec, "alice")
	if ev.ApproverAllowed || ev.ApproverDenialReason == "" {
		t.Fatalf("expected SoD denial, got %+v", ev)
	}
}
