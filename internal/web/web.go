package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/logging"
	"github.com/mel-project/mel/internal/meshstate"
	"github.com/mel-project/mel/internal/policy"
	"github.com/mel-project/mel/internal/privacy"
	statuspkg "github.com/mel-project/mel/internal/status"
	"github.com/mel-project/mel/internal/transport"
)

type Server struct {
	cfg             config.Config
	log             *logging.Logger
	db              *db.DB
	state           *meshstate.State
	bus             *events.Bus
	http            *http.Server
	transportHealth func() []transport.Health
	recommendations func() []policy.Recommendation
	statusSnapshot  func() (statuspkg.Snapshot, error)
	controlStatus   func() (map[string]any, error)
	controlHistory  func(string, string, string, int, int) (map[string]any, error)
}

func New(cfg config.Config, log *logging.Logger, d *db.DB, st *meshstate.State, bus *events.Bus, th func() []transport.Health, rec func() []policy.Recommendation, statusSnapshot func() (statuspkg.Snapshot, error), controlStatus func() (map[string]any, error), controlHistory func(string, string, string, int, int) (map[string]any, error)) *Server {
	snapFn := statusSnapshot
	if snapFn == nil {
		snapFn = func() (statuspkg.Snapshot, error) { return statuspkg.Collect(cfg, d, th()) }
	}
	controlStatusFn := controlStatus
	if controlStatusFn == nil {
		controlStatusFn = func() (map[string]any, error) {
			return map[string]any{"mode": cfg.Control.Mode, "status": "control unavailable without service control hooks"}, nil
		}
	}
	controlHistoryFn := controlHistory
	if controlHistoryFn == nil {
		controlHistoryFn = func(start, end, transport string, limit, offset int) (map[string]any, error) {
			return map[string]any{"actions": []any{}, "decisions": []any{}, "start": start, "end": end, "transport": transport, "pagination": map[string]any{"limit": limit, "offset": offset}}, nil
		}
	}
	s := &Server{cfg: cfg, log: log, db: d, state: st, bus: bus, transportHealth: th, recommendations: rec, statusSnapshot: snapFn, controlStatus: controlStatusFn, controlHistory: controlHistoryFn}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/readyz", s.readyz)
	mux.HandleFunc("/metrics", s.metrics)
	mux.HandleFunc("/api/status", s.status)
	mux.HandleFunc("/api/nodes", s.nodes)
	mux.HandleFunc("/api/transports", s.transports)
	mux.HandleFunc("/api/privacy/audit", s.audit)
	mux.HandleFunc("/api/recommendations", s.recs)
	mux.HandleFunc("/api/logs", s.logs)
	mux.HandleFunc("/api/dead-letters", s.deadLetters)
	mux.HandleFunc("/api/v1/status", s.status)
	mux.HandleFunc("/api/v1/nodes", s.nodes)
	mux.HandleFunc("/api/v1/node/", s.nodeDetail)
	mux.HandleFunc("/api/v1/transports", s.transports)
	mux.HandleFunc("/api/v1/transports/health", s.transportHealthSummary)
	mux.HandleFunc("/api/v1/transports/alerts", s.transportAlerts)
	mux.HandleFunc("/api/v1/transports/anomalies", s.transportAnomalies)
	mux.HandleFunc("/api/v1/transports/health/history", s.transportHealthHistory)
	mux.HandleFunc("/api/v1/transports/alerts/history", s.transportAlertsHistory)
	mux.HandleFunc("/api/v1/transports/anomalies/history", s.transportAnomaliesHistory)
	mux.HandleFunc("/api/v1/transports/inspect/", s.transportInspect)
	mux.HandleFunc("/api/v1/mesh", s.mesh)
	mux.HandleFunc("/api/v1/mesh/inspect", s.meshInspect)
	mux.HandleFunc("/api/v1/messages", s.messages)
	mux.HandleFunc("/api/v1/metrics", s.metrics)
	mux.HandleFunc("/api/v1/panel", s.panel)
	mux.HandleFunc("/api/v1/privacy/audit", s.audit)
	mux.HandleFunc("/api/v1/policy/explain", s.recs)
	mux.HandleFunc("/api/v1/events", s.logs)
	mux.HandleFunc("/api/v1/audit-logs", s.logs)
	mux.HandleFunc("/api/v1/dead-letters", s.deadLetters)
	mux.HandleFunc("/api/v1/incidents", s.incidents)
	mux.HandleFunc("/api/v1/control/status", s.controlStatusHandler)
	mux.HandleFunc("/api/v1/control/actions", s.controlActionsHandler)
	mux.HandleFunc("/api/v1/control/history", s.controlHistoryHandler)
	if cfg.Features.WebUI {
		mux.HandleFunc("/", s.ui)
	}
	s.http = &http.Server{Addr: cfg.Bind.API, Handler: s.withAuth(mux), ReadHeaderTimeout: 5 * time.Second}
	return s
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
func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
func (s *Server) readyz(w http.ResponseWriter, _ *http.Request) {
	snap, err := s.statusSnapshot()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ready": false, "error": err.Error()})
		return
	}
	panel := statuspkg.BuildPanel(snap)
	ingestReady := false
	for _, tr := range snap.Transports {
		if tr.EffectiveState == transport.StateIngesting {
			ingestReady = true
			break
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ready":          true,
		"process_ready":  true,
		"ingest_ready":   ingestReady,
		"operator_state": panel.OperatorState,
		"summary":        panel.Summary,
		"transports":     snap.Transports,
	})
}
func (s *Server) status(w http.ResponseWriter, _ *http.Request) {
	snap, err := s.statusSnapshot()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]any{"code": "status_failed", "message": err.Error()}})
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
func (s *Server) nodes(w http.ResponseWriter, _ *http.Request) {
	rows, err := s.db.QueryRows("SELECT n.node_num,n.node_id,n.long_name,n.short_name,n.last_seen,n.last_gateway_id,n.lat_redacted,n.lon_redacted,n.altitude,n.last_snr,n.last_rssi,(SELECT COUNT(*) FROM messages m WHERE m.from_node=n.node_num) AS message_count FROM nodes n ORDER BY n.updated_at DESC;")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]any{"code": "db_query_failed", "message": err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"nodes": rows})
}
func (s *Server) nodeDetail(w http.ResponseWriter, r *http.Request) {
	nodeID := strings.TrimPrefix(r.URL.Path, "/api/v1/node/")
	if nodeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"code": "missing_node", "message": "node identifier is required"}})
		return
	}
	query := fmt.Sprintf("SELECT n.node_num,n.node_id,n.long_name,n.short_name,n.last_seen,n.last_gateway_id,n.lat_redacted,n.lon_redacted,n.altitude,n.last_snr,n.last_rssi,(SELECT COUNT(*) FROM messages m WHERE m.from_node=n.node_num) AS message_count FROM nodes n WHERE CAST(n.node_num AS TEXT)='%s' OR n.node_id='%s' LIMIT 1;", escape(nodeID), escape(nodeID))
	rows, err := s.db.QueryRows(query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]any{"code": "db_query_failed", "message": err.Error()}})
		return
	}
	if len(rows) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": map[string]any{"code": "node_not_found", "message": "node not present in local observations"}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"node": rows[0]})
}
func (s *Server) transports(w http.ResponseWriter, _ *http.Request) {
	snap, err := s.statusSnapshot()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"transports": snap.Transports, "configured_modes": snap.ConfiguredTransportModes, "recent_transport_incidents": snap.RecentTransportIncidents, "active_transport_alerts": snap.ActiveTransportAlerts})
}

