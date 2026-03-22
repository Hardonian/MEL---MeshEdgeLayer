package service

import (
	"context"
	"fmt"
	"time"

	"github.com/mel-project/mel/internal/topology"
)

// topologySnapshotWorker periodically rescores all nodes and links,
// runs full topology analysis, and saves snapshots. It also marks stale
// nodes/links based on configured thresholds.
func (a *App) topologySnapshotWorker(ctx context.Context) {
	if a.topoStore == nil || !a.Cfg.Topology.Enabled {
		return
	}

	interval := time.Duration(a.Cfg.Topology.SnapshotIntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	// Run once at startup after a brief delay to let initial data flow in
	startupDelay := time.NewTimer(30 * time.Second)
	select {
	case <-ctx.Done():
		startupDelay.Stop()
		return
	case <-startupDelay.C:
	}
	a.runTopologySnapshot()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.runTopologySnapshot()
		}
	}
}

// runTopologySnapshot performs a single topology scoring + analysis + snapshot cycle.
func (a *App) runTopologySnapshot() {
	now := time.Now().UTC()
	thresholds := topology.StaleThresholds{
		NodeStaleDuration: time.Duration(a.Cfg.Topology.NodeStaleMinutes) * time.Minute,
		LinkStaleDuration: time.Duration(a.Cfg.Topology.LinkStaleMinutes) * time.Minute,
		ObservationMaxAge: 1 * time.Hour,
	}
	if thresholds.NodeStaleDuration <= 0 {
		thresholds.NodeStaleDuration = 30 * time.Minute
	}
	if thresholds.LinkStaleDuration <= 0 {
		thresholds.LinkStaleDuration = 30 * time.Minute
	}

	// 1. Mark staleness
	if err := a.topoStore.MarkStaleness(thresholds); err != nil {
		a.Log.Error("topology_staleness_failed", "failed to mark stale nodes/links", map[string]any{"error": err.Error()})
	}

	// 2. Load current nodes and links
	nodes, err := a.topoStore.ListNodes(10000)
	if err != nil {
		a.Log.Error("topology_snapshot_nodes_failed", "failed to load nodes for scoring", map[string]any{"error": err.Error()})
		return
	}
	links, err := a.topoStore.ListLinks(10000)
	if err != nil {
		a.Log.Error("topology_snapshot_links_failed", "failed to load links for scoring", map[string]any{"error": err.Error()})
		return
	}

	if len(nodes) == 0 {
		return
	}

	// 3. Rescore every node
	for _, n := range nodes {
		nodeLinks := linksForNode(links, n.NodeNum)
		score, healthState, factors := topology.ScoreNode(n, nodeLinks, thresholds, now)
		if err := a.topoStore.UpdateNodeHealth(n.NodeNum, score, healthState, factors, n.Stale); err != nil {
			a.Log.Error("topology_score_node_failed", "failed to update node health", map[string]any{
				"node_num": n.NodeNum, "error": err.Error(),
			})
		}
		// Record score history (sampled — only on snapshot intervals)
		_ = a.topoStore.RecordNodeScoreHistory(n.NodeNum, score, healthState, factors)
	}

	// 4. Rescore every link
	for _, l := range links {
		score, factors := topology.ScoreLink(l, thresholds, now)
		l.QualityScore = score
		l.QualityFactors = factors
		if err := a.topoStore.UpsertLink(l); err != nil {
			a.Log.Error("topology_score_link_failed", "failed to update link quality", map[string]any{
				"edge_id": l.EdgeID, "error": err.Error(),
			})
		}
	}

	// 5. Run full topology analysis and save snapshot
	// Reload nodes after scoring to get updated health values
	nodes, _ = a.topoStore.ListNodes(10000)
	links, _ = a.topoStore.ListLinks(10000)
	result := topology.Analyze(nodes, links, thresholds, now)
	if err := a.topoStore.SaveSnapshot(result.Snapshot); err != nil {
		a.Log.Error("topology_snapshot_save_failed", "failed to save topology snapshot", map[string]any{"error": err.Error()})
	}

	// 6. Prune old snapshots beyond retention limit
	a.pruneOldSnapshots()

	// 7. Prune old score history beyond retention
	a.pruneOldScoreHistory()

	a.Log.Info("topology_snapshot_complete", "topology snapshot generated", map[string]any{
		"nodes": result.Snapshot.NodeCount, "edges": result.Snapshot.EdgeCount,
		"healthy": result.Snapshot.HealthyNodes, "stale": result.Snapshot.StaleNodes,
		"snapshot_id": result.Snapshot.SnapshotID,
	})
}

// linksForNode returns all links connected to a given node.
func linksForNode(allLinks []topology.Link, nodeNum int64) []topology.Link {
	var result []topology.Link
	for _, l := range allLinks {
		if l.SrcNodeNum == nodeNum || l.DstNodeNum == nodeNum {
			result = append(result, l)
		}
	}
	return result
}

// pruneOldSnapshots removes snapshots beyond the configured retention limit.
func (a *App) pruneOldSnapshots() {
	maxHistory := a.Cfg.Topology.MaxSnapshotHistory
	if maxHistory <= 0 {
		maxHistory = 200
	}
	sql := `DELETE FROM topology_snapshots WHERE snapshot_id NOT IN (SELECT snapshot_id FROM topology_snapshots ORDER BY created_at DESC LIMIT ` + itoa(maxHistory) + `);`
	_ = a.topoStore.DB.Exec(sql)
}

// pruneOldScoreHistory removes score history entries older than the configured retention.
func (a *App) pruneOldScoreHistory() {
	days := a.Cfg.Topology.ScoreHistoryDays
	if days <= 0 {
		days = 14
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format(time.RFC3339)
	sql := `DELETE FROM node_score_history WHERE recorded_at < '` + cutoff + `';`
	_ = a.topoStore.DB.Exec(sql)
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
