package service

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/models"
)

func newSoDTestApp(t *testing.T) *App {
	t.Helper()
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "sod_test.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	cfg.Control.Mode = control.ModeGuardedAuto
	cfg.Control.AllowTransportRestart = true
	cfg.Control.RequireSeparateApprover = true
	cfg.Transports = []config.TransportConfig{{
		Name:    "mqtt-sod",
		Type:    "mqtt",
		Enabled: true,
	}}
	a, err := New(cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func insertSODPending(t *testing.T, d *db.DB, id, submitter string) {
	t.Helper()
	if err := d.UpsertControlAction(db.ControlActionRecord{
		ID:                       id,
		ActionType:               control.ActionRestartTransport,
		TargetTransport:          "mqtt-sod",
		Reason:                   "sod test",
		Confidence:               0.9,
		ExecutionMode:            control.ExecutionModeApprovalRequired,
		LifecycleState:           control.LifecyclePendingApproval,
		ProposedBy:               submitter,
		SubmittedBy:              submitter,
		RequiresSeparateApprover: true,
		CreatedAt:                time.Now().UTC().Format(time.RFC3339),
		BlastRadiusClass:         "transport",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestApproveAction_SoDBlocksSameSubmitter(t *testing.T) {
	a := newSoDTestApp(t)
	insertSODPending(t, a.DB, "act-sod-1", "alice")

	err := a.ApproveAction("act-sod-1", "alice", "self-approve", false, "")
	if err == nil {
		t.Fatal("expected SoD rejection for same submitter/approver")
	}
}

func TestApproveAction_SoDDifferentApproverSucceeds(t *testing.T) {
	a := newSoDTestApp(t)
	insertSODPending(t, a.DB, "act-sod-2", "alice")

	if err := a.ApproveAction("act-sod-2", "bob", "ok", false, ""); err != nil {
		t.Fatalf("ApproveAction: %v", err)
	}
	rec, ok, err := a.DB.ControlActionByID("act-sod-2")
	if err != nil || !ok {
		t.Fatalf("reload: err=%v ok=%v", err, ok)
	}
	if rec.ApprovedBy != "bob" {
		t.Errorf("approved_by=%q", rec.ApprovedBy)
	}
	if rec.SodBypass {
		t.Error("expected no sod bypass for different approver")
	}
}

func TestApproveAction_SoDBreakGlassSameActor(t *testing.T) {
	a := newSoDTestApp(t)
	insertSODPending(t, a.DB, "act-sod-3", "alice")

	if err := a.ApproveAction("act-sod-3", "alice", "note", true, "emergency single operator on call"); err != nil {
		t.Fatalf("ApproveAction: %v", err)
	}
	rec, ok, err := a.DB.ControlActionByID("act-sod-3")
	if err != nil || !ok {
		t.Fatalf("reload: err=%v ok=%v", err, ok)
	}
	if !rec.SodBypass {
		t.Error("expected sod_bypass=1")
	}
	if rec.SodBypassReason == "" {
		t.Error("expected sod_bypass_reason set")
	}
}

func TestApproveAction_NonSoDActionSameActorAllowed(t *testing.T) {
	a := newSoDTestApp(t)
	// Auto execution mode: SoD record flag false — same actor may approve (no maker-checker for non-protected).
	if err := a.DB.UpsertControlAction(db.ControlActionRecord{
		ID:                       "act-nosod",
		ActionType:               control.ActionRestartTransport,
		TargetTransport:          "mqtt-sod",
		Reason:                   "test",
		Confidence:               0.9,
		ExecutionMode:            control.ExecutionModeAuto,
		LifecycleState:           control.LifecyclePendingApproval,
		ProposedBy:               "alice",
		SubmittedBy:              "alice",
		RequiresSeparateApprover: false,
		CreatedAt:                time.Now().UTC().Format(time.RFC3339),
		BlastRadiusClass:         "transport",
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.ApproveAction("act-nosod", "alice", "ok", false, ""); err != nil {
		t.Fatalf("ApproveAction: %v", err)
	}
}

func TestQueueOperatorControlAction_LinksIncident(t *testing.T) {
	a := newSoDTestApp(t)
	incID := "inc-sod-1"
	if err := a.DB.UpsertIncident(models.Incident{
		ID:         incID,
		Category:   "test",
		Severity:   "info",
		Title:      "t",
		Summary:    "s",
		State:      "open",
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	actionID, err := a.QueueOperatorControlAction("alice", control.ActionTriggerHealthRecheck, "mqtt-sod", "", "", "from test", 0.9, incID)
	if err != nil {
		t.Fatalf("QueueOperatorControlAction: %v", err)
	}
	rec, ok, err := a.DB.ControlActionByID(actionID)
	if err != nil || !ok {
		t.Fatalf("reload action: %v %v", err, ok)
	}
	if rec.IncidentID != incID {
		t.Errorf("incident_id=%q want %q", rec.IncidentID, incID)
	}
	if rec.SubmittedBy != "alice" {
		t.Errorf("submitted_by=%q", rec.SubmittedBy)
	}

	linked, err := a.DB.ControlActionsByIncidentID(incID, 10)
	if err != nil || len(linked) != 1 {
		t.Fatalf("linked actions: err=%v n=%d", err, len(linked))
	}
}

func TestIncidentByID_IncludesLinkedActions(t *testing.T) {
	a := newSoDTestApp(t)
	incID := "inc-link-1"
	if err := a.DB.UpsertIncident(models.Incident{
		ID:         incID,
		Category:   "test",
		Severity:   "info",
		Title:      "t",
		Summary:    "s",
		State:      "open",
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.QueueOperatorControlAction("bob", control.ActionTriggerHealthRecheck, "mqtt-sod", "", "", "link test", 0.8, incID); err != nil {
		t.Fatal(err)
	}
	inc, ok, err := a.IncidentByID(incID)
	if err != nil || !ok {
		t.Fatalf("IncidentByID: err=%v ok=%v", err, ok)
	}
	if len(inc.LinkedControlActions) != 1 {
		t.Fatalf("linked_control_actions: got %d", len(inc.LinkedControlActions))
	}
	if inc.LinkedControlActions[0].IncidentID != incID {
		t.Errorf("action incident_id=%q", inc.LinkedControlActions[0].IncidentID)
	}
}
