package proofpack

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/models"
)

// DataSource abstracts the database queries needed by the assembler.
// This allows testing without a real database.
type DataSource interface {
	// IncidentByID returns the incident with the given ID.
	IncidentByID(id string) (models.Incident, bool, error)
	// ControlActionsByIncidentID returns control actions linked to the incident.
	ControlActionsByIncidentID(incidentID string, limit int) ([]ActionEvidence, error)
	// SignatureKeyForIncident returns the deterministic incident signature key.
	SignatureKeyForIncident(incidentID string) (string, error)
	// ActionOutcomeSnapshotsBySignature returns per-action historical evaluation snapshots
	// for incidents sharing the same deterministic signature as the scoped incident.
	ActionOutcomeSnapshotsBySignature(signatureKey, excludeIncidentID string, limit int) ([]ActionOutcomeSnapshot, error)
	// TimelineEventsForIncident returns timeline events whose resource_id
	// matches the incident ID, or that fall within the time window.
	TimelineEventsForIncident(incidentID, from, to string, limit int) ([]TimelineEntry, error)
	// TransportHealthSnapshotsInWindow returns health snapshots within a window.
	TransportHealthSnapshotsInWindow(from, to string, limit int) ([]TransportSnapshot, error)
	// DeadLettersInWindow returns dead letters within a time window.
	DeadLettersInWindow(from, to string, limit int) ([]DeadLetterEntry, error)
	// OperatorNotesForResource returns notes attached to a resource.
	OperatorNotesForResource(refType, refID string, limit int) ([]OperatorNote, error)
	// AuditEntriesForResource returns audit log entries for a resource.
	AuditEntriesForResource(resourceType, resourceID string, limit int) ([]AuditEntry, error)
	// RecommendationOutcomesForIncident returns operator adjudication rows for assistive recommendations.
	RecommendationOutcomesForIncident(incidentID string, limit int) ([]RecommendationOutcomeEntry, error)
	// CorrelationGroupsForIncident returns persisted structural correlation groups.
	CorrelationGroupsForIncident(incidentID string) ([]CorrelationGroupEntry, error)
}

// AssemblerConfig controls assembly behavior.
type AssemblerConfig struct {
	// ActorID is the operator or system identity assembling the proofpack.
	ActorID string
	// InstanceID is the MEL instance ID.
	InstanceID string
	// MaxActions caps the number of linked actions included.
	MaxActions int
	// MaxActionOutcomeSnapshots caps per-action outcome snapshots.
	MaxActionOutcomeSnapshots int
	// MaxTimeline caps the number of timeline entries.
	MaxTimeline int
	// MaxDeadLetters caps dead letter entries.
	MaxDeadLetters int
	// MaxNotes caps operator notes.
	MaxNotes int
	// MaxAuditEntries caps audit log entries.
	MaxAuditEntries int
	// MaxTransportSnapshots caps transport health snapshots.
	MaxTransportSnapshots int
}

// DefaultConfig returns a sensible default assembler configuration.
func DefaultConfig() AssemblerConfig {
	return AssemblerConfig{
		ActorID:                   "system",
		MaxActions:                200,
		MaxActionOutcomeSnapshots: 400,
		MaxTimeline:               500,
		MaxDeadLetters:            200,
		MaxNotes:                  100,
		MaxAuditEntries:           200,
		MaxTransportSnapshots:     50,
	}
}

// Assembler builds proofpacks from a data source.
type Assembler struct {
	src DataSource
	cfg AssemblerConfig
}

