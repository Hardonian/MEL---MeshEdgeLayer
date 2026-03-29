package proofpack

import (
	"fmt"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/models"
)

// mockDataSource implements DataSource for testing.
type mockDataSource struct {
	incident          models.Incident
	incidentFound     bool
	incidentErr       error
	actions           []ActionEvidence
	actionsErr        error
	timeline          []TimelineEntry
	timelineErr       error
	transports        []TransportSnapshot
	transportsErr     error
	deadLetters       []DeadLetterEntry
	deadLettersErr    error
	notes             []OperatorNote
	notesErr          error
	auditEntries      []AuditEntry
	auditErr          error
	signatureKey      string
	signatureKeyErr   error
	actionOutcomes    []ActionOutcomeSnapshot
	actionOutcomesErr error
	recOutcomes       []RecommendationOutcomeEntry
	recOutcomesErr    error
	corrGroups        []CorrelationGroupEntry
	corrGroupsErr     error
}

func (m *mockDataSource) IncidentByID(id string) (models.Incident, bool, error) {
	return m.incident, m.incidentFound, m.incidentErr
}

func (m *mockDataSource) ControlActionsByIncidentID(incidentID string, limit int) ([]ActionEvidence, error) {
	return m.actions, m.actionsErr
}

func (m *mockDataSource) SignatureKeyForIncident(incidentID string) (string, error) {
	return m.signatureKey, m.signatureKeyErr
}

func (m *mockDataSource) ActionOutcomeSnapshotsBySignature(signatureKey, excludeIncidentID string, limit int) ([]ActionOutcomeSnapshot, error) {
	return m.actionOutcomes, m.actionOutcomesErr
}

func (m *mockDataSource) TimelineEventsForIncident(incidentID, from, to string, limit int) ([]TimelineEntry, error) {
	return m.timeline, m.timelineErr
}

func (m *mockDataSource) TransportHealthSnapshotsInWindow(from, to string, limit int) ([]TransportSnapshot, error) {
	return m.transports, m.transportsErr
}

func (m *mockDataSource) DeadLettersInWindow(from, to string, limit int) ([]DeadLetterEntry, error) {
	return m.deadLetters, m.deadLettersErr
}

func (m *mockDataSource) OperatorNotesForResource(refType, refID string, limit int) ([]OperatorNote, error) {
	return m.notes, m.notesErr
}

func (m *mockDataSource) AuditEntriesForResource(resourceType, resourceID string, limit int) ([]AuditEntry, error) {
	return m.auditEntries, m.auditErr
}

func (m *mockDataSource) RecommendationOutcomesForIncident(incidentID string, limit int) ([]RecommendationOutcomeEntry, error) {
	return m.recOutcomes, m.recOutcomesErr
}

func (m *mockDataSource) CorrelationGroupsForIncident(incidentID string) ([]CorrelationGroupEntry, error) {
	return m.corrGroups, m.corrGroupsErr
}

func TestAssemble_EmptyIncidentID(t *testing.T) {
	a := NewAssembler(&mockDataSource{}, DefaultConfig())
	_, err := a.Assemble("")
	if err == nil {
		t.Fatal("expected error for empty incident ID")
	}
}

func TestAssemble_IncidentNotFound(t *testing.T) {
	src := &mockDataSource{incidentFound: false}
	a := NewAssembler(src, DefaultConfig())
	_, err := a.Assemble("inc-001")
	if err == nil || err.Error() != "incident not found: inc-001" {
		t.Fatalf("expected 'incident not found' error, got: %v", err)
	}
}

func TestAssemble_IncidentLoadError(t *testing.T) {
	src := &mockDataSource{incidentErr: fmt.Errorf("db offline")}
	a := NewAssembler(src, DefaultConfig())
	_, err := a.Assemble("inc-001")
	if err == nil {
		t.Fatal("expected error when incident load fails")
	}
}

