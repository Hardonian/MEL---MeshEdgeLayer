package service

import (
	"context"
	"time"

	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/planning"
	"github.com/mel-project/mel/internal/topology"
)

func (a *App) topologyWorker(ctx context.Context) {
	if a == nil || a.topo == nil || !a.Cfg.Topology.Enabled {
		return
	}
	snapEvery := a.Cfg.Topology.SnapshotIntervalMinutes
	if snapEvery <= 0 {
		snapEvery = 5
	}
	th := topology.StaleThresholdsFromConfig(a.Cfg.Topology.NodeStaleMinutes, a.Cfg.Topology.LinkStaleMinutes)
	// One pass at startup so API is useful before the first minute tick.
	a.runTopologyRefresh(th)

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	ticks := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ticks++
			_ = a.topo.RefreshStale(th)
			if ticks >= snapEvery {
				ticks = 0
				a.runTopologyRefresh(th)
			}
		}
	}
}

func (a *App) runTopologyRefresh(th topology.StaleThresholds) {
	if a == nil || a.topo == nil {
		return
	}
	now := time.Now().UTC()
	ar, err := a.topo.RefreshScores(th, now)
	if err != nil {
		return
	}
	_ = a.topo.SaveSnapshot(ar.Snapshot)
	maxHist := a.Cfg.Topology.MaxSnapshotHistory
	if maxHist <= 0 {
		maxHist = 200
	}
	_ = a.topo.PruneSnapshots(maxHist)
	_ = a.topo.PruneObservations(a.Cfg.Topology.MaxObservationsPerNode)

	transportOK := a.transportIngestLikely()
	sig, _ := meshintel.RollupRecentMessages(a.DB, 24*time.Hour, transportOK)
	assess := meshintel.Compute(a.Cfg, ar, sig, transportOK, now)
	a.setMeshIntel(assess)
	if a.DB != nil {
		advisories := planning.DeriveResilienceAdvisoryAlerts(ar, assess, now)
		_ = planning.SyncAdvisoryAlertsToStore(a.DB, advisories)
	}
	keepMI := a.Cfg.Topology.MaxMeshIntelHistory
	if keepMI <= 0 {
		keepMI = 120
	}
	_ = meshintel.SaveSnapshot(a.DB, assess, keepMI)
	_ = meshintel.EvaluateViabilityRegression(a.DB, assess, sig, transportOK)
}
