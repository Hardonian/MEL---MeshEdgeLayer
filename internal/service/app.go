package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/diagnostics"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/fleet"
	"github.com/mel-project/mel/internal/intelligence"
	"github.com/mel-project/mel/internal/investigation"
	"github.com/mel-project/mel/internal/logging"
	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/meshstate"
	"github.com/mel-project/mel/internal/meshtastic"
	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/plugins"
	"github.com/mel-project/mel/internal/policy"
	"github.com/mel-project/mel/internal/privacy"
	"github.com/mel-project/mel/internal/retention"
	statuspkg "github.com/mel-project/mel/internal/status"
	"github.com/mel-project/mel/internal/topology"
	"github.com/mel-project/mel/internal/transport"
	"github.com/mel-project/mel/internal/web"
)

type App struct {
	Cfg config.Config
	// ConfigPath is the on-disk path used to load Cfg (set by cmd/mel for boot metadata).
	ConfigPath              string
	processStartedAt        time.Time
	Log                     *logging.Logger
	DB                      *db.DB
	Bus                     *events.Bus
	State                   *meshstate.State
	Web                     *web.Server
	Transports              []transport.Transport
	Plugins                 []plugins.Plugin
	dlMu                    sync.Mutex
	dlEpisodes              map[string]deadLetterEpisode
	observationEpisodes     map[string]deadLetterEpisode
	ingestCh                chan ingestRequest
	observationCh           chan transport.Observation
	wg                      sync.WaitGroup
	incidentLogLimit        int
	intelligenceEvery       time.Duration
	controlQueue            chan control.ControlAction
	transportControls       map[string]*transportControlState
	kb                      *kernelBridge
	lastTransportHealth     map[string]transport.Health
	healthMu                sync.Mutex
	lastExecutorHeartbeatMu sync.Mutex
	lastExecutorHeartbeat   time.Time
	topo                    *topology.Store
	meshIntelMu             sync.RWMutex
	meshIntelLatest         meshintel.Assessment
	meshIntelHas            bool
}

type ingestRequest struct {
	transport transport.Transport
	topic     string
	payload   []byte
}

type deadLetterEpisode struct {
	fingerprint string
	recordedAt  time.Time
}

const (
	deadLetterSuppressionWindow       = 30 * time.Second
	observationAuditSuppressionWindow = 2 * time.Second
	defaultIngestQueueSize            = 2048
	defaultObservationQueueSize       = 2048
	defaultIngestWorkers              = 4
)

