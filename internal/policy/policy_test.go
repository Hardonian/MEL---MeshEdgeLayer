package policy

import (
	"testing"

	"github.com/mel-project/mel/internal/config"
)

func TestExplain(t *testing.T) {
	cfg := config.Default()
	cfg.Privacy.StorePrecisePositions = true
	recs := Explain(cfg)
	if len(recs) == 0 {
		t.Fatal("expected recommendation")
	}
	if recs[0].ID == "" || recs[0].Reason == "" {
		t.Fatal("expected structured recommendation")
	}
}
