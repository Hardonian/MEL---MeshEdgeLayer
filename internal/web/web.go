package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/version"

	"github.com/mel-project/mel/internal/auth"
	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/diagnostics"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/logging"
	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/meshstate"
	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/operatorlang"
	"github.com/mel-project/mel/internal/policy"
	"github.com/mel-project/mel/internal/privacy"
	"github.com/mel-project/mel/internal/readiness"
	"github.com/mel-project/mel/internal/security"
	statuspkg "github.com/mel-project/mel/internal/status"
	"github.com/mel-project/mel/internal/support"
	"github.com/mel-project/mel/internal/transport"
)

type Server struct {
	cfg              config.Config
	configPath       string
	log              *logging.Logger
	db               *db.DB
	state            *meshstate.State
	bus              *events.Bus
	http             *http.Server
	transportHealth  func() []transport.Health
	recommendations  func() []policy.Recommendation
	statusSnapshot   func() (statuspkg.Snapshot, error)
	controlStatus    func() (map[string]any, error)
	controlHistory   func(string, string, string, string, int, int) (map[string]any, error)
	diagnosticsRun   func(config.Config, *db.DB) []diagnostics.Finding
	operatorBriefing func() models.OperatorBriefingDTO
	queueDepths      func() map[string]int
	// processStartedAt is set by SetProcessStartedAt when running under mel serve; zero means status was assembled outside a long-lived server (e.g. CLI-only).
	processStartedAt time.Time

	// Federation hooks
	federationHandlers *FederationHandlers

	// Trust / operability hooks (wired from service layer)
	approveAction           func(actionID, actorID, note string, breakGlassSodAck bool, breakGlassSodReason string) (*models.ApproveActionResponse, error)
	rejectAction            func(actionID, actorID, note string, breakGlassSodAck bool, breakGlassSodReason string) (*models.RejectActionResponse, error)
	createFreeze            func(scopeType, scopeValue, reason, createdBy, expiresAt string) (string, error)
	clearFreeze             func(freezeID, clearedBy string) error
	createMaintenanceWindow func(title, reason, scopeType, scopeValue, createdBy, startsAt, endsAt string) (string, error)
	cancelMaintenanceWindow func(windowID, cancelledBy string) error
	addOperatorNote         func(refType, refID, actorID, content string) (string, error)
	timeline                func(start, end string, limit int) ([]db.TimelineEvent, error)
	inspectAction           func(actionID string) (map[string]any, error)
	operationalState        func() (map[string]any, error)
	incidentHandoff         func(incidentID, fromActor string, req models.IncidentHandoffRequest) error
	incidentByID            func(id string) (models.Incident, bool, error)
	recentIncidents         func(limit int) ([]models.Incident, error)
	queueOperatorControl    func(actorID, actionType, targetTransport, targetSegment, targetNode, reason string, confidence float64, incidentID string) (string, error)

	// Topology: optional callback — true when at least one enabled transport is live or idle (ingest-capable).
	topologyTransportLive func() bool
	// meshIntelLatest returns the last mesh deployment intelligence assessment from the service (if any).
	meshIntelLatest func() (meshintel.Assessment, bool)
}

// SetConfigPath records the on-disk config path used at process start (for support bundle doctor.json parity).
func (s *Server) SetConfigPath(path string) {
	s.configPath = strings.TrimSpace(path)
}

// SetProcessStartedAt records when the API process started so status snapshots include PID, uptime, and process-scoped truth.
func (s *Server) SetProcessStartedAt(t time.Time) {
	if t.IsZero() {
		s.processStartedAt = time.Time{}
		return
	}
	s.processStartedAt = t.UTC()
}

func (s *Server) SetQueueDepthsFunc(f func() map[string]int) {
	s.queueDepths = f
}

func (s *Server) SetTrustFuncs(
	approve func(string, string, string, bool, string) (*models.ApproveActionResponse, error),
	reject func(string, string, string, bool, string) (*models.RejectActionResponse, error),
	createFreeze func(string, string, string, string, string) (string, error),
	clearFreeze func(string, string) error,
	createMW func(string, string, string, string, string, string, string) (string, error),
	cancelMW func(string, string) error,
	addNote func(string, string, string, string) (string, error),
	timeline func(string, string, int) ([]db.TimelineEvent, error),
	inspect func(string) (map[string]any, error),
	opState func() (map[string]any, error),
) {
	s.approveAction = approve
	s.rejectAction = reject
	s.createFreeze = createFreeze
	s.clearFreeze = clearFreeze
	s.createMaintenanceWindow = createMW
	s.cancelMaintenanceWindow = cancelMW
	s.addOperatorNote = addNote
	s.timeline = timeline
	s.inspectAction = inspect
	s.operationalState = opState
}

// SetIncidentCollaboration wires incident handoff and lookup for multi-operator coordination.
func (s *Server) SetIncidentCollaboration(
	handoff func(string, string, models.IncidentHandoffRequest) error,
	byID func(string) (models.Incident, bool, error),
) {
	s.incidentHandoff = handoff
	s.incidentByID = byID
}

// SetRecentIncidents wires list enrichment (e.g. linked control actions); falls back to DB when nil.
func (s *Server) SetRecentIncidents(f func(limit int) ([]models.Incident, error)) {
	s.recentIncidents = f
}

// SetOperatorControlQueue wires operator-initiated control action enqueue (canonical service path).
func (s *Server) SetOperatorControlQueue(f func(string, string, string, string, string, string, float64, string) (string, error)) {
	s.queueOperatorControl = f
}

// SetTopologyTransportLive sets a callback used by GET /api/v1/topology for explicit transport connectivity in the intelligence bundle.
func (s *Server) SetTopologyTransportLive(f func() bool) {
	s.topologyTransportLive = f
}

// SetMeshIntelProvider wires the latest persisted in-memory mesh intelligence snapshot (from topology worker).
func (s *Server) SetMeshIntelProvider(f func() (meshintel.Assessment, bool)) {
	s.meshIntelLatest = f
}

