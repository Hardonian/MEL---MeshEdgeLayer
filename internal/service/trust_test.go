package service

// trust_test.go — Service-layer tests for the control-plane trust model.
//
// Tests cover:
//  - Approval enforcement: approval_required actions are held pending_approval
//  - Freeze blocking: frozen actions are denied at proposal and execution
//  - Maintenance window blocking: actions during active window are denied
//  - Expiry: approval window expiry correctly transitions action to expired state
//  - InspectAction: returns full evidence + decision + notes structure
//  - OperationalState: returns correct mode, freeze count, backlog
//  - Timeline events: approve/reject/freeze/maintenance emit timeline events
//  - Evidence capture: bundle is linked to action on creation

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
)

// ─── Test helpers ─────────────────────────────────────────────────────────────

func newTrustTestApp(t *testing.T) *App {
	t.Helper()
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "trust_test.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	cfg.Control.Mode = control.ModeGuardedAuto
	cfg.Control.AllowTransportRestart = true
	cfg.Transports = []config.TransportConfig{{
		Name:    "mqtt-test",
		Type:    "mqtt",
		Enabled: true,
	}}
	a, err := New(cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func insertPendingApprovalAction(t *testing.T, d *db.DB, id, transportName string) {
	t.Helper()
	if err := d.UpsertControlAction(db.ControlActionRecord{
		ID:               id,
		ActionType:       control.ActionRestartTransport,
		TargetTransport:  transportName,
		Reason:           "test action",
		Confidence:       0.9,
		ExecutionMode:    control.ExecutionModeApprovalRequired,
		LifecycleState:   control.LifecyclePendingApproval,
		ProposedBy:       "system",
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		BlastRadiusClass: "transport",
	}); err != nil {
		t.Fatalf("insertPendingApprovalAction: %v", err)
	}
}

// ─── ApproveAction tests ──────────────────────────────────────────────────────

func TestServiceApproveAction_Succeeds(t *testing.T) {
	a := newTrustTestApp(t)
	insertPendingApprovalAction(t, a.DB, "act-approve-1", "mqtt-test")

	if err := a.ApproveAction("act-approve-1", "operator@test", "looks fine", false, ""); err != nil {
		t.Fatalf("ApproveAction: %v", err)
	}

	rec, ok, err := a.DB.ControlActionByID("act-approve-1")
	if err != nil || !ok {
		t.Fatalf("could not reload action: err=%v ok=%v", err, ok)
	}
	if rec.LifecycleState != control.LifecyclePending {
		t.Errorf("expected lifecycle_state=%q after approve, got %q", control.LifecyclePending, rec.LifecycleState)
	}
	if rec.ApprovedBy != "operator@test" {
		t.Errorf("expected approved_by=operator@test, got %q", rec.ApprovedBy)
	}
	if rec.ApprovedAt == "" {
		t.Error("approved_at must be set after approval")
	}
}

func TestServiceApproveAction_NotFound(t *testing.T) {
	a := newTrustTestApp(t)
	err := a.ApproveAction("nonexistent-action", "op", "", false, "")
	if err == nil {
		t.Fatal("expected error for nonexistent action, got nil")
	}
}

func TestServiceApproveAction_NotPendingApproval(t *testing.T) {
	a := newTrustTestApp(t)
	// Insert an already-completed action
	if err := a.DB.UpsertControlAction(db.ControlActionRecord{
		ID:             "act-completed",
		ActionType:     "restart_transport",
		LifecycleState: "completed",
		ExecutionMode:  "auto",
		ProposedBy:     "system",
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	err := a.ApproveAction("act-completed", "op", "", false, "")
	if err == nil {
		t.Fatal("expected error when approving non-pending action")
	}
}

func TestServiceApproveAction_ExpiredApprovalWindow(t *testing.T) {
	a := newTrustTestApp(t)
	// Insert action with already-expired approval window
	past := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	if err := a.DB.UpsertControlAction(db.ControlActionRecord{
		ID:                "act-expired",
		ActionType:        "restart_transport",
		LifecycleState:    control.LifecyclePendingApproval,
		ExecutionMode:     control.ExecutionModeApprovalRequired,
		ProposedBy:        "system",
		CreatedAt:         time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
		ApprovalExpiresAt: past,
	}); err != nil {
		t.Fatal(err)
	}
	err := a.ApproveAction("act-expired", "op", "", false, "")
	if err == nil {
		t.Fatal("expected error when approving expired action")
	}
}

func TestServiceApproveAction_SameActorBlockedBySoD(t *testing.T) {
	a := newTrustTestApp(t)
	if err := a.DB.UpsertControlAction(db.ControlActionRecord{
		ID:               "act-sod-1",
		ActionType:       control.ActionRestartTransport,
		TargetTransport:  "mqtt-test",
		Reason:           "test",
		Confidence:       0.9,
		ExecutionMode:    control.ExecutionModeApprovalRequired,
		LifecycleState:   control.LifecyclePendingApproval,
		ProposedBy:       "operator-a",
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		BlastRadiusClass: "transport",
	}); err != nil {
		t.Fatal(err)
	}
	err := a.ApproveAction("act-sod-1", "operator-a", "self-approve")
	if err == nil {
		t.Fatal("expected SoD error")
	}
	if !strings.Contains(err.Error(), "separation of duties") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceApproveActionWithOpts_BreakGlassAllowsSameActorSoD(t *testing.T) {
	a := newTrustTestApp(t)
	if err := a.DB.UpsertControlAction(db.ControlActionRecord{
		ID:               "act-sod-bg",
		ActionType:       control.ActionRestartTransport,
		TargetTransport:  "mqtt-test",
		Reason:           "test",
		Confidence:       0.9,
		ExecutionMode:    control.ExecutionModeApprovalRequired,
		LifecycleState:   control.LifecyclePendingApproval,
		ProposedBy:       "operator-a",
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		BlastRadiusClass: "transport",
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.ApproveActionWithOpts("act-sod-bg", "operator-a", "emergency", ApprovalOpts{BreakGlassLegacyCLI: true}); err != nil {
		t.Fatalf("break-glass approve: %v", err)
	}
	rec, ok, err := a.DB.ControlActionByID("act-sod-bg")
	if err != nil || !ok {
		t.Fatalf("reload: err=%v ok=%v", err, ok)
	}
	v, has := rec.Metadata["mel_break_glass_approval"]
	if !has || v != true {
		t.Fatalf("expected mel_break_glass_approval true in metadata, got %#v", rec.Metadata)
	}
}

// ─── RejectAction tests ───────────────────────────────────────────────────────

func TestServiceRejectAction_Succeeds(t *testing.T) {
	a := newTrustTestApp(t)
	insertPendingApprovalAction(t, a.DB, "act-reject-1", "mqtt-test")

	if err := a.RejectAction("act-reject-1", "operator@test", "too risky"); err != nil {
		t.Fatalf("RejectAction: %v", err)
	}

	rec, ok, err := a.DB.ControlActionByID("act-reject-1")
	if err != nil || !ok {
		t.Fatalf("could not reload action: err=%v ok=%v", err, ok)
	}
	if rec.Result != "rejected" {
		t.Errorf("expected result=rejected after reject, got %q", rec.Result)
	}
	if rec.RejectedBy != "operator@test" {
		t.Errorf("expected rejected_by=operator@test, got %q", rec.RejectedBy)
	}
	if rec.ClosureState != "rejected_by_operator" {
		t.Errorf("expected closure_state=rejected_by_operator, got %q", rec.ClosureState)
	}
}

// ─── Freeze tests ─────────────────────────────────────────────────────────────

func TestServiceCreateFreeze_BlocksExecution(t *testing.T) {
	a := newTrustTestApp(t)

	id, err := a.CreateFreeze("global", "", "emergency stop", "operator@test", "")
	if err != nil {
		t.Fatalf("CreateFreeze: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty freeze ID")
	}

	// Verify freeze is in effect
	frozen, reason, err := a.DB.IsFrozen("mqtt-test", "restart_transport")
	if err != nil {
		t.Fatal(err)
	}
	if !frozen {
		t.Error("expected global freeze to block action execution")
	}
	if reason == "" {
		t.Error("expected non-empty freeze reason")
	}
}

func TestServiceCreateFreeze_ScopedTransport(t *testing.T) {
	a := newTrustTestApp(t)

	_, err := a.CreateFreeze("transport", "mqtt-test", "mqtt maintenance", "op", "")
	if err != nil {
		t.Fatalf("CreateFreeze: %v", err)
	}

	// Frozen for targeted transport
	frozen, _, _ := a.DB.IsFrozen("mqtt-test", "restart_transport")
	if !frozen {
		t.Error("expected transport-scoped freeze to block targeted transport")
	}

	// NOT frozen for different transport
	frozen2, _, _ := a.DB.IsFrozen("serial-primary", "restart_transport")
	if frozen2 {
		t.Error("transport-scoped freeze must not block unrelated transport")
	}
}

func TestServiceClearFreeze_RemovesBlock(t *testing.T) {
	a := newTrustTestApp(t)

	id, _ := a.CreateFreeze("global", "", "test freeze", "op", "")

	// Confirm frozen
	frozen, _, _ := a.DB.IsFrozen("", "")
	if !frozen {
		t.Fatal("freeze should be active before clear")
	}

	if err := a.ClearFreeze(id, "op"); err != nil {
		t.Fatalf("ClearFreeze: %v", err)
	}

	frozen2, _, _ := a.DB.IsFrozen("", "")
	if frozen2 {
		t.Error("freeze should be inactive after clear")
	}
}

// ─── Maintenance window tests ─────────────────────────────────────────────────

func TestServiceCreateMaintenanceWindow_BlocksDuringWindow(t *testing.T) {
	a := newTrustTestApp(t)

	now := time.Now().UTC()
	start := now.Add(-5 * time.Minute).Format(time.RFC3339)
	end := now.Add(30 * time.Minute).Format(time.RFC3339)

	id, err := a.CreateMaintenanceWindow("Test Maintenance", "scheduled test", "global", "", "op", start, end)
	if err != nil {
		t.Fatalf("CreateMaintenanceWindow: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty window ID")
	}

	// Verify maintenance is in effect now
	inMaint, reason, err := a.DB.IsInMaintenance("mqtt-test", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if !inMaint {
		t.Error("expected active maintenance window to block execution")
	}
	if reason == "" {
		t.Error("expected non-empty maintenance reason")
	}
}

func TestServiceCancelMaintenanceWindow(t *testing.T) {
	a := newTrustTestApp(t)

	now := time.Now().UTC()
	id, _ := a.CreateMaintenanceWindow("Test", "test", "global", "", "op",
		now.Add(-5*time.Minute).Format(time.RFC3339),
		now.Add(30*time.Minute).Format(time.RFC3339),
	)

	if err := a.CancelMaintenanceWindow(id, "op"); err != nil {
		t.Fatalf("CancelMaintenanceWindow: %v", err)
	}

	// Should no longer be in maintenance
	inMaint, _, _ := a.DB.IsInMaintenance("any", time.Now().UTC())
	if inMaint {
		t.Error("maintenance should be inactive after cancel")
	}
}

// ─── InspectAction tests ──────────────────────────────────────────────────────

func TestServiceInspectAction_ReturnsFullBundle(t *testing.T) {
	a := newTrustTestApp(t)
	insertPendingApprovalAction(t, a.DB, "act-inspect-1", "mqtt-test")

	// Add an evidence bundle
	bundleID := "eb-inspect-1"
	if err := a.DB.UpsertEvidenceBundle(db.EvidenceBundleRecord{
		ID:          bundleID,
		ActionID:    "act-inspect-1",
		Explanation: map[string]any{"reason": "test evidence"},
	}); err != nil {
		t.Fatal(err)
	}
	// Link bundle to action
	rec, _, _ := a.DB.ControlActionByID("act-inspect-1")
	rec.EvidenceBundleID = bundleID
	_ = a.DB.UpsertControlAction(rec)

	// Add an operator note
	_, _ = a.AddOperatorNote("action", "act-inspect-1", "op@test", "investigating this")

	result, err := a.InspectAction("act-inspect-1")
	if err != nil {
		t.Fatalf("InspectAction: %v", err)
	}

	if _, ok := result["action"]; !ok {
		t.Error("inspect result missing 'action' field")
	}
	if _, ok := result["evidence_bundle"]; !ok {
		t.Error("inspect result missing 'evidence_bundle' field")
	}
	if _, ok := result["notes"]; !ok {
		t.Error("inspect result missing 'notes' field")
	}
	if _, ok := result["inspected_at"]; !ok {
		t.Error("inspect result missing 'inspected_at' field")
	}
	if ov, ok := result["operator_view"].(map[string]any); !ok || ov["queue_status"] == nil {
		t.Error("inspect result missing operator_view with queue_status")
	}

	// Evidence bundle should be present since we linked it
	if result["evidence_bundle"] == nil {
		t.Error("inspect result has nil evidence_bundle despite bundle being set")
	}

	// Notes should contain our note
	notes, _ := result["notes"].([]db.OperatorNoteRecord)
	if len(notes) == 0 {
		t.Error("expected at least one operator note in inspect result")
	}
}

func TestServiceInspectAction_NotFound(t *testing.T) {
	a := newTrustTestApp(t)
	_, err := a.InspectAction("does-not-exist")
	if err == nil {
		t.Fatal("expected error for nonexistent action")
	}
}

// ─── OperationalState tests ───────────────────────────────────────────────────

func TestServiceOperationalState_NormalMode(t *testing.T) {
	a := newTrustTestApp(t)
	state, err := a.OperationalState()
	if err != nil {
		t.Fatalf("OperationalState: %v", err)
	}
	if _, ok := state["automation_mode"]; !ok {
		t.Error("operational state missing 'automation_mode'")
	}
	if _, ok := state["freeze_count"]; !ok {
		t.Error("operational state missing 'freeze_count'")
	}
	if _, ok := state["approval_backlog"]; !ok {
		t.Error("operational state missing 'approval_backlog'")
	}
}

func TestServiceOperationalState_FrozenMode(t *testing.T) {
	a := newTrustTestApp(t)

	_, err := a.CreateFreeze("global", "", "test freeze", "op", "")
	if err != nil {
		t.Fatalf("CreateFreeze: %v", err)
	}

	state, err := a.OperationalState()
	if err != nil {
		t.Fatalf("OperationalState: %v", err)
	}
	if state["automation_mode"] != "frozen" {
		t.Errorf("expected automation_mode=frozen, got %v", state["automation_mode"])
	}
	fc, _ := state["freeze_count"].(int)
	if fc < 1 {
		t.Errorf("expected freeze_count >= 1, got %v", state["freeze_count"])
	}
}

func TestServiceOperationalState_ApprovalBacklog(t *testing.T) {
	a := newTrustTestApp(t)

	// Insert some pending-approval actions
	for i := 0; i < 3; i++ {
		insertPendingApprovalAction(t, a.DB, "act-backlog-"+string(rune('a'+i)), "mqtt-test")
	}

	state, err := a.OperationalState()
	if err != nil {
		t.Fatalf("OperationalState: %v", err)
	}
	bl, _ := state["approval_backlog"].(int)
	if bl < 3 {
		t.Errorf("expected approval_backlog >= 3, got %v", state["approval_backlog"])
	}
	pending, _ := state["pending_approvals"].([]map[string]any)
	if len(pending) < 3 {
		t.Fatalf("expected enriched pending_approvals, got %T %#v", state["pending_approvals"], state["pending_approvals"])
	}
	if _, ok := pending[0]["operator_view"].(map[string]any); !ok {
		t.Error("expected operator_view on pending approval row")
	}
}

// ─── Timeline event tests ─────────────────────────────────────────────────────

func TestServiceTimelineEvents_ApproveEmitsEvent(t *testing.T) {
	a := newTrustTestApp(t)
	insertPendingApprovalAction(t, a.DB, "act-tl-approve", "mqtt-test")

	before := time.Now().UTC()
	_ = a.ApproveAction("act-tl-approve", "op@test", "approved in test", false, "")
	after := time.Now().UTC()

	events, err := a.Timeline(
		before.Add(-time.Second).Format(time.RFC3339),
		after.Add(time.Second).Format(time.RFC3339),
		100,
	)
	if err != nil {
		t.Fatalf("Timeline: %v", err)
	}

	found := false
	for _, ev := range events {
		if ev.EventType == "action_approved" && ev.ResourceID == "act-tl-approve" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected action_approved timeline event after ApproveAction")
	}
}

func TestServiceTimelineEvents_RejectEmitsEvent(t *testing.T) {
	a := newTrustTestApp(t)
	insertPendingApprovalAction(t, a.DB, "act-tl-reject", "mqtt-test")

	before := time.Now().UTC()
	_ = a.RejectAction("act-tl-reject", "op@test", "rejected in test")
	after := time.Now().UTC()

	events, err := a.Timeline(
		before.Add(-time.Second).Format(time.RFC3339),
		after.Add(time.Second).Format(time.RFC3339),
		100,
	)
	if err != nil {
		t.Fatalf("Timeline: %v", err)
	}

	found := false
	for _, ev := range events {
		if ev.EventType == "action_rejected" && ev.ResourceID == "act-tl-reject" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected action_rejected timeline event after RejectAction")
	}
}

func TestServiceTimelineEvents_FreezeEmitsEvent(t *testing.T) {
	a := newTrustTestApp(t)

	before := time.Now().UTC()
	id, _ := a.CreateFreeze("global", "", "test freeze", "op@test", "")
	_ = a.ClearFreeze(id, "op@test")
	after := time.Now().UTC()

	events, err := a.Timeline(
		before.Add(-time.Second).Format(time.RFC3339),
		after.Add(time.Second).Format(time.RFC3339),
		100,
	)
	if err != nil {
		t.Fatalf("Timeline: %v", err)
	}

	hasCreated, hasCleared := false, false
	for _, ev := range events {
		if ev.EventType == "freeze_created" {
			hasCreated = true
		}
		if ev.EventType == "freeze_cleared" {
			hasCleared = true
		}
	}
	if !hasCreated {
		t.Error("expected freeze_created timeline event")
	}
	if !hasCleared {
		t.Error("expected freeze_cleared timeline event")
	}
}

// ─── Operator notes tests ─────────────────────────────────────────────────────

func TestServiceAddOperatorNote(t *testing.T) {
	a := newTrustTestApp(t)
	insertPendingApprovalAction(t, a.DB, "act-note-1", "mqtt-test")

	id, err := a.AddOperatorNote("action", "act-note-1", "op@test", "monitoring this closely")
	if err != nil {
		t.Fatalf("AddOperatorNote: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty note ID")
	}

	notes, err := a.DB.OperatorNotesByRef("action", "act-note-1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) == 0 {
		t.Fatal("expected at least one note after add")
	}
	if notes[0].Content != "monitoring this closely" {
		t.Errorf("unexpected note content: %q", notes[0].Content)
	}
	if notes[0].ActorID != "op@test" {
		t.Errorf("unexpected actor_id: %q", notes[0].ActorID)
	}
}

// ─── cleanupExpiredApprovals tests ────────────────────────────────────────────

func TestServiceCleanupExpiredApprovals_ExpiresStaleActions(t *testing.T) {
	a := newTrustTestApp(t)

	// Insert action whose approval window is already expired
	past := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	if err := a.DB.UpsertControlAction(db.ControlActionRecord{
		ID:                "act-cleanup-expired",
		ActionType:        "restart_transport",
		LifecycleState:    control.LifecyclePendingApproval,
		ExecutionMode:     control.ExecutionModeApprovalRequired,
		ProposedBy:        "system",
		CreatedAt:         time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
		ApprovalExpiresAt: past,
	}); err != nil {
		t.Fatal(err)
	}

	a.cleanupExpiredApprovals()

	rec, ok, err := a.DB.ControlActionByID("act-cleanup-expired")
	if err != nil || !ok {
		t.Fatalf("could not reload action: err=%v ok=%v", err, ok)
	}
	if rec.LifecycleState != "completed" {
		t.Errorf("expected expired action to be completed, got %q", rec.LifecycleState)
	}
	if rec.Result != "approval_expired" {
		t.Errorf("expected result=approval_expired, got %q", rec.Result)
	}
}

func TestServiceCleanupExpiredApprovals_ExpiresOldFreezes(t *testing.T) {
	a := newTrustTestApp(t)

	// Insert freeze with already-expired expires_at
	past := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	if err := a.DB.CreateFreeze(db.FreezeRecord{
		ID:        "frz-expired",
		ScopeType: "global",
		Reason:    "test",
		CreatedBy: "system",
		ExpiresAt: past,
	}); err != nil {
		t.Fatal(err)
	}

	// Verify freeze is active before cleanup
	frozen, _, _ := a.DB.IsFrozen("", "")
	if !frozen {
		t.Fatal("expected freeze to be active before cleanup")
	}

	a.cleanupExpiredApprovals()

	// Freeze should now be expired
	frozen2, _, _ := a.DB.IsFrozen("", "")
	if frozen2 {
		t.Error("expected freeze to be inactive after expiry cleanup")
	}
}
