package fleet

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// DedupeDisposition is the honest outcome when duplicate or related evidence is classified.
type DedupeDisposition string

const (
	DedupeExactDuplicate  DedupeDisposition = "exact_duplicate"
	DedupeNearDuplicate   DedupeDisposition = "near_duplicate_candidate"
	DedupeRelatedDistinct DedupeDisposition = "related_distinct"
	DedupeConflicting     DedupeDisposition = "conflicting"
	DedupeSuperseded      DedupeDisposition = "superseded"
	DedupeAmbiguous       DedupeDisposition = "ambiguous"
	DedupeMergedCanonical DedupeDisposition = "merged_canonical_summary"
)

// MergePosture describes how a summary relates to underlying observations.
type MergePosture string

const (
	MergePostureRawOnly            MergePosture = "raw_only"
	MergePostureSummaryWithLineage MergePosture = "summary_with_contributor_lineage"
	MergePostureNoSilentCollapse   MergePosture = "no_silent_ambiguity_collapse"
)

// MergeClassification is a deterministic, testable merge/dedupe classification for evidence keys.
type MergeClassification struct {
	Disposition  DedupeDisposition `json:"disposition"`
	MergePosture MergePosture      `json:"merge_posture"`
	MergeKey     string            `json:"merge_key"`
	Contributors []string          `json:"contributors,omitempty"`
	Notes        string            `json:"notes,omitempty"`
}

// ClassifyMerge compares two opaque observation keys (e.g. packet fingerprint + observer id) and returns disposition.
// This does not claim RF or network-wide truth — only structural dedupe posture for operator tooling.
func ClassifyMerge(keyA, keyB string, sameObserver bool) MergeClassification {
	a := strings.TrimSpace(keyA)
	b := strings.TrimSpace(keyB)
	if a == "" || b == "" {
		return MergeClassification{
			Disposition:  DedupeAmbiguous,
			MergePosture: MergePostureNoSilentCollapse,
			MergeKey:     "",
			Notes:        "empty key; cannot classify",
		}
	}
	if a == b {
		if sameObserver {
			return MergeClassification{
				Disposition:  DedupeExactDuplicate,
				MergePosture: MergePostureRawOnly,
				MergeKey:     mergeKeySHA256(a),
				Contributors: []string{"same_observer"},
				Notes:        "identical key within same observer scope",
			}
		}
		return MergeClassification{
			Disposition:  DedupeNearDuplicate,
			MergePosture: MergePostureSummaryWithLineage,
			MergeKey:     mergeKeySHA256(a),
			Contributors: []string{"observer_a", "observer_b"},
			Notes:        "same structural key reported by different observers; preserve per-observer metrics — not proof of network-wide flooding",
		}
	}
	return MergeClassification{
		Disposition:  DedupeRelatedDistinct,
		MergePosture: MergePostureRawOnly,
		MergeKey:     mergeKeySHA256(a + "|" + b),
		Notes:        "distinct keys; do not merge without operator review",
	}
}

func mergeKeySHA256(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// SortStringsDeterministic sorts for stable contributor ordering in exports.
func SortStringsDeterministic(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

// ExplainMergeForOperator returns a short honest sentence for CLI/UI.
func ExplainMergeForOperator(c MergeClassification) string {
	switch c.Disposition {
	case DedupeExactDuplicate:
		return "Treated as exact duplicate within one observer scope."
	case DedupeNearDuplicate:
		return "Same observation key from multiple observers: keep both raw rows; do not infer fleet-wide root cause."
	case DedupeRelatedDistinct:
		return "Distinct evidence paths: not auto-merged."
	case DedupeConflicting:
		return "Conflicting evidence: ambiguity preserved."
	default:
		return fmt.Sprintf("Merge disposition: %s", c.Disposition)
	}
}
