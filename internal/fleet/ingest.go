package fleet

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Remote evidence ingest is offline file/bundle only: no live mesh sync, no cryptographic authenticity in core.

const (
	// RemoteEvidenceBundleSchemaVersion is the only schema revision accepted by this build for strict validation.
	RemoteEvidenceBundleSchemaVersion = "1.0"
	// RemoteEvidenceBundleKind identifies the canonical JSON wrapper for one evidence envelope import.
	RemoteEvidenceBundleKind = "mel_remote_evidence_bundle"
)

// TimingOrderPosture describes how ordering/timing should be interpreted for imported or merged views.
type TimingOrderPosture string

const (
	TimingOrderLocalOrdered                TimingOrderPosture = "local_ordered"
	TimingOrderImportedPreserved           TimingOrderPosture = "imported_preserved_order"
	TimingOrderMergedBestEffort            TimingOrderPosture = "merged_best_effort_order"
	TimingOrderUncertainClockSkew          TimingOrderPosture = "ordering_uncertain_due_to_clock_skew"
	TimingOrderReceiveDiffersFromObserved  TimingOrderPosture = "receive_time_differs_from_observed_time"
	TimingOrderImportTimeNotEqualEventTime TimingOrderPosture = "import_time_not_equal_event_time"
)

// ValidationOutcome is the structural validation result for a remote bundle (validation ≠ ground truth).
type ValidationOutcome string

const (
	ValidationAccepted            ValidationOutcome = "accepted"
	ValidationAcceptedWithCaveats ValidationOutcome = "accepted_with_caveats"
	ValidationRejected            ValidationOutcome = "rejected"
)

// ValidationReasonCode is a machine-readable reason for acceptance caveats or rejection.
type ValidationReasonCode string

const (
	ReasonOK                           ValidationReasonCode = "ok"
	ReasonMalformedJSON                ValidationReasonCode = "malformed_json"
	ReasonUnsupportedSchema            ValidationReasonCode = "unsupported_schema_version"
	ReasonUnsupportedBundleKind        ValidationReasonCode = "unsupported_bundle_kind"
	ReasonMissingEvidenceClass         ValidationReasonCode = "missing_evidence_class"
	ReasonMissingOriginInstance        ValidationReasonCode = "missing_origin_instance_id"
	ReasonInvalidOriginClass           ValidationReasonCode = "invalid_observation_origin_class"
	ReasonConflictingOriginSite        ValidationReasonCode = "conflicting_origin_site"
	ReasonConflictingClaimedFleet      ValidationReasonCode = "conflicting_claimed_fleet"
	ReasonOriginSiteAbsent             ValidationReasonCode = "origin_site_id_absent"
	ReasonClaimedOriginMismatch        ValidationReasonCode = "claimed_origin_instance_mismatch"
	ReasonMissingEventID               ValidationReasonCode = "missing_event_id"
	ReasonMissingEventType             ValidationReasonCode = "missing_event_type"
	ReasonMissingEventSummary          ValidationReasonCode = "missing_event_summary"
	ReasonEventOriginMismatch          ValidationReasonCode = "event_origin_instance_mismatch"
	ReasonEventCorrelationMismatch     ValidationReasonCode = "event_correlation_id_mismatch"
	CaveatNotCryptographicallyVerified ValidationReasonCode = "authenticity_not_cryptographically_verified"
	CaveatPartialObservationOnly       ValidationReasonCode = "partial_observation_only"
	CaveatReceiveDiffersFromObserved   ValidationReasonCode = "receive_time_differs_from_observed_time"
)

// RemoteEvidenceBundle is the canonical JSON wrapper for importing one EvidenceEnvelope (offline).
type RemoteEvidenceBundle struct {
	SchemaVersion string `json:"schema_version"`
	Kind          string `json:"kind"`
	// ClaimedOriginInstanceID is optional declared sender; must match Evidence.OriginInstanceID when both set (strict mode).
	ClaimedOriginInstanceID string `json:"claimed_origin_instance_id,omitempty"`
	ClaimedOriginSiteID     string `json:"claimed_origin_site_id,omitempty"`
	ClaimedFleetID          string `json:"claimed_fleet_id,omitempty"`
	ImportContext           struct {
		SourceLabel  string `json:"source_label,omitempty"`
		ImportReason string `json:"import_reason,omitempty"`
	} `json:"import_context,omitempty"`
	Evidence EvidenceEnvelope `json:"evidence"`
	Event    *EventEnvelope   `json:"event,omitempty"`
}

