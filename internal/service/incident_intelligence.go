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
	var outcomeDegradedReasons []string
	out.ActionOutcomeMemory, out.ActionOutcomeSnapshots, out.ActionOutcomeTrace, outcomeDegradedReasons = a.actionOutcomeMemory(sigKey, inc)
	out.DegradedReasons = append(out.DegradedReasons, outcomeDegradedReasons...)
	if len(outcomeDegradedReasons) > 0 {
		out.Degraded = true
	}
	out.ImplicatedDomains = deriveDomains(out.EvidenceItems, inc)
	out.WirelessContext = deriveWirelessContext(inc, out)
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
	if len(out.ActionOutcomeMemory) == 0 {
		out.Degraded = true
		out.DegradedReasons = append(out.DegradedReasons, "insufficient_action_outcome_history")
	}
	if len(out.EvidenceItems) < 2 {
		out.Degraded = true
		out.DegradedReasons = append(out.DegradedReasons, "limited_correlated_evidence")
	}
	return out
}

func (a *App) actionOutcomeMemory(signatureKey string, current models.Incident) ([]models.IncidentActionOutcomeMemory, []models.IncidentActionOutcomeSnapshot, *models.IncidentActionOutcomeTrace, []string) {
	trace := &models.IncidentActionOutcomeTrace{
		SnapshotRetrievalStatus: "unavailable",
		Completeness:            "unavailable",
	}
	if a == nil || a.DB == nil || strings.TrimSpace(signatureKey) == "" {
		return nil, nil, trace, nil
	}
	similar, err := a.DB.SimilarIncidentsBySignature(signatureKey, current.ID, 8)
	if err != nil || len(similar) == 0 {
		return nil, nil, trace, nil
	}
	ids := make([]string, 0, len(similar))
	simByID := map[string]models.Incident{}
	for _, inc := range similar {
		ids = append(ids, inc.ID)
		simByID[inc.ID] = inc
	}
	actions, err := a.DB.ControlActionsForIncidentIDs(ids, 400)
	if err != nil || len(actions) == 0 {
		return nil, nil, trace, nil
	}
	trace.ExpectedSnapshotWrites = len(actions)
	degradedReasons := []string{}
	type score struct {
		model      models.IncidentActionOutcomeMemory
		observed   int
		refsSeen   map[string]struct{}
		caveats    map[string]struct{}
		preInspect map[string]struct{}
		snapshots  []string
	}
	perAction := map[string]*score{}
	allSnapshots := make([]models.IncidentActionOutcomeSnapshot, 0, len(actions))
	for _, action := range actions {
		actionType := strings.TrimSpace(action.ActionType)
		incidentID := strings.TrimSpace(action.IncidentID)
		if actionType == "" || incidentID == "" {
			continue
		}
		simInc, ok := simByID[incidentID]
		if !ok {
			continue
		}
		eval := evaluatePostActionSignals(a, action, simInc)
		if err := a.DB.UpsertIncidentActionOutcomeSnapshot(db.IncidentActionOutcomeSnapshotRecord{
			SignatureKey:          signatureKey,
			IncidentID:            incidentID,
			ActionID:              action.ID,
			ActionType:            actionType,
			ActionLabel:           strings.ReplaceAll(actionType, "_", " "),
			DerivedClassification: normalizeSnapshotClassification(eval.outcome),
			EvidenceSufficiency:   eval.evidenceSufficiency,
			PreActionSummary: map[string]any{
				"transport_name":         eval.transportName,
				"dead_letters_count":     eval.preDeadLetters,
				"transport_alerts_count": eval.preAlerts,
				"incident_state":         eval.incidentState,
				"action_result":          eval.actionResult,
				"action_lifecycle":       eval.actionLifecycle,
			},
			PostActionSummary: map[string]any{
				"transport_name":         eval.transportName,
				"dead_letters_count":     eval.postDeadLetters,
				"transport_alerts_count": eval.postAlerts,
				"incident_state":         eval.incidentState,
				"action_result":          eval.actionResult,
				"action_lifecycle":       eval.actionLifecycle,
			},
			ObservedSignalCount: eval.observedSignals,
			Caveats:             eval.caveats,
			InspectBeforeReuse:  eval.inspectBeforeReuse,
			EvidenceRefs:        eval.evidenceRefs,
			AssociationOnly:     true,
			WindowStart:         eval.windowStart,
			WindowEnd:           eval.windowEnd,
			DerivedAt:           time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			trace.SnapshotWriteFailures++
			trace.SnapshotWriteFailureIDs = append(trace.SnapshotWriteFailureIDs, action.ID)
		}
	}
	persistedSnapshots, err := a.DB.ActionOutcomeSnapshotsBySignature(signatureKey, current.ID, 400)
	if err != nil {
		trace.SnapshotRetrievalStatus = "error"
		trace.SnapshotRetrievalReason = "snapshot_query_failed"
		trace.Completeness = "partial"
		degradedReasons = append(degradedReasons, "action_outcome_snapshot_retrieval_failed")
	} else {
		trace.SnapshotRetrievalStatus = "available"
		trace.SnapshotRetrievalReason = "historical_snapshots_loaded"
	}
	if err == nil {
		for _, snap := range persistedSnapshots {
			actionType := strings.TrimSpace(snap.ActionType)
			if actionType == "" {
				continue
			}
			entry, ok := perAction[actionType]
			if !ok {
				entry = &score{
					model: models.IncidentActionOutcomeMemory{
						ActionType:  actionType,
						ActionLabel: strings.ReplaceAll(actionType, "_", " "),
					},
					refsSeen:   map[string]struct{}{},
					caveats:    map[string]struct{}{},
					preInspect: map[string]struct{}{},
				}
				perAction[actionType] = entry
			}
			entry.model.OccurrenceCount++
			entry.model.SampleSize++
			entry.observed += snap.ObservedSignalCount
			switch snap.DerivedClassification {
			case "improvement_observed":
				entry.model.ImprovementObservedCount++
			case "deterioration_observed":
				entry.model.DeteriorationObservedCount++
			default:
				entry.model.InconclusiveCount++
			}
			for _, caveat := range snap.Caveats {
				entry.caveats[caveat] = struct{}{}
			}
			for _, inspect := range snap.InspectBeforeReuse {
				entry.preInspect[inspect] = struct{}{}
			}
			for _, ref := range snap.EvidenceRefs {
				entry.refsSeen[ref] = struct{}{}
			}
			entry.snapshots = append(entry.snapshots, snap.SnapshotID)
			allSnapshots = append(allSnapshots, modelSnapshotFromDB(snap))
		}
		trace.PersistedSnapshotCount = len(persistedSnapshots)
	}
	if trace.SnapshotWriteFailures > 0 {
		degradedReasons = append(degradedReasons, "action_outcome_snapshot_write_partial_failure")
	}
	switch {
	case trace.SnapshotRetrievalStatus == "error":
		trace.Completeness = "partial"
	case trace.PersistedSnapshotCount == 0:
		trace.SnapshotRetrievalReason = "no_historical_snapshots"
		trace.Completeness = "unavailable"
	case trace.SnapshotWriteFailures > 0:
		trace.Completeness = "partial"
	default:
		trace.Completeness = "complete"
	}
	if len(perAction) == 0 {
		return nil, nil, trace, degradedReasons
	}
	out := make([]models.IncidentActionOutcomeMemory, 0, len(perAction))
	for _, mem := range perAction {
		mem.model.Caveats = setToSortedSlice(mem.caveats)
		mem.model.InspectBeforeReuse = setToSortedSlice(mem.preInspect)
		mem.model.EvidenceRefs = setToSortedSlice(mem.refsSeen)
		sort.Strings(mem.snapshots)
		mem.model.SnapshotRefs = mem.snapshots
		mem.model.SnapshotCoveragePercent = snapshotCoveragePercent(mem.model.OccurrenceCount, len(mem.model.SnapshotRefs))
		mem.model.SnapshotTraceStatus, mem.model.SnapshotCoveragePosture = snapshotTracePosture(mem.model.OccurrenceCount, len(mem.model.SnapshotRefs))
		mem.model.OutcomeFraming, mem.model.ObservedPostActionStatus = classifyOutcome(mem.model)
		mem.model.EvidenceStrength = evidenceStrengthForSample(mem.model.SampleSize, mem.observed)
		if mem.model.SampleSize < 2 {
			mem.model.Caveats = append(mem.model.Caveats, "Single historical sample; treat as weak association only.")
		}
		out = append(out, mem.model)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].OccurrenceCount == out[j].OccurrenceCount {
			return out[i].ActionType < out[j].ActionType
		}
		return out[i].OccurrenceCount > out[j].OccurrenceCount
	})
	if len(out) > 5 {
		out = out[:5]
	}
	sort.Slice(allSnapshots, func(i, j int) bool {
		if allSnapshots[i].DerivedAt == allSnapshots[j].DerivedAt {
			return allSnapshots[i].SnapshotID < allSnapshots[j].SnapshotID
		}
		return allSnapshots[i].DerivedAt > allSnapshots[j].DerivedAt
	})
	if len(allSnapshots) > 40 {
		allSnapshots = allSnapshots[:40]
	}
	return out, allSnapshots, trace, degradedReasons
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
	if intel.WirelessContext != nil {
		out = append(out, models.IncidentGuidanceItem{
			ID:           "guide-wireless-context",
			Title:        "Inspect mixed wireless context boundaries",
			Rationale:    intel.WirelessContext.Summary,
			EvidenceRefs: wirelessRefs(intel.WirelessContext.Reasons),
			Confidence:   "low",
		})
	}
	return out
}

