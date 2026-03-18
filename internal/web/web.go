package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

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
	mux.HandleFunc("/", s.ui)
	s.http = &http.Server{Addr: cfg.Bind.API, Handler: mux}
	return s
}
func (s *Server) Start(ctx context.Context) {
	go func() { <-ctx.Done(); _ = s.http.Shutdown(context.Background()) }()
	s.log.Info("web starting", map[string]any{"addr": s.cfg.Bind.API})
	_ = s.http.ListenAndServe()
}
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"ok": true})
}
func (s *Server) readyz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"ready": true, "transports": s.transportHealth()})
}
func (s *Server) status(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"snapshot": s.state.Snapshot(), "transports": s.transportHealth()})
}
func (s *Server) nodes(w http.ResponseWriter, _ *http.Request) {
	rows, _ := s.db.QueryJSON("SELECT node_num,node_id,long_name,short_name,last_seen,last_gateway_id,lat_redacted,lon_redacted,altitude FROM nodes ORDER BY updated_at DESC;")
	writeJSON(w, map[string]any{"nodes": rows})
}
func (s *Server) transports(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"transports": s.transportHealth()})
}
func (s *Server) audit(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"findings": privacy.Audit(s.cfg)})
}
func (s *Server) recs(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"recommendations": s.recommendations()})
}
func (s *Server) logs(w http.ResponseWriter, _ *http.Request) {
	rows, _ := s.db.QueryJSON("SELECT category,level,message,created_at FROM audit_logs ORDER BY id DESC LIMIT 100;")
	writeJSON(w, map[string]any{"events": rows})
}
func (s *Server) ui(w http.ResponseWriter, _ *http.Request) {
	snap := s.state.Snapshot()
	sort.Slice(snap.Nodes, func(i, j int) bool { return snap.Nodes[i].Num < snap.Nodes[j].Num })
	fmt.Fprintf(w, `<!doctype html><html><head><title>MEL</title><style>body{font-family:sans-serif;max-width:1000px;margin:2rem auto;padding:0 1rem}table{border-collapse:collapse;width:100%%}td,th{border:1px solid #ccc;padding:.4rem}.muted{color:#666}</style></head><body><h1>MEL — MeshEdgeLayer</h1><p>Privacy-first, local-first observability for stock Meshtastic nodes.</p><h2>Dashboard</h2><p>Messages observed: %d</p><h2>Transports</h2><pre>%s</pre><h2>Nodes</h2>`, snap.Messages, asJSON(s.transportHealth()))
	if len(snap.Nodes) == 0 {
		fmt.Fprint(w, `<p class="muted">No radio packets have been ingested yet. Connect a supported transport and wait for live traffic.</p>`)
	} else {
		fmt.Fprint(w, `<table><tr><th>Node</th><th>ID</th><th>Name</th><th>Last Seen</th><th>Gateway</th></tr>`)
		for _, n := range snap.Nodes {
			fmt.Fprintf(w, `<tr><td>%d</td><td>%s</td><td>%s %s</td><td>%s</td><td>%s</td></tr>`, n.Num, n.ID, n.LongName, n.ShortName, n.LastSeen, n.GatewayID)
		}
		fmt.Fprint(w, `</table>`)
	}
	fmt.Fprintf(w, `<h2>Privacy Audit</h2><pre>%s</pre><h2>Recommendations</h2><pre>%s</pre></body></html>`, asJSON(privacy.Audit(s.cfg)), asJSON(s.recommendations()))
}
func asJSON(v any) string { b, _ := json.MarshalIndent(v, "", "  "); return string(b) }
