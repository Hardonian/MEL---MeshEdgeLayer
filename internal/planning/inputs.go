package planning

import (
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/meshintel"
)

// Known assumption keys (subset — extend as estimators consume more).
const (
	KeyPlacementClass   = "placement_class"
	KeyElevationClass   = "elevation_class"
	KeyUptimeClass      = "uptime_class"
	KeyMobilityClass    = "mobility_class"
	KeyIndoorOutdoor    = "indoor_outdoor_hint"
	KeyObjective        = "operator_objective"
	KeyTerrainNote      = "terrain_note"
	KeyObstructionNote  = "obstruction_note"
	KeyAlwaysOnIntent   = "intended_always_on"
	KeyTempVsPermanent  = "temporary_vs_permanent"
	KeyNeighborhoodGoal = "neighborhood_objective"
)

// BuildInputVersionPayload constructs a versioned payload from items; classifies evidence and detects gaps.
func BuildInputVersionPayload(inputSetID string, versionNum int, items []AssumptionItem, arGraphHash, assessmentID string) PlanningInputVersionPayload {
	now := time.Now().UTC().Format(time.RFC3339)
	p := PlanningInputVersionPayload{
		VersionNum: versionNum,
		InputSetID: inputSetID,
		CreatedAt:  now,
	}
	if strings.TrimSpace(arGraphHash) != "" {
		p.ObservedAnchors = append(p.ObservedAnchors, EvidenceReference{Kind: "graph_hash", Ref: arGraphHash})
	}
	if strings.TrimSpace(assessmentID) != "" {
		p.ObservedAnchors = append(p.ObservedAnchors, EvidenceReference{Kind: "mesh_assessment", Ref: assessmentID})
	}

	hasOp := false
	for _, it := range items {
		if it.Source == AssumptionSourceOperator && strings.TrimSpace(it.Value) != "" {
			hasOp = true
		}
		it2 := it
		if !assumptionKeyModeled(it.Key) {
			it2.UsedByModel = false
			if it2.Description == "" {
				it2.Description = "Captured for traceability; not yet consumed by planning estimators."
			}
		} else {
			it2.UsedByModel = true
		}
		p.Assumptions = append(p.Assumptions, it2)
	}

	if hasOp {
		p.EvidenceModel = EvidenceTopologyAssumptionAugmented
	} else {
		p.EvidenceModel = EvidenceTopologyOnly
	}

	p.MissingInputs = defaultMissingInputs()
	p.Conflicts = detectConflicts(items)
	p.ValidationTargets = defaultValidationTargets()
	return p
}

func assumptionKeyModeled(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	switch k {
	case KeyPlacementClass, KeyUptimeClass, KeyObjective, KeyAlwaysOnIntent, KeyTempVsPermanent:
		return true
	default:
		return false
	}
}

func defaultMissingInputs() []MissingInputNotice {
	return []MissingInputNotice{
		{Key: "rf_propagation", Impact: "weakens_confidence", Description: "No terrain/RF model — coverage claims are not available."},
		{Key: "stable_observation_window", Impact: "weakens_confidence", Description: "Short or intermittent observation weakens trend claims."},
	}
}

func defaultValidationTargets() []ValidationTarget {
	return []ValidationTarget{
		{Label: "fragmentation_delta", MetricHint: "meshintel.Topology.FragmentationScore", ObserveHours: 24},
		{Label: "bootstrap_viability", MetricHint: "meshintel.Bootstrap.Viability", ObserveHours: 48},
	}
}

func detectConflicts(items []AssumptionItem) []InputConflictNotice {
	var out []InputConflictNotice
	temp := ""
	always := ""
	for _, it := range items {
		if strings.EqualFold(it.Key, KeyTempVsPermanent) {
			temp = strings.ToLower(strings.TrimSpace(it.Value))
		}
		if strings.EqualFold(it.Key, KeyAlwaysOnIntent) {
			always = strings.ToLower(strings.TrimSpace(it.Value))
		}
	}
	if temp == "permanent" && strings.Contains(always, "intermittent") {
		out = append(out, InputConflictNotice{
			Keys:        []string{KeyTempVsPermanent, KeyAlwaysOnIntent},
			Description: "Permanent deployment intent conflicts with intermittent always-on expectation — clarify duty cycle.",
		})
	}
	return out
}

// AssumptionFragilityScore returns 0–1 higher = more sensitive to unknowns.
func AssumptionFragilityScore(items []AssumptionItem, mi meshintel.Assessment) float64 {
	base := 0.35
	if mi.Bootstrap.Confidence == meshintel.ConfidenceLow {
		base += 0.25
	}
	for _, it := range items {
		if it.Sensitivity == SensitivityHigh {
			base += 0.08
		}
		if !it.UsedByModel && strings.TrimSpace(it.Value) != "" {
			base += 0.02
		}
	}
	if base > 1 {
		base = 1
	}
	return base
}

// SynthesizeAssumptionNotices for API responses.
func SynthesizeAssumptionNotices(items []AssumptionItem) []string {
	var lines []string
	for _, it := range items {
		if strings.TrimSpace(it.Key) == "" {
			continue
		}
		tag := string(it.Source)
		if !it.UsedByModel {
			lines = append(lines, fmt.Sprintf("[%s] %s=%s (stored; not used by estimators yet)", tag, it.Key, it.Value))
		} else {
			lines = append(lines, fmt.Sprintf("[%s] %s=%s", tag, it.Key, it.Value))
		}
	}
	return lines
}
