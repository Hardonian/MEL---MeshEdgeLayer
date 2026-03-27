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
	// RemoteEvidenceBundleSchemaVersion is the schema revision for one bundle/item wrapper.
	RemoteEvidenceBundleSchemaVersion = "1.0"
	// RemoteEvidenceBundleKind identifies the canonical JSON wrapper for one evidence envelope import.
	RemoteEvidenceBundleKind = "mel_remote_evidence_bundle"
	// RemoteEvidenceBatchSchemaVersion is the schema revision for canonical multi-item export/import payloads.
	RemoteEvidenceBatchSchemaVersion = "1.0"
	// RemoteEvidenceBatchKind identifies the canonical JSON wrapper for a multi-item offline import.
	RemoteEvidenceBatchKind = "mel_remote_evidence_batch"
)

// TimingOrderPosture describes how ordering/timing should be interpreted for imported or merged views.
type TimingOrderPosture string

const (
	TimingOrderLocalOrdered                TimingOrderPosture = "local_ordered"
	TimingOrderImportedPreserved           TimingOrderPosture = "imported_preserved_order"
	TimingOrderMergedBestEffort            TimingOrderPosture = "merged_best_effort_order"
	TimingOrderUncertainClockSkew          TimingOrderPosture = "ordering_uncertain_due_to_clock_skew"
	TimingOrderMissingTimestamps           TimingOrderPosture = "ordering_uncertain_missing_timestamps"
	TimingOrderReceiveDiffersFromObserved  TimingOrderPosture = "receive_time_differs_from_observed_time"
	TimingOrderImportTimeNotEqualEventTime TimingOrderPosture = "import_time_not_equal_event_time"
	TimingOrderHistoricalImportNotLive     TimingOrderPosture = "historical_import_not_live"
	TimingOrderStaleReporterContributed    TimingOrderPosture = "stale_reporter_contributed"
	TimingOrderMixedFreshnessWindow        TimingOrderPosture = "mixed_freshness_window"
)

// ValidationOutcome is the structural validation result for a remote import (validation != authenticity or authority).
type ValidationOutcome string

const (
	ValidationAccepted            ValidationOutcome = "accepted"
	ValidationAcceptedWithCaveats ValidationOutcome = "accepted_with_caveats"
	ValidationAcceptedPartial     ValidationOutcome = "accepted_partial_bundle"
	ValidationRejected            ValidationOutcome = "rejected"
)

// ValidationReasonCode is a machine-readable reason for acceptance caveats or rejection.
type ValidationReasonCode string

const (
	ReasonOK                           ValidationReasonCode = "ok"
	ReasonMalformedJSON                ValidationReasonCode = "malformed_json"
	ReasonUnsupportedSchema            ValidationReasonCode = "unsupported_schema_version"
	ReasonUnsupportedBundleKind        ValidationReasonCode = "unsupported_bundle_kind"
	ReasonEmptyPayload                 ValidationReasonCode = "empty_payload"
	ReasonMissingEvidenceClass         ValidationReasonCode = "missing_evidence_class"
	ReasonMissingOriginInstance        ValidationReasonCode = "missing_origin_instance_id"
	ReasonMissingScope                 ValidationReasonCode = "missing_scope"
	ReasonMissingTimestamps            ValidationReasonCode = "missing_timestamps"
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
	ReasonUnsupportedEvidenceType      ValidationReasonCode = "unsupported_evidence_type"
	ReasonInvalidMergeBasis            ValidationReasonCode = "invalid_merge_basis"
	CaveatNotCryptographicallyVerified ValidationReasonCode = "authenticity_not_cryptographically_verified"
	CaveatPartialObservationOnly       ValidationReasonCode = "partial_observation_only"
	CaveatReceiveDiffersFromObserved   ValidationReasonCode = "receive_time_differs_from_observed_time"
	CaveatHistoricalImportOnly         ValidationReasonCode = "historical_import_not_live"
	CaveatAcceptedPartialBundle        ValidationReasonCode = "accepted_partial_bundle"
	CaveatUnverifiedOrigin             ValidationReasonCode = "accepted_unverified_origin"
)

