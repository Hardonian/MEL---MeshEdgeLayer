package policy

import (
	"github.com/mel-project/mel/internal/config"
	"testing"
)

func TestExplain(t *testing.T) {
	cfg := config.Default()
	cfg.Privacy.StorePrecisePositions = true
	recs := Explain(cfg)
	if len(recs) == 0 {
		t.Fatal("expected recommendation")
	}
}
