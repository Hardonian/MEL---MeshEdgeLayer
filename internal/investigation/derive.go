package investigation

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/diagnostics"
	"github.com/mel-project/mel/internal/fleet"
	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/transport"
)

type deriveContext struct {
	truth         fleet.FleetTruthSummary
	incidents     []models.Incident
	alerts        []db.TransportAlertRecord
	imports       []db.ImportedRemoteEvidenceRecord
	batches       []db.RemoteImportBatchRecord
	timeline      []db.TimelineEvent
	runtimeStates []db.TransportRuntime
}

// Derive assembles a canonical investigation Summary from current system
// state. It reads from real sources only and keeps uncertainty explicit.
func Derive(
	cfg config.Config,
	d *db.DB,
	runtimeTransports []transport.Health,
	transportStates []db.TransportRuntime,
	now time.Time,
) Summary {
	nowStr := now.UTC().Format(time.RFC3339)
	ctx := loadDeriveContext(cfg, d, transportStates)

	var findings []Finding
	var gaps []EvidenceGap
	var recs []Recommendation

	diagRun := diagnostics.RunAllChecks(cfg, d, runtimeTransports, transportStates, now)
	diagFindings, diagGaps, diagRecs := deriveDiagnostics(diagRun.Diagnostics, nowStr)
	findings = append(findings, diagFindings...)
	gaps = append(gaps, diagGaps...)
	recs = append(recs, diagRecs...)

	tFindings, tGaps, tRecs := deriveTransportState(cfg, runtimeTransports, ctx, nowStr)
	findings = append(findings, tFindings...)
	gaps = append(gaps, tGaps...)
	recs = append(recs, tRecs...)

	iFindings, iGaps, iRecs := deriveIncidents(ctx, nowStr)
	findings = append(findings, iFindings...)
	gaps = append(gaps, iGaps...)
	recs = append(recs, iRecs...)

	fTruthFindings, fTruthGaps, fTruthRecs := deriveFleetTruth(ctx.truth, nowStr)
	findings = append(findings, fTruthFindings...)
	gaps = append(gaps, fTruthGaps...)
	recs = append(recs, fTruthRecs...)

	fFindings, fGaps, fRecs := deriveFleetPosture(ctx, nowStr)
	findings = append(findings, fFindings...)
	gaps = append(gaps, fGaps...)
	recs = append(recs, fRecs...)

	sFindings, sGaps, sRecs := deriveStaleness(ctx.timeline, nowStr, now)
	findings = append(findings, sFindings...)
	gaps = append(gaps, sGaps...)
	recs = append(recs, sRecs...)

	findings = dedupeFindings(findings)
	gaps = dedupeGaps(gaps)
	recs = dedupeRecommendations(recs)
	crossLink(&findings, &gaps, &recs)

	sortFindings(findings)
	cases := buildCases(findings, gaps, ctx)
	sortCases(cases)

	counts := computeCounts(findings, gaps, recs)
	overall := computeOverallAttention(findings)
	certainty := computeOverallCertainty(findings)
	headline := buildHeadline(counts, overall, certainty)
	scopePosture := computeScopePosture(findings)
	attentionSummary := buildAttentionSummary(overall, certainty, counts)

	return Summary{
		GeneratedAt:      nowStr,
		OverallAttention: overall,
		OverallCertainty: certainty,
		Headline:         headline,
		AttentionSummary: attentionSummary,
		Findings:         findings,
		EvidenceGaps:     gaps,
		Recommendations:  recs,
		Cases:            cases,
		Counts:           counts,
		CaseCounts:       caseCounts(cases),
		ScopePosture:     scopePosture,
		PhysicsBoundary:  DefaultPhysicsBoundary(),
	}
}

func loadDeriveContext(cfg config.Config, d *db.DB, transportStates []db.TransportRuntime) deriveContext {
	ctx := deriveContext{runtimeStates: transportStates}

	truth, err := fleet.BuildTruthSummary(cfg, d)
	if err != nil {
		truth, _ = fleet.BuildTruthSummary(cfg, nil)
		truth.PartialVisibilityReasons = appendUnique(truth.PartialVisibilityReasons, "fleet_truth_unavailable:"+err.Error())
	}
	ctx.truth = truth

	if d == nil {
		return ctx
	}

	ctx.incidents, _ = d.RecentIncidents(25)
	ctx.alerts, _ = d.TransportAlerts(true)
	ctx.imports, _ = d.ListImportedRemoteEvidence(100)
	ctx.batches, _ = d.ListRemoteImportBatches(25)
	ctx.timeline, _ = d.TimelineEvents("", "", 200)
	return ctx
}

func deriveDiagnostics(diags []diagnostics.Finding, nowStr string) ([]Finding, []EvidenceGap, []Recommendation) {
	var findings []Finding
	var recs []Recommendation

	for _, diag := range diags {
		attention := mapDiagSeverity(diag.Severity)
		certainty := 0.8
		if diag.Severity == diagnostics.SeverityInfo {
			certainty = 0.9
		}

		f := NewFinding(
			"diag_"+diag.Code,
			mapDiagCategory(diag.Component),
			attention,
			certainty,
			diag.Title,
			diag.Explanation,
			mustParseTime(nowStr),
		)
		f.Source = "diagnostics"
		f.ObservedAt = nowStr
		f.ResourceID = strings.TrimSpace(diag.AffectedTransport)
		f.OperatorActionRequired = diag.OperatorActionRequired
		f.CanAutoRecover = diag.CanAutoRecover
		f.WhyItMatters = diagnosticWhyItMatters(diag)

		snapshot := map[string]any{}
		if diag.Component != "" {
			snapshot["component"] = diag.Component
		}
		if diag.AffectedTransport != "" {
			snapshot["affected_transport"] = diag.AffectedTransport
		}
		if len(diag.Evidence) > 0 {
			snapshot["diagnostic_evidence"] = diag.Evidence
		}
		if len(snapshot) > 0 {
			f.EvidenceSnapshot = snapshot
		}

		if diag.OperatorActionRequired || diag.Severity == diagnostics.SeverityCritical || diag.Severity == diagnostics.SeverityWarning {
			rec := NewRecommendation(
				RecRunDiagnostics,
				fmt.Sprintf("Investigate diagnostic finding: %s", diag.Title),
				fmt.Sprintf("Diagnostic %s reported %s severity for %s. This is a bounded observation, not a confirmed root cause.",
					diag.Code, diag.Severity, diag.Component),
				"operator_only",
				ScopeLocal,
				nowStr,
			)
			rec.ID = "diag_rec_" + diag.Code
			rec.FindingIDs = []string{f.ID}
			recs = append(recs, rec)
			f.RecommendationIDs = append(f.RecommendationIDs, rec.ID)
		}

		findings = append(findings, f)
	}

	return findings, nil, recs
}