func (s *Server) transportHealthSummary(w http.ResponseWriter, _ *http.Request) {
	snap, err := s.statusSnapshot()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	health := make([]any, 0, len(snap.Transports))
	for _, tr := range snap.Transports {
		health = append(health, map[string]any{
			"transport_name":    tr.Name,
			"transport_type":    tr.Type,
			"runtime_state":     tr.RuntimeState,
			"effective_state":   tr.EffectiveState,
			"health":            tr.Health,
			"active_alerts":     tr.ActiveAlerts,
			"recent_anomalies":  tr.RecentAnomalies,
			"failure_clusters":  tr.FailureClusters,
			"last_failure_at":   tr.LastFailureAt,
			"episode_id":        tr.EpisodeID,
			"failure_count":     tr.FailureCount,
			"observation_drops": tr.ObservationDrops,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"transport_health": health})
}

func (s *Server) transportAlerts(w http.ResponseWriter, _ *http.Request) {
	snap, err := s.statusSnapshot()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"transport_alerts": snap.ActiveTransportAlerts})
}

func (s *Server) transportAnomalies(w http.ResponseWriter, _ *http.Request) {
	snap, err := s.statusSnapshot()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
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
	transportName, start, end, limit, offset := historyParams(s.cfg, r)
	rows, err := s.db.TransportHealthSnapshots(transportName, start, end, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": rows, "pagination": map[string]any{"limit": limit, "offset": offset}, "transport": transportName, "start": start, "end": end})
}

func (s *Server) transportAlertsHistory(w http.ResponseWriter, r *http.Request) {
	transportName, start, end, limit, offset := historyParams(s.cfg, r)
	rows, err := s.db.TransportAlertsHistory(transportName, start, end, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": rows, "pagination": map[string]any{"limit": limit, "offset": offset}, "transport": transportName, "start": start, "end": end})
}

func (s *Server) transportAnomaliesHistory(w http.ResponseWriter, r *http.Request) {
	transportName, start, end, limit, offset := historyParams(s.cfg, r)
	rows, err := s.db.TransportAnomalyHistory(transportName, start, end, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": rows, "pagination": map[string]any{"limit": limit, "offset": offset}, "transport": transportName, "start": start, "end": end})
}

func (s *Server) transportInspect(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/transports/inspect/")
	if strings.TrimSpace(name) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"code": "missing_transport", "message": "transport name is required"}})
		return
	}
	drilldown, err := statuspkg.InspectTransport(s.cfg, s.db, s.transportHealth(), name, time.Now().UTC())
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": map[string]any{"code": "transport_not_found", "message": err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, drilldown)
}

