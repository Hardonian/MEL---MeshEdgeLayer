package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/logging"
	"github.com/mel-project/mel/internal/meshstate"
	"github.com/mel-project/mel/internal/policy"
	"github.com/mel-project/mel/internal/privacy"
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
}

func New(cfg config.Config, log *logging.Logger, d *db.DB, st *meshstate.State, bus *events.Bus, th func() []transport.Health, rec func() []policy.Recommendation) *Server {
	s := &Server{cfg: cfg, log: log, db: d, state: st, bus: bus, transportHealth: th, recommendations: rec}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/readyz", s.readyz)
	mux.HandleFunc("/api/status", s.status)
	mux.HandleFunc("/api/nodes", s.nodes)
	mux.HandleFunc("/api/transports", s.transports)
	mux.HandleFunc("/api/privacy/audit", s.audit)
	mux.HandleFunc("/api/recommendations", s.recs)
	mux.HandleFunc("/api/logs", s.logs)
	mux.HandleFunc("/api/v1/status", s.status)
	mux.HandleFunc("/api/v1/nodes", s.nodes)
	mux.HandleFunc("/api/v1/node/", s.nodeDetail)
	mux.HandleFunc("/api/v1/transports", s.transports)
	mux.HandleFunc("/api/v1/messages", s.messages)
	mux.HandleFunc("/api/v1/privacy/audit", s.audit)
	mux.HandleFunc("/api/v1/policy/explain", s.recs)
	mux.HandleFunc("/api/v1/events", s.logs)
	if cfg.Features.WebUI {
		mux.HandleFunc("/", s.ui)
	}
	s.http = &http.Server{Addr: cfg.Bind.API, Handler: s.withAuth(mux), ReadHeaderTimeout: 5 * time.Second}
	return s
}
func (s *Server) Start(ctx context.Context) {
	go func() { <-ctx.Done(); _ = s.http.Shutdown(context.Background()) }()
	s.log.Info("web starting", map[string]any{"addr": s.cfg.Bind.API})
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
	health := s.transportHealth()
	writeJSON(w, http.StatusOK, map[string]any{
		"process_ready":  true,
		"ingest_ready":   readyForIngest(health),
		"operator_state": operatorState(health),
		"transports":     health,
	})
}
func (s *Server) status(w http.ResponseWriter, _ *http.Request) {
	schemaVersion, _ := s.db.SchemaVersion()
	persistedMessages, _ := s.db.Scalar("SELECT COUNT(*) FROM messages;")
	persistedNodes, _ := s.db.Scalar("SELECT COUNT(*) FROM nodes;")
	lastPersistedIngest, _ := s.db.Scalar("SELECT COALESCE(MAX(rx_time), '') FROM messages;")
	writeJSON(w, http.StatusOK, map[string]any{
		"runtime_snapshot": s.state.Snapshot(),
		"persisted_summary": map[string]any{
			"nodes":                  persistedNodes,
			"messages":               persistedMessages,
			"last_successful_ingest": lastPersistedIngest,
		},
		"snapshot_boundary":  "runtime_snapshot reflects only observations seen by the current MEL process; persisted_summary reflects SQLite history across restarts",
		"transports":         s.transportHealth(),
		"privacy_summary":    privacy.Summary(privacy.Audit(s.cfg)),
		"schema_version":     schemaVersion,
		"bind_local_default": !s.cfg.Bind.AllowRemote,
		"configured_modes":   configuredModes(s.cfg),
	})
}
func (s *Server) nodes(w http.ResponseWriter, _ *http.Request) {
	rows, err := s.db.QueryRows("SELECT node_num,node_id,long_name,short_name,last_seen,last_gateway_id,lat_redacted,lon_redacted,altitude FROM nodes ORDER BY updated_at DESC;")
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
	query := fmt.Sprintf("SELECT node_num,node_id,long_name,short_name,last_seen,last_gateway_id,lat_redacted,lon_redacted,altitude,last_snr,last_rssi FROM nodes WHERE CAST(node_num AS TEXT)='%s' OR node_id='%s' LIMIT 1;", escape(nodeID), escape(nodeID))
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
	writeJSON(w, http.StatusOK, map[string]any{"transports": s.transportHealth(), "configured_modes": configuredModes(s.cfg)})
}
func (s *Server) messages(w http.ResponseWriter, _ *http.Request) {
	rows, err := s.db.QueryRows("SELECT transport_name,packet_id,from_node,to_node,portnum,payload_text,rx_time,created_at FROM messages ORDER BY id DESC LIMIT 100;")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]any{"code": "db_query_failed", "message": err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": rows})
}
func (s *Server) audit(w http.ResponseWriter, _ *http.Request) {
	findings := privacy.Audit(s.cfg)
	writeJSON(w, http.StatusOK, map[string]any{"findings": findings, "summary": privacy.Summary(findings)})
}
func (s *Server) recs(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"recommendations": s.recommendations()})
}
func (s *Server) logs(w http.ResponseWriter, _ *http.Request) {
	rows, err := s.db.QueryRows("SELECT category,level,message,details_json,created_at FROM audit_logs ORDER BY id DESC LIMIT 100;")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]any{"code": "db_query_failed", "message": err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": rows})
}
func (s *Server) ui(w http.ResponseWriter, _ *http.Request) {
	snap := s.state.Snapshot()
	sort.Slice(snap.Nodes, func(i, j int) bool { return snap.Nodes[i].Num < snap.Nodes[j].Num })
	findings := privacy.Audit(s.cfg)
	messages, _ := s.db.QueryRows("SELECT transport_name,packet_id,from_node,to_node,portnum,payload_text,rx_time FROM messages ORDER BY id DESC LIMIT 20;")
	persistedMessages, _ := s.db.Scalar("SELECT COUNT(*) FROM messages;")
	persistedNodes, _ := s.db.Scalar("SELECT COUNT(*) FROM nodes;")
	lastPersistedIngest, _ := s.db.Scalar("SELECT COALESCE(MAX(rx_time), '') FROM messages;")
	logs, _ := s.db.QueryRows("SELECT category,level,message,created_at FROM audit_logs ORDER BY id DESC LIMIT 20;")
	fmt.Fprintf(w, `<!doctype html><html><head><title>MEL</title><meta name="viewport" content="width=device-width, initial-scale=1"><style>
body{font-family:system-ui,sans-serif;max-width:1200px;margin:2rem auto;padding:0 1rem;line-height:1.45;background:#fafafa;color:#111}
nav a{margin-right:1rem}section{background:#fff;border:1px solid #ddd;border-radius:8px;padding:1rem;margin:1rem 0}
table{border-collapse:collapse;width:100%%}td,th{border:1px solid #ddd;padding:.45rem;text-align:left;vertical-align:top}.muted{color:#666}.sev-critical{color:#8b0000}.sev-high{color:#b04a00}.sev-medium{color:#805b00}
code,pre{background:#f5f5f5;padding:.2rem .35rem;border-radius:4px;overflow:auto}ul{padding-left:1.25rem}.pill{display:inline-block;padding:.15rem .5rem;border:1px solid #ccc;border-radius:999px;margin-right:.35rem;margin-bottom:.35rem}
</style></head><body><h1>MEL — MeshEdgeLayer</h1><p>Truthful local-first observability for stock Meshtastic nodes. No demo data is injected when transports are idle.</p><nav><a href="#onboarding">Onboarding</a><a href="#status">Status</a><a href="#transports">Transport health</a><a href="#nodes">Nodes</a><a href="#messages">Messages</a><a href="#privacy">Privacy findings</a><a href="#recommendations">Recommendations</a><a href="#events">Events</a></nav>`)
	fmt.Fprint(w, `<section id="onboarding"><h2>Onboarding</h2><ol><li>Run <code>mel init --config /etc/mel/mel.json</code> if you do not have a config yet.</li><li>Run <code>mel doctor --config /etc/mel/mel.json</code> to validate direct-node reachability, local permissions, and privacy posture.</li><li>Prefer one real direct transport (<code>serial</code> or <code>tcp</code>) for Pi/Linux deployment, then start <code>mel serve --config /etc/mel/mel.json</code>.</li><li>Return here to confirm whether MEL is disconnected, connected but idle, or receiving real mesh packets.</li></ol></section>`)
	fmt.Fprint(w, `<section id="status"><h2>Status</h2><p>Configured transport modes: `)
	for _, mode := range configuredModes(s.cfg) {
		fmt.Fprintf(w, `<span class="pill">%s</span>`, mode)
	}
	fmt.Fprintf(w, `</p><p>Runtime process message count: <strong>%d</strong>.</p><p>Persisted message count: <strong>%s</strong>. Persisted node count: <strong>%s</strong>. Last persisted ingest: <strong>%s</strong>.</p>`, snap.Messages, blankIfEmpty(persistedMessages, "0"), blankIfEmpty(persistedNodes, "0"), blankIfEmpty(lastPersistedIngest, "none"))
	if len(snap.Nodes) == 0 {
		fmt.Fprint(w, `<p class="muted">The current MEL process has not observed any nodes yet. Persisted counts above may still show historical data from prior runs. No sample mesh data is shown.</p>`)
	} else {
		fmt.Fprintf(w, `<p>Observed nodes: <strong>%d</strong>.</p>`, len(snap.Nodes))
	}
	fmt.Fprint(w, `</section><section id="transports"><h2>Transport health</h2><table><tr><th>Name</th><th>Type</th><th>State</th><th>Operator view</th><th>Detail</th><th>Capabilities</th><th>Packets</th><th>Last attempt</th><th>Last packet</th><th>Last error</th></tr>`)
	for _, h := range s.transportHealth() {
		fmt.Fprintf(w, `<tr><td>%s<br><span class="muted">%s</span></td><td>%s</td><td><code>%s</code></td><td>%s</td><td>%s</td><td><pre>%s</pre></td><td>%d read / %d dropped<br><span class="muted">reconnect attempts: %d</span></td><td>%s</td><td>%s</td><td>%s</td></tr>`, h.Name, blankIfEmpty(h.Source, "—"), h.Type, blankIfEmpty(h.State, "unknown"), transportStateLabel(h), h.Detail, asJSON(h.Capabilities), h.PacketsRead, h.PacketsDropped, h.ReconnectAttempts, blankIfEmpty(h.LastAttemptAt, "—"), blankIfEmpty(h.LastPacketAt, "—"), blankIfEmpty(h.LastError, "—"))
	}
	fmt.Fprint(w, `</table><p class="muted">If multiple transports are enabled, operators must verify radio ownership and contention behavior themselves; MEL does not claim shared-radio arbitration that stock nodes do not provide.</p></section>`)
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

func escape(v string) string { return strings.ReplaceAll(v, "'", "''") }

func remoteClient(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func configuredModes(cfg config.Config) []string {
	out := make([]string, 0)
	for _, t := range cfg.Transports {
		if t.Enabled {
			out = append(out, t.Type)
		}
	}
	if len(out) == 0 {
		return []string{"none"}
	}
	return out
}

func transportStateLabel(h transport.Health) string {
	switch h.State {
	case "disabled":
		return "disabled"
	case "configured_not_attempted":
		return "configured but not yet started"
	case "attempting":
		return "connect in progress"
	case "connected_no_ingest_evidence":
		return "connected but idle"
	case "ingesting":
		return "live data flowing"
	case "error":
		return "error; see detail and last error"
	case "unsupported":
		return "unsupported in this release"
	case "configured_offline":
		return "configured; offline-only doctor evidence"
	default:
		if h.Unsupported {
			return "unsupported in this release"
		}
		if h.OK && h.PacketsRead == 0 {
			return "connected but idle"
		}
		if h.OK {
			return "live data flowing"
		}
		return "state unknown"
	}
}

func readyForIngest(health []transport.Health) bool {
	for _, h := range health {
		if h.State == "ingesting" {
			return true
		}
	}
	return false
}

func operatorState(health []transport.Health) string {
	if len(health) == 0 {
		return "idle_no_transports"
	}
	if readyForIngest(health) {
		return "ingesting"
	}
	for _, h := range health {
		if h.State == "error" || h.State == "unsupported" {
			return "degraded"
		}
		if h.State == "connected_no_ingest_evidence" {
			return "connected_no_ingest_evidence"
		}
		if h.State == "attempting" {
			return "attempting"
		}
	}
	return "configured_not_attempted"
}

var _ = remoteClient
