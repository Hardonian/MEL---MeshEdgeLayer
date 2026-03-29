package service

import (
	"testing"
	"time"

	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/models"
)

func TestIncidentIntelligence_WirelessContext_WifiBackhaulClassification(t *testing.T) {
	a := newSoDTestApp(t)
	now := time.Now().UTC()
	if err := a.DB.UpsertIncident(models.Incident{
		ID:           "inc-wifi",
		Category:     "transport",
		Severity:     "warning",
		Title:        "Backhaul instability",
		Summary:      "Wi-Fi backhaul disconnects observed near mesh gateway.",
		ResourceType: "transport",
		ResourceID:   "mqtt-sod",
		State:        "open",
		OccurredAt:   now.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.DB.UpsertTransportAlert(db.TransportAlertRecord{
		ID:               "alert-wifi",
		TransportName:    "mqtt-sod",
		TransportType:    "mqtt",
		Severity:         "warning",
		Reason:           "disconnect",
		Summary:          "backhaul disconnect",
		FirstTriggeredAt: now.Add(-5 * time.Minute).Format(time.RFC3339),
		LastUpdatedAt:    now.Add(1 * time.Minute).Format(time.RFC3339),
		Active:           true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.DB.InsertDeadLetter(db.DeadLetter{TransportName: "mqtt-sod", TransportType: "mqtt", Topic: "msh/test", Reason: "timeout"}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := a.IncidentByID("inc-wifi")
	if err != nil || !ok || got.Intelligence == nil || got.Intelligence.WirelessContext == nil {
		t.Fatalf("incident wireless context: ok=%v err=%v", ok, err)
	}
	ctx := got.Intelligence.WirelessContext
	if ctx.Classification != "wifi_backhaul_instability" {
		t.Fatalf("classification=%q", ctx.Classification)
	}
	if ctx.PrimaryDomain != "wifi" {
		t.Fatalf("primary_domain=%q", ctx.PrimaryDomain)
	}
}

func TestIncidentIntelligence_WirelessContext_UnsupportedBluetooth(t *testing.T) {
	a := newSoDTestApp(t)
	now := time.Now().UTC()
	if err := a.DB.UpsertIncident(models.Incident{
		ID:           "inc-bt",
		Category:     "provisioning",
		Severity:     "warning",
		Title:        "Bluetooth onboarding issue",
		Summary:      "Nearby Bluetooth provisioning failed repeatedly.",
		ResourceType: "transport",
		ResourceID:   "mqtt-sod",
		State:        "open",
		OccurredAt:   now.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.DB.UpsertTransportAlert(db.TransportAlertRecord{
		ID:               "alert-bt",
		TransportName:    "mqtt-sod",
		TransportType:    "mqtt",
		Severity:         "warning",
		Reason:           "disconnect",
		Summary:          "transport disconnect near onboarding window",
		FirstTriggeredAt: now.Add(-4 * time.Minute).Format(time.RFC3339),
		LastUpdatedAt:    now.Add(2 * time.Minute).Format(time.RFC3339),
		Active:           true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.DB.InsertDeadLetter(db.DeadLetter{TransportName: "mqtt-sod", TransportType: "mqtt", Topic: "msh/test", Reason: "decode_failure"}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := a.IncidentByID("inc-bt")
	if err != nil || !ok || got.Intelligence == nil || got.Intelligence.WirelessContext == nil {
		t.Fatalf("incident wireless context: ok=%v err=%v", ok, err)
	}
	ctx := got.Intelligence.WirelessContext
	if ctx.Classification != "unsupported_wireless_domain_observed" {
		t.Fatalf("classification=%q", ctx.Classification)
	}
	if len(ctx.Unsupported) == 0 || ctx.Unsupported[0].Domain != "bluetooth" {
		t.Fatalf("unsupported=%+v", ctx.Unsupported)
	}
}

func TestIncidentIntelligence_WirelessContext_MixedPathClassification(t *testing.T) {
	a := newSoDTestApp(t)
	now := time.Now().UTC()
	if err := a.DB.UpsertIncident(models.Incident{
		ID:           "inc-mixed",
		Category:     "mesh_topology",
		Severity:     "warning",
		Title:        "Mixed LoRa and Wi-Fi degradation",
		Summary:      "LoRa link drop observed with Wi-Fi backhaul flapping.",
		ResourceType: "transport",
		ResourceID:   "mqtt-sod",
		State:        "open",
		OccurredAt:   now.Format(time.RFC3339),
		Metadata:     map[string]any{"frequency": "915"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.DB.UpsertTransportAlert(db.TransportAlertRecord{
		ID:               "alert-mixed",
		TransportName:    "mqtt-sod",
		TransportType:    "mqtt",
		Severity:         "critical",
		Reason:           "disconnect",
		Summary:          "wifi backhaul loss",
		FirstTriggeredAt: now.Add(-3 * time.Minute).Format(time.RFC3339),
		LastUpdatedAt:    now.Add(2 * time.Minute).Format(time.RFC3339),
		Active:           true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.DB.InsertDeadLetter(db.DeadLetter{TransportName: "mqtt-sod", TransportType: "mqtt", Topic: "msh/test", Reason: "timeout"}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := a.IncidentByID("inc-mixed")
	if err != nil || !ok || got.Intelligence == nil || got.Intelligence.WirelessContext == nil {
		t.Fatalf("incident wireless context: ok=%v err=%v", ok, err)
	}
	ctx := got.Intelligence.WirelessContext
	if ctx.Classification != "mixed_path_degradation" {
		t.Fatalf("classification=%q", ctx.Classification)
	}
	if ctx.PrimaryDomain != "mixed" {
		t.Fatalf("primary_domain=%q", ctx.PrimaryDomain)
	}
}

func TestAssembleProofpack_EmbedsWirelessContext(t *testing.T) {
	a := newSoDTestApp(t)
	now := time.Now().UTC()
	if err := a.DB.UpsertIncident(models.Incident{
		ID:           "inc-proofpack-wireless",
		Category:     "transport",
		Severity:     "warning",
		Title:        "Wi-Fi backhaul warning",
		Summary:      "Wi-Fi backhaul instability in node uplink.",
		ResourceType: "transport",
		ResourceID:   "mqtt-sod",
		State:        "open",
		OccurredAt:   now.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.DB.UpsertTransportAlert(db.TransportAlertRecord{
		ID:               "alert-proofpack",
		TransportName:    "mqtt-sod",
		TransportType:    "mqtt",
		Severity:         "warning",
		Reason:           "disconnect",
		Summary:          "wifi backhaul unstable",
		FirstTriggeredAt: now.Add(-5 * time.Minute).Format(time.RFC3339),
		LastUpdatedAt:    now.Add(2 * time.Minute).Format(time.RFC3339),
		Active:           true,
	}); err != nil {
		t.Fatal(err)
	}
	pack, err := a.AssembleProofpack("inc-proofpack-wireless", "operator-a")
	if err != nil {
		t.Fatalf("AssembleProofpack: %v", err)
	}
	incidentRaw, ok := pack["incident"].(map[string]any)
	if !ok {
		t.Fatalf("incident payload missing")
	}
	ctx, ok := incidentRaw["wireless_context"].(map[string]any)
	if !ok {
		t.Fatalf("wireless_context missing from proofpack incident: %+v", incidentRaw)
	}
	if ctx["classification"] != "wifi_backhaul_instability" {
		t.Fatalf("classification=%v", ctx["classification"])
	}
}