const remoteEvidenceAuthenticityNote = "Import authenticity is not cryptographically verified in core; treat origin and scope as claimed-origin text unless you verify them outside MEL."

// RemoteEvidenceImportContext captures operator-supplied context that remains part of the raw imported record.
type RemoteEvidenceImportContext struct {
	SourceLabel  string `json:"source_label,omitempty"`
	ImportReason string `json:"import_reason,omitempty"`
}

// RemoteEvidenceBundle is the canonical JSON wrapper for importing one EvidenceEnvelope (offline).
type RemoteEvidenceBundle struct {
	SchemaVersion string `json:"schema_version"`
	Kind          string `json:"kind"`
	// ClaimedOriginInstanceID is optional declared sender; must match Evidence.OriginInstanceID when both set (strict mode).
	ClaimedOriginInstanceID string                     `json:"claimed_origin_instance_id,omitempty"`
	ClaimedOriginSiteID     string                     `json:"claimed_origin_site_id,omitempty"`
	ClaimedFleetID          string                     `json:"claimed_fleet_id,omitempty"`
	ImportContext           RemoteEvidenceImportContext `json:"import_context,omitempty"`
	Evidence                EvidenceEnvelope            `json:"evidence"`
	Event                   *EventEnvelope              `json:"event,omitempty"`
}

// RemoteEvidenceBatchClaimedOrigin preserves claimed batch-level sender scope separately from item-level evidence.
type RemoteEvidenceBatchClaimedOrigin struct {
	InstanceID string `json:"instance_id,omitempty"`
	SiteID     string `json:"site_id,omitempty"`
	FleetID    string `json:"fleet_id,omitempty"`
}

// RemoteEvidenceImportSource records how this instance received a payload.
type RemoteEvidenceImportSource struct {
	SourceType      string `json:"source_type,omitempty"`
	SourceName      string `json:"source_name,omitempty"`
	SourcePath      string `json:"source_path,omitempty"`
	SupportBundleID string `json:"support_bundle_id,omitempty"`
}

// RemoteEvidenceBatch is the canonical offline multi-item import/export payload.
type RemoteEvidenceBatch struct {
	SchemaVersion    string                         `json:"schema_version"`
	Kind             string                         `json:"kind"`
	ExportedAt       string                         `json:"exported_at,omitempty"`
	ClaimedOrigin    RemoteEvidenceBatchClaimedOrigin `json:"claimed_origin,omitempty"`
	CapabilityPosture CapabilityPosture             `json:"capability_posture,omitempty"`
	SourceContext    RemoteEvidenceImportSource     `json:"source_context,omitempty"`
	Items            []RemoteEvidenceBundle         `json:"items"`
}

// RemoteEvidenceValidation is the typed validation/trust result for one imported item.
type RemoteEvidenceValidation struct {
	Outcome          ValidationOutcome      `json:"outcome"`
	Reasons          []ValidationReasonCode `json:"reasons"`
	TrustPosture     string                 `json:"trust_posture"`
	AuthenticityNote string                 `json:"authenticity_note"`
	OrderingPosture  TimingOrderPosture     `json:"ordering_posture"`
	Summary          string                 `json:"summary"`
}

// RemoteEvidenceBatchValidation is the typed validation/trust result for the overall payload.
type RemoteEvidenceBatchValidation struct {
	Outcome                  ValidationOutcome      `json:"outcome"`
	Reasons                  []ValidationReasonCode `json:"reasons,omitempty"`
	TrustPosture             string                 `json:"trust_posture"`
	AuthenticityNote         string                 `json:"authenticity_note"`
	OfflineOnlyNote          string                 `json:"offline_only_note"`
	Summary                  string                 `json:"summary"`
	StructurallyValid        bool                   `json:"structurally_valid"`
	ItemCount                int                    `json:"item_count"`
	AcceptedCount            int                    `json:"accepted_count"`
	AcceptedWithCaveatsCount int                    `json:"accepted_with_caveats_count"`
	RejectedCount            int                    `json:"rejected_count"`
}