func New(cfg config.Config, debug bool) (*App, error) {
	log := logging.New(cfg.Logging.Level, debug)
	database, err := db.Open(cfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	bus := events.New()
	state := meshstate.New()
	startedAt := time.Now().UTC()
	app := &App{Cfg: cfg, processStartedAt: startedAt, Log: log, DB: database, Bus: bus, State: state, Plugins: []plugins.Plugin{plugins.UnsafeMQTTPlugin{}}, dlEpisodes: map[string]deadLetterEpisode{}, observationEpisodes: map[string]deadLetterEpisode{}, ingestCh: make(chan ingestRequest, defaultIngestQueueSize), observationCh: make(chan transport.Observation, defaultObservationQueueSize), incidentLogLimit: 100, controlQueue: make(chan control.ControlAction, cfg.Control.MaxQueue), transportControls: map[string]*transportControlState{}, lastTransportHealth: map[string]transport.Health{}, topo: topology.NewStore(database)}
	app.Web = web.New(cfg, log, database, state, bus, app.TransportHealth, app.recommendations, app.statusSnapshot, app.controlExplanation, app.controlHistory, diagnostics.Run, app.GenerateBriefing, app.Investigate)
	app.Web.SetTopologyStore(app.topo)
	app.Web.SetTopologyTransportLive(app.transportIngestLikely)
	app.Web.SetMeshIntelProvider(app.meshIntelSnapshot)
	if strings.TrimSpace(app.ConfigPath) != "" {
		app.Web.SetConfigPath(app.ConfigPath)
	}
	app.Web.SetProcessStartedAt(startedAt)
	app.Web.SetQueueDepthsFunc(app.getQueueDepths)
	app.Web.SetTrustFuncs(
		app.ApproveAction,
		app.RejectActionHTTP,
		app.CreateFreeze,
		app.ClearFreeze,
		app.CreateMaintenanceWindow,
		app.CancelMaintenanceWindow,
		app.AddOperatorNote,
		app.Timeline,
		app.InspectAction,
		app.OperationalState,
	)
	app.Web.SetFleetImport(
		app.ImportRemoteEvidenceBundle,
		app.ListImportedRemoteEvidence,
		app.GetImportedRemoteEvidence,
	)
	app.Web.SetIncidentCollaboration(app.IncidentHandoff, app.IncidentByIDForAPI)
	app.Web.SetIncidentMoatExtensions(app.PatchIncidentWorkflow, app.RecordRecommendationOutcome, app.IncidentReplayView, app.BuildEscalationBundle)
	app.Web.SetRecentIncidents(app.RecentIncidentsForAPI)
	app.Web.SetProofpackAssembler(app.AssembleProofpack)
	app.Web.SetOperatorControlQueue(app.QueueOperatorControlAction)
	for _, tc := range cfg.Transports {
		app.transportControls[tc.Name] = newTransportControlState()
		t, err := transport.Build(tc, log, bus)
		if err != nil {
			return nil, err
		}
		app.Transports = append(app.Transports, t)
	}

	// Initialize kernel bridge (event log, federation, region, replay, etc.)
	app.kb = app.initKernelBridge()
	if app.kb != nil {
		app.wireFederationHandlers(app.kb)
	}

	return app, nil
}

func (a *App) recommendations() []policy.Recommendation { return policy.Explain(a.Cfg) }
func (a *App) meshIntelSnapshot() (meshintel.Assessment, bool) {
	a.meshIntelMu.RLock()
	defer a.meshIntelMu.RUnlock()
	if !a.meshIntelHas {
		return meshintel.Assessment{}, false
	}
	return a.meshIntelLatest, true
}

func (a *App) setMeshIntel(m meshintel.Assessment) {
	a.meshIntelMu.Lock()
	defer a.meshIntelMu.Unlock()
	a.meshIntelLatest = m
	a.meshIntelHas = true
}

func (a *App) transportIngestLikely() bool {
	for _, t := range a.Transports {
		tc := findTransport(a.Cfg, t.Name())
		if !tc.Enabled {
			continue
		}
		h := t.Health()
		if h.State == transport.StateLive || h.State == transport.StateIdle {
			return true
		}
	}
	return false
}

func (a *App) TransportHealth() []transport.Health {
	out := make([]transport.Health, 0, len(a.Transports))
	for _, t := range a.Transports {
		out = append(out, t.Health())
	}
	return out
}

func (a *App) statusSnapshot() (statuspkg.Snapshot, error) {
	pt := a.processStartedAt
	return statuspkg.Collect(a.Cfg, a.DB, a.TransportHealth(), &pt, a.ConfigPath)
}

func (a *App) getQueueDepths() map[string]int {
	return map[string]int{
		"ingest":      len(a.ingestCh),
		"observation": len(a.observationCh),
		"control":     len(a.controlQueue),
	}
}

// GenerateBriefing gathers current system state and produces a ranked operational briefing
func (a *App) GenerateBriefing() models.OperatorBriefingDTO {
	now := time.Now().UTC()
	var incidents []models.Incident
	if a.DB != nil {
		incidents, _ = a.DB.RecentIncidents(20)
	}

	findings := diagnostics.Run(a.Cfg, a.DB)
	priorities := intelligence.RankOperationalIssues(incidents, findings, now)
	recommendations := intelligence.RecommendActions(priorities)
	sequence := intelligence.SequenceRecoveryActions(recommendations)

	snapshot := a.State.Snapshot()
	var nodes []models.Node
	for _, n := range snapshot.Nodes {
		nodes = append(nodes, models.Node{
			NodeNum:  n.Num,
			NodeID:   n.ID,
			LongName: n.LongName,
			LastSeen: n.LastSeen,
		})
	}

	_, blastMessage := intelligence.EstimateBlastRadius(priorities, nodes)

	briefing := intelligence.GenerateBriefing(priorities, recommendations, sequence, blastMessage, now)
	return briefing
}

// Investigate assembles a fresh canonical investigation summary.
func (a *App) Investigate() investigation.Summary {
	return investigation.Derive(a.Cfg, a.DB, a.TransportHealth(), a.TransportRuntimeStates(), time.Now().UTC())
}

// TransportRuntimeStates returns the persisted runtime states of all transports.
func (a *App) TransportRuntimeStates() []db.TransportRuntime {
	if a.DB == nil {
		return nil
	}
	states, _ := a.DB.TransportRuntimeStatuses()
	return states
}

func (a *App) Start(ctx context.Context) error {
	if err := os.MkdirAll(a.Cfg.Storage.DataDir, 0o755); err != nil {
		return err
	}
	if err := retention.Run(a.DB, a.Cfg); err != nil {
		return err
	}
	if a.DB != nil {
		_, _ = a.DB.EnsureInstanceID()
		_ = fleet.SyncScopeMetadata(a.Cfg, a.DB)
		canonFP, err := config.CanonicalFingerprintSHA256(a.Cfg)
		if err == nil {
			prev, ok, _ := a.DB.GetInstanceMetadata(db.MetaBootConfigFingerprint)
			if ok && prev != "" && prev != canonFP {
				a.insertAuditLog("config", "warning", "effective config canonical fingerprint changed since last boot", map[string]any{
					"previous_canonical_fingerprint": prev,
					"current_canonical_fingerprint":  canonFP,
				})
			}
			_ = a.DB.SetInstanceMetadata(db.MetaBootConfigFingerprint, canonFP)
			if strings.TrimSpace(a.ConfigPath) != "" {
				_ = a.DB.SetInstanceMetadata(db.MetaBootConfigPath, strings.TrimSpace(a.ConfigPath))
			}
			_ = a.DB.SetInstanceMetadata(db.MetaBootAt, time.Now().UTC().Format(time.RFC3339))
		}
		a.syncControlReality()
		a.recoverIncompleteControlActions(time.Now().UTC())
	}
	for _, finding := range privacy.Audit(a.Cfg) {
		a.insertAuditLog("privacy", finding.Severity, finding.Message, finding)
	}
	if len(enabledTransportConfigs(a.Cfg)) == 0 {
		a.Log.Info("transport_idle", "no transports enabled; MEL will remain idle", map[string]any{"state": transport.StateConfiguredNotAttempted})
		a.insertAuditLog("transport", "warning", "no transports enabled; MEL will remain explicitly idle", map[string]any{"guidance": "Enable one transport before expecting stored packets."})
	}
	for _, tc := range a.Cfg.Transports {
		state := transport.StateConfiguredNotAttempted
		detail := "configured; MEL has not attempted a live connection in this process yet"
		if !tc.Enabled {
			state = transport.StateDisabled
			detail = "disabled by config"
		} else if tc.Type == "serial" || tc.Type == "tcp" || tc.Type == "serialtcp" {
			a.insertAuditLog("transport", "warning", "direct-node transport is implemented but not hardware-verified in this build context", map[string]any{"transport": tc.Name, "type": tc.Type, "source": tc.SourceLabel()})
		}
		a.persistTransportRuntime(tc, state, detail, "", "")

	}
	a.startWorkers(ctx)
	for _, t := range a.Transports {
		cfgTransport := findTransport(a.Cfg, t.Name())
		if !cfgTransport.Enabled {
			continue
		}
		a.wg.Add(1)
		go func(tr transport.Transport, cfgTransport config.TransportConfig) {
			defer a.wg.Done()
			a.runTransport(ctx, tr, cfgTransport)
		}(t, cfgTransport)
	}
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.Web.Start(ctx)
	}()
	<-ctx.Done()
	for _, t := range a.Transports {
		_ = t.Close(context.Background())
		cfgTransport := findTransport(a.Cfg, t.Name())
		a.persistTransportRuntime(cfgTransport, transport.StateConfiguredNotAttempted, "configured; process stopped", "", "")
	}
	a.wg.Wait()
	return nil
}

