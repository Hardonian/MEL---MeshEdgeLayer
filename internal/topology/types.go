// Package topology implements the canonical node/link/topology model for MEL.
// All scoring is transparent and explainable. No black-box health states.
package topology

import "time"

// HealthState classifies a node or link's operational state.
type HealthState string

const (
	HealthHealthy        HealthState = "healthy"
	HealthDegraded       HealthState = "degraded"
	HealthUnstable       HealthState = "unstable"
	HealthStale          HealthState = "stale"
	HealthWeaklyObserved HealthState = "weakly_observed"
	HealthInferredOnly   HealthState = "inferred_only"
	HealthIsolated       HealthState = "isolated"
	HealthBridgeCritical HealthState = "bridge_critical"
	HealthFlapping       HealthState = "flapping"
	HealthQuarantined    HealthState = "quarantined"
	HealthUnknown        HealthState = "unknown"
)

// LocationState describes confidence in a node's position.
type LocationState string

const (
	LocExact       LocationState = "exact"
	LocApproximate LocationState = "approximate"
	LocStale       LocationState = "stale"
	LocUnknown     LocationState = "unknown"
	LocRedacted    LocationState = "redacted"
)

// MobilityState describes whether a node is static or mobile.
type MobilityState string

const (
	MobStatic       MobilityState = "static"
	MobLikelyMobile MobilityState = "likely_mobile"
	MobUnknown      MobilityState = "unknown"
)

// TrustClass classifies source trust level.
type TrustClass string

const (
	TrustDirectLocal TrustClass = "direct_local"
	TrustTrusted     TrustClass = "trusted"
	TrustPartial     TrustClass = "partial"
	TrustUntrusted   TrustClass = "untrusted"
	TrustUnknown     TrustClass = "unknown"
)

// SourceTrust describes a connector's trust posture.
type SourceTrust struct {
	ConnectorID        string     `json:"connector_id"`
	ConnectorName      string     `json:"connector_name"`
	ConnectorType      string     `json:"connector_type"`
	TrustClass         TrustClass `json:"trust_class"`
	TrustLevel         float64    `json:"trust_level"`
	FirstSeenAt        string     `json:"first_seen_at"`
	LastSeenAt         string     `json:"last_seen_at"`
	ObservationCount   int64      `json:"observation_count"`
	ContradictionCount int64      `json:"contradiction_count"`
	StaleCount         int64      `json:"stale_count"`
	OperatorNotes      string     `json:"operator_notes,omitempty"`
}

// Node is the canonical mesh node entity.
type Node struct {
	NodeNum          int64         `json:"node_num"`
	NodeID           string        `json:"node_id"`
	LongName         string        `json:"long_name"`
	ShortName        string        `json:"short_name"`
	Role             string        `json:"role,omitempty"`
	HardwareModel    string        `json:"hardware_model,omitempty"`
	FirstSeenAt      string        `json:"first_seen_at,omitempty"`
	LastSeenAt       string        `json:"last_seen_at,omitempty"`
	LastDirectSeenAt string        `json:"last_direct_seen_at,omitempty"`
	LastBrokerSeenAt string        `json:"last_broker_seen_at,omitempty"`
	LastGatewayID    string        `json:"last_gateway_id,omitempty"`
	TrustClass       TrustClass    `json:"trust_class"`
	LocationState    LocationState `json:"location_state"`
	MobilityState    MobilityState `json:"mobility_state"`
	HealthState      HealthState   `json:"health_state"`
	HealthScore      float64       `json:"health_score"`
	HealthFactors    []ScoreFactor `json:"health_factors,omitempty"`
	LatRedacted      float64       `json:"lat_redacted,omitempty"`
	LonRedacted      float64       `json:"lon_redacted,omitempty"`
	Altitude         int64         `json:"altitude,omitempty"`
	LastSNR          float64       `json:"last_snr,omitempty"`
	LastRSSI         int64         `json:"last_rssi,omitempty"`
	MessageCount     int64         `json:"message_count,omitempty"`
	Stale            bool          `json:"stale"`
	Quarantined      bool          `json:"quarantined"`
	QuarantineReason string        `json:"quarantine_reason,omitempty"`
	SourceConnector  string        `json:"source_connector,omitempty"`
}

// Link represents a node-to-node edge in the topology graph.
type Link struct {
	EdgeID              string        `json:"edge_id"`
	SrcNodeNum          int64         `json:"src_node_num"`
	DstNodeNum          int64         `json:"dst_node_num"`
	Observed            bool          `json:"observed"`
	Directional         bool          `json:"directional"`
	TransportPath       string        `json:"transport_path,omitempty"`
	FirstObservedAt     string        `json:"first_observed_at"`
	LastObservedAt      string        `json:"last_observed_at"`
	QualityScore        float64       `json:"quality_score"`
	Reliability         float64       `json:"reliability"`
	IntermittenceCount  int64         `json:"intermittence_count"`
	SourceTrustLevel    float64       `json:"source_trust_level"`
	SourceConnectorID   string        `json:"source_connector_id,omitempty"`
	Stale               bool          `json:"stale"`
	Contradiction       bool          `json:"contradiction"`
	ContradictionDetail string        `json:"contradiction_detail,omitempty"`
	RelayDependent      bool          `json:"relay_dependent"`
	QualityFactors      []ScoreFactor `json:"quality_factors,omitempty"`
	ObservationCount    int64         `json:"observation_count"`
}

