package runtime

import (
	"testing"

	"github.com/mel-project/mel/internal/config"
)

func TestBuildProductEnvelopeHonestScope(t *testing.T) {
	cfg := config.Default()
	e := BuildProductEnvelope(cfg)
	if e.ProductName != ProductName {
		t.Fatalf("product name: got %q", e.ProductName)
	}
	if e.ProductScope != ProductScopeSingleGateway {
		t.Fatalf("scope: got %q want %q", e.ProductScope, ProductScopeSingleGateway)
	}
	if e.MultiSiteFleetSupported {
		t.Fatal("multi_site_fleet_supported must be false for honest single-gateway posture")
	}
	if e.CapabilityPosture.FederationMode == "" {
		t.Fatal("expected capability_posture.federation_mode")
	}
	if e.CapabilityPosture.FleetAggregationSupported {
		t.Fatal("fleet_aggregation_supported must be false in core")
	}
	if len(e.TransportKinds) < 3 {
		t.Fatalf("expected transport kinds list, got %d", len(e.TransportKinds))
	}
}