func (a *App) startWorkers(ctx context.Context) {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.consumeTransportEvents(ctx)
	}()
	for i := 0; i < defaultIngestWorkers; i++ {
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			a.ingestWorker(ctx)
		}()
	}
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.observationWorker(ctx)
	}()
	if a.Cfg.Integration.Enabled {
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			a.integrationWorker(ctx)
		}()
	}
	if a.DB != nil {
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			a.intelligenceWorker(ctx)
		}()
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			a.controlExecutor(ctx)
		}()
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			a.retentionWorker(ctx)
		}()
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			a.trustCleanupWorker(ctx)
		}()
		if a.Cfg.Topology.Enabled {
			a.wg.Add(1)
			go func() {
				defer a.wg.Done()
				a.topologyWorker(ctx)
			}()
		}
	}

	// Start kernel bridge workers (event bus adapter, federation, region health, etc.)
	if a.kb != nil {
		a.startKernelWorkers(ctx, a.kb)
	}
}

// trustCleanupWorker runs periodic cleanup of expired approval actions, freezes,
// and maintenance windows. Runs every 60 seconds.
func (a *App) trustCleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.cleanupExpiredApprovals()
		}
	}
}

func (a *App) retentionWorker(ctx context.Context) {
	if a == nil || a.DB == nil {
		return
	}
	interval := time.Duration(a.Cfg.Intelligence.Retention.PruneEverySeconds) * time.Second
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := retention.Run(a.DB, a.Cfg); err != nil {
				a.Log.Error("retention_worker_failed", "failed to prune retained data", map[string]any{"error": err.Error()})
			}
		}
	}
}

