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
	inc := models.Incident{
		ID:    "x",
		Title: "t",
		State: "open",
		Intelligence: &models.IncidentIntelligence{
			EvidenceStrength:    "moderate",
			SignatureMatchCount: 2,
		},
		AssistSignals: &models.IncidentAssistSignals{SchemaVersion: "incident_assist_signals_v1", Signals: []models.IncidentAssistSignal{{Code: "recurrence_review_recommended", Title: "r"}}},
	}
	pack := Build(inc, nil, operatorreadiness.OperatorReadinessDTO{Semantic: operatorreadiness.SemanticAvailable}, time.Unix(1, 0))
	if pack.SchemaVersion != models.IncidentDecisionPackSchemaVersion {
		t.Fatalf("schema: %q", pack.SchemaVersion)
	}
	if pack.Readiness == nil || pack.Readiness.ProofpackPath == "" {
		t.Fatalf("readiness missing: %#v", pack.Readiness)
	}
	if pack.Guidance == nil {
		t.Fatalf("guidance missing")
	}
	if pack.Guidance.SupportPosture != "ready" {
		t.Fatalf("support posture: %#v", pack.Guidance)
	}
	if pack.Uncertainty == nil || len(pack.Uncertainty.NonClaims) == 0 {
		t.Fatalf("uncertainty missing")
	}
	if pack.AssistSignals == nil || len(pack.AssistSignals.Signals) == 0 {
		t.Fatal("expected assist_signals re-export on pack")
	}
}

func TestBuild_GuidanceUsesDeterministicPosture(t *testing.T) {
	inc := models.Incident{
		ID:          "x2",
		Title:       "fragile",
		State:       "open",
		ReviewState: "pending_review",
		ActionVisibility: &models.IncidentActionVisibilityPosture{
			Kind: "references_only",
		},
		Intelligence: &models.IncidentIntelligence{
			EvidenceStrength: "sparse",
			MitigationDurabilityMemory: &models.IncidentMitigationDurabilityMemory{
				Posture: "reopened_after_resolution_in_family",
			},
		},
		TriageSignals: &models.IncidentTriageSignals{Tier: 0},
	}
	pack := Build(inc, nil, operatorreadiness.OperatorReadinessDTO{Semantic: operatorreadiness.SemanticPolicyLimited}, time.Unix(2, 0))
	if pack.Guidance == nil {
		t.Fatalf("guidance missing")
	}
	if !pack.Guidance.MitigationFragilityWatch || !pack.Guidance.RepeatedFamilyConcern {
		t.Fatalf("expected fragility and family concern: %#v", pack.Guidance)
	}
	if pack.Guidance.ActionPosture != "guarded" {
		t.Fatalf("action posture = %q", pack.Guidance.ActionPosture)
	}
	if pack.Guidance.SupportPosture != "blocked" {
		t.Fatalf("support posture = %q", pack.Guidance.SupportPosture)
	}
}
