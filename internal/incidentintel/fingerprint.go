// Package incidentintel holds deterministic, versioned incident fingerprinting and
// similarity primitives. Outputs are assistive and association-bounded unless
// backed by explicit persisted evidence references.
package incidentintel

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/models"
)

const (
	SchemaVersion  = "mel.incident_fingerprint/v1"
	ProfileVersion = "weights/v1"
)

// ComponentKey names stable fingerprint dimensions for explainable similarity.
type ComponentKey string

const (
	CompAnomalyFamily    ComponentKey = "anomaly_family"
	CompEvidenceKinds    ComponentKey = "evidence_chain_kinds"
	CompTransportRuntime ComponentKey = "transport_runtime_mode"
	CompDeadLetterClass  ComponentKey = "dead_letter_reason_top"
	CompControlActions   ComponentKey = "linked_control_action_types"
	CompDriftSegment     ComponentKey = "topology_drift_transport_anomaly"
	CompDegradedMarkers  ComponentKey = "intel_degraded_markers"
	CompProvenance       ComponentKey = "source_provenance"
	CompRecurrenceTiming ComponentKey = "recurrence_timing_bucket"
	CompEvidenceDensity  ComponentKey = "evidence_density_profile"
)

// FingerprintV1 is the canonical structured fingerprint for an incident at compute time.
type FingerprintV1 struct {
	IncidentID         string              `json:"incident_id"`
	LegacySignatureKey string              `json:"legacy_signature_key"`
	SchemaVersion      string              `json:"schema_version"`
	ProfileVersion     string              `json:"profile_version"`
	Components         map[string][]string `json:"components"`
	SparsityMarkers    []string            `json:"sparsity_markers"`
	CanonicalHash      string              `json:"canonical_hash"`
	ComputedAt         string              `json:"computed_at"`
}

// BuildFingerprintV1 derives a versioned fingerprint from incident row + evidence items + drift hints.
func BuildFingerprintV1(inc models.Incident, legacySigKey string, evidence []models.IncidentEvidenceItem, drift []models.IncidentDriftFingerprint, intelDegradedReasons []string) FingerprintV1 {
	now := time.Now().UTC().Format(time.RFC3339)
	comp := map[string][]string{}

	cat := strings.TrimSpace(strings.ToLower(inc.Category))
	rt := strings.TrimSpace(strings.ToLower(inc.ResourceType))
	comp[string(CompAnomalyFamily)] = []string{firstNonEmpty(cat, "unknown"), firstNonEmpty(rt, "unknown")}

	kindSet := map[string]struct{}{}
	dlTop := ""
	for _, it := range evidence {
		k := strings.TrimSpace(strings.ToLower(it.Kind))
		if k != "" {
			kindSet[k] = struct{}{}
		}
		if it.Kind == "dead_letter_reason_cluster" && dlTop == "" {
			dlTop = extractQuotedReason(it.Summary)
		}
	}
	comp[string(CompEvidenceKinds)] = sortedKeys(kindSet)

	trMode := "unknown"
	if rt == "transport" && strings.TrimSpace(inc.ResourceID) != "" {
		trMode = "transport_resource_observed"
	}
	comp[string(CompTransportRuntime)] = []string{trMode}

	if dlTop != "" {
		comp[string(CompDeadLetterClass)] = []string{truncate(dlTop, 120)}
	} else {
		comp[string(CompDeadLetterClass)] = []string{"none_observed_in_window"}
	}

	atypes := map[string]struct{}{}
	for _, ca := range inc.LinkedControlActions {
		t := strings.TrimSpace(strings.ToLower(ca.ActionType))
		if t != "" {
			atypes[t] = struct{}{}
		}
	}
	comp[string(CompControlActions)] = sortedKeys(atypes)

	driftSeg := []string{"none"}
	if len(drift) > 0 {
		driftSeg = driftSeg[:0]
		for _, d := range drift {
			if strings.TrimSpace(d.Reason) == "" {
				continue
			}
			driftSeg = append(driftSeg, truncate(strings.ToLower(d.Reason), 80))
			if len(driftSeg) >= 4 {
				break
			}
		}
		if len(driftSeg) == 0 {
			driftSeg = []string{"none"}
		}
	}
	comp[string(CompDriftSegment)] = driftSeg

	degr := make([]string, 0, len(intelDegradedReasons))
	for _, r := range intelDegradedReasons {
		r = strings.TrimSpace(r)
		if r != "" {
			degr = append(degr, r)
		}
	}
	sort.Strings(degr)
	comp[string(CompDegradedMarkers)] = degr

	prov := []string{"incident_row"}
	if len(evidence) > 0 {
		prov = append(prov, "correlated_evidence_items")
	}
	if len(inc.LinkedControlActions) > 0 {
		prov = append(prov, "linked_control_actions")
	}
	comp[string(CompProvenance)] = prov

	occ := parseTime(firstNonEmpty(inc.OccurredAt, inc.UpdatedAt))
	if !occ.IsZero() {
		h := occ.UTC().Hour()
		bucket := "utc_hour_unknown"
		switch {
		case h >= 6 && h < 12:
			bucket = "utc_morning"
		case h >= 12 && h < 18:
			bucket = "utc_afternoon"
		case h >= 18 && h < 23:
			bucket = "utc_evening"
		default:
			bucket = "utc_night"
		}
		comp[string(CompRecurrenceTiming)] = []string{bucket}
	} else {
		comp[string(CompRecurrenceTiming)] = []string{"occurred_at_unparsed"}
	}

	density := "sparse"
	switch {
	case len(evidence) >= 5:
		density = "dense"
	case len(evidence) >= 3:
		density = "moderate"
	}
	comp[string(CompEvidenceDensity)] = []string{density}

	sparse := sparsityFrom(inc, evidence, len(drift))

	h := canonicalHash(legacySigKey, comp)
	return FingerprintV1{
		IncidentID:         inc.ID,
		LegacySignatureKey: legacySigKey,
		SchemaVersion:      SchemaVersion,
		ProfileVersion:     ProfileVersion,
		Components:         comp,
		SparsityMarkers:    sparse,
		CanonicalHash:      h,
		ComputedAt:         now,
	}
}

