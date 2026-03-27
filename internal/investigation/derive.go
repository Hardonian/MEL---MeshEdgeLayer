package investigation

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/diagnostics"
	"github.com/mel-project/mel/internal/transport"
)

// Derive assembles a canonical investigation Summary from current system
// state. It reads from real sources — diagnostics, transport status, incidents,
// fleet posture, and timeline — and produces findings, evidence gaps, and
// evidence-constrained recommendations.
//
// This function is deterministic for a given input state. It does not cache,
// does not persist, and does not mutate state. Each invocation produces a
// fresh view.
func Derive(
	cfg config.Config,
	d *db.DB,
	runtimeTransports []transport.Health,
	transportStates []db.TransportRuntime,
	now time.Time,
) Summary {
	nowStr := now.UTC().Format(time.RFC3339)

	var findings []Finding
	var gaps []EvidenceGap
	var recs []Recommendation

	// ─── Source 1: Diagnostics ───
	diagRun := diagnostics.RunAllChecks(cfg, d, runtimeTransports, transportStates, now)
	diagFindings, diagGaps, diagRecs := deriveDiagnostics(diagRun.Diagnostics, nowStr)
	findings = append(findings, diagFindings...)
	gaps = append(gaps, diagGaps...)
	recs = append(recs, diagRecs...)

	// ─── Source 2: Transport state ───
	tFindings, tGaps, tRecs := deriveTransportState(cfg, runtimeTransports, transportStates, d, nowStr)
	findings = append(findings, tFindings...)
	gaps = append(gaps, tGaps...)
	recs = append(recs, tRecs...)

	// ─── Source 3: Incidents ───
	iFindings, iGaps, iRecs := deriveIncidents(d, nowStr)
	findings = append(findings, iFindings...)
	gaps = append(gaps, iGaps...)
	recs = append(recs, iRecs...)

	// ─── Source 4: Fleet evidence posture ───
	fFindings, fGaps, fRecs := deriveFleetPosture(cfg, d, nowStr)
	findings = append(findings, fFindings...)
	gaps = append(gaps, fGaps...)
	recs = append(recs, fRecs...)

	// ─── Source 5: Stale evidence ───
	sFindings, sGaps, sRecs := deriveStaleness(d, nowStr, now)
	findings = append(findings, sFindings...)
	gaps = append(gaps, sGaps...)
	recs = append(recs, sRecs...)

	// ─── Cross-link findings ↔ gaps ↔ recommendations ───
	crossLink(&findings, &gaps, &recs)

	// ─── Sort by attention then certainty ───
	sortFindings(findings)

	// ─── Compute counts and summary ───
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
		Counts:           counts,
		ScopePosture:     scopePosture,
		PhysicsBoundary:  DefaultPhysicsBoundary(),
	}
}

