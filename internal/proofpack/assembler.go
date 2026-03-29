package proofpack

import (
	"fmt"
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
	if err != nil {
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryActions,
			Severity:    "warning",
			Description: fmt.Sprintf("could not load linked actions: %v", err),
		})
		actions = []ActionEvidence{}
	}
	if len(actions) == 0 {
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryActions,
			Severity:    "info",
			Description: "no control actions linked to this incident",
		})
	}
	if len(actions) >= a.cfg.MaxActions {
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryActions,
			Severity:    "warning",
			Description: fmt.Sprintf("action count reached limit (%d); additional actions may exist", a.cfg.MaxActions),
		})
	}

	signatureKey, err := a.src.SignatureKeyForIncident(incidentID)
	if err != nil {
		signatureKey = ""
	}
	signatureKey = strings.TrimSpace(signatureKey)
	actionOutcomeSnapshots := []ActionOutcomeSnapshot{}
	actionOutcomeSnapshotStatus := "unavailable"
	if signatureKey == "" {
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryActions,
			Severity:    "info",
			Description: "no signature key available; action outcome snapshots omitted",
		})
	} else {
		actionOutcomeSnapshots, err = a.src.ActionOutcomeSnapshotsBySignature(signatureKey, incidentID, a.cfg.MaxActionOutcomeSnapshots)
		if err != nil {
			actionOutcomeSnapshotStatus = "partial"
			gaps = append(gaps, EvidenceGap{
				Category:    GapCategoryActions,
				Severity:    "warning",
				Description: fmt.Sprintf("could not load action outcome snapshots: %v", err),
			})
			actionOutcomeSnapshots = []ActionOutcomeSnapshot{}
		} else if len(actionOutcomeSnapshots) == 0 {
			actionOutcomeSnapshotStatus = "unavailable"
		} else {
			actionOutcomeSnapshotStatus = "complete"
		}
		if len(actionOutcomeSnapshots) >= a.cfg.MaxActionOutcomeSnapshots {
			actionOutcomeSnapshotStatus = "partial"
			gaps = append(gaps, EvidenceGap{
				Category:    GapCategoryActions,
				Severity:    "warning",
				Description: fmt.Sprintf("action outcome snapshot count reached limit (%d); additional snapshots may exist", a.cfg.MaxActionOutcomeSnapshots),
			})
		}
	}

	// Timeline events.
	timeline, err := a.src.TimelineEventsForIncident(incidentID, windowFrom, windowTo, a.cfg.MaxTimeline)
	if err != nil {
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryTimeline,
			Severity:    "warning",
			Description: fmt.Sprintf("could not load timeline events: %v", err),
		})
		timeline = []TimelineEntry{}
	}
	if len(timeline) == 0 {
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryTimeline,
			Severity:    "info",
			Description: "no timeline events found in incident window",
		})
	}
	if len(timeline) >= a.cfg.MaxTimeline {
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryTimeline,
			Severity:    "warning",
			Description: fmt.Sprintf("timeline count reached limit (%d); additional events may exist", a.cfg.MaxTimeline),
		})
	}

	// Transport health context.
	transports, err := a.src.TransportHealthSnapshotsInWindow(windowFrom, windowTo, a.cfg.MaxTransportSnapshots)
	if err != nil {
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryTransportHealth,
			Severity:    "warning",
			Description: fmt.Sprintf("could not load transport health snapshots: %v", err),
		})
		transports = []TransportSnapshot{}
	}
	if len(transports) == 0 {
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryTransportHealth,
			Severity:    "info",
			Description: "no transport health snapshots available in incident window",
		})
	}

	// Dead letters.
	deadLetters, err := a.src.DeadLettersInWindow(windowFrom, windowTo, a.cfg.MaxDeadLetters)
	if err != nil {
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryDeadLetters,
			Severity:    "warning",
			Description: fmt.Sprintf("could not load dead letters: %v", err),
		})
		deadLetters = []DeadLetterEntry{}
	}

	// Operator notes.
	notes, err := a.src.OperatorNotesForResource("incident", incidentID, a.cfg.MaxNotes)
	if err != nil {
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryIncident,
			Severity:    "info",
			Description: fmt.Sprintf("could not load operator notes: %v", err),
		})
		notes = []OperatorNote{}
	}

	// Audit entries.
	auditEntries, err := a.src.AuditEntriesForResource("incident", incidentID, a.cfg.MaxAuditEntries)
	if err != nil {
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryAudit,
			Severity:    "warning",
			Description: fmt.Sprintf("could not load audit entries: %v", err),
		})
		auditEntries = []AuditEntry{}
	}
	if len(auditEntries) == 0 {
		gaps = append(gaps, EvidenceGap{
			Category:    GapCategoryAudit,
			Severity:    "info",
			Description: "no audit log entries found for this incident",
		})
	}

	// If no gaps were found at all, mark explicitly.
	if len(gaps) == 0 {
		gaps = append(gaps, EvidenceGap{
			Category:    "assessment",
			Severity:    "info",
			Description: "no evidence gaps detected during assembly",
		})
	}

	elapsed := time.Since(start)

	pack := &Proofpack{
		FormatVersion: FormatVersion,
		Assembly: AssemblyMetadata{
			AssembledAt:                 time.Now().UTC().Format(time.RFC3339),
			AssembledBy:                 a.cfg.ActorID,
			InstanceID:                  a.cfg.InstanceID,
			IncidentID:                  incidentID,
			TimeWindowFrom:              windowFrom,
			TimeWindowTo:                windowTo,
			ActionCount:                 len(actions),
			ActionOutcomeSnapshotCount:  len(actionOutcomeSnapshots),
			ActionOutcomeSnapshotStatus: actionOutcomeSnapshotStatus,
			TimelineCount:               len(timeline),
			TransportCount:              len(transports),
			DeadLetterCount:             len(deadLetters),
			NoteCount:                   len(notes),
			AuditEntryCount:             len(auditEntries),
			EvidenceGapCount:            len(gaps),
			AssemblyDurationMs:          elapsed.Milliseconds(),
		},
		Incident:               incEvidence,
		LinkedActions:          actions,
		ActionOutcomeSnapshots: actionOutcomeSnapshots,
		Timeline:               timeline,
		TransportContext:       transports,
		DeadLetterEvidence:     deadLetters,
		OperatorNotes:          notes,
		AuditEntries:           auditEntries,
		EvidenceGaps:           gaps,
	}

	return pack, nil
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
		ID:             inc.ID,
		Category:       inc.Category,
		Severity:       inc.Severity,
		Title:          inc.Title,
		Summary:        inc.Summary,
		ResourceType:   inc.ResourceType,
		ResourceID:     inc.ResourceID,
		State:          inc.State,
		ActorID:        inc.ActorID,
		OccurredAt:     inc.OccurredAt,
		UpdatedAt:      inc.UpdatedAt,
		ResolvedAt:     inc.ResolvedAt,
		OwnerActorID:   inc.OwnerActorID,
		HandoffSummary: inc.HandoffSummary,
		PendingActions: inc.PendingActions,
		RecentActions:  inc.RecentActions,
		LinkedEvidence: inc.LinkedEvidence,
		Risks:          inc.Risks,
		Metadata:       inc.Metadata,
	}
}
