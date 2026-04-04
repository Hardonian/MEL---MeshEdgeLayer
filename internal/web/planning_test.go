package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/logging"
	"github.com/mel-project/mel/internal/investigation"
	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/meshstate"
	"github.com/mel-project/mel/internal/planning"
	"github.com/mel-project/mel/internal/topology"
	"github.com/mel-project/mel/internal/transport"
)

func setupPlanningServer(t *testing.T) (*Server, *db.DB) {
	t.Helper()
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Features.WebUI = false
	cfg.Topology.Enabled = true
	d, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	srv := New(cfg, logging.New("info", false), d, meshstate.New(), events.New(),
		func() []transport.Health { return nil },
		nil, nil, nil, nil, nil, nil, nil,
		func() investigation.Summary { return investigation.Summary{} })
	srv.SetTopologyStore(topology.NewStore(d))
	return srv, d
}

func TestPlanningBundleEndpointIncludesTypedEvidenceFlags(t *testing.T) {
	srv, _ := setupPlanningServer(t)
	srv.SetMeshIntelProvider(func() (meshintel.Assessment, bool) {
		return meshintel.Assessment{
			AssessmentID: "assessment-1",
			Bootstrap: meshintel.BootstrapAssessment{
				Confidence: meshintel.ConfidenceMedium,
			},
			Recommendations: []meshintel.MeshRecommendation{
				{Rank: 1, Title: "Hold rollout while uncertainty remains"},
			},
		}, true
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/planning/bundle", nil)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["evidence_model"] == nil {
		t.Fatalf("missing evidence_model")
	}
	if payload["evidence_flags"] == nil {
		t.Fatalf("missing evidence_flags")
	}
	flags := payload["evidence_flags"].(map[string]any)
	if flags["directional_only"] != true {
		t.Fatalf("expected directional_only true, got %#v", flags["directional_only"])
	}
	if flags["limited_confidence"] != true {
		t.Fatalf("expected limited_confidence true, got %#v", flags["limited_confidence"])
	}
	if flags["recommendation_present_with_uncertain_evidence"] != true {
		t.Fatalf("expected recommendation_present_with_uncertain_evidence true, got %#v", flags["recommendation_present_with_uncertain_evidence"])
	}
}

func TestPlanningAdvisoryAlertsEndpointIncludesNoAdvisoriesFlag(t *testing.T) {
	srv, _ := setupPlanningServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/planning/advisory-alerts", nil)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	flags, ok := payload["evidence_flags"].(map[string]any)
	if !ok {
		t.Fatalf("missing evidence_flags object")
	}
	if flags["no_advisories"] != true {
		t.Fatalf("expected no_advisories true, got %#v", flags["no_advisories"])
	}
}

func TestPlanningAdvisoryAlertsEndpointClearsNoAdvisoriesWhenAlertsExist(t *testing.T) {
	srv, d := setupPlanningServer(t)
	if err := d.UpsertPlanningAdvisoryAlert(
		"planning|bridge_fragility|1|graph",
		"warning",
		"bridge_node_fragility",
		"Advisory",
		"node:1",
		[]string{"observed_bridge_node"},
		"bridge_node_detected",
	); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/planning/advisory-alerts", nil)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	flags, ok := payload["evidence_flags"].(map[string]any)
	if !ok {
		t.Fatalf("missing evidence_flags object")
	}
	if flags["no_advisories"] != false {
		t.Fatalf("expected no_advisories false when alerts exist, got %#v", flags["no_advisories"])
	}
}

func TestPlanningExecutionValidateEndpointEmitsBaselineMissingFlags(t *testing.T) {
	srv, d := setupPlanningServer(t)
	srv.SetMeshIntelProvider(func() (meshintel.Assessment, bool) {
		return meshintel.Assessment{
			AssessmentID: "after-baseline-missing",
			GraphHash:    "graph-a",
			Topology:     meshintel.MeshTopologyMetrics{FragmentationScore: 0.50},
		}, true
	})
	execID := "pex-baseline-missing"
	now := time.Now().UTC().Format(time.RFC3339)
	err := d.Exec(fmt.Sprintf(`INSERT INTO plan_executions(execution_id, plan_id, plan_graph_hash, mesh_assessment_id, baseline_metrics_json, status, started_at, updated_at, observation_horizon_hours, notes)
		VALUES('%s','%s','%s','%s','%s','attempted','%s','%s',%d,'%s');`,
		execID, "plan-1", "graph-a", "", `{"captured":false}`, now, now, 0, "no baseline id recorded"))
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/planning/executions/validate", strings.NewReader(`{"execution_id":"`+execID+`"}`))
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	validation := payload["validation"].(map[string]any)
	flags := validation["evidence_flags"].(map[string]any)
	if flags["baseline_missing"] != true {
		t.Fatalf("expected baseline_missing true, got %#v", flags["baseline_missing"])
	}
	if flags["directional_only"] != true {
		t.Fatalf("expected directional_only true, got %#v", flags["directional_only"])
	}
	if flags["limited_confidence"] != true {
		t.Fatalf("expected limited_confidence true, got %#v", flags["limited_confidence"])
	}
}

func TestPlanningExecutionValidateEndpointEmitsConfoundedFlags(t *testing.T) {
	srv, d := setupPlanningServer(t)
	srv.SetMeshIntelProvider(func() (meshintel.Assessment, bool) {
		return meshintel.Assessment{
			AssessmentID: "assessment-same",
			GraphHash:    "graph-same",
			Topology:     meshintel.MeshTopologyMetrics{FragmentationScore: 0.40},
		}, true
	})
	execID, err := planning.StartPlanExecution(d, "plan-1", "graph-same", "assessment-same", planning.PostChangeMetricsSnapshot{
		Captured:            true,
		FragmentationBefore: 0.50,
		ResilienceBefore:    0.40,
	}, 0, "same assessment id")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/planning/executions/validate", strings.NewReader(`{"execution_id":"`+execID+`"}`))
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	validation := payload["validation"].(map[string]any)
	flags := validation["evidence_flags"].(map[string]any)
	if flags["confounded_same_assessment_context"] != true {
		t.Fatalf("expected confounded_same_assessment_context true, got %#v", flags["confounded_same_assessment_context"])
	}
	if flags["inconclusive"] != true {
		t.Fatalf("expected inconclusive true, got %#v", flags["inconclusive"])
	}
	if flags["limited_confidence"] != true {
		t.Fatalf("expected limited_confidence true, got %#v", flags["limited_confidence"])
	}
}

func TestPlanningExecutionValidateEndpointEmitsDriftAndInconclusiveFlags(t *testing.T) {
	srv, d := setupPlanningServer(t)
	srv.SetMeshIntelProvider(func() (meshintel.Assessment, bool) {
		return meshintel.Assessment{
			AssessmentID: "assessment-after",
			GraphHash:    "graph-after",
			Topology:     meshintel.MeshTopologyMetrics{FragmentationScore: 0.49},
		}, true
	})
	execID, err := planning.StartPlanExecution(d, "plan-1", "graph-plan", "assessment-before", planning.PostChangeMetricsSnapshot{
		Captured:            true,
		FragmentationBefore: 0.50,
		ResilienceBefore:    0.50,
	}, 0, "graph drift observed")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/planning/executions/validate", strings.NewReader(`{"execution_id":"`+execID+`"}`))
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	validation := payload["validation"].(map[string]any)
	flags := validation["evidence_flags"].(map[string]any)
	if flags["topology_or_graph_drift_detected"] != true {
		t.Fatalf("expected topology_or_graph_drift_detected true, got %#v", flags["topology_or_graph_drift_detected"])
	}
	if flags["inconclusive"] != true {
		t.Fatalf("expected inconclusive true, got %#v", flags["inconclusive"])
	}
	if flags["directional_only"] != true {
		t.Fatalf("expected directional_only true, got %#v", flags["directional_only"])
	}
}

func TestPlanningExecutionValidationsEndpointPreservesTypedEvidenceFlagsFromPersistence(t *testing.T) {
	srv, d := setupPlanningServer(t)

	execID, err := planning.StartPlanExecution(d, "plan-typed-evidence", "graph-before", "assessment-before", planning.PostChangeMetricsSnapshot{
		Captured:            true,
		FragmentationBefore: 0.61,
		ResilienceBefore:    0.39,
	}, 24, "typed evidence parity check")
	if err != nil {
		t.Fatal(err)
	}

	validation := planning.ValidationResult{
		ExecutionID:           execID,
		GraphHashAfter:        "graph-after",
		MeshAssessmentIDAfter: "assessment-after",
		Verdict:               planning.OutcomeVerdictInconclusive,
		EvidenceFlags: planning.PlanningEvidenceFlags{
			BaselineMissing:                            true,
			ConfoundedSameAssessmentContext:            false,
			DirectionalOnly:                            true,
			Inconclusive:                               true,
			TopologyOrGraphDriftDetected:               false,
			LimitedConfidence:                          true,
			NoAdvisories:                               false,
			RecommendationPresentWithUncertainEvidence: true,
		},
		Caveat: "persistence parity only",
		Lines: []string{
			"evidence posture is cautionary and typed",
		},
		Metrics: planning.PostChangeMetricsSnapshot{
			Captured:            true,
			FragmentationBefore: 0.61,
			FragmentationAfter:  0.59,
			ResilienceBefore:    0.39,
			ResilienceAfter:     0.41,
		},
	}
	if _, err := planning.SaveValidation(d, execID, validation); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/planning/executions/validations?execution_id="+execID, nil)
	rec := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Validations []planning.ValidationResult `json:"validations"`
		Count       int                         `json:"count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Count != 1 {
		t.Fatalf("expected count 1, got %d", payload.Count)
	}
	if len(payload.Validations) != 1 {
		t.Fatalf("expected one validation result, got %#v", payload.Validations)
	}
	gotValidation := payload.Validations[0]
	if gotValidation.ExecutionID != execID {
		t.Fatalf("expected execution_id %q, got %q", execID, gotValidation.ExecutionID)
	}
	if gotValidation.ValidationID == "" {
		t.Fatalf("expected persisted validation_id")
	}
	if !reflect.DeepEqual(gotValidation.EvidenceFlags, validation.EvidenceFlags) {
		t.Fatalf("expected evidence_flags %#v, got %#v", validation.EvidenceFlags, gotValidation.EvidenceFlags)
	}
}