// RemoteEvidenceImportItem is the normalized per-item validation result inside an import payload.
type RemoteEvidenceImportItem struct {
	Sequence   int                      `json:"sequence"`
	Bundle     RemoteEvidenceBundle     `json:"bundle"`
	Validation RemoteEvidenceValidation `json:"validation"`
}

// RemoteEvidenceImportPayload is the normalized import contract after single-item or batch parsing.
type RemoteEvidenceImportPayload struct {
	InputKind  string                      `json:"input_kind"`
	Batch      RemoteEvidenceBatch         `json:"batch"`
	Validation RemoteEvidenceBatchValidation `json:"validation"`
	Items      []RemoteEvidenceImportItem  `json:"items,omitempty"`
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

// ValidateRemoteEvidenceImportPayload validates a canonical offline remote-evidence payload.
// It accepts either one mel_remote_evidence_bundle or one mel_remote_evidence_batch.
func ValidateRemoteEvidenceImportPayload(raw []byte, localSiteID, localFleetID string, opts IngestValidateOptions) (RemoteEvidenceImportPayload, error) {
	rejectBase := RemoteEvidenceBatchValidation{
		Outcome:          ValidationRejected,
		TrustPosture:     TrustPostureRejected,
		AuthenticityNote: remoteEvidenceAuthenticityNote,
		OfflineOnlyNote:  "Remote evidence import is offline/file-scoped in core MEL; it is not live federation, remote liveness, or remote control.",
	}
	raw = trimUTF8BOM(raw)
	if len(strings.TrimSpace(string(raw))) == 0 {
		rejectBase.Reasons = []ValidationReasonCode{ReasonEmptyPayload}
		rejectBase.Summary = "Empty payload."
		return RemoteEvidenceImportPayload{Validation: rejectBase}, nil
	}

	var head struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		rejectBase.Reasons = []ValidationReasonCode{ReasonMalformedJSON}
		rejectBase.Summary = "Malformed JSON."
		return RemoteEvidenceImportPayload{Validation: rejectBase}, fmt.Errorf("parse remote evidence payload: %w", err)
	}

	switch strings.TrimSpace(head.Kind) {
	case RemoteEvidenceBundleKind:
		bundle, validation, err := ValidateRemoteEvidenceBundle(raw, localSiteID, localFleetID, opts)
		payload := RemoteEvidenceImportPayload{
			InputKind: RemoteEvidenceBundleKind,
			Batch: RemoteEvidenceBatch{
				SchemaVersion: RemoteEvidenceBatchSchemaVersion,
				Kind:          RemoteEvidenceBatchKind,
				ClaimedOrigin: RemoteEvidenceBatchClaimedOrigin{
					InstanceID: strings.TrimSpace(bundle.ClaimedOriginInstanceID),
					SiteID:     strings.TrimSpace(bundle.ClaimedOriginSiteID),
					FleetID:    strings.TrimSpace(bundle.ClaimedFleetID),
				},
				Items: []RemoteEvidenceBundle{bundle},
			},
			Items: []RemoteEvidenceImportItem{{Sequence: 1, Bundle: bundle, Validation: validation}},
		}
		payload.Validation = summarizeBatchValidation(payload.Items)
		if err != nil {
			return payload, err
		}
		return payload, nil
	case RemoteEvidenceBatchKind:
		var batch RemoteEvidenceBatch
		if err := json.Unmarshal(raw, &batch); err != nil {
			rejectBase.Reasons = []ValidationReasonCode{ReasonMalformedJSON}
			rejectBase.Summary = "Malformed JSON."
			return RemoteEvidenceImportPayload{Validation: rejectBase}, fmt.Errorf("parse remote evidence batch: %w", err)
		}
		payload := RemoteEvidenceImportPayload{
			InputKind: RemoteEvidenceBatchKind,
			Batch:     batch,
		}
		if strings.TrimSpace(batch.SchemaVersion) != RemoteEvidenceBatchSchemaVersion {
			rejectBase.Reasons = []ValidationReasonCode{ReasonUnsupportedSchema}
			rejectBase.Summary = fmt.Sprintf("Unsupported schema_version %q (want %s).", batch.SchemaVersion, RemoteEvidenceBatchSchemaVersion)
			payload.Validation = rejectBase
			return payload, nil
		}
		if strings.TrimSpace(batch.Kind) != RemoteEvidenceBatchKind {
			rejectBase.Reasons = []ValidationReasonCode{ReasonUnsupportedBundleKind}
			rejectBase.Summary = fmt.Sprintf("Unsupported kind %q (want %s).", batch.Kind, RemoteEvidenceBatchKind)
			payload.Validation = rejectBase
			return payload, nil
		}
		if len(batch.Items) == 0 {
			rejectBase.Reasons = []ValidationReasonCode{ReasonEmptyPayload}
			rejectBase.Summary = "Batch contains no importable items."
			payload.Validation = rejectBase
			return payload, nil
		}
		items := make([]RemoteEvidenceImportItem, 0, len(batch.Items))
		for idx, item := range batch.Items {
			effective := mergeBatchDefaultsIntoBundle(batch, item)
			validation := validateRemoteEvidenceBundleStruct(effective, localSiteID, localFleetID, opts)
			items = append(items, RemoteEvidenceImportItem{
				Sequence:   idx + 1,
				Bundle:     effective,
				Validation: validation,
			})
		}
		payload.Items = items
		payload.Validation = summarizeBatchValidation(items)
		return payload, nil
	default:
		rejectBase.Reasons = []ValidationReasonCode{ReasonUnsupportedBundleKind}
		rejectBase.Summary = fmt.Sprintf("Unsupported kind %q.", strings.TrimSpace(head.Kind))
		return RemoteEvidenceImportPayload{
			InputKind:  strings.TrimSpace(head.Kind),
			Validation: rejectBase,
		}, nil
	}
}