// NewAssembler creates an assembler with the given data source and config.
func NewAssembler(src DataSource, cfg AssemblerConfig) *Assembler {
	if cfg.MaxActions <= 0 {
		cfg.MaxActions = 200
	}
	if cfg.MaxActionOutcomeSnapshots <= 0 {
		cfg.MaxActionOutcomeSnapshots = 400
	}
	if cfg.MaxTimeline <= 0 {
		cfg.MaxTimeline = 500
	}
	if cfg.MaxDeadLetters <= 0 {
		cfg.MaxDeadLetters = 200
	}
	if cfg.MaxNotes <= 0 {
		cfg.MaxNotes = 100
	}
	if cfg.MaxAuditEntries <= 0 {
		cfg.MaxAuditEntries = 200
	}
	if cfg.MaxTransportSnapshots <= 0 {
		cfg.MaxTransportSnapshots = 50
	}
	return &Assembler{src: src, cfg: cfg}
}

// Assemble builds a proofpack for the given incident ID.
// Returns an error if the incident does not exist or the data source fails.
func (a *Assembler) Assemble(incidentID string) (*Proofpack, error) {
	start := time.Now()
	incidentID = strings.TrimSpace(incidentID)
	if incidentID == "" {
		return nil, fmt.Errorf("incident ID is required")
	}

	// 1. Load the incident.
	inc, ok, err := a.src.IncidentByID(incidentID)
	if err != nil {
		return nil, fmt.Errorf("could not load incident: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("incident not found: %s", incidentID)
	}

	// 2. Compute the evidence time window.
	windowFrom, windowTo := computeTimeWindow(inc)

	// 3. Assemble evidence in sequence (each step records gaps).
	gaps := []EvidenceGap{}

	// Incident evidence.
	incEvidence := incidentToEvidence(inc)
	if inc.State == "" {
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryIncident,
			Severity:    "warning",
			Description: "incident state is empty; lifecycle position unknown",
		})
	}

	// Linked actions.
	actions, err := a.src.ControlActionsByIncidentID(incidentID, a.cfg.MaxActions)
	actionStatus := ProofpackSectionStatus{Section: "linked_actions", Status: "complete"}
	if err != nil {
		actionStatus.Status = "partial"
		actionStatus.Reason = "action_query_failed"
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryActions,
			Severity:    "warning",
			Description: fmt.Sprintf("could not load linked actions: %v", err),
		})
		actions = []ActionEvidence{}
	}
	if len(actions) == 0 {
		actionStatus.Status = "unavailable"
		actionStatus.Reason = "no_linked_actions"
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryActions,
			Severity:    "info",
			Description: "no control actions linked to this incident",
		})
	}
	if len(actions) >= a.cfg.MaxActions {
		actionStatus.Status = "partial"
		actionStatus.Reason = "linked_actions_limit_reached"
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryActions,
			Severity:    "warning",
			Description: fmt.Sprintf("action count reached limit (%d); additional actions may exist", a.cfg.MaxActions),
		})
	}

	snapshotTrace := ActionOutcomeSnapshotTrace{
		RetrievalStatus: "unavailable",
		StatusReason:    "no_signature_key",
		MaxSnapshots:    a.cfg.MaxActionOutcomeSnapshots,
	}
	signatureKey, err := a.src.SignatureKeyForIncident(incidentID)
	if err != nil {
		snapshotTrace.RetrievalStatus = "error"
		snapshotTrace.RetrievalError = err.Error()
		snapshotTrace.StatusReason = "signature_lookup_failed"
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryActions,
			Severity:    "warning",
			Description: fmt.Sprintf("could not load incident signature key: %v", err),
		})
		signatureKey = ""
	}
	signatureKey = strings.TrimSpace(signatureKey)
	actionOutcomeSnapshots := []ActionOutcomeSnapshot{}
	actionOutcomeSnapshotStatus := "unavailable"
	actionOutcomeStatus := ProofpackSectionStatus{Section: "action_outcome_snapshots", Status: "unavailable", Reason: "no_signature_key"}
	if signatureKey == "" {
		if snapshotTrace.RetrievalStatus == "unavailable" {
			snapshotTrace.StatusReason = "no_signature_key"
		}
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryActions,
			Severity:    "info",
			Description: "no signature key available; action outcome snapshots omitted",
		})
	} else {
		snapshotTrace.SignatureKeyPresent = true
		actionOutcomeSnapshots, err = a.src.ActionOutcomeSnapshotsBySignature(signatureKey, incidentID, a.cfg.MaxActionOutcomeSnapshots)
		if err != nil {
			snapshotTrace.RetrievalStatus = "error"
			snapshotTrace.RetrievalError = err.Error()
			snapshotTrace.StatusReason = "snapshot_query_failed"
			actionOutcomeSnapshotStatus = "partial"
			actionOutcomeStatus = ProofpackSectionStatus{Section: "action_outcome_snapshots", Status: "partial", Reason: "snapshot_query_failed"}
			gaps = append(gaps, EvidenceGap{
				Category:    GapCategoryActions,
				Severity:    "warning",
				Description: fmt.Sprintf("could not load action outcome snapshots: %v", err),
			})
			actionOutcomeSnapshots = []ActionOutcomeSnapshot{}
		} else if len(actionOutcomeSnapshots) == 0 {
			snapshotTrace.RetrievalStatus = "available"
			snapshotTrace.StatusReason = "no_matching_snapshots"
			actionOutcomeSnapshotStatus = "unavailable"
			actionOutcomeStatus = ProofpackSectionStatus{Section: "action_outcome_snapshots", Status: "unavailable", Reason: "no_historical_snapshots"}
		} else {
			snapshotTrace.RetrievalStatus = "available"
			snapshotTrace.StatusReason = "snapshots_loaded"
			actionOutcomeSnapshotStatus = "complete"
			actionOutcomeStatus = ProofpackSectionStatus{Section: "action_outcome_snapshots", Status: "complete"}
		}
		if len(actionOutcomeSnapshots) >= a.cfg.MaxActionOutcomeSnapshots {
			snapshotTrace.RetrievalLimited = true
			snapshotTrace.StatusReason = "snapshot_limit_reached"
			actionOutcomeSnapshotStatus = "partial"
			actionOutcomeStatus = ProofpackSectionStatus{Section: "action_outcome_snapshots", Status: "partial", Reason: "snapshot_limit_reached"}
			gaps = append(gaps, EvidenceGap{
				Category:    GapCategoryActions,
				Severity:    "warning",
				Description: fmt.Sprintf("action outcome snapshot count reached limit (%d); additional snapshots may exist", a.cfg.MaxActionOutcomeSnapshots),
			})
		}
	}
	actions = attachActionSnapshotRefs(actions, actionOutcomeSnapshots)

	// Timeline events.
	timeline, err := a.src.TimelineEventsForIncident(incidentID, windowFrom, windowTo, a.cfg.MaxTimeline)
	timelineStatus := ProofpackSectionStatus{Section: "timeline", Status: "complete"}
	if err != nil {
		timelineStatus.Status = "partial"
		timelineStatus.Reason = "timeline_query_failed"
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryTimeline,
			Severity:    "warning",
			Description: fmt.Sprintf("could not load timeline events: %v", err),
		})
		timeline = []TimelineEntry{}
	}
	if len(timeline) == 0 {
		timelineStatus.Status = "unavailable"
		timelineStatus.Reason = "no_timeline_events"
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryTimeline,
			Severity:    "info",
			Description: "no timeline events found in incident window",
		})
	}
	if len(timeline) >= a.cfg.MaxTimeline {
		timelineStatus.Status = "partial"
		timelineStatus.Reason = "timeline_limit_reached"
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryTimeline,
			Severity:    "warning",
			Description: fmt.Sprintf("timeline count reached limit (%d); additional events may exist", a.cfg.MaxTimeline),
		})
	}

	// Transport health context.
	transports, err := a.src.TransportHealthSnapshotsInWindow(windowFrom, windowTo, a.cfg.MaxTransportSnapshots)
	transportStatus := ProofpackSectionStatus{Section: "transport_context", Status: "complete"}
	if err != nil {
		transportStatus.Status = "partial"
		transportStatus.Reason = "transport_snapshot_query_failed"
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryTransportHealth,
			Severity:    "warning",
			Description: fmt.Sprintf("could not load transport health snapshots: %v", err),
		})
		transports = []TransportSnapshot{}
	}
	if len(transports) == 0 {
		transportStatus.Status = "unavailable"
		transportStatus.Reason = "no_transport_snapshots_in_window"
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryTransportHealth,
			Severity:    "info",
			Description: "no transport health snapshots available in incident window",
		})
	}

	// Dead letters.
	deadLetters, err := a.src.DeadLettersInWindow(windowFrom, windowTo, a.cfg.MaxDeadLetters)
	deadLetterStatus := ProofpackSectionStatus{Section: "dead_letter_evidence", Status: "complete"}
	if err != nil {
		deadLetterStatus.Status = "partial"
		deadLetterStatus.Reason = "dead_letter_query_failed"
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryDeadLetters,
			Severity:    "warning",
			Description: fmt.Sprintf("could not load dead letters: %v", err),
		})
		deadLetters = []DeadLetterEntry{}
	}
	if len(deadLetters) == 0 && deadLetterStatus.Status == "complete" {
		deadLetterStatus.Status = "unavailable"
		deadLetterStatus.Reason = "no_dead_letters_in_window"
	}

	// Operator notes.
	notes, err := a.src.OperatorNotesForResource("incident", incidentID, a.cfg.MaxNotes)
	notesStatus := ProofpackSectionStatus{Section: "operator_notes", Status: "complete"}
	if err != nil {
		notesStatus.Status = "partial"
		notesStatus.Reason = "operator_notes_query_failed"
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryIncident,
			Severity:    "info",
			Description: fmt.Sprintf("could not load operator notes: %v", err),
		})
		notes = []OperatorNote{}
	}
	if len(notes) == 0 && notesStatus.Status == "complete" {
		notesStatus.Status = "unavailable"
		notesStatus.Reason = "no_operator_notes"
	}

	// Audit entries.
	auditEntries, err := a.src.AuditEntriesForResource("incident", incidentID, a.cfg.MaxAuditEntries)
	auditStatus := ProofpackSectionStatus{Section: "audit_entries", Status: "complete"}
	if err != nil {
		auditStatus.Status = "partial"
		auditStatus.Reason = "audit_query_failed"
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryAudit,
			Severity:    "warning",
			Description: fmt.Sprintf("could not load audit entries: %v", err),
		})
		auditEntries = []AuditEntry{}
	}
	if len(auditEntries) == 0 {
		if auditStatus.Status == "complete" {
			auditStatus.Status = "unavailable"
			auditStatus.Reason = "no_audit_entries"
		}
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryAudit,
			Severity:    "info",
			Description: "no audit log entries found for this incident",
		})
	}

	// Recommendation outcomes (operator adjudication).
	recOutcomes, err := a.src.RecommendationOutcomesForIncident(incidentID, a.cfg.MaxNotes)
	recStatus := ProofpackSectionStatus{Section: "recommendation_outcomes", Status: "complete"}
	if err != nil {
		recStatus.Status = "partial"
		recStatus.Reason = "recommendation_outcomes_query_failed"
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryIntelligence,
			Severity:    "info",
			Description: fmt.Sprintf("could not load recommendation outcomes: %v", err),
		})
		recOutcomes = []RecommendationOutcomeEntry{}
	}
	if len(recOutcomes) == 0 && recStatus.Status == "complete" {
		recStatus.Status = "unavailable"
		recStatus.Reason = "no_recommendation_outcomes"
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryIntelligence,
			Severity:    "info",
			Description: "no operator recommendation adjudication rows recorded for this incident",
		})
	}

	// Cross-incident correlation groups.
	corrGroups, err := a.src.CorrelationGroupsForIncident(incidentID)
	corrStatus := ProofpackSectionStatus{Section: "correlation_groups", Status: "complete"}
	if err != nil {
		corrStatus.Status = "partial"
		corrStatus.Reason = "correlation_query_failed"
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryIntelligence,
			Severity:    "warning",
			Description: fmt.Sprintf("could not load correlation groups: %v", err),
		})
		corrGroups = []CorrelationGroupEntry{}
	}
	if len(corrGroups) == 0 && corrStatus.Status == "complete" {
		corrStatus.Status = "unavailable"
		corrStatus.Reason = "no_correlation_groups"
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryIntelligence,
			Severity:    "info",
			Description: "no persisted cross-incident correlation groups for this incident",
		})
	}

	// Always end with an assessment marker when none exists yet so consumers can
	// distinguish "assembly evaluated" from "gaps omitted"; section_statuses carry per-section posture.
	hasAssessment := false
	for _, g := range gaps {
		if g.Category == "assessment" {
			hasAssessment = true
			break
		}
	}
	if !hasAssessment {
		desc := "no evidence gaps detected during assembly"
		if len(gaps) > 0 {
			desc = "assembly complete with explicit section-level markers; review evidence_gaps and section_statuses together"
		}
		gaps = append(gaps, EvidenceGap{
			Category:    "assessment",
			Severity:    "info",
			Description: desc,
		})
	}

	elapsed := time.Since(start)
	sectionStatuses := []ProofpackSectionStatus{
		actionStatus,
		actionOutcomeStatus,
		timelineStatus,
		transportStatus,
		deadLetterStatus,
		notesStatus,
		auditStatus,
		recStatus,
		corrStatus,
	}
	completeness, reasons := deriveProofpackCompleteness(sectionStatuses)

	pack := &Proofpack{
		FormatVersion: FormatVersion,
		Assembly: AssemblyMetadata{
			AssembledAt:                  time.Now().UTC().Format(time.RFC3339),
			AssembledBy:                  a.cfg.ActorID,
			InstanceID:                   a.cfg.InstanceID,
			IncidentID:                   incidentID,
			ManifestVersion:              ManifestVersion,
			IntegrityNote:                "Counts and section_statuses are deterministic assembly metadata; export is a snapshot, not live truth or cryptographic attestation.",
			TimeWindowFrom:               windowFrom,
			TimeWindowTo:                 windowTo,
			ActionCount:                  len(actions),
			ActionOutcomeSnapshotCount:   len(actionOutcomeSnapshots),
			ActionOutcomeSnapshotStatus:  actionOutcomeSnapshotStatus,
			ActionOutcomeSnapshotTrace:   snapshotTrace,
			TimelineCount:                len(timeline),
			TransportCount:               len(transports),
			DeadLetterCount:              len(deadLetters),
			NoteCount:                    len(notes),
			AuditEntryCount:              len(auditEntries),
			RecommendationOutcomeCount:   len(recOutcomes),
			CorrelationGroupCount:        len(corrGroups),
			EvidenceGapCount:             len(gaps),
			ProofpackCompleteness:        completeness,
			ProofpackCompletenessReasons: reasons,
			AssemblyDurationMs:           elapsed.Milliseconds(),
		},
		Incident:               incEvidence,
		LinkedActions:          actions,
		ActionOutcomeSnapshots: actionOutcomeSnapshots,
		Timeline:               timeline,
		TransportContext:       transports,
		DeadLetterEvidence:     deadLetters,
		OperatorNotes:          notes,
		AuditEntries:           auditEntries,
		RecommendationOutcomes: recOutcomes,
		CorrelationGroups:      corrGroups,
		EvidenceGaps:           gaps,
		SectionStatuses:        sectionStatuses,
	}

	return pack, nil
}

