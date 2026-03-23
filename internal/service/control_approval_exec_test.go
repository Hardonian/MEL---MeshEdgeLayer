package service

import (
	"context"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
)

// Regression: approval_required actions must not run actuators while still pending_approval.
func TestExecuteControlAction_BlocksUnapprovedApprovalRequired(t *testing.T) {
	a := newTrustTestApp(t)
	act := control.ControlAction{
		ID:               "act-unapproved-exec",
		ActionType:       control.ActionRestartTransport,
		TargetTransport:  "mqtt-test",
		Reason:           "test",
		Confidence:       0.9,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		Mode:             control.ModeGuardedAuto,
		LifecycleState:   control.LifecyclePendingApproval,
		ExecutionMode:    control.ExecutionModeApprovalRequired,
		ProposedBy:       "system",
		BlastRadiusClass: control.BlastRadiusTransport,
	}
	if err := a.DB.UpsertControlAction(controlActionRecord(act)); err != nil {
		t.Fatal(err)
	}

	a.executeControlAction(context.Background(), act)

	rec, ok, err := a.DB.ControlActionByID("act-unapproved-exec")
	if err != nil || !ok {
		t.Fatalf("reload: err=%v ok=%v", err, ok)
	}
	if rec.LifecycleState != control.LifecycleCompleted {
		t.Errorf("expected completed lifecycle after blocked execution, got %q", rec.LifecycleState)
	}
	if rec.Result != control.ResultDeniedByPolicy {
		t.Errorf("expected denied_by_policy result, got %q", rec.Result)
	}
}

func TestApproveControlAction_DBReturnsErrorWhenWrongState(t *testing.T) {
	a := newTrustTestApp(t)
	if err := a.DB.UpsertControlAction(db.ControlActionRecord{
		ID:             "act-wrong",
		ActionType:     control.ActionRestartTransport,
		LifecycleState: control.LifecycleCompleted,
		ExecutionMode:  control.ExecutionModeAuto,
		ProposedBy:     "system",
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.DB.ApproveControlAction("act-wrong", "op", "", false, "", ""); err == nil {
		t.Fatal("expected error approving non-pending action")
	}
}
