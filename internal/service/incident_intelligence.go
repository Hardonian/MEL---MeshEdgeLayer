package service

import (
	"crypto/sha1"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/models"
)

func (a *App) buildIncidentIntelligence(inc models.Incident) *models.IncidentIntelligence {
	if a == nil || a.DB == nil {
		return nil
	}
	out := &models.IncidentIntelligence{
		EvidenceStrength: "sparse",
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	reasonKey := incidentReasonKey(inc)
	sigKey := signatureKey(inc, reasonKey)
	sigLabel := signatureLabel(inc, reasonKey)
	out.SignatureKey = sigKey
	out.SignatureLabel = sigLabel
	if err := a.DB.UpsertIncidentSignature(db.IncidentSignatureRecord{
		SignatureKey:      sigKey,
		SignatureLabel:    sigLabel,
		Category:          strings.TrimSpace(inc.Category),
		ResourceType:      strings.TrimSpace(inc.ResourceType),
		ReasonKey:         reasonKey,
		ExampleIncidentID: inc.ID,
		LastSummary:       inc.Summary,
	}); err != nil {
		out.Degraded = true
		out.DegradedReasons = append(out.DegradedReasons, "signature_persistence_failed")
	}
	if err := a.DB.LinkIncidentToSignature(sigKey, inc.ID); err != nil {
		out.Degraded = true
		out.DegradedReasons = append(out.DegradedReasons, "signature_link_persistence_failed")
	}
	if sig, ok, err := a.DB.SignatureByKey(sigKey); err == nil && ok {
		out.SignatureMatchCount = sig.MatchCount
	} else if err != nil {
		out.Degraded = true
		out.DegradedReasons = append(out.DegradedReasons, "signature_lookup_failed")
	}

	out.EvidenceItems = append(out.EvidenceItems, models.IncidentEvidenceItem{
		Kind:         "incident_record",
		ReferenceID:  "incident:" + inc.ID,
		Summary:      fmt.Sprintf("Stored incident category=%s severity=%s resource=%s/%s state=%s.", inc.Category, inc.Severity, inc.ResourceType, inc.ResourceID, inc.State),
		ObservedAt:   firstNonEmpty(inc.OccurredAt, inc.UpdatedAt),
		SupportsOnly: "association",
	})

	if inc.ResourceType == "transport" && strings.TrimSpace(inc.ResourceID) != "" {
		from, to := incidentEvidenceWindow(inc)
		dlRows, err := a.DB.DeadLettersForTransportBetween(inc.ResourceID, from, to, 20)
		if err != nil {
			out.Degraded = true
			out.DegradedReasons = append(out.DegradedReasons, "dead_letter_lookup_failed")
		} else if len(dlRows) > 0 {
			reasonCounts := map[string]int{}
			for _, row := range dlRows {
				reason := strings.TrimSpace(fmt.Sprint(row["reason"]))
				if reason != "" {
					reasonCounts[reason]++
				}
			}
			for _, p := range topCountPairs(reasonCounts, 2) {
				out.EvidenceItems = append(out.EvidenceItems, models.IncidentEvidenceItem{
					Kind:         "dead_letter_reason_cluster",
					Summary:      fmt.Sprintf("Dead-letter reason %q occurred %d times near incident window on transport %s.", p.Key, p.Count, inc.ResourceID),
					SupportsOnly: "association",
				})
			}
		}
		alerts, err := a.DB.TransportAlertsForWindow(inc.ResourceID, from, to, 20)
		if err != nil {
			out.Degraded = true
			out.DegradedReasons = append(out.DegradedReasons, "transport_alert_lookup_failed")
		} else {
			for _, al := range alerts {
				out.EvidenceItems = append(out.EvidenceItems, models.IncidentEvidenceItem{
					Kind:         "transport_alert",
					ReferenceID:  "transport_alert:" + al.ID,
					Summary:      fmt.Sprintf("Transport alert reason=%s severity=%s active=%t.", al.Reason, al.Severity, al.Active),
					ObservedAt:   al.LastUpdatedAt,
					SupportsOnly: "chronology",
				})
			}
		}
	}

	if len(inc.LinkedControlActions) > 0 {
		failures := 0
		typeCounts := map[string]int{}
		for _, ca := range inc.LinkedControlActions {
			typeCounts[ca.ActionType]++
			if strings.EqualFold(ca.Result, "failed") || strings.EqualFold(ca.LifecycleState, "failed") {
				failures++
			}
		}
		if failures > 0 {
			out.EvidenceItems = append(out.EvidenceItems, models.IncidentEvidenceItem{
				Kind:         "linked_action_failures",
				Summary:      fmt.Sprintf("%d linked control actions are in failed state/result.", failures),
				SupportsOnly: "association",
			})
		}
		for _, p := range topCountPairs(typeCounts, 3) {
			out.HistoricallyUsedActions = append(out.HistoricallyUsedActions, models.IncidentActionPattern{ActionType: p.Key, Count: p.Count})
		}
	}

	out.SimilarIncidents = a.similarIncidents(sigKey, inc.ID)
	out.ImplicatedDomains = deriveDomains(out.EvidenceItems, inc)
	out.InvestigateNext = deriveInvestigationGuidance(out, inc)
	switch {
	case len(out.EvidenceItems) >= 5:
		out.EvidenceStrength = "strong"
	case len(out.EvidenceItems) >= 3:
		out.EvidenceStrength = "moderate"
	default:
		out.EvidenceStrength = "sparse"
	}
	if len(out.SimilarIncidents) == 0 {
		out.Degraded = true
		out.DegradedReasons = append(out.DegradedReasons, "no_similar_incident_history")
	}
	if len(out.EvidenceItems) < 2 {
		out.Degraded = true
		out.DegradedReasons = append(out.DegradedReasons, "limited_correlated_evidence")
	}
	return out
}

func (a *App) similarIncidents(signatureKey, currentID string) []models.IncidentSimilarityRecord {
	if a == nil || a.DB == nil {
		return nil
	}
	incs, err := a.DB.SimilarIncidentsBySignature(signatureKey, currentID, 3)
	if err != nil {
		return nil
	}
	out := make([]models.IncidentSimilarityRecord, 0, len(incs))
	for _, inc := range incs {
		out = append(out, models.IncidentSimilarityRecord{
			IncidentID:       inc.ID,
			Title:            inc.Title,
			State:            inc.State,
			OccurredAt:       inc.OccurredAt,
			SimilarityReason: []string{"same_deterministic_signature_key"},
		})
	}
	return out
}

func incidentReasonKey(inc models.Incident) string {
	if inc.Metadata != nil {
		for _, k := range []string{"reason", "primary_reason", "trigger_reason"} {
			if v, ok := inc.Metadata[k]; ok {
				s := strings.TrimSpace(fmt.Sprint(v))
				if s != "" {
					return strings.ToLower(s)
				}
			}
		}
	}
	return "unspecified"
}

func signatureKey(inc models.Incident, reason string) string {
	base := strings.ToLower(strings.Join([]string{
		strings.TrimSpace(inc.Category),
		strings.TrimSpace(inc.ResourceType),
		strings.TrimSpace(inc.ResourceID),
		reason,
	}, "|"))
	sum := sha1.Sum([]byte(base))
	return fmt.Sprintf("sig-%x", sum[:6])
}

func signatureLabel(inc models.Incident, reason string) string {
	part := reason
	if part == "unspecified" {
		part = "no explicit reason"
	}
	return strings.TrimSpace(fmt.Sprintf("%s/%s pattern (%s)", firstNonEmpty(inc.Category, "incident"), firstNonEmpty(inc.ResourceType, "resource"), part))
}

func incidentEvidenceWindow(inc models.Incident) (string, string) {
	base := parseRFC3339Fallback(firstNonEmpty(inc.OccurredAt, inc.UpdatedAt), time.Now().UTC())
	return base.Add(-2 * time.Hour).Format(time.RFC3339), base.Add(4 * time.Hour).Format(time.RFC3339)
}

func parseRFC3339Fallback(v string, fallback time.Time) time.Time {
	ts, err := time.Parse(time.RFC3339, strings.TrimSpace(v))
	if err != nil {
		return fallback
	}
	return ts.UTC()
}

type countPair struct {
	Key   string
	Count int
}

func topCountPairs(in map[string]int, limit int) []countPair {
	if len(in) == 0 || limit <= 0 {
		return nil
	}
	all := make([]countPair, 0, len(in))
	for k, c := range in {
		if strings.TrimSpace(k) == "" {
			continue
		}
		all = append(all, countPair{Key: k, Count: c})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Count == all[j].Count {
			return all[i].Key < all[j].Key
		}
		return all[i].Count > all[j].Count
	})
	if len(all) > limit {
		return all[:limit]
	}
	return all
}