func attachActionSnapshotRefs(actions []ActionEvidence, snapshots []ActionOutcomeSnapshot) []ActionEvidence {
	if len(actions) == 0 {
		return actions
	}
	refsByType := map[string]map[string]struct{}{}
	for _, snap := range snapshots {
		actionType := strings.TrimSpace(snap.ActionType)
		if actionType == "" || strings.TrimSpace(snap.SnapshotID) == "" {
			continue
		}
		if refsByType[actionType] == nil {
			refsByType[actionType] = map[string]struct{}{}
		}
		refsByType[actionType][snap.SnapshotID] = struct{}{}
	}
	for i := range actions {
		typed := refsByType[strings.TrimSpace(actions[i].ActionType)]
		if len(typed) == 0 {
			continue
		}
		refs := make([]string, 0, len(typed))
		for id := range typed {
			refs = append(refs, id)
		}
		sort.Strings(refs)
		actions[i].HistoricalActionOutcomeSnapshotRefs = refs
	}
	return actions
}

func deriveProofpackCompleteness(sections []ProofpackSectionStatus) (string, []string) {
	if len(sections) == 0 {
		return "unavailable", []string{"no_section_statuses"}
	}
	out := "complete"
	reasons := []string{}
	for _, sec := range sections {
		switch sec.Status {
		case "partial":
			if out != "unavailable" {
				out = "partial"
			}
			if sec.Reason != "" {
				reasons = append(reasons, sec.Section+":"+sec.Reason)
			}
		case "unavailable":
			out = "partial"
			if sec.Reason != "" {
				reasons = append(reasons, sec.Section+":"+sec.Reason)
			}
		}
	}
	if len(reasons) == 0 && out == "complete" {
		reasons = append(reasons, "all_sections_available")
	}
	return out, reasons
}

