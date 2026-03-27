package fleet

import (
	"strconv"
	"strings"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
)

// ObservationOriginClass classifies how evidence entered this instance (typed; not ambiguous "source").
type ObservationOriginClass string

const (
	OriginDirectIngest       ObservationOriginClass = "direct_ingest"
	OriginForwardedTransport ObservationOriginClass = "forwarded_transport"
	OriginRemoteReported     ObservationOriginClass = "remote_reported"
	OriginAggregated         ObservationOriginClass = "aggregated"
	OriginInferred           ObservationOriginClass = "inferred"
	OriginUnknown            ObservationOriginClass = "unknown"
)

// EvidenceClass is a coarse evidence category for contracts and export.
type EvidenceClass string

const (
	EvidenceClassPacketObservation EvidenceClass = "packet_observation"
	EvidenceClassTransportHealth   EvidenceClass = "transport_health"
	EvidenceClassControlAction     EvidenceClass = "control_action"
	EvidenceClassIncident          EvidenceClass = "incident"
	EvidenceClassOperatorNote      EvidenceClass = "operator_note"
	EvidenceClassOther             EvidenceClass = "other"
)

// PhysicalUncertaintyPosture states what network/RF conclusions are NOT claimed.
type PhysicalUncertaintyPosture string

const (
	PhysicalUncertaintyDefault PhysicalUncertaintyPosture = "partial_observation_clock_skew_duplication_delay"
)

// EvidenceEnvelope is the canonical cross-surface evidence contract (export/import safe).
// Confidence is not a numeric RF guarantee; omit or use qualitative fields only when grounded.
type EvidenceEnvelope struct {
	EvidenceClass EvidenceClass `json:"evidence_class"`

	OriginInstanceID   string                 `json:"origin_instance_id"`
	OriginSiteID       string                 `json:"origin_site_id,omitempty"`
	ObserverInstanceID string                 `json:"observer_instance_id,omitempty"`
	OriginClass        ObservationOriginClass `json:"observation_origin_class"`

	ObservedAt   string `json:"observed_at,omitempty"`
	ReceivedAt   string `json:"received_at,omitempty"`
	RecordedAt   string `json:"recorded_at,omitempty"`
	EventTimeSrc string `json:"event_time_source,omitempty"`

	CorrelationID string `json:"correlation_id,omitempty"`

	PhysicalUncertainty PhysicalUncertaintyPosture `json:"physical_uncertainty_posture"`

	Details map[string]any `json:"details,omitempty"`
}

// EventEnvelope is the canonical event contract for timelines and federation-oriented interchange.
type EventEnvelope struct {
	EventID       string `json:"event_id"`
	CorrelationID string `json:"correlation_id,omitempty"`

	OriginInstanceID string `json:"origin_instance_id"`
	OriginSiteID     string `json:"origin_site_id,omitempty"`

	ObservedAt    string `json:"observed_at,omitempty"`
	RecordedAt    string `json:"recorded_at,omitempty"`
	ReceivedAt    string `json:"received_at,omitempty"`
	OrderingBasis string `json:"ordering_basis,omitempty"`

	EventType string         `json:"event_type"`
	Summary   string         `json:"summary"`
	Details   map[string]any `json:"details,omitempty"`
}

// FleetTruthSummary is operator-visible instance/fleet boundary truth without implying global health.
type FleetTruthSummary struct {
	InstanceID   string `json:"instance_id"`
	SiteID       string `json:"site_id,omitempty"`
	FleetID      string `json:"fleet_id,omitempty"`
	FleetLabel   string `json:"fleet_label,omitempty"`
	GatewayLabel string `json:"gateway_label,omitempty"`

	TruthPosture string `json:"truth_posture"`
	Visibility   string `json:"visibility_posture"`

	ExpectedFleetReporters  int `json:"expected_fleet_reporters"`
	ReportingInstancesKnown int `json:"reporting_instances_known"`

	PartialVisibilityReasons []string `json:"partial_visibility_reasons,omitempty"`

	OrderingPosture    string            `json:"ordering_posture"`
	CapabilityPosture  CapabilityPosture `json:"capability_posture"`
	PhysicsNetworkNote string            `json:"physics_network_note"`
}

// BuildTruthSummary loads persisted scope metadata and config to produce a stable summary.
func BuildTruthSummary(cfg config.Config, database *db.DB) (FleetTruthSummary, error) {
	out := FleetTruthSummary{
		TruthPosture:            TruthPostureSingleInstance,
		Visibility:              VisibilitySingleObserver,
		ExpectedFleetReporters:  1,
		ReportingInstancesKnown: 1,
		OrderingPosture:         OrderingPostureInstanceBestEffort,
		CapabilityPosture:       DefaultCapabilityPosture(),
		PhysicsNetworkNote: "Observation is not coverage; missing observations are not proof of absence. " +
			"RSSI/SNR are local evidence points, not path or topology guarantees. " +
			"Repeated observations across observers are not automatic flooding or congestion proof.",
	}
	site := ""
	fleet := ""
	label := ""
	expStr := ""
	if database != nil {
		id, err := database.EnsureInstanceID()
		if err != nil {
			return out, err
		}
		out.InstanceID = id
		site, _, _ = database.GetInstanceMetadata(db.MetaSiteID)
		fleet, _, _ = database.GetInstanceMetadata(db.MetaFleetID)
		label, _, _ = database.GetInstanceMetadata(db.MetaFleetLabel)
		expStr, _, _ = database.GetInstanceMetadata(db.MetaExpectedFleetReporters)
	} else {
		out.PartialVisibilityReasons = append(out.PartialVisibilityReasons,
			"database_unavailable_scope_metadata_not_loaded")
	}

	if strings.TrimSpace(cfg.Scope.SiteID) != "" {
		site = strings.TrimSpace(cfg.Scope.SiteID)
	}
	if strings.TrimSpace(cfg.Scope.FleetID) != "" {
		fleet = strings.TrimSpace(cfg.Scope.FleetID)
	}
	if strings.TrimSpace(cfg.Scope.FleetLabel) != "" {
		label = strings.TrimSpace(cfg.Scope.FleetLabel)
	}
	out.SiteID = strings.TrimSpace(site)
	out.FleetID = strings.TrimSpace(fleet)
	out.FleetLabel = strings.TrimSpace(label)
	out.GatewayLabel = strings.TrimSpace(cfg.Scope.GatewayLabel)

	expected := 1
	if cfg.Scope.ExpectedFleetReporterCount > 0 {
		expected = cfg.Scope.ExpectedFleetReporterCount
	} else if strings.TrimSpace(expStr) != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(expStr)); err == nil && n > 0 {
			expected = n
		}
	}
	out.ExpectedFleetReporters = expected

	if out.SiteID == "" {
		out.TruthPosture = TruthPostureUnknownSite
		out.PartialVisibilityReasons = append(out.PartialVisibilityReasons, "site_id_not_configured")
	}

	if expected > 1 {
		out.Visibility = VisibilityPartialFleet
		out.PartialVisibilityReasons = append(out.PartialVisibilityReasons,
			"fleet_declared_multi_reporter_but_this_database_observes_only_this_instance")
		out.TruthPosture = TruthPosturePartialFleet
	}

	return out, nil
}
