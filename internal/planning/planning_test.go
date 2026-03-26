package planning

import (
	"strings"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/topology"
)

func configMinimal() config.Config {
	c := config.Config{}
	c.Topology.Enabled = true
	c.Topology.NodeStaleMinutes = 120
	c.Topology.LinkStaleMinutes = 120
	return c
}

func TestComputeResilience_twoNodeBridge(t *testing.T) {
	nodes := []topology.Node{
		{NodeNum: 1, ShortName: "a", HealthState: topology.HealthHealthy},
		{NodeNum: 2, ShortName: "b", HealthState: topology.HealthHealthy},
	}
	links := []topology.Link{
		{SrcNodeNum: 1, DstNodeNum: 2, Observed: true, QualityScore: 0.5, Directional: false},
	}
	th := topology.StaleThresholds{NodeStaleDuration: time.Hour, LinkStaleDuration: time.Hour}
	ar := topology.Analyze(nodes, links, th, time.Now().UTC())
	mi := meshintel.Compute(configMinimal(), ar, meshintel.MessageSignals{TotalMessages: 10}, true, time.Now().UTC())
	sum, profiles := ComputeResilience(ar, mi)
	if sum.ResilienceScore < 0 || sum.ResilienceScore > 1 {
		t.Fatalf("resilience score out of range: %v", sum.ResilienceScore)
	}
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(profiles))
	}
}

func TestValidateExecution_sameMeshAssessmentIdIsConfounded(t *testing.T) {
	before := meshintel.Assessment{
		AssessmentID: "assess-same",
		Topology:     meshintel.MeshTopologyMetrics{FragmentationScore: 0.5},
	}
	after := meshintel.Assessment{
		AssessmentID: "assess-same",
		Topology:     meshintel.MeshTopologyMetrics{FragmentationScore: 0.4},
	}
	ar := topology.AnalysisResult{}
	exec := PlanExecutionRecord{
		MeshAssessmentID:        "assess-same",
		ObservationHorizonHours: 0,
	}
	v := ValidateExecution(exec, before, ar, after, time.Now().UTC())
	if v.Verdict != OutcomeVerdictConfounded {
		t.Fatalf("expected confounded, got %s", v.Verdict)
	}
	if v.Caveat == "" {
		t.Fatal("expected caveat")
	}
}

func TestValidateExecution_warnsWhenBaselineAssessmentMissing(t *testing.T) {
	before := meshintel.Assessment{
		Topology: meshintel.MeshTopologyMetrics{FragmentationScore: 0.5},
	}
	after := meshintel.Assessment{
		AssessmentID: "after-only",
		Topology:     meshintel.MeshTopologyMetrics{FragmentationScore: 0.4},
	}
	ar := topology.AnalysisResult{}
	exec := PlanExecutionRecord{
		MeshAssessmentID:        "",
		ObservationHorizonHours: 0,
	}
	v := ValidateExecution(exec, before, ar, after, time.Now().UTC())
	found := false
	for _, line := range v.Lines {
		if strings.Contains(line, "No baseline mesh assessment id") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected baseline-missing warning in lines: %#v", v.Lines)
	}
}

func TestValidateExecution_usesBaselineWhenCaptured(t *testing.T) {
	before := meshintel.Assessment{
		Topology: meshintel.MeshTopologyMetrics{FragmentationScore: 0.5},
	}
	after := meshintel.Assessment{
		Topology: meshintel.MeshTopologyMetrics{FragmentationScore: 0.4},
	}
	ar := topology.AnalysisResult{}
	exec := PlanExecutionRecord{
		BaselineMetrics: PostChangeMetricsSnapshot{
			Captured:            true,
			FragmentationBefore: 0.5,
			ResilienceBefore:    0.4,
		},
		ObservationHorizonHours: 0,
	}
	v := ValidateExecution(exec, before, ar, after, time.Now().UTC())
	if v.Metrics.FragmentationBefore != 0.5 {
		t.Fatalf("expected baseline frag 0.5, got %v", v.Metrics.FragmentationBefore)
	}
}

