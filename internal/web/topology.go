package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/topology"
)

// topologyStore is set during wiring from the service layer.
var topologyStoreGlobal *topology.Store

// SetTopologyStore wires the topology store into the web handlers.
func (s *Server) SetTopologyStore(ts *topology.Store) {
	topologyStoreGlobal = ts
}

func (s *Server) topologyThresholds() topology.StaleThresholds {
	return topology.StaleThresholdsFromConfig(s.cfg.Topology.NodeStaleMinutes, s.cfg.Topology.LinkStaleMinutes)
}

func (s *Server) topologyTransportConnected() bool {
	if s.topologyTransportLive != nil {
		return s.topologyTransportLive()
	}
	for _, h := range s.transportHealth() {
		if h.State == "live" || h.State == "idle" {
			return true
		}
	}
	return false
}

// topologyIntelligenceHandler GET /api/v1/topology — summary + full analysis (bounded).
func (s *Server) topologyIntelligenceHandler(w http.ResponseWriter, r *http.Request) {
	if topologyStoreGlobal == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	if !s.cfg.Topology.Enabled {
		writeJSON(w, http.StatusOK, map[string]any{
			"topology_enabled": false,
			"message":          "topology model disabled in config",
		})
		return
	}
	nodes, err := topologyStoreGlobal.ListNodes(5000)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load nodes"})
		return
	}
	links, err := topologyStoreGlobal.ListLinks(10000)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load links"})
		return
	}
	th := s.topologyThresholds()
	ar := topology.Analyze(nodes, links, th, time.Now().UTC())
	view := topology.BuildIntelligenceView(s.cfg, ar, s.topologyTransportConnected(), time.Now().UTC())
	writeJSON(w, http.StatusOK, view)
}

// --- Topology nodes with health scoring ---

func (s *Server) topologyNodesHandler(w http.ResponseWriter, r *http.Request) {
	if topologyStoreGlobal == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	limit := intParam(r, "limit", 500)
	nodes, err := topologyStoreGlobal.ListNodes(limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to list nodes", "detail": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"nodes": nodes,
		"count": len(nodes),
		"limit": limit,
	})
}

func (s *Server) topologyNodeDetailHandler(w http.ResponseWriter, r *http.Request) {
	if topologyStoreGlobal == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	// Extract node_num from path: /api/v1/topology/nodes/{num}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/topology/nodes/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "node_num required"})
		return
	}
	nodeNum, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid node_num"})
		return
	}

	node, found, err := topologyStoreGlobal.GetNode(nodeNum)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to get node"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "node not found"})
		return
	}

	// Get connected links
	links, _ := topologyStoreGlobal.LinksForNode(nodeNum)

	// Get recent observations
	observations, _ := topologyStoreGlobal.RecentObservations(nodeNum, 50)

	// Get bookmarks
	bookmarks, _ := topologyStoreGlobal.BookmarksForNode(nodeNum)

	th := s.topologyThresholds()
	drill := topology.BuildNodeDrilldown(node, links, bookmarks, observations, th, time.Now().UTC())
	writeJSON(w, http.StatusOK, drill)
}

// --- Topology links ---

func (s *Server) topologyLinksHandler(w http.ResponseWriter, r *http.Request) {
	if topologyStoreGlobal == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	limit := intParam(r, "limit", 500)
	links, err := topologyStoreGlobal.ListLinks(limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to list links"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"links": links,
		"count": len(links),
		"limit": limit,
	})
}

// --- Topology analysis ---

func (s *Server) topologyAnalysisHandler(w http.ResponseWriter, r *http.Request) {
	if topologyStoreGlobal == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	nodes, err := topologyStoreGlobal.ListNodes(5000)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load nodes"})
		return
	}
	links, err := topologyStoreGlobal.ListLinks(10000)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load links"})
		return
	}
	th := s.topologyThresholds()
	result := topology.Analyze(nodes, links, th, time.Now().UTC())
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) topologyLinkDetailHandler(w http.ResponseWriter, r *http.Request) {
	if topologyStoreGlobal == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	edgeID := strings.TrimPrefix(r.URL.Path, "/api/v1/topology/links/")
	edgeID = strings.Trim(edgeID, "/")
	if edgeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "edge_id required"})
		return
	}
	l, ok, err := topologyStoreGlobal.GetLink(edgeID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load link"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "link not found"})
		return
	}
	th := s.topologyThresholds()
	drill := topology.BuildLinkDrilldown(l, th, time.Now().UTC())
	writeJSON(w, http.StatusOK, drill)
}

