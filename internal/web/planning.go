package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/planning"
	"github.com/mel-project/mel/internal/topology"
)

func (s *Server) planningContext(now time.Time) (topology.AnalysisResult, meshintel.Assessment, bool) {
	if topologyStoreGlobal == nil {
		return topology.AnalysisResult{}, meshintel.Assessment{}, false
	}
	transportOK := s.topologyTransportConnected()
	if s.db != nil && !transportOK {
		transportOK = meshintel.TransportLikelyConnectedFromRuntime(s.db)
	}
	th := s.topologyThresholds()
	nodes, err := topologyStoreGlobal.ListNodes(5000)
	if err != nil {
		nodes = nil
	}
	links, err := topologyStoreGlobal.ListLinks(10000)
	if err != nil {
		links = nil
	}
	ar := topology.Analyze(nodes, links, th, now)
	mi := s.meshIntelBundle(now)
	return ar, mi, true
}

// planningBundleHandler GET /api/v1/planning/bundle
func (s *Server) planningBundleHandler(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	ar, mi, ok := s.planningContext(now)
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	transportOK := s.topologyTransportConnected()
	if s.db != nil && !transportOK {
		transportOK = meshintel.TransportLikelyConnectedFromRuntime(s.db)
	}
	b := planning.BuildBundle(s.cfg, ar, mi, transportOK, now)
	writeJSON(w, http.StatusOK, b)
}

// planningPlansHandler GET/POST /api/v1/planning/plans
func (s *Server) planningPlansHandler(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "database unavailable"})
		return
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		list, err := planning.ListPlans(s.db, 200)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"plans": list, "count": len(list)})
	case http.MethodPost:
		var p planning.DeploymentPlan
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
			return
		}
		if err := planning.SavePlan(s.db, &p); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		saved, _, _ := planning.GetPlan(s.db, p.PlanID)
		writeJSON(w, http.StatusOK, saved)
	default:
		w.Header().Set("Allow", "GET, HEAD, POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

// planningPlanItemHandler GET /api/v1/planning/plans/{id}
func (s *Server) planningPlanItemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if s.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "database unavailable"})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/planning/plans/")
	id = strings.Trim(id, "/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "plan_id required"})
		return
	}
	p, ok, err := planning.GetPlan(s.db, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "plan not found"})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

type scenarioRequest struct {
	Kind           string `json:"kind"`
	NodeNum        int64  `json:"node_num"`
	CandidateClass string `json:"candidate_class,omitempty"`
}

// planningScenarioHandler POST /api/v1/planning/scenario
func (s *Server) planningScenarioHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	now := time.Now().UTC()
	ar, mi, ok := s.planningContext(now)
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	var req scenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return
	}
	kind, kOK := planning.NormalizeScenarioKind(req.Kind)
	if !kOK {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unknown scenario kind"})
		return
	}
	sa := planning.RunScenarioWithClass(kind, req.NodeNum, req.CandidateClass, ar, mi, now)
	if s.db != nil {
		_ = planning.SaveArtifact(s.db, "scenario", ar.Snapshot.GraphHash, mi.AssessmentID, sa, 300)
	}
	writeJSON(w, http.StatusOK, sa)
}

type compareRequest struct {
	PlanIDs []string               `json:"plan_ids"`
	Plans   []planning.DeploymentPlan `json:"plans,omitempty"`
}

// planningCompareHandler POST /api/v1/planning/compare
func (s *Server) planningCompareHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	now := time.Now().UTC()
	ar, mi, ok := s.planningContext(now)
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	var req compareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return
	}
	var plans []planning.DeploymentPlan
	if len(req.Plans) > 0 {
		plans = req.Plans
	} else if s.db != nil {
		for _, id := range req.PlanIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if p, found, err := planning.GetPlan(s.db, id); err == nil && found {
				plans = append(plans, p)
			}
		}
	}
	pc := planning.ComparePlans(plans, ar, mi, now)
	if s.db != nil {
		_ = planning.SaveArtifact(s.db, "compare", ar.Snapshot.GraphHash, mi.AssessmentID, pc, 300)
	}
	writeJSON(w, http.StatusOK, pc)
}

// planningImpactHandler GET /api/v1/planning/impact?kind=remove&node=1
func (s *Server) planningImpactHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	now := time.Now().UTC()
	ar, mi, ok := s.planningContext(now)
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	kindStr := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("kind")))
	nodeStr := strings.TrimSpace(r.URL.Query().Get("node"))
	classStr := strings.TrimSpace(r.URL.Query().Get("class"))
	var nodeNum int64
	if nodeStr != "" {
		n, err := strconv.ParseInt(nodeStr, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid node"})
			return
		}
		nodeNum = n
	}
	var ik planning.ImpactKind
	switch kindStr {
	case "add":
		ik = planning.ImpactAdd
	case "move", "elevate":
		ik = planning.ImpactMove
	case "remove":
		ik = planning.ImpactRemove
	case "role":
		ik = planning.ImpactRole
	case "uptime":
		ik = planning.ImpactUptime
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "kind must be add|move|remove|role|uptime"})
		return
	}
	cc := planning.ParseImpactCandidateClass(classStr)
	ni := planning.EstimateImpact(ik, nodeNum, cc, ar, mi)
	writeJSON(w, http.StatusOK, ni)
}

// planningPlaybooksHandler GET /api/v1/planning/playbooks
func (s *Server) planningPlaybooksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	now := time.Now().UTC()
	ar, mi, ok := s.planningContext(now)
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	pb := planning.SuggestPlaybooks(ar, mi)
	writeJSON(w, http.StatusOK, map[string]any{"playbooks": pb, "count": len(pb)})
}
