package db

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/models"
)

func newTrustTestDB(t *testing.T) *DB {
	t.Helper()
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	d, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// ─── Approval lifecycle ───────────────────────────────────────────────────────

func TestApprovalLifecycle_ApproveAction(t *testing.T) {
	d := newTrustTestDB(t)

	// Insert an action in pending_approval state
	if err := d.UpsertControlAction(ControlActionRecord{
		ID:             "action-pending-1",
		ActionType:     "restart_transport",
		LifecycleState: "pending_approval",
		ExecutionMode:  "approval_required",
		ProposedBy:     "system",
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}

	// Approve it
	if err := d.ApproveControlAction("action-pending-1", "operator@test", "looks good", false, "", ""); err != nil {
		t.Fatalf("ApproveControlAction: %v", err)
	}

	// Verify state transition
	action, ok, err := d.ControlActionByID("action-pending-1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("action not found after approve")
	}
	if action.LifecycleState != "pending" {
		t.Errorf("expected lifecycle_state=pending after approve, got %q", action.LifecycleState)
	}
	if action.Result != "approved" {
		t.Errorf("expected result=approved, got %q", action.Result)
	}
	if action.ApprovedBy != "operator@test" {
		t.Errorf("expected approved_by=operator@test, got %q", action.ApprovedBy)
	}
	if action.ApprovedAt == "" {
		t.Error("expected approved_at to be set")
	}
}

func TestApprovalLifecycle_RejectAction(t *testing.T) {
	d := newTrustTestDB(t)

	if err := d.UpsertControlAction(ControlActionRecord{
		ID:             "action-pending-2",
		ActionType:     "restart_transport",
		LifecycleState: "pending_approval",
		ExecutionMode:  "approval_required",
		ProposedBy:     "system",
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}

	if err := d.RejectControlAction("action-pending-2", "operator@test", "too risky"); err != nil {
		t.Fatalf("RejectControlAction: %v", err)
	}

	action, ok, err := d.ControlActionByID("action-pending-2")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("action not found after reject")
	}
	if action.LifecycleState != "completed" {
		t.Errorf("expected lifecycle_state=completed after reject, got %q", action.LifecycleState)
	}
	if action.Result != "rejected" {
		t.Errorf("expected result=rejected, got %q", action.Result)
	}
	if action.RejectedBy != "operator@test" {
		t.Errorf("expected rejected_by=operator@test, got %q", action.RejectedBy)
	}
}

func TestApprovalLifecycle_ApproveNonPendingFails(t *testing.T) {
	d := newTrustTestDB(t)

	// Insert a completed action (not pending_approval)
	if err := d.UpsertControlAction(ControlActionRecord{
		ID:             "action-completed-1",
		ActionType:     "restart_transport",
		LifecycleState: "completed",
		Result:         "executed_successfully",
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}

	// Approving a non-pending action must not change state
	_ = d.ApproveControlAction("action-completed-1", "operator@test", "", false, "", "")

	action, _, err := d.ControlActionByID("action-completed-1")
	if err != nil {
		t.Fatal(err)
	}
	// State must remain completed, not changed
	if action.LifecycleState != "completed" {
		t.Errorf("approving completed action changed state to %q — state bypass!", action.LifecycleState)
	}
	if action.Result == "approved" {
		t.Error("approving completed action changed result to 'approved' — INVARIANT VIOLATION")
	}
}

func TestApprovalExpiry(t *testing.T) {
	d := newTrustTestDB(t)

	past := time.Now().UTC().Add(-1 * time.Minute).Format(time.RFC3339)
	if err := d.UpsertControlAction(ControlActionRecord{
		ID:                "action-expired-1",
		ActionType:        "restart_transport",
		LifecycleState:    "pending_approval",
		ExecutionMode:     "approval_required",
		ApprovalExpiresAt: past,
		CreatedAt:         time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}

	if err := d.ExpireStaleApprovalActions(time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	action, ok, err := d.ControlActionByID("action-expired-1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("action not found after expiry")
	}
	if action.LifecycleState != "completed" {
		t.Errorf("expected lifecycle_state=completed after expiry, got %q", action.LifecycleState)
	}
	if action.Result != "approval_expired" {
		t.Errorf("expected result=approval_expired, got %q", action.Result)
	}
}

func TestPendingApprovalActions(t *testing.T) {
	d := newTrustTestDB(t)

	for i, state := range []string{"pending_approval", "pending_approval", "completed", "running"} {
		if err := d.UpsertControlAction(ControlActionRecord{
			ID:             "action-list-" + string(rune('0'+i)),
			ActionType:     "restart_transport",
			LifecycleState: state,
			CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			t.Fatal(err)
		}
	}

	pending, err := d.PendingApprovalActions(100)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 2 {
		t.Errorf("expected 2 pending_approval actions, got %d", len(pending))
	}
	for _, a := range pending {
		if a.LifecycleState != "pending_approval" {
			t.Errorf("PendingApprovalActions returned action with state %q", a.LifecycleState)
		}
	}
}

// ─── Control Freezes ─────────────────────────────────────────────────────────

func TestFreeze_CreateAndList(t *testing.T) {
	d := newTrustTestDB(t)

	if err := d.CreateFreeze(FreezeRecord{
		ID:        "frz-1",
		ScopeType: "global",
		Reason:    "emergency stop",
		CreatedBy: "ops",
	}); err != nil {
		t.Fatalf("CreateFreeze: %v", err)
	}

	freezes, err := d.ActiveFreezes()
	if err != nil {
		t.Fatal(err)
	}
	if len(freezes) != 1 {
		t.Fatalf("expected 1 active freeze, got %d", len(freezes))
	}
	if freezes[0].ID != "frz-1" {
		t.Errorf("wrong freeze ID: %q", freezes[0].ID)
	}
	if !freezes[0].Active {
		t.Error("freeze should be active")
	}
}

func TestFreeze_IsFrozen_Global(t *testing.T) {
	d := newTrustTestDB(t)

	if err := d.CreateFreeze(FreezeRecord{
		ID:        "frz-global",
		ScopeType: "global",
		Reason:    "global freeze test",
		CreatedBy: "ops",
	}); err != nil {
		t.Fatal(err)
	}

	frozen, _, err := d.IsFrozen("any-transport", "any_action_type")
	if err != nil {
		t.Fatal(err)
	}
	if !frozen {
		t.Error("expected global freeze to block all actions")
	}
}

func TestFreeze_IsFrozen_TransportScoped(t *testing.T) {
	d := newTrustTestDB(t)

	if err := d.CreateFreeze(FreezeRecord{
		ID:         "frz-transport",
		ScopeType:  "transport",
		ScopeValue: "mqtt-primary",
		Reason:     "transport-scoped freeze",
		CreatedBy:  "ops",
	}); err != nil {
		t.Fatal(err)
	}

	// Frozen for the targeted transport
	frozen, _, err := d.IsFrozen("mqtt-primary", "restart_transport")
	if err != nil {
		t.Fatal(err)
	}
	if !frozen {
		t.Error("expected transport-scoped freeze to block targeted transport")
	}

	// NOT frozen for a different transport
	frozen2, _, err := d.IsFrozen("serial-primary", "restart_transport")
	if err != nil {
		t.Fatal(err)
	}
	if frozen2 {
		t.Error("transport-scoped freeze must not block unrelated transport")
	}
}

func TestFreeze_ClearFreeze(t *testing.T) {
	d := newTrustTestDB(t)

	if err := d.CreateFreeze(FreezeRecord{
		ID:        "frz-clear",
		ScopeType: "global",
		Reason:    "test clear",
		CreatedBy: "ops",
	}); err != nil {
		t.Fatal(err)
	}

	// Confirm frozen
	frozen, _, _ := d.IsFrozen("", "")
	if !frozen {
		t.Fatal("expected freeze to be active before clear")
	}

	if err := d.ClearFreeze("frz-clear", "ops"); err != nil {
		t.Fatal(err)
	}

	// No longer frozen
	frozen, _, _ = d.IsFrozen("", "")
	if frozen {
		t.Error("expected no freeze after clear")
	}

	// Active list must be empty
	freezes, _ := d.ActiveFreezes()
	if len(freezes) != 0 {
		t.Errorf("expected 0 active freezes after clear, got %d", len(freezes))
	}
}

func TestFreeze_Expiry(t *testing.T) {
	d := newTrustTestDB(t)

	past := time.Now().UTC().Add(-1 * time.Minute).Format(time.RFC3339)
	if err := d.CreateFreeze(FreezeRecord{
		ID:        "frz-expiring",
		ScopeType: "global",
		Reason:    "should expire",
		CreatedBy: "ops",
		ExpiresAt: past,
	}); err != nil {
		t.Fatal(err)
	}

	if err := d.ExpireOldFreezes(time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	frozen, _, _ := d.IsFrozen("", "")
	if frozen {
		t.Error("expired freeze must not block actions")
	}
}

// ─── Maintenance Windows ──────────────────────────────────────────────────────

func TestMaintenanceWindow_CreateAndActive(t *testing.T) {
	d := newTrustTestDB(t)

	now := time.Now().UTC()
	mw := MaintenanceWindowRecord{
		ID:        "mw-1",
		Title:     "Nightly maintenance",
		ScopeType: "global",
		StartsAt:  now.Add(-5 * time.Minute).Format(time.RFC3339),
		EndsAt:    now.Add(55 * time.Minute).Format(time.RFC3339),
		CreatedBy: "ops",
	}
	if err := d.CreateMaintenanceWindow(mw); err != nil {
		t.Fatalf("CreateMaintenanceWindow: %v", err)
	}

	inMaint, _, err := d.IsInMaintenance("any", now)
	if err != nil {
		t.Fatal(err)
	}
	if !inMaint {
		t.Error("expected to be in maintenance window")
	}
}

func TestMaintenanceWindow_NotActive_Future(t *testing.T) {
	d := newTrustTestDB(t)

	now := time.Now().UTC()
	mw := MaintenanceWindowRecord{
		ID:        "mw-future",
		Title:     "Future maintenance",
		ScopeType: "global",
		StartsAt:  now.Add(10 * time.Minute).Format(time.RFC3339),
		EndsAt:    now.Add(70 * time.Minute).Format(time.RFC3339),
		CreatedBy: "ops",
	}
	if err := d.CreateMaintenanceWindow(mw); err != nil {
		t.Fatal(err)
	}

	inMaint, _, _ := d.IsInMaintenance("any", now)
	if inMaint {
		t.Error("future maintenance window must not suppress current actions")
	}
}

func TestMaintenanceWindow_Cancel(t *testing.T) {
	d := newTrustTestDB(t)

	now := time.Now().UTC()
	mw := MaintenanceWindowRecord{
		ID:        "mw-cancel",
		Title:     "Cancellable maintenance",
		ScopeType: "global",
		StartsAt:  now.Add(-5 * time.Minute).Format(time.RFC3339),
		EndsAt:    now.Add(55 * time.Minute).Format(time.RFC3339),
		CreatedBy: "ops",
	}
	if err := d.CreateMaintenanceWindow(mw); err != nil {
		t.Fatal(err)
	}

	if err := d.CancelMaintenanceWindow("mw-cancel", "ops"); err != nil {
		t.Fatal(err)
	}

	inMaint, _, _ := d.IsInMaintenance("any", now)
	if inMaint {
		t.Error("cancelled maintenance window must not block actions")
	}
}

// ─── Evidence Bundles ─────────────────────────────────────────────────────────

func TestEvidenceBundle_UpsertAndRetrieve(t *testing.T) {
	d := newTrustTestDB(t)

	if err := d.UpsertControlAction(ControlActionRecord{
		ID:         "action-ev-1",
		ActionType: "restart_transport",
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}

	bundle := EvidenceBundleRecord{
		ID:            "ev-1",
		ActionID:      "action-ev-1",
		PolicyVersion: "v2",
		IntegrityHash: "sha256-abc123",
		SourceType:    "system",
	}
	if err := d.UpsertEvidenceBundle(bundle); err != nil {
		t.Fatalf("UpsertEvidenceBundle: %v", err)
	}

	got, ok, err := d.EvidenceBundleByActionID("action-ev-1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("evidence bundle not found by action ID")
	}
	if got.IntegrityHash != "sha256-abc123" {
		t.Errorf("expected integrity_hash=sha256-abc123, got %q", got.IntegrityHash)
	}
	if got.PolicyVersion != "v2" {
		t.Errorf("expected policy_version=v2, got %q", got.PolicyVersion)
	}
}

// ─── Operator Notes ───────────────────────────────────────────────────────────

func TestOperatorNotes_CreateAndList(t *testing.T) {
	d := newTrustTestDB(t)

	notes := []OperatorNoteRecord{
		{ID: "note-1", RefType: "action", RefID: "action-x", ActorID: "ops1", Content: "first note"},
		{ID: "note-2", RefType: "action", RefID: "action-x", ActorID: "ops2", Content: "second note"},
		{ID: "note-3", RefType: "transport", RefID: "mqtt-1", ActorID: "ops1", Content: "unrelated note"},
	}
	for _, n := range notes {
		if err := d.CreateOperatorNote(n); err != nil {
			t.Fatalf("CreateOperatorNote(%s): %v", n.ID, err)
		}
	}

	got, err := d.OperatorNotesByRef("action", "action-x", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 notes for action-x, got %d", len(got))
	}
	for _, n := range got {
		if n.RefType != "action" || n.RefID != "action-x" {
			t.Errorf("unexpected note: ref_type=%q ref_id=%q", n.RefType, n.RefID)
		}
	}
}

// ─── Timeline ────────────────────────────────────────────────────────────────

func TestTimeline_IncludesMultipleEventTypes(t *testing.T) {
	d := newTrustTestDB(t)

	now := time.Now().UTC()
	ts := now.Format(time.RFC3339)

	// Action
	if err := d.UpsertControlAction(ControlActionRecord{
		ID:             "tl-action-1",
		ActionType:     "restart_transport",
		LifecycleState: "completed",
		CreatedAt:      ts,
	}); err != nil {
		t.Fatal(err)
	}

	// Freeze
	if err := d.CreateFreeze(FreezeRecord{
		ID:        "tl-frz-1",
		ScopeType: "global",
		Reason:    "timeline test",
		CreatedBy: "ops",
	}); err != nil {
		t.Fatal(err)
	}

	// Operator note
	if err := d.CreateOperatorNote(OperatorNoteRecord{
		ID:      "tl-note-1",
		RefType: "action",
		RefID:   "tl-action-1",
		ActorID: "ops",
		Content: "test note for timeline",
	}); err != nil {
		t.Fatal(err)
	}

	start := now.Add(-1 * time.Second).Format(time.RFC3339)
	end := now.Add(5 * time.Second).Format(time.RFC3339)
	events, err := d.TimelineEvents(start, end, 100)
	if err != nil {
		t.Fatalf("TimelineEvents: %v", err)
	}

	typesSeen := make(map[string]bool)
	for _, ev := range events {
		typesSeen[ev.EventType] = true
	}

	if !typesSeen["control_action"] {
		t.Error("timeline missing control_action events")
	}
	if !typesSeen["freeze_created"] {
		t.Error("timeline missing freeze_created events")
	}
	if !typesSeen["operator_note"] {
		t.Error("timeline missing operator_note events")
	}
}

func TestTimelineEventsForIncidentResource_IncludesIncidentOperatorNotes(t *testing.T) {
	d := newTrustTestDB(t)
	incID := "inc-tl-notes-1"
	ts := time.Now().UTC().Format(time.RFC3339)
	if err := d.UpsertIncident(models.Incident{
		ID:           incID,
		Category:     "test",
		Severity:     "info",
		Title:        "t",
		Summary:      "s",
		ResourceType: "transport",
		ResourceID:   "mqtt",
		State:        "open",
		ActorID:      "system",
		OccurredAt:   ts,
		UpdatedAt:    ts,
	}); err != nil {
		t.Fatal(err)
	}
	if err := d.CreateOperatorNote(OperatorNoteRecord{
		ID:      "note-on-inc-1",
		RefType: "incident",
		RefID:   incID,
		ActorID: "ops",
		Content: "handoff context from prior shift",
	}); err != nil {
		t.Fatal(err)
	}
	from := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	to := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	events, err := d.TimelineEventsForIncidentResource(incID, from, to, 50)
	if err != nil {
		t.Fatalf("TimelineEventsForIncidentResource: %v", err)
	}
	found := false
	for _, ev := range events {
		if ev.EventType == "operator_note" && strings.Contains(ev.Summary, "handoff") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected operator_note on incident in scoped timeline, got %#v", events)
	}
}

// ─── Control Plane State ─────────────────────────────────────────────────────

func TestControlPlaneStateSnapshot(t *testing.T) {
	d := newTrustTestDB(t)

	state, err := d.ControlPlaneStateSnapshot(time.Now().UTC())
	if err != nil {
		t.Fatalf("ControlPlaneStateSnapshot: %v", err)
	}

	// Must return a map with essential fields
	if state == nil {
		t.Fatal("state snapshot must not be nil")
	}
	if _, ok := state["active_freezes"]; !ok {
		t.Error("state snapshot missing active_freezes")
	}
	if _, ok := state["active_maintenance"]; !ok {
		t.Error("state snapshot missing active_maintenance")
	}
	if _, ok := state["pending_approvals"]; !ok {
		t.Error("state snapshot missing pending_approvals")
	}
}

// ─── Trust invariant: freeze blocks action listing ────────────────────────────

func TestFreezeBlocksNewActionsInspection(t *testing.T) {
	d := newTrustTestDB(t)

	// No freeze: IsFrozen must be false
	frozen, _, err := d.IsFrozen("test-transport", "restart_transport")
	if err != nil {
		t.Fatal(err)
	}
	if frozen {
		t.Fatal("expected no freeze initially")
	}

	// Install global freeze
	if err := d.CreateFreeze(FreezeRecord{
		ID:        "invariant-frz",
		ScopeType: "global",
		Reason:    "invariant test",
		CreatedBy: "test",
	}); err != nil {
		t.Fatal(err)
	}

	// Now must be frozen
	frozen, reason, err := d.IsFrozen("test-transport", "restart_transport")
	if err != nil {
		t.Fatal(err)
	}
	if !frozen {
		t.Error("INVARIANT VIOLATION: global freeze must block all action types")
	}
	if reason == "" {
		t.Error("freeze reason must be surfaced to caller")
	}

	// Clear freeze
	if err := d.ClearFreeze("invariant-frz", "test"); err != nil {
		t.Fatal(err)
	}

	// Must be unfrozen again
	frozen, _, _ = d.IsFrozen("test-transport", "restart_transport")
	if frozen {
		t.Error("INVARIANT VIOLATION: cleared freeze must not persist")
	}
}