// RemoteEvidenceValidation is the typed validation/trust result for an import attempt.
type RemoteEvidenceValidation struct {
	Outcome          ValidationOutcome      `json:"outcome"`
	Reasons          []ValidationReasonCode `json:"reasons"`
	TrustPosture     string                 `json:"trust_posture"`
	AuthenticityNote string                 `json:"authenticity_note"`
	OrderingPosture  TimingOrderPosture     `json:"ordering_posture"`
	Summary          string                 `json:"summary"`
}

const (
	// TrustPostureImportedReadOnly is structural acceptance: data is imported for review, not verified as live or authoritative.
	TrustPostureImportedReadOnly = "imported_read_only_not_live_authority"
	// TrustPostureRejected remains explicit when nothing is persisted as accepted evidence.
	TrustPostureRejected = "rejected_no_persisted_acceptance"
)

// IngestValidateOptions toggles strict scope checks for an import.
type IngestValidateOptions struct {
	// StrictOriginMatch requires ClaimedOriginInstanceID (if set) to equal Evidence.OriginInstanceID.
	StrictOriginMatch bool
}

// ValidateRemoteEvidenceBundle validates JSON bytes and returns bundle + validation. err is set only on malformed JSON.
func ValidateRemoteEvidenceBundle(raw []byte, localSiteID, localFleetID string, opts IngestValidateOptions) (RemoteEvidenceBundle, RemoteEvidenceValidation, error) {
	rejectBase := RemoteEvidenceValidation{
		Outcome:          ValidationRejected,
		TrustPosture:     TrustPostureRejected,
		AuthenticityNote: "Import authenticity is not cryptographically verified in core; treat as claimed-origin text.",
		OrderingPosture:  TimingOrderImportedPreserved,
	}
	raw = trimUTF8BOM(raw)
	var b RemoteEvidenceBundle
	if err := json.Unmarshal(raw, &b); err != nil {
		rejectBase.Reasons = []ValidationReasonCode{ReasonMalformedJSON}
		rejectBase.Summary = "Malformed JSON."
		return b, rejectBase, fmt.Errorf("parse remote evidence bundle: %w", err)
	}

	v := RemoteEvidenceValidation{
		AuthenticityNote: "Import authenticity is not cryptographically verified in core; treat as claimed-origin text.",
		OrderingPosture:  TimingOrderImportedPreserved,
	}

	if strings.TrimSpace(b.SchemaVersion) != RemoteEvidenceBundleSchemaVersion {
		v.Outcome = ValidationRejected
		v.TrustPosture = TrustPostureRejected
		v.Reasons = []ValidationReasonCode{ReasonUnsupportedSchema}
		v.Summary = fmt.Sprintf("Unsupported schema_version %q (want %s).", b.SchemaVersion, RemoteEvidenceBundleSchemaVersion)
		return b, v, nil
	}
	if strings.TrimSpace(b.Kind) != RemoteEvidenceBundleKind {
		v.Outcome = ValidationRejected
		v.TrustPosture = TrustPostureRejected
		v.Reasons = []ValidationReasonCode{ReasonUnsupportedBundleKind}
		v.Summary = fmt.Sprintf("Unsupported kind %q (want %s).", b.Kind, RemoteEvidenceBundleKind)
		return b, v, nil
	}
	if strings.TrimSpace(string(b.Evidence.EvidenceClass)) == "" {
		v.Outcome = ValidationRejected
		v.TrustPosture = TrustPostureRejected
		v.Reasons = []ValidationReasonCode{ReasonMissingEvidenceClass}
		v.Summary = "Missing evidence.evidence_class."
		return b, v, nil
	}
	if strings.TrimSpace(b.Evidence.OriginInstanceID) == "" {
		v.Outcome = ValidationRejected
		v.TrustPosture = TrustPostureRejected
		v.Reasons = []ValidationReasonCode{ReasonMissingOriginInstance}
		v.Summary = "Missing evidence.origin_instance_id."
		return b, v, nil
	}
	if !isKnownOriginClass(b.Evidence.OriginClass) {
		v.Outcome = ValidationRejected
		v.TrustPosture = TrustPostureRejected
		v.Reasons = []ValidationReasonCode{ReasonInvalidOriginClass}
		v.Summary = fmt.Sprintf("Invalid observation_origin_class %q.", b.Evidence.OriginClass)
		return b, v, nil
	}
	if b.Event != nil {
		if strings.TrimSpace(b.Event.EventID) == "" {
			v.Outcome = ValidationRejected
			v.TrustPosture = TrustPostureRejected
			v.Reasons = []ValidationReasonCode{ReasonMissingEventID}
			v.Summary = "Missing event.event_id."
			return b, v, nil
		}
		if strings.TrimSpace(b.Event.EventType) == "" {
			v.Outcome = ValidationRejected
			v.TrustPosture = TrustPostureRejected
			v.Reasons = []ValidationReasonCode{ReasonMissingEventType}
			v.Summary = "Missing event.event_type."
			return b, v, nil
		}
		if strings.TrimSpace(b.Event.Summary) == "" {
			v.Outcome = ValidationRejected
			v.TrustPosture = TrustPostureRejected
			v.Reasons = []ValidationReasonCode{ReasonMissingEventSummary}
			v.Summary = "Missing event.summary."
			return b, v, nil
		}
		if strings.TrimSpace(b.Event.OriginInstanceID) == "" {
			v.Outcome = ValidationRejected
			v.TrustPosture = TrustPostureRejected
			v.Reasons = []ValidationReasonCode{ReasonMissingOriginInstance}
			v.Summary = "Missing event.origin_instance_id."
			return b, v, nil
		}
		if strings.TrimSpace(b.Event.OriginInstanceID) != strings.TrimSpace(b.Evidence.OriginInstanceID) {
			v.Outcome = ValidationRejected
			v.TrustPosture = TrustPostureRejected
			v.Reasons = []ValidationReasonCode{ReasonEventOriginMismatch}
			v.Summary = "event.origin_instance_id must match evidence.origin_instance_id."
			return b, v, nil
		}
		if strings.TrimSpace(b.Event.OriginSiteID) != "" && strings.TrimSpace(b.Evidence.OriginSiteID) != "" &&
			strings.TrimSpace(b.Event.OriginSiteID) != strings.TrimSpace(b.Evidence.OriginSiteID) {
			v.Outcome = ValidationRejected
			v.TrustPosture = TrustPostureRejected
			v.Reasons = []ValidationReasonCode{ReasonConflictingOriginSite}
			v.Summary = "event.origin_site_id must match evidence.origin_site_id when both are present."
			return b, v, nil
		}
		if strings.TrimSpace(b.Event.CorrelationID) != "" && strings.TrimSpace(b.Evidence.CorrelationID) != "" &&
			strings.TrimSpace(b.Event.CorrelationID) != strings.TrimSpace(b.Evidence.CorrelationID) {
			v.Outcome = ValidationRejected
			v.TrustPosture = TrustPostureRejected
			v.Reasons = []ValidationReasonCode{ReasonEventCorrelationMismatch}
			v.Summary = "event.correlation_id must match evidence.correlation_id when both are present."
			return b, v, nil
		}
	}

	claimedInst := strings.TrimSpace(b.ClaimedOriginInstanceID)
	evOrigin := strings.TrimSpace(b.Evidence.OriginInstanceID)
	if claimedInst != "" && claimedInst != evOrigin {
		if opts.StrictOriginMatch {
			v.Outcome = ValidationRejected
			v.TrustPosture = TrustPostureRejected
			v.Reasons = []ValidationReasonCode{ReasonClaimedOriginMismatch}
			v.Summary = "claimed_origin_instance_id does not match evidence.origin_instance_id."
			return b, v, nil
		}
	}

	evSite := strings.TrimSpace(b.Evidence.OriginSiteID)
	localSite := strings.TrimSpace(localSiteID)
	if localSite != "" && evSite != "" && evSite != localSite {
		v.Outcome = ValidationRejected
		v.TrustPosture = TrustPostureRejected
		v.Reasons = []ValidationReasonCode{ReasonConflictingOriginSite}
		v.Summary = "Evidence origin_site_id conflicts with this instance's configured site scope."
		return b, v, nil
	}
	claimSite := strings.TrimSpace(b.ClaimedOriginSiteID)
	if localSite != "" && claimSite != "" && claimSite != localSite {
		v.Outcome = ValidationRejected
		v.TrustPosture = TrustPostureRejected
		v.Reasons = []ValidationReasonCode{ReasonConflictingOriginSite}
		v.Summary = "claimed_origin_site_id conflicts with this instance's configured site scope."
		return b, v, nil
	}

	lf := strings.TrimSpace(localFleetID)
	cf := strings.TrimSpace(b.ClaimedFleetID)
	if lf != "" && cf != "" && cf != lf {
		v.Outcome = ValidationRejected
		v.TrustPosture = TrustPostureRejected
		v.Reasons = []ValidationReasonCode{ReasonConflictingClaimedFleet}
		v.Summary = "claimed_fleet_id conflicts with this instance's configured fleet scope."
		return b, v, nil
	}

	outcome := ValidationAccepted
	v.Reasons = []ValidationReasonCode{CaveatNotCryptographicallyVerified, CaveatPartialObservationOnly}
	if claimedInst != "" && claimedInst != evOrigin && !opts.StrictOriginMatch {
		v.Reasons = append(v.Reasons, ReasonClaimedOriginMismatch)
	}
	if evSite == "" && localSite != "" {
		v.Reasons = append(v.Reasons, ReasonOriginSiteAbsent)
		outcome = ValidationAcceptedWithCaveats
	}
	if strings.TrimSpace(b.Evidence.ReceivedAt) != "" && strings.TrimSpace(b.Evidence.ObservedAt) != "" &&
		strings.TrimSpace(b.Evidence.ReceivedAt) != strings.TrimSpace(b.Evidence.ObservedAt) {
		v.Reasons = append(v.Reasons, CaveatReceiveDiffersFromObserved)
		v.OrderingPosture = TimingOrderReceiveDiffersFromObserved
	} else {
		v.OrderingPosture = TimingOrderImportTimeNotEqualEventTime
	}

	v.Outcome = outcome
	v.TrustPosture = TrustPostureImportedReadOnly
	if outcome == ValidationAcceptedWithCaveats {
		v.Summary = "Accepted with caveats: imported evidence is observer-bounded; not live fleet truth; authenticity not verified."
	} else {
		v.Summary = "Accepted: structurally valid remote evidence envelope for offline import (not cryptographic proof of origin)."
	}
	return b, v, nil
}