// computeTimeWindow determines the evidence gathering window based on
// the incident's occurred_at and resolved_at (or now if unresolved).
func computeTimeWindow(inc models.Incident) (string, string) {
	occurred, err := time.Parse(time.RFC3339, inc.OccurredAt)
	if err != nil {
		occurred = time.Now().UTC().Add(-24 * time.Hour)
	}

	var end time.Time
	if inc.ResolvedAt != "" {
		resolved, err := time.Parse(time.RFC3339, inc.ResolvedAt)
		if err == nil {
			end = resolved
		}
	}
	if end.IsZero() {
		end = time.Now().UTC()
	}

	from := occurred.Add(-TimeWindowPadding)
	to := end.Add(TimeWindowPadding)

	return from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339)
}

// incidentToEvidence maps a models.Incident to the proofpack evidence type.
func incidentToEvidence(inc models.Incident) IncidentEvidence {
	return IncidentEvidence{
		ID:                     inc.ID,
		Category:               inc.Category,
		Severity:               inc.Severity,
		Title:                  inc.Title,
		Summary:                inc.Summary,
		ResourceType:           inc.ResourceType,
		ResourceID:             inc.ResourceID,
		State:                  inc.State,
		ActorID:                inc.ActorID,
		OccurredAt:             inc.OccurredAt,
		UpdatedAt:              inc.UpdatedAt,
		ResolvedAt:             inc.ResolvedAt,
		OwnerActorID:           inc.OwnerActorID,
		HandoffSummary:         inc.HandoffSummary,
		PendingActions:         inc.PendingActions,
		RecentActions:          inc.RecentActions,
		LinkedEvidence:         inc.LinkedEvidence,
		Risks:                  inc.Risks,
		Metadata:               inc.Metadata,
		ReviewState:            inc.ReviewState,
		InvestigationNotes:     inc.InvestigationNotes,
		ResolutionSummary:      inc.ResolutionSummary,
		CloseoutReason:         inc.CloseoutReason,
		LessonsLearned:         inc.LessonsLearned,
		ReopenedFromIncidentID: inc.ReopenedFromIncidentID,
		ReopenedAt:             inc.ReopenedAt,
	}
}
