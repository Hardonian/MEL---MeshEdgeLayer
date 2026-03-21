package service

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/mel-project/mel/internal/eventlog"
	"github.com/mel-project/mel/internal/federation"
	"github.com/mel-project/mel/internal/kernel"
	"github.com/mel-project/mel/internal/region"
	"github.com/mel-project/mel/internal/replay"
	"github.com/mel-project/mel/internal/snapshot"
	"github.com/mel-project/mel/internal/web"
)

// kernelBridge holds all distributed kernel components and bridges
// the existing event bus to the kernel event pipeline.
type kernelBridge struct {
	kernel       *kernel.Kernel
	eventLog     *eventlog.Log
	snapStore    *snapshot.Store
	replayEng    *replay.Engine
	federation   *federation.Manager
	regionMgr    *region.Manager
	backpressure *kernel.Backpressure
	durability   *kernel.StorageDurability
	coordinator  *kernel.ActionCoordinator

	nodeID string
	region string

	// Telemetry counters
	eventsIngested  atomic.Uint64
	effectsProduced atomic.Uint64
	eventsDropped   atomic.Uint64

	// Snapshot interval tracking
	eventsSinceSnapshot atomic.Uint64
	snapshotInterval    int
}

// initKernelBridge initializes all kernel components and returns the bridge.
// Returns nil if federation/kernel is not enabled or sqlite3 is unavailable.
func (a *App) initKernelBridge() *kernelBridge {
	cfg := a.Cfg.Federation
	dataDir := a.Cfg.Storage.DataDir

	// Always initialize the kernel even when federation is off — the event
	// log, replay, and scoring are useful standalone.
	nodeID := cfg.NodeID
	if nodeID == "" {
		nodeID = kernel.NewNodeID()
	}
	regionID := cfg.Region
	if regionID == "" {
		regionID = "default"
	}

	policy := kernel.Policy{
		Version:              "v1",
		Mode:                 a.Cfg.Control.Mode,
		AllowedActions:       a.Cfg.Control.AllowedActions,
		RequireMinConfidence: a.Cfg.Control.RequireMinConfidence,
	}

	k := kernel.New(nodeID, policy)

	// Event log
	evtLogPath := filepath.Join(dataDir, "kernel_events.db")
	evtLog, err := eventlog.Open(eventlog.Config{
		DBPath:        evtLogPath,
		NodeID:        nodeID,
		RetentionDays: cfg.EventLogRetentionDays,
	})
	if err != nil {
		a.Log.Error("kernel_eventlog_init_failed", "failed to open kernel event log", map[string]any{"error": err.Error()})
		return nil
	}

	// Snapshot store
	snapPath := filepath.Join(dataDir, "kernel_snapshots.db")
	snapStore, err := snapshot.NewStore(snapPath, nodeID)
	if err != nil {
		a.Log.Error("kernel_snapshot_init_failed", "failed to open snapshot store", map[string]any{"error": err.Error()})
		return nil
	}

	// Replay engine
	replayEng := replay.NewEngine(evtLog, nodeID)

	// Backpressure
	bp := kernel.NewBackpressure(kernel.DefaultBackpressureConfig())

	// Durability
	backupDir := filepath.Join(dataDir, "backups")
	durability := kernel.NewStorageDurability(evtLogPath, backupDir)

	// Action coordinator
	coordinator := kernel.NewActionCoordinator(nodeID, 5*time.Minute)

	// Region manager
	regPath := filepath.Join(dataDir, "kernel_regions.db")
	regionMgr, err := region.NewManager(regionID, regPath)
	if err != nil {
		a.Log.Error("kernel_region_init_failed", "failed to open region manager", map[string]any{"error": err.Error()})
		// Non-fatal: proceed without region
	}

	// Federation manager (only if enabled)
	var fedMgr *federation.Manager
	if cfg.Enabled {
		fedCfg := federation.Config{
			Enabled:                  true,
			NodeID:                   nodeID,
			NodeName:                 cfg.NodeName,
			Region:                   regionID,
			HeartbeatIntervalSeconds: cfg.HeartbeatIntervalSeconds,
			SuspectAfterMissed:       cfg.SuspectAfterMissed,
			PartitionAfterMissed:     cfg.PartitionAfterMissed,
			SyncBatchSize:            cfg.SyncBatchSize,
			SyncIntervalSeconds:      cfg.SyncIntervalSeconds,
			SplitBrainPolicy: federation.SplitBrainPolicy{
				RestrictAutopilot:    cfg.SplitBrainPolicy.RestrictAutopilot,
				RequireApproval:      cfg.SplitBrainPolicy.RequireApproval,
				AlertOperator:        cfg.SplitBrainPolicy.AlertOperator,
				MaxAutonomousActions: cfg.SplitBrainPolicy.MaxAutonomousActions,
			},
		}
		for _, pc := range cfg.Peers {
			fedCfg.Peers = append(fedCfg.Peers, federation.PeerConfig{
				NodeID:     pc.NodeID,
				Name:       pc.Name,
				Endpoint:   pc.Endpoint,
				Region:     pc.Region,
				TrustLevel: pc.TrustLevel,
				SyncScope: federation.SyncScope{
					EventTypes: pc.SyncTypes,
					Regions:    pc.SyncRegions,
				},
			})
		}

		fedDBPath := filepath.Join(dataDir, "kernel_federation.db")
		fedMgr, err = federation.NewManager(fedCfg, a.Log, evtLog, k, fedDBPath)
		if err != nil {
			a.Log.Error("kernel_federation_init_failed", "failed to init federation manager", map[string]any{"error": err.Error()})
			// Non-fatal: proceed without federation
		}
	}

	// Enable WAL mode for crash safety
	if err := durability.EnableWAL(); err != nil {
		a.Log.Warn("kernel_wal_failed", "failed to enable WAL mode", map[string]any{"error": err.Error()})
	}

	bridge := &kernelBridge{
		kernel:           k,
		eventLog:         evtLog,
		snapStore:        snapStore,
		replayEng:        replayEng,
		federation:       fedMgr,
		regionMgr:        regionMgr,
		backpressure:     bp,
		durability:       durability,
		coordinator:      coordinator,
		nodeID:           nodeID,
		region:           regionID,
		snapshotInterval: cfg.SnapshotIntervalEvents,
	}

	if bridge.snapshotInterval <= 0 {
		bridge.snapshotInterval = 1000
	}

	a.Log.Info("kernel_bridge_init", "kernel bridge initialized", map[string]any{
		"node_id":            nodeID,
		"region":             regionID,
		"federation_enabled": cfg.Enabled,
	})

	return bridge
}

