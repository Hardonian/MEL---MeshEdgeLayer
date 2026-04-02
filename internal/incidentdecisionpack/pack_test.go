package incidentdecisionpack

import (
	"testing"
	"time"

	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/operatorreadiness"
)

func TestWhySurfacedOneLiner_FollowUpReview(t *testing.T) {
	inc := models.Incident{ID: "i1", ReviewState: "follow_up_needed"}
	got := WhySurfacedOneLiner(inc)
	if got == "" || got == "Open in workflow" {
		t.Fatalf("unexpected line: %q", got)
	}
}

func TestBuild_HasSchemaVersion(t *testing.T) {
	inc := models.Incident{ID: "x", Title: "t", State: "open"}
	pack := Build(inc, nil, operatorreadiness.OperatorReadinessDTO{Semantic: operatorreadiness.SemanticAvailable}, time.Unix(1, 0))
	if pack.SchemaVersion != models.IncidentDecisionPackSchemaVersion {
		t.Fatalf("schema: %q", pack.SchemaVersion)
	}
	if pack.Readiness == nil || pack.Readiness.ProofpackPath == "" {
		t.Fatalf("readiness missing: %#v", pack.Readiness)
	}
	if pack.Uncertainty == nil || len(pack.Uncertainty.NonClaims) == 0 {
		t.Fatalf("uncertainty missing")
	}
}
