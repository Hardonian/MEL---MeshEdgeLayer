package topology

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"time"
)

// AnalysisResult contains the full topology analysis output.
type AnalysisResult struct {
	Snapshot        TopologySnapshot `json:"snapshot"`
	IsolatedNodes   []int64          `json:"isolated_nodes,omitempty"`
	BridgeNodes     []int64          `json:"bridge_nodes,omitempty"`
	WeakClusters    []Cluster        `json:"weak_clusters,omitempty"`
	StaleRegions    []StaleRegion    `json:"stale_regions,omitempty"`
	Bottlenecks     []Bottleneck     `json:"bottlenecks,omitempty"`
	Recommendations []Recommendation `json:"recommendations,omitempty"`
	AnalyzedAt      string           `json:"analyzed_at"`
}

// Cluster represents a group of connected nodes.
type Cluster struct {
	ID        string      `json:"id"`
	NodeNums  []int64     `json:"node_nums"`
	State     HealthState `json:"state"`
	AvgScore  float64     `json:"avg_score"`
	MinScore  float64     `json:"min_score"`
	EdgeCount int         `json:"edge_count"`
}

// StaleRegion is a set of nodes where visibility has degraded.
type StaleRegion struct {
	NodeNums    []int64 `json:"node_nums"`
	StaleRatio  float64 `json:"stale_ratio"`
	LastFreshAt string  `json:"last_fresh_at,omitempty"`
}

// Bottleneck identifies a topology fragility point.
type Bottleneck struct {
	Type        string  `json:"type"` // single_point_of_failure, weak_bridge, relay_dependent
	NodeNums    []int64 `json:"node_nums,omitempty"`
	Severity    string  `json:"severity"` // critical, high, medium, low
	Explanation string  `json:"explanation"`
}

// Analyze performs full topology analysis on a set of nodes and links.
func Analyze(nodes []Node, links []Link, thresholds StaleThresholds, now time.Time) AnalysisResult {
	// Build adjacency
	adj := buildAdjacency(nodes, links)
	linkMap := buildLinkMap(links)

	// Score all nodes
	scoredNodes := make([]Node, len(nodes))
	for i, n := range nodes {
		nodeLinks := getNodeLinks(n.NodeNum, linkMap)
		score, state, factors := ScoreNode(n, nodeLinks, thresholds, now)
		scored := n
		scored.HealthScore = score
		scored.HealthState = state
		scored.HealthFactors = factors
		scoredNodes[i] = scored
	}

	// Score all links
	scoredLinks := make([]Link, len(links))
	for i, l := range links {
		score, factors := ScoreLink(l, thresholds, now)
		scored := l
		scored.QualityScore = score
		scored.QualityFactors = factors
		scoredLinks[i] = scored
	}

	// Find clusters via BFS
	clusters := findClusters(scoredNodes, adj)

	// Find isolated nodes
	isolated := findIsolatedNodes(scoredNodes, adj)

	// Find bridge nodes (articulation points approximation)
	bridges := findBridgeNodes(scoredNodes, adj)

	// Find stale regions
	staleRegions := findStaleRegions(scoredNodes, adj)

	// Find bottlenecks
	bottlenecks := findBottlenecks(scoredNodes, scoredLinks, adj, bridges)

	// Generate recommendations
	recs := generateRecommendations(scoredNodes, scoredLinks, clusters, isolated, bridges, bottlenecks)

	// Build snapshot
	snapshot := buildSnapshot(scoredNodes, scoredLinks, clusters, now)

	return AnalysisResult{
		Snapshot:        snapshot,
		IsolatedNodes:   isolated,
		BridgeNodes:     bridges,
		WeakClusters:    filterWeakClusters(clusters),
		StaleRegions:    staleRegions,
		Bottlenecks:     bottlenecks,
		Recommendations: recs,
		AnalyzedAt:      now.UTC().Format(time.RFC3339),
	}
}

func buildAdjacency(nodes []Node, links []Link) map[int64][]int64 {
	adj := make(map[int64][]int64)
	for _, n := range nodes {
		if _, ok := adj[n.NodeNum]; !ok {
			adj[n.NodeNum] = nil
		}
	}
	for _, l := range links {
		adj[l.SrcNodeNum] = append(adj[l.SrcNodeNum], l.DstNodeNum)
		if !l.Directional {
			adj[l.DstNodeNum] = append(adj[l.DstNodeNum], l.SrcNodeNum)
		}
	}
	return adj
}

