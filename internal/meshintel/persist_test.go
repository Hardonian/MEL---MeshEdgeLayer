package meshintel

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/topology"
)

func TestSaveAndRecentSnapshots(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mi.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	nodes := []topology.Node{{NodeNum: 1, LastSeenAt: time.Now().UTC().Format(time.RFC3339)}}
	ar := topology.Analyze(nodes, nil, topology.DefaultStaleThresholds(), time.Now().UTC())
	a := Compute(cfg, ar, MessageSignals{}, false, time.Now().UTC())
	if err := SaveSnapshot(database, a, 10); err != nil {
		t.Fatal(err)
	}
	list, err := RecentSnapshots(database, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].AssessmentID != a.AssessmentID {
		t.Fatalf("unexpected list: %+v", list)
	}
}