func deriveWirelessContext(inc models.Incident, intel *models.IncidentIntelligence) *models.IncidentWirelessContext {
	if intel == nil {
		return nil
	}
	tokens := strings.ToLower(strings.Join([]string{
		inc.Category,
		inc.ResourceType,
		inc.ResourceID,
		inc.Title,
		inc.Summary,
		collectMetadataText(inc.Metadata),
	}, " "))
	for _, it := range intel.EvidenceItems {
		tokens += " " + strings.ToLower(it.Kind) + " " + strings.ToLower(it.Summary)
	}
	for _, action := range inc.LinkedControlActions {
		tokens += " " + strings.ToLower(action.ActionType) + " " + strings.ToLower(action.TransportName) + " " + strings.ToLower(action.TargetSegment)
	}

	loraHit := containsAny(tokens, []string{"lora", "long range", "frequency", "band", "channel", "snr", "rssi"})
	wifiHit := containsAny(tokens, []string{"wifi", "wi-fi", "backhaul", "ssid", "ap ", "access point", "wlan"})
	bluetoothHit := containsAny(tokens, []string{"bluetooth", "ble", "provision", "pairing", "nearby"})

	observedDomains := make([]string, 0, 3)
	if loraHit {
		observedDomains = append(observedDomains, "lora")
	}
	if bluetoothHit {
		observedDomains = append(observedDomains, "bluetooth")
	}
	if wifiHit {
		observedDomains = append(observedDomains, "wifi")
	}
	if len(observedDomains) == 0 {
		observedDomains = append(observedDomains, "unknown")
	}

	gaps := []string{}
	for _, r := range intel.DegradedReasons {
		if strings.TrimSpace(r) != "" {
			gaps = append(gaps, r)
		}
	}
	if len(intel.EvidenceItems) < 2 {
		gaps = append(gaps, "limited_correlated_evidence")
	}

	reasons := make([]models.IncidentWirelessReason, 0, 4)
	if loraHit {
		reasons = append(reasons, models.IncidentWirelessReason{
			Code:         "lora_terms_present",
			Statement:    "LoRa or frequency-linked terms appear in incident/evidence text; this is association only.",
			EvidenceRefs: []string{"incident:" + inc.ID},
		})
	}
	if wifiHit {
		reasons = append(reasons, models.IncidentWirelessReason{
			Code:         "wifi_backhaul_terms_present",
			Statement:    "Wi-Fi/backhaul terms appear in incident/evidence text; inspect transport continuity and dead letters.",
			EvidenceRefs: []string{"incident:" + inc.ID},
		})
	}
	if bluetoothHit {
		reasons = append(reasons, models.IncidentWirelessReason{
			Code:         "bluetooth_terms_present",
			Statement:    "Bluetooth/nearby onboarding terms are present, but MEL BLE ingest is currently unsupported.",
			EvidenceRefs: []string{"incident:" + inc.ID},
		})
	}
	if len(reasons) == 0 {
		reasons = append(reasons, models.IncidentWirelessReason{
			Code:         "no_domain_specific_terms",
			Statement:    "No clear LoRa/Bluetooth/Wi-Fi terms were found in incident-linked evidence.",
			EvidenceRefs: []string{"incident:" + inc.ID},
		})
	}

	hasTransportSignals := false
	for _, item := range intel.EvidenceItems {
		if item.Kind == "transport_alert" || item.Kind == "dead_letter_reason_cluster" {
			hasTransportSignals = true
			break
		}
	}
	unsupported := []models.IncidentWirelessUnsupported{}
	if bluetoothHit {
		unsupported = append(unsupported, models.IncidentWirelessUnsupported{
			Domain: "bluetooth",
			Scope:  "ingest",
			Note:   "BLE ingest is unsupported in current MEL contract; preserve evidence but do not infer direct runtime diagnosis.",
		})
	}

	sparseGap := containsAny(strings.Join(gaps, " "), []string{"limited_correlated_evidence"}) &&
		!loraHit && !wifiHit && !bluetoothHit
	if !sparseGap && intel.EvidenceStrength == "sparse" && !hasTransportSignals && len(observedDomains) == 1 && observedDomains[0] == "unknown" {
		sparseGap = true
	}

	classification := "recurring_unknown_pattern"
	primary := "unknown"
	evidencePosture := "historical"
	confidence := "inconclusive"
	summary := "Wireless context is ambiguous; compare timeline, linked evidence, and prior incidents before attributing cause."

	switch {
	case sparseGap:
		classification = "sparse_evidence_incident"
		evidencePosture = "sparse"
		confidence = "sparse"
		summary = "Sparse evidence: MEL can preserve context but cannot distinguish mesh, node, frequency, Bluetooth, or Wi-Fi path with confidence."
	case len(observedDomains) > 1:
		classification = "mixed_path_degradation"
		primary = "mixed"
		evidencePosture = "partial"
		confidence = "mixed"
		summary = "Mixed wireless context: multiple domains are implicated by associated evidence; inspect each domain before action."
	case wifiHit && hasTransportSignals:
		classification = "wifi_backhaul_instability"
		primary = "wifi"
		evidencePosture = "partial"
		confidence = "evidence_backed"
		summary = "Wireless context suggests Wi-Fi/backhaul instability association from transport evidence; this is not root-cause proof."
	case loraHit:
		classification = "lora_mesh_pressure"
		primary = "lora"
		evidencePosture = "partial"
		confidence = "mixed"
		summary = "Mesh context includes LoRa/frequency-associated signals; verify node/link history before treating as reach or RF failure."
	case bluetoothHit:
		classification = "unsupported_wireless_domain_observed"
		primary = "bluetooth"
		evidencePosture = "unsupported"
		confidence = "inconclusive"
		summary = "Bluetooth-side context is observed, but MEL currently preserves only bounded evidence and cannot diagnose BLE runtime state."
	case intel.SignatureMatchCount > 1:
		classification = "recurring_unknown_pattern"
		evidencePosture = "historical"
		confidence = "mixed"
		summary = "Recurring pattern detected from similar incidents, but wireless domain attribution remains uncertain."
	}
	if primary == "unknown" && len(observedDomains) == 1 && observedDomains[0] != "unknown" {
		primary = observedDomains[0]
	}
	inspectNext := []string{
		"Review incident timeline ordering and linked evidence before attributing cause.",
	}
	if wifiHit {
		inspectNext = append(inspectNext, "Inspect Wi-Fi/backhaul transport disconnect and dead-letter evidence in the same window.")
	}
	if loraHit {
		inspectNext = append(inspectNext, "Inspect node/link message recency and frequency metadata if available.")
	}
	if bluetoothHit {
		inspectNext = append(inspectNext, "Treat Bluetooth context as unsupported telemetry; use manual nearby checks before action.")
	}
	return &models.IncidentWirelessContext{
		Classification:    classification,
		PrimaryDomain:     primary,
		ObservedDomains:   observedDomains,
		EvidencePosture:   evidencePosture,
		ConfidencePosture: confidence,
		Summary:           summary,
		Reasons:           reasons,
		EvidenceGaps:      dedupeStrings(gaps),
		InspectNext:       dedupeStrings(inspectNext),
		Unsupported:       unsupported,
	}
}