// ValidateRemoteEvidenceBundle validates JSON bytes and returns bundle + validation. err is set only on malformed JSON.
func ValidateRemoteEvidenceBundle(raw []byte, localSiteID, localFleetID string, opts IngestValidateOptions) (RemoteEvidenceBundle, RemoteEvidenceValidation, error) {
	rejectBase := RemoteEvidenceValidation{
		Outcome:          ValidationRejected,
		TrustPosture:     TrustPostureRejected,
		AuthenticityNote: remoteEvidenceAuthenticityNote,
		OrderingPosture:  TimingOrderImportedPreserved,
	}
	raw = trimUTF8BOM(raw)
	var b RemoteEvidenceBundle
	if err := json.Unmarshal(raw, &b); err != nil {
		rejectBase.Reasons = []ValidationReasonCode{ReasonMalformedJSON}
		rejectBase.Summary = "Malformed JSON."
		return b, rejectBase, fmt.Errorf("parse remote evidence bundle: %w", err)
	}
	return b, validateRemoteEvidenceBundleStruct(b, localSiteID, localFleetID, opts), nil
}

func validateRemoteEvidenceBundleStruct(b RemoteEvidenceBundle, localSiteID, localFleetID string, opts IngestValidateOptions) RemoteEvidenceValidation {
	v := RemoteEvidenceValidation{
		Outcome:          ValidationRejected,
		TrustPosture:     TrustPostureRejected,
		AuthenticityNote: remoteEvidenceAuthenticityNote,
		OrderingPosture:  TimingOrderImportedPreserved,
	}
	if strings.TrimSpace(b.SchemaVersion) != RemoteEvidenceBundleSchemaVersion {
		v.Reasons = []ValidationReasonCode{ReasonUnsupportedSchema}
		v.Summary = fmt.Sprintf("Unsupported schema_version %q (want %s).", b.SchemaVersion, RemoteEvidenceBundleSchemaVersion)
		return v
	}
	if strings.TrimSpace(b.Kind) != RemoteEvidenceBundleKind {
		v.Reasons = []ValidationReasonCode{ReasonUnsupportedBundleKind}
		v.Summary = fmt.Sprintf("Unsupported kind %q (want %s).", b.Kind, RemoteEvidenceBundleKind)
		return v
	}
	if strings.TrimSpace(string(b.Evidence.EvidenceClass)) == "" {
		v.Reasons = []ValidationReasonCode{ReasonMissingEvidenceClass}
		v.Summary = "Missing evidence.evidence_class."
		return v
	}
	if !isKnownEvidenceClass(b.Evidence.EvidenceClass) {
		v.Reasons = []ValidationReasonCode{ReasonUnsupportedEvidenceType}
		v.Summary = fmt.Sprintf("Unsupported evidence.evidence_class %q.", b.Evidence.EvidenceClass)
		return v
	}
	if strings.TrimSpace(b.Evidence.OriginInstanceID) == "" {
		v.Reasons = []ValidationReasonCode{ReasonMissingOriginInstance, ReasonMissingScope}
		v.Summary = "Missing evidence.origin_instance_id."
		return v
	}
	if !isKnownOriginClass(b.Evidence.OriginClass) {
		v.Reasons = []ValidationReasonCode{ReasonInvalidOriginClass}
		v.Summary = fmt.Sprintf("Invalid observation_origin_class %q.", b.Evidence.OriginClass)
		return v
	}
	if !bundleHasTimestamps(b) {
		v.Reasons = []ValidationReasonCode{ReasonMissingTimestamps}
		v.OrderingPosture = TimingOrderMissingTimestamps
		v.Summary = "Missing observed_at/received_at/recorded_at timing fields in evidence or event envelope."
		return v
	}
	if !bundleHasMergeBasis(b) {
		v.Reasons = []ValidationReasonCode{ReasonInvalidMergeBasis}
		v.Summary = "Bundle lacks a stable correlation basis for merge/timeline inspection."
		return v
	}
	if b.Event != nil {
		if strings.TrimSpace(b.Event.EventID) == "" {
			v.Reasons = []ValidationReasonCode{ReasonMissingEventID}
			v.Summary = "Missing event.event_id."
			return v
		}
		if strings.TrimSpace(b.Event.EventType) == "" {
			v.Reasons = []ValidationReasonCode{ReasonMissingEventType}
			v.Summary = "Missing event.event_type."
			return v
		}
		if strings.TrimSpace(b.Event.Summary) == "" {
			v.Reasons = []ValidationReasonCode{ReasonMissingEventSummary}
			v.Summary = "Missing event.summary."
			return v
		}
		if strings.TrimSpace(b.Event.OriginInstanceID) == "" {
			v.Reasons = []ValidationReasonCode{ReasonMissingOriginInstance, ReasonMissingScope}
			v.Summary = "Missing event.origin_instance_id."
			return v
		}
		if strings.TrimSpace(b.Event.OriginInstanceID) != strings.TrimSpace(b.Evidence.OriginInstanceID) {
			v.Reasons = []ValidationReasonCode{ReasonEventOriginMismatch}
			v.Summary = "event.origin_instance_id must match evidence.origin_instance_id."
			return v
		}
		if strings.TrimSpace(b.Event.OriginSiteID) != "" && strings.TrimSpace(b.Evidence.OriginSiteID) != "" &&
			strings.TrimSpace(b.Event.OriginSiteID) != strings.TrimSpace(b.Evidence.OriginSiteID) {
			v.Reasons = []ValidationReasonCode{ReasonConflictingOriginSite}
			v.Summary = "event.origin_site_id must match evidence.origin_site_id when both are present."
			return v
		}
		if strings.TrimSpace(b.Event.CorrelationID) != "" && strings.TrimSpace(b.Evidence.CorrelationID) != "" &&
			strings.TrimSpace(b.Event.CorrelationID) != strings.TrimSpace(b.Evidence.CorrelationID) {
			v.Reasons = []ValidationReasonCode{ReasonEventCorrelationMismatch}
			v.Summary = "event.correlation_id must match evidence.correlation_id when both are present."
			return v
		}
	}

	claimedInst := strings.TrimSpace(b.ClaimedOriginInstanceID)
	evOrigin := strings.TrimSpace(b.Evidence.OriginInstanceID)
	if claimedInst != "" && claimedInst != evOrigin {
		if opts.StrictOriginMatch {
			v.Reasons = []ValidationReasonCode{ReasonClaimedOriginMismatch}
			v.Summary = "claimed_origin_instance_id does not match evidence.origin_instance_id."
			return v
		}
	}

	evSite := strings.TrimSpace(b.Evidence.OriginSiteID)
	localSite := strings.TrimSpace(localSiteID)
	if localSite != "" && evSite != "" && evSite != localSite {
		v.Reasons = []ValidationReasonCode{ReasonConflictingOriginSite}
		v.Summary = "Evidence origin_site_id conflicts with this instance's configured site scope."
		return v
	}
	claimSite := strings.TrimSpace(b.ClaimedOriginSiteID)
	if localSite != "" && claimSite != "" && claimSite != localSite {
		v.Reasons = []ValidationReasonCode{ReasonConflictingOriginSite}
		v.Summary = "claimed_origin_site_id conflicts with this instance's configured site scope."
		return v
	}

	lf := strings.TrimSpace(localFleetID)
	cf := strings.TrimSpace(b.ClaimedFleetID)
	if lf != "" && cf != "" && cf != lf {
		v.Reasons = []ValidationReasonCode{ReasonConflictingClaimedFleet}
		v.Summary = "claimed_fleet_id conflicts with this instance's configured fleet scope."
		return v
	}

	reasons := []ValidationReasonCode{
		CaveatNotCryptographicallyVerified,
		CaveatUnverifiedOrigin,
		CaveatHistoricalImportOnly,
		CaveatPartialObservationOnly,
	}
	ordering := TimingOrderHistoricalImportNotLive
	if strings.TrimSpace(b.Evidence.ReceivedAt) != "" && strings.TrimSpace(b.Evidence.ObservedAt) != "" &&
		strings.TrimSpace(b.Evidence.ReceivedAt) != strings.TrimSpace(b.Evidence.ObservedAt) {
		reasons = append(reasons, CaveatReceiveDiffersFromObserved)
		ordering = TimingOrderReceiveDiffersFromObserved
	} else if strings.TrimSpace(b.Evidence.ObservedAt) != "" || strings.TrimSpace(b.Evidence.RecordedAt) != "" || eventHasAnyTimestamp(b.Event) {
		ordering = TimingOrderImportTimeNotEqualEventTime
	}
	if claimedInst != "" && claimedInst != evOrigin && !opts.StrictOriginMatch {
		reasons = append(reasons, ReasonClaimedOriginMismatch)
	}
	if evSite == "" {
		reasons = append(reasons, ReasonOriginSiteAbsent)
	}

	v.Outcome = ValidationAcceptedWithCaveats
	v.Reasons = uniqueReasonCodes(reasons)
	v.TrustPosture = TrustPostureImportedReadOnly
	v.OrderingPosture = ordering
	v.Summary = "Accepted with caveats: structurally valid offline remote evidence; authenticity and authority are not verified; import remains historical/read-only."
	return v
}