func (a *App) runTransport(ctx context.Context, t transport.Transport, cfgTransport config.TransportConfig) {
	baseBackoff := time.Duration(a.Cfg.RateLimits.TransportReconnectSeconds) * time.Second
	if cfgTransport.ReconnectSeconds > 0 {
		baseBackoff = time.Duration(cfgTransport.ReconnectSeconds) * time.Second
	}
	if baseBackoff <= 0 {
		baseBackoff = 10 * time.Second
	}
	retryThreshold := cfgTransport.MaxTimeouts
	if retryThreshold <= 0 {
		retryThreshold = 3
	}
	controller, _ := t.(transport.RuntimeStateController)
	consecutiveFailures := 0
	thresholdRecorded := false
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		a.syncTransportRuntime(cfgTransport, t)
		a.Log.Debug("transport_attempt", "attempting transport connect", map[string]any{"transport": t.Name(), "type": cfgTransport.Type, "source": cfgTransport.SourceLabel()})
		if err := t.Connect(ctx); err != nil {
			consecutiveFailures++
			episodeID, failureCount := beginFailureEpisode(controller, err)
			if thresholdRecorded && controller != nil {
				controller.ForceState(transport.StateFailed, "retry threshold already exceeded; waiting for successful recovery", err.Error())
			} else if controller != nil {
				controller.ForceState(transport.StateRetrying, "connect failed; retry backoff active", err.Error())
			}
			a.persistTransportRuntime(cfgTransport, stateForFailure(consecutiveFailures, retryThreshold), failureDetail(consecutiveFailures, retryThreshold, "connect failed; retry backoff active"), err.Error(), "")
			a.Log.Error("transport_failed", "transport connect failed", map[string]any{"transport": t.Name(), "type": cfgTransport.Type, "error": err.Error(), "failure_count": failureCount, "episode_id": episodeID})
			a.insertAuditLog("transport", "error", "transport connect failed", map[string]any{"transport": t.Name(), "error": err.Error(), "phase": "connect", "failure_count": failureCount, "episode_id": episodeID})
			if consecutiveFailures >= retryThreshold && !thresholdRecorded {
				thresholdRecorded = true
				if controller != nil {
					controller.ForceState(transport.StateFailed, "transport retry threshold exceeded during connect", err.Error())
				}
				a.emitTransportObservation(cfgTransport, transport.NewObservation(cfgTransport.Name, cfgTransport.Type, cfgTransport.Topic, transport.ReasonRetryThresholdExceeded, nil, true, "transport retry threshold exceeded during connect", map[string]any{
					"error":                err.Error(),
					"retry_threshold":      retryThreshold,
					"consecutive_failures": consecutiveFailures,
					"failure_count":        failureCount,
					"episode_id":           episodeID,
					"last_error":           err.Error(),
					"phase":                "connect",
					"reconnect_seconds":    baseBackoff.Seconds(),
				}))
			}
			a.syncTransportRuntime(cfgTransport, t)
			if !sleepWithControl(ctx, a.interruptCh(cfgTransport.Name), jitterBackoff(a.effectiveBackoff(baseBackoff, cfgTransport.Name, time.Now().UTC()), consecutiveFailures)) {
				return
			}
			continue
		}
		consecutiveFailures = 0
		thresholdRecorded = false
		if controller != nil {
			controller.CloseFailureEpisode()
		}
		a.clearDeadLetterEpisode(cfgTransport.Name, transport.ReasonRetryThresholdExceeded)
		a.Log.Info("transport_connected", "transport connected", map[string]any{"transport": t.Name(), "type": cfgTransport.Type, "source": cfgTransport.SourceLabel()})
		a.insertAuditLog("transport", "info", "transport connected", map[string]any{"transport": t.Name(), "type": cfgTransport.Type, "source": cfgTransport.SourceLabel()})
		a.syncTransportRuntime(cfgTransport, t)
		handler := func(topic string, payload []byte) error { return a.enqueueIngest(ctx, t, topic, payload) }
		if err := t.Subscribe(ctx, handler); err != nil && ctx.Err() == nil {
			consecutiveFailures++
			episodeID, failureCount := beginFailureEpisode(controller, err)
			if thresholdRecorded && controller != nil {
				controller.ForceState(transport.StateFailed, "retry threshold already exceeded; waiting for successful recovery", err.Error())
			} else if controller != nil {
				controller.ForceState(transport.StateRetrying, "subscribe failed; retry backoff active", err.Error())
			}
			a.persistTransportRuntime(cfgTransport, stateForFailure(consecutiveFailures, retryThreshold), failureDetail(consecutiveFailures, retryThreshold, "subscribe failed; retry backoff active"), err.Error(), "")
			a.Log.Error("transport_failed", "transport subscribe failed", map[string]any{"transport": t.Name(), "error": err.Error(), "failure_count": failureCount, "episode_id": episodeID})
			a.insertAuditLog("transport", "error", "transport subscribe failed", map[string]any{"transport": t.Name(), "error": err.Error(), "phase": "subscribe", "failure_count": failureCount, "episode_id": episodeID})
			if consecutiveFailures >= retryThreshold && !thresholdRecorded {
				thresholdRecorded = true
				if controller != nil {
					controller.ForceState(transport.StateFailed, "transport retry threshold exceeded during subscribe", err.Error())
				}
				a.emitTransportObservation(cfgTransport, transport.NewObservation(cfgTransport.Name, cfgTransport.Type, cfgTransport.Topic, transport.ReasonRetryThresholdExceeded, nil, true, "transport retry threshold exceeded during subscribe", map[string]any{
					"error":                err.Error(),
					"retry_threshold":      retryThreshold,
					"consecutive_failures": consecutiveFailures,
					"failure_count":        failureCount,
					"episode_id":           episodeID,
					"last_error":           err.Error(),
					"phase":                "subscribe",
					"reconnect_seconds":    baseBackoff.Seconds(),
				}))
			}
		} else {
			consecutiveFailures = 0
			thresholdRecorded = false
			if controller != nil {
				controller.CloseFailureEpisode()
			}
			a.clearDeadLetterEpisode(cfgTransport.Name, transport.ReasonRetryThresholdExceeded)
		}
		a.syncTransportRuntime(cfgTransport, t)
		_ = t.Close(context.Background())
		a.syncTransportRuntime(cfgTransport, t)
		if !sleepWithControl(ctx, a.interruptCh(cfgTransport.Name), jitterBackoff(a.effectiveBackoff(baseBackoff, cfgTransport.Name, time.Now().UTC()), maxInt(consecutiveFailures, 1))) {
			return
		}
	}
}

func beginFailureEpisode(controller transport.RuntimeStateController, err error) (string, uint64) {
	if controller == nil {
		return "", 0
	}
	return controller.BeginFailureEpisode(err)
}

func stateForFailure(consecutiveFailures, threshold int) string {
	if consecutiveFailures >= threshold {
		return transport.StateFailed
	}
	return transport.StateRetrying
}

func failureDetail(consecutiveFailures, threshold int, detail string) string {
	if consecutiveFailures >= threshold {
		return "retry threshold exceeded; operator intervention or successful recovery required"
	}
	return detail
}