// ScoreFactor explains one component of a health or quality score.
type ScoreFactor struct {
	Name         string  `json:"name"`
	Weight       float64 `json:"weight"`
	Value        float64 `json:"value"`
	Contribution float64 `json:"contribution"` // weight * value
	Basis        string  `json:"basis"`        // observed, inferred, stale, policy
	Evidence     string  `json:"evidence,omitempty"`
}

// TopologySnapshot is a point-in-time summary of mesh topology.
type TopologySnapshot struct {
	SnapshotID        string             `json:"snapshot_id"`
	CreatedAt         string             `json:"created_at"`
	NodeCount         int                `json:"node_count"`
	EdgeCount         int                `json:"edge_count"`
	DirectEdgeCount   int                `json:"direct_edge_count"`
	InferredEdgeCount int                `json:"inferred_edge_count"`
	HealthyNodes      int                `json:"healthy_nodes"`
	DegradedNodes     int                `json:"degraded_nodes"`
	StaleNodes        int                `json:"stale_nodes"`
	IsolatedNodes     int                `json:"isolated_nodes"`
	GraphHash         string             `json:"graph_hash,omitempty"`
	SourceCoverage    map[string]int     `json:"source_coverage,omitempty"`
	ConfidenceSummary map[string]float64 `json:"confidence_summary,omitempty"`
	Explanation       []string           `json:"explanation,omitempty"`
	RegionSummary     []RegionScore      `json:"region_summary,omitempty"`
}

// RegionScore summarizes health for a topology cluster or region.
type RegionScore struct {
	RegionID    string      `json:"region_id"`
	Label       string      `json:"label"`
	State       HealthState `json:"state"` // strong, weak, stale, fragile, etc.
	NodeCount   int         `json:"node_count"`
	EdgeCount   int         `json:"edge_count"`
	Confidence  float64     `json:"confidence"`
	Bottlenecks []string    `json:"bottlenecks,omitempty"`
	Explanation []string    `json:"explanation,omitempty"`
}

// NodeObservation is a raw source-attributed sighting of a node.
type NodeObservation struct {
	NodeNum     int64   `json:"node_num"`
	ConnectorID string  `json:"connector_id"`
	SourceType  string  `json:"source_type"` // direct, broker, relay, import
	TrustLevel  float64 `json:"trust_level"`
	ObservedAt  string  `json:"observed_at"`
	SNR         float64 `json:"snr,omitempty"`
	RSSI        int64   `json:"rssi,omitempty"`
	Lat         float64 `json:"lat,omitempty"`
	Lon         float64 `json:"lon,omitempty"`
	Altitude    int64   `json:"altitude,omitempty"`
	HopCount    int     `json:"hop_count,omitempty"`
	ViaMQTT     bool    `json:"via_mqtt"`
	GatewayID   string  `json:"gateway_id,omitempty"`
}

// Bookmark represents an operator preference or label on a node.
type Bookmark struct {
	ID           int64  `json:"id"`
	NodeNum      int64  `json:"node_num"`
	BookmarkType string `json:"bookmark_type"`
	Label        string `json:"label,omitempty"`
	Notes        string `json:"notes,omitempty"`
	ActorID      string `json:"actor_id"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	Active       bool   `json:"active"`
}

// LinkPreference represents an operator preference on a link.
type LinkPreference struct {
	ID         int64  `json:"id"`
	SrcNodeNum int64  `json:"src_node_num"`
	DstNodeNum int64  `json:"dst_node_num"`
	Preference string `json:"preference"` // preferred, avoid, neutral
	Reason     string `json:"reason,omitempty"`
	ActorID    string `json:"actor_id"`
	CreatedAt  string `json:"created_at"`
	Active     bool   `json:"active"`
}

// Recommendation is an actionable topology improvement suggestion.
type Recommendation struct {
	ID            string   `json:"id"`
	Type          string   `json:"type"` // add_node, add_relay, move_node, prefer_anchor, inspect, reduce_trust, isolate, split
	Summary       string   `json:"summary"`
	Confidence    float64  `json:"confidence"`
	Impact        string   `json:"impact"`
	Assumptions   []string `json:"assumptions,omitempty"`
	Evidence      []string `json:"evidence,omitempty"`
	RefuteWith    []string `json:"refute_with,omitempty"` // what data would disprove this
	Basis         string   `json:"basis"`                 // topology, source, policy
	AffectedNodes []int64  `json:"affected_nodes,omitempty"`
}

// RecoveryState tracks MEL startup/shutdown state for crash recovery.
type RecoveryState struct {
	LastCleanShutdownAt string   `json:"last_clean_shutdown_at,omitempty"`
	LastStartupAt       string   `json:"last_startup_at,omitempty"`
	UncleanShutdown     bool     `json:"unclean_shutdown"`
	RecoveredJobs       []string `json:"recovered_jobs,omitempty"`
	PendingActions      []string `json:"pending_actions,omitempty"`
	StartupMode         string   `json:"startup_mode"` // normal, recovery, degraded
}

// StaleThresholds configures when data is considered stale.
type StaleThresholds struct {
	NodeStaleDuration time.Duration
	LinkStaleDuration time.Duration
	ObservationMaxAge time.Duration
}

// DefaultStaleThresholds returns sensible defaults for stale detection.
func DefaultStaleThresholds() StaleThresholds {
	return StaleThresholds{
		NodeStaleDuration: 30 * time.Minute,
		LinkStaleDuration: 30 * time.Minute,
		ObservationMaxAge: 1 * time.Hour,
	}
}
