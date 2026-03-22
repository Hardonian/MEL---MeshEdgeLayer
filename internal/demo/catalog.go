package demo

import "slices"

// Scenarios returns the canonical demo scenario catalog (deterministic IDs).
func Scenarios() []DemoScenario {
	return []DemoScenario{
		{
			ID:      "healthy-private-mesh",
			Title:   "Healthy private mesh (RF-only)",
			Summary: "Small private mesh with sane roles, single serial ingest, no MQTT bridge.",
			Class:   ClassHealthy,
			Profile: ProfilePrivateRFOnly,
			Nodes: []DemoNodeProfile{
				{NodeNum: 0x1001, NodeID: "!1001", LongName: "Rooftop Relay", ShortName: "RR1", Role: "relay", LastSNR: 9.5, LastRSSI: -65, GatewayID: "local-serial", AltitudeM: 18},
				{NodeNum: 0x1002, NodeID: "!1002", LongName: "Indoor Handset", ShortName: "HH1", Role: "client", LastSNR: 6.2, LastRSSI: -78, GatewayID: "!1001", AltitudeM: 3},
				{NodeNum: 0x1003, NodeID: "!1003", LongName: "Field Node", ShortName: "FN1", Role: "client", LastSNR: 4.1, LastRSSI: -88, GatewayID: "!1001", AltitudeM: 120},
			},
			Bridges:           nil,
			OperatorNarrative: "All traffic arrives on one trusted RF path; mesh drilldown should show stable scores without duplicate-path warnings.",
		},
		{
			ID:      "indoor-gateway-vs-rooftop",
			Title:   "Indoor gateway underperforming vs rooftop relay",
			Summary: "Same mesh footprint but indoor gateway shows weak RF versus elevated relay.",
			Class:   ClassRFPerformance,
			Profile: ProfilePrivateRFOnly,
			Nodes: []DemoNodeProfile{
				{NodeNum: 0x2001, NodeID: "!2001", LongName: "Desk Gateway", ShortName: "GW0", Role: "gateway", LastSNR: 1.2, LastRSSI: -102, GatewayID: "local-serial", AltitudeM: 2},
				{NodeNum: 0x2002, NodeID: "!2002", LongName: "Rooftop Candidate", ShortName: "RRX", Role: "relay_candidate", LastSNR: 11.0, LastRSSI: -58, GatewayID: "local-serial", AltitudeM: 22},
				{NodeNum: 0x2003, NodeID: "!2003", LongName: "Patrol Handheld", ShortName: "PT1", Role: "client", LastSNR: 3.0, LastRSSI: -95, GatewayID: "!2002", AltitudeM: 1},
			},
			OperatorNarrative: "Compare last_snr/last_rssi and hop patterns; operators relocate the primary gateway or add the rooftop relay as backbone.",
		},
		{
			ID:      "handheld-as-backbone",
			Title:   "Handheld misconfigured as backbone router",
			Summary: "A handheld-class node is carrying disproportionate relay load.",
			Class:   ClassRoleMisconfiguration,
			Profile: ProfilePrivateRFOnly,
			Nodes: []DemoNodeProfile{
				{NodeNum: 0x3001, NodeID: "!3001", LongName: "Intended Relay", ShortName: "RL1", Role: "relay", LastSNR: 8.0, LastRSSI: -72, GatewayID: "local-serial", AltitudeM: 15},
				{NodeNum: 0x3002, NodeID: "!3002", LongName: "Handheld Router", ShortName: "HR1", Role: "misconfigured_router", LastSNR: 2.1, LastRSSI: -99, GatewayID: "!3001", AltitudeM: 1},
				{NodeNum: 0x3003, NodeID: "!3003", LongName: "Remote Sensor", ShortName: "SN1", Role: "sensor", LastSNR: 1.5, LastRSSI: -105, GatewayID: "!3002", AltitudeM: 50},
			},
			OperatorNarrative: "Relay hop counts and gateway_id chains show the handheld as an unintended hop; fix device role / antenna placement on the intended relay.",
		},
		{
			ID:      "mqtt-privacy-json-risk",
			Title:   "MQTT bridge privacy / JSON exposure risk",
			Summary: "MQTT path ingests with map reporting enabled and a wide topic filter — privacy audit and incidents reflect the posture.",
			Class:   ClassMQTTPrivacy,
			Profile: ProfileRFPlusMQTTBridge,
			Nodes: []DemoNodeProfile{
				{NodeNum: 0x4001, NodeID: "!4001", LongName: "MQTT Edge", ShortName: "MQ1", Role: "bridge_edge", LastSNR: 0, LastRSSI: -80, GatewayID: "mqtt-bridge", AltitudeM: 0},
			},
			Bridges: []DemoBridgeProfile{
				{Name: "mqtt-bridge", Endpoint: "127.0.0.1:1883", Topic: "msh/US/#", ClientID: "mel-demo-bridge", TLSEnabled: false, Notes: "Wide filter + cleartext broker: operator must verify channel scope and TLS."},
			},
			OperatorNarrative: "Run mel privacy audit and inspect mesh; expect map-reporting and MQTT encryption lints when replaying this fixture config.",
		},
		{
			ID:      "rf-mqtt-duplicate-path",
			Title:   "RF + MQTT duplicate path contamination",
			Summary: "Same logical packet identity may appear on RF and MQTT; fixture seeds correlated transport stress.",
			Class:   ClassDuplicatePath,
			Profile: ProfileDualMQTTIngest,
			Nodes: []DemoNodeProfile{
				{NodeNum: 0x5001, NodeID: "!5001", LongName: "Shared Node A", ShortName: "SNA", Role: "dual_seen", LastSNR: 5.0, LastRSSI: -82, GatewayID: "rf-serial", AltitudeM: 5},
				{NodeNum: 0x5002, NodeID: "!5002", LongName: "Shared Node B", ShortName: "SNB", Role: "dual_seen", LastSNR: 4.5, LastRSSI: -85, GatewayID: "mqtt-uplink", AltitudeM: 5},
			},
			Bridges: []DemoBridgeProfile{
				{Name: "mqtt-uplink", Endpoint: "127.0.0.1:1883", Topic: "msh/US/2/e/DemoNet/#", ClientID: "mel-demo-uplink", TLSEnabled: true, Notes: "Second path alongside RF serial."},
			},
			OperatorNarrative: "Dual transports enabled; operators verify dedupe_hash behavior and avoid conflicting control plane assumptions.",
		},
		{
			ID:      "store-and-forward",
			Title:   "Store-and-forward relay pressure",
			Summary: "Relay shows high observation drops and backlog-style signals while still receiving some frames.",
			Class:   ClassStoreForward,
			Profile: ProfileStoreForwardRelay,
			Nodes: []DemoNodeProfile{
				{NodeNum: 0x6001, NodeID: "!6001", LongName: "Busy Relay", ShortName: "BR1", Role: "store_forward", LastSNR: 3.5, LastRSSI: -90, GatewayID: "local-serial", AltitudeM: 8},
				{NodeNum: 0x6002, NodeID: "!6002", LongName: "Deferred Client", ShortName: "DC1", Role: "client", LastSNR: 2.0, LastRSSI: -100, GatewayID: "!6001", AltitudeM: 2},
			},
			OperatorNarrative: "Mesh drilldown should surface observation drops / saturation-style guidance; reduce airtime or add relay capacity.",
		},
	}
}