func deriveTransportState(cfg config.Config, runtimeTransports []transport.Health, ctx deriveContext, nowStr string) ([]Finding, []EvidenceGap, []Recommendation) {
	var findings []Finding
	var gaps []EvidenceGap
	var recs []Recommendation

	enabledCount := 0
	for _, t := range cfg.Transports {
		if t.Enabled {
			enabledCount++
		}
	}

	if enabledCount == 0 {
		f := NewFinding(
			"no_transports_enabled",
			CategoryTransport,
			AttentionInfo,
			1.0,
			"No transports enabled",
			"No transports are enabled in the configuration. MEL is explicitly idle and not expecting ingest.",
			mustParseTime(nowStr),
		)
		f.WhyItMatters = "Without an enabled transport, MEL cannot collect live mesh evidence."
		f.Source = "transport_state"
		f.ObservedAt = nowStr
		findings = append(findings, f)
		return findings, gaps, recs
	}

	ingestingCount := 0
	failedTransports := map[string]transport.Health{}
	for _, tr := range runtimeTransports {
		switch tr.State {
		case transport.StateIngesting:
			ingestingCount++
		case transport.StateFailed:
			failedTransports[tr.Name] = tr
		}
	}

	if ingestingCount == 0 {
		gap := NewEvidenceGap(
			GapMissingExpectedReporters,
			"No active transport reporters",
			fmt.Sprintf("%d transport(s) are enabled but none are currently proving live ingest. Missing telemetry is an evidence gap, not proof that the mesh is quiet.", enabledCount),
			"Current mesh conclusions are bounded to historical evidence until a transport resumes ingest.",
			ScopeLocal,
			nowStr,
		)
		gap.ID = string(GapMissingExpectedReporters) + ":local:transports"

		rec := NewRecommendation(
			RecInspectTransport,
			"Verify transport connectivity and ingest configuration",
			"Enabled transports are present but none are proving live ingest. Inspect runtime state, last errors, and physical or broker connectivity before treating the mesh as quiet.",
			"operator_only",
			ScopeLocal,
			nowStr,
		)
		rec.ID = "rec_verify_transports"
		rec.EvidenceGapIDs = []string{gap.ID}

		f := NewFinding(
			"no_active_ingest",
			CategoryTransport,
			AttentionHigh,
			0.65,
			"No transport is actively ingesting",
			fmt.Sprintf("%d transport(s) are enabled but none are in ingesting state. MEL does not currently have live local ingest proof.", enabledCount),
			mustParseTime(nowStr),
		)
		f.WhyItMatters = "Without current ingest proof, operator decisions drift toward historical-only evidence."
		f.Source = "transport_state"
		f.ObservedAt = nowStr
		f.OperatorActionRequired = true
		f.EvidenceGapIDs = []string{gap.ID}
		f.RecommendationIDs = []string{rec.ID}
		f.EvidenceSnapshot = map[string]any{
			"enabled_transports": enabledCount,
			"ingesting_count":    ingestingCount,
		}
		rec.FindingIDs = []string{f.ID}

		findings = append(findings, f)
		gaps = append(gaps, gap)
		recs = append(recs, rec)
	}

	for name, runtime := range failedTransports {
		rec := NewRecommendation(
			RecInspectTransport,
			fmt.Sprintf("Investigate failed transport '%s'", name),
			fmt.Sprintf("Transport '%s' is reporting a failure state. Inspect the transport runtime, alerts, and supporting timeline evidence before inferring broader mesh impact.", name),
			"operator_only",
			ScopeLocal,
			nowStr,
		)
		rec.ID = "rec_inspect_transport:" + name

		f := NewFinding(
			"transport_failed",
			CategoryTransport,
			AttentionCritical,
			0.95,
			fmt.Sprintf("Transport '%s' is in failed state", name),
			fmt.Sprintf("Transport '%s' is reporting state=%s. This is a local transport observation, not proof of fleet-wide failure.", name, runtime.State),
			mustParseTime(nowStr),
		)
		f.ID = "transport_failed:" + name
		f.ResourceID = name
		f.Source = "transport_state"
		f.ObservedAt = nowStr
		f.OperatorActionRequired = true
		f.WhyItMatters = "A failed transport removes or degrades visibility through that ingest path."
		f.RecommendationIDs = []string{rec.ID}
		f.EvidenceSnapshot = map[string]any{
			"state":         runtime.State,
			"last_error":    runtime.LastError,
			"failure_count": runtime.FailureCount,
		}
		rec.FindingIDs = []string{f.ID}

		findings = append(findings, f)
		recs = append(recs, rec)
	}

	alertsByTransport := make(map[string][]db.TransportAlertRecord)
	for _, alert := range ctx.alerts {
		if !alert.Active {
			continue
		}
		alertsByTransport[alert.TransportName] = append(alertsByTransport[alert.TransportName], alert)
	}
	for transportName, alerts := range alertsByTransport {
		if len(alerts) == 0 {
			continue
		}
		rec := NewRecommendation(
			RecInspectTransport,
			fmt.Sprintf("Inspect transport '%s' alert evidence", transportName),
			fmt.Sprintf("%d active transport alert(s) are associated with '%s'. Review alert reasons and timeline context before assuming a single root cause.", len(alerts), transportName),
			"operator_only",
			ScopeLocal,
			nowStr,
		)
		rec.ID = "rec_transport_alerts:" + transportName

		f := NewFinding(
			"transport_alerts_active",
			CategoryTransport,
			AttentionHigh,
			0.85,
			fmt.Sprintf("Transport '%s' has %d active alert(s)", transportName, len(alerts)),
			fmt.Sprintf("Transport '%s' has %d active alert(s). These alerts are evidence-backed symptoms and should be inspected with their contributing reasons.", transportName, len(alerts)),
			mustParseTime(nowStr),
		)
		f.ID = "transport_alerts:" + transportName
		f.ResourceID = transportName
		f.Source = "transport_state"
		f.ObservedAt = newestAlertUpdate(alerts, nowStr)
		f.WhyItMatters = "Active transport alerts signal degraded ingest quality or repeated transport symptoms."
		f.RecommendationIDs = []string{rec.ID}
		f.EvidenceIDs = alertIDs(alerts)
		f.EvidenceSnapshot = map[string]any{
			"active_alert_count": len(alerts),
			"alert_reasons":      alertReasons(alerts),
		}
		rec.FindingIDs = []string{f.ID}

		findings = append(findings, f)
		recs = append(recs, rec)
	}

	return findings, gaps, recs
}

func deriveIncidents(ctx deriveContext, nowStr string) ([]Finding, []EvidenceGap, []Recommendation) {
	var findings []Finding
	var recs []Recommendation

	for _, inc := range ctx.incidents {
		if inc.State == "resolved" || inc.State == "suppressed" {
			continue
		}

		rec := NewRecommendation(
			RecRunDiagnostics,
			fmt.Sprintf("Investigate incident %s", inc.ID),
			fmt.Sprintf("Incident %s is still open. Treat it as an operator attention object backed by the stored incident record, not as root-cause proof.", inc.ID),
			"operator_only",
			ScopeLocal,
			nowStr,
		)
		rec.ID = "rec_incident:" + inc.ID

		f := NewFinding(
			"open_incident",
			mapIncidentCategory(inc.Category),
			mapIncidentSeverity(inc.Severity),
			0.7,
			fmt.Sprintf("Open incident: %s", inc.Title),
			fmt.Sprintf("Incident %s remains open with severity=%s and category=%s. Review the incident record, linked evidence, and control history before drawing conclusions.", inc.ID, inc.Severity, inc.Category),
			mustParseTime(nowStr),
		)
		f.ID = "incident:" + inc.ID
		f.ResourceID = inc.ID
		f.Source = "incidents"
		f.ObservedAt = firstNonEmptyString(inc.OccurredAt, nowStr)
		f.OperatorActionRequired = true
		f.WhyItMatters = incidentWhyItMatters(inc)
		f.EvidenceIDs = []string{inc.ID}
		f.RecommendationIDs = []string{rec.ID}
		f.EvidenceSnapshot = map[string]any{
			"incident_id":   inc.ID,
			"state":         inc.State,
			"resource_type": inc.ResourceType,
			"resource_id":   inc.ResourceID,
		}
		rec.FindingIDs = []string{f.ID}

		findings = append(findings, f)
		recs = append(recs, rec)
	}

	return findings, nil, recs
}

