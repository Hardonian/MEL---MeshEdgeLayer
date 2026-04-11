package service

// Tests for runbook review + application service layer.
// These use the SoD test harness to get a real SQLite DB with migrations.

import (
	"testing"
	"time"

	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/models"
)

// insertTestRunbook is a helper that writes a proposed runbook directly to the DB.
func insertTestRunbook(t *testing.T, a *App, id, sigKey, title string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	if err := a.DB.InsertRunbookEntry(db.RunbookEntryRecord{
		ID:                    id,
		Status:                db.RunbookStatusProposed,
		SourceKind:            "test_fixture",
		LegacySignatureKey:    sigKey,
		Title:                 title,
		Body:                  "Test runbook body for " + id,
		EvidenceRefJSON:       `["incident:seed-1"]`,
		SourceIncidentIDsJSON: `["seed-1"]`,
		PromotionBasis:        "test seed",
		CreatedAt:             now,
		UpdatedAt:             now,
	}); err != nil {
		t.Fatal(err)
	}
}

// insertTestIncident is a helper for making a minimal open incident.
func insertTestIncident(t *testing.T, a *App, id, title string) {
	t.Helper()
	if err := a.DB.UpsertIncident(models.Incident{
		ID:           id,
		Category:     "transport",
		Severity:     "warning",
		Title:        title,
		Summary:      "s",
		ResourceType: "transport",
		ResourceID:   "mqtt-sod",
		State:        "open",
		OccurredAt:   time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
}

func TestListRunbookEntries_FiltersByStatus(t *testing.T) {
	a := newSoDTestApp(t)
	insertTestRunbook(t, a, "rb-filter-1", "sig:a", "Candidate A")
	insertTestRunbook(t, a, "rb-filter-2", "sig:a", "Candidate B")
	if err := a.PromoteRunbookEntry("rb-filter-2", "alice", "looks good"); err != nil {
		t.Fatal(err)
	}
	proposed, err := a.ListRunbookEntries(db.RunbookStatusProposed, "", "", "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(proposed) != 1 || proposed[0].ID != "rb-filter-1" {
		t.Fatalf("expected only rb-filter-1 in proposed list, got %+v", proposed)
	}
	promoted, err := a.ListRunbookEntries(db.RunbookStatusPromoted, "", "", "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(promoted) != 1 || promoted[0].ID != "rb-filter-2" {
		t.Fatalf("expected only rb-filter-2 in promoted list, got %+v", promoted)
	}
	if promoted[0].PromotedByActorID != "alice" {
		t.Errorf("expected promoted_by_actor_id=alice, got %q", promoted[0].PromotedByActorID)
	}
	if promoted[0].PromotedAt == "" {
		t.Errorf("expected promoted_at to be set")
	}
}

func TestPromoteRunbookEntry_NotFound(t *testing.T) {
	a := newSoDTestApp(t)
	err := a.PromoteRunbookEntry("rb-missing", "alice", "")
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDeprecateRunbookEntry_RequiresReason(t *testing.T) {
	a := newSoDTestApp(t)
	insertTestRunbook(t, a, "rb-dep-1", "sig:b", "To be deprecated")
	if err := a.DeprecateRunbookEntry("rb-dep-1", "alice", ""); err == nil {
		t.Fatal("expected error when reason is missing")
	}
	if err := a.DeprecateRunbookEntry("rb-dep-1", "alice", "superseded by newer guidance"); err != nil {
		t.Fatal(err)
	}
	detail, ok, err := a.GetRunbookEntry("rb-dep-1")
	if err != nil || !ok {
		t.Fatalf("reload: err=%v ok=%v", err, ok)
	}
	if detail.Entry.Status != db.RunbookStatusDeprecated {
		t.Errorf("status=%q want deprecated", detail.Entry.Status)
	}
	if detail.Entry.DeprecatedReason != "superseded by newer guidance" {
		t.Errorf("deprecated_reason=%q", detail.Entry.DeprecatedReason)
	}
	if detail.Entry.DeprecatedByActorID != "alice" {
		t.Errorf("deprecated_by_actor_id=%q", detail.Entry.DeprecatedByActorID)
	}
}

func TestApplyRunbookToIncident_BumpsCountersAndAudit(t *testing.T) {
	a := newSoDTestApp(t)
	insertTestRunbook(t, a, "rb-apply-1", "sig:c", "Restart mqtt tier")
	insertTestIncident(t, a, "inc-apply-1", "mqtt down")

	rec, err := a.ApplyRunbookToIncident("inc-apply-1", "alice", models.ApplyRunbookRequest{
		RunbookID: "rb-apply-1",
		Outcome:   db.RunbookOutcomeHelped,
		Note:      "worked on first try",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Outcome != db.RunbookOutcomeHelped {
		t.Errorf("outcome=%q", rec.Outcome)
	}
	if rec.IncidentID != "inc-apply-1" || rec.RunbookID != "rb-apply-1" {
		t.Errorf("link fields=%+v", rec)
	}

	detail, ok, err := a.GetRunbookEntry("rb-apply-1")
	if err != nil || !ok {
		t.Fatalf("reload detail: err=%v ok=%v", err, ok)
	}
	if detail.Entry.AppliedCount != 1 {
		t.Errorf("applied_count=%d want 1", detail.Entry.AppliedCount)
	}
	if detail.Entry.UsefulCount != 1 {
		t.Errorf("useful_count=%d want 1", detail.Entry.UsefulCount)
	}
	if detail.Entry.LastAppliedIncidentID != "inc-apply-1" {
		t.Errorf("last_applied_incident_id=%q", detail.Entry.LastAppliedIncidentID)
	}
	if len(detail.Applications) != 1 || detail.Applications[0].Outcome != db.RunbookOutcomeHelped {
		t.Errorf("applications=%+v", detail.Applications)
	}

	// Second application with did_not_help should bump ineffective_count.
	if _, err := a.ApplyRunbookToIncident("inc-apply-1", "bob", models.ApplyRunbookRequest{
		RunbookID: "rb-apply-1",
		Outcome:   db.RunbookOutcomeDidNotHelp,
	}); err != nil {
		t.Fatal(err)
	}
	detail2, _, _ := a.GetRunbookEntry("rb-apply-1")
	if detail2.Entry.AppliedCount != 2 {
		t.Errorf("applied_count=%d want 2", detail2.Entry.AppliedCount)
	}
	if detail2.Entry.IneffectiveCount != 1 {
		t.Errorf("ineffective_count=%d want 1", detail2.Entry.IneffectiveCount)
	}
	if detail2.Entry.UsefulCount != 1 {
		t.Errorf("useful_count=%d want 1", detail2.Entry.UsefulCount)
	}
}

func TestApplyRunbookToIncident_InvalidOutcomeRejected(t *testing.T) {
	a := newSoDTestApp(t)
	insertTestRunbook(t, a, "rb-bad-outcome", "sig:d", "x")
	insertTestIncident(t, a, "inc-bad-outcome", "t")
	_, err := a.ApplyRunbookToIncident("inc-bad-outcome", "alice", models.ApplyRunbookRequest{
		RunbookID: "rb-bad-outcome",
		Outcome:   "not_a_real_outcome",
	})
	if err == nil {
		t.Fatal("expected error on invalid outcome")
	}
}

func TestApplyRunbookToIncident_MissingRow(t *testing.T) {
	a := newSoDTestApp(t)
	insertTestIncident(t, a, "inc-missing-rb", "t")
	_, err := a.ApplyRunbookToIncident("inc-missing-rb", "alice", models.ApplyRunbookRequest{
		RunbookID: "rb-does-not-exist",
	})
	if err == nil {
		t.Fatal("expected error when runbook is missing")
	}
	insertTestRunbook(t, a, "rb-orphan", "sig:e", "x")
	_, err = a.ApplyRunbookToIncident("inc-does-not-exist", "alice", models.ApplyRunbookRequest{
		RunbookID: "rb-orphan",
	})
	if err == nil {
		t.Fatal("expected error when incident is missing")
	}
}

func TestBuildOperatorWorklist_CollectsOwnedPendingAndRunbooks(t *testing.T) {
	a := newSoDTestApp(t)

	// Owned by alice, pending_review flagged via workflow patch.
	insertTestIncident(t, a, "inc-w-owned", "owned by alice")
	inc, ok, _ := a.DB.IncidentByID("inc-w-owned")
	if !ok {
		t.Fatal("seed missing")
	}
	inc.OwnerActorID = "alice"
	rs := "pending_review"
	inc.ReviewState = rs
	if err := a.DB.UpsertIncident(inc); err != nil {
		t.Fatal(err)
	}

	// Follow-up needed on a different incident
	insertTestIncident(t, a, "inc-w-followup", "follow-up")
	fu, _, _ := a.DB.IncidentByID("inc-w-followup")
	fu.ReviewState = "follow_up_needed"
	if err := a.DB.UpsertIncident(fu); err != nil {
		t.Fatal(err)
	}

	// Proposed runbook candidate
	insertTestRunbook(t, a, "rb-worklist", "sig:w", "worklist candidate")

	worklist, err := a.BuildOperatorWorklist("alice")
	if err != nil {
		t.Fatal(err)
	}
	if worklist.ActorID != "alice" {
		t.Errorf("actor=%q", worklist.ActorID)
	}
	if len(worklist.OwnedOpenIncidents) != 1 || worklist.OwnedOpenIncidents[0].IncidentID != "inc-w-owned" {
		t.Errorf("owned=%+v", worklist.OwnedOpenIncidents)
	}
	if len(worklist.PendingReview) != 1 || worklist.PendingReview[0].IncidentID != "inc-w-owned" {
		t.Errorf("pending_review=%+v", worklist.PendingReview)
	}
	if len(worklist.FollowUpNeeded) != 1 || worklist.FollowUpNeeded[0].IncidentID != "inc-w-followup" {
		t.Errorf("follow_up=%+v", worklist.FollowUpNeeded)
	}
	if len(worklist.RunbookCandidates) != 1 || worklist.RunbookCandidates[0].ID != "rb-worklist" {
		t.Errorf("runbook_candidates=%+v", worklist.RunbookCandidates)
	}
	if worklist.Counts.RunbookCandidates != 1 {
		t.Errorf("counts.runbook_candidates=%d", worklist.Counts.RunbookCandidates)
	}
	if worklist.Counts.OwnedOpen != 1 {
		t.Errorf("counts.owned_open=%d", worklist.Counts.OwnedOpen)
	}
	if len(worklist.EvidenceBasis) == 0 {
		t.Errorf("expected evidence_basis to be populated")
	}
}

func TestBuildOperatorWorklist_PendingApprovalsSoDAware(t *testing.T) {
	a := newSoDTestApp(t)
	insertSODPending(t, a.DB, "act-w-1", "alice")
	insertSODPending(t, a.DB, "act-w-2", "bob")

	// Alice should only see bob's row (hers is SoD-excluded).
	alice, err := a.BuildOperatorWorklist("alice")
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, p := range alice.PendingApprovals {
		ids[p.ActionID] = true
	}
	if ids["act-w-1"] {
		t.Errorf("alice should not see her own pending action in worklist")
	}
	if !ids["act-w-2"] {
		t.Errorf("alice should see bob's pending action in worklist")
	}
}

func TestBuildShiftHandoffPacket_BoundedWindow(t *testing.T) {
	a := newSoDTestApp(t)
	insertTestIncident(t, a, "inc-shift-open", "opened during shift")
	insertTestIncident(t, a, "inc-shift-resolved", "resolved during shift")
	// Mark the second incident resolved now so it lands in the window.
	inc, _, _ := a.DB.IncidentByID("inc-shift-resolved")
	inc.State = "resolved"
	inc.ResolvedAt = time.Now().UTC().Format(time.RFC3339)
	if err := a.DB.UpsertIncident(inc); err != nil {
		t.Fatal(err)
	}

	insertTestRunbook(t, a, "rb-shift-prop", "sig:sh", "shift candidate")

	packet, err := a.BuildShiftHandoffPacket("alice", 8)
	if err != nil {
		t.Fatal(err)
	}
	if packet.WindowHours != 8 {
		t.Errorf("window_hours=%d", packet.WindowHours)
	}
	if packet.WindowStart == "" || packet.WindowEnd == "" {
		t.Errorf("window bounds missing")
	}
	var openedHas, resolvedHas bool
	for _, it := range packet.OpenedIncidents {
		if it.IncidentID == "inc-shift-open" {
			openedHas = true
		}
	}
	for _, it := range packet.ResolvedIncidents {
		if it.IncidentID == "inc-shift-resolved" {
			resolvedHas = true
		}
	}
	if !openedHas {
		t.Errorf("opened_incidents missing inc-shift-open: %+v", packet.OpenedIncidents)
	}
	if !resolvedHas {
		t.Errorf("resolved_incidents missing inc-shift-resolved: %+v", packet.ResolvedIncidents)
	}
	var candHas bool
	for _, c := range packet.RunbookCandidates {
		if c.ID == "rb-shift-prop" {
			candHas = true
		}
	}
	if !candHas {
		t.Errorf("runbook_candidates_created missing rb-shift-prop: %+v", packet.RunbookCandidates)
	}
	if packet.Counts.OpenedIncidents < 1 || packet.Counts.ResolvedIncidents < 1 || packet.Counts.RunbookCandidates < 1 {
		t.Errorf("counts=%+v", packet.Counts)
	}
	if len(packet.EvidenceBasis) == 0 {
		t.Errorf("expected evidence_basis populated")
	}
}

func TestBuildShiftHandoffPacket_WindowClamps(t *testing.T) {
	a := newSoDTestApp(t)
	p, err := a.BuildShiftHandoffPacket("", 0)
	if err != nil {
		t.Fatal(err)
	}
	if p.WindowHours != 8 {
		t.Errorf("default window_hours=%d want 8", p.WindowHours)
	}
	p, err = a.BuildShiftHandoffPacket("", 9999)
	if err != nil {
		t.Fatal(err)
	}
	if p.WindowHours != 72 {
		t.Errorf("capped window_hours=%d want 72", p.WindowHours)
	}
}
