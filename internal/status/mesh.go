package status

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/transport"
)

type MeshHealth struct {
	Score                 int               `json:"score"`
	State                 string            `json:"state"`
	DegradedSegments      []DegradedSegment `json:"degraded_segments,omitempty"`
	CriticalSegments      []DegradedSegment `json:"critical_segments,omitempty"`
	DominantFailureReason string            `json:"dominant_failure_reason,omitempty"`
}

type MeshHealthExplanation struct {
	MeshScore             int                 `json:"mesh_score"`
	MeshState             string              `json:"mesh_state"`
	DominantFailureReason string              `json:"dominant_failure_reason,omitempty"`
	AffectedTransports    []string            `json:"affected_transports,omitempty"`
	AffectedNodes         []string            `json:"affected_nodes,omitempty"`
	TopPenalties          []HealthPenalty     `json:"top_penalties,omitempty"`
	ActiveClusters        []FailureCluster    `json:"active_clusters,omitempty"`
	ActiveAlerts          []TransportAlert    `json:"active_alerts,omitempty"`
	DegradedSegments      []DegradedSegment   `json:"degraded_segments,omitempty"`
	EvidenceLossSummary   EvidenceLossSummary `json:"evidence_loss_summary"`
	RecoveryBlockers      []string            `json:"recovery_blockers,omitempty"`
}

type EvidenceLossSummary struct {
	IngestDrops      uint64 `json:"ingest_drops"`
	ObservationDrops uint64 `json:"observation_drops"`
	BusDrops         uint64 `json:"bus_drops"`
}

type CorrelatedFailure struct {
	Reason      string   `json:"reason"`
	Transports  []string `json:"transports"`
	NodeIDs     []string `json:"node_ids,omitempty"`
	Count       uint64   `json:"count"`
	Window      string   `json:"window"`
	Severity    string   `json:"severity"`
	Explanation string   `json:"explanation"`
}

type DegradedSegment struct {
	SegmentID   string   `json:"segment_id"`
	Transports  []string `json:"transports"`
	Nodes       []string `json:"nodes,omitempty"`
	Reason      string   `json:"reason"`
	Severity    string   `json:"severity"`
	Explanation string   `json:"explanation"`
}

type RootCauseAnalysis struct {
	PrimaryCause       string   `json:"primary_cause"`
	Confidence         string   `json:"confidence"`
	SupportingEvidence []string `json:"supporting_evidence,omitempty"`
	Explanation        string   `json:"explanation"`
}

type OperatorRecommendation struct {
	Action            string   `json:"action"`
	Reason            string   `json:"reason"`
	Confidence        string   `json:"confidence"`
	RelatedTransports []string `json:"related_transports,omitempty"`
	RelatedSegments   []string `json:"related_segments,omitempty"`
}

type RoutingRecommendation struct {
	Action          string `json:"action"`
	TargetTransport string `json:"target_transport,omitempty"`
	Reason          string `json:"reason"`
	Confidence      string `json:"confidence"`
}

type MeshHistorySummary struct {
	HealthPoints      int    `json:"health_points"`
	AlertPoints       int    `json:"alert_points"`
	AnomalyPoints     int    `json:"anomaly_points"`
	RetainedSince     string `json:"retained_since,omitempty"`
	RetentionBoundary string `json:"retention_boundary,omitempty"`
}

type MeshDrilldown struct {
	MeshHealth              MeshHealth               `json:"mesh_health"`
	MeshHealthExplanation   MeshHealthExplanation    `json:"mesh_health_explanation"`
	CorrelatedFailures      []CorrelatedFailure      `json:"correlated_failures"`
	DegradedSegments        []DegradedSegment        `json:"degraded_segments"`
	RootCauseAnalysis       RootCauseAnalysis        `json:"root_cause_analysis"`
	OperatorRecommendations []OperatorRecommendation `json:"operator_recommendations"`
	RoutingRecommendations  []RoutingRecommendation  `json:"routing_recommendations"`
	ActiveAlerts            []TransportAlert         `json:"active_alerts"`
	RecentClusters          []FailureCluster         `json:"recent_clusters"`
	HistorySummary          MeshHistorySummary       `json:"history_summary"`
}

type meshPenalty struct {
	HealthPenalty
	TransportName string
}

