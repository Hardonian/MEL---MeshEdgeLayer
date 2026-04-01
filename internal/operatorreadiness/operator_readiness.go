// Package operatorreadiness derives a single canonical operator-facing readiness DTO from
// platform posture and privacy/export policy. No silent "clean" when policy is unknown.
package operatorreadiness

import (
	"strings"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/platform"
)

// Semantic is a bounded reason code for operator surfaces (export, support bundle, proofpack, handoff).
type Semantic string

const (
	SemanticAvailable         Semantic = "available"
	SemanticDegraded          Semantic = "degraded"
	SemanticGated             Semantic = "gated"
	SemanticUnsupported       Semantic = "unsupported"
	SemanticUnavailable       Semantic = "unavailable"
	SemanticUnknownPartial    Semantic = "unknown_partial"
	SemanticSparse            Semantic = "sparse"
	SemanticPartial           Semantic = "partial"
	SemanticCapabilityLimited Semantic = "capability_limited"
	SemanticPolicyLimited     Semantic = "policy_limited"
	SemanticStale             Semantic = "stale"
)

// ArtifactStrength describes honest usefulness of assembled exports/bundles at this moment.
type ArtifactStrength string

const (
	ArtifactUsefulNow          ArtifactStrength = "useful_now"
	ArtifactUsableDegraded     ArtifactStrength = "usable_degraded"
	ArtifactWeakerCheckRuntime ArtifactStrength = "weaker_until_runtime_checked"
	ArtifactBlocked            ArtifactStrength = "blocked"
)

// BlockerSummary is a short machine+human line for chooser-adjacent UI.
type BlockerSummary struct {
	Code    string `json:"code"`
	Summary string `json:"summary"`
}

// OperatorReadinessDTO is the canonical JSON shape for version and posture endpoints.
type OperatorReadinessDTO struct {
	Semantic          Semantic         `json:"semantic"`
	Summary           string           `json:"summary"`
	ArtifactStrength  ArtifactStrength `json:"artifact_strength"`
	Blockers          []BlockerSummary `json:"blockers,omitempty"`
	EvidenceBasis     []string         `json:"evidence_basis,omitempty"`
	GeneratedFromNote string           `json:"generated_from_note,omitempty"`
}

// FromConfig builds readiness from live config (same inputs as platform.BuildPosture).
func FromConfig(cfg config.Config) OperatorReadinessDTO {
	post := platform.BuildPosture(cfg)
	return FromPlatformPosture(post)
}

// FromPlatformPosture maps already-built posture to the operator DTO (for version handler tests).
func FromPlatformPosture(post platform.PlatformPosture) OperatorReadinessDTO {
	out := OperatorReadinessDTO{
		EvidenceBasis:     []string{"platform_posture.evidence_export_delete", "platform_posture.export_redaction_enabled", "platform_posture.inference_enabled"},
		GeneratedFromNote: "deterministic_mapping_from_platform_posture",
	}
	exp := post.EvidenceExportDelete
	redact := post.ExportRedactionEnabled

	if !exp.ExportEnabled {
		out.Semantic = SemanticPolicyLimited
		out.ArtifactStrength = ArtifactBlocked
		out.Summary = "Instance policy disables evidence export — proofpack and escalation bundles are blocked; use plain handoff where allowed."
		out.Blockers = append(out.Blockers, BlockerSummary{
			Code:    "export_disabled_by_policy",
			Summary: out.Summary,
		})
		mergeInferenceBlockers(&out, post)
		return out
	}

	// Export allowed
	out.Semantic = SemanticAvailable
	out.ArtifactStrength = ArtifactUsefulNow
	out.Summary = "Evidence export allowed by policy — proofpack still reflects assembly-time gaps; review completeness before external handoff."

	if redact {
		out.Semantic = SemanticDegraded
		out.ArtifactStrength = ArtifactUsableDegraded
		out.Summary += " Privacy redaction is enabled — exports omit or redact operator-identifying fields per policy."
		out.Blockers = append(out.Blockers, BlockerSummary{
			Code:    "export_redaction_enabled",
			Summary: "Exports are redacted per platform.privacy.redact_exports — not a full-fidelity archive.",
		})
	}

	mergeInferenceBlockers(&out, post)
	return out
}

func mergeInferenceBlockers(out *OperatorReadinessDTO, post platform.PlatformPosture) {
	if post.InferenceEnabled && post.InferenceDegraded {
		note := strings.TrimSpace(post.InferenceCaveat)
		if note == "" {
			note = "Inference is enabled in config but no runtime provider is available; assistive layers are degraded."
		}
		out.Blockers = append(out.Blockers, BlockerSummary{
			Code:    "assist_runtime_not_ready",
			Summary: note,
		})
		if out.Semantic == SemanticAvailable {
			out.Semantic = SemanticDegraded
		}
		if out.ArtifactStrength == ArtifactUsefulNow {
			out.ArtifactStrength = ArtifactUsableDegraded
		}
	}
}
