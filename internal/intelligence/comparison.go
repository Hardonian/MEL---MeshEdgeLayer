package intelligence

import (
	"time"
)

// CompareStatusAgainstSnapshot for last-known-good diffing
func CompareStatusAgainstSnapshot(currentTime time.Time, lastHealthyTime time.Time, currentSummary map[string]any, baselineSummary map[string]any) StateComparison {
	diff := make(map[string]any)
	regressions := []string{}
	changed := []string{}

	if val, _ := currentSummary["transport_count"].(int); val < baselineSummary["transport_count"].(int) {
		regressions = append(regressions, "transports_missing")
		changed = append(changed, "transport")
		diff["transport_count"] = map[string]any{"current": val, "baseline": baselineSummary["transport_count"]}
	}

	if val, _ := currentSummary["healthy_transport_count"].(int); val < baselineSummary["healthy_transport_count"].(int) {
		regressions = append(regressions, "transport_health_regression")
		changed = append(changed, "transport")
		diff["healthy_transport_count"] = map[string]any{"current": val, "baseline": baselineSummary["healthy_transport_count"]}
	}

	if val, _ := currentSummary["node_count"].(int); val < baselineSummary["node_count"].(int) {
		regressions = append(regressions, "node_population_loss")
		changed = append(changed, "mesh")
		diff["node_count"] = map[string]any{"current": val, "baseline": baselineSummary["node_count"]}
	}

	if val, _ := currentSummary["control_mode"].(string); val != baselineSummary["control_mode"].(string) {
		changed = append(changed, "control")
		diff["control_mode"] = map[string]any{"current": val, "baseline": baselineSummary["control_mode"]}
	}

	if len(regressions) > 0 {
	}

	return StateComparison{
		CurrentTime:       currentTime,
		BaselineTime:      lastHealthyTime,
		BaselineName:      "last-known-good",
		ChangedComponents: changed,
		Regressions:       regressions,
		Diff:              diff,
	}
}
