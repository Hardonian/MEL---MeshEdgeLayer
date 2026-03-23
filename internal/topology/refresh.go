package topology

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
)

// MeshtasticBroadcastDest is the conventional broadcast destination in mesh packets.
const MeshtasticBroadcastDest uint32 = 0xffffffff

// RefreshStale marks stale flags on nodes and links from thresholds.
func (s *Store) RefreshStale(thresholds StaleThresholds) error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.MarkStaleness(thresholds)
}

// RefreshScores recomputes analysis from current DB rows, persists per-node and per-link scores, and returns the result.
func (s *Store) RefreshScores(thresholds StaleThresholds, now time.Time) (AnalysisResult, error) {
	var empty AnalysisResult
	if s == nil || s.DB == nil {
		return empty, fmt.Errorf("topology store unavailable")
	}
	nodes, err := s.ListNodes(10000)
	if err != nil {
		return empty, err
	}
	links, err := s.ListLinks(20000)
	if err != nil {
		return empty, err
	}
	result := Analyze(nodes, links, thresholds, now)
	for _, n := range result.ScoredNodes {
		staleFlag := n.Stale
		if t := parseRFC3339OrZero(n.LastSeenAt); !t.IsZero() && now.Sub(t) > thresholds.NodeStaleDuration {
			staleFlag = true
		}
		_ = s.UpdateNodeHealth(n.NodeNum, n.HealthScore, n.HealthState, n.HealthFactors, staleFlag)
	}
	for _, l := range result.ScoredLinks {
		_ = s.UpdateLinkQuality(l.EdgeID, l.QualityScore, l.QualityFactors, l.Stale)
	}
	return result, nil
}

func parseRFC3339OrZero(s string) time.Time {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(s))
	if err != nil {
		return time.Time{}
	}
	return t
}

// PruneObservations keeps at most maxPerNode newest rows per node_num.
func (s *Store) PruneObservations(maxPerNode int) error {
	if s == nil || s.DB == nil || maxPerNode <= 0 {
		return nil
	}
	sql := fmt.Sprintf(`
DELETE FROM node_observations WHERE id IN (
  SELECT o1.id FROM node_observations o1
  WHERE (
    SELECT COUNT(*) FROM node_observations o2
    WHERE o2.node_num = o1.node_num
      AND (o2.observed_at > o1.observed_at OR (o2.observed_at = o1.observed_at AND o2.rowid >= o1.rowid))
  ) > %d
);`, maxPerNode)
	return s.DB.Exec(sql)
}

func parseMapEligible(explanation []string) int {
	for _, line := range explanation {
		if strings.HasPrefix(line, "map_eligible_nodes=") {
			var n int
			_, _ = fmt.Sscanf(strings.TrimPrefix(line, "map_eligible_nodes="), "%d", &n)
			return n
		}
	}
	return 0
}

// BuildIntelligenceView assembles the operator summary for GET /api/v1/topology.
func BuildIntelligenceView(cfg config.Config, analysis AnalysisResult, transportConnected bool, now time.Time) map[string]any {
	snap := analysis.Snapshot
	mapEligible := parseMapEligible(snap.Explanation)
	viewMode := "graph"
	if cfg.Privacy.MapReportingAllowed && mapEligible > 0 && snap.NodeCount > 0 {
		if mapEligible >= snap.NodeCount {
			viewMode = "map"
		} else {
			viewMode = "map_partial"
		}
	}
	return map[string]any{
		"generated_at":                  now.UTC().Format(time.RFC3339),
		"topology_enabled":              cfg.Topology.Enabled,
		"view_mode":                     viewMode,
		"privacy_map_reporting_allowed": cfg.Privacy.MapReportingAllowed,
		"redact_exports":                cfg.Privacy.RedactExports,
		"node_count":                    snap.NodeCount,
		"link_count":                    snap.EdgeCount,
		"map_eligible_node_count":       mapEligible,
		"transport_connected":           transportConnected,
		"analysis":                    analysis,
		"confidence_summary":          snap.ConfidenceSummary,
		"staleness": map[string]any{
			"node_stale_minutes": cfg.Topology.NodeStaleMinutes,
			"link_stale_minutes": cfg.Topology.LinkStaleMinutes,
		},
		"evidence_model": "Links are derived from ingested mesh packets (relay_node and unicast to_node fields). This is packet-level evidence, not verified RF adjacency.",
	}
}

// UpdateLinkQuality persists recomputed link score factors.
func (s *Store) UpdateLinkQuality(edgeID string, quality float64, factors []ScoreFactor, stale bool) error {
	if s == nil || s.DB == nil {
		return nil
	}
	factorsJSON := "[]"
	if len(factors) > 0 {
		b, _ := json.Marshal(factors)
		factorsJSON = string(b)
	}
	st := 0
	if stale {
		st = 1
	}
	sql := fmt.Sprintf(`UPDATE topology_links SET quality_score=%f, quality_factors_json='%s', stale=%d, updated_at='%s' WHERE edge_id='%s';`,
		quality, db.EscString(factorsJSON), st, time.Now().UTC().Format(time.RFC3339), db.EscString(edgeID))
	return s.DB.Exec(sql)
}

// StableEdgeID returns a deterministic edge id from components.
func StableEdgeID(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		fmt.Fprint(h, p)
		fmt.Fprint(h, "\x00")
	}
	sum := h.Sum(nil)
	return fmt.Sprintf("e%x", sum[:12])
}