func buildLinkMap(links []Link) map[int64][]Link {
	m := make(map[int64][]Link)
	for _, l := range links {
		m[l.SrcNodeNum] = append(m[l.SrcNodeNum], l)
		m[l.DstNodeNum] = append(m[l.DstNodeNum], l)
	}
	return m
}

func getNodeLinks(nodeNum int64, linkMap map[int64][]Link) []Link {
	return linkMap[nodeNum]
}

func findClusters(nodes []Node, adj map[int64][]int64) []Cluster {
	visited := make(map[int64]bool)
	var clusters []Cluster
	idx := 0
	for _, n := range nodes {
		if visited[n.NodeNum] {
			continue
		}
		// BFS
		cluster := bfs(n.NodeNum, adj, visited)
		if len(cluster) == 0 {
			continue
		}
		idx++
		clusters = append(clusters, Cluster{
			ID:       fmt.Sprintf("cluster-%d", idx),
			NodeNums: cluster,
		})
	}
	// Score clusters
	nodeMap := make(map[int64]Node)
	for _, n := range nodes {
		nodeMap[n.NodeNum] = n
	}
	for i := range clusters {
		c := &clusters[i]
		totalScore := 0.0
		minScore := 1.0
		for _, num := range c.NodeNums {
			if nd, ok := nodeMap[num]; ok {
				totalScore += nd.HealthScore
				if nd.HealthScore < minScore {
					minScore = nd.HealthScore
				}
			}
		}
		if len(c.NodeNums) > 0 {
			c.AvgScore = totalScore / float64(len(c.NodeNums))
		}
		c.MinScore = minScore
		// Count internal edges
		nodeSet := make(map[int64]bool)
		for _, num := range c.NodeNums {
			nodeSet[num] = true
		}
		edgeCount := 0
		for _, num := range c.NodeNums {
			for _, neighbor := range adj[num] {
				if nodeSet[neighbor] && neighbor > num { // count each edge once
					edgeCount++
				}
			}
		}
		c.EdgeCount = edgeCount
		c.State = classifyClusterHealth(c.AvgScore, c.MinScore, len(c.NodeNums))
	}
	return clusters
}

func bfs(start int64, adj map[int64][]int64, visited map[int64]bool) []int64 {
	var result []int64
	queue := []int64{start}
	visited[start] = true
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		result = append(result, curr)
		for _, neighbor := range adj[curr] {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}
	return result
}

func findIsolatedNodes(nodes []Node, adj map[int64][]int64) []int64 {
	var isolated []int64
	for _, n := range nodes {
		if len(adj[n.NodeNum]) == 0 {
			isolated = append(isolated, n.NodeNum)
		}
	}
	return isolated
}

