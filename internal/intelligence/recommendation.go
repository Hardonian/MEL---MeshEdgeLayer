package intelligence

import (
	"fmt"
	"strings"

	"github.com/mel-project/mel/internal/models"
)

const (
	RiskLow    = "low"
	RiskMedium = "medium"
	RiskHigh   = "high"

	RevHigh   = "high"
	RevMedium = "medium"
	RevLow    = "low"
	RevNone   = "none"
)

// RecommendActions for a set of prioritized problems
func RecommendActions(priorities []models.PriorityItem) []Recommendation {
	var out []Recommendation
	seen := make(map[string]bool)

	for _, p := range priorities {
		recs := RecommendationsFor(p)
		for _, r := range recs {
			if !seen[r.Code] {
				out = append(out, r)
				seen[r.Code] = true
			}
		}
	}

	return out
}

// RecommendationsFor a single prioritized problem
func RecommendationsFor(p models.PriorityItem) []Recommendation {
	switch p.Category {
	case "transport":
		return recommendationsForTransport(p)
	case "database":
		return recommendationsForDatabase(p)
	case "config":
		return recommendationsForConfig(p)
	case "security":
		return recommendationsForSecurity(p)
	case "control":
		return recommendationsForControl(p)
	default:
		return []Recommendation{
			{
				Code:              "inspect_generic",
				Action:            "Inspect " + p.Title,
				Rationale:         "Generic problem requires manual inspection",
				EvidenceReference: []string{"incident:" + p.ID},
				Confidence:        0.5,
				RiskLevel:         RiskLow,
				Reversibility:     RevHigh,
			},
		}
	}
}

func recommendationsForTransport(p models.PriorityItem) []Recommendation {
	transportName, _ := p.Metadata["resource_id"].(string)
	if transportName == "" {
		transportName, _ = p.Metadata["affected_transport"].(string)
	}

	recs := []Recommendation{
		{
			Code:              "inspect_transport_" + transportName,
			Action:            fmt.Sprintf("Inspect transport %s status", transportName),
			Rationale:         fmt.Sprintf("Transport %s has reported issues or anomalies recently", transportName),
			EvidenceReference: []string{"incident:" + p.ID},
			Confidence:        0.9,
			RiskLevel:         RiskLow,
			Reversibility:     RevHigh,
		},
	}

	if p.Severity == "critical" || p.Severity == "error" {
		recs = append(recs, Recommendation{
			Code:              "reconnect_transport_" + transportName,
			Action:            fmt.Sprintf("Force reconnect transport %s", transportName),
			Rationale:         "Transport is failed or stuck in retrying state; forcing reconnect might clear transient issues",
			EvidenceReference: []string{"state:failed", "incident:" + p.ID},
			Confidence:        0.7,
			RiskLevel:         RiskMedium,
			Reversibility:     RevMedium,
			CanAutomate:       true,
		})
	}

	// Restraint rationale
	if strings.Contains(p.Summary, "heartbeat stale") {
		recs = append(recs, Recommendation{
			Code:              "wait_for_evidence_" + transportName,
			Action:            "Wait for fresh heartbeat before acting",
			Rationale:         "Heartbeat is stale but transport might be silently healthy; acting too early could cause churn",
			EvidenceReference: []string{"freshness:stale"},
			Confidence:        0.8,
			RiskLevel:         RiskLow,
			Reversibility:     RevHigh,
		})
	}

	return recs
}

func recommendationsForDatabase(p models.PriorityItem) []Recommendation {
	return []Recommendation{
		{
			Code:              "verify_database_permissions",
			Action:            "Verify database file permissions",
			Rationale:         "Database unreachable or write-failing often points to permission regressions",
			EvidenceReference: []string{"incident:" + p.ID},
			Confidence:        0.85,
			RiskLevel:         RiskLow,
			Reversibility:     RevHigh,
		},
		{
			Code:              "run_database_doctor",
			Action:            "Run 'mel doctor' on database",
			Rationale:         "Comprehensive database check required to identify corruption or schema drift",
			EvidenceReference: []string{"incident:" + p.ID},
			Confidence:        0.95,
			RiskLevel:         RiskLow,
			Reversibility:     RevHigh,
		},
	}
}

func recommendationsForConfig(p models.PriorityItem) []Recommendation {
	return []Recommendation{
		{
			Code:              "audit_config_safeguards",
			Action:            "Audit control safeguards",
			Rationale:         "Unsafe configuration detected; safeguards should be prioritized to prevent accidental churn",
			EvidenceReference: []string{"incident:" + p.ID},
			Confidence:        0.9,
			RiskLevel:         RiskLow,
			Reversibility:     RevHigh,
		},
	}
}

func recommendationsForSecurity(p models.PriorityItem) []Recommendation {
	return []Recommendation{
		{
			Code:              "rotate_credentials",
			Action:            "Rotate transport credentials",
			Rationale:         "Potential security regression or unauthorized access attempt detected",
			EvidenceReference: []string{"incident:" + p.ID},
			Confidence:        0.6,
			RiskLevel:         RiskHigh,
			Reversibility:     RevLow,
			CanAutomate:       false,
		},
	}
}

func recommendationsForControl(p models.PriorityItem) []Recommendation {
	return []Recommendation{
		{
			Code:              "acknowledge_control_noise",
			Action:            "Acknowledge and suppress duplicate control noise",
			Rationale:         "High volume of similar control actions detected; manual suppression required to clear the queue",
			EvidenceReference: []string{"incident:" + p.ID},
			Confidence:        0.8,
			RiskLevel:         RiskLow,
			Reversibility:     RevHigh,
		},
	}
}
