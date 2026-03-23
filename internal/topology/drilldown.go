package topology

import (
	"fmt"
	"time"
)

// NodeDrilldown is the explainable operator view for a single node.
type NodeDrilldown struct {
	Node                 Node              `json:"node"`
	Links                []Link            `json:"links"`
	Observations         []NodeObservation `json:"observations"`
	Bookmarks            []Bookmark        `json:"bookmarks"`
	ScoredHealth         float64           `json:"scored_health"`
	ScoredState          HealthState       `json:"scored_state"`
	ScoreFactors         []ScoreFactor     `json:"score_factors"`
	FreshnessAgeSeconds  float64           `json:"freshness_age_seconds,omitempty"`
	NextActions          []string          `json:"next_actions"`
	EvidenceNotes        []string          `json:"evidence_notes"`
	LinkCount            int               `json:"link_count"`
}

// BuildNodeDrilldown computes explainable drilldown from stored row + live scoring.
func BuildNodeDrilldown(n Node, links []Link, bookmarks []Bookmark, observations []NodeObservation, thresholds StaleThresholds, now time.Time) NodeDrilldown {
	score, state, factors := ScoreNode(n, links, thresholds, now)
	dd := NodeDrilldown{
		Node:         n,
		Links:        links,
		Observations: observations,
		Bookmarks:    bookmarks,
		ScoredHealth: score,
		ScoredState:  state,
		ScoreFactors: factors,
		LinkCount:    len(links),
		EvidenceNotes: []string{
			"Node row is updated from mesh packet ingest; topology links use relay_node / to_node from packets (not RF ground truth).",
		},
	}
	if t := parseRFC3339OrZero(n.LastSeenAt); !t.IsZero() {
		age := now.Sub(t).Seconds()
		if age >= 0 {
			dd.FreshnessAgeSeconds = age
		}
	}
	dd.NextActions = nodeNextActions(n, links, state, score, observations)
	return dd
}

func nodeNextActions(n Node, links []Link, state HealthState, score float64, observations []NodeObservation) []string {
	var a []string
	switch state {
	case HealthStale:
		a = append(a, fmt.Sprintf("Verify power and RF path for node %d; last_seen is older than the configured stale window.", n.NodeNum))
	case HealthIsolated:
		a = append(a, fmt.Sprintf("No topology_links touch node %d yet — confirm gateways see traffic or add a relay in range.", n.NodeNum))
	case HealthBridgeCritical:
		a = append(a, fmt.Sprintf("Node %d has a single link in the observed graph; add redundancy if this matches your physical mesh.", n.NodeNum))
	case HealthInferredOnly:
		a = append(a, "All adjacent links are marked inferred in storage; wait for direct packet-derived edges or inspect transport paths.")
	case HealthQuarantined:
		if n.QuarantineReason != "" {
			a = append(a, "Quarantine: "+n.QuarantineReason)
		} else {
			a = append(a, "Review quarantine reason on the node record.")
		}
	}
	if n.TrustClass == TrustUnknown || n.TrustClass == TrustPartial {
		a = append(a, fmt.Sprintf("Set transport trust_class in config for connector %q to tighten scoring.", n.SourceConnector))
	}
	if len(observations) > 0 && observations[0].ViaMQTT {
		a = append(a, "Latest observation arrived via MQTT; confirm broker and gateway stability if visibility is intermittent.")
	}
	if score < 0.35 && len(a) == 0 {
		a = append(a, "Low composite health score — review score_factors and correlated transport diagnostics.")
	}
	return a
}

// LinkDrilldown explains one edge.
type LinkDrilldown struct {
	Link            Link          `json:"link"`
	ScoredQuality   float64       `json:"scored_quality"`
	QualityFactors  []ScoreFactor `json:"quality_factors"`
	NextActions     []string      `json:"next_actions"`
	EvidenceNotes   []string      `json:"evidence_notes"`
}

// BuildLinkDrilldown scores a single link with explainable factors.
func BuildLinkDrilldown(l Link, thresholds StaleThresholds, now time.Time) LinkDrilldown {
	q, factors := ScoreLink(l, thresholds, now)
	dd := LinkDrilldown{
		Link:           l,
		ScoredQuality:  q,
		QualityFactors: factors,
		EvidenceNotes: []string{
			"Edge derived from mesh packet fields seen by MEL transports; not proof of RF link quality.",
		},
	}
	if l.Stale {
		dd.NextActions = append(dd.NextActions, "Link last_observed_at exceeds stale threshold — check whether traffic still traverses this path.")
	}
	if l.RelayDependent {
		dd.NextActions = append(dd.NextActions, "Relay-dependent path — broker or intermediate gateway stability affects visibility.")
	}
	if l.Contradiction {
		dd.NextActions = append(dd.NextActions, "Contradiction flag set — compare source_connector_id and observation history before trusting this edge.")
	}
	if len(dd.NextActions) == 0 && q < 0.4 {
		dd.NextActions = append(dd.NextActions, "Low quality score — inspect quality_factors and upstream transport health.")
	}
	return dd
}

// ClusterDrilldown summarizes a segment (cluster) by id from analysis.
type ClusterDrilldown struct {
	Cluster       Cluster   `json:"cluster"`
	NextActions   []string  `json:"next_actions"`
	EvidenceNotes []string  `json:"evidence_notes"`
}

// BuildClusterDrilldown finds cluster id in weak list or full analysis clusters.
func BuildClusterDrilldown(clusterID string, weak []Cluster, all []Cluster) (ClusterDrilldown, bool) {
	var c *Cluster
	for i := range all {
		if all[i].ID == clusterID {
			c = &all[i]
			break
		}
	}
	if c == nil {
		return ClusterDrilldown{}, false
	}
	dd := ClusterDrilldown{
		Cluster: *c,
		EvidenceNotes: []string{
			"Cluster is a connected component in the topology_links graph as stored by MEL.",
		},
	}
	if c.State != HealthHealthy {
		dd.NextActions = append(dd.NextActions, fmt.Sprintf("Cluster %s avg_health=%.2f min=%.2f — prioritize stale or bridge-critical nodes inside the segment.", c.ID, c.AvgScore, c.MinScore))
	}
	if c.EdgeCount < len(c.NodeNums)-1 && len(c.NodeNums) > 1 {
		dd.NextActions = append(dd.NextActions, "Sparse internal edges — discovery may be incomplete; more traffic or gateways improve the graph.")
	}
	return dd, true
}