func deriveFleetTruth(truth fleet.FleetTruthSummary, nowStr string) ([]Finding, []EvidenceGap, []Recommendation) {
	var findings []Finding
	var gaps []EvidenceGap
	var recs []Recommendation

	if truth.Visibility != fleet.VisibilityPartialFleet && truth.TruthPosture != fleet.TruthPostureUnknownSite {
		return findings, gaps, recs
	}

	explanation := "MEL's evidence boundary does not cover the full declared fleet scope."
	if len(truth.PartialVisibilityReasons) > 0 {
		explanation = explanation + " Reasons: " + strings.Join(truth.PartialVisibilityReasons, ", ") + "."
	}

	gap := NewEvidenceGap(
		GapScopeIncomplete,
		"Fleet visibility is partial",
		explanation,
		"Missing or out-of-scope reporters limit fleet-wide certainty. Missing observations are unknown, not healthy.",
		ScopePartialFleet,
		nowStr,
	)
	gap.ID = string(GapScopeIncomplete) + ":partial_fleet"

	rec := NewRecommendation(
		RecCollectMoreEvidence,
		"Seek corroboration from missing reporters or adjacent scopes",
		"This instance is operating with partial fleet visibility. Verify expected reporters and collect corroborating evidence before treating any fleet-wide conclusion as settled.",
		"operator_verify",
		ScopePartialFleet,
		nowStr,
	)
	rec.ID = "rec_partial_fleet_visibility"
	rec.EvidenceGapIDs = []string{gap.ID}
	rec.UncertaintyLimits = append(rec.UncertaintyLimits,
		"Partial fleet visibility prevents fleet-wide absence or health claims.")

	f := NewFinding(
		"partial_fleet_visibility",
		CategoryFleet,
		AttentionHigh,
		0.55,
		"Fleet visibility is partial",
		explanation,
		mustParseTime(nowStr),
	)
	f.Scope = ScopePartialFleet
	f.Source = "fleet_truth"
	f.ObservedAt = nowStr
	f.WhyItMatters = "High operator attention can coexist with low certainty when reporters are missing or scope is incomplete."
	f.EvidenceGapIDs = []string{gap.ID}
	f.RecommendationIDs = []string{rec.ID}
	f.EvidenceSnapshot = map[string]any{
		"truth_posture":              truth.TruthPosture,
		"visibility_posture":         truth.Visibility,
		"expected_fleet_reporters":   truth.ExpectedFleetReporters,
		"reporting_instances_known":  truth.ReportingInstancesKnown,
		"partial_visibility_reasons": truth.PartialVisibilityReasons,
	}
	rec.FindingIDs = []string{f.ID}

	findings = append(findings, f)
	gaps = append(gaps, gap)
	recs = append(recs, rec)
	return findings, gaps, recs
}

func deriveFleetPosture(ctx deriveContext, nowStr string) ([]Finding, []EvidenceGap, []Recommendation) {
	if len(ctx.imports) == 0 {
		return nil, nil, nil
	}

	var findings []Finding
	var gaps []EvidenceGap
	var recs []Recommendation

	importIDs := importIDs(ctx.imports)

	historicalGap := NewEvidenceGap(
		GapImportedHistoricalOnly,
		"Imported evidence is historical context only",
		fmt.Sprintf("MEL has %d imported remote evidence item(s). These imports are offline historical context, not live federation or current local proof.", len(ctx.imports)),
		"Imported evidence may shape investigation context, but it cannot by itself prove current local or fleet-wide state.",
		ScopeImported,
		nowStr,
	)
	historicalGap.ID = string(GapImportedHistoricalOnly) + ":imported:evidence"
	gaps = append(gaps, historicalGap)

	orderingGap := NewEvidenceGap(
		GapOrderingUncertain,
		"Imported evidence ordering is best-effort",
		"Imported evidence preserves reported timing posture, but cross-instance ordering remains best-effort and clock skew may be present.",
		"Do not infer strict causality between imported and local events unless another record establishes it.",
		ScopeImported,
		nowStr,
	)
	orderingGap.ID = string(GapOrderingUncertain) + ":imported:evidence"
	gaps = append(gaps, orderingGap)

	localConfirmation := hasLocalTimelineEvidence(ctx.timeline)
	if !localConfirmation {
		noLocalGap := NewEvidenceGap(
			GapNoLocalConfirmation,
			"Imported evidence has no local confirmation",
			"Imported evidence is present, but this instance does not currently have corroborating local timeline evidence for the same condition.",
			"Recommendations based on imported evidence remain advisory until locally confirmed or independently corroborated.",
			ScopeImported,
			nowStr,
		)
		noLocalGap.ID = string(GapNoLocalConfirmation) + ":imported:evidence"
		gaps = append(gaps, noLocalGap)
	}

	baseRec := NewRecommendation(
		RecTreatAsHistoricalOnly,
		"Treat imported evidence as historical context until locally confirmed",
		"Imported evidence is useful for investigation context, but MEL does not implement live federation or cryptographic origin proof in core.",
		"operator_verify",
		ScopeImported,
		nowStr,
	)
	baseRec.ID = "rec_imported_historical_only"
	baseRec.EvidenceBasis = append(baseRec.EvidenceBasis, importIDs...)
	baseRec.EvidenceGapIDs = append(baseRec.EvidenceGapIDs, historicalGap.ID, orderingGap.ID)
	if !localConfirmation {
		baseRec.EvidenceGapIDs = append(baseRec.EvidenceGapIDs, string(GapNoLocalConfirmation)+":imported:evidence")
		baseRec.Code = RecNoSafeConclusionYet
		baseRec.Action = "Do not draw a safe current-state conclusion from imported evidence alone"
		baseRec.Rationale = "Imported evidence is present without local confirmation. Treat it as historical context and collect corroborating local evidence before acting."
	} else {
		baseRec.Code = RecCompareLocalVsImported
		baseRec.Action = "Compare imported evidence against local records"
		baseRec.Rationale = "Imported evidence is present and local records exist. Compare local timeline, transport state, and incident records before deciding whether the imported pattern still applies."
	}
	recs = append(recs, baseRec)

	f := NewFinding(
		"imported_historical_context",
		CategoryImport,
		AttentionMedium,
		0.6,
		"Imported evidence is present as historical context",
		fmt.Sprintf("%d imported remote evidence item(s) are available for investigation. They remain bounded to their origin and import posture.", len(ctx.imports)),
		mustParseTime(nowStr),
	)
	f.Scope = ScopeImported
	f.Source = "fleet_posture"
	f.ObservedAt = newestImportTime(ctx.imports, nowStr)
	f.WhyItMatters = "Imported evidence can explain why operators should look deeper, but it is not the same as current local proof."
	f.EvidenceIDs = importIDs
	f.EvidenceGapIDs = []string{historicalGap.ID, orderingGap.ID}
	if !localConfirmation {
		f.EvidenceGapIDs = append(f.EvidenceGapIDs, string(GapNoLocalConfirmation)+":imported:evidence")
	}
	f.RecommendationIDs = []string{baseRec.ID}
	f.EvidenceSnapshot = map[string]any{
		"import_count":       len(ctx.imports),
		"batch_count":        len(ctx.batches),
		"local_confirmation": localConfirmation,
	}
	baseRec.FindingIDs = append(baseRec.FindingIDs, f.ID)
	findings = append(findings, f)

	unverifiedCount := 0
	for _, imp := range ctx.imports {
		if imp.ValidationStatus != "accepted" {
			unverifiedCount++
		}
	}
	if unverifiedCount > 0 {
		authGap := NewEvidenceGap(
			GapAuthenticityUnverified,
			"Imported evidence has validation caveats",
			fmt.Sprintf("%d imported evidence item(s) were accepted with caveats or rejected. Their claimed origin remains unverified in core MEL.", unverifiedCount),
			"Authenticity and authority remain bounded; imported evidence should not be treated as fully trusted current state.",
			ScopeImported,
			nowStr,
		)
		authGap.ID = string(GapAuthenticityUnverified) + ":imported:evidence"
		gaps = append(gaps, authGap)

		f2 := NewFinding(
			"fleet_unverified_imports",
			CategoryImport,
			AttentionMedium,
			0.45,
			fmt.Sprintf("%d imported evidence item(s) carry validation caveats", unverifiedCount),
			"Some imported evidence rows were not fully accepted. The validation outcome itself is evidence that imported conclusions need extra caution.",
			mustParseTime(nowStr),
		)
		f2.Scope = ScopeImported
		f2.Source = "fleet_posture"
		f2.ObservedAt = newestImportTime(ctx.imports, nowStr)
		f2.WhyItMatters = "Validation caveats further reduce certainty for any conclusion that leans on imported evidence."
		f2.EvidenceIDs = importIDs
		f2.EvidenceGapIDs = []string{authGap.ID, historicalGap.ID}
		f2.RecommendationIDs = []string{baseRec.ID}
		findings = append(findings, f2)
	}

	return findings, gaps, recs
}