// ApplyPacketEvidence updates observations and links from one ingested mesh packet (after deduped message insert).
func (s *Store) ApplyPacketEvidence(cfg config.Config, transportName, transportType string, fromNode int64, toNode uint32, relayNode uint32, rxTime string, gatewayID string, snr float64, rssi int64, hopLimit uint32, lat, lon float64, altitude int64) error {
	if s == nil || s.DB == nil || !cfg.Topology.Enabled {
		return nil
	}
	tc := findTransportCfg(cfg, transportName)
	trust := trustLevelForTransport(tc, cfg)
	trustClass := topologyTrustClass(tc)
	connectorType := connectorTypeForTransport(transportType)
	_ = s.UpsertSourceTrust(SourceTrust{
		ConnectorID:        transportName,
		ConnectorName:      transportName,
		ConnectorType:      connectorType,
		TrustClass:         trustClass,
		TrustLevel:         trust,
		FirstSeenAt:        rxTime,
		ObservationCount:   1,
		ContradictionCount: 0,
		StaleCount:         0,
	})
	srcType := "broker"
	if transportType == "serial" || transportType == "tcp" || transportType == "serialtcp" {
		srcType = "direct"
	}
	viaMQTT := transportType == "mqtt"
	obs := NodeObservation{
		NodeNum:     fromNode,
		ConnectorID: transportName,
		SourceType:  srcType,
		TrustLevel:  trust,
		ObservedAt:  rxTime,
		SNR:         snr,
		RSSI:        rssi,
		Lat:         lat,
		Lon:         lon,
		Altitude:    altitude,
		HopCount:    int(hopLimit),
		ViaMQTT:     viaMQTT,
		GatewayID:   gatewayID,
	}
	if err := s.InsertObservation(obs); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	locState := ""
	locSQL := ""
	if lat != 0 || lon != 0 {
		if cfg.Privacy.StorePrecisePositions {
			locState = string(LocExact)
		} else {
			locState = string(LocApproximate)
		}
		locSQL = fmt.Sprintf(", location_state='%s'", db.EscString(locState))
	}
	if viaMQTT {
		_ = s.DB.Exec(fmt.Sprintf(`UPDATE nodes SET source_connector_id='%s', trust_class='%s', last_broker_seen_at='%s', first_seen_at=CASE WHEN first_seen_at IS NULL OR first_seen_at='' THEN '%s' ELSE first_seen_at END%s, updated_at='%s' WHERE node_num=%d;`,
			db.EscString(transportName), db.EscString(string(trustClass)), db.EscString(rxTime), db.EscString(rxTime), locSQL, now, fromNode))
	} else {
		_ = s.DB.Exec(fmt.Sprintf(`UPDATE nodes SET source_connector_id='%s', trust_class='%s', last_direct_seen_at='%s', first_seen_at=CASE WHEN first_seen_at IS NULL OR first_seen_at='' THEN '%s' ELSE first_seen_at END%s, updated_at='%s' WHERE node_num=%d;`,
			db.EscString(transportName), db.EscString(string(trustClass)), db.EscString(rxTime), db.EscString(rxTime), locSQL, now, fromNode))
	}

	if relayNode != 0 && int64(relayNode) != fromNode {
		eid := StableEdgeID("relay", transportName, fmt.Sprint(relayNode), fmt.Sprint(fromNode))
		lnk := Link{
			EdgeID:            eid,
			SrcNodeNum:        int64(relayNode),
			DstNodeNum:        fromNode,
			Observed:          true,
			Directional:       true,
			TransportPath:     transportName,
			FirstObservedAt:   rxTime,
			LastObservedAt:    rxTime,
			QualityScore:      0.5,
			Reliability:       0.5,
			SourceTrustLevel:  trust,
			SourceConnectorID: transportName,
			RelayDependent:    true,
			ObservationCount:  1,
		}
		if err := s.UpsertLink(lnk); err != nil {
			return err
		}
	}
	if toNode != 0 && toNode != MeshtasticBroadcastDest && int64(toNode) != fromNode {
		eid := StableEdgeID("dst", transportName, fmt.Sprint(fromNode), fmt.Sprint(toNode))
		lnk := Link{
			EdgeID:            eid,
			SrcNodeNum:        fromNode,
			DstNodeNum:        int64(toNode),
			Observed:          true,
			Directional:       true,
			TransportPath:     transportName,
			FirstObservedAt:   rxTime,
			LastObservedAt:    rxTime,
			QualityScore:      0.5,
			Reliability:       0.5,
			SourceTrustLevel:  trust,
			SourceConnectorID: transportName,
			RelayDependent:    relayNode != 0,
			ObservationCount:  1,
		}
		if err := s.UpsertLink(lnk); err != nil {
			return err
		}
	}
	return nil
}

func findTransportCfg(cfg config.Config, name string) config.TransportConfig {
	for _, t := range cfg.Transports {
		if t.Name == name {
			return t
		}
	}
	return config.TransportConfig{Name: name, Type: "unknown"}
}

func trustLevelForTransport(tc config.TransportConfig, cfg config.Config) float64 {
	switch TrustClass(strings.TrimSpace(tc.TrustClass)) {
	case TrustDirectLocal:
		return 1.0
	case TrustTrusted:
		return 0.85
	case TrustPartial:
		return 0.5
	case TrustUntrusted:
		return 0.15
	default:
		return trustClassToValue(TrustClass(strings.TrimSpace(cfg.Topology.DefaultTrustClass)))
	}
}

func topologyTrustClass(tc config.TransportConfig) TrustClass {
	s := strings.TrimSpace(tc.TrustClass)
	if s == "" {
		return TrustUnknown
	}
	return TrustClass(s)
}

func connectorTypeForTransport(transportType string) string {
	switch transportType {
	case "mqtt":
		return "trusted_broker"
	case "serial", "tcp", "serialtcp":
		return "local_direct"
	default:
		return "unknown"
	}
}