// ReplayFor returns a suggested CLI/API walkthrough for a scenario.
func ReplayFor(id string) (DemoScenarioReplay, bool) {
	if ScenarioByID(id) == nil {
		return DemoScenarioReplay{}, false
	}
	switch id {
	case "healthy-private-mesh":
		return DemoScenarioReplay{
			ScenarioID: id,
			Events: []DemoScenarioEvent{
				{Step: 1, Title: "Validate sandbox config", CLI: "mel config validate --config <cfg>", Expectation: "Config valid; note demo_sandbox marker in storage path."},
				{Step: 2, Title: "Doctor", CLI: "mel doctor --config <cfg> --json", Expectation: "Structured findings; transports idle without live broker."},
				{Step: 3, Title: "Mesh reality", CLI: "mel inspect mesh --config <cfg>", Expectation: "Non-empty nodes; mesh_health reflects seeded evidence."},
				{Step: 4, Title: "Optional API", API: "GET /api/v1/status", Expectation: "JSON status includes transport summaries consistent with DB."},
			},
		}, true
	case "mqtt-privacy-json-risk":
		return DemoScenarioReplay{
			ScenarioID: id,
			Events: []DemoScenarioEvent{
				{Step: 1, Title: "Privacy audit", CLI: "mel privacy audit --config <cfg> --format json", Expectation: "Findings include map reporting / MQTT encryption posture."},
				{Step: 2, Title: "Incidents surface", API: "GET /api/v1/incidents", Expectation: "Seeded security incident visible when daemon runs."},
			},
		}, true
	default:
		return DemoScenarioReplay{
			ScenarioID: id,
			Events: []DemoScenarioEvent{
				{Step: 1, Title: "Seed and validate", CLI: "mel demo seed --scenario <id> --config <cfg>", Expectation: "Seed applied; evidence manifest written."},
				{Step: 2, Title: "Mesh drilldown", CLI: "mel inspect mesh --config <cfg>", Expectation: "Correlations match scenario class."},
			},
		}, true
	}
}

// ScenarioByID returns a pointer to the catalog entry or nil.
func ScenarioByID(id string) *DemoScenario {
	list := Scenarios()
	for i := range list {
		if list[i].ID == id {
			s := list[i]
			return &s
		}
	}
	return nil
}

// ScenarioIDs returns sorted scenario ids for CLI completion / tests.
func ScenarioIDs() []string {
	list := Scenarios()
	ids := make([]string, 0, len(list))
	for _, s := range list {
		ids = append(ids, s.ID)
	}
	slices.Sort(ids)
	return ids
}
