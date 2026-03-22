package topology

import (
	"math"
	"time"
)

// ScoreNode computes a transparent health score for a node.
// Every factor is named, weighted, and explained.
func ScoreNode(n Node, links []Link, thresholds StaleThresholds, now time.Time) (float64, HealthState, []ScoreFactor) {
	var factors []ScoreFactor

	// Factor 1: Recency of observation (0-1, weight 0.25)
	recency := computeRecencyFactor(n.LastSeenAt, thresholds.NodeStaleDuration, now)
	factors = append(factors, ScoreFactor{
		Name: "observation_recency", Weight: 0.25, Value: recency,
		Contribution: 0.25 * recency, Basis: basisForRecency(recency),
		Evidence: "last_seen_at=" + n.LastSeenAt,
	})

	// Factor 2: Direct reachability (0 or 1, weight 0.20)
	directReach := 0.0
	directBasis := "inferred"
	if n.LastDirectSeenAt != "" {
		dt, err := time.Parse(time.RFC3339, n.LastDirectSeenAt)
		if err == nil && now.Sub(dt) < thresholds.NodeStaleDuration {
			directReach = 1.0
			directBasis = "observed"
		}
	}
	factors = append(factors, ScoreFactor{
		Name: "direct_reachability", Weight: 0.20, Value: directReach,
		Contribution: 0.20 * directReach, Basis: directBasis,
		Evidence: "last_direct_seen_at=" + n.LastDirectSeenAt,
	})

	// Factor 3: Source trust quality (0-1, weight 0.15)
	trustVal := trustClassToValue(n.TrustClass)
	factors = append(factors, ScoreFactor{
		Name: "source_trust", Weight: 0.15, Value: trustVal,
		Contribution: 0.15 * trustVal, Basis: "policy",
		Evidence: "trust_class=" + string(n.TrustClass),
	})

	// Factor 4: Link stability (average quality of connected links, weight 0.15)
	linkStability := 0.0
	linkBasis := "inferred"
	if len(links) > 0 {
		sum := 0.0
		for _, l := range links {
			sum += l.QualityScore
		}
		linkStability = sum / float64(len(links))
		linkBasis = "observed"
	}
	factors = append(factors, ScoreFactor{
		Name: "link_stability", Weight: 0.15, Value: linkStability,
		Contribution: 0.15 * linkStability, Basis: linkBasis,
	})

	// Factor 5: Isolation degree (0=isolated, 1=well-connected, weight 0.10)
	isolation := 0.0
	if len(links) >= 3 {
		isolation = 1.0
	} else if len(links) >= 1 {
		isolation = float64(len(links)) / 3.0
	}
	factors = append(factors, ScoreFactor{
		Name: "connectivity", Weight: 0.10, Value: isolation,
		Contribution: 0.10 * isolation, Basis: "observed",
	})

	// Factor 6: Stale/contradictory penalty (0=bad, 1=clean, weight 0.10)
	stalePenalty := 1.0
	if n.Stale {
		stalePenalty = 0.0
	}
	if n.Quarantined {
		stalePenalty = 0.0
	}
	factors = append(factors, ScoreFactor{
		Name: "stale_penalty", Weight: 0.10, Value: stalePenalty,
		Contribution: 0.10 * stalePenalty, Basis: "observed",
	})

	// Factor 7: SNR quality proxy (0-1, weight 0.05)
	snrQuality := snrToQuality(n.LastSNR)
	factors = append(factors, ScoreFactor{
		Name: "snr_quality", Weight: 0.05, Value: snrQuality,
		Contribution: 0.05 * snrQuality, Basis: "observed",
	})

	// Compute total score
	total := 0.0
	for _, f := range factors {
		total += f.Contribution
	}
	total = clamp01(total)

	state := classifyNodeHealth(total, n, links)
	return total, state, factors
}