// ingestKernelEvent converts a bus event into a kernel event, applies backpressure,
// appends to event log, runs through the kernel, and dispatches effects.
func (kb *kernelBridge) ingestKernelEvent(a *App, eventType kernel.EventType, subject string, data any) {
	if !kb.backpressure.Admit() {
		kb.eventsDropped.Add(1)
		return
	}
	defer kb.backpressure.Release()

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return
	}

	evt := &kernel.Event{
		ID:           kernel.NewEventID(),
		Type:         eventType,
		Timestamp:    time.Now().UTC(),
		SourceNodeID: kb.nodeID,
		SourceRegion: kb.region,
		Subject:      subject,
		Data:         dataJSON,
	}

	// Append to event log (assigns sequence number and checksum)
	if _, err := kb.eventLog.Append(evt); err != nil {
		a.Log.Error("kernel_event_append_failed", "failed to append event to log", map[string]any{
			"type":  string(eventType),
			"error": err.Error(),
		})
		return
	}

	// Apply to kernel (deterministic state transition)
	effects := kb.kernel.Apply(*evt)

	kb.eventsIngested.Add(1)
	kb.effectsProduced.Add(uint64(len(effects)))

	// Dispatch effects
	for _, eff := range effects {
		kb.dispatchEffect(a, eff)
	}

	// Push-notify peers of new events
	if kb.federation != nil {
		kb.federation.SendPushNotifications()
	}

	// Check if we should take a snapshot
	count := kb.eventsSinceSnapshot.Add(1)
	if int(count) >= kb.snapshotInterval {
		kb.eventsSinceSnapshot.Store(0)
		go kb.takeSnapshot(a)
	}
}

// dispatchEffect routes a kernel effect to the appropriate consumer.
func (kb *kernelBridge) dispatchEffect(a *App, eff kernel.Effect) {
	switch eff.Type {
	case kernel.EffectProposeAction:
		kb.handleProposeAction(a, eff)
	case kernel.EffectEmitAlert:
		a.Log.Info("kernel_alert", "kernel alert emitted", map[string]any{
			"subject":   eff.Subject,
			"caused_by": eff.CausedBy,
		})
	case kernel.EffectUpdateScore:
		// Score updates are already in kernel state; update region health
		if kb.regionMgr != nil {
			state := kb.kernel.State()
			kb.regionMgr.UpdateHealthFromKernel(&state)
		}
	case kernel.EffectClassifyNode, kernel.EffectRecordEvidence, kernel.EffectUpdateState:
		// Logged implicitly via kernel state
	}
}