func New(cfg config.Config, log *logging.Logger, d *db.DB, st *meshstate.State, bus *events.Bus, th func() []transport.Health, rec func() []policy.Recommendation, statusSnapshot func() (statuspkg.Snapshot, error), controlStatus func() (map[string]any, error), controlHistory func(string, string, string, string, int, int) (map[string]any, error), diagnosticsRun func(config.Config, *db.DB) []diagnostics.Finding, operatorBriefing func() models.OperatorBriefingDTO) *Server {
	controlStatusFn := controlStatus
	if controlStatusFn == nil {
		controlStatusFn = func() (map[string]any, error) {
			return map[string]any{"mode": cfg.Control.Mode, "status": "control unavailable without service control hooks"}, nil
		}
	}
	controlHistoryFn := controlHistory
	if controlHistoryFn == nil {
		controlHistoryFn = func(start, end, transport, lifecycle string, limit, offset int) (map[string]any, error) {
			return map[string]any{"actions": []any{}, "decisions": []any{}, "start": start, "end": end, "transport": transport, "lifecycle_state": lifecycle, "pagination": map[string]any{"limit": limit, "offset": offset}}, nil
		}
	}
	diagnosticsRunFn := diagnosticsRun
	if diagnosticsRunFn == nil {
		diagnosticsRunFn = func(cfg config.Config, database *db.DB) []diagnostics.Finding {
			return []diagnostics.Finding{}
		}
	}
	operatorBriefingFn := operatorBriefing
	if operatorBriefingFn == nil {
		operatorBriefingFn = func() models.OperatorBriefingDTO {
			return models.OperatorBriefingDTO{OverallStatus: "unknown", GeneratedAt: time.Now().UTC().Format(time.RFC3339)}
		}
	}
	s := &Server{cfg: cfg, log: log, db: d, state: st, bus: bus, transportHealth: th, recommendations: rec, statusSnapshot: statusSnapshot, controlStatus: controlStatusFn, controlHistory: controlHistoryFn, diagnosticsRun: diagnosticsRunFn, operatorBriefing: operatorBriefingFn}
	if s.statusSnapshot == nil {
		s.statusSnapshot = func() (statuspkg.Snapshot, error) {
			var pt *time.Time
			if !s.processStartedAt.IsZero() {
				t := s.processStartedAt
				pt = &t
			}
			return statuspkg.Collect(cfg, d, s.transportHealth(), pt, s.configPath)
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.requireMethod(s.healthz, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/readyz", s.requireMethod(s.readyz, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/readyz", s.requireMethod(s.apiV1readyz, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/metrics", s.requireMethod(s.metrics, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/version", s.requireMethod(s.versionHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/health/upgrade", s.requireMethod(s.upgradeHealthHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/audit/verify", s.requireMethod(s.auditVerifyHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/status", s.requireMethod(s.status, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/nodes", s.requireMethod(s.nodes, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/transports", s.requireMethod(s.transports, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/privacy/audit", s.requireMethod(s.audit, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/recommendations", s.requireMethod(s.recs, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/logs", s.requireMethod(s.logs, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/dead-letters", s.requireMethod(s.deadLetters, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/status", s.requireMethod(s.status, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/nodes", s.requireMethod(s.nodes, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/node/", s.requireMethod(s.nodeDetail, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/transports", s.requireMethod(s.transports, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/transports/health", s.requireMethod(s.transportHealthSummary, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/transports/alerts", s.requireMethod(s.transportAlerts, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/transports/anomalies", s.requireMethod(s.transportAnomalies, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/transports/health/history", s.requireMethod(s.transportHealthHistory, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/transports/alerts/history", s.requireMethod(s.transportAlertsHistory, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/transports/anomalies/history", s.requireMethod(s.transportAnomaliesHistory, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/transports/inspect/", s.requireMethod(s.transportInspect, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/mesh", s.requireMethod(s.mesh, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/mesh/inspect", s.requireMethod(s.meshInspect, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/messages", s.requireMethod(s.messages, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/metrics", s.requireMethod(s.metrics, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/panel", s.requireMethod(s.panel, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/privacy/audit", s.requireMethod(s.audit, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/policy/explain", s.requireMethod(s.recs, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/events", s.requireMethod(s.logs, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/audit-logs", s.requireMethod(s.logs, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/dead-letters", s.requireMethod(s.deadLetters, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/incidents", s.requireMethod(security.RequireAny([]security.Capability{security.CapReadIncidents, security.CapReadStatus}, s.incidents), http.MethodGet, http.MethodHead))
	// Register static incident paths before /api/v1/incidents/ so they are not captured as incident IDs.
	mux.HandleFunc("/api/v1/incidents/acknowledge", s.requireMethod(security.RequireAny([]security.Capability{security.CapIncidentUpdate, security.CapAcknowledgeAlerts}, s.acknowledgeIncident), http.MethodPost))
	mux.HandleFunc("/api/v1/incidents/escalate", s.requireMethod(security.RequireAny([]security.Capability{security.CapIncidentUpdate, security.CapEscalateAlerts}, s.escalateIncident), http.MethodPost))
	mux.HandleFunc("/api/v1/incidents/resolve", s.requireMethod(security.RequireAny([]security.Capability{security.CapIncidentUpdate, security.CapSuppressAlerts}, s.resolveIncident), http.MethodPost))
	mux.HandleFunc("/api/v1/incidents/", s.requireMethod(s.incidentsPathHandler, http.MethodGet, http.MethodHead, http.MethodPost))
	mux.HandleFunc("/api/v1/diagnostics", s.requireMethod(s.diagnosticsHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/intelligence/briefing", s.requireMethod(s.operatorBriefingHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/support/manifest", s.requireMethod(security.Require(security.CapExportBundle, s.manifestHandler), http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/support-bundle", s.requireMethod(security.Require(security.CapExportBundle, s.supportBundleHandler), http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/control/status", s.requireMethod(security.RequireAny([]security.Capability{security.CapReadActions, security.CapReadStatus}, s.controlStatusHandler), http.MethodGet, http.MethodHead))
	// Self-observability endpoints
	mux.HandleFunc("/api/v1/health/internal", s.requireMethod(s.InternalHealthHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/health/freshness", s.requireMethod(s.FreshnessHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/health/slo", s.requireMethod(s.SLOHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/metrics/internal", s.requireMethod(s.InternalMetricsHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/health/trust", s.requireMethod(s.TrustHealthHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/control/actions", s.requireMethod(security.RequireAny([]security.Capability{security.CapReadActions, security.CapReadStatus}, s.controlActionsHandler), http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/control/operator-action", s.requireMethod(security.Require(security.CapExecuteAction, s.operatorControlQueueHandler), http.MethodPost))
	mux.HandleFunc("/api/v1/actions", s.requireMethod(security.RequireAny([]security.Capability{security.CapReadActions, security.CapReadStatus}, s.controlActionsHandler), http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/operator-action", s.requireMethod(security.Require(security.CapExecuteAction, s.operatorControlQueueHandler), http.MethodPost))
	mux.HandleFunc("/api/v1/control/history", s.requireMethod(security.RequireAny([]security.Capability{security.CapReadActions, security.CapReadStatus}, s.controlHistoryHandler), http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/config/inspect", s.requireMethod(security.Require(security.CapInspectConfig, s.configInspectHandler), http.MethodGet, http.MethodHead))

	// Trust / operability endpoints
	mux.HandleFunc("/api/v1/control/operational-state", s.requireMethod(security.RequireAny([]security.Capability{security.CapReadActions, security.CapReadStatus}, s.operationalStateHandler), http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/control/actions/", s.requireMethod(s.controlActionSubHandler, http.MethodGet, http.MethodPost))
	mux.HandleFunc("/api/v1/actions/", s.requireMethod(s.controlActionSubHandler, http.MethodGet, http.MethodPost))
	mux.HandleFunc("/api/v1/control/freeze", s.requireMethod(s.freezeHandler, http.MethodGet, http.MethodPost))
	mux.HandleFunc("/api/v1/control/freeze/", s.requireMethod(security.Require(security.CapExecuteAction, s.freezeItemHandler), http.MethodDelete))
	mux.HandleFunc("/api/v1/control/maintenance", s.requireMethod(s.maintenanceHandler, http.MethodGet, http.MethodPost))
	mux.HandleFunc("/api/v1/control/maintenance/", s.requireMethod(security.Require(security.CapExecuteAction, s.maintenanceItemHandler), http.MethodDelete))
	mux.HandleFunc("/api/v1/timeline", s.requireMethod(security.RequireAny([]security.Capability{security.CapReadStatus, security.CapReadIncidents}, s.timelineHandler), http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/operator/notes", s.requireMethod(s.operatorNotesHandler, http.MethodGet, http.MethodPost))

	// Federation / distributed kernel endpoints
	mux.HandleFunc("/api/v1/federation/status", s.requireMethod(s.federationStatusHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/federation/peers", s.requireMethod(s.federationPeersHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/federation/heartbeat", s.requireMethod(s.federationHeartbeatHandler, http.MethodPost))
	mux.HandleFunc("/api/v1/federation/sync", s.requireMethod(s.federationSyncHandler, http.MethodPost))
	mux.HandleFunc("/api/v1/federation/sync/notify", s.requireMethod(s.federationPushNotifyHandler, http.MethodPost))
	mux.HandleFunc("/api/v1/federation/sync/health", s.requireMethod(s.federationSyncHealthHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/kernel/replay", s.requireMethod(s.replayHandler, http.MethodPost))
	mux.HandleFunc("/api/v1/kernel/snapshots", s.snapshotSubHandler)
	mux.HandleFunc("/api/v1/kernel/eventlog/stats", s.requireMethod(s.eventLogStatsHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/kernel/eventlog/query", s.requireMethod(s.eventLogQueryHandler, http.MethodPost))
	mux.HandleFunc("/api/v1/kernel/backpressure", s.requireMethod(s.backpressureStatsHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/kernel/durability", s.requireMethod(s.durabilityStatusHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/kernel/backup", s.requireMethod(s.backupCreateHandler, http.MethodPost))
	mux.HandleFunc("/api/v1/kernel/backups", s.requireMethod(s.backupListHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/topology/global", s.requireMethod(s.globalTopologyHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/topology/region/", s.requireMethod(s.regionHealthHandler, http.MethodGet, http.MethodHead))

	// Topology model endpoints (Phase 1-8: canonical node/link/topology)
	mux.HandleFunc("/api/v1/topology", s.requireMethod(s.topologyIntelligenceHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/topology/nodes", s.requireMethod(s.topologyNodesHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/topology/nodes/", s.requireMethod(s.topologyNodeDetailHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/topology/links", s.requireMethod(s.topologyLinksHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/topology/links/", s.requireMethod(s.topologyLinkDetailHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/topology/segments/", s.requireMethod(s.topologySegmentHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/topology/analysis", s.requireMethod(s.topologyAnalysisHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/topology/snapshots", s.requireMethod(s.topologySnapshotsHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/topology/sources", s.requireMethod(s.sourceTrustHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/topology/bookmarks", s.bookmarksHandler)
	mux.HandleFunc("/api/v1/topology/export", s.requireMethod(s.topologyExportHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/mesh/intelligence/history", s.requireMethod(s.meshIntelligenceHistoryHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/mesh/intelligence", s.requireMethod(s.meshIntelligenceHandler, http.MethodGet, http.MethodHead))
	// Deployment planning (bounded what-if; topology evidence — not RF simulation)
	mux.HandleFunc("/api/v1/planning/bundle", s.requireMethod(s.planningBundleHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/planning/recommend/next", s.requireMethod(s.planningRecommendNextHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/planning/advisory-alerts", s.requireMethod(s.planningAdvisoryAlertsHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/planning/inputs", s.requireMethod(s.planningInputsHandler, http.MethodGet, http.MethodHead, http.MethodPost))
	mux.HandleFunc("/api/v1/planning/input-versions", s.requireMethod(s.planningInputVersionHandler, http.MethodPost))
	mux.HandleFunc("/api/v1/planning/input-versions/", s.requireMethod(s.planningInputVersionGetHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/planning/executions/start", s.requireMethod(s.planningExecutionStartHandler, http.MethodPost))
	mux.HandleFunc("/api/v1/planning/executions/step", s.requireMethod(s.planningExecStepHandler, http.MethodPost))
	mux.HandleFunc("/api/v1/planning/executions/validate", s.requireMethod(s.planningExecValidateHandler, http.MethodPost))
	mux.HandleFunc("/api/v1/planning/executions/validations", s.requireMethod(s.planningExecutionValidationsHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/planning/executions", s.requireMethod(s.planningPlanExecutionsHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/planning/outcomes", s.requireMethod(s.planningOutcomeRecordHandler, http.MethodPost))
	mux.HandleFunc("/api/v1/planning/retrospective", s.requireMethod(s.planningRetrospectiveHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/planning/plans", s.requireMethod(s.planningPlansHandler, http.MethodGet, http.MethodHead, http.MethodPost))
	mux.HandleFunc("/api/v1/planning/plans/", s.requireMethod(s.planningPlanItemHandler, http.MethodGet, http.MethodHead, http.MethodPut))
	mux.HandleFunc("/api/v1/planning/scenario", s.requireMethod(s.planningScenarioHandler, http.MethodPost))
	mux.HandleFunc("/api/v1/planning/compare", s.requireMethod(s.planningCompareHandler, http.MethodPost))
	mux.HandleFunc("/api/v1/planning/impact", s.requireMethod(s.planningImpactHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/planning/playbooks", s.requireMethod(s.planningPlaybooksHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/recovery/state", s.requireMethod(s.recoveryStateHandler, http.MethodGet, http.MethodHead))

	if cfg.Features.WebUI {
		mux.HandleFunc("/", s.requireMethod(s.ui, http.MethodGet, http.MethodHead))
	}

	s.http = &http.Server{Addr: cfg.Bind.API, Handler: s.withSecurityHeaders(s.withAuth(mux)), ReadHeaderTimeout: 5 * time.Second}
	return s
}

func (s *Server) requireMethod(handler http.HandlerFunc, allowedMethods ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, method := range allowedMethods {
			if r.Method == method {
				handler(w, r)
				return
			}
		}
		w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
		s.log.Security("http_method_not_allowed", "invalid HTTP method attempted", "medium", map[string]any{
			"method":  r.Method,
			"path":    r.URL.Path,
			"remote":  remoteClient(r),
			"allowed": allowedMethods,
		})
		writeJSON(w, http.StatusMethodNotAllowed, logging.APIErrorResponse(
			logging.NewSafeError(fmt.Sprintf("Method %s is not allowed for this endpoint", r.Method), nil, "http", false),
		))
	}
}

func (s *Server) withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		if r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:;")
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) Start(ctx context.Context) {
	go func() { <-ctx.Done(); _ = s.http.Shutdown(context.Background()) }()
	s.log.Info("web_start", "web starting", map[string]any{"addr": s.cfg.Bind.API})
	_ = s.http.ListenAndServe()
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg, detail string) {
	body := map[string]any{"error": msg}
	if detail != "" {
		body["detail"] = detail
	}
	writeJSON(w, code, body)
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	s.writeReadinessResponse(w, r)
}

func (s *Server) apiV1readyz(w http.ResponseWriter, r *http.Request) {
	s.writeReadinessResponse(w, r)
}

// writeReadinessResponse serves GET /readyz and GET /api/v1/readyz with identical semantics:
// HTTP 200 only when readiness.Evaluate reports ready; HTTP 503 when snapshot fails or ingest is not proven for enabled transports.
func (s *Server) writeReadinessResponse(w http.ResponseWriter, r *http.Request) {
	snap, err := s.statusSnapshot()
	now := time.Now().UTC()
	if err != nil {
		s.log.Error("readyz_failed", "readiness check failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"api_version":   "v1",
			"ready":         false,
			"status":        "not_ready",
			"reason_codes":  []string{"SNAPSHOT_UNAVAILABLE"},
			"summary":       "Readiness evidence could not be assembled; the HTTP process is up but subsystem status is unavailable.",
			"checked_at":    now.Format(time.RFC3339),
			"process_ready": true,
			"ingest_ready":  false,
			"error_class":   "snapshot_unavailable",
			"message":       "Readiness evidence could not be assembled; the HTTP process is up but subsystem status is unavailable.",
			"operator_next_steps": []string{
				"Inspect logs for database or migration errors.",
				"Verify storage.database_path and permissions.",
				"Run `mel doctor --config <path>` on the host.",
			},
		})
		return
	}
	eval := readiness.Evaluate(s.cfg, snap, true, now)
	next := readinessNextSteps(snap, eval)
	body := map[string]any{
		"api_version":           "v1",
		"ready":                 eval.Ready,
		"status":                eval.Status,
		"reason_codes":          eval.ReasonCodes,
		"summary":               eval.Summary,
		"checked_at":            eval.CheckedAt,
		"process_ready":         eval.ProcessReady,
		"ingest_ready":          eval.IngestReady,
		"stale_ingest_evidence": eval.StaleIngest,
		"snapshot_generated_at": eval.SnapshotAt,
		"schema_version":        eval.SchemaVersion,
		"operator_state":        eval.OperatorState,
		"mesh_state":            eval.MeshState,
		"components":            eval.Components,
		"transports":            snap.Transports,
		"operator_next_steps":   next,
		"product_scope":         snap.Product.ProductScope,
		"instance_id":           snap.Instance.InstanceID,
	}
	code := http.StatusOK
	if !eval.Ready {
		code = http.StatusServiceUnavailable
		body["error_class"] = "not_ready"
		body["message"] = eval.Summary
	}
	writeJSON(w, code, body)
}

func readinessNextSteps(snap statuspkg.Snapshot, eval readiness.Result) []string {
	panel := statuspkg.BuildPanel(snap)
	var steps []string
	if !eval.IngestReady {
		enabled := 0
		for _, t := range snap.Transports {
			if t.Enabled {
				enabled++
			}
		}
		if enabled > 0 {
			steps = append(steps, "Live ingest is not proven on any enabled transport; confirm broker or node reachability and that packets are reaching MEL.")
		}
	}
	if panel.OperatorState == "idle" {
		steps = append(steps, "No transports are configured or all are disabled; MEL will remain idle until you enable serial, TCP, or MQTT.")
	}
	if eval.StaleIngest && eval.IngestReady {
		steps = append(steps, "Last persisted ingest is older than expected; verify traffic is still flowing or investigate transport disconnects.")
	}
	if panel.OperatorState == "degraded" {
		steps = append(steps, "One or more transports are degraded; inspect effective_state, last_error, and guidance in GET /api/v1/status.")
	}
	if len(steps) == 0 {
		steps = append(steps, "Subsystem evidence looks consistent; use GET /api/v1/status for drill-down and transport-level truth.")
	} else {
		steps = append(steps, "Use GET /api/v1/status for evidence-backed transport state and recent errors.")
	}
	if snap.LastSuccessfulIngest == "" && len(snap.Transports) > 0 {
		steps = append(steps, "No messages are persisted yet; first successful SQLite writes define ingest truth.")
	}
	steps = append(steps, "For host-level checks (paths, DB, serial/TCP reachability) run `mel doctor` or `mel preflight` on the server.")
	return steps
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	snap, err := s.statusSnapshot()
	if err != nil {
		s.log.Error("status_snapshot_failed", "status snapshot failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.ClassifyError(err),
		))
		return
	}
	persistedMessages, _ := s.db.Scalar("SELECT COUNT(*) FROM messages;")
	persistedNodes, _ := s.db.Scalar("SELECT COUNT(*) FROM nodes;")
	lastPersistedIngest, _ := s.db.Scalar("SELECT COALESCE(MAX(rx_time), '') FROM messages;")
	writeJSON(w, http.StatusOK, map[string]any{
		"snapshot":           s.state.Snapshot(),
		"runtime_snapshot":   s.state.Snapshot(),
		"persisted_summary":  map[string]any{"messages": persistedMessages, "nodes": persistedNodes, "last_ingest": lastPersistedIngest},
		"status":             snap,
		"panel":              statuspkg.BuildPanel(snap),
		"privacy_summary":    privacy.Summary(privacy.Audit(s.cfg)),
		"bind_local_default": !s.cfg.Bind.AllowRemote,
	})
}

func (s *Server) nodes(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	where := "1=1"
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		safeQ, err := db.ValidateSQLInput(q)
		if err == nil {
			where = fmt.Sprintf("(n.node_id LIKE '%%%s%%' OR n.long_name LIKE '%%%s%%' OR n.short_name LIKE '%%%s%%')", safeQ, safeQ, safeQ)
		}
	}

	query := fmt.Sprintf("SELECT n.node_num,n.node_id,n.long_name,n.short_name,n.last_seen,n.last_gateway_id,n.lat_redacted,n.lon_redacted,n.altitude,n.last_snr,n.last_rssi,(SELECT COUNT(*) FROM messages m WHERE m.from_node=n.node_num) AS message_count FROM nodes n WHERE %s ORDER BY n.updated_at DESC LIMIT %d OFFSET %d;", where, limit, offset)
	rows, err := s.db.QueryRows(query)
	if err != nil {
		s.log.Error("db_query_failed", "database query failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.SanitizeDBError(err),
		))
		return
	}
	nodes := make([]models.Node, 0, len(rows))
	for _, row := range rows {
		nodes = append(nodes, models.Node{
			NodeNum:       asInt(row["node_num"]),
			NodeID:        asString(row["node_id"]),
			LongName:      asString(row["long_name"]),
			ShortName:     asString(row["short_name"]),
			LastSeen:      asString(row["last_seen"]),
			LastGatewayID: asString(row["last_gateway_id"]),
			LatRedacted:   asFloat(row["lat_redacted"]),
			LonRedacted:   asFloat(row["lon_redacted"]),
			Altitude:      asInt(row["altitude"]),
			LastSNR:       asFloat(row["last_snr"]),
			LastRSSI:      asInt(row["last_rssi"]),
			MessageCount:  asInt(row["message_count"]),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"nodes": nodes, "pagination": map[string]int{"limit": limit, "offset": offset}})
}

func (s *Server) nodeDetail(w http.ResponseWriter, r *http.Request) {
	nodeID := strings.TrimPrefix(r.URL.Path, "/api/v1/node/")
	if nodeID == "" {
		writeJSON(w, http.StatusBadRequest, logging.APIErrorResponse(
			logging.NewSafeError("node identifier is required", nil, "validation", false),
		))
		return
	}
	if !db.IsSafeNodeID(nodeID) {
		s.log.Security("suspicious_input", "invalid characters in node identifier", "medium", map[string]any{
			"path":   r.URL.Path,
			"remote": remoteClient(r),
			"node":   nodeID,
		})
		writeJSON(w, http.StatusBadRequest, logging.APIErrorResponse(
			logging.NewSafeError("node identifier contains invalid characters", nil, "validation", false),
		))
		return
	}
	query := fmt.Sprintf("SELECT n.node_num,n.node_id,n.long_name,n.short_name,n.last_seen,n.last_gateway_id,n.lat_redacted,n.lon_redacted,n.altitude,n.last_snr,n.last_rssi,(SELECT COUNT(*) FROM messages m WHERE m.from_node=n.node_num) AS message_count FROM nodes n WHERE n.node_id='%s' LIMIT 1;", db.EscString(nodeID))
	rows, err := s.db.QueryRows(query)
	if err != nil {
		s.log.Error("db_query_failed", "database query failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.SanitizeDBError(err),
		))
		return
	}
	if len(rows) == 0 {
		writeJSON(w, http.StatusNotFound, logging.APIErrorResponse(
			logging.NewSafeError("node not present in local observations", nil, "not_found", false),
		))
		return
	}
	row := rows[0]
	node := models.Node{
		NodeNum:       asInt(row["node_num"]),
		NodeID:        asString(row["node_id"]),
		LongName:      asString(row["long_name"]),
		ShortName:     asString(row["short_name"]),
		LastSeen:      asString(row["last_seen"]),
		LastGatewayID: asString(row["last_gateway_id"]),
		LatRedacted:   asFloat(row["lat_redacted"]),
		LonRedacted:   asFloat(row["lon_redacted"]),
		Altitude:      asInt(row["altitude"]),
		LastSNR:       asFloat(row["last_snr"]),
		LastRSSI:      asInt(row["last_rssi"]),
		MessageCount:  asInt(row["message_count"]),
	}
	writeJSON(w, http.StatusOK, map[string]any{"node": node})
}

func (s *Server) transports(w http.ResponseWriter, r *http.Request) {
	snap, err := s.statusSnapshot()
	if err != nil {
		s.log.Error("transports_failed", "transport data retrieval failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.ClassifyError(err),
		))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"transports": snap.Transports, "configured_modes": snap.ConfiguredTransportModes, "recent_incidents": snap.RecentIncidents, "active_transport_alerts": snap.ActiveTransportAlerts})
}

func (s *Server) transportHealthSummary(w http.ResponseWriter, r *http.Request) {
	snap, err := s.statusSnapshot()
	if err != nil {
		s.log.Error("transport_health_failed", "transport health data retrieval failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.ClassifyError(err),
		))
		return
	}
	health := make([]models.TransportSummary, 0, len(snap.Transports))
	for _, tr := range snap.Transports {
		reasons := make([]string, 0, len(tr.ActiveAlerts))
		for _, alert := range tr.ActiveAlerts {
			reasons = append(reasons, alert.Reason)
		}
		anomalyCount := 0
		for _, w := range tr.RecentAnomalies {
			for _, count := range w.CountsByReason {
				anomalyCount += int(count)
			}
		}
		health = append(health, models.TransportSummary{
			Name:               tr.Name,
			Type:               tr.Type,
			RuntimeState:       tr.RuntimeState,
			EffectiveState:     tr.EffectiveState,
			Health:             tr.Health.Score,
			ActiveAlertCount:   len(tr.ActiveAlerts),
			RecentAnomalyCount: anomalyCount,
			LastFailureAt:      tr.LastFailureAt,
			ActiveAlertReasons: reasons,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"transport_health": health})
}

func (s *Server) transportAlerts(w http.ResponseWriter, r *http.Request) {
	snap, err := s.statusSnapshot()
	if err != nil {
		s.log.Error("transport_alerts_failed", "transport alerts retrieval failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.ClassifyError(err),
		))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"transport_alerts": snap.ActiveTransportAlerts})
}

func (s *Server) transportAnomalies(w http.ResponseWriter, r *http.Request) {
	snap, err := s.statusSnapshot()
	if err != nil {
		s.log.Error("transport_anomalies_failed", "transport anomalies retrieval failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.ClassifyError(err),
		))
		return
	}
	rows := make([]any, 0, len(snap.Transports))
	for _, tr := range snap.Transports {
		rows = append(rows, map[string]any{
			"transport_name":   tr.Name,
			"transport_type":   tr.Type,
			"recent_anomalies": tr.RecentAnomalies,
			"failure_clusters": tr.FailureClusters,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"transport_anomalies": rows})
}

func (s *Server) transportHealthHistory(w http.ResponseWriter, r *http.Request) {
	transportName, start, end, limit, offset, err := historyParams(s.cfg, r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, logging.APIErrorResponse(err))
		return
	}
	rows, err := s.db.TransportHealthSnapshots(transportName, start, end, limit, offset)
	if err != nil {
		s.log.Error("db_query_failed", "database query failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.SanitizeDBError(err),
		))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": rows, "pagination": map[string]any{"limit": limit, "offset": offset}, "transport": transportName, "start": start, "end": end})
}

func (s *Server) transportAlertsHistory(w http.ResponseWriter, r *http.Request) {
	transportName, start, end, limit, offset, err := historyParams(s.cfg, r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, logging.APIErrorResponse(err))
		return
	}
	rows, err := s.db.TransportAlertsHistory(transportName, start, end, limit, offset)
	if err != nil {
		s.log.Error("db_query_failed", "database query failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.SanitizeDBError(err),
		))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": rows, "pagination": map[string]any{"limit": limit, "offset": offset}, "transport": transportName, "start": start, "end": end})
}

func (s *Server) transportAnomaliesHistory(w http.ResponseWriter, r *http.Request) {
	transportName, start, end, limit, offset, err := historyParams(s.cfg, r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, logging.APIErrorResponse(err))
		return
	}
	rows, err := s.db.TransportAnomalyHistory(transportName, start, end, limit, offset)
	if err != nil {
		s.log.Error("db_query_failed", "database query failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.SanitizeDBError(err),
		))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": rows, "pagination": map[string]any{"limit": limit, "offset": offset}, "transport": transportName, "start": start, "end": end})
}

func (s *Server) transportInspect(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/transports/inspect/")
	if strings.TrimSpace(name) == "" {
		writeJSON(w, http.StatusBadRequest, logging.APIErrorResponse(
			logging.NewSafeError("transport name is required", nil, "validation", false),
		))
		return
	}
	drilldown, err := statuspkg.InspectTransport(s.cfg, s.db, s.transportHealth(), name, time.Now().UTC())
	if err != nil {
		writeJSON(w, http.StatusNotFound, logging.APIErrorResponse(
			logging.NewSafeError("transport not found", err, "not_found", false),
		))
		return
	}
	writeJSON(w, http.StatusOK, drilldown)
}

func (s *Server) mesh(w http.ResponseWriter, r *http.Request) {
	snap, err := s.statusSnapshot()
	if err != nil {
		s.log.Error("mesh_failed", "mesh data retrieval failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.ClassifyError(err),
		))
		return
	}
	writeJSON(w, http.StatusOK, snap.Mesh)
}

func (s *Server) meshInspect(w http.ResponseWriter, r *http.Request) {
	drilldown, err := statuspkg.InspectMesh(s.cfg, s.db, s.transportHealth(), time.Now().UTC())
	if err != nil {
		s.log.Error("mesh_inspect_failed", "mesh inspection failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.ClassifyError(err),
		))
		return
	}
	writeJSON(w, http.StatusOK, drilldown)
}

func (s *Server) controlStatusHandler(w http.ResponseWriter, r *http.Request) {
	payload, err := s.controlStatus()
	if err != nil {
		s.log.Error("control_status_failed", "control status query failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.ClassifyError(err),
		))
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) supportBundleHandler(w http.ResponseWriter, r *http.Request) {
	bundle, err := support.Create(s.cfg, s.db, version.GetFullVersionString(), s.configPath, s.processStartedAt)
	if err != nil {
		s.log.Error("support_bundle_failed", "support bundle generation failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.ClassifyError(err),
		))
		return
	}
	z, err := bundle.ToZip()
	if err != nil {
		s.log.Error("support_bundle_zip_failed", "support bundle zip failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.NewSafeError("failed to assemble support bundle archive", err, "internal", false),
		))
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="mel-support-bundle.zip"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(z)
}

func (s *Server) manifestHandler(w http.ResponseWriter, r *http.Request) {
	manifest := models.SupportManifest{
		ID:        fmt.Sprintf("MEL-%d", time.Now().Unix()),
		Version:   version.Version,
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Features:  s.cfg.Features.Active(),
		Checklist: map[string]any{
			"db_connected": s.db != nil,
			"mesh_active":  s.state.MeshActive(),
			"last_audit":   s.db.LastAuditTime(),
			"log_level":    s.cfg.Logging.Level,
		},
	}
	writeJSON(w, http.StatusOK, manifest)
}

func (s *Server) controlActionsHandler(w http.ResponseWriter, r *http.Request) {
	transportName, start, end, limit, offset, err := historyParams(s.cfg, r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, logging.APIErrorResponse(err))
		return
	}
	lifecycleState := strings.TrimSpace(r.URL.Query().Get("lifecycle_state"))
	payload, err := s.controlHistory(start, end, transportName, lifecycleState, limit, offset)
	if err != nil {
		s.log.Error("control_history_failed", "control history query failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.ClassifyError(err),
		))
		return
	}

	dbActions, _ := payload["actions"].([]db.ControlActionRecord)
	actions := make([]models.ActionRecord, 0, len(dbActions))
	for _, row := range dbActions {
		actions = append(actions, models.ActionRecord{
			ID:                row.ID,
			TransportName:     row.TargetTransport,
			TargetNode:        row.TargetNode,
			TargetSegment:     row.TargetSegment,
			ActionType:        row.ActionType,
			LifecycleState:    row.LifecycleState,
			Result:            row.Result,
			Reason:            row.Reason,
			OutcomeDetail:     row.OutcomeDetail,
			CreatedAt:         row.CreatedAt,
			ExecutedAt:        row.ExecutedAt,
			CompletedAt:       row.CompletedAt,
			ExpiresAt:         row.ExpiresAt,
			TriggerEvidence:   row.TriggerEvidence,
			Details:           row.Metadata,
			ExecutionMode:     row.ExecutionMode,
			ProposedBy:        row.ProposedBy,
			ApprovedBy:        row.ApprovedBy,
			ApprovedAt:        row.ApprovedAt,
			RejectedBy:        row.RejectedBy,
			RejectedAt:        row.RejectedAt,
			ApprovalNote:      row.ApprovalNote,
			ApprovalExpiresAt: row.ApprovalExpiresAt,
			BlastRadiusClass:  row.BlastRadiusClass,
			EvidenceBundleID:  row.EvidenceBundleID,
			OperatorView:      operatorlang.ActionOperatorLabels(row),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"actions":         actions,
		"transport":       transportName,
		"lifecycle_state": lifecycleState,
		"start":           start,
		"end":             end,
		"pagination":      payload["pagination"],
	})
}

func (s *Server) controlHistoryHandler(w http.ResponseWriter, r *http.Request) {
	transportName, start, end, limit, offset, err := historyParams(s.cfg, r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, logging.APIErrorResponse(err))
		return
	}
	lifecycleState := strings.TrimSpace(r.URL.Query().Get("lifecycle_state"))
	payload, err := s.controlHistory(start, end, transportName, lifecycleState, limit, offset)
	if err != nil {
		s.log.Error("control_history_failed", "control history query failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.ClassifyError(err),
		))
		return
	}

	dbActions, _ := payload["actions"].([]db.ControlActionRecord)
	dbDecisions, _ := payload["decisions"].([]db.ControlDecisionRecord)

	actions := make([]models.ActionRecord, 0, len(dbActions))
	for _, row := range dbActions {
		actions = append(actions, models.ActionRecord{
			ID:                row.ID,
			TransportName:     row.TargetTransport,
			TargetNode:        row.TargetNode,
			TargetSegment:     row.TargetSegment,
			ActionType:        row.ActionType,
			LifecycleState:    row.LifecycleState,
			Result:            row.Result,
			Reason:            row.Reason,
			OutcomeDetail:     row.OutcomeDetail,
			CreatedAt:         row.CreatedAt,
			ExecutedAt:        row.ExecutedAt,
			CompletedAt:       row.CompletedAt,
			ExpiresAt:         row.ExpiresAt,
			TriggerEvidence:   row.TriggerEvidence,
			Details:           row.Metadata,
			ExecutionMode:     row.ExecutionMode,
			ProposedBy:        row.ProposedBy,
			ApprovedBy:        row.ApprovedBy,
			ApprovedAt:        row.ApprovedAt,
			RejectedBy:        row.RejectedBy,
			RejectedAt:        row.RejectedAt,
			ApprovalNote:      row.ApprovalNote,
			ApprovalExpiresAt: row.ApprovalExpiresAt,
			BlastRadiusClass:  row.BlastRadiusClass,
			EvidenceBundleID:  row.EvidenceBundleID,
			OperatorView:      operatorlang.ActionOperatorLabels(row),
		})
	}

	decisions := make([]models.DecisionRecord, 0, len(dbDecisions))
	for _, row := range dbDecisions {
		decisions = append(decisions, models.DecisionRecord{
			ID:                row.ID,
			CandidateActionID: row.CandidateActionID,
			ActionType:        row.ActionType,
			TargetTransport:   row.TargetTransport,
			Reason:            row.Reason,
			Confidence:        row.Confidence,
			Allowed:           row.Allowed,
			DenialReason:      row.DenialReason,
			CreatedAt:         row.CreatedAt,
			Mode:              row.Mode,
			PolicySummary:     row.PolicySummary,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"actions":         actions,
		"decisions":       decisions,
		"in_flight":       payload["in_flight"],
		"reality_matrix":  payload["reality_matrix"],
		"transport":       transportName,
		"lifecycle_state": lifecycleState,
		"start":           start,
		"end":             end,
		"pagination":      payload["pagination"],
	})
}

func (s *Server) configInspectHandler(w http.ResponseWriter, r *http.Request) {
	// We do not have the original []byte of the file here in the Server struct.
	// We will just pass nil and fingerprint will be empty, which is acceptable for the API
	// unless we modify the server to hold it. For now, this suffices.
	eff := config.Inspect(s.cfg, nil)
	writeJSON(w, http.StatusOK, eff)
}

func historyParams(cfg config.Config, r *http.Request) (string, string, string, int, int, error) {
	transportName := strings.TrimSpace(r.URL.Query().Get("transport"))
	if transportName != "" && !isValidTransportName(transportName) {
		return "", "", "", 0, 0, logging.NewSafeError("invalid transport name: contains forbidden characters", nil, "validation", false)
	}
	start := strings.TrimSpace(r.URL.Query().Get("start"))
	end := strings.TrimSpace(r.URL.Query().Get("end"))
	limit := cfg.Intelligence.Queries.DefaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			if parsed > cfg.Intelligence.Queries.MaxLimit {
				limit = cfg.Intelligence.Queries.MaxLimit
			} else {
				limit = parsed
			}
		}
	}
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			if parsed < 0 {
				return "", "", "", 0, 0, logging.NewSafeError("offset must be >= 0", nil, "validation", false)
			}
			offset = parsed
		}
	}
	return transportName, start, end, limit, offset, nil
}

func isValidTransportName(name string) bool {
	if strings.Contains(name, ";") {
		return false
	}
	if strings.Contains(name, "--") {
		return false
	}
	if strings.Contains(name, "/*") || strings.Contains(name, "*/") {
		return false
	}
	return true
}

func (s *Server) messages(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	clauses := []string{"1=1"}
	if node := r.URL.Query().Get("node"); node != "" {
		if !db.IsSafeNodeID(node) {
			s.log.Security("suspicious_input", "invalid characters in node parameter", "medium", map[string]any{
				"path":   r.URL.Path,
				"remote": remoteClient(r),
				"param":  "node",
			})
			writeJSON(w, http.StatusBadRequest, logging.APIErrorResponse(
				logging.NewSafeError("node parameter contains invalid characters", nil, "validation", false),
			))
			return
		}
		clauses = append(clauses, fmt.Sprintf("from_node=(SELECT node_num FROM nodes WHERE node_id='%s' LIMIT 1)", db.EscString(node)))
	}
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		safeQ, err := db.ValidateSQLInput(q)
		if err == nil {
			clauses = append(clauses, fmt.Sprintf("(payload_text LIKE '%%%s%%' OR payload_json LIKE '%%%s%%')", safeQ, safeQ))
		}
	}
	if messageType := r.URL.Query().Get("type"); messageType != "" {
		if !isSafeIdentifier(messageType) {
			s.log.Security("suspicious_input", "invalid characters in type parameter", "medium", map[string]any{
				"path":   r.URL.Path,
				"remote": remoteClient(r),
				"param":  "type",
			})
			writeJSON(w, http.StatusBadRequest, logging.APIErrorResponse(
				logging.NewSafeError("type parameter contains invalid characters", nil, "validation", false),
			))
			return
		}
		clauses = append(clauses, fmt.Sprintf("payload_json LIKE '%%%s%%'", escape(fmt.Sprintf(`\"message_type\":\"%s\"`, messageType))))
	}
	rows, err := s.db.QueryRows(fmt.Sprintf("SELECT transport_name,packet_id,from_node,to_node,portnum,payload_text,payload_json,rx_time,created_at FROM messages WHERE %s ORDER BY id DESC LIMIT %d OFFSET %d;", strings.Join(clauses, " AND "), limit, offset))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(logging.SanitizeDBError(err)))
		return
	}

	if s.cfg.Privacy.RedactExports {
		rows = privacy.RedactMessages(rows)
	}

	writeJSON(w, http.StatusOK, map[string]any{"messages": rows, "pagination": map[string]int{"limit": limit, "offset": offset}})
}

func (s *Server) panel(w http.ResponseWriter, r *http.Request) {
	snap, err := s.statusSnapshot()
	if err != nil {
		s.log.Error("panel_failed", "panel generation failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.ClassifyError(err),
		))
		return
	}
	panel := statuspkg.BuildPanel(snap)
	if id, ok := security.GetIdentity(r.Context()); ok {
		panel = statuspkg.EnrichPanelAuth(panel, id)
	}
	writeJSON(w, http.StatusOK, panel)
}

func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	snap, err := s.statusSnapshot()
	if err != nil {
		s.log.Error("metrics_failed", "metrics generation failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.ClassifyError(err),
		))
		return
	}
	rateByTransport := map[string]float64{}
	if s.db != nil {
		cutoff := time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339)
		rows, err := s.db.QueryRows(fmt.Sprintf("SELECT transport_name, COUNT(*) AS recent_messages FROM messages WHERE rx_time >= '%s' GROUP BY transport_name;", cutoff))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		for _, row := range rows {
			rateByTransport[fmt.Sprint(row["transport_name"])] = float64(toInt(row["recent_messages"])) / 300.0
		}
	}
	metrics := map[string]any{
		"generated_at":        time.Now().UTC().Format(time.RFC3339),
		"window_seconds":      300,
		"total_messages":      snap.Messages,
		"last_ingest_time":    snap.LastSuccessfulIngest,
		"transport_metrics":   snap.Transports,
		"ingest_rate_per_sec": rateByTransport,
		"dead_letters_total":  totalDeadLetters(snap.Transports),
		"diagnostics":         s.diagnosticsRun(s.cfg, s.db),
	}
	if s.queueDepths != nil {
		metrics["queue_depths"] = s.queueDepths()
	}
	if s.db != nil {
		metrics["control_metrics"] = map[string]any{
			"decisions_total":           scalarInt(s.db, "SELECT COUNT(*) FROM control_decisions;"),
			"executions_total":          scalarInt(s.db, "SELECT COUNT(*) FROM control_actions WHERE result='executed_successfully';"),
			"denials_total":             scalarInt(s.db, "SELECT COUNT(*) FROM control_decisions WHERE allowed=0;"),
			"cooldown_denials":          scalarInt(s.db, "SELECT COUNT(*) FROM control_decisions WHERE denial_code='cooldown';"),
			"override_denials":          scalarInt(s.db, "SELECT COUNT(*) FROM control_decisions WHERE denial_code='override';"),
			"missing_actuator_denials":  scalarInt(s.db, "SELECT COUNT(*) FROM control_decisions WHERE denial_code='missing_actuator';"),
			"active_actions":            scalarInt(s.db, "SELECT COUNT(*) FROM control_actions WHERE lifecycle_state IN ('pending','running') OR (result='executed_successfully' AND reversible=1 AND (expires_at='' OR expires_at > datetime('now')));"),
			"queue_depth":               scalarInt(s.db, "SELECT COUNT(*) FROM control_actions WHERE lifecycle_state='pending';"),
			"execution_latency_seconds": scalarFloat(s.db, "SELECT COALESCE(AVG((julianday(completed_at)-julianday(executed_at))*86400.0),0) FROM control_actions WHERE executed_at != '' AND completed_at != '';"),
			"denials_by_reason": map[string]any{
				"policy":               scalarInt(s.db, "SELECT COUNT(*) FROM control_decisions WHERE denial_code='policy';"),
				"mode":                 scalarInt(s.db, "SELECT COUNT(*) FROM control_decisions WHERE denial_code='mode';"),
				"override":             scalarInt(s.db, "SELECT COUNT(*) FROM control_decisions WHERE denial_code='override';"),
				"low_confidence":       scalarInt(s.db, "SELECT COUNT(*) FROM control_decisions WHERE denial_code='low_confidence';"),
				"transient":            scalarInt(s.db, "SELECT COUNT(*) FROM control_decisions WHERE denial_code='transient';"),
				"cooldown":             scalarInt(s.db, "SELECT COUNT(*) FROM control_decisions WHERE denial_code='cooldown';"),
				"budget":               scalarInt(s.db, "SELECT COUNT(*) FROM control_decisions WHERE denial_code='budget';"),
				"missing_actuator":     scalarInt(s.db, "SELECT COUNT(*) FROM control_decisions WHERE denial_code='missing_actuator';"),
				"unknown_blast_radius": scalarInt(s.db, "SELECT COUNT(*) FROM control_decisions WHERE denial_code='unknown_blast_radius';"),
				"no_alternate_path":    scalarInt(s.db, "SELECT COUNT(*) FROM control_decisions WHERE denial_code='no_alternate_path';"),
				"irreversible":         scalarInt(s.db, "SELECT COUNT(*) FROM control_decisions WHERE denial_code='irreversible';"),
				"conflict":             scalarInt(s.db, "SELECT COUNT(*) FROM control_decisions WHERE denial_code='conflict';"),
				"attribution_weak":     scalarInt(s.db, "SELECT COUNT(*) FROM control_decisions WHERE denial_code='attribution_weak';"),
			},
		}
	}
	writeJSON(w, http.StatusOK, metrics)
}

func (s *Server) audit(w http.ResponseWriter, _ *http.Request) {
	findings := privacy.Audit(s.cfg)
	writeJSON(w, http.StatusOK, map[string]any{"findings": findings, "summary": privacy.Summary(findings)})
}

func (s *Server) recs(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"recommendations": s.recommendations()})
}

func (s *Server) logs(w http.ResponseWriter, r *http.Request) {
	query := "SELECT category,level,message,details_json,created_at FROM audit_logs"
	if transportName := strings.TrimSpace(r.URL.Query().Get("transport")); transportName != "" {
		if !isValidTransportName(transportName) {
			s.log.Security("suspicious_input", "invalid transport name in query", "medium", map[string]any{
				"path":      r.URL.Path,
				"remote":    remoteClient(r),
				"transport": transportName,
			})
			writeJSON(w, http.StatusBadRequest, logging.APIErrorResponse(
				logging.NewSafeError("transport parameter contains invalid characters", nil, "validation", false),
			))
			return
		}
		query += fmt.Sprintf(" WHERE details_json LIKE '%%%s%%'", escape(fmt.Sprintf(`\"transport\":\"%s\"`, transportName)))
	}
	query += " ORDER BY id DESC LIMIT 100;"
	rows, err := s.db.QueryRows(query)
	if err != nil {
		s.log.Error("db_query_failed", "database query failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.SanitizeDBError(err),
		))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": rows})
}

func (s *Server) incidentsPathHandler(w http.ResponseWriter, r *http.Request) {
	const prefix = "/api/v1/incidents/"
	path := r.URL.Path
	if !strings.HasPrefix(path, prefix) {
		writeError(w, http.StatusNotFound, "not found", "")
		return
	}
	rest := strings.TrimPrefix(path, prefix)
	if rest == "" {
		writeError(w, http.StatusBadRequest, "incident id required", "")
		return
	}
	parts := strings.SplitN(rest, "/", 2)
	id := parts[0]
	sub := ""
	if len(parts) > 1 {
		sub = parts[1]
	}
	switch {
	case sub == "" && (r.Method == http.MethodGet || r.Method == http.MethodHead):
		security.RequireAny([]security.Capability{security.CapReadIncidents, security.CapReadStatus}, func(w http.ResponseWriter, r *http.Request) {
			s.incidentDetailHandler(w, r, id)
		})(w, r)
	case sub == "handoff" && r.Method == http.MethodPost:
		security.Require(security.CapIncidentHandoffWrite, func(w http.ResponseWriter, r *http.Request) {
			s.incidentHandoffHandler(w, r, id)
		})(w, r)
	default:
		writeError(w, http.StatusNotFound, "unknown incident path", "")
	}
}

func (s *Server) incidentDetailHandler(w http.ResponseWriter, _ *http.Request, incidentID string) {
	if s.incidentByID == nil {
		writeError(w, http.StatusServiceUnavailable, "incident detail not available", "")
		return
	}
	inc, ok, err := s.incidentByID(incidentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load incident", err.Error())
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "incident not found", "")
		return
	}
	writeJSON(w, http.StatusOK, inc)
}

func (s *Server) incidentHandoffHandler(w http.ResponseWriter, r *http.Request, incidentID string) {
	if s.incidentHandoff == nil {
		writeError(w, http.StatusServiceUnavailable, "handoff not available", "")
		return
	}
	var req models.IncidentHandoffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	from := s.actorFromTrustContext(r)
	if err := s.incidentHandoff(incidentID, from, req); err != nil {
		code := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			code = http.StatusNotFound
		} else if strings.Contains(err.Error(), "required") {
			code = http.StatusBadRequest
		}
		writeError(w, code, "could not record handoff", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "handed_off",
		"incident_id": incidentID,
		"to":          strings.TrimSpace(req.ToOperatorID),
		"from":        from,
	})
}

func (s *Server) incidents(w http.ResponseWriter, r *http.Request) {
	var records []models.Incident
	var err error
	if s.recentIncidents != nil {
		records, err = s.recentIncidents(100)
	} else {
		records, err = s.db.RecentIncidents(100)
	}
	if err != nil {
		s.log.Error("db_query_failed", "database query failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.SanitizeDBError(err),
		))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recent_incidents": records})
}

func (s *Server) deadLetters(w http.ResponseWriter, r *http.Request) {
	query := "SELECT transport_name,transport_type,topic,reason,payload_hex,details_json,created_at FROM dead_letters"
	if transportName := strings.TrimSpace(r.URL.Query().Get("transport")); transportName != "" {
		if !isValidTransportName(transportName) {
			s.log.Security("suspicious_input", "invalid transport name in query", "medium", map[string]any{
				"path":      r.URL.Path,
				"remote":    remoteClient(r),
				"transport": transportName,
			})
			writeJSON(w, http.StatusBadRequest, logging.APIErrorResponse(
				logging.NewSafeError("transport parameter contains invalid characters", nil, "validation", false),
			))
			return
		}
		query += fmt.Sprintf(" WHERE transport_name='%s'", escape(transportName))
	}
	query += " ORDER BY id DESC LIMIT 100;"
	rows, err := s.db.QueryRows(query)
	if err != nil {
		s.log.Error("db_query_failed", "database query failed", map[string]any{
			"error": err.Error(),
			"path":  r.URL.Path,
		})
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.SanitizeDBError(err),
		))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"dead_letters": rows})
}

func (s *Server) acknowledgeIncident(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     string `json:"id"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	incident, found, err := s.db.IncidentByID(req.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "database error"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "incident not found"})
		return
	}
	incident.State = "acknowledged"
	identity, _ := security.GetIdentity(r.Context())
	if incident.Metadata == nil {
		incident.Metadata = make(map[string]any)
	}
	incident.Metadata["acknowledged_by"] = identity.ActorID
	incident.Metadata["acknowledge_reason"] = req.Reason

	if err := s.db.UpsertIncident(incident); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to update incident"})
		return
	}
	// Audit log the action
	auditEntry := auth.LogIncidentAction(auth.GetAuthContextFromRequest(r), req.ID, "acknowledge", req.Reason, auth.AuditResultSuccess, nil)
	_ = s.db.InsertAuditLog("incident", "info", "incident acknowledged", auditEntry.ToMap())

	s.log.Info("incident_acknowledged", "incident acknowledged by operator", map[string]any{"incident_id": req.ID, "actor": identity.ActorID})
	writeJSON(w, http.StatusOK, map[string]any{"status": "acknowledged"})
}

func (s *Server) resolveIncident(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     string `json:"id"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	incident, found, err := s.db.IncidentByID(req.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "database error"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "incident not found"})
		return
	}
	incident.State = "resolved"
	incident.ResolvedAt = time.Now().UTC().Format(time.RFC3339)
	identity, _ := security.GetIdentity(r.Context())
	if incident.Metadata == nil {
		incident.Metadata = make(map[string]any)
	}
	incident.Metadata["resolved_by"] = identity.ActorID
	incident.Metadata["resolve_reason"] = req.Reason

	if err := s.db.UpsertIncident(incident); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to update incident"})
		return
	}
	// Audit log the action
	auditEntry := auth.LogIncidentAction(auth.GetAuthContextFromRequest(r), req.ID, "resolve", req.Reason, auth.AuditResultSuccess, nil)
	_ = s.db.InsertAuditLog("incident", "info", "incident resolved", auditEntry.ToMap())

	s.log.Info("incident_resolved", "incident resolved by operator", map[string]any{"incident_id": req.ID, "actor": identity.ActorID})
	writeJSON(w, http.StatusOK, map[string]any{"status": "resolved"})
}

func (s *Server) escalateIncident(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     string `json:"id"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	incident, found, err := s.db.IncidentByID(req.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "database error"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "incident not found"})
		return
	}
	incident.State = "escalated"
	identity, _ := security.GetIdentity(r.Context())
	if incident.Metadata == nil {
		incident.Metadata = make(map[string]any)
	}
	incident.Metadata["escalated_by"] = identity.ActorID
	incident.Metadata["escalate_reason"] = req.Reason

	if err := s.db.UpsertIncident(incident); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to update incident"})
		return
	}
	// Audit log the action
	auditEntry := auth.LogIncidentAction(auth.GetAuthContextFromRequest(r), req.ID, "escalate", req.Reason, auth.AuditResultSuccess, nil)
	_ = s.db.InsertAuditLog("incident", "warning", "incident escalated", auditEntry.ToMap())

	s.log.Warn("incident_escalated", "incident escalated by operator", map[string]any{"incident_id": req.ID, "actor": identity.ActorID})
	writeJSON(w, http.StatusOK, models.StatusResponse{
		Status:  "escalated",
		Message: fmt.Sprintf("Incident %s has been escalated for administrative review.", req.ID),
	})
}

func (s *Server) ui(w http.ResponseWriter, _ *http.Request) {
	snap := s.state.Snapshot()
	statusSnap, _ := s.statusSnapshot()
	sort.Slice(snap.Nodes, func(i, j int) bool { return snap.Nodes[i].Num < snap.Nodes[j].Num })
	findings := privacy.Audit(s.cfg)
	messages, _ := s.db.QueryRows("SELECT transport_name,packet_id,from_node,to_node,portnum,payload_text,rx_time FROM messages ORDER BY id DESC LIMIT 20;")
	if s.cfg.Privacy.RedactExports {
		messages = privacy.RedactMessages(messages)
	}
	persistedMessages, _ := s.db.Scalar("SELECT COUNT(*) FROM messages;")
	persistedNodes, _ := s.db.Scalar("SELECT COUNT(*) FROM nodes;")
	lastPersistedIngest, _ := s.db.Scalar("SELECT COALESCE(MAX(rx_time), '') FROM messages;")
	logs, _ := s.db.QueryRows("SELECT category,level,message,created_at FROM audit_logs ORDER BY id DESC LIMIT 20;")
	fmt.Fprintf(w, `<!doctype html><html><head><title>MEL — MeshEdgeLayer</title><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<style>
:root{--bg:#0f172a;--card:#1e293b;--text:#f8fafc;--muted:#94a3b8;--accent:#38bdf8;--border:#334155;--success:#22c55e;--warning:#f59e0b;--critical:#ef4444}
body{font-family:Inter,system-ui,-apple-system,sans-serif;background-color:var(--bg);color:var(--text);margin:0;line-height:1.6;display:flex;flex-direction:column;min-height:100vh}
header{padding:2rem 1rem;background:linear-gradient(135deg,#0f172a 0%%,#1e293b 100%%);border-bottom:1px solid var(--border);text-align:center}
h1{margin:0;font-size:2.5rem;font-weight:800;letter-spacing:-0.025em;background:linear-gradient(to right,#38bdf8,#818cf8);-webkit-background-clip:text;-webkit-text-fill-color:transparent}
nav{position:sticky;top:0;background:rgba(15,23,42,0.8);backdrop-filter:blur(8px);padding:0.75rem;border-bottom:1px solid var(--border);z-index:100;display:flex;justify-content:center;gap:1rem;flex-wrap:wrap}
nav a{color:var(--muted);text-decoration:none;font-size:0.875rem;font-weight:500;transition:color 0.2s}
nav a:hover{color:var(--accent)}
main{max-width:1200px;margin:2rem auto;padding:0 1rem;width:100%%;box-sizing:border-box}
section{background:var(--card);border:1px solid var(--border);border-radius:12px;padding:1.5rem;margin-bottom:2rem;box-shadow:0 4px 6px -1px rgba(0,0,0,0.1),0 2px 4px -1px rgba(0,0,0,0.06)}
h2{margin-top:0;font-size:1.25rem;font-weight:600;display:flex;align-items:center;gap:0.5rem;border-bottom:1px solid var(--border);padding-bottom:0.75rem;margin-bottom:1.25rem}
table{border-collapse:collapse;width:100%%;font-size:0.875rem}
th{text-align:left;color:var(--muted);font-weight:500;padding:0.75rem;border-bottom:1px solid var(--border)}
td{padding:0.75rem;border-bottom:1px solid var(--border);vertical-align:top}
code,pre{font-family:JetBrains Mono,Menlo,Monaco,Consolas,monospace;background:rgba(15,23,42,0.5);padding:0.2rem 0.4rem;border-radius:4px;font-size:0.8125rem}
pre{padding:1rem;overflow:auto;max-height:400px;border:1px solid var(--border)}
.pill{background:var(--border);color:var(--text);border-radius:9999px;padding:0.25rem 0.75rem;font-size:0.75rem;font-weight:600}
.status-pill{width:8px;height:8px;border-radius:50%%;display:inline-block}
.status-good{background:var(--success);box-shadow:0 0 8px var(--success)}
.status-warn{background:var(--warning)}
.status-crit{background:var(--critical)}
.muted{color:var(--muted)}
.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(250px,1fr));gap:1.5rem;margin-bottom:2rem}
.stat-card{background:var(--card);border:1px solid var(--border);padding:1.25rem;border-radius:12px;text-align:center}
.stat-value{display:block;font-size:1.5rem;font-weight:700;margin-top:0.25rem}
.stat-label{font-size:0.75rem;color:var(--muted);text-transform:uppercase;letter-spacing:0.05em}
.sev-critical{color:var(--critical);font-weight:600}
.sev-high{color:var(--warning)}
.sev-medium{color:#fbbf24}
@media (max-width:768px){h1{font-size:1.75rem} .grid{grid-template-columns:1fr}}
</style>
</head><body><header><h1>MEL</h1><p class="muted">MeshEdgeLayer — Truthful Local Mesh Observability</p></header>
<nav><a href="#status">Status</a><a href="#transports">Transports</a><a href="#nodes">Nodes</a><a href="#messages">Messages</a><a href="#privacy">Privacy</a><a href="#events">Events</a><a href="#onboarding">Guide</a></nav>
<main>`)
	fmt.Fprintf(w, `<div class="grid">
<div class="stat-card"><span class="stat-label">Runtime Messages</span><span class="stat-value">%d</span></div>
<div class="stat-card"><span class="stat-label">Persisted Messages</span><span class="stat-value">%s</span></div>
<div class="stat-card"><span class="stat-label">Persisted Nodes</span><span class="stat-value">%s</span></div>
<div class="stat-card"><span class="stat-label">Last Ingest</span><span class="stat-value">%s</span></div>
</div>`, snap.Messages, blankIfEmpty(persistedMessages, "0"), blankIfEmpty(persistedNodes, "0"), blankIfEmpty(lastPersistedIngest, "Never"))

	fmt.Fprint(w, `<section id="status"><h2>Status</h2><p>Active transport modes: `)
	for _, mode := range statusSnap.ConfiguredTransportModes {
		fmt.Fprintf(w, `<span class="pill">%s</span> `, mode)
	}
	fmt.Fprint(w, `</p>`)
	if len(snap.Nodes) == 0 {
		fmt.Fprint(w, `<p class="muted">No live nodes observed yet. Historical counts above reflect stored database state.</p>`)
	}
	fmt.Fprint(w, `</section>`)

	panel := statuspkg.BuildPanel(statusSnap)
	fmt.Fprintf(w, `<section id="panel"><h2>Instrument Panel</h2><p><strong>Operator State:</strong> %s</p><p>%s</p><pre>%s</pre></section>`, panel.OperatorState, panel.Summary, asJSON(panel.OperatorMenu))

	fmt.Fprint(w, `<section id="transports"><h2>Transport Health</h2><div style="overflow-x:auto"><table><tr><th>Transport</th><th>State</th><th>Health</th><th>Alerts</th><th>Stats</th><th>Last Seen</th></tr>`)
	for _, h := range statusSnap.Transports {
		statusClass := "status-good"
		if h.Health.Score < 100 {
			statusClass = "status-warn"
		}
		if h.Health.Score < 50 {
			statusClass = "status-crit"
		}

		fmt.Fprintf(w, `<tr>
<td><strong>%s</strong><br><span class="muted">%s</span></td>
<td><code>%s</code></td>
<td><span class="status-pill %s"></span> <strong>%d</strong><br><span class="muted">%s</span></td>
<td>%d active<br><span class="muted">%s</span></td>
<td>%d msg / %d DL</td>
<td>%s</td>
</tr>`, h.Name, h.Type, h.EffectiveState, statusClass, h.Health.Score, h.Health.State, len(h.ActiveAlerts), h.Health.PrimaryReason, h.TotalMessages, h.DeadLetters, blankIfEmpty(h.LastIngestAt, "—"))
	}
	fmt.Fprint(w, `</table></div></section>`)

	fmt.Fprint(w, `<section id="nodes"><h2>Node Inventory</h2>`)
	if len(snap.Nodes) == 0 {
		fmt.Fprint(w, `<p class="muted">No nodes available.</p>`)
	} else {
		fmt.Fprint(w, `<div style="overflow-x:auto"><table><tr><th>Node</th><th>ID</th><th>Name</th><th>Last Seen</th><th>Gateway</th></tr>`)
		for _, n := range snap.Nodes {
			fmt.Fprintf(w, `<tr><td>%d</td><td><code>%s</code></td><td>%s <span class="muted">%s</span></td><td>%s</td><td>%s</td></tr>`, n.Num, n.ID, n.LongName, n.ShortName, n.LastSeen, n.GatewayID)
		}
		fmt.Fprint(w, `</table></div>`)
	}
	fmt.Fprint(w, `</section>`)

	fmt.Fprint(w, `<section id="messages"><h2>Recent Messages</h2>`)
	if len(messages) == 0 {
		fmt.Fprint(w, `<p class="muted">No messages found.</p>`)
	} else {
		fmt.Fprint(w, `<pre>`+asJSON(messages)+`</pre>`)
	}
	fmt.Fprint(w, `</section>`)

	fmt.Fprint(w, `<section id="privacy"><h2>Privacy / Security Findings</h2>`)
	if len(findings) == 0 {
		fmt.Fprint(w, `<p>No active security findings.</p>`)
	} else {
		fmt.Fprint(w, `<ul>`)
		for _, f := range findings {
			fmt.Fprintf(w, `<li class="sev-%s"><strong>[%s]</strong> %s<br><span class="muted">%s</span></li>`, f.Severity, strings.ToUpper(f.Severity), f.Message, f.Remediation)
		}
		fmt.Fprint(w, `</ul>`)
	}
	fmt.Fprint(w, `</section>`)

	fmt.Fprint(w, `<section id="events"><h2>System Events</h2><pre>`+asJSON(logs)+`</pre></section>`)

	fmt.Fprintf(w, `<section id="onboarding"><h2>Onboarding Guide</h2><ol>
<li>Run <code>mel init</code> to bootstrap a fresh configuration (bound to localhost by default).</li>
<li>Run <code>mel doctor</code> to verify database permissions, schema, and device connectivity.</li>
<li>Connect a Meshtastic device via Serial or TCP.</li>
<li>Start the server with <code>mel serve</code>.</li>
</ol></section></main>
<footer style="text-align:center;padding:2rem;color:var(--muted);font-size:0.75rem;border-top:1px solid var(--border)">
MEL Version %s — %s
</footer></body></html>`, version.Version, version.BuildTime)
}

func asJSON(v any) string { b, _ := json.MarshalIndent(v, "", "  "); return string(b) }

func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.cfg.Auth.Enabled {
			// No auth enabled; assume local admin.
			ctx := security.WithIdentity(r.Context(), security.BuildAdminIdentity("local_unauthenticated"))
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
		if apiKey != "" {
			for _, ent := range s.cfg.OperatorAPIKeyEntries {
				if ent.Key == apiKey {
					sum := sha256.Sum256([]byte(ent.Key))
					short := hex.EncodeToString(sum[:])[:12]
					id := security.IdentityFromAPIKey(short, ent.Capabilities)
					next.ServeHTTP(w, r.WithContext(security.WithIdentity(r.Context(), id)))
					return
				}
			}
			s.log.Security("auth_failure", "invalid X-API-Key", "high", map[string]any{"path": r.URL.Path, "remote": remoteClient(r)})
			w.Header().Set("WWW-Authenticate", `Basic realm="mel"`)
			writeJSON(w, http.StatusUnauthorized, logging.APIErrorResponse(
				logging.NewSafeError("authentication is required for this MEL endpoint", nil, "auth", false),
			))
			return
		}

		user, pass, ok := r.BasicAuth()
		if ok && user == s.cfg.Auth.UIUser && pass == s.cfg.Auth.UIPassword {
			ctx := security.WithIdentity(r.Context(), security.BuildAdminIdentity(user))
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		if len(s.cfg.OperatorAPIKeys) > 0 {
			s.log.Security("auth_required", "X-API-Key or Basic auth required", "warning", map[string]any{"path": r.URL.Path, "remote": remoteClient(r)})
			w.Header().Set("WWW-Authenticate", `Basic realm="mel"`)
			writeJSON(w, http.StatusUnauthorized, logging.APIErrorResponse(
				logging.NewSafeError("authentication is required for this MEL endpoint", nil, "auth", false),
			))
			return
		}

		severity := "warning"
		if ok {
			severity = "high"
			s.log.Security("auth_failure", "authentication failed with invalid credentials", severity, map[string]any{
				"path":   r.URL.Path,
				"remote": remoteClient(r),
				"user":   user,
			})
		} else {
			s.log.Security("auth_required", "authentication required but not provided", severity, map[string]any{
				"path":   r.URL.Path,
				"remote": remoteClient(r),
			})
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="mel"`)
		writeJSON(w, http.StatusUnauthorized, logging.APIErrorResponse(
			logging.NewSafeError("authentication is required for this MEL endpoint", nil, "auth", false),
		))
	})
}

func blankIfEmpty(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func totalDeadLetters(transports []statuspkg.TransportReport) uint64 {
	var total uint64
	for _, tr := range transports {
		total += tr.DeadLetters
	}
	return total
}

func escape(v string) string { return db.EscString(v) }

func isSafeIdentifier(v string) bool { return db.IsSafeIdentifier(v) }

func containsPathTraversal(v string) bool {
	return strings.Contains(v, "..") || strings.Contains(v, "%2e%2e") || strings.Contains(v, "%2E%2E")
}

func remoteClient(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func toInt(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case string:
		var parsed int64
		fmt.Sscan(x, &parsed)
		return parsed
	}
	var parsed int64
	fmt.Sscan(fmt.Sprint(v), &parsed)
	return parsed
}

func scalarInt(d *db.DB, sql string) int64 {
	if d == nil {
		return 0
	}
	value, err := d.Scalar(sql)
	if err != nil {
		return 0
	}
	return toInt(value)
}

func scalarFloat(d *db.DB, sql string) float64 {
	if d == nil {
		return 0
	}
	value, err := d.Scalar(sql)
	if err != nil {
		return 0
	}
	var parsed float64
	fmt.Sscan(fmt.Sprint(value), &parsed)
	return parsed
}

func asInt(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case string:
		var parsed int64
		fmt.Sscan(x, &parsed)
		return parsed
	}
	var parsed int64
	fmt.Sscan(fmt.Sprint(v), &parsed)
	return parsed
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func asFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case string:
		var parsed float64
		fmt.Sscan(x, &parsed)
		return parsed
	}
	var parsed float64
	fmt.Sscan(fmt.Sprint(v), &parsed)
	return parsed
}

func (s *Server) operatorBriefingHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.operatorBriefing())
}
