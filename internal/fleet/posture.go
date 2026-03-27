package fleet

// Truth and capability posture strings are stable API contract values (logs, JSON, operators).

const (
	// TruthPostureSingleInstance is the default: this SQLite database is authoritative only for evidence it stores.
	TruthPostureSingleInstance = "single_instance_local_authority"

	// TruthPosturePartialFleet means operators declared multi-site context but this process only sees its own reporters.
	TruthPosturePartialFleet = "partial_fleet_visibility"

	// TruthPostureUnknownSite means no site_id is configured or persisted; cross-site correlation is undefined.
	TruthPostureUnknownSite = "site_scope_unknown"

	// VisibilityPartialFleet means declared fleet size exceeds what this instance can observe (always true when expected > 1 and only local DB).
	VisibilityPartialFleet = "partial_fleet"

	// VisibilitySingleObserver means only this MEL instance contributes to stored evidence in this database.
	VisibilitySingleObserver = "single_observer_instance"

	// FederationModeNone is the supported mode: no cross-instance sync in core.
	FederationModeNone = "none"

	// FederationEvidencePlaneUnsupported means live remote-ingest federation is not implemented as a core capability.
	FederationEvidencePlaneUnsupported = "unsupported"

	// FederationOfflineEvidenceImport means operator-initiated file/bundle import into local SQLite (not live sync).
	FederationOfflineEvidenceImport = "offline_bundle_import_local_sqlite"

	// ActionExecutionPlaneLocalOnly means control actions execute only in this process against configured transports.
	ActionExecutionPlaneLocalOnly = "local_execution_only"

	// OrderingPostureInstanceBestEffort means timeline ordering is by recorded timestamps in this DB, not globally total.
	OrderingPostureInstanceBestEffort = "instance_best_effort_chronological"

	// OrderingPostureNoCrossInstanceTotalOrder means merged or federated views must not imply Lamport/global order.
	OrderingPostureNoCrossInstanceTotalOrder = "no_cross_instance_total_order"
)

// CapabilityPosture summarizes what this build honestly supports for multi-instance operations.
type CapabilityPosture struct {
	FederationMode string `json:"federation_mode"`
	// FederationReadOnlyEvidenceIngest describes offline import posture (see FederationOfflineEvidenceImport); not live mesh sync.
	FederationReadOnlyEvidenceIngest string `json:"federation_read_only_evidence_ingest"`
	CrossInstanceActionExecution     string `json:"cross_instance_action_execution"`
	FleetAggregationSupported        bool   `json:"fleet_aggregation_supported"`
	Notes                            string `json:"notes"`
}

// DefaultCapabilityPosture returns the honest envelope for the shipping binary.
func DefaultCapabilityPosture() CapabilityPosture {
	return CapabilityPosture{
		FederationMode:                   FederationModeNone,
		FederationReadOnlyEvidenceIngest: FederationOfflineEvidenceImport,
		CrossInstanceActionExecution:     ActionExecutionPlaneLocalOnly,
		FleetAggregationSupported:        false,
		Notes: "Core MEL does not ship live cross-instance evidence sync or cross-instance action execution. " +
			"Offline remote evidence bundles may be imported into this instance's SQLite for audit and review with explicit validation posture; " +
			"that is not global fleet authority, cryptographic authenticity, or real-time federation.",
	}
}