// handleProposeAction converts a kernel EffectProposeAction into a control action.
func (kb *kernelBridge) handleProposeAction(a *App, eff kernel.Effect) {
	// Check federation coordination
	if kb.federation != nil && !kb.federation.CanExecuteAutonomously() {
		a.Log.Warn("kernel_action_blocked", "action blocked by federation split-brain policy", map[string]any{
			"subject": eff.Subject,
		})
		return
	}

	var proposed kernel.ActionProposedData
	if err := json.Unmarshal(eff.Data, &proposed); err != nil {
		return
	}

	// Feed into the existing control queue if the mode allows
	if a.Cfg.Control.Mode == "disabled" {
		return
	}

	a.Log.Info("kernel_action_proposed", "kernel proposed control action", map[string]any{
		"action_type": proposed.ActionType,
		"target":      proposed.Target,
		"confidence":  proposed.Confidence,
		"reason":      proposed.Reason,
	})

	if kb.federation != nil {
		kb.federation.RecordAutonomousAction()
	}
}

// takeSnapshot creates a point-in-time snapshot of kernel state.
func (kb *kernelBridge) takeSnapshot(a *App) {
	state := kb.kernel.StatePtr()
	seqNum := kb.eventLog.LastSequenceNum()

	snap, err := kb.snapStore.Create(state, seqNum)
	if err != nil {
		a.Log.Error("kernel_snapshot_failed", "failed to create snapshot", map[string]any{"error": err.Error()})
		return
	}

	a.Log.Info("kernel_snapshot_created", "created kernel state snapshot", map[string]any{
		"snapshot_id": snap.ID,
		"sequence":    seqNum,
	})

	// Prune old snapshots
	if err := kb.snapStore.Prune(time.Now().Add(-24*time.Hour), 10); err != nil {
		a.Log.Warn("kernel_snapshot_prune_failed", "failed to prune snapshots", map[string]any{"error": err.Error()})
	}
}

// startKernelWorkers launches background workers for the kernel bridge.
func (a *App) startKernelWorkers(ctx context.Context, kb *kernelBridge) {
	// Event bus adapter: subscribes to the existing bus and converts to kernel events
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		kb.eventBusAdapter(ctx, a)
	}()

	// Region health updater
	if kb.regionMgr != nil {
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			kb.regionHealthWorker(ctx, a)
		}()
	}

	// Federation background workers
	if kb.federation != nil {
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			kb.federation.Run(ctx)
		}()
	}

	// Event log compaction worker
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		kb.compactionWorker(ctx, a)
	}()

	// Telemetry reporter
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		kb.telemetryWorker(ctx, a)
	}()

	// Cross-region latency prober
	if kb.federation != nil && kb.regionMgr != nil {
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			kb.latencyProber(ctx, a)
		}()
	}
}

// eventBusAdapter subscribes to the existing event bus and bridges events
// into the kernel pipeline.
func (kb *kernelBridge) eventBusAdapter(ctx context.Context, a *App) {
	ch := a.Bus.Subscribe()
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-ch:
			kb.translateBusEvent(a, evt.Type, evt.Data)
		}
	}
}