func InspectMesh(cfg config.Config, database *db.DB, runtime []transport.Health, now time.Time) (MeshDrilldown, error) {
	intel, err := EvaluateTransportIntelligence(cfg, database, runtime, now)
	if err != nil {
		return MeshDrilldown{}, err
	}
	activeAlerts := activeTransportAlerts(database)
	mesh := buildMeshDrilldown(cfg, database, intel, activeAlerts, now)
	return mesh, nil
}

func buildMeshDrilldown(cfg config.Config, database *db.DB, intel TransportIntelligence, activeAlerts []TransportAlert, now time.Time) MeshDrilldown {
	nodesByTransport := recentNodeAttribution(database, cfg, now)
	correlated := correlateFailures(cfg, intel, activeAlerts, nodesByTransport)
	segments := detectDegradedSegments(cfg, intel, correlated, nodesByTransport)
	meshHealth, penalties := scoreMeshHealth(cfg, intel, correlated, segments, activeAlerts)
	recentClusters := aggregateRecentClusters(intel)
	rootCause := analyzeRootCause(intel, correlated, activeAlerts)
	operatorRecommendations := operatorRecommendationsFor(rootCause, segments)
	routingRecommendations := routingRecommendationsFor(intel, meshHealth, correlated)
	historySummary := meshHistorySummary(database, cfg, now)
	explanation := buildMeshHealthExplanation(meshHealth, penalties, correlated, activeAlerts, recentClusters, segments, intel)
	return MeshDrilldown{
		MeshHealth:              meshHealth,
		MeshHealthExplanation:   explanation,
		CorrelatedFailures:      correlated,
		DegradedSegments:        segments,
		RootCauseAnalysis:       rootCause,
		OperatorRecommendations: operatorRecommendations,
		RoutingRecommendations:  routingRecommendations,
		ActiveAlerts:            activeAlerts,
		RecentClusters:          recentClusters,
		HistorySummary:          historySummary,
	}
}

func activeTransportAlerts(database *db.DB) []TransportAlert {
	if database == nil {
		return nil
	}
	rows, err := database.TransportAlerts(true)
	if err != nil {
		return nil
	}
	out := make([]TransportAlert, 0, len(rows))
	for _, alert := range rows {
		penalties := make([]HealthPenalty, 0, len(alert.PenaltySnapshot))
		for _, penalty := range alert.PenaltySnapshot {
			penalties = append(penalties, HealthPenalty{Reason: penalty.Reason, Penalty: penalty.Penalty, Count: penalty.Count, Window: penalty.Window})
		}
		out = append(out, TransportAlert{
			ID:                  alert.ID,
			TransportName:       alert.TransportName,
			TransportType:       alert.TransportType,
			Severity:            alert.Severity,
			Reason:              alert.Reason,
			Summary:             alert.Summary,
			FirstTriggeredAt:    alert.FirstTriggeredAt,
			LastUpdatedAt:       alert.LastUpdatedAt,
			Active:              alert.Active,
			EpisodeID:           alert.EpisodeID,
			ClusterKey:          alert.ClusterKey,
			ContributingReasons: alert.ContributingReasons,
			ClusterReference:    alert.ClusterReference,
			PenaltySnapshot:     penalties,
			TriggerCondition:    alert.TriggerCondition,
		})
	}
	return out
}