// deriveDiagnostics converts diagnostics.Finding into investigation.Finding.
func deriveDiagnostics(diags []diagnostics.Finding, nowStr string) ([]Finding, []EvidenceGap, []Recommendation) {
	var findings []Finding
	var gaps []EvidenceGap
	var recs []Recommendation

	for _, diag := range diags {
		attention := mapDiagSeverity(diag.Severity)
		certainty := 0.8 // diagnostics are local observations; fairly certain
		if diag.Severity == "info" {
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
		f.ResourceID = diag.AffectedTransport

		if diag.Severity == "critical" || diag.Severity == "warning" {
			f.OperatorActionRequired = true
		}

		// Evidence snapshot from diagnostic data
		snapshot := make(map[string]any)
		if diag.AffectedTransport != "" {
			snapshot["resource"] = diag.AffectedTransport
		}
		if diag.Component != "" {
			snapshot["component"] = diag.Component
		}
		if len(diag.Evidence) > 0 {
			snapshot["diagnostic_evidence"] = diag.Evidence
		}
		if len(snapshot) > 0 {
			f.EvidenceSnapshot = snapshot
		}

		// Recommendation for actionable diagnostics
		if diag.Severity == "critical" || diag.Severity == "warning" {
			rec := NewRecommendation(
				RecRunDiagnostics,
				fmt.Sprintf("Investigate diagnostic finding: %s", diag.Title),
				fmt.Sprintf("Diagnostic check '%s' reported %s severity for %s: %s",
					diag.Code, diag.Severity, diag.Component, diag.Explanation),
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

	return findings, gaps, recs
}

// deriveTransportState produces findings for transport health.
func deriveTransportState(cfg config.Config, runtimeTransports []transport.Health, transportStates []db.TransportRuntime, d *db.DB, nowStr string) ([]Finding, []EvidenceGap, []Recommendation) {
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
		f.WhyItMatters = "Without enabled transports, MEL cannot ingest data from the mesh."
		f.Source = "transport_state"
		f.ObservedAt = nowStr
		findings = append(findings, f)
		return findings, gaps, recs
	}

	ingestingCount := 0
	failedTransports := []string{}
	for _, tr := range runtimeTransports {
		if tr.State == transport.StateIngesting {
			ingestingCount++
		} else if tr.State == transport.StateFailed || tr.State == transport.StateError {
			failedTransports = append(failedTransports, tr.Name)
		}
	}

	if ingestingCount == 0 && enabledCount > 0 {
		f := NewFinding(
			"no_active_ingest",
			CategoryTransport,
			AttentionHigh,
			0.9,
			"No transport is actively ingesting",
			fmt.Sprintf("%d transport(s) are enabled but none are in ingesting state. MEL is not receiving live mesh data.", enabledCount),
			mustParseTime(nowStr),
		)
		f.WhyItMatters = "Without active ingest, all evidence becomes stale and operator decisions are based on historical data only."
		f.Source = "transport_state"
		f.ObservedAt = nowStr
		f.OperatorActionRequired = true
		findings = append(findings, f)

		rec := NewRecommendation(
			RecInspectTransport,
			"Verify transport connectivity and configuration",
			"All enabled transports are idle or disconnected. Check physical connections, broker availability, and transport configuration.",
			"operator_only",
			ScopeLocal,
			nowStr,
		)
		rec.ID = "rec_verify_transports"
		rec.FindingIDs = []string{f.ID}
		recs = append(recs, rec)
		f.RecommendationIDs = append(f.RecommendationIDs, rec.ID)

		gap := NewEvidenceGap(
			GapMissingExpectedReporters,
			"No active transport reporters",
			"No transport is providing live data. All findings that depend on fresh telemetry are degraded.",
			"Certainty on mesh health, node status, and message flow is reduced to zero until ingest resumes.",
			ScopeLocal,
			nowStr,
		)
		gaps = append(gaps, gap)
	}

	for _, name := range failedTransports {
		f := NewFinding(
			"transport_failed",
			CategoryTransport,
			AttentionCritical,
			0.95,
			fmt.Sprintf("Transport '%s' is in failed state", name),
			fmt.Sprintf("Transport '%s' has entered a failure state. Check connection and review dead letters for this transport.", name),
			mustParseTime(nowStr),
		)
		f.WhyItMatters = "A failed transport means partial or total loss of mesh visibility through that path."
		f.ID = "transport_failed:" + name
		f.ResourceID = name
		f.Source = "transport_state"
		f.ObservedAt = nowStr
		f.OperatorActionRequired = true
		findings = append(findings, f)

		rec := NewRecommendation(
			RecInspectTransport,
			fmt.Sprintf("Investigate failed transport '%s'", name),
			fmt.Sprintf("Transport '%s' is in failed state. Check mel alerts, mel inspect transport %s, and dead letters for root cause.", name, name),
			"operator_only",
			ScopeLocal,
			nowStr,
		)
		rec.ID = "rec_inspect_" + name
		rec.FindingIDs = []string{f.ID}
		recs = append(recs, rec)
		f.RecommendationIDs = append(f.RecommendationIDs, rec.ID)
	}

	// Check for active alerts via DB
	alertCounts, err := d.ActiveAlertCounts()
	if err == nil {
		for tName, count := range alertCounts {
			if count > 0 {
				f := NewFinding(
					"transport_alerts_active",
					CategoryTransport,
					AttentionHigh,
					0.85,
					fmt.Sprintf("Transport '%s' has %d active alert(s)", tName, count),
					fmt.Sprintf("Transport '%s' has %d active alerts. Use `mel alerts --filter %s` to inspect.", tName, count, tName),
					mustParseTime(nowStr),
				)
				f.ID = "transport_alerts:" + tName
				f.ResourceID = tName
				f.Source = "transport_state"
				f.ObservedAt = nowStr
				f.EvidenceSnapshot = map[string]any{"active_alert_count": count}
				findings = append(findings, f)
			}
		}
	}

	return findings, gaps, recs
}

// deriveIncidents produces findings from recent open incidents.
func deriveIncidents(d *db.DB, nowStr string) ([]Finding, []EvidenceGap, []Recommendation) {
	var findings []Finding
	var gaps []EvidenceGap
	var recs []Recommendation

	incidents, err := d.RecentIncidents(25)
	if err != nil {
		return findings, gaps, recs
	}

	openCount := 0
	for _, inc := range incidents {
		if inc.State != "resolved" && inc.State != "suppressed" {
			openCount++
			f := NewFinding(
				"open_incident",
				mapIncidentCategory(inc.Category),
				mapIncidentSeverity(inc.Severity),
				0.7, // Incidents are observations that may need investigation
				fmt.Sprintf("Open incident: %s", inc.Summary),
				fmt.Sprintf("Incident %s (category=%s, severity=%s): %s",
					inc.ID, inc.Category, inc.Severity, inc.Summary),
				mustParseTime(nowStr),
			)
			f.ID = "incident:" + inc.ID
			f.ResourceID = inc.ID
			f.Source = "incidents"
			f.ObservedAt = inc.OccurredAt
			f.OperatorActionRequired = true
			f.EvidenceSnapshot = map[string]any{
				"incident_id": inc.ID,
				"category":    inc.Category,
				"severity":    inc.Severity,
				"state":       inc.State,
			}

			rec := NewRecommendation(
				RecRunDiagnostics,
				fmt.Sprintf("Investigate incident %s: %s", inc.ID, inc.Summary),
				fmt.Sprintf("Open incident of %s severity in %s category. Use `mel incident inspect %s` for details.", inc.Severity, inc.Category, inc.ID),
				"operator_only",
				ScopeLocal,
				nowStr,
			)
			rec.ID = "rec_incident_" + inc.ID
			rec.FindingIDs = []string{f.ID}
			recs = append(recs, rec)
			f.RecommendationIDs = append(f.RecommendationIDs, rec.ID)

			findings = append(findings, f)
		}
	}

	return findings, gaps, recs
}

// deriveFleetPosture produces findings from fleet evidence posture.
func deriveFleetPosture(cfg config.Config, d *db.DB, nowStr string) ([]Finding, []EvidenceGap, []Recommendation) {
	var findings []Finding
	var gaps []EvidenceGap
	var recs []Recommendation

	imports, err := d.ListImportedRemoteEvidence(100)
	if err != nil || len(imports) == 0 {
		return findings, gaps, recs
	}

	// Check for unverified imports
	unverifiedCount := 0
	for _, imp := range imports {
		if imp.ValidationStatus != "accepted" {
			unverifiedCount++
		}
	}

	if unverifiedCount > 0 {
		f := NewFinding(
			"fleet_unverified_imports",
			CategoryImport,
			AttentionMedium,
			0.6,
			fmt.Sprintf("%d imported evidence item(s) have caveats or were rejected", unverifiedCount),
			fmt.Sprintf("Out of %d imported remote evidence items, %d have validation caveats or were rejected. Use `mel fleet evidence list` to inspect.",
				len(imports), unverifiedCount),
			mustParseTime(nowStr),
		)
		f.WhyItMatters = "Imported evidence with caveats reduces certainty on fleet-wide conclusions."
		f.Source = "fleet_posture"
		f.ObservedAt = nowStr
		f.Scope = ScopeImported
		findings = append(findings, f)

		gap := NewEvidenceGap(
			GapAuthenticityUnverified,
			"Remote imports have validation caveats",
			fmt.Sprintf("%d imported evidence items were not fully validated. Conclusions based on imported evidence carry additional uncertainty.", unverifiedCount),
			"Fleet-wide conclusions that rely on imported evidence should be treated as lower certainty.",
			ScopeImported,
			nowStr,
		)
		gaps = append(gaps, gap)
	}

	// All imported evidence carries a general evidence gap
	if len(imports) > 0 {
		gap := NewEvidenceGap(
			GapImportedHistoricalOnly,
			"Imported evidence is historical only",
			fmt.Sprintf("MEL has %d imported remote evidence items. These are from offline bundles, not live federation.", len(imports)),
			"Imported evidence provides historical context but does not represent current state. Do not treat as live confirmation.",
			ScopeImported,
			nowStr,
		)
		gaps = append(gaps, gap)
	}

	return findings, gaps, recs
}

// deriveStaleness detects stale evidence.
func deriveStaleness(d *db.DB, nowStr string, now time.Time) ([]Finding, []EvidenceGap, []Recommendation) {
	var findings []Finding
	var gaps []EvidenceGap
	var recs []Recommendation

	events, err := d.TimelineEvents("", "", 1)
	if err != nil || len(events) == 0 {
		return findings, gaps, recs
	}

	lastEvent := events[0]
	lastTime, err := time.Parse(time.RFC3339, lastEvent.EventTime)
	if err == nil && now.Sub(lastTime) > 24*time.Hour {
		staleDuration := now.Sub(lastTime).Round(time.Hour)
		f := NewFinding(
			"stale_evidence",
			CategoryStorage,
			AttentionHigh,
			0.9,
			fmt.Sprintf("Last evidence is %s old", staleDuration),
			fmt.Sprintf("The most recent timeline event is from %s (%s ago). No new evidence has been recorded since then.",
				lastEvent.EventTime, staleDuration),
			mustParseTime(nowStr),
		)
		f.WhyItMatters = "All findings and recommendations based on this evidence may be outdated."
		f.Source = "staleness"
		f.ObservedAt = nowStr
		f.OperatorActionRequired = true
		findings = append(findings, f)

		gap := NewEvidenceGap(
			GapStaleContributors,
			"Evidence is stale",
			fmt.Sprintf("Last evidence was recorded %s ago. All observations are based on data that may no longer reflect current system state.", staleDuration),
			"All findings carry reduced certainty. Fresh evidence is needed before acting on recommendations.",
			ScopeLocal,
			nowStr,
		)
		gaps = append(gaps, gap)

		rec := NewRecommendation(
			RecWaitForFreshEvidence,
			"Verify evidence pipeline is functioning",
			"Evidence has gone stale. Check that transports are connected and delivering data. Use `mel status` and `mel doctor` to verify.",
			"operator_only",
			ScopeLocal,
			nowStr,
		)
		rec.ID = "rec_stale_evidence"
		rec.FindingIDs = []string{f.ID}
		rec.EvidenceGapIDs = []string{gap.ID}
		recs = append(recs, rec)
	}

	return findings, gaps, recs
}

// crossLink establishes bidirectional references between findings, gaps, and recommendations.
func crossLink(findings *[]Finding, gaps *[]EvidenceGap, recs *[]Recommendation) {
	gapByID := make(map[string]*EvidenceGap)
	for i := range *gaps {
		gapByID[(*gaps)[i].ID] = &(*gaps)[i]
	}
	for i := range *findings {
		f := &(*findings)[i]
		for _, gID := range f.EvidenceGapIDs {
			if g, ok := gapByID[gID]; ok {
				g.ConstrainedFindingIDs = appendUnique(g.ConstrainedFindingIDs, f.ID)
			}
		}
	}
	for i := range *recs {
		r := &(*recs)[i]
		for _, gID := range r.EvidenceGapIDs {
			if g, ok := gapByID[gID]; ok {
				g.ConstrainedRecommendationIDs = appendUnique(g.ConstrainedRecommendationIDs, r.ID)
				// Add uncertainty caveat to recommendation
				r.UncertaintyLimits = appendUnique(r.UncertaintyLimits,
					fmt.Sprintf("Limited by evidence gap: %s", g.Title))
			}
		}
	}
}

// ─── Helpers ───

func sortFindings(findings []Finding) {
	priority := map[AttentionLevel]int{
		AttentionCritical: 0,
		AttentionHigh:     1,
		AttentionMedium:   2,
		AttentionLow:      3,
		AttentionInfo:     4,
	}
	sort.SliceStable(findings, func(i, j int) bool {
		pi := priority[findings[i].Attention]
		pj := priority[findings[j].Attention]
		if pi != pj {
			return pi < pj
		}
		// Higher certainty first within same attention level
		return findings[i].Certainty > findings[j].Certainty
	})
}

func computeCounts(findings []Finding, gaps []EvidenceGap, recs []Recommendation) SummaryCounts {
	c := SummaryCounts{
		TotalFindings:   len(findings),
		EvidenceGaps:    len(gaps),
		Recommendations: len(recs),
	}
	for _, f := range findings {
		switch f.Attention {
		case AttentionCritical:
			c.CriticalFindings++
		case AttentionHigh:
			c.HighFindings++
		case AttentionMedium:
			c.MediumFindings++
		case AttentionLow:
			c.LowFindings++
		case AttentionInfo:
			c.InfoFindings++
		}
		if f.OperatorActionRequired {
			c.OperatorActionRequired++
		}
		if len(f.EvidenceGapIDs) > 0 {
			c.FindingsConstrainedByGaps++
		}
	}
	for _, r := range recs {
		if len(r.UncertaintyLimits) > 0 {
			c.RecommendationsWithCaveats++
		}
	}
	return c
}

func computeOverallAttention(findings []Finding) AttentionLevel {
	if len(findings) == 0 {
		return AttentionInfo
	}
	best := AttentionInfo
	priority := map[AttentionLevel]int{
		AttentionCritical: 0,
		AttentionHigh:     1,
		AttentionMedium:   2,
		AttentionLow:      3,
		AttentionInfo:     4,
	}
	for _, f := range findings {
		if priority[f.Attention] < priority[best] {
			best = f.Attention
		}
	}
	return best
}

func computeOverallCertainty(findings []Finding) float64 {
	min := 1.0
	found := false
	for _, f := range findings {
		if f.Attention == AttentionCritical || f.Attention == AttentionHigh {
			found = true
			if f.Certainty < min {
				min = f.Certainty
			}
		}
	}
	if !found {
		return 1.0
	}
	return min
}

func computeScopePosture(findings []Finding) string {
	hasImported := false
	for _, f := range findings {
		if f.Scope == ScopeImported || f.Scope == ScopePartialFleet {
			hasImported = true
		}
	}
	if hasImported {
		return "mixed_local_and_imported"
	}
	return "local_only"
}

func buildHeadline(counts SummaryCounts, attention AttentionLevel, certainty float64) string {
	if counts.TotalFindings == 0 {
		return "No active findings. System is nominal from available evidence."
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
	if certainty < 0.5 {
		headline += ". Certainty is LOW — investigate before acting."
	} else if certainty < 0.8 {
		headline += ". Some conclusions have limited certainty."
	}

	return headline
}

func buildAttentionSummary(attention AttentionLevel, certainty float64, counts SummaryCounts) string {
	if counts.TotalFindings == 0 {
		return "MEL has no active findings to report. Evidence coverage may be limited — check evidence gaps."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("MEL has identified %d finding(s) requiring operator attention. ", counts.TotalFindings))

	if counts.OperatorActionRequired > 0 {
		sb.WriteString(fmt.Sprintf("%d require operator action. ", counts.OperatorActionRequired))
	}

	if certainty < 0.5 {
		sb.WriteString("IMPORTANT: Certainty is LOW. Evidence gaps mean MEL cannot draw firm conclusions. Investigate the evidence gaps before acting on recommendations.")
	} else if certainty < 0.8 {
		sb.WriteString("Some findings have limited certainty due to evidence gaps. Recommendations carry caveats.")
	}

	return sb.String()
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
	case "security":
		return CategorySecurity
	case "storage":
		return CategoryStorage
	default:
		return CategoryConfig
	}
}

func mapIncidentCategory(cat string) FindingCategory {
	switch strings.ToLower(cat) {
	case "transport":
		return CategoryTransport
	case "mesh":
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
	case "medium":
		return AttentionMedium
	case "low":
		return AttentionLow
	default:
		return AttentionMedium
	}
}

func appendUnique(s []string, v string) []string {
	for _, e := range s {
		if e == v {
			return s
		}
	}
	return append(s, v)
}

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Now().UTC()
	}
	return t
}