func deriveStaleness(timeline []db.TimelineEvent, nowStr string, now time.Time) ([]Finding, []EvidenceGap, []Recommendation) {
	if len(timeline) == 0 {
		return nil, nil, nil
	}

	lastEvent := timeline[0]
	lastTime, err := time.Parse(time.RFC3339, lastEvent.EventTime)
	if err != nil || now.Sub(lastTime) <= 24*time.Hour {
		return nil, nil, nil
	}

	staleDuration := now.Sub(lastTime).Round(time.Hour)
	gap := NewEvidenceGap(
		GapStaleContributors,
		"Evidence is stale",
		fmt.Sprintf("The newest timeline evidence is %s old. Fresh reporters may be missing, delayed, or disconnected.", staleDuration),
		"Any conclusion that depends on current state is constrained until fresh evidence arrives.",
		ScopeLocal,
		nowStr,
	)
	gap.ID = string(GapStaleContributors) + ":local:timeline"

	rec := NewRecommendation(
		RecWaitForFreshEvidence,
		"Verify the evidence pipeline and wait for fresh local evidence",
		"Timeline evidence is stale. Check live ingest posture and verify transports before treating current conditions as settled.",
		"operator_only",
		ScopeLocal,
		nowStr,
	)
	rec.ID = "rec_stale_evidence"
	rec.EvidenceGapIDs = []string{gap.ID}

	f := NewFinding(
		"stale_evidence",
		CategoryStorage,
		AttentionHigh,
		0.75,
		fmt.Sprintf("Latest evidence is %s old", staleDuration),
		fmt.Sprintf("The newest timeline event is %s (%s ago). MEL is operating on stale recorded evidence.", lastEvent.EventTime, staleDuration),
		mustParseTime(nowStr),
	)
	f.Source = "staleness"
	f.ObservedAt = nowStr
	f.OperatorActionRequired = true
	f.WhyItMatters = "Stale evidence can make an active problem look quiet or resolved when it may simply be unobserved."
	f.EvidenceIDs = []string{lastEvent.EventID}
	f.EvidenceGapIDs = []string{gap.ID}
	f.RecommendationIDs = []string{rec.ID}
	rec.FindingIDs = []string{f.ID}

	return []Finding{f}, []EvidenceGap{gap}, []Recommendation{rec}
}

func buildCases(findings []Finding, gaps []EvidenceGap, ctx deriveContext) []Case {
	gapByID := make(map[string]EvidenceGap, len(gaps))
	for _, gap := range gaps {
		gapByID[gap.ID] = gap
	}

	var cases []Case
	cases = append(cases, buildTransportCases(findings, gapByID, ctx)...)
	if c, ok := buildEvidenceFreshnessCase(findings, gapByID, ctx); ok {
		cases = append(cases, c)
	}
	if c, ok := buildPartialFleetCase(findings, gapByID, ctx); ok {
		cases = append(cases, c)
	}
	if c, ok := buildImportedHistoricalCase(findings, gapByID, ctx); ok {
		cases = append(cases, c)
	}
	cases = append(cases, buildIncidentCases(findings, gapByID, ctx)...)
	return cases
}

func buildTransportCases(findings []Finding, gapByID map[string]EvidenceGap, ctx deriveContext) []Case {
	grouped := map[string][]Finding{}
	for _, finding := range findings {
		if !findingBelongsToTransportCase(finding) {
			continue
		}
		resource := strings.TrimSpace(finding.ResourceID)
		if resource == "" {
			continue
		}
		grouped[resource] = append(grouped[resource], finding)
	}

	names := make([]string, 0, len(grouped))
	for name := range grouped {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]Case, 0, len(names))
	for _, name := range names {
		group := grouped[name]
		top := topFinding(group)
		c := NewCase(
			"case:transport:"+name,
			CaseTransportDegradation,
			caseStatus(group, false),
			caseAttention(group),
			caseCertainty(group),
			top.Title,
			top.Explanation,
			caseUpdatedAt(group),
		)
		c.Scope = caseScope(group, ScopeLocal)
		c.AttentionReason = top.Title
		c.WhyItMatters = firstNonEmptyString(top.WhyItMatters, "Transport symptoms reduce or remove visibility through a local ingest path.")
		c.FindingIDs = collectFindingIDs(group)
		c.EvidenceGapIDs = collectCaseGapIDs(group)
		c.RecommendationIDs = collectCaseRecommendationIDs(group)
		c.RelatedRecords = transportRelatedRecords(name, ctx)
		c.SafeToConsider = fmt.Sprintf("Treat transport '%s' as a bounded local visibility problem and inspect its runtime, alerts, and event history.", name)
		c.OutOfScope = "Do not infer fleet-wide outage, RF coverage loss, or congestion from one transport's symptoms alone."
		c.MissingEvidence = summarizeMissingEvidence(c.EvidenceGapIDs, gapByID)
		c.ObservedAt = caseObservedAt(group)
		c.CurrentEvidence = true
		c.Source = "investigation"
		out = append(out, c)
	}
	return out
}

func buildEvidenceFreshnessCase(findings []Finding, gapByID map[string]EvidenceGap, ctx deriveContext) (Case, bool) {
	var selected []Finding
	for _, finding := range findings {
		switch finding.Code {
		case "no_active_ingest", "stale_evidence":
			selected = append(selected, finding)
		}
	}
	if len(selected) == 0 {
		return Case{}, false
	}

	top := topFinding(selected)
	c := NewCase(
		"case:evidence:freshness",
		CaseEvidenceFreshnessGap,
		caseStatus(selected, false),
		caseAttention(selected),
		caseCertainty(selected),
		"Current live evidence is not proven",
		top.Explanation,
		caseUpdatedAt(selected),
	)
	c.Scope = ScopeLocal
	c.AttentionReason = top.Title
	c.WhyItMatters = "Missing or stale current evidence makes silence ambiguous. The mesh may be quiet, disconnected, or simply unobserved."
	c.FindingIDs = collectFindingIDs(selected)
	c.EvidenceGapIDs = collectCaseGapIDs(selected)
	c.RecommendationIDs = collectCaseRecommendationIDs(selected)
	c.RelatedRecords = evidenceFreshnessRecords(ctx)
	c.SafeToConsider = "Treat current state as unconfirmed until fresh local evidence or active ingest is restored."
	c.OutOfScope = "Do not conclude that the mesh is healthy, quiet, or absent of problems from missing fresh evidence."
	c.MissingEvidence = summarizeMissingEvidence(c.EvidenceGapIDs, gapByID)
	c.ObservedAt = caseObservedAt(selected)
	c.CurrentEvidence = false
	return c, true
}

func buildPartialFleetCase(findings []Finding, gapByID map[string]EvidenceGap, ctx deriveContext) (Case, bool) {
	var selected []Finding
	for _, finding := range findings {
		if finding.Code == "partial_fleet_visibility" {
			selected = append(selected, finding)
		}
	}
	if len(selected) == 0 {
		return Case{}, false
	}

	top := topFinding(selected)
	c := NewCase(
		"case:fleet:partial-visibility",
		CasePartialFleetVisibility,
		caseStatus(selected, false),
		caseAttention(selected),
		caseCertainty(selected),
		top.Title,
		top.Explanation,
		caseUpdatedAt(selected),
	)
	c.Scope = ScopePartialFleet
	c.AttentionReason = top.Title
	c.WhyItMatters = firstNonEmptyString(top.WhyItMatters, "Partial fleet visibility raises attention while lowering certainty for fleet-wide conclusions.")
	c.FindingIDs = collectFindingIDs(selected)
	c.EvidenceGapIDs = collectCaseGapIDs(selected)
	c.RecommendationIDs = collectCaseRecommendationIDs(selected)
	c.RelatedRecords = partialFleetRecords(ctx)
	c.SafeToConsider = "Investigate within this instance's truth boundary and seek corroboration from missing or out-of-scope reporters."
	c.OutOfScope = "Do not claim fleet-wide health, absence, or root cause when MEL is operating with partial fleet visibility."
	c.MissingEvidence = summarizeMissingEvidence(c.EvidenceGapIDs, gapByID)
	c.ObservedAt = caseObservedAt(selected)
	c.CurrentEvidence = true
	return c, true
}