func (s *Server) mesh(w http.ResponseWriter, _ *http.Request) {
	snap, err := s.statusSnapshot()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, snap.Mesh)
}

func (s *Server) meshInspect(w http.ResponseWriter, _ *http.Request) {
	drilldown, err := statuspkg.InspectMesh(s.cfg, s.db, s.transportHealth(), time.Now().UTC())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]any{"code": "mesh_inspect_failed", "message": err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, drilldown)
}

func (s *Server) controlStatusHandler(w http.ResponseWriter, _ *http.Request) {
	payload, err := s.controlStatus()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]any{"code": "control_status_failed", "message": err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) controlActionsHandler(w http.ResponseWriter, r *http.Request) {
	transportName, start, end, limit, offset := historyParams(s.cfg, r)
	payload, err := s.controlHistory(start, end, transportName, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]any{"code": "control_actions_failed", "message": err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"actions":    payload["actions"],
		"transport":  transportName,
		"start":      start,
		"end":        end,
		"pagination": payload["pagination"],
	})
}

func (s *Server) controlHistoryHandler(w http.ResponseWriter, r *http.Request) {
	transportName, start, end, limit, offset := historyParams(s.cfg, r)
	payload, err := s.controlHistory(start, end, transportName, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]any{"code": "control_history_failed", "message": err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func historyParams(cfg config.Config, r *http.Request) (string, string, string, int, int) {
	transportName := strings.TrimSpace(r.URL.Query().Get("transport"))
	start := strings.TrimSpace(r.URL.Query().Get("start"))
	end := strings.TrimSpace(r.URL.Query().Get("end"))
	limit := cfg.Intelligence.Queries.DefaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= cfg.Intelligence.Queries.MaxLimit {
			limit = parsed
		}
	}
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	return transportName, start, end, limit, offset
}

func (s *Server) messages(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}
	clauses := []string{"1=1"}
	if node := r.URL.Query().Get("node"); node != "" {
		clauses = append(clauses, fmt.Sprintf("(CAST(from_node AS TEXT)='%s' OR CAST(to_node AS TEXT)='%s')", escape(node), escape(node)))
	}
	if messageType := r.URL.Query().Get("type"); messageType != "" {
		clauses = append(clauses, fmt.Sprintf("payload_json LIKE '%%%s%%'", escape(fmt.Sprintf(`\"message_type\":\"%s\"`, messageType))))
	}
	rows, err := s.db.QueryRows(fmt.Sprintf("SELECT transport_name,packet_id,from_node,to_node,portnum,payload_text,payload_json,rx_time,created_at FROM messages WHERE %s ORDER BY id DESC LIMIT %d;", strings.Join(clauses, " AND "), limit))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]any{"code": "db_query_failed", "message": err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": rows, "filters": r.URL.Query()})
}
func (s *Server) panel(w http.ResponseWriter, _ *http.Request) {
	snap, err := s.statusSnapshot()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]any{"code": "panel_failed", "message": err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, statuspkg.BuildPanel(snap))
}