func collectMetadataText(metadata map[string]any) string {
	if len(metadata) == 0 {
		return ""
	}
	parts := make([]string, 0, len(metadata)*2)
	for k, v := range metadata {
		parts = append(parts, strings.ToLower(strings.TrimSpace(k)))
		parts = append(parts, strings.ToLower(strings.TrimSpace(fmt.Sprint(v))))
	}
	sort.Strings(parts)
	return strings.Join(parts, " ")
}

func containsAny(haystack string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(haystack, strings.ToLower(strings.TrimSpace(n))) {
			return true
		}
	}
	return false
}

func dedupeStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		key := strings.TrimSpace(v)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func wirelessRefs(reasons []models.IncidentWirelessReason) []string {
	refs := []string{}
	for _, reason := range reasons {
		refs = append(refs, reason.EvidenceRefs...)
	}
	return dedupeStrings(refs)
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

type actionEvaluation struct {
	outcome             string
	evidenceSufficiency string
	observedSignals     int
	caveats             []string
	inspectBeforeReuse  []string
	evidenceRefs        []string
	transportName       string
	preDeadLetters      int
	postDeadLetters     int
	preAlerts           int
	postAlerts          int
	windowStart         string
	windowEnd           string
	incidentState       string
	actionResult        string
	actionLifecycle     string
}

func evaluatePostActionSignals(a *App, action db.ControlActionRecord, incident models.Incident) actionEvaluation {
	eval := actionEvaluation{
		outcome:             "inconclusive",
		evidenceSufficiency: "insufficient",
		caveats:             []string{"Temporal association only; this is not causal proof."},
		inspectBeforeReuse:  []string{"Confirm current incident signature still matches historical context.", "Validate approval/execution lifecycle before repeating action."},
		evidenceRefs:        []string{"incident:" + incident.ID, "action:" + action.ID},
	}
	base := parseRFC3339Fallback(firstNonEmpty(action.CompletedAt, action.ExecutedAt, action.ExecutionStartedAt, action.CreatedAt), parseRFC3339Fallback(firstNonEmpty(incident.OccurredAt, incident.UpdatedAt), time.Now().UTC()))
	preFrom, preTo := base.Add(-2*time.Hour).Format(time.RFC3339), base.Format(time.RFC3339)
	postFrom, postTo := base.Format(time.RFC3339), base.Add(2*time.Hour).Format(time.RFC3339)
	transport := firstNonEmpty(action.TargetTransport, incident.ResourceID)
	eval.transportName = transport
	eval.windowStart = preFrom
	eval.windowEnd = postTo
	eval.incidentState = incident.State
	eval.actionResult = strings.TrimSpace(action.Result)
	eval.actionLifecycle = strings.TrimSpace(action.LifecycleState)
	if incident.ResourceType == "transport" && transport != "" && a != nil && a.DB != nil {
		preDL, _ := a.DB.DeadLettersForTransportBetween(transport, preFrom, preTo, 200)
		postDL, _ := a.DB.DeadLettersForTransportBetween(transport, postFrom, postTo, 200)
		eval.preDeadLetters = len(preDL)
		eval.postDeadLetters = len(postDL)
		if len(preDL) > 0 || len(postDL) > 0 {
			eval.observedSignals++
			eval.evidenceRefs = append(eval.evidenceRefs, "dead_letters:"+transport)
			switch {
			case len(postDL) < len(preDL):
				eval.outcome = "improvement_observed"
				eval.inspectBeforeReuse = append(eval.inspectBeforeReuse, "Check dead-letter reasons and ensure reduction persisted beyond 2h.")
			case len(postDL) > len(preDL):
				eval.outcome = "deterioration_observed"
				eval.inspectBeforeReuse = append(eval.inspectBeforeReuse, "Dead letters increased after prior use; inspect transport runtime before retry.")
			}
		}
		preAlerts, _ := a.DB.TransportAlertsForWindow(transport, preFrom, preTo, 200)
		postAlerts, _ := a.DB.TransportAlertsForWindow(transport, postFrom, postTo, 200)
		eval.preAlerts = len(preAlerts)
		eval.postAlerts = len(postAlerts)
		if len(preAlerts) > 0 || len(postAlerts) > 0 {
			eval.observedSignals++
			eval.evidenceRefs = append(eval.evidenceRefs, "transport_alerts:"+transport)
			switch {
			case len(postAlerts) < len(preAlerts) && eval.outcome != "deterioration_observed":
				eval.outcome = "improvement_observed"
			case len(postAlerts) > len(preAlerts):
				eval.outcome = "deterioration_observed"
			}
		}
	}
	if isResolvedState(incident.State) {
		eval.observedSignals++
		eval.evidenceRefs = append(eval.evidenceRefs, "incident_state:"+incident.State)
		if eval.outcome != "deterioration_observed" {
			eval.outcome = "improvement_observed"
		}
	}
	if strings.EqualFold(strings.TrimSpace(action.Result), "failed") || strings.EqualFold(strings.TrimSpace(action.LifecycleState), "failed") {
		eval.observedSignals++
		eval.evidenceRefs = append(eval.evidenceRefs, "action_result:failed")
		eval.outcome = "deterioration_observed"
		eval.inspectBeforeReuse = append(eval.inspectBeforeReuse, "Prior execution recorded failed lifecycle/result; inspect action preconditions first.")
	}
	if eval.observedSignals == 0 {
		eval.inspectBeforeReuse = append(eval.inspectBeforeReuse, "No clear post-action signal in bounded window; inspect full timeline manually.")
	}
	switch {
	case eval.observedSignals >= 2:
		eval.evidenceSufficiency = "sufficient"
	case eval.observedSignals == 1:
		eval.evidenceSufficiency = "partial"
	default:
		eval.evidenceSufficiency = "insufficient"
	}
	return eval
}

func modelSnapshotFromDB(snap db.IncidentActionOutcomeSnapshotRecord) models.IncidentActionOutcomeSnapshot {
	return models.IncidentActionOutcomeSnapshot{
		SnapshotID:            snap.SnapshotID,
		SignatureKey:          snap.SignatureKey,
		IncidentID:            snap.IncidentID,
		ActionID:              snap.ActionID,
		ActionType:            snap.ActionType,
		ActionLabel:           snap.ActionLabel,
		DerivedClassification: normalizeSnapshotClassification(snap.DerivedClassification),
		EvidenceSufficiency:   snap.EvidenceSufficiency,
		WindowStart:           snap.WindowStart,
		WindowEnd:             snap.WindowEnd,
		PreActionEvidence: models.IncidentActionEvidenceSummary{
			TransportName:        firstNonEmpty(asMapString(snap.PreActionSummary, "transport_name")),
			DeadLettersCount:     asMapInt(snap.PreActionSummary, "dead_letters_count"),
			TransportAlertsCount: asMapInt(snap.PreActionSummary, "transport_alerts_count"),
			IncidentState:        asMapString(snap.PreActionSummary, "incident_state"),
			ActionResult:         asMapString(snap.PreActionSummary, "action_result"),
			ActionLifecycle:      asMapString(snap.PreActionSummary, "action_lifecycle"),
		},
		PostActionEvidence: models.IncidentActionEvidenceSummary{
			TransportName:        firstNonEmpty(asMapString(snap.PostActionSummary, "transport_name")),
			DeadLettersCount:     asMapInt(snap.PostActionSummary, "dead_letters_count"),
			TransportAlertsCount: asMapInt(snap.PostActionSummary, "transport_alerts_count"),
			IncidentState:        asMapString(snap.PostActionSummary, "incident_state"),
			ActionResult:         asMapString(snap.PostActionSummary, "action_result"),
			ActionLifecycle:      asMapString(snap.PostActionSummary, "action_lifecycle"),
		},
		ObservedSignalCount: snap.ObservedSignalCount,
		Caveats:             append([]string(nil), snap.Caveats...),
		InspectBeforeReuse:  append([]string(nil), snap.InspectBeforeReuse...),
		EvidenceRefs:        append([]string(nil), snap.EvidenceRefs...),
		AssociationOnly:     snap.AssociationOnly,
		DerivationVersion:   snap.DerivationVersion,
		SchemaVersion:       snap.SchemaVersion,
		DerivedAt:           snap.DerivedAt,
	}
}

func normalizeSnapshotClassification(v string) string {
	switch strings.TrimSpace(v) {
	case "improvement_observed", "deterioration_observed", "mixed_historical_evidence", "inconclusive", "insufficient_evidence":
		return v
	case "no_clear_post_action_signal":
		return "inconclusive"
	default:
		return "inconclusive"
	}
}

func asMapInt(in map[string]any, k string) int {
	if in == nil {
		return 0
	}
	switch v := in[k].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case string:
		var out int
		_, _ = fmt.Sscan(strings.TrimSpace(v), &out)
		return out
	default:
		var out int
		_, _ = fmt.Sscan(fmt.Sprint(v), &out)
		return out
	}
}