func buildImportedHistoricalCase(findings []Finding, gapByID map[string]EvidenceGap, ctx deriveContext) (Case, bool) {
	var selected []Finding
	for _, finding := range findings {
		if finding.Scope == ScopeImported || finding.Scope == ScopeHistoricalOnly || finding.Category == CategoryImport {
			selected = append(selected, finding)
		}
	}
	if len(selected) == 0 && len(ctx.imports) == 0 {
		return Case{}, false
	}

	if len(selected) == 0 {
		f := NewFinding(
			"imported_historical_context",
			CategoryImport,
			AttentionMedium,
			0.6,
			"Imported evidence is present as historical context",
			"Imported evidence exists, but no explicit imported finding was materialized.",
			time.Now().UTC(),
		)
		f.Scope = ScopeImported
		f.ObservedAt = newestImportTime(ctx.imports, time.Now().UTC().Format(time.RFC3339))
		selected = append(selected, f)
	}

	top := topFinding(selected)
	c := NewCase(
		"case:import:historical-context",
		CaseImportedHistorical,
		CaseStatusHistoricalOnly,
		caseAttention(selected),
		caseCertainty(selected),
		"Imported evidence requires historical-only handling",
		top.Explanation,
		caseUpdatedAt(selected),
	)
	c.Scope = ScopeImported
	c.HistoricalOnly = true
	c.AttentionReason = top.Title
	c.WhyItMatters = firstNonEmptyString(top.WhyItMatters, "Imported evidence can support investigation context without becoming current local proof.")
	c.FindingIDs = collectFindingIDs(selected)
	c.EvidenceGapIDs = collectCaseGapIDs(selected)
	c.RecommendationIDs = collectCaseRecommendationIDs(selected)
	c.RelatedRecords = importedRelatedRecords(ctx)
	c.SafeToConsider = "Use imported evidence as historical context, compare it against local records, and keep its origin and timing posture visible."
	c.OutOfScope = "Do not treat imported bundles as live federation, current local confirmation, or fleet-wide authority."
	c.MissingEvidence = summarizeMissingEvidence(c.EvidenceGapIDs, gapByID)
	c.ObservedAt = caseObservedAt(selected)
	c.CurrentEvidence = false
	return c, true
}

func buildIncidentCases(findings []Finding, gapByID map[string]EvidenceGap, ctx deriveContext) []Case {
	findingByIncident := make(map[string]Finding)
	for _, finding := range findings {
		if strings.HasPrefix(finding.ID, "incident:") {
			findingByIncident[strings.TrimPrefix(finding.ID, "incident:")] = finding
		}
	}

	var out []Case
	for _, inc := range ctx.incidents {
		if inc.State == "resolved" || inc.State == "suppressed" {
			continue
		}
		finding, ok := findingByIncident[inc.ID]
		if !ok {
			continue
		}
		c := NewCase(
			"case:incident:"+inc.ID,
			CaseIncidentCandidate,
			caseStatus([]Finding{finding}, false),
			finding.Attention,
			finding.Certainty,
			inc.Title,
			finding.Explanation,
			firstNonEmptyString(inc.UpdatedAt, finding.ObservedAt),
		)
		c.Scope = ScopeLocal
		c.AttentionReason = finding.Title
		c.WhyItMatters = firstNonEmptyString(finding.WhyItMatters, incidentWhyItMatters(inc))
		c.FindingIDs = []string{finding.ID}
		c.EvidenceGapIDs = append([]string(nil), finding.EvidenceGapIDs...)
		c.RecommendationIDs = append([]string(nil), finding.RecommendationIDs...)
		c.RelatedRecords = incidentRelatedRecords(inc, ctx)
		c.SafeToConsider = "Treat this incident as a bounded operator attention object backed by the stored incident record and linked evidence."
		c.OutOfScope = "Do not treat an open incident record as automatic root-cause proof."
		c.MissingEvidence = summarizeMissingEvidence(c.EvidenceGapIDs, gapByID)
		c.ObservedAt = firstNonEmptyString(inc.OccurredAt, finding.ObservedAt)
		c.CurrentEvidence = true
		out = append(out, c)
	}
	return out
}

func transportRelatedRecords(name string, ctx deriveContext) []RelatedRecord {
	out := []RelatedRecord{{
		Kind:       RecordTransportRuntime,
		ID:         name,
		Relation:   "runtime_state",
		Summary:    "Inspect transport runtime, state, counters, and guidance",
		Scope:      ScopeLocal,
		InspectCLI: fmt.Sprintf("mel inspect transport %s --config <path>", name),
		InspectAPI: "/api/v1/transports/inspect/" + name,
	}}
	for _, alert := range ctx.alerts {
		if alert.TransportName != name {
			continue
		}
		out = append(out, RelatedRecord{
			Kind:       RecordTransportAlert,
			ID:         alert.ID,
			Relation:   "alert",
			Summary:    alert.Summary,
			Scope:      ScopeLocal,
			InspectCLI: fmt.Sprintf("mel alerts --filter %s --config <path>", name),
			InspectAPI: "/api/v1/transports/alerts",
		})
	}
	for _, event := range ctx.timeline {
		if !timelineMatchesTransport(event, name) {
			continue
		}
		out = append(out, RelatedRecord{
			Kind:       RecordTimelineEvent,
			ID:         event.EventID,
			Relation:   "timeline",
			Summary:    event.Summary,
			Scope:      scopeFromTimeline(event),
			InspectCLI: fmt.Sprintf("mel timeline inspect %s --config <path>", event.EventID),
			InspectAPI: "/api/v1/timeline/" + event.EventID,
		})
		if len(out) >= 8 {
			break
		}
	}
	return dedupeRelatedRecords(out)
}

func evidenceFreshnessRecords(ctx deriveContext) []RelatedRecord {
	out := []RelatedRecord{{
		Kind:       RecordStatusSnapshot,
		ID:         "status",
		Relation:   "status_snapshot",
		Summary:    "Inspect current status and transport truth",
		Scope:      ScopeLocal,
		InspectCLI: "mel status --config <path>",
		InspectAPI: "/api/v1/status",
	}}
	for _, state := range ctx.runtimeStates {
		out = append(out, RelatedRecord{
			Kind:       RecordTransportRuntime,
			ID:         state.Name,
			Relation:   "enabled_transport",
			Summary:    fmt.Sprintf("Transport %s state=%s", state.Name, state.State),
			Scope:      ScopeLocal,
			InspectCLI: fmt.Sprintf("mel inspect transport %s --config <path>", state.Name),
			InspectAPI: "/api/v1/transports/inspect/" + state.Name,
		})
	}
	if len(ctx.timeline) > 0 {
		event := ctx.timeline[0]
		out = append(out, RelatedRecord{
			Kind:       RecordTimelineEvent,
			ID:         event.EventID,
			Relation:   "latest_timeline_evidence",
			Summary:    event.Summary,
			Scope:      scopeFromTimeline(event),
			InspectCLI: fmt.Sprintf("mel timeline inspect %s --config <path>", event.EventID),
			InspectAPI: "/api/v1/timeline/" + event.EventID,
		})
	}
	return dedupeRelatedRecords(out)
}