func jitterBackoff(base time.Duration, attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	max := base << (attempt - 1)
	if max > 30*time.Second {
		max = 30 * time.Second
	}
	jitter := time.Duration((attempt*137)%1000) * time.Millisecond
	return max + jitter
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func sleepWithControl(ctx context.Context, interrupt <-chan struct{}, d time.Duration) bool {
	if interrupt == nil {
		return sleepWithContext(ctx, d)
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-interrupt:
		return true
	case <-timer.C:
		return true
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (a *App) enqueueIngest(ctx context.Context, t transport.Transport, topic string, payload []byte) error {
	req := ingestRequest{transport: t, topic: topic, payload: append([]byte(nil), payload...)}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case a.ingestCh <- req:
		return nil
	default:
		cfgTransport := findTransport(a.Cfg, t.Name())
		if controller, ok := t.(transport.RuntimeStateController); ok {
			controller.RecordObservationDrop(1)
		}
		a.emitTransportObservation(cfgTransport, transport.NewObservation(cfgTransport.Name, cfgTransport.Type, topic, transport.ReasonObservationDropped, payload, false, "ingest queue full; packet dropped to preserve transport liveness", map[string]any{"queue": "ingest", "queue_capacity": cap(a.ingestCh), "drop_count": 1, "drop_cause": "ingest_queue_saturation"}))
		return fmt.Errorf("ingest queue full for transport %s", t.Name())
	}
}

func (a *App) ingestWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-a.ingestCh:
			if req.transport == nil {
				continue
			}
			_ = a.ingest(req.transport, req.topic, req.payload)
		}
	}
}

func (a *App) consumeTransportEvents(ctx context.Context) {
	if a.Bus == nil {
		return
	}
	eventsCh := a.Bus.Subscribe()
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-eventsCh:
			if evt.Type != "transport.observation" {
				continue
			}
			obs, ok := evt.Data.(transport.Observation)
			if !ok || !obs.Valid() {
				a.Log.Error("transport_observation_invalid", "ignored malformed transport observation", map[string]any{"event_type": evt.Type})
				continue
			}
			// Block until the observation worker accepts the event. A non-blocking send
			// dropped observations under parallel test load when the bounded queue briefly
			// filled, causing flaky dead-letter persistence tests without improving safety.
			select {
			case <-ctx.Done():
				return
			case a.observationCh <- obs:
			}
		}
	}
}

func (a *App) observationWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case obs := <-a.observationCh:
			if !obs.Valid() {
				continue
			}
			obs.DeadLetter = transport.ShouldDeadLetter(obs.Reason, obs.Details)
			if obs.DeadLetter {
				a.recordTransportDeadLetter(findTransport(a.Cfg, obs.TransportName), obs.Topic, obs.Reason, obs.PayloadHex, mergeDetails(obs.Detail, obs.Details))
			}
			// Escalate to persistent incident if severe
			a.escalateToIncident(obs)

			if a.shouldSuppressObservationAudit(obs) {
				continue
			}
			a.insertAuditLog("transport", severityForObservation(obs), obs.Reason, map[string]any{
				"transport":      obs.TransportName,
				"type":           obs.TransportType,
				"topic":          obs.Topic,
				"detail":         obs.Detail,
				"payload_hex":    obs.PayloadHex,
				"dead_letter":    obs.DeadLetter,
				"observation_id": obs.ObservationID,
				"episode_id":     obs.EpisodeID,
				"timestamp":      obs.Timestamp,
				"details":        obs.Details,
			})
		}
	}
}

func (a *App) escalateToIncident(obs transport.Observation) {
	if a.DB == nil {
		return
	}
	severity := severityForObservation(obs)
	if !obs.DeadLetter && severity != "error" {
		return
	}

	// Deterministic ID to avoid duplicates for the same root cause
	incidentID := fmt.Sprintf("inc-%s-%s", obs.TransportName, strings.ReplaceAll(obs.Reason, " ", "_"))

	// Check if already exists and its state
	existing, found, err := a.DB.IncidentByID(incidentID)
	if err == nil && found && existing.State == "resolved" {
		// If resolved, we create a new one with a timestamp-suffixed ID or just reuse?
		// Minimal entropy: if it was resolved, a new occurrence means a NEW incident.
		incidentID = fmt.Sprintf("%s-%d", incidentID, time.Now().Unix())
	}

	incident := models.Incident{
		ID:           incidentID,
		Category:     "transport",
		Severity:     severity,
		Title:        fmt.Sprintf("Transport Incident: %s", obs.Reason),
		Summary:      obs.Detail,
		ResourceType: "transport",
		ResourceID:   obs.TransportName,
		State:        "open",
		OccurredAt:   obs.Timestamp,
		Metadata: map[string]any{
			"observation_id": obs.ObservationID,
			"episode_id":     obs.EpisodeID,
			"dead_letter":    obs.DeadLetter,
			"details":        obs.Details,
		},
	}
	if obs.DeadLetter && incident.Category == "transport" && severity == "warning" {
		incident.Category = "data_integrity"
	}

	if err := a.DB.UpsertIncident(incident); err != nil {
		a.Log.Error("incident_escalation_failed", "failed to escalate observation to incident", map[string]any{"incident_id": incidentID, "error": err.Error()})
	}
}

func (a *App) transportByName(name string) transport.Transport {
	for _, t := range a.Transports {
		if t.Name() == name {
			return t
		}
	}
	return nil
}