func (s *Server) metrics(w http.ResponseWriter, _ *http.Request) {
	snap, err := s.statusSnapshot()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	cutoff := time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339)
	rows, err := s.db.QueryRows(fmt.Sprintf("SELECT transport_name, COUNT(*) AS recent_messages FROM messages WHERE rx_time >= '%s' GROUP BY transport_name;", cutoff))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	rateByTransport := map[string]float64{}
	for _, row := range rows {
		rateByTransport[fmt.Sprint(row["transport_name"])] = float64(toInt(row["recent_messages"])) / 300.0
	}
	metrics := map[string]any{
		"generated_at":        time.Now().UTC().Format(time.RFC3339),
		"window_seconds":      300,
		"total_messages":      snap.Messages,
		"last_ingest_time":    snap.LastSuccessfulIngest,
		"transport_metrics":   snap.Transports,
		"ingest_rate_per_sec": rateByTransport,
		"dead_letters_total":  totalDeadLetters(snap.Transports),
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
		query += fmt.Sprintf(" WHERE details_json LIKE '%%%s%%'", escape(fmt.Sprintf(`\"transport\":\"%s\"`, transportName)))
	}
	query += " ORDER BY id DESC LIMIT 100;"
	rows, err := s.db.QueryRows(query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]any{"code": "db_query_failed", "message": err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": rows})
}
func (s *Server) incidents(w http.ResponseWriter, _ *http.Request) {
	incidents, err := s.db.RecentTransportIncidents(100)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]any{"code": "db_query_failed", "message": err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recent_transport_incidents": incidents})
}