func summarizeBatchValidation(items []RemoteEvidenceImportItem) RemoteEvidenceBatchValidation {
	out := RemoteEvidenceBatchValidation{
		Outcome:          ValidationRejected,
		TrustPosture:     TrustPostureRejected,
		AuthenticityNote: remoteEvidenceAuthenticityNote,
		OfflineOnlyNote:  "Remote evidence import is offline/file-scoped in core MEL; it does not establish live federation, remote execution, or fleet-wide authority.",
		ItemCount:        len(items),
	}
	if len(items) == 0 {
		out.Reasons = []ValidationReasonCode{ReasonEmptyPayload}
		out.Summary = "Batch contains no importable items."
		return out
	}
	out.StructurallyValid = true
	reasons := []ValidationReasonCode{CaveatNotCryptographicallyVerified, CaveatHistoricalImportOnly}
	for _, item := range items {
		switch item.Validation.Outcome {
		case ValidationRejected:
			out.RejectedCount++
		case ValidationAccepted:
			out.AcceptedCount++
		case ValidationAcceptedWithCaveats:
			out.AcceptedWithCaveatsCount++
		case ValidationAcceptedPartial:
			out.AcceptedWithCaveatsCount++
		}
		reasons = append(reasons, item.Validation.Reasons...)
	}
	acceptedTotal := out.AcceptedCount + out.AcceptedWithCaveatsCount
	switch {
	case acceptedTotal == 0:
		out.Outcome = ValidationRejected
		out.TrustPosture = TrustPostureRejected
		out.Summary = "All items were rejected. The payload is retained only as an audit trail."
	case out.RejectedCount > 0:
		out.Outcome = ValidationAcceptedPartial
		out.TrustPosture = TrustPostureImportedReadOnly
		reasons = append(reasons, CaveatAcceptedPartialBundle)
		out.Summary = fmt.Sprintf("Accepted partial bundle: %d item(s) imported, %d rejected. Accepted evidence remains historical/read-only and unverified.", acceptedTotal, out.RejectedCount)
	default:
		out.Outcome = ValidationAcceptedWithCaveats
		out.TrustPosture = TrustPostureImportedReadOnly
		out.Summary = fmt.Sprintf("Accepted %d item(s) with caveats: offline import remains read-only, historical, and authenticity-unverified.", acceptedTotal)
	}
	out.Reasons = uniqueReasonCodes(reasons)
	return out
}