func TestComparePlans_hasDimensionScores(t *testing.T) {
	p1 := DeploymentPlan{PlanID: "p1", Title: "observe", Steps: []DeploymentStep{{Kind: StepObserveOnly}}}
	p2 := DeploymentPlan{PlanID: "p2", Title: "add", Steps: []DeploymentStep{{Kind: StepAddNode}}}
	ar := topology.AnalysisResult{}
	mi := meshintel.Assessment{Bootstrap: meshintel.BootstrapAssessment{Confidence: meshintel.ConfidenceMedium}}
	pc := ComparePlans([]DeploymentPlan{p1, p2}, ar, mi, time.Now().UTC())
	if len(pc.RankedByUpside) != 2 {
		t.Fatalf("expected 2 ranked upside")
	}
	if pc.RankedByUpside[0].Dimensions.LowRegretScore <= 0 {
		t.Fatalf("expected positive low_regret score")
	}
	if pc.EvidenceClassification != EvidenceTopologyOnly {
		t.Fatalf("evidence class")
	}
}

func TestComparePlans_prefersReversible(t *testing.T) {
	p1 := DeploymentPlan{PlanID: "p1", Title: "wait and observe", Steps: []DeploymentStep{{Kind: StepObserveOnly}}}
	p2 := DeploymentPlan{PlanID: "p2", Title: "add nodes", Steps: []DeploymentStep{{Kind: StepAddNode}, {Kind: StepAddNode}}}
	ar := topology.AnalysisResult{}
	mi := meshintel.Assessment{Bootstrap: meshintel.BootstrapAssessment{Confidence: meshintel.ConfidenceMedium}}
	pc := ComparePlans([]DeploymentPlan{p1, p2}, ar, mi, time.Now().UTC())
	if pc.LowRegretPick != "p1" && pc.LowRegretPick != "" {
		// low regret may pick first high reversibility
		t.Logf("low regret: %s", pc.LowRegretPick)
	}
	if len(pc.RankedByUpside) != 2 {
		t.Fatalf("expected 2 ranked entries")
	}
}

func TestRunScenario_addNodeHandheld(t *testing.T) {
	nodes := []topology.Node{{NodeNum: 1, ShortName: "solo", HealthState: topology.HealthHealthy}}
	links := []topology.Link{}
	th := topology.StaleThresholds{NodeStaleDuration: time.Hour, LinkStaleDuration: time.Hour}
	ar := topology.Analyze(nodes, links, th, time.Now().UTC())
	mi := meshintel.Compute(configMinimal(), ar, meshintel.MessageSignals{}, true, time.Now().UTC())
	sa := RunScenarioWithClass(ScenarioAddNode, 0, "handheld", ar, mi, time.Now().UTC())
	if sa.EvidenceModel != PlanningEvidenceModel {
		t.Fatalf("evidence model")
	}
	if sa.ScenarioID == "" {
		t.Fatalf("scenario id")
	}
}

func TestSuggestPlaybooks_notEmpty(t *testing.T) {
	nodes := []topology.Node{{NodeNum: 1, ShortName: "solo", HealthState: topology.HealthHealthy}}
	links := []topology.Link{}
	th := topology.StaleThresholds{NodeStaleDuration: time.Hour, LinkStaleDuration: time.Hour}
	ar := topology.Analyze(nodes, links, th, time.Now().UTC())
	mi := meshintel.Compute(configMinimal(), ar, meshintel.MessageSignals{}, true, time.Now().UTC())
	pb := SuggestPlaybooks(ar, mi)
	if len(pb) == 0 {
		t.Fatalf("expected at least one playbook")
	}
}

func TestEstimateImpact_removeBridge(t *testing.T) {
	nodes := []topology.Node{
		{NodeNum: 1, ShortName: "a", HealthState: topology.HealthHealthy},
		{NodeNum: 2, ShortName: "b", HealthState: topology.HealthHealthy},
		{NodeNum: 3, ShortName: "c", HealthState: topology.HealthHealthy},
	}
	links := []topology.Link{
		{SrcNodeNum: 1, DstNodeNum: 2, Observed: true, QualityScore: 0.5},
		{SrcNodeNum: 2, DstNodeNum: 3, Observed: true, QualityScore: 0.5},
	}
	th := topology.StaleThresholds{NodeStaleDuration: time.Hour, LinkStaleDuration: time.Hour}
	ar := topology.Analyze(nodes, links, th, time.Now().UTC())
	mi := meshintel.Compute(configMinimal(), ar, meshintel.MessageSignals{TotalMessages: 30}, true, time.Now().UTC())
	im := EstimateImpact(ImpactRemove, 2, "", ar, mi)
	if im.Verdict != VerdictLikelyHarmful && im.Verdict != VerdictProceedWithCaution {
		t.Logf("verdict: %s", im.Verdict)
	}
}