func sparsityFrom(inc models.Incident, evidence []models.IncidentEvidenceItem, driftN int) []string {
	var out []string
	if len(evidence) < 2 {
		out = append(out, "limited_correlated_evidence")
	}
	if len(inc.LinkedControlActions) == 0 {
		out = append(out, "no_linked_control_actions")
	}
	if strings.TrimSpace(inc.ResourceID) == "" {
		out = append(out, "empty_resource_id")
	}
	if driftN == 0 && inc.ResourceType == "transport" && strings.TrimSpace(inc.ResourceID) != "" {
		out = append(out, "no_transport_anomaly_history_in_window")
	}
	sort.Strings(out)
	return dedupeSorted(out)
}

func canonicalHash(legacySig string, comp map[string][]string) string {
	keys := make([]string, 0, len(comp))
	for k := range comp {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("legacy_sig:")
	b.WriteString(legacySig)
	b.WriteByte('\n')
	for _, k := range keys {
		vals := append([]string(nil), comp[k]...)
		sort.Strings(vals)
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(strings.Join(vals, "|"))
		b.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return "fp-" + hex.EncodeToString(sum[:12])
}

// ComponentWeights for weighted Jaccard-style similarity (deterministic profile).
var ComponentWeights = map[string]float64{
	string(CompAnomalyFamily):    1.4,
	string(CompEvidenceKinds):    1.2,
	string(CompTransportRuntime): 1.0,
	string(CompDeadLetterClass):  1.1,
	string(CompControlActions):   1.0,
	string(CompDriftSegment):     0.9,
	string(CompDegradedMarkers):  0.4,
	string(CompProvenance):       0.3,
	string(CompRecurrenceTiming): 0.25,
	string(CompEvidenceDensity):  0.5,
}

// SimilarityResult is an explainable comparison between two fingerprints.
type SimilarityResult struct {
	Score                float64  `json:"score"`
	Category             string   `json:"category"`
	MatchedComponents    []string `json:"matched_components"`
	MismatchedComponents []string `json:"mismatched_components"`
	WeakDimensions       []string `json:"weak_or_sparse_dimensions"`
	Explanation          []string `json:"explanation"`
	InsufficientEvidence bool     `json:"insufficient_evidence"`
}

// CompareFingerprints returns weighted overlap and human-readable reasons.
func CompareFingerprints(a, b FingerprintV1) SimilarityResult {
	var matched, mismatched, weak []string
	var expl []string
	insufficient := false

	allKeys := map[string]struct{}{}
	for k := range a.Components {
		allKeys[k] = struct{}{}
	}
	for k := range b.Components {
		allKeys[k] = struct{}{}
	}
	keys := sortedKeys(allKeys)

	var num, den float64
	for _, k := range keys {
		w := ComponentWeights[k]
		if w <= 0 {
			w = 0.5
		}
		va := a.Components[k]
		vb := b.Components[k]
		if len(va) == 0 && len(vb) == 0 {
			continue
		}
		inter := intersectionStrings(va, vb)
		unionN := unionLen(va, vb)
		if unionN == 0 {
			continue
		}
		j := float64(len(inter)) / float64(unionN)
		num += w * j
		den += w
		if len(inter) > 0 {
			matched = append(matched, k+":"+strings.Join(sortedCopy(inter), ","))
		}
		if j < 1 && (len(va) > 0 || len(vb) > 0) {
			mismatched = append(mismatched, k)
		}
		if j < 0.35 && (len(va) > 0 || len(vb) > 0) {
			weak = append(weak, k)
		}
	}

	score := 0.0
	if den > 0 {
		score = num / den
	}

	if len(a.Components[string(CompEvidenceKinds)]) < 2 && len(b.Components[string(CompEvidenceKinds)]) < 2 {
		insufficient = true
		expl = append(expl, "Both incidents have thin evidence-chain fingerprints; treat similarity as weak.")
	}

	cat := categorizeSimilarity(score, insufficient, a.LegacySignatureKey == b.LegacySignatureKey && a.LegacySignatureKey != "")
	expl = append(expl, explainCategory(cat, score)...)

	return SimilarityResult{
		Score:                round4(score),
		Category:             cat,
		MatchedComponents:    matched,
		MismatchedComponents: mismatched,
		WeakDimensions:       weak,
		Explanation:          expl,
		InsufficientEvidence: insufficient,
	}
}

func categorizeSimilarity(score float64, insufficient bool, sameLegacySig bool) string {
	if sameLegacySig {
		return "same_recurring_signature_bucket"
	}
	if insufficient && score < 0.45 {
		return "insufficient_evidence"
	}
	switch {
	case score >= 0.82:
		return "strong_family_match"
	case score >= 0.58:
		return "moderate_similarity"
	case score >= 0.35:
		return "weak_similarity"
	default:
		return "inconclusive_or_distant"
	}
}

func explainCategory(cat string, score float64) []string {
	s := fmt.Sprintf("Weighted component overlap score=%.3f (deterministic profile %s).", score, ProfileVersion)
	switch cat {
	case "same_recurring_signature_bucket":
		return []string{s, "Same legacy deterministic signature key as historical clustering; still not proof of identical root cause."}
	case "strong_family_match":
		return []string{s, "Multiple fingerprint components overlap strongly; review timelines before treating as same operational failure."}
	case "moderate_similarity":
		return []string{s, "Partial overlap across fingerprint components; could be related family or shared symptoms."}
	case "weak_similarity":
		return []string{s, "Limited overlap; may be adjacent symptoms or coincidental resemblance."}
	case "insufficient_evidence":
		return []string{s, "Evidence-chain features are sparse; MEL cannot support a strong resemblance claim."}
	default:
		return []string{s, "Low resemblance under the current fingerprint profile."}
	}
}

func intersectionStrings(a, b []string) []string {
	set := map[string]struct{}{}
	for _, x := range a {
		x = strings.TrimSpace(x)
		if x != "" {
			set[x] = struct{}{}
		}
	}
	var out []string
	seen := map[string]struct{}{}
	for _, x := range b {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if _, ok := set[x]; ok {
			if _, du := seen[x]; !du {
				seen[x] = struct{}{}
				out = append(out, x)
			}
		}
	}
	sort.Strings(out)
	return out
}

func unionLen(a, b []string) int {
	m := map[string]struct{}{}
	for _, x := range a {
		x = strings.TrimSpace(x)
		if x != "" {
			m[x] = struct{}{}
		}
	}
	for _, x := range b {
		x = strings.TrimSpace(x)
		if x != "" {
			m[x] = struct{}{}
		}
	}
	return len(m)
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func round4(f float64) float64 {
	return float64(int(f*10000+0.5)) / 10000
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return strings.TrimSpace(b)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func extractQuotedReason(summary string) string {
	// summary like: Dead-letter reason "foo" occurred ...
	i := strings.Index(summary, `"`)
	if i < 0 {
		return ""
	}
	j := strings.Index(summary[i+1:], `"`)
	if j < 0 {
		return ""
	}
	return summary[i+1 : i+1+j]
}

func parseTime(v string) time.Time {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func dedupeSorted(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := []string{in[0]}
	for i := 1; i < len(in); i++ {
		if in[i] != in[i-1] {
			out = append(out, in[i])
		}
	}
	return out
}

// ToModel converts to API model.
func (f FingerprintV1) ToModel() models.IncidentFingerprint {
	return models.IncidentFingerprint{
		SchemaVersion:      f.SchemaVersion,
		ProfileVersion:     f.ProfileVersion,
		LegacySignatureKey: f.LegacySignatureKey,
		CanonicalHash:      f.CanonicalHash,
		Components:         f.Components,
		SparsityMarkers:    append([]string(nil), f.SparsityMarkers...),
		ComputedAt:         f.ComputedAt,
	}
}

// FingerprintFromModel reconstructs internal struct for comparison (tests).
func FingerprintFromModel(m models.IncidentFingerprint, incidentID string) FingerprintV1 {
	return FingerprintV1{
		IncidentID:         incidentID,
		LegacySignatureKey: m.LegacySignatureKey,
		SchemaVersion:      m.SchemaVersion,
		ProfileVersion:     m.ProfileVersion,
		Components:         m.Components,
		SparsityMarkers:    m.SparsityMarkers,
		CanonicalHash:      m.CanonicalHash,
		ComputedAt:         m.ComputedAt,
	}
}

// ComponentMapJSON for persistence.
func ComponentMapJSON(f FingerprintV1) (string, error) {
	b, err := json.Marshal(f.Components)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// SparsityJSON for persistence.
func SparsityJSON(f FingerprintV1) (string, error) {
	b, err := json.Marshal(f.SparsityMarkers)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