func (a *App) recordTransportDeadLetter(tc config.TransportConfig, topic, reason, payloadHex string, details map[string]any) {
	if a.DB == nil {
		return
	}
	episodeKey := deadLetterEpisodeKey(tc.Name, reason)
	episodeFingerprint := deadLetterEpisodeFingerprint(topic, payloadHex, details)
	now := time.Now().UTC()
	a.dlMu.Lock()
	if previous, ok := a.dlEpisodes[episodeKey]; ok && previous.fingerprint == episodeFingerprint && now.Sub(previous.recordedAt) < deadLetterSuppressionWindow {
		a.dlMu.Unlock()
		return
	}
	a.dlEpisodes[episodeKey] = deadLetterEpisode{fingerprint: episodeFingerprint, recordedAt: now}
	a.dlMu.Unlock()
	detailsCopy := map[string]any{}
	for k, v := range details {
		detailsCopy[k] = v
	}
	if tc.Name != "" {
		detailsCopy["transport_name"] = tc.Name
	}
	if tc.Type != "" {
		detailsCopy["transport_type"] = tc.Type
	}
	if tc.SourceLabel() != "" {
		detailsCopy["source"] = tc.SourceLabel()
	}
	if topic == "" {
		topic = tc.Topic
	}
	if err := a.DB.InsertDeadLetter(db.DeadLetter{
		TransportName: tc.Name,
		TransportType: tc.Type,
		Topic:         topic,
		Reason:        reason,
		PayloadHex:    payloadHex,
		Details:       detailsCopy,
	}); err != nil {
		a.Log.Error("dead_letter_persist_failed", "failed to persist transport dead-letter", map[string]any{"transport": tc.Name, "reason": reason, "error": err.Error()})
	}
}

func (a *App) shouldSuppressObservationAudit(obs transport.Observation) bool {
	if a == nil || a.DB == nil || obs.DeadLetter || observationDetailBool(obs.Details, "final") {
		return false
	}
	fingerprint := deadLetterEpisodeFingerprint(obs.Topic, obs.PayloadHex, mergeDetails(obs.Detail, obs.Details))
	key := "audit|" + deadLetterEpisodeKey(obs.TransportName, obs.Reason)
	now := time.Now().UTC()
	a.dlMu.Lock()
	defer a.dlMu.Unlock()
	if previous, ok := a.observationEpisodes[key]; ok && previous.fingerprint == fingerprint && now.Sub(previous.recordedAt) < observationAuditSuppressionWindow {
		return true
	}
	a.observationEpisodes[key] = deadLetterEpisode{fingerprint: fingerprint, recordedAt: now}
	return false
}

func observationDetailBool(details map[string]any, key string) bool {
	if len(details) == 0 {
		return false
	}
	value, ok := details[key]
	if !ok {
		return false
	}
	boolean, ok := value.(bool)
	return ok && boolean
}

func (a *App) emitTransportObservation(tc config.TransportConfig, obs transport.Observation) {
	if a.Bus == nil {
		return
	}
	if obs.TransportName == "" {
		obs.TransportName = tc.Name
	}
	if obs.TransportType == "" {
		obs.TransportType = tc.Type
	}
	if obs.Topic == "" {
		obs.Topic = tc.Topic
	}
	_, dropped := a.Bus.Publish(events.Event{Type: "transport.observation", Data: obs})
	if dropped > 0 {
		a.insertAuditLog("transport", "warning", transport.ReasonObservationDropped, map[string]any{
			"transport":   obs.TransportName,
			"type":        obs.TransportType,
			"topic":       obs.Topic,
			"dead_letter": false,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
			"drop_count":  dropped,
			"drop_cause":  "event_bus_saturation",
			"details": map[string]any{
				"reason": obs.Reason,
				"queue":  "event_bus",
			},
		})
		if t := a.transportByName(obs.TransportName); t != nil {
			if controller, ok := t.(transport.RuntimeStateController); ok {
				controller.RecordObservationDrop(uint64(dropped))
			}
		}
	}
}

func mergeDetails(detail string, details map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range details {
		out[k] = v
	}
	if strings.TrimSpace(detail) != "" {
		out["detail"] = detail
	}
	return out
}

func severityForObservation(obs transport.Observation) string {
	switch obs.Reason {
	case transport.ReasonRetryThresholdExceeded, transport.ReasonTimeoutFailure, transport.ReasonStreamFailure, transport.ReasonSubscribeFailure:
		return "error"
	default:
		return "warning"
	}
}

func deadLetterEpisodeKey(name, reason string) string {
	return name + "|" + reason
}

func deadLetterEpisodeFingerprint(topic, payloadHex string, details map[string]any) string {
	phase := fmt.Sprint(details["phase"])
	return strings.Join([]string{topic, payloadHex, phase, fmt.Sprint(details["error"]), fmt.Sprint(details["consecutive_failures"])}, "|")
}

func (a *App) clearDeadLetterEpisode(name, reason string) {
	a.dlMu.Lock()
	defer a.dlMu.Unlock()
	delete(a.dlEpisodes, deadLetterEpisodeKey(name, reason))
}

func (a *App) insertAuditLog(category, level, message string, details any) {
	if a.DB == nil {
		return
	}
	if err := a.DB.InsertAuditLog(category, level, message, details); err != nil {
		a.Log.Error("audit_log_persist_failed", "failed to persist audit log", map[string]any{"category": category, "message": message, "error": err.Error()})
	}
}

func (a *App) persistTransportRuntime(tc config.TransportConfig, state, detail, lastError, lastMessageAt string) {
	if a.DB == nil {
		return
	}
	_ = a.DB.UpsertTransportRuntime(db.TransportRuntime{
		Name:          tc.Name,
		Type:          tc.Type,
		Source:        tc.SourceLabel(),
		Enabled:       tc.Enabled,
		State:         state,
		Detail:        detail,
		LastError:     lastError,
		LastMessageAt: lastMessageAt,
	})
}

func findTransport(cfg config.Config, name string) config.TransportConfig {
	for _, t := range cfg.Transports {
		if t.Name == name {
			return t
		}
	}
	return config.TransportConfig{}
}