// findBridgeNodes finds articulation point approximation.
// Uses degree-1 neighbor detection and simple cut-vertex heuristic.
func findBridgeNodes(nodes []Node, adj map[int64][]int64) []int64 {
	bridges := make(map[int64]bool)
	// A node is bridge-critical if removing it would disconnect any neighbor
	for _, n := range nodes {
		neighbors := adj[n.NodeNum]
		if len(neighbors) < 2 {
			continue
		}
		// Check if all neighbors can still reach each other without this node
		for _, neighbor := range neighbors {
			// Simple check: does neighbor have any other connection?
			otherConnections := 0
			for _, nn := range adj[neighbor] {
				if nn != n.NodeNum {
					otherConnections++
				}
			}
			if otherConnections == 0 {
				bridges[n.NodeNum] = true
				break
			}
		}
	}
	var result []int64
	for num := range bridges {
		result = append(result, num)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func findStaleRegions(nodes []Node, adj map[int64][]int64) []StaleRegion {
	// Group contiguous stale nodes
	visited := make(map[int64]bool)
	var regions []StaleRegion
	for _, n := range nodes {
		if visited[n.NodeNum] || !n.Stale {
			continue
		}
		// BFS among stale nodes only
		var region []int64
		queue := []int64{n.NodeNum}
		visited[n.NodeNum] = true
		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			region = append(region, curr)
			for _, neighbor := range adj[curr] {
				if !visited[neighbor] {
					// Check if neighbor is stale too
					for _, nn := range nodes {
						if nn.NodeNum == neighbor && nn.Stale {
							visited[neighbor] = true
							queue = append(queue, neighbor)
							break
						}
					}
				}
			}
		}
		if len(region) >= 2 {
			regions = append(regions, StaleRegion{
				NodeNums:   region,
				StaleRatio: 1.0,
			})
		}
	}
	return regions
}

func findBottlenecks(nodes []Node, links []Link, adj map[int64][]int64, bridges []int64) []Bottleneck {
	var bottlenecks []Bottleneck

	// Bridge nodes are single points of failure
	for _, b := range bridges {
		bottlenecks = append(bottlenecks, Bottleneck{
			Type:        "single_point_of_failure",
			NodeNums:    []int64{b},
			Severity:    "critical",
			Explanation: fmt.Sprintf("Node %d is a bridge: removing it would isolate at least one neighbor", b),
		})
	}

	// Weak bridges: links with low quality between clusters
	for _, l := range links {
		if l.QualityScore < 0.3 && l.Observed {
			bottlenecks = append(bottlenecks, Bottleneck{
				Type:        "weak_bridge",
				NodeNums:    []int64{l.SrcNodeNum, l.DstNodeNum},
				Severity:    "high",
				Explanation: fmt.Sprintf("Link %d<->%d has quality %.2f: degraded corridor", l.SrcNodeNum, l.DstNodeNum, l.QualityScore),
			})
		}
	}

	// Relay-dependent links
	for _, l := range links {
		if l.RelayDependent {
			bottlenecks = append(bottlenecks, Bottleneck{
				Type:        "relay_dependent",
				NodeNums:    []int64{l.SrcNodeNum, l.DstNodeNum},
				Severity:    "medium",
				Explanation: fmt.Sprintf("Link %d<->%d depends on relay forwarding", l.SrcNodeNum, l.DstNodeNum),
			})
		}
	}

	return bottlenecks
}

func generateRecommendations(nodes []Node, links []Link, clusters []Cluster, isolated, bridges []int64, bottlenecks []Bottleneck) []Recommendation {
	var recs []Recommendation
	recIdx := 0
	nextID := func() string {
		recIdx++
		return fmt.Sprintf("rec-%d", recIdx)
	}

	// Recommend adding nodes near isolated ones
	for _, nodeNum := range isolated {
		recs = append(recs, Recommendation{
			ID:            nextID(),
			Type:          "add_relay",
			Summary:       fmt.Sprintf("Add a relay or gateway near isolated node %d to restore connectivity", nodeNum),
			Confidence:    0.7,
			Impact:        "Restores connectivity for an isolated node",
			Assumptions:   []string{"Node location is approximately known", "A relay within range would restore links"},
			Evidence:      []string{fmt.Sprintf("Node %d has zero observed links", nodeNum)},
			RefuteWith:    []string{"Node may have been intentionally isolated", "Node may be powered off"},
			Basis:         "topology",
			AffectedNodes: []int64{nodeNum},
		})
	}

	// Recommend redundancy for bridge-critical nodes
	for _, nodeNum := range bridges {
		recs = append(recs, Recommendation{
			ID:            nextID(),
			Type:          "add_relay",
			Summary:       fmt.Sprintf("Add redundant path around bridge-critical node %d to reduce single-point-of-failure risk", nodeNum),
			Confidence:    0.8,
			Impact:        "Reduces dependency on single bridge node",
			Assumptions:   []string{"Graph structure reflects actual RF reachability"},
			Evidence:      []string{fmt.Sprintf("Node %d is the sole connection for at least one neighbor", nodeNum)},
			RefuteWith:    []string{"Hidden links not yet observed may provide alternate paths"},
			Basis:         "topology",
			AffectedNodes: []int64{nodeNum},
		})
	}

	// Recommend inspection of stale nodes
	for _, n := range nodes {
		if n.Stale && !n.Quarantined {
			recs = append(recs, Recommendation{
				ID:            nextID(),
				Type:          "inspect",
				Summary:       fmt.Sprintf("Inspect node %d (%s): no recent observations, may need power/placement check", n.NodeNum, n.ShortName),
				Confidence:    0.6,
				Impact:        "Restoring a stale node may improve local coverage",
				Assumptions:   []string{"Node was previously active"},
				Evidence:      []string{fmt.Sprintf("last_seen_at=%s, health_state=%s", n.LastSeenAt, n.HealthState)},
				RefuteWith:    []string{"Node intentionally powered down"},
				Basis:         "topology",
				AffectedNodes: []int64{n.NodeNum},
			})
		}
	}

	// Recommend trust reduction for contradicting connectors
	for _, n := range nodes {
		if n.Quarantined {
			recs = append(recs, Recommendation{
				ID:            nextID(),
				Type:          "reduce_trust",
				Summary:       fmt.Sprintf("Review quarantined node %d (%s): %s", n.NodeNum, n.ShortName, n.QuarantineReason),
				Confidence:    0.5,
				Impact:        "Removing noisy/contradictory source improves topology accuracy",
				Evidence:      []string{n.QuarantineReason},
				Basis:         "source",
				AffectedNodes: []int64{n.NodeNum},
			})
		}
	}

	// Recommend splitting if clusters are large and disconnected
	for _, c := range clusters {
		if len(c.NodeNums) > 50 && c.AvgScore < 0.4 {
			recs = append(recs, Recommendation{
				ID:         nextID(),
				Type:       "split",
				Summary:    fmt.Sprintf("Consider splitting deployment for cluster %s (%d nodes, avg score %.2f) into separate monitored sites", c.ID, len(c.NodeNums), c.AvgScore),
				Confidence: 0.4,
				Impact:     "Reduces noise and improves per-site visibility",
				Basis:      "topology",
			})
		}
	}

	return recs
}

func filterWeakClusters(clusters []Cluster) []Cluster {
	var weak []Cluster
	for _, c := range clusters {
		if c.State != HealthHealthy {
			weak = append(weak, c)
		}
	}
	return weak
}

func classifyClusterHealth(avgScore, minScore float64, nodeCount int) HealthState {
	if avgScore >= 0.75 && minScore >= 0.4 {
		return HealthHealthy
	}
	if avgScore >= 0.5 {
		return HealthDegraded
	}
	if avgScore >= 0.25 {
		return HealthUnstable
	}
	return HealthWeaklyObserved
}

func buildSnapshot(nodes []Node, links []Link, clusters []Cluster, now time.Time) TopologySnapshot {
	healthy, degraded, stale, isolated := 0, 0, 0, 0
	for _, n := range nodes {
		switch n.HealthState {
		case HealthHealthy:
			healthy++
		case HealthDegraded, HealthBridgeCritical:
			degraded++
		case HealthStale:
			stale++
		case HealthIsolated:
			isolated++
		}
	}
	directEdges, inferredEdges := 0, 0
	for _, l := range links {
		if l.Observed {
			directEdges++
		} else {
			inferredEdges++
		}
	}

	// Compute graph hash from sorted node/edge IDs for fingerprinting
	h := sha256.New()
	for _, n := range nodes {
		fmt.Fprintf(h, "n:%d;", n.NodeNum)
	}
	for _, l := range links {
		fmt.Fprintf(h, "e:%d-%d;", l.SrcNodeNum, l.DstNodeNum)
	}
	graphHash := fmt.Sprintf("%x", h.Sum(nil))[:16]

	// Build region summaries from clusters
	var regionSummary []RegionScore
	for _, c := range clusters {
		regionSummary = append(regionSummary, RegionScore{
			RegionID:   c.ID,
			Label:      fmt.Sprintf("Cluster of %d nodes", len(c.NodeNums)),
			State:      c.State,
			NodeCount:  len(c.NodeNums),
			EdgeCount:  c.EdgeCount,
			Confidence: c.AvgScore,
		})
	}

	return TopologySnapshot{
		SnapshotID:        fmt.Sprintf("topo-%s-%s", now.UTC().Format("20060102-150405"), graphHash[:8]),
		CreatedAt:         now.UTC().Format(time.RFC3339),
		NodeCount:         len(nodes),
		EdgeCount:         len(links),
		DirectEdgeCount:   directEdges,
		InferredEdgeCount: inferredEdges,
		HealthyNodes:      healthy,
		DegradedNodes:     degraded,
		StaleNodes:        stale,
		IsolatedNodes:     isolated,
		GraphHash:         graphHash,
		RegionSummary:     regionSummary,
	}
}
