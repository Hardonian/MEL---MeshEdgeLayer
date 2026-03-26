package planning

import (
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/meshintel"
	"github.com/mel-project/mel/internal/topology"
)

const AdvisoryTransportName = "planning"
const AdvisoryTransportType = "advisory"

// AdvisoryAlert is a resilience-derived advisory for operator attention (not transport failure).
type AdvisoryAlert struct {
	ID                  string
	Severity            string
	Reason              string
	Summary             string
	ClusterKey          string
	ContributingReasons []string
	TriggerCondition    string
}

// DeriveResilienceAdvisoryAlerts produces bounded, evidence-linked advisories from graph analysis.
// Language is advisory: analytical concern, not proof of future partition.
func DeriveResilienceAdvisoryAlerts(ar topology.AnalysisResult, mi meshintel.Assessment, now time.Time) []AdvisoryAlert {
	var out []AdvisoryAlert
	sum, profiles := ComputeResilience(ar, mi)
	gh := strings.TrimSpace(ar.Snapshot.GraphHash)
	aid := strings.TrimSpace(mi.AssessmentID)
	evidence := []string{}
	if gh != "" {
		evidence = append(evidence, "graph_hash="+gh)
	}
	if aid != "" {
		evidence = append(evidence, "mesh_assessment_id="+aid)
	}
	evidenceStr := strings.Join(evidence, " ")
	prefix := "Advisory (topology analysis, not observed partition): "

	// Single bridge / probable SPOF
	for _, bn := range ar.BridgeNodes {
		if bn == 0 {
			continue
		}
		id := fmt.Sprintf("planning|bridge_fragility|%d|%s", bn, shortHash(gh))
		out = append(out, AdvisoryAlert{
			ID:                  id,
			Severity:            "warning",
			Reason:              "bridge_node_fragility",
			Summary:             prefix + fmt.Sprintf("Node %d appears as a bridge between observed components — loss may increase fragmentation.", bn),
			ClusterKey:          fmt.Sprintf("node:%d", bn),
			ContributingReasons: append([]string{"observed_bridge_node"}, evidence...),
			TriggerCondition:    "bridge_node_detected " + evidenceStr,
		})
		break // one alert for strongest bridge signal to limit noise
	}

	// Analytical partition risk proxy from fragmentation + dependency (not observed partition)
	partRiskProxy := clamp01(mi.Topology.FragmentationScore*0.6 + mi.Topology.DependencyConcentrationScore*0.4)
	if partRiskProxy > 0.55 {
		id := "planning|partition_risk_elevated|" + shortHash(gh)
		out = append(out, AdvisoryAlert{
			ID:                  id,
			Severity:            "warning",
			Reason:              "partition_risk_elevated",
			Summary:             prefix + fmt.Sprintf("Fragmentation/dependency signals are elevated (proxy %.2f) from current graph topology — not proof of partition.", partRiskProxy),
			ClusterKey:          "mesh:partition_risk_proxy",
			ContributingReasons: append([]string{"fragmentation_and_dependency_scores"}, evidence...),
			TriggerCondition:    fmt.Sprintf("partition_risk_proxy=%.2f frag=%.2f dep=%.2f %s", partRiskProxy, mi.Topology.FragmentationScore, mi.Topology.DependencyConcentrationScore, evidenceStr),
		})
	}

	if sum.ResilienceScore < 0.35 && sum.ResilienceScore > 0 {
		id := "planning|resilience_degraded|" + shortHash(gh)
		out = append(out, AdvisoryAlert{
			ID:                  id,
			Severity:            "info",
			Reason:              "resilience_degraded",
			Summary:             prefix + fmt.Sprintf("Aggregate resilience score from topology signals is low (%.2f); redundancy may be limited.", sum.ResilienceScore),
			ClusterKey:          "mesh:resilience",
			ContributingReasons: append([]string{"planning_resilience_summary_low"}, evidence...),
			TriggerCondition:    fmt.Sprintf("resilience_score=%.2f %s", sum.ResilienceScore, evidenceStr),
		})
	}

	for _, p := range profiles {
		if p.SPOFClass == SPOFProbable && p.RecoveryPriority <= 3 {
			id := fmt.Sprintf("planning|probable_spof|%d|%s", p.NodeNum, shortHash(gh))
			out = append(out, AdvisoryAlert{
				ID:                  id,
				Severity:            "warning",
				Reason:              "probable_single_point_of_failure",
				Summary:             prefix + fmt.Sprintf("Node %s (%d) scores as a probable single-point-of-failure bridge in the observed graph.", p.ShortName, p.NodeNum),
				ClusterKey:          fmt.Sprintf("node:%d", p.NodeNum),
				ContributingReasons: append([]string{"spof_class_probable"}, evidence...),
				TriggerCondition:    fmt.Sprintf("spof_class=probable node=%d %s", p.NodeNum, evidenceStr),
			})
			break
		}
	}

	_ = now.UTC()
	return dedupeAdvisories(out)
}

func shortHash(h string) string {
	h = strings.TrimSpace(h)
	if len(h) > 12 {
		return h[:12]
	}
	if h == "" {
		return "nogh"
	}
	return h
}

// SyncAdvisoryAlertsToStore upserts advisory rows and resolves stale ones (dedupe by id).
func SyncAdvisoryAlertsToStore(d *db.DB, alerts []AdvisoryAlert) error {
	if d == nil {
		return nil
	}
	var ids []string
	now := time.Now().UTC().Format(time.RFC3339)
	for _, a := range alerts {
		if strings.TrimSpace(a.ID) == "" {
			continue
		}
		ids = append(ids, a.ID)
		if err := d.UpsertPlanningAdvisoryAlert(a.ID, a.Severity, a.Reason, a.Summary, a.ClusterKey, a.ContributingReasons, a.TriggerCondition); err != nil {
			return err
		}
	}
	return d.ResolvePlanningAdvisoryAlertsNotIn(ids, now)
}

func dedupeAdvisories(in []AdvisoryAlert) []AdvisoryAlert {
	seen := map[string]struct{}{}
	var out []AdvisoryAlert
	for _, a := range in {
		if a.ID == "" {
			continue
		}
		if _, ok := seen[a.ID]; ok {
			continue
		}
		seen[a.ID] = struct{}{}
		out = append(out, a)
	}
	return out
}