func (s *Server) topologySegmentHandler(w http.ResponseWriter, r *http.Request) {
	if topologyStoreGlobal == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/topology/segments/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "segment id required"})
		return
	}
	segID := parts[0]
	nodes, err := topologyStoreGlobal.ListNodes(5000)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load nodes"})
		return
	}
	links, err := topologyStoreGlobal.ListLinks(10000)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load links"})
		return
	}
	th := s.topologyThresholds()
	ar := topology.Analyze(nodes, links, th, time.Now().UTC())
	dd, ok := topology.BuildClusterDrilldown(segID, ar.WeakClusters, ar.Clusters)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "segment not found"})
		return
	}
	writeJSON(w, http.StatusOK, dd)
}

// --- Topology snapshots ---

func (s *Server) topologySnapshotsHandler(w http.ResponseWriter, r *http.Request) {
	if topologyStoreGlobal == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	limit := intParam(r, "limit", 20)
	snapshots, err := topologyStoreGlobal.RecentSnapshots(limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load snapshots"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"snapshots": snapshots,
		"count":     len(snapshots),
	})
}

// --- Source trust ---

func (s *Server) sourceTrustHandler(w http.ResponseWriter, r *http.Request) {
	if topologyStoreGlobal == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	trusts, err := topologyStoreGlobal.ListSourceTrust()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load source trust"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"sources": trusts,
		"count":   len(trusts),
	})
}

// --- Bookmarks ---

func (s *Server) bookmarksHandler(w http.ResponseWriter, r *http.Request) {
	if topologyStoreGlobal == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		bmType := r.URL.Query().Get("type")
		limit := intParam(r, "limit", 100)
		bookmarks, err := topologyStoreGlobal.ListBookmarks(bmType, limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to list bookmarks"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"bookmarks": bookmarks, "count": len(bookmarks)})

	case http.MethodPost:
		var bm topology.Bookmark
		if err := json.NewDecoder(r.Body).Decode(&bm); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
			return
		}
		if bm.NodeNum == 0 || bm.BookmarkType == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "node_num and bookmark_type required"})
			return
		}
		if bm.ActorID == "" {
			bm.ActorID = "operator"
		}
		bm.Active = true
		if err := topologyStoreGlobal.UpsertBookmark(bm); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to save bookmark"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "bookmark": bm})

	case http.MethodDelete:
		nodeNumStr := r.URL.Query().Get("node_num")
		bmType := r.URL.Query().Get("type")
		if nodeNumStr == "" || bmType == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "node_num and type required"})
			return
		}
		nodeNum, err := strconv.ParseInt(nodeNumStr, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid node_num"})
			return
		}
		if err := topologyStoreGlobal.DeleteBookmark(nodeNum, bmType); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to delete bookmark"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted"})
	}
}

// --- Recovery state ---

func (s *Server) recoveryStateHandler(w http.ResponseWriter, r *http.Request) {
	if topologyStoreGlobal == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	rs, err := topologyStoreGlobal.GetRecoveryState()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load recovery state"})
		return
	}
	writeJSON(w, http.StatusOK, rs)
}

// --- Topology export ---

func (s *Server) topologyExportHandler(w http.ResponseWriter, r *http.Request) {
	if topologyStoreGlobal == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	nodes, _ := topologyStoreGlobal.ListNodes(5000)
	links, _ := topologyStoreGlobal.ListLinks(10000)
	trusts, _ := topologyStoreGlobal.ListSourceTrust()
	bookmarks, _ := topologyStoreGlobal.ListBookmarks("", 1000)
	snapshots, _ := topologyStoreGlobal.RecentSnapshots(10)

	// Redact coordinates if privacy requires it
	if s.cfg.Privacy.RedactExports {
		for i := range nodes {
			nodes[i].LatRedacted = 0
			nodes[i].LonRedacted = 0
			nodes[i].LocationState = topology.LocRedacted
		}
	}

	bundle := map[string]any{
		"version":          "mel-topology-export-v1",
		"exported_at":      time.Now().UTC().Format(time.RFC3339),
		"privacy_redacted": s.cfg.Privacy.RedactExports,
		"nodes":            nodes,
		"links":            links,
		"sources":          trusts,
		"bookmarks":        bookmarks,
		"snapshots":        snapshots,
		"node_count":       len(nodes),
		"link_count":       len(links),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=mel-topology-%s.json", time.Now().UTC().Format("20060102-150405")))
	json.NewEncoder(w).Encode(bundle)
}

// intParam parses an integer query parameter with a default.
func intParam(r *http.Request, name string, def int) int {
	v := r.URL.Query().Get(name)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
