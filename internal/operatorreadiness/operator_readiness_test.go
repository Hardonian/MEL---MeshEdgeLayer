package operatorreadiness

import (
	"testing"

	"github.com/mel-project/mel/internal/platform"
)

func TestFromPlatformPosture_ExportDisabled(t *testing.T) {
	p := platform.PlatformPosture{
		EvidenceExportDelete: platform.ExportDeleteSemantics{
			ExportEnabled: false,
			DeleteEnabled: false,
		},
		InferenceEnabled: false,
	}
	r := FromPlatformPosture(p)
	if r.Semantic != SemanticPolicyLimited {
		t.Fatalf("semantic: got %q", r.Semantic)
	}
	if r.ArtifactStrength != ArtifactBlocked {
		t.Fatalf("artifact: got %q", r.ArtifactStrength)
	}
	if len(r.Blockers) < 1 {
		t.Fatalf("expected export blocker")
	}
}

func TestFromPlatformPosture_ExportAllowedRedaction(t *testing.T) {
	p := platform.PlatformPosture{
		EvidenceExportDelete: platform.ExportDeleteSemantics{
			ExportEnabled: true,
		},
		ExportRedactionEnabled: true,
		InferenceEnabled:       false,
	}
	r := FromPlatformPosture(p)
	if r.Semantic != SemanticDegraded {
		t.Fatalf("semantic: got %q want degraded", r.Semantic)
	}
	if r.ArtifactStrength != ArtifactUsableDegraded {
		t.Fatalf("artifact: got %q", r.ArtifactStrength)
	}
}

func TestFromPlatformPosture_InferenceDegradedBlocker(t *testing.T) {
	p := platform.PlatformPosture{
		EvidenceExportDelete: platform.ExportDeleteSemantics{
			ExportEnabled: true,
		},
		ExportRedactionEnabled: false,
		InferenceEnabled:       true,
		InferenceDegraded:      true,
		InferenceCaveat:        "test caveat",
	}
	r := FromPlatformPosture(p)
	if r.Semantic != SemanticDegraded {
		t.Fatalf("semantic: got %q", r.Semantic)
	}
	found := false
	for _, b := range r.Blockers {
		if b.Code == "assist_runtime_not_ready" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected assist_runtime_not_ready blocker, got %#v", r.Blockers)
	}
}