func partialFleetRecords(ctx deriveContext) []RelatedRecord {
	out := []RelatedRecord{{
		Kind:       RecordFleetTruth,
		ID:         ctx.truth.InstanceID,
		Relation:   "fleet_truth",
		Summary:    fmt.Sprintf("truth_posture=%s visibility=%s", ctx.truth.TruthPosture, ctx.truth.Visibility),
		Scope:      ScopePartialFleet,
		InspectCLI: "mel fleet truth --config <path>",
		InspectAPI: "/api/v1/fleet/truth",
	}}
	for _, batch := range ctx.batches {
		out = append(out, RelatedRecord{
			Kind:       RecordImportBatch,
			ID:         batch.ID,
			Relation:   "import_batch",
			Summary:    firstNonEmptyString(batch.Note, batch.SourceName),
			Scope:      ScopeImported,
			InspectCLI: fmt.Sprintf("mel fleet evidence batch-show %s --config <path>", batch.ID),
			InspectAPI: "/api/v1/fleet/imports/" + batch.ID,
		})
		if len(out) >= 5 {
			break
		}
	}
	return dedupeRelatedRecords(out)
}

func importedRelatedRecords(ctx deriveContext) []RelatedRecord {
	out := make([]RelatedRecord, 0, len(ctx.imports)+len(ctx.batches))
	for _, batch := range ctx.batches {
		out = append(out, RelatedRecord{
			Kind:       RecordImportBatch,
			ID:         batch.ID,
			Relation:   "batch",
			Summary:    firstNonEmptyString(batch.Note, batch.SourceName),
			Scope:      ScopeImported,
			InspectCLI: fmt.Sprintf("mel fleet evidence batch-show %s --config <path>", batch.ID),
			InspectAPI: "/api/v1/fleet/imports/" + batch.ID,
		})
	}
	for _, imp := range ctx.imports {
		out = append(out, RelatedRecord{
			Kind:       RecordImportedEvidence,
			ID:         imp.ID,
			Relation:   "imported_evidence",
			Summary:    fmt.Sprintf("validation=%s merge=%s", imp.ValidationStatus, firstNonEmptyString(imp.MergeDisposition, "raw_only")),
			Scope:      ScopeImported,
			InspectCLI: fmt.Sprintf("mel fleet evidence show %s --config <path>", imp.ID),
			InspectAPI: "/api/v1/fleet/remote-evidence/" + imp.ID,
		})
		if len(out) >= 10 {
			break
		}
	}
	for _, event := range ctx.timeline {
		if event.ScopePosture != "remote_imported" {
			continue
		}
		out = append(out, RelatedRecord{
			Kind:       RecordTimelineEvent,
			ID:         event.EventID,
			Relation:   "import_timeline",
			Summary:    event.Summary,
			Scope:      ScopeImported,
			InspectCLI: fmt.Sprintf("mel timeline inspect %s --config <path>", event.EventID),
			InspectAPI: "/api/v1/timeline/" + event.EventID,
		})
		if len(out) >= 12 {
			break
		}
	}
	return dedupeRelatedRecords(out)
}

func incidentRelatedRecords(inc models.Incident, ctx deriveContext) []RelatedRecord {
	out := []RelatedRecord{{
		Kind:       RecordIncident,
		ID:         inc.ID,
		Relation:   "incident",
		Summary:    inc.Summary,
		Scope:      ScopeLocal,
		InspectCLI: fmt.Sprintf("mel incident inspect %s --config <path>", inc.ID),
		InspectAPI: "/api/v1/incidents/" + inc.ID,
	}}
	for _, actionID := range append(append([]string(nil), inc.PendingActions...), inc.RecentActions...) {
		actionID = strings.TrimSpace(actionID)
		if actionID == "" {
			continue
		}
		out = append(out, RelatedRecord{
			Kind:       RecordControlAction,
			ID:         actionID,
			Relation:   "linked_control_action",
			Summary:    "Inspect linked control action context",
			Scope:      ScopeLocal,
			InspectCLI: fmt.Sprintf("mel action inspect %s --config <path>", actionID),
		})
	}
	for _, event := range ctx.timeline {
		if event.EventID != inc.ID && event.ResourceID != inc.ID {
			continue
		}
		out = append(out, RelatedRecord{
			Kind:       RecordTimelineEvent,
			ID:         event.EventID,
			Relation:   "timeline",
			Summary:    event.Summary,
			Scope:      scopeFromTimeline(event),
			InspectCLI: fmt.Sprintf("mel timeline inspect %s --config <path>", event.EventID),
			InspectAPI: "/api/v1/timeline/" + event.EventID,
		})
	}
	return dedupeRelatedRecords(out)
}

func crossLink(findings *[]Finding, gaps *[]EvidenceGap, recs *[]Recommendation) {
	gapByID := make(map[string]*EvidenceGap, len(*gaps))
	for i := range *gaps {
		gapByID[(*gaps)[i].ID] = &(*gaps)[i]
	}
	for i := range *findings {
		finding := &(*findings)[i]
		for _, gapID := range finding.EvidenceGapIDs {
			if gap, ok := gapByID[gapID]; ok {
				gap.ConstrainedFindingIDs = appendUnique(gap.ConstrainedFindingIDs, finding.ID)
			}
		}
	}
	for i := range *recs {
		rec := &(*recs)[i]
		for _, gapID := range rec.EvidenceGapIDs {
			if gap, ok := gapByID[gapID]; ok {
				gap.ConstrainedRecommendationIDs = appendUnique(gap.ConstrainedRecommendationIDs, rec.ID)
				rec.UncertaintyLimits = appendUnique(rec.UncertaintyLimits, "Limited by evidence gap: "+gap.Title)
			}
		}
	}
}

func dedupeFindings(findings []Finding) []Finding {
	index := map[string]int{}
	out := make([]Finding, 0, len(findings))
	for _, finding := range findings {
		if idx, ok := index[finding.ID]; ok {
			out[idx] = mergeFinding(out[idx], finding)
			continue
		}
		index[finding.ID] = len(out)
		out = append(out, finding)
	}
	return out
}

func dedupeGaps(gaps []EvidenceGap) []EvidenceGap {
	index := map[string]int{}
	out := make([]EvidenceGap, 0, len(gaps))
	for _, gap := range gaps {
		if idx, ok := index[gap.ID]; ok {
			out[idx] = mergeGap(out[idx], gap)
			continue
		}
		index[gap.ID] = len(out)
		out = append(out, gap)
	}
	return out
}

func dedupeRecommendations(recs []Recommendation) []Recommendation {
	index := map[string]int{}
	out := make([]Recommendation, 0, len(recs))
	for _, rec := range recs {
		if idx, ok := index[rec.ID]; ok {
			out[idx] = mergeRecommendation(out[idx], rec)
			continue
		}
		index[rec.ID] = len(out)
		out = append(out, rec)
	}
	return out
}

func mergeFinding(base, incoming Finding) Finding {
	if attentionPriority(incoming.Attention) < attentionPriority(base.Attention) {
		base.Attention = incoming.Attention
	}
	if incoming.Certainty < base.Certainty {
		base.Certainty = incoming.Certainty
	}
	base.EvidenceIDs = uniqueStrings(append(base.EvidenceIDs, incoming.EvidenceIDs...))
	base.EvidenceGapIDs = uniqueStrings(append(base.EvidenceGapIDs, incoming.EvidenceGapIDs...))
	base.RecommendationIDs = uniqueStrings(append(base.RecommendationIDs, incoming.RecommendationIDs...))
	base.OperatorActionRequired = base.OperatorActionRequired || incoming.OperatorActionRequired
	base.CanAutoRecover = base.CanAutoRecover || incoming.CanAutoRecover
	base.ObservedAt = oldestTimestamp(base.ObservedAt, incoming.ObservedAt)
	base.GeneratedAt = newestTimestamp(base.GeneratedAt, incoming.GeneratedAt)
	if base.ResourceID == "" {
		base.ResourceID = incoming.ResourceID
	}
	if base.WhyItMatters == "" {
		base.WhyItMatters = incoming.WhyItMatters
	}
	if base.EvidenceSnapshot == nil && incoming.EvidenceSnapshot != nil {
		base.EvidenceSnapshot = incoming.EvidenceSnapshot
	}
	return base
}