func correlateFailures(cfg config.Config, intel TransportIntelligence, activeAlerts []TransportAlert, nodesByTransport map[string][]string) []CorrelatedFailure {
	recentLabel := anomalyWindows(cfg)[1].label
	type aggregate struct {
		reason     string
		transports map[string]struct{}
		nodes      map[string]struct{}
		count      uint64
		window     string
		severity   string
		evidence   []string
	}
	groups := map[string]*aggregate{}
	for transportName, clusters := range intel.ClustersByTransport {
		recent := AnomalyWindow(intel.AnomaliesByTransport[transportName], recentLabel)
		for _, cluster := range clusters {
			if cluster.Severity == "info" {
				continue
			}
			if recent.CountsByReason[cluster.Reason] == 0 && !(cluster.Reason == transport.ReasonRetryThresholdExceeded && recent.CountsByReason[transport.ReasonRetryThresholdExceeded] > 0) && !(cluster.Reason != "" && recent.ObservationDrops > 0 && recent.DropCauses[cluster.Reason] > 0) {
				continue
			}
			key := cluster.Reason + "|" + recentLabel
			agg := groups[key]
			if agg == nil {
				agg = &aggregate{reason: cluster.Reason, transports: map[string]struct{}{}, nodes: map[string]struct{}{}, window: recentLabel, severity: cluster.Severity}
				groups[key] = agg
			}
			agg.transports[transportName] = struct{}{}
			for _, nodeID := range nodesByTransport[transportName] {
				agg.nodes[nodeID] = struct{}{}
			}
			agg.count += cluster.Count
			if severityRank(cluster.Severity) > severityRank(agg.severity) {
				agg.severity = cluster.Severity
			}
			agg.evidence = append(agg.evidence, fmt.Sprintf("%s cluster=%s x%d", transportName, cluster.Reason, cluster.Count))
		}
	}
	for _, alert := range activeAlerts {
		if alert.Severity == "info" || alert.TransportName == "" {
			continue
		}
		key := alert.Reason + "|alerts"
		agg := groups[key]
		if agg == nil {
			agg = &aggregate{reason: alert.Reason, transports: map[string]struct{}{}, nodes: map[string]struct{}{}, window: "active_alerts", severity: alert.Severity}
			groups[key] = agg
		}
		agg.transports[alert.TransportName] = struct{}{}
		for _, nodeID := range nodesByTransport[alert.TransportName] {
			agg.nodes[nodeID] = struct{}{}
		}
		agg.count++
		if severityRank(alert.Severity) > severityRank(agg.severity) {
			agg.severity = alert.Severity
		}
		agg.evidence = append(agg.evidence, fmt.Sprintf("%s alert=%s", alert.TransportName, alert.Reason))
	}
	out := make([]CorrelatedFailure, 0, len(groups))
	for _, agg := range groups {
		transports := setToSortedSlice(agg.transports)
		if len(transports) < 2 {
			continue
		}
		out = append(out, CorrelatedFailure{
			Reason:      agg.reason,
			Transports:  transports,
			NodeIDs:     setToSortedSlice(agg.nodes),
			Count:       agg.count,
			Window:      agg.window,
			Severity:    agg.severity,
			Explanation: fmt.Sprintf("%s observed across %d transports within %s from aggregated clusters/alerts: %s", agg.reason, len(transports), agg.window, strings.Join(dedupeStrings(agg.evidence), "; ")),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if severityRank(out[i].Severity) == severityRank(out[j].Severity) {
			if len(out[i].Transports) == len(out[j].Transports) {
				return out[i].Reason < out[j].Reason
			}
			return len(out[i].Transports) > len(out[j].Transports)
		}
		return severityRank(out[i].Severity) > severityRank(out[j].Severity)
	})
	return out
}

func detectDegradedSegments(cfg config.Config, intel TransportIntelligence, correlated []CorrelatedFailure, nodesByTransport map[string][]string) []DegradedSegment {
	segmentByID := map[string]DegradedSegment{}
	topicByTransport := map[string]string{}
	for _, tc := range cfg.Transports {
		topicByTransport[tc.Name] = strings.TrimSpace(tc.Topic)
	}
	for _, failure := range correlated {
		sharedTopic := sharedTopicForTransports(failure.Transports, topicByTransport)
		segmentID := fmt.Sprintf("segment:%s:%s", failure.Reason, sharedTopic)
		explanation := failure.Explanation
		if sharedTopic != "" {
			explanation += fmt.Sprintf(" Shared topic scope=%s.", sharedTopic)
		}
		segmentByID[segmentID] = DegradedSegment{SegmentID: segmentID, Transports: append([]string(nil), failure.Transports...), Nodes: append([]string(nil), failure.NodeIDs...), Reason: failure.Reason, Severity: failure.Severity, Explanation: explanation}
	}
	for transportName, anomalies := range intel.AnomaliesByTransport {
		recent := AnomalyWindow(anomalies, anomalyWindows(cfg)[1].label)
		if recent.ObservationDrops == 0 || len(recent.DropCauses) == 0 {
			continue
		}
		reason := dominantDropCause(recent)
		segmentID := fmt.Sprintf("segment:%s:%s", reason, topicByTransport[transportName])
		if _, exists := segmentByID[segmentID]; exists {
			continue
		}
		segmentByID[segmentID] = DegradedSegment{SegmentID: segmentID, Transports: []string{transportName}, Nodes: append([]string(nil), nodesByTransport[transportName]...), Reason: reason, Severity: "warn", Explanation: fmt.Sprintf("%s shows %d observation drops dominated by %s within %s.", transportName, recent.ObservationDrops, reason, recent.Window)}
	}
	out := make([]DegradedSegment, 0, len(segmentByID))
	for _, segment := range segmentByID {
		out = append(out, segment)
	}
	sort.Slice(out, func(i, j int) bool {
		if severityRank(out[i].Severity) == severityRank(out[j].Severity) {
			return out[i].SegmentID < out[j].SegmentID
		}
		return severityRank(out[i].Severity) > severityRank(out[j].Severity)
	})
	if len(out) > 8 {
		out = out[:8]
	}
	return out
}

func scoreMeshHealth(cfg config.Config, intel TransportIntelligence, correlated []CorrelatedFailure, segments []DegradedSegment, activeAlerts []TransportAlert) (MeshHealth, []meshPenalty) {
	if len(intel.HealthByTransport) == 0 {
		return MeshHealth{Score: 0, State: "degraded", DominantFailureReason: "no_transport_connected"}, nil
	}
	baseWeight := 0
	weightedTotal := 0
	penalties := make([]meshPenalty, 0)
	affected := map[string]struct{}{}
	dominant := ""
	for name, health := range intel.HealthByTransport {
		weight := 1
		if health.State == "failed" || health.State == "unstable" {
			weight++
		}
		if health.Signals.ActiveEpisode {
			weight++
		}
		baseWeight += weight
		weightedTotal += health.Score * weight
		if health.State != "healthy" {
			affected[name] = struct{}{}
		}
		for _, penalty := range health.Explanation.TopPenalties {
			penalties = append(penalties, meshPenalty{HealthPenalty: penalty, TransportName: name})
		}
		if dominant == "" && health.PrimaryReason != "" {
			dominant = health.PrimaryReason
		}
	}
	score := 100
	if baseWeight > 0 {
		score = weightedTotal / baseWeight
	}
	if len(correlated) > 0 {
		worst := correlated[0]
		score -= 10 * len(worst.Transports)
		if worst.Severity == "critical" {
			score -= 10
		}
		dominant = worst.Reason
	}
	deadLetterPenalty := 0
	evidenceLossPenalty := 0
	for transportName, anomalies := range intel.AnomaliesByTransport {
		recent := AnomalyWindow(anomalies, anomalyWindows(cfg)[1].label)
		if recent.DeadLetters > 0 {
			deadLetterPenalty += minInt(12, int(recent.DeadLetters)*3)
			affected[transportName] = struct{}{}
		}
		if recent.ObservationDrops > 0 {
			evidenceLossPenalty += minInt(12, int(recent.ObservationDrops))
			affected[transportName] = struct{}{}
		}
	}
	score -= deadLetterPenalty + evidenceLossPenalty
	for _, alert := range activeAlerts {
		if alert.Reason == "evidence_loss" && alert.Active {
			score -= 6
			affected[alert.TransportName] = struct{}{}
		}
	}
	if score < 0 {
		score = 0
	}
	criticalSegments := make([]DegradedSegment, 0)
	degradedSegments := make([]DegradedSegment, 0)
	for _, segment := range segments {
		degradedSegments = append(degradedSegments, segment)
		if segment.Severity == "critical" {
			criticalSegments = append(criticalSegments, segment)
		}
	}
	state := healthStateForScore(score)
	if len(correlated) > 0 && state == "degraded" {
		state = "unstable"
	}
	if len(criticalSegments) > 0 && score > 0 && score < 40 {
		state = "failed"
	}
	return MeshHealth{Score: score, State: state, DegradedSegments: degradedSegments, CriticalSegments: criticalSegments, DominantFailureReason: dominant}, penalties
}

func buildMeshHealthExplanation(mesh MeshHealth, penalties []meshPenalty, correlated []CorrelatedFailure, activeAlerts []TransportAlert, recentClusters []FailureCluster, segments []DegradedSegment, intel TransportIntelligence) MeshHealthExplanation {
	explanation := MeshHealthExplanation{
		MeshScore:             mesh.Score,
		MeshState:             mesh.State,
		DominantFailureReason: mesh.DominantFailureReason,
		AffectedTransports:    affectedTransportNames(mesh, correlated, activeAlerts, intel),
		AffectedNodes:         affectedNodeIDs(correlated, segments),
		TopPenalties:          topMeshPenalties(penalties, 5),
		ActiveClusters:        trimClusters(recentClusters, 5),
		ActiveAlerts:          trimAlerts(activeAlerts, 5),
		DegradedSegments:      segments,
		EvidenceLossSummary:   summarizeEvidenceLoss(intel),
	}
	for _, failure := range correlated {
		explanation.RecoveryBlockers = append(explanation.RecoveryBlockers, fmt.Sprintf("correlated:%s[%s]", failure.Reason, strings.Join(failure.Transports, ",")))
	}
	for _, alert := range activeAlerts {
		explanation.RecoveryBlockers = append(explanation.RecoveryBlockers, fmt.Sprintf("alert:%s:%s", alert.TransportName, alert.Reason))
	}
	if explanation.EvidenceLossSummary.IngestDrops > 0 {
		explanation.RecoveryBlockers = append(explanation.RecoveryBlockers, fmt.Sprintf("ingest_drops:%d", explanation.EvidenceLossSummary.IngestDrops))
	}
	if explanation.EvidenceLossSummary.ObservationDrops > 0 {
		explanation.RecoveryBlockers = append(explanation.RecoveryBlockers, fmt.Sprintf("observation_drops:%d", explanation.EvidenceLossSummary.ObservationDrops))
	}
	if explanation.EvidenceLossSummary.BusDrops > 0 {
		explanation.RecoveryBlockers = append(explanation.RecoveryBlockers, fmt.Sprintf("bus_drops:%d", explanation.EvidenceLossSummary.BusDrops))
	}
	explanation.RecoveryBlockers = dedupeStrings(explanation.RecoveryBlockers)
	return explanation
}

func analyzeRootCause(intel TransportIntelligence, correlated []CorrelatedFailure, activeAlerts []TransportAlert) RootCauseAnalysis {
	supporting := []string{}
	for _, failure := range correlated {
		supporting = append(supporting, fmt.Sprintf("correlated_failure:%s transports=%s severity=%s", failure.Reason, strings.Join(failure.Transports, ","), failure.Severity))
	}
	for _, alert := range activeAlerts {
		supporting = append(supporting, fmt.Sprintf("alert:%s:%s severity=%s", alert.TransportName, alert.Reason, alert.Severity))
	}
	counts := map[string]int{}
	for name, anomalies := range intel.AnomaliesByTransport {
		recent := anomalies
		if len(recent) > 1 {
			recent = recent[1:2]
		}
		for _, summary := range recent {
			if summary.CountsByReason[transport.ReasonMalformedPublish] > 0 || summary.CountsByReason[transport.ReasonMalformedFrame] > 0 || summary.CountsByReason[transport.ReasonDecodeFailure] > 0 {
				counts["upstream_data_issue"] += 2
				supporting = append(supporting, fmt.Sprintf("cluster/penalty:%s malformed_or_decode anomalies", name))
			}
			if summary.CountsByReason[transport.ReasonTimeoutFailure] > 0 || summary.CountsByReason[transport.ReasonTimeoutStall] > 0 || summary.CountsByReason[transport.ReasonRetryThresholdExceeded] > 0 {
				counts["connectivity_issue"] += 2
				supporting = append(supporting, fmt.Sprintf("cluster/penalty:%s timeout_or_retry anomalies", name))
			}
			if summary.ObservationDrops > 0 {
				counts["internal_saturation"] += 2
				supporting = append(supporting, fmt.Sprintf("cluster/penalty:%s observation_drops=%d", name, summary.ObservationDrops))
			}
			if summary.CountsByReason[transport.ReasonHandlerRejection] > 0 || summary.CountsByReason[transport.ReasonRejectedPublish] > 0 || summary.CountsByReason[transport.ReasonRejectedSend] > 0 {
				counts["downstream_processing_issue"] += 2
				supporting = append(supporting, fmt.Sprintf("cluster/penalty:%s handler_or_publish rejection", name))
			}
		}
	}
	for _, failure := range correlated {
		switch failure.Reason {
		case transport.ReasonMalformedPublish, transport.ReasonMalformedFrame, transport.ReasonDecodeFailure:
			counts["upstream_data_issue"] += 3
		case transport.ReasonTimeoutFailure, transport.ReasonTimeoutStall, transport.ReasonRetryThresholdExceeded, "heartbeat_loss":
			counts["connectivity_issue"] += 3
		case "ingest_queue_saturation", "observation_queue_saturation", "event_bus_drops":
			counts["internal_saturation"] += 3
		case transport.ReasonHandlerRejection, transport.ReasonRejectedPublish, transport.ReasonRejectedSend:
			counts["downstream_processing_issue"] += 3
		}
	}
	best := ""
	bestScore := -1
	for _, key := range []string{"connectivity_issue", "internal_saturation", "upstream_data_issue", "downstream_processing_issue"} {
		if counts[key] > bestScore {
			best, bestScore = key, counts[key]
		}
	}
	if best == "" || bestScore <= 0 {
		return RootCauseAnalysis{PrimaryCause: "insufficient_mesh_evidence", Confidence: "low", SupportingEvidence: dedupeStrings(supporting), Explanation: "Mesh-level evidence is currently insufficient to attribute a root cause beyond transport-local degradation."}
	}
	confidence := "medium"
	if bestScore >= 6 {
		confidence = "high"
	}
	explanation := map[string]string{
		"upstream_data_issue":         "Malformed/decode failures dominate the active aggregated evidence, which points to an upstream data quality issue rather than transport liveness alone.",
		"connectivity_issue":          "Timeout, retry-threshold, or heartbeat-loss evidence dominates across transports, which points to a connectivity issue.",
		"internal_saturation":         "Observation-drop evidence dominates the active aggregated windows, which points to internal saturation in MEL pipelines rather than upstream absence of data.",
		"downstream_processing_issue": "Handler/publish rejection evidence dominates the active aggregated windows, which points to downstream processing issues after ingest succeeds.",
	}[best]
	return RootCauseAnalysis{PrimaryCause: best, Confidence: confidence, SupportingEvidence: dedupeStrings(supporting), Explanation: explanation}
}

func operatorRecommendationsFor(rootCause RootCauseAnalysis, segments []DegradedSegment) []OperatorRecommendation {
	segmentIDs := make([]string, 0, len(segments))
	transports := map[string]struct{}{}
	for _, segment := range segments {
		segmentIDs = append(segmentIDs, segment.SegmentID)
		for _, transportName := range segment.Transports {
			transports[transportName] = struct{}{}
		}
	}
	relatedTransports := setToSortedSlice(transports)
	actions := map[string]OperatorRecommendation{
		"upstream_data_issue":         {Action: "Investigate upstream data source", Reason: "Malformed flood or decode failures were observed in current mesh evidence.", Confidence: rootCause.Confidence, RelatedTransports: relatedTransports, RelatedSegments: segmentIDs},
		"connectivity_issue":          {Action: "Check network connectivity", Reason: "Timeout, retry-threshold, or heartbeat-loss evidence is correlated across transports.", Confidence: rootCause.Confidence, RelatedTransports: relatedTransports, RelatedSegments: segmentIDs},
		"internal_saturation":         {Action: "Increase ingest capacity or reduce load", Reason: "Observation drops indicate MEL pipeline saturation rather than silent recovery.", Confidence: rootCause.Confidence, RelatedTransports: relatedTransports, RelatedSegments: segmentIDs},
		"downstream_processing_issue": {Action: "Inspect handler logic", Reason: "Handler or publish rejections indicate downstream processing issues after ingest.", Confidence: rootCause.Confidence, RelatedTransports: relatedTransports, RelatedSegments: segmentIDs},
	}
	if rec, ok := actions[rootCause.PrimaryCause]; ok {
		return []OperatorRecommendation{rec}
	}
	return nil
}

func routingRecommendationsFor(intel TransportIntelligence, mesh MeshHealth, correlated []CorrelatedFailure) []RoutingRecommendation {
	if len(intel.HealthByTransport) == 0 {
		return []RoutingRecommendation{{Action: "advisory_only_no_transport_connected", Reason: "No transport connectivity is currently proven; MEL will not auto-reroute.", Confidence: "high"}}
	}
	recommendations := []RoutingRecommendation{}
	healthyAlt := ""
	for name, health := range intel.HealthByTransport {
		if health.State == "healthy" || health.State == "degraded" {
			healthyAlt = name
			break
		}
	}
	for name, health := range intel.HealthByTransport {
		if health.State == "failed" || health.State == "unstable" {
			recommendations = append(recommendations, RoutingRecommendation{Action: "deprioritize_degraded_transport", TargetTransport: name, Reason: fmt.Sprintf("%s is %s with score=%d; keep routing advisory-only and visible to operators.", name, health.State, health.Score), Confidence: "medium"})
			if healthyAlt != "" && healthyAlt != name {
				recommendations = append(recommendations, RoutingRecommendation{Action: "suggest_alternate_ingest_path", TargetTransport: name, Reason: fmt.Sprintf("%s is currently healthier than %s; operators may prefer that path temporarily, but MEL will not auto-reroute.", healthyAlt, name), Confidence: "medium"})
			}
		}
		recent := AnomalyWindow(intel.AnomaliesByTransport[name], "5m")
		if recent.CountsByReason[transport.ReasonMalformedPublish] >= 2 || recent.CountsByReason[transport.ReasonMalformedFrame] >= 2 {
			recommendations = append(recommendations, RoutingRecommendation{Action: "temporarily_suppress_noisy_source", TargetTransport: name, Reason: fmt.Sprintf("%s is carrying a malformed flood pattern; suppress only with operator review.", name), Confidence: "medium"})
		}
		if recent.ObservationDrops >= 3 {
			recommendations = append(recommendations, RoutingRecommendation{Action: "temporarily_suppress_noisy_source", TargetTransport: name, Reason: fmt.Sprintf("%s is dropping observations under load; reduce source pressure before considering routing changes.", name), Confidence: "medium"})
		}
	}
	if len(recommendations) == 0 && mesh.State == "healthy" {
		recommendations = append(recommendations, RoutingRecommendation{Action: "no_routing_change_advised", Reason: "Mesh health is currently stable; no advisory reroute is warranted.", Confidence: "high"})
	}
	return dedupeRoutingRecommendations(recommendations)
}

func meshHistorySummary(database *db.DB, cfg config.Config, now time.Time) MeshHistorySummary {
	if database == nil {
		return MeshHistorySummary{RetentionBoundary: "history unavailable without database evidence"}
	}
	start := now.Add(-24 * time.Hour).Format(time.RFC3339)
	end := now.Format(time.RFC3339)
	healthRows, _ := database.TransportHealthSnapshots("", start, end, cfg.Intelligence.Queries.DefaultLimit, 0)
	alertRows, _ := database.TransportAlertsHistory("", start, end, cfg.Intelligence.Queries.DefaultLimit, 0)
	anomalyRows, _ := database.TransportAnomalyHistory("", start, end, cfg.Intelligence.Queries.DefaultLimit, 0)
	retainedSince := now.AddDate(0, 0, -cfg.Intelligence.Retention.HealthSnapshotDays).Format(time.RFC3339)
	return MeshHistorySummary{HealthPoints: len(healthRows), AlertPoints: len(alertRows), AnomalyPoints: len(anomalyRows), RetainedSince: retainedSince, RetentionBoundary: fmt.Sprintf("bounded by intelligence.retention.health_snapshot_days=%d and intelligence.retention.health_snapshot_max_rows=%d", cfg.Intelligence.Retention.HealthSnapshotDays, cfg.Intelligence.Retention.HealthSnapshotMaxRows)}
}

func recentNodeAttribution(database *db.DB, cfg config.Config, now time.Time) map[string][]string {
	if database == nil {
		return map[string][]string{}
	}
	window := 15 * time.Minute
	if windows := anomalyWindows(cfg); len(windows) > 0 && windows[len(windows)-1].duration > 0 {
		window = windows[len(windows)-1].duration
	}
	nodes, err := database.RecentTransportNodeAttribution(now.Add(-window).UTC().Format(time.RFC3339), now.UTC().Format(time.RFC3339), 5)
	if err != nil {
		return map[string][]string{}
	}
	return nodes
}

func affectedNodeIDs(correlated []CorrelatedFailure, segments []DegradedSegment) []string {
	set := map[string]struct{}{}
	for _, failure := range correlated {
		for _, nodeID := range failure.NodeIDs {
			set[nodeID] = struct{}{}
		}
	}
	for _, segment := range segments {
		for _, nodeID := range segment.Nodes {
			set[nodeID] = struct{}{}
		}
	}
	return setToSortedSlice(set)
}

func aggregateRecentClusters(intel TransportIntelligence) []FailureCluster {
	out := make([]FailureCluster, 0)
	for _, clusters := range intel.ClustersByTransport {
		out = append(out, clusters...)
	}
	sort.Slice(out, func(i, j int) bool {
		if severityRank(out[i].Severity) == severityRank(out[j].Severity) {
			return out[i].LastSeen > out[j].LastSeen
		}
		return severityRank(out[i].Severity) > severityRank(out[j].Severity)
	})
	if len(out) > 10 {
		out = out[:10]
	}
	return out
}

func trimAlerts(alerts []TransportAlert, limit int) []TransportAlert {
	if len(alerts) <= limit {
		return append([]TransportAlert(nil), alerts...)
	}
	out := append([]TransportAlert(nil), alerts...)
	sort.Slice(out, func(i, j int) bool {
		if alertSeverityRank(out[i].Severity) == alertSeverityRank(out[j].Severity) {
			return out[i].LastUpdatedAt > out[j].LastUpdatedAt
		}
		return alertSeverityRank(out[i].Severity) > alertSeverityRank(out[j].Severity)
	})
	return out[:limit]
}

func trimClusters(clusters []FailureCluster, limit int) []FailureCluster {
	if len(clusters) <= limit {
		return append([]FailureCluster(nil), clusters...)
	}
	return append([]FailureCluster(nil), clusters[:limit]...)
}

func summarizeEvidenceLoss(intel TransportIntelligence) EvidenceLossSummary {
	summary := EvidenceLossSummary{}
	for _, anomalies := range intel.AnomaliesByTransport {
		recent := AnomalyWindow(anomalies, "5m")
		summary.ObservationDrops += recent.ObservationDrops
		for cause, count := range recent.DropCauses {
			switch {
			case strings.Contains(cause, "ingest"):
				summary.IngestDrops += count
			case strings.Contains(cause, "bus"):
				summary.BusDrops += count
			default:
				summary.ObservationDrops += 0
			}
		}
	}
	return summary
}

func affectedTransportNames(mesh MeshHealth, correlated []CorrelatedFailure, activeAlerts []TransportAlert, intel TransportIntelligence) []string {
	set := map[string]struct{}{}
	for _, failure := range correlated {
		for _, transportName := range failure.Transports {
			set[transportName] = struct{}{}
		}
	}
	for _, alert := range activeAlerts {
		set[alert.TransportName] = struct{}{}
	}
	for name, health := range intel.HealthByTransport {
		if health.State != "healthy" {
			set[name] = struct{}{}
		}
	}
	if len(set) == 0 && mesh.State != "healthy" {
		for name := range intel.HealthByTransport {
			set[name] = struct{}{}
		}
	}
	return setToSortedSlice(set)
}

func topMeshPenalties(in []meshPenalty, limit int) []HealthPenalty {
	sort.Slice(in, func(i, j int) bool {
		if in[i].Penalty == in[j].Penalty {
			if in[i].Reason == in[j].Reason {
				return in[i].TransportName < in[j].TransportName
			}
			return in[i].Reason < in[j].Reason
		}
		return in[i].Penalty > in[j].Penalty
	})
	if len(in) > limit {
		in = in[:limit]
	}
	out := make([]HealthPenalty, 0, len(in))
	for _, penalty := range in {
		out = append(out, penalty.HealthPenalty)
	}
	return out
}

func sharedTopicForTransports(transports []string, topicByTransport map[string]string) string {
	if len(transports) == 0 {
		return ""
	}
	topic := strings.TrimSpace(topicByTransport[transports[0]])
	if topic == "" {
		return ""
	}
	for _, transportName := range transports[1:] {
		if strings.TrimSpace(topicByTransport[transportName]) != topic {
			return ""
		}
	}
	return topic
}

func setToSortedSlice(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for item := range set {
		if strings.TrimSpace(item) == "" {
			continue
		}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func dedupeRoutingRecommendations(in []RoutingRecommendation) []RoutingRecommendation {
	seen := map[string]struct{}{}
	out := make([]RoutingRecommendation, 0, len(in))
	for _, item := range in {
		key := item.Action + "|" + item.TargetTransport + "|" + item.Reason
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func alertSeverityRank(severity string) int {
	switch severity {
	case "critical":
		return 3
	case "warn":
		return 2
	default:
		return 1
	}
}
