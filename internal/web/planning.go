package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/db"
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
	var retro planning.RecommendationRetrospective
	if s.db != nil && len(mi.Recommendations) > 0 {
		r := mi.Recommendations[0]
		key := planning.RecordRecommendationOutcomeKey(r.Rank, r.Class)
		if r2, err := planning.RecommendationRetrospectiveForKey(s.db, key); err == nil {
			retro = r2
		}
	}
	b := planning.BuildBundle(s.cfg, ar, mi, transportOK, now, retro)
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

// planningPlanItemHandler GET/PUT /api/v1/planning/plans/{id}
func (s *Server) planningPlanItemHandler(w http.ResponseWriter, r *http.Request) {
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
	switch r.Method {
	case http.MethodGet, http.MethodHead:
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
	case http.MethodPut:
		var p planning.DeploymentPlan
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
			return
		}
		p.PlanID = id
		if err := planning.SavePlan(s.db, &p); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		saved, _, _ := planning.GetPlan(s.db, id)
		writeJSON(w, http.StatusOK, saved)
	default:
		w.Header().Set("Allow", "GET, HEAD, PUT")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

type inputSetRequest struct {
	InputSetID string `json:"input_set_id"`
	Title      string `json:"title"`
}

type inputVersionRequest struct {
	InputSetID string                    `json:"input_set_id"`
	Items      []planning.AssumptionItem `json:"items"`
}

// planningInputsHandler GET/POST /api/v1/planning/inputs
func (s *Server) planningInputsHandler(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "database unavailable"})
		return
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		list, err := planning.ListInputSets(s.db, 200)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"input_sets": list, "count": len(list)})
	case http.MethodPost:
		var req inputSetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
			return
		}
		id, err := planning.SaveInputSet(s.db, req.InputSetID, req.Title)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"input_set_id": id})
	default:
		w.Header().Set("Allow", "GET, HEAD, POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

// planningInputVersionHandler POST /api/v1/planning/input-versions
func (s *Server) planningInputVersionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if s.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "database unavailable"})
		return
	}
	now := time.Now().UTC()
	ar, mi, ok := s.planningContext(now)
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	var req inputVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return
	}
	if strings.TrimSpace(req.InputSetID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "input_set_id required"})
		return
	}
	payload := planning.BuildInputVersionPayload(req.InputSetID, 0, req.Items, ar.Snapshot.GraphHash, mi.AssessmentID)
	vid, err := planning.SaveInputVersion(s.db, req.InputSetID, payload)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	p, _, _ := planning.GetInputVersion(s.db, vid)
	writeJSON(w, http.StatusOK, map[string]any{"version_id": vid, "payload": p})
}

// planningInputVersionGetHandler GET /api/v1/planning/input-versions/{id}
func (s *Server) planningInputVersionGetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if s.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "database unavailable"})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/planning/input-versions/")
	id = strings.Trim(id, "/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "version_id required"})
		return
	}
	p, ok, err := planning.GetInputVersion(s.db, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

type executionStartReq struct {
	PlanID           string `json:"plan_id"`
	ObservationHours int    `json:"observation_hours"`
	Notes            string `json:"notes"`
}

type stepMarkReq struct {
	ExecutionID string `json:"execution_id"`
	StepID      string `json:"step_id"`
	Note        string `json:"note"`
}

type validateReq struct {
	ExecutionID string `json:"execution_id"`
}

// planningExecutionStartHandler POST /api/v1/planning/executions/start
func (s *Server) planningExecutionStartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if s.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "database unavailable"})
		return
	}
	now := time.Now().UTC()
	ar, mi, ok := s.planningContext(now)
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	var req executionStartReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return
	}
	sum, _ := planning.ComputeResilience(ar, mi)
	base := planning.PostChangeMetricsSnapshot{
		FragmentationBefore: mi.Topology.FragmentationScore,
		ResilienceBefore:    sum.ResilienceScore,
	}
	eid, err := planning.StartPlanExecution(s.db, req.PlanID, ar.Snapshot.GraphHash, mi.AssessmentID, base, req.ObservationHours, req.Notes)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"execution_id": eid})
}

// planningExecStepHandler POST /api/v1/planning/executions/step
func (s *Server) planningExecStepHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if s.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "database unavailable"})
		return
	}
	var req stepMarkReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return
	}
	sid, err := planning.MarkStepExecuted(s.db, req.ExecutionID, req.StepID, req.Note)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"step_execution_id": sid})
}

