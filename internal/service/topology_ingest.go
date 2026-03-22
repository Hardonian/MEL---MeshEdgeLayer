package service

import (
	"fmt"
	"time"

	"github.com/mel-project/mel/internal/meshtastic"
	"github.com/mel-project/mel/internal/topology"
)

// recordTopologyObservation creates a node observation and infers topology links
// from a successfully ingested Meshtastic packet. Called from ingest() after
// PersistIngest succeeds and the message is not a duplicate.
func (a *App) recordTopologyObservation(env meshtastic.Envelope, transportName string) {
	if a.topoStore == nil || !a.Cfg.Topology.Enabled {
		return
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	// Determine source type and trust level from transport config
	trustClass := a.Cfg.Topology.DefaultTrustClass
	sourceType := "broker"
	for _, tc := range a.Cfg.Transports {
		if tc.Name == transportName {
			if tc.TrustClass != "" {
				trustClass = tc.TrustClass
			}
			if tc.Type == "serial" || tc.Type == "tcp" || tc.Type == "serialtcp" {
				sourceType = "direct"
			}
			break
		}
	}
	trustLevel := trustClassToLevel(trustClass)

	fromNode := int64(env.Packet.From)
	gatewayNode := int64(0)
	if env.GatewayID != "" {
		// Gateway ID is typically a hex node ID like !aabbccdd
		// We record it as a string-based observation, not a node_num link
		gatewayNode = meshtastic.GatewayIDToNum(env.GatewayID)
	}

	// 1. Record node observation for the from_node
	obs := topology.NodeObservation{
		NodeNum:     fromNode,
		ConnectorID: transportName,
		SourceType:  sourceType,
		TrustLevel:  trustLevel,
		ObservedAt:  nowStr,
		SNR:         float64(env.Packet.RXSNR),
		RSSI:        int64(env.Packet.RXRSSI),
		HopCount:    int(env.Packet.HopLimit),
		GatewayID:   env.GatewayID,
	}
	if env.Packet.Lat != nil {
		obs.Lat = *env.Packet.Lat
	}
	if env.Packet.Lon != nil {
		obs.Lon = *env.Packet.Lon
	}
	obs.Altitude = int64(env.Packet.Altitude)
	if sourceType == "broker" {
		obs.ViaMQTT = true
	}

	if err := a.topoStore.InsertObservation(obs); err != nil {
		a.Log.Error("topology_observation_failed", "failed to record node observation", map[string]any{
			"node_num": fromNode, "error": err.Error(),
		})
	}

	// 2. Infer topology link: from_node → gateway (if gateway is known and different)
	if gatewayNode != 0 && gatewayNode != fromNode {
		a.upsertInferredLink(fromNode, gatewayNode, transportName, trustLevel, nowStr, false)
	}

	// 3. Infer topology link: from_node → to_node (if to_node is a specific node, not broadcast)
	toNode := int64(env.Packet.To)
	if toNode != 0 && toNode != fromNode && toNode != 0xFFFFFFFF {
		a.upsertInferredLink(fromNode, toNode, transportName, trustLevel, nowStr, true)
	}

	// 4. Update node trust metadata
	if sourceType == "direct" {
		// Update last_direct_seen_at
		_ = a.topoStore.DB.Exec(fmt.Sprintf(
			`UPDATE nodes SET last_direct_seen_at='%s', trust_class='%s', source_connector_id='%s' WHERE node_num=%d;`,
			nowStr, trustClass, transportName, fromNode))
	} else {
		_ = a.topoStore.DB.Exec(fmt.Sprintf(
			`UPDATE nodes SET last_broker_seen_at='%s', source_connector_id='%s' WHERE node_num=%d AND (trust_class IS NULL OR trust_class='unknown');`,
			nowStr, transportName, fromNode))
	}

	// 5. Update source trust record for the connector
	_ = a.topoStore.UpsertSourceTrust(topology.SourceTrust{
		ConnectorID:   transportName,
		ConnectorName: transportName,
		ConnectorType: sourceType,
		TrustClass:    topology.TrustClass(trustClass),
		TrustLevel:    trustLevel,
		FirstSeenAt:   nowStr,
	})
}

// upsertInferredLink creates or updates an inferred topology link.
func (a *App) upsertInferredLink(srcNode, dstNode int64, transportName string, trustLevel float64, nowStr string, directional bool) {
	// Canonical edge ordering: smaller node_num first for undirected links
	src, dst := srcNode, dstNode
	if !directional && src > dst {
		src, dst = dst, src
	}

	edgeID := fmt.Sprintf("%d-%d-%s", src, dst, transportName)
	link := topology.Link{
		EdgeID:           edgeID,
		SrcNodeNum:       src,
		DstNodeNum:       dst,
		Observed:         true,
		Directional:      directional,
		TransportPath:    transportName,
		FirstObservedAt:  nowStr,
		LastObservedAt:   nowStr,
		QualityScore:     0.5, // default; rescored by snapshot worker
		Reliability:      0.5,
		SourceTrustLevel: trustLevel,
		SourceConnectorID: transportName,
		ObservationCount: 1,
	}

	if err := a.topoStore.UpsertLink(link); err != nil {
		a.Log.Error("topology_link_failed", "failed to upsert topology link", map[string]any{
			"src": src, "dst": dst, "error": err.Error(),
		})
	}
}

// trustClassToLevel converts a trust class string to a numeric trust level [0,1].
func trustClassToLevel(tc string) float64 {
	switch tc {
	case "direct_local":
		return 1.0
	case "trusted":
		return 0.8
	case "partial":
		return 0.5
	case "untrusted":
		return 0.2
	default:
		return 0.3
	}
}