func mergeBatchDefaultsIntoBundle(batch RemoteEvidenceBatch, item RemoteEvidenceBundle) RemoteEvidenceBundle {
	merged := item
	if strings.TrimSpace(merged.SchemaVersion) == "" {
		merged.SchemaVersion = RemoteEvidenceBundleSchemaVersion
	}
	if strings.TrimSpace(merged.Kind) == "" {
		merged.Kind = RemoteEvidenceBundleKind
	}
	if strings.TrimSpace(merged.ClaimedOriginInstanceID) == "" {
		merged.ClaimedOriginInstanceID = strings.TrimSpace(batch.ClaimedOrigin.InstanceID)
	}
	if strings.TrimSpace(merged.ClaimedOriginSiteID) == "" {
		merged.ClaimedOriginSiteID = strings.TrimSpace(batch.ClaimedOrigin.SiteID)
	}
	if strings.TrimSpace(merged.ClaimedFleetID) == "" {
		merged.ClaimedFleetID = strings.TrimSpace(batch.ClaimedOrigin.FleetID)
	}
	return merged
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

func isKnownEvidenceClass(class EvidenceClass) bool {
	switch class {
	case EvidenceClassPacketObservation, EvidenceClassTransportHealth, EvidenceClassControlAction,
		EvidenceClassIncident, EvidenceClassOperatorNote, EvidenceClassOther:
		return true
	default:
		return false
	}
}

func bundleHasTimestamps(b RemoteEvidenceBundle) bool {
	return strings.TrimSpace(b.Evidence.ObservedAt) != "" ||
		strings.TrimSpace(b.Evidence.ReceivedAt) != "" ||
		strings.TrimSpace(b.Evidence.RecordedAt) != "" ||
		eventHasAnyTimestamp(b.Event)
}

func eventHasAnyTimestamp(ev *EventEnvelope) bool {
	if ev == nil {
		return false
	}
	return strings.TrimSpace(ev.ObservedAt) != "" ||
		strings.TrimSpace(ev.ReceivedAt) != "" ||
		strings.TrimSpace(ev.RecordedAt) != ""
}

func bundleHasMergeBasis(b RemoteEvidenceBundle) bool {
	if strings.TrimSpace(b.Evidence.CorrelationID) != "" {
		return true
	}
	if b.Event != nil && strings.TrimSpace(b.Event.EventID) != "" {
		return true
	}
	if strings.TrimSpace(b.Evidence.ObservedAt) != "" || strings.TrimSpace(b.Evidence.ReceivedAt) != "" || strings.TrimSpace(b.Evidence.RecordedAt) != "" {
		return true
	}
	return len(b.Evidence.Details) > 0
}

func uniqueReasonCodes(in []ValidationReasonCode) []ValidationReasonCode {
	seen := map[ValidationReasonCode]bool{}
	out := make([]ValidationReasonCode, 0, len(in))
	for _, item := range in {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
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
