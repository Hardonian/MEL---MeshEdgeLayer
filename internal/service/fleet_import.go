package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/fleet"
)

// ImportRemoteEvidenceBundle validates and persists an offline remote evidence JSON bundle.
// This is not live federation: it is file/import scoped, instance-local storage, read-only with respect to remote execution.
func (a *App) ImportRemoteEvidenceBundle(raw []byte, strictOrigin bool, actor string) (map[string]any, error) {
	if a.DB == nil {
		return nil, fmt.Errorf("database not available")
	}
	localID, err := a.DB.EnsureInstanceID()
	if err != nil {
		return nil, err
	}
	summary, err := fleet.BuildTruthSummary(a.Cfg, a.DB)
	if err != nil {
		return nil, err
	}
	opts := fleet.IngestValidateOptions{StrictOriginMatch: strictOrigin}
	bundle, val, perr := fleet.ValidateRemoteEvidenceBundle(raw, summary.SiteID, summary.FleetID, opts)
	if perr != nil {
		return map[string]any{
			"status":     "error",
			"error":      perr.Error(),
			"validation": val,
		}, nil
	}

	id := "imp-" + uuid.NewString()
	now := fleet.ImportNowRFC3339()
	rejected := val.Outcome == fleet.ValidationRejected

	valJSON, err := json.Marshal(val)
	if err != nil {
		return nil, err
	}
	bundleJSON, err := json.Marshal(bundle)
	if err != nil {
		return nil, err
	}
	evJSON, err := json.Marshal(bundle.Evidence)
	if err != nil {
		return nil, err
	}

	rec := db.ImportedRemoteEvidenceRecord{
		ID:                     id,
		ImportedAt:             now,
		LocalInstanceID:        localID,
		Validation:             valJSON,
		Bundle:                 bundleJSON,
		Evidence:               evJSON,
		OriginInstanceID:       strings.TrimSpace(bundle.Evidence.OriginInstanceID),
		OriginSiteID:           strings.TrimSpace(bundle.Evidence.OriginSiteID),
		EvidenceClass:          string(bundle.Evidence.EvidenceClass),
		ObservationOriginClass: string(bundle.Evidence.OriginClass),
		Rejected:               rejected,
	}
	if err := a.DB.InsertImportedRemoteEvidence(rec); err != nil {
		return nil, err
	}

	actorID := strings.TrimSpace(actor)
	if actorID == "" {
		actorID = "system"
	}

	sum := fmt.Sprintf("remote evidence import %s (outcome=%s, origin=%s)", id, val.Outcome, rec.OriginInstanceID)
	if rejected {
		sum = fmt.Sprintf("remote evidence import rejected %s (%s)", id, strings.Join(reasonCodes(val.Reasons), ", "))
	}
	details := fleet.BuildImportTimelineDetails(localID, bundle, val, id)
	details["actor"] = actorID
	details["status"] = map[string]any{"import_id": id, "rejected": rejected}

	_ = a.DB.InsertTimelineEvent(db.TimelineEvent{
		EventID:    id,
		EventTime:  now,
		EventType:  "remote_evidence_import",
		Summary:    sum,
		Severity:   ternary(rejected, "warning", "info"),
		ActorID:    actorID,
		ResourceID: id,
		Details:    details,
	})

	out := map[string]any{
		"status":            ternary(rejected, "rejected", "imported"),
		"import_id":         id,
		"validation":        val,
		"rejected":          rejected,
		"local_instance_id": localID,
		"evidence_preview": map[string]any{
			"origin_instance_id":       rec.OriginInstanceID,
			"origin_site_id":           rec.OriginSiteID,
			"evidence_class":           rec.EvidenceClass,
			"observation_origin_class": rec.ObservationOriginClass,
		},
	}
	if rejected {
		out["note"] = "Rejected imports are audited but not treated as accepted normalized evidence."
	} else {
		out["normalized_evidence_stored"] = bundle.Evidence
	}
	return out, nil
}

func reasonCodes(codes []fleet.ValidationReasonCode) []string {
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		out = append(out, string(c))
	}
	return out
}

func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}

// ListImportedRemoteEvidence returns recent import audit rows.
func (a *App) ListImportedRemoteEvidence(limit int) ([]db.ImportedRemoteEvidenceRecord, error) {
	if a.DB == nil {
		return nil, fmt.Errorf("database not available")
	}
	return a.DB.ListImportedRemoteEvidence(limit)
}

// GetImportedRemoteEvidence returns one import record.
func (a *App) GetImportedRemoteEvidence(id string) (db.ImportedRemoteEvidenceRecord, bool, error) {
	if a.DB == nil {
		return db.ImportedRemoteEvidenceRecord{}, false, fmt.Errorf("database not available")
	}
	return a.DB.GetImportedRemoteEvidence(id)
}