// planningExecValidateHandler POST /api/v1/planning/executions/validate
func (s *Server) planningExecValidateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if s.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "database unavailable"})
		return
	}
	now := time.Now().UTC()
	ar, miAfter, ok := s.planningContext(now)
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	var req validateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return
	}
	exec, found, err := planning.GetPlanExecution(s.db, req.ExecutionID)
	if err != nil || !found {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "execution not found"})
		return
	}
	miBefore := miAfter
	if strings.TrimSpace(exec.MeshAssessmentID) != "" {
		if a, ok2, err := meshintel.GetAssessmentByID(s.db, exec.MeshAssessmentID); err == nil && ok2 {
			miBefore = a
		} else if s.meshIntelLatest != nil {
			if a, ok2 := s.meshIntelLatest(); ok2 && exec.MeshAssessmentID == a.AssessmentID {
				miBefore = a
			}
		}
	}
	vr := planning.ValidateExecution(exec, miBefore, ar, miAfter, now)
	vid, err := planning.SaveValidation(s.db, req.ExecutionID, vr)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	key := ""
	if len(miAfter.Recommendations) > 0 {
		r0 := miAfter.Recommendations[0]
		key = planning.RecordRecommendationOutcomeKey(r0.Rank, r0.Class)
	}
	if key != "" {
		_ = planning.RecordRecommendationOutcome(s.db, key, miAfter.GraphHash, miAfter.AssessmentID, vr.Verdict, vr)
	}
	writeJSON(w, http.StatusOK, map[string]any{"validation_id": vid, "validation": vr})
}

// planningPlanExecutionsHandler GET /api/v1/planning/executions?plan_id=
func (s *Server) planningPlanExecutionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if s.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "database unavailable"})
		return
	}
	planID := strings.TrimSpace(r.URL.Query().Get("plan_id"))
	if planID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "plan_id query required"})
		return
	}
	list, err := planning.ListPlanExecutions(s.db, planID, 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"executions": list})
}

// planningExecutionValidationsHandler GET /api/v1/planning/executions/validations?execution_id=
func (s *Server) planningExecutionValidationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if s.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "database unavailable"})
		return
	}
	eid := strings.TrimSpace(r.URL.Query().Get("execution_id"))
	if eid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "execution_id query required"})
		return
	}
	list, err := planning.ListValidationsForExecution(s.db, eid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"validations": list, "count": len(list)})
}

// planningAdvisoryAlertsHandler GET /api/v1/planning/advisory-alerts
func (s *Server) planningAdvisoryAlertsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if s.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "database unavailable"})
		return
	}
	alerts, err := s.db.ListPlanningAdvisoryAlerts(true)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"advisory_nature": "synthetic topology advisories stored under transport planning/advisory",
		"alerts":          alertsDTO(alerts),
		"count":           len(alerts),
		"evidence_flags": planning.PlanningEvidenceFlags{
			NoAdvisories: len(alerts) == 0,
		},
	})
}

func alertsDTO(in []db.TransportAlertRecord) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, a := range in {
		out = append(out, map[string]any{
			"id":                   a.ID,
			"severity":             a.Severity,
			"reason":               a.Reason,
			"summary":              a.Summary,
			"cluster_key":          a.ClusterKey,
			"contributing_reasons": a.ContributingReasons,
			"trigger_condition":    a.TriggerCondition,
			"last_updated_at":      a.LastUpdatedAt,
			"active":               a.Active,
		})
	}
	return out
}

// planningRecommendNextHandler GET /api/v1/planning/recommend/next
func (s *Server) planningRecommendNextHandler(w http.ResponseWriter, r *http.Request) {
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
	var retro planning.RecommendationRetrospective
	if s.db != nil && len(mi.Recommendations) > 0 {
		r0 := mi.Recommendations[0]
		key := planning.RecordRecommendationOutcomeKey(r0.Rank, r0.Class)
		if r2, err := planning.RecommendationRetrospectiveForKey(s.db, key); err == nil {
			retro = r2
		}
	}
	bm := planning.ComputeBestNextMove(ar, mi, retro)
	writeJSON(w, http.StatusOK, bm)
}

type outcomeRecordReq struct {
	RecommendationKey string                  `json:"recommendation_key"`
	Verdict           planning.OutcomeVerdict `json:"verdict"`
	Payload           map[string]any          `json:"payload"`
}

// planningOutcomeRecordHandler POST /api/v1/planning/outcomes
func (s *Server) planningOutcomeRecordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if s.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "database unavailable"})
		return
	}
	now := time.Now().UTC()
	_, mi, ok := s.planningContext(now)
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "topology store not initialized"})
		return
	}
	var req outcomeRecordReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return
	}
	if err := planning.RecordRecommendationOutcome(s.db, req.RecommendationKey, mi.GraphHash, mi.AssessmentID, req.Verdict, req.Payload); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// planningRetrospectiveHandler GET /api/v1/planning/retrospective?key=
func (s *Server) planningRetrospectiveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if s.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "database unavailable"})
		return
	}
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "key query required"})
		return
	}
	retro, err := planning.RecommendationRetrospectiveForKey(s.db, key)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, retro)
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
	PlanIDs []string                  `json:"plan_ids"`
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