func asMapString(in map[string]any, k string) string {
	if in == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(in[k]))
}

func classifyOutcome(mem models.IncidentActionOutcomeMemory) (string, string) {
	if mem.SampleSize < 2 {
		return "insufficient_evidence", "insufficient_history"
	}
	if mem.ImprovementObservedCount > 0 && mem.DeteriorationObservedCount > 0 {
		return "mixed_historical_evidence", "mixed_signals"
	}
	if mem.ImprovementObservedCount > 0 {
		return "improvement_observed", "subsequent_improvement_observed"
	}
	if mem.DeteriorationObservedCount > 0 {
		return "deterioration_observed", "subsequent_deterioration_observed"
	}
	return "no_clear_post_action_signal", "inconclusive"
}

func evidenceStrengthForSample(sampleSize, observedSignals int) string {
	switch {
	case sampleSize >= 5 && observedSignals >= 5:
		return "strong"
	case sampleSize >= 3 && observedSignals >= 2:
		return "moderate"
	default:
		return "sparse"
	}
}

func isResolvedState(state string) bool {
	s := strings.ToLower(strings.TrimSpace(state))
	return s == "resolved" || s == "closed"
}

func setToSortedSlice(in map[string]struct{}) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for k := range in {
		if strings.TrimSpace(k) == "" {
			continue
		}
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func snapshotCoveragePercent(occurrenceCount, snapshotRefCount int) float64 {
	if occurrenceCount <= 0 || snapshotRefCount <= 0 {
		return 0
	}
	if snapshotRefCount >= occurrenceCount {
		return 100
	}
	return (float64(snapshotRefCount) / float64(occurrenceCount)) * 100
}

func snapshotTracePosture(occurrenceCount, snapshotRefCount int) (string, string) {
	switch {
	case occurrenceCount <= 0 || snapshotRefCount <= 0:
		return "unavailable", "missing"
	case snapshotRefCount >= occurrenceCount:
		return "complete", "matched"
	default:
		return "partial", "sparse"
	}
}