func mergeGap(base, incoming EvidenceGap) EvidenceGap {
	base.ConstrainedFindingIDs = uniqueStrings(append(base.ConstrainedFindingIDs, incoming.ConstrainedFindingIDs...))
	base.ConstrainedRecommendationIDs = uniqueStrings(append(base.ConstrainedRecommendationIDs, incoming.ConstrainedRecommendationIDs...))
	base.GeneratedAt = newestTimestamp(base.GeneratedAt, incoming.GeneratedAt)
	if base.Impact == "" {
		base.Impact = incoming.Impact
	}
	return base
}

func mergeRecommendation(base, incoming Recommendation) Recommendation {
	base.EvidenceBasis = uniqueStrings(append(base.EvidenceBasis, incoming.EvidenceBasis...))
	base.FindingIDs = uniqueStrings(append(base.FindingIDs, incoming.FindingIDs...))
	base.UncertaintyLimits = uniqueStrings(append(base.UncertaintyLimits, incoming.UncertaintyLimits...))
	base.EvidenceGapIDs = uniqueStrings(append(base.EvidenceGapIDs, incoming.EvidenceGapIDs...))
	base.GeneratedAt = newestTimestamp(base.GeneratedAt, incoming.GeneratedAt)
	if base.Action == "" {
		base.Action = incoming.Action
	}
	if base.Rationale == "" {
		base.Rationale = incoming.Rationale
	}
	return base
}

func sortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		pi := attentionPriority(findings[i].Attention)
		pj := attentionPriority(findings[j].Attention)
		if pi != pj {
			return pi < pj
		}
		if findings[i].Certainty != findings[j].Certainty {
			return findings[i].Certainty > findings[j].Certainty
		}
		return findings[i].ID < findings[j].ID
	})
}

func sortCases(cases []Case) {
	sort.SliceStable(cases, func(i, j int) bool {
		pi := attentionPriority(cases[i].Attention)
		pj := attentionPriority(cases[j].Attention)
		if pi != pj {
			return pi < pj
		}
		if cases[i].Status != cases[j].Status {
			return cases[i].Status < cases[j].Status
		}
		if cases[i].Certainty != cases[j].Certainty {
			return cases[i].Certainty < cases[j].Certainty
		}
		return cases[i].ID < cases[j].ID
	})
}

func computeCounts(findings []Finding, gaps []EvidenceGap, recs []Recommendation) SummaryCounts {
	counts := SummaryCounts{
		TotalFindings:   len(findings),
		EvidenceGaps:    len(gaps),
		Recommendations: len(recs),
	}
	for _, finding := range findings {
		switch finding.Attention {
		case AttentionCritical:
			counts.CriticalFindings++
		case AttentionHigh:
			counts.HighFindings++
		case AttentionMedium:
			counts.MediumFindings++
		case AttentionLow:
			counts.LowFindings++
		case AttentionInfo:
			counts.InfoFindings++
		}
		if finding.OperatorActionRequired {
			counts.OperatorActionRequired++
		}
		if len(finding.EvidenceGapIDs) > 0 {
			counts.FindingsConstrainedByGaps++
		}
	}
	for _, rec := range recs {
		if len(rec.UncertaintyLimits) > 0 {
			counts.RecommendationsWithCaveats++
		}
	}
	return counts
}

func computeOverallAttention(findings []Finding) AttentionLevel {
	if len(findings) == 0 {
		return AttentionInfo
	}
	best := AttentionInfo
	for _, finding := range findings {
		if attentionPriority(finding.Attention) < attentionPriority(best) {
			best = finding.Attention
		}
	}
	return best
}

func computeOverallCertainty(findings []Finding) float64 {
	if len(findings) == 0 {
		return 1.0
	}
	min := 1.0
	seen := false
	for _, finding := range findings {
		if finding.Attention == AttentionCritical || finding.Attention == AttentionHigh {
			seen = true
			if finding.Certainty < min {
				min = finding.Certainty
			}
		}
	}
	if !seen {
		for _, finding := range findings {
			if finding.Certainty < min {
				min = finding.Certainty
			}
		}
	}
	return min
}

func computeScopePosture(findings []Finding) string {
	hasLocal := false
	hasImported := false
	hasPartial := false
	for _, finding := range findings {
		switch finding.Scope {
		case ScopeImported, ScopeHistoricalOnly:
			hasImported = true
		case ScopePartialFleet:
			hasPartial = true
		default:
			hasLocal = true
		}
	}
	switch {
	case hasPartial && hasImported:
		return "partial_fleet_with_imported_context"
	case hasPartial:
		return "partial_fleet"
	case hasImported && hasLocal:
		return "mixed_local_and_imported"
	case hasImported:
		return "imported_context_only"
	default:
		return "local_only"
	}
}

func buildHeadline(counts SummaryCounts, attention AttentionLevel, certainty float64) string {
	if counts.TotalFindings == 0 {
		return "No active findings. Available evidence does not justify an operator attention object right now."
	}

	var parts []string
	if counts.CriticalFindings > 0 {
		parts = append(parts, fmt.Sprintf("%d critical", counts.CriticalFindings))
	}
	if counts.HighFindings > 0 {
		parts = append(parts, fmt.Sprintf("%d high", counts.HighFindings))
	}
	if counts.MediumFindings > 0 {
		parts = append(parts, fmt.Sprintf("%d medium", counts.MediumFindings))
	}

	headline := fmt.Sprintf("%d finding(s)", counts.TotalFindings)
	if len(parts) > 0 {
		headline += " (" + strings.Join(parts, ", ") + ")"
	}
	if counts.EvidenceGaps > 0 {
		headline += fmt.Sprintf(" with %d evidence gap(s) limiting certainty", counts.EvidenceGaps)
	}
	switch {
	case attention == AttentionCritical && certainty < 0.8:
		headline += ". Operator attention is HIGH while certainty remains bounded."
	case certainty < 0.5:
		headline += ". Certainty is LOW; investigate missing evidence before acting."
	case certainty < 0.8:
		headline += ". Some conclusions remain evidence-bounded."
	}
	return headline
}

func buildAttentionSummary(attention AttentionLevel, certainty float64, counts SummaryCounts) string {
	if counts.TotalFindings == 0 {
		return "MEL has no active findings to report. Evidence coverage may still be partial; check physics boundaries and current ingest posture."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("MEL assembled %d finding(s) that may deserve operator attention. ", counts.TotalFindings))
	if counts.OperatorActionRequired > 0 {
		sb.WriteString(fmt.Sprintf("%d finding(s) currently suggest operator investigation or verification. ", counts.OperatorActionRequired))
	}
	switch {
	case attention == AttentionCritical && certainty < 0.8:
		sb.WriteString("Attention is high because the observed condition matters operationally, not because MEL has proven a root cause. ")
	case certainty < 0.5:
		sb.WriteString("Certainty is LOW. Evidence gaps are first-class and should be investigated before acting on recommendations. ")
	case certainty < 0.8:
		sb.WriteString("Some findings have limited certainty due to missing, stale, or imported-only evidence. ")
	}
	if counts.RecommendationsWithCaveats > 0 {
		sb.WriteString("Recommendations include uncertainty limits where MEL cannot safely conclude more.")
	}
	return strings.TrimSpace(sb.String())
}

func diagnosticWhyItMatters(diag diagnostics.Finding) string {
	switch strings.ToLower(diag.Component) {
	case diagnostics.ComponentTransport:
		return "Transport diagnostics affect how much current mesh evidence MEL can trust."
	case diagnostics.ComponentDatabase:
		return "Database diagnostics affect whether evidence is being stored and retrievable."
	case diagnostics.ComponentMesh:
		return "Mesh diagnostics highlight what MEL is observing, not what it can prove about RF coverage."
	default:
		return "Diagnostic findings explain what MEL is observing and where the operator should look next."
	}
}