func (s *Server) deadLetters(w http.ResponseWriter, r *http.Request) {
	query := "SELECT transport_name,transport_type,topic,reason,payload_hex,details_json,created_at FROM dead_letters"
	if transportName := strings.TrimSpace(r.URL.Query().Get("transport")); transportName != "" {
		query += fmt.Sprintf(" WHERE transport_name='%s'", escape(transportName))
	}
	query += " ORDER BY id DESC LIMIT 100;"
	rows, err := s.db.QueryRows(query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]any{"code": "db_query_failed", "message": err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"dead_letters": rows})
}
func (s *Server) ui(w http.ResponseWriter, _ *http.Request) {
	snap := s.state.Snapshot()
	statusSnap, _ := s.statusSnapshot()
	sort.Slice(snap.Nodes, func(i, j int) bool { return snap.Nodes[i].Num < snap.Nodes[j].Num })
	findings := privacy.Audit(s.cfg)
	messages, _ := s.db.QueryRows("SELECT transport_name,packet_id,from_node,to_node,portnum,payload_text,rx_time FROM messages ORDER BY id DESC LIMIT 20;")
	persistedMessages, _ := s.db.Scalar("SELECT COUNT(*) FROM messages;")
	persistedNodes, _ := s.db.Scalar("SELECT COUNT(*) FROM nodes;")
	lastPersistedIngest, _ := s.db.Scalar("SELECT COALESCE(MAX(rx_time), '') FROM messages;")
	logs, _ := s.db.QueryRows("SELECT category,level,message,created_at FROM audit_logs ORDER BY id DESC LIMIT 20;")
	deadLetters, _ := s.db.QueryRows("SELECT transport_name,transport_type,topic,reason,created_at FROM dead_letters ORDER BY id DESC LIMIT 20;")
	fmt.Fprintf(w, `<!doctype html><html><head><title>MEL</title><meta name="viewport" content="width=device-width, initial-scale=1"><style>
body{font-family:system-ui,sans-serif;max-width:1200px;margin:2rem auto;padding:0 1rem;line-height:1.45;background:#fafafa;color:#111}
nav a{margin-right:1rem}section{background:#fff;border:1px solid #ddd;border-radius:8px;padding:1rem;margin:1rem 0}
table{border-collapse:collapse;width:100%%}td,th{border:1px solid #ddd;padding:.45rem;text-align:left;vertical-align:top}.muted{color:#666}.sev-critical{color:#8b0000}.sev-high{color:#b04a00}.sev-medium{color:#805b00}
code,pre{background:#f5f5f5;padding:.2rem .35rem;border-radius:4px;overflow:auto}ul{padding-left:1.25rem}.pill{display:inline-block;padding:.15rem .5rem;border:1px solid #ccc;border-radius:999px;margin-right:.35rem;margin-bottom:.35rem}
</style></head><body><h1>MEL — MeshEdgeLayer</h1><p>Truthful local-first observability for stock Meshtastic nodes. No demo data is injected when transports are idle.</p><nav><a href="#onboarding">Onboarding</a><a href="#panel">Panel</a><a href="#status">Status</a><a href="#transports">Transport health</a><a href="#deadletters">Dead letters</a><a href="#nodes">Nodes</a><a href="#messages">Messages</a><a href="#privacy">Privacy findings</a><a href="#recommendations">Recommendations</a><a href="#events">Events</a></nav>`)
	fmt.Fprint(w, `<section id="onboarding"><h2>Onboarding</h2><ol><li>Run <code>mel init --config /etc/mel/mel.json</code> if you do not have a config yet.</li><li>Run <code>mel doctor --config /etc/mel/mel.json</code> to validate direct-node reachability, local permissions, and privacy posture.</li><li>Prefer one real direct transport (<code>serial</code> or <code>tcp</code>) for Pi/Linux deployment, then start <code>mel serve --config /etc/mel/mel.json</code>.</li><li>Use <code>mel panel --config /etc/mel/mel.json</code> or <code>/api/v1/panel</code> for a compact instrument panel.</li><li>Return here to confirm whether MEL is disconnected, connected but idle, or receiving real mesh packets.</li></ol></section>`)
	fmt.Fprint(w, `<section id="status"><h2>Status</h2><p>Configured transport modes: `)
	for _, mode := range statusSnap.ConfiguredTransportModes {
		fmt.Fprintf(w, `<span class="pill">%s</span>`, mode)
	}
	fmt.Fprintf(w, `</p><p>Runtime process message count: <strong>%d</strong>.</p><p>Persisted message count: <strong>%s</strong>. Persisted node count: <strong>%s</strong>. Last persisted ingest: <strong>%s</strong>.</p>`, snap.Messages, blankIfEmpty(persistedMessages, "0"), blankIfEmpty(persistedNodes, "0"), blankIfEmpty(lastPersistedIngest, "none"))
	if len(snap.Nodes) == 0 {
		fmt.Fprint(w, `<p class="muted">The current MEL process has not observed any nodes yet. Persisted counts above may still show historical data from prior runs. No sample mesh data is shown.</p>`)
	} else {
		fmt.Fprintf(w, `<p>Observed nodes: <strong>%d</strong>.</p>`, len(snap.Nodes))
	}
	panel := statuspkg.BuildPanel(statusSnap)
	fmt.Fprint(w, `</section><section id="panel"><h2>Instrument panel</h2>`)
	fmt.Fprintf(w, `<p><strong>Operator state:</strong> %s</p><p>%s</p><p><strong>Short commands:</strong> %s</p><pre>%s</pre></section><section id="transports"><h2>Transport health</h2><table><tr><th>Name</th><th>Type</th><th>Effective state</th><th>Health</th><th>Why unhealthy</th><th>Alerts</th><th>Scope</th><th>Detail</th><th>Messages</th><th>Heartbeat</th><th>Timeouts</th><th>Retry status</th><th>Dead letters</th><th>Observation drops</th><th>Last attempt</th><th>Last ingest</th><th>Last error</th></tr>`, panel.OperatorState, panel.Summary, strings.Join(panel.ShortCommands, " | "), asJSON(panel.DeviceMenu))
	for _, h := range statusSnap.Transports {
		fmt.Fprintf(w, `<tr><td>%s<br><span class="muted">%s</span></td><td>%s</td><td><code>%s</code><br><span class="muted">runtime=%s</span></td><td><strong>%d</strong> / %s<br><span class="muted">%s</span></td><td><pre>%s</pre></td><td><pre>%s</pre></td><td>%s</td><td>%s<br><span class="muted">%s</span></td><td>%d runtime / %d persisted</td><td>%s</td><td>%d</td><td>%s</td><td>%d</td><td>%d</td><td>%s</td><td>%s</td><td>%s</td></tr>`, h.Name, blankIfEmpty(h.Source, "—"), h.Type, blankIfEmpty(h.EffectiveState, "unknown"), blankIfEmpty(h.RuntimeState, "unknown"), h.Health.Score, blankIfEmpty(h.Health.State, "unknown"), blankIfEmpty(h.Health.PrimaryReason, "no dominant reason"), asJSON(h.Health.Explanation), asJSON(h.ActiveAlerts), h.StatusScope, h.Detail, h.Guidance, h.TotalMessages, h.PersistedMessages, blankIfEmpty(h.LastHeartbeatAt, "—"), h.ConsecutiveTimeouts, h.RetryStatus, h.DeadLetters, h.ObservationDrops, blankIfEmpty(h.LastAttemptAt, "—"), blankIfEmpty(h.LastIngestAt, "—"), blankIfEmpty(h.LastError, "—"))
	}
	fmt.Fprint(w, `</table><p class="muted">If multiple transports are enabled, operators must verify radio ownership and contention behavior themselves; MEL does not claim shared-radio arbitration that stock nodes do not provide.</p></section>`)
	fmt.Fprint(w, `<section id="deadletters"><h2>Recent transport dead letters</h2>`)
	if len(deadLetters) == 0 {
		fmt.Fprint(w, `<p class="muted">No persisted transport dead letters are currently stored.</p>`)
	} else {
		fmt.Fprint(w, `<pre>`+asJSON(deadLetters)+`</pre>`)
	}
	fmt.Fprint(w, `</section>`)
	fmt.Fprint(w, `<section id="nodes"><h2>Nodes</h2>`)
	if len(snap.Nodes) == 0 {
		fmt.Fprint(w, `<p class="muted">Node inventory is empty because no live observations have been stored yet.</p>`)
	} else {
		fmt.Fprint(w, `<table><tr><th>Node</th><th>ID</th><th>Name</th><th>Last Seen</th><th>Gateway</th></tr>`)
		for _, n := range snap.Nodes {
			fmt.Fprintf(w, `<tr><td>%d</td><td>%s</td><td>%s %s</td><td>%s</td><td>%s</td></tr>`, n.Num, n.ID, n.LongName, n.ShortName, n.LastSeen, n.GatewayID)
		}
		fmt.Fprint(w, `</table>`)
	}
	fmt.Fprint(w, `</section><section id="messages"><h2>Recent messages</h2>`)
	if len(messages) == 0 {
		fmt.Fprint(w, `<p class="muted">No live message observations have been stored yet.</p>`)
	} else {
		fmt.Fprint(w, `<pre>`+asJSON(messages)+`</pre>`)
	}
	fmt.Fprint(w, `</section><section id="privacy"><h2>Privacy findings</h2>`)
	if len(findings) == 0 {
		fmt.Fprint(w, `<p>No active privacy findings for the current config.</p>`)
	} else {
		fmt.Fprint(w, `<ul>`)
		for _, finding := range findings {
			fmt.Fprintf(w, `<li class="sev-%s"><strong>[%s]</strong> %s<br><span class="muted">%s</span></li>`, finding.Severity, strings.ToUpper(finding.Severity), finding.Message, finding.Remediation)
		}
		fmt.Fprint(w, `</ul>`)
	}
	fmt.Fprint(w, `</section><section id="recommendations"><h2>Config recommendations</h2><pre>`+asJSON(s.recommendations())+`</pre></section>`)
	fmt.Fprint(w, `<section id="events"><h2>Logs / events</h2><pre>`+asJSON(logs)+`</pre></section></body></html>`)
}
func asJSON(v any) string { b, _ := json.MarshalIndent(v, "", "  "); return string(b) }

func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.cfg.Auth.Enabled {
			next.ServeHTTP(w, r)
			return
		}
		user, pass, ok := r.BasicAuth()
		if ok && user == s.cfg.Auth.UIUser && pass == s.cfg.Auth.UIPassword {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="mel"`)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]any{"code": "auth_required", "message": "authentication is required for this MEL endpoint"}})
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

func escape(v string) string { return strings.ReplaceAll(v, "'", "''") }

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