func deriveDomains(items []models.IncidentEvidenceItem, inc models.Incident) []models.IncidentDomainHint {
	domainRefs := map[string][]string{}
	for _, it := range items {
		switch it.Kind {
		case "transport_alert", "dead_letter_reason_cluster":
			domainRefs["transport"] = append(domainRefs["transport"], it.Kind)
		case "linked_action_failures":
			domainRefs["control"] = append(domainRefs["control"], it.Kind)
		}
	}
	if inc.ResourceType == "mesh" || inc.Category == "mesh_topology" {
		domainRefs["topology"] = append(domainRefs["topology"], "incident_record")
	}
	keys := make([]string, 0, len(domainRefs))
	for k := range domainRefs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]models.IncidentDomainHint, 0, len(keys))
	for _, k := range keys {
		out = append(out, models.IncidentDomainHint{
			Domain:       k,
			EvidenceRefs: domainRefs[k],
			Note:         "Association only; inspect timeline and raw records before attributing cause.",
		})
	}
	return out
}

func deriveInvestigationGuidance(intel *models.IncidentIntelligence, inc models.Incident) []models.IncidentGuidanceItem {
	if intel == nil {
		return nil
	}
	out := []models.IncidentGuidanceItem{
		{
			ID:           "guide-incident-record",
			Title:        "Review incident chronology and linked evidence",
			Rationale:    "Confirm sequencing across incident row, timeline events, and linked control actions before deciding next actions.",
			EvidenceRefs: []string{"incident:" + inc.ID},
			Confidence:   "medium",
		},
	}
	for _, d := range intel.ImplicatedDomains {
		out = append(out, models.IncidentGuidanceItem{
			ID:           "guide-domain-" + d.Domain,
			Title:        "Inspect " + d.Domain + " evidence surfaces",
			Rationale:    "Domain hint is based on recurring associated evidence, not root-cause proof.",
			EvidenceRefs: d.EvidenceRefs,
			Confidence:   "low",
		})
	}
	if len(intel.SimilarIncidents) > 0 {
		out = append(out, models.IncidentGuidanceItem{
			ID:           "guide-similar-incidents",
			Title:        "Compare with similar prior incidents",
			Rationale:    "Shared signature indicates historical resemblance by deterministic fields; compare outcomes before reuse.",
			EvidenceRefs: []string{intel.SimilarIncidents[0].IncidentID},
			Confidence:   "low",
		})
	}
	return out
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
