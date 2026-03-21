package intelligence

import (
	"sort"
	"time"

	"github.com/mel-project/mel/internal/diagnostics"
	"github.com/mel-project/mel/internal/models"
)

const (
	CategoryTransport = "transport"
	CategorySystem    = "system"
	CategorySecurity  = "security"
	CategoryControl   = "control"
	CategoryData      = "data"
)

// RankOperationalIssues takes all operational problems and ranks them by priority
func RankOperationalIssues(incidents []models.Incident, findings []diagnostics.Finding, now time.Time) []PriorityItem {
	var items []PriorityItem

	// Translate Incidents to PriorityItems
	for _, incident := range incidents {
		if incident.State == "resolved" || incident.State == "suppressed" {
			continue
		}
		items = append(items, priorityFromIncident(incident, now))
	}

	// Translate Diagnostic findings to PriorityItems
	for _, finding := range findings {
		items = append(items, priorityFromFinding(finding, now))
	}

	// Dedupe and Rank
	items = dedupePriorityItems(items)
	sortItems(items)

	return items
}

func priorityFromIncident(inc models.Incident, now time.Time) PriorityItem {
	rank := scoreSeverity(inc.Severity)
	rank *= scoreCategory(inc.Category)

	confidence := 1.0 // Incidents are usually high confidence
	if val, ok := inc.Metadata["confidence"].(float64); ok {
		confidence = val
	}
	rank *= confidence

	// Freshness
	occAt, _ := time.Parse(time.RFC3339, inc.OccurredAt)
	freshness := "High"
	if now.Sub(occAt) > 1*time.Hour {
		freshness = "Medium"
	}
	if now.Sub(occAt) > 24*time.Hour {
		freshness = "Low"
	}

	return PriorityItem{
		ID:                inc.ID,
		Category:          inc.Category,
		Severity:          inc.Severity,
		Title:             inc.Title,
		Summary:           inc.Summary,
		Rank:              rank,
		Confidence:        confidence,
		EvidenceFreshness: freshness,
		IsActionable:      true,
		BlocksRecovery:    inc.Severity == "critical",
		GeneratedAt:       now,
		Metadata:          inc.Metadata,
	}
}

func priorityFromFinding(f diagnostics.Finding, now time.Time) PriorityItem {
	rank := scoreSeverity(f.Severity)
	rank *= scoreCategory(f.Component)

	confidence := 0.8 // Findings are inferences
	if f.Severity == "critical" {
		confidence = 0.95
	}
	rank *= confidence

	// Freshness
	genAt, _ := time.Parse(time.RFC3339, f.GeneratedAt)
	freshness := "High"
	if now.Sub(genAt) > 5*time.Minute {
		freshness = "Medium"
	}

	return PriorityItem{
		ID:                f.Code + ":" + f.Component + ":" + f.AffectedTransport,
		Category:          f.Component,
		Severity:          f.Severity,
		Title:             f.Title,
		Summary:           f.Explanation,
		Rank:              rank,
		Confidence:        confidence,
		EvidenceFreshness: freshness,
		IsActionable:      true,
		BlocksRecovery:    f.Severity == "critical" && (f.Component == "database" || f.Component == "config"),
		GeneratedAt:       now,
		Metadata:          f.Evidence,
	}
}

func scoreSeverity(s string) float64 {
	switch s {
	case "critical":
		return 100
	case "high":
		return 80
	case "medium", "warning", "warn":
		return 50
	case "low", "info":
		return 10
	default:
		return 5
	}
}

func scoreCategory(c string) float64 {
	switch c {
	case "security":
		return 1.5
	case "system", "database", "config":
		return 1.4
	case "transport":
		return 1.2
	case "control":
		return 1.1
	case "data", "retention":
		return 1.0
	default:
		return 1.0
	}
}

func dedupePriorityItems(items []PriorityItem) []PriorityItem {
	seen := make(map[string]int)
	var out []PriorityItem
	for _, item := range items {
		if i, ok := seen[item.ID]; ok {
			// If already seen, take the one with higher rank
			if item.Rank > out[i].Rank {
				out[i] = item
			}
			continue
		}
		seen[item.ID] = len(out)
		out = append(out, item)
	}
	return out
}

func sortItems(items []PriorityItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Rank == items[j].Rank {
			return items[i].GeneratedAt.After(items[j].GeneratedAt)
		}
		return items[i].Rank > items[j].Rank
	})
}