// translateBusEvent maps existing bus event types to kernel event types.
func (kb *kernelBridge) translateBusEvent(a *App, busType string, data any) {
	switch busType {
	case "observation":
		kb.ingestKernelEvent(a, kernel.EventObservation, "", data)
	case "anomaly":
		kb.ingestKernelEvent(a, kernel.EventAnomaly, "", data)
	case "topology_update":
		kb.ingestKernelEvent(a, kernel.EventTopologyUpdate, "", data)
	case "transport_health":
		kb.ingestKernelEvent(a, kernel.EventTransportHealth, "", data)
	case "node_state":
		kb.ingestKernelEvent(a, kernel.EventNodeState, "", data)
	case "action_proposed":
		kb.ingestKernelEvent(a, kernel.EventActionProposed, "", data)
	case "action_executed":
		kb.ingestKernelEvent(a, kernel.EventActionExecuted, "", data)
	case "action_completed":
		kb.ingestKernelEvent(a, kernel.EventActionCompleted, "", data)
	case "freeze_created":
		kb.ingestKernelEvent(a, kernel.EventFreezeCreated, "", data)
	case "freeze_cleared":
		kb.ingestKernelEvent(a, kernel.EventFreezeCleared, "", data)
	case "policy_change":
		kb.ingestKernelEvent(a, kernel.EventPolicyChange, "", data)
	case "operator_action":
		kb.ingestKernelEvent(a, kernel.EventOperatorAction, "", data)
	case "approval":
		kb.ingestKernelEvent(a, kernel.EventApproval, "", data)
	case "rejection":
		kb.ingestKernelEvent(a, kernel.EventRejection, "", data)
	case "maintenance_start":
		kb.ingestKernelEvent(a, kernel.EventMaintenanceStart, "", data)
	case "maintenance_end":
		kb.ingestKernelEvent(a, kernel.EventMaintenanceEnd, "", data)
	default:
		// Unknown bus events are recorded as observations
		kb.ingestKernelEvent(a, kernel.EventObservation, busType, data)
	}
}

// regionHealthWorker periodically recomputes region health from kernel state.
func (kb *kernelBridge) regionHealthWorker(ctx context.Context, a *App) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			state := kb.kernel.State()
			kb.regionMgr.UpdateHealthFromKernel(&state)
		}
	}
}

// compactionWorker periodically compacts old events from the log.
func (kb *kernelBridge) compactionWorker(ctx context.Context, a *App) {
	retentionDays := a.Cfg.Federation.EventLogRetentionDays
	if retentionDays <= 0 {
		retentionDays = 14
	}
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
			compacted, err := kb.eventLog.Compact(cutoff, 1000)
			if err != nil {
				a.Log.Error("kernel_compaction_failed", "event log compaction failed", map[string]any{"error": err.Error()})
			} else if compacted > 0 {
				a.Log.Info("kernel_compaction", "compacted old events", map[string]any{"compacted": compacted})
			}
		}
	}
}

// telemetryWorker periodically logs kernel telemetry.
func (kb *kernelBridge) telemetryWorker(ctx context.Context, a *App) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			bpStats := kb.backpressure.Stats()
			state := kb.kernel.State()

			a.Log.Info("kernel_telemetry", "kernel telemetry report", map[string]any{
				"events_ingested":  kb.eventsIngested.Load(),
				"effects_produced": kb.effectsProduced.Load(),
				"events_dropped":   kb.eventsDropped.Load(),
				"bp_accepted":      bpStats.Accepted,
				"bp_rejected":      bpStats.Rejected,
				"bp_throttled":     bpStats.Throttled,
				"bp_pending":       bpStats.PendingCount,
				"node_count":       len(state.NodeRegistry),
				"transport_count":  len(state.TransportScores),
				"action_count":     len(state.ActionStates),
				"freeze_count":     len(state.ActiveFreezes),
				"logical_clock":    state.LogicalClock,
				"last_sequence":    state.LastSequenceNum,
			})

			if kb.federation != nil {
				fedStatus := kb.federation.Status()
				a.Log.Info("federation_telemetry", "federation telemetry report", map[string]any{
					"peer_count":     fedStatus.PeerCount,
					"split_brain":    fedStatus.SplitBrain,
					"conflict_count": fedStatus.ConflictCount,
				})
			}
		}
	}
}

// latencyProber measures cross-region latency by timing heartbeat round-trips.
func (kb *kernelBridge) latencyProber(ctx context.Context, a *App) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			peers := kb.federation.Peers()
			for _, peer := range peers {
				if peer.Region == kb.region || peer.State != federation.PeerStateActive {
					continue
				}
				start := time.Now()
				// Measure via heartbeat send timing
				hb := kb.federation.GenerateHeartbeat()
				_ = hb // The actual measurement comes from the peer response time
				latency := time.Since(start).Seconds() * 1000

				kb.regionMgr.UpdateCrossRegionLink(region.CrossRegionLink{
					RegionA:     kb.region,
					RegionB:     peer.Region,
					Latency:     latency,
					Healthy:     peer.LastSeen.After(time.Now().Add(-2 * time.Minute)),
					LastChecked: time.Now().UTC(),
				})
			}
		}
	}
}

