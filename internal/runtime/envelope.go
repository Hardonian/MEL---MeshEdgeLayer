// Package runtime holds process and product-boundary truth for MEL (single-operator gateway scope; no fleet theatre).
package runtime

import (
	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/fleet"
)

const (
	// ProductName is the public product identifier for APIs and bundles.
	ProductName = "MEL"

	// ProductScopeSingleGateway is the supported deployment posture: one MEL process, one local SQLite, one operator site.
	ProductScopeSingleGateway = "single_gateway_operator"

	// DeploymentModeLocalProcess is the canonical mode string for a single OS process serving the API and transports.
	DeploymentModeLocalProcess = "local_process"

	// NotesHonestBoundary is a fixed operator-facing boundary statement (keep synchronized with docs/architecture where cited).
	NotesHonestBoundary = "MEL is instance-first: one process, one local SQLite per deployment. Core does not ship live federated evidence sync or cross-instance action execution. Operators may import offline remote evidence bundles into the local database for audit (explicit validation posture; not cryptographic proof of origin). Optional scope.* config labels site/fleet boundaries for operator truth and partial-fleet semantics; it does not create fleet-wide authority or global ordering."
)

// TransportKind describes an ingest path kind implemented in this binary (honest capability, not per-deployment config).
type TransportKind struct {
	Kind                 string `json:"kind"`
	IngestImplemented    bool   `json:"ingest_implemented"`
	SendImplemented      bool   `json:"send_implemented"`
	ImplementationStatus string `json:"implementation_status"`
	Notes                string `json:"notes,omitempty"`
}

// ProductEnvelope is the canonical operator-facing capability and scope envelope for this build.
type ProductEnvelope struct {
	ProductName             string          `json:"product_name"`
	ProductScope            string          `json:"product_scope"`
	DeploymentMode          string          `json:"deployment_mode"`
	MultiSiteFleetSupported bool            `json:"multi_site_fleet_supported"`
	Notes                   string          `json:"notes"`
	TransportKinds          []TransportKind `json:"transport_kinds"`
	IntegrationEnabled      bool            `json:"integration_enabled"`
	// CapabilityPosture states federation and cross-instance boundaries honestly (typed).
	CapabilityPosture fleet.CapabilityPosture `json:"capability_posture"`
}

// BuildProductEnvelope returns the fixed envelope for the running binary plus config-derived integration flag.
func BuildProductEnvelope(cfg config.Config) ProductEnvelope {
	integration := cfg.Integration.Enabled
	return ProductEnvelope{
		ProductName:             ProductName,
		ProductScope:            ProductScopeSingleGateway,
		DeploymentMode:          DeploymentModeLocalProcess,
		MultiSiteFleetSupported: false,
		Notes:                   NotesHonestBoundary,
		IntegrationEnabled:      integration,
		CapabilityPosture:       fleet.DefaultCapabilityPosture(),
		TransportKinds: []TransportKind{
			{
				Kind:                 "serial",
				IngestImplemented:    true,
				SendImplemented:      false,
				ImplementationStatus: "supported",
				Notes:                "Direct serial ingest path; hardware verification is deployment-specific.",
			},
			{
				Kind:                 "tcp",
				IngestImplemented:    true,
				SendImplemented:      false,
				ImplementationStatus: "supported",
				Notes:                "Meshtastic TCP device bridge; reachability and TLS are operator responsibilities.",
			},
			{
				Kind:                 "mqtt",
				IngestImplemented:    true,
				SendImplemented:      false,
				ImplementationStatus: "supported",
				Notes:                "Subscribe-oriented ingest with broker QoS; MEL does not claim full remote config/publish control in core scope.",
			},
			{
				Kind:                 "serialtcp",
				IngestImplemented:    true,
				SendImplemented:      false,
				ImplementationStatus: "supported",
				Notes:                "Combined serial-over-TCP path when configured; same honesty rules as tcp/serial.",
			},
		},
	}
}
