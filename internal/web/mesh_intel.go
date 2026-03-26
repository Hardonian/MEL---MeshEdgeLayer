package web

import (
	"net/http"
	"time"

	"github.com/mel-project/mel/internal/meshintel"
)

func (s *Server) meshIntelBundle(now time.Time) meshintel.Assessment {
	if topologyStoreGlobal == nil || s.db == nil {
		return meshintel.Assessment{}
	}
	transportOK := s.topologyTransportConnected()
	if !transportOK {
		transportOK = meshintel.TransportLikelyConnectedFromRuntime(s.db)
	}
	if s.meshIntelLatest != nil {
		if a, ok := s.meshIntelLatest(); ok {
			return a
		}
	}
	return meshintel.ComputeLive(s.cfg, s.db, topologyStoreGlobal, transportOK, now)
}

// meshIntelligenceHandler GET /api/v1/mesh/intelligence
func (s *Server) meshIntelligenceHandler(w http.ResponseWriter, r *http.Request) {
	if topologyStoreGlobal == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	a := s.meshIntelBundle(time.Now().UTC())
	writeJSON(w, http.StatusOK, a)
}

// meshIntelligenceHistoryHandler GET /api/v1/mesh/intelligence/history
func (s *Server) meshIntelligenceHistoryHandler(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "database unavailable"})
		return
	}
	limit := intParam(r, "limit", 20)
	list, err := meshintel.RecentSnapshots(s.db, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"assessments": list, "count": len(list)})
}

func findNodeIntel(a meshintel.Assessment, nodeNum int64) *meshintel.NodeTopologyIntel {
	for i := range a.NodeIntel {
		if a.NodeIntel[i].NodeNum == nodeNum {
			return &a.NodeIntel[i]
		}
	}
	return nil
}