func trimUTF8BOM(b []byte) []byte {
	if len(b) >= 3 && b[0] == 0xef && b[1] == 0xbb && b[2] == 0xbf {
		return b[3:]
	}
	return b
}

func isKnownOriginClass(o ObservationOriginClass) bool {
	switch o {
	case OriginDirectIngest, OriginForwardedTransport, OriginRemoteReported,
		OriginAggregated, OriginInferred, OriginUnknown:
		return true
	default:
		return false
	}
}

// MergeInspection surfaces merge/dedupe posture for operators (no silent black box).
type MergeInspection struct {
	Classification MergeClassification `json:"classification"`
	Explain        string              `json:"explain_operator"`
	TimingNote     string              `json:"timing_note"`
}

// MergeInspectionFromClassification attaches timing and explanation to a merge classification.
func MergeInspectionFromClassification(c MergeClassification) MergeInspection {
	return MergeInspection{
		Classification: c,
		Explain:        ExplainMergeForOperator(c),
		TimingNote:     "Ordering is instance-local or import-local; cross-instance total order is not implied.",
	}
}

// ImportNowRFC3339 returns UTC timestamp for audit fields.
func ImportNowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// ErrValidationRejected is returned when validation outcome is rejected (caller checks validation.Outcome).
var ErrValidationRejected = errors.New("remote evidence validation rejected")