func (a *App) ingest(t transport.Transport, topic string, payload []byte) error {
	if st := a.transportControls[t.Name()]; st != nil {
		st.deprioritizeSleep()
	}
	now := time.Now().UTC()
	env, err := meshtastic.ParseEnvelope(payload)
	if err != nil {
		a.Log.Error("ingest_dropped", "failed to parse packet", map[string]any{"transport": t.Name(), "error": err.Error()})
		t.MarkDrop("failed to parse packet")
		cfgTransport := findTransport(a.Cfg, t.Name())
		a.emitTransportObservation(cfgTransport, transport.NewObservation(t.Name(), cfgTransport.Type, topic, transport.ReasonDecodeFailure, payload, true, "failed to parse packet", map[string]any{"error": err.Error(), "stage": "parse_envelope", "unrecoverable": true}))
		a.syncTransportRuntime(findTransport(a.Cfg, t.Name()), t)
		return err
	}
	if st := a.transportControls[t.Name()]; st != nil && st.shouldDropSuppressed(int64(env.Packet.From), now) {
		a.Log.Info("ingest_suppressed", "dropped packet under active source suppression window", map[string]any{"transport": t.Name(), "from_node": env.Packet.From})
		t.MarkDrop("source suppressed by guarded control")
		cfgTransport := findTransport(a.Cfg, t.Name())
		a.emitTransportObservation(cfgTransport, transport.NewObservation(t.Name(), cfgTransport.Type, topic, transport.ReasonHandlerRejection, payload, false, "ingest suppressed for attributed noisy source", map[string]any{"from_node": env.Packet.From, "control": "temporarily_suppress_noisy_source"}))
		a.syncTransportRuntime(findTransport(a.Cfg, t.Name()), t)
		return nil
	}
	rxAt := time.Now().UTC()
	rxTime := time.Unix(int64(env.Packet.RXTime), 0).UTC().Format(time.RFC3339)
	if env.Packet.RXTime == 0 {
		rxTime = rxAt.Format(time.RFC3339)
	}
	messageType, payloadJSON, telemetryType, telemetryValue := buildPayloadEnvelope(t.Name(), topic, env)
	node := map[string]any{"node_num": int64(env.Packet.From), "node_id": env.Packet.NodeID, "long_name": env.Packet.LongName, "short_name": env.Packet.ShortName, "last_seen": rxTime, "last_gateway_id": env.GatewayID, "last_snr": float64(env.Packet.RXSNR), "last_rssi": int64(env.Packet.RXRSSI), "lat_redacted": meshtastic.RedactCoord(env.Packet.Lat), "lon_redacted": meshtastic.RedactCoord(env.Packet.Lon), "altitude": int64(env.Packet.Altitude)}
	msg := map[string]any{"transport_name": t.Name(), "packet_id": int64(env.Packet.ID), "dedupe_hash": meshtastic.DedupeHash(env), "channel_id": env.ChannelID, "gateway_id": env.GatewayID, "from_node": int64(env.Packet.From), "to_node": int64(env.Packet.To), "portnum": int64(env.Packet.PortNum), "payload_text": env.Packet.PayloadText, "payload_json": payloadJSON, "raw_hex": env.RawHex, "rx_time": rxTime, "hop_limit": int64(env.Packet.HopLimit), "relay_node": int64(env.Packet.RelayNode)}
	inserted, err := a.DB.PersistIngest(db.IngestRecord{Message: msg, Node: node, TelemetryType: telemetryType, TelemetryValue: telemetryValue, ObservedAt: rxAt.Format(time.RFC3339)})
	if err != nil {
		a.Log.Error("db_error", "message insert failed", map[string]any{"transport": t.Name(), "error": err.Error()})
		t.MarkDrop("database write failed")
		cfgTransport := findTransport(a.Cfg, t.Name())
		a.emitTransportObservation(cfgTransport, transport.NewObservation(t.Name(), cfgTransport.Type, topic, transport.ReasonHandlerRejection, []byte(env.RawHex), true, "database write failed", map[string]any{"error": err.Error(), "from_node": env.Packet.From, "packet_id": env.Packet.ID, "stage": "insert_message", "final": true}))
		a.syncTransportRuntime(findTransport(a.Cfg, t.Name()), t)
		return err
	}
	if !inserted {
		a.Log.Info("ingest_dropped", "duplicate message ignored", map[string]any{"transport": t.Name(), "dedupe_hash": msg["dedupe_hash"]})
		t.MarkDrop("duplicate packet ignored after dedupe")
		a.syncTransportRuntime(findTransport(a.Cfg, t.Name()), t)
		return nil
	}
	a.State.UpsertNode(meshstate.Node{Num: int64(env.Packet.From), ID: env.Packet.NodeID, LongName: env.Packet.LongName, ShortName: env.Packet.ShortName, LastSeen: rxTime, GatewayID: env.GatewayID})
	a.State.IncMessages()
	if a.topo != nil {
		cfgTransport := findTransport(a.Cfg, t.Name())
		lat := meshtastic.RedactCoord(env.Packet.Lat)
		lon := meshtastic.RedactCoord(env.Packet.Lon)
		_ = a.topo.ApplyPacketEvidence(a.Cfg, t.Name(), cfgTransport.Type, int64(env.Packet.From), env.Packet.To, env.Packet.RelayNode, rxTime, env.GatewayID, float64(env.Packet.RXSNR), int64(env.Packet.RXRSSI), env.Packet.HopLimit, lat, lon, int64(env.Packet.Altitude))
	}
	t.MarkIngest(rxAt)
	summary := strings.TrimSpace(env.Packet.PayloadText)
	if summary == "" {
		summary = fmt.Sprintf("%s packet", messageType)
	}
	evt := events.Event{Type: "meshtastic.packet", Data: fmt.Sprintf("%s packet from %d (%s)", t.Name(), env.Packet.From, summary)}
	a.Bus.Publish(evt)
	for _, p := range a.Plugins {
		if alert := p.Handle(evt); alert != nil {
			_ = a.DB.InsertAuditLog("plugin", "warning", alert.Message, alert)
		}
	}
	_ = a.DB.InsertAuditLog("node", "info", "node observed via transport", map[string]any{"transport": t.Name(), "topic": topic, "node_num": env.Packet.From, "node_id": env.Packet.NodeID, "gateway_id": env.GatewayID})
	a.syncTransportRuntime(findTransport(a.Cfg, t.Name()), t)
	a.Log.Info("ingest_received", "message persisted", map[string]any{"transport": t.Name(), "message_type": messageType, "from_node": env.Packet.From, "portnum": env.Packet.PortNum})
	return nil
}