func TestAssemble_FullProofpack(t *testing.T) {
	now := time.Now().UTC()
	src := &mockDataSource{
		incident: models.Incident{
			ID:         "inc-001",
			Category:   "transport_failure",
			Severity:   "high",
			Title:      "MQTT transport degraded",
			Summary:    "MQTT connection unstable for 15 minutes",
			State:      "open",
			OccurredAt: now.Add(-1 * time.Hour).Format(time.RFC3339),
			Metadata:   map[string]any{"source": "mesh_intel"},
		},
		incidentFound: true,
		signatureKey:  "sig-abc123",
		actions: []ActionEvidence{
			{
				ID:             "act-001",
				ActionType:     "restart_transport",
				TransportName:  "mqtt-primary",
				LifecycleState: "completed",
				Result:         "executed_successfully",
				CreatedAt:      now.Add(-50 * time.Minute).Format(time.RFC3339),
				ProposedBy:     "system",
				ApprovedBy:     "operator-1",
				IncidentID:     "inc-001",
			},
		},
		actionOutcomes: []ActionOutcomeSnapshot{
			{
				SnapshotID:            "aos-1",
				SignatureKey:          "sig-abc123",
				IncidentID:            "inc-older",
				ActionID:              "act-old-1",
				ActionType:            "restart_transport",
				DerivedClassification: "improvement_observed",
				EvidenceSufficiency:   "sufficient",
				WindowStart:           now.Add(-3 * time.Hour).Format(time.RFC3339),
				WindowEnd:             now.Add(-2 * time.Hour).Format(time.RFC3339),
				DerivedAt:             now.Add(-2 * time.Hour).Format(time.RFC3339),
			},
		},
		timeline: []TimelineEntry{
			{
				EventTime: now.Add(-55 * time.Minute).Format(time.RFC3339),
				EventType: "incident",
				EventID:   "inc-001",
				Summary:   "MQTT transport degraded",
				Severity:  "high",
			},
			{
				EventTime: now.Add(-50 * time.Minute).Format(time.RFC3339),
				EventType: "control_action",
				EventID:   "act-001",
				Summary:   "restart_transport: mqtt-primary (completed)",
			},
		},
		transports: []TransportSnapshot{
			{
				TransportName: "mqtt-primary",
				TransportType: "mqtt",
				Score:         42,
				State:         "degraded",
				SnapshotTime:  now.Add(-55 * time.Minute).Format(time.RFC3339),
			},
		},
		deadLetters: []DeadLetterEntry{
			{
				TransportName: "mqtt-primary",
				Reason:        "parse_error",
				CreatedAt:     now.Add(-52 * time.Minute).Format(time.RFC3339),
			},
		},
		notes: []OperatorNote{
			{
				ID:        "note-001",
				ActorID:   "operator-1",
				Content:   "Investigating MQTT broker connectivity",
				CreatedAt: now.Add(-45 * time.Minute).Format(time.RFC3339),
			},
		},
		auditEntries: []AuditEntry{
			{
				ID:           "aud-001",
				Timestamp:    now.Add(-50 * time.Minute).Format(time.RFC3339),
				ActorID:      "operator-1",
				ActionClass:  "control",
				ActionDetail: "approve_action",
				ResourceType: "incident",
				ResourceID:   "inc-001",
				Result:       "success",
			},
		},
	}

	cfg := DefaultConfig()
	cfg.ActorID = "test-assembler"
	cfg.InstanceID = "mel-test-01"
	a := NewAssembler(src, cfg)

	pack, err := a.Assemble("inc-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Format version.
	if pack.FormatVersion != FormatVersion {
		t.Errorf("format_version = %q, want %q", pack.FormatVersion, FormatVersion)
	}

	// Assembly metadata.
	if pack.Assembly.IncidentID != "inc-001" {
		t.Errorf("assembly.incident_id = %q, want %q", pack.Assembly.IncidentID, "inc-001")
	}
	if pack.Assembly.AssembledBy != "test-assembler" {
		t.Errorf("assembly.assembled_by = %q, want %q", pack.Assembly.AssembledBy, "test-assembler")
	}
	if pack.Assembly.InstanceID != "mel-test-01" {
		t.Errorf("assembly.instance_id = %q, want %q", pack.Assembly.InstanceID, "mel-test-01")
	}
	if pack.Assembly.ActionCount != 1 {
		t.Errorf("assembly.action_count = %d, want 1", pack.Assembly.ActionCount)
	}
	if pack.Assembly.ActionOutcomeSnapshotCount != 1 {
		t.Errorf("assembly.action_outcome_snapshot_count = %d, want 1", pack.Assembly.ActionOutcomeSnapshotCount)
	}
	if pack.Assembly.ActionOutcomeSnapshotStatus != "complete" {
		t.Errorf("assembly.action_outcome_snapshot_status = %q, want complete", pack.Assembly.ActionOutcomeSnapshotStatus)
	}
	if pack.Assembly.ActionOutcomeSnapshotTrace.RetrievalStatus != "available" {
		t.Errorf("assembly.action_outcome_snapshot_trace.retrieval_status = %q, want available", pack.Assembly.ActionOutcomeSnapshotTrace.RetrievalStatus)
	}
	if !pack.Assembly.ActionOutcomeSnapshotTrace.SignatureKeyPresent {
		t.Errorf("assembly.action_outcome_snapshot_trace.signature_key_present = false, want true")
	}
	if pack.Assembly.TimelineCount != 2 {
		t.Errorf("assembly.timeline_count = %d, want 2", pack.Assembly.TimelineCount)
	}
	if pack.Assembly.TransportCount != 1 {
		t.Errorf("assembly.transport_count = %d, want 1", pack.Assembly.TransportCount)
	}
	if pack.Assembly.DeadLetterCount != 1 {
		t.Errorf("assembly.dead_letter_count = %d, want 1", pack.Assembly.DeadLetterCount)
	}
	if pack.Assembly.NoteCount != 1 {
		t.Errorf("assembly.note_count = %d, want 1", pack.Assembly.NoteCount)
	}
	if pack.Assembly.AuditEntryCount != 1 {
		t.Errorf("assembly.audit_entry_count = %d, want 1", pack.Assembly.AuditEntryCount)
	}
	if pack.Assembly.AssemblyDurationMs < 0 {
		t.Error("assembly duration should be non-negative")
	}

	// Incident.
	if pack.Incident.ID != "inc-001" {
		t.Errorf("incident.id = %q, want %q", pack.Incident.ID, "inc-001")
	}
	if pack.Incident.Severity != "high" {
		t.Errorf("incident.severity = %q, want %q", pack.Incident.Severity, "high")
	}

	// Actions.
	if len(pack.LinkedActions) != 1 {
		t.Fatalf("linked_actions length = %d, want 1", len(pack.LinkedActions))
	}
	if pack.LinkedActions[0].ActionType != "restart_transport" {
		t.Errorf("action.action_type = %q, want %q", pack.LinkedActions[0].ActionType, "restart_transport")
	}
	if len(pack.LinkedActions[0].HistoricalActionOutcomeSnapshotRefs) != 1 || pack.LinkedActions[0].HistoricalActionOutcomeSnapshotRefs[0] != "aos-1" {
		t.Fatalf("historical refs = %v, want [aos-1]", pack.LinkedActions[0].HistoricalActionOutcomeSnapshotRefs)
	}
	if len(pack.ActionOutcomeSnapshots) != 1 {
		t.Fatalf("action_outcome_snapshots length = %d, want 1", len(pack.ActionOutcomeSnapshots))
	}
	if len(pack.SectionStatuses) == 0 {
		t.Fatalf("expected section statuses")
	}

	// Timeline.
	if len(pack.Timeline) != 2 {
		t.Fatalf("timeline length = %d, want 2", len(pack.Timeline))
	}

	// Transport context.
	if len(pack.TransportContext) != 1 {
		t.Fatalf("transport_context length = %d, want 1", len(pack.TransportContext))
	}
	if pack.TransportContext[0].Score != 42 {
		t.Errorf("transport score = %d, want 42", pack.TransportContext[0].Score)
	}

	// Dead letters.
	if len(pack.DeadLetterEvidence) != 1 {
		t.Fatalf("dead_letter_evidence length = %d, want 1", len(pack.DeadLetterEvidence))
	}

	// Notes.
	if len(pack.OperatorNotes) != 1 {
		t.Fatalf("operator_notes length = %d, want 1", len(pack.OperatorNotes))
	}

	// Audit.
	if len(pack.AuditEntries) != 1 {
		t.Fatalf("audit_entries length = %d, want 1", len(pack.AuditEntries))
	}

	// Evidence gaps: full core sections plus explicit intelligence-section empties.
	if len(pack.EvidenceGaps) != 3 {
		t.Fatalf("evidence_gaps length = %d, want 3", len(pack.EvidenceGaps))
	}
	intelInfo := 0
	var assessment *EvidenceGap
	for i := range pack.EvidenceGaps {
		g := pack.EvidenceGaps[i]
		if g.Category == GapCategoryIntelligence && g.Severity == "info" {
			intelInfo++
		}
		if g.Category == "assessment" {
			assessment = &pack.EvidenceGaps[i]
		}
	}
	if intelInfo != 2 {
		t.Fatalf("expected 2 intelligence info gaps, got %d", intelInfo)
	}
	if assessment == nil || assessment.Severity != "info" {
		t.Fatalf("expected assessment no-gap marker, got %+v", assessment)
	}
	if pack.Assembly.RecommendationOutcomeCount != 0 || pack.Assembly.CorrelationGroupCount != 0 {
		t.Fatalf("expected zero recommendation/correlation counts in full mock, got rec=%d corr=%d", pack.Assembly.RecommendationOutcomeCount, pack.Assembly.CorrelationGroupCount)
	}
}

func TestAssemble_SparseEvidence_RecordsGaps(t *testing.T) {
	now := time.Now().UTC()
	src := &mockDataSource{
		incident: models.Incident{
			ID:         "inc-sparse",
			Category:   "unknown",
			Severity:   "low",
			Title:      "Minor anomaly",
			State:      "open",
			OccurredAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
		},
		incidentFound: true,
		actions:       []ActionEvidence{}, // no actions
		timeline:      []TimelineEntry{},  // no timeline
		transports:    []TransportSnapshot{},
		deadLetters:   []DeadLetterEntry{},
		notes:         []OperatorNote{},
		auditEntries:  []AuditEntry{},
	}

	a := NewAssembler(src, DefaultConfig())
	pack, err := a.Assemble("inc-sparse")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have info-level gaps for actions, timeline, transport, audit.
	if len(pack.EvidenceGaps) < 4 {
		t.Errorf("expected at least 4 evidence gaps for sparse evidence, got %d", len(pack.EvidenceGaps))
	}

	gapCategories := map[string]bool{}
	for _, g := range pack.EvidenceGaps {
		gapCategories[g.Category] = true
	}

	for _, expected := range []string{GapCategoryActions, GapCategoryTimeline, GapCategoryTransportHealth, GapCategoryAudit} {
		if !gapCategories[expected] {
			t.Errorf("missing expected evidence gap category: %s", expected)
		}
	}
}

func TestAssemble_PartialFailures_StillProduces(t *testing.T) {
	now := time.Now().UTC()
	src := &mockDataSource{
		incident: models.Incident{
			ID:         "inc-partial",
			Severity:   "medium",
			Title:      "Test partial",
			State:      "open",
			OccurredAt: now.Format(time.RFC3339),
		},
		incidentFound:  true,
		actionsErr:     fmt.Errorf("actions table locked"),
		timelineErr:    fmt.Errorf("timeline query timeout"),
		transportsErr:  fmt.Errorf("transport snapshots unavailable"),
		deadLettersErr: fmt.Errorf("dead letters unavailable"),
		notesErr:       fmt.Errorf("notes unavailable"),
		auditErr:       fmt.Errorf("audit log corrupt"),
	}

	a := NewAssembler(src, DefaultConfig())
	pack, err := a.Assemble("inc-partial")
	if err != nil {
		t.Fatalf("assembler should produce a proofpack even with partial failures, got: %v", err)
	}

	// Should still have the incident.
	if pack.Incident.ID != "inc-partial" {
		t.Error("incident should be present even with data source failures")
	}

	// Should have warning-level gaps for each failed data source.
	warningCount := 0
	for _, g := range pack.EvidenceGaps {
		if g.Severity == "warning" {
			warningCount++
		}
	}
	if warningCount < 5 {
		t.Errorf("expected at least 5 warning gaps for partial failures, got %d", warningCount)
	}
	if pack.Assembly.ActionOutcomeSnapshotStatus != "unavailable" {
		t.Errorf("assembly.action_outcome_snapshot_status = %q, want unavailable", pack.Assembly.ActionOutcomeSnapshotStatus)
	}
	if pack.Assembly.ProofpackCompleteness != "partial" {
		t.Errorf("proofpack completeness=%q, want partial", pack.Assembly.ProofpackCompleteness)
	}
}

func TestAssemble_LimitCapping_RecordsGap(t *testing.T) {
	now := time.Now().UTC()

	// Create exactly MaxActions actions to trigger the limit gap.
	cfg := DefaultConfig()
	cfg.MaxActions = 3

	actions := make([]ActionEvidence, 3)
	for i := range actions {
		actions[i] = ActionEvidence{
			ID:             fmt.Sprintf("act-%03d", i),
			ActionType:     "test",
			LifecycleState: "completed",
			CreatedAt:      now.Format(time.RFC3339),
		}
	}

	src := &mockDataSource{
		incident: models.Incident{
			ID:         "inc-limit",
			State:      "open",
			OccurredAt: now.Format(time.RFC3339),
		},
		incidentFound: true,
		actions:       actions,
	}

	a := NewAssembler(src, cfg)
	pack, err := a.Assemble("inc-limit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have a warning about action limit.
	foundLimitGap := false
	for _, g := range pack.EvidenceGaps {
		if g.Category == GapCategoryActions && g.Severity == "warning" {
			foundLimitGap = true
		}
	}
	if !foundLimitGap {
		t.Error("expected a warning gap for actions reaching limit")
	}
}

func TestAssemble_ActionOutcomeSnapshotRetrievalFailure_MarksPartialStatus(t *testing.T) {
	now := time.Now().UTC()
	src := &mockDataSource{
		incident: models.Incident{
			ID:         "inc-snapshot-partial",
			State:      "open",
			OccurredAt: now.Format(time.RFC3339),
		},
		incidentFound:     true,
		signatureKey:      "sig-partial",
		actionOutcomesErr: fmt.Errorf("snapshot query failed"),
	}

	a := NewAssembler(src, DefaultConfig())
	pack, err := a.Assemble("inc-snapshot-partial")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pack.Assembly.ActionOutcomeSnapshotStatus != "partial" {
		t.Fatalf("snapshot status=%q, want partial", pack.Assembly.ActionOutcomeSnapshotStatus)
	}
	if pack.Assembly.ActionOutcomeSnapshotTrace.RetrievalStatus != "error" {
		t.Fatalf("retrieval status=%q, want error", pack.Assembly.ActionOutcomeSnapshotTrace.RetrievalStatus)
	}
	if pack.Assembly.ActionOutcomeSnapshotTrace.StatusReason != "snapshot_query_failed" {
		t.Fatalf("status reason=%q, want snapshot_query_failed", pack.Assembly.ActionOutcomeSnapshotTrace.StatusReason)
	}
	found := false
	for _, sec := range pack.SectionStatuses {
		if sec.Section == "action_outcome_snapshots" {
			found = true
			if sec.Status != "partial" {
				t.Fatalf("section action_outcome_snapshots status=%q, want partial", sec.Status)
			}
			if sec.Reason != "snapshot_query_failed" {
				t.Fatalf("section action_outcome_snapshots reason=%q, want snapshot_query_failed", sec.Reason)
			}
		}
	}
	if !found {
		t.Fatal("expected action_outcome_snapshots section status")
	}
	gapFound := false
	for _, g := range pack.EvidenceGaps {
		if g.Category == GapCategoryActions && g.Severity == "warning" {
			gapFound = true
			break
		}
	}
	if !gapFound {
		t.Fatal("expected action warning gap for snapshot retrieval failure")
	}
}

func TestAssemble_ActionOutcomeSignatureLookupFailure_MarksTraceError(t *testing.T) {
	now := time.Now().UTC()
	src := &mockDataSource{
		incident: models.Incident{
			ID:         "inc-signature-error",
			State:      "open",
			OccurredAt: now.Format(time.RFC3339),
		},
		incidentFound:   true,
		signatureKeyErr: fmt.Errorf("signature table unavailable"),
	}
	a := NewAssembler(src, DefaultConfig())
	pack, err := a.Assemble("inc-signature-error")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := pack.Assembly.ActionOutcomeSnapshotTrace.RetrievalStatus; got != "error" {
		t.Fatalf("retrieval status=%q, want error", got)
	}
	if got := pack.Assembly.ActionOutcomeSnapshotTrace.StatusReason; got != "signature_lookup_failed" {
		t.Fatalf("status reason=%q, want signature_lookup_failed", got)
	}
	if got := pack.Assembly.ActionOutcomeSnapshotStatus; got != "unavailable" {
		t.Fatalf("snapshot status=%q, want unavailable", got)
	}
}

func TestAssemble_ResolvedIncident_WindowIncludesResolution(t *testing.T) {
	occurred := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	resolved := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

	src := &mockDataSource{
		incident: models.Incident{
			ID:         "inc-resolved",
			State:      "resolved",
			OccurredAt: occurred.Format(time.RFC3339),
			ResolvedAt: resolved.Format(time.RFC3339),
		},
		incidentFound: true,
	}

	a := NewAssembler(src, DefaultConfig())
	pack, err := a.Assemble("inc-resolved")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Window should start before occurrence and end after resolution.
	windowFrom, _ := time.Parse(time.RFC3339, pack.Assembly.TimeWindowFrom)
	windowTo, _ := time.Parse(time.RFC3339, pack.Assembly.TimeWindowTo)

	if !windowFrom.Before(occurred) {
		t.Errorf("window_from (%v) should be before occurred_at (%v)", windowFrom, occurred)
	}
	if !windowTo.After(resolved) {
		t.Errorf("window_to (%v) should be after resolved_at (%v)", windowTo, resolved)
	}
}

func TestComputeTimeWindow_InvalidTimestamp(t *testing.T) {
	inc := models.Incident{
		OccurredAt: "not-a-timestamp",
	}
	from, to := computeTimeWindow(inc)
	// Should not panic, should produce valid RFC3339 strings.
	_, err1 := time.Parse(time.RFC3339, from)
	_, err2 := time.Parse(time.RFC3339, to)
	if err1 != nil {
		t.Errorf("from is not valid RFC3339: %q", from)
	}
	if err2 != nil {
		t.Errorf("to is not valid RFC3339: %q", to)
	}
}

func TestIncidentToEvidence_PreservesFields(t *testing.T) {
	inc := models.Incident{
		ID:             "inc-fields",
		Category:       "transport_failure",
		Severity:       "critical",
		Title:          "Test Title",
		Summary:        "Test Summary",
		ResourceType:   "transport",
		ResourceID:     "mqtt-1",
		State:          "open",
		ActorID:        "operator-1",
		OccurredAt:     "2025-01-15T10:00:00Z",
		UpdatedAt:      "2025-01-15T11:00:00Z",
		OwnerActorID:   "operator-2",
		HandoffSummary: "Handed off with context",
		PendingActions: []string{"act-1"},
		Risks:          []string{"broker may be down"},
		Metadata:       map[string]any{"key": "value"},
	}

	ev := incidentToEvidence(inc)

	if ev.ID != inc.ID {
		t.Error("ID mismatch")
	}
	if ev.Category != inc.Category {
		t.Error("Category mismatch")
	}
	if ev.Severity != inc.Severity {
		t.Error("Severity mismatch")
	}
	if ev.HandoffSummary != inc.HandoffSummary {
		t.Error("HandoffSummary mismatch")
	}
	if len(ev.PendingActions) != 1 || ev.PendingActions[0] != "act-1" {
		t.Error("PendingActions mismatch")
	}
	if len(ev.Risks) != 1 || ev.Risks[0] != "broker may be down" {
		t.Error("Risks mismatch")
	}
}
