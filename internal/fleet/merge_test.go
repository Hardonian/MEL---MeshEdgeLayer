package fleet

import (
	"testing"

	"github.com/mel-project/mel/internal/config"
)

func TestClassifyMergeExactDuplicate(t *testing.T) {
	c := ClassifyMerge("k1", "k1", true)
	if c.Disposition != DedupeExactDuplicate {
		t.Fatalf("got %s", c.Disposition)
	}
}

func TestClassifyMergeNearDuplicateMultiObserver(t *testing.T) {
	c := ClassifyMerge("k1", "k1", false)
	if c.Disposition != DedupeNearDuplicate {
		t.Fatalf("got %s", c.Disposition)
	}
	if c.MergePosture != MergePostureSummaryWithLineage {
		t.Fatalf("merge posture %s", c.MergePosture)
	}
}

func TestClassifyMergeDistinct(t *testing.T) {
	c := ClassifyMerge("a", "b", false)
	if c.Disposition != DedupeRelatedDistinct {
		t.Fatalf("got %s", c.Disposition)
	}
}

func TestBuildTruthSummaryPartialFleet(t *testing.T) {
	cfg := config.Default()
	cfg.Scope.SiteID = "site-a"
	cfg.Scope.ExpectedFleetReporterCount = 3
	s, err := BuildTruthSummary(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.TruthPosture != TruthPosturePartialFleet {
		t.Fatalf("truth posture %s", s.TruthPosture)
	}
	if s.Visibility != VisibilityPartialFleet {
		t.Fatalf("visibility %s", s.Visibility)
	}
}