func (a *App) syncTransportRuntime(tc config.TransportConfig, t transport.Transport) {
	if a.DB == nil {
		return
	}
	h := t.Health()
	if a.Cfg.Integration.Enabled && a.Cfg.Integration.StateChanges {
		a.healthMu.Lock()
		prev := a.lastTransportHealth[tc.Name]
		a.lastTransportHealth[tc.Name] = h
		a.healthMu.Unlock()
		if strings.TrimSpace(prev.State) != "" && !transportStatesEqual(prev.State, h.State) {
			a.publishTransportStateChange(tc.Name, tc.Type, prev.State, h.State, h.Detail)
		}
	}
	lastMessageAt := h.LastIngestAt
	if lastMessageAt == "" {
		lastMessageAt = h.LastSuccessAt
	}
	_ = a.DB.UpsertTransportRuntime(db.TransportRuntime{
		Name:                tc.Name,
		Type:                tc.Type,
		Source:              tc.SourceLabel(),
		Enabled:             tc.Enabled,
		State:               h.State,
		Detail:              h.Detail,
		LastAttemptAt:       h.LastAttemptAt,
		LastConnectedAt:     h.LastConnectedAt,
		LastSuccessAt:       h.LastSuccessAt,
		LastMessageAt:       lastMessageAt,
		LastHeartbeatAt:     h.LastHeartbeatAt,
		LastFailureAt:       h.LastFailureAt,
		LastObservationDrop: h.LastObservationDropAt,
		LastError:           h.LastError,
		EpisodeID:           h.EpisodeID,
		TotalMessages:       h.TotalMessages,
		PacketsDropped:      h.PacketsDropped,
		Reconnects:          h.ReconnectAttempts,
		Timeouts:            h.ConsecutiveTimeouts,
		FailureCount:        h.FailureCount,
		ObservationDrops:    h.ObservationDrops,
	})
}

func buildPayloadEnvelope(transportName, topic string, env meshtastic.Envelope) (string, map[string]any, string, map[string]any) {
	messageType := meshtastic.MessageType(env.Packet)
	payloadJSON := map[string]any{
		"node_id":         env.Packet.NodeID,
		"long_name":       env.Packet.LongName,
		"short_name":      env.Packet.ShortName,
		"topic":           topic,
		"channel_id":      env.ChannelID,
		"gateway_id":      env.GatewayID,
		"transport_name":  transportName,
		"message_type":    messageType,
		"raw_payload_hex": env.Packet.PayloadHex(),
	}
	telemetryType := ""
	telemetryValue := map[string]any{}
	switch messageType {
	case "position":
		payloadJSON["position"] = map[string]any{"lat": meshtastic.RedactCoord(env.Packet.Lat), "lon": meshtastic.RedactCoord(env.Packet.Lon), "altitude": env.Packet.Altitude}
		telemetryType = "position"
		telemetryValue = map[string]any{"lat_redacted": meshtastic.RedactCoord(env.Packet.Lat), "lon_redacted": meshtastic.RedactCoord(env.Packet.Lon), "altitude": int64(env.Packet.Altitude), "transport_name": transportName}
	case "node_info":
		payloadJSON["user"] = map[string]any{"node_id": env.Packet.NodeID, "long_name": env.Packet.LongName, "short_name": env.Packet.ShortName}
	case "telemetry":
		payloadJSON["telemetry"] = map[string]any{"parser": "raw", "note": "payload stored as raw bytes because this repo does not vendor the full telemetry protobuf schema"}
		telemetryType = "telemetry_raw"
		telemetryValue = map[string]any{"transport_name": transportName, "raw_payload_hex": env.Packet.PayloadHex(), "portnum": env.Packet.PortNum}
	case "text":
		payloadJSON["text"] = strings.TrimSpace(env.Packet.PayloadText)
	default:
		payloadJSON["unknown"] = true
	}
	return messageType, payloadJSON, telemetryType, telemetryValue
}

func enabledTransportConfigs(cfg config.Config) []config.TransportConfig {
	out := make([]config.TransportConfig, 0, len(cfg.Transports))
	for _, t := range cfg.Transports {
		if t.Enabled {
			out = append(out, t)
		}
	}
	return out
}
