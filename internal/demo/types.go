// Package demo provides deterministic, sandbox-only deployment fixtures and
// scenario metadata for field-style validation. It does not ship mock mesh
// traffic: seeded rows are written through the same persistence paths as live
// ingest (messages, nodes, transport_alerts, incidents, dead_letters).
package demo

import "time"

// DemoDeploymentProfile names a target environment shape (single hub, MQTT
// bridge present, dual path, etc.).
type DemoDeploymentProfile string

const (
	ProfilePrivateRFOnly     DemoDeploymentProfile = "private_rf_only"
	ProfileRFPlusMQTTBridge  DemoDeploymentProfile = "rf_plus_mqtt_bridge"
	ProfileDualMQTTIngest    DemoDeploymentProfile = "dual_mqtt_ingest"
	ProfileStoreForwardRelay DemoDeploymentProfile = "store_forward_relay"
)

// DemoNodeProfile describes a synthetic node stance for commissioning stories.
type DemoNodeProfile struct {
	NodeNum   int64   `json:"node_num"`
	NodeID    string  `json:"node_id"`
	LongName  string  `json:"long_name"`
	ShortName string  `json:"short_name"`
	Role      string  `json:"role"` // operator narrative only
	LastSNR   float64 `json:"last_snr"`
	LastRSSI  int64   `json:"last_rssi"`
	GatewayID string  `json:"gateway_id"`
	AltitudeM int64   `json:"altitude_m"`
}

// DemoBridgeProfile describes an MQTT uplink/downlink path in fixture configs.
type DemoBridgeProfile struct {
	Name       string `json:"name"`
	Endpoint   string `json:"endpoint"`
	Topic      string `json:"topic"`
	ClientID   string `json:"client_id"`
	TLSEnabled bool   `json:"tls_enabled"` // narrative: broker uses TLS or not
	Notes      string `json:"notes"`
}

// DemoScenarioClass groups scenarios by failure / health theme.
type DemoScenarioClass string

const (
	ClassHealthy              DemoScenarioClass = "healthy"
	ClassRFPerformance        DemoScenarioClass = "rf_performance"
	ClassRoleMisconfiguration DemoScenarioClass = "role_misconfiguration"
	ClassMQTTPrivacy          DemoScenarioClass = "mqtt_privacy"
	ClassDuplicatePath        DemoScenarioClass = "duplicate_path"
	ClassStoreForward         DemoScenarioClass = "store_forward"
)

// DemoScenario is a named, replayable seed bundle (deterministic clock).
type DemoScenario struct {
	ID                string                `json:"id"`
	Title             string                `json:"title"`
	Summary           string                `json:"summary"`
	Class             DemoScenarioClass     `json:"class"`
	Profile           DemoDeploymentProfile `json:"deployment_profile"`
	Nodes             []DemoNodeProfile     `json:"nodes"`
	Bridges           []DemoBridgeProfile   `json:"bridges,omitempty"`
	OperatorNarrative string                `json:"operator_narrative"`
}

// DemoScenarioEvent is a single step in a drill or walkthrough (documentation
// and automation share this schema).
type DemoScenarioEvent struct {
	Step        int    `json:"step"`
	Title       string `json:"title"`
	CLI         string `json:"cli,omitempty"`
	API         string `json:"api,omitempty"`
	Expectation string `json:"expectation"`
}

// DemoScenarioReplay lists ordered events for an operator or script.
type DemoScenarioReplay struct {
	ScenarioID string              `json:"scenario_id"`
	Events     []DemoScenarioEvent `json:"events"`
}

// DemoEvidenceBundle is written after seeding; paths are relative to OutDir.
type DemoEvidenceBundle struct {
	GeneratedAt       time.Time `json:"generated_at"`
	ScenarioID        string    `json:"scenario_id"`
	ConfigPath        string    `json:"config_path"`
	DatabasePath      string    `json:"database_path"`
	EvidenceDir       string    `json:"evidence_dir,omitempty"`
	CLIOutputs        []string  `json:"cli_outputs,omitempty"`
	ManifestVersion   string    `json:"manifest_version"`
	SandboxMarkerNote string    `json:"sandbox_marker_note"`
}