// ScoreLink computes a transparent quality score for a link.
func ScoreLink(l Link, thresholds StaleThresholds, now time.Time) (float64, []ScoreFactor) {
	var factors []ScoreFactor

	// Factor 1: Recency
	recency := computeRecencyFactor(l.LastObservedAt, thresholds.LinkStaleDuration, now)
	factors = append(factors, ScoreFactor{
		Name: "observation_recency", Weight: 0.25, Value: recency,
		Contribution: 0.25 * recency, Basis: basisForRecency(recency),
	})

	// Factor 2: Observed vs inferred
	obsVal := 0.5
	obsBasis := "inferred"
	if l.Observed {
		obsVal = 1.0
		obsBasis = "observed"
	}
	factors = append(factors, ScoreFactor{
		Name: "observation_basis", Weight: 0.20, Value: obsVal,
		Contribution: 0.20 * obsVal, Basis: obsBasis,
	})

	// Factor 3: Source trust
	factors = append(factors, ScoreFactor{
		Name: "source_trust", Weight: 0.15, Value: l.SourceTrustLevel,
		Contribution: 0.15 * l.SourceTrustLevel, Basis: "policy",
	})

	// Factor 4: Reliability history
	factors = append(factors, ScoreFactor{
		Name: "reliability", Weight: 0.15, Value: l.Reliability,
		Contribution: 0.15 * l.Reliability, Basis: "observed",
	})

	// Factor 5: Intermittence penalty
	intermittence := 1.0
	if l.IntermittenceCount > 10 {
		intermittence = 0.2
	} else if l.IntermittenceCount > 5 {
		intermittence = 0.5
	} else if l.IntermittenceCount > 0 {
		intermittence = 0.8
	}
	factors = append(factors, ScoreFactor{
		Name: "intermittence", Weight: 0.10, Value: intermittence,
		Contribution: 0.10 * intermittence, Basis: "observed",
	})

	// Factor 6: Contradiction penalty — contradicted observations must not read as "healthy"
	// even when recency and direct observation flags are strong.
	contradictionPenalty := 1.0
	if l.Contradiction {
		contradictionPenalty = 0.0
	}
	factors = append(factors, ScoreFactor{
		Name: "contradiction_penalty", Weight: 0.10, Value: contradictionPenalty,
		Contribution: 0.10 * contradictionPenalty, Basis: "observed",
	})

	// Factor 7: Observation count confidence
	obsCount := math.Min(float64(l.ObservationCount)/20.0, 1.0)
	factors = append(factors, ScoreFactor{
		Name: "observation_confidence", Weight: 0.05, Value: obsCount,
		Contribution: 0.05 * obsCount, Basis: "observed",
	})

	total := 0.0
	for _, f := range factors {
		total += f.Contribution
	}
	total = clamp01(total)
	if l.Contradiction {
		// Hard ceiling: contradictory link evidence is not operator-trustworthy as "good".
		total = math.Min(total, 0.40)
	}
	return total, factors
}

func computeRecencyFactor(lastSeen string, staleThreshold time.Duration, now time.Time) float64 {
	if lastSeen == "" {
		return 0.0
	}
	t, err := time.Parse(time.RFC3339, lastSeen)
	if err != nil {
		return 0.0
	}
	age := now.Sub(t)
	if age <= 0 {
		return 1.0
	}
	if age >= staleThreshold {
		return 0.0
	}
	return 1.0 - (float64(age) / float64(staleThreshold))
}

func basisForRecency(recency float64) string {
	if recency >= 0.8 {
		return "observed"
	}
	if recency >= 0.3 {
		return "stale"
	}
	return "inferred"
}

func trustClassToValue(tc TrustClass) float64 {
	switch tc {
	case TrustDirectLocal:
		return 1.0
	case TrustTrusted:
		return 0.85
	case TrustPartial:
		return 0.5
	case TrustUntrusted:
		return 0.15
	default:
		return 0.3
	}
}

func snrToQuality(snr float64) float64 {
	if snr >= 10 {
		return 1.0
	}
	if snr <= -10 {
		return 0.0
	}
	return (snr + 10) / 20.0
}

func classifyNodeHealth(score float64, n Node, links []Link) HealthState {
	if n.Quarantined {
		return HealthQuarantined
	}
	if n.Stale {
		return HealthStale
	}
	if len(links) == 0 {
		return HealthIsolated
	}

	// Check bridge criticality: single connection point
	bridgeCritical := false
	if len(links) == 1 {
		bridgeCritical = true
	}

	// Check flapping: high intermittence across links
	totalIntermittence := int64(0)
	for _, l := range links {
		totalIntermittence += l.IntermittenceCount
	}
	if totalIntermittence > 20 {
		return HealthFlapping
	}

	// Check if only inferred
	allInferred := true
	for _, l := range links {
		if l.Observed {
			allInferred = false
			break
		}
	}
	if allInferred {
		return HealthInferredOnly
	}

	if bridgeCritical && score >= 0.5 {
		return HealthBridgeCritical
	}

	if score >= 0.75 {
		return HealthHealthy
	}
	if score >= 0.5 {
		return HealthDegraded
	}
	if score >= 0.25 {
		return HealthUnstable
	}
	return HealthWeaklyObserved
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