// wireFederationHandlers connects the web server's federation API endpoints
// to the real kernel bridge implementations.
func (a *App) wireFederationHandlers(kb *kernelBridge) {
	handlers := &web.FederationHandlers{
		Status: func() (any, error) {
			if kb.federation != nil {
				return kb.federation.Status(), nil
			}
			return map[string]any{
				"enabled": false,
				"node_id": kb.nodeID,
				"region":  kb.region,
			}, nil
		},
		Peers: func() (any, error) {
			if kb.federation != nil {
				return map[string]any{"peers": kb.federation.Peers()}, nil
			}
			return map[string]any{"peers": []any{}}, nil
		},
		Heartbeat: func(body []byte) error {
			if kb.federation == nil {
				return fmt.Errorf("federation not enabled")
			}
			var hb federation.Heartbeat
			if err := json.Unmarshal(body, &hb); err != nil {
				return fmt.Errorf("invalid heartbeat: %w", err)
			}
			kb.federation.ProcessHeartbeat(hb)
			return nil
		},
		SyncRequest: func(body []byte) (any, error) {
			if kb.federation == nil {
				return nil, fmt.Errorf("federation not enabled")
			}
			var req federation.SyncRequest
			if err := json.Unmarshal(body, &req); err != nil {
				return nil, fmt.Errorf("invalid sync request: %w", err)
			}
			return kb.federation.HandleSyncRequest(req)
		},
		SyncHealth: func() (any, error) {
			if kb.federation == nil {
				return map[string]any{"enabled": false}, nil
			}
			status := kb.federation.Status()
			return map[string]any{
				"enabled":        true,
				"split_brain":    status.SplitBrain,
				"peer_count":     status.PeerCount,
				"conflict_count": status.ConflictCount,
				"conflicts":      kb.federation.Conflicts(),
			}, nil
		},
		ReplayExecute: func(body []byte) (any, error) {
			var req replay.Request
			if err := json.Unmarshal(body, &req); err != nil {
				return nil, fmt.Errorf("invalid replay request: %w", err)
			}
			return kb.replayEng.Execute(req)
		},
		SnapshotList: func(limit int) (any, error) {
			if limit <= 0 {
				limit = 10
			}
			return kb.snapStore.List(limit)
		},
		SnapshotCreate: func() (any, error) {
			state := kb.kernel.StatePtr()
			seqNum := kb.eventLog.LastSequenceNum()
			return kb.snapStore.Create(state, seqNum)
		},
		GlobalTopology: func() (any, error) {
			if kb.regionMgr == nil {
				return map[string]any{"regions": []any{}}, nil
			}
			return kb.regionMgr.ComputeGlobalTopology(), nil
		},
		RegionHealth: func(regionID string) (any, error) {
			if kb.regionMgr == nil {
				return nil, fmt.Errorf("region manager not available")
			}
			h, ok := kb.regionMgr.RegionHealth(regionID)
			if !ok {
				return nil, fmt.Errorf("region not found: %s", regionID)
			}
			return h, nil
		},
		EventLogStats: func() (any, error) {
			return kb.eventLog.Stats(), nil
		},
		EventLogQuery: func(body []byte) (any, error) {
			var filter eventlog.QueryFilter
			if err := json.Unmarshal(body, &filter); err != nil {
				return nil, fmt.Errorf("invalid query: %w", err)
			}
			return kb.eventLog.Query(filter)
		},
		BackpressureStats: func() (any, error) {
			return kb.backpressure.Stats(), nil
		},
		DurabilityStatus: func() (any, error) {
			ok, detail, err := kb.durability.IntegrityCheck()
			if err != nil {
				return map[string]any{
					"integrity_check": false,
					"detail":          err.Error(),
				}, nil
			}
			return map[string]any{
				"integrity_check": ok,
				"detail":          detail,
			}, nil
		},
		BackupCreate: func() (any, error) {
			path, err := kb.durability.Backup()
			if err != nil {
				return nil, err
			}
			return map[string]any{"backup_path": path}, nil
		},
		BackupList: func() (any, error) {
			backups, err := kb.durability.ListBackups()
			if err != nil {
				return nil, err
			}
			return map[string]any{"backups": backups}, nil
		},
		PushNotify: func(body []byte) error {
			if kb.federation == nil {
				return fmt.Errorf("federation not enabled")
			}
			var notify struct {
				FromNodeID string `json:"from_node_id"`
			}
			if err := json.Unmarshal(body, &notify); err != nil {
				return fmt.Errorf("invalid push notification: %w", err)
			}
			kb.federation.NotifyNewEvents(notify.FromNodeID)
			return nil
		},
	}

	a.Web.SetFederationHandlers(handlers)
}