func incidentWhyItMatters(inc models.Incident) string {
	if note, ok := inc.Metadata["evidence_note"].(string); ok && strings.TrimSpace(note) != "" {
		return strings.TrimSpace(note)
	}
	return "An open incident is a durable attention object for operator investigation, not automatic diagnosis."
}

func mapDiagSeverity(sev string) AttentionLevel {
	switch strings.ToLower(sev) {
	case "critical":
		return AttentionCritical
	case "warning":
		return AttentionHigh
	case "info":
		return AttentionInfo
	default:
		return AttentionMedium
	}
}

func mapDiagCategory(component string) FindingCategory {
	switch strings.ToLower(component) {
	case "transport":
		return CategoryTransport
	case "database":
		return CategoryDatabase
	case "config":
		return CategoryConfig
	case "mesh":
		return CategoryMesh
	case "storage":
		return CategoryStorage
	case "retention":
		return CategoryRetention
	default:
		return CategoryConfig
	}
}

func mapIncidentCategory(cat string) FindingCategory {
	switch strings.ToLower(cat) {
	case "transport":
		return CategoryTransport
	case "mesh", "mesh_topology":
		return CategoryMesh
	case "ingest", "evidence":
		return CategoryStorage
	case "control":
		return CategoryControl
	default:
		return CategoryConfig
	}
}

func mapIncidentSeverity(sev string) AttentionLevel {
	switch strings.ToLower(sev) {
	case "critical":
		return AttentionCritical
	case "high":
		return AttentionHigh
	case "medium", "warning":
		return AttentionMedium
	case "low":
		return AttentionLow
	default:
		return AttentionMedium
	}
}

func findingBelongsToTransportCase(finding Finding) bool {
	return finding.Source != "incidents" &&
		finding.Category == CategoryTransport &&
		strings.TrimSpace(finding.ResourceID) != ""
}

func collectFindingIDs(findings []Finding) []string {
	out := make([]string, 0, len(findings))
	for _, finding := range findings {
		out = append(out, finding.ID)
	}
	return uniqueStrings(out)
}

func collectCaseGapIDs(findings []Finding) []string {
	var out []string
	for _, finding := range findings {
		out = append(out, finding.EvidenceGapIDs...)
	}
	return uniqueStrings(out)
}

func collectCaseRecommendationIDs(findings []Finding) []string {
	var out []string
	for _, finding := range findings {
		out = append(out, finding.RecommendationIDs...)
	}
	return uniqueStrings(out)
}

func caseAttention(findings []Finding) AttentionLevel {
	attention := AttentionInfo
	for _, finding := range findings {
		if attentionPriority(finding.Attention) < attentionPriority(attention) {
			attention = finding.Attention
		}
	}
	return attention
}

func caseCertainty(findings []Finding) float64 {
	if len(findings) == 0 {
		return 1.0
	}
	certainty := 1.0
	for _, finding := range findings {
		if finding.Certainty < certainty {
			certainty = finding.Certainty
		}
	}
	return certainty
}

func caseStatus(findings []Finding, historicalOnly bool) CaseStatus {
	if historicalOnly {
		return CaseStatusHistoricalOnly
	}
	attention := caseAttention(findings)
	if attention == AttentionCritical || attention == AttentionHigh {
		return CaseStatusActiveAttention
	}
	return CaseStatusMonitoring
}

func caseScope(findings []Finding, fallback ScopePosture) ScopePosture {
	scope := fallback
	for _, finding := range findings {
		if finding.Scope == ScopePartialFleet {
			return ScopePartialFleet
		}
		if finding.Scope == ScopeImported || finding.Scope == ScopeHistoricalOnly {
			scope = finding.Scope
		}
	}
	return scope
}

func caseObservedAt(findings []Finding) string {
	oldest := ""
	for _, finding := range findings {
		oldest = oldestTimestamp(oldest, finding.ObservedAt)
	}
	return oldest
}

func caseUpdatedAt(findings []Finding) string {
	newest := ""
	for _, finding := range findings {
		newest = newestTimestamp(newest, firstNonEmptyString(finding.GeneratedAt, finding.ObservedAt))
	}
	return newest
}

func topFinding(findings []Finding) Finding {
	if len(findings) == 0 {
		return Finding{}
	}
	sorted := append([]Finding(nil), findings...)
	sortFindings(sorted)
	return sorted[0]
}

func summarizeMissingEvidence(ids []string, gapByID map[string]EvidenceGap) string {
	titles := make([]string, 0, len(ids))
	for _, id := range ids {
		if gap, ok := gapByID[id]; ok {
			titles = append(titles, gap.Title)
		}
	}
	titles = uniqueStrings(titles)
	if len(titles) == 0 {
		return ""
	}
	return strings.Join(titles, "; ")
}

func dedupeRelatedRecords(records []RelatedRecord) []RelatedRecord {
	index := map[string]int{}
	out := make([]RelatedRecord, 0, len(records))
	for _, record := range records {
		key := string(record.Kind) + ":" + record.ID + ":" + record.Relation
		if _, ok := index[key]; ok {
			continue
		}
		index[key] = len(out)
		out = append(out, record)
	}
	return out
}

func attentionPriority(level AttentionLevel) int {
	switch level {
	case AttentionCritical:
		return 0
	case AttentionHigh:
		return 1
	case AttentionMedium:
		return 2
	case AttentionLow:
		return 3
	default:
		return 4
	}
}

func alertIDs(alerts []db.TransportAlertRecord) []string {
	out := make([]string, 0, len(alerts))
	for _, alert := range alerts {
		out = append(out, alert.ID)
	}
	return out
}

func alertReasons(alerts []db.TransportAlertRecord) []string {
	out := make([]string, 0, len(alerts))
	for _, alert := range alerts {
		out = append(out, alert.Reason)
	}
	return uniqueStrings(out)
}

func newestAlertUpdate(alerts []db.TransportAlertRecord, fallback string) string {
	out := fallback
	for _, alert := range alerts {
		out = newestTimestamp(out, alert.LastUpdatedAt)
	}
	return out
}

func importIDs(imports []db.ImportedRemoteEvidenceRecord) []string {
	out := make([]string, 0, len(imports))
	for _, imp := range imports {
		out = append(out, imp.ID)
	}
	return out
}

func newestImportTime(imports []db.ImportedRemoteEvidenceRecord, fallback string) string {
	out := fallback
	for _, imp := range imports {
		out = newestTimestamp(out, imp.ImportedAt)
	}
	return out
}

func hasLocalTimelineEvidence(timeline []db.TimelineEvent) bool {
	for _, event := range timeline {
		if event.ScopePosture != "remote_imported" {
			return true
		}
	}
	return false
}

func timelineMatchesTransport(event db.TimelineEvent, transportName string) bool {
	if strings.EqualFold(event.ResourceID, transportName) {
		return true
	}
	for _, key := range []string{"transport", "transport_name", "name"} {
		if strings.EqualFold(detailsString(event.Details, key), transportName) {
			return true
		}
	}
	return false
}

func detailsString(details map[string]any, key string) string {
	if details == nil {
		return ""
	}
	value, ok := details[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func scopeFromTimeline(event db.TimelineEvent) ScopePosture {
	switch strings.TrimSpace(event.ScopePosture) {
	case "remote_imported":
		return ScopeImported
	case "best_effort_fleet":
		return ScopePartialFleet
	default:
		return ScopeLocal
	}
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func oldestTimestamp(current, candidate string) string {
	if candidate == "" {
		return current
	}
	if current == "" {
		return candidate
	}
	currentTime, currentErr := time.Parse(time.RFC3339, current)
	candidateTime, candidateErr := time.Parse(time.RFC3339, candidate)
	if currentErr != nil || candidateErr != nil {
		if candidate < current {
			return candidate
		}
		return current
	}
	if candidateTime.Before(currentTime) {
		return candidate
	}
	return current
}

func mustParseTime(value string) time.Time {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Now().UTC()
	}
	return t
}
